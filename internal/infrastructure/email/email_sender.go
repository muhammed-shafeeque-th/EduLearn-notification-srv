package email

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"sync"
	"time"

	"github.com/jordan-wright/email"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/application/ports"
	entity "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/entities"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/infrastructure/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// EmailSender is responsible for sending emails over SMTP.
type EmailSender struct {
	config      EmailConfig
	logger      *zap.Logger
	pool        *smtpPool
	rateLimiter ports.RateLimiter
	emailPool   sync.Pool
	metrics     *EmailMetrics
}

// EmailConfig holds the configuration for connecting to an SMTP server.
type EmailConfig struct {
	SMTPHost    string
	SMTPPort    string
	Username    string
	Password    string
	FromName    string
	PoolSize    int
	PoolTimeout time.Duration
	SendTimeout time.Duration
	MaxRetries  int
}

// DefaultEmailConfig returns an EmailConfig with reasonable defaults filled in.
func DefaultEmailConfig(host, port, username, password string) EmailConfig {
	return EmailConfig{
		SMTPHost:    host,
		SMTPPort:    port,
		Username:    username,
		Password:    password,
		FromName:    "EduLearn",
		PoolSize:    10,
		PoolTimeout: 30 * time.Second,
		SendTimeout: 10 * time.Second,
		MaxRetries:  3,
	}
}

type EmailMetrics struct {
	mu               sync.RWMutex
	EmailSentTotal   prometheus.Counter
	EmailSendErrors  prometheus.Counter
	EmailRetries     prometheus.Counter
	EmailRateLimited prometheus.Counter
}

// NewEmailSender initializes a new EmailSender.
func NewEmailSender(
	config EmailConfig,
	rateLimiter ports.RateLimiter,
	logger *zap.Logger,
) (*EmailSender, error) {
	pool, err := newSMTPPool(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create SMTP pool: %w", err)
	}

	sender := &EmailSender{
		config:      config,
		logger:      logger,
		pool:        pool,
		rateLimiter: rateLimiter,
		metrics: &EmailMetrics{
			EmailSentTotal:   metrics.EmailSentTotal,
			EmailSendErrors:  metrics.EmailSendErrors,
			EmailRateLimited: metrics.EmailRateLimited,
			EmailRetries:     metrics.EmailRetries,
		},
		emailPool: sync.Pool{
			New: func() interface{} {
				e := email.NewEmail()
				return e
			},
		},
	}

	logger.Info("Email sender initialized",
		zap.String("smtp_host", config.SMTPHost),
		zap.Int("pool_size", config.PoolSize),
	)

	return sender, nil
}

// Send attempts to deliver the notification by retrying up to MaxRetries.
func (s *EmailSender) Send(ctx context.Context, notification ports.NotificationLike) error {
	start := time.Now()

	// Check rate limit
	if err := s.rateLimiter.Allow(ctx, notification.GetRecipient()); err != nil {
		s.metrics.incrementRateLimited()
		return fmt.Errorf("rate limit exceeded: %w", err)
	}
	// Validate notification
	if notification.GetType() != entity.NotificationType(entity.EmailNotification) {
		return fmt.Errorf("invalid notification type: %s", notification.GetType())
	}

	var lastErr error
	for attempt := 1; attempt <= s.config.MaxRetries; attempt++ {
		if err := s.sendWithTimeout(ctx, notification); err == nil {
			s.metrics.recordSuccess(time.Since(start))
			return nil
		} else {
			lastErr = err
			s.metrics.incrementRetries()
			s.logger.Warn("Email send attempt failed",
				zap.Int("attempt", attempt),
				zap.String("recipient", notification.GetRecipient()),
				zap.Error(err),
			)

			if attempt < s.config.MaxRetries {
				time.Sleep(time.Duration(attempt) * time.Second)
			}
		}
	}

	s.metrics.incrementFailed()
	return fmt.Errorf("failed after %d attempts: %w", s.config.MaxRetries, lastErr)
}

func (s *EmailSender) sendWithTimeout(ctx context.Context, notification ports.NotificationLike) error {
	ctx, cancel := context.WithTimeout(ctx, s.config.SendTimeout)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- s.sendEmail(ctx, notification)
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return fmt.Errorf("send timeout: %w", ctx.Err())
	}
}

// sendEmail sends the individual email using SMTP client from pool, returning resources to their pools.
func (s *EmailSender) sendEmail(ctx context.Context, notification ports.NotificationLike) error {
	msg := s.emailPool.Get().(*email.Email)
	defer s.resetAndReturnEmail(msg)

	msg.From = fmt.Sprintf("%s <%s>", s.config.FromName, s.config.Username)
	msg.To = []string{notification.GetRecipient()}
	msg.Subject = notification.GetSubject()
	msg.HTML = []byte(notification.GetBody())
	if msg.Headers == nil {
		msg.Headers = make(map[string][]string)
	}
	msg.Headers.Set("X-Notification-ID", notification.GetUserId())
	msg.Headers.Set("X-User-ID", notification.GetUserId())

	client, err := s.pool.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to get SMTP client: %w", err)
	}

	// Carefully manage connection: always Put/Close even on panic
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("panic in sendEmail; closing smtp client", zap.Any("recover", r))
			client.Close()
			return
		}
		if err := client.Reset(); err != nil {
			client.Close()
			return
		}
		s.pool.Put(client)
	}()

	if err := s.sendViaSMTP(client, msg); err != nil {
		return fmt.Errorf("failed to send via SMTP: %w", err)
	}

	s.logger.Info("Email sent successfully",
		zap.String("recipient", notification.GetRecipient()),
	)
	return nil
}

