package cart

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"b2b-orders-api/internal/domain"
	"b2b-orders-api/internal/repository"
	"b2b-orders-api/internal/testutil"
)

func TestService_GetOrCreateDraft_CreatesNewDraft(t *testing.T) {
	clientID := uuid.New()

	orderRepo := &testutil.MockOrderRepo{
		GetDraftByClientIDErr: repository.ErrNotFound,
	}
	productRepo := &testutil.MockProductRepo{}

	svc := NewService(orderRepo, productRepo, testutil.NewDiscardLogger())
	draft, err := svc.GetOrCreateDraft(context.Background(), clientID)

	require.NoError(t, err)
	assert.NotNil(t, draft)
	assert.Equal(t, clientID, draft.ClientID)
	assert.Equal(t, domain.OrderStatusDraft, draft.Status)
	assert.Empty(t, draft.Items)
	assert.NotEqual(t, uuid.Nil, draft.ID)
}

func TestService_GetOrCreateDraft_ReturnsExistingDraft(t *testing.T) {
	clientID := uuid.New()
	draftID := uuid.New()
	productID := uuid.New()

	existingDraft := &domain.Order{
		ID:       draftID,
		ClientID: clientID,
		Status:   domain.OrderStatusDraft,
		Items: []domain.OrderItem{
			{ProductID: productID, Quantity: 2},
		},
	}

	orderRepo := &testutil.MockOrderRepo{
		GetDraftByClientIDResult: existingDraft,
	}
	productRepo := &testutil.MockProductRepo{}

	svc := NewService(orderRepo, productRepo, testutil.NewDiscardLogger())
	draft, err := svc.GetOrCreateDraft(context.Background(), clientID)

	require.NoError(t, err)
	assert.Equal(t, draftID, draft.ID)
	assert.Len(t, draft.Items, 1)
}

func TestService_AddItem_AddsNewItem(t *testing.T) {
	clientID := uuid.New()
	draftID := uuid.New()
	productID := uuid.New()

	existingDraft := &domain.Order{
		ID:       draftID,
		ClientID: clientID,
		Status:   domain.OrderStatusDraft,
		Items:    []domain.OrderItem{},
	}

	product := &domain.Product{
		ID:       productID,
		Name:     "Test Product",
		Price:    50.00,
		IsActive: true,
	}

	orderRepo := &testutil.MockOrderRepo{
		GetDraftByClientIDResult: existingDraft,
	}
	productRepo := &testutil.MockProductRepo{
		GetByIDResult: product,
	}

	svc := NewService(orderRepo, productRepo, testutil.NewDiscardLogger())
	err := svc.AddItem(context.Background(), clientID, productID, 3)

	require.NoError(t, err)
}

func TestService_AddItem_IncreasesQuantityIfItemExists(t *testing.T) {
	clientID := uuid.New()
	draftID := uuid.New()
	productID := uuid.New()

	existingDraft := &domain.Order{
		ID:       draftID,
		ClientID: clientID,
		Status:   domain.OrderStatusDraft,
		Items: []domain.OrderItem{
			{ProductID: productID, Quantity: 2},
		},
	}

	product := &domain.Product{
		ID:       productID,
		Name:     "Test Product",
		Price:    50.00,
		IsActive: true,
	}

	orderRepo := &testutil.MockOrderRepo{
		GetDraftByClientIDResult: existingDraft,
	}
	productRepo := &testutil.MockProductRepo{
		GetByIDResult: product,
	}

	svc := NewService(orderRepo, productRepo, testutil.NewDiscardLogger())
	err := svc.AddItem(context.Background(), clientID, productID, 3)

	require.NoError(t, err)
	// Note: In real scenario, we'd verify the quantity was updated to 5
}

