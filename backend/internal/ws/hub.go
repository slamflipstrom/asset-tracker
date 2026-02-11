package ws

import (
	"fmt"
	"sync"
)

type Subscriber struct {
	SessionID string
	UserID    string
	Portfolio bool
	AssetIDs  map[int64]struct{}
}

type Hub struct {
	mu          sync.RWMutex
	subscribers map[string]*Subscriber
}

func NewHub() *Hub {
	return &Hub{subscribers: make(map[string]*Subscriber)}
}

func (h *Hub) Add(sessionID, userID string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if sessionID == "" || userID == "" {
		return fmt.Errorf("sessionID and userID are required")
	}
	if _, ok := h.subscribers[sessionID]; ok {
		return fmt.Errorf("session already exists")
	}
	h.subscribers[sessionID] = &Subscriber{
		SessionID: sessionID,
		UserID:    userID,
		AssetIDs:  map[int64]struct{}{},
	}
	return nil
}

func (h *Hub) Remove(sessionID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.subscribers, sessionID)
}

func (h *Hub) SubscribePortfolio(sessionID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	sub, ok := h.subscribers[sessionID]
	if !ok {
		return
	}
	sub.Portfolio = true
}

func (h *Hub) UnsubscribePortfolio(sessionID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	sub, ok := h.subscribers[sessionID]
	if !ok {
		return
	}
	sub.Portfolio = false
}

func (h *Hub) SubscribeAsset(sessionID string, assetID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	sub, ok := h.subscribers[sessionID]
	if !ok {
		return
	}
	sub.AssetIDs[assetID] = struct{}{}
}

func (h *Hub) UnsubscribeAsset(sessionID string, assetID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	sub, ok := h.subscribers[sessionID]
	if !ok {
		return
	}
	delete(sub.AssetIDs, assetID)
}
