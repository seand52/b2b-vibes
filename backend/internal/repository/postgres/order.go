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
	// Preserve the status set by the caller (draft for carts, pending for direct orders)
	if order.Status == "" {
		order.Status = domain.OrderStatusPending
	}
	order.CreatedAt = now
	order.UpdatedAt = now

	orderQuery := `
		INSERT INTO orders (id, client_id, status, notes, admin_notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err = tx.Exec(ctx, orderQuery,
		order.ID,
		order.ClientID,
		order.Status,
		order.Notes,
		order.AdminNotes, // Defaults to empty string, not NULL
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

// getItemsForOrders fetches items for multiple orders in a single query
func (r *OrderRepository) getItemsForOrders(ctx context.Context, orderIDs []uuid.UUID) (map[uuid.UUID][]domain.OrderItem, error) {
	if len(orderIDs) == 0 {
		return make(map[uuid.UUID][]domain.OrderItem), nil
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(orderIDs))
	args := make([]any, len(orderIDs))
	for i, id := range orderIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT id, order_id, product_id, quantity
		FROM order_items
		WHERE order_id IN (%s)`, strings.Join(placeholders, ", "))

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying order items: %w", err)
	}
	defer rows.Close()

	itemsByOrderID := make(map[uuid.UUID][]domain.OrderItem)
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
		itemsByOrderID[item.OrderID] = append(itemsByOrderID[item.OrderID], item)
	}

	return itemsByOrderID, nil
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
		conditions = append(conditions, fmt.Sprintf("o.client_id = $%d", argIdx))
		args = append(args, *filter.ClientID)
		argIdx++
	}

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("o.status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}

	if len(filter.ExcludeStatuses) > 0 {
		placeholders := make([]string, len(filter.ExcludeStatuses))
		for i, status := range filter.ExcludeStatuses {
			placeholders[i] = fmt.Sprintf("$%d", argIdx)
			args = append(args, status)
			argIdx++
		}
		conditions = append(conditions, fmt.Sprintf("o.status NOT IN (%s)", strings.Join(placeholders, ", ")))
	}

	query := `
		SELECT o.id, o.client_id, o.status, o.notes, o.admin_notes, o.holded_invoice_id,
		       o.approved_at, o.approved_by, o.rejected_at, o.rejection_reason,
		       o.created_at, o.updated_at,
		       (SELECT COUNT(*) FROM order_items WHERE order_id = o.id) as item_count,
		       (SELECT COALESCE(SUM(quantity), 0) FROM order_items WHERE order_id = o.id) as total_quantity
		FROM orders o`

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY o.created_at DESC"

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
			&o.ItemCount,
			&o.TotalQuantity,
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

// GetDraftByClientID retrieves the draft order for a specific client
func (r *OrderRepository) GetDraftByClientID(ctx context.Context, clientID uuid.UUID) (*domain.Order, error) {
	query := `
		SELECT id, client_id, status, notes, admin_notes, holded_invoice_id, approved_at, approved_by, rejected_at, rejection_reason, created_at, updated_at
		FROM orders
		WHERE client_id = $1 AND status = $2`

	var o domain.Order
	err := r.db.QueryRow(ctx, query, clientID, domain.OrderStatusDraft).Scan(
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
		return nil, fmt.Errorf("scanning draft order: %w", err)
	}

	items, err := r.getOrderItems(ctx, o.ID)
	if err != nil {
		return nil, err
	}
	o.Items = items

	return &o, nil
}

// UpdateItems replaces all items in an order (used for cart updates)
func (r *OrderRepository) UpdateItems(ctx context.Context, orderID uuid.UUID, items []domain.OrderItem) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Delete existing items
	deleteQuery := `DELETE FROM order_items WHERE order_id = $1`
	_, err = tx.Exec(ctx, deleteQuery, orderID)
	if err != nil {
		return fmt.Errorf("deleting existing items: %w", err)
	}

	// Insert new items
	if len(items) > 0 {
		insertQuery := `
			INSERT INTO order_items (id, order_id, product_id, quantity)
			VALUES ($1, $2, $3, $4)`

		for i := range items {
			item := &items[i]
			if item.ID == uuid.Nil {
				item.ID = uuid.New()
			}
			item.OrderID = orderID

			_, err = tx.Exec(ctx, insertQuery,
				item.ID,
				item.OrderID,
				item.ProductID,
				item.Quantity,
			)
			if err != nil {
				return fmt.Errorf("inserting order item: %w", err)
			}
		}
	}

	// Update order updated_at timestamp
	updateOrderQuery := `UPDATE orders SET updated_at = $1 WHERE id = $2`
	_, err = tx.Exec(ctx, updateOrderQuery, time.Now(), orderID)
	if err != nil {
		return fmt.Errorf("updating order timestamp: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

// UpdateNotes updates the notes field of an order
func (r *OrderRepository) UpdateNotes(ctx context.Context, orderID uuid.UUID, notes string) error {
	query := `UPDATE orders SET notes = $1, updated_at = $2 WHERE id = $3`

	result, err := r.db.Exec(ctx, query, notes, time.Now(), orderID)
	if err != nil {
		return fmt.Errorf("updating order notes: %w", err)
	}

	if result.RowsAffected() == 0 {
		return repository.ErrNotFound
	}

	return nil
}

// SubmitDraft transitions a draft order to pending status with price snapshots
func (r *OrderRepository) SubmitDraft(ctx context.Context, orderID uuid.UUID, items []domain.OrderItem) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	now := time.Now()

	// Update items with unit_price and line_total
	updateItemQuery := `
		UPDATE order_items
		SET unit_price = $1, line_total = $2
		WHERE id = $3`

	for i := range items {
		item := &items[i]
		_, err = tx.Exec(ctx, updateItemQuery,
			item.UnitPrice,
			item.LineTotal,
			item.ID,
		)
		if err != nil {
			return fmt.Errorf("updating order item prices: %w", err)
		}
	}

	// Update order: set status to pending and submitted_at timestamp
	updateOrderQuery := `
		UPDATE orders
		SET status = $1, submitted_at = $2, updated_at = $3
		WHERE id = $4 AND status = $5`

	result, err := tx.Exec(ctx, updateOrderQuery,
		domain.OrderStatusPending,
		now,
		now,
		orderID,
		domain.OrderStatusDraft,
	)
	if err != nil {
		return fmt.Errorf("updating order to pending: %w", err)
	}

	if result.RowsAffected() == 0 {
		return repository.ErrNotFound
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}
