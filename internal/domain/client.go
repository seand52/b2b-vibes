package domain

import (
	"time"

	"github.com/google/uuid"
)

// VATType represents the type of VAT identification number
type VATType string

const (
	VATTypeNIF VATType = "NIF" // Individuals / self-employed
	VATTypeCIF VATType = "CIF" // Companies
)

// Address represents a physical address
type Address struct {
	Street     string `json:"street"`
	City       string `json:"city"`
	PostalCode string `json:"postal_code"`
	Province   string `json:"province"`
	Country    string `json:"country"`
}

// Client represents a business client (synced from Holded contact)
type Client struct {
	ID              uuid.UUID  `json:"id"`
	HoldedID        string     `json:"holded_id"`
	Auth0ID         *string    `json:"auth0_id,omitempty"` // NULL until they sign up
	Email           string     `json:"email"`
	CompanyName     string     `json:"company_name"`
	ContactName     string     `json:"contact_name,omitempty"`
	Phone           string     `json:"phone,omitempty"`
	VATType         VATType    `json:"vat_type,omitempty"`   // NIF or CIF
	VATNumber       string     `json:"vat_number,omitempty"` // The actual number
	BillingAddress  *Address   `json:"billing_address,omitempty"`
	ShippingAddress *Address   `json:"shipping_address,omitempty"`
	IsActive        bool       `json:"is_active"`
	SyncedAt        *time.Time `json:"synced_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// IsLinked returns true if the client has linked their Auth0 account
func (c *Client) IsLinked() bool {
	return c.Auth0ID != nil && *c.Auth0ID != ""
}

// IsCompany returns true if the client is a company (CIF)
func (c *Client) IsCompany() bool {
	return c.VATType == VATTypeCIF
}
