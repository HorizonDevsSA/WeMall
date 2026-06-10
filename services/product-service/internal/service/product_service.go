package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/gosimple/slug"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	inventoryv1 "github.com/wemall/gen/inventory/v1"
	productv1 "github.com/wemall/gen/product/v1"
	"github.com/wemall/product-service/internal/db"
)

func float64ToNumeric(f float64) pgtype.Numeric {
	var n pgtype.Numeric
	_ = n.Scan(fmt.Sprintf("%.2f", f))
	return n
}

func numericToFloat64(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	val, err := n.Value()
	if err != nil {
		return 0
	}
	if s, ok := val.(string); ok {
		var f float64
		fmt.Sscanf(s, "%f", &f)
		return f
	}
	return 0
}

func getProductByIDRowToTranslation(row db.GetProductByIDRow) db.ProductWithTranslation {
	return db.ProductWithTranslation{
		ID:            row.ID,
		SellerID:      row.SellerID,
		CategoryID:    row.CategoryID,
		Slug:          row.Slug,
		Attributes:    row.Attributes,
		Brand:         row.Brand,
		OriginCountry: row.OriginCountry,
		Status:        row.Status,
		Rating:        row.Rating,
		ReviewCount:   row.ReviewCount,
		SoldCount:     row.SoldCount,
		ViewCount:     row.ViewCount,
		MinPrice:      row.MinPrice,
		MaxPrice:      row.MaxPrice,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
		DeletedAt:     row.DeletedAt,
		ProductType:   row.ProductType,
		Title:         row.Title,
		Description:   row.Description,
		Latitude:      &row.Latitude,
		Longitude:     &row.Longitude,
	}
}

func getProductBySlugRowToTranslation(row db.GetProductBySlugRow) db.ProductWithTranslation {
	return db.ProductWithTranslation{
		ID:            row.ID,
		SellerID:      row.SellerID,
		CategoryID:    row.CategoryID,
		Slug:          row.Slug,
		Attributes:    row.Attributes,
		Brand:         row.Brand,
		OriginCountry: row.OriginCountry,
		Status:        row.Status,
		Rating:        row.Rating,
		ReviewCount:   row.ReviewCount,
		SoldCount:     row.SoldCount,
		ViewCount:     row.ViewCount,
		MinPrice:      row.MinPrice,
		MaxPrice:      row.MaxPrice,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
		DeletedAt:     row.DeletedAt,
		ProductType:   row.ProductType,
		Title:         row.Title,
		Description:   row.Description,
		Latitude:      &row.Latitude,
		Longitude:     &row.Longitude,
	}
}

func getProductBatchRowToTranslation(row db.GetProductBatchRow) db.ProductWithTranslation {
	return db.ProductWithTranslation{
		ID:            row.ID,
		SellerID:      row.SellerID,
		CategoryID:    row.CategoryID,
		Slug:          row.Slug,
		Attributes:    row.Attributes,
		Brand:         row.Brand,
		OriginCountry: row.OriginCountry,
		Status:        row.Status,
		Rating:        row.Rating,
		ReviewCount:   row.ReviewCount,
		SoldCount:     row.SoldCount,
		ViewCount:     row.ViewCount,
		MinPrice:      row.MinPrice,
		MaxPrice:      row.MaxPrice,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
		DeletedAt:     row.DeletedAt,
		ProductType:   row.ProductType,
		Title:         row.Title,
		Description:   row.Description,
		Latitude:      &row.Latitude,
		Longitude:     &row.Longitude,
	}
}

type ProductService struct {
	q    *db.Queries
	pool *pgxpool.Pool
	nc   *nats.Conn
}

func NewProductService(q *db.Queries, pool *pgxpool.Pool, nc *nats.Conn) *ProductService {
	return &ProductService{q: q, pool: pool, nc: nc}
}

// ── Categories ───────────────────────────────────────────────────────────────