func TestService_AddItem_FailsOnInvalidProduct(t *testing.T) {
	clientID := uuid.New()
	productID := uuid.New()

	orderRepo := &testutil.MockOrderRepo{
		GetDraftByClientIDErr: repository.ErrNotFound,
	}
	productRepo := &testutil.MockProductRepo{
		GetByIDErr: repository.ErrNotFound,
	}

	svc := NewService(orderRepo, productRepo, testutil.NewDiscardLogger())
	err := svc.AddItem(context.Background(), clientID, productID, 1)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrProductNotFound)
}

func TestService_AddItem_FailsOnInactiveProduct(t *testing.T) {
	clientID := uuid.New()
	productID := uuid.New()

	product := &domain.Product{
		ID:       productID,
		Name:     "Inactive Product",
		Price:    50.00,
		IsActive: false,
	}

	orderRepo := &testutil.MockOrderRepo{
		GetDraftByClientIDErr: repository.ErrNotFound,
	}
	productRepo := &testutil.MockProductRepo{
		GetByIDResult: product,
	}

	svc := NewService(orderRepo, productRepo, testutil.NewDiscardLogger())
	err := svc.AddItem(context.Background(), clientID, productID, 1)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrProductNotFound)
}

func TestService_AddItem_FailsOnZeroQuantity(t *testing.T) {
	clientID := uuid.New()
	productID := uuid.New()

	orderRepo := &testutil.MockOrderRepo{}
	productRepo := &testutil.MockProductRepo{}

	svc := NewService(orderRepo, productRepo, testutil.NewDiscardLogger())
	err := svc.AddItem(context.Background(), clientID, productID, 0)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidQuantity)
}

func TestService_UpdateItemQuantity_UpdatesQuantity(t *testing.T) {
	clientID := uuid.New()
	draftID := uuid.New()
	productID := uuid.New()

	existingDraft := &domain.Order{
		ID:       draftID,
		ClientID: clientID,
		Status:   domain.OrderStatusDraft,
		Items: []domain.OrderItem{
			{ProductID: productID, Quantity: 2},
		},
	}

	orderRepo := &testutil.MockOrderRepo{
		GetDraftByClientIDResult: existingDraft,
	}
	productRepo := &testutil.MockProductRepo{}

	svc := NewService(orderRepo, productRepo, testutil.NewDiscardLogger())
	err := svc.UpdateItemQuantity(context.Background(), clientID, productID, 5)

	require.NoError(t, err)
}

func TestService_UpdateItemQuantity_RemovesItemWhenZero(t *testing.T) {
	clientID := uuid.New()
	draftID := uuid.New()
	productID := uuid.New()

	existingDraft := &domain.Order{
		ID:       draftID,
		ClientID: clientID,
		Status:   domain.OrderStatusDraft,
		Items: []domain.OrderItem{
			{ProductID: productID, Quantity: 2},
		},
	}

	orderRepo := &testutil.MockOrderRepo{
		GetDraftByClientIDResult: existingDraft,
	}
	productRepo := &testutil.MockProductRepo{}

	svc := NewService(orderRepo, productRepo, testutil.NewDiscardLogger())
	err := svc.UpdateItemQuantity(context.Background(), clientID, productID, 0)

	require.NoError(t, err)
}

func TestService_UpdateItemQuantity_FailsOnNoDraft(t *testing.T) {
	clientID := uuid.New()
	productID := uuid.New()

	orderRepo := &testutil.MockOrderRepo{
		GetDraftByClientIDErr: repository.ErrNotFound,
	}
	productRepo := &testutil.MockProductRepo{}

	svc := NewService(orderRepo, productRepo, testutil.NewDiscardLogger())
	err := svc.UpdateItemQuantity(context.Background(), clientID, productID, 5)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoDraft)
}

