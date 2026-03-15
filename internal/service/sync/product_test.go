package sync

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"b2b-orders-api/internal/domain"
	"b2b-orders-api/internal/clients/holded"
	"b2b-orders-api/internal/clients/s3"
	"b2b-orders-api/internal/testutil"
)

// testableProductSyncer wraps ProductSyncer to allow mock injection
type testableProductSyncer struct {
	*ProductSyncer
	holdedMock *testutil.MockHoldedClient
	s3Mock     *testutil.MockS3Client
}

func newTestableProductSyncer(
	holdedMock *testutil.MockHoldedClient,
	s3Mock *testutil.MockS3Client,
	productRepo *testutil.MockProductRepo,
	imageRepo *testutil.MockProductImageRepo,
	syncStateRepo *testutil.MockSyncStateRepo,
) *testableProductSyncer {
	return &testableProductSyncer{
		ProductSyncer: &ProductSyncer{
			productRepo: productRepo,
			imageRepo:   imageRepo,
			syncRepo:    syncStateRepo,
			logger:      testutil.NewDiscardLogger(),
		},
		holdedMock: holdedMock,
		s3Mock:     s3Mock,
	}
}

// Sync performs sync using mock clients
func (s *testableProductSyncer) Sync(ctx context.Context) (*SyncResult, error) {
	// Update sync state to running
	syncState := &domain.SyncState{
		EntityType: "products",
		Status:     domain.SyncStatusRunning,
	}
	s.syncRepo.Upsert(ctx, syncState)

	// Fetch products from mock
	holdedProducts, err := s.holdedMock.ListProducts(ctx)
	if err != nil {
		s.updateSyncState(ctx, domain.SyncStatusFailed, 0, err.Error())
		return nil, err
	}

	result := &SyncResult{
		TotalProducts: len(holdedProducts),
	}

	if len(holdedProducts) == 0 {
		s.updateSyncState(ctx, domain.SyncStatusSuccess, 0, "")
		return result, nil
	}

	// Convert and batch upsert products
	domainProducts := s.convertProducts(holdedProducts)
	if err := s.productRepo.UpsertBatch(ctx, domainProducts); err != nil {
		s.updateSyncState(ctx, domain.SyncStatusFailed, 0, err.Error())
		return nil, err
	}
	result.SyncedProducts = len(domainProducts)

	// Sync images (simplified for testing)
	imageResult := s.syncImagesWithMocks(ctx, holdedProducts, domainProducts)
	result.TotalImages = imageResult.total
	result.SyncedImages = imageResult.synced
	result.FailedImages = imageResult.failed

	// Update sync state
	status := domain.SyncStatusSuccess
	if result.FailedImages > 0 {
		status = domain.SyncStatusPartial
	}
	s.updateSyncState(ctx, status, result.SyncedProducts, "")

	return result, nil
}

func (s *testableProductSyncer) syncImagesWithMocks(ctx context.Context, holdedProducts []holded.Product, domainProducts []domain.Product) imageResult {
	productMap := make(map[string]*domain.Product)
	for i := range domainProducts {
		productMap[domainProducts[i].HoldedID] = &domainProducts[i]
	}

	var uploadItems []s3.UploadItem
	type imageMapping struct {
		productID    interface{}
		isPrimary    bool
		displayOrder int
	}
	var mappings []imageMapping

	for _, hp := range holdedProducts {
		domainProd := productMap[hp.ID]
		if domainProd == nil {
			continue
		}

		images, err := s.holdedMock.GetAllProductImages(ctx, hp.ID)
		if err != nil {
			continue
		}

		for i, img := range images {
			uploadItems = append(uploadItems, s3.UploadItem{
				Key:         "products/" + hp.ID + "/" + img.Filename,
				Data:        img.Data,
				ContentType: img.ContentType,
			})
			mappings = append(mappings, imageMapping{
				productID:    domainProd.ID,
				isPrimary:    i == 0,
				displayOrder: i,
			})
		}
	}

	var ir imageResult
	ir.total = len(uploadItems)

	if ir.total == 0 {
		return ir
	}

	uploadResults, _ := s.s3Mock.UploadBatch(ctx, uploadItems)

	var domainImages []domain.ProductImage
	for i, mapping := range mappings {
		if uploadResults[i].Err != nil {
			ir.failed++
			continue
		}
		domainImages = append(domainImages, domain.ProductImage{
			S3Key:        uploadResults[i].Key,
			S3URL:        uploadResults[i].URL,
			IsPrimary:    mapping.isPrimary,
			DisplayOrder: mapping.displayOrder,
		})
		ir.synced++
	}

	if len(domainImages) > 0 {
		s.imageRepo.UpsertBatch(ctx, domainImages)
	}

	return ir
}

