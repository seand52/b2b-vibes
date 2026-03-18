package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"b2b-orders-api/internal/domain"
)

// ProductImageRepository implements repository.ProductImageRepository
type ProductImageRepository struct {
	db *pgxpool.Pool
}

// NewProductImageRepository creates a new ProductImageRepository
func NewProductImageRepository(db *pgxpool.Pool) *ProductImageRepository {
	return &ProductImageRepository{db: db}
}

// Upsert inserts or updates a product image
func (r *ProductImageRepository) Upsert(ctx context.Context, image *domain.ProductImage) error {
	if image.ID == uuid.Nil {
		image.ID = uuid.New()
	}
	image.CreatedAt = time.Now()

	query := `
		INSERT INTO product_images (id, product_id, s3_key, s3_url, is_primary, display_order, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			s3_key = EXCLUDED.s3_key,
			s3_url = EXCLUDED.s3_url,
			is_primary = EXCLUDED.is_primary,
			display_order = EXCLUDED.display_order`

	_, err := r.db.Exec(ctx, query,
		image.ID,
		image.ProductID,
		image.S3Key,
		image.S3URL,
		image.IsPrimary,
		image.DisplayOrder,
		image.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("upserting product image: %w", err)
	}

	return nil
}

// UpsertBatch inserts or updates multiple product images in a single query
func (r *ProductImageRepository) UpsertBatch(ctx context.Context, images []domain.ProductImage) error {
	if len(images) == 0 {
		return nil
	}

	now := time.Now()

	query := `
		INSERT INTO product_images (id, product_id, s3_key, s3_url, is_primary, display_order, created_at)
		VALUES `

	var valueStrings []string
	var args []any
	argIdx := 1

	for i := range images {
		img := &images[i]
		if img.ID == uuid.Nil {
			img.ID = uuid.New()
		}

		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			argIdx, argIdx+1, argIdx+2, argIdx+3, argIdx+4, argIdx+5, argIdx+6))

		args = append(args,
			img.ID,
			img.ProductID,
			img.S3Key,
			img.S3URL,
			img.IsPrimary,
			img.DisplayOrder,
			now,
		)
		argIdx += 7
	}

	query += strings.Join(valueStrings, ", ")
	query += `
		ON CONFLICT (id) DO UPDATE SET
			s3_key = EXCLUDED.s3_key,
			s3_url = EXCLUDED.s3_url,
			is_primary = EXCLUDED.is_primary,
			display_order = EXCLUDED.display_order`

	_, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("batch upserting product images: %w", err)
	}

	return nil
}

// ListByProductID retrieves all images for a product
func (r *ProductImageRepository) ListByProductID(ctx context.Context, productID uuid.UUID) ([]domain.ProductImage, error) {
	query := `
		SELECT id, product_id, s3_key, s3_url, is_primary, display_order, created_at
		FROM product_images
		WHERE product_id = $1
		ORDER BY is_primary DESC, display_order`

	rows, err := r.db.Query(ctx, query, productID)
	if err != nil {
		return nil, fmt.Errorf("querying product images: %w", err)
	}
	defer rows.Close()

	var images []domain.ProductImage
	for rows.Next() {
		var img domain.ProductImage
		err := rows.Scan(
			&img.ID,
			&img.ProductID,
			&img.S3Key,
			&img.S3URL,
			&img.IsPrimary,
			&img.DisplayOrder,
			&img.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning product image: %w", err)
		}
		images = append(images, img)
	}

	return images, nil
}

// DeleteByProductID removes all images for a product
func (r *ProductImageRepository) DeleteByProductID(ctx context.Context, productID uuid.UUID) error {
	query := `DELETE FROM product_images WHERE product_id = $1`

	_, err := r.db.Exec(ctx, query, productID)
	if err != nil {
		return fmt.Errorf("deleting product images: %w", err)
	}

	return nil
}
