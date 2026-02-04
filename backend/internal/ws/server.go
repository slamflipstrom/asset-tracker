package ws

import (
	"context"
	"net/http"

	"nhooyr.io/websocket"
)

type Server struct {
	Hub *Hub
}

func NewServer(hub *Hub) *Server {
	return &Server{Hub: hub}
}

func (s *Server) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true,
		})
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusInternalError, "server error")

		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		_ = ctx
		// TODO: verify JWT, register user, and handle subscribe messages.

		conn.Close(websocket.StatusNormalClosure, "bye")
	}
}