func (s *ProductService) ListCategories(ctx context.Context, lang string) ([]*productv1.Category, error) {
	flat, err := s.q.ListCategories(ctx, lang)
	if err != nil {
		return nil, err
	}

	// Map to hold categories by their ID
	catMap := make(map[string]*productv1.Category)
	var roots []*productv1.Category

	// Initialize all category nodes
	for _, c := range flat {
		node := &productv1.Category{
			Id:              c.ID.String(),
			ParentId:        getParentIDString(c.ParentID),
			Slug:            c.Slug,
			IconUrl:         getVal(c.IconUrl),
			BannerUrl:       getVal(c.BannerUrl),
			Level:           c.Level,
			AttributeSchema: jsonToStruct(c.AttributeSchema),
			SortOrder:       c.SortOrder,
			Children:        []*productv1.Category{},
		}
		// Since translation is handled in SQL:
		node.Name = c.Name

		catMap[node.Id] = node
	}

	// Build the tree
	for _, c := range flat {
		idStr := c.ID.String()
		node := catMap[idStr]
		if c.ParentID == nil {
			roots = append(roots, node)
		} else {
			pIDStr := c.ParentID.String()
			parent, exists := catMap[pIDStr]
			if exists {
				parent.Children = append(parent.Children, node)
			} else {
				// Parent not in active/active list, treat as root fallback
				roots = append(roots, node)
			}
		}
	}

	// Recursively sort children by sort_order
	var sortTree func(nodes []*productv1.Category)
	sortTree = func(nodes []*productv1.Category) {
		sort.Slice(nodes, func(i, j int) bool {
			return nodes[i].SortOrder < nodes[j].SortOrder
		})
		for _, n := range nodes {
			if len(n.Children) > 0 {
				sortTree(n.Children)
			}
		}
	}
	sortTree(roots)

	return roots, nil
}

func (s *ProductService) GetCategory(ctx context.Context, slugStr, lang string) (*productv1.Category, error) {
	row, err := s.q.GetCategoryBySlug(ctx, db.GetCategoryBySlugParams{
		Language: lang,
		Slug:     slugStr,
	})
	if err != nil {
		return nil, fmt.Errorf("category not found: %w", err)
	}
	c := db.CategoryWithTranslation(row)

	return &productv1.Category{
		Id:              c.ID.String(),
		ParentId:        getParentIDString(c.ParentID),
		Name:            c.Name,
		Slug:            c.Slug,
		IconUrl:         getVal(c.IconUrl),
		BannerUrl:       getVal(c.BannerUrl),
		Level:           c.Level,
		AttributeSchema: jsonToStruct(c.AttributeSchema),
		SortOrder:       c.SortOrder,
	}, nil
}

// ── Products ─────────────────────────────────────────────────────────────────

func (s *ProductService) CreateProduct(ctx context.Context, req *productv1.CreateProductRequest) (*productv1.Product, error) {
	sellerUID, err := uuid.Parse(req.SellerId)
	if err != nil {
		return nil, fmt.Errorf("invalid seller id: %w", err)
	}
	catUID, err := uuid.Parse(req.CategoryId)
	if err != nil {
		return nil, fmt.Errorf("invalid category id: %w", err)
	}

	// Start a transaction
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)

	// Validate category exists
	_, err = qtx.GetCategoryByID(ctx, db.GetCategoryByIDParams{
		Language: "en",
		ID:       catUID,
	})
	if err != nil {
		return nil, fmt.Errorf("invalid category: %w", err)
	}

	// Calculate slug
	productSlug := slug.Make(req.Title)
	// Append random string to slug if conflicts, for now simple uniqueness
	productSlug = fmt.Sprintf("%s-%s", productSlug, uuid.New().String()[:8])

	// Calculate prices
	if len(req.Variants) == 0 {
		return nil, fmt.Errorf("at least one variant is required")
	}

	minPrice := req.Variants[0].Price
	maxPrice := req.Variants[0].Price
	for _, v := range req.Variants {
		if v.Price < minPrice {
			minPrice = v.Price
		}
		if v.Price > maxPrice {
			maxPrice = v.Price
		}
	}

	// Create Product base
	attributesJSON := structToJSON(req.Attributes)
	createdProduct, err := qtx.CreateProduct(ctx, db.CreateProductParams{
		SellerID:      sellerUID,
		CategoryID:    catUID,
		Slug:          productSlug,
		Attributes:    attributesJSON,
		Brand:         &req.Brand,
		OriginCountry: &req.OriginCountry,
		Status:        "active", // default status
		MinPrice:      float64ToNumeric(minPrice),
		MaxPrice:      float64ToNumeric(maxPrice),
		Column10:      req.Latitude,
		Column11:      req.Longitude,
		Column12:      productTypeToString(req.ProductType),
	})
	if err != nil {
		return nil, fmt.Errorf("create product: %w", err)
	}
	productID := createdProduct.ID

	// Add translation (default english, or language provided)
	lang := "en"
	if req.Language != "" {
		lang = req.Language
	}
	err = qtx.CreateProductTranslation(ctx, db.CreateProductTranslationParams{
		ProductID:   productID,
		Language:    lang,
		Title:       req.Title,
		Description: &req.Description,
	})
	if err != nil {
		return nil, fmt.Errorf("create product translation: %w", err)
	}
	if lang != "en" {
		// Always create English backup translation if default is another language
		_ = qtx.CreateProductTranslation(ctx, db.CreateProductTranslationParams{
			ProductID:   productID,
			Language:    "en",
			Title:       req.Title,
			Description: &req.Description,
		})
	}

	// Create Variants
	for _, v := range req.Variants {
		optsJSON := structToJSON(v.Options)
		variant, err := qtx.CreateProductVariant(ctx, db.CreateProductVariantParams{
			ProductID:    productID,
			Sku:          v.Sku,
			Options:      optsJSON,
			Price:        float64ToNumeric(v.Price),
			ComparePrice: float64ToNumeric(v.ComparePrice),
			WeightGrams:  nil, // weightGrams not present in proto input
			ImageUrl:     nil, // imageUrl not present in proto input
			IsDefault:    false,
		})
		if err != nil {
			return nil, fmt.Errorf("create variant: %w", err)
		}

		_, err = qtx.UpsertStock(ctx, db.UpsertStockParams{
			VariantID: variant.ID,
			Quantity:  v.InitialQuantity,
		})
		if err != nil {
			return nil, fmt.Errorf("upsert initial stock for variant %s: %w", variant.ID.String(), err)
		}
	}

	// Create Tags
	for _, tagName := range req.Tags {
		tagSlug := slug.Make(tagName)
		tag, err := qtx.CreateTag(ctx, db.CreateTagParams{
			Name: tagName,
			Slug: tagSlug,
		})
		if err != nil {
			return nil, fmt.Errorf("create tag: %w", err)
		}
		err = qtx.AddProductTag(ctx, db.AddProductTagParams{
			ProductID: productID,
			TagID:     tag.ID,
		})
		if err != nil {
			return nil, fmt.Errorf("link product tag: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit product: %w", err)
	}

	// 5. Publish NATS event for newly created product
	if s.nc != nil {
		eventData := map[string]string{
			"product_id": productID.String(),
			"seller_id":  req.SellerId,
			"title":      req.Title,
			"image_url":  "", // Images are added separately
		}
		if b, err := json.Marshal(eventData); err == nil {
			_ = s.nc.Publish("wemall.product.created", b)
		}
	}

	return s.GetProduct(ctx, productID.String(), "", lang)
}

