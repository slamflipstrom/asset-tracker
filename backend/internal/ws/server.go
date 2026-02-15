package ws

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"asset-tracker/internal/auth"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type Server struct {
	Hub      *Hub
	Verifier auth.Verifier
}

type messageType string
type messageScope string
type messageAction string

const (
	messageTypeReady        messageType = "ready"
	messageTypeError        messageType = "error"
	messageTypeSubscribed   messageType = "subscribed"
	messageTypeUnsubscribed messageType = "unsubscribed"

	messageScopePortfolio messageScope = "portfolio"
	messageScopeAsset     messageScope = "asset"

	messageActionSubscribe   messageAction = "subscribe"
	messageActionUnsubscribe messageAction = "unsubscribe"
)

type clientMessage struct {
	Type    string `json:"type"`
	Scope   string `json:"scope"`
	AssetID int64  `json:"asset_id"`
}

type serverMessage struct {
	Type    string `json:"type"`
	Scope   string `json:"scope,omitempty"`
	AssetID int64  `json:"asset_id,omitempty"`
	UserID  string `json:"user_id,omitempty"`
	Message string `json:"message,omitempty"`
}

func NewServer(hub *Hub, verifier auth.Verifier) *Server {
	return &Server{Hub: hub, Verifier: verifier}
}

func (s *Server) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token == "" {
			http.Error(w, "missing auth token", http.StatusUnauthorized)
			return
		}

		claims, err := s.Verifier.Verify(r.Context(), token)
		if err != nil {
			http.Error(w, "invalid auth token", http.StatusUnauthorized)
			return
		}

		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true,
		})
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusInternalError, "server error")

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()
		conn.SetReadLimit(1 << 20)

		sessionID := claims.Subject + ":" + strconv.FormatInt(time.Now().UnixNano(), 10)
		if err := s.Hub.Add(sessionID, claims.Subject); err != nil {
			_ = wsjson.Write(ctx, conn, serverMessage{
				Type:    string(messageTypeError),
				Message: "failed to initialize websocket session",
			})
			_ = conn.Close(websocket.StatusInternalError, "failed to initialize session")
			return
		}
		defer s.Hub.Remove(sessionID)

		if err := wsjson.Write(ctx, conn, serverMessage{
			Type:   string(messageTypeReady),
			UserID: claims.Subject,
		}); err != nil {
			_ = conn.Close(websocket.StatusInternalError, "failed to write ready message")
			return
		}

		for {
			var msg clientMessage
			if err := wsjson.Read(ctx, conn, &msg); err != nil {
				status := websocket.CloseStatus(err)
				if status == websocket.StatusNormalClosure || status == websocket.StatusGoingAway {
					_ = conn.Close(websocket.StatusNormalClosure, "bye")
					return
				}
				_ = conn.Close(websocket.StatusUnsupportedData, "invalid websocket message")
				return
			}

			if err := s.handleMessage(ctx, conn, sessionID, msg); err != nil {
				_ = wsjson.Write(ctx, conn, serverMessage{
					Type:    string(messageTypeError),
					Message: err.Error(),
				})
			}
		}
	}
}

func (s *Server) handleMessage(ctx context.Context, conn *websocket.Conn, sessionID string, msg clientMessage) error {
	action := messageAction(strings.ToLower(strings.TrimSpace(msg.Type)))
	scope := messageScope(strings.ToLower(strings.TrimSpace(msg.Scope)))

	switch action {
	case messageActionSubscribe:
		switch scope {
		case messageScopePortfolio:
			s.Hub.SubscribePortfolio(sessionID)
		case messageScopeAsset:
			if msg.AssetID <= 0 {
				return fmt.Errorf("asset_id is required for asset subscriptions")
			}
			s.Hub.SubscribeAsset(sessionID, msg.AssetID)
		default:
			return fmt.Errorf("invalid scope: use portfolio or asset")
		}
		return wsjson.Write(ctx, conn, serverMessage{
			Type:    string(messageTypeSubscribed),
			Scope:   string(scope),
			AssetID: msg.AssetID,
		})
	case messageActionUnsubscribe:
		switch scope {
		case messageScopePortfolio:
			s.Hub.UnsubscribePortfolio(sessionID)
		case messageScopeAsset:
			if msg.AssetID <= 0 {
				return fmt.Errorf("asset_id is required for asset subscriptions")
			}
			s.Hub.UnsubscribeAsset(sessionID, msg.AssetID)
		default:
			return fmt.Errorf("invalid scope: use portfolio or asset")
		}
		return wsjson.Write(ctx, conn, serverMessage{
			Type:    string(messageTypeUnsubscribed),
			Scope:   string(scope),
			AssetID: msg.AssetID,
		})
	default:
		return fmt.Errorf("invalid message type: use subscribe or unsubscribe")
	}
}

func extractToken(r *http.Request) string {
	if authz := strings.TrimSpace(r.Header.Get("Authorization")); authz != "" {
		if strings.HasPrefix(strings.ToLower(authz), "bearer ") {
			return strings.TrimSpace(authz[7:])
		}
	}
	if token := strings.TrimSpace(r.URL.Query().Get("token")); token != "" {
		return token
	}
	return ""
}
