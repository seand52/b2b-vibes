package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"b2b-orders-api/internal/domain"
	"b2b-orders-api/internal/repository"
)

// OrderRepository implements repository.OrderRepository
type OrderRepository struct {
	db *pgxpool.Pool
}

// NewOrderRepository creates a new OrderRepository
func NewOrderRepository(db *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{db: db}
}

// Create inserts a new order with its items
func (r *OrderRepository) Create(ctx context.Context, order *domain.Order) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	now := time.Now()

	if order.ID == uuid.Nil {
		order.ID = uuid.New()
	}
	order.Status = domain.OrderStatusPending
	order.CreatedAt = now
	order.UpdatedAt = now

	orderQuery := `
		INSERT INTO orders (id, client_id, status, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err = tx.Exec(ctx, orderQuery,
		order.ID,
		order.ClientID,
		order.Status,
		order.Notes,
		order.CreatedAt,
		order.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting order: %w", err)
	}

	itemQuery := `
		INSERT INTO order_items (id, order_id, product_id, quantity)
		VALUES ($1, $2, $3, $4)`

	for i := range order.Items {
		item := &order.Items[i]
		if item.ID == uuid.Nil {
			item.ID = uuid.New()
		}
		item.OrderID = order.ID

		_, err = tx.Exec(ctx, itemQuery,
			item.ID,
			item.OrderID,
			item.ProductID,
			item.Quantity,
		)
		if err != nil {
			return fmt.Errorf("inserting order item: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

// GetByID retrieves an order by its UUID, including items
func (r *OrderRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	query := `
		SELECT id, client_id, status, notes, admin_notes, holded_invoice_id, approved_at, approved_by, rejected_at, rejection_reason, created_at, updated_at
		FROM orders
		WHERE id = $1`

	var o domain.Order
	err := r.db.QueryRow(ctx, query, id).Scan(
		&o.ID,
		&o.ClientID,
		&o.Status,
		&o.Notes,
		&o.AdminNotes,
		&o.HoldedInvoiceID,
		&o.ApprovedAt,
		&o.ApprovedBy,
		&o.RejectedAt,
		&o.RejectionReason,
		&o.CreatedAt,
		&o.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scanning order: %w", err)
	}

	items, err := r.getOrderItems(ctx, o.ID)
	if err != nil {
		return nil, err
	}
	o.Items = items

	return &o, nil
}

func (r *OrderRepository) getOrderItems(ctx context.Context, orderID uuid.UUID) ([]domain.OrderItem, error) {
	query := `
		SELECT id, order_id, product_id, quantity
		FROM order_items
		WHERE order_id = $1`

	rows, err := r.db.Query(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("querying order items: %w", err)
	}
	defer rows.Close()

	var items []domain.OrderItem
	for rows.Next() {
		var item domain.OrderItem
		err := rows.Scan(
			&item.ID,
			&item.OrderID,
			&item.ProductID,
			&item.Quantity,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning order item: %w", err)
		}
		items = append(items, item)
	}

	return items, nil
}

// ListByClientID retrieves orders for a specific client
func (r *OrderRepository) ListByClientID(ctx context.Context, clientID uuid.UUID, filter repository.OrderFilter) ([]domain.Order, error) {
	filter.ClientID = &clientID
	return r.List(ctx, filter)
}

// List retrieves orders with optional filtering
func (r *OrderRepository) List(ctx context.Context, filter repository.OrderFilter) ([]domain.Order, error) {
	var conditions []string
	var args []any
	argIdx := 1

	if filter.ClientID != nil {
		conditions = append(conditions, fmt.Sprintf("client_id = $%d", argIdx))
		args = append(args, *filter.ClientID)
		argIdx++
	}

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}

	query := `SELECT id, client_id, status, notes, admin_notes, holded_invoice_id, approved_at, approved_by, rejected_at, rejection_reason, created_at, updated_at FROM orders`

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filter.Offset)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying orders: %w", err)
	}
	defer rows.Close()

	var orders []domain.Order
	for rows.Next() {
		var o domain.Order
		err := rows.Scan(
			&o.ID,
			&o.ClientID,
			&o.Status,
			&o.Notes,
			&o.AdminNotes,
			&o.HoldedInvoiceID,
			&o.ApprovedAt,
			&o.ApprovedBy,
			&o.RejectedAt,
			&o.RejectionReason,
			&o.CreatedAt,
			&o.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning order row: %w", err)
		}
		orders = append(orders, o)
	}

	return orders, nil
}

// UpdateStatus updates the status of an order
func (r *OrderRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.OrderStatus) error {
	query := `UPDATE orders SET status = $1, updated_at = $2 WHERE id = $3`

	result, err := r.db.Exec(ctx, query, status, time.Now(), id)
	if err != nil {
		return fmt.Errorf("updating order status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return repository.ErrNotFound
	}

	return nil
}

// SetHoldedInvoiceID sets the Holded invoice ID for an order
func (r *OrderRepository) SetHoldedInvoiceID(ctx context.Context, id uuid.UUID, invoiceID string) error {
	query := `UPDATE orders SET holded_invoice_id = $1, updated_at = $2 WHERE id = $3`

	result, err := r.db.Exec(ctx, query, invoiceID, time.Now(), id)
	if err != nil {
		return fmt.Errorf("setting holded invoice id: %w", err)
	}

	if result.RowsAffected() == 0 {
		return repository.ErrNotFound
	}

	return nil
}

// Approve marks an order as approved and stores the Holded invoice ID
func (r *OrderRepository) Approve(ctx context.Context, id uuid.UUID, approvedBy string, holdedInvoiceID string) error {
	now := time.Now()
	query := `
		UPDATE orders
		SET status = $1, approved_at = $2, approved_by = $3, holded_invoice_id = $4, updated_at = $5
		WHERE id = $6 AND status = $7`

	result, err := r.db.Exec(ctx, query,
		domain.OrderStatusApproved,
		now,
		approvedBy,
		holdedInvoiceID,
		now,
		id,
		domain.OrderStatusPending,
	)
	if err != nil {
		return fmt.Errorf("approving order: %w", err)
	}

	if result.RowsAffected() == 0 {
		return repository.ErrNotFound
	}

	return nil
}

// Reject marks an order as rejected with a reason
func (r *OrderRepository) Reject(ctx context.Context, id uuid.UUID, reason string) error {
	now := time.Now()
	query := `
		UPDATE orders
		SET status = $1, rejected_at = $2, rejection_reason = $3, updated_at = $4
		WHERE id = $5 AND status = $6`

	result, err := r.db.Exec(ctx, query,
		domain.OrderStatusRejected,
		now,
		reason,
		now,
		id,
		domain.OrderStatusPending,
	)
	if err != nil {
		return fmt.Errorf("rejecting order: %w", err)
	}

	if result.RowsAffected() == 0 {
		return repository.ErrNotFound
	}

	return nil
}
