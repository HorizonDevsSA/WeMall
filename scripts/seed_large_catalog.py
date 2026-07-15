#!/usr/bin/env python3
import json
import urllib.request
import urllib.error
import random
import time
import sys

GATEWAY_URL = "https://15.240.45.232.nip.io/graphql"
SELLER_EMAIL = "akotoxmpimbo@gmail.com"

# User-provided images for Bags, Watches & Jewelry
BAG_IMAGES = [
    "https://m.media-amazon.com/images/I/81i2F9xHYFL._AC_SX675_.jpg",
    "https://image.made-in-china.com/43f34j00KUCkWzudgIpY/2023-Promotion-Cheap-Fashion-Quality-Ladies-Shoulder-Bags-with-Customized-Metal-Chain-Logo-Women-Wholesale-Handbags.jpg"
]

# Curated high-quality Unsplash images for other categories
CATEGORY_IMAGES = {
    "Home & Living": {
        "Cookware & Pots": "https://images.unsplash.com/photo-1584269600464-37b1b58a9fe7?auto=format&fit=crop&w=600&q=80",
        "Tableware & Plates": "https://images.unsplash.com/photo-1610701596007-11502861dcfa?auto=format&fit=crop&w=600&q=80",
        "Kitchen Appliances": "https://images.unsplash.com/photo-1574269909862-7e1d70bb8078?auto=format&fit=crop&w=600&q=80",
        "Rugs & Carpets": "https://images.unsplash.com/photo-1600121848594-d8644e57abab?auto=format&fit=crop&w=600&q=80",
        "Curtains & Blinds": "https://images.unsplash.com/photo-1513694203232-719a280e022f?auto=format&fit=crop&w=600&q=80",
        "Light Fixtures & Lamps": "https://images.unsplash.com/photo-1507473885765-e6ed057f782c?auto=format&fit=crop&w=600&q=80",
        "Sofas & Living Room": "https://images.unsplash.com/photo-1555041469-a586c61ea9bc?auto=format&fit=crop&w=600&q=80",
        "Beds & Bedroom": "https://images.unsplash.com/photo-1505693416388-ac5ce068fe85?auto=format&fit=crop&w=600&q=80",
        "Desks & Office": "https://images.unsplash.com/photo-1524758631624-e2822e304c36?auto=format&fit=crop&w=600&q=80"
    },
    "Beauty & Personal Care": {
        "Face Moisturizers": "https://images.unsplash.com/photo-1608248597279-f99d160bfcbc?auto=format&fit=crop&w=600&q=80",
        "Sunscreens": "https://images.unsplash.com/photo-1598440947619-2c35fc9aa908?auto=format&fit=crop&w=600&q=80",
        "Cleansers & Face Wash": "https://images.unsplash.com/photo-1556228720-195a672e8a03?auto=format&fit=crop&w=600&q=80",
        "Foundations & Powders": "https://images.unsplash.com/photo-1522335789203-aabd1fc54bc9?auto=format&fit=crop&w=600&q=80",
        "Lipsticks & Lip Care": "https://images.unsplash.com/photo-1586495777744-4413f21062fa?auto=format&fit=crop&w=600&q=80",
        "Eye Makeup": "https://images.unsplash.com/photo-1596462502278-27bfdc403348?auto=format&fit=crop&w=600&q=80",
        "Shampoos & Conditioners": "https://images.unsplash.com/photo-1535585209827-a15fcdbc4c2d?auto=format&fit=crop&w=600&q=80",
        "Hair Styling & Dyes": "https://images.unsplash.com/photo-1562322140-8baeececf3df?auto=format&fit=crop&w=600&q=80",
        "Hair Dryers & Tools": "https://images.unsplash.com/photo-1522337360788-8b13dee7a37e?auto=format&fit=crop&w=600&q=80"
    },
    "Groceries & Fresh Food": {
        "Rice & Grains": "https://images.unsplash.com/photo-1586201375761-83865001e31c?auto=format&fit=crop&w=600&q=80",
        "Cooking Oils": "https://images.unsplash.com/photo-1474979266404-7eaacbcd87c5?auto=format&fit=crop&w=600&q=80",
        "Canned Goods": "https://images.unsplash.com/photo-1536638317175-32449e112c58?auto=format&fit=crop&w=600&q=80",
        "Chips & Biscuits": "https://images.unsplash.com/photo-1599490659213-e2b9527bc087?auto=format&fit=crop&w=600&q=80",
        "Coffee & Tea": "https://images.unsplash.com/photo-1514432324607-a09d9b4aefdd?auto=format&fit=crop&w=600&q=80",
        "Soft Drinks & Juices": "https://images.unsplash.com/photo-1622483767028-3f66f32aef97?auto=format&fit=crop&w=600&q=80"
    },
    "Sports & Fitness": {
        "Treadmills & Cardio": "https://images.unsplash.com/photo-1517838277536-f5f99be501cd?auto=format&fit=crop&w=600&q=80",
        "Dumbbells & Weights": "https://images.unsplash.com/photo-1638536532686-d610adfc8e5c?auto=format&fit=crop&w=600&q=80",
        "Yoga Mats": "https://images.unsplash.com/photo-1592432678016-e910b452f9a2?auto=format&fit=crop&w=600&q=80",
        "Tents & Camping": "https://images.unsplash.com/photo-1504280390367-361c6d9f38f4?auto=format&fit=crop&w=600&q=80",
        "Bicycles & Cycling": "https://images.unsplash.com/photo-1485965120184-e220f721d03e?auto=format&fit=crop&w=600&q=80",
        "Hiking & Climbing": "https://images.unsplash.com/photo-1533240332313-0db49b459ad6?auto=format&fit=crop&w=600&q=80"
    },
    "Electronics & Technology": {
        "iPhones": "https://images.unsplash.com/photo-1510557880182-3d4d3cba35a5?auto=format&fit=crop&w=600&q=80",
        "Android Phones": "https://images.unsplash.com/photo-1598327105666-5b89351aff97?auto=format&fit=crop&w=600&q=80",
        "Tablets & iPads": "https://images.unsplash.com/photo-1544244015-0df4b3ffc6b0?auto=format&fit=crop&w=600&q=80",
        "DSLR & Mirrorless": "https://images.unsplash.com/photo-1516035069371-29a1b244cc32?auto=format&fit=crop&w=600&q=80",
        "Action Cameras": "https://images.unsplash.com/photo-1509198397868-475647b2a1e5?auto=format&fit=crop&w=600&q=80",
        "Drones": "https://images.unsplash.com/photo-1508614589041-895b88991e3e?auto=format&fit=crop&w=600&q=80",
        "Headphones & Earbuds": "https://images.unsplash.com/photo-1505740420928-5e560c06d30e?auto=format&fit=crop&w=600&q=80",
        "Bluetooth Speakers": "https://images.unsplash.com/photo-1608043152269-423dbba4e7e1?auto=format&fit=crop&w=600&q=80",
        "Home Theater Systems": "https://images.unsplash.com/photo-1545454675-3531b543be5d?auto=format&fit=crop&w=600&q=80",
        "Laptops": "https://images.unsplash.com/photo-1496181130204-755241544e35?auto=format&fit=crop&w=600&q=80",
        "Desktop PCs": "https://images.unsplash.com/photo-1587831990711-23ca6441447b?auto=format&fit=crop&w=600&q=80",
        "Computer Monitors": "https://images.unsplash.com/photo-1527443224154-c4a3942d3acf?auto=format&fit=crop&w=600&q=80"
    },
    "Fashion & Apparel": {
        "Sunglasses": "https://images.unsplash.com/photo-1511499767150-a48a237f0083?auto=format&fit=crop&w=600&q=80",
        "Belts": "https://images.unsplash.com/photo-1624222247344-550fb8ecf7db?auto=format&fit=crop&w=600&q=80",
        "Hats & Caps": "https://images.unsplash.com/photo-1534215754734-18e55d13e346?auto=format&fit=crop&w=600&q=80",
        "Shirts & T-Shirts": "https://images.unsplash.com/photo-1521572267360-ee0c2909d518?auto=format&fit=crop&w=600&q=80",
        "Jeans & Trousers": "https://images.unsplash.com/photo-1542272604-787c3835535d?auto=format&fit=crop&w=600&q=80",
        "Suits & Blazers": "https://images.unsplash.com/photo-1594938298603-c8148c4dae35?auto=format&fit=crop&w=600&q=80",
        "Dresses": "https://images.unsplash.com/photo-1595777457583-95e059d581b8?auto=format&fit=crop&w=600&q=80",
        "Tops & Blouses": "https://images.unsplash.com/photo-1503342217505-b0a15ec3261c?auto=format&fit=crop&w=600&q=80",
        "Skirts & Pants": "https://images.unsplash.com/photo-1583496661160-fb488b2c1a82?auto=format&fit=crop&w=600&q=80",
        "Jackets & Coats": "https://images.unsplash.com/photo-1551028719-00167b16eac5?auto=format&fit=crop&w=600&q=80",
        "Sneakers & Athletic": "https://images.unsplash.com/photo-1542291026-7eec264c27ff?auto=format&fit=crop&w=600&q=80",
        "Formal Shoes": "https://images.unsplash.com/photo-1533867617858-e7b97e060509?auto=format&fit=crop&w=600&q=80",
        "Heels & Sandals": "https://images.unsplash.com/photo-1543163521-1bf539c55dd2?auto=format&fit=crop&w=600&q=80"
    }
}

