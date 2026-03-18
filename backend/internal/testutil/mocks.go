package testutil

import (
	"context"
	"io"
	"log/slog"
	"net/http"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/google/uuid"

	"b2b-orders-api/internal/domain"
	"b2b-orders-api/internal/clients/holded"
	"b2b-orders-api/internal/clients/s3"
	"b2b-orders-api/internal/middleware"
	"b2b-orders-api/internal/repository"
)

// NewDiscardLogger creates a logger that discards all output (for tests)
func NewDiscardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// WithAuthContext adds mock Auth0 claims to the request context
func WithAuthContext(r *http.Request, auth0ID, email string) *http.Request {
	claims := &validator.ValidatedClaims{
		RegisteredClaims: validator.RegisteredClaims{
			Subject: auth0ID,
		},
		CustomClaims: &middleware.CustomClaims{
			Email: email,
		},
	}
	ctx := context.WithValue(r.Context(), jwtmiddleware.ContextKey{}, claims)
	return r.WithContext(ctx)
}

// WithAdminAuthContext adds mock Auth0 claims with admin role to the request context
func WithAdminAuthContext(r *http.Request, auth0ID, email, roleClaim string) *http.Request {
	claims := &validator.ValidatedClaims{
		RegisteredClaims: validator.RegisteredClaims{
			Subject: auth0ID,
		},
		CustomClaims: &middleware.CustomClaims{
			Email: email,
			Extra: map[string]interface{}{
				roleClaim: []interface{}{"admin"},
			},
		},
	}
	ctx := context.WithValue(r.Context(), jwtmiddleware.ContextKey{}, claims)
	return r.WithContext(ctx)
}

// MockProductRepo implements repository.ProductRepository for testing
type MockProductRepo struct {
	UpsertedProducts    []domain.Product
	UpsertErr           error
	UpsertBatchErr      error
	GetByIDResult       *domain.Product
	GetByIDErr          error
	GetByHoldedIDResult *domain.Product
	GetByHoldedIDErr    error
	ListResult          []domain.Product
	ListErr             error
}

func (m *MockProductRepo) Upsert(ctx context.Context, product *domain.Product) error {
	if m.UpsertErr != nil {
		return m.UpsertErr
	}
	m.UpsertedProducts = append(m.UpsertedProducts, *product)
	return nil
}

func (m *MockProductRepo) UpsertBatch(ctx context.Context, products []domain.Product) error {
	if m.UpsertBatchErr != nil {
		return m.UpsertBatchErr
	}
	m.UpsertedProducts = append(m.UpsertedProducts, products...)
	return nil
}

func (m *MockProductRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	return m.GetByIDResult, m.GetByIDErr
}

func (m *MockProductRepo) GetByHoldedID(ctx context.Context, holdedID string) (*domain.Product, error) {
	return m.GetByHoldedIDResult, m.GetByHoldedIDErr
}

func (m *MockProductRepo) List(ctx context.Context, filter repository.ProductFilter) ([]domain.Product, error) {
	return m.ListResult, m.ListErr
}

// MockProductImageRepo implements repository.ProductImageRepository for testing
type MockProductImageRepo struct {
	UpsertedImages        []domain.ProductImage
	UpsertErr             error
	UpsertBatchErr        error
	ListByProductIDResult []domain.ProductImage
	ListByProductIDErr    error
	DeleteByProductIDErr  error
}

func (m *MockProductImageRepo) Upsert(ctx context.Context, image *domain.ProductImage) error {
	if m.UpsertErr != nil {
		return m.UpsertErr
	}
	m.UpsertedImages = append(m.UpsertedImages, *image)
	return nil
}

func (m *MockProductImageRepo) UpsertBatch(ctx context.Context, images []domain.ProductImage) error {
	if m.UpsertBatchErr != nil {
		return m.UpsertBatchErr
	}
	m.UpsertedImages = append(m.UpsertedImages, images...)
	return nil
}

func (m *MockProductImageRepo) ListByProductID(ctx context.Context, productID uuid.UUID) ([]domain.ProductImage, error) {
	return m.ListByProductIDResult, m.ListByProductIDErr
}

