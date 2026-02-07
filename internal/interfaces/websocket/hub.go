package websocket_interface

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	entity "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/entities"
	"go.uber.org/zap"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024 // 512KB
)


// --- CONSTANTS AND IMPORTS ---
// Standard library, github & project imports, plus timeouts/constants for connection health.
//    - writeWait: Max time for writing a websocket message (prevents stuck connections).
//    - pongWait: Allowed time to wait for a pong resp (connection health).
//    - pingPeriod: How often to ping clients to check if alive.
//    - maxMessageSize: Clamp message payload to prevent misuse.

// --- CLIENT STRUCT ---
// Represents a single WebSocket connection for a specific user.
type Client struct {
	hub      *Hub             // Reference to parent hub for dereg, communication.
	conn     *websocket.Conn  // Actual websocket connection (from gorilla/websocket).
	userID   string           // ID to group connections by user.
	send     chan []byte      // Outbound channel: hub pushes messages here, writePump reads.
	lastSeen time.Time        // Last time a message/ping/pong received (for liveness).
}

// --- HUB STRUCT ---
// The central manager for all websockets and message flow.
type Hub struct {
	logger *zap.Logger

	// Core client management:
	clients    map[string]map[*Client]struct{} // userID -> set of Client pointers
	broadcast  chan *BroadcastMessage          // Hub goroutine listens: any client, kafka, etc.
	register   chan *Client                    // New client connections notify here.
	unregister chan *Client                    // Closed/stale clients go here for cleanup.

	// Metrics and concurrency.
	mu              sync.RWMutex               // Protects access to clients & stats.
	totalClients    int                        // Count of active client connections.
	messagesSent    int64                      // Successful downstream deliveries.
	messagesDropped int64                      // Dropped messages due to backpressure.
	stopCh          chan struct{}              // Signals shutdown (closes hub goroutine).

	// Integration with async event sources (e.g., Kafka):
	NewInAppNotificationCh chan *entity.InAppWSMessage // Messages to send out to websockets.
}

// Message published by Kafka consumer (or any backend) to the user
type BroadcastMessage struct {
	UserID  string      // Target user
	Payload interface{} // Notification/message body (serializable to JSON)
}

// --- HUB CONSTRUCTOR ---
// Sets up all data structures, channels, then runs goroutines for core logic.
func NewHub(logger *zap.Logger) *Hub {
	hub := &Hub{
		logger:                 logger,
		clients:                make(map[string]map[*Client]struct{}),
		broadcast:              make(chan *BroadcastMessage, 512),
		register:               make(chan *Client, 64),
		unregister:             make(chan *Client, 64),
		stopCh:                 make(chan struct{}),
		NewInAppNotificationCh: make(chan *entity.InAppWSMessage, 1024),
	}

	go hub.run()                        // Main router/event loop for the hub.
	go hub.startKafkaSubscriberForwarder() // Goroutine: delivers notifications from Kafka to client(s).
	return hub
}

// --- KAFKA-FORWARDER GOROUTINE ---
// Listens for new in-app notification events from backend (Kafka or similar).
// When it receives a message, it forwards it into the hub's broadcast channel.
// This triggers a fan-out to *all* websocket clients for the target user.
func (h *Hub) startKafkaSubscriberForwarder() {
	for {
		select {
		case msg := <-h.NewInAppNotificationCh:
			h.Broadcast(msg.UserID, msg)
		case <-h.stopCh:
			return
		}
	}
}

// --- HUB GOROUTINE ("EVENT LOOP") ---
// This goroutine owns ALL cross-client updates in the hub. It's the "orchestrator".
// Signals/requests flow in via the four channels, are handled sequentially (thread-safe!):
//   - register: add a new client & log
//   - unregister: remove a dead client & log
//   - broadcast: deliver a message to all that user's clients
//   - ticker.C: periodic health/ping, removes zombies
//   - stopCh: initiates shutdown (used by shutdown method)
func (h *Hub) run() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case client := <-h.register:
			h.handleRegister(client)
		case client := <-h.unregister:
			h.handleUnregister(client)
		case message := <-h.broadcast:
			h.handleBroadcast(message)
		case <-ticker.C:
			h.handlePing()
		case <-h.stopCh:
			return
		}
	}
}

// --- HANDLE REGISTER ---
// Adds a brand new websocket connection to the in-memory map, increments counters.
func (h *Hub) handleRegister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client.userID]; !ok {
		h.clients[client.userID] = make(map[*Client]struct{})
	}
	h.clients[client.userID][client] = struct{}{}
	h.totalClients++

	userConnections := len(h.clients[client.userID])
	h.logger.Info("WebSocket client registered",
		zap.String("user_id", client.userID),
		zap.Int("user_connections", userConnections),
		zap.Int("total_clients", h.totalClients),
	)
}

