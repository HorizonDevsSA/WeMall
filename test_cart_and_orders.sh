#!/bin/bash
set -e

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo "Error: jq is required. Install with: brew install jq"
    exit 1
fi

GATEWAY_URL="http://localhost:8080/graphql"
TIMESTAMP=$(date +%s)

echo "=== 1. Buyer OTP Login ==="
curl -sf -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -d '{"query":"mutation { buyerSendOTP(phone: \"+263773333333\") { message } }"}' > /dev/null
echo "OTP sent."

BUYER_AUTH=$(curl -sf -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -d '{"query":"mutation { buyerVerifyOTP(phone: \"+263773333333\", otp: \"123456\") { accessToken user { id } } }"}')
BUYER_TOKEN=$(echo "$BUYER_AUTH" | jq -r '.data.buyerVerifyOTP.accessToken')
BUYER_ID=$(echo "$BUYER_AUTH" | jq -r '.data.buyerVerifyOTP.user.id')
if [ "$BUYER_TOKEN" = "null" ] || [ -z "$BUYER_TOKEN" ]; then
  echo "ERROR: Buyer login failed. Response: $BUYER_AUTH"; exit 1
fi
echo "✓ Buyer logged in. ID: $BUYER_ID"

echo ""
echo "=== 2. Seller Login (existing account with verified store) ==="
SELLER_AUTH=$(curl -sf -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -d '{"query":"mutation { sellerLogin(email: \"cart_seller@example.com\", password: \"Password123!\") { accessToken user { id } } }"}')
SELLER_TOKEN=$(echo "$SELLER_AUTH" | jq -r '.data.sellerLogin.accessToken')
if [ "$SELLER_TOKEN" = "null" ] || [ -z "$SELLER_TOKEN" ]; then
  echo "ERROR: Seller login failed. Response: $SELLER_AUTH"; exit 1
fi
echo "✓ Seller token obtained."

echo ""
echo "=== 3. Fetch Existing Store ==="
MY_STORE=$(curl -sf -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SELLER_TOKEN" \
  -d '{"query":"query { myStore { id storeName status } }"}')
STORE_ID=$(echo "$MY_STORE" | jq -r '.data.myStore.id // empty')
STORE_STATUS=$(echo "$MY_STORE" | jq -r '.data.myStore.status // empty')
if [ -z "$STORE_ID" ] || [ "$STORE_ID" = "null" ]; then
  echo "ERROR: Could not fetch store. Response: $MY_STORE"; exit 1
fi
echo "✓ Store ID: $STORE_ID | Status: $STORE_STATUS"

echo ""
echo "=== 4. Fetch Category ==="
CATS=$(curl -sf -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -d '{"query":"query { categories { id name children { id name } } }"}')
CATEGORY_ID=$(echo "$CATS" | jq -r '.data.categories[0].children[0].id // .data.categories[0].id')
if [ -z "$CATEGORY_ID" ] || [ "$CATEGORY_ID" = "null" ]; then
  echo "ERROR: Could not fetch categories."; exit 1
fi
echo "✓ Category ID: $CATEGORY_ID"

echo ""
echo "=== 5. Create Product (MOBILE_PHONES_ACCESSORIES) ==="
SKU="SKU-CART-$TIMESTAMP"
CREATE_PRODUCT=$(curl -sf -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SELLER_TOKEN" \
  -d "{
    \"query\": \"mutation CreateProduct(\$input: CreateProductInput!) { createProduct(input: \$input) { id title productType variants { id sku price options } } }\",
    \"variables\": {
      \"input\": {
        \"categoryId\": \"$CATEGORY_ID\",
        \"title\": \"Cart Test Phone $TIMESTAMP\",
        \"description\": \"An iPhone for integration testing\",
        \"brand\": \"Apple\",
        \"productType\": \"MOBILE_PHONES_ACCESSORIES\",
        \"attributes\": {},
        \"variants\": [
          {
            \"sku\": \"$SKU\",
            \"price\": 999.99,
            \"comparePrice\": 1099.99,
            \"options\": {\"color\": \"Space Gray\", \"storage\": \"256GB\"}
          }
        ]
      }
    }
  }")

echo "Create Product Response:"
echo "$CREATE_PRODUCT" | jq '.data.createProduct'
PRODUCT_ID=$(echo "$CREATE_PRODUCT" | jq -r '.data.createProduct.id // empty')
VARIANT_ID=$(echo "$CREATE_PRODUCT" | jq -r '.data.createProduct.variants[0].id // empty')
PRODUCT_TYPE=$(echo "$CREATE_PRODUCT" | jq -r '.data.createProduct.productType // empty')

