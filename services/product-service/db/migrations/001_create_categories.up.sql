CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE categories (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_id        UUID REFERENCES categories(id),
    slug             TEXT NOT NULL UNIQUE,
    icon_url         TEXT,
    banner_url       TEXT,
    level            INTEGER NOT NULL DEFAULT 1,
    attribute_schema JSONB,
    sort_order       INTEGER NOT NULL DEFAULT 0,
    is_active        BOOLEAN NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE category_translations (
    category_id  UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    language     TEXT NOT NULL CHECK (language IN ('en', 'sn', 'nd')),
    name         TEXT NOT NULL,
    PRIMARY KEY  (category_id, language)
);

CREATE INDEX idx_categories_parent_id ON categories(parent_id);
CREATE INDEX idx_categories_slug ON categories(slug);