# Mapping root categories to ProductType GraphQL Enum
PRODUCT_TYPE_MAP = {
    "Bags, Watches & Jewelry": "FASHION",
    "Home & Living": "HOME_FURNITURE",
    "Beauty & Personal Care": "BEAUTY_HEALTH",
    "Groceries & Fresh Food": "AGRICULTURE",
    "Sports & Fitness": "SPORTS_OUTDOORS",
    "Electronics & Technology": "ELECTRONICS",
    "Fashion & Apparel": "FASHION"
}

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
                print(f"  GraphQL error: {res['errors']}")
                return None
            return res.get("data")
    except Exception as e:
        print(f"  Exception: {e}")
        return None

# Brand pools for realistic seeding
BRANDS = {
    "Bags, Watches & Jewelry": ["Horizon Luxe", "Apex Luxury", "Vanguard", "Aura Premium", "Chrono Elite", "Heritage Rings", "Imperial Gold", "Solitaire & Co"],
    "Home & Living": ["Horizon Home", "EcoLiving", "Stellar Kitchen", "Nordic Comfort", "Apex Woodcraft", "Loom & Rug", "Aura Lighting"],
    "Beauty & Personal Care": ["Horizon Glow", "Aura Organics", "PureSkin", "Vocal Silk", "Nourish Botanicals", "Zenith Beauty"],
    "Groceries & Fresh Food": ["Fresh Farms", "Horizon Roasters", "Valley Harvest", "Organica", "ZWG Farms", "Zambesi Springs"],
    "Sports & Fitness": ["Horizon Fit", "Apex Outdoors", "Nomad Trail", "Summit Active", "Velo Sport", "Ranger Gear"],
    "Electronics & Technology": ["Apple", "Samsung", "Google", "Sony", "Dell", "HP", "Lenovo", "ASUS", "DJI", "Bose", "Sennheiser"],
    "Fashion & Apparel": ["Horizon Wear", "Denim Co.", "Urban Stitch", "Classic Fit", "Vanguard Leather", "Aura Chic", "Apex Athletics"]
}

