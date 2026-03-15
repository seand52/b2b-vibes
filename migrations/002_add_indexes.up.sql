-- Orders: list by client (main access pattern)
CREATE INDEX idx_orders_client_id ON orders(client_id);

-- Clients: email lookup for auth linking
CREATE INDEX idx_clients_email ON clients(email);

-- Product images: fetch images for a product
CREATE INDEX idx_product_images_product ON product_images(product_id, display_order);