// --- HANDLE UNREGISTER ---
// Removes a websocket client from the user group & all records;
// closes their send channel (halting outbound delivery), decrements metrics, deletes user group if empty.
func (h *Hub) handleUnregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.clients[client.userID]; ok {
		if _, exists := clients[client]; exists {
			delete(clients, client)
			close(client.send)
			h.totalClients--

			if len(clients) == 0 {
				delete(h.clients, client.userID)
			}
		}
	}
	h.logger.Info("WebSocket client unregistered",
		zap.String("user_id", client.userID),
		zap.Int("total_clients", h.totalClients),
	)
}

// --- HANDLE BROADCAST ---
// Receives a BroadcastMessage (generated by Kafka subscriber, or hub.Broadcast), and
// delivers it to all websocket connections for the specified user. Serializes payload as JSON for each client.
//
// If a client buffer is full (backpressure), the message is dropped for that client and a warning logged.
// Metrics for sent/dropped are updated. Entire process protected from race by RLock.
func (h *Hub) handleBroadcast(message *BroadcastMessage) {
	h.mu.RLock()
	clients := h.clients[message.UserID]
	h.mu.RUnlock()

	if len(clients) == 0 {
		h.logger.Debug("No websocket clients for user",
			zap.String("user_id", message.UserID),
		)
		return
	}

	data, err := json.Marshal(message.Payload)
	if err != nil {
		h.logger.Error("Failed to marshal broadcast payload",
			zap.Error(err),
			zap.String("user_id", message.UserID),
		)
		return
	}

	sent := 0
	dropped := 0

	for client := range clients {
		select {
		case client.send <- data:
			sent++
		default:
			dropped++
			h.logger.Warn("Message dropped (send buffer full)",
				zap.String("user_id", client.userID),
			)
		}
	}

	h.mu.Lock()
	h.messagesSent += int64(sent)
	h.messagesDropped += int64(dropped)
	h.mu.Unlock()

	if sent > 0 {
		h.logger.Debug("Broadcast sent",
			zap.String("user_id", message.UserID),
			zap.Int("sent", sent),
			zap.Int("dropped", dropped),
		)
	}
}

// --- HANDLE PING ---
// Runs periodically from a timer. Finds all stale clients (have not responded to pings or sent any data).
// Those clients are flagged for unregistration; their connections are closed and resources released.
func (h *Hub) handlePing() {
	staleThreshold := time.Now().Add(-2 * pongWait)

	h.mu.RLock()
	var staleClients []*Client
	for _, clients := range h.clients {
		for client := range clients {
			if client.lastSeen.Before(staleThreshold) {
				staleClients = append(staleClients, client)
			}
		}
	}
	h.mu.RUnlock()

	for _, client := range staleClients {
		h.logger.Warn("Removing stale websocket client",
			zap.String("user_id", client.userID),
			zap.Time("last_seen", client.lastSeen),
		)
		h.unregister <- client
		// Best practice is to close connection asynchronously
		go func(c *Client) {
			_ = c.conn.Close()
		}(client)
	}
}

// --- PUBLIC BROADCAST API ---
// Non-blocking: attempts to send a Notification (userID + payload) into the hub's broadcast channel.
// If channel is full, logs the event and drops the message to prevent deadlocks.
func (h *Hub) Broadcast(userID string, payload interface{}) {
	select {
	case h.broadcast <- &BroadcastMessage{UserID: userID, Payload: payload}:
	default:
		h.logger.Error("Broadcast channel full, dropping message",
			zap.String("user_id", userID),
		)
	}
}

// --- MONITORING / METRICS ---
// Returns current statistics for monitoring: total websocket conns, number of unique users receiving notifications,
// # of successful messages sent, and dropped count due to overload.
func (h *Hub) GetMetrics() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return map[string]interface{}{
		"total_clients":    h.totalClients,
		"unique_users":     len(h.clients),
		"messages_sent":    h.messagesSent,
		"messages_dropped": h.messagesDropped,
	}
}

// ---- CLIENT SIDE: READPUMP ---
// Dedicated goroutine for each websocket client that:
//   - Reads *incoming* messages from the client socket
//   - Checks for ping/pong from the client as a liveness/heartbeat signal
//   - Updates lastSeen timestamp for health checks
//   - Responds to a "ping" type message with a "pong" (for client-side healthcheck)
// On error or disconnect, it unregisters itself and closes the socket.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.lastSeen = time.Now()
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.hub.logger.Error("Unexpected websocket close",
					zap.Error(err),
					zap.String("user_id", c.userID),
				)
			}
			break
		}

		// Parse for heartbeat (ping).
		var msg struct {
			Type string `json:"type"`
		}
		if json.Unmarshal(message, &msg) == nil && msg.Type == "ping" {
			c.lastSeen = time.Now()
			pongMsg := struct {
				Type      string `json:"type"`
				Timestamp string `json:"timestamp"`
			}{"pong", time.Now().Format(time.RFC3339Nano)}
			resp, _ := json.Marshal(pongMsg)
			select {
			case c.send <- resp:
			default:
				// Silent fail if outbound channel full.
			}
		}
	}
}

