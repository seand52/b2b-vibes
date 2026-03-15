package sync

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"b2b-orders-api/internal/domain"
	"b2b-orders-api/internal/clients/holded"
	"b2b-orders-api/internal/clients/s3"
	"b2b-orders-api/internal/repository"
)

const (
	maxConcurrentImageFetch = 5
	defaultMinOrderQty      = 1
)

// ProductSyncer handles syncing products from Holded to the local database
type ProductSyncer struct {
	holded      holded.ClientInterface
	s3          *s3.Client
	productRepo repository.ProductRepository
	imageRepo   repository.ProductImageRepository
	syncRepo    repository.SyncStateRepository
	logger      *slog.Logger
}

// NewProductSyncer creates a new ProductSyncer
func NewProductSyncer(
	holded holded.ClientInterface,
	s3 *s3.Client,
	productRepo repository.ProductRepository,
	imageRepo repository.ProductImageRepository,
	syncRepo repository.SyncStateRepository,
	logger *slog.Logger,
) *ProductSyncer {
	return &ProductSyncer{
		holded:      holded,
		s3:          s3,
		productRepo: productRepo,
		imageRepo:   imageRepo,
		syncRepo:    syncRepo,
		logger:      logger,
	}
}

// SyncResult contains the result of a sync operation
type SyncResult struct {
	TotalProducts  int
	SyncedProducts int
	FailedProducts int
	TotalImages    int
	SyncedImages   int
	FailedImages   int
	Errors         []error
}

// Sync fetches all products from Holded and syncs them to the local database
func (s *ProductSyncer) Sync(ctx context.Context) (*SyncResult, error) {
	s.logger.Info("starting product sync")

	// Update sync state to running
	syncState := &domain.SyncState{
		EntityType: "products",
		Status:     domain.SyncStatusRunning,
	}
	if err := s.syncRepo.Upsert(ctx, syncState); err != nil {
		s.logger.Warn("failed to update sync state", "error", err)
	}

	// Fetch all products from Holded
	holdedProducts, err := s.holded.ListProducts(ctx)
	if err != nil {
		s.updateSyncState(ctx, domain.SyncStatusFailed, 0, err.Error())
		return nil, fmt.Errorf("fetching products from Holded: %w", err)
	}

	result := &SyncResult{
		TotalProducts: len(holdedProducts),
	}

	if len(holdedProducts) == 0 {
		s.updateSyncState(ctx, domain.SyncStatusSuccess, 0, "")
		return result, nil
	}

	// Phase 1: Convert and batch upsert all products
	domainProducts := s.convertProducts(holdedProducts)
	if err := s.productRepo.UpsertBatch(ctx, domainProducts); err != nil {
		s.updateSyncState(ctx, domain.SyncStatusFailed, 0, err.Error())
		return nil, fmt.Errorf("batch upserting products: %w", err)
	}
	result.SyncedProducts = len(domainProducts)

	// Phase 2: Fetch images concurrently for all products
	imageResults := s.syncImagesForProducts(ctx, holdedProducts, domainProducts)
	result.TotalImages = imageResults.total
	result.SyncedImages = imageResults.synced
	result.FailedImages = imageResults.failed
	result.Errors = imageResults.errors

	// Update sync state
	status := domain.SyncStatusSuccess
	errMsg := ""
	if result.FailedImages > 0 {
		status = domain.SyncStatusPartial
		errMsg = fmt.Sprintf("%d images failed to sync", result.FailedImages)
	}
	s.updateSyncState(ctx, status, result.SyncedProducts, errMsg)

	s.logger.Info("product sync completed",
		"total_products", result.TotalProducts,
		"synced_products", result.SyncedProducts,
		"total_images", result.TotalImages,
		"synced_images", result.SyncedImages,
		"failed_images", result.FailedImages,
	)

	return result, nil
}

type imageResult struct {
	total  int
	synced int
	failed int
	errors []error
}

