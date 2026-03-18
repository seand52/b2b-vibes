package cart

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"b2b-orders-api/internal/domain"
	"b2b-orders-api/internal/repository"
)

var (
	ErrNoDraft           = errors.New("no draft order found")
	ErrNotDraft          = errors.New("order is not a draft")
	ErrEmptyCart         = errors.New("cannot submit empty cart")
	ErrProductNotFound   = errors.New("product not found")
	ErrInsufficientStock = errors.New("insufficient stock")
	ErrInvalidQuantity   = errors.New("invalid quantity")
)

const (
	DefaultTaxRate = 21.0 // Spain VAT
)

// CartItemResponse represents an item with enriched product data
type CartItemResponse struct {
	ProductID        uuid.UUID `json:"product_id"`
	ProductName      string    `json:"product_name"`
	ProductSKU       string    `json:"product_sku"`
	Quantity         int       `json:"quantity"`
	UnitPrice        float64   `json:"unit_price"`
	LineTotal        float64   `json:"line_total"`
	StockAvailable   int       `json:"stock_available"`
	MinOrderQuantity int       `json:"min_order_quantity"`
	InStock          bool      `json:"in_stock"`
}

// CartSummary contains pricing totals
type CartSummary struct {
	Subtotal   float64 `json:"subtotal"`
	TaxRate    float64 `json:"tax_rate"`
	TaxAmount  float64 `json:"tax_amount"`
	Total      float64 `json:"total"`
	ItemCount  int     `json:"item_count"`
	TotalUnits int     `json:"total_units"`
}

