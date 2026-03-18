-- Add submitted_at to track when draft was submitted
ALTER TABLE orders ADD COLUMN submitted_at TIMESTAMPTZ;

-- Add price snapshot columns to order_items (captured at submit time)
ALTER TABLE order_items ADD COLUMN unit_price DECIMAL(12,4);
ALTER TABLE order_items ADD COLUMN line_total DECIMAL(12,4);

-- Index for efficiently finding a client's draft order
CREATE INDEX idx_orders_client_draft ON orders(client_id)
    WHERE status = 'draft';
