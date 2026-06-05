CREATE TYPE product_status AS ENUM ('draft', 'active', 'paused', 'banned');

CREATE TABLE products (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    seller_id       UUID NOT NULL,
    category_id     UUID NOT NULL REFERENCES categories(id),
    slug            TEXT NOT NULL UNIQUE,
    attributes      JSONB NOT NULL DEFAULT '{}',
    brand           TEXT,
    origin_country  TEXT,
    status          product_status NOT NULL DEFAULT 'draft',
    rating          NUMERIC(3,2) NOT NULL DEFAULT 0,
    review_count    INTEGER NOT NULL DEFAULT 0,
    sold_count      INTEGER NOT NULL DEFAULT 0,
    view_count      INTEGER NOT NULL DEFAULT 0,
    min_price       NUMERIC(12,2),
    max_price       NUMERIC(12,2),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE TABLE product_translations (
    product_id   UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    language     TEXT NOT NULL CHECK (language IN ('en', 'sn', 'nd')),
    title        TEXT NOT NULL,
    description  TEXT,
    PRIMARY KEY  (product_id, language)
);

CREATE TABLE product_variants (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id      UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    sku             TEXT NOT NULL UNIQUE,
    options         JSONB NOT NULL DEFAULT '{}',
    price           NUMERIC(12,2) NOT NULL,
    compare_price   NUMERIC(12,2),
    weight_grams    INTEGER,
    image_url       TEXT,
    is_default      BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE product_images (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id  UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    url         TEXT NOT NULL,
    alt_text    TEXT,
    sort_order  INTEGER NOT NULL DEFAULT 0,
    is_primary  BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE tags (
    id    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name  TEXT NOT NULL UNIQUE,
    slug  TEXT NOT NULL UNIQUE
);

CREATE TABLE product_tags (
    product_id  UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    tag_id      UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (product_id, tag_id)
);

-- Indexes
CREATE INDEX products_attributes_gin ON products USING GIN (attributes);
CREATE INDEX idx_products_category ON products(category_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_products_seller ON products(seller_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_products_status ON products(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_variants_product ON product_variants(product_id);
