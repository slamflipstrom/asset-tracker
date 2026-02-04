package ws

import (
	"sync"
)

type Subscriber struct {
	UserID   string
	AssetIDs map[int64]struct{}
}

type Hub struct {
	mu          sync.RWMutex
	subscribers map[string]*Subscriber
}

func NewHub() *Hub {
	return &Hub{subscribers: make(map[string]*Subscriber)}
}

func (h *Hub) Add(userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.subscribers[userID]; !ok {
		h.subscribers[userID] = &Subscriber{UserID: userID, AssetIDs: map[int64]struct{}{}}
	}
}

func (h *Hub) Remove(userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.subscribers, userID)
}

func (h *Hub) SubscribeAsset(userID string, assetID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	sub, ok := h.subscribers[userID]
	if !ok {
		sub = &Subscriber{UserID: userID, AssetIDs: map[int64]struct{}{}}
		h.subscribers[userID] = sub
	}
	sub.AssetIDs[assetID] = struct{}{}
}

func (h *Hub) UnsubscribeAsset(userID string, assetID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	sub, ok := h.subscribers[userID]
	if !ok {
		return
	}
	delete(sub.AssetIDs, assetID)
}
