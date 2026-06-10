CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE product_views (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    buyer_id VARCHAR(255) NOT NULL,
    product_id VARCHAR(255) NOT NULL,
    view_count INT DEFAULT 1,
    last_viewed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(buyer_id, product_id)
);

CREATE INDEX idx_product_views_buyer_id ON product_views (buyer_id);
CREATE INDEX idx_product_views_product_id ON product_views (product_id);

CREATE TABLE product_co_purchases (
    product_a_id VARCHAR(255) NOT NULL,
    product_b_id VARCHAR(255) NOT NULL,
    co_purchase_count INT DEFAULT 1,
    last_purchased_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (product_a_id, product_b_id)
);

CREATE INDEX idx_co_purchases_a ON product_co_purchases (product_a_id);
