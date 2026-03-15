package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"b2b-orders-api/internal/domain"
	apierrors "b2b-orders-api/internal/errors"
	"b2b-orders-api/internal/repository"
	"b2b-orders-api/internal/service/order"
)

// AdminHandler handles admin-related HTTP requests
type AdminHandler struct {
	orderService *order.Service
	clientRepo   repository.ClientRepository
	logger       *slog.Logger
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(
	orderService *order.Service,
	clientRepo repository.ClientRepository,
	logger *slog.Logger,
) *AdminHandler {
	return &AdminHandler{
		orderService: orderService,
		clientRepo:   clientRepo,
		logger:       logger,
	}
}

// approveRequest is the request body for approving an order
type approveRequest struct {
	ApprovedBy string `json:"approved_by"`
}

// rejectRequest is the request body for rejecting an order
type rejectRequest struct {
	Reason string `json:"reason"`
}

// ListOrders returns all orders (admin view)
func (h *AdminHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse filters
	statusParam := r.URL.Query().Get("status")
	filter := repository.OrderFilter{}
	if statusParam != "" {
		filter.Status = domain.OrderStatus(statusParam)
	}

	orders, err := h.orderService.ListAll(ctx, filter)
	if err != nil {
		h.logger.Error("failed to list orders", "error", err)
		apierrors.Internal(w)
		return
	}

	response := make([]orderResponse, 0, len(orders))
	for i := range orders {
		response = append(response, toOrderResponse(&orders[i]))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetOrder returns a single order (admin view)
func (h *AdminHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	orderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		apierrors.BadRequest(w, "invalid order ID")
		return
	}

	o, err := h.orderService.GetByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			apierrors.NotFound(w, "order not found")
			return
		}
		h.logger.Error("failed to get order", "error", err)
		apierrors.Internal(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toOrderResponse(o))
}

// ApproveOrder approves an order and creates a Holded invoice
func (h *AdminHandler) ApproveOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	orderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		apierrors.BadRequest(w, "invalid order ID")
		return
	}

	var req approveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	if req.ApprovedBy == "" {
		apierrors.BadRequest(w, "approved_by is required")
		return
	}

	approvedOrder, err := h.orderService.Approve(ctx, orderID, req.ApprovedBy)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			apierrors.NotFound(w, "order not found")
			return
		}
		if errors.Is(err, order.ErrOrderNotPending) {
			apierrors.WriteJSON(w, http.StatusConflict, apierrors.New(apierrors.CodeInvalidStatus, "order is not pending"))
			return
		}
		h.logger.Error("failed to approve order", "order_id", orderID, "error", err)
		apierrors.Internal(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toOrderResponse(approvedOrder))
}

// RejectOrder rejects an order
func (h *AdminHandler) RejectOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	orderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		apierrors.BadRequest(w, "invalid order ID")
		return
	}

	var req rejectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	if req.Reason == "" {
		apierrors.BadRequest(w, "reason is required")
		return
	}

	err = h.orderService.Reject(ctx, orderID, req.Reason)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			apierrors.NotFound(w, "order not found")
			return
		}
		if errors.Is(err, order.ErrOrderNotPending) {
			apierrors.WriteJSON(w, http.StatusConflict, apierrors.New(apierrors.CodeInvalidStatus, "order is not pending"))
			return
		}
		h.logger.Error("failed to reject order", "order_id", orderID, "error", err)
		apierrors.Internal(w)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// clientResponse is the API response for a client
type clientResponse struct {
	ID              uuid.UUID       `json:"id"`
	HoldedID        string          `json:"holded_id"`
	Email           string          `json:"email"`
	CompanyName     string          `json:"company_name"`
	ContactName     string          `json:"contact_name,omitempty"`
	Phone           string          `json:"phone,omitempty"`
	VATType         domain.VATType  `json:"vat_type,omitempty"`
	VATNumber       string          `json:"vat_number,omitempty"`
	BillingAddress  *domain.Address `json:"billing_address,omitempty"`
	ShippingAddress *domain.Address `json:"shipping_address,omitempty"`
	IsActive        bool            `json:"is_active"`
	IsLinked        bool            `json:"is_linked"`
	CreatedAt       string          `json:"created_at"`
}

func toClientResponse(c *domain.Client) clientResponse {
	return clientResponse{
		ID:              c.ID,
		HoldedID:        c.HoldedID,
		Email:           c.Email,
		CompanyName:     c.CompanyName,
		ContactName:     c.ContactName,
		Phone:           c.Phone,
		VATType:         c.VATType,
		VATNumber:       c.VATNumber,
		BillingAddress:  c.BillingAddress,
		ShippingAddress: c.ShippingAddress,
		IsActive:        c.IsActive,
		IsLinked:        c.IsLinked(),
		CreatedAt:       c.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// ListClients returns all clients (admin view)
func (h *AdminHandler) ListClients(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse filters
	filter := repository.ClientFilter{}
	if search := r.URL.Query().Get("search"); search != "" {
		filter.SearchTerm = search
	}
	if activeOnly := r.URL.Query().Get("active"); activeOnly == "true" {
		active := true
		filter.IsActive = &active
	}

	clients, err := h.clientRepo.List(ctx, filter)
	if err != nil {
		h.logger.Error("failed to list clients", "error", err)
		apierrors.Internal(w)
		return
	}

	response := make([]clientResponse, 0, len(clients))
	for i := range clients {
		response = append(response, toClientResponse(&clients[i]))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetClient returns a single client (admin view)
func (h *AdminHandler) GetClient(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	clientID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		apierrors.BadRequest(w, "invalid client ID")
		return
	}

	client, err := h.clientRepo.GetByID(ctx, clientID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			apierrors.NotFound(w, "client not found")
			return
		}
		h.logger.Error("failed to get client", "error", err)
		apierrors.Internal(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toClientResponse(client))
}
