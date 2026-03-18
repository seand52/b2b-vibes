package domain

import "time"

// SyncStatus represents the status of a sync operation
type SyncStatus string

const (
	SyncStatusSuccess SyncStatus = "success"
	SyncStatusPartial SyncStatus = "partial"
	SyncStatusFailed  SyncStatus = "failed"
	SyncStatusRunning SyncStatus = "running"
)

// SyncState tracks the state of sync operations for an entity type
type SyncState struct {
	EntityType   string     `json:"entity_type"` // "products" or "clients"
	LastSyncAt   *time.Time `json:"last_sync_at,omitempty"`
	Status       SyncStatus `json:"status"`
	ItemsSynced  int        `json:"items_synced"`
	ErrorMessage string     `json:"error_message,omitempty"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