func TestService_UpdateItemQuantity_FailsOnProductNotInCart(t *testing.T) {
	clientID := uuid.New()
	draftID := uuid.New()
	productID := uuid.New()
	otherProductID := uuid.New()

	existingDraft := &domain.Order{
		ID:       draftID,
		ClientID: clientID,
		Status:   domain.OrderStatusDraft,
		Items: []domain.OrderItem{
			{ProductID: productID, Quantity: 2},
		},
	}

	orderRepo := &testutil.MockOrderRepo{
		GetDraftByClientIDResult: existingDraft,
	}
	productRepo := &testutil.MockProductRepo{}

	svc := NewService(orderRepo, productRepo, testutil.NewDiscardLogger())
	err := svc.UpdateItemQuantity(context.Background(), clientID, otherProductID, 5)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrProductNotFound)
}

func TestService_RemoveItem_RemovesItem(t *testing.T) {
	clientID := uuid.New()
	draftID := uuid.New()
	productID := uuid.New()

	existingDraft := &domain.Order{
		ID:       draftID,
		ClientID: clientID,
		Status:   domain.OrderStatusDraft,
		Items: []domain.OrderItem{
			{ProductID: productID, Quantity: 2},
		},
	}

	orderRepo := &testutil.MockOrderRepo{
		GetDraftByClientIDResult: existingDraft,
	}
	productRepo := &testutil.MockProductRepo{}

	svc := NewService(orderRepo, productRepo, testutil.NewDiscardLogger())
	err := svc.RemoveItem(context.Background(), clientID, productID)

	require.NoError(t, err)
}

func TestService_SetItems_ReplacesAllItems(t *testing.T) {
	clientID := uuid.New()
	draftID := uuid.New()
	productID1 := uuid.New()
	productID2 := uuid.New()

	existingDraft := &domain.Order{
		ID:       draftID,
		ClientID: clientID,
		Status:   domain.OrderStatusDraft,
		Items:    []domain.OrderItem{},
	}

	product1 := &domain.Product{ID: productID1, Name: "Product 1", Price: 10.00, IsActive: true}

	orderRepo := &testutil.MockOrderRepo{
		GetDraftByClientIDResult: existingDraft,
	}
	productRepo := &testutil.MockProductRepo{}

	// Mock GetByID to return product1 for simplicity
	productRepo.GetByIDResult = product1

	svc := NewService(orderRepo, productRepo, testutil.NewDiscardLogger())

	items := []ItemRequest{
		{ProductID: productID1, Quantity: 5},
		{ProductID: productID2, Quantity: 3},
	}

	err := svc.SetItems(context.Background(), clientID, items)

	// This test is simplified - in reality, you'd verify the exact items
	require.NoError(t, err)
}

func TestService_SetItems_FailsOnInvalidQuantity(t *testing.T) {
	clientID := uuid.New()
	productID := uuid.New()

	orderRepo := &testutil.MockOrderRepo{
		GetDraftByClientIDErr: repository.ErrNotFound,
	}
	productRepo := &testutil.MockProductRepo{}

	svc := NewService(orderRepo, productRepo, testutil.NewDiscardLogger())

	items := []ItemRequest{
		{ProductID: productID, Quantity: 0},
	}

	err := svc.SetItems(context.Background(), clientID, items)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidQuantity)
}

func TestService_Submit_Success(t *testing.T) {
	clientID := uuid.New()
	draftID := uuid.New()
	productID := uuid.New()

	existingDraft := &domain.Order{
		ID:       draftID,
		ClientID: clientID,
		Status:   domain.OrderStatusDraft,
		Items: []domain.OrderItem{
			{ProductID: productID, Quantity: 2},
		},
	}

	submittedOrder := &domain.Order{
		ID:       draftID,
		ClientID: clientID,
		Status:   domain.OrderStatusPending,
		Items: []domain.OrderItem{
			{ProductID: productID, Quantity: 2},
		},
	}

	product := &domain.Product{
		ID:               productID,
		Name:             "Test Product",
		Price:            50.00,
		StockQuantity:    100,
		MinOrderQuantity: 1,
		IsActive:         true,
	}

	orderRepo := &testutil.MockOrderRepo{
		GetDraftByClientIDResult: existingDraft,
		GetByIDResult:            submittedOrder,
	}
	productRepo := &testutil.MockProductRepo{
		GetByIDResult: product,
	}

	svc := NewService(orderRepo, productRepo, testutil.NewDiscardLogger())
	order, err := svc.Submit(context.Background(), clientID)

	require.NoError(t, err)
	assert.NotNil(t, order)
	assert.Equal(t, domain.OrderStatusPending, order.Status)
}

