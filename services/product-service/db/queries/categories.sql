-- name: ListCategories :many
-- Returns all active categories with translated name (falls back to 'en').
SELECT
    c.id, c.parent_id, c.slug, c.icon_url, c.banner_url,
    c.level, c.attribute_schema, c.sort_order, c.is_active,
    c.created_at, c.updated_at,
    COALESCE(t.name, t_en.name, '') AS name
FROM categories c
LEFT JOIN category_translations t    ON t.category_id = c.id AND t.language = $1
LEFT JOIN category_translations t_en ON t_en.category_id = c.id AND t_en.language = 'en'
WHERE c.is_active = TRUE
ORDER BY c.level ASC, c.sort_order ASC;

-- name: GetCategoryBySlug :one
SELECT
    c.id, c.parent_id, c.slug, c.icon_url, c.banner_url,
    c.level, c.attribute_schema, c.sort_order, c.is_active,
    c.created_at, c.updated_at,
    COALESCE(t.name, t_en.name, '') AS name
FROM categories c
LEFT JOIN category_translations t    ON t.category_id = c.id AND t.language = $1
LEFT JOIN category_translations t_en ON t_en.category_id = c.id AND t_en.language = 'en'
WHERE c.slug = $2 AND c.is_active = TRUE;

-- name: GetCategoryByID :one
SELECT
    c.id, c.parent_id, c.slug, c.icon_url, c.banner_url,
    c.level, c.attribute_schema, c.sort_order, c.is_active,
    c.created_at, c.updated_at,
    COALESCE(t.name, t_en.name, '') AS name
FROM categories c
LEFT JOIN category_translations t    ON t.category_id = c.id AND t.language = $1
LEFT JOIN category_translations t_en ON t_en.category_id = c.id AND t_en.language = 'en'
WHERE c.id = $2 AND c.is_active = TRUE;

-- name: CreateCategory :one
INSERT INTO categories (parent_id, slug, icon_url, banner_url, level, attribute_schema, sort_order, is_active)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, parent_id, slug, icon_url, banner_url, level, attribute_schema, sort_order, is_active, created_at, updated_at;

-- name: CreateCategoryTranslation :exec
INSERT INTO category_translations (category_id, language, name)
VALUES ($1, $2, $3)
ON CONFLICT (category_id, language) DO UPDATE SET name = EXCLUDED.name;
