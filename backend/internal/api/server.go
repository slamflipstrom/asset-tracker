package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"asset-tracker/internal/auth"
	"asset-tracker/internal/db"
	"github.com/go-chi/chi/v5"
)

type Server struct {
	DB       Store
	Verifier auth.Verifier
}

type Store interface {
	FetchPositionsForUser(ctx context.Context, userID string) ([]db.Position, error)
	ListLotsByUser(ctx context.Context, userID string) ([]db.Lot, error)
	InsertLot(ctx context.Context, lot db.Lot) (int64, error)
	UpdateLotForUser(ctx context.Context, userID string, lotID int64, quantity float64, unitCost float64, purchasedAt time.Time) (bool, error)
	DeleteLotForUser(ctx context.Context, userID string, lotID int64) (bool, error)
	SearchAssets(ctx context.Context, query string, assetType string, limit int) ([]db.Asset, error)
	ListAssetsByIDs(ctx context.Context, ids []int64) ([]db.Asset, error)
}

type contextKey string

const userIDContextKey contextKey = "userID"

func NewServer(store Store, verifier auth.Verifier) *Server {
	return &Server{DB: store, Verifier: verifier}
}

func (s *Server) Mount(r chi.Router) {
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(s.authMiddleware)
		r.Get("/positions", s.handleListPositions)
		r.Get("/lots", s.handleListLots)
		r.Post("/lots", s.handleCreateLot)
		r.Patch("/lots/{lotID}", s.handleUpdateLot)
		r.Delete("/lots/{lotID}", s.handleDeleteLot)
		r.Get("/assets/search", s.handleSearchAssets)
	})
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token == "" {
			writeError(w, http.StatusUnauthorized, "missing auth token")
			return
		}

		claims, err := s.Verifier.Verify(r.Context(), token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid auth token")
			return
		}

		ctx := context.WithValue(r.Context(), userIDContextKey, claims.Subject)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func userIDFromContext(ctx context.Context) string {
	userID, _ := ctx.Value(userIDContextKey).(string)
	return strings.TrimSpace(userID)
}

func extractToken(r *http.Request) string {
	authz := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(authz), "bearer ") {
		return strings.TrimSpace(authz[7:])
	}
	return ""
}

type apiError struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, apiError{Error: message})
}

func decodeJSONBody(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("request body must contain a single JSON object")
	}
	return nil
}

type positionResponse struct {
	AssetID      int64    `json:"asset_id"`
	Symbol       string   `json:"symbol"`
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	TotalQty     float64  `json:"total_qty"`
	AvgCost      float64  `json:"avg_cost"`
	CurrentPrice *float64 `json:"current_price"`
	UnrealizedPL *float64 `json:"unrealized_pl"`
}

func (s *Server) handleListPositions(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "missing user context")
		return
	}

	positions, err := s.DB.FetchPositionsForUser(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load positions")
		return
	}

	assetMap, err := s.loadAssetMapForPositions(r.Context(), positions)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load assets")
		return
	}

	response := make([]positionResponse, 0, len(positions))
	for _, position := range positions {
		asset := assetMap[position.AssetID]
		item := positionResponse{
			AssetID:      position.AssetID,
			Symbol:       asset.Symbol,
			Name:         asset.Name,
			Type:         string(asset.Type),
			TotalQty:     position.TotalQty,
			AvgCost:      position.AvgCost,
			CurrentPrice: nullFloatToPtr(position.CurrentPrice),
			UnrealizedPL: nullFloatToPtr(position.UnrealizedPL),
		}
		if item.Symbol == "" {
			item.Symbol = fmt.Sprintf("#%d", position.AssetID)
		}
		if item.Name == "" {
			item.Name = "Unknown asset"
		}
		if item.Type == "" {
			item.Type = string(db.AssetTypeCrypto)
		}
		response = append(response, item)
	}

	writeJSON(w, http.StatusOK, response)
}

type lotResponse struct {
	ID          int64   `json:"id"`
	AssetID     int64   `json:"asset_id"`
	Symbol      string  `json:"symbol"`
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	Quantity    float64 `json:"quantity"`
	UnitCost    float64 `json:"unit_cost"`
	PurchasedAt string  `json:"purchased_at"`
}

func (s *Server) handleListLots(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "missing user context")
		return
	}

	lots, err := s.DB.ListLotsByUser(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load lots")
		return
	}

	assetMap, err := s.loadAssetMapForLots(r.Context(), lots)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load assets")
		return
	}

	response := make([]lotResponse, 0, len(lots))
	for _, lot := range lots {
		asset := assetMap[lot.AssetID]
		item := lotResponse{
			ID:          lot.ID,
			AssetID:     lot.AssetID,
			Symbol:      asset.Symbol,
			Name:        asset.Name,
			Type:        string(asset.Type),
			Quantity:    lot.Quantity,
			UnitCost:    lot.UnitCost,
			PurchasedAt: lot.PurchasedAt.UTC().Format(time.RFC3339),
		}
		if item.Symbol == "" {
			item.Symbol = fmt.Sprintf("#%d", lot.AssetID)
		}
		if item.Name == "" {
			item.Name = "Unknown asset"
		}
		if item.Type == "" {
			item.Type = string(db.AssetTypeCrypto)
		}
		response = append(response, item)
	}

	writeJSON(w, http.StatusOK, response)
}

type createLotRequest struct {
	AssetID     int64   `json:"asset_id"`
	Quantity    float64 `json:"quantity"`
	UnitCost    float64 `json:"unit_cost"`
	PurchasedAt string  `json:"purchased_at"`
}

type createLotResponse struct {
	ID int64 `json:"id"`
}

