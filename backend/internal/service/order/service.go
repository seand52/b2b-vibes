package order

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"b2b-orders-api/internal/domain"
	"b2b-orders-api/internal/clients/holded"
	"b2b-orders-api/internal/repository"
)

var (
	ErrInvalidStatus     = errors.New("invalid order status transition")
	ErrOrderNotPending   = errors.New("order is not pending")
	ErrInsufficientStock = errors.New("insufficient stock")
	ErrProductNotFound   = errors.New("product not found")
)

// HoldedClient defines the Holded operations needed by the order service
type HoldedClient interface {
	CreateInvoice(ctx context.Context, req *holded.CreateInvoiceRequest) (*holded.Invoice, error)
}

// Service handles order business logic
type Service struct {
	orderRepo   repository.OrderRepository
	productRepo repository.ProductRepository
	clientRepo  repository.ClientRepository
	holded      HoldedClient
	logger      *slog.Logger
}

// NewService creates a new order service
func NewService(
	orderRepo repository.OrderRepository,
	productRepo repository.ProductRepository,
	clientRepo repository.ClientRepository,
	holded HoldedClient,
	logger *slog.Logger,
) *Service {
	return &Service{
		orderRepo:   orderRepo,
		productRepo: productRepo,
		clientRepo:  clientRepo,
		holded:      holded,
		logger:      logger,
	}
}

// CreateOrderRequest contains the data needed to create an order
type CreateOrderRequest struct {
	ClientID uuid.UUID
	Items    []OrderItemRequest
	Notes    string
}

// OrderItemRequest represents an item in an order request
type OrderItemRequest struct {
	ProductID uuid.UUID
	Quantity  int
}

// Create creates a new order
func (s *Service) Create(ctx context.Context, req CreateOrderRequest) (*domain.Order, error) {
	// Validate items
	if len(req.Items) == 0 {
		return nil, errors.New("order must have at least one item")
	}

	// Validate each item and check stock
	orderItems := make([]domain.OrderItem, 0, len(req.Items))
	for _, item := range req.Items {
		if item.Quantity <= 0 {
			return nil, fmt.Errorf("invalid quantity for product %s", item.ProductID)
		}

		product, err := s.productRepo.GetByID(ctx, item.ProductID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil, fmt.Errorf("%w: %s", ErrProductNotFound, item.ProductID)
			}
			return nil, fmt.Errorf("looking up product: %w", err)
		}

		if !product.IsActive {
			return nil, fmt.Errorf("%w: %s", ErrProductNotFound, item.ProductID)
		}

		if !product.HasStock(item.Quantity) {
			return nil, fmt.Errorf("%w: %s", ErrInsufficientStock, product.Name)
		}

		if item.Quantity < product.MinOrderQuantity {
			return nil, fmt.Errorf("minimum order quantity for %s is %d", product.Name, product.MinOrderQuantity)
		}

		orderItems = append(orderItems, domain.OrderItem{
			ID:        uuid.New(),
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		})
	}

	order := &domain.Order{
		ID:       uuid.New(),
		ClientID: req.ClientID,
		Status:   domain.OrderStatusPending,
		Notes:    req.Notes,
		Items:    orderItems,
	}

	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("creating order: %w", err)
	}

	s.logger.Info("order created",
		"order_id", order.ID,
		"client_id", req.ClientID,
		"items", len(orderItems),
	)

	return order, nil
}

// GetByID retrieves an order by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	return s.orderRepo.GetByID(ctx, id)
}

// ListByClient retrieves orders for a specific client
func (s *Service) ListByClient(ctx context.Context, clientID uuid.UUID, filter repository.OrderFilter) ([]domain.Order, error) {
	return s.orderRepo.ListByClientID(ctx, clientID, filter)
}

// Cancel cancels a pending order
func (s *Service) Cancel(ctx context.Context, orderID, clientID uuid.UUID) error {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return err
	}

	// Verify ownership
	if order.ClientID != clientID {
		return repository.ErrNotFound
	}

	if !order.IsCancellable() {
		return ErrOrderNotPending
	}

	return s.orderRepo.UpdateStatus(ctx, orderID, domain.OrderStatusCancelled)
}

// Approve approves an order and creates an invoice in Holded
func (s *Service) Approve(ctx context.Context, orderID uuid.UUID, approvedBy string) (*domain.Order, error) {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if !order.IsApprovable() {
		return nil, ErrOrderNotPending
	}

	// Get client for Holded contact ID
	client, err := s.clientRepo.GetByID(ctx, order.ClientID)
	if err != nil {
		return nil, fmt.Errorf("getting client: %w", err)
	}

	// Build invoice items
	invoiceItems := make([]holded.InvoiceItem, 0, len(order.Items))
	for _, item := range order.Items {
		product, err := s.productRepo.GetByID(ctx, item.ProductID)
		if err != nil {
			return nil, fmt.Errorf("getting product %s: %w", item.ProductID, err)
		}

		invoiceItems = append(invoiceItems, holded.InvoiceItem{
			Name:     product.Name,
			Units:    item.Quantity,
			Subtotal: product.Price * float64(item.Quantity),
			Tax:      product.TaxRate,
		})
	}

	// Create invoice in Holded
	invoiceReq := &holded.CreateInvoiceRequest{
		ContactID: client.HoldedID,
		Date:      time.Now().Unix(),
		Items:     invoiceItems,
		Notes:     order.Notes,
	}

	invoice, err := s.holded.CreateInvoice(ctx, invoiceReq)
	if err != nil {
		return nil, fmt.Errorf("creating invoice in Holded: %w", err)
	}

	s.logger.Info("invoice created in Holded",
		"order_id", orderID,
		"invoice_id", invoice.ID,
		"invoice_number", invoice.InvoiceNum,
	)

	// Update order with approval and Holded invoice ID
	if err := s.orderRepo.Approve(ctx, orderID, approvedBy, invoice.ID); err != nil {
		return nil, fmt.Errorf("approving order: %w", err)
	}

	// Fetch updated order
	return s.orderRepo.GetByID(ctx, orderID)
}

// Reject rejects an order with a reason
func (s *Service) Reject(ctx context.Context, orderID uuid.UUID, reason string) error {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return err
	}

	if !order.IsApprovable() {
		return ErrOrderNotPending
	}

	return s.orderRepo.Reject(ctx, orderID, reason)
}

// ListAll retrieves all orders (admin)
func (s *Service) ListAll(ctx context.Context, filter repository.OrderFilter) ([]domain.Order, error) {
	return s.orderRepo.List(ctx, filter)
}
