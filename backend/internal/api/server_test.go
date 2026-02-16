package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"asset-tracker/internal/auth"
	"asset-tracker/internal/db"
	"github.com/go-chi/chi/v5"
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

type mockStore struct {
	positions    []db.Position
	positionsErr error
	positionsUID string

	lots    []db.Lot
	lotsErr error
	lotsUID string

	assetsByID   map[int64]db.Asset
	listIDsErr   error
	listAssetIDs []int64

	insertLotID  int64
	insertLotErr error
	insertedLots []db.Lot

	updatedFound       bool
	updateErr          error
	updatedLotID       int64
	updatedUserID      string
	updatedQuantity    float64
	updatedUnitCost    float64
	updatedPurchasedAt time.Time

	deletedFound  bool
	deleteErr     error
	deletedLotID  int64
	deletedUserID string

	searchAssets []db.Asset
	searchErr    error
	searchQuery  string
	searchType   string
	searchLimit  int
}

func (m *mockStore) FetchPositionsForUser(ctx context.Context, userID string) ([]db.Position, error) {
	m.positionsUID = userID
	if m.positionsErr != nil {
		return nil, m.positionsErr
	}
	return m.positions, nil
}

func (m *mockStore) ListLotsByUser(ctx context.Context, userID string) ([]db.Lot, error) {
	m.lotsUID = userID
	if m.lotsErr != nil {
		return nil, m.lotsErr
	}
	return m.lots, nil
}

func (m *mockStore) InsertLot(ctx context.Context, lot db.Lot) (int64, error) {
	m.insertedLots = append(m.insertedLots, lot)
	if m.insertLotErr != nil {
		return 0, m.insertLotErr
	}
	if m.insertLotID == 0 {
		return 1, nil
	}
	return m.insertLotID, nil
}

func (m *mockStore) UpdateLotForUser(ctx context.Context, userID string, lotID int64, quantity float64, unitCost float64, purchasedAt time.Time) (bool, error) {
	m.updatedUserID = userID
	m.updatedLotID = lotID
	m.updatedQuantity = quantity
	m.updatedUnitCost = unitCost
	m.updatedPurchasedAt = purchasedAt
	if m.updateErr != nil {
		return false, m.updateErr
	}
	return m.updatedFound, nil
}

func (m *mockStore) DeleteLotForUser(ctx context.Context, userID string, lotID int64) (bool, error) {
	m.deletedUserID = userID
	m.deletedLotID = lotID
	if m.deleteErr != nil {
		return false, m.deleteErr
	}
	return m.deletedFound, nil
}

func (m *mockStore) SearchAssets(ctx context.Context, query string, assetType string, limit int) ([]db.Asset, error) {
	m.searchQuery = query
	m.searchType = assetType
	m.searchLimit = limit
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	return m.searchAssets, nil
}

func (m *mockStore) ListAssetsByIDs(ctx context.Context, ids []int64) ([]db.Asset, error) {
	m.listAssetIDs = append(m.listAssetIDs[:0], ids...)
	if m.listIDsErr != nil {
		return nil, m.listIDsErr
	}
	if len(ids) == 0 {
		return []db.Asset{}, nil
	}

	out := make([]db.Asset, 0, len(ids))
	for _, id := range ids {
		asset, ok := m.assetsByID[id]
		if !ok {
			continue
		}
		out = append(out, asset)
	}
	return out, nil
}

func newAPIRouter(store Store, verifier auth.Verifier) http.Handler {
	r := chi.NewRouter()
	NewServer(store, verifier).Mount(r)
	return r
}

func newRequest(t *testing.T, method, path, token string, body []byte) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func TestAPIMissingTokenUnauthorized(t *testing.T) {
	t.Parallel()

	router := newAPIRouter(&mockStore{}, mockVerifier{claims: auth.Claims{Subject: "user-1"}})
	res := httptest.NewRecorder()

	router.ServeHTTP(res, newRequest(t, http.MethodGet, "/api/v1/positions", "", nil))
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.Code)
	}
}

