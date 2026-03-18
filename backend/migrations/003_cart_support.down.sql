DROP INDEX IF EXISTS idx_orders_client_draft;
ALTER TABLE order_items DROP COLUMN IF EXISTS line_total;
ALTER TABLE order_items DROP COLUMN IF EXISTS unit_price;
ALTER TABLE orders DROP COLUMN IF EXISTS submitted_at;
