#!/usr/bin/env python3
import json
import urllib.request
import urllib.error
import random

GATEWAY_URL  = "https://15.240.45.232.nip.io/graphql"
SELLER_EMAIL = "horizondevs19@gmail.com"

# Products to seed mapped by slug keyword
SEED_PRODUCTS = [
    # Bags, Watches & Jewelry
    {
        "category_keyword": "bags-watches-and-jewelry",
        "title": "Leather Travel Duffel Bag",
        "desc": "Spacious premium leather duffel bag for weekends, travel, or gym.",
        "price": 85.00,
        "type": "FASHION",
        "brand": "Horizon Leather"
    },
    {
        "category_keyword": "bags-watches-and-jewelry",
        "title": "Classic Quartz Wristwatch",
        "desc": "Water-resistant, elegant design quartz wristwatch with stainless steel band.",
        "price": 120.00,
        "type": "FASHION",
        "brand": "Horizon Chrono"
    },
    # Home & Living
    {
        "category_keyword": "home-and-living",
        "title": "Cozy Cotton Bed Sheets",
        "desc": "100% breathable organic cotton bed sheets, King size, hypoallergenic.",
        "price": 45.00,
        "type": "HOME_FURNITURE",
        "brand": "Horizon Home"
    },
    {
        "category_keyword": "home-and-living",
        "title": "Decorative Ceramic Flower Vase",
        "desc": "Minimalist hand-crafted ceramic vase for dining table or living room.",
        "price": 25.00,
        "type": "HOME_FURNITURE",
        "brand": "Horizon Decor"
    },
    # Beauty & Personal Care
    {
        "category_keyword": "beauty-and-personal-care",
        "title": "Organic Aloe Vera Gel",
        "desc": "Soothing organic gel for skin hydration, sunburn relief, and hair care.",
        "price": 15.00,
        "type": "BEAUTY_HEALTH",
        "brand": "Horizon Organics"
    },
    {
        "category_keyword": "beauty-and-personal-care",
        "title": "Moisturizing Sunscreen SPF 50",
        "desc": "Non-greasy, broad-spectrum UVA/UVB protection sunscreen for all skin types.",
        "price": 19.99,
        "type": "BEAUTY_HEALTH",
        "brand": "Horizon Sun"
    },
    # Groceries & Fresh Food
    {
        "category_keyword": "groceries-and-fresh-food",
        "title": "Fresh Organic Gala Apples (1kg)",
        "desc": "Sweet, crisp, and fresh local organic gala apples.",
        "price": 4.50,
        "type": "AGRICULTURE",
        "brand": "Fresh Farms"
    },
    {
        "category_keyword": "groceries-and-fresh-food",
        "title": "Premium ZWG Roasted Coffee Beans",
        "desc": "Medium roast, single-origin ZWG arabica coffee beans, whole bean (250g).",
        "price": 12.50,
        "type": "AGRICULTURE",
        "brand": "Horizon Roasters"
    },
    # Sports & Fitness
    {
        "category_keyword": "sports-and-fitness",
        "title": "Non-Slip Exercise Yoga Mat",
        "desc": "Eco-friendly, double-layered non-slip exercise mat with carrying strap.",
        "price": 29.99,
        "type": "SPORTS_OUTDOORS",
        "brand": "Horizon Fit"
    },
    {
        "category_keyword": "sports-and-fitness",
        "title": "Adjustable Dumbbells Set (20kg)",
        "desc": "All-in-one dumbbell weight set with connector bar for home workout.",
        "price": 89.99,
        "type": "SPORTS_OUTDOORS",
        "brand": "Horizon Gym"
    },
    # Electronics & Technology
    {
        "category_keyword": "electronics-and-technology",
        "title": "Universal Fast-Charging Power Bank",
        "desc": "20000mAh external battery pack with PD 22.5W fast charge, USB-C ports.",
        "price": 34.99,
        "type": "ELECTRONICS",
        "brand": "Horizon Power"
    },
    {
        "category_keyword": "electronics-and-technology",
        "title": "Wireless Ergonomic Mouse",
        "desc": "Quiet click, comfortable thumb rest, adjustable DPI, dual bluetooth mode.",
        "price": 24.50,
        "type": "ELECTRONICS",
        "brand": "Horizon Tech"
    },
    # Fashion & Apparel
    {
        "category_keyword": "fashion-and-apparel",
        "title": "Unisex Denim Jacket",
        "desc": "Classic blue button-down denim jacket, regular fit, premium cotton denim.",
        "price": 49.99,
        "type": "FASHION",
        "brand": "Horizon Denim"
    },
    {
        "category_keyword": "fashion-and-apparel",
        "title": "Classic White Cotton T-Shirt",
        "desc": "Comfortable 100% combed cotton crewneck t-shirt.",
        "price": 15.00,
        "type": "FASHION",
        "brand": "Horizon Wear"
    }
]

# Image fallbacks
THUMB_URL = "https://d2v0vjgmrxer0s.cloudfront.net/uploads/thumbnails/28b60766-c64a-4383-80a3-d654858aa131-hotels.webp"
MAIN_URL = "https://d2v0vjgmrxer0s.cloudfront.net/uploads/originals/28b60766-c64a-4383-80a3-d654858aa131-hotels.jpg"

