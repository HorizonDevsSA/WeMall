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
echo "✓ Buyer logged in. ID: $BUYER_ID"

echo ""
echo "=== 2. Seller Login ==="
SELLER_AUTH=$(curl -sf -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -d '{"query":"mutation { sellerLogin(email: \"cart_seller@example.com\", password: \"Password123!\") { accessToken user { id } } }"}')
SELLER_TOKEN=$(echo "$SELLER_AUTH" | jq -r '.data.sellerLogin.accessToken')
echo "✓ Seller token obtained."

echo ""
echo "=== 3. Fetch Store ==="
MY_STORE=$(curl -sf -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SELLER_TOKEN" \
  -d '{"query":"query { myStore { id storeName status } }"}')
STORE_ID=$(echo "$MY_STORE" | jq -r '.data.myStore.id')
echo "✓ Store ID: $STORE_ID"

echo ""
echo "=== 4. Fetch Category ==="
CATS=$(curl -sf -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -d '{"query":"query { categories { id name children { id name } } }"}')
CATEGORY_ID=$(echo "$CATS" | jq -r '.data.categories[0].children[0].id // .data.categories[0].id')
echo "✓ Category ID: $CATEGORY_ID"

echo ""
echo "=== 5. Create Product ==="
SKU="SKU-REVIEW-$TIMESTAMP"
CREATE_PRODUCT=$(curl -sf -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SELLER_TOKEN" \
  -d "{
    \"query\": \"mutation CreateProduct(\$input: CreateProductInput!) { createProduct(input: \$input) { id title variants { id sku price } } }\",
    \"variables\": {
      \"input\": {
        \"categoryId\": \"$CATEGORY_ID\",
        \"title\": \"Review Test Product $TIMESTAMP\",
        \"description\": \"Product for testing review-service\",
        \"brand\": \"WeMall\",
        \"productType\": \"ELECTRONICS\",
        \"attributes\": {},
        \"variants\": [
          {
            \"sku\": \"$SKU\",
            \"price\": 19.99,
            \"options\": {\"color\": \"Black\"}
          }
        ]
      }
    }
  }")
PRODUCT_ID=$(echo "$CREATE_PRODUCT" | jq -r '.data.createProduct.id')
VARIANT_ID=$(echo "$CREATE_PRODUCT" | jq -r '.data.createProduct.variants[0].id')
echo "✓ Product ID: $PRODUCT_ID | Variant ID: $VARIANT_ID"

echo ""
echo "=== 6. Add to Cart ==="
curl -sf -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  -d "{\"query\":\"mutation { addToCart(variantId: \\\"$VARIANT_ID\\\", quantity: 1) { id } }\"}" > /dev/null
echo "✓ Added to cart."

echo ""
echo "=== 7. Checkout ==="
CHECKOUT=$(curl -sf -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  -d '{
    "query": "mutation Checkout($input: CheckoutInput!) { checkout(input: $input) { id orderNumber } }",
    "variables": {
      "input": {
        "shippingAddress": {
          "fullName": "Jane Reviewer",
          "phone": "+263773333333",
          "addressLine1": "123 Review St",
          "city": "Harare",
          "country": "Zimbabwe"
        },
        "currency": "USD"
      }
    }
  }')
ORDER_ID=$(echo "$CHECKOUT" | jq -r '.data.checkout.id')
ORDER_NUMBER=$(echo "$CHECKOUT" | jq -r '.data.checkout.orderNumber')
echo "✓ Order created: #$ORDER_NUMBER (ID: $ORDER_ID)"

echo ""
echo "=== 8. Create Review (Buyer) ==="
CREATE_REVIEW=$(curl -sf -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  -d "{
    \"query\": \"mutation CreateReview(\$input: CreateReviewInput!) { createReview(input: \$input) { id ratingDescription content reviewType } }\",
    \"variables\": {
      \"input\": {
        \"orderId\": \"$ORDER_ID\",
        \"productId\": \"$PRODUCT_ID\",
        \"variantId\": \"$VARIANT_ID\",
        \"ratingDescription\": 3,
        \"ratingService\": 4,
        \"ratingDelivery\": 3,
        \"content\": \"Original review: decent product, fast shipping.\",
        \"isAnonymous\": false,
        \"mediaUrls\": [\"http://example.com/image1.png\"]
      }
    }
  }")