func TestAPIInvalidTokenUnauthorized(t *testing.T) {
	t.Parallel()

	router := newAPIRouter(&mockStore{}, mockVerifier{err: errors.New("bad token")})
	res := httptest.NewRecorder()

	router.ServeHTTP(res, newRequest(t, http.MethodGet, "/api/v1/positions", "bad", nil))
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.Code)
	}
}

func TestAPIListPositionsSuccess(t *testing.T) {
	t.Parallel()

	store := &mockStore{
		positions: []db.Position{{
			UserID:       "user-1",
			AssetID:      10,
			TotalQty:     1.5,
			AvgCost:      100,
			CurrentPrice: sql.NullFloat64{Float64: 150, Valid: true},
			UnrealizedPL: sql.NullFloat64{Float64: 75, Valid: true},
		}},
		assetsByID: map[int64]db.Asset{
			10: {ID: 10, Symbol: "BTC", Name: "Bitcoin", Type: db.AssetTypeCrypto},
		},
	}
	router := newAPIRouter(store, mockVerifier{claims: auth.Claims{Subject: "user-1"}})
	res := httptest.NewRecorder()

	router.ServeHTTP(res, newRequest(t, http.MethodGet, "/api/v1/positions", "good", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	if store.positionsUID != "user-1" {
		t.Fatalf("expected user id user-1, got %q", store.positionsUID)
	}

	var got []positionResponse
	if err := json.Unmarshal(res.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 position, got %d", len(got))
	}
	if got[0].Symbol != "BTC" {
		t.Fatalf("expected symbol BTC, got %q", got[0].Symbol)
	}
	if got[0].CurrentPrice == nil || *got[0].CurrentPrice != 150 {
		t.Fatalf("unexpected current_price: %+v", got[0].CurrentPrice)
	}
}

func TestAPICreateLotValidation(t *testing.T) {
	t.Parallel()

	store := &mockStore{}
	router := newAPIRouter(store, mockVerifier{claims: auth.Claims{Subject: "user-1"}})
	res := httptest.NewRecorder()

	body := []byte(`{"asset_id":1,"quantity":0,"unit_cost":10,"purchased_at":"2026-02-16"}`)
	router.ServeHTTP(res, newRequest(t, http.MethodPost, "/api/v1/lots", "good", body))

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
	if len(store.insertedLots) != 0 {
		t.Fatalf("expected no inserts, got %d", len(store.insertedLots))
	}
}

func TestAPICreateLotSuccess(t *testing.T) {
	t.Parallel()

	store := &mockStore{insertLotID: 99}
	router := newAPIRouter(store, mockVerifier{claims: auth.Claims{Subject: "user-1"}})
	res := httptest.NewRecorder()

	body := []byte(`{"asset_id":1,"quantity":0.25,"unit_cost":38000,"purchased_at":"2026-02-16"}`)
	router.ServeHTTP(res, newRequest(t, http.MethodPost, "/api/v1/lots", "good", body))

	if res.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.Code)
	}
	if len(store.insertedLots) != 1 {
		t.Fatalf("expected 1 insert, got %d", len(store.insertedLots))
	}
	inserted := store.insertedLots[0]
	if inserted.UserID != "user-1" {
		t.Fatalf("expected user-1 insert, got %q", inserted.UserID)
	}
	if inserted.AssetID != 1 || inserted.Quantity != 0.25 || inserted.UnitCost != 38000 {
		t.Fatalf("unexpected inserted lot: %+v", inserted)
	}

	var got createLotResponse
	if err := json.Unmarshal(res.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if got.ID != 99 {
		t.Fatalf("expected id 99, got %d", got.ID)
	}
}

func TestAPICreateLotStoreError(t *testing.T) {
	t.Parallel()

	store := &mockStore{insertLotErr: errors.New("boom")}
	router := newAPIRouter(store, mockVerifier{claims: auth.Claims{Subject: "user-1"}})
	res := httptest.NewRecorder()

	body := []byte(`{"asset_id":1,"quantity":0.25,"unit_cost":38000,"purchased_at":"2026-02-16"}`)
	router.ServeHTTP(res, newRequest(t, http.MethodPost, "/api/v1/lots", "good", body))

	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", res.Code)
	}
}

