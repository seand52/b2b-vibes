package domain

import (
	"time"

	"github.com/google/uuid"
)

// Product represents a product (synced from Holded)
type Product struct {
	ID               uuid.UUID      `json:"id"`
	HoldedID         string         `json:"holded_id"`
	SKU              string         `json:"sku"`
	Name             string         `json:"name"`
	Description      string         `json:"description,omitempty"`
	Category         string         `json:"category,omitempty"`
	Price            float64        `json:"price"`
	TaxRate          float64        `json:"tax_rate"` // VAT percentage (default 21% for Spain)
	StockQuantity    int            `json:"stock_quantity"`
	MinOrderQuantity int            `json:"min_order_quantity"`
	IsActive         bool           `json:"is_active"`
	SyncedAt         *time.Time     `json:"synced_at,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	Images           []ProductImage `json:"images,omitempty"`
}

// ProductImage represents an image for a product (stored in S3)
type ProductImage struct {
	ID           uuid.UUID `json:"id"`
	ProductID    uuid.UUID `json:"product_id"`
	S3Key        string    `json:"s3_key"`
	S3URL        string    `json:"s3_url"`
	IsPrimary    bool      `json:"is_primary"`
	DisplayOrder int       `json:"display_order"`
	CreatedAt    time.Time `json:"created_at"`
}

// HasStock returns true if product has stock available
func (p *Product) HasStock(quantity int) bool {
	return p.StockQuantity >= quantity
}

// PrimaryImage returns the primary image or nil if none
func (p *Product) PrimaryImage() *ProductImage {
	for i := range p.Images {
		if p.Images[i].IsPrimary {
			return &p.Images[i]
		}
	}
	if len(p.Images) > 0 {
		return &p.Images[0]
	}
	return nil
}