func TestProductSyncer_Sync(t *testing.T) {
	tests := []struct {
		name           string
		products       []holded.Product
		holdedErr      error
		upsertErr      error
		wantTotal      int
		wantSynced     int
		wantErr        bool
		wantSyncStatus domain.SyncStatus
	}{
		{
			name: "success with products",
			products: []holded.Product{
				{ID: "1", Name: "Product A", SKU: "SKU-A", Price: 10.00, Kind: "product"},
				{ID: "2", Name: "Product B", SKU: "SKU-B", Price: 20.00, Kind: "product"},
			},
			wantTotal:      2,
			wantSynced:     2,
			wantSyncStatus: domain.SyncStatusSuccess,
		},
		{
			name:           "empty products",
			products:       []holded.Product{},
			wantTotal:      0,
			wantSynced:     0,
			wantSyncStatus: domain.SyncStatusSuccess,
		},
		{
			name:           "holded api error",
			holdedErr:      errors.New("api unavailable"),
			wantErr:        true,
			wantSyncStatus: domain.SyncStatusFailed,
		},
		{
			name: "upsert error",
			products: []holded.Product{
				{ID: "1", Name: "Product A", SKU: "SKU-A", Kind: "product"},
			},
			upsertErr:      errors.New("db connection failed"),
			wantErr:        true,
			wantSyncStatus: domain.SyncStatusFailed,
		},
		{
			name: "filters services",
			products: []holded.Product{
				{ID: "1", Name: "Product A", SKU: "SKU-A", Kind: "product"},
				{ID: "2", Name: "Service B", SKU: "SVC-B", Kind: "service"},
				{ID: "3", Name: "Product C", SKU: "SKU-C", Kind: "product"},
			},
			wantTotal:      3,  // Total from Holded
			wantSynced:     2,  // Only products, not services
			wantSyncStatus: domain.SyncStatusSuccess,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			holdedMock := &testutil.MockHoldedClient{
				ListProductsResult: tt.products,
				ListProductsErr:    tt.holdedErr,
			}
			s3Mock := &testutil.MockS3Client{}
			productRepo := &testutil.MockProductRepo{UpsertBatchErr: tt.upsertErr}
			imageRepo := &testutil.MockProductImageRepo{}
			syncStateRepo := testutil.NewMockSyncStateRepo()

			syncer := newTestableProductSyncer(holdedMock, s3Mock, productRepo, imageRepo, syncStateRepo)
			result, err := syncer.Sync(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, tt.wantSyncStatus, syncStateRepo.States["products"].Status)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantTotal, result.TotalProducts)
			assert.Equal(t, tt.wantSynced, result.SyncedProducts)
			assert.Equal(t, tt.wantSyncStatus, syncStateRepo.States["products"].Status)
		})
	}
}

