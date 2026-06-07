# WeMall — Microservices + GraphQL Gateway Architecture

A production-grade, multi-vendor marketplace platform built the way Amazon, Alibaba, and eBay do it: **domain-isolated microservices** behind a **GraphQL API Gateway**.

---

## Overview

```
┌─────────────────────────────────────────────┐
│              Clients                        │
│   Mobile App │ Web App │ Admin Dashboard    │
└──────────────────┬──────────────────────────┘
                   │  GraphQL (HTTP + WS)
┌──────────────────▼──────────────────────────┐
│          API Gateway Service                │
│       (gqlgen · Go · Fiber)                 │
│  • GraphQL schema + resolvers               │
│  • DataLoaders (batch gRPC calls)           │
│  • JWT validation                           │
│  • @hasRole directive                       │
└──┬──────┬──────┬──────┬──────┬──────┬──────┘
   │      │      │      │      │      │   gRPC
   ▼      ▼      ▼      ▼      ▼      ▼
┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐
│ User │ │Seller│ │Prod. │ │Inven-│ │Order │ │Pay-  │
│ Svc  │ │ Svc  │ │ Svc  │ │tory  │ │ Svc  │ │ment  │
│      │ │      │ │      │ │ Svc  │ │      │ │ Svc  │
└──┬───┘ └──┬───┘ └──┬───┘ └──┬───┘ └──┬───┘ └──┬───┘
   │        │        │        │        │        │
   └────────┴────────┴────────┴────────┴────────┘
                          │
              ┌───────────▼───────────┐
              │   NATS JetStream      │  ← async events
              │  (event bus)          │
              └───────────┬───────────┘
                          │
              ┌───────────▼───────────┐
              │  Notification Service │
              │  Search Service       │
              └───────────────────────┘
```

---

## Technology Stack

| Layer | Technology | Rationale |
|---|---|---|
| **Language** | Go 1.22+ | Fast, small binaries, excellent concurrency, first-class gRPC support |
| **Gateway Framework** | Fiber v2 + `gqlgen` | GraphQL schema-first, fastest HTTP layer |
| **Service Framework** | Fiber v2 (health/metrics) | Each service exposes `/health` and `/metrics` |
| **Service RPC** | gRPC + Protocol Buffers | Typed contracts, bi-directional streaming, ~10× faster than REST |
| **Event Bus** | NATS JetStream | Go-native, persistent, at-least-once delivery; simpler than Kafka for this scale |
| **Primary DB** | PostgreSQL 16 (per-service) | Each service owns its own database — no shared schema |
| **DB Driver** | `pgx/v5` | Fastest pure-Go Postgres driver |
| **Query Layer** | `sqlc` | Type-safe Go code generated from SQL |
| **Migrations** | `golang-migrate` | Per-service migration files |
| **Search** | Meilisearch (Search Service) | Full-text + faceted search |
| **Cache** | Redis 7 | Gateway-level caching, session store, rate limiting |
| **Object Storage** | Cloudflare R2 / AWS S3 | Product images |
| **Auth** | JWT (`golang-jwt/jwt/v5`) | Validated at the gateway; services trust gateway-injected headers |
| **Queue / Jobs** | `hibiken/asynq` (Redis) | Background jobs inside services (email, sync) |
| **Config** | `spf13/viper` | 12-factor, per-service `.env` |
| **Logging** | `rs/zerolog` | Structured JSON, correlation IDs propagated via context |
| **Tracing** | OpenTelemetry + Jaeger | Distributed trace across all services |
| **Containerisation** | Docker + Docker Compose | Local dev; each service in its own container |

---

## Confirmed Decisions

> [!NOTE]
> All decisions locked in — implementation begins now.

| # | Decision | Choice |
|---|---|---|
| 1 | **Deployment** | Docker Compose only (MVP) |
| 2 | **Auth — Buyers** | Google OAuth + Phone OTP (SMS) |
| 3 | **Auth — Sellers** | Email + Password with email verification |
| 4 | **Payments** | Stripe + PayPal |
| 5 | **Currencies** | USD (base) + ZWG (Zimbabwe Gold) |
| 6 | **Languages** | English (`en`), Shona (`sn`), Ndebele (`nd`) |
| 7 | **Phase 1 scope** | Full working resolvers included |
| 8 | **Services (Phase 1)** | 4 core services: api-gateway, user-service, product-service, order-service |

### Auth Architecture Detail

**Buyers** — two paths:
- `Google OAuth 2.0` → `golang.org/x/oauth2` → exchange code for profile → upsert user
- `Phone OTP` → Africa's Talking SMS API (Zimbabwe-optimised) → 6-digit OTP → verify → JWT

