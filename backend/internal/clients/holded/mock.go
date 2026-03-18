package holded

import (
	"context"
	"fmt"
	"sync/atomic"
)

// MockClient is a mock implementation of ClientInterface that returns realistic
// sample data. It's intended for local development without a real Holded account.
type MockClient struct {
	invoiceCounter atomic.Int64
}

// NewMockClient creates a new MockClient instance.
func NewMockClient() *MockClient {
	return &MockClient{}
}

// Ensure MockClient implements ClientInterface
var _ ClientInterface = (*MockClient)(nil)

// ListProducts returns a set of sample products for development.
func (m *MockClient) ListProducts(_ context.Context) ([]Product, error) {
	return []Product{
		{
			ID:          "mock-prod-001",
			SKU:         "ELEC-LAPTOP-STAND",
			Name:        "Laptop Stand Adjustable",
			Description: "Ergonomic aluminum laptop stand with adjustable height and angle",
			Price:       45.00,
			Tax:         21.0,
			Stock:       150,
			Kind:        "product",
			Tags:        []string{"electronics", "office"},
		},
		{
			ID:          "mock-prod-002",
			SKU:         "ELEC-USB-HUB",
			Name:        "USB-C Hub 7-in-1",
			Description: "Multi-port USB-C hub with HDMI, USB-A, SD card reader, and power delivery",
			Price:       35.00,
			Tax:         21.0,
			Stock:       200,
			Kind:        "product",
			Tags:        []string{"electronics", "accessories"},
		},
		{
			ID:          "mock-prod-003",
			SKU:         "ELEC-MOUSE-WL",
			Name:        "Wireless Ergonomic Mouse",
			Description: "Bluetooth wireless mouse with ergonomic design and rechargeable battery",
			Price:       25.00,
			Tax:         21.0,
			Stock:       300,
			Kind:        "product",
			Tags:        []string{"electronics", "peripherals"},
		},
		{
			ID:          "mock-prod-004",
			SKU:         "FURN-CHAIR-ERGO",
			Name:        "Ergonomic Office Chair",
			Description: "Premium ergonomic office chair with lumbar support and adjustable armrests",
			Price:       299.00,
			Tax:         21.0,
			Stock:       50,
			Kind:        "product",
			Tags:        []string{"furniture", "office"},
		},
		{
			ID:          "mock-prod-005",
			SKU:         "FURN-DESK-STAND",
			Name:        "Electric Standing Desk",
			Description: "Height-adjustable electric standing desk with memory presets",
			Price:       450.00,
			Tax:         21.0,
			Stock:       30,
			Kind:        "product",
			Tags:        []string{"furniture", "office"},
		},
		{
			ID:          "mock-prod-006",
			SKU:         "SUPP-NOTEBOOK-SET",
			Name:        "Professional Notebook Set",
			Description: "Set of 3 premium notebooks with dotted, lined, and blank pages",
			Price:       12.00,
			Tax:         21.0,
			Stock:       500,
			Kind:        "product",
			Tags:        []string{"supplies", "office"},
		},
		{
			ID:          "mock-prod-007",
			SKU:         "ELEC-LAMP-DESK",
			Name:        "LED Desk Lamp",
			Description: "Adjustable LED desk lamp with multiple brightness levels and color temperatures",
			Price:       55.00,
			Tax:         21.0,
			Stock:       120,
			Kind:        "product",
			Tags:        []string{"electronics", "lighting"},
		},
		{
			ID:          "mock-prod-008",
			SKU:         "ACCS-CABLE-ORG",
			Name:        "Cable Management Kit",
			Description: "Complete cable organizer kit with clips, ties, and sleeves",
			Price:       8.00,
			Tax:         21.0,
			Stock:       400,
			Kind:        "product",
			Tags:        []string{"accessories", "office"},
		},
	}, nil
}

// GetProduct returns a single product by ID.
func (m *MockClient) GetProduct(ctx context.Context, id string) (*Product, error) {
	products, _ := m.ListProducts(ctx)
	for _, p := range products {
		if p.ID == id {
			return &p, nil
		}
	}
	return nil, &APIError{StatusCode: 404, Message: "product not found"}
}