func TestAPIListPositionsStoreError(t *testing.T) {
	t.Parallel()

	store := &mockStore{positionsErr: errors.New("boom")}
	router := newAPIRouter(store, mockVerifier{claims: auth.Claims{Subject: "user-1"}})
	res := httptest.NewRecorder()

	router.ServeHTTP(res, newRequest(t, http.MethodGet, "/api/v1/positions", "good", nil))
	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", res.Code)
	}
}

func TestAPIListPositionsAssetLookupError(t *testing.T) {
	t.Parallel()

	store := &mockStore{
		positions:  []db.Position{{AssetID: 10, TotalQty: 1, AvgCost: 100}},
		listIDsErr: errors.New("boom"),
	}
	router := newAPIRouter(store, mockVerifier{claims: auth.Claims{Subject: "user-1"}})
	res := httptest.NewRecorder()

	router.ServeHTTP(res, newRequest(t, http.MethodGet, "/api/v1/positions", "good", nil))
	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", res.Code)
	}
}

func TestAPIListLotsSuccess(t *testing.T) {
	t.Parallel()

	purchasedAt := time.Date(2026, 2, 16, 12, 0, 0, 0, time.UTC)
	store := &mockStore{
		lots: []db.Lot{{
			ID:          7,
			UserID:      "user-1",
			AssetID:     10,
			Quantity:    1.25,
			UnitCost:    95,
			PurchasedAt: purchasedAt,
		}},
		assetsByID: map[int64]db.Asset{
			10: {ID: 10, Symbol: "BTC", Name: "Bitcoin", Type: db.AssetTypeCrypto},
		},
	}
	router := newAPIRouter(store, mockVerifier{claims: auth.Claims{Subject: "user-1"}})
	res := httptest.NewRecorder()

	router.ServeHTTP(res, newRequest(t, http.MethodGet, "/api/v1/lots", "good", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	if store.lotsUID != "user-1" {
		t.Fatalf("expected user id user-1, got %q", store.lotsUID)
	}

	var got []lotResponse
	if err := json.Unmarshal(res.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 lot, got %d", len(got))
	}
	if got[0].ID != 7 || got[0].Symbol != "BTC" {
		t.Fatalf("unexpected lot response: %+v", got[0])
	}
	if got[0].PurchasedAt != purchasedAt.Format(time.RFC3339) {
		t.Fatalf("unexpected purchased_at: %q", got[0].PurchasedAt)
	}
}

func TestAPIListLotsStoreError(t *testing.T) {
	t.Parallel()

	store := &mockStore{lotsErr: errors.New("boom")}
	router := newAPIRouter(store, mockVerifier{claims: auth.Claims{Subject: "user-1"}})
	res := httptest.NewRecorder()

	router.ServeHTTP(res, newRequest(t, http.MethodGet, "/api/v1/lots", "good", nil))
	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", res.Code)
	}
}

func TestAPIListLotsAssetLookupError(t *testing.T) {
	t.Parallel()

	store := &mockStore{
		lots:       []db.Lot{{ID: 1, AssetID: 10}},
		listIDsErr: errors.New("boom"),
	}
	router := newAPIRouter(store, mockVerifier{claims: auth.Claims{Subject: "user-1"}})
	res := httptest.NewRecorder()

	router.ServeHTTP(res, newRequest(t, http.MethodGet, "/api/v1/lots", "good", nil))
	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", res.Code)
	}
}

func TestAPIUpdateLotNotFound(t *testing.T) {
	t.Parallel()

	store := &mockStore{updatedFound: false}
	router := newAPIRouter(store, mockVerifier{claims: auth.Claims{Subject: "user-1"}})
	res := httptest.NewRecorder()

	body := []byte(`{"quantity":0.5,"unit_cost":39000,"purchased_at":"2026-02-16T00:00:00Z"}`)
	router.ServeHTTP(res, newRequest(t, http.MethodPatch, "/api/v1/lots/55", "good", body))

	if res.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.Code)
	}
}