**Sellers** — email/password path:
- `Register` → bcrypt password → send email verification link
- `Login` → verify email confirmed → return JWT
- Sellers must complete store setup (store name, description) before listing products

### i18n Architecture Detail

Translations stored in the DB alongside base records:
```sql
-- product_translations (in product-service)
product_id   UUID REFERENCES products(id)
language     TEXT NOT NULL   -- 'en' | 'sn' | 'nd'
title        TEXT NOT NULL
description  TEXT
PRIMARY KEY (product_id, language)

-- category_translations
category_id  UUID REFERENCES categories(id)
language     TEXT NOT NULL
name         TEXT NOT NULL
PRIMARY KEY (category_id, language)
```
GraphQL queries accept an optional `language: String` argument (defaults to `en`).

### Currency Architecture Detail

- All prices stored in **USD** (base currency)
- ZWG exchange rate fetched daily and cached in Redis
- GraphQL `Product.price` field accepts a `currency: Currency` argument → converts on the fly
- `enum Currency { USD ZWG }`

---

## Service Catalog

| Service | Port | DB | Responsibility |
|---|---|---|---|
| `api-gateway` | 8080 | Redis (cache) | GraphQL schema, resolvers → gRPC calls, JWT auth |
| `user-service` | 9001 | `wemall_users` | Register, login, profiles, addresses, JWT issuance |
| `seller-service` | 9002 | `wemall_sellers` | Store management, seller verification, payouts |
| `product-service` | 9003 | `wemall_products` | Products, categories, variants, images, tags |
| `inventory-service` | 9004 | `wemall_inventory` | Stock levels, reservations, warehouse management |
| `order-service` | 9005 | `wemall_orders` | Cart, checkout, order lifecycle |
| `payment-service` | 9006 | `wemall_payments` | Payment processing, webhook handling, refunds |
| `notification-service` | 9007 | — | Email, push notifications (event-driven, no gRPC) |
| `search-service` | 9008 | Meilisearch | Full-text + faceted product search |

---

## Database Schema Per Service

Each service **owns its own PostgreSQL database**. No cross-service joins — data needed across boundaries is fetched via gRPC.

### `wemall_users` (User Service)

```sql
-- users
id              UUID PRIMARY KEY DEFAULT gen_random_uuid()
email           TEXT UNIQUE NOT NULL
phone           TEXT UNIQUE
password_hash   TEXT NOT NULL
full_name       TEXT NOT NULL
avatar_url      TEXT
role            TEXT NOT NULL DEFAULT 'buyer'   -- buyer | seller | admin
is_verified     BOOLEAN DEFAULT false
created_at      TIMESTAMPTZ DEFAULT now()
updated_at      TIMESTAMPTZ DEFAULT now()
deleted_at      TIMESTAMPTZ

-- refresh_tokens
id          UUID PRIMARY KEY DEFAULT gen_random_uuid()
user_id     UUID NOT NULL REFERENCES users(id)
token_hash  TEXT NOT NULL UNIQUE
expires_at  TIMESTAMPTZ NOT NULL
revoked_at  TIMESTAMPTZ
created_at  TIMESTAMPTZ DEFAULT now()

-- addresses
id              UUID PRIMARY KEY DEFAULT gen_random_uuid()
user_id         UUID NOT NULL REFERENCES users(id)
label           TEXT
full_name       TEXT NOT NULL
phone           TEXT NOT NULL
address_line1   TEXT NOT NULL
address_line2   TEXT
city            TEXT NOT NULL
state           TEXT
postal_code     TEXT
country         TEXT NOT NULL DEFAULT 'CN'
is_default      BOOLEAN DEFAULT false
created_at      TIMESTAMPTZ DEFAULT now()
```

---

### `wemall_sellers` (Seller Service)

```sql
-- sellers
id              UUID PRIMARY KEY DEFAULT gen_random_uuid()
user_id         UUID NOT NULL UNIQUE      -- FK to users DB (enforced by app logic)
store_name      TEXT NOT NULL UNIQUE
store_slug      TEXT NOT NULL UNIQUE
logo_url        TEXT
banner_url      TEXT
description     TEXT
rating          NUMERIC(3,2) DEFAULT 0
total_sales     INTEGER DEFAULT 0
is_verified     BOOLEAN DEFAULT false
created_at      TIMESTAMPTZ DEFAULT now()
updated_at      TIMESTAMPTZ DEFAULT now()

-- seller_payouts
id          UUID PRIMARY KEY DEFAULT gen_random_uuid()
seller_id   UUID NOT NULL REFERENCES sellers(id)
amount      NUMERIC(12,2) NOT NULL
currency    TEXT DEFAULT 'USD'
status      TEXT DEFAULT 'pending'        -- pending | processing | paid | failed
provider_ref TEXT
paid_at     TIMESTAMPTZ
created_at  TIMESTAMPTZ DEFAULT now()
```

