package sync

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// ProductSyncerInterface defines the interface for product syncing
type ProductSyncerInterface interface {
	Sync(ctx context.Context) (*SyncResult, error)
}

// ClientSyncerInterface defines the interface for client syncing
type ClientSyncerInterface interface {
	Sync(ctx context.Context) (*ClientSyncResult, error)
}

// SchedulerConfig holds configuration for the sync scheduler
type SchedulerConfig struct {
	ProductSyncInterval time.Duration
	ClientSyncInterval  time.Duration
}

// DefaultSchedulerConfig returns default configuration
func DefaultSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		ProductSyncInterval: 15 * time.Minute,
		ClientSyncInterval:  30 * time.Minute,
	}
}

// Scheduler manages periodic syncing of products and clients
type Scheduler struct {
	productSyncer ProductSyncerInterface
	clientSyncer  ClientSyncerInterface
	config        SchedulerConfig
	logger        *slog.Logger

	// Internal state for lifecycle management
	mu              sync.Mutex
	running         bool
	cancelFunc      context.CancelFunc
	wg              sync.WaitGroup
	productSyncing  bool
	clientSyncing   bool
}

// NewScheduler creates a new sync scheduler
func NewScheduler(
	productSyncer ProductSyncerInterface,
	clientSyncer ClientSyncerInterface,
	config SchedulerConfig,
	logger *slog.Logger,
) *Scheduler {
	return &Scheduler{
		productSyncer: productSyncer,
		clientSyncer:  clientSyncer,
		config:        config,
		logger:        logger,
	}
}

// Start begins the scheduler's background sync loops
// It starts separate goroutines for product and client syncing
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("scheduler is already running")
	}

	s.logger.Info("starting sync scheduler",
		"product_interval", s.config.ProductSyncInterval,
		"client_interval", s.config.ClientSyncInterval,
	)

	// Create a cancellable context for the scheduler
	ctx, cancel := context.WithCancel(ctx)
	s.cancelFunc = cancel
	s.running = true

	// Start product sync loop
	if s.productSyncer != nil {
		s.wg.Add(1)
		go s.productSyncLoop(ctx)
	}

	// Start client sync loop
	if s.clientSyncer != nil {
		s.wg.Add(1)
		go s.clientSyncLoop(ctx)
	}

	return nil
}

// Stop gracefully stops the scheduler
// It waits for ongoing syncs to complete
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}

	s.logger.Info("stopping sync scheduler")
	s.running = false

	// Cancel the context to signal goroutines to stop
	if s.cancelFunc != nil {
		s.cancelFunc()
	}
	s.mu.Unlock()

	// Wait for all goroutines to finish
	s.wg.Wait()
	s.logger.Info("sync scheduler stopped")
}

// productSyncLoop runs the periodic product sync
func (s *Scheduler) productSyncLoop(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.ProductSyncInterval)
	defer ticker.Stop()

	s.logger.Info("product sync loop started")

	// Run initial sync immediately
	s.syncProducts(ctx)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("product sync loop stopped")
			return
		case <-ticker.C:
			s.syncProducts(ctx)
		}
	}
}

// clientSyncLoop runs the periodic client sync
func (s *Scheduler) clientSyncLoop(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.ClientSyncInterval)
	defer ticker.Stop()

	s.logger.Info("client sync loop started")

	// Run initial sync immediately
	s.syncClients(ctx)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("client sync loop stopped")
			return
		case <-ticker.C:
			s.syncClients(ctx)
		}
	}
}

// syncProducts performs a product sync if one isn't already running
func (s *Scheduler) syncProducts(ctx context.Context) {
	s.mu.Lock()
	if s.productSyncing {
		s.logger.Info("skipping product sync - previous sync still running")
		s.mu.Unlock()
		return
	}
	s.productSyncing = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.productSyncing = false
		s.mu.Unlock()
	}()

	s.logger.Info("starting scheduled product sync")
	start := time.Now()

	result, err := s.productSyncer.Sync(ctx)
	duration := time.Since(start)

	if err != nil {
		s.logger.Error("product sync failed",
			"error", err,
			"duration", duration,
		)
		return
	}

	s.logger.Info("product sync completed",
		"total_products", result.TotalProducts,
		"synced_products", result.SyncedProducts,
		"failed_products", result.FailedProducts,
		"total_images", result.TotalImages,
		"synced_images", result.SyncedImages,
		"failed_images", result.FailedImages,
		"duration", duration,
	)
}

// syncClients performs a client sync if one isn't already running
func (s *Scheduler) syncClients(ctx context.Context) {
	s.mu.Lock()
	if s.clientSyncing {
		s.logger.Info("skipping client sync - previous sync still running")
		s.mu.Unlock()
		return
	}
	s.clientSyncing = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.clientSyncing = false
		s.mu.Unlock()
	}()

	s.logger.Info("starting scheduled client sync")
	start := time.Now()

	result, err := s.clientSyncer.Sync(ctx)
	duration := time.Since(start)

	if err != nil {
		s.logger.Error("client sync failed",
			"error", err,
			"duration", duration,
		)
		return
	}

	s.logger.Info("client sync completed",
		"total_clients", result.TotalClients,
		"synced_clients", result.SyncedClients,
		"failed_clients", result.FailedClients,
		"duration", duration,
	)
}

// SyncProductsNow triggers an immediate product sync
// Returns an error if a sync is already in progress
func (s *Scheduler) SyncProductsNow(ctx context.Context) (*SyncResult, error) {
	s.mu.Lock()
	if s.productSyncing {
		s.mu.Unlock()
		return nil, fmt.Errorf("product sync already in progress")
	}
	s.productSyncing = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.productSyncing = false
		s.mu.Unlock()
	}()

	s.logger.Info("manual product sync triggered")
	start := time.Now()

	result, err := s.productSyncer.Sync(ctx)
	duration := time.Since(start)

	if err != nil {
		s.logger.Error("manual product sync failed",
			"error", err,
			"duration", duration,
		)
		return nil, err
	}

	s.logger.Info("manual product sync completed",
		"total_products", result.TotalProducts,
		"synced_products", result.SyncedProducts,
		"failed_products", result.FailedProducts,
		"total_images", result.TotalImages,
		"synced_images", result.SyncedImages,
		"failed_images", result.FailedImages,
		"duration", duration,
	)

	return result, nil
}

// SyncClientsNow triggers an immediate client sync
// Returns an error if a sync is already in progress
func (s *Scheduler) SyncClientsNow(ctx context.Context) (*ClientSyncResult, error) {
	s.mu.Lock()
	if s.clientSyncing {
		s.mu.Unlock()
		return nil, fmt.Errorf("client sync already in progress")
	}
	s.clientSyncing = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.clientSyncing = false
		s.mu.Unlock()
	}()

	s.logger.Info("manual client sync triggered")
	start := time.Now()

	result, err := s.clientSyncer.Sync(ctx)
	duration := time.Since(start)

	if err != nil {
		s.logger.Error("manual client sync failed",
			"error", err,
			"duration", duration,
		)
		return nil, err
	}

	s.logger.Info("manual client sync completed",
		"total_clients", result.TotalClients,
		"synced_clients", result.SyncedClients,
		"failed_clients", result.FailedClients,
		"duration", duration,
	)

	return result, nil
}

// IsRunning returns whether the scheduler is currently running
func (s *Scheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// IsSyncing returns the current sync status for products and clients
func (s *Scheduler) IsSyncing() (productSyncing, clientSyncing bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.productSyncing, s.clientSyncing
}
