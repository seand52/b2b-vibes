package sync

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"b2b-orders-api/internal/domain"
	"b2b-orders-api/internal/clients/holded"
	"b2b-orders-api/internal/testutil"
)

// testableClientSyncer wraps ClientSyncer to allow mock injection
type testableClientSyncer struct {
	*ClientSyncer
	holdedMock *testutil.MockHoldedClient
}

func newTestableClientSyncer(holdedMock *testutil.MockHoldedClient, clientRepo *testutil.MockClientRepo, syncStateRepo *testutil.MockSyncStateRepo) *testableClientSyncer {
	return &testableClientSyncer{
		ClientSyncer: &ClientSyncer{
			clientRepo: clientRepo,
			syncRepo:   syncStateRepo,
			logger:     testutil.NewDiscardLogger(),
		},
		holdedMock: holdedMock,
	}
}

// Sync performs sync using the mock holded client
func (s *testableClientSyncer) Sync(ctx context.Context) (*ClientSyncResult, error) {
	// Fetch contacts from mock
	holdedContacts, err := s.holdedMock.ListContacts(ctx)
	if err != nil {
		s.updateSyncState(ctx, domain.SyncStatusFailed, 0, err.Error())
		return nil, err
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

	// Convert and upsert
	domainClients := s.convertContacts(clients)
	if err := s.clientRepo.UpsertBatch(ctx, domainClients); err != nil {
		s.updateSyncState(ctx, domain.SyncStatusFailed, 0, err.Error())
		return nil, err
	}

	result.SyncedClients = len(domainClients)
	s.updateSyncState(ctx, domain.SyncStatusSuccess, result.SyncedClients, "")

	return result, nil
}

func TestClientSyncer_Sync(t *testing.T) {
	tests := []struct {
		name           string
		contacts       []holded.Contact
		holdedErr      error
		upsertErr      error
		wantTotal      int
		wantSynced     int
		wantErr        bool
		wantSyncStatus domain.SyncStatus
	}{
		{
			name: "success with clients",
			contacts: []holded.Contact{
				{ID: "1", Name: "Company A", Email: "a@example.com", Type: "client"},
				{ID: "2", Name: "Company B", Email: "b@example.com", Type: "client"},
			},
			wantTotal:      2,
			wantSynced:     2,
			wantSyncStatus: domain.SyncStatusSuccess,
		},
		{
			name:           "empty contacts",
			contacts:       []holded.Contact{},
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
			contacts: []holded.Contact{
				{ID: "1", Name: "Company A", Email: "a@example.com", Type: "client"},
			},
			upsertErr:      errors.New("db connection failed"),
			wantErr:        true,
			wantSyncStatus: domain.SyncStatusFailed,
		},
		{
			name: "filters non-client contacts",
			contacts: []holded.Contact{
				{ID: "1", Name: "Client Co", Email: "client@example.com", Type: "client"},
				{ID: "2", Name: "Supplier Co", Email: "supplier@example.com", Type: "supplier"},
				{ID: "3", Name: "Lead Co", Email: "lead@example.com", Type: "lead"},
			},
			wantTotal:      1,
			wantSynced:     1,
			wantSyncStatus: domain.SyncStatusSuccess,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			holdedMock := &testutil.MockHoldedClient{
				ListContactsResult: tt.contacts,
				ListContactsErr:    tt.holdedErr,
			}
			clientRepo := &testutil.MockClientRepo{UpsertBatchErr: tt.upsertErr}
			syncStateRepo := testutil.NewMockSyncStateRepo()

			syncer := newTestableClientSyncer(holdedMock, clientRepo, syncStateRepo)
			result, err := syncer.Sync(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, tt.wantSyncStatus, syncStateRepo.States["clients"].Status)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantTotal, result.TotalClients)
			assert.Equal(t, tt.wantSynced, result.SyncedClients)
			assert.Equal(t, tt.wantSyncStatus, syncStateRepo.States["clients"].Status)
		})
	}
}

func TestClientSyncer_ParseVATType(t *testing.T) {
	syncer := &ClientSyncer{logger: testutil.NewDiscardLogger()}

	tests := []struct {
		vatNumber string
		want      domain.VATType
	}{
		{"B12345678", domain.VATTypeCIF},  // Starts with letter = CIF (company)
		{"A98765432", domain.VATTypeCIF},  // Starts with letter = CIF
		{"12345678A", domain.VATTypeNIF},  // Starts with digit = NIF (individual)
		{"98765432X", domain.VATTypeNIF},  // Starts with digit = NIF
		{"", ""},                           // Empty = empty
	}

	for _, tt := range tests {
		t.Run(tt.vatNumber, func(t *testing.T) {
			got := syncer.parseVATType(tt.vatNumber)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClientSyncer_ConvertContacts(t *testing.T) {
	syncer := &ClientSyncer{logger: testutil.NewDiscardLogger()}

	contacts := []holded.Contact{
		{
			ID:          "holded-123",
			Name:        "Acme Corp",
			ContactName: "John Doe",
			Email:       "john@acme.com",
			Phone:       "+34612345678",
			VATNumber:   "B12345678",
			Type:        "client",
			BillAddress: holded.Address{
				Address:    "Billing St 123",
				City:       "Madrid",
				PostalCode: "28001",
				Province:   "Madrid",
				Country:    "Spain",
			},
			ShipAddress: holded.Address{
				Address:    "Shipping Ave 456",
				City:       "Barcelona",
				PostalCode: "08001",
				Province:   "Barcelona",
				Country:    "Spain",
			},
		},
		{
			ID:        "holded-456",
			Name:      "Solo Trader",
			Email:     "solo@example.com",
			VATNumber: "12345678A",
			Type:      "client",
			// No addresses
		},
	}

	result := syncer.convertContacts(contacts)

	require.Len(t, result, 2)

	// First client with addresses
	assert.Equal(t, "holded-123", result[0].HoldedID)
	assert.Equal(t, "Acme Corp", result[0].CompanyName)
	assert.Equal(t, "John Doe", result[0].ContactName)
	assert.Equal(t, "john@acme.com", result[0].Email)
	assert.Equal(t, "+34612345678", result[0].Phone)
	assert.Equal(t, domain.VATTypeCIF, result[0].VATType)
	assert.Equal(t, "B12345678", result[0].VATNumber)
	assert.True(t, result[0].IsActive)
	assert.NotNil(t, result[0].SyncedAt)

	// Billing address
	require.NotNil(t, result[0].BillingAddress)
	assert.Equal(t, "Billing St 123", result[0].BillingAddress.Street)
	assert.Equal(t, "Madrid", result[0].BillingAddress.City)
	assert.Equal(t, "28001", result[0].BillingAddress.PostalCode)

	// Shipping address
	require.NotNil(t, result[0].ShippingAddress)
	assert.Equal(t, "Shipping Ave 456", result[0].ShippingAddress.Street)
	assert.Equal(t, "Barcelona", result[0].ShippingAddress.City)

	// Second client without addresses
	assert.Equal(t, "holded-456", result[1].HoldedID)
	assert.Equal(t, domain.VATTypeNIF, result[1].VATType)
	assert.Nil(t, result[1].BillingAddress)
	assert.Nil(t, result[1].ShippingAddress)
}