func (s *ProductService) UpdateProduct(ctx context.Context, req *productv1.UpdateProductRequest) (*productv1.Product, error) {
	productUID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, fmt.Errorf("invalid product id: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)

	// Fetch to ensure exists and owned by seller
	row, err := qtx.GetProductByID(ctx, db.GetProductByIDParams{
		Language: "en",
		ID:       productUID,
	})
	if err != nil {
		return nil, fmt.Errorf("product not found: %w", err)
	}
	p := getProductByIDRowToTranslation(row)
	if p.SellerID.String() != req.SellerId {
		return nil, fmt.Errorf("permission denied")
	}

	// Update base fields
	statusStr := ""
	if req.Status != productv1.ProductStatus_PRODUCT_STATUS_UNSPECIFIED {
		switch req.Status {
		case productv1.ProductStatus_PRODUCT_STATUS_DRAFT:
			statusStr = "draft"
		case productv1.ProductStatus_PRODUCT_STATUS_ACTIVE:
			statusStr = "active"
		case productv1.ProductStatus_PRODUCT_STATUS_PAUSED:
			statusStr = "paused"
		case productv1.ProductStatus_PRODUCT_STATUS_BANNED:
			statusStr = "banned"
		}
	}

	err = qtx.UpdateProduct(ctx, db.UpdateProductParams{
		ID:     productUID,
		Brand:  req.Brand,
		Status: statusStr,
	})
	if err != nil {
		return nil, fmt.Errorf("update product: %w", err)
	}

	// Update translation
	lang := "en"
	if req.Language != "" {
		lang = req.Language
	}
	err = qtx.CreateProductTranslation(ctx, db.CreateProductTranslationParams{
		ProductID:   productUID,
		Language:    lang,
		Title:       req.Title,
		Description: &req.Description,
	})
	if err != nil {
		return nil, fmt.Errorf("update translation: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit update: %w", err)
	}

	return s.GetProduct(ctx, req.Id, "", lang)
}

func (s *ProductService) DeleteProduct(ctx context.Context, id, sellerID string) error {
	pUID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid product id: %w", err)
	}
	sUID, err := uuid.Parse(sellerID)
	if err != nil {
		return fmt.Errorf("invalid seller id: %w", err)
	}
	return s.q.DeleteProduct(ctx, db.DeleteProductParams{
		ID:       pUID,
		SellerID: sUID,
	})
}

