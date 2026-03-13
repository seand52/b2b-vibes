package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"b2b-orders-api/internal/domain"
	"b2b-orders-api/internal/repository"
)

// ClientRepository implements repository.ClientRepository
type ClientRepository struct {
	db *pgxpool.Pool
}

// NewClientRepository creates a new ClientRepository
func NewClientRepository(db *pgxpool.Pool) *ClientRepository {
	return &ClientRepository{db: db}
}

// Upsert inserts or updates a client based on holded_id
func (r *ClientRepository) Upsert(ctx context.Context, client *domain.Client) error {
	now := time.Now()

	billingAddr, err := json.Marshal(client.BillingAddress)
	if err != nil {
		return fmt.Errorf("marshaling billing address: %w", err)
	}

	shippingAddr, err := json.Marshal(client.ShippingAddress)
	if err != nil {
		return fmt.Errorf("marshaling shipping address: %w", err)
	}

	query := `
		INSERT INTO clients (id, holded_id, auth0_id, email, company_name, contact_name, phone, vat_type, vat_number, billing_address, shipping_address, is_active, synced_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (holded_id) DO UPDATE SET
			email = EXCLUDED.email,
			company_name = EXCLUDED.company_name,
			contact_name = EXCLUDED.contact_name,
			phone = EXCLUDED.phone,
			vat_type = EXCLUDED.vat_type,
			vat_number = EXCLUDED.vat_number,
			billing_address = EXCLUDED.billing_address,
			shipping_address = EXCLUDED.shipping_address,
			is_active = EXCLUDED.is_active,
			synced_at = EXCLUDED.synced_at,
			updated_at = EXCLUDED.updated_at
		RETURNING id`

	if client.ID == uuid.Nil {
		client.ID = uuid.New()
	}

	err = r.db.QueryRow(ctx, query,
		client.ID,
		client.HoldedID,
		client.Auth0ID,
		client.Email,
		client.CompanyName,
		client.ContactName,
		client.Phone,
		client.VATType,
		client.VATNumber,
		billingAddr,
		shippingAddr,
		client.IsActive,
		&now,
		now,
		now,
	).Scan(&client.ID)

	if err != nil {
		return fmt.Errorf("upserting client: %w", err)
	}

	return nil
}

// GetByID retrieves a client by its UUID
func (r *ClientRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Client, error) {
	query := `
		SELECT id, holded_id, auth0_id, email, company_name, contact_name, phone, vat_type, vat_number, billing_address, shipping_address, is_active, synced_at, created_at, updated_at
		FROM clients
		WHERE id = $1`

	return r.scanClient(ctx, query, id)
}

// GetByHoldedID retrieves a client by its Holded ID
func (r *ClientRepository) GetByHoldedID(ctx context.Context, holdedID string) (*domain.Client, error) {
	query := `
		SELECT id, holded_id, auth0_id, email, company_name, contact_name, phone, vat_type, vat_number, billing_address, shipping_address, is_active, synced_at, created_at, updated_at
		FROM clients
		WHERE holded_id = $1`

	return r.scanClient(ctx, query, holdedID)
}

// GetByEmail retrieves a client by email
func (r *ClientRepository) GetByEmail(ctx context.Context, email string) (*domain.Client, error) {
	query := `
		SELECT id, holded_id, auth0_id, email, company_name, contact_name, phone, vat_type, vat_number, billing_address, shipping_address, is_active, synced_at, created_at, updated_at
		FROM clients
		WHERE email = $1`

	return r.scanClient(ctx, query, email)
}

// GetByAuth0ID retrieves a client by their Auth0 ID
func (r *ClientRepository) GetByAuth0ID(ctx context.Context, auth0ID string) (*domain.Client, error) {
	query := `
		SELECT id, holded_id, auth0_id, email, company_name, contact_name, phone, vat_type, vat_number, billing_address, shipping_address, is_active, synced_at, created_at, updated_at
		FROM clients
		WHERE auth0_id = $1`

	return r.scanClient(ctx, query, auth0ID)
}

// LinkAuth0ID links an Auth0 ID to a client
func (r *ClientRepository) LinkAuth0ID(ctx context.Context, clientID uuid.UUID, auth0ID string) error {
	query := `UPDATE clients SET auth0_id = $1, updated_at = $2 WHERE id = $3`

	result, err := r.db.Exec(ctx, query, auth0ID, time.Now(), clientID)
	if err != nil {
		return fmt.Errorf("linking auth0 id: %w", err)
	}

	if result.RowsAffected() == 0 {
		return repository.ErrNotFound
	}

	return nil
}

func (r *ClientRepository) scanClient(ctx context.Context, query string, args ...any) (*domain.Client, error) {
	var c domain.Client
	var billingAddr, shippingAddr []byte

	err := r.db.QueryRow(ctx, query, args...).Scan(
		&c.ID,
		&c.HoldedID,
		&c.Auth0ID,
		&c.Email,
		&c.CompanyName,
		&c.ContactName,
		&c.Phone,
		&c.VATType,
		&c.VATNumber,
		&billingAddr,
		&shippingAddr,
		&c.IsActive,
		&c.SyncedAt,
		&c.CreatedAt,
		&c.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scanning client: %w", err)
	}

	if len(billingAddr) > 0 {
		if err := json.Unmarshal(billingAddr, &c.BillingAddress); err != nil {
			return nil, fmt.Errorf("unmarshaling billing address: %w", err)
		}
	}

	if len(shippingAddr) > 0 {
		if err := json.Unmarshal(shippingAddr, &c.ShippingAddress); err != nil {
			return nil, fmt.Errorf("unmarshaling shipping address: %w", err)
		}
	}

	return &c, nil
}

// List retrieves clients with optional filtering
func (r *ClientRepository) List(ctx context.Context, filter repository.ClientFilter) ([]domain.Client, error) {
	var conditions []string
	var args []any
	argIdx := 1

	if filter.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *filter.IsActive)
		argIdx++
	}

	if filter.SearchTerm != "" {
		conditions = append(conditions, fmt.Sprintf("(company_name ILIKE $%d OR email ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+filter.SearchTerm+"%")
		argIdx++
	}

	query := `SELECT id, holded_id, auth0_id, email, company_name, contact_name, phone, vat_type, vat_number, billing_address, shipping_address, is_active, synced_at, created_at, updated_at FROM clients`

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY company_name"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filter.Offset)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying clients: %w", err)
	}
	defer rows.Close()

	var clients []domain.Client
	for rows.Next() {
		var c domain.Client
		var billingAddr, shippingAddr []byte

		err := rows.Scan(
			&c.ID,
			&c.HoldedID,
			&c.Auth0ID,
			&c.Email,
			&c.CompanyName,
			&c.ContactName,
			&c.Phone,
			&c.VATType,
			&c.VATNumber,
			&billingAddr,
			&shippingAddr,
			&c.IsActive,
			&c.SyncedAt,
			&c.CreatedAt,
			&c.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning client row: %w", err)
		}

		if len(billingAddr) > 0 {
			json.Unmarshal(billingAddr, &c.BillingAddress)
		}
		if len(shippingAddr) > 0 {
			json.Unmarshal(shippingAddr, &c.ShippingAddress)
		}

		clients = append(clients, c)
	}

	return clients, nil
}
