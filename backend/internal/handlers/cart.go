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
	"b2b-orders-api/internal/middleware"
	"b2b-orders-api/internal/service/auth"
	"b2b-orders-api/internal/service/cart"
)

// CartHandler handles cart-related HTTP requests
type CartHandler struct {
	cartService *cart.Service
	authService *auth.Service
	logger      *slog.Logger
}

// NewCartHandler creates a new cart handler
func NewCartHandler(
	cartService *cart.Service,
	authService *auth.Service,
	logger *slog.Logger,
) *CartHandler {
	return &CartHandler{
		cartService: cartService,
		authService: authService,
		logger:      logger,
	}
}

// Request types
type addItemRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type setItemsRequest struct {
	Items []addItemRequest `json:"items"`
}

type updateQuantityRequest struct {
	Quantity int `json:"quantity"`
}

type updateNotesRequest struct {
	Notes string `json:"notes"`
}

// GetCart returns the current cart with enriched details
func (h *CartHandler) GetCart(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	client, err := h.getClientFromContext(r)
	if err != nil {
		h.logger.Error("failed to get client from context", "error", err)
		apierrors.Unauthorized(w, "unable to identify client")
		return
	}

	cartResp, err := h.cartService.GetDraftWithDetails(ctx, client.ID)
	if err != nil {
		if errors.Is(err, cart.ErrNoDraft) {
			apierrors.NotFound(w, "no active cart found")
			return
		}
		h.logger.Error("failed to get cart", "error", err)
		apierrors.Internal(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cartResp)
}

// CreateCart creates a new cart or returns the existing one
func (h *CartHandler) CreateCart(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	client, err := h.getClientFromContext(r)
	if err != nil {
		h.logger.Error("failed to get client from context", "error", err)
		apierrors.Unauthorized(w, "unable to identify client")
		return
	}

	_, isNew, err := h.cartService.GetOrCreateDraft(ctx, client.ID)
	if err != nil {
		h.logger.Error("failed to create cart", "error", err)
		apierrors.Internal(w)
		return
	}

	// Return enriched response
	cartResp, err := h.cartService.GetDraftWithDetails(ctx, client.ID)
	if err != nil {
		h.logger.Error("failed to get cart details", "error", err)
		apierrors.Internal(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if isNew {
		w.WriteHeader(http.StatusCreated)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	json.NewEncoder(w).Encode(cartResp)
}

// DiscardCart deletes the current cart
func (h *CartHandler) DiscardCart(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	client, err := h.getClientFromContext(r)
	if err != nil {
		h.logger.Error("failed to get client from context", "error", err)
		apierrors.Unauthorized(w, "unable to identify client")
		return
	}

	err = h.cartService.DiscardDraft(ctx, client.ID)
	if err != nil {
		if errors.Is(err, cart.ErrNoDraft) {
			apierrors.NotFound(w, "no active cart found")
			return
		}
		h.logger.Error("failed to discard cart", "error", err)
		apierrors.Internal(w)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AddItem adds an item to the cart
func (h *CartHandler) AddItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	client, err := h.getClientFromContext(r)
	if err != nil {
		h.logger.Error("failed to get client from context", "error", err)
		apierrors.Unauthorized(w, "unable to identify client")
		return
	}

	var req addItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	productID, err := uuid.Parse(req.ProductID)
	if err != nil {
		apierrors.BadRequest(w, "invalid product_id")
		return
	}

	err = h.cartService.AddItem(ctx, client.ID, productID, req.Quantity)
	if err != nil {
		if errors.Is(err, cart.ErrProductNotFound) {
			apierrors.BadRequest(w, "product not found or inactive")
			return
		}
		if errors.Is(err, cart.ErrInvalidQuantity) {
			apierrors.BadRequest(w, "quantity must be positive")
			return
		}
		h.logger.Error("failed to add item to cart", "error", err)
		apierrors.Internal(w)
		return
	}

	// Return updated cart
	cartResp, err := h.cartService.GetDraftWithDetails(ctx, client.ID)
	if err != nil {
		h.logger.Error("failed to get cart details", "error", err)
		apierrors.Internal(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cartResp)
}

// SetItems replaces all items in the cart
func (h *CartHandler) SetItems(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	client, err := h.getClientFromContext(r)
	if err != nil {
		h.logger.Error("failed to get client from context", "error", err)
		apierrors.Unauthorized(w, "unable to identify client")
		return
	}

	var req setItemsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	// Convert request items to service items
	items := make([]cart.ItemRequest, 0, len(req.Items))
	for _, item := range req.Items {
		productID, err := uuid.Parse(item.ProductID)
		if err != nil {
			apierrors.BadRequest(w, "invalid product_id: "+item.ProductID)
			return
		}
		items = append(items, cart.ItemRequest{
			ProductID: productID,
			Quantity:  item.Quantity,
		})
	}

	err = h.cartService.SetItems(ctx, client.ID, items)
	if err != nil {
		if errors.Is(err, cart.ErrProductNotFound) {
			apierrors.BadRequest(w, err.Error())
			return
		}
		if errors.Is(err, cart.ErrInvalidQuantity) {
			apierrors.BadRequest(w, err.Error())
			return
		}
		h.logger.Error("failed to set cart items", "error", err)
		apierrors.Internal(w)
		return
	}

	// Return updated cart
	cartResp, err := h.cartService.GetDraftWithDetails(ctx, client.ID)
	if err != nil {
		h.logger.Error("failed to get cart details", "error", err)
		apierrors.Internal(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cartResp)
}

// UpdateItemQuantity updates the quantity of a specific item
func (h *CartHandler) UpdateItemQuantity(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	client, err := h.getClientFromContext(r)
	if err != nil {
		h.logger.Error("failed to get client from context", "error", err)
		apierrors.Unauthorized(w, "unable to identify client")
		return
	}

	productIDStr := chi.URLParam(r, "product_id")
	productID, err := uuid.Parse(productIDStr)
	if err != nil {
		apierrors.BadRequest(w, "invalid product_id")
		return
	}

	var req updateQuantityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	err = h.cartService.UpdateItemQuantity(ctx, client.ID, productID, req.Quantity)
	if err != nil {
		if errors.Is(err, cart.ErrNoDraft) {
			apierrors.NotFound(w, "no active cart found")
			return
		}
		if errors.Is(err, cart.ErrProductNotFound) {
			apierrors.NotFound(w, "item not found in cart")
			return
		}
		if errors.Is(err, cart.ErrInvalidQuantity) {
			apierrors.BadRequest(w, "quantity must be non-negative")
			return
		}
		h.logger.Error("failed to update item quantity", "error", err)
		apierrors.Internal(w)
		return
	}

	// Return updated cart
	cartResp, err := h.cartService.GetDraftWithDetails(ctx, client.ID)
	if err != nil {
		h.logger.Error("failed to get cart details", "error", err)
		apierrors.Internal(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cartResp)
}

// RemoveItem removes an item from the cart
func (h *CartHandler) RemoveItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	client, err := h.getClientFromContext(r)
	if err != nil {
		h.logger.Error("failed to get client from context", "error", err)
		apierrors.Unauthorized(w, "unable to identify client")
		return
	}

	productIDStr := chi.URLParam(r, "product_id")
	productID, err := uuid.Parse(productIDStr)
	if err != nil {
		apierrors.BadRequest(w, "invalid product_id")
		return
	}

	err = h.cartService.RemoveItem(ctx, client.ID, productID)
	if err != nil {
		if errors.Is(err, cart.ErrNoDraft) {
			apierrors.NotFound(w, "no active cart found")
			return
		}
		if errors.Is(err, cart.ErrProductNotFound) {
			apierrors.NotFound(w, "item not found in cart")
			return
		}
		h.logger.Error("failed to remove item from cart", "error", err)
		apierrors.Internal(w)
		return
	}

	// Return updated cart
	cartResp, err := h.cartService.GetDraftWithDetails(ctx, client.ID)
	if err != nil {
		h.logger.Error("failed to get cart details", "error", err)
		apierrors.Internal(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cartResp)
}

// UpdateNotes updates the notes on the cart
func (h *CartHandler) UpdateNotes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	client, err := h.getClientFromContext(r)
	if err != nil {
		h.logger.Error("failed to get client from context", "error", err)
		apierrors.Unauthorized(w, "unable to identify client")
		return
	}

	var req updateNotesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.BadRequest(w, "invalid request body")
		return
	}

	err = h.cartService.UpdateNotes(ctx, client.ID, req.Notes)
	if err != nil {
		if errors.Is(err, cart.ErrNoDraft) {
			apierrors.NotFound(w, "no active cart found")
			return
		}
		h.logger.Error("failed to update cart notes", "error", err)
		apierrors.Internal(w)
		return
	}

	// Return updated cart
	cartResp, err := h.cartService.GetDraftWithDetails(ctx, client.ID)
	if err != nil {
		h.logger.Error("failed to get cart details", "error", err)
		apierrors.Internal(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cartResp)
}

// Submit submits the cart as a pending order
func (h *CartHandler) Submit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	client, err := h.getClientFromContext(r)
	if err != nil {
		h.logger.Error("failed to get client from context", "error", err)
		apierrors.Unauthorized(w, "unable to identify client")
		return
	}

	order, err := h.cartService.Submit(ctx, client.ID)
	if err != nil {
		if errors.Is(err, cart.ErrNoDraft) {
			apierrors.NotFound(w, "no active cart found")
			return
		}
		if errors.Is(err, cart.ErrEmptyCart) {
			apierrors.BadRequest(w, "cannot submit empty cart")
			return
		}
		if errors.Is(err, cart.ErrProductNotFound) {
			apierrors.BadRequest(w, err.Error())
			return
		}
		if errors.Is(err, cart.ErrInsufficientStock) {
			apierrors.WriteJSON(w, http.StatusConflict, apierrors.New(apierrors.CodeInsufficientStock, err.Error()))
			return
		}
		h.logger.Error("failed to submit cart", "error", err)
		apierrors.Internal(w)
		return
	}

	// Convert to order response
	resp := toOrderResponse(order)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func (h *CartHandler) getClientFromContext(r *http.Request) (*domain.Client, error) {
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