echo "Create Review Response:"
echo "$CREATE_REVIEW" | jq .
REVIEW_ID=$(echo "$CREATE_REVIEW" | jq -r '.data.createReview.id')

echo ""
echo "=== 9. Update Review (Buyer - Upgrade star rating to Positive) ==="
# Note: Modifications are only allowed to upgrade ratings to positive (4 or 5 stars) from neutral/bad (<= 3 stars)
UPDATE_REVIEW=$(curl -sf -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  -d "{
    \"query\": \"mutation UpdateReview(\$input: UpdateReviewInput!) { updateReview(input: \$input) { id ratingDescription content reviewType } }\",
    \"variables\": {
      \"input\": {
        \"reviewId\": \"$REVIEW_ID\",
        \"ratingDescription\": 5,
        \"ratingService\": 5,
        \"ratingDelivery\": 5,
        \"content\": \"Updated review: actually this is extremely good!\"
      }
    }
  }")
echo "Update Review Response:"
echo "$UPDATE_REVIEW" | jq .

echo ""
echo "=== 10. Append Review (Buyer) ==="
APPEND_REVIEW=$(curl -sf -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  -d "{
    \"query\": \"mutation AppendReview(\$input: AppendReviewInput!) { appendReview(input: \$input) { id content hasMedia } }\",
    \"variables\": {
      \"input\": {
        \"reviewId\": \"$REVIEW_ID\",
        \"content\": \"Additional feedback: still loving it after 3 days!\",
        \"mediaUrls\": [\"http://example.com/image2.png\"]
      }
    }
  }")
echo "Append Review Response:"
echo "$APPEND_REVIEW" | jq .

echo ""
echo "=== 11. Create Seller Reply (Seller) ==="
SELLER_REPLY=$(curl -sf -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SELLER_TOKEN" \
  -d "{
    \"query\": \"mutation CreateSellerReply(\$input: SellerReplyInput!) { createSellerReply(input: \$input) { id content replyType } }\",
    \"variables\": {
      \"input\": {
        \"reviewId\": \"$REVIEW_ID\",
        \"replyType\": \"initial\",
        \"content\": \"Thank you for the positive review! We hope to serve you again.\"
      }
    }
  }")
echo "Seller Reply Response:"
echo "$SELLER_REPLY" | jq .

echo ""
echo "=== 12. Query Product Reviews & Stats ==="
PRODUCT_DETAILS=$(curl -sf -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -d "{
    \"query\": \"query { product(id: \\\"$PRODUCT_ID\\\") { rating reviewCount reviews { edges { id content ratingDescription buyerName isAnonymous appendReview { content } replies { content } } } reviewStats { averageRating totalReviews goodCount neutralCount badCount topTags } } }\"
  }")
echo "Product Details with Reviews:"
echo "$PRODUCT_DETAILS" | jq .

echo ""
echo "=== 13. Query Seller DSR (Detailed Seller Rating) ==="
SELLER_DETAILS=$(curl -sf -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -d "{
    \"query\": \"query { seller(id: \\\"$STORE_ID\\\") { storeName rating dsr { avgDescription avgService avgDelivery reputationScore } } }\"
  }")
echo "Seller Details with DSR:"
echo "$SELLER_DETAILS" | jq .

echo ""
echo "=== 14. Delete Review (Buyer) ==="
DELETE_REVIEW=$(curl -sf -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  -d "{
    \"query\": \"mutation { deleteReview(reviewId: \\\"$REVIEW_ID\\\") }\"
  }")
echo "Delete Review Response:"
echo "$DELETE_REVIEW" | jq .

echo ""
echo "=== 15. Verify Deletion ==="
VERIFY_DETAILS=$(curl -sf -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -d "{
    \"query\": \"query { product(id: \\\"$PRODUCT_ID\\\") { reviewCount reviews { totalCount } } }\"
  }")
echo "Verified Reviews count after deletion:"
echo "$VERIFY_DETAILS" | jq .