func TestAPIUpdateLotSuccess(t *testing.T) {
	t.Parallel()

	store := &mockStore{updatedFound: true}
	router := newAPIRouter(store, mockVerifier{claims: auth.Claims{Subject: "user-1"}})
	res := httptest.NewRecorder()

	body := []byte(`{"quantity":0.5,"unit_cost":39000,"purchased_at":"2026-02-16"}`)
	router.ServeHTTP(res, newRequest(t, http.MethodPatch, "/api/v1/lots/55", "good", body))

	if res.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.Code)
	}
	if store.updatedLotID != 55 || store.updatedUserID != "user-1" {
		t.Fatalf("unexpected update args: lot=%d user=%q", store.updatedLotID, store.updatedUserID)
	}
	if store.updatedQuantity != 0.5 || store.updatedUnitCost != 39000 {
		t.Fatalf("unexpected update fields: quantity=%f unit_cost=%f", store.updatedQuantity, store.updatedUnitCost)
	}
	wantPurchasedAt := time.Date(2026, 2, 16, 0, 0, 0, 0, time.UTC)
	if !store.updatedPurchasedAt.Equal(wantPurchasedAt) {
		t.Fatalf("unexpected purchased_at: got=%s want=%s", store.updatedPurchasedAt, wantPurchasedAt)
	}
}

func TestAPIUpdateLotInvalidID(t *testing.T) {
	t.Parallel()

	store := &mockStore{updatedFound: true}
	router := newAPIRouter(store, mockVerifier{claims: auth.Claims{Subject: "user-1"}})
	res := httptest.NewRecorder()

	body := []byte(`{"quantity":0.5,"unit_cost":39000,"purchased_at":"2026-02-16"}`)
	router.ServeHTTP(res, newRequest(t, http.MethodPatch, "/api/v1/lots/bad", "good", body))

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
}

func TestAPIUpdateLotInvalidBody(t *testing.T) {
	t.Parallel()

	store := &mockStore{updatedFound: true}
	router := newAPIRouter(store, mockVerifier{claims: auth.Claims{Subject: "user-1"}})
	res := httptest.NewRecorder()

	body := []byte(`{"quantity":0.5,"unit_cost":39000,"purchased_at":"2026-02-16","extra":1}`)
	router.ServeHTTP(res, newRequest(t, http.MethodPatch, "/api/v1/lots/55", "good", body))

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
}

func TestAPIUpdateLotInvalidTimestamp(t *testing.T) {
	t.Parallel()

	store := &mockStore{updatedFound: true}
	router := newAPIRouter(store, mockVerifier{claims: auth.Claims{Subject: "user-1"}})
	res := httptest.NewRecorder()

	body := []byte(`{"quantity":0.5,"unit_cost":39000,"purchased_at":"not-a-date"}`)
	router.ServeHTTP(res, newRequest(t, http.MethodPatch, "/api/v1/lots/55", "good", body))

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
}

func TestAPIUpdateLotStoreError(t *testing.T) {
	t.Parallel()

	store := &mockStore{updateErr: errors.New("boom")}
	router := newAPIRouter(store, mockVerifier{claims: auth.Claims{Subject: "user-1"}})
	res := httptest.NewRecorder()

	body := []byte(`{"quantity":0.5,"unit_cost":39000,"purchased_at":"2026-02-16"}`)
	router.ServeHTTP(res, newRequest(t, http.MethodPatch, "/api/v1/lots/55", "good", body))

	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", res.Code)
	}
}

func TestAPIDeleteLotSuccess(t *testing.T) {
	t.Parallel()

	store := &mockStore{deletedFound: true}
	router := newAPIRouter(store, mockVerifier{claims: auth.Claims{Subject: "user-1"}})
	res := httptest.NewRecorder()

	router.ServeHTTP(res, newRequest(t, http.MethodDelete, "/api/v1/lots/55", "good", nil))
	if res.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.Code)
	}
	if store.deletedLotID != 55 || store.deletedUserID != "user-1" {
		t.Fatalf("unexpected delete args: lot=%d user=%q", store.deletedLotID, store.deletedUserID)
	}
}

