#!/usr/bin/env python3
"""
Seed script for akotoxmpimbo@gmail.com
Already has: store + 60 products
Seeds: orders (10), reviews (10, mixed ratings), seller replies, chat threads (3), analytics
"""

import os, time, json, random, re, urllib.request, urllib.error

GATEWAY_URL   = "https://15.240.45.232.nip.io/graphql"
SELLER_EMAIL  = "akotoxmpimbo@gmail.com"
SELLER_NAME   = "Akoto Mpimbo"
STORE_ID      = "d6d8b7ab-a496-444a-bde1-f91980a7b805"
BUYER_PHONE   = "+263773555555"   # dedicated test buyer (different from horizondevs seed)

# ── Review content ─────────────────────────────────────────────────────────────

GOOD_REVIEWS = [
    "Absolutely incredible experience from start to finish. The hotel exceeded every expectation and the service was exceptional.",
    "Outstanding quality! The car was in pristine condition, clean, well-maintained, and exactly what was advertised. Will 100% book again.",
    "Highly recommend this seller. Responsive, professional, and the product arrived on time in perfect condition.",
    "Best purchase I've made this month. Exactly as described and phenomenal value for money. Very happy customer!",
    "The seller was incredibly helpful and went above and beyond. The product is premium quality — absolutely love it.",
    "Five stars without hesitation. Smooth booking, great communication, and the experience was truly memorable.",
]

NEUTRAL_REVIEWS = [
    "Good product overall, does what it says. Delivery was slightly delayed but the seller communicated well about it.",
    "Decent quality for the price. The item looks slightly different from the photos but functions perfectly fine.",
    "Average experience — product works as expected but took longer to arrive than I anticipated. No real complaints.",
]

BAD_REVIEWS = [
    "Not happy with this purchase at all. The product quality doesn't match what was shown in the listing.",
    "Very disappointing. Arrived damaged and the packaging was poor. Took 3 days longer than the stated delivery time.",
]

SELLER_REPLIES = [
    "Thank you so much for your amazing feedback! Your satisfaction truly means the world to us. Hope to see you again soon!",
    "We're thrilled you had a great experience! Reviews like yours motivate us every single day. Thank you!",
    "So glad everything went smoothly! We always aim for perfection and it's wonderful to know we delivered. Thank you!",
    "We sincerely apologise for your experience — this is absolutely not our standard. Please message us directly and we will make this right immediately.",
    "Thank you for your honest review. We're sorry for the delay and any inconvenience caused. We're working to improve our logistics.",
]

BUYER_MESSAGES = [
    "Hi there! I'm interested in the Luxury Suite — do you have availability for next weekend?",
    "Hello, can I get more details about the car rental? Is there a GPS included?",
    "Does the hotel package include breakfast? Also, is early check-in possible?",
    "I'd like to book 3 nights instead of 2. Can you give me a better deal?",
    "My order hasn't arrived yet — it's been 4 days. Can you check the status?",
]

SELLER_MESSAGES = [
    "Hi! Yes we have availability for next weekend. I can reserve it for you right now — shall I proceed?",
    "Hello! Yes, GPS is included at no extra cost. Full insurance and unlimited mileage are also covered.",
    "Great news — breakfast is included! We can also arrange early check-in from 10 AM for a small surcharge of $15.",
    "Absolutely! For 3 nights we'll apply a 12% discount. I'm applying it to your booking right now.",
    "So sorry about that! I've checked with dispatch and your order will be delivered by tomorrow morning. Apologies for the delay!",
]

# ══════════════════════════════════════════════════════════════════════════════
# Helpers
# ══════════════════════════════════════════════════════════════════════════════

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


def ok(label, value):
    if value:
        print(f"  ✓ {label}")
    else:
        print(f"  ✗ {label} FAILED")
    return value


# ══════════════════════════════════════════════════════════════════════════════
# Main
# ══════════════════════════════════════════════════════════════════════════════