---

### `wemall_products` (Product Service)

```sql
-- categories  (self-referencing tree)
id               UUID PRIMARY KEY DEFAULT gen_random_uuid()
parent_id        UUID REFERENCES categories(id)
name             TEXT NOT NULL
slug             TEXT NOT NULL UNIQUE
icon_url         TEXT
banner_url       TEXT
level            INTEGER NOT NULL DEFAULT 1   -- 1=top, 2=mid, 3=leaf
attribute_schema JSONB                        -- JSON Schema for category attributes
sort_order       INTEGER DEFAULT 0
is_active        BOOLEAN DEFAULT true
created_at       TIMESTAMPTZ DEFAULT now()
updated_at       TIMESTAMPTZ DEFAULT now()

-- products
id              UUID PRIMARY KEY DEFAULT gen_random_uuid()
seller_id       UUID NOT NULL              -- FK to sellers DB (app-level)
category_id     UUID NOT NULL REFERENCES categories(id)
title           TEXT NOT NULL
slug            TEXT NOT NULL UNIQUE
description     TEXT
attributes      JSONB NOT NULL DEFAULT '{}'  -- e.g. {"ram":"8GB","storage":"256GB"}
brand           TEXT
origin_country  TEXT
status          TEXT DEFAULT 'draft'         -- draft | active | paused | banned
rating          NUMERIC(3,2) DEFAULT 0
review_count    INTEGER DEFAULT 0
sold_count      INTEGER DEFAULT 0
view_count      INTEGER DEFAULT 0
min_price       NUMERIC(12,2)
max_price       NUMERIC(12,2)
created_at      TIMESTAMPTZ DEFAULT now()
updated_at      TIMESTAMPTZ DEFAULT now()
deleted_at      TIMESTAMPTZ

-- product_variants
id              UUID PRIMARY KEY DEFAULT gen_random_uuid()
product_id      UUID NOT NULL REFERENCES products(id)
sku             TEXT NOT NULL UNIQUE
options         JSONB NOT NULL DEFAULT '{}'   -- {"color":"Red","size":"XL"}
price           NUMERIC(12,2) NOT NULL
compare_price   NUMERIC(12,2)
weight_grams    INTEGER
image_url       TEXT
is_default      BOOLEAN DEFAULT false
created_at      TIMESTAMPTZ DEFAULT now()
updated_at      TIMESTAMPTZ DEFAULT now()

-- product_images
id          UUID PRIMARY KEY DEFAULT gen_random_uuid()
product_id  UUID NOT NULL REFERENCES products(id)
url         TEXT NOT NULL
alt_text    TEXT
sort_order  INTEGER DEFAULT 0
is_primary  BOOLEAN DEFAULT false
created_at  TIMESTAMPTZ DEFAULT now()

-- tags  &  product_tags (junction)
-- tags: id UUID, name TEXT, slug TEXT
-- product_tags: product_id UUID, tag_id UUID

-- GIN indexes for JSONB filtering
CREATE INDEX products_attributes_gin ON products USING GIN (attributes);
CREATE INDEX products_category_status ON products (category_id, status);
```

---

### `wemall_inventory` (Inventory Service)

```sql
-- warehouses
id          UUID PRIMARY KEY DEFAULT gen_random_uuid()
seller_id   UUID NOT NULL        -- app-level FK
name        TEXT NOT NULL
city        TEXT
country     TEXT DEFAULT 'CN'
is_default  BOOLEAN DEFAULT false
created_at  TIMESTAMPTZ DEFAULT now()

-- inventory
id              UUID PRIMARY KEY DEFAULT gen_random_uuid()
variant_id      UUID NOT NULL UNIQUE    -- FK to product variants (app-level)
warehouse_id    UUID REFERENCES warehouses(id)
quantity        INTEGER NOT NULL DEFAULT 0
reserved        INTEGER NOT NULL DEFAULT 0   -- held by pending orders
low_stock_alert INTEGER DEFAULT 10
updated_at      TIMESTAMPTZ DEFAULT now()

-- inventory_events  (append-only audit log)
id          UUID PRIMARY KEY DEFAULT gen_random_uuid()
variant_id  UUID NOT NULL
event_type  TEXT NOT NULL    -- reserved | released | sold | restocked | adjusted
delta       INTEGER NOT NULL
reference   TEXT             -- order_id or reason
created_at  TIMESTAMPTZ DEFAULT now()
```

---

### `wemall_orders` (Order Service)

