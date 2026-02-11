package ws

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"asset-tracker/internal/auth"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type mockVerifier struct {
	claims auth.Claims
	err    error
}

func (m mockVerifier) Verify(ctx context.Context, token string) (auth.Claims, error) {
	if m.err != nil {
		return auth.Claims{}, m.err
	}
	if strings.TrimSpace(token) == "" {
		return auth.Claims{}, errors.New("missing token")
	}
	return m.claims, nil
}

func TestWSMissingTokenUnauthorized(t *testing.T) {
	t.Parallel()

	hub := NewHub()
	srv := NewServer(hub, mockVerifier{claims: auth.Claims{Subject: "user-1"}})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Fatalf("http get failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestWSInvalidTokenUnauthorized(t *testing.T) {
	t.Parallel()

	hub := NewHub()
	srv := NewServer(hub, mockVerifier{err: errors.New("invalid")})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "?token=bad")
	if err != nil {
		t.Fatalf("http get failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestWSSubscribeAndUnsubscribeFlow(t *testing.T) {
	t.Parallel()

	hub := NewHub()
	srv := NewServer(hub, mockVerifier{claims: auth.Claims{Subject: "user-1"}})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "?token=good"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("websocket dial failed: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "test done")

	var ready serverMessage
	if err := wsjson.Read(ctx, conn, &ready); err != nil {
		t.Fatalf("failed reading ready message: %v", err)
	}
	if ready.Type != "ready" || ready.UserID != "user-1" {
		t.Fatalf("unexpected ready message: %+v", ready)
	}

	if err := wsjson.Write(ctx, conn, clientMessage{Type: "subscribe", Scope: "portfolio"}); err != nil {
		t.Fatalf("failed writing subscribe portfolio: %v", err)
	}
	var subscribed serverMessage
	if err := wsjson.Read(ctx, conn, &subscribed); err != nil {
		t.Fatalf("failed reading subscribed message: %v", err)
	}
	if subscribed.Type != "subscribed" || subscribed.Scope != "portfolio" {
		t.Fatalf("unexpected subscribed response: %+v", subscribed)
	}

	sessionID := singleSessionID(t, hub)
	if !hub.subscribers[sessionID].Portfolio {
		t.Fatal("expected portfolio subscription to be enabled")
	}

	if err := wsjson.Write(ctx, conn, clientMessage{Type: "subscribe", Scope: "asset", AssetID: 42}); err != nil {
		t.Fatalf("failed writing subscribe asset: %v", err)
	}
	if err := wsjson.Read(ctx, conn, &subscribed); err != nil {
		t.Fatalf("failed reading asset subscribed message: %v", err)
	}
	if _, ok := hub.subscribers[sessionID].AssetIDs[42]; !ok {
		t.Fatal("expected asset 42 to be subscribed")
	}

	if err := wsjson.Write(ctx, conn, clientMessage{Type: "unsubscribe", Scope: "asset", AssetID: 42}); err != nil {
		t.Fatalf("failed writing unsubscribe asset: %v", err)
	}
	var unsubscribed serverMessage
	if err := wsjson.Read(ctx, conn, &unsubscribed); err != nil {
		t.Fatalf("failed reading unsubscribed message: %v", err)
	}
	if unsubscribed.Type != "unsubscribed" || unsubscribed.Scope != "asset" {
		t.Fatalf("unexpected unsubscribed response: %+v", unsubscribed)
	}
	if _, ok := hub.subscribers[sessionID].AssetIDs[42]; ok {
		t.Fatal("expected asset 42 to be removed from subscriptions")
	}

	if err := wsjson.Write(ctx, conn, clientMessage{Type: "unsubscribe", Scope: "portfolio"}); err != nil {
		t.Fatalf("failed writing unsubscribe portfolio: %v", err)
	}
	if err := wsjson.Read(ctx, conn, &unsubscribed); err != nil {
		t.Fatalf("failed reading portfolio unsubscribed message: %v", err)
	}
	if hub.subscribers[sessionID].Portfolio {
		t.Fatal("expected portfolio subscription to be disabled")
	}
}

func TestWSInvalidAssetSubscribeMessage(t *testing.T) {
	t.Parallel()

	hub := NewHub()
	srv := NewServer(hub, mockVerifier{claims: auth.Claims{Subject: "user-1"}})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "?token=good"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("websocket dial failed: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "test done")

	var ready serverMessage
	if err := wsjson.Read(ctx, conn, &ready); err != nil {
		t.Fatalf("failed reading ready message: %v", err)
	}

	if err := wsjson.Write(ctx, conn, clientMessage{Type: "subscribe", Scope: "asset"}); err != nil {
		t.Fatalf("failed writing invalid subscribe message: %v", err)
	}
	var got serverMessage
	if err := wsjson.Read(ctx, conn, &got); err != nil {
		t.Fatalf("failed reading error message: %v", err)
	}
	if got.Type != "error" || !strings.Contains(got.Message, "asset_id") {
		t.Fatalf("expected asset_id error, got %+v", got)
	}
}

func singleSessionID(t *testing.T, hub *Hub) string {
	t.Helper()

	hub.mu.RLock()
	defer hub.mu.RUnlock()
	if len(hub.subscribers) != 1 {
		t.Fatalf("expected exactly one subscriber session, got %d", len(hub.subscribers))
	}
	for sessionID := range hub.subscribers {
		return sessionID
	}
	return ""
}
