package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSupabaseVerifierVerifySuccess(t *testing.T) {
	t.Parallel()

	const token = "test-token"
	const apiKey = "service-key"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/v1/user" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+token {
			t.Fatalf("unexpected authorization header: %q", got)
		}
		if got := r.Header.Get("apikey"); got != apiKey {
			t.Fatalf("unexpected apikey header: %q", got)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Fatalf("unexpected accept header: %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"user-123","email":"person@example.com"}`))
	}))
	defer ts.Close()

	verifier := NewSupabaseVerifier(ts.URL+"/", apiKey)
	claims, err := verifier.Verify(context.Background(), token)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if claims.Subject != "user-123" || claims.Email != "person@example.com" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestSupabaseVerifierVerifyValidation(t *testing.T) {
	t.Parallel()

	verifier := NewSupabaseVerifier("https://example.com", "service-key")
	if _, err := verifier.Verify(context.Background(), ""); err == nil || !strings.Contains(err.Error(), "token is required") {
		t.Fatalf("expected token required error, got %v", err)
	}

	verifier = NewSupabaseVerifier("", "service-key")
	if _, err := verifier.Verify(context.Background(), "token"); err == nil || !strings.Contains(err.Error(), "supabase url is not configured") {
		t.Fatalf("expected missing url error, got %v", err)
	}

	verifier = NewSupabaseVerifier("https://example.com", "")
	if _, err := verifier.Verify(context.Background(), "token"); err == nil || !strings.Contains(err.Error(), "supabase api key is not configured") {
		t.Fatalf("expected missing key error, got %v", err)
	}
}

func TestSupabaseVerifierVerifyFailureResponses(t *testing.T) {
	t.Parallel()

	t.Run("non-200", func(t *testing.T) {
		t.Parallel()

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "bad token", http.StatusUnauthorized)
		}))
		defer ts.Close()

		verifier := NewSupabaseVerifier(ts.URL, "service-key")
		_, err := verifier.Verify(context.Background(), "token")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "status 401") || !strings.Contains(err.Error(), "bad token") {
			t.Fatalf("expected status/body in error, got %v", err)
		}
	})

	t.Run("invalid-json", func(t *testing.T) {
		t.Parallel()

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":`))
		}))
		defer ts.Close()

		verifier := NewSupabaseVerifier(ts.URL, "service-key")
		_, err := verifier.Verify(context.Background(), "token")
		if err == nil {
			t.Fatal("expected decode error, got nil")
		}
	})

	t.Run("missing-user-id", func(t *testing.T) {
		t.Parallel()

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"email":"missing-id@example.com"}`))
		}))
		defer ts.Close()

		verifier := NewSupabaseVerifier(ts.URL, "service-key")
		_, err := verifier.Verify(context.Background(), "token")
		if err == nil || !strings.Contains(err.Error(), "missing user id") {
			t.Fatalf("expected missing user id error, got %v", err)
		}
	})
}
