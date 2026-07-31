package websocket_interface

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/internal/application/ports"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/pkg/logger"
	"github.com/muhammed-shafeeque-th/EduLearn-notification-srv/pkg/ws"
)

type JWTAuthenticator struct {
	secretKey []byte
	logger    ports.LoggerService
}

func NewJWTAuthenticator(secretKey string, logger ports.LoggerService) *JWTAuthenticator {
	return &JWTAuthenticator{
		secretKey: []byte(secretKey),
		logger:    logger,
	}
}

func (a *JWTAuthenticator) Authenticate(r *http.Request) (string, error) {
	// Try to get token from Authorization header first
	token := a.extractTokenFromQuery(r)

	// Fallback to query parameter (for browser WebSocket connections)
	if token == "" {
		token = r.URL.Query().Get("token")
	}

	if token == "" {
		return "", errors.New("no token provided")
	}

	userID, err := a.validateToken(token)
	if err != nil {
		a.logger.Warn("Invalid token",
			logger.Error(err),
			logger.String("remote_addr", r.RemoteAddr),
		)
		return "", err
	}

	return userID, nil
}

/*
func (a *JWTAuthenticator) extractTokenFromHeader(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}

	return parts[1]
}
*/

// Now extracts token only from query params (token param)
func (a *JWTAuthenticator) extractTokenFromQuery(r *http.Request) string {
	return r.URL.Query().Get("token")
}

func (a *JWTAuthenticator) validateToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.secretKey, nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return "", errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid token claims")
	}

	userID, ok := claims["user_id"].(string)
	if !ok || userID == "" {
		// Try alternative claim names
		if sub, ok := claims["sub"].(string); ok {
			userID = sub
		} else if id, ok := claims["id"].(string); ok {
			userID = id
		} else {
			return "", errors.New("user_id not found in token")
		}
	}

	return userID, nil
}

// Helper to create auth function for Hub
func CreateAuthFunc(secretKey string, logger ports.LoggerService) ws.AuthFunc {
	authenticator := NewJWTAuthenticator(secretKey, logger)
	return authenticator.Authenticate
}