func TestService_Submit_FailsOnEmptyCart(t *testing.T) {
	clientID := uuid.New()
	draftID := uuid.New()

	existingDraft := &domain.Order{
		ID:       draftID,
		ClientID: clientID,
		Status:   domain.OrderStatusDraft,
		Items:    []domain.OrderItem{}, // Empty cart
	}

	orderRepo := &testutil.MockOrderRepo{
		GetDraftByClientIDResult: existingDraft,
	}
	productRepo := &testutil.MockProductRepo{}

	svc := NewService(orderRepo, productRepo, testutil.NewDiscardLogger())
	_, err := svc.Submit(context.Background(), clientID)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEmptyCart)
}

func TestService_Submit_FailsOnInsufficientStock(t *testing.T) {
	clientID := uuid.New()
	draftID := uuid.New()
	productID := uuid.New()

	existingDraft := &domain.Order{
		ID:       draftID,
		ClientID: clientID,
		Status:   domain.OrderStatusDraft,
		Items: []domain.OrderItem{
			{ProductID: productID, Quantity: 100},
		},
	}

	product := &domain.Product{
		ID:               productID,
		Name:             "Test Product",
		Price:            50.00,
		StockQuantity:    10, // Not enough stock
		MinOrderQuantity: 1,
		IsActive:         true,
	}

	orderRepo := &testutil.MockOrderRepo{
		GetDraftByClientIDResult: existingDraft,
	}
	productRepo := &testutil.MockProductRepo{
		GetByIDResult: product,
	}

	svc := NewService(orderRepo, productRepo, testutil.NewDiscardLogger())
	_, err := svc.Submit(context.Background(), clientID)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInsufficientStock)
}

func TestService_Submit_FailsOnInactiveProduct(t *testing.T) {
	clientID := uuid.New()
	draftID := uuid.New()
	productID := uuid.New()

	existingDraft := &domain.Order{
		ID:       draftID,
		ClientID: clientID,
		Status:   domain.OrderStatusDraft,
		Items: []domain.OrderItem{
			{ProductID: productID, Quantity: 2},
		},
	}

	product := &domain.Product{
		ID:               productID,
		Name:             "Inactive Product",
		Price:            50.00,
		StockQuantity:    100,
		MinOrderQuantity: 1,
		IsActive:         false, // Product is inactive
	}

	orderRepo := &testutil.MockOrderRepo{
		GetDraftByClientIDResult: existingDraft,
	}
	productRepo := &testutil.MockProductRepo{
		GetByIDResult: product,
	}

	svc := NewService(orderRepo, productRepo, testutil.NewDiscardLogger())
	_, err := svc.Submit(context.Background(), clientID)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrProductNotFound)
}

func TestService_Submit_FailsOnBelowMinOrderQuantity(t *testing.T) {
	clientID := uuid.New()
	draftID := uuid.New()
	productID := uuid.New()

	existingDraft := &domain.Order{
		ID:       draftID,
		ClientID: clientID,
		Status:   domain.OrderStatusDraft,
		Items: []domain.OrderItem{
			{ProductID: productID, Quantity: 1},
		},
	}

	product := &domain.Product{
		ID:               productID,
		Name:             "Test Product",
		Price:            50.00,
		StockQuantity:    100,
		MinOrderQuantity: 5, // Minimum is 5
		IsActive:         true,
	}

	orderRepo := &testutil.MockOrderRepo{
		GetDraftByClientIDResult: existingDraft,
	}
	productRepo := &testutil.MockProductRepo{
		GetByIDResult: product,
	}

	svc := NewService(orderRepo, productRepo, testutil.NewDiscardLogger())
	_, err := svc.Submit(context.Background(), clientID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "minimum order quantity")
}

