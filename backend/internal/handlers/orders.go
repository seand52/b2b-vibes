package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"b2b-orders-api/internal/domain"
	apierrors "b2b-orders-api/internal/errors"
	"b2b-orders-api/internal/middleware"
	"b2b-orders-api/internal/repository"
	"b2b-orders-api/internal/service/auth"
	"b2b-orders-api/internal/service/order"
)

// OrderHandler handles order-related HTTP requests
type OrderHandler struct {
	orderService *order.Service
	authService  *auth.Service
	logger       *slog.Logger
}

// NewOrderHandler creates a new order handler
func NewOrderHandler(
	orderService *order.Service,
	authService *auth.Service,
	logger *slog.Logger,
) *OrderHandler {
	return &OrderHandler{
		orderService: orderService,
		authService:  authService,
		logger:       logger,
	}
}

// createOrderRequest is the request body for creating an order
type createOrderRequest struct {
	Items []orderItemRequest `json:"items"`
	Notes string             `json:"notes,omitempty"`
}

type orderItemRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

// orderResponse is the API response for an order
type orderResponse struct {
	ID              uuid.UUID           `json:"id"`
	Status          domain.OrderStatus  `json:"status"`
	Notes           string              `json:"notes,omitempty"`
	HoldedInvoiceID *string             `json:"holded_invoice_id,omitempty"`
	ApprovedAt      *time.Time          `json:"approved_at,omitempty"`
	RejectedAt      *time.Time          `json:"rejected_at,omitempty"`
	RejectionReason *string             `json:"rejection_reason,omitempty"`
	CreatedAt       time.Time           `json:"created_at"`
	Items           []orderItemResponse `json:"items"`
}

type orderItemResponse struct {
	ProductID uuid.UUID `json:"product_id"`
	Quantity  int       `json:"quantity"`
}

func toOrderResponse(o *domain.Order) orderResponse {
	resp := orderResponse{
		ID:              o.ID,
		Status:          o.Status,
		Notes:           o.Notes,
		HoldedInvoiceID: o.HoldedInvoiceID,
		ApprovedAt:      o.ApprovedAt,
		RejectedAt:      o.RejectedAt,
		RejectionReason: o.RejectionReason,
		CreatedAt:       o.CreatedAt,
		Items:           make([]orderItemResponse, 0, len(o.Items)),
	}

	for _, item := range o.Items {
		resp.Items = append(resp.Items, orderItemResponse{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		})
	}

	return resp
}

// Create creates a new order
func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get client from auth
	client, err := h.getClientFromContext(r)
	if err != nil {
		h.logger.Error("failed to get client from context", "error", err)
		apierrors.Unauthorized(w, "unable to identify client")
		return
	}

	var req createOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	if len(req.Items) == 0 {
		apierrors.BadRequest(w, "order must have at least one item")
		return
	}

	// Convert request items
	items := make([]order.OrderItemRequest, 0, len(req.Items))
	for _, item := range req.Items {
		productID, err := uuid.Parse(item.ProductID)
		if err != nil {
			apierrors.BadRequest(w, "invalid product_id: "+item.ProductID)
			return
		}
		items = append(items, order.OrderItemRequest{
			ProductID: productID,
			Quantity:  item.Quantity,
		})
	}

	createReq := order.CreateOrderRequest{
		ClientID: client.ID,
		Items:    items,
		Notes:    req.Notes,
	}

	newOrder, err := h.orderService.Create(ctx, createReq)
	if err != nil {
		if errors.Is(err, order.ErrProductNotFound) {
			apierrors.BadRequest(w, err.Error())
			return
		}
		if errors.Is(err, order.ErrInsufficientStock) {
			apierrors.WriteJSON(w, http.StatusConflict, apierrors.New(apierrors.CodeInsufficientStock, err.Error()))
			return
		}
		h.logger.Error("failed to create order", "error", err)
		apierrors.Internal(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(toOrderResponse(newOrder))
}

// List returns the client's orders
func (h *OrderHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	client, err := h.getClientFromContext(r)
	if err != nil {
		apierrors.Unauthorized(w, "unable to identify client")
		return
	}

	// Parse status filter
	statusParam := r.URL.Query().Get("status")
	filter := repository.OrderFilter{}
	if statusParam != "" {
		filter.Status = domain.OrderStatus(statusParam)
	}

	orders, err := h.orderService.ListByClient(ctx, client.ID, filter)
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

// Get returns a single order
func (h *OrderHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	client, err := h.getClientFromContext(r)
	if err != nil {
		apierrors.Unauthorized(w, "unable to identify client")
		return
	}

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

	// Verify ownership
	if o.ClientID != client.ID {
		apierrors.NotFound(w, "order not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toOrderResponse(o))
}

// Cancel cancels a pending order
func (h *OrderHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	client, err := h.getClientFromContext(r)
	if err != nil {
		apierrors.Unauthorized(w, "unable to identify client")
		return
	}

	orderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		apierrors.BadRequest(w, "invalid order ID")
		return
	}

	err = h.orderService.Cancel(ctx, orderID, client.ID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			apierrors.NotFound(w, "order not found")
			return
		}
		if errors.Is(err, order.ErrOrderNotPending) {
			apierrors.WriteJSON(w, http.StatusConflict, apierrors.New(apierrors.CodeInvalidStatus, "order cannot be cancelled"))
			return
		}
		h.logger.Error("failed to cancel order", "error", err)
		apierrors.Internal(w)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *OrderHandler) getClientFromContext(r *http.Request) (*domain.Client, error) {
	ctx := r.Context()

	auth0ID, err := middleware.GetAuth0ID(ctx)
	if err != nil {
		return nil, err
	}

	email, err := middleware.GetEmail(ctx)
	if err != nil {
		return nil, err
	}

	return h.authService.GetOrLinkClient(ctx, auth0ID, email)
}
