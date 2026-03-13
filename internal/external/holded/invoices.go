package holded

import (
	"context"
	"fmt"
)

// CreateInvoice creates a new invoice in Holded
func (c *Client) CreateInvoice(ctx context.Context, req *CreateInvoiceRequest) (*Invoice, error) {
	var invoice Invoice
	if err := c.post(ctx, "/documents/invoice", req, &invoice); err != nil {
		return nil, fmt.Errorf("creating invoice: %w", err)
	}
	return &invoice, nil
}
