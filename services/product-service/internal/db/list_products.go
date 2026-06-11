package db

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type ProductFilter struct {
	Search      *string
	CategoryID  *string
	SellerID    *string
	MinPrice    *float64
	MaxPrice    *float64
	MinRating   *float64
	Tags        []string
	Attributes  map[string]interface{}
	InStockOnly bool
}

func (q *Queries) ListProducts(ctx context.Context, filter ProductFilter, limit int32, pageToken string, lang string) ([]ProductWithTranslation, int32, string, error) {
	var wheres []string
	var args []interface{}

	// Language is always the first parameter
	args = append(args, lang)
	argCount := 1

	// Filter active products and non-deleted
	wheres = append(wheres, "p.deleted_at IS NULL")
	wheres = append(wheres, "p.status = 'active'")

	if filter.Search != nil && *filter.Search != "" {
		argCount++
		args = append(args, "%"+*filter.Search+"%")
		wheres = append(wheres, fmt.Sprintf("(t.title ILIKE $%d OR t_en.title ILIKE $%d OR t.description ILIKE $%d OR t_en.description ILIKE $%d)", argCount, argCount, argCount, argCount))
	}

	if filter.CategoryID != nil && *filter.CategoryID != "" {
		if catUID, err := uuid.Parse(*filter.CategoryID); err == nil {
			argCount++
			args = append(args, catUID)
			wheres = append(wheres, fmt.Sprintf("p.category_id = $%d", argCount))
		}
	}

	if filter.SellerID != nil && *filter.SellerID != "" {
		if sellerUID, err := uuid.Parse(*filter.SellerID); err == nil {
			argCount++
			args = append(args, sellerUID)
			wheres = append(wheres, fmt.Sprintf("p.seller_id = $%d", argCount))
		}
	}

	if filter.MinPrice != nil {
		argCount++
		args = append(args, fmt.Sprintf("%.2f", *filter.MinPrice))
		wheres = append(wheres, fmt.Sprintf("p.min_price >= $%d::numeric", argCount))
	}

	if filter.MaxPrice != nil {
		argCount++
		args = append(args, fmt.Sprintf("%.2f", *filter.MaxPrice))
		wheres = append(wheres, fmt.Sprintf("p.max_price <= $%d::numeric", argCount))
	}

	if filter.MinRating != nil {
		argCount++
		args = append(args, *filter.MinRating)
		wheres = append(wheres, fmt.Sprintf("p.rating >= $%d", argCount))
	}

	if len(filter.Tags) > 0 {
		argCount++
		args = append(args, pq.Array(filter.Tags))
		wheres = append(wheres, fmt.Sprintf("p.id IN (SELECT pt.product_id FROM product_tags pt JOIN tags tag ON tag.id = pt.tag_id WHERE tag.slug = ANY($%d))", argCount))
	}

	if len(filter.Attributes) > 0 {
		attrsJSON, _ := json.Marshal(filter.Attributes)
		argCount++
		args = append(args, attrsJSON)
		wheres = append(wheres, fmt.Sprintf("p.attributes @> $%d::jsonb", argCount))
	}

	// Count total
	countSQL := "SELECT COUNT(*) FROM products p LEFT JOIN product_translations t ON t.product_id = p.id AND t.language = $1 LEFT JOIN product_translations t_en ON t_en.product_id = p.id AND t_en.language = 'en' WHERE " + strings.Join(wheres, " AND ")
	var total int64
	err := q.db.QueryRow(ctx, countSQL, args...).Scan(&total)
	if err != nil {
		return nil, 0, "", fmt.Errorf("count products: %w", err)
	}

	// Parsing PageToken for Offset
	offset := 0
	if pageToken != "" {
		fmt.Sscanf(pageToken, "offset_%d", &offset)
	}

	// Build main query
	sqlParts := []string{
		"SELECT p.id, p.seller_id, p.category_id, p.slug, p.attributes, p.brand, p.origin_country,",
		"       p.status, p.rating, p.review_count, p.sold_count, p.view_count, p.min_price, p.max_price,",
		"       p.image_url, p.thumbnail_url,",
		"       p.created_at, p.updated_at, p.deleted_at, p.product_type,",
		"       ST_Y(p.location::geometry)::float AS latitude,",
		"       ST_X(p.location::geometry)::float AS longitude,",
		"       COALESCE(t.title, t_en.title, '') AS title,",
		"       COALESCE(t.description, t_en.description) AS description",
		"FROM products p",
		"LEFT JOIN product_translations t    ON t.product_id = p.id AND t.language = $1",
		"LEFT JOIN product_translations t_en ON t_en.product_id = p.id AND t_en.language = 'en'",
		"WHERE " + strings.Join(wheres, " AND "),
		"ORDER BY p.created_at DESC",
	}

	// Append LIMIT and OFFSET
	argCount++
	args = append(args, limit)
	sqlParts = append(sqlParts, fmt.Sprintf("LIMIT $%d", argCount))

	argCount++
	args = append(args, offset)
	sqlParts = append(sqlParts, fmt.Sprintf("OFFSET $%d", argCount))

	finalSQL := strings.Join(sqlParts, "\n")

	rows, err := q.db.Query(ctx, finalSQL, args...)
	if err != nil {
		return nil, 0, "", fmt.Errorf("query products: %w", err)
	}
	defer rows.Close()

	var items []ProductWithTranslation
	for rows.Next() {
		var i ProductWithTranslation
		if err := rows.Scan(
			&i.ID,
			&i.SellerID,
			&i.CategoryID,
			&i.Slug,
			&i.Attributes,
			&i.Brand,
			&i.OriginCountry,
			&i.Status,
			&i.Rating,
			&i.ReviewCount,
			&i.SoldCount,
			&i.ViewCount,
			&i.MinPrice,
			&i.MaxPrice,
			&i.ImageUrl,
			&i.ThumbnailUrl,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.DeletedAt,
			&i.ProductType,
			&i.Latitude,
			&i.Longitude,
			&i.Title,
			&i.Description,
		); err != nil {
			return nil, 0, "", err
		}
		items = append(items, i)
	}

	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, 0, "", err
	}

	nextPageToken := ""
	if int64(offset)+int64(limit) < total {
		nextPageToken = fmt.Sprintf("offset_%d", offset+int(limit))
	}

	return items, int32(total), nextPageToken, nil
}
