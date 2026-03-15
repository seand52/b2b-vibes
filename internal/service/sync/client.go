package sync

import (
	"context"
	"fmt"
	"log/slog"
	"time"
	"unicode"

	"github.com/google/uuid"

	"b2b-orders-api/internal/domain"
	"b2b-orders-api/internal/clients/holded"
	"b2b-orders-api/internal/repository"
)

// ClientSyncer handles syncing clients from Holded contacts to the local database
type ClientSyncer struct {
	holded     holded.ClientInterface
	clientRepo repository.ClientRepository
	syncRepo   repository.SyncStateRepository
	logger     *slog.Logger
}

// NewClientSyncer creates a new ClientSyncer
func NewClientSyncer(
	holded holded.ClientInterface,
	clientRepo repository.ClientRepository,
	syncRepo repository.SyncStateRepository,
	logger *slog.Logger,
) *ClientSyncer {
	return &ClientSyncer{
		holded:     holded,
		clientRepo: clientRepo,
		syncRepo:   syncRepo,
		logger:     logger,
	}
}

// ClientSyncResult contains the result of a client sync operation
type ClientSyncResult struct {
	TotalClients  int
	SyncedClients int
	FailedClients int
	Errors        []error
}

// Sync fetches all contacts from Holded and syncs client-type contacts to the local database
func (s *ClientSyncer) Sync(ctx context.Context) (*ClientSyncResult, error) {
	s.logger.Info("starting client sync")

	// Update sync state to running
	syncState := &domain.SyncState{
		EntityType: "clients",
		Status:     domain.SyncStatusRunning,
	}
	if err := s.syncRepo.Upsert(ctx, syncState); err != nil {
		s.logger.Warn("failed to update sync state", "error", err)
	}

	// Fetch all contacts from Holded
	holdedContacts, err := s.holded.ListContacts(ctx)
	if err != nil {
		s.updateSyncState(ctx, domain.SyncStatusFailed, 0, err.Error())
		return nil, fmt.Errorf("fetching contacts from Holded: %w", err)
	}

	// Filter to only client-type contacts
	var clients []holded.Contact
	for _, contact := range holdedContacts {
		if contact.Type == "client" {
			clients = append(clients, contact)
		}
	}

	result := &ClientSyncResult{
		TotalClients: len(clients),
	}

	if len(clients) == 0 {
		s.updateSyncState(ctx, domain.SyncStatusSuccess, 0, "")
		return result, nil
	}

	// Convert contacts to domain clients
	domainClients := s.convertContacts(clients)

	// Batch upsert all clients
	if err := s.clientRepo.UpsertBatch(ctx, domainClients); err != nil {
		s.updateSyncState(ctx, domain.SyncStatusFailed, 0, err.Error())
		return nil, fmt.Errorf("batch upserting clients: %w", err)
	}

	result.SyncedClients = len(domainClients)

	// Update sync state to success
	s.updateSyncState(ctx, domain.SyncStatusSuccess, result.SyncedClients, "")

	s.logger.Info("client sync completed",
		"total_clients", result.TotalClients,
		"synced_clients", result.SyncedClients,
	)

	return result, nil
}

func (s *ClientSyncer) convertContacts(holdedContacts []holded.Contact) []domain.Client {
	clients := make([]domain.Client, 0, len(holdedContacts))
	now := time.Now()

	for _, hc := range holdedContacts {
		// Determine VAT type based on first character
		vatType := s.parseVATType(hc.VATNumber)

		// Convert billing address
		var billingAddr *domain.Address
		if hc.BillAddress.Address != "" || hc.BillAddress.City != "" {
			billingAddr = &domain.Address{
				Street:     hc.BillAddress.Address,
				City:       hc.BillAddress.City,
				PostalCode: hc.BillAddress.PostalCode,
				Province:   hc.BillAddress.Province,
				Country:    hc.BillAddress.Country,
			}
		}

		// Convert shipping address
		var shippingAddr *domain.Address
		if hc.ShipAddress.Address != "" || hc.ShipAddress.City != "" {
			shippingAddr = &domain.Address{
				Street:     hc.ShipAddress.Address,
				City:       hc.ShipAddress.City,
				PostalCode: hc.ShipAddress.PostalCode,
				Province:   hc.ShipAddress.Province,
				Country:    hc.ShipAddress.Country,
			}
		}

		clients = append(clients, domain.Client{
			ID:              uuid.New(),
			HoldedID:        hc.ID,
			Email:           hc.Email,
			CompanyName:     hc.Name,
			ContactName:     hc.ContactName,
			Phone:           hc.Phone,
			VATType:         vatType,
			VATNumber:       hc.VATNumber,
			BillingAddress:  billingAddr,
			ShippingAddress: shippingAddr,
			IsActive:        true,
			SyncedAt:        &now,
		})
	}

	return clients
}

// parseVATType determines the VAT type based on the first character of the VAT number
// If it starts with a letter -> CIF (company)
// If it starts with a digit -> NIF (individual)
func (s *ClientSyncer) parseVATType(vatNumber string) domain.VATType {
	if vatNumber == "" {
		return "" // Empty VATType is allowed
	}

	firstChar := rune(vatNumber[0])
	if unicode.IsLetter(firstChar) {
		return domain.VATTypeCIF
	} else if unicode.IsDigit(firstChar) {
		return domain.VATTypeNIF
	}

	// Default to empty if we can't determine
	return ""
}

func (s *ClientSyncer) updateSyncState(ctx context.Context, status domain.SyncStatus, itemsSynced int, errMsg string) {
	now := time.Now()
	state := &domain.SyncState{
		EntityType:   "clients",
		LastSyncAt:   &now,
		Status:       status,
		ItemsSynced:  itemsSynced,
		ErrorMessage: errMsg,
	}
	if err := s.syncRepo.Upsert(ctx, state); err != nil {
		s.logger.Warn("failed to update sync state", "error", err)
	}
}