// GetAllProductImages returns empty images for mock products.
// In a real scenario, you could return placeholder image data here.
func (m *MockClient) GetAllProductImages(_ context.Context, _ string) ([]ProductImageData, error) {
	// Return empty slice - no mock images
	return nil, nil
}

// ListContacts returns a set of sample contacts for development.
func (m *MockClient) ListContacts(_ context.Context) ([]Contact, error) {
	return []Contact{
		{
			ID:        "mock-contact-001",
			Name:      "Acme Corporation",
			Email:     "orders@acme-corp.example.com",
			Phone:     "+34 912 345 678",
			VATNumber: "B12345678",
			Type:      "client",
			BillAddress: Address{
				Address:    "Calle Gran Vía 42",
				City:       "Madrid",
				PostalCode: "28013",
				Province:   "Madrid",
				Country:    "ES",
			},
			ShipAddress: Address{
				Address:    "Calle Gran Vía 42",
				City:       "Madrid",
				PostalCode: "28013",
				Province:   "Madrid",
				Country:    "ES",
			},
			TradeName:   "ACME Corp",
			ContactName: "María García",
		},
		{
			ID:        "mock-contact-002",
			Name:      "TechStart SL",
			Email:     "purchasing@techstart.example.com",
			Phone:     "+34 932 456 789",
			VATNumber: "B87654321",
			Type:      "client",
			BillAddress: Address{
				Address:    "Avinguda Diagonal 211",
				City:       "Barcelona",
				PostalCode: "08018",
				Province:   "Barcelona",
				Country:    "ES",
			},
			ShipAddress: Address{
				Address:    "Carrer de la Marina 15",
				City:       "Barcelona",
				PostalCode: "08005",
				Province:   "Barcelona",
				Country:    "ES",
			},
			TradeName:   "TechStart",
			ContactName: "Carlos Rodríguez",
		},
		{
			ID:        "mock-contact-003",
			Name:      "Green Solutions SA",
			Email:     "info@green-solutions.example.com",
			Phone:     "+34 963 567 890",
			VATNumber: "A11223344",
			Type:      "client",
			BillAddress: Address{
				Address:    "Calle Colón 58",
				City:       "Valencia",
				PostalCode: "46004",
				Province:   "Valencia",
				Country:    "ES",
			},
			ShipAddress: Address{
				Address:    "Polígono Industrial Norte, Nave 12",
				City:       "Paterna",
				PostalCode: "46980",
				Province:   "Valencia",
				Country:    "ES",
			},
			TradeName:   "Green Solutions",
			ContactName: "Ana Martínez",
		},
		{
			ID:        "mock-contact-004",
			Name:      "López Fernández, Juan",
			Email:     "juan.lopez@example.com",
			Phone:     "+34 954 678 901",
			VATNumber: "12345678A",
			Type:      "client",
			BillAddress: Address{
				Address:    "Calle Sierpes 23",
				City:       "Sevilla",
				PostalCode: "41004",
				Province:   "Sevilla",
				Country:    "ES",
			},
			ShipAddress: Address{
				Address:    "Calle Sierpes 23",
				City:       "Sevilla",
				PostalCode: "41004",
				Province:   "Sevilla",
				Country:    "ES",
			},
			TradeName:   "Local Shop",
			ContactName: "Juan López",
		},
	}, nil
}

// GetContact returns a single contact by ID.
func (m *MockClient) GetContact(ctx context.Context, id string) (*Contact, error) {
	contacts, _ := m.ListContacts(ctx)
	for _, c := range contacts {
		if c.ID == id {
			return &c, nil
		}
	}
	return nil, &APIError{StatusCode: 404, Message: "contact not found"}
}

// CreateInvoice returns a mock invoice with a sequential invoice number.
func (m *MockClient) CreateInvoice(_ context.Context, req *CreateInvoiceRequest) (*Invoice, error) {
	num := m.invoiceCounter.Add(1)
	return &Invoice{
		ID:         fmt.Sprintf("mock-invoice-%04d", num),
		InvoiceNum: fmt.Sprintf("MOCK-%04d", num),
		ContactID:  req.ContactID,
		Status:     1, // Draft status
	}, nil
}
