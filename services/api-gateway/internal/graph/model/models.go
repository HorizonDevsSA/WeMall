// Package model contains the Go types that correspond to the GraphQL schema types.
// In a full gqlgen setup these would be generated; here they are hand-written to
// match schema.graphql exactly so the resolver package compiles without running
// the code generator.
package model

import "time"

// ── Enums ─────────────────────────────────────────────────────────────────────

type Role string

const (
	RoleBuyer  Role = "BUYER"
	RoleSeller Role = "SELLER"
	RoleAdmin  Role = "ADMIN"
)

type SellerStatus string

const (
	SellerStatusPending    SellerStatus = "PENDING"
	SellerStatusProcessing SellerStatus = "PROCESSING"
	SellerStatusVerified   SellerStatus = "VERIFIED"
	SellerStatusSuspended  SellerStatus = "SUSPENDED"
)

type Currency string

const (
	CurrencyUSD Currency = "USD"
	CurrencyZWG Currency = "ZWG"
)

type ProductStatus string

const (
	ProductStatusDraft  ProductStatus = "DRAFT"
	ProductStatusActive ProductStatus = "ACTIVE"
	ProductStatusPaused ProductStatus = "PAUSED"
	ProductStatusBanned ProductStatus = "BANNED"
)

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "PENDING"
	OrderStatusConfirmed OrderStatus = "CONFIRMED"
	OrderStatusShipped   OrderStatus = "SHIPPED"
	OrderStatusDelivered OrderStatus = "DELIVERED"
	OrderStatusCancelled OrderStatus = "CANCELLED"
	OrderStatusRefunded  OrderStatus = "REFUNDED"
)

type ProductType string

const (
	ProductTypeElectronics             ProductType = "ELECTRONICS"
	ProductTypeMobilePhonesAccessories ProductType = "MOBILE_PHONES_ACCESSORIES"
	ProductTypeFashion                 ProductType = "FASHION"
	ProductTypeHomeFurniture           ProductType = "HOME_FURNITURE"
	ProductTypeBeautyHealth            ProductType = "BEAUTY_HEALTH"
	ProductTypeAppliances              ProductType = "APPLIANCES"
	ProductTypeAutomotive              ProductType = "AUTOMOTIVE"
	ProductTypeHardwareConstruction    ProductType = "HARDWARE_CONSTRUCTION"
	ProductTypeAgriculture             ProductType = "AGRICULTURE"
	ProductTypeSportsOutdoors          ProductType = "SPORTS_OUTDOORS"
	ProductTypeBabyKids                ProductType = "BABY_KIDS"
	ProductTypeOfficeSupplies          ProductType = "OFFICE_SUPPLIES"
	ProductTypeBooksEducation          ProductType = "BOOKS_EDUCATION"
	ProductTypePetSupplies             ProductType = "PET_SUPPLIES"
	ProductTypeDigitalProducts         ProductType = "DIGITAL_PRODUCTS"
	ProductTypeServices                ProductType = "SERVICES"
	ProductTypeLiquids                 ProductType = "LIQUIDS"
	ProductTypeBeverages               ProductType = "BEVERAGES"
)

// ── User Types ────────────────────────────────────────────────────────────────

type User struct {
	ID         string    `json:"id"`
	Email      *string   `json:"email"`
	Phone      *string   `json:"phone"`
	FullName   string    `json:"fullName"`
	AvatarURL  *string   `json:"avatarUrl"`
	Role       Role      `json:"role"`
	IsVerified bool      `json:"isVerified"`
	CreatedAt  time.Time `json:"createdAt"`
}

type Address struct {
	ID           string  `json:"id"`
	UserID       string  `json:"userId"`
	Label        *string `json:"label"`
	FullName     string  `json:"fullName"`
	Phone        string  `json:"phone"`
	AddressLine1 string  `json:"addressLine1"`
	AddressLine2 *string `json:"addressLine2"`
	City         string  `json:"city"`
	State        *string `json:"state"`
	PostalCode   *string `json:"postalCode"`
	Country      string  `json:"country"`
	IsDefault    bool    `json:"isDefault"`
}

type AuthPayload struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	User         *User  `json:"user"`
}

type OTPPayload struct {
	Message   string `json:"message"`
	RequestID string `json:"requestId"`
}

// ── Category Types ────────────────────────────────────────────────────────────

type Category struct {
	ID              string                 `json:"id"`
	ParentID        *string                `json:"parentId"`
	Name            string                 `json:"name"`
	Slug            string                 `json:"slug"`
	IconURL         *string                `json:"iconUrl"`
	BannerURL       *string                `json:"bannerUrl"`
	Level           int                    `json:"level"`
	SortOrder       int                    `json:"sortOrder"`
	AttributeSchema map[string]interface{} `json:"attributeSchema"`
	Children        []*Category            `json:"children"`
}

