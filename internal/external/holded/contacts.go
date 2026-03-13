package holded

import (
	"context"
	"fmt"
)

// ListContacts fetches all contacts from Holded
func (c *Client) ListContacts(ctx context.Context) ([]Contact, error) {
	var contacts []Contact
	if err := c.get(ctx, "/contacts", &contacts); err != nil {
		return nil, fmt.Errorf("listing contacts: %w", err)
	}
	return contacts, nil
}

// GetContact fetches a single contact by ID
func (c *Client) GetContact(ctx context.Context, id string) (*Contact, error) {
	var contact Contact
	if err := c.get(ctx, "/contacts/"+id, &contact); err != nil {
		return nil, fmt.Errorf("getting contact %s: %w", id, err)
	}
	return &contact, nil
}
