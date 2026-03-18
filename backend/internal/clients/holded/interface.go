package holded

import "context"

// ClientInterface defines the interface for Holded API operations.
// This allows for mock implementations during development and testing.
type ClientInterface interface {
	// ListProducts fetches all products from Holded
	ListProducts(ctx context.Context) ([]Product, error)

	// GetProduct fetches a single product by ID
	GetProduct(ctx context.Context, id string) (*Product, error)

	// GetAllProductImages fetches all images for a product
	GetAllProductImages(ctx context.Context, productID string) ([]ProductImageData, error)

	// ListContacts fetches all contacts from Holded
	ListContacts(ctx context.Context) ([]Contact, error)

	// GetContact fetches a single contact by ID
	GetContact(ctx context.Context, id string) (*Contact, error)

	// CreateInvoice creates a new invoice in Holded
	CreateInvoice(ctx context.Context, req *CreateInvoiceRequest) (*Invoice, error)
}

// Ensure Client implements ClientInterface
var _ ClientInterface = (*Client)(nil)