func (m *MockProductImageRepo) DeleteByProductID(ctx context.Context, productID uuid.UUID) error {
	return m.DeleteByProductIDErr
}

// MockClientRepo implements repository.ClientRepository for testing
type MockClientRepo struct {
	UpsertedClients     []domain.Client
	UpsertErr           error
	UpsertBatchErr      error
	GetByIDResult       *domain.Client
	GetByIDErr          error
	GetByHoldedIDResult *domain.Client
	GetByHoldedIDErr    error
	GetByEmailResult    *domain.Client
	GetByEmailErr       error
	GetByAuth0IDResult  *domain.Client
	GetByAuth0IDErr     error
	LinkAuth0IDErr      error
	LinkedClientID      uuid.UUID
	LinkedAuth0ID       string
	ListResult          []domain.Client
	ListErr             error
}

func (m *MockClientRepo) Upsert(ctx context.Context, client *domain.Client) error {
	if m.UpsertErr != nil {
		return m.UpsertErr
	}
	m.UpsertedClients = append(m.UpsertedClients, *client)
	return nil
}

func (m *MockClientRepo) UpsertBatch(ctx context.Context, clients []domain.Client) error {
	if m.UpsertBatchErr != nil {
		return m.UpsertBatchErr
	}
	m.UpsertedClients = append(m.UpsertedClients, clients...)
	return nil
}

func (m *MockClientRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Client, error) {
	return m.GetByIDResult, m.GetByIDErr
}

func (m *MockClientRepo) GetByHoldedID(ctx context.Context, holdedID string) (*domain.Client, error) {
	return m.GetByHoldedIDResult, m.GetByHoldedIDErr
}

func (m *MockClientRepo) GetByEmail(ctx context.Context, email string) (*domain.Client, error) {
	return m.GetByEmailResult, m.GetByEmailErr
}

func (m *MockClientRepo) GetByAuth0ID(ctx context.Context, auth0ID string) (*domain.Client, error) {
	return m.GetByAuth0IDResult, m.GetByAuth0IDErr
}

func (m *MockClientRepo) LinkAuth0ID(ctx context.Context, clientID uuid.UUID, auth0ID string) error {
	m.LinkedClientID = clientID
	m.LinkedAuth0ID = auth0ID
	return m.LinkAuth0IDErr
}

func (m *MockClientRepo) List(ctx context.Context, filter repository.ClientFilter) ([]domain.Client, error) {
	return m.ListResult, m.ListErr
}

// MockOrderRepo implements repository.OrderRepository for testing
type MockOrderRepo struct {
	CreatedOrder             *domain.Order
	CreateErr                error
	GetByIDResult            *domain.Order
	GetByIDErr               error
	ListByClientIDResult     []domain.Order
	ListByClientIDErr        error
	ListResult               []domain.Order
	ListErr                  error
	UpdateStatusErr          error
	SetHoldedInvoiceIDErr    error
	ApproveErr               error
	RejectErr                error
	GetDraftByClientIDResult *domain.Order
	GetDraftByClientIDErr    error
	UpdateItemsErr           error
	UpdateNotesErr           error
	SubmitDraftErr           error
}

func (m *MockOrderRepo) Create(ctx context.Context, order *domain.Order) error {
	m.CreatedOrder = order
	return m.CreateErr
}

func (m *MockOrderRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	return m.GetByIDResult, m.GetByIDErr
}

func (m *MockOrderRepo) ListByClientID(ctx context.Context, clientID uuid.UUID, filter repository.OrderFilter) ([]domain.Order, error) {
	return m.ListByClientIDResult, m.ListByClientIDErr
}

func (m *MockOrderRepo) List(ctx context.Context, filter repository.OrderFilter) ([]domain.Order, error) {
	return m.ListResult, m.ListErr
}

func (m *MockOrderRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.OrderStatus) error {
	return m.UpdateStatusErr
}

func (m *MockOrderRepo) SetHoldedInvoiceID(ctx context.Context, id uuid.UUID, invoiceID string) error {
	return m.SetHoldedInvoiceIDErr
}

