package holded

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListProducts(t *testing.T) {
	tests := []struct {
		name           string
		responseStatus int
		responseBody   any
		wantProducts   int
		wantErr        bool
	}{
		{
			name:           "successful response",
			responseStatus: http.StatusOK,
			responseBody: []Product{
				{ID: "1", Name: "Product 1", SKU: "SKU1", Price: 10.00},
				{ID: "2", Name: "Product 2", SKU: "SKU2", Price: 20.00},
			},
			wantProducts: 2,
			wantErr:      false,
		},
		{
			name:           "empty list",
			responseStatus: http.StatusOK,
			responseBody:   []Product{},
			wantProducts:   0,
			wantErr:        false,
		},
		{
			name:           "server error",
			responseStatus: http.StatusInternalServerError,
			responseBody:   map[string]string{"error": "internal error"},
			wantErr:        true,
		},
		{
			name:           "unauthorized",
			responseStatus: http.StatusUnauthorized,
			responseBody:   map[string]string{"error": "invalid api key"},
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/products", r.URL.Path)
				assert.Equal(t, "test-api-key", r.Header.Get("key"))

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.responseStatus)
				json.NewEncoder(w).Encode(tt.responseBody)
			}))
			defer server.Close()

			client := NewClient(Config{
				APIKey:  "test-api-key",
				BaseURL: server.URL,
			})

			products, err := client.ListProducts(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, products, tt.wantProducts)
		})
	}
}

func TestGetProduct(t *testing.T) {
	tests := []struct {
		name           string
		productID      string
		responseStatus int
		responseBody   any
		wantErr        bool
	}{
		{
			name:           "product found",
			productID:      "123",
			responseStatus: http.StatusOK,
			responseBody:   Product{ID: "123", Name: "Test Product", SKU: "TEST", Price: 99.99},
			wantErr:        false,
		},
		{
			name:           "product not found",
			productID:      "999",
			responseStatus: http.StatusNotFound,
			responseBody:   map[string]string{"error": "not found"},
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/products/"+tt.productID, r.URL.Path)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.responseStatus)
				json.NewEncoder(w).Encode(tt.responseBody)
			}))
			defer server.Close()

			client := NewClient(Config{
				APIKey:  "test-api-key",
				BaseURL: server.URL,
			})

			product, err := client.GetProduct(context.Background(), tt.productID)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.productID, product.ID)
		})
	}
}

func TestListContacts(t *testing.T) {
	tests := []struct {
		name           string
		responseStatus int
		responseBody   any
		wantContacts   int
		wantErr        bool
	}{
		{
			name:           "successful response",
			responseStatus: http.StatusOK,
			responseBody: []Contact{
				{ID: "1", Name: "Client A", Email: "a@example.com"},
				{ID: "2", Name: "Client B", Email: "b@example.com"},
			},
			wantContacts: 2,
			wantErr:      false,
		},
		{
			name:           "empty list",
			responseStatus: http.StatusOK,
			responseBody:   []Contact{},
			wantContacts:   0,
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/contacts", r.URL.Path)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.responseStatus)
				json.NewEncoder(w).Encode(tt.responseBody)
			}))
			defer server.Close()

			client := NewClient(Config{
				APIKey:  "test-api-key",
				BaseURL: server.URL,
			})

			contacts, err := client.ListContacts(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, contacts, tt.wantContacts)
		})
	}
}

func TestCreateInvoice(t *testing.T) {
	tests := []struct {
		name           string
		request        *CreateInvoiceRequest
		responseStatus int
		responseBody   any
		wantErr        bool
	}{
		{
			name: "invoice created",
			request: &CreateInvoiceRequest{
				ContactID: "contact-123",
				Items: []InvoiceItem{
					{Name: "Product 1", Units: 2, Subtotal: 20.00, Tax: 21},
				},
			},
			responseStatus: http.StatusOK,
			responseBody:   Invoice{ID: "inv-456", InvoiceNum: "INV-2024-001"},
			wantErr:        false,
		},
		{
			name: "invalid contact",
			request: &CreateInvoiceRequest{
				ContactID: "invalid",
				Items:     []InvoiceItem{},
			},
			responseStatus: http.StatusBadRequest,
			responseBody:   map[string]string{"error": "invalid contact"},
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/documents/invoice", r.URL.Path)
				assert.Equal(t, http.MethodPost, r.Method)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.responseStatus)
				json.NewEncoder(w).Encode(tt.responseBody)
			}))
			defer server.Close()

			client := NewClient(Config{
				APIKey:  "test-api-key",
				BaseURL: server.URL,
			})

			invoice, err := client.CreateInvoice(context.Background(), tt.request)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, invoice.ID)
		})
	}
}

func TestGetAllProductImages(t *testing.T) {
	t.Run("fetches all images", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/products/123/image" {
				// List images endpoint
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode([]string{"main.jpg", "side.jpg"})
				return
			}

			// Individual image fetch
			w.Header().Set("Content-Type", "image/jpeg")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("fake-image-data"))
		}))
		defer server.Close()

		client := NewClient(Config{
			APIKey:  "test-api-key",
			BaseURL: server.URL,
		})

		images, err := client.GetAllProductImages(context.Background(), "123")

		require.NoError(t, err)
		assert.Len(t, images, 2)
		assert.Equal(t, "main.jpg", images[0].Filename)
		assert.Equal(t, "image/jpeg", images[0].ContentType)
	})

	t.Run("no images returns empty slice", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		client := NewClient(Config{
			APIKey:  "test-api-key",
			BaseURL: server.URL,
		})

		images, err := client.GetAllProductImages(context.Background(), "123")

		require.NoError(t, err)
		assert.Empty(t, images)
	})
}
