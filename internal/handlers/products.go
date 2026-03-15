package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"b2b-orders-api/internal/domain"
	apierrors "b2b-orders-api/internal/errors"
	"b2b-orders-api/internal/repository"
)

// ProductHandler handles product-related HTTP requests
type ProductHandler struct {
	productRepo repository.ProductRepository
	imageRepo   repository.ProductImageRepository
	logger      *slog.Logger
}

// NewProductHandler creates a new product handler
func NewProductHandler(
	productRepo repository.ProductRepository,
	imageRepo repository.ProductImageRepository,
	logger *slog.Logger,
) *ProductHandler {
	return &ProductHandler{
		productRepo: productRepo,
		imageRepo:   imageRepo,
		logger:      logger,
	}
}

// productResponse is the API response for a product
type productResponse struct {
	ID               uuid.UUID              `json:"id"`
	SKU              string                 `json:"sku"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description,omitempty"`
	Category         string                 `json:"category,omitempty"`
	Price            float64                `json:"price"`
	TaxRate          float64                `json:"tax_rate"`
	StockQuantity    int                    `json:"stock_quantity"`
	MinOrderQuantity int                    `json:"min_order_quantity"`
	Images           []productImageResponse `json:"images,omitempty"`
}

type productImageResponse struct {
	URL          string `json:"url"`
	IsPrimary    bool   `json:"is_primary"`
	DisplayOrder int    `json:"display_order"`
}

func toProductResponse(p *domain.Product) productResponse {
	resp := productResponse{
		ID:               p.ID,
		SKU:              p.SKU,
		Name:             p.Name,
		Description:      p.Description,
		Category:         p.Category,
		Price:            p.Price,
		TaxRate:          p.TaxRate,
		StockQuantity:    p.StockQuantity,
		MinOrderQuantity: p.MinOrderQuantity,
	}

	for _, img := range p.Images {
		resp.Images = append(resp.Images, productImageResponse{
			URL:          img.S3URL,
			IsPrimary:    img.IsPrimary,
			DisplayOrder: img.DisplayOrder,
		})
	}

	return resp
}

// List returns all active products
func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query params for filtering
	category := r.URL.Query().Get("category")
	search := r.URL.Query().Get("search")

	isActive := true
	filter := repository.ProductFilter{
		Category:   category,
		IsActive:   &isActive,
		SearchTerm: search,
	}

	products, err := h.productRepo.List(ctx, filter)
	if err != nil {
		h.logger.Error("failed to list products", "error", err)
		apierrors.Internal(w)
		return
	}

	// Load images for each product
	response := make([]productResponse, 0, len(products))
	for i := range products {
		images, err := h.imageRepo.ListByProductID(ctx, products[i].ID)
		if err != nil {
			h.logger.Warn("failed to load images for product", "product_id", products[i].ID, "error", err)
		}
		products[i].Images = images
		response = append(response, toProductResponse(&products[i]))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Get returns a single product by ID
func (h *ProductHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		apierrors.BadRequest(w, "invalid product ID")
		return
	}

	product, err := h.productRepo.GetByID(ctx, id)
	if err != nil {
		if err == repository.ErrNotFound {
			apierrors.NotFound(w, "product not found")
			return
		}
		h.logger.Error("failed to get product", "product_id", id, "error", err)
		apierrors.Internal(w)
		return
	}

	// Check if product is active
	if !product.IsActive {
		apierrors.NotFound(w, "product not found")
		return
	}

	// Load images
	images, err := h.imageRepo.ListByProductID(ctx, product.ID)
	if err != nil {
		h.logger.Warn("failed to load images for product", "product_id", product.ID, "error", err)
	}
	product.Images = images

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toProductResponse(product))
}
