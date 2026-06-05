#!/bin/bash
set -e

GATEWAY_URL="http://localhost:8080/graphql"

echo "=== 1. Registering Buyer (buyer@example.com) ==="
# Let's send OTP first:
curl -s -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -d '{"query":"mutation { buyerSendOTP(phone: \"+263771111111\") { message requestId } }"}'

echo -e "\nVerifying OTP to log in buyer..."
BUYER_AUTH=$(curl -s -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -d '{"query":"mutation { buyerVerifyOTP(phone: \"+263771111111\", otp: \"123456\") { accessToken user { id } } }"}')

BUYER_TOKEN=$(echo $BUYER_AUTH | grep -o '"accessToken":"[^"]*' | grep -o '[^"]*$')
echo "Buyer Token: $BUYER_TOKEN"

echo -e "\n=== 2. Registering Seller (harare_seller@example.com) ==="
SELLER_AUTH=$(curl -s -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -d '{"query":"mutation { sellerRegister(email: \"harare_seller@example.com\", password: \"Password123!\", fullName: \"Harare Store Owner\") { accessToken user { id } } }"}')

if [[ $SELLER_AUTH == *"already exists"* ]] || [[ $SELLER_AUTH == *"registration failed"* ]]; then
  echo "Seller already exists. Logging in..."
  SELLER_AUTH=$(curl -s -X POST $GATEWAY_URL \
    -H "Content-Type: application/json" \
    -d '{"query":"mutation { sellerLogin(email: \"harare_seller@example.com\", password: \"Password123!\") { accessToken user { id } } }"}')
fi

SELLER_TOKEN=$(echo $SELLER_AUTH | grep -o '"accessToken":"[^"]*' | grep -o '[^"]*$')
echo "Seller Token: $SELLER_TOKEN"

echo -e "\n=== 3. Creating Store around Harare CBD ==="
CREATE_STORE_RESP=$(curl -s -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SELLER_TOKEN" \
  -d '{"query":"mutation { createStore(input: { storeName: \"Harare CBD Premium Store\", description: \"High quality electronics in Harare CBD\", latitude: -17.8292, longitude: 31.0522, logoUrl: \"https://wemall.co.zw/assets/store_logo_placeholder.png\", bannerUrl: \"https://wemall.co.zw/assets/store_banner_placeholder.png\" }) { id storeName latitude longitude status isVerified } }"}')

if [[ $CREATE_STORE_RESP == *"already exists"* ]] || [[ $CREATE_STORE_RESP == *"already exists for this user"* ]]; then
  echo "Store already exists. Fetching existing store..."
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

echo -e "\n=== 4. Querying Store Status ==="
QUERY_STORE_RESP=$(curl -s -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -d "{\"query\":\"query { seller(id: \\\"$STORE_ID\\\") { storeName latitude longitude status isVerified } }\"}")
echo "Query Store Response: $QUERY_STORE_RESP"

echo -e "\n=== 5. Promoting an Admin Account ==="
ADMIN_AUTH=$(curl -s -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -d '{"query":"mutation { sellerRegister(email: \"admin@example.com\", password: \"AdminPass123!\", fullName: \"System Admin\") { accessToken user { id } } }"}')

if [[ $ADMIN_AUTH == *"already exists"* ]] || [[ $ADMIN_AUTH == *"registration failed"* ]]; then
  echo "Admin already exists."
fi

echo "Promoting user admin@example.com to admin role in database..."
docker compose exec -T postgres-users psql -U wemall -d wemall_users -c "UPDATE users SET role = 'admin' WHERE email = 'admin@example.com';"

echo "Logging in Admin to refresh token/role..."
ADMIN_LOGIN_RESP=$(curl -s -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -d '{"query":"mutation { sellerLogin(email: \"admin@example.com\", password: \"AdminPass123!\") { accessToken user { id role } } }"}')
echo "Admin Login Response: $ADMIN_LOGIN_RESP"

ADMIN_TOKEN=$(echo $ADMIN_LOGIN_RESP | grep -o '"accessToken":"[^"]*' | grep -o '[^"]*$')
echo "Admin Token: $ADMIN_TOKEN"

echo -e "\n=== 6. Admin updates Seller status to PROCESSING ==="
STATUS_PROC_RESP=$(curl -s -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d "{\"query\":\"mutation { updateSellerStatus(sellerId: \\\"$STORE_ID\\\", status: PROCESSING) { id storeName status isVerified } }\"}")
echo "Status Update (PROCESSING) Response: $STATUS_PROC_RESP"

echo -e "\n=== 7. Admin updates Seller status to VERIFIED ==="
STATUS_VER_RESP=$(curl -s -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d "{\"query\":\"mutation { updateSellerStatus(sellerId: \\\"$STORE_ID\\\", status: VERIFIED) { id storeName status isVerified } }\"}")
echo "Status Update (VERIFIED) Response: $STATUS_VER_RESP"

echo -e "\n=== 8. Buyer follows Store ==="
FOLLOW_RESP=$(curl -s -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  -d "{\"query\":\"mutation { followStore(sellerId: \\\"$STORE_ID\\\") }\"}")
echo "Follow Response: $FOLLOW_RESP"

echo -e "\n=== 9. Querying Follow status ==="
IS_FOLLOWING_RESP=$(curl -s -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  -d "{\"query\":\"query { isFollowingStore(sellerId: \\\"$STORE_ID\\\") }\"}")
echo "Is Following Response: $IS_FOLLOWING_RESP"

echo -e "\n=== 10. Querying My Followed Stores ==="
MY_FOLLOWS_RESP=$(curl -s -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  -d '{"query":"query { myFollowedStores { sellers { id storeName status } total } }"}')
echo "My Followed Stores: $MY_FOLLOWS_RESP"

echo -e "\n=== 11. Buyer unfollows Store ==="
UNFOLLOW_RESP=$(curl -s -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  -d "{\"query\":\"mutation { unfollowStore(sellerId: \\\"$STORE_ID\\\") }\"}")
echo "Unfollow Response: $UNFOLLOW_RESP"

echo -e "\n=== 12. Querying Follow status after Unfollow ==="
IS_FOLLOWING_AFTER_RESP=$(curl -s -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $BUYER_TOKEN" \
  -d "{\"query\":\"query { isFollowingStore(sellerId: \\\"$STORE_ID\\\") }\"}")
echo "Is Following After Unfollow: $IS_FOLLOWING_AFTER_RESP"

echo -e "\n=== 13. Admin updates Seller status to SUSPENDED ==="
STATUS_SUSP_RESP=$(curl -s -X POST $GATEWAY_URL \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d "{\"query\":\"mutation { updateSellerStatus(sellerId: \\\"$STORE_ID\\\", status: SUSPENDED) { id storeName status isVerified } }\"}")
echo "Status Update (SUSPENDED) Response: $STATUS_SUSP_RESP"

echo "=== All Tests Completed ==="
