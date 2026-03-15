package repository

import (
	"context"

	"github.com/google/uuid"

	"b2b-orders-api/internal/domain"
)

// ProductRepository handles product data access
type ProductRepository interface {
	Upsert(ctx context.Context, product *domain.Product) error
	UpsertBatch(ctx context.Context, products []domain.Product) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error)
	GetByHoldedID(ctx context.Context, holdedID string) (*domain.Product, error)
	List(ctx context.Context, filter ProductFilter) ([]domain.Product, error)
}

// ProductFilter holds filtering options for product queries
type ProductFilter struct {
	Category   string
	IsActive   *bool
	InStock    *bool
	SearchTerm string
	Limit      int
	Offset     int
}

// ProductImageRepository handles product image data access
type ProductImageRepository interface {
	Upsert(ctx context.Context, image *domain.ProductImage) error
	UpsertBatch(ctx context.Context, images []domain.ProductImage) error
	ListByProductID(ctx context.Context, productID uuid.UUID) ([]domain.ProductImage, error)
	DeleteByProductID(ctx context.Context, productID uuid.UUID) error
}

// ClientRepository handles client data access
type ClientRepository interface {
	Upsert(ctx context.Context, client *domain.Client) error
	UpsertBatch(ctx context.Context, clients []domain.Client) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Client, error)
	GetByHoldedID(ctx context.Context, holdedID string) (*domain.Client, error)
	GetByEmail(ctx context.Context, email string) (*domain.Client, error)
	GetByAuth0ID(ctx context.Context, auth0ID string) (*domain.Client, error)
	LinkAuth0ID(ctx context.Context, clientID uuid.UUID, auth0ID string) error
	List(ctx context.Context, filter ClientFilter) ([]domain.Client, error)
}

// ClientFilter holds filtering options for client queries
type ClientFilter struct {
	IsActive   *bool
	SearchTerm string
	Limit      int
	Offset     int
}

// OrderRepository handles order data access
type OrderRepository interface {
	Create(ctx context.Context, order *domain.Order) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Order, error)
	ListByClientID(ctx context.Context, clientID uuid.UUID, filter OrderFilter) ([]domain.Order, error)
	List(ctx context.Context, filter OrderFilter) ([]domain.Order, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.OrderStatus) error
	SetHoldedInvoiceID(ctx context.Context, id uuid.UUID, invoiceID string) error
	Approve(ctx context.Context, id uuid.UUID, approvedBy string, holdedInvoiceID string) error
	Reject(ctx context.Context, id uuid.UUID, reason string) error

	// Cart-specific methods
	GetDraftByClientID(ctx context.Context, clientID uuid.UUID) (*domain.Order, error)
	UpdateItems(ctx context.Context, orderID uuid.UUID, items []domain.OrderItem) error
	UpdateNotes(ctx context.Context, orderID uuid.UUID, notes string) error
	SubmitDraft(ctx context.Context, orderID uuid.UUID, items []domain.OrderItem) error
}

// OrderFilter holds filtering options for order queries
type OrderFilter struct {
	Status   domain.OrderStatus
	ClientID *uuid.UUID
	Limit    int
	Offset   int
}

// SyncStateRepository handles sync state data access
type SyncStateRepository interface {
	Get(ctx context.Context, entityType string) (*domain.SyncState, error)
	Upsert(ctx context.Context, state *domain.SyncState) error
}
