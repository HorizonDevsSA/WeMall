#!/bin/bash
set -e

GATEWAY_URL="http://localhost:8080/graphql"

echo "=== 1. Registering / Logging in Buyer ==="
curl -s -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -d '{"query":"mutation { buyerSendOTP(phone: \"+263772222222\") { message requestId } }"}'

echo -e "\nVerifying OTP to log in buyer..."
BUYER_AUTH=$(curl -s -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -d '{"query":"mutation { buyerVerifyOTP(phone: \"+263772222222\", otp: \"123456\") { accessToken user { id } } }"}')

BUYER_TOKEN=$(echo $BUYER_AUTH | grep -o '"accessToken":"[^"]*' | grep -o '[^"]*$')
echo "Buyer Token: $BUYER_TOKEN"

echo -e "\n=== 2. Registering / Logging in Seller ==="
SELLER_AUTH=$(curl -s -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -d '{"query":"mutation { sellerRegister(email: \"harare_seller_nearby@example.com\", password: \"Password123!\", fullName: \"Harare Local Owner\") { accessToken user { id } } }"}')

# If already registered, login:
if [[ $SELLER_AUTH == *"already exists"* ]] || [[ $SELLER_AUTH == *"registration failed"* ]]; then
  echo "User already exists. Logging in..."
  SELLER_AUTH=$(curl -s -X POST $GATEWAY_URL \
    -H "Content-Type: application/json" \
    -d '{"query":"mutation { sellerLogin(email: \"harare_seller_nearby@example.com\", password: \"Password123!\") { accessToken user { id } } }"}')
fi

SELLER_TOKEN=$(echo $SELLER_AUTH | grep -o '"accessToken":"[^"]*' | grep -o '[^"]*$')
echo "Seller Token: $SELLER_TOKEN"

echo -e "\n=== 3. Creating Store around Harare CBD ==="
# Harare CBD coordinates: Latitude: -17.8292, Longitude: 31.0522
CREATE_STORE_RESP=$(curl -s -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SELLER_TOKEN" \
  -d '{"query":"mutation { createStore(input: { storeName: \"Harare CBD Local Shop\", description: \"High quality goods in central Harare\", latitude: -17.8292, longitude: 31.0522, logoUrl: \"https://wemall.co.zw/assets/store_logo_placeholder.png\", bannerUrl: \"https://wemall.co.zw/assets/store_banner_placeholder.png\" }) { id storeName latitude longitude status isVerified } }"}')

# If store already exists, get it:
if [[ $CREATE_STORE_RESP == *"already exists"* ]]; then
  echo "Store already exists. Getting existing store..."
  CREATE_STORE_RESP=$(curl -s -X POST $GATEWAY_URL \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $SELLER_TOKEN" \
    -d '{"query":"query { myStore { id storeName latitude longitude status isVerified } }"}')
  STORE_ID=$(echo $CREATE_STORE_RESP | grep -o '"id":"[^"]*' | grep -o '[^"]*$')
else
  echo "Create Store Response: $CREATE_STORE_RESP"
  STORE_ID=$(echo $CREATE_STORE_RESP | grep -o '"id":"[^"]*' | grep -o '[^"]*$')
fi
echo "Store ID: $STORE_ID"

echo -e "\n=== 4. Fetching Category ID ==="
CATEGORIES_RESP=$(curl -s -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -d '{"query":"query { categories { id name slug children { id name slug } } }"}')

# Extract the first subcategory ID (e.g. Women's clothing / Men's clothing / etc)
CATEGORY_ID=$(echo $CATEGORIES_RESP | grep -o '"id":"[^"]*' | head -n 2 | tail -n 1 | grep -o '[^"]*$')
echo "Selected Category ID: $CATEGORY_ID"

SKU="HARARE-$RANDOM"
CREATE_PRODUCT_RESP=$(curl -s -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SELLER_TOKEN" \
  -d "{\"query\":\"mutation { createProduct(input: { categoryId: \\\"$CATEGORY_ID\\\", title: \\\"Harare Local Product\\\", description: \\\"Fabulous product right in Harare CBD\\\", brand: \\\"HarareBrand\\\", tags: [\\\"local\\\", \\\"harare\\\"], attributes: {}, variants: [{ sku: \\\"$SKU\\\", price: 29.99, options: {} }] }) { id title latitude longitude seller { id storeName } } }\"}")

echo "Create Product Response: $CREATE_PRODUCT_RESP"
PRODUCT_ID=$(echo $CREATE_PRODUCT_RESP | grep -o '"id":"[^"]*' | head -n 1 | grep -o '[^"]*$')

echo "Product ID: $PRODUCT_ID"

echo -e "\n=== 6. Querying Nearby Products (Expects the created product with distance and seller info) ==="
# Query nearby products from a coordinate very close to Harare CBD: -17.83, 31.05 (distance should be < 5km)
NEARBY_RESP=$(curl -s -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -d '{"query":"query { nearbyProducts(latitude: -17.83, longitude: 31.05, radiusMeters: 5000) { distance product { id title latitude longitude seller { id storeName } } } }"}')

echo "Nearby Products Response: $NEARBY_RESP"

echo -e "\n=== 7. Buyer follows Harare Store ==="
FOLLOW_RESP=$(curl -s -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  -d "{\"query\":\"mutation { followStore(sellerId: \\\"$STORE_ID\\\") }\"}")
echo "Follow Response: $FOLLOW_RESP"

echo -e "\n=== 8. Checking Follow Status ==="
IS_FOLLOWING_RESP=$(curl -s -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  -d "{\"query\":\"query { isFollowingStore(sellerId: \\\"$STORE_ID\\\") }\"}")
echo "Is Following: $IS_FOLLOWING_RESP"

echo -e "\n=== 9. Listing Followed Stores via myFollowedStores ==="
MY_FOLLOWS_RESP=$(curl -s -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  -d '{"query":"query { myFollowedStores { sellers { id storeName status } total } }"}')
echo "My Followed Stores: $MY_FOLLOWS_RESP"

echo -e "\n=== 10. Buyer unfollows Harare Store ==="
UNFOLLOW_RESP=$(curl -s -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  -d "{\"query\":\"mutation { unfollowStore(sellerId: \\\"$STORE_ID\\\") }\"}")
echo "Unfollow Response: $UNFOLLOW_RESP"

echo -e "\n=== 11. Checking Follow Status after Unfollow ==="
IS_FOLLOWING_AFTER_RESP=$(curl -s -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  -d "{\"query\":\"query { isFollowingStore(sellerId: \\\"$STORE_ID\\\") }\"}")
echo "Is Following After: $IS_FOLLOWING_AFTER_RESP"

echo -e "\n=== Verification complete! ==="
