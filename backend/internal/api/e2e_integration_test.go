package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"asset-tracker/internal/auth"
	"asset-tracker/internal/db"
	"github.com/go-chi/chi/v5"
)

const defaultAPIE2EDBURL = "postgresql://postgres:postgres@127.0.0.1:54322/postgres"

type e2eCreateLotResponse struct {
	ID int64 `json:"id"`
}

type e2eLotResponse struct {
	ID       int64   `json:"id"`
	AssetID  int64   `json:"asset_id"`
	Quantity float64 `json:"quantity"`
	UnitCost float64 `json:"unit_cost"`
}

type e2ePositionResponse struct {
	AssetID  int64   `json:"asset_id"`
	TotalQty float64 `json:"total_qty"`
	AvgCost  float64 `json:"avg_cost"`
}

type e2eAssetResponse struct {
	ID     int64  `json:"id"`
	Symbol string `json:"symbol"`
}

func TestAPIE2EHappyPath(t *testing.T) {
	database := mustOpenAPIE2EDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	userID := randomUUID(t)
	email := fmt.Sprintf("api-e2e-%s@example.com", randomHex(t, 4))
	assetSymbol := fmt.Sprintf("E2E_%s", randomHex(t, 4))
	assetName := "E2E Asset"

	mustInsertAuthUser(t, ctx, database, userID, email)
	t.Cleanup(func() { cleanupAuthUser(t, context.Background(), database, userID) })

	assetID := mustInsertStockAsset(t, ctx, database, assetSymbol, assetName)
	t.Cleanup(func() { cleanupAsset(t, context.Background(), database, assetID) })

	const serviceKey = "test-service-key"
	const validToken = "valid-token"
	authCalls := 0

	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/v1/user" {
			http.NotFound(w, r)
			return
		}
		authCalls++

		if got := r.Header.Get("apikey"); got != serviceKey {
			http.Error(w, "invalid apikey", http.StatusUnauthorized)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+validToken {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"id":    userID,
			"email": email,
		})
	}))
	defer authServer.Close()

	verifier := auth.NewSupabaseVerifier(authServer.URL, serviceKey)
	router := chi.NewRouter()
	NewServer(database, verifier).Mount(router)
	apiServer := httptest.NewServer(router)
	defer apiServer.Close()

	unauthorizedRes := doRequest(t, apiServer.Client(), http.MethodGet, apiServer.URL+"/api/v1/lots", "", nil)
	if unauthorizedRes.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized status 401, got %d", unauthorizedRes.StatusCode)
	}

	searchPath := fmt.Sprintf("%s/api/v1/assets/search?q=%s&limit=5", apiServer.URL, url.QueryEscape(assetSymbol))
	searchRes := doRequest(t, apiServer.Client(), http.MethodGet, searchPath, validToken, nil)
	if searchRes.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on search, got %d", searchRes.StatusCode)
	}
	var assets []e2eAssetResponse
	decodeJSON(t, searchRes, &assets)
	if len(assets) == 0 {
		t.Fatal("expected at least one asset in search response")
	}
	if assets[0].ID != assetID || assets[0].Symbol != assetSymbol {
		t.Fatalf("unexpected asset payload: %+v", assets[0])
	}

	createPayload := []byte(fmt.Sprintf(`{"asset_id":%d,"quantity":2.5,"unit_cost":123.45,"purchased_at":"2026-02-16"}`, assetID))
	createRes := doRequest(t, apiServer.Client(), http.MethodPost, apiServer.URL+"/api/v1/lots", validToken, createPayload)
	if createRes.StatusCode != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d", createRes.StatusCode)
	}
	var created e2eCreateLotResponse
	decodeJSON(t, createRes, &created)
	if created.ID <= 0 {
		t.Fatalf("expected created lot id > 0, got %d", created.ID)
	}

	lotsRes := doRequest(t, apiServer.Client(), http.MethodGet, apiServer.URL+"/api/v1/lots", validToken, nil)
	if lotsRes.StatusCode != http.StatusOK {
		t.Fatalf("expected list lots status 200, got %d", lotsRes.StatusCode)
	}
	var lots []e2eLotResponse
	decodeJSON(t, lotsRes, &lots)
	if len(lots) != 1 {
		t.Fatalf("expected one lot after create, got %d", len(lots))
	}
	if lots[0].ID != created.ID || lots[0].AssetID != assetID {
		t.Fatalf("unexpected listed lot: %+v", lots[0])
	}

	positionsRes := doRequest(t, apiServer.Client(), http.MethodGet, apiServer.URL+"/api/v1/positions", validToken, nil)
	if positionsRes.StatusCode != http.StatusOK {
		t.Fatalf("expected list positions status 200, got %d", positionsRes.StatusCode)
	}
	var positions []e2ePositionResponse
	decodeJSON(t, positionsRes, &positions)
	if len(positions) != 1 {
		t.Fatalf("expected one position after create, got %d", len(positions))
	}
	if positions[0].AssetID != assetID || positions[0].TotalQty != 2.5 || positions[0].AvgCost != 123.45 {
		t.Fatalf("unexpected position payload: %+v", positions[0])
	}

	updatePayload := []byte(`{"quantity":3,"unit_cost":100,"purchased_at":"2026-02-17"}`)
	updateURL := fmt.Sprintf("%s/api/v1/lots/%d", apiServer.URL, created.ID)
	updateRes := doRequest(t, apiServer.Client(), http.MethodPatch, updateURL, validToken, updatePayload)
	if updateRes.StatusCode != http.StatusNoContent {
		t.Fatalf("expected update status 204, got %d", updateRes.StatusCode)
	}

	updatedLotsRes := doRequest(t, apiServer.Client(), http.MethodGet, apiServer.URL+"/api/v1/lots", validToken, nil)
	if updatedLotsRes.StatusCode != http.StatusOK {
		t.Fatalf("expected updated list lots status 200, got %d", updatedLotsRes.StatusCode)
	}
	var updatedLots []e2eLotResponse
	decodeJSON(t, updatedLotsRes, &updatedLots)
	if len(updatedLots) != 1 || updatedLots[0].Quantity != 3 || updatedLots[0].UnitCost != 100 {
		t.Fatalf("unexpected updated lot payload: %+v", updatedLots)
	}

	deleteRes := doRequest(t, apiServer.Client(), http.MethodDelete, updateURL, validToken, nil)
	if deleteRes.StatusCode != http.StatusNoContent {
		t.Fatalf("expected delete status 204, got %d", deleteRes.StatusCode)
	}

	finalLotsRes := doRequest(t, apiServer.Client(), http.MethodGet, apiServer.URL+"/api/v1/lots", validToken, nil)
	if finalLotsRes.StatusCode != http.StatusOK {
		t.Fatalf("expected final list lots status 200, got %d", finalLotsRes.StatusCode)
	}
	var finalLots []e2eLotResponse
	decodeJSON(t, finalLotsRes, &finalLots)
	if len(finalLots) != 0 {
		t.Fatalf("expected zero lots after delete, got %d", len(finalLots))
	}

	if authCalls < 7 {
		t.Fatalf("expected verifier to call auth service multiple times, got %d", authCalls)
	}
}