func (s *ProductService) GetProduct(ctx context.Context, id, slugStr, lang string) (*productv1.Product, error) {
	var p db.ProductWithTranslation
	var err error

	if id != "" {
		pUID, err2 := uuid.Parse(id)
		if err2 != nil {
			return nil, fmt.Errorf("invalid product id: %w", err2)
		}
		row, err2 := s.q.GetProductByID(ctx, db.GetProductByIDParams{
			Language: lang,
			ID:       pUID,
		})
		err = err2
		if err2 == nil {
			p = getProductByIDRowToTranslation(row)
		}
	} else if slugStr != "" {
		row, err2 := s.q.GetProductBySlug(ctx, db.GetProductBySlugParams{
			Language: lang,
			Slug:     slugStr,
		})
		err = err2
		if err2 == nil {
			p = getProductBySlugRowToTranslation(row)
		}
	} else {
		return nil, fmt.Errorf("either id or slug must be provided")
	}

	if err != nil {
		return nil, fmt.Errorf("get product: %w", err)
	}

	// Fetch variants, images, tags
	variants, err := s.q.GetProductVariants(ctx, p.ID)
	if err != nil {
		return nil, fmt.Errorf("get variants: %w", err)
	}
	images, err := s.q.GetProductImages(ctx, p.ID)
	if err != nil {
		return nil, fmt.Errorf("get images: %w", err)
	}
	tags, err := s.q.GetProductTags(ctx, p.ID)
	if err != nil {
		return nil, fmt.Errorf("get tags: %w", err)
	}

	return assembleProduct(&p, variants, images, tags), nil
}

func (s *ProductService) ListProducts(ctx context.Context, req *productv1.ListProductsRequest) ([]*productv1.Product, int32, string, error) {
	filter := db.ProductFilter{}
	if req.Filter != nil {
		if req.Filter.Search != "" {
			filter.Search = &req.Filter.Search
		}
		if req.Filter.CategoryId != "" {
			filter.CategoryID = &req.Filter.CategoryId
		}
		if req.Filter.SellerId != "" {
			filter.SellerID = &req.Filter.SellerId
		}
		if req.Filter.MinPrice > 0 {
			filter.MinPrice = &req.Filter.MinPrice
		}
		if req.Filter.MaxPrice > 0 {
			filter.MaxPrice = &req.Filter.MaxPrice
		}
		if req.Filter.MinRating > 0 {
			filter.MinRating = &req.Filter.MinRating
		}
		filter.Tags = req.Filter.Tags
		if req.Filter.Attributes != nil {
			filter.Attributes = req.Filter.Attributes.AsMap()
		}
	}

	pageSize := int32(20)
	if req.PageSize > 0 {
		pageSize = req.PageSize
	}

	lang := "en"
	if req.Language != "" {
		lang = req.Language
	}

	list, total, nextToken, err := s.q.ListProducts(ctx, filter, pageSize, req.PageToken, lang)
	if err != nil {
		return nil, 0, "", err
	}

	products := make([]*productv1.Product, len(list))
	for i := range list {
		variants, _ := s.q.GetProductVariants(ctx, list[i].ID)
		images, _ := s.q.GetProductImages(ctx, list[i].ID)
		tags, _ := s.q.GetProductTags(ctx, list[i].ID)
		products[i] = assembleProduct(&list[i], variants, images, tags)
	}

	if req.Filter != nil && req.Filter.InStockOnly {
		filtered := make([]*productv1.Product, 0, len(products))
		for _, product := range products {
			variantUUIDs := make([]uuid.UUID, 0, len(product.Variants))
			for _, variant := range product.Variants {
				if uid, err := uuid.Parse(variant.Id); err == nil {
					variantUUIDs = append(variantUUIDs, uid)
				}
			}
			stockResp, err := s.q.GetStockBatch(ctx, variantUUIDs)
			if err != nil {
				return nil, 0, "", fmt.Errorf("fetch stock batch: %w", err)
			}

			inStock := false
			for _, stock := range stockResp {
				if stock.Quantity > 0 {
					inStock = true
					break
				}
			}
			if inStock {
				filtered = append(filtered, product)
			}
		}
		products = filtered
		total = int32(len(filtered))
	}

	return products, total, nextToken, nil
}