def generate_products_for_subcat(root_cat, subcat_name):
    # Determine the ProductType enum value
    product_type = PRODUCT_TYPE_MAP.get(root_cat, "FASHION")
    
    # Specific override rules for subcategories
    if subcat_name == "Kitchen Appliances":
        product_type = "APPLIANCES"
    elif subcat_name in ["iPhones", "Android Phones", "Tablets & iPads"]:
        product_type = "MOBILE_PHONES_ACCESSORIES"
    elif subcat_name == "Soft Drinks & Juices":
        product_type = "BEVERAGES"
    elif subcat_name == "Cooking Oils":
        product_type = "LIQUIDS"

    # Select brand pool
    brand_pool = BRANDS.get(root_cat, ["Horizon"])
    
    # Select image
    if root_cat == "Bags, Watches & Jewelry":
        # Alternate between the user-provided bag images
        img_generator = lambda idx: BAG_IMAGES[idx % 2]
    else:
        # Use curated category specific Unsplash image
        img_url = CATEGORY_IMAGES.get(root_cat, {}).get(subcat_name, "https://images.unsplash.com/photo-1523275335684-37898b6baf30?auto=format&fit=crop&w=600&q=80")
        img_generator = lambda idx: img_url

    products = []
    
    # Let's generate 10 unique product variations based on subcategory name
    for i in range(1, 11):
        brand = random.choice(brand_pool)
        
        # Prices are set depending on the category to look extremely realistic
        if root_cat == "Groceries & Fresh Food":
            price = round(2.50 + i * 1.50 + random.random(), 2)
        elif root_cat == "Beauty & Personal Care":
            price = round(9.99 + i * 4.00 + random.random(), 2)
        elif root_cat == "Fashion & Apparel" or root_cat == "Bags, Watches & Jewelry":
            if "Suitcases" in subcat_name or "Mechanical" in subcat_name or "Rings" in subcat_name:
                price = round(89.00 + i * 25.00 + random.random(), 2)
            else:
                price = round(19.99 + i * 8.00 + random.random(), 2)
        elif root_cat == "Electronics & Technology":
            if subcat_name in ["iPhones", "Android Phones", "Laptops"]:
                price = round(299.00 + i * 120.00 + random.random(), 2)
            elif subcat_name in ["Drones", "DSLR & Mirrorless", "Computer Monitors"]:
                price = round(149.00 + i * 65.00 + random.random(), 2)
            else:
                price = round(29.99 + i * 15.00 + random.random(), 2)
        elif root_cat == "Home & Living":
            if subcat_name in ["Sofas & Living Room", "Beds & Bedroom", "Kitchen Appliances"]:
                price = round(120.00 + i * 80.00 + random.random(), 2)
            else:
                price = round(15.99 + i * 9.00 + random.random(), 2)
        else:
            price = round(15.00 + i * 10.00 + random.random(), 2)

        # Generate unique title & description
        title = f"{brand} {subcat_name[:-1] if subcat_name.endswith('s') and not subcat_name.endswith('ss') else subcat_name} Edition {i}"
        
        # Override titles for some known plural names/specific terms to read better
        if subcat_name == "iPhones":
            models = ["iPhone 15 Pro Max", "iPhone 15 Pro", "iPhone 15 Plus", "iPhone 15", "iPhone 14 Pro Max", "iPhone 14 Pro", "iPhone 14", "iPhone 13 Pro Max", "iPhone 13 Pro", "iPhone 13"]
            title = f"{models[(i-1)%len(models)]} ({128 * (1 + (i % 3))}GB)"
            brand = "Apple"
        elif subcat_name == "Android Phones":
            models = ["Galaxy S24 Ultra", "Galaxy S24+", "Galaxy S23 Ultra", "Pixel 8 Pro", "OnePlus 12", "Xiaomi 14 Ultra", "Galaxy Z Fold 5", "Pixel 8", "Galaxy A55", "Edge 50 Pro"]
            title = f"Samsung {models[(i-1)%len(models)]}" if "Galaxy" in models[(i-1)%len(models)] else f"{models[(i-1)%len(models)]}"
            if "Pixel" in title: brand = "Google"
            elif "OnePlus" in title: brand = "OnePlus"
            elif "Xiaomi" in title: brand = "Xiaomi"
            elif "Edge" in title: brand = "Motorola"
            else: brand = "Samsung"
        elif subcat_name == "Tablets & iPads":
            models = ["iPad Pro 12.9 (M2)", "iPad Air 10.9 (M1)", "iPad 10.9 (10th Gen)", "iPad Mini (6th Gen)", "Galaxy Tab S9 Ultra", "Galaxy Tab S9 FE", "Lenovo Tab P12", "Xiaomi Pad 6", "Fire HD 10", "OnePlus Pad"]
            title = f"{models[(i-1)%len(models)]}"
            if "iPad" in title: brand = "Apple"
            elif "Tab" in title: brand = "Samsung" if "Galaxy" in title else "Lenovo"
            elif "Pad" in title: brand = "Xiaomi" if "Xiaomi" in title else "OnePlus"
            else: brand = "Amazon"
        elif subcat_name == "Laptops":
            models = ["MacBook Pro 16", "MacBook Air 13 (M3)", "Dell XPS 13", "ThinkPad X1 Carbon", "ROG Zephyrus G14", "Spectre x360", "Swift Go 14", "Galaxy Book4 Pro", "Zenbook 14 OLED", "Razer Blade 16"]
            title = f"{models[(i-1)%len(models)]}"
            if "MacBook" in title: brand = "Apple"
            elif "XPS" in title: brand = "Dell"
            elif "ThinkPad" in title: brand = "Lenovo"
            elif "ROG" in title or "Zenbook" in title: brand = "ASUS"
            elif "Spectre" in title: brand = "HP"
            elif "Swift" in title: brand = "Acer"
            elif "Book4" in title: brand = "Samsung"
            else: brand = "Razer"
        elif subcat_name == "Drones":
            models = ["Mavic 3 Pro", "Mini 4 Pro", "Air 3", "EVO Lite+", "Avata Pro-View", "HS720G GPS", "F11GIM2 4K", "Atom 3-Axis", "Cetus Pro FPV", "Tello Mini"]
            title = f"{brand} {models[(i-1)%len(models)]}"
            if i <= 5: brand = "DJI"
        elif subcat_name == "Rings & Bands":
            styles = ["Platinum Wedding Band", "Diamond Engagement Ring", "Sterling Silver Signet", "Gold Eternity Band", "Emerald Cut Statement Ring", "Titanium Hammered Band", "Rose Gold Infinity Band", "Opal Cushion Ring", "Classic Carbon Band", "Sapphire Halo Ring"]
            title = f"{brand} {styles[(i-1)%10]}"
        elif subcat_name == "Necklaces & Chains":
            styles = ["18K Gold Link Chain", "South Sea Pearl Pendant", "Sterling Silver Bar", "Diamond Locket Choker", "Crystal Layered Statement", "Titanium Rope Chain", "Rose Gold Love Knot", "Emerald Solitaire Pendant", "Beaded Boho Set", "Sapphire Teardrop Collar"]
            title = f"{brand} {styles[(i-1)%10]}"
        elif subcat_name == "Earrings":
            styles = ["Diamond Princess Studs", "24K Gold Textured Hoops", "Silver Huggie Sleeper Set", "Tahitian Pearl Drops", "Emerald Tassel Dangles", "Minimalist Threader Hooks", "Rose Gold Leaf Studs", "Sapphire Climber Bars", "Turquoise Boho Hoops", "White Gold Halo Clusters"]
            title = f"{brand} {styles[(i-1)%10]}"
        elif subcat_name == "Handbags & Totes":
            styles = ["Classic Leather Shoulder Tote", "Elegant Chain Crossbody Bag", "Minimalist Vegan Shoulder Handbag", "Luxury Designer Satchel", "Casual Canvas Beach Tote", "Quilted Evening Clutch", "Vintage Pebbled Hobo Bag", "Sleek Executive Briefcase", "Studded Rocker Messenger", "Chic Suede Purse"]
            title = f"{brand} {styles[(i-1)%10]}"
        elif subcat_name == "Backpacks":
            styles = ["Urban Tech Laptop Pack", "Vanguard Anti-Theft Daypack", "Nomad Waterproof Hiking Pack", "Minimalist Commuter Bag", "Classic School Canvas Rucksack", "Ranger MOLLE Tactical Pack", "Slim Executive Briefcase Pack", "Kids Lightweight Dino Bag", "Roll-Top Commuter Backpack", "Active Gym Shoe Pocket Duffle"]
            title = f"{brand} {styles[(i-1)%10]}"
        elif subcat_name == "Suitcases & Travel Bags":
            styles = ["Hardshell Carry-On Spinner", "Premium Leather Travel Duffel", "Foldable Packing Cube Duffle", "Expandable Large Checked Suitcase", "Heavy-Duty Rolling Cargo Bag", "Underseat Cabin Trolley", "Waterproof Hanging Washbag", "Garment Suit Carrier", "Kids Ride-On Suitcase", "IPX6 Waterproof Marine Drybag"]
            title = f"{brand} {styles[(i-1)%10]}"
        elif subcat_name == "Smartwatches":
            styles = ["Fit Tracker Pro", "Active Sport GPS Watch", "Health Monitoring Band", "Vantage Connected Watch", "Pulse Waterproof Smartwatch", "Elite Titanium Smartwatch", "Chrono Health Tracker", "Rugged Outdoor GPS", "Graceful Jewel Smartwatch", "Neo Hybrid Wrist Watch"]
            title = f"{brand} {styles[(i-1)%10]}"
        elif subcat_name == "Mechanical & Quartz":
            styles = ["Skeleton Automatic mechanical", "Chronograph Quartz steel", "Vintage Leather mechanical", "Minimalist Classic Quartz", "Regal Dress automatic", "Titanium Sports Quartz", "Officer Pilot mechanical", "Rose Gold Monarch Quartz", "Heritage Luxury automatic", "Legacy Diver Quartz"]
            title = f"{brand} {styles[(i-1)%10]}"

        desc = f"Premium {title.lower()} designed by {brand}. This top-of-the-line product offers exceptional quality, durability, and style. Ideal for daily use or as a luxurious gift. Built using high-grade materials with careful attention to detail."

        products.append({
            "title": title,
            "desc": desc,
            "price": price,
            "brand": brand,
            "productType": product_type,
            "imageUrl": img_generator(i),
            "thumbnailUrl": img_generator(i)
        })
        
    return products