// ── Product Types ─────────────────────────────────────────────────────────────

type InventoryInfo struct {
	Quantity  int       `json:"quantity"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ProductVariant struct {
	ID           string                 `json:"id"`
	ProductID    string                 `json:"productId"`
	Sku          string                 `json:"sku"`
	Options      map[string]interface{} `json:"options"`
	Price        float64                `json:"price"`
	ComparePrice *float64               `json:"comparePrice"`
	ImageURL     *string                `json:"imageUrl"`
	IsDefault    bool                   `json:"isDefault"`
	Inventory    *InventoryInfo         `json:"inventory"`
}

type ProductImage struct {
	ID        string  `json:"id"`
	ProductID string  `json:"productId"`
	URL       string  `json:"url"`
	AltText   *string `json:"altText"`
	SortOrder int     `json:"sortOrder"`
	IsPrimary bool    `json:"isPrimary"`
}

type Seller struct {
	ID          string       `json:"id"`
	UserID      string       `json:"userId"`
	StoreName   string       `json:"storeName"`
	StoreSlug   string       `json:"storeSlug"`
	LogoURL     *string      `json:"logoUrl"`
	BannerURL   *string      `json:"bannerUrl"`
	Description *string      `json:"description"`
	Rating      float64      `json:"rating"`
	TotalSales  int          `json:"totalSales"`
	IsVerified  bool         `json:"isVerified"`
	Status      SellerStatus `json:"status"`
	Latitude    *float64     `json:"latitude"`
	Longitude   *float64     `json:"longitude"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`
}

type FollowedStoresList struct {
	Sellers       []*Seller `json:"sellers"`
	NextPageToken *string   `json:"nextPageToken"`
	Total         int       `json:"total"`
}

type Product struct {
	ID            string                 `json:"id"`
	SellerID      string                 `json:"sellerId"`
	CategoryID    string                 `json:"categoryId"`
	Title         string                 `json:"title"`
	Slug          string                 `json:"slug"`
	Description   *string                `json:"description"`
	Attributes    map[string]interface{} `json:"attributes"`
	Brand         *string                `json:"brand"`
	OriginCountry *string                `json:"originCountry"`
	Status        ProductStatus          `json:"status"`
	Rating        float64                `json:"rating"`
	ReviewCount   int                    `json:"reviewCount"`
	SoldCount     int                    `json:"soldCount"`
	MinPrice      *float64               `json:"minPrice"`
	MaxPrice      *float64               `json:"maxPrice"`
	Variants      []*ProductVariant      `json:"variants"`
	Images        []*ProductImage        `json:"images"`
	Tags          []string               `json:"tags"`
	CreatedAt     time.Time              `json:"createdAt"`
	UpdatedAt     time.Time              `json:"updatedAt"`
	Latitude      *float64               `json:"latitude"`
	Longitude     *float64               `json:"longitude"`
	Distance      *float64               `json:"distance"`
	ProductType   ProductType            `json:"productType"`
	Thumbnail     *string                `json:"thumbnail"`
	ImageURL      *string                `json:"imageUrl"`
	Seller        *Seller                `json:"seller"`
}

type ProductList struct {
	Products      []*Product `json:"products"`
	NextPageToken *string    `json:"nextPageToken"`
	Total         int        `json:"total"`
}

// ── Cart & Order Types ────────────────────────────────────────────────────────

type CartItem struct {
	ID                 string                 `json:"id"`
	VariantID          string                 `json:"variantId"`
	ProductID          string                 `json:"productId"`
	Quantity           int                    `json:"quantity"`
	UnitPrice          float64                `json:"unitPrice"`
	ProductTitle       string                 `json:"productTitle"`
	Variation          string                 `json:"variation"`
	VariationThumbnail string                 `json:"variationThumbnail"`
	SellerID           string                 `json:"sellerId"`
	StoreTitle         string                 `json:"storeTitle"`
	StoreLogo          string                 `json:"storeLogo"`
	Options            map[string]interface{} `json:"options"`
	ProductType        ProductType            `json:"productType"`
}

type Cart struct {
	ID        string      `json:"id"`
	UserID    string      `json:"userId"`
	Items     []*CartItem `json:"items"`
	ItemCount int         `json:"itemCount"`
	Subtotal  float64     `json:"subtotal"`
}

type OrderItem struct {
	ID                 string                 `json:"id"`
	VariantID          string                 `json:"variantId"`
	ProductID          string                 `json:"productId"`
	SellerID           string                 `json:"sellerId"`
	Quantity           int                    `json:"quantity"`
	UnitPrice          float64                `json:"unitPrice"`
	Snapshot           map[string]interface{} `json:"snapshot"`
	Status             OrderStatus            `json:"status"`
	ProductTitle       string                 `json:"productTitle"`
	Variation          string                 `json:"variation"`
	VariationThumbnail string                 `json:"variationThumbnail"`
	StoreTitle         string                 `json:"storeTitle"`
	StoreLogo          string                 `json:"storeLogo"`
	Options            map[string]interface{} `json:"options"`
	ProductType        ProductType            `json:"productType"`
}

type Order struct {
	ID              string                 `json:"id"`
	OrderNumber     string                 `json:"orderNumber"`
	UserID          string                 `json:"userId"`
	Status          OrderStatus            `json:"status"`
	Subtotal        float64                `json:"subtotal"`
	ShippingFee     float64                `json:"shippingFee"`
	DiscountAmount  float64                `json:"discountAmount"`
	Total           float64                `json:"total"`
	ShippingAddress map[string]interface{} `json:"shippingAddress"`
	Items           []*OrderItem           `json:"items"`
	CouponCode      *string                `json:"couponCode"`
	Notes           *string                `json:"notes"`
	Currency        string                 `json:"currency"`
	CreatedAt       time.Time              `json:"createdAt"`
	UpdatedAt       time.Time              `json:"updatedAt"`
}

type OrderList struct {
	Orders        []*Order `json:"orders"`
	NextPageToken *string  `json:"nextPageToken"`
	Total         int      `json:"total"`
}

// ── Input Types ───────────────────────────────────────────────────────────────

type ProductFilterInput struct {
	Search      *string                `json:"search"`
	CategoryID  *string                `json:"categoryId"`
	SellerID    *string                `json:"sellerId"`
	MinPrice    *float64               `json:"minPrice"`
	MaxPrice    *float64               `json:"maxPrice"`
	MinRating   *float64               `json:"minRating"`
	Tags        []string               `json:"tags"`
	Attributes  map[string]interface{} `json:"attributes"`
	InStockOnly *bool                  `json:"inStockOnly"`
}

type VariantInput struct {
	Sku          string                 `json:"sku"`
	Options      map[string]interface{} `json:"options"`
	Price        float64                `json:"price"`
	ComparePrice *float64               `json:"comparePrice"`
}

type CreateProductInput struct {
	CategoryID    string                 `json:"categoryId"`
	Title         string                 `json:"title"`
	Description   *string                `json:"description"`
	Attributes    map[string]interface{} `json:"attributes"`
	Brand         *string                `json:"brand"`
	OriginCountry *string                `json:"originCountry"`
	Variants      []VariantInput         `json:"variants"`
	Tags          []string               `json:"tags"`
	Language      *string                `json:"language"`
	ProductType   *ProductType           `json:"productType"`
}

type UpdateProductInput struct {
	Title       *string                `json:"title"`
	Description *string                `json:"description"`
	Attributes  map[string]interface{} `json:"attributes"`
	Brand       *string                `json:"brand"`
	Status      *ProductStatus         `json:"status"`
	Language    *string                `json:"language"`
}

type AddressInput struct {
	Label        *string `json:"label"`
	FullName     string  `json:"fullName"`
	Phone        string  `json:"phone"`
	AddressLine1 string  `json:"addressLine1"`
	AddressLine2 *string `json:"addressLine2"`
	City         string  `json:"city"`
	State        *string `json:"state"`
	PostalCode   *string `json:"postalCode"`
	Country      string  `json:"country"`
	IsDefault    *bool   `json:"isDefault"`
}

type ShippingAddressInput struct {
	FullName     string  `json:"fullName"`
	Phone        string  `json:"phone"`
	AddressLine1 string  `json:"addressLine1"`
	AddressLine2 *string `json:"addressLine2"`
	City         string  `json:"city"`
	State        *string `json:"state"`
	PostalCode   *string `json:"postalCode"`
	Country      string  `json:"country"`
}

type CheckoutInput struct {
	ShippingAddress ShippingAddressInput `json:"shippingAddress"`
	CouponCode      *string              `json:"couponCode"`
	Notes           *string              `json:"notes"`
	Currency        *Currency            `json:"currency"`
}

type CreateStoreInput struct {
	StoreName   string   `json:"storeName"`
	Description *string  `json:"description"`
	LogoURL     *string  `json:"logoUrl"`
	BannerURL   *string  `json:"bannerUrl"`
	Latitude    *float64 `json:"latitude"`
	Longitude   *float64 `json:"longitude"`
}

type UpdateStoreInput struct {
	StoreName   *string  `json:"storeName"`
	Description *string  `json:"description"`
	LogoURL     *string  `json:"logoUrl"`
	BannerURL   *string  `json:"bannerUrl"`
	Latitude    *float64 `json:"latitude"`
	Longitude   *float64 `json:"longitude"`
}