def gql(query, variables=None, token=None):
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    payload = {"query": query}
    if variables:
        payload["variables"] = variables
    data = json.dumps(payload).encode("utf-8")
    req = urllib.request.Request(GATEWAY_URL, data=data, headers=headers, method="POST")
    try:
        with urllib.request.urlopen(req, timeout=30) as response:
            res = json.loads(response.read().decode("utf-8"))
            if "errors" in res:
                msg = res["errors"][0].get("message", "unknown") if res["errors"] else "unknown"
                print(f"  ⚠ GraphQL error: {msg}")
                return None
            return res.get("data")
    except urllib.error.HTTPError as e:
        print(f"  ✗ HTTP {e.code}: {e.read().decode()[:200]}")
        return None
    except Exception as e:
        print(f"  ✗ Exception: {e}")
        return None

def main():
    print("============================================================")
    print("  WeMall Seeder — Seeding Products Across All 7 Categories")
    print("============================================================")

    # 1. Authenticate Seller
    print("\n[1] Authenticating seller...")
    auth_res = gql("""
        mutation SellerFirebaseSignIn($idToken: String!, $fullName: String!) {
          sellerFirebaseSignIn(idToken: $idToken, fullName: $fullName) {
            accessToken
            user { id email role }
          }
        }
    """, {"idToken": f"mock-firebase-token-{SELLER_EMAIL}", "fullName": "Horizon Devs"})

    if not auth_res or not auth_res.get("sellerFirebaseSignIn"):
        print("  ✗ Seller login failed")
        return
    seller_token = auth_res["sellerFirebaseSignIn"]["accessToken"]
    print(f"  ✓ Logged in (token active)")

    # 2. Get/Create Store
    print("\n[2] Checking store...")
    my_store_data = gql("query { myStore { id storeName status } }", token=seller_token)
    if not my_store_data or not my_store_data.get("myStore"):
        print("  ✗ Store not found. Creating store...")
        store_res = gql("""
            mutation CreateStore($input: CreateStoreInput!) {
                createStore(input: $input) { id storeName }
            }
        """, {
            "input": {
                "storeName": "Horizon Premium Store",
                "description": "Multi-category premium goods and services",
                "latitude": -17.8292,
                "longitude": 31.0522
            }
        }, token=seller_token)
        if store_res and store_res.get("createStore"):
            store_id = store_res["createStore"]["id"]
            print(f"  ✓ Store created: {store_id}")
        else:
            print("  ✗ Could not create store. Exiting.")
            return
    else:
        store_id = my_store_data["myStore"]["id"]
        print(f"  ✓ Found store: {my_store_data['myStore']['storeName']} ({store_id})")

    # 3. Fetch Categories
    print("\n[3] Fetching categories...")
    cats_data = gql("query { categories { id name slug } }")
    if not cats_data or not cats_data.get("categories"):
        print("  ✗ No categories retrieved from server")
        return
    cats = cats_data["categories"]
    print(f"  ✓ Retrieved {len(cats)} categories from server")

    # Map categories by slug keyword
    category_map = {}
    for c in cats:
        category_map[c["slug"]] = c["id"]

    # 4. Seed Products
    print(f"\n[4] Seeding {len(SEED_PRODUCTS)} products across categories...")
    created_count = 0
    for i, p in enumerate(SEED_PRODUCTS):
        cat_id = category_map.get(p["category_keyword"])
        if not cat_id:
            # Fallback to first category if slug keyword doesn't match exactly
            cat_id = cats[0]["id"]
            print(f"  ⚠ Category '{p['category_keyword']}' not matched. Using fallback: {cats[0]['name']}")

        sku = f"SKU-{int(random.random()*100000000)}-{i}"
        
        variables = {
            "input": {
                "categoryId": cat_id,
                "title": p["title"],
                "description": p["desc"],
                "brand": p["brand"],
                "productType": p["type"],
                "attributes": {},
                "imageUrl": MAIN_URL,
                "thumbnailUrl": THUMB_URL,
                "images": [MAIN_URL],
                "variants": [{
                    "sku": sku,
                    "price": p["price"],
                    "options": {"standard": "Default"},
                    "initialQuantity": 100
                }]
            }
        }

        print(f"  → Creating product {i+1}/{len(SEED_PRODUCTS)}: {p['title']}...")
        res = gql("""
            mutation CreateProduct($input: CreateProductInput!) {
                createProduct(input: $input) {
                    id
                    title
                }
            }
        """, variables, token=seller_token)

        if res and res.get("createProduct"):
            created_count += 1
            print(f"    ✓ Successfully created product: {res['createProduct']['title']}")
        else:
            print(f"    ✗ Failed to create product")

    print("\n" + "=" * 60)
    print(f"  PRODUCT SEEDING COMPLETE ✓ (Total: {created_count}/{len(SEED_PRODUCTS)} created)")
    print("=" * 60)

if __name__ == "__main__":
    main()
