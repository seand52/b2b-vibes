package order

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"b2b-orders-api/internal/domain"
	"b2b-orders-api/internal/clients/holded"
	"b2b-orders-api/internal/repository"
	"b2b-orders-api/internal/testutil"
)

func TestService_Create(t *testing.T) {
	productID := uuid.New()
	clientID := uuid.New()

	product := &domain.Product{
		ID:               productID,
		Name:             "Test Product",
		Price:            10.00,
		StockQuantity:    100,
		MinOrderQuantity: 1,
		IsActive:         true,
	}

	tests := []struct {
		name       string
		req        CreateOrderRequest
		products   map[uuid.UUID]*domain.Product
		createErr  error
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "successful order creation",
			req: CreateOrderRequest{
				ClientID: clientID,
				Items: []OrderItemRequest{
					{ProductID: productID, Quantity: 5},
				},
				Notes: "Test order",
			},
			products: map[uuid.UUID]*domain.Product{productID: product},
		},
		{
			name: "empty items",
			req: CreateOrderRequest{
				ClientID: clientID,
				Items:    []OrderItemRequest{},
			},
			wantErr:    true,
			wantErrMsg: "at least one item",
		},
		{
			name: "product not found",
			req: CreateOrderRequest{
				ClientID: clientID,
				Items: []OrderItemRequest{
					{ProductID: uuid.New(), Quantity: 1},
				},
			},
			products:   map[uuid.UUID]*domain.Product{},
			wantErr:    true,
			wantErrMsg: "product not found",
		},
		{
			name: "insufficient stock",
			req: CreateOrderRequest{
				ClientID: clientID,
				Items: []OrderItemRequest{
					{ProductID: productID, Quantity: 500},
				},
			},
			products: map[uuid.UUID]*domain.Product{productID: product},
			wantErr:  true,
		},
		{
			name: "below minimum order quantity",
			req: CreateOrderRequest{
				ClientID: clientID,
				Items: []OrderItemRequest{
					{ProductID: productID, Quantity: 0},
				},
			},
			products:   map[uuid.UUID]*domain.Product{productID: product},
			wantErr:    true,
			wantErrMsg: "invalid quantity",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderRepo := &testutil.MockOrderRepo{CreateErr: tt.createErr}

			// Create a product repo that returns products from the map
			var productResult *domain.Product
			var productErr error
			if len(tt.req.Items) > 0 {
				p, ok := tt.products[tt.req.Items[0].ProductID]
				if ok {
					productResult = p
				} else {
					productErr = repository.ErrNotFound
				}
			}
			productRepo := &testutil.MockProductRepo{
				GetByIDResult: productResult,
				GetByIDErr:    productErr,
			}
			clientRepo := &testutil.MockClientRepo{}

			svc := NewService(orderRepo, productRepo, clientRepo, nil, testutil.NewDiscardLogger())
			order, err := svc.Create(context.Background(), tt.req)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrMsg != "" {
					assert.Contains(t, err.Error(), tt.wantErrMsg)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.req.ClientID, order.ClientID)
			assert.Equal(t, domain.OrderStatusPending, order.Status)
			assert.Len(t, order.Items, len(tt.req.Items))
		})
	}
}