```sql
-- carts
id          UUID PRIMARY KEY DEFAULT gen_random_uuid()
user_id     UUID NOT NULL UNIQUE
created_at  TIMESTAMPTZ DEFAULT now()
updated_at  TIMESTAMPTZ DEFAULT now()

-- cart_items
id          UUID PRIMARY KEY DEFAULT gen_random_uuid()
cart_id     UUID NOT NULL REFERENCES carts(id)
variant_id  UUID NOT NULL         -- app-level FK
quantity    INTEGER NOT NULL DEFAULT 1
added_at    TIMESTAMPTZ DEFAULT now()
UNIQUE(cart_id, variant_id)

-- orders
id                  UUID PRIMARY KEY DEFAULT gen_random_uuid()
order_number        TEXT NOT NULL UNIQUE    -- WM-20260531-00001
user_id             UUID NOT NULL
shipping_address    JSONB NOT NULL          -- snapshot of address at purchase time
status              TEXT DEFAULT 'pending'
-- pending | confirmed | shipped | delivered | cancelled | refunded
subtotal            NUMERIC(12,2) NOT NULL
shipping_fee        NUMERIC(12,2) DEFAULT 0
discount_amount     NUMERIC(12,2) DEFAULT 0
total               NUMERIC(12,2) NOT NULL
coupon_code         TEXT
notes               TEXT
created_at          TIMESTAMPTZ DEFAULT now()
updated_at          TIMESTAMPTZ DEFAULT now()

-- order_items
id          UUID PRIMARY KEY DEFAULT gen_random_uuid()
order_id    UUID NOT NULL REFERENCES orders(id)
variant_id  UUID NOT NULL
seller_id   UUID NOT NULL
quantity    INTEGER NOT NULL
unit_price  NUMERIC(12,2) NOT NULL
snapshot    JSONB NOT NULL     -- {title, sku, options, image_url} frozen at purchase
status      TEXT DEFAULT 'pending'
created_at  TIMESTAMPTZ DEFAULT now()

-- promotions
id              UUID PRIMARY KEY DEFAULT gen_random_uuid()
seller_id       UUID       -- NULL = platform-wide
name            TEXT NOT NULL
type            TEXT NOT NULL  -- percentage | fixed | bogo | flash_sale
value           NUMERIC(10,2) NOT NULL
min_order_value NUMERIC(12,2)
max_discount    NUMERIC(12,2)
starts_at       TIMESTAMPTZ NOT NULL
ends_at         TIMESTAMPTZ NOT NULL
is_active       BOOLEAN DEFAULT true
created_at      TIMESTAMPTZ DEFAULT now()

-- coupons
id              UUID PRIMARY KEY DEFAULT gen_random_uuid()
code            TEXT NOT NULL UNIQUE
promotion_id    UUID REFERENCES promotions(id)
max_uses        INTEGER
used_count      INTEGER DEFAULT 0
per_user_limit  INTEGER DEFAULT 1
expires_at      TIMESTAMPTZ
created_at      TIMESTAMPTZ DEFAULT now()
```

---

### `wemall_payments` (Payment Service)

```sql
-- payments
id              UUID PRIMARY KEY DEFAULT gen_random_uuid()
order_id        UUID NOT NULL UNIQUE
provider        TEXT NOT NULL    -- stripe | paypal | alipay | wechat
provider_ref    TEXT             -- external transaction ID
amount          NUMERIC(12,2) NOT NULL
currency        TEXT DEFAULT 'USD'
status          TEXT DEFAULT 'pending'   -- pending | paid | failed | refunded
metadata        JSONB DEFAULT '{}'       -- provider-specific data
paid_at         TIMESTAMPTZ
created_at      TIMESTAMPTZ DEFAULT now()

-- refunds
id          UUID PRIMARY KEY DEFAULT gen_random_uuid()
payment_id  UUID NOT NULL REFERENCES payments(id)
amount      NUMERIC(12,2) NOT NULL
reason      TEXT
status      TEXT DEFAULT 'pending'
created_at  TIMESTAMPTZ DEFAULT now()

-- shipments
id              UUID PRIMARY KEY DEFAULT gen_random_uuid()
order_id        UUID NOT NULL
carrier         TEXT
tracking_number TEXT
status          TEXT DEFAULT 'preparing'
shipped_at      TIMESTAMPTZ
delivered_at    TIMESTAMPTZ
created_at      TIMESTAMPTZ DEFAULT now()
```

---

## Category Attribute Schemas (JSONB — stored in Product Service)

The `attribute_schema` column on `categories` defines validation rules and frontend filter widgets. No DB migration needed when adding attributes — just update the JSON.

