# Cart & Orders API — Integration Guide

> **Environment:** Local Docker (`http://localhost:8080/graphql`)  
> **Date recorded:** 2026-06-05  
> **Protocol:** GraphQL over HTTP POST  
> **Auth:** JWT Bearer token (15-minute access token)

This document walks through the complete buyer journey — from phone OTP login, through product discovery, cart management, and checkout — using real `curl` commands with live captured responses.

---

## Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [Prerequisites](#2-prerequisites)
3. [Step 1 — Buyer: Send OTP](#step-1--buyer-send-otp)
4. [Step 2 — Buyer: Verify OTP & Get Token](#step-2--buyer-verify-otp--get-token)
5. [Step 3 — Seller: Login](#step-3--seller-login)
6. [Step 4 — Seller: Get My Store](#step-4--seller-get-my-store)
7. [Step 5 — Browse: List Categories](#step-5--browse-list-categories)
8. [Step 6 — Seller: Create Product with Variants](#step-6--seller-create-product-with-variants)
9. [Step 7 — Buyer: Add to Cart](#step-7--buyer-add-to-cart)
10. [Step 8 — Buyer: View Full Cart](#step-8--buyer-view-full-cart)
11. [Step 9 — Buyer: Add Second Variant](#step-9--buyer-add-second-variant)
12. [Step 10 — Buyer: Checkout](#step-10--buyer-checkout)
13. [Step 11 — Buyer: Get Order by ID](#step-11--buyer-get-order-by-id)
14. [Step 12 — Buyer: List All Orders](#step-12--buyer-list-all-orders)
15. [Data Model Reference](#data-model-reference)
16. [ProductType Enum Values](#producttype-enum-values)
17. [Error Handling](#error-handling)

---

## 1. Architecture Overview

```
Client (curl / Mobile App)
        │
        ▼
  API Gateway :8080           ← GraphQL (gqlgen)
  ┌─────────────┐
  │  Resolvers  │
  └──────┬──────┘
         │  gRPC
    ┌────┴─────────────────────────────────┐
    │                                      │
    ▼                                      ▼
User Service :9001          Product Service :9003
(OTP, JWT)                  (Products, Variants, Categories)
                                      │
                                      ▼
                            Seller Service :9002
                            (Stores, Status)
                                      │
                                      ▼
                            Order Service :9005
                            (Cart, Checkout, Orders)
```

All services run in Docker. The API Gateway is the **single entry point** for all clients.

---

## 2. Prerequisites

```bash
# Start all services
docker compose up -d

# Verify services are running
docker ps --format "table {{.Names}}\t{{.Status}}"

# Requires jq for JSON parsing
brew install jq
```

Set base URL:
```bash
export GW="http://localhost:8080/graphql"
```

---

## Step 1 — Buyer: Send OTP

The buyer provides their phone number. An OTP is generated, stored (hashed with bcrypt), and sent via SMS. In `development` mode, SMS is mocked and the master OTP `123456` always works.

### Request

```bash
curl -s -X POST $GW \
  -H "Content-Type: application/json" \
  -d '{
    "query": "mutation { buyerSendOTP(phone: \"+263773333333\") { message requestId } }"
  }' | jq .
```

### Live Response

```json
{
  "data": {
    "buyerSendOTP": {
      "message": "OTP sent successfully",
      "requestId": "mock-request-id-123456"
    }
  }
}
```

> **Note:** In production, `requestId` is the Africa's Talking SMS message ID. In development/mock mode, it returns a fixed placeholder.

---

## Step 2 — Buyer: Verify OTP & Get Token

Submit the 6-digit OTP to receive a JWT access token and refresh token. The user is created automatically if they don't exist yet (upsert on phone number).

### Request

```bash
curl -s -X POST $GW \
  -H "Content-Type: application/json" \
  -d '{
    "query": "mutation { buyerVerifyOTP(phone: \"+263773333333\", otp: \"123456\") { accessToken refreshToken user { id fullName role } } }"
  }' | jq .
```

### Live Response

```json
{
  "data": {
    "buyerVerifyOTP": {
      "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiMDA5OWQyNjYtOThjNC00MjgxLTkyYzItZWMyODk1YjFmZDRjIiwicm9sZSI6ImJ1eWVyIiwiZXhwIjoxNzgwNjY1MzY1LCJpYXQiOjE3ODA2NjQ0NjV9.OLWx5a1B7kSvXo_bG1h8tJQb1rcbpXT8J161BuyqDn8",
      "refreshToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiMDA5OWQyNjYtOThjNC00MjgxLTkyYzItZWMyODk1YjFmZDRjIiwicm9sZSI6ImJ1eWVyIiwiZXhwIjoxNzgxMjY5MjY1LCJpYXQiOjE3ODA2NjQ0NjV9.G8zNawNTlBpxMxEUU8txifB25gxKahkDcpY0opeDeLGA",
      "user": {
        "id": "0099d266-98c4-4281-92c2-ec2895b1fd4c",
        "fullName": "User",
        "role": "BUYER"
      }
    }
  }
}
```

### Save Token

```bash
export BUYER_TOKEN="<accessToken from above>"
export BUYER_ID="0099d266-98c4-4281-92c2-ec2895b1fd4c"
```

> **Token TTL:** Access token = 15 minutes. Refresh token = 7 days.  
> **Master OTP:** In `ENVIRONMENT=development`, the OTP `123456` always passes regardless of the hashed value stored.

---

## Step 3 — Seller: Login

Sellers authenticate with email and password (registered via `sellerRegister` mutation). For this walkthrough, a pre-existing verified seller is used.

### Request

```bash
curl -s -X POST $GW \
  -H "Content-Type: application/json" \
  -d '{
    "query": "mutation { sellerLogin(email: \"cart_seller@example.com\", password: \"Password123!\") { accessToken user { id fullName role } } }"
  }' | jq .
```

### Live Response

```json
{
  "data": {
    "sellerLogin": {
      "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiYjUwZDAzNzAtNDgwOS00MWExLWE0OGMtZWEzZGI0ODA1YTBjIiwicm9sZSI6InNlbGxlciIsImV4cCI6MTc4MDY2NTM4MywiaWF0IjoxNzgwNjY0NDgzfQ.QnLYSRf-5ujlAAGdtaNlo77YkvtApk9TzdCdwjsp8DI",
      "user": {
        "id": "b50d0370-4809-41a1-a48c-ea3db4805a0c",
        "fullName": "Cart Owner",
        "role": "SELLER"
      }
    }
  }
}
```

### Save Token

```bash
export SELLER_TOKEN="<accessToken from above>"
```

---

## Step 4 — Seller: Get My Store

Every seller has one store. The store must be in `VERIFIED` status before products can be listed publicly.

### Request

```bash
curl -s -X POST $GW \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SELLER_TOKEN" \
  -d '{
    "query": "query { myStore { id storeName storeSlug status isVerified logoUrl } }"
  }' | jq .
```

### Live Response

```json
{
  "data": {
    "myStore": {
      "id": "117f5472-ac34-44ab-8e5d-9687dfdfc443",
      "storeName": "Cart Test Store",
      "storeSlug": "cart-test-store",
      "status": "VERIFIED",
      "isVerified": true,
      "logoUrl": "https://wemall.co.zw/assets/store_logo.png"
    }
  }
}
```

### Save Store ID

```bash
export STORE_ID="117f5472-ac34-44ab-8e5d-9687dfdfc443"
```

---

## Step 5 — Browse: List Categories

Categories are public (no auth required). They are hierarchical — top-level categories contain child sub-categories. Products are assigned to leaf (child) categories.

### Request

```bash
curl -s -X POST $GW \
  -H "Content-Type: application/json" \
  -d '{
    "query": "query { categories { id name slug children { id name slug } } }"
  }' | jq '.data.categories[:2]'
```

### Live Response (first 2 categories)

```json
[
  {
    "id": "5d7d3a1d-df46-47e0-98d7-4b2c5d5e4a6d",
    "name": "Pet Supplies",
    "slug": "pet-supplies",
    "children": [
      { "id": "17670f37-2637-45ac-8c51-dc3e9b3d69b7", "name": "Dog Food",  "slug": "dog-food-d5f6" },
      { "id": "3525ec7f-4649-4bbf-af22-104891c7a58f", "name": "Cat Food",  "slug": "cat-food-a227" },
      { "id": "2f1b8d0c-0be9-423a-86ff-8b28a9d95ce3", "name": "Pet Toys",  "slug": "pet-toys-320d" },
      { "id": "a15c7234-1b93-4d1e-8cff-d4d329146f60", "name": "Pet Beds",  "slug": "pet-beds-2b06" }
    ]
  },
  {
    "id": "c2801f85-9318-4fbd-860b-dda3db380ee3",
    "name": "Electronics & Technology",
    "slug": "electronics-and-technology",
    "children": [
      { "id": "564bb67f-a84d-4238-8d9a-f89351055c6e", "name": "Smartphones",           "slug": "smartphones-8601" },
      { "id": "8d634072-2d18-4a12-9b09-b40caf10e2d6", "name": "Phone Accessories",     "slug": "phone-accessories-dbc1" },
      { "id": "f5249ddb-52d7-4250-82d7-3ccf9207d1cb", "name": "Chargers & Cables",     "slug": "chargers-and-cables-b97d" },
      { "id": "00260adc-eeac-4bf1-8c22-4b35237f73b8", "name": "Earbuds & Headphones",  "slug": "earbuds-and-headphones-c5d3" },
      { "id": "cb589805-8bb6-473d-820c-f775e4e8cd1a", "name": "Computer Accessories",  "slug": "computer-accessories-4f9b" },
      { "id": "476b969a-5d28-4dcf-8301-0069f76cb66b", "name": "Gaming Accessories",    "slug": "gaming-accessories-6baf" },
      { "id": "690f684c-c86b-4e96-a919-09e9d8fbe022", "name": "Smart Home Devices",    "slug": "smart-home-devices-f67d" },
      { "id": "22fc7f31-0cef-46c1-bf08-f1662002a352", "name": "Cameras & Photography", "slug": "cameras-and-photography-1fba" }
    ]
  }
]
```

### Save Category ID

```bash
# Using "Smartphones" sub-category
export CATEGORY_ID="564bb67f-a84d-4238-8d9a-f89351055c6e"
```

---

## Step 6 — Seller: Create Product with Variants

Products support multiple **variants** (different color/storage/size combinations). Each variant has its own SKU, price, and `options` JSON object. The `productType` enum tells the frontend how to render the product and what attributes matter.

> **Important:** `attributes`, `options` are `JSON!` scalar types. Always pass them via GraphQL **variables**, never inline in the query string, to avoid parse errors.

### Request

```bash
curl -s -X POST $GW \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SELLER_TOKEN" \
  -d '{
    "query": "mutation CreateProduct($input: CreateProductInput!) { createProduct(input: $input) { id title productType status variants { id sku price comparePrice options } } }",
    "variables": {
      "input": {
        "categoryId": "564bb67f-a84d-4238-8d9a-f89351055c6e",
        "title": "iPhone 15 Pro Max",
        "description": "Apple flagship smartphone with A17 Pro chip and titanium design.",
        "brand": "Apple",
        "productType": "MOBILE_PHONES_ACCESSORIES",
        "attributes": { "network": "5G", "os": "iOS 17" },
        "variants": [
          {
            "sku": "IPHONE15PM-256-BLK",
            "price": 1199.99,
            "comparePrice": 1299.99,
            "options": { "color": "Black Titanium", "storage": "256GB" }
          },
          {
            "sku": "IPHONE15PM-512-WHT",
            "price": 1399.99,
            "comparePrice": 1499.99,
            "options": { "color": "White Titanium", "storage": "512GB" }
          }
        ]
      }
    }
  }' | jq .
```

### Live Response

```json
{
  "data": {
    "createProduct": {
      "id": "bc316519-c607-44f2-8405-d110b599c084",
      "title": "iPhone 15 Pro Max",
      "productType": "MOBILE_PHONES_ACCESSORIES",
      "status": "ACTIVE",
      "variants": [
        {
          "id": "2eef484a-ea4b-4d5f-b762-401a55bd1f97",
          "sku": "IPHONE15PM-256-BLK",
          "price": 1199.99,
          "comparePrice": 1299.99,
          "options": { "color": "Black Titanium", "storage": "256GB" }
        },
        {
          "id": "865de37d-68cd-4829-9fbe-3fd86abecc2c",
          "sku": "IPHONE15PM-512-WHT",
          "price": 1399.99,
          "comparePrice": 1499.99,
          "options": { "color": "White Titanium", "storage": "512GB" }
        }
      ]
    }
  }
}
```

### Save IDs

```bash
export PRODUCT_ID="bc316519-c607-44f2-8405-d110b599c084"
export VARIANT_1="2eef484a-ea4b-4d5f-b762-401a55bd1f97"   # Black Titanium 256GB
export VARIANT_2="865de37d-68cd-4829-9fbe-3fd86abecc2c"   # White Titanium 512GB
```

---

## Step 7 — Buyer: Add to Cart

The buyer adds a specific variant to their cart. The cart is created automatically on the first add. Adding the same variant again **increments** the quantity (upsert).

### Request

```bash
curl -s -X POST $GW \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  -d "{
    \"query\": \"mutation { addToCart(variantId: \\\"$VARIANT_1\\\", quantity: 1) { id itemCount subtotal } }\"
  }" | jq .
```

### Live Response

```json
{
  "data": {
    "addToCart": {
      "id": "427005c0-c1be-4cd7-aa2c-afe7265e4c3b",
      "itemCount": 1,
      "subtotal": 1199.99
    }
  }
}
```

### Save Cart ID

```bash
export CART_ID="427005c0-c1be-4cd7-aa2c-afe7265e4c3b"
```

---

## Step 8 — Buyer: View Full Cart

Fetches the cart with all enriched item details — product title, variation label, store info, `options` JSON, and `productType` enum — assembled live from the product and seller services.

### Request

```bash
curl -s -X POST $GW \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  -d '{
    "query": "query { cart { id itemCount subtotal items { id variantId productId quantity unitPrice productTitle variation variationThumbnail sellerId storeTitle storeLogo options productType } } }"
  }' | jq .
```

### Live Response

```json
{
  "data": {
    "cart": {
      "id": "427005c0-c1be-4cd7-aa2c-afe7265e4c3b",
      "itemCount": 1,
      "subtotal": 1199.99,
      "items": [
        {
          "id": "f50e226f-0e9a-4ffb-a159-58d637e98da7",
          "variantId": "2eef484a-ea4b-4d5f-b762-401a55bd1f97",
          "productId": "bc316519-c607-44f2-8405-d110b599c084",
          "quantity": 1,
          "unitPrice": 1199.99,
          "productTitle": "iPhone 15 Pro Max",
          "variation": "Black Titanium / storage: 256GB",
          "variationThumbnail": "",
          "sellerId": "117f5472-ac34-44ab-8e5d-9687dfdfc443",
          "storeTitle": "Cart Test Store",
          "storeLogo": "https://wemall.co.zw/assets/store_logo.png",
          "options": {
            "color": "Black Titanium",
            "storage": "256GB"
          },
          "productType": "MOBILE_PHONES_ACCESSORIES"
        }
      ]
    }
  }
}
```

> **Cart enrichment:** The order service calls `product-service` and `seller-service` at runtime to hydrate `productTitle`, `storeTitle`, `storeLogo`, `options`, and `productType`. This data is **not stored in the cart table** — only `variant_id`, `product_id`, `quantity`, and `unit_price` are persisted.

---

## Step 9 — Buyer: Add Second Variant

Adding the second variant (White Titanium 512GB) to the existing cart.

### Request

```bash
curl -s -X POST $GW \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  -d "{
    \"query\": \"mutation { addToCart(variantId: \\\"$VARIANT_2\\\", quantity: 1) { id itemCount subtotal } }\"
  }" | jq .
```

### Live Response

```json
{
  "data": {
    "addToCart": {
      "id": "427005c0-c1be-4cd7-aa2c-afe7265e4c3b",
      "itemCount": 2,
      "subtotal": 2599.98
    }
  }
}
```

---

## Step 10 — Buyer: Checkout

Converts the cart into a confirmed order. The cart is cleared after successful checkout. Each `order_item` stores a **frozen snapshot** of all product/seller attributes at the time of purchase — these never change even if the seller later edits the product.

### Request

```bash
curl -s -X POST $GW \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  -d '{
    "query": "mutation Checkout($input: CheckoutInput!) { checkout(input: $input) { id orderNumber status subtotal shippingFee total items { id variantId productId quantity unitPrice productTitle variation variationThumbnail storeTitle storeLogo options productType } } }",
    "variables": {
      "input": {
        "shippingAddress": {
          "fullName": "Tendai Moyo",
          "phone": "+263773333333",
          "addressLine1": "123 Samora Machel Ave",
          "city": "Harare",
          "country": "Zimbabwe"
        },
        "currency": "USD"
      }
    }
  }' | jq .
```

### Live Response

```json
{
  "data": {
    "checkout": {
      "id": "a35c3574-f16a-4770-84e6-3b7948e1b15d",
      "orderNumber": "WM-1780664885-cae5e5",
      "status": "PENDING",
      "subtotal": 2599.98,
      "shippingFee": 5,
      "total": 2604.98,
      "items": [
        {
          "id": "5f0e410b-daaa-4ac6-ad0f-d1a831165c1a",
          "variantId": "2eef484a-ea4b-4d5f-b762-401a55bd1f97",
          "productId": "bc316519-c607-44f2-8405-d110b599c084",
          "quantity": 1,
          "unitPrice": 1199.99,
          "productTitle": "iPhone 15 Pro Max",
          "variation": "Black Titanium / storage: 256GB",
          "variationThumbnail": "",
          "storeTitle": "Cart Test Store",
          "storeLogo": "https://wemall.co.zw/assets/store_logo.png",
          "options": { "color": "Black Titanium", "storage": "256GB" },
          "productType": "MOBILE_PHONES_ACCESSORIES"
        },
        {
          "id": "d08ade1a-631d-48f5-b476-be2c65050dc8",
          "variantId": "865de37d-68cd-4829-9fbe-3fd86abecc2c",
          "productId": "bc316519-c607-44f2-8405-d110b599c084",
          "quantity": 1,
          "unitPrice": 1399.99,
          "productTitle": "iPhone 15 Pro Max",
          "variation": "White Titanium / storage: 512GB",
          "variationThumbnail": "",
          "storeTitle": "Cart Test Store",
          "storeLogo": "https://wemall.co.zw/assets/store_logo.png",
          "options": { "color": "White Titanium", "storage": "512GB" },
          "productType": "MOBILE_PHONES_ACCESSORIES"
        }
      ]
    }
  }
}
```

### Save Order ID

```bash
export ORDER_ID="a35c3574-f16a-4770-84e6-3b7948e1b15d"
export ORDER_NUM="WM-1780664885-cae5e5"
```

> **Order number format:** `WM-{unix_timestamp}-{6-char-hex}` — globally unique and human-readable.  
> **Snapshot integrity:** The `productTitle`, `storeTitle`, `storeLogo`, `options`, and `productType` on each order item are written once at checkout time and are immutable.

---

## Step 11 — Buyer: Get Order by ID

Fetch the full frozen order with all item details. This is safe to call any time after checkout — the data is permanently snapshotted.

### Request

```bash
curl -s -X POST $GW \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  -d "{
    \"query\": \"query { order(id: \\\"$ORDER_ID\\\") { id orderNumber status subtotal shippingFee total createdAt items { id variantId productId quantity unitPrice productTitle variation variationThumbnail storeTitle storeLogo options productType } } }\"
  }" | jq .
```

### Live Response

```json
{
  "data": {
    "order": {
      "id": "a35c3574-f16a-4770-84e6-3b7948e1b15d",
      "orderNumber": "WM-1780664885-cae5e5",
      "status": "PENDING",
      "subtotal": 2599.98,
      "shippingFee": 5,
      "total": 2604.98,
      "createdAt": "2026-06-05T13:08:05.222238Z",
      "items": [
        {
          "id": "5f0e410b-daaa-4ac6-ad0f-d1a831165c1a",
          "variantId": "2eef484a-ea4b-4d5f-b762-401a55bd1f97",
          "productId": "bc316519-c607-44f2-8405-d110b599c084",
          "quantity": 1,
          "unitPrice": 1199.99,
          "productTitle": "iPhone 15 Pro Max",
          "variation": "Black Titanium / storage: 256GB",
          "variationThumbnail": "",
          "storeTitle": "Cart Test Store",
          "storeLogo": "https://wemall.co.zw/assets/store_logo.png",
          "options": { "color": "Black Titanium", "storage": "256GB" },
          "productType": "MOBILE_PHONES_ACCESSORIES"
        },
        {
          "id": "d08ade1a-631d-48f5-b476-be2c65050dc8",
          "variantId": "865de37d-68cd-4829-9fbe-3fd86abecc2c",
          "productId": "bc316519-c607-44f2-8405-d110b599c084",
          "quantity": 1,
          "unitPrice": 1399.99,
          "productTitle": "iPhone 15 Pro Max",
          "variation": "White Titanium / storage: 512GB",
          "variationThumbnail": "",
          "storeTitle": "Cart Test Store",
          "storeLogo": "https://wemall.co.zw/assets/store_logo.png",
          "options": { "color": "White Titanium", "storage": "512GB" },
          "productType": "MOBILE_PHONES_ACCESSORIES"
        }
      ]
    }
  }
}
```

---

## Step 12 — Buyer: List All Orders

Paginated list of the buyer's order history, newest first.

### Request

```bash
curl -s -X POST $GW \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  -d '{
    "query": "query { orders(pageSize: 5) { orders { id orderNumber status total createdAt } total } }"
  }' | jq .
```

### Live Response

```json
{
  "data": {
    "orders": {
      "orders": [
        {
          "id": "a35c3574-f16a-4770-84e6-3b7948e1b15d",
          "orderNumber": "WM-1780664885-cae5e5",
          "status": "PENDING",
          "total": 2604.98,
          "createdAt": "2026-06-05T13:08:05.222238Z"
        },
        {
          "id": "53453d86-40cb-4f8f-8afd-a8252b889391",
          "orderNumber": "WM-1780663622-57e00c",
          "status": "PENDING",
          "total": 4004.96,
          "createdAt": "2026-06-05T12:47:02.857635Z"
        }
      ],
      "total": 2
    }
  }
}
```

---

## Data Model Reference

### CartItem Fields

| Field | Type | Source | Description |
|-------|------|--------|-------------|
| `id` | `ID` | DB | Cart item UUID |
| `variantId` | `ID` | DB | Reference to product variant |
| `productId` | `ID` | DB | Reference to parent product |
| `quantity` | `Int` | DB | Number of units |
| `unitPrice` | `Float` | DB (locked at add time) | Price per unit at time of adding |
| `productTitle` | `String` | Product Service (live) | Product display name |
| `variation` | `String` | Product Service (live) | Human-readable options e.g. `"Black / 256GB"` |
| `variationThumbnail` | `String` | Product Service (live) | Variant-specific image URL |
| `sellerId` | `ID` | Product Service (live) | Store UUID |
| `storeTitle` | `String` | Seller Service (live) | Store display name |
| `storeLogo` | `String` | Seller Service (live) | Store logo URL |
| `options` | `JSON` | Product Service (live) | Raw options map e.g. `{"color":"Black","storage":"256GB"}` |
| `productType` | `ProductType` | Product Service (live) | Enum e.g. `MOBILE_PHONES_ACCESSORIES` |

### OrderItem Fields

| Field | Type | Source | Description |
|-------|------|--------|-------------|
| `id` | `ID` | DB | Order item UUID |
| `variantId` | `ID` | Snapshot | Frozen variant reference |
| `productId` | `ID` | Snapshot | Frozen product reference |
| `quantity` | `Int` | Snapshot | Units ordered |
| `unitPrice` | `Float` | Snapshot | Price locked at checkout |
| `productTitle` | `String` | **Snapshot** | Frozen at checkout — never changes |
| `variation` | `String` | **Snapshot** | Frozen at checkout |
| `variationThumbnail` | `String` | **Snapshot** | Frozen at checkout |
| `storeTitle` | `String` | **Snapshot** | Frozen at checkout |
| `storeLogo` | `String` | **Snapshot** | Frozen at checkout |
| `options` | `JSON` | **Snapshot** | Frozen at checkout |
| `productType` | `ProductType` | **Snapshot** | Frozen at checkout |

> **Key difference:** Cart items are enriched **live** at query time. Order items are stored as a **frozen JSON snapshot** (`snapshot` column in `order_items` table), making them immutable historical records.

---

## ProductType Enum Values

Used in `createProduct` input and returned in `CartItem` / `OrderItem`.

| Enum Value | Description |
|---|---|
| `ELECTRONICS` | General electronics (TVs, laptops, etc.) |
| `MOBILE_PHONES_ACCESSORIES` | Phones, cases, screen protectors |
| `FASHION` | Clothing, shoes, bags |
| `HOME_FURNITURE` | Furniture, décor, bedding |
| `BEAUTY_HEALTH` | Cosmetics, skincare, supplements |
| `APPLIANCES` | Fridges, washing machines, microwaves |
| `AUTOMOTIVE` | Car parts, tyres, accessories |
| `HARDWARE_CONSTRUCTION` | Tools, cement, building materials |
| `AGRICULTURE` | Seeds, fertilizer, farming equipment |
| `SPORTS_OUTDOORS` | Gym equipment, sportswear, camping |
| `BABY_KIDS` | Toys, prams, baby clothing |
| `OFFICE_SUPPLIES` | Stationery, printers, chairs |
| `BOOKS_EDUCATION` | Books, courses, educational materials |
| `PET_SUPPLIES` | Pet food, toys, accessories |
| `DIGITAL_PRODUCTS` | Software, e-books, game codes |
| `SERVICES` | Freelance, repairs, delivery |
| `LIQUIDS` | Oils, chemicals, cleaning products |
| `BEVERAGES` | Drinks, juices, water |

---

## Error Handling

All errors follow the GraphQL error format:

```json
{
  "errors": [
    {
      "message": "human-readable error message",
      "extensions": {
        "code": "UNAUTHENTICATED",
        "grpc_code": "Unauthenticated"
      }
    }
  ],
  "data": null
}
```

### Common Error Codes

| `extensions.code` | Cause | Fix |
|---|---|---|
| `UNAUTHENTICATED` | Missing or expired Bearer token | Re-authenticate and get a new token |
| `NOT_FOUND` | Resource (product, order, store) does not exist | Check the ID is correct |
| `CONFLICT` | Duplicate resource (e.g. store already exists) | Use the existing resource |
| `INVALID_ARGUMENT` | Malformed input (bad UUID, missing required field) | Validate input before sending |
| `INTERNAL_SERVER_ERROR` | Upstream service failure | Check service logs with `docker logs wemall-<service>-1` |

### Debugging Commands

```bash
# View API Gateway GraphQL errors
docker logs wemall-api-gateway-1 --tail 20

# View order service errors (cart/checkout)
docker logs wemall-order-service-1 --tail 20

# View product service errors
docker logs wemall-product-service-1 --tail 20

# View user service errors (auth/OTP)
docker logs wemall-user-service-1 --tail 20

# Run full integration test
bash test_cart_and_orders.sh
```

---

*Generated from live integration test run on 2026-06-05. All UUIDs and tokens are from a local development environment.*
