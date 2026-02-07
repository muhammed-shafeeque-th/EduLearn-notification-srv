package ports

import (
	"context"
	"net/http"

	entity "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/domain/entities"
	ws "github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/interfaces/websocket"
)

type WsHubAdaptor interface {
	Broadcast(userID string, payload any)
	GetMetrics() map[string]any
	NotifyInAppMessage(msg *entity.InAppWSMessage) error
	ServeWS(auth ws.AuthFunc) http.HandlerFunc
	Shutdown(ctx context.Context) error
}