| Category | Required Attributes | Optional Attributes |
|---|---|---|
| **Women's/Men's Clothing** | `size` (XS–3XL), `color` | `material`, `style`, `season` |
| **Shoes & Boots** | `size` (EU/US), `color` | `material`, `gender` |
| **Smartphones** | `brand`, `os`, `storage`, `ram` | `screen_size`, `battery_mah`, `condition` |
| **Skincare** | `skin_type`, `product_type` | `ingredient[]`, `volume_ml`, `spf` |
| **Furniture** | `material`, `color` | `dimensions`, `weight_kg`, `assembly` |
| **Food & Beverages** | `weight_g` OR `volume_ml` | `allergens[]`, `expiry_days`, `organic` |
| **Toys** | `age_min`, `age_max` | `material`, `battery_required` |
| **Fitness Equipment** | `equipment_type` | `weight_capacity_kg`, `dimensions` |
| **Car Accessories** | `car_brand`, `car_model` | `year_from`, `year_to` |
| **Pet Food** | `pet_type`, `weight_g` | `breed`, `life_stage` |
| **Gaming** | `platform`, `region` | `genre`, `player_count` |

---

## gRPC Service Contracts (Protobuf)

### User Service (`proto/user/v1/user.proto`)

```protobuf
service UserService {
  rpc Register(RegisterRequest) returns (AuthResponse);
  rpc Login(LoginRequest) returns (AuthResponse);
  rpc RefreshToken(RefreshTokenRequest) returns (AuthResponse);
  rpc GetUser(GetUserRequest) returns (User);
  rpc GetUserBatch(GetUserBatchRequest) returns (GetUserBatchResponse);  // DataLoader
  rpc UpdateProfile(UpdateProfileRequest) returns (User);
  rpc ListAddresses(ListAddressesRequest) returns (ListAddressesResponse);
  rpc CreateAddress(CreateAddressRequest) returns (Address);
  rpc ValidateToken(ValidateTokenRequest) returns (ValidateTokenResponse);
}
```

### Product Service (`proto/product/v1/product.proto`)

```protobuf
service ProductService {
  rpc ListCategories(ListCategoriesRequest) returns (ListCategoriesResponse);
  rpc GetCategory(GetCategoryRequest) returns (Category);
  rpc ListProducts(ListProductsRequest) returns (ListProductsResponse);
  rpc GetProduct(GetProductRequest) returns (Product);
  rpc GetProductBatch(GetProductBatchRequest) returns (GetProductBatchResponse);
  rpc CreateProduct(CreateProductRequest) returns (Product);
  rpc UpdateProduct(UpdateProductRequest) returns (Product);
  rpc DeleteProduct(DeleteProductRequest) returns (google.protobuf.Empty);
  rpc GetVariantBatch(GetVariantBatchRequest) returns (GetVariantBatchResponse);
}
```

### Inventory Service (`proto/inventory/v1/inventory.proto`)

```protobuf
service InventoryService {
  rpc GetInventory(GetInventoryRequest) returns (Inventory);
  rpc GetInventoryBatch(GetInventoryBatchRequest) returns (GetInventoryBatchResponse);
  rpc ReserveStock(ReserveStockRequest) returns (ReserveStockResponse);
  rpc ReleaseReservation(ReleaseReservationRequest) returns (google.protobuf.Empty);
  rpc ConfirmSale(ConfirmSaleRequest) returns (google.protobuf.Empty);
  rpc UpdateStock(UpdateStockRequest) returns (Inventory);
}
```

### Order Service (`proto/order/v1/order.proto`)

```protobuf
service OrderService {
  rpc GetCart(GetCartRequest) returns (Cart);
  rpc AddToCart(AddToCartRequest) returns (Cart);
  rpc UpdateCartItem(UpdateCartItemRequest) returns (Cart);
  rpc RemoveCartItem(RemoveCartItemRequest) returns (Cart);
  rpc Checkout(CheckoutRequest) returns (Order);
  rpc GetOrder(GetOrderRequest) returns (Order);
  rpc ListOrders(ListOrdersRequest) returns (ListOrdersResponse);
  rpc CancelOrder(CancelOrderRequest) returns (Order);
  rpc UpdateOrderStatus(UpdateOrderStatusRequest) returns (Order);  // internal
}
```

### Payment Service (`proto/payment/v1/payment.proto`)

```protobuf
service PaymentService {
  rpc InitiatePayment(InitiatePaymentRequest) returns (PaymentSession);
  rpc GetPayment(GetPaymentRequest) returns (Payment);
  rpc RequestRefund(RequestRefundRequest) returns (Refund);
}
// Webhook handled via REST: POST /webhooks/payment
```

---

## Event-Driven Flows (NATS JetStream)

Async events decouple services and prevent cascading failures.

### NATS Subjects (Naming Convention: `wemall.<domain>.<event>`)