func (s *Server) handleCreateLot(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "missing user context")
		return
	}

	var req createLotRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	purchasedAt, err := parseTimestamp(req.PurchasedAt)
	if err != nil {
		writeError(w, http.StatusBadRequest, "purchased_at must be RFC3339 or YYYY-MM-DD")
		return
	}
	if req.AssetID <= 0 {
		writeError(w, http.StatusBadRequest, "asset_id must be greater than 0")
		return
	}
	if req.Quantity <= 0 {
		writeError(w, http.StatusBadRequest, "quantity must be greater than 0")
		return
	}
	if req.UnitCost < 0 {
		writeError(w, http.StatusBadRequest, "unit_cost must be greater than or equal to 0")
		return
	}

	id, err := s.DB.InsertLot(r.Context(), db.Lot{
		UserID:      userID,
		AssetID:     req.AssetID,
		Quantity:    req.Quantity,
		UnitCost:    req.UnitCost,
		PurchasedAt: purchasedAt,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create lot")
		return
	}

	writeJSON(w, http.StatusCreated, createLotResponse{ID: id})
}

type updateLotRequest struct {
	Quantity    float64 `json:"quantity"`
	UnitCost    float64 `json:"unit_cost"`
	PurchasedAt string  `json:"purchased_at"`
}

func (s *Server) handleUpdateLot(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "missing user context")
		return
	}

	lotID, err := parseIDParam(r, "lotID")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid lot id")
		return
	}

	var req updateLotRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	purchasedAt, err := parseTimestamp(req.PurchasedAt)
	if err != nil {
		writeError(w, http.StatusBadRequest, "purchased_at must be RFC3339 or YYYY-MM-DD")
		return
	}
	if req.Quantity <= 0 {
		writeError(w, http.StatusBadRequest, "quantity must be greater than 0")
		return
	}
	if req.UnitCost < 0 {
		writeError(w, http.StatusBadRequest, "unit_cost must be greater than or equal to 0")
		return
	}

	updated, err := s.DB.UpdateLotForUser(r.Context(), userID, lotID, req.Quantity, req.UnitCost, purchasedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update lot")
		return
	}
	if !updated {
		writeError(w, http.StatusNotFound, "lot not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDeleteLot(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "missing user context")
		return
	}

	lotID, err := parseIDParam(r, "lotID")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid lot id")
		return
	}

	deleted, err := s.DB.DeleteLotForUser(r.Context(), userID, lotID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete lot")
		return
	}
	if !deleted {
		writeError(w, http.StatusNotFound, "lot not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type assetResponse struct {
	ID     int64  `json:"id"`
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
	Type   string `json:"type"`
}

func (s *Server) handleSearchAssets(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	assetType := strings.TrimSpace(r.URL.Query().Get("type"))
	if assetType != "" && assetType != string(db.AssetTypeCrypto) && assetType != string(db.AssetTypeStock) {
		writeError(w, http.StatusBadRequest, "type must be crypto or stock")
		return
	}

	limit := 20
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil || parsedLimit <= 0 {
			writeError(w, http.StatusBadRequest, "limit must be a positive integer")
			return
		}
		if parsedLimit > 100 {
			parsedLimit = 100
		}
		limit = parsedLimit
	}

	assets, err := s.DB.SearchAssets(r.Context(), query, assetType, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to search assets")
		return
	}

	response := make([]assetResponse, 0, len(assets))
	for _, asset := range assets {
		response = append(response, assetResponse{
			ID:     asset.ID,
			Symbol: asset.Symbol,
			Name:   asset.Name,
			Type:   string(asset.Type),
		})
	}

	writeJSON(w, http.StatusOK, response)
}

func parseIDParam(r *http.Request, key string) (int64, error) {
	value := strings.TrimSpace(chi.URLParam(r, key))
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("invalid %s", key)
	}
	return parsed, nil
}

func parseTimestamp(value string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, errors.New("empty timestamp")
	}

	if parsed, err := time.Parse(time.RFC3339, trimmed); err == nil {
		return parsed.UTC(), nil
	}
	if parsed, err := time.Parse("2006-01-02", trimmed); err == nil {
		return parsed.UTC(), nil
	}
	return time.Time{}, errors.New("invalid timestamp")
}

func nullFloatToPtr(value sql.NullFloat64) *float64 {
	if !value.Valid {
		return nil
	}
	out := value.Float64
	return &out
}

func (s *Server) loadAssetMapForLots(ctx context.Context, lots []db.Lot) (map[int64]db.Asset, error) {
	assetIDs := make([]int64, 0, len(lots))
	seen := make(map[int64]struct{}, len(lots))
	for _, lot := range lots {
		if _, exists := seen[lot.AssetID]; exists {
			continue
		}
		seen[lot.AssetID] = struct{}{}
		assetIDs = append(assetIDs, lot.AssetID)
	}

	assets, err := s.DB.ListAssetsByIDs(ctx, assetIDs)
	if err != nil {
		return nil, err
	}

	assetMap := make(map[int64]db.Asset, len(assets))
	for _, asset := range assets {
		assetMap[asset.ID] = asset
	}
	return assetMap, nil
}

func (s *Server) loadAssetMapForPositions(ctx context.Context, positions []db.Position) (map[int64]db.Asset, error) {
	assetIDs := make([]int64, 0, len(positions))
	seen := make(map[int64]struct{}, len(positions))
	for _, position := range positions {
		if _, exists := seen[position.AssetID]; exists {
			continue
		}
		seen[position.AssetID] = struct{}{}
		assetIDs = append(assetIDs, position.AssetID)
	}

	assets, err := s.DB.ListAssetsByIDs(ctx, assetIDs)
	if err != nil {
		return nil, err
	}

	assetMap := make(map[int64]db.Asset, len(assets))
	for _, asset := range assets {
		assetMap[asset.ID] = asset
	}
	return assetMap, nil
}