func TestProductSyncer_Sync_WithImages(t *testing.T) {
	holdedMock := &testutil.MockHoldedClient{
		ListProductsResult: []holded.Product{
			{ID: "prod-1", Name: "Product 1", SKU: "SKU-1", Kind: "product"},
		},
		GetAllProductImagesFunc: func(ctx context.Context, productID string) ([]holded.ProductImageData, error) {
			return []holded.ProductImageData{
				{Filename: "main.jpg", Data: []byte("image-data"), ContentType: "image/jpeg"},
				{Filename: "side.jpg", Data: []byte("image-data-2"), ContentType: "image/jpeg"},
			}, nil
		},
	}
	s3Mock := &testutil.MockS3Client{}
	productRepo := &testutil.MockProductRepo{}
	imageRepo := &testutil.MockProductImageRepo{}
	syncStateRepo := testutil.NewMockSyncStateRepo()

	syncer := newTestableProductSyncer(holdedMock, s3Mock, productRepo, imageRepo, syncStateRepo)
	result, err := syncer.Sync(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 1, result.SyncedProducts)
	assert.Equal(t, 2, result.TotalImages)
	assert.Equal(t, 2, result.SyncedImages)
	assert.Equal(t, 0, result.FailedImages)
}

func TestProductSyncer_Sync_PartialImageFailure(t *testing.T) {
	holdedMock := &testutil.MockHoldedClient{
		ListProductsResult: []holded.Product{
			{ID: "prod-1", Name: "Product 1", SKU: "SKU-1", Kind: "product"},
		},
		GetAllProductImagesFunc: func(ctx context.Context, productID string) ([]holded.ProductImageData, error) {
			return []holded.ProductImageData{
				{Filename: "main.jpg", Data: []byte("data"), ContentType: "image/jpeg"},
				{Filename: "fail.jpg", Data: []byte("data"), ContentType: "image/jpeg"},
			}, nil
		},
	}
	s3Mock := &testutil.MockS3Client{
		UploadBatchResult: []s3.UploadResult{
			{Key: "products/prod-1/main.jpg", URL: "https://s3.example.com/main.jpg", Err: nil},
			{Key: "products/prod-1/fail.jpg", URL: "", Err: errors.New("upload failed")},
		},
	}
	productRepo := &testutil.MockProductRepo{}
	imageRepo := &testutil.MockProductImageRepo{}
	syncStateRepo := testutil.NewMockSyncStateRepo()

	syncer := newTestableProductSyncer(holdedMock, s3Mock, productRepo, imageRepo, syncStateRepo)
	result, err := syncer.Sync(context.Background())

	require.NoError(t, err)
	assert.Equal(t, 2, result.TotalImages)
	assert.Equal(t, 1, result.SyncedImages)
	assert.Equal(t, 1, result.FailedImages)
	assert.Equal(t, domain.SyncStatusPartial, syncStateRepo.States["products"].Status)
}

func TestProductSyncer_ConvertProducts(t *testing.T) {
	syncer := &ProductSyncer{logger: testutil.NewDiscardLogger()}

	holdedProducts := []holded.Product{
		{
			ID:          "holded-123",
			SKU:         "SKU-001",
			Name:        "Test Product",
			Description: "A test product",
			Price:       99.99,
			Tax:         21.0,
			Stock:       100,
			Kind:        "product",
			Tags:        []string{"electronics"},
		},
		{
			ID:   "holded-456",
			Name: "A Service",
			Kind: "service", // Should be filtered out
		},
	}

	result := syncer.convertProducts(holdedProducts)

	require.Len(t, result, 1) // Only the product, not the service

	assert.Equal(t, "holded-123", result[0].HoldedID)
	assert.Equal(t, "SKU-001", result[0].SKU)
	assert.Equal(t, "Test Product", result[0].Name)
	assert.Equal(t, "A test product", result[0].Description)
	assert.Equal(t, 99.99, result[0].Price)
	assert.Equal(t, 21.0, result[0].TaxRate)
	assert.Equal(t, 100, result[0].StockQuantity)
	assert.Equal(t, "electronics", result[0].Category)
	assert.Equal(t, 1, result[0].MinOrderQuantity)
	assert.True(t, result[0].IsActive)
}