```
wemall.order.created          → Inventory (reserve stock), Payment (initiate), Notification
wemall.payment.completed      → Order (confirm), Notification (buyer receipt)
wemall.payment.failed         → Order (cancel), Inventory (release reservation)
wemall.order.shipped          → Notification (tracking info to buyer)
wemall.order.delivered        → Seller payout queued
wemall.inventory.low_stock    → Notification (alert seller)
wemall.product.created        → Search (index new product)
wemall.product.updated        → Search (reindex product)
wemall.product.deleted        → Search (remove from index)
wemall.user.registered        → Notification (welcome email)
```

### Example Flow: Checkout

```
1. Client: mutation checkout(...)
2. Gateway → Order Service (gRPC): Checkout()
3. Order Service:
   a. Validates coupon
   b. Publishes wemall.order.created → NATS
   c. Returns Order (status: pending)
4. Inventory Service (subscribes): ReserveStock() for each variant
   • On failure → publishes wemall.order.cancelled
5. Payment Service (subscribes): Creates payment session
   • Returns payment URL to client via Order.paymentUrl
6. Client completes payment → Provider webhook → POST /webhooks/payment
7. Payment Service:
   a. Validates webhook signature
   b. Updates payment record
   c. Publishes wemall.payment.completed
8. Order Service (subscribes): status → confirmed
9. Notification Service (subscribes): sends confirmation email
```

---

## GraphQL Gateway Design

The gateway is **not** a service itself — it's a **thin translation layer** that:
1. Parses the GraphQL query
2. Calls the appropriate gRPC services
3. Uses DataLoaders to batch N+1 calls
4. Merges and returns the result

### GraphQL Schema (same as before, unchanged)

All types, queries, mutations, and subscriptions remain exactly as designed. The gateway resolvers now call gRPC instead of hitting a DB directly.

### DataLoaders in the Gateway

```go
// internal/loaders/loaders.go
type Loaders struct {
    // Batch gRPC calls instead of N+1
    UserByID        *dataloadgen.Loader[string, *userv1.User]
    SellerByID      *dataloadgen.Loader[string, *sellerv1.Seller]
    CategoryByID    *dataloadgen.Loader[string, *productv1.Category]
    ProductByID     *dataloadgen.Loader[string, *productv1.Product]
    VariantsByProductID  *dataloadgen.Loader[string, []*productv1.Variant]
    ImagesByProductID    *dataloadgen.Loader[string, []*productv1.Image]
    InventoryByVariantID *dataloadgen.Loader[string, *inventoryv1.Inventory]
}
// Each batch function calls GetXxxBatch() gRPC which does:
// SELECT ... WHERE id = ANY($1)
```

### Resolver Example

```go
// internal/graph/resolver/query.resolvers.go
func (r *queryResolver) Product(ctx context.Context, id *string, slug *string) (*model.Product, error) {
    resp, err := r.ProductClient.GetProduct(ctx, &productv1.GetProductRequest{
        Id: id, Slug: slug,
    })
    if err != nil { return nil, err }
    return mapProduct(resp), nil
}

// Product.Seller is resolved via DataLoader (1 gRPC call for N products)
func (r *productResolver) Seller(ctx context.Context, obj *model.Product) (*model.Seller, error) {
    return loaders.For(ctx).SellerByID.Load(obj.SellerID)
}
```

---

## Monorepo Project Structure

```
wemall/
├── proto/                          # Protobuf definitions (source of truth)
│   ├── user/v1/user.proto
│   ├── seller/v1/seller.proto
│   ├── product/v1/product.proto
│   ├── inventory/v1/inventory.proto
│   ├── order/v1/order.proto
│   └── payment/v1/payment.proto
│
├── services/
│   ├── api-gateway/                # GraphQL gateway
│   │   ├── cmd/main.go
│   │   ├── internal/
│   │   │   ├── graph/
│   │   │   │   ├── schema/         # *.graphql files
│   │   │   │   ├── generated/      # gqlgen output
│   │   │   │   └── resolver/       # resolver implementations
│   │   │   ├── loaders/            # DataLoader definitions
│   │   │   └── clients/            # gRPC client wrappers
│   │   ├── gqlgen.yml
│   │   └── go.mod
│   │
│   ├── user-service/
│   │   ├── cmd/main.go
│   │   ├── internal/
│   │   │   ├── handler/            # gRPC handler implementations
│   │   │   ├── service/            # business logic
│   │   │   └── repository/         # sqlc queries
│   │   ├── db/
│   │   │   ├── migrations/
│   │   │   ├── queries/
│   │   │   └── sqlc/
│   │   ├── sqlc.yaml
│   │   └── go.mod
│   │
│   ├── seller-service/             # (same structure)
│   ├── product-service/            # (same structure)
│   ├── inventory-service/          # (same structure)
│   ├── order-service/              # (same structure)
│   ├── payment-service/            # (same structure)
│   ├── notification-service/       # (NATS subscriber only, no gRPC)
│   └── search-service/             # (Meilisearch wrapper)
│
├── pkg/                            # Shared Go packages
│   ├── auth/                       # JWT helpers (used by gateway + user-svc)
│   ├── logger/                     # zerolog setup
│   ├── tracing/                    # OpenTelemetry init
│   ├── nats/                       # NATS client + publisher helpers
│   └── errors/                     # Common error types mapped to gRPC codes
│
├── gen/                            # Generated protobuf Go code (do not edit)
│   ├── user/v1/
│   ├── product/v1/
│   └── ...
│
├── scripts/
│   └── seed/
│       └── main.go                 # Seeds all categories + attribute schemas
│
├── docker-compose.yml              # Full local dev stack
├── docker-compose.override.yml     # Dev overrides (live reload)
├── Makefile                        # All commands
├── buf.yaml                        # buf.build proto linting + generation
└── buf.gen.yaml                    # buf code generation config
```

