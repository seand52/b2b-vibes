package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestProduct_HasStock(t *testing.T) {
	tests := []struct {
		name          string
		stockQuantity int
		requested     int
		want          bool
	}{
		{
			name:          "sufficient stock",
			stockQuantity: 10,
			requested:     5,
			want:          true,
		},
		{
			name:          "exact stock",
			stockQuantity: 5,
			requested:     5,
			want:          true,
		},
		{
			name:          "insufficient stock",
			stockQuantity: 3,
			requested:     5,
			want:          false,
		},
		{
			name:          "zero stock",
			stockQuantity: 0,
			requested:     1,
			want:          false,
		},
		{
			name:          "zero requested",
			stockQuantity: 10,
			requested:     0,
			want:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Product{StockQuantity: tt.stockQuantity}
			assert.Equal(t, tt.want, p.HasStock(tt.requested))
		})
	}
}

func TestProduct_PrimaryImage(t *testing.T) {
	productID := uuid.New()

	tests := []struct {
		name   string
		images []ProductImage
		want   *ProductImage
	}{
		{
			name:   "no images",
			images: nil,
			want:   nil,
		},
		{
			name:   "empty images slice",
			images: []ProductImage{},
			want:   nil,
		},
		{
			name: "single non-primary image returns first",
			images: []ProductImage{
				{ID: uuid.New(), ProductID: productID, S3Key: "img1.jpg", IsPrimary: false},
			},
			want: &ProductImage{S3Key: "img1.jpg"},
		},
		{
			name: "primary image returned",
			images: []ProductImage{
				{ID: uuid.New(), ProductID: productID, S3Key: "img1.jpg", IsPrimary: false},
				{ID: uuid.New(), ProductID: productID, S3Key: "img2.jpg", IsPrimary: true},
				{ID: uuid.New(), ProductID: productID, S3Key: "img3.jpg", IsPrimary: false},
			},
			want: &ProductImage{S3Key: "img2.jpg", IsPrimary: true},
		},
		{
			name: "first image when no primary",
			images: []ProductImage{
				{ID: uuid.New(), ProductID: productID, S3Key: "first.jpg", IsPrimary: false},
				{ID: uuid.New(), ProductID: productID, S3Key: "second.jpg", IsPrimary: false},
			},
			want: &ProductImage{S3Key: "first.jpg"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Product{Images: tt.images}
			got := p.PrimaryImage()

			if tt.want == nil {
				assert.Nil(t, got)
				return
			}

			assert.NotNil(t, got)
			assert.Equal(t, tt.want.S3Key, got.S3Key)
			assert.Equal(t, tt.want.IsPrimary, got.IsPrimary)
		})
	}
}
