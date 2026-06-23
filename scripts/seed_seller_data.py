import os
import time
import json
import random

GATEWAY_URL = "https://15.240.45.232.nip.io/graphql"
SELLER_TOKEN = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiYTg3NTNhODUtNmU3NS00ZGJiLThmZmEtZGVjODdmZjdiNjZkIiwicm9sZSI6InNlbGxlciIsImV4cCI6MTc4MjQ2NTg5NCwiaWF0IjoxNzgxODYxMDk0fQ.WEznfn34an_wvfTCutp496CRUsgc8WlkfP7ig_Xfk5c"

IMAGES = [
    {
        "originalName": "hotels.jpg",
        "thumbnail": "https://d2v0vjgmrxer0s.cloudfront.net/uploads/thumbnails/28b60766-c64a-4383-80a3-d654858aa131-hotels.webp",
        "main": "https://d2v0vjgmrxer0s.cloudfront.net/uploads/originals/28b60766-c64a-4383-80a3-d654858aa131-hotels.jpg",
        "compressed": "https://d2v0vjgmrxer0s.cloudfront.net/uploads/compressed/28b60766-c64a-4383-80a3-d654858aa131-hotels.webp"
    },
    {
        "originalName": "drive_now.jpg",
        "thumbnail": "https://d2v0vjgmrxer0s.cloudfront.net/uploads/thumbnails/29aa70df-1575-4a7e-8d30-a9e4ddff556d-drive_now.webp",
        "main": "https://d2v0vjgmrxer0s.cloudfront.net/uploads/originals/29aa70df-1575-4a7e-8d30-a9e4ddff556d-drive_now.jpg",
        "compressed": "https://d2v0vjgmrxer0s.cloudfront.net/uploads/compressed/29aa70df-1575-4a7e-8d30-a9e4ddff556d-drive_now.webp"
    }
]

PRODUCT_TITLES = [
    "Luxury Suite Weekend Getaway", "Economy Car Rental - 1 Day", 
    "Premium Sedan Rental - Weekend", "Boutique Hotel Standard Room", 
    "Family SUV Rental", "Executive Hotel Suite",
    "Convertible Sports Car", "Resort Villa Stay",
    "Budget Friendly Car Rental", "Downtown Business Hotel",
    "Minivan for Family Trips", "Oceanview Resort Room",
    "Off-road 4x4 Rental", "Ski Lodge Retreat",
    "City Compact Car", "Bed & Breakfast Stay",
    "Luxury SUV Chauffeur Service", "Motel Stopover",
    "Electric Vehicle Rental", "Penthouse Suite Experience"
]

import urllib.request
import urllib.error

def gql_request(query, variables=None, token=None):
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    payload = {"query": query}
    if variables:
        payload["variables"] = variables
    
    data = json.dumps(payload).encode('utf-8')
    req = urllib.request.Request(GATEWAY_URL, data=data, headers=headers, method="POST")
    
    try:
        with urllib.request.urlopen(req) as response:
            res_json = json.loads(response.read().decode('utf-8'))
            if "errors" in res_json:
                print(f"GraphQL Error in request: {json.dumps(payload, indent=2)}")
                print(f"Errors: {json.dumps(res_json['errors'], indent=2)}")
                return None
            return res_json.get("data")
    except urllib.error.HTTPError as e:
        print(f"Error: HTTP {e.code}")
        print(e.read().decode('utf-8'))
        return None
    except Exception as e:
        print(f"Exception: {e}")
        return None