// ---- CLIENT SIDE: WRITEPUMP ---
// Dedicated goroutine for *each* client that:
//   - Awaits outbound messages (from hub) on c.send chan, delivers them to the websocket socket
//   - Periodically sends ping frames to client (for connectivity)
//   - On error/closed/unexpected state, tears down connection and halts goroutine
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				c.hub.logger.Error("Write to websocket failed",
					zap.Error(err),
					zap.String("user_id", c.userID),
				)
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// --- UPGRADER ---
// Sets up the HTTP handler to accept a websocket upgrade. In prod, CheckOrigin should restrict connections by Origin header.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		/*
			origin := r.Header.Get("Origin")
			return origin == "https://your-app.com"
		*/
		// UPDATE: In production ensure allowed domains only!
		return true
	},
}

// --- AUTHENTICATION FUNCTION TYPE ---
// Used to extract and validate user credentials on websocket connect.
type AuthFunc func(r *http.Request) (string, error)

func (h *Hub) ServeWS(auth AuthFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := auth(r)
		if err != nil || userID == "" {
			h.logger.Warn("Unauthorized web socket connection attempt",
				zap.Error(err),
			)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			h.logger.Error("Websocket upgrade failure",
				zap.Error(err),
				zap.String("user_id", userID),
			)
			return
		}

		client := &Client{
			hub:      h,
			conn:     conn,
			userID:   userID,
			send:     make(chan []byte, 256),
			lastSeen: time.Now(),
		}

		h.register <- client
		go client.writePump() // Outbound sender (hub-to-client)
		go client.readPump()  // Inbound reader (client-to-hub, liveness only)
	}
}

// --- KAFKA/EXTERNAL MESSAGE INJECTION ---
// External systems (e.g. Kafka consumer goroutines) call this to enqueue a notification
// to a specific user, which flows through the hub and will reach all active websocket clients.
func (h *Hub) NotifyInAppMessage(msg *entity.InAppWSMessage) error {
	if msg == nil || msg.UserID == "" {
		return errors.New("missing user ID in in-app ws message")
	}
	select {
	case h.NewInAppNotificationCh <- msg:
		return nil
	default:
		h.logger.Error("InApp notification channel full, dropping notification",
			zap.String("user_id", msg.UserID),
		)
		return errors.New("notify in-app channel full")
	}
}

// --- SHUTDOWN LOGIC ---
// To be called on graceful service shutdown. Shuts down all goroutines, closes active connections,
// and resets internal state to blank (safe for process exit).
func (h *Hub) Shutdown(ctx context.Context) error {
	h.logger.Info("Shutting down WebSocket hub")
	close(h.stopCh) // Signal all listening goroutines to exit

	h.mu.Lock()
	defer h.mu.Unlock()
	for _, clients := range h.clients {
		for client := range clients {
			close(client.send)
			client.conn.Close()
		}
	}
	h.clients = make(map[string]map[*Client]struct{})
	return nil
}


/* -------   DATA FLOW EXPLANATION (from sender to client)   -------

1. Notification source (e.g., Kafka) receives a new in-app notification, triggers NotifyInAppMessage().
2. NotifyInAppMessage() puts it on the NewInAppNotificationCh channel (or returns error if full).
3. The startKafkaSubscriberForwarder goroutine (on Hub) reads from this channel.
4. It immediately calls Hub.Broadcast(userID, payload), feeding it onto the broadcast channel.
5. The hub.run() goroutine receives this on its .broadcast channel, then calls handleBroadcast().
6. handleBroadcast() finds all Client conns for user, prepares message (marshals as JSON), and sends to each
   Client's .send channel (if not full).
7. Each Client has an active writePump() goroutine, reading its .send channel. When new data arrives, it writes
   a websocket TextMessage (JSON) to the underlying socket.
8. The client-side app receives the notification, e.g. as a browser onmessage event.

Client side health is maintained via:
   - readPump(): watches for pings from client (heartbeat)
   - writePump(): sends periodic ping frames
Stale/inactive connections are routinely killed via handlePing(), based on lastSeen timestamps.

All resource cleanup, reconnections, and ordering are thread-safe and mostly handled via main goroutine.

*/