func (s *ProductService) ListNearbyProducts(ctx context.Context, req *productv1.ListNearbyProductsRequest) ([]*productv1.Product, int32, string, error) {
	pageSize := int32(20)
	if req.PageSize > 0 {
		pageSize = req.PageSize
	}

	lang := "en"
	if req.Language != "" {
		lang = req.Language
	}

	offset := int32(0)
	if req.PageToken != "" {
		fmt.Sscanf(req.PageToken, "offset_%d", &offset)
	}

	total, err := s.q.CountNearbyProducts(ctx, db.CountNearbyProductsParams{
		Longitude:    req.Longitude,
		Latitude:     req.Latitude,
		RadiusMeters: req.RadiusMeters,
	})
	if err != nil {
		return nil, 0, "", fmt.Errorf("count nearby products: %w", err)
	}

	rows, err := s.q.ListNearbyProducts(ctx, db.ListNearbyProductsParams{
		Longitude:    req.Longitude,
		Latitude:     req.Latitude,
		Language:     lang,
		RadiusMeters: req.RadiusMeters,
		OffsetVal:    offset,
		LimitVal:     pageSize,
	})
	if err != nil {
		return nil, 0, "", fmt.Errorf("list nearby products: %w", err)
	}

	products := make([]*productv1.Product, len(rows))
	for i, row := range rows {
		p := db.ProductWithTranslation{
			ID:            row.ID,
			SellerID:      row.SellerID,
			CategoryID:    row.CategoryID,
			Slug:          row.Slug,
			Attributes:    row.Attributes,
			Brand:         row.Brand,
			OriginCountry: row.OriginCountry,
			Status:        row.Status,
			Rating:        row.Rating,
			ReviewCount:   row.ReviewCount,
			SoldCount:     row.SoldCount,
			ViewCount:     row.ViewCount,
			MinPrice:      row.MinPrice,
			MaxPrice:      row.MaxPrice,
			CreatedAt:     row.CreatedAt,
			UpdatedAt:     row.UpdatedAt,
			DeletedAt:     row.DeletedAt,
			Title:         row.Title,
			Description:   row.Description,
			Latitude:      &row.Latitude,
			Longitude:     &row.Longitude,
			Distance:      &row.DistanceMeters,
		}

		variants, _ := s.q.GetProductVariants(ctx, p.ID)
		images, _ := s.q.GetProductImages(ctx, p.ID)
		tags, _ := s.q.GetProductTags(ctx, p.ID)
		products[i] = assembleProduct(&p, variants, images, tags)
	}

	nextPageToken := ""
	if offset+pageSize < int32(total) {
		nextPageToken = fmt.Sprintf("offset_%d", offset+pageSize)
	}

	return products, int32(total), nextPageToken, nil
}

func (s *ProductService) GetProductBatch(ctx context.Context, ids []string, lang string) (map[string]*productv1.Product, error) {
	uids := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		uid, err := uuid.Parse(id)
		if err == nil {
			uids = append(uids, uid)
		}
	}

	rows, err := s.q.GetProductBatch(ctx, db.GetProductBatchParams{
		Language: lang,
		Column2:  uids,
	})
	if err != nil {
		return nil, err
	}

	res := make(map[string]*productv1.Product)
	for i := range rows {
		p := getProductBatchRowToTranslation(rows[i])
		variants, _ := s.q.GetProductVariants(ctx, p.ID)
		images, _ := s.q.GetProductImages(ctx, p.ID)
		tags, _ := s.q.GetProductTags(ctx, p.ID)
		res[p.ID.String()] = assembleProduct(&p, variants, images, tags)
	}

	return res, nil
}

func (s *ProductService) GetVariantBatch(ctx context.Context, ids []string) (map[string]*productv1.ProductVariant, error) {
	uids := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		uid, err := uuid.Parse(id)
		if err == nil {
			uids = append(uids, uid)
		}
	}

	list, err := s.q.GetVariantBatch(ctx, uids)
	if err != nil {
		return nil, err
	}

	res := make(map[string]*productv1.ProductVariant)
	for i := range list {
		res[list[i].ID.String()] = mapVariant(&list[i])
	}

	return res, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func getParentIDString(parentID *uuid.UUID) string {
	if parentID == nil {
		return ""
	}
	return parentID.String()
}

func getVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func jsonToStruct(b []byte) *structpb.Struct {
	if len(b) == 0 {
		s, _ := structpb.NewStruct(map[string]interface{}{})
		return s
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		s, _ := structpb.NewStruct(map[string]interface{}{})
		return s
	}
	s, _ := structpb.NewStruct(m)
	return s
}

func structToJSON(s *structpb.Struct) []byte {
	if s == nil {
		return []byte("{}")
	}
	b, err := json.Marshal(s.AsMap())
	if err != nil {
		return []byte("{}")
	}
	return b
}

