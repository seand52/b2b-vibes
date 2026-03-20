-- Optimize order_items queries for counting and summing
CREATE INDEX idx_order_items_order_id ON order_items(order_id);
