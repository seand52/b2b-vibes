package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"b2b-orders-api/internal/domain"
	"b2b-orders-api/internal/repository"
	"b2b-orders-api/internal/testutil"
)

func TestProductHandler_List(t *testing.T) {
	productID := uuid.New()

	tests := []struct {
		name           string
		queryParams    string
		products       []domain.Product
		listErr        error
		wantStatusCode int
		wantLen        int
	}{
		{
			name: "successful list",
			products: []domain.Product{
				{ID: productID, Name: "Product 1", Price: 10.00, IsActive: true},
				{ID: uuid.New(), Name: "Product 2", Price: 20.00, IsActive: true},
			},
			wantStatusCode: http.StatusOK,
			wantLen:        2,
		},
		{
			name:           "empty list",
			products:       []domain.Product{},
			wantStatusCode: http.StatusOK,
			wantLen:        0,
		},
		{
			name:        "with category filter",
			queryParams: "?category=electronics",
			products: []domain.Product{
				{ID: productID, Name: "Phone", Category: "electronics", Price: 500.00, IsActive: true},
			},
			wantStatusCode: http.StatusOK,
			wantLen:        1,
		},
		{
			name:           "repository error",
			listErr:        assert.AnError,
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			productRepo := &testutil.MockProductRepo{ListResult: tt.products, ListErr: tt.listErr}
			imageRepo := &testutil.MockProductImageRepo{}
			handler := NewProductHandler(productRepo, imageRepo, testutil.NewDiscardLogger())

			req := httptest.NewRequest(http.MethodGet, "/products"+tt.queryParams, nil)
			rec := httptest.NewRecorder()

			handler.List(rec, req)

			assert.Equal(t, tt.wantStatusCode, rec.Code)

			if tt.wantStatusCode == http.StatusOK {
				var response []productResponse
				err := json.NewDecoder(rec.Body).Decode(&response)
				require.NoError(t, err)
				assert.Len(t, response, tt.wantLen)
			}
		})
	}
}

func TestProductHandler_Get(t *testing.T) {
	productID := uuid.New()

	tests := []struct {
		name           string
		urlID          string
		product        *domain.Product
		getErr         error
		wantStatusCode int
	}{
		{
			name:  "successful get",
			urlID: productID.String(),
			product: &domain.Product{
				ID:       productID,
				Name:     "Test Product",
				Price:    99.99,
				IsActive: true,
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "invalid uuid",
			urlID:          "not-a-uuid",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "product not found",
			urlID:          uuid.New().String(),
			getErr:         repository.ErrNotFound,
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:  "inactive product returns not found",
			urlID: productID.String(),
			product: &domain.Product{
				ID:       productID,
				Name:     "Inactive Product",
				IsActive: false,
			},
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:           "repository error",
			urlID:          productID.String(),
			getErr:         assert.AnError,
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			productRepo := &testutil.MockProductRepo{GetByIDResult: tt.product, GetByIDErr: tt.getErr}
			imageRepo := &testutil.MockProductImageRepo{}
			handler := NewProductHandler(productRepo, imageRepo, testutil.NewDiscardLogger())

			r := chi.NewRouter()
			r.Get("/products/{id}", handler.Get)

			req := httptest.NewRequest(http.MethodGet, "/products/"+tt.urlID, nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatusCode, rec.Code)

			if tt.wantStatusCode == http.StatusOK {
				var response productResponse
				err := json.NewDecoder(rec.Body).Decode(&response)
				require.NoError(t, err)
				assert.Equal(t, tt.product.ID, response.ID)
				assert.Equal(t, tt.product.Name, response.Name)
			}
		})
	}
}

func TestProductHandler_Get_WithImages(t *testing.T) {
	productID := uuid.New()

	productRepo := &testutil.MockProductRepo{
		GetByIDResult: &domain.Product{
			ID:       productID,
			Name:     "Product with Images",
			Price:    50.00,
			IsActive: true,
		},
	}
	imageRepo := &testutil.MockProductImageRepo{
		ListByProductIDResult: []domain.ProductImage{
			{ID: uuid.New(), ProductID: productID, S3URL: "https://s3.example.com/img1.jpg", IsPrimary: true, DisplayOrder: 0},
			{ID: uuid.New(), ProductID: productID, S3URL: "https://s3.example.com/img2.jpg", IsPrimary: false, DisplayOrder: 1},
		},
	}
	handler := NewProductHandler(productRepo, imageRepo, testutil.NewDiscardLogger())

	r := chi.NewRouter()
	r.Get("/products/{id}", handler.Get)

	req := httptest.NewRequest(http.MethodGet, "/products/"+productID.String(), nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response productResponse
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)
	assert.Len(t, response.Images, 2)
	assert.Equal(t, "https://s3.example.com/img1.jpg", response.Images[0].URL)
	assert.True(t, response.Images[0].IsPrimary)
}
