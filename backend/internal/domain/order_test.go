package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOrderStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		status OrderStatus
		want   bool
	}{
		{OrderStatusPending, false},
		{OrderStatusApproved, false},
		{OrderStatusShipped, false},
		{OrderStatusDelivered, true},
		{OrderStatusRejected, true},
		{OrderStatusCancelled, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.want, tt.status.IsTerminal())
		})
	}
}

func TestOrderStatus_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name   string
		from   OrderStatus
		to     OrderStatus
		canDo  bool
	}{
		// From pending
		{"pending to approved", OrderStatusPending, OrderStatusApproved, true},
		{"pending to rejected", OrderStatusPending, OrderStatusRejected, true},
		{"pending to cancelled", OrderStatusPending, OrderStatusCancelled, true},
		{"pending to shipped", OrderStatusPending, OrderStatusShipped, false},
		{"pending to delivered", OrderStatusPending, OrderStatusDelivered, false},

		// From approved
		{"approved to shipped", OrderStatusApproved, OrderStatusShipped, true},
		{"approved to cancelled", OrderStatusApproved, OrderStatusCancelled, true},
		{"approved to pending", OrderStatusApproved, OrderStatusPending, false},
		{"approved to delivered", OrderStatusApproved, OrderStatusDelivered, false},

		// From shipped
		{"shipped to delivered", OrderStatusShipped, OrderStatusDelivered, true},
		{"shipped to cancelled", OrderStatusShipped, OrderStatusCancelled, false},
		{"shipped to pending", OrderStatusShipped, OrderStatusPending, false},

		// Terminal states
		{"delivered to anything", OrderStatusDelivered, OrderStatusCancelled, false},
		{"rejected to anything", OrderStatusRejected, OrderStatusPending, false},
		{"cancelled to anything", OrderStatusCancelled, OrderStatusPending, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.canDo, tt.from.CanTransitionTo(tt.to))
		})
	}
}

func TestOrder_IsCancellable(t *testing.T) {
	tests := []struct {
		status OrderStatus
		want   bool
	}{
		{OrderStatusPending, true},
		{OrderStatusApproved, false},
		{OrderStatusShipped, false},
		{OrderStatusDelivered, false},
		{OrderStatusRejected, false},
		{OrderStatusCancelled, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			o := &Order{Status: tt.status}
			assert.Equal(t, tt.want, o.IsCancellable())
		})
	}
}

func TestOrder_IsApprovable(t *testing.T) {
	tests := []struct {
		status OrderStatus
		want   bool
	}{
		{OrderStatusPending, true},
		{OrderStatusApproved, false},
		{OrderStatusShipped, false},
		{OrderStatusDelivered, false},
		{OrderStatusRejected, false},
		{OrderStatusCancelled, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			o := &Order{Status: tt.status}
			assert.Equal(t, tt.want, o.IsApprovable())
		})
	}
}