func mapStatus(statusStr string) productv1.ProductStatus {
	switch statusStr {
	case "draft":
		return productv1.ProductStatus_PRODUCT_STATUS_DRAFT
	case "active":
		return productv1.ProductStatus_PRODUCT_STATUS_ACTIVE
	case "paused":
		return productv1.ProductStatus_PRODUCT_STATUS_PAUSED
	case "banned":
		return productv1.ProductStatus_PRODUCT_STATUS_BANNED
	default:
		return productv1.ProductStatus_PRODUCT_STATUS_UNSPECIFIED
	}
}

func mapVariant(v *db.ProductVariant) *productv1.ProductVariant {
	if v == nil {
		return nil
	}
	return &productv1.ProductVariant{
		Id:           v.ID.String(),
		ProductId:    v.ProductID.String(),
		Sku:          v.Sku,
		Options:      jsonToStruct(v.Options),
		Price:        numericToFloat64(v.Price),
		ComparePrice: numericToFloat64(v.ComparePrice),
		ImageUrl:     getVal(v.ImageUrl),
		IsDefault:    v.IsDefault,
	}
}

func mapImage(img *db.ProductImage) *productv1.ProductImage {
	if img == nil {
		return nil
	}
	return &productv1.ProductImage{
		Id:        img.ID.String(),
		ProductId: img.ProductID.String(),
		Url:       img.Url,
		AltText:   getVal(img.AltText),
		SortOrder: img.SortOrder,
		IsPrimary: img.IsPrimary,
	}
}

func assembleProduct(p *db.ProductWithTranslation, variants []db.ProductVariant, images []db.ProductImage, tags []db.Tag) *productv1.Product {
	vList := make([]*productv1.ProductVariant, len(variants))
	for i := range variants {
		vList[i] = mapVariant(&variants[i])
	}

	iList := make([]*productv1.ProductImage, len(images))
	for i := range images {
		iList[i] = mapImage(&images[i])
	}

	tList := make([]string, len(tags))
	for i := range tags {
		tList[i] = tags[i].Name
	}

	var lat, lon, dist float64
	if p.Latitude != nil {
		lat = *p.Latitude
	}
	if p.Longitude != nil {
		lon = *p.Longitude
	}
	if p.Distance != nil {
		dist = *p.Distance
	}

	var imageUrl, thumbnail string
	for _, img := range iList {
		if img.IsPrimary {
			imageUrl = img.Url
			thumbnail = img.Url
			break
		}
	}
	if imageUrl == "" && len(iList) > 0 {
		imageUrl = iList[0].Url
		thumbnail = iList[0].Url
	}
	if imageUrl == "" {
		for _, v := range vList {
			if v.IsDefault && v.ImageUrl != "" {
				imageUrl = v.ImageUrl
				thumbnail = v.ImageUrl
				break
			}
		}
	}
	if imageUrl == "" && len(vList) > 0 {
		for _, v := range vList {
			if v.ImageUrl != "" {
				imageUrl = v.ImageUrl
				thumbnail = v.ImageUrl
				break
			}
		}
	}

	return &productv1.Product{
		Id:            p.ID.String(),
		SellerId:      p.SellerID.String(),
		CategoryId:    p.CategoryID.String(),
		Title:         p.Title,
		Slug:          p.Slug,
		Description:   getVal(p.Description),
		Attributes:    jsonToStruct(p.Attributes),
		Brand:         getVal(p.Brand),
		OriginCountry: getVal(p.OriginCountry),
		Status:        mapStatus(p.Status),
		Rating:        numericToFloat64(p.Rating),
		ReviewCount:   p.ReviewCount,
		SoldCount:     p.SoldCount,
		MinPrice:      numericToFloat64(p.MinPrice),
		MaxPrice:      numericToFloat64(p.MaxPrice),
		Variants:      vList,
		Images:        iList,
		Tags:          tList,
		CreatedAt:     timestamppb.New(p.CreatedAt),
		UpdatedAt:     timestamppb.New(p.UpdatedAt),
		Latitude:      lat,
		Longitude:     lon,
		Distance:      dist,
		ProductType:   stringToProductType(p.ProductType),
		Thumbnail:     thumbnail,
		ImageUrl:      imageUrl,
	}
}

// ── Inventory Service Methods ────────────────────────────────────────────────

