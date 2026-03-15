package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	apierrors "b2b-orders-api/internal/errors"
	"b2b-orders-api/internal/service/sync"
)

// SyncHandler handles sync-related HTTP requests
type SyncHandler struct {
	productSyncer *sync.ProductSyncer
	clientSyncer  *sync.ClientSyncer
	logger        *slog.Logger
}

// NewSyncHandler creates a new sync handler
func NewSyncHandler(
	productSyncer *sync.ProductSyncer,
	clientSyncer *sync.ClientSyncer,
	logger *slog.Logger,
) *SyncHandler {
	return &SyncHandler{
		productSyncer: productSyncer,
		clientSyncer:  clientSyncer,
		logger:        logger,
	}
}

type productSyncResponse struct {
	TotalProducts  int `json:"total_products"`
	SyncedProducts int `json:"synced_products"`
	FailedProducts int `json:"failed_products"`
	TotalImages    int `json:"total_images"`
	SyncedImages   int `json:"synced_images"`
	FailedImages   int `json:"failed_images"`
}

type clientSyncResponse struct {
	TotalClients  int `json:"total_clients"`
	SyncedClients int `json:"synced_clients"`
	FailedClients int `json:"failed_clients"`
}

// SyncProducts triggers a manual product sync
func (h *SyncHandler) SyncProducts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	h.logger.Info("manual product sync triggered")

	result, err := h.productSyncer.Sync(ctx)
	if err != nil {
		h.logger.Error("product sync failed", "error", err)
		apierrors.Internal(w)
		return
	}

	h.logger.Info("product sync completed",
		"total", result.TotalProducts,
		"synced", result.SyncedProducts,
		"failed", result.FailedProducts,
	)

	response := productSyncResponse{
		TotalProducts:  result.TotalProducts,
		SyncedProducts: result.SyncedProducts,
		FailedProducts: result.FailedProducts,
		TotalImages:    result.TotalImages,
		SyncedImages:   result.SyncedImages,
		FailedImages:   result.FailedImages,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// SyncClients triggers a manual client sync
func (h *SyncHandler) SyncClients(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	h.logger.Info("manual client sync triggered")

	result, err := h.clientSyncer.Sync(ctx)
	if err != nil {
		h.logger.Error("client sync failed", "error", err)
		apierrors.Internal(w)
		return
	}

	h.logger.Info("client sync completed",
		"total", result.TotalClients,
		"synced", result.SyncedClients,
		"failed", result.FailedClients,
	)

	response := clientSyncResponse{
		TotalClients:  result.TotalClients,
		SyncedClients: result.SyncedClients,
		FailedClients: result.FailedClients,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