func TestAPIDeleteLotNotFound(t *testing.T) {
	t.Parallel()

	store := &mockStore{deletedFound: false}
	router := newAPIRouter(store, mockVerifier{claims: auth.Claims{Subject: "user-1"}})
	res := httptest.NewRecorder()

	router.ServeHTTP(res, newRequest(t, http.MethodDelete, "/api/v1/lots/55", "good", nil))
	if res.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.Code)
	}
}

func TestAPIDeleteLotInvalidID(t *testing.T) {
	t.Parallel()

	store := &mockStore{deletedFound: true}
	router := newAPIRouter(store, mockVerifier{claims: auth.Claims{Subject: "user-1"}})
	res := httptest.NewRecorder()

	router.ServeHTTP(res, newRequest(t, http.MethodDelete, "/api/v1/lots/bad", "good", nil))
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
}

func TestAPIDeleteLotStoreError(t *testing.T) {
	t.Parallel()

	store := &mockStore{deleteErr: errors.New("boom")}
	router := newAPIRouter(store, mockVerifier{claims: auth.Claims{Subject: "user-1"}})
	res := httptest.NewRecorder()

	router.ServeHTTP(res, newRequest(t, http.MethodDelete, "/api/v1/lots/55", "good", nil))
	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", res.Code)
	}
}

func TestAPISearchAssetsValidation(t *testing.T) {
	t.Parallel()

	router := newAPIRouter(&mockStore{}, mockVerifier{claims: auth.Claims{Subject: "user-1"}})
	res := httptest.NewRecorder()

	router.ServeHTTP(res, newRequest(t, http.MethodGet, "/api/v1/assets/search?type=bad", "good", nil))
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
}

func TestAPISearchAssetsLimitValidation(t *testing.T) {
	t.Parallel()

	router := newAPIRouter(&mockStore{}, mockVerifier{claims: auth.Claims{Subject: "user-1"}})
	res := httptest.NewRecorder()

	router.ServeHTTP(res, newRequest(t, http.MethodGet, "/api/v1/assets/search?limit=bad", "good", nil))
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
}

func TestAPISearchAssetsLimitClampsToMax(t *testing.T) {
	t.Parallel()

	store := &mockStore{
		searchAssets: []db.Asset{{ID: 1, Symbol: "BTC", Name: "Bitcoin", Type: db.AssetTypeCrypto}},
	}
	router := newAPIRouter(store, mockVerifier{claims: auth.Claims{Subject: "user-1"}})
	res := httptest.NewRecorder()

	router.ServeHTTP(res, newRequest(t, http.MethodGet, "/api/v1/assets/search?q=bt&limit=500", "good", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	if store.searchLimit != 100 {
		t.Fatalf("expected limit to be clamped to 100, got %d", store.searchLimit)
	}
}

func TestAPISearchAssetsStoreError(t *testing.T) {
	t.Parallel()

	store := &mockStore{searchErr: errors.New("boom")}
	router := newAPIRouter(store, mockVerifier{claims: auth.Claims{Subject: "user-1"}})
	res := httptest.NewRecorder()

	router.ServeHTTP(res, newRequest(t, http.MethodGet, "/api/v1/assets/search?q=bt", "good", nil))
	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", res.Code)
	}
}

func TestAPISearchAssetsSuccess(t *testing.T) {
	t.Parallel()

	store := &mockStore{
		searchAssets: []db.Asset{{ID: 1, Symbol: "BTC", Name: "Bitcoin", Type: db.AssetTypeCrypto}},
	}
	router := newAPIRouter(store, mockVerifier{claims: auth.Claims{Subject: "user-1"}})
	res := httptest.NewRecorder()

	router.ServeHTTP(res, newRequest(t, http.MethodGet, "/api/v1/assets/search?q=bt&type=crypto&limit=5", "good", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	if store.searchQuery != "bt" || store.searchType != "crypto" || store.searchLimit != 5 {
		t.Fatalf("unexpected search args: q=%q type=%q limit=%d", store.searchQuery, store.searchType, store.searchLimit)
	}

	var got []assetResponse
	if err := json.Unmarshal(res.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(got) != 1 || got[0].Symbol != "BTC" {
		t.Fatalf("unexpected search response: %+v", got)
	}
}