// CartResponse is the enriched cart view
type CartResponse struct {
	ID        uuid.UUID          `json:"id"`
	Status    domain.OrderStatus `json:"status"`
	Notes     string             `json:"notes,omitempty"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
	Items     []CartItemResponse `json:"items"`
	Summary   CartSummary        `json:"summary"`
}

// ItemRequest represents a request to add/update an item
type ItemRequest struct {
	ProductID uuid.UUID
	Quantity  int
}

// Service handles cart/draft order business logic
type Service struct {
	orderRepo   repository.OrderRepository
	productRepo repository.ProductRepository
	logger      *slog.Logger
}

// NewService creates a new cart service
func NewService(
	orderRepo repository.OrderRepository,
	productRepo repository.ProductRepository,
	logger *slog.Logger,
) *Service {
	return &Service{
		orderRepo:   orderRepo,
		productRepo: productRepo,
		logger:      logger,
	}
}

// GetOrCreateDraft gets the client's existing draft order or creates a new one.
// Returns the draft and a boolean indicating if it was newly created.
func (s *Service) GetOrCreateDraft(ctx context.Context, clientID uuid.UUID) (*domain.Order, bool, error) {
	// Try to get existing draft
	draft, err := s.orderRepo.GetDraftByClientID(ctx, clientID)
	if err == nil {
		return draft, false, nil
	}

	if !errors.Is(err, repository.ErrNotFound) {
		return nil, false, fmt.Errorf("looking up draft: %w", err)
	}

	// Create new draft order
	draft = &domain.Order{
		ID:       uuid.New(),
		ClientID: clientID,
		Status:   domain.OrderStatusDraft,
		Items:    []domain.OrderItem{},
	}

	if err := s.orderRepo.Create(ctx, draft); err != nil {
		return nil, false, fmt.Errorf("creating draft order: %w", err)
	}

	s.logger.Info("draft order created",
		"order_id", draft.ID,
		"client_id", clientID,
	)

	return draft, true, nil
}

// GetDraftWithDetails gets the cart with enriched product info and pricing
func (s *Service) GetDraftWithDetails(ctx context.Context, clientID uuid.UUID) (*CartResponse, error) {
	draft, err := s.orderRepo.GetDraftByClientID(ctx, clientID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNoDraft
		}
		return nil, fmt.Errorf("getting draft: %w", err)
	}

	if !draft.IsDraft() {
		return nil, ErrNotDraft
	}

	// Enrich items with product data
	items := make([]CartItemResponse, 0, len(draft.Items))
	var subtotal float64
	var totalUnits int

	for _, item := range draft.Items {
		product, err := s.productRepo.GetByID(ctx, item.ProductID)
		if err != nil {
			s.logger.Warn("product not found in cart",
				"order_id", draft.ID,
				"product_id", item.ProductID,
				"error", err,
			)
			continue // Skip items for products that no longer exist
		}

		lineTotal := product.Price * float64(item.Quantity)
		items = append(items, CartItemResponse{
			ProductID:        item.ProductID,
			ProductName:      product.Name,
			ProductSKU:       product.SKU,
			Quantity:         item.Quantity,
			UnitPrice:        product.Price,
			LineTotal:        lineTotal,
			StockAvailable:   product.StockQuantity,
			MinOrderQuantity: product.MinOrderQuantity,
			InStock:          product.HasStock(item.Quantity),
		})

		subtotal += lineTotal
		totalUnits += item.Quantity
	}

	// Calculate tax and total
	taxAmount := subtotal * (DefaultTaxRate / 100.0)
	total := subtotal + taxAmount

	summary := CartSummary{
		Subtotal:   subtotal,
		TaxRate:    DefaultTaxRate,
		TaxAmount:  taxAmount,
		Total:      total,
		ItemCount:  len(items),
		TotalUnits: totalUnits,
	}

	return &CartResponse{
		ID:        draft.ID,
		Status:    draft.Status,
		Notes:     draft.Notes,
		CreatedAt: draft.CreatedAt,
		UpdatedAt: draft.UpdatedAt,
		Items:     items,
		Summary:   summary,
	}, nil
}

// AddItem adds an item to the cart or updates quantity if already exists
func (s *Service) AddItem(ctx context.Context, clientID, productID uuid.UUID, quantity int) error {
	if quantity <= 0 {
		return ErrInvalidQuantity
	}

	// Validate product exists and is active
	product, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrProductNotFound
		}
		return fmt.Errorf("looking up product: %w", err)
	}

	if !product.IsActive {
		return ErrProductNotFound
	}

	// Get or create draft
	draft, _, err := s.GetOrCreateDraft(ctx, clientID)
	if err != nil {
		return err
	}

	// Check if item already exists
	found := false
	for i := range draft.Items {
		if draft.Items[i].ProductID == productID {
			draft.Items[i].Quantity += quantity
			found = true
			break
		}
	}

	if !found {
		// Add new item
		draft.Items = append(draft.Items, domain.OrderItem{
			ID:        uuid.New(),
			OrderID:   draft.ID,
			ProductID: productID,
			Quantity:  quantity,
		})
	}

	if err := s.orderRepo.UpdateItems(ctx, draft.ID, draft.Items); err != nil {
		return fmt.Errorf("updating cart items: %w", err)
	}

	s.logger.Info("item added to cart",
		"order_id", draft.ID,
		"product_id", productID,
		"quantity", quantity,
	)

	return nil
}

// UpdateItemQuantity updates the quantity of a specific item in the cart
func (s *Service) UpdateItemQuantity(ctx context.Context, clientID, productID uuid.UUID, quantity int) error {
	if quantity < 0 {
		return ErrInvalidQuantity
	}

	draft, err := s.orderRepo.GetDraftByClientID(ctx, clientID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrNoDraft
		}
		return fmt.Errorf("getting draft: %w", err)
	}

	if !draft.IsDraft() {
		return ErrNotDraft
	}

	// Find and update item (or remove if quantity is 0)
	found := false
	newItems := make([]domain.OrderItem, 0, len(draft.Items))

	for _, item := range draft.Items {
		if item.ProductID == productID {
			found = true
			if quantity > 0 {
				item.Quantity = quantity
				newItems = append(newItems, item)
			}
			// If quantity is 0, skip (remove item)
		} else {
			newItems = append(newItems, item)
		}
	}

	if !found {
		return ErrProductNotFound
	}

	if err := s.orderRepo.UpdateItems(ctx, draft.ID, newItems); err != nil {
		return fmt.Errorf("updating cart items: %w", err)
	}

	s.logger.Info("cart item quantity updated",
		"order_id", draft.ID,
		"product_id", productID,
		"quantity", quantity,
	)

	return nil
}

// RemoveItem removes an item from the cart
func (s *Service) RemoveItem(ctx context.Context, clientID, productID uuid.UUID) error {
	return s.UpdateItemQuantity(ctx, clientID, productID, 0)
}

// SetItems replaces all items in the cart
func (s *Service) SetItems(ctx context.Context, clientID uuid.UUID, items []ItemRequest) error {
	// Validate all items first
	for _, item := range items {
		if item.Quantity <= 0 {
			return fmt.Errorf("%w: product %s", ErrInvalidQuantity, item.ProductID)
		}

		product, err := s.productRepo.GetByID(ctx, item.ProductID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return fmt.Errorf("%w: %s", ErrProductNotFound, item.ProductID)
			}
			return fmt.Errorf("looking up product %s: %w", item.ProductID, err)
		}

		if !product.IsActive {
			return fmt.Errorf("%w: %s", ErrProductNotFound, item.ProductID)
		}
	}

	// Get or create draft
	draft, _, err := s.GetOrCreateDraft(ctx, clientID)
	if err != nil {
		return err
	}

	// Build new items list
	orderItems := make([]domain.OrderItem, 0, len(items))
	for _, item := range items {
		orderItems = append(orderItems, domain.OrderItem{
			ID:        uuid.New(),
			OrderID:   draft.ID,
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		})
	}

	if err := s.orderRepo.UpdateItems(ctx, draft.ID, orderItems); err != nil {
		return fmt.Errorf("updating cart items: %w", err)
	}

	s.logger.Info("cart items replaced",
		"order_id", draft.ID,
		"item_count", len(items),
	)

	return nil
}

// UpdateNotes updates the notes on the draft order
func (s *Service) UpdateNotes(ctx context.Context, clientID uuid.UUID, notes string) error {
	draft, err := s.orderRepo.GetDraftByClientID(ctx, clientID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrNoDraft
		}
		return fmt.Errorf("getting draft: %w", err)
	}

	if !draft.IsDraft() {
		return ErrNotDraft
	}

	if err := s.orderRepo.UpdateNotes(ctx, draft.ID, notes); err != nil {
		return fmt.Errorf("updating notes: %w", err)
	}

	s.logger.Info("cart notes updated",
		"order_id", draft.ID,
	)

	return nil
}

// Submit validates the cart and submits it as a pending order
func (s *Service) Submit(ctx context.Context, clientID uuid.UUID) (*domain.Order, error) {
	draft, err := s.orderRepo.GetDraftByClientID(ctx, clientID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNoDraft
		}
		return nil, fmt.Errorf("getting draft: %w", err)
	}

	if !draft.IsDraft() {
		return nil, ErrNotDraft
	}

	if len(draft.Items) == 0 {
		return nil, ErrEmptyCart
	}

	// Validate each item: stock, min quantities, and capture prices
	finalItems := make([]domain.OrderItem, 0, len(draft.Items))

	for _, item := range draft.Items {
		product, err := s.productRepo.GetByID(ctx, item.ProductID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil, fmt.Errorf("%w: %s", ErrProductNotFound, item.ProductID)
			}
			return nil, fmt.Errorf("looking up product %s: %w", item.ProductID, err)
		}

		if !product.IsActive {
			return nil, fmt.Errorf("%w: %s (inactive)", ErrProductNotFound, product.Name)
		}

		if !product.HasStock(item.Quantity) {
			return nil, fmt.Errorf("%w: %s (requested: %d, available: %d)",
				ErrInsufficientStock, product.Name, item.Quantity, product.StockQuantity)
		}

		if item.Quantity < product.MinOrderQuantity {
			return nil, fmt.Errorf("minimum order quantity for %s is %d (requested: %d)",
				product.Name, product.MinOrderQuantity, item.Quantity)
		}

		// Capture unit price and line total
		unitPrice := product.Price
		lineTotal := unitPrice * float64(item.Quantity)

		finalItems = append(finalItems, domain.OrderItem{
			ID:        item.ID,
			OrderID:   draft.ID,
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			UnitPrice: &unitPrice,
			LineTotal: &lineTotal,
		})
	}

	// Submit the draft (updates items with prices and status to pending)
	if err := s.orderRepo.SubmitDraft(ctx, draft.ID, finalItems); err != nil {
		return nil, fmt.Errorf("submitting draft: %w", err)
	}

	s.logger.Info("cart submitted",
		"order_id", draft.ID,
		"client_id", clientID,
		"item_count", len(finalItems),
	)

	// Return updated order
	return s.orderRepo.GetByID(ctx, draft.ID)
}

// DiscardDraft deletes the draft order
func (s *Service) DiscardDraft(ctx context.Context, clientID uuid.UUID) error {
	draft, err := s.orderRepo.GetDraftByClientID(ctx, clientID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrNoDraft
		}
		return fmt.Errorf("getting draft: %w", err)
	}

	if !draft.IsDraft() {
		return ErrNotDraft
	}

	if err := s.orderRepo.UpdateStatus(ctx, draft.ID, domain.OrderStatusCancelled); err != nil {
		return fmt.Errorf("discarding draft: %w", err)
	}

	s.logger.Info("draft order discarded",
		"order_id", draft.ID,
		"client_id", clientID,
	)

	return nil
}
