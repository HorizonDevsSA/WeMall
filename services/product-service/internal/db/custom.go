package db

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type CategoryWithTranslation struct {
	ID              uuid.UUID  `json:"id"`
	ParentID        *uuid.UUID `json:"parent_id"`
	Slug            string     `json:"slug"`
	IconUrl         *string    `json:"icon_url"`
	BannerUrl       *string    `json:"banner_url"`
	Level           int32      `json:"level"`
	AttributeSchema []byte     `json:"attribute_schema"`
	SortOrder       int32      `json:"sort_order"`
	IsActive        bool       `json:"is_active"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	Name            string     `json:"name"`
}

type ProductWithTranslation struct {
	ID            uuid.UUID      `json:"id"`
	SellerID      uuid.UUID      `json:"seller_id"`
	CategoryID    uuid.UUID      `json:"category_id"`
	Slug          string         `json:"slug"`
	Attributes    []byte         `json:"attributes"`
	Brand         *string        `json:"brand"`
	OriginCountry *string        `json:"origin_country"`
	Status        string         `json:"status"`
	Rating        pgtype.Numeric `json:"rating"`
	ReviewCount   int32          `json:"review_count"`
	SoldCount     int32          `json:"sold_count"`
	ViewCount     int32          `json:"view_count"`
	MinPrice      pgtype.Numeric `json:"min_price"`
	MaxPrice      pgtype.Numeric `json:"max_price"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     *time.Time     `json:"deleted_at"`
	ProductType   string         `json:"product_type"`
	Title         string         `json:"title"`
	Description   *string        `json:"description"`
	Latitude      *float64       `json:"latitude"`
	Longitude     *float64       `json:"longitude"`
	Distance      *float64       `json:"distance"`
}