def main():
    print("=== 1. Checking Store ===")
    my_store_data = gql_request("query { myStore { id storeName status } }", token=SELLER_TOKEN)
    if not my_store_data:
        return
    store = my_store_data.get("myStore")
    if not store:
        print("Creating Store...")
        store_res = gql_request("""
            mutation CreateStore($input: CreateStoreInput!) {
                createStore(input: $input) { id storeName }
            }
        """, {
            "input": {
                "storeName": "Horizon Premium Store",
                "description": "Premium services and rentals",
                "latitude": -17.8292,
                "longitude": 31.0522
            }
        }, token=SELLER_TOKEN)
        if store_res and store_res.get("createStore"):
            store_id = store_res["createStore"]["id"]
            print(f"Store created: {store_id}")
        else:
            print("Could not create store. Exiting.")
            return
    else:
        store_id = store["id"]
        print(f"Found store: {store['storeName']} ({store_id})")

    print("\n=== 2. Fetching Category ===")
    cats_data = gql_request("query { categories { id name children { id name } } }")
    if not cats_data or not cats_data.get("categories"):
        print("No categories found.")
        return
    category_id = cats_data["categories"][0]["id"]
    if cats_data["categories"][0].get("children"):
        category_id = cats_data["categories"][0]["children"][0]["id"]
    print(f"Using category ID: {category_id}")

    print("\n=== 3. Seeding 20 Products ===")
    product_ids = []
    variant_ids = []
    for i, title in enumerate(PRODUCT_TITLES):
        image_data = IMAGES[i % 2]
        product_type = "SERVICES"
        price = random.randint(20, 500) + 0.99
        sku = f"SKU-{int(time.time())}-{i}"
        
        product_res = gql_request("""
            mutation CreateProduct($input: CreateProductInput!) {
                createProduct(input: $input) {
                    id
                    title
                    variants { id sku price }
                }
            }
        """, {
            "input": {
                "categoryId": category_id,
                "title": title,
                "description": f"Amazing {title} provided by Horizon.",
                "brand": "Horizon",
                "productType": product_type,
                "attributes": {},
                "imageUrl": image_data["main"],
                "thumbnailUrl": image_data["thumbnail"],
                "images": [image_data["main"]],
                "variants": [{
                    "sku": sku,
                    "price": price,
                    "options": {"standard": "Default"},
                    "initialQuantity": 100
                }]
            }
        }, token=SELLER_TOKEN)
        
        if product_res and product_res.get("createProduct"):
            prod_id = product_res["createProduct"]["id"]
            var_id = product_res["createProduct"]["variants"][0]["id"]
            product_ids.append(prod_id)
            variant_ids.append(var_id)
            print(f"Created Product {i+1}: {title} (ID: {prod_id})")
        else:
            print(f"Failed to create product {i+1}")

    import os
    
    print("\n=== 4. Buyer Setup ===")
    phone = "+263773333333"
    print(f"Sending OTP for {phone}")
    gql_request("""
        mutation BuyerSendOTP($phone: String!) {
            buyerSendOTP(phone: $phone) { message }
        }
    """, {"phone": phone})
    time.sleep(2)
    
    # Extract the OTP from the remote server's user-service logs
    cmd = f"ssh -o StrictHostKeyChecking=no -i wemall-prod-key.pem ubuntu@15.240.45.232 \"docker logs wemall-user-service-1 2>&1 | grep 'To: {phone}' | tail -1 | grep -o 'code: [0-9]*' | cut -d' ' -f2\""
    otp = os.popen(cmd).read().strip()
    
    if not otp:
        print("Failed to extract OTP from remote server logs.")
        return
        
    print(f"Extracted OTP: {otp}")
    
    buyer_auth_data = gql_request("""
        mutation BuyerVerifyOTP($phone: String!, $otp: String!) {
            buyerVerifyOTP(phone: $phone, otp: $otp) {
                accessToken
                user { id }
            }
        }
    """, {"phone": phone, "otp": otp})
    
    if not buyer_auth_data or not buyer_auth_data.get("buyerVerifyOTP"):
        print("Buyer login failed.")
        return
        
    buyer_token = buyer_auth_data["buyerVerifyOTP"]["accessToken"]
    buyer_id = buyer_auth_data["buyerVerifyOTP"]["user"]["id"]
    print(f"Buyer logged in. ID: {buyer_id}")

    print("\n=== 5. Creating Orders & Reviews ===")
    orders = []
    reviews = []
    for i in range(5):
        # Pick a random product
        idx = random.randint(0, len(product_ids)-1)
        var_id = variant_ids[idx]
        prod_id = product_ids[idx]
        
        print(f"Adding variant {var_id} to cart...")
        gql_request("""
            mutation AddToCart($variantId: ID!, $quantity: Int!) {
                addToCart(variantId: $variantId, quantity: $quantity) { id }
            }
        """, {"variantId": var_id, "quantity": 1}, token=buyer_token)
        
        print("Checking out...")
        checkout_res = gql_request("""
            mutation Checkout($input: CheckoutInput!) {
                checkout(input: $input) { id orderNumber }
            }
        """, {
            "input": {
                "shippingAddress": {
                    "fullName": "Test Buyer",
                    "phone": phone,
                    "addressLine1": "123 Test St",
                    "city": "Harare",
                    "country": "Zimbabwe"
                },
                "currency": "USD"
            }
        }, token=buyer_token)
        
        if checkout_res and checkout_res.get("checkout"):
            order_id = checkout_res["checkout"]["id"]
            orders.append(order_id)
            print(f"Order created: {order_id}")
            
            # Create review
            rating = random.choice([3, 4, 5])
            content = f"Really enjoyed this service! Rating: {rating} stars."
            rev_res = gql_request("""
                mutation CreateReview($input: CreateReviewInput!) {
                    createReview(input: $input) { id }
                }
            """, {
                "input": {
                    "orderId": order_id,
                    "productId": prod_id,
                    "variantId": var_id,
                    "ratingDescription": rating,
                    "ratingService": rating,
                    "ratingDelivery": rating,
                    "content": content
                }
            }, token=buyer_token)
            
            if rev_res and rev_res.get("createReview"):
                review_id = rev_res["createReview"]["id"]
                reviews.append(review_id)
                print(f"Review created: {review_id}")

    print("\n=== 6. Seller Replies to Reviews ===")
    for review_id in reviews[:2]:  # Reply to first two reviews
        reply_res = gql_request("""
            mutation CreateSellerReply($input: SellerReplyInput!) {
                createSellerReply(input: $input) { id }
            }
        """, {
            "input": {
                "reviewId": review_id,
                "replyType": "initial",
                "content": "Thank you so much for your feedback!"
            }
        }, token=SELLER_TOKEN)
        print(f"Replied to review {review_id}")

    print("\n=== 7. Chat Simulation ===")
    if orders:
        chat_thread_res = gql_request("""
            mutation CreateChatThread($sellerId: ID!, $orderId: ID) {
                createChatThread(sellerId: $sellerId, orderId: $orderId) { id }
            }
        """, {
            "sellerId": store_id,
            "orderId": orders[0]
        }, token=buyer_token)
        
        if chat_thread_res and chat_thread_res.get("createChatThread"):
            thread_id = chat_thread_res["createChatThread"]["id"]
            print(f"Chat Thread created: {thread_id}")
            
            # Buyer sends message
            gql_request("""
                mutation SendChatMessage($threadId: ID!, $content: String!) {
                    sendChatMessage(threadId: $threadId, content: $content) { id }
                }
            """, {
                "threadId": thread_id,
                "content": "Hello! I have a question about my booking."
            }, token=buyer_token)
            print("Buyer message sent.")
            
            # Seller sends message
            gql_request("""
                mutation SendChatMessage($threadId: ID!, $content: String!) {
                    sendChatMessage(threadId: $threadId, content: $content) { id }
                }
            """, {
                "threadId": thread_id,
                "content": "Hi there! I would be happy to help. What do you need?"
            }, token=SELLER_TOKEN)
            print("Seller message sent.")
            
    print("\n=== 8. Analytics Hits ===")
    for prod_id in product_ids[:5]:
        gql_request("""
            mutation RecordProductView($productId: ID!) {
                recordProductView(productId: $productId)
            }
        """, {"productId": prod_id}, token=buyer_token)
        print(f"Recorded view for product {prod_id}")

    print("\n=== ALL SEEDING COMPLETE ===")

if __name__ == "__main__":
    main()
