package service

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wemall/product-service/internal/db"
)

func TestAssembleProductMediaFallback(t *testing.T) {
	prodID := uuid.New()
	p := &db.ProductWithTranslation{
		ID:        prodID,
		Title:     "Test Product",
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	tags := []db.Tag{{ID: uuid.New(), Name: "Tech"}}

	t.Run("Primary image is selected first", func(t *testing.T) {
		images := []db.ProductImage{
			{Url: "img1.png", IsPrimary: false},
			{Url: "img2_primary.png", IsPrimary: true},
		}
		variants := []db.ProductVariant{
			{ImageUrl: strPtr("var1.png"), IsDefault: true},
		}

		res := assembleProduct(p, variants, images, tags)
		if res.ImageUrl != "img2_primary.png" {
			t.Errorf("expected ImageUrl to be img2_primary.png, got %s", res.ImageUrl)
		}
		if res.Thumbnail != "img2_primary.png" {
			t.Errorf("expected Thumbnail to be img2_primary.png, got %s", res.Thumbnail)
		}
	})

	t.Run("Fallback to first image if no primary is specified", func(t *testing.T) {
		images := []db.ProductImage{
			{Url: "img1.png", IsPrimary: false},
			{Url: "img2.png", IsPrimary: false},
		}
		variants := []db.ProductVariant{
			{ImageUrl: strPtr("var1.png"), IsDefault: true},
		}

		res := assembleProduct(p, variants, images, tags)
		if res.ImageUrl != "img1.png" {
			t.Errorf("expected ImageUrl to be img1.png, got %s", res.ImageUrl)
		}
	})

	t.Run("Fallback to default variant image if no product images exist", func(t *testing.T) {
		images := []db.ProductImage{}
		variants := []db.ProductVariant{
			{ImageUrl: strPtr("var_normal.png"), IsDefault: false},
			{ImageUrl: strPtr("var_default.png"), IsDefault: true},
		}

		res := assembleProduct(p, variants, images, tags)
		if res.ImageUrl != "var_default.png" {
			t.Errorf("expected ImageUrl to be var_default.png, got %s", res.ImageUrl)
		}
	})

	t.Run("Fallback to first available variant image if no default variant has image", func(t *testing.T) {
		images := []db.ProductImage{}
		variants := []db.ProductVariant{
			{ImageUrl: strPtr("var_normal.png"), IsDefault: false},
		}

		res := assembleProduct(p, variants, images, tags)
		if res.ImageUrl != "var_normal.png" {
			t.Errorf("expected ImageUrl to be var_normal.png, got %s", res.ImageUrl)
		}
	})
}

func strPtr(s string) *string {
	return &s
}