func (m *MockOrderRepo) Approve(ctx context.Context, id uuid.UUID, approvedBy string, holdedInvoiceID string) error {
	return m.ApproveErr
}

func (m *MockOrderRepo) Reject(ctx context.Context, id uuid.UUID, reason string) error {
	return m.RejectErr
}

func (m *MockOrderRepo) GetDraftByClientID(ctx context.Context, clientID uuid.UUID) (*domain.Order, error) {
	return m.GetDraftByClientIDResult, m.GetDraftByClientIDErr
}

func (m *MockOrderRepo) UpdateItems(ctx context.Context, orderID uuid.UUID, items []domain.OrderItem) error {
	return m.UpdateItemsErr
}

func (m *MockOrderRepo) UpdateNotes(ctx context.Context, orderID uuid.UUID, notes string) error {
	return m.UpdateNotesErr
}

func (m *MockOrderRepo) SubmitDraft(ctx context.Context, orderID uuid.UUID, items []domain.OrderItem) error {
	return m.SubmitDraftErr
}

// MockSyncStateRepo implements repository.SyncStateRepository for testing
type MockSyncStateRepo struct {
	States    map[string]*domain.SyncState
	GetErr    error
	UpsertErr error
}

func NewMockSyncStateRepo() *MockSyncStateRepo {
	return &MockSyncStateRepo{
		States: make(map[string]*domain.SyncState),
	}
}

func (m *MockSyncStateRepo) Get(ctx context.Context, entityType string) (*domain.SyncState, error) {
	if m.GetErr != nil {
		return nil, m.GetErr
	}
	state, ok := m.States[entityType]
	if !ok {
		return nil, nil
	}
	return state, nil
}

func (m *MockSyncStateRepo) Upsert(ctx context.Context, state *domain.SyncState) error {
	if m.UpsertErr != nil {
		return m.UpsertErr
	}
	m.States[state.EntityType] = state
	return nil
}

// MockHoldedClient implements holded client methods for testing
type MockHoldedClient struct {
	ListProductsResult      []holded.Product
	ListProductsErr         error
	GetAllProductImagesFunc func(ctx context.Context, productID string) ([]holded.ProductImageData, error)
	ListContactsResult      []holded.Contact
	ListContactsErr         error
	CreateInvoiceResult     *holded.Invoice
	CreateInvoiceErr        error
}

func (m *MockHoldedClient) ListProducts(ctx context.Context) ([]holded.Product, error) {
	return m.ListProductsResult, m.ListProductsErr
}

func (m *MockHoldedClient) GetAllProductImages(ctx context.Context, productID string) ([]holded.ProductImageData, error) {
	if m.GetAllProductImagesFunc != nil {
		return m.GetAllProductImagesFunc(ctx, productID)
	}
	return nil, nil
}

func (m *MockHoldedClient) ListContacts(ctx context.Context) ([]holded.Contact, error) {
	return m.ListContactsResult, m.ListContactsErr
}

func (m *MockHoldedClient) CreateInvoice(ctx context.Context, req *holded.CreateInvoiceRequest) (*holded.Invoice, error) {
	return m.CreateInvoiceResult, m.CreateInvoiceErr
}

// MockS3Client implements s3 client methods for testing
type MockS3Client struct {
	UploadBatchResult []s3.UploadResult
	UploadBatchErr    error
}

func (m *MockS3Client) UploadBatch(ctx context.Context, items []s3.UploadItem) ([]s3.UploadResult, error) {
	if m.UploadBatchErr != nil {
		return m.UploadBatchResult, m.UploadBatchErr
	}

	// If no custom results provided, generate successful results
	if m.UploadBatchResult == nil {
		results := make([]s3.UploadResult, len(items))
		for i, item := range items {
			results[i] = s3.UploadResult{
				Key: item.Key,
				URL: "https://s3.example.com/" + item.Key,
				Err: nil,
			}
		}
		return results, nil
	}

	return m.UploadBatchResult, nil
}
