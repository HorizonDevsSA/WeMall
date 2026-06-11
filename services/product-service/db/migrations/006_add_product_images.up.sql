-- Add image_url and thumbnail_url columns to the products table.
-- These store the primary display image and the thumbnail used in list views.
ALTER TABLE products
    ADD COLUMN IF NOT EXISTS image_url     TEXT,
    ADD COLUMN IF NOT EXISTS thumbnail_url TEXT;
