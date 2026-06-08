package resolver

import (
	"encoding/json"
	"time"

	"github.com/wemall/api-gateway/internal/graph/model"
	inventoryv1 "github.com/wemall/gen/inventory/v1"
	notificationv1 "github.com/wemall/gen/notification/v1"
	orderv1 "github.com/wemall/gen/order/v1"
	productv1 "github.com/wemall/gen/product/v1"
	sellerv1 "github.com/wemall/gen/seller/v1"
	userv1 "github.com/wemall/gen/user/v1"
	reviewv1 "github.com/wemall/gen/review/v1"
	paymentv1 "github.com/wemall/gen/payment/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// ── User Mappers ──────────────────────────────────────────────────────────────

func mapUser(u *userv1.User) *model.User {
	if u == nil {
		return nil
	}
	role := model.RoleBuyer
	switch u.Role {
	case userv1.UserRole_USER_ROLE_SELLER:
		role = model.RoleSeller
	case userv1.UserRole_USER_ROLE_ADMIN:
		role = model.RoleAdmin
	}
	var createdAt time.Time
	if u.CreatedAt != nil {
		createdAt = u.CreatedAt.AsTime()
	}
	return &model.User{
		ID:         u.Id,
		Email:      strPtr(u.Email),
		Phone:      strPtr(u.Phone),
		FullName:   u.FullName,
		AvatarURL:  strPtr(u.AvatarUrl),
		Role:       role,
		IsVerified: u.IsVerified,
		CreatedAt:  createdAt,
	}
}

func mapAddress(a *userv1.Address) *model.Address {
	if a == nil {
		return nil
	}
	return &model.Address{
		ID:           a.Id,
		UserID:       a.UserId,
		Label:        strPtr(a.Label),
		FullName:     a.FullName,
		Phone:        a.Phone,
		AddressLine1: a.AddressLine1,
		AddressLine2: strPtr(a.AddressLine2),
		City:         a.City,
		State:        strPtr(a.State),
		PostalCode:   strPtr(a.PostalCode),
		Country:      a.Country,
		IsDefault:    a.IsDefault,
	}
}

func mapAuthPayload(r *userv1.AuthResponse) *model.AuthPayload {
	if r == nil {
		return nil
	}
	return &model.AuthPayload{
		AccessToken:  r.AccessToken,
		RefreshToken: r.RefreshToken,
		User:         mapUser(r.User),
	}
}

// ── Category Mappers ──────────────────────────────────────────────────────────

func mapCategory(c *productv1.Category) *model.Category {
	if c == nil {
		return nil
	}
	children := make([]*model.Category, len(c.Children))
	for i, ch := range c.Children {
		children[i] = mapCategory(ch)
	}
	return &model.Category{
		ID:              c.Id,
		ParentID:        strPtr(c.ParentId),
		Name:            c.Name,
		Slug:            c.Slug,
		IconURL:         strPtr(c.IconUrl),
		BannerURL:       strPtr(c.BannerUrl),
		Level:           int(c.Level),
		SortOrder:       int(c.SortOrder),
		AttributeSchema: structToAny(c.AttributeSchema),
		Children:        children,
	}
}

// ── Product Mappers ───────────────────────────────────────────────────────────

func mapProduct(p *productv1.Product) *model.Product {
	if p == nil {
		return nil
	}

	variants := make([]*model.ProductVariant, len(p.Variants))
	for i, v := range p.Variants {
		variants[i] = mapVariant(v)
	}

	images := make([]*model.ProductImage, len(p.Images))
	for i, img := range p.Images {
		images[i] = mapImage(img)
	}

	status := model.ProductStatusDraft
	switch p.Status {
	case productv1.ProductStatus_PRODUCT_STATUS_ACTIVE:
		status = model.ProductStatusActive
	case productv1.ProductStatus_PRODUCT_STATUS_PAUSED:
		status = model.ProductStatusPaused
	case productv1.ProductStatus_PRODUCT_STATUS_BANNED:
		status = model.ProductStatusBanned
	}

	var createdAt, updatedAt time.Time
	if p.CreatedAt != nil {
		createdAt = p.CreatedAt.AsTime()
	}
	if p.UpdatedAt != nil {
		updatedAt = p.UpdatedAt.AsTime()
	}

	return &model.Product{
		ID:            p.Id,
		SellerID:      p.SellerId,
		CategoryID:    p.CategoryId,
		Title:         p.Title,
		Slug:          p.Slug,
		Description:   strPtr(p.Description),
		Attributes:    structToAny(p.Attributes),
		Brand:         strPtr(p.Brand),
		OriginCountry: strPtr(p.OriginCountry),
		Status:        status,
		Rating:        p.Rating,
		ReviewCount:   int(p.ReviewCount),
		SoldCount:     int(p.SoldCount),
		MinPrice:      float64Ptr(p.MinPrice),
		MaxPrice:      float64Ptr(p.MaxPrice),
		Variants:      variants,
		Images:        images,
		Tags:          p.Tags,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
		Latitude:      float64Ptr(p.Latitude),
		Longitude:     float64Ptr(p.Longitude),
		Distance:      float64Ptr(p.Distance),
		ProductType:   mapProductType(p.ProductType),
		Thumbnail:     strPtr(p.Thumbnail),
		ImageURL:      strPtr(p.ImageUrl),
	}
}

func mapVariant(v *productv1.ProductVariant) *model.ProductVariant {
	if v == nil {
		return nil
	}
	return &model.ProductVariant{
		ID:           v.Id,
		ProductID:    v.ProductId,
		Sku:          v.Sku,
		Options:      structToAny(v.Options),
		Price:        v.Price,
		ComparePrice: float64Ptr(v.ComparePrice),
		ImageURL:     strPtr(v.ImageUrl),
		IsDefault:    v.IsDefault,
	}
}

func mapImage(img *productv1.ProductImage) *model.ProductImage {
	if img == nil {
		return nil
	}
	return &model.ProductImage{
		ID:        img.Id,
		ProductID: img.ProductId,
		URL:       img.Url,
		AltText:   strPtr(img.AltText),
		SortOrder: int(img.SortOrder),
		IsPrimary: img.IsPrimary,
	}
}

func mapProductFilter(f *model.ProductFilterInput) *productv1.ProductFilter {
	if f == nil {
		return nil
	}
	var attrs *structpb.Struct
	if f.Attributes != nil {
		attrs, _ = structpb.NewStruct(jsonToMap(f.Attributes))
	}
	return &productv1.ProductFilter{
		Search:      derefStr(f.Search),
		CategoryId:  derefStr(f.CategoryID),
		SellerId:    derefStr(f.SellerID),
		MinPrice:    derefFloat(f.MinPrice),
		MaxPrice:    derefFloat(f.MaxPrice),
		MinRating:   derefFloat(f.MinRating),
		Tags:        f.Tags,
		Attributes:  attrs,
		InStockOnly: derefBool(f.InStockOnly),
	}
}

// ── Cart & Order Mappers ──────────────────────────────────────────────────────

func mapCart(c *orderv1.Cart) *model.Cart {
	if c == nil {
		return nil
	}
	items := make([]*model.CartItem, len(c.Items))
	for i, item := range c.Items {
		items[i] = &model.CartItem{
			ID:                 item.Id,
			VariantID:          item.VariantId,
			ProductID:          item.ProductId,
			Quantity:           int(item.Quantity),
			UnitPrice:          item.UnitPrice,
			ProductTitle:       item.ProductTitle,
			Variation:          item.Variation,
			VariationThumbnail: item.VariationThumbnail,
			SellerID:           item.SellerId,
			StoreTitle:         item.StoreTitle,
			StoreLogo:          item.StoreLogo,
			Options:            structToAny(item.Options),
			ProductType:        mapOrderProductType(item.ProductType),
		}
	}
	return &model.Cart{
		ID:        c.Id,
		UserID:    c.UserId,
		Items:     items,
		ItemCount: int(c.ItemCount),
		Subtotal:  c.Subtotal,
	}
}

func mapOrder(o *orderv1.Order) *model.Order {
	if o == nil {
		return nil
	}
	items := make([]*model.OrderItem, len(o.Items))
	for i, item := range o.Items {
		items[i] = mapOrderItem(item)
	}

	status := model.OrderStatusPending
	switch o.Status {
	case orderv1.OrderStatus_ORDER_STATUS_CONFIRMED:
		status = model.OrderStatusConfirmed
	case orderv1.OrderStatus_ORDER_STATUS_SHIPPED:
		status = model.OrderStatusShipped
	case orderv1.OrderStatus_ORDER_STATUS_DELIVERED:
		status = model.OrderStatusDelivered
	case orderv1.OrderStatus_ORDER_STATUS_CANCELLED:
		status = model.OrderStatusCancelled
	case orderv1.OrderStatus_ORDER_STATUS_REFUNDED:
		status = model.OrderStatusRefunded
	}

	var createdAt, updatedAt time.Time
	if o.CreatedAt != nil {
		createdAt = o.CreatedAt.AsTime()
	}
	if o.UpdatedAt != nil {
		updatedAt = o.UpdatedAt.AsTime()
	}

	return &model.Order{
		ID:              o.Id,
		OrderNumber:     o.OrderNumber,
		UserID:          o.UserId,
		Status:          status,
		Subtotal:        o.Subtotal,
		ShippingFee:     o.ShippingFee,
		DiscountAmount:  o.DiscountAmount,
		Total:           o.Total,
		ShippingAddress: structToAny(o.ShippingAddress),
		Items:           items,
		CouponCode:      strPtr(o.CouponCode),
		Notes:           strPtr(o.Notes),
		Currency:        o.Currency,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	}
}

func mapOrderItem(item *orderv1.OrderItem) *model.OrderItem {
	if item == nil {
		return nil
	}
	status := model.OrderStatusPending
	switch item.Status {
	case orderv1.OrderStatus_ORDER_STATUS_CONFIRMED:
		status = model.OrderStatusConfirmed
	case orderv1.OrderStatus_ORDER_STATUS_SHIPPED:
		status = model.OrderStatusShipped
	case orderv1.OrderStatus_ORDER_STATUS_DELIVERED:
		status = model.OrderStatusDelivered
	case orderv1.OrderStatus_ORDER_STATUS_CANCELLED:
		status = model.OrderStatusCancelled
	case orderv1.OrderStatus_ORDER_STATUS_REFUNDED:
		status = model.OrderStatusRefunded
	}
	return &model.OrderItem{
		ID:                 item.Id,
		VariantID:          item.VariantId,
		ProductID:          item.ProductId,
		SellerID:           item.SellerId,
		Quantity:           int(item.Quantity),
		UnitPrice:          item.UnitPrice,
		Snapshot:           structToAny(item.Snapshot),
		Status:             status,
		ProductTitle:       item.ProductTitle,
		Variation:          item.Variation,
		VariationThumbnail: item.VariationThumbnail,
		StoreTitle:         item.StoreTitle,
		StoreLogo:          item.StoreLogo,
		Options:            structToAny(item.Options),
		ProductType:        mapOrderProductType(item.ProductType),
	}
}

// ── Seller Mappers ───────────────────────────────────────────────────────────

func mapSeller(s *sellerv1.Seller) *model.Seller {
	if s == nil {
		return nil
	}
	var createdAt, updatedAt time.Time
	if s.CreatedAt != nil {
		createdAt = s.CreatedAt.AsTime()
	}
	if s.UpdatedAt != nil {
		updatedAt = s.UpdatedAt.AsTime()
	}

	// Map seller status from proto to GraphQL model
	sellerStatus := model.SellerStatusPending
	switch s.Status {
	case sellerv1.SellerStatus_SELLER_STATUS_PROCESSING:
		sellerStatus = model.SellerStatusProcessing
	case sellerv1.SellerStatus_SELLER_STATUS_VERIFIED:
		sellerStatus = model.SellerStatusVerified
	case sellerv1.SellerStatus_SELLER_STATUS_SUSPENDED:
		sellerStatus = model.SellerStatusSuspended
	}

	return &model.Seller{
		ID:          s.Id,
		UserID:      s.UserId,
		StoreName:   s.StoreName,
		StoreSlug:   s.StoreSlug,
		LogoURL:     strPtr(s.LogoUrl),
		BannerURL:   strPtr(s.BannerUrl),
		Description: strPtr(s.Description),
		Rating:      s.Rating,
		TotalSales:  int(s.TotalSales),
		IsVerified:  s.IsVerified,
		Status:      sellerStatus,
		Latitude:    float64Ptr(s.Latitude),
		Longitude:   float64Ptr(s.Longitude),
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}

// ── Inventory Mappers ─────────────────────────────────────────────────────────

func mapInventoryInfo(stock *inventoryv1.StockItem) *model.InventoryInfo {
	if stock == nil {
		return &model.InventoryInfo{
			Quantity:  0,
			UpdatedAt: time.Now(),
		}
	}
	var updatedAt time.Time
	if stock.UpdatedAt != nil {
		updatedAt = stock.UpdatedAt.AsTime()
	}
	return &model.InventoryInfo{
		Quantity:  int(stock.Quantity),
		UpdatedAt: updatedAt,
	}
}

// ── Utility Helpers ───────────────────────────────────────────────────────────

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func float64Ptr(f float64) *float64 {
	if f == 0 {
		return nil
	}
	return &f
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefInt(i *int, def int) int {
	if i == nil {
		return def
	}
	return *i
}

func derefFloat(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}

func derefBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

// structToAny converts a protobuf Struct to a map[string]interface{} for the
// GraphQL JSON scalar. Returns nil when the struct is nil or empty.
func structToAny(s *structpb.Struct) map[string]interface{} {
	if s == nil {
		return map[string]interface{}{}
	}
	return s.AsMap()
}

// marshalJSON is a convenience round-trip for arbitrary values.
func marshalJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}

// jsonToMap converts arbitrary JSON-like data to a map for structpb conversion.
func jsonToMap(v interface{}) map[string]interface{} {
	if v == nil {
		return nil
	}
	// Round-trip through JSON for safety
	b := marshalJSON(v)
	var m map[string]interface{}
	json.Unmarshal(b, &m)
	return m
}

// ── Notification Mappers ──────────────────────────────────────────────────────

func mapNotificationCategory(c notificationv1.NotificationCategory) model.NotificationCategory {
	switch c {
	case notificationv1.NotificationCategory_NOTIFICATION_CATEGORY_TRANSACTIONAL:
		return model.NotificationCategoryTransactional
	case notificationv1.NotificationCategory_NOTIFICATION_CATEGORY_SECURITY:
		return model.NotificationCategorySecurity
	case notificationv1.NotificationCategory_NOTIFICATION_CATEGORY_LOW_STOCK:
		return model.NotificationCategoryLowStock
	case notificationv1.NotificationCategory_NOTIFICATION_CATEGORY_FOLLOWS:
		return model.NotificationCategoryFollows
	case notificationv1.NotificationCategory_NOTIFICATION_CATEGORY_MARKETING:
		return model.NotificationCategoryMarketing
	default:
		return model.NotificationCategoryTransactional
	}
}

func unmapNotificationCategory(c model.NotificationCategory) notificationv1.NotificationCategory {
	switch c {
	case model.NotificationCategoryTransactional:
		return notificationv1.NotificationCategory_NOTIFICATION_CATEGORY_TRANSACTIONAL
	case model.NotificationCategorySecurity:
		return notificationv1.NotificationCategory_NOTIFICATION_CATEGORY_SECURITY
	case model.NotificationCategoryLowStock:
		return notificationv1.NotificationCategory_NOTIFICATION_CATEGORY_LOW_STOCK
	case model.NotificationCategoryFollows:
		return notificationv1.NotificationCategory_NOTIFICATION_CATEGORY_FOLLOWS
	case model.NotificationCategoryMarketing:
		return notificationv1.NotificationCategory_NOTIFICATION_CATEGORY_MARKETING
	default:
		return notificationv1.NotificationCategory_NOTIFICATION_CATEGORY_UNSPECIFIED
	}
}

func mapNotificationPreference(p *notificationv1.NotificationPreference) *model.NotificationPreference {
	if p == nil {
		return nil
	}
	return &model.NotificationPreference{
		Category:     mapNotificationCategory(p.Category),
		EmailEnabled: p.EmailEnabled,
		PushEnabled:  p.PushEnabled,
	}
}

func mapNotificationLog(l *notificationv1.NotificationLog) *model.NotificationLog {
	if l == nil {
		return nil
	}
	var createdAt time.Time
	if l.CreatedAt != nil {
		createdAt = l.CreatedAt.AsTime()
	}
	return &model.NotificationLog{
		ID:        l.Id,
		Category:  l.Category,
		Channel:   l.Channel,
		Title:     l.Title,
		Content:   l.Content,
		Status:    l.Status,
		CreatedAt: createdAt,
	}
}

func mapProductType(pt productv1.ProductType) model.ProductType {
	switch pt {
	case productv1.ProductType_PRODUCT_TYPE_ELECTRONICS:
		return model.ProductTypeElectronics
	case productv1.ProductType_PRODUCT_TYPE_MOBILE_PHONES_ACCESSORIES:
		return model.ProductTypeMobilePhonesAccessories
	case productv1.ProductType_PRODUCT_TYPE_FASHION:
		return model.ProductTypeFashion
	case productv1.ProductType_PRODUCT_TYPE_HOME_FURNITURE:
		return model.ProductTypeHomeFurniture
	case productv1.ProductType_PRODUCT_TYPE_BEAUTY_HEALTH:
		return model.ProductTypeBeautyHealth
	case productv1.ProductType_PRODUCT_TYPE_APPLIANCES:
		return model.ProductTypeAppliances
	case productv1.ProductType_PRODUCT_TYPE_AUTOMOTIVE:
		return model.ProductTypeAutomotive
	case productv1.ProductType_PRODUCT_TYPE_HARDWARE_CONSTRUCTION:
		return model.ProductTypeHardwareConstruction
	case productv1.ProductType_PRODUCT_TYPE_AGRICULTURE:
		return model.ProductTypeAgriculture
	case productv1.ProductType_PRODUCT_TYPE_SPORTS_OUTDOORS:
		return model.ProductTypeSportsOutdoors
	case productv1.ProductType_PRODUCT_TYPE_BABY_KIDS:
		return model.ProductTypeBabyKids
	case productv1.ProductType_PRODUCT_TYPE_OFFICE_SUPPLIES:
		return model.ProductTypeOfficeSupplies
	case productv1.ProductType_PRODUCT_TYPE_BOOKS_EDUCATION:
		return model.ProductTypeBooksEducation
	case productv1.ProductType_PRODUCT_TYPE_PET_SUPPLIES:
		return model.ProductTypePetSupplies
	case productv1.ProductType_PRODUCT_TYPE_DIGITAL_PRODUCTS:
		return model.ProductTypeDigitalProducts
	case productv1.ProductType_PRODUCT_TYPE_SERVICES:
		return model.ProductTypeServices
	case productv1.ProductType_PRODUCT_TYPE_LIQUIDS:
		return model.ProductTypeLiquids
	case productv1.ProductType_PRODUCT_TYPE_BEVERAGES:
		return model.ProductTypeBeverages
	default:
		return model.ProductTypeElectronics
	}
}

// unmapProductType converts a GraphQL model.ProductType → productv1.ProductType (for gRPC requests).
func unmapProductType(pt *model.ProductType) productv1.ProductType {
	if pt == nil {
		return productv1.ProductType_PRODUCT_TYPE_UNSPECIFIED
	}
	switch *pt {
	case model.ProductTypeElectronics:
		return productv1.ProductType_PRODUCT_TYPE_ELECTRONICS
	case model.ProductTypeMobilePhonesAccessories:
		return productv1.ProductType_PRODUCT_TYPE_MOBILE_PHONES_ACCESSORIES
	case model.ProductTypeFashion:
		return productv1.ProductType_PRODUCT_TYPE_FASHION
	case model.ProductTypeHomeFurniture:
		return productv1.ProductType_PRODUCT_TYPE_HOME_FURNITURE
	case model.ProductTypeBeautyHealth:
		return productv1.ProductType_PRODUCT_TYPE_BEAUTY_HEALTH
	case model.ProductTypeAppliances:
		return productv1.ProductType_PRODUCT_TYPE_APPLIANCES
	case model.ProductTypeAutomotive:
		return productv1.ProductType_PRODUCT_TYPE_AUTOMOTIVE
	case model.ProductTypeHardwareConstruction:
		return productv1.ProductType_PRODUCT_TYPE_HARDWARE_CONSTRUCTION
	case model.ProductTypeAgriculture:
		return productv1.ProductType_PRODUCT_TYPE_AGRICULTURE
	case model.ProductTypeSportsOutdoors:
		return productv1.ProductType_PRODUCT_TYPE_SPORTS_OUTDOORS
	case model.ProductTypeBabyKids:
		return productv1.ProductType_PRODUCT_TYPE_BABY_KIDS
	case model.ProductTypeOfficeSupplies:
		return productv1.ProductType_PRODUCT_TYPE_OFFICE_SUPPLIES
	case model.ProductTypeBooksEducation:
		return productv1.ProductType_PRODUCT_TYPE_BOOKS_EDUCATION
	case model.ProductTypePetSupplies:
		return productv1.ProductType_PRODUCT_TYPE_PET_SUPPLIES
	case model.ProductTypeDigitalProducts:
		return productv1.ProductType_PRODUCT_TYPE_DIGITAL_PRODUCTS
	case model.ProductTypeServices:
		return productv1.ProductType_PRODUCT_TYPE_SERVICES
	case model.ProductTypeLiquids:
		return productv1.ProductType_PRODUCT_TYPE_LIQUIDS
	case model.ProductTypeBeverages:
		return productv1.ProductType_PRODUCT_TYPE_BEVERAGES
	default:
		return productv1.ProductType_PRODUCT_TYPE_UNSPECIFIED
	}
}

func mapOrderProductType(pt orderv1.ProductType) model.ProductType {
	switch pt {
	case orderv1.ProductType_PRODUCT_TYPE_ELECTRONICS:
		return model.ProductTypeElectronics
	case orderv1.ProductType_PRODUCT_TYPE_MOBILE_PHONES_ACCESSORIES:
		return model.ProductTypeMobilePhonesAccessories
	case orderv1.ProductType_PRODUCT_TYPE_FASHION:
		return model.ProductTypeFashion
	case orderv1.ProductType_PRODUCT_TYPE_HOME_FURNITURE:
		return model.ProductTypeHomeFurniture
	case orderv1.ProductType_PRODUCT_TYPE_BEAUTY_HEALTH:
		return model.ProductTypeBeautyHealth
	case orderv1.ProductType_PRODUCT_TYPE_APPLIANCES:
		return model.ProductTypeAppliances
	case orderv1.ProductType_PRODUCT_TYPE_AUTOMOTIVE:
		return model.ProductTypeAutomotive
	case orderv1.ProductType_PRODUCT_TYPE_HARDWARE_CONSTRUCTION:
		return model.ProductTypeHardwareConstruction
	case orderv1.ProductType_PRODUCT_TYPE_AGRICULTURE:
		return model.ProductTypeAgriculture
	case orderv1.ProductType_PRODUCT_TYPE_SPORTS_OUTDOORS:
		return model.ProductTypeSportsOutdoors
	case orderv1.ProductType_PRODUCT_TYPE_BABY_KIDS:
		return model.ProductTypeBabyKids
	case orderv1.ProductType_PRODUCT_TYPE_OFFICE_SUPPLIES:
		return model.ProductTypeOfficeSupplies
	case orderv1.ProductType_PRODUCT_TYPE_BOOKS_EDUCATION:
		return model.ProductTypeBooksEducation
	case orderv1.ProductType_PRODUCT_TYPE_PET_SUPPLIES:
		return model.ProductTypePetSupplies
	case orderv1.ProductType_PRODUCT_TYPE_DIGITAL_PRODUCTS:
		return model.ProductTypeDigitalProducts
	case orderv1.ProductType_PRODUCT_TYPE_SERVICES:
		return model.ProductTypeServices
	case orderv1.ProductType_PRODUCT_TYPE_LIQUIDS:
		return model.ProductTypeLiquids
	case orderv1.ProductType_PRODUCT_TYPE_BEVERAGES:
		return model.ProductTypeBeverages
	default:
		return model.ProductTypeElectronics
	}
}

// ── Review Mappers ───────────────────────────────────────────────────────────

func mapReview(r *reviewv1.Review) *model.Review {
	if r == nil {
		return nil
	}
	mediaList := make([]*model.ReviewMedia, len(r.Media))
	for i, m := range r.Media {
		mediaList[i] = &model.ReviewMedia{
			ID:        m.Id,
			MediaURL:  m.Url,
			MediaType: m.MediaType,
		}
	}
	repliesList := make([]*model.SellerReply, len(r.Replies))
	for i, rep := range r.Replies {
		var createdAt time.Time
		if rep.CreatedAt != nil {
			createdAt = rep.CreatedAt.AsTime()
		}
		repliesList[i] = &model.SellerReply{
			ID:        rep.Id,
			ReplyType: rep.ReplyType,
			Content:   rep.Content,
			CreatedAt: createdAt,
		}
	}
	var createdAt, updatedAt time.Time
	if r.CreatedAt != nil {
		createdAt = r.CreatedAt.AsTime()
	}
	if r.UpdatedAt != nil {
		updatedAt = r.UpdatedAt.AsTime()
	}
	return &model.Review{
		ID:                r.Id,
		OrderID:           r.OrderId,
		BuyerID:           r.BuyerId,
		BuyerName:         "Buyer " + r.BuyerId,
		BuyerAvatar:       nil,
		RatingDescription: int(r.RatingDescription),
		RatingService:     int(r.RatingService),
		RatingDelivery:    int(r.RatingDelivery),
		ReviewType:        r.ReviewType,
		Content:           strPtr(r.Content),
		IsAnonymous:       r.IsAnonymous,
		HasMedia:          r.HasMedia,
		Media:             mediaList,
		NlpTags:           r.NlpTags,
		IsSystemGenerated: r.IsSystemGenerated,
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
		AppendReview:      mapAppendReview(r.AppendReview),
		Replies:           repliesList,
	}
}

func mapAppendReview(ar *reviewv1.AppendReview) *model.AppendReview {
	if ar == nil {
		return nil
	}
	mediaList := make([]*model.ReviewMedia, len(ar.Media))
	for i, m := range ar.Media {
		mediaList[i] = &model.ReviewMedia{
			ID:        m.Id,
			MediaURL:  m.Url,
			MediaType: m.MediaType,
		}
	}
	var createdAt time.Time
	if ar.CreatedAt != nil {
		createdAt = ar.CreatedAt.AsTime()
	}
	return &model.AppendReview{
		ID:        ar.Id,
		Content:   ar.Content,
		HasMedia:  ar.HasMedia,
		Media:     mediaList,
		CreatedAt: createdAt,
	}
}

func mapSellerReply(rep *reviewv1.SellerReply) *model.SellerReply {
	if rep == nil {
		return nil
	}
	var createdAt time.Time
	if rep.CreatedAt != nil {
		createdAt = rep.CreatedAt.AsTime()
	}
	return &model.SellerReply{
		ID:        rep.Id,
		ReplyType: rep.ReplyType,
		Content:   rep.Content,
		CreatedAt: createdAt,
	}
}

func mapProductReviewStats(s *reviewv1.ProductRatingStats) *model.ProductReviewStats {
	if s == nil {
		return nil
	}
	return &model.ProductReviewStats{
		AverageRating: float64(s.AverageRating),
		TotalReviews:  int(s.TotalReviews),
		GoodCount:     int(s.GoodCount),
		NeutralCount:  int(s.NeutralCount),
		BadCount:      int(s.BadCount),
		HasMediaCount: int(s.HasMediaCount),
		AppendCount:   int(s.AppendCount),
		TopTags:       s.TopNlpTags,
	}
}

func mapSellerDsr(d *reviewv1.SellerDSR) *model.SellerDsr {
	if d == nil {
		return nil
	}
	return &model.SellerDsr{
		AvgDescription:  float64(d.AvgDescription),
		AvgService:      float64(d.AvgService),
		AvgDelivery:     float64(d.AvgDelivery),
		ReputationScore: int(d.ReputationScore),
	}
}

// ── Payment Mappers ──────────────────────────────────────────────────────────

func mapPayment(p *paymentv1.Payment) *model.Payment {
	if p == nil {
		return nil
	}
	var createdAt, updatedAt time.Time
	if p.CreatedAt != nil {
		createdAt = p.CreatedAt.AsTime()
	}
	if p.UpdatedAt != nil {
		updatedAt = p.UpdatedAt.AsTime()
	}
	txnID := ""
	if p.TransactionId != "" {
		txnID = p.TransactionId
	}
	return &model.Payment{
		ID:            p.Id,
		OrderID:       p.OrderId,
		UserID:        p.UserId,
		Amount:        p.Amount,
		Currency:      p.Currency,
		Provider:      mapPaymentProvider(p.Provider),
		Status:        mapPaymentStatus(p.Status),
		TransactionID: &txnID,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}
}

func mapPaymentProvider(p paymentv1.PaymentProvider) model.PaymentProvider {
	switch p {
	case paymentv1.PaymentProvider_PAYMENT_PROVIDER_GOOGLE_PAY:
		return model.PaymentProviderGooglePay
	case paymentv1.PaymentProvider_PAYMENT_PROVIDER_STRIPE:
		return model.PaymentProviderStripe
	default:
		return model.PaymentProviderGooglePay
	}
}

func mapPaymentStatus(s paymentv1.PaymentStatus) model.PaymentStatus {
	switch s {
	case paymentv1.PaymentStatus_PAYMENT_STATUS_PENDING:
		return model.PaymentStatusPending
	case paymentv1.PaymentStatus_PAYMENT_STATUS_COMPLETED:
		return model.PaymentStatusCompleted
	case paymentv1.PaymentStatus_PAYMENT_STATUS_FAILED:
		return model.PaymentStatusFailed
	case paymentv1.PaymentStatus_PAYMENT_STATUS_REFUNDED:
		return model.PaymentStatusRefunded
	default:
		return model.PaymentStatusPending
	}
}

func unmapPaymentProvider(p model.PaymentProvider) paymentv1.PaymentProvider {
	switch p {
	case model.PaymentProviderGooglePay:
		return paymentv1.PaymentProvider_PAYMENT_PROVIDER_GOOGLE_PAY
	case model.PaymentProviderStripe:
		return paymentv1.PaymentProvider_PAYMENT_PROVIDER_STRIPE
	default:
		return paymentv1.PaymentProvider_PAYMENT_PROVIDER_UNSPECIFIED
	}
}


