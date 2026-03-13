package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"b2b-orders-api/internal/domain"
	"b2b-orders-api/internal/repository"
)

// SyncStateRepository implements repository.SyncStateRepository
type SyncStateRepository struct {
	db *pgxpool.Pool
}

// NewSyncStateRepository creates a new SyncStateRepository
func NewSyncStateRepository(db *pgxpool.Pool) *SyncStateRepository {
	return &SyncStateRepository{db: db}
}

// Get retrieves the sync state for an entity type
func (r *SyncStateRepository) Get(ctx context.Context, entityType string) (*domain.SyncState, error) {
	query := `
		SELECT entity_type, last_sync_at, status, items_synced, error_message, updated_at
		FROM sync_state
		WHERE entity_type = $1`

	var s domain.SyncState
	err := r.db.QueryRow(ctx, query, entityType).Scan(
		&s.EntityType,
		&s.LastSyncAt,
		&s.Status,
		&s.ItemsSynced,
		&s.ErrorMessage,
		&s.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scanning sync state: %w", err)
	}

	return &s, nil
}

// Upsert inserts or updates the sync state for an entity type
func (r *SyncStateRepository) Upsert(ctx context.Context, state *domain.SyncState) error {
	now := time.Now()
	state.UpdatedAt = now

	query := `
		INSERT INTO sync_state (entity_type, last_sync_at, status, items_synced, error_message, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (entity_type) DO UPDATE SET
			last_sync_at = EXCLUDED.last_sync_at,
			status = EXCLUDED.status,
			items_synced = EXCLUDED.items_synced,
			error_message = EXCLUDED.error_message,
			updated_at = EXCLUDED.updated_at`

	_, err := r.db.Exec(ctx, query,
		state.EntityType,
		state.LastSyncAt,
		state.Status,
		state.ItemsSynced,
		state.ErrorMessage,
		state.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("upserting sync state: %w", err)
	}

	return nil
}