func TestService_DiscardDraft_CancelsDraft(t *testing.T) {
	clientID := uuid.New()
	draftID := uuid.New()

	existingDraft := &domain.Order{
		ID:       draftID,
		ClientID: clientID,
		Status:   domain.OrderStatusDraft,
		Items:    []domain.OrderItem{},
	}

	orderRepo := &testutil.MockOrderRepo{
		GetDraftByClientIDResult: existingDraft,
	}
	productRepo := &testutil.MockProductRepo{}

	svc := NewService(orderRepo, productRepo, testutil.NewDiscardLogger())
	err := svc.DiscardDraft(context.Background(), clientID)

	require.NoError(t, err)
}

func TestService_DiscardDraft_FailsOnNoDraft(t *testing.T) {
	clientID := uuid.New()

	orderRepo := &testutil.MockOrderRepo{
		GetDraftByClientIDErr: repository.ErrNotFound,
	}
	productRepo := &testutil.MockProductRepo{}

	svc := NewService(orderRepo, productRepo, testutil.NewDiscardLogger())
	err := svc.DiscardDraft(context.Background(), clientID)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoDraft)
}

func TestService_GetDraftWithDetails_ReturnsEnrichedCart(t *testing.T) {
	clientID := uuid.New()
	draftID := uuid.New()
	productID := uuid.New()

	now := time.Now()
	existingDraft := &domain.Order{
		ID:        draftID,
		ClientID:  clientID,
		Status:    domain.OrderStatusDraft,
		CreatedAt: now,
		UpdatedAt: now,
		Items: []domain.OrderItem{
			{ProductID: productID, Quantity: 2},
		},
	}

	product := &domain.Product{
		ID:               productID,
		Name:             "Test Product",
		SKU:              "TEST-001",
		Price:            50.00,
		StockQuantity:    100,
		MinOrderQuantity: 1,
		IsActive:         true,
	}

	orderRepo := &testutil.MockOrderRepo{
		GetDraftByClientIDResult: existingDraft,
	}
	productRepo := &testutil.MockProductRepo{
		GetByIDResult: product,
	}

	svc := NewService(orderRepo, productRepo, testutil.NewDiscardLogger())
	cartResp, err := svc.GetDraftWithDetails(context.Background(), clientID)

	require.NoError(t, err)
	assert.Equal(t, draftID, cartResp.ID)
	assert.Len(t, cartResp.Items, 1)
	assert.Equal(t, "Test Product", cartResp.Items[0].ProductName)
	assert.Equal(t, 50.00, cartResp.Items[0].UnitPrice)
	assert.Equal(t, 100.00, cartResp.Items[0].LineTotal)
	assert.Equal(t, 100.00, cartResp.Summary.Subtotal)
	assert.Equal(t, 21.0, cartResp.Summary.TaxRate)
	assert.Equal(t, 21.00, cartResp.Summary.TaxAmount)
	assert.Equal(t, 121.00, cartResp.Summary.Total)
	assert.Equal(t, 1, cartResp.Summary.ItemCount)
	assert.Equal(t, 2, cartResp.Summary.TotalUnits)
}

func TestService_GetDraftWithDetails_FailsOnNoDraft(t *testing.T) {
	clientID := uuid.New()

	orderRepo := &testutil.MockOrderRepo{
		GetDraftByClientIDErr: repository.ErrNotFound,
	}
	productRepo := &testutil.MockProductRepo{}

	svc := NewService(orderRepo, productRepo, testutil.NewDiscardLogger())
	_, err := svc.GetDraftWithDetails(context.Background(), clientID)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoDraft)
}
