CREATE TABLE reviews (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id            UUID NOT NULL,
    buyer_id            UUID NOT NULL,
    seller_id           UUID NOT NULL,
    product_id          UUID NOT NULL,
    variant_id          UUID NOT NULL,
    
    rating_description  INT NOT NULL CHECK (rating_description BETWEEN 1 AND 5),
    rating_service      INT NOT NULL CHECK (rating_service BETWEEN 1 AND 5),
    rating_delivery     INT NOT NULL CHECK (rating_delivery BETWEEN 1 AND 5),
    
    review_type         VARCHAR(10) GENERATED ALWAYS AS (
        CASE 
            WHEN rating_description >= 4 THEN 'good'
            WHEN rating_description = 3 THEN 'neutral'
            ELSE 'bad'
        END
    ) STORED,
    
    content             TEXT,
    is_anonymous        BOOLEAN DEFAULT FALSE,
    has_media           BOOLEAN DEFAULT FALSE,
    nlp_tags            JSONB DEFAULT '[]',
    is_system_generated BOOLEAN DEFAULT FALSE,
    
    created_at          TIMESTAMPTZ DEFAULT now(),
    updated_at          TIMESTAMPTZ DEFAULT now(),
    deleted_at          TIMESTAMPTZ,
    
    CONSTRAINT unique_order_variant UNIQUE (order_id, variant_id)
);

CREATE TABLE review_media (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    review_id   UUID NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
    media_url   TEXT NOT NULL,
    media_type  VARCHAR(20) NOT NULL, -- 'image' | 'video'
    sort_order  INT DEFAULT 0,
    created_at  TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE append_reviews (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    review_id   UUID UNIQUE NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
    content     TEXT NOT NULL,
    has_media   BOOLEAN DEFAULT FALSE,
    created_at  TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE append_review_media (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    append_review_id    UUID NOT NULL REFERENCES append_reviews(id) ON DELETE CASCADE,
    media_url           TEXT NOT NULL,
    media_type          VARCHAR(20) NOT NULL, -- 'image' | 'video'
    sort_order          INT DEFAULT 0,
    created_at          TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE seller_replies (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    review_id   UUID NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
    reply_type  VARCHAR(10) NOT NULL CHECK (reply_type IN ('initial', 'append')),
    content     TEXT NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT now(),
    
    CONSTRAINT unique_reply_per_stage UNIQUE (review_id, reply_type)
);

CREATE INDEX idx_reviews_product_id ON reviews(product_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_reviews_seller_id ON reviews(seller_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_reviews_type ON reviews(product_id, review_type) WHERE deleted_at IS NULL;
CREATE INDEX idx_reviews_created_at ON reviews(created_at DESC);

CREATE TABLE order_deliveries (
    order_id     UUID PRIMARY KEY,
    buyer_id     UUID NOT NULL,
    delivered_at TIMESTAMPTZ NOT NULL,
    is_processed BOOLEAN DEFAULT FALSE
);

CREATE INDEX idx_order_deliveries_processed ON order_deliveries(is_processed) WHERE is_processed = FALSE;