func (s *ProductService) UpsertStock(ctx context.Context, variantID string, quantity int32) (*inventoryv1.StockItem, error) {
	variantUUID, err := uuid.Parse(variantID)
	if err != nil {
		return nil, err
	}

	row, err := s.q.UpsertStock(ctx, db.UpsertStockParams{
		VariantID: variantUUID,
		Quantity:  quantity,
	})
	if err != nil {
		return nil, err
	}
	return toStockItem(row.VariantID.String(), row.Quantity, row.UpdatedAt), nil
}

func (s *ProductService) GetStock(ctx context.Context, variantID string) (*inventoryv1.StockItem, error) {
	variantUUID, err := uuid.Parse(variantID)
	if err != nil {
		return nil, err
	}

	row, err := s.q.GetStock(ctx, variantUUID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return toStockItem(variantID, 0, time.Now().UTC()), nil
		}
		return nil, err
	}
	return toStockItem(row.VariantID.String(), row.Quantity, row.UpdatedAt), nil
}

func (s *ProductService) GetStockBatch(ctx context.Context, variantIDs []string) (map[string]*inventoryv1.StockItem, error) {
	uuidIDs := make([]uuid.UUID, 0, len(variantIDs))
	for _, variantID := range variantIDs {
		if parsed, err := uuid.Parse(variantID); err == nil {
			uuidIDs = append(uuidIDs, parsed)
		}
	}

	rows, err := s.q.GetStockBatch(ctx, uuidIDs)
	if err != nil {
		return nil, err
	}

	res := make(map[string]*inventoryv1.StockItem, len(variantIDs))
	now := time.Now().UTC()
	for _, row := range rows {
		res[row.VariantID.String()] = toStockItem(row.VariantID.String(), row.Quantity, row.UpdatedAt)
	}

	for _, variantID := range variantIDs {
		if _, ok := res[variantID]; !ok {
			res[variantID] = toStockItem(variantID, 0, now)
		}
	}

	return res, nil
}

func toStockItem(variantID string, quantity int32, updatedAt time.Time) *inventoryv1.StockItem {
	return &inventoryv1.StockItem{
		VariantId: variantID,
		Quantity:  quantity,
		UpdatedAt: timestamppb.New(updatedAt),
	}
}

func productTypeToString(pt productv1.ProductType) string {
	switch pt {
	case productv1.ProductType_PRODUCT_TYPE_ELECTRONICS:
		return "Electronics"
	case productv1.ProductType_PRODUCT_TYPE_MOBILE_PHONES_ACCESSORIES:
		return "Mobile Phones & Accessories"
	case productv1.ProductType_PRODUCT_TYPE_FASHION:
		return "Fashion"
	case productv1.ProductType_PRODUCT_TYPE_HOME_FURNITURE:
		return "Home & Furniture"
	case productv1.ProductType_PRODUCT_TYPE_BEAUTY_HEALTH:
		return "Beauty & Health"
	case productv1.ProductType_PRODUCT_TYPE_APPLIANCES:
		return "Appliances"
	case productv1.ProductType_PRODUCT_TYPE_AUTOMOTIVE:
		return "Automotive"
	case productv1.ProductType_PRODUCT_TYPE_HARDWARE_CONSTRUCTION:
		return "Hardware & Construction"
	case productv1.ProductType_PRODUCT_TYPE_AGRICULTURE:
		return "Agriculture"
	case productv1.ProductType_PRODUCT_TYPE_SPORTS_OUTDOORS:
		return "Sports & Outdoors"
	case productv1.ProductType_PRODUCT_TYPE_BABY_KIDS:
		return "Baby & Kids"
	case productv1.ProductType_PRODUCT_TYPE_OFFICE_SUPPLIES:
		return "Office Supplies"
	case productv1.ProductType_PRODUCT_TYPE_BOOKS_EDUCATION:
		return "Books & Education"
	case productv1.ProductType_PRODUCT_TYPE_PET_SUPPLIES:
		return "Pet Supplies"
	case productv1.ProductType_PRODUCT_TYPE_DIGITAL_PRODUCTS:
		return "Digital Products"
	case productv1.ProductType_PRODUCT_TYPE_SERVICES:
		return "Services"
	case productv1.ProductType_PRODUCT_TYPE_LIQUIDS:
		return "Liquids"
	case productv1.ProductType_PRODUCT_TYPE_BEVERAGES:
		return "Beverages"
	default:
		return "Electronics"
	}
}