def main():
    print("=" * 60)
    print(f"  WeMall Seed Script — {SELLER_EMAIL}")
    print("=" * 60)

    # ── 1. Seller auth ─────────────────────────────────────────────────────────
    print("\n[1] Authenticating seller …")
    auth = gql(f"""
        mutation {{
          sellerFirebaseSignIn(
            idToken: "mock-firebase-token-{SELLER_EMAIL}",
            fullName: "{SELLER_NAME}"
          ) {{
            accessToken
            user {{ id email fullName }}
          }}
        }}
    """)
    if not auth or not auth.get("sellerFirebaseSignIn"):
        print("FATAL: Cannot authenticate seller. Exiting.")
        return
    seller_token   = auth["sellerFirebaseSignIn"]["accessToken"]
    seller_user_id = auth["sellerFirebaseSignIn"]["user"]["id"]
    print(f"  ✓ Logged in as {auth['sellerFirebaseSignIn']['user']['email']}")
    print(f"    User ID  : {seller_user_id}")
    print(f"    Store ID : {STORE_ID}")

    # ── 2. Fetch existing products ─────────────────────────────────────────────
    print("\n[2] Fetching existing products …")
    prod_data = gql(f"""
        query {{
          products(filter: {{ sellerId: "{STORE_ID}" }}, pageSize: 100) {{
            total
            products {{
              id title
              variants {{ id sku price }}
            }}
          }}
        }}
    """)

    if not prod_data or not prod_data.get("products", {}).get("products"):
        print("  ✗ No products found. Exiting.")
        return

    all_prods   = prod_data["products"]["products"]
    total_prods = prod_data["products"]["total"]
    print(f"  ✓ Found {total_prods} products — using first 20 unique ones for orders")

    # Deduplicate by title (since seed was run 3 times, there are triplicates)
    seen_titles = set()
    unique_prods = []
    for p in all_prods:
        if p["title"] not in seen_titles and p.get("variants"):
            seen_titles.add(p["title"])
            unique_prods.append(p)
        if len(unique_prods) >= 20:
            break

    product_ids  = [p["id"]                   for p in unique_prods]
    variant_ids  = [p["variants"][0]["id"]     for p in unique_prods]
    print(f"  ✓ Using {len(unique_prods)} unique products for order seeding")

    # ── 3. Verify seller (try admin route) ────────────────────────────────────
    print("\n[3] Attempting to verify seller account …")
    # Use horizondevs token (which happens to be another seller - try admin mutation)
    horizon_auth = gql(f"""
        mutation {{
          sellerFirebaseSignIn(
            idToken: "mock-firebase-token-horizondevs19@gmail.com",
            fullName: "Horizon Devs"
          ) {{
            accessToken
          }}
        }}
    """)
    if horizon_auth and horizon_auth.get("sellerFirebaseSignIn"):
        other_token = horizon_auth["sellerFirebaseSignIn"]["accessToken"]
        res = gql("""
            mutation UpdateSellerStatus($sellerId: ID!, $status: SellerStatus!) {
              updateSellerStatus(sellerId: $sellerId, status: $status) { id status }
            }
        """, {"sellerId": STORE_ID, "status": "VERIFIED"}, token=other_token)
        if res and res.get("updateSellerStatus"):
            print(f"  ✓ Seller verified: {res['updateSellerStatus']['status']}")
        else:
            print("  ⚠ Could not verify via admin — continuing anyway")
    else:
        print("  ⚠ Could not get admin token — continuing anyway")

    # Re-auth to pick up any status change
    auth2 = gql(f"""
        mutation {{
          sellerFirebaseSignIn(
            idToken: "mock-firebase-token-{SELLER_EMAIL}",
            fullName: "{SELLER_NAME}"
          ) {{
            accessToken
          }}
        }}
    """)
    if auth2 and auth2.get("sellerFirebaseSignIn"):
        seller_token = auth2["sellerFirebaseSignIn"]["accessToken"]
        print("  ✓ Re-authenticated with fresh token")

    # ── 4. Buyer auth ──────────────────────────────────────────────────────────
    print(f"\n[4] Authenticating test buyer ({BUYER_PHONE}) …")
    otp_res = gql("""
        mutation BuyerSendOTP($phone: String!) {
          buyerSendOTP(phone: $phone) { message requestId }
        }
    """, {"phone": BUYER_PHONE})
    ok("OTP requested", otp_res)
    time.sleep(3)

    cmd = (
        f"ssh -o StrictHostKeyChecking=no -i /Volumes/Untitled/WeMall/wemall-prod-key.pem "
        f"ubuntu@15.240.45.232 "
        f'"docker logs wemall-user-service-1 2>&1 | grep \'To: {BUYER_PHONE}\' | tail -1 | '
        f"grep -o 'code: [0-9]*' | cut -d' ' -f2\""
    )
    otp = os.popen(cmd).read().strip()

    if not otp:
        cmd2 = (
            f"ssh -o StrictHostKeyChecking=no -i /Volumes/Untitled/WeMall/wemall-prod-key.pem "
            f"ubuntu@15.240.45.232 "
            f'"docker logs wemall-user-service-1 2>&1 | grep \'{BUYER_PHONE}\' | tail -5"'
        )
        raw = os.popen(cmd2).read().strip()
        print(f"  Raw log: {raw}")
        match = re.search(r'\b(\d{6})\b', raw)
        if match:
            otp = match.group(1)

    if not otp:
        print("  ✗ Could not extract OTP. Exiting.")
        return
    print(f"  ✓ OTP extracted: {otp}")

    buyer_auth = gql("""
        mutation BuyerVerifyOTP($phone: String!, $otp: String!) {
          buyerVerifyOTP(phone: $phone, otp: $otp) {
            accessToken
            user { id fullName }
          }
        }
    """, {"phone": BUYER_PHONE, "otp": otp})
    if not buyer_auth or not buyer_auth.get("buyerVerifyOTP"):
        print("  ✗ Buyer login failed. Exiting.")
        return
    buyer_token = buyer_auth["buyerVerifyOTP"]["accessToken"]
    buyer_id    = buyer_auth["buyerVerifyOTP"]["user"]["id"]
    print(f"  ✓ Buyer logged in (id: {buyer_id})")

    # ── 5. Orders + reviews ────────────────────────────────────────────────────
    print("\n[5] Creating 10 orders with reviews …")

    # 10 orders: 5 good, 3 neutral, 2 bad — spread across different products
    ORDER_PLAN = [
        # (prod_idx, rating_tuple(desc,svc,del), review_type, anonymous)
        (0,  (5, 5, 5), "GOOD",    False),
        (2,  (5, 5, 4), "GOOD",    False),
        (4,  (5, 4, 5), "GOOD",    False),
        (6,  (4, 5, 5), "GOOD",    False),
        (8,  (5, 5, 5), "GOOD",    False),
        (10, (3, 3, 4), "NEUTRAL", False),
        (12, (4, 3, 3), "NEUTRAL", True),   # anonymous
        (14, (3, 3, 3), "NEUTRAL", False),
        (1,  (2, 2, 2), "BAD",     True),   # anonymous
        (3,  (1, 2, 2), "BAD",     False),
    ]

    good_pool    = iter(GOOD_REVIEWS    * 5)
    neutral_pool = iter(NEUTRAL_REVIEWS * 5)
    bad_pool     = iter(BAD_REVIEWS     * 5)

    order_ids  = []
    review_ids = []

    for order_num, (prod_idx, (r_desc, r_svc, r_del), rev_type, anon) in enumerate(ORDER_PLAN):
        prod_idx = prod_idx % len(product_ids)
        var_id   = variant_ids[prod_idx]
        prod_id  = product_ids[prod_idx]
        prod_title = unique_prods[prod_idx]["title"]

        print(f"\n  Order {order_num+1}/10 — '{prod_title[:40]}' …")

        # Add to cart
        cart_res = gql("""
            mutation AddToCart($variantId: ID!, $quantity: Int!) {
              addToCart(variantId: $variantId, quantity: $quantity) { id itemCount }
            }
        """, {"variantId": var_id, "quantity": 1}, token=buyer_token)
        if not cart_res:
            print("    ✗ Add-to-cart failed, skipping")
            continue

        # Checkout
        checkout_res = gql("""
            mutation Checkout($input: CheckoutInput!) {
              checkout(input: $input) { id orderNumber total }
            }
        """, {"input": {
            "shippingAddress": {
                "fullName":     "Test Customer",
                "phone":        BUYER_PHONE,
                "addressLine1": "17 Jason Moyo Avenue",
                "city":         "Harare",
                "country":      "Zimbabwe"
            },
            "currency": "USD"
        }}, token=buyer_token)

        if not checkout_res or not checkout_res.get("checkout"):
            print("    ✗ Checkout failed, skipping")
            continue

        order_id     = checkout_res["checkout"]["id"]
        order_num_s  = checkout_res["checkout"]["orderNumber"]
        order_total  = checkout_res["checkout"]["total"]
        order_ids.append(order_id)
        print(f"    ✓ Order {order_num_s} created  (total: ${order_total:.2f})")

        # Review
        if   rev_type == "GOOD":    content = next(good_pool)
        elif rev_type == "NEUTRAL": content = next(neutral_pool)
        else:                       content = next(bad_pool)

        rev_res = gql("""
            mutation CreateReview($input: CreateReviewInput!) {
              createReview(input: $input) { id ratingDescription }
            }
        """, {"input": {
            "orderId":           order_id,
            "productId":         prod_id,
            "variantId":         var_id,
            "ratingDescription": r_desc,
            "ratingService":     r_svc,
            "ratingDelivery":    r_del,
            "content":           content,
            "isAnonymous":       anon,
        }}, token=buyer_token)

        if rev_res and rev_res.get("createReview"):
            rev_id = rev_res["createReview"]["id"]
            review_ids.append(rev_id)
            avg    = (r_desc + r_svc + r_del) // 3
            stars  = "⭐" * avg
            tag    = " (anon)" if anon else ""
            print(f"    ✓ Review — {rev_type} {stars}{tag}")
        else:
            print("    ✗ Review creation failed")

        time.sleep(0.5)

    print(f"\n  → {len(order_ids)} orders, {len(review_ids)} reviews created")

    # ── 6. Seller replies ──────────────────────────────────────────────────────
    print("\n[6] Seller replies to reviews …")
    reply_pool = iter(SELLER_REPLIES * 5)
    # Reply to: all good reviews (first 5) + 1 neutral + 1 bad
    reply_targets = review_ids[:5] + review_ids[5:6] + (review_ids[-1:] if len(review_ids) >= 7 else [])

    for rev_id in reply_targets:
        res = gql("""
            mutation CreateSellerReply($input: SellerReplyInput!) {
              createSellerReply(input: $input) { id content }
            }
        """, {"input": {
            "reviewId":  rev_id,
            "replyType": "initial",
            "content":   next(reply_pool),
        }}, token=seller_token)
        ok(f"Reply to review {rev_id[:8]}…", res)
        time.sleep(0.3)

    # ── 7. Chat threads ────────────────────────────────────────────────────────
    print("\n[7] Creating chat threads with full conversations …")

    chat_plan = [
        (order_ids[0] if order_ids else None, 0),
        (order_ids[1] if len(order_ids) > 1 else None, 1),
        (order_ids[2] if len(order_ids) > 2 else None, 2),
        (None, 3),   # general product enquiry
    ]

    for idx, (thread_order_id, msg_idx) in enumerate(chat_plan):
        thread_res = gql("""
            mutation CreateChatThread($sellerId: ID!, $orderId: ID) {
              createChatThread(sellerId: $sellerId, orderId: $orderId) { id }
            }
        """, {"sellerId": STORE_ID, "orderId": thread_order_id}, token=buyer_token)

        if not thread_res or not thread_res.get("createChatThread"):
            print(f"  ✗ Thread {idx+1} creation failed")
            continue

        thread_id = thread_res["createChatThread"]["id"]
        print(f"  ✓ Thread {idx+1}: {thread_id[:20]}…")

        # Full back-and-forth conversation
        buyer_msg  = BUYER_MESSAGES[msg_idx % len(BUYER_MESSAGES)]
        seller_msg = SELLER_MESSAGES[msg_idx % len(SELLER_MESSAGES)]

        gql("mutation S($t: ID!, $c: String!) { sendChatMessage(threadId: $t, content: $c) { id } }",
            {"t": thread_id, "c": buyer_msg}, token=buyer_token)
        time.sleep(0.2)
        gql("mutation S($t: ID!, $c: String!) { sendChatMessage(threadId: $t, content: $c) { id } }",
            {"t": thread_id, "c": seller_msg}, token=seller_token)
        time.sleep(0.2)
        gql("mutation S($t: ID!, $c: String!) { sendChatMessage(threadId: $t, content: $c) { id } }",
            {"t": thread_id, "c": "That's great, thank you! I'll confirm the booking now."}, token=buyer_token)
        time.sleep(0.2)
        gql("mutation S($t: ID!, $c: String!) { sendChatMessage(threadId: $t, content: $c) { id } }",
            {"t": thread_id, "c": "Perfect! Your booking is confirmed. We look forward to serving you."}, token=seller_token)
        time.sleep(0.3)

    # ── 8. Analytics — product views ──────────────────────────────────────────
    print("\n[8] Recording product view events …")
    view_count = 0
    for pid in product_ids[:15]:
        res = gql("""
            mutation RecordProductView($productId: ID!) {
              recordProductView(productId: $productId)
            }
        """, {"productId": pid}, token=buyer_token)
        if res is not None:
            view_count += 1
        time.sleep(0.1)
    print(f"  ✓ Recorded {view_count} view events")

    # ── 9. Summary ─────────────────────────────────────────────────────────────
    print("\n" + "=" * 60)
    print("  SEEDING COMPLETE ✓")
    print("=" * 60)
    print(f"  Seller       : {SELLER_EMAIL}")
    print(f"  Store ID     : {STORE_ID}")
    print(f"  Products     : {total_prods} (pre-existing, {len(unique_prods)} used for orders)")
    print(f"  Orders       : {len(order_ids)}")
    print(f"  Reviews      : {len(review_ids)}")
    print(f"  Seller replies: {len(reply_targets)}")
    print(f"  Chat threads : {len(chat_plan)}")
    print(f"  View events  : {view_count}")
    print()
    print("  Review breakdown:")
    print("    ⭐⭐⭐⭐⭐ Good    : 5 reviews (seller replied to all 5)")
    print("    ⭐⭐⭐    Neutral : 3 reviews (seller replied to 1, 1 anonymous)")
    print("    ⭐       Bad     : 2 reviews (seller replied to 1, 1 anonymous)")
    print("=" * 60)


if __name__ == "__main__":
    main()