func TestService_Cancel(t *testing.T) {
	orderID := uuid.New()
	clientID := uuid.New()
	otherClientID := uuid.New()

	tests := []struct {
		name        string
		orderID     uuid.UUID
		clientID    uuid.UUID
		order       *domain.Order
		getErr      error
		updateErr   error
		wantErr     bool
		wantErrType error
	}{
		{
			name:     "successful cancellation",
			orderID:  orderID,
			clientID: clientID,
			order: &domain.Order{
				ID:       orderID,
				ClientID: clientID,
				Status:   domain.OrderStatusPending,
			},
		},
		{
			name:     "order not found",
			orderID:  orderID,
			clientID: clientID,
			getErr:   repository.ErrNotFound,
			wantErr:  true,
		},
		{
			name:     "not owner",
			orderID:  orderID,
			clientID: otherClientID,
			order: &domain.Order{
				ID:       orderID,
				ClientID: clientID,
				Status:   domain.OrderStatusPending,
			},
			wantErr:     true,
			wantErrType: repository.ErrNotFound,
		},
		{
			name:     "order not pending",
			orderID:  orderID,
			clientID: clientID,
			order: &domain.Order{
				ID:       orderID,
				ClientID: clientID,
				Status:   domain.OrderStatusApproved,
			},
			wantErr:     true,
			wantErrType: ErrOrderNotPending,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderRepo := &testutil.MockOrderRepo{
				GetByIDResult:   tt.order,
				GetByIDErr:      tt.getErr,
				UpdateStatusErr: tt.updateErr,
			}

			svc := NewService(orderRepo, nil, nil, nil, testutil.NewDiscardLogger())
			err := svc.Cancel(context.Background(), tt.orderID, tt.clientID)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrType != nil {
					assert.ErrorIs(t, err, tt.wantErrType)
				}
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestService_Approve(t *testing.T) {
	orderID := uuid.New()
	clientID := uuid.New()
	productID := uuid.New()

	pendingOrder := &domain.Order{
		ID:       orderID,
		ClientID: clientID,
		Status:   domain.OrderStatusPending,
		Items: []domain.OrderItem{
			{ProductID: productID, Quantity: 2},
		},
	}

	client := &domain.Client{
		ID:       clientID,
		HoldedID: "holded-client-123",
	}

	product := &domain.Product{
		ID:    productID,
		Name:  "Test Product",
		Price: 50.00,
	}

	tests := []struct {
		name       string
		orderID    uuid.UUID
		approvedBy string
		order      *domain.Order
		client     *domain.Client
		product    *domain.Product
		invoice    *holded.Invoice
		invoiceErr error
		wantErr    bool
	}{
		{
			name:       "successful approval",
			orderID:    orderID,
			approvedBy: "admin@example.com",
			order:      pendingOrder,
			client:     client,
			product:    product,
			invoice:    &holded.Invoice{ID: "inv-123", InvoiceNum: "INV-001"},
		},
		{
			name:       "order not pending",
			orderID:    orderID,
			approvedBy: "admin@example.com",
			order: &domain.Order{
				ID:     orderID,
				Status: domain.OrderStatusApproved,
			},
			wantErr: true,
		},
		{
			name:       "holded invoice creation fails",
			orderID:    orderID,
			approvedBy: "admin@example.com",
			order:      pendingOrder,
			client:     client,
			product:    product,
			invoiceErr: errors.New("holded api error"),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderRepo := &testutil.MockOrderRepo{
				GetByIDResult: tt.order,
			}
			productRepo := &testutil.MockProductRepo{GetByIDResult: tt.product}
			clientRepo := &testutil.MockClientRepo{GetByIDResult: tt.client}
			holdedClient := &testutil.MockHoldedClient{
				CreateInvoiceResult: tt.invoice,
				CreateInvoiceErr:    tt.invoiceErr,
			}

			svc := &Service{
				orderRepo:   orderRepo,
				productRepo: productRepo,
				clientRepo:  clientRepo,
				holded:      holdedClient,
				logger:      testutil.NewDiscardLogger(),
			}

			result, err := svc.Approve(context.Background(), tt.orderID, tt.approvedBy)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}

func TestService_Reject(t *testing.T) {
	orderID := uuid.New()

	tests := []struct {
		name      string
		order     *domain.Order
		reason    string
		rejectErr error
		wantErr   bool
	}{
		{
			name: "successful rejection",
			order: &domain.Order{
				ID:     orderID,
				Status: domain.OrderStatusPending,
			},
			reason: "Out of stock",
		},
		{
			name: "order not pending",
			order: &domain.Order{
				ID:     orderID,
				Status: domain.OrderStatusApproved,
			},
			reason:  "Too late",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orderRepo := &testutil.MockOrderRepo{
				GetByIDResult: tt.order,
				RejectErr:     tt.rejectErr,
			}

			svc := NewService(orderRepo, nil, nil, nil, testutil.NewDiscardLogger())
			err := svc.Reject(context.Background(), orderID, tt.reason)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}