func (s *ProductSyncer) syncImagesForProducts(ctx context.Context, holdedProducts []holded.Product, domainProducts []domain.Product) imageResult {
	// Build a map of holded_id -> domain product for quick lookup
	productMap := make(map[string]*domain.Product)
	for i := range domainProducts {
		productMap[domainProducts[i].HoldedID] = &domainProducts[i]
	}

	// Fetch images concurrently with semaphore
	type productImages struct {
		productID   uuid.UUID
		holdedID    string
		images      []holded.ProductImageData
		err         error
	}

	sem := make(chan struct{}, maxConcurrentImageFetch)
	var wg sync.WaitGroup
	results := make([]productImages, len(holdedProducts))

	for i, hp := range holdedProducts {
		wg.Add(1)
		go func(idx int, holdedProd holded.Product) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			domainProd := productMap[holdedProd.ID]
			if domainProd == nil {
				return
			}

			imgs, err := s.holded.GetAllProductImages(ctx, holdedProd.ID)
			results[idx] = productImages{
				productID: domainProd.ID,
				holdedID:  holdedProd.ID,
				images:    imgs,
				err:       err,
			}
		}(i, hp)
	}
	wg.Wait()

	// Collect all images to upload to S3
	var uploadItems []s3.UploadItem
	type imageMapping struct {
		productID   uuid.UUID
		resultIdx   int
		isPrimary   bool
		displayOrder int
	}
	var mappings []imageMapping

	for _, pr := range results {
		if pr.err != nil {
			s.logger.Warn("failed to fetch images for product", "holded_id", pr.holdedID, "error", pr.err)
			continue
		}
		for i, img := range pr.images {
			key := fmt.Sprintf("products/%s/%s", pr.holdedID, img.Filename)
			uploadItems = append(uploadItems, s3.UploadItem{
				Key:         key,
				Data:        img.Data,
				ContentType: img.ContentType,
			})
			mappings = append(mappings, imageMapping{
				productID:    pr.productID,
				resultIdx:    len(uploadItems) - 1,
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

	// Batch upload images to S3
	uploadResults, uploadErr := s.s3.UploadBatch(ctx, uploadItems)

	// Build domain images from successful uploads
	var domainImages []domain.ProductImage
	for _, mapping := range mappings {
		uploadResult := uploadResults[mapping.resultIdx]
		if uploadResult.Err != nil {
			ir.failed++
			ir.errors = append(ir.errors, uploadResult.Err)
			continue
		}

		domainImages = append(domainImages, domain.ProductImage{
			ID:           uuid.New(),
			ProductID:    mapping.productID,
			S3Key:        uploadResult.Key,
			S3URL:        uploadResult.URL,
			IsPrimary:    mapping.isPrimary,
			DisplayOrder: mapping.displayOrder,
		})
		ir.synced++
	}

	// Batch upsert all images
	if len(domainImages) > 0 {
		if err := s.imageRepo.UpsertBatch(ctx, domainImages); err != nil {
			s.logger.Error("failed to batch upsert images", "error", err)
			ir.errors = append(ir.errors, err)
		}
	}

	if uploadErr != nil {
		s.logger.Warn("some image uploads failed", "error", uploadErr)
	}

	return ir
}

func (s *ProductSyncer) convertProducts(holdedProducts []holded.Product) []domain.Product {
	products := make([]domain.Product, 0, len(holdedProducts))

	for _, hp := range holdedProducts {
		// Only sync actual products, not services
		if hp.Kind == "service" {
			continue
		}

		products = append(products, domain.Product{
			ID:               uuid.New(),
			HoldedID:         hp.ID,
			SKU:              hp.SKU,
			Name:             hp.Name,
			Description:      hp.Description,
			Category:         strings.Join(hp.Tags, ", "), // Use tags as category
			Price:            hp.Price,
			TaxRate:          hp.Tax,
			StockQuantity:    hp.Stock,
			MinOrderQuantity: defaultMinOrderQty,
			IsActive:         true,
		})
	}

	return products
}

func (s *ProductSyncer) updateSyncState(ctx context.Context, status domain.SyncStatus, itemsSynced int, errMsg string) {
	now := time.Now()
	state := &domain.SyncState{
		EntityType:   "products",
		LastSyncAt:   &now,
		Status:       status,
		ItemsSynced:  itemsSynced,
		ErrorMessage: errMsg,
	}
	if err := s.syncRepo.Upsert(ctx, state); err != nil {
		s.logger.Warn("failed to update sync state", "error", err)
	}
}