func doRequest(t *testing.T, client *http.Client, method string, url string, token string, body []byte) *http.Response {
	t.Helper()

	req, err := http.NewRequestWithContext(context.Background(), method, url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create request (%s %s): %v", method, url, err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed (%s %s): %v", method, url, err)
	}
	t.Cleanup(func() { _ = res.Body.Close() })
	return res
}

func decodeJSON[T any](t *testing.T, res *http.Response, dst *T) {
	t.Helper()
	if err := json.NewDecoder(res.Body).Decode(dst); err != nil {
		t.Fatalf("failed decoding json response: %v", err)
	}
}

func mustOpenAPIE2EDB(t *testing.T) *db.DB {
	t.Helper()

	dbURL := os.Getenv("ASSET_TRACKER_TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = defaultAPIE2EDBURL
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	database, err := db.New(ctx, dbURL)
	if err != nil {
		t.Skipf("skipping API E2E integration test: %v", err)
	}
	if err := database.Pool().Ping(ctx); err != nil {
		database.Close()
		t.Skipf("skipping API E2E integration test: %v", err)
	}
	t.Cleanup(database.Close)
	return database
}

func mustInsertAuthUser(t *testing.T, ctx context.Context, database *db.DB, userID string, email string) {
	t.Helper()
	_, err := database.Pool().Exec(ctx, `
		insert into auth.users (id, email, aud, role, encrypted_password, created_at, updated_at, is_sso_user, is_anonymous)
		values ($1::uuid, $2, 'authenticated', 'authenticated', '', now(), now(), false, false)
	`, userID, email)
	if err != nil {
		t.Fatalf("failed to insert auth user: %v", err)
	}
}

func cleanupAuthUser(t *testing.T, ctx context.Context, database *db.DB, userID string) {
	t.Helper()
	_, _ = database.Pool().Exec(ctx, `delete from auth.users where id = $1::uuid`, userID)
}

func mustInsertStockAsset(t *testing.T, ctx context.Context, database *db.DB, symbol string, name string) int64 {
	t.Helper()
	var assetID int64
	err := database.Pool().QueryRow(ctx, `
		insert into public.assets (symbol, type, name)
		values ($1, 'stock', $2)
		returning id
	`, symbol, name).Scan(&assetID)
	if err != nil {
		t.Fatalf("failed to insert stock asset: %v", err)
	}
	return assetID
}

func cleanupAsset(t *testing.T, ctx context.Context, database *db.DB, assetID int64) {
	t.Helper()
	_, _ = database.Pool().Exec(ctx, `delete from public.assets where id = $1`, assetID)
}

func randomUUID(t *testing.T) string {
	t.Helper()
	b := randomBytes(t, 16)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func randomHex(t *testing.T, n int) string {
	t.Helper()
	return fmt.Sprintf("%x", randomBytes(t, n))
}

func randomBytes(t *testing.T, n int) []byte {
	t.Helper()
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("failed to read random bytes: %v", err)
	}
	return b
}