func stringToProductType(s string) productv1.ProductType {
	switch s {
	case "Electronics":
		return productv1.ProductType_PRODUCT_TYPE_ELECTRONICS
	case "Mobile Phones & Accessories":
		return productv1.ProductType_PRODUCT_TYPE_MOBILE_PHONES_ACCESSORIES
	case "Fashion":
		return productv1.ProductType_PRODUCT_TYPE_FASHION
	case "Home & Furniture":
		return productv1.ProductType_PRODUCT_TYPE_HOME_FURNITURE
	case "Beauty & Health":
		return productv1.ProductType_PRODUCT_TYPE_BEAUTY_HEALTH
	case "Appliances":
		return productv1.ProductType_PRODUCT_TYPE_APPLIANCES
	case "Automotive":
		return productv1.ProductType_PRODUCT_TYPE_AUTOMOTIVE
	case "Hardware & Construction":
		return productv1.ProductType_PRODUCT_TYPE_HARDWARE_CONSTRUCTION
	case "Agriculture":
		return productv1.ProductType_PRODUCT_TYPE_AGRICULTURE
	case "Sports & Outdoors":
		return productv1.ProductType_PRODUCT_TYPE_SPORTS_OUTDOORS
	case "Baby & Kids":
		return productv1.ProductType_PRODUCT_TYPE_BABY_KIDS
	case "Office Supplies":
		return productv1.ProductType_PRODUCT_TYPE_OFFICE_SUPPLIES
	case "Books & Education":
		return productv1.ProductType_PRODUCT_TYPE_BOOKS_EDUCATION
	case "Pet Supplies":
		return productv1.ProductType_PRODUCT_TYPE_PET_SUPPLIES
	case "Digital Products":
		return productv1.ProductType_PRODUCT_TYPE_DIGITAL_PRODUCTS
	case "Services":
		return productv1.ProductType_PRODUCT_TYPE_SERVICES
	case "Liquids":
		return productv1.ProductType_PRODUCT_TYPE_LIQUIDS
	case "Beverages":
		return productv1.ProductType_PRODUCT_TYPE_BEVERAGES
	default:
		return productv1.ProductType_PRODUCT_TYPE_UNSPECIFIED
	}
}

func (s *ProductService) ListRecommendedProducts(ctx context.Context, req *productv1.ListRecommendedProductsRequest) ([]*productv1.Product, int32, string, error) {
	pageSize := int32(20)
	if req.PageSize > 0 {
		pageSize = req.PageSize
	}

	lang := "en"
	if req.Language != "" {
		lang = req.Language
	}

	offset := int32(0)
	if req.PageToken != "" {
		fmt.Sscanf(req.PageToken, "offset_%d", &offset)
	}

	var total int64
	countSQL := `
		SELECT COUNT(*) 
		FROM products 
		WHERE deleted_at IS NULL AND status = 'active'`
	err := s.pool.QueryRow(ctx, countSQL).Scan(&total)
	if err != nil {
		return nil, 0, "", fmt.Errorf("count recommended products: %w", err)
	}

	querySQL := `
		SELECT p.id, p.seller_id, p.category_id, p.slug, p.attributes, p.brand, p.origin_country,
		       p.status, p.rating, p.review_count, p.sold_count, p.view_count, p.min_price, p.max_price,
		       p.created_at, p.updated_at, p.deleted_at, p.product_type,
		       ST_Y(p.location::geometry)::float AS latitude,
		       ST_X(p.location::geometry)::float AS longitude,
		       COALESCE(t.title, t_en.title, '') AS title,
		       COALESCE(t.description, t_en.description) AS description
		FROM products p
		LEFT JOIN product_translations t    ON t.product_id = p.id AND t.language = $1
		LEFT JOIN product_translations t_en ON t_en.product_id = p.id AND t_en.language = 'en'
		WHERE p.deleted_at IS NULL AND p.status = 'active'
		ORDER BY p.rating DESC, p.sold_count DESC, p.view_count DESC, p.created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := s.pool.Query(ctx, querySQL, lang, pageSize, offset)
	if err != nil {
		return nil, 0, "", fmt.Errorf("query recommended products: %w", err)
	}
	defer rows.Close()

	var list []db.ProductWithTranslation
	for rows.Next() {
		var i db.ProductWithTranslation
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
		list = append(list, i)
	}
	rows.Close()

	products := make([]*productv1.Product, len(list))
	for i := range list {
		variants, _ := s.q.GetProductVariants(ctx, list[i].ID)
		images, _ := s.q.GetProductImages(ctx, list[i].ID)
		tags, _ := s.q.GetProductTags(ctx, list[i].ID)
		products[i] = assembleProduct(&list[i], variants, images, tags)
	}

	nextPageToken := ""
	if int64(offset)+int64(pageSize) < total {
		nextPageToken = fmt.Sprintf("offset_%d", offset+pageSize)
	}

	return products, int32(total), nextPageToken, nil
}