---

## Makefile Commands

```makefile
## Proto
make proto           # buf generate → regenerates all Go proto code in gen/

## Per-service (run from repo root)
make migrate-up svc=user-service
make migrate-down svc=user-service
make generate svc=user-service   # sqlc generate for a service
make seed                         # seed categories into product-service DB

## Gateway
make generate-gql    # gqlgen generate in api-gateway

## Dev
make dev             # docker compose up --build
make dev-gateway     # run only gateway with hot reload (air)
make dev-svc svc=product-service

## Test
make test            # go test ./... across all services
make test-svc svc=order-service

## Build
make build           # builds all service binaries into bin/
```

---

## Docker Compose Topology

```yaml
# docker-compose.yml (abbreviated)
services:
  # Infrastructure
  postgres-users:    { image: postgres:16, port: 5432 }
  postgres-sellers:  { image: postgres:16, port: 5433 }
  postgres-products: { image: postgres:16, port: 5434 }
  postgres-inventory:{ image: postgres:16, port: 5435 }
  postgres-orders:   { image: postgres:16, port: 5436 }
  postgres-payments: { image: postgres:16, port: 5437 }
  redis:             { image: redis:7-alpine }
  nats:              { image: nats:2-alpine, args: ["-js"] }   # JetStream enabled
  meilisearch:       { image: getmeili/meilisearch:latest }
  jaeger:            { image: jaegertracing/all-in-one }       # distributed tracing

  # Services
  user-service:       { build: ./services/user-service, port: 9001 }
  seller-service:     { build: ./services/seller-service, port: 9002 }
  product-service:    { build: ./services/product-service, port: 9003 }
  inventory-service:  { build: ./services/inventory-service, port: 9004 }
  order-service:      { build: ./services/order-service, port: 9005 }
  payment-service:    { build: ./services/payment-service, port: 9006 }
  notification-service: { build: ./services/notification-service }
  search-service:     { build: ./services/search-service, port: 9008 }

  # Gateway
  api-gateway:        { build: ./services/api-gateway, port: 8080 }
```

---

## Proposed Changes — Phased Delivery

### Phase 1 — Foundation

> [!NOTE]
> **Recommended starting point.** Get the skeleton running end-to-end before filling in business logic.

#### [NEW] `proto/*.proto` + `buf.yaml`
All 6 service proto definitions. Running `make proto` generates Go code.

#### [NEW] `services/*/db/migrations/*.sql`
SQL migration files per service — exact schemas above.

#### [NEW] `services/*/db/queries/*.sql` + `sqlc.yaml`
sqlc query files per service.

#### [NEW] `scripts/seed/main.go`
Inserts all 15 top-level categories + ~60 subcategories + attribute schemas into `product-service` DB.

#### [NEW] `docker-compose.yml`
Full infrastructure: 6× PostgreSQL, Redis, NATS, Meilisearch, Jaeger.

#### [NEW] `services/api-gateway/` skeleton
Fiber server + gqlgen with stub resolvers that return mock data. Proves the GraphQL layer works before services are ready.

#### [NEW] `pkg/` shared packages
`auth`, `logger`, `tracing`, `nats`, `errors` — used across all services.

---

### Phase 2 — User + Product Services

#### User Service
- `Register`, `Login`, `RefreshToken`, `ValidateToken` gRPC handlers
- JWT signing + refresh token rotation
- Address CRUD

#### Product Service
- Category tree with attribute schemas
- Product CRUD with JSONB attribute validation
- Variant management

#### Gateway Integration
- Wire up User + Product resolvers (replace stubs)
- DataLoaders for Seller, Category, Product

---

### Phase 3 — Inventory + Order + Cart

#### Inventory Service
- Stock levels, reservation logic (`SELECT FOR UPDATE`)
- `inventory_events` append-only audit log

#### Order Service
- Cart management
- `Checkout` → publishes `wemall.order.created`
- NATS subscriber: listens for `wemall.payment.completed` → updates status

