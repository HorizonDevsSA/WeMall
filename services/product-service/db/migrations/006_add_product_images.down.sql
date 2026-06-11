-- Revert: remove image_url and thumbnail_url from products.
ALTER TABLE products
    DROP COLUMN IF EXISTS image_url,
    DROP COLUMN IF EXISTS thumbnail_url;
