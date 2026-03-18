package postgres

import (
	"context"
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

// ProductRepository implements repository.ProductRepository
type ProductRepository struct {
	db *pgxpool.Pool
}

// NewProductRepository creates a new ProductRepository
func NewProductRepository(db *pgxpool.Pool) *ProductRepository {
	return &ProductRepository{db: db}
}

// Upsert inserts or updates a product based on holded_id
func (r *ProductRepository) Upsert(ctx context.Context, product *domain.Product) error {
	now := time.Now()

	query := `
		INSERT INTO products (id, holded_id, sku, name, description, category, price, tax_rate, stock_quantity, min_order_quantity, is_active, synced_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (holded_id) DO UPDATE SET
			sku = EXCLUDED.sku,
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			category = EXCLUDED.category,
			price = EXCLUDED.price,
			tax_rate = EXCLUDED.tax_rate,
			stock_quantity = EXCLUDED.stock_quantity,
			min_order_quantity = EXCLUDED.min_order_quantity,
			is_active = EXCLUDED.is_active,
			synced_at = EXCLUDED.synced_at,
			updated_at = EXCLUDED.updated_at
		RETURNING id`

	if product.ID == uuid.Nil {
		product.ID = uuid.New()
	}

	err := r.db.QueryRow(ctx, query,
		product.ID,
		product.HoldedID,
		product.SKU,
		product.Name,
		product.Description,
		product.Category,
		product.Price,
		product.TaxRate,
		product.StockQuantity,
		product.MinOrderQuantity,
		product.IsActive,
		&now,
		now,
		now,
	).Scan(&product.ID)

	if err != nil {
		return fmt.Errorf("upserting product: %w", err)
	}

	return nil
}

// UpsertBatch inserts or updates multiple products in a single transaction
func (r *ProductRepository) UpsertBatch(ctx context.Context, products []domain.Product) error {
	if len(products) == 0 {
		return nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	now := time.Now()

	// Build batch insert with multiple value sets
	query := `
		INSERT INTO products (id, holded_id, sku, name, description, category, price, tax_rate, stock_quantity, min_order_quantity, is_active, synced_at, created_at, updated_at)
		VALUES `

	var valueStrings []string
	var args []any
	argIdx := 1

	for i := range products {
		p := &products[i]
		if p.ID == uuid.Nil {
			p.ID = uuid.New()
		}

		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			argIdx, argIdx+1, argIdx+2, argIdx+3, argIdx+4, argIdx+5, argIdx+6, argIdx+7, argIdx+8, argIdx+9, argIdx+10, argIdx+11, argIdx+12, argIdx+13))

		args = append(args,
			p.ID,
			p.HoldedID,
			p.SKU,
			p.Name,
			p.Description,
			p.Category,
			p.Price,
			p.TaxRate,
			p.StockQuantity,
			p.MinOrderQuantity,
			p.IsActive,
			now,
			now,
			now,
		)
		argIdx += 14
	}

	query += strings.Join(valueStrings, ", ")
	query += `
		ON CONFLICT (holded_id) DO UPDATE SET
			sku = EXCLUDED.sku,
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			category = EXCLUDED.category,
			price = EXCLUDED.price,
			tax_rate = EXCLUDED.tax_rate,
			stock_quantity = EXCLUDED.stock_quantity,
			min_order_quantity = EXCLUDED.min_order_quantity,
			is_active = EXCLUDED.is_active,
			synced_at = EXCLUDED.synced_at,
			updated_at = EXCLUDED.updated_at`

	_, err = tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("batch upserting products: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

// GetByID retrieves a product by its UUID
func (r *ProductRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	query := `
		SELECT id, holded_id, sku, name, description, category, price, tax_rate, stock_quantity, min_order_quantity, is_active, synced_at, created_at, updated_at
		FROM products
		WHERE id = $1`

	return r.scanProduct(ctx, query, id)
}

// GetByHoldedID retrieves a product by its Holded ID
func (r *ProductRepository) GetByHoldedID(ctx context.Context, holdedID string) (*domain.Product, error) {
	query := `
		SELECT id, holded_id, sku, name, description, category, price, tax_rate, stock_quantity, min_order_quantity, is_active, synced_at, created_at, updated_at
		FROM products
		WHERE holded_id = $1`

	return r.scanProduct(ctx, query, holdedID)
}

func (r *ProductRepository) scanProduct(ctx context.Context, query string, args ...any) (*domain.Product, error) {
	var p domain.Product
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&p.ID,
		&p.HoldedID,
		&p.SKU,
		&p.Name,
		&p.Description,
		&p.Category,
		&p.Price,
		&p.TaxRate,
		&p.StockQuantity,
		&p.MinOrderQuantity,
		&p.IsActive,
		&p.SyncedAt,
		&p.CreatedAt,
		&p.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scanning product: %w", err)
	}

	return &p, nil
}

// List retrieves products with optional filtering
func (r *ProductRepository) List(ctx context.Context, filter repository.ProductFilter) ([]domain.Product, error) {
	var conditions []string
	var args []any
	argIdx := 1

	if filter.Category != "" {
		conditions = append(conditions, fmt.Sprintf("category = $%d", argIdx))
		args = append(args, filter.Category)
		argIdx++
	}

	if filter.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *filter.IsActive)
		argIdx++
	}

	if filter.InStock != nil && *filter.InStock {
		conditions = append(conditions, "stock_quantity > 0")
	}

	if filter.SearchTerm != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR sku ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+filter.SearchTerm+"%")
		argIdx++
	}

	query := `SELECT id, holded_id, sku, name, description, category, price, tax_rate, stock_quantity, min_order_quantity, is_active, synced_at, created_at, updated_at FROM products`

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY name"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filter.Offset)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying products: %w", err)
	}
	defer rows.Close()

	var products []domain.Product
	for rows.Next() {
		var p domain.Product
		err := rows.Scan(
			&p.ID,
			&p.HoldedID,
			&p.SKU,
			&p.Name,
			&p.Description,
			&p.Category,
			&p.Price,
			&p.TaxRate,
			&p.StockQuantity,
			&p.MinOrderQuantity,
			&p.IsActive,
			&p.SyncedAt,
			&p.CreatedAt,
			&p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning product row: %w", err)
		}
		products = append(products, p)
	}

	return products, nil
}
