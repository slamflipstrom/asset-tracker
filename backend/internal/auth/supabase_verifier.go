package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type SupabaseVerifier struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

type supabaseUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

func NewSupabaseVerifier(baseURL, apiKey string) *SupabaseVerifier {
	return &SupabaseVerifier{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (v *SupabaseVerifier) Verify(ctx context.Context, token string) (Claims, error) {
	if strings.TrimSpace(token) == "" {
		return Claims{}, fmt.Errorf("token is required")
	}
	if v.baseURL == "" {
		return Claims{}, fmt.Errorf("supabase url is not configured")
	}
	if v.apiKey == "" {
		return Claims{}, fmt.Errorf("supabase api key is not configured")
	}

	url := v.baseURL + "/auth/v1/user"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Claims{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("apikey", v.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := v.client.Do(req)
	if err != nil {
		return Claims{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return Claims{}, fmt.Errorf("token verification failed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var user supabaseUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return Claims{}, err
	}
	if user.ID == "" {
		return Claims{}, fmt.Errorf("token verification failed: missing user id")
	}

	return Claims{Subject: user.ID, Email: user.Email}, nil
}