func (s *EmailSender) sendViaSMTP(client *smtp.Client, msg *email.Email) error {
	// Set sender
	if err := client.Mail(s.config.Username); err != nil {
		return fmt.Errorf("MAIL command failed: %w", err)
	}

	// Set recipients
	for _, addr := range msg.To {
		if err := client.Rcpt(addr); err != nil {
			return fmt.Errorf("RCPT command failed for %s: %w", addr, err)
		}
	}

	// Send data
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("DATA command failed: %w", err)
	}
	defer writer.Close()

	data, err := msg.Bytes()
	if err != nil {
		return fmt.Errorf("failed to generate email bytes: %w", err)
	}

	if _, err := writer.Write(data); err != nil {
		return fmt.Errorf("failed to write email data: %w", err)
	}

	return nil
}

// resetAndReturnEmail resets the email instance for reuse and safely puts it back into the pool.
func (s *EmailSender) resetAndReturnEmail(msg *email.Email) {
	// Use Reset method if available in future, otherwise manually clean fields
	msg.From = ""
	msg.To = nil
	msg.Cc = nil
	msg.Bcc = nil
	msg.Subject = ""
	msg.Text = nil
	msg.HTML = nil
	msg.Attachments = nil
	if msg.Headers != nil {
		for k := range msg.Headers {
			delete(msg.Headers, k)
		}
	} else {
		msg.Headers = make(map[string][]string)
	}
	s.emailPool.Put(msg)
}

// Close releases all resources, especially underlying SMTP connections.
func (s *EmailSender) Close() error {
	return s.pool.Close()
}

// GetMetrics returns a snapshot of the current email metrics.
// func (s *EmailSender) GetMetrics() EmailMetrics {
// 	s.metrics.mu.RLock()
// 	defer s.metrics.mu.RUnlock()
// 	return *s.metrics
// }

func (m *EmailMetrics) recordSuccess(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	metrics.EmailSentTotal.Inc()
}

func (m *EmailMetrics) incrementFailed() {
	m.mu.Lock()
	defer m.mu.Unlock()
	metrics.EmailSendErrors.Inc()
}

func (m *EmailMetrics) incrementRetries() {
	m.mu.Lock()
	defer m.mu.Unlock()
	metrics.EmailRetries.Inc()
}

func (m *EmailMetrics) incrementRateLimited() {
	m.mu.Lock()
	defer m.mu.Unlock()
	metrics.EmailRateLimited.Inc()
}

// --- SMTP Connection Pool ---

type smtpPool struct {
	config EmailConfig
	conns  chan *smtp.Client
	mu     sync.Mutex
	auth   smtp.Auth
}

func newSMTPPool(config EmailConfig) (*smtpPool, error) {
	if config.PoolSize <= 0 {
		return nil, errors.New("PoolSize must be greater than 0")
	}
	return &smtpPool{
		config: config,
		conns:  make(chan *smtp.Client, config.PoolSize),
		auth:   smtp.PlainAuth("", config.Username, config.Password, config.SMTPHost),
	}, nil
}

func (p *smtpPool) Get(ctx context.Context) (*smtp.Client, error) {
	select {
	case client := <-p.conns:
		if err := client.Noop(); err == nil {
			return client, nil
		}
		_ = client.Close()
	default:
	}

	return p.createClient(ctx)
}

func (p *smtpPool) createClient(ctx context.Context) (*smtp.Client, error) {
	addr := net.JoinHostPort(p.config.SMTPHost, p.config.SMTPPort)

	// Support context cancellation on Dial
	dialer := &net.Dialer{}
	connCh := make(chan net.Conn, 1)
	errCh := make(chan error, 1)
	go func() {
		c, err := dialer.DialContext(ctx, "tcp", addr)
		if err != nil {
			errCh <- err
			return
		}
		connCh <- c
	}()

	var conn net.Conn
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-errCh:
		return nil, fmt.Errorf("dial failed: %w", err)
	case c := <-connCh:
		conn = c
	}

	client, err := smtp.NewClient(conn, p.config.SMTPHost)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("smtp.NewClient failed: %w", err)
	}

	// StartTLS if supported
	ok, _ := client.Extension("STARTTLS")
	if ok {
		tlsConfig := &tls.Config{
			ServerName:         p.config.SMTPHost,
			InsecureSkipVerify: false,
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			client.Close()
			return nil, fmt.Errorf("STARTTLS failed: %w", err)
		}
	}

	// Authenticate if the server supports AUTH
	if ok, _ := client.Extension("AUTH"); ok && p.auth != nil {
		if err := client.Auth(p.auth); err != nil {
			client.Close()
			return nil, fmt.Errorf("authentication failed: %w", err)
		}
	}

	return client, nil
}

// Put returns the smtp.Client to the pool if healthy, otherwise closes it.
func (p *smtpPool) Put(client *smtp.Client) {
	if err := client.Noop(); err != nil {
		client.Close()
		return
	}

	select {
	case p.conns <- client:
	default:
		client.Close()
	}
}

// Close closes all underlying smtp.Clients and the connection channel.
func (p *smtpPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	close(p.conns)
	for client := range p.conns {
		client.Close()
	}
	return nil
}