#### Gateway Integration
- Cart mutations, checkout mutation, order queries

---

### Phase 4 — Payment + Real-time

#### Payment Service
- Stripe/PayPal integration
- Webhook handler (`POST /webhooks/payment` — REST, outside GraphQL)
- Publishes `wemall.payment.completed` or `wemall.payment.failed`

#### GraphQL Subscriptions (Gateway)
- `orderStatusChanged` — Redis pub/sub → WebSocket
- `inventoryUpdated` — for live stock on product pages
- `flashSaleUpdated` — for promotion countdowns
- `newOrder` — seller dashboard

---

### Phase 5 — Search + Notifications + Admin

#### Search Service
- Meilisearch indexing from NATS events (`product.created`, `product.updated`)
- Faceted search by category, price, attributes, rating, stock

#### Notification Service
- NATS subscriber for `user.registered`, `order.shipped`, `payment.completed`
- Email via SendGrid / SMTP
- Push notifications via FCM

#### Admin
- `banProduct`, `verifySeller` mutations with `@hasRole(role: ADMIN)` directive

---

## Verification Plan

### Automated Tests

```bash
# Generate everything first
make proto
make generate svc=user-service
make generate svc=product-service
# ... etc
make generate-gql

# Run all tests
make test

# Integration tests (needs docker-compose up)
docker compose up -d
go test ./services/... -tags=integration -v

# Migration smoke test
make migrate-up svc=user-service
make migrate-up svc=product-service
# ... etc

# Seed check
make seed && \
  psql $PRODUCT_DB_URL -c "SELECT count(*) FROM categories;"
# Expected: 75+ rows
```

### End-to-End Flow Verification

1. **GraphQL Playground** at `http://localhost:8080/graphql`
2. `mutation register` → returns JWT
3. `query categories` → returns 15 top-level with subcategories
4. `mutation createProduct` (as seller, with smartphone attributes) → validates against schema
5. `query products(filter: {attributes: {"ram": "8GB"}})` → JSONB GIN index hit
6. `mutation addToCart` → `mutation checkout` → NATS event fires → inventory reserved
7. Mock payment webhook → `orderStatusChanged` subscription fires on client

---

## Summary

This gives you a platform built for real scale:

| Capability | How |
|---|---|
| **Independent deployability** | Each service in its own container/binary |
| **Independent scaling** | Product Service can scale to 100 pods; User Service stays at 2 |
| **Fault isolation** | Search Service down ≠ checkout broken |
| **Type-safe internal API** | gRPC + Protobuf: no guess-work across service boundaries |
| **Type-safe DB layer** | sqlc per service: all queries typed at compile time |
| **Async resilience** | NATS JetStream: messages persist if a subscriber is temporarily down |
| **Developer experience** | Single `make dev` spins up everything; GraphQL Playground for exploration |
| **Observability** | OpenTelemetry traces flow through Gateway → gRPC → DB |
| **Extensibility** | Add a Review Service or Recommendation Service without touching existing code |

---

# Implementation Plan: Media Service AWS IAM & CloudFront Signed URLs Updates

## User Review Required
- **Local Credentials Store**: The new AWS IAM user keys must be configured only in the root `.env` file (which is git-ignored by `.gitignore` to prevent leaking keys). Do not add these keys to any versioned file.
- **AWS Session Token Removal**: Comment out or delete `AWS_SESSION_TOKEN` in the `.env` configuration. If `AWS_SESSION_TOKEN` is present, the AWS SDK will attempt to use it alongside the permanent IAM User keys (`AKIA...`), which will fail.

## Proposed Changes

### Media Service

#### [MODIFY] [media.go](file:///Volumes/Untitled/WeMall/services/media-service/internal/service/media.go)
- Modify the mock-mode detection on startup to run in real S3 mode when `AWS_ACCESS_KEY_ID` is set (even if the bucket is named `wemall-media-raw`).
- Change CloudFront URL signing duration to 100 years (`time.Now().AddDate(100, 0, 0)`) to generate permanently signed URLs for private assets.

#### [MODIFY] [.env](file:///Volumes/Untitled/WeMall/.env)
- Update `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` with the permanent IAM User credentials.
- Comment out or delete `AWS_SESSION_TOKEN`.

## Verification Plan

### Manual Verification
1. **Container Rebuild & Startup**: Rebuild and restart the media service to load the new config.
2. **Review Logs**: Run `docker compose logs -f media-service` and verify that the logs state `S3 client and presigner initialized successfully` instead of the local mock mode fallback warning.
3. **E2E Upload & URL Generation**: Upload an image through the media service, confirm the upload, and verify that the returned variation URLs are signed CloudFront URLs with expiration timestamps set in the far future (~100 years).