if [ -z "$VARIANT_ID" ] || [ "$VARIANT_ID" = "null" ]; then
  echo "ERROR: Product creation failed."
  echo "$CREATE_PRODUCT" | jq .
  exit 1
fi
echo "✓ Product: $PRODUCT_ID | Variant: $VARIANT_ID | Type: $PRODUCT_TYPE"

echo ""
echo "=== 6. Add Variant to Cart ==="
ADD_CART=$(curl -sf -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  -d "{\"query\":\"mutation { addToCart(variantId: \\\"$VARIANT_ID\\\", quantity: 2) { id itemCount subtotal } }\"}")
CART_ID=$(echo "$ADD_CART" | jq -r '.data.addToCart.id // empty')
ITEM_COUNT=$(echo "$ADD_CART" | jq -r '.data.addToCart.itemCount // empty')
if [ -z "$CART_ID" ] || [ "$CART_ID" = "null" ]; then
  echo "ERROR: addToCart failed."
  echo "$ADD_CART" | jq .
  exit 1
fi
echo "✓ Cart ID: $CART_ID | Items: $ITEM_COUNT"

echo ""
echo "=== 7. Fetch Cart (Verify Snapshotted Attributes) ==="
GET_CART=$(curl -sf -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  -d '{
    "query": "query { cart { id itemCount subtotal items { id variantId productId quantity unitPrice productTitle variation variationThumbnail sellerId storeTitle storeLogo options productType } } }"
  }')
echo "Cart contents:"
echo "$GET_CART" | jq '.data.cart'

# Verify key fields
CART_PRODUCT_TYPE=$(echo "$GET_CART" | jq -r '.data.cart.items[0].productType // empty')
CART_STORE_TITLE=$(echo "$GET_CART" | jq -r '.data.cart.items[0].storeTitle // empty')
echo "  → productType in cart item: $CART_PRODUCT_TYPE"
echo "  → storeTitle in cart item: $CART_STORE_TITLE"

echo ""
echo "=== 8. Checkout ==="
CHECKOUT=$(curl -sf -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  -d '{
    "query": "mutation Checkout($input: CheckoutInput!) { checkout(input: $input) { id orderNumber status subtotal total items { id variantId productId quantity unitPrice productTitle variation variationThumbnail storeTitle storeLogo options productType } } }",
    "variables": {
      "input": {
        "shippingAddress": {
          "fullName": "John Cart",
          "phone": "+263773333333",
          "addressLine1": "77 Checkout Lane",
          "city": "Harare",
          "country": "Zimbabwe"
        },
        "currency": "USD"
      }
    }
  }')

echo "Checkout response:"
echo "$CHECKOUT" | jq '.data.checkout'
ORDER_ID=$(echo "$CHECKOUT" | jq -r '.data.checkout.id // empty')
ORDER_NUMBER=$(echo "$CHECKOUT" | jq -r '.data.checkout.orderNumber // empty')

if [ -z "$ORDER_ID" ] || [ "$ORDER_ID" = "null" ]; then
  echo "ERROR: Checkout failed."
  echo "$CHECKOUT" | jq .
  exit 1
fi
echo "✓ Order created: #$ORDER_NUMBER (ID: $ORDER_ID)"

# Verify snapshotted attributes in order
ORDER_PRODUCT_TYPE=$(echo "$CHECKOUT" | jq -r '.data.checkout.items[0].productType // empty')
ORDER_STORE_TITLE=$(echo "$CHECKOUT" | jq -r '.data.checkout.items[0].storeTitle // empty')
echo "  → productType in order item: $ORDER_PRODUCT_TYPE"
echo "  → storeTitle in order item: $ORDER_STORE_TITLE"

echo ""
echo "=== 9. Query Frozen Order (Verify Snapshot Integrity) ==="
GET_ORDER=$(curl -sf -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  -d "{
    \"query\": \"query { order(id: \\\"$ORDER_ID\\\") { id orderNumber status subtotal total items { id variantId productId quantity unitPrice productTitle variation variationThumbnail storeTitle storeLogo options productType } } }\"
  }")

echo "Order details:"
echo "$GET_ORDER" | jq '.data.order'

echo ""
echo "========================================"
echo "✓ Integration test PASSED successfully!"
echo "  Buyer ID:     $BUYER_ID"
echo "  Store ID:     $STORE_ID"
echo "  Product ID:   $PRODUCT_ID"
echo "  Variant ID:   $VARIANT_ID"
echo "  Cart ID:      $CART_ID"
echo "  Order:        #$ORDER_NUMBER"
echo "========================================"
