package domain

import (
	"time"

	"github.com/google/uuid"
)

// OrderStatus represents the status of an order
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusApproved  OrderStatus = "approved"
	OrderStatusRejected  OrderStatus = "rejected"
	OrderStatusShipped   OrderStatus = "shipped"
	OrderStatusDelivered OrderStatus = "delivered"
	OrderStatusCancelled OrderStatus = "cancelled"
)

// IsTerminal returns true if the order is in a terminal state
func (s OrderStatus) IsTerminal() bool {
	return s == OrderStatusDelivered || s == OrderStatusCancelled || s == OrderStatusRejected
}

// CanTransitionTo returns true if the order can transition to the target status
func (s OrderStatus) CanTransitionTo(target OrderStatus) bool {
	transitions := map[OrderStatus][]OrderStatus{
		OrderStatusPending:   {OrderStatusApproved, OrderStatusRejected, OrderStatusCancelled},
		OrderStatusApproved:  {OrderStatusShipped, OrderStatusCancelled},
		OrderStatusShipped:   {OrderStatusDelivered},
		OrderStatusDelivered: {},
		OrderStatusRejected:  {},
		OrderStatusCancelled: {},
	}

	allowed, ok := transitions[s]
	if !ok {
		return false
	}

	for _, status := range allowed {
		if status == target {
			return true
		}
	}
	return false
}

// Order represents a client order
type Order struct {
	ID              uuid.UUID   `json:"id"`
	ClientID        uuid.UUID   `json:"client_id"`
	Status          OrderStatus `json:"status"`
	Notes           string      `json:"notes,omitempty"`
	AdminNotes      string      `json:"admin_notes,omitempty"`
	HoldedInvoiceID *string     `json:"holded_invoice_id,omitempty"`
	ApprovedAt      *time.Time  `json:"approved_at,omitempty"`
	ApprovedBy      *string     `json:"approved_by,omitempty"`
	RejectedAt      *time.Time  `json:"rejected_at,omitempty"`
	RejectionReason *string     `json:"rejection_reason,omitempty"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
	Items           []OrderItem `json:"items,omitempty"`
}

// OrderItem represents a line item in an order
type OrderItem struct {
	ID        uuid.UUID `json:"id"`
	OrderID   uuid.UUID `json:"order_id"`
	ProductID uuid.UUID `json:"product_id"`
	Quantity  int       `json:"quantity"`
}

// IsCancellable returns true if the order can be cancelled by the client
func (o *Order) IsCancellable() bool {
	return o.Status == OrderStatusPending
}

// IsApprovable returns true if the order can be approved by admin
func (o *Order) IsApprovable() bool {
	return o.Status == OrderStatusPending
}
