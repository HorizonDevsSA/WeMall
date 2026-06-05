package content

import (
	"github.com/wemall/services/documentation-service/internal/models"
)

func GetAPICategories() []models.APICategory {
	return []models.APICategory{
		{
			Slug:     "user-api",
			Title:    "User & Authentication API",
			Overview: "Manages buyer and seller registration, login sessions, address books, profile details, and JWT validation. Buyers log in using passive OTP SMS codes, while sellers register with validated email addresses. The gateway handles token checking for protected resources.",
			Icon:     "👤",
			Endpoints: []models.Endpoint{
				{
					Name:         "buyerSendOTP(phone: String!)",
					Protocol:     "GraphQL Mutation",
					Description:  "Triggers a 6-digit verification code to the buyer's phone number. Bypassed in local development using master code 123456.",
					AuthRequired: false,
					RequestBody:  `mutation { buyerSendOTP(phone: "+263773333333") { message requestId } }`,
					ResponseBody: `{ "data": { "buyerSendOTP": { "message": "OTP sent successfully", "requestId": "mock-request-id-123456" } } }`,
				},
				{
					Name:         "buyerVerifyOTP(phone: String!, otp: String!)",
					Protocol:     "GraphQL Mutation",
					Description:  "Submits OTP code to authenticate. Performs an upsert: creates the user profile automatically if it does not exist, and returns JWT access/refresh tokens.",
					AuthRequired: false,
					RequestBody:  `mutation { buyerVerifyOTP(phone: "+263773333333", otp: "123456") { accessToken refreshToken user { id fullName role } } }`,
					ResponseBody: `{
  "data": {
    "buyerVerifyOTP": {
      "accessToken": "eyJhbGciOiJIUzI1NiIsInR5c...",
      "refreshToken": "eyJhbGciOiJIUzI1NiIsInR5...",
      "user": {
        "id": "0099d266-98c4-4281-92c2-ec2895b1fd4c",
        "fullName": "Tendai Moyo",
        "role": "BUYER"
      }
    }
  }
}`,
				},
				{
					Name:         "sellerRegister(email: String!, password: String!, fullName: String!)",
					Protocol:     "GraphQL Mutation",
					Description:  "Registers a new seller profile with email verification workflow.",
					AuthRequired: false,
					RequestBody:  `mutation { sellerRegister(email: "seller@example.com", password: "Password123!", fullName: "Tatenda Store") { accessToken user { id role } } }`,
					ResponseBody: `{ "data": { "sellerRegister": { "accessToken": "eyJhbG...", "user": { "id": "b50d0370-4809-41a1-a48c-ea3db4805a0c", "role": "SELLER" } } } }`,
				},
				{
					Name:         "me",
					Protocol:     "GraphQL Query",
					Description:  "Retrieves the active user session metadata based on the Authorization Bearer token.",
					AuthRequired: true,
					Roles:        []string{"BUYER", "SELLER", "ADMIN"},
					RequestBody:  `query { me { id email phone fullName role isVerified } }`,
					ResponseBody: `{ "data": { "me": { "id": "0099d266-98c4-4281-92c2", "fullName": "Tendai Moyo", "role": "BUYER", "isVerified": true } } }`,
				},
				{
					Name:         "UserService.ValidateToken(ValidateTokenRequest)",
					Protocol:     "gRPC",
					Description:  "Internal RPC called by API Gateway to check JWT signatures, verify expiration, and resolve role authorization lists.",
					AuthRequired: false,
					RequestBody:  `message ValidateTokenRequest { string token = 1; }`,
					ResponseBody: `message ValidateTokenResponse { string user_id = 1; UserRole role = 2; bool valid = 3; }`,
				},
			},
			DataModels: []models.DataModel{
				{
					Name:        "users (PostgreSQL)",
					Description: "Holds profiles for buyers, sellers, and system administrators.",
					Fields: []models.ModelField{
						{Name: "id", Type: "UUID (Primary Key)", Description: "Unique system user identifier."},
						{Name: "email", Type: "VARCHAR(255)", Description: "Unique user email address (nullable for buyers)."},
						{Name: "phone", Type: "VARCHAR(50)", Description: "Unique mobile phone number (nullable for email-registered sellers)."},
						{Name: "password_hash", Type: "VARCHAR(255)", Description: "Bcrypt hash of user password (empty for phone-only logins)."},
						{Name: "role", Type: "VARCHAR(20)", Description: "User system capability: 'buyer' | 'seller' | 'admin'."},
						{Name: "is_verified", Type: "BOOLEAN", Description: "Indicates whether OTP verification or email confirmation is completed."},
					},
				},
				{
					Name:        "addresses (PostgreSQL)",
					Description: "User delivery and billing address book entries.",
					Fields: []models.ModelField{
						{Name: "id", Type: "UUID (Primary Key)", Description: "Unique address identifier."},
						{Name: "user_id", Type: "UUID", Description: "Foreign key reference to users table."},
						{Name: "full_name", Type: "VARCHAR(255)", Description: "Recipient name for shipping label."},
						{Name: "address_line1", Type: "TEXT", Description: "Street name and number details."},
						{Name: "city", Type: "VARCHAR(100)", Description: "Shipping city."},
						{Name: "country", Type: "VARCHAR(5)", Description: "2-letter ISO country code (e.g. 'ZW')."},
						{Name: "is_default", Type: "BOOLEAN", Description: "Marks standard address used for checkout defaults."},
					},
				},
			},
		},
		{
			Slug:     "seller-api",
			Title:    "Seller & Store API",
			Overview: "Manages seller storefront registration, logo and banner multimedia paths, geographical location settings, following status tracking, and seller payouts.",
			Icon:     "🏬",
			Endpoints: []models.Endpoint{
				{
					Name:         "myStore",
					Protocol:     "GraphQL Query",
					Description:  "Returns the storefront metadata belonging to the authenticated seller session.",
					AuthRequired: true,
					Roles:        []string{"SELLER"},
					RequestBody:  `query { myStore { id storeName storeSlug status isVerified rating } }`,
					ResponseBody: `{ "data": { "myStore": { "id": "117f5472-ac34-44ab-8e5d-9687dfdfc443", "storeName": "Harare CBD Premium Store", "storeSlug": "harare-cbd-premium-store", "status": "VERIFIED", "isVerified": true, "rating": 4.8 } } }`,
				},
				{
					Name:         "createStore(input: CreateStoreInput!)",
					Protocol:     "GraphQL Mutation",
					Description:  "Registers a storefront profile, listing description, and geographical coordinates (essential for nearby products searches). Sets status to PENDING.",
					AuthRequired: true,
					Roles:        []string{"SELLER"},
					RequestBody:  `mutation { createStore(input: { storeName: "Harare CBD Premium Store", description: "Electronics CBD", latitude: -17.8292, longitude: 31.0522 }) { id status } }`,
					ResponseBody: `{ "data": { "createStore": { "id": "117f5472-ac34-44ab-8e5d-9687dfdfc443", "status": "PENDING" } } }`,
				},
				{
					Name:         "updateSellerStatus(sellerId: ID!, status: SellerStatus!)",
					Protocol:     "GraphQL Mutation",
					Description:  "Promotes, suspends, or verifies a storefront profile. Restated for admins only.",
					AuthRequired: true,
					Roles:        []string{"ADMIN"},
					RequestBody:  `mutation { updateSellerStatus(sellerId: "117f5472-ac34-44ab-8e5d-9687dfdfc443", status: VERIFIED) { id isVerified status } }`,
					ResponseBody: `{ "data": { "updateSellerStatus": { "id": "117f5472-ac34-44ab-8e5d-9687dfdfc443", "isVerified": true, "status": "VERIFIED" } } }`,
				},
				{
					Name:         "followStore(sellerId: ID!)",
					Protocol:     "GraphQL Mutation",
					Description:  "Registers a buyer follow connection to receive alerts when products are published.",
					AuthRequired: true,
					Roles:        []string{"BUYER"},
					RequestBody:  `mutation { followStore(sellerId: "117f5472-ac34-44ab-8e5d-9687dfdfc443") }`,
					ResponseBody: `{ "data": { "followStore": true } }`,
				},
			},
			DataModels: []models.DataModel{
				{
					Name:        "sellers (PostgreSQL)",
					Description: "Contains storefront settings, geolocation points, and status ratings.",
					Fields: []models.ModelField{
						{Name: "id", Type: "UUID (Primary Key)", Description: "Unique storefront identifier."},
						{Name: "user_id", Type: "UUID", Description: "Associated owner user identifier."},
						{Name: "store_name", Type: "VARCHAR(255)", Description: "Display name of storefront."},
						{Name: "store_slug", Type: "VARCHAR(255)", Description: "URL-friendly store slug (unique)."},
						{Name: "latitude / longitude", Type: "DOUBLE PRECISION", Description: "Geographical coordinates for local search filters."},
						{Name: "status", Type: "VARCHAR(30)", Description: "Onboarding state: 'PENDING' | 'PROCESSING' | 'VERIFIED' | 'SUSPENDED'."},
					},
				},
				{
					Name:        "store_follows (PostgreSQL)",
					Description: "Junction map tracking which buyers follow which storefronts.",
					Fields: []models.ModelField{
						{Name: "user_id", Type: "UUID", Description: "Follower buyer identifier."},
						{Name: "seller_id", Type: "UUID", Description: "Target storefront seller identifier."},
						{Name: "created_at", Type: "TIMESTAMPTZ", Description: "Timestamp when following began."},
					},
				},
			},
		},
		{
			Slug:     "product-api",
			Title:    "Product & Catalog API",
			Overview: "Governs items taxonomy, categories, variants attributes (price, compare price, options maps), and location-aware distance searches. Integrates PostGIS extensions for spatial queries.",
			Icon:     "🏷️",
			Endpoints: []models.Endpoint{
				{
					Name:         "nearbyProducts(latitude: Float!, longitude: Float!, radiusMeters: Float!)",
					Protocol:     "GraphQL Query",
					Description:  "Fetches products listed near coordinates using PostGIS geography index sorting.",
					AuthRequired: false,
					RequestBody:  `query { nearbyProducts(latitude: -17.83, longitude: 31.05, radiusMeters: 5000) { distance product { id title latitude longitude } } }`,
					ResponseBody: `{
  "data": {
    "nearbyProducts": [
      {
        "distance": 312.4,
        "product": {
          "id": "bc316519-c607-44f2-8405-d110b599c084",
          "title": "Harare Local Product",
          "latitude": -17.8292,
          "longitude": 31.0522
        }
      }
    ]
  }
}`,
				},
				{
					Name:         "createProduct(input: CreateProductInput!)",
					Protocol:     "GraphQL Mutation",
					Description:  "Saves a catalog listing with multiple customizable variants and options. Enforces Category attribute JSON schemas.",
					AuthRequired: true,
					Roles:        []string{"SELLER"},
					RequestBody:  `mutation { createProduct(input: { categoryId: "564bb", title: "iPhone 15", variants: [{ sku: "IP15-256", price: 1199.99, options: { "color": "Black" } }], attributes: { "os": "iOS" } }) { id status } }`,
					ResponseBody: `{ "data": { "createProduct": { "id": "bc316519-c607-44f2-8405-d110b599c084", "status": "ACTIVE" } } }`,
				},
				{
					Name:         "categories",
					Protocol:     "GraphQL Query",
					Description:  "Returns recursive hierarchical category menus with nested child links.",
					AuthRequired: false,
					RequestBody:  `query { categories { id name slug children { id name slug } } }`,
					ResponseBody: `{ "data": { "categories": [ { "id": "5d7d3", "name": "Electronics", "children": [ { "id": "564bb", "name": "Smartphones" } ] } ] } }`,
				},
			},
			DataModels: []models.DataModel{
				{
					Name:        "products (PostgreSQL)",
					Description: "Stores core listing detail records and attributes JSON schemas.",
					Fields: []models.ModelField{
						{Name: "id", Type: "UUID (Primary Key)", Description: "Unique listing identifier."},
						{Name: "seller_id", Type: "UUID", Description: "Associated storefront identifier."},
						{Name: "category_id", Type: "UUID", Description: "Leaf category mapping reference."},
						{Name: "title / description", Type: "TEXT", Description: "Catalog texts."},
						{Name: "attributes", Type: "JSONB", Description: "Category-specific specification maps (e.g. {'os':'iOS'})."},
						{Name: "geom", Type: "GEOGRAPHY(POINT, 4326)", Description: "PostGIS coordinate location marker (synced from seller coordinates)."},
					},
				},
				{
					Name:        "product_variants (PostgreSQL)",
					Description: "Specific purchase models under a parent product (e.g. storage sizes/colors).",
					Fields: []models.ModelField{
						{Name: "id", Type: "UUID (Primary Key)", Description: "Unique variant identifier."},
						{Name: "product_id", Type: "UUID", Description: "Parent product reference key."},
						{Name: "sku", Type: "VARCHAR(100)", Description: "Unique Stock Keeping Unit barcode."},
						{Name: "price / compare_price", Type: "NUMERIC(12,2)", Description: "Active currency listing value and old cross-out value."},
						{Name: "options", Type: "JSONB", Description: "Combination identifier map (e.g. {'storage':'256GB','color':'Titanium'})."},
					},
				},
			},
		},
		{
			Slug:     "inventory-api",
			Title:    "Inventory API",
			Overview: "Private internal gRPC microservice managing stock reservation checkouts, replenishments, and low stock alert thresholds.",
			Icon:     "📦",
			Endpoints: []models.Endpoint{
				{
					Name:         "InventoryService.GetStock(GetStockRequest)",
					Protocol:     "gRPC",
					Description:  "Resolves active physical items and reserved quantities for a specific variant.",
					AuthRequired: false,
					RequestBody:  `message GetStockRequest { string variant_id = 1; }`,
					ResponseBody: `message StockItem { string variant_id = 1; int32 quantity = 2; google.protobuf.Timestamp updated_at = 3; }`,
				},
				{
					Name:         "InventoryService.UpsertStock(UpsertStockRequest)",
					Protocol:     "gRPC",
					Description:  "Saves or updates variant stock records. Triggers NATS notifications if quantity falls below low stock alert thresholds.",
					AuthRequired: false,
					RequestBody:  `message UpsertStockRequest { string variant_id = 1; int32 quantity = 2; }`,
					ResponseBody: `message StockItem { string variant_id = 1; int32 quantity = 2; }`,
				},
			},
			DataModels: []models.DataModel{
				{
					Name:        "inventory (PostgreSQL)",
					Description: "Contains stock quantities and low alert triggers.",
					Fields: []models.ModelField{
						{Name: "id", Type: "UUID (Primary Key)", Description: "Unique inventory line identifier."},
						{Name: "variant_id", Type: "UUID", Description: "Associated product variant identifier."},
						{Name: "quantity", Type: "INTEGER", Description: "Active physical units currently on shelves."},
						{Name: "reserved", Type: "INTEGER", Description: "Units currently locked in pending buyer checkouts (not yet paid)."},
						{Name: "low_stock_alert", Type: "INTEGER", Description: "Alert trigger threshold (defaults to 10)."},
					},
				},
			},
		},
		{
			Slug:     "cart-api",
			Title:    "Cart & Orders API",
			Overview: "Governs buyer shopping carts, checkout transformations, and order histories. Items added to a cart are dynamically hydrated with title and seller details at query time from Product & Seller services. During checkout, an immutable JSON snapshot is generated and frozen inside the order database column to protect receipt history integrity.",
			Icon:     "🛒",
			Endpoints: []models.Endpoint{
				{
					Name:         "cart",
					Protocol:     "GraphQL Query",
					Description:  "Retrieves the hydrated active cart content for the buyer session.",
					AuthRequired: true,
					Roles:        []string{"BUYER"},
					RequestBody:  `query { cart { id itemCount subtotal items { variantId quantity productTitle storeTitle unitPrice } } }`,
					ResponseBody: `{
  "data": {
    "cart": {
      "id": "427005c0-c1be-4cd7-aa2c-afe7265e4c3b",
      "itemCount": 1,
      "subtotal": 1199.99,
      "items": [
        {
          "variantId": "2eef484a-ea4b-4d5f-b762-401a55bd1f97",
          "quantity": 1,
          "productTitle": "iPhone 15 Pro Max",
          "storeTitle": "Cart Test Store",
          "unitPrice": 1199.99
        }
      ]
    }
  }
}`,
				},
				{
					Name:         "addToCart(variantId: ID!, quantity: Int!)",
					Protocol:     "GraphQL Mutation",
					Description:  "Adds a variant item to the cart. Increments quantity if already present.",
					AuthRequired: true,
					Roles:        []string{"BUYER"},
					RequestBody:  `mutation { addToCart(variantId: "2eef484a-ea4b-4d5f-b762-401a55bd1f97", quantity: 1) { itemCount subtotal } }`,
					ResponseBody: `{ "data": { "addToCart": { "itemCount": 1, "subtotal": 1199.99 } } }`,
				},
				{
					Name:         "checkout(input: CheckoutInput!)",
					Protocol:     "GraphQL Mutation",
					Description:  "Transforms active cart contents into an Order, compiles frozen product snapshots, clears the cart, and publishes a wemall.order.created event to NATS.",
					AuthRequired: true,
					Roles:        []string{"BUYER"},
					RequestBody:  `mutation { checkout(input: { shippingAddress: { fullName: "Tendai Moyo", addressLine1: "123 Samora Machel", city: "Harare", country: "ZW" }, currency: USD }) { id orderNumber total status } }`,
					ResponseBody: `{
  "data": {
    "checkout": {
      "id": "a35c3574-f16a-4770-84e6-3b7948e1b15d",
      "orderNumber": "WM-1780664885-cae5e5",
      "total": 1204.99,
      "status": "PENDING"
    }
  }
}`,
				},
			},
			DataModels: []models.DataModel{
				{
					Name:        "cart_items (PostgreSQL)",
					Description: "Contains temporary items added by users. Un-enriched minimal table.",
					Fields: []models.ModelField{
						{Name: "id", Type: "UUID (Primary Key)", Description: "Unique cart item identifier."},
						{Name: "cart_id", Type: "UUID", Description: "Parent shopping cart reference."},
						{Name: "variant_id", Type: "UUID", Description: "Product Variant identifier."},
						{Name: "quantity", Type: "INTEGER", Description: "Requested count."},
					},
				},
				{
					Name:        "order_items (PostgreSQL)",
					Description: "Contains frozen receipt lines containing immutable catalog snapshots.",
					Fields: []models.ModelField{
						{Name: "id", Type: "UUID (Primary Key)", Description: "Unique order item receipt identifier."},
						{Name: "order_id", Type: "UUID", Description: "Parent order reference."},
						{Name: "variant_id", Type: "UUID", Description: "Purchased variant reference."},
						{Name: "snapshot", Type: "JSONB", Description: "Immutable serialized snapshot record containing: variant options, seller logo, product title, and category codes at checkout time."},
					},
				},
			},
		},
		{
			Slug:     "notification-api",
			Title:    "Notification API",
			Overview: "Subscribes to NATS JetStream events, queues delivery jobs via Redis-backed Asynq workers, and dispatches multi-channel outputs (Email via Google SMTP on ports 587/465, Push via Firebase Cloud Messaging). Tracks token validity and opt-ins.",
			Icon:     "🔔",
			Endpoints: []models.Endpoint{
				{
					Name:         "registerDeviceToken(token: String!, platform: String!, deviceName: String)",
					Protocol:     "GraphQL Mutation",
					Description:  "Saves an FCM registration token for mobile push notifications.",
					AuthRequired: true,
					Roles:        []string{"BUYER", "SELLER"},
					RequestBody:  `mutation { registerDeviceToken(token: "fcm_token_12345", platform: "android", deviceName: "Pixel 8") }`,
					ResponseBody: `{ "data": { "registerDeviceToken": true } }`,
				},
				{
					Name:         "updateNotificationPreferences(category: NotificationCategory!, emailEnabled: Boolean!, pushEnabled: Boolean!)",
					Protocol:     "GraphQL Mutation",
					Description:  "Configures user preferences for Transactional, Security, Low Stock, Follows, or Marketing notifications.",
					AuthRequired: true,
					Roles:        []string{"BUYER", "SELLER"},
					RequestBody:  `mutation { updateNotificationPreferences(category: TRANSACTIONAL, emailEnabled: true, pushEnabled: true) { category emailEnabled pushEnabled } }`,
					ResponseBody: `{ "data": { "updateNotificationPreferences": { "category": "TRANSACTIONAL", "emailEnabled": true, "pushEnabled": true } } }`,
				},
			},
			DataModels: []models.DataModel{
				{
					Name:        "user_device_tokens (PostgreSQL)",
					Description: "Valid Firebase FCM registration tokens associated with user accounts.",
					Fields: []models.ModelField{
						{Name: "id", Type: "UUID (Primary Key)", Description: "Unique token line identifier."},
						{Name: "user_id", Type: "UUID", Description: "Associated recipient user identifier."},
						{Name: "token", Type: "TEXT (Unique)", Description: "Firebase registration token string."},
						{Name: "platform", Type: "VARCHAR(20)", Description: "Target OS platform: 'ios' | 'android' | 'web'."},
					},
				},
				{
					Name:        "user_notification_preferences (PostgreSQL)",
					Description: "Opt-in / opt-out preferences governing message channels.",
					Fields: []models.ModelField{
						{Name: "user_id", Type: "UUID", Description: "Target user reference identifier."},
						{Name: "category", Type: "VARCHAR(30)", Description: "Notification category: 'transactional' | 'security' | 'low_stock' | 'follows' | 'marketing'."},
						{Name: "email_enabled / push_enabled", Type: "BOOLEAN", Description: "Channel transmission permissions."},
					},
				},
			},
		},
		{
			Slug:     "media-api",
			Title:    "Media Service API",
			Overview: "Governs secure multimedia file uploads. Integrates direct S3 raw bucket transfers, AWS Lambda event triggers for AVIF/WebP image rendering across 6 size variants, HLS video transcoders, and Origin Access Control signed CloudFront serving.",
			Icon:     "🖼️",
			Endpoints: []models.Endpoint{
				{
					Name:         "POST /api/v1/media/upload",
					Protocol:     "HTTP Route",
					Description:  "Accepts multipart/form-data upload. Internally streams bytes directly to S3 Raw bucket, performs file size checks, and catalogs asset metadata.",
					AuthRequired: true,
					RequestBody:  `[Multipart Form Payload containing "file" binary, "scope" string, and "is_private" boolean]`,
					ResponseBody: `{
  "id": "c71fa793-1386-4554-b52b-b9d9c228807d",
  "owner_id": "8c25345a-c5c9-4b62-97b7-6bbce3725458",
  "original_name": "headset.png",
  "mime_type": "image/png",
  "status": "processing",
  "is_private": false,
  "created_at": "2026-06-05T18:00:00Z"
}`,
				},
				{
					Name:         "GET /api/v1/media",
					Protocol:     "HTTP Route",
					Description:  "Returns a paginated list of media files uploaded by the authenticated user session, with all generated AVIF/WebP responsive URL paths.",
					AuthRequired: true,
					RequestBody:  `GET /api/v1/media?limit=1&scope=product-image HTTP/1.1`,
					ResponseBody: `{
  "assets": [
    {
      "id": "c71fa793-1386-4554-b52b-b9d9c228807d",
      "original_name": "headset.png",
      "status": "completed",
      "variants": {
        "image": {
          "thumbnail_small_avif": "https://cdn.wemall.com/images/c71f/thumbnail_small.avif",
          "thumbnail_large_avif": "https://cdn.wemall.com/images/c71f/thumbnail_large.avif",
          "main_desktop_avif": "https://cdn.wemall.com/images/c71f/main_desktop.avif",
          "main_desktop_webp": "https://cdn.wemall.com/images/c71f/main_desktop.webp"
        }
      }
    }
  ],
  "total_count": 1
}`,
				},
			},
			DataModels: []models.DataModel{
				{
					Name:        "media_assets (PostgreSQL)",
					Description: "Contains asset metadata, ownership scopes, processing states, and variation URL paths maps.",
					Fields: []models.ModelField{
						{Name: "id", Type: "UUID (Primary Key)", Description: "Unique asset identifier."},
						{Name: "owner_id", Type: "UUID", Description: "Uploader user identifier."},
						{Name: "service_scope", Type: "VARCHAR(50)", Description: "Domain context: 'user-avatar' | 'product-image' | 'seller-kyc'."},
						{Name: "status", Type: "VARCHAR(30)", Description: "Processing state: 'pending_upload' | 'uploaded' | 'processing' | 'completed' | 'failed'."},
						{Name: "variants", Type: "JSONB", Description: "Maps containing generated CDN URLs for WebP/AVIF images or HLS manifests."},
						{Name: "is_private", Type: "BOOLEAN", Description: "Determines if files require CloudFront private signatures for access."},
					},
				},
			},
		},
	}
}
