package holded

// Product represents a product from the Holded API
type Product struct {
	ID          string   `json:"id"`
	SKU         string   `json:"sku"`
	Name        string   `json:"name"`
	Description string   `json:"desc"`
	Price       float64  `json:"price"`
	Tax         float64  `json:"tax"`
	Stock       int      `json:"stock"`
	Kind        string   `json:"kind"` // "product" or "service"
	Tags        []string `json:"tags"`
}

// Contact represents a contact from the Holded API
type Contact struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Email        string  `json:"email"`
	Phone        string  `json:"phone"`
	VATNumber    string  `json:"vatnumber"`
	Type         string  `json:"type"`       // "client", "supplier", etc.
	BillAddress  Address `json:"billAddress"`
	ShipAddress  Address `json:"shipAddress"`
	TradeName    string  `json:"tradeName"`
	ContactName  string  `json:"contactName"`
}

// Address represents an address in Holded
type Address struct {
	Address    string `json:"address"`
	City       string `json:"city"`
	PostalCode string `json:"postalCode"`
	Province   string `json:"province"`
	Country    string `json:"country"`
}

// CreateInvoiceRequest represents a request to create an invoice
type CreateInvoiceRequest struct {
	ContactID   string        `json:"contactId"`
	Date        int64         `json:"date"`           // Required: Unix timestamp
	Description string        `json:"desc,omitempty"`
	Items       []InvoiceItem `json:"items"`
	Notes       string        `json:"notes,omitempty"`
}

// InvoiceItem represents a line item in an invoice
type InvoiceItem struct {
	Name     string  `json:"name"`
	Units    int     `json:"units"`
	Subtotal float64 `json:"subtotal"`
	Tax      float64 `json:"tax"`
}

// Invoice represents an invoice response from Holded
type Invoice struct {
	ID        string `json:"id"`
	InvoiceNum string `json:"invoiceNum"`
	ContactID string `json:"contactId"`
	Status    int    `json:"status"`
}