def main():
    print("=================================================================")
    print("  WeMall Seed Script — 10 Products Per Category & Subcategory    ")
    print("=================================================================")

    # 1. Authenticate Seller
    print("\n[1] Authenticating seller...")
    auth_res = gql("""
        mutation SellerFirebaseSignIn($idToken: String!, $fullName: String!) {
          sellerFirebaseSignIn(idToken: $idToken, fullName: $fullName) {
            accessToken
            user { id email role }
          }
        }
    """, {"idToken": f"mock-firebase-token-{SELLER_EMAIL}", "fullName": "Horizon Developers"})

    if not auth_res or not auth_res.get("sellerFirebaseSignIn"):
        print("  ✗ Seller authentication failed. Exiting.")
        sys.exit(1)
    seller_token = auth_res["sellerFirebaseSignIn"]["accessToken"]
    print(f"  ✓ Logged in as: {auth_res['sellerFirebaseSignIn']['user']['email']}")

    # 2. Check Store
    print("\n[2] Checking seller store...")
    my_store_data = gql("query { myStore { id storeName status } }", token=seller_token)
    if not my_store_data or not my_store_data.get("myStore"):
        print("  ✗ Store not found for authenticated seller. Exiting.")
        sys.exit(1)
    store = my_store_data["myStore"]
    print(f"  ✓ Store verified: {store['storeName']} (ID: {store['id']}, Status: {store['status']})")

    # 3. Fetch Category Tree
    print("\n[3] Fetching category tree...")
    query = """
    query {
      categories {
        id
        name
        slug
        children {
          id
          name
          slug
          children {
            id
            name
            slug
          }
        }
      }
    }
    """
    res = gql(query)
    if not res or not res.get("categories"):
        print("  ✗ Failed to retrieve categories. Exiting.")
        sys.exit(1)
    
    categories = res["categories"]
    print(f"  ✓ Fetched {len(categories)} root categories.")

    # Flatten the tree to find all level 3 (leaf) subcategories
    # A level 3 subcategory is a child of a level 2 category, which is a child of a level 1 root.
    leaf_subcategories = []
    
    for root in categories:
        root_name = root["name"]
        for level2 in root.get("children", []):
            for level3 in level2.get("children", []):
                leaf_subcategories.append({
                    "root_name": root_name,
                    "parent_name": level2["name"],
                    "id": level3["id"],
                    "name": level3["name"],
                    "slug": level3["slug"]
                })

    print(f"  ✓ Found {len(leaf_subcategories)} leaf subcategories to seed.")

    # 4. Create products
    print(f"\n[4] Seeding products (10 per subcategory)...")
    total_created = 0
    total_expected = len(leaf_subcategories) * 10
    start_time = time.time()

    # Define the product creation GraphQL mutation
    create_product_mutation = """
    mutation CreateProduct($input: CreateProductInput!) {
      createProduct(input: $input) {
        id
        title
      }
    }
    """

    for idx, subcat in enumerate(leaf_subcategories):
        print(f"\n▶ Seeding subcategory {idx+1}/{len(leaf_subcategories)}: '{subcat['name']}' (Under '{subcat['root_name']}')")
        
        # Generate the 10 products
        products_to_seed = generate_products_for_subcat(subcat["root_name"], subcat["name"])
        
        for p_idx, p in enumerate(products_to_seed):
            sku = f"SKU-{subcat['slug'][:10]}-{p_idx+1}-{int(time.time() * 1000) % 1000000}"
            
            variables = {
                "input": {
                    "categoryId": subcat["id"],
                    "title": p["title"],
                    "description": p["desc"],
                    "brand": p["brand"],
                    "productType": p["productType"],
                    "attributes": {},
                    "imageUrl": p["imageUrl"],
                    "thumbnailUrl": p["thumbnailUrl"],
                    "images": [p["imageUrl"]],
                    "variants": [{
                        "sku": sku,
                        "price": p["price"],
                        "options": {"standard": "Default"},
                        "initialQuantity": 100
                    }]
                }
            }

            res_prod = gql(create_product_mutation, variables=variables, token=seller_token)
            if res_prod and res_prod.get("createProduct"):
                total_created += 1
                print(f"  ✓ [{p_idx+1}/10] Created: {res_prod['createProduct']['title']} (${p['price']:.2f})")
            else:
                print(f"  ✗ [{p_idx+1}/10] FAILED to create: {p['title']}")
            
            # Tiny sleep to prevent slamming the server too hard
            time.sleep(0.02)

    elapsed = time.time() - start_time
    print("\n" + "=" * 60)
    print("  SEEDING SUMMARY")
    print("=" * 60)
    print(f"  Subcategories processed: {len(leaf_subcategories)}")
    print(f"  Total products created:  {total_created}/{total_expected}")
    print(f"  Time elapsed:            {elapsed:.2f} seconds")
    print("=" * 60)

if __name__ == "__main__":
    main()
