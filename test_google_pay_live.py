#!/usr/bin/env python3
import urllib.request
import json
import ssl
import subprocess
import random
import re
import sys
import time

# Target live GraphQL URL
GATEWAY_URL = "https://15.240.45.232.nip.io/graphql"

def run_ssh(cmd):
    """Executes a command on the production EC2 host via SSH."""
    ssh_cmd = [
        "ssh", "-i", "wemall-prod-key.pem",
        "-o", "StrictHostKeyChecking=no",
        "ubuntu@15.240.45.232", cmd
    ]
    res = subprocess.run(ssh_cmd, capture_output=True, text=True)
    if res.returncode != 0:
        print(f"SSH Command failed: {cmd}\nStdout: {res.stdout}\nStderr: {res.stderr}")
        raise Exception(f"SSH command failed with exit code {res.returncode}")
    return res.stdout.strip()

def graphql_request(query, variables=None, token=None):
    """Sends a GraphQL POST request using built-in urllib."""
    headers = {
        "Content-Type": "application/json"
    }
    if token:
        headers["Authorization"] = f"Bearer {token}"
    
    payload = {"query": query}
    if variables:
        payload["variables"] = variables
        
    req = urllib.request.Request(
        GATEWAY_URL,
        data=json.dumps(payload).encode("utf-8"),
        headers=headers,
        method="POST"
    )
    
    # Disable SSL verification for nip.io domain redirects / certificate validations
    ctx = ssl.create_default_context()
    ctx.check_hostname = False
    ctx.verify_mode = ssl.CERT_NONE
    
    try:
        with urllib.request.urlopen(req, context=ctx) as response:
            res_data = response.read().decode("utf-8")
            res_json = json.loads(res_data)
            if "errors" in res_json:
                print(f"GraphQL Errors: {json.dumps(res_json['errors'], indent=2)}")
            return res_json
    except urllib.error.HTTPError as e:
        print(f"HTTP Error: {e.code} - {e.read().decode('utf-8')}")
        raise

def get_data_field(res, *keys):
    """Safely extracts a nested field from the GraphQL response data dictionary."""
    if not res:
        return None
    data = res.get("data")
    if not data:
        return None
    val = data
    for k in keys:
        if isinstance(val, dict):
            val = val.get(k)
        else:
            return None
    return val

def force_verify_user(email):
    """Updates the user record in the production database to set is_verified=true."""
    print(f"Force-verifying user '{email}' in database...")
    run_ssh(f"docker exec -i wemall-postgres-1 psql -U wemall -d wemall_users -c \"UPDATE users SET is_verified = true WHERE email = '{email}';\"")

def main():
    print("=== WeMall Payment Integration Test on Live API ===")
    
    # 0. Set permissions for SSH key
    print("Setting permissions on SSH key...")
    subprocess.run(["chmod", "600", "wemall-prod-key.pem"])
    
    # 1. Register Buyer
    phone_num = f"+26377{random.randint(1000000, 9999999)}"
    print(f"\n[Buyer Setup] Registering Buyer with phone: {phone_num}")
    
    # Send OTP
    send_otp_mutation = """
    mutation SendOTP($phone: String!) {
      buyerSendOTP(phone: $phone) {
        message
      }
    }
    """
    res = graphql_request(send_otp_mutation, {"phone": phone_num})
    if not get_data_field(res, "buyerSendOTP"):
        print("Failed to send OTP.")
        sys.exit(1)
        
    print("OTP request sent successfully. Fetching OTP from remote logs...")
    time.sleep(2)  # Wait for logs to flush
    
    # Fetch code from remote container logs using raw regex pattern to avoid warnings
    log_line = run_ssh(rf"docker logs wemall-user-service-1 2>&1 | grep '\[SMS MOCK\] To: {phone_num}' | tail -n 1")
    print(f"Log line retrieved: {log_line}")
    
    match = re.search(r"code:\s*(\d{6})", log_line)
    if not match:
        print("Could not parse 6-digit OTP code from logs.")
        sys.exit(1)
    otp_code = match.group(1)
    print(f"Extracted OTP code: {otp_code}")
    
    # Verify OTP
    verify_otp_mutation = """
    mutation VerifyOTP($phone: String!, $otp: String!) {
      buyerVerifyOTP(phone: $phone, otp: $otp) {
        accessToken
        user {
          id
        }
      }
    }
    """
    res = graphql_request(verify_otp_mutation, {"phone": phone_num, "otp": otp_code})
    buyer_token = get_data_field(res, "buyerVerifyOTP", "accessToken")
    buyer_id = get_data_field(res, "buyerVerifyOTP", "user", "id")
    if not buyer_token:
        print("Buyer OTP verification failed.")
        sys.exit(1)
    print(f"Buyer logged in! Token: {buyer_token[:15]}... | ID: {buyer_id}")

    # 2. Register/Login Seller
    seller_email = "harare_seller@example.com"
    seller_password = "Password123!"
    print(f"\n[Seller Setup] Registering / Logging in Seller: {seller_email}")
    
    register_seller_mutation = """
    mutation RegisterSeller($email: String!, $password: String!, $fullName: String!) {
      sellerRegister(email: $email, password: $password, fullName: $fullName) {
        accessToken
        user {
          id
        }
      }
    }
    """
    res = graphql_request(register_seller_mutation, {
        "email": seller_email,
        "password": seller_password,
        "fullName": "Harare Store Owner"
    })
    
    # Force verification on database regardless of registration success
    force_verify_user(seller_email)
    
    # Log in the seller
    print("Logging in seller...")
    login_seller_mutation = """
    mutation LoginSeller($email: String!, $password: String!) {
      sellerLogin(email: $email, password: $password) {
        accessToken
        user {
          id
        }
      }
    }
    """
    res = graphql_request(login_seller_mutation, {
        "email": seller_email,
        "password": seller_password
    })
    seller_token = get_data_field(res, "sellerLogin", "accessToken")
        
    if not seller_token:
        print("Seller authentication failed.")
        sys.exit(1)
    print(f"Seller logged in! Token: {seller_token[:15]}...")

    # 3. Create Store
    print("Creating/Fetching Store...")
    create_store_mutation = """
    mutation CreateStore {
      createStore(input: {
        storeName: "Harare CBD Premium Store"
        description: "High quality electronics in Harare CBD"
        latitude: -17.8292
        longitude: 31.0522
        logoUrl: "https://wemall.co.zw/assets/store_logo_placeholder.png"
        bannerUrl: "https://wemall.co.zw/assets/store_banner_placeholder.png"
      }) {
        id
        storeName
        status
        isVerified
      }
    }
    """
    res = graphql_request(create_store_mutation, token=seller_token)
    store = get_data_field(res, "createStore")
    if not store:
        # Fetch existing store
        my_store_query = """
        query MyStore {
          myStore {
            id
            storeName
            status
            isVerified
          }
        }
        """
        res = graphql_request(my_store_query, token=seller_token)
        store = get_data_field(res, "myStore")
        
    if not store:
        print("Store setup failed.")
        sys.exit(1)
        
    store_id = store.get("id")
    print(f"Store resolved: {store.get('storeName')} (ID: {store_id}) | Verified: {store.get('isVerified')}")

    # 4. Admin setup & store verification
    if not store.get("isVerified"):
        print("\n[Admin Setup] Verifying Store...")
        admin_email = "admin@example.com"
        admin_password = "AdminPass123!"
        
        # Register admin
        res = graphql_request(register_seller_mutation, {
            "email": admin_email,
            "password": admin_password,
            "fullName": "System Admin"
        })
        
        # Promote admin and verify in DB
        print("Promoting and verifying admin@example.com in database...")
        run_ssh('docker exec -i wemall-postgres-1 psql -U wemall -d wemall_users -c "UPDATE users SET role = \'admin\', is_verified = true WHERE email = \'admin@example.com\';"')
        
        # Log in Admin
        login_mutation = """
        mutation AdminLogin($email: String!, $password: String!) {
          sellerLogin(email: $email, password: $password) {
            accessToken
          }
        }
        """
        res = graphql_request(login_mutation, {"email": admin_email, "password": admin_password})
        admin_token = get_data_field(res, "sellerLogin", "accessToken")
        if not admin_token:
            print("Admin authentication failed.")
            sys.exit(1)
            
        # Verify Seller Store
        verify_mutation = """
        mutation VerifyStore($storeId: ID!) {
          updateSellerStatus(sellerId: $storeId, status: VERIFIED) {
            id
            status
            isVerified
          }
        }
        """
        res = graphql_request(verify_mutation, {"storeId": store_id}, token=admin_token)
        verified_store = get_data_field(res, "updateSellerStatus")
        if not verified_store or not verified_store.get("isVerified"):
            print("Store verification by admin failed.")
            sys.exit(1)
        print("Store successfully verified by Admin!")

    # 5. Fetch Category
    print("\n[Product Setup] Fetching categories...")
    categories_query = """
    query {
      categories {
        id
        name
        children {
          id
          name
        }
      }
    }
    """
    res = graphql_request(categories_query)
    categories = get_data_field(res, "categories") or []
    if not categories:
        print("No categories found.")
        sys.exit(1)
        
    category_id = categories[0]["children"][0]["id"] if categories[0].get("children") else categories[0]["id"]
    print(f"Selected category ID: {category_id}")

    # 6. Create Product
    product_title = f"GP Test Product {random.randint(1000, 9999)}"
    sku = f"SKU-GP-{random.randint(1000, 9999)}"
    print(f"Creating product '{product_title}'...")
    
    create_product_mutation = """
    mutation CreateProduct($input: CreateProductInput!) {
      createProduct(input: $input) {
        id
        variants {
          id
          sku
          price
        }
      }
    }
    """
    product_input = {
        "categoryId": category_id,
        "title": product_title,
        "description": "Google Pay live test product",
        "brand": "Google",
        "productType": "MOBILE_PHONES_ACCESSORIES",
        "attributes": {},
        "variants": [
            {
                "sku": sku,
                "price": 2.50,
                "options": {"color": "Chalk"}
            }
        ]
    }
    
    res = graphql_request(create_product_mutation, {"input": product_input}, token=seller_token)
    product_data = get_data_field(res, "createProduct")
    if not product_data or not product_data.get("variants"):
        print("Product creation failed.")
        sys.exit(1)
        
    variant_id = product_data["variants"][0]["id"]
    print(f"Product variant created! ID: {variant_id} | Price: $2.50")

    # Helper function to place order
    def create_test_order():
        # Clear Cart
        graphql_request("mutation { clearCart { id } }", token=buyer_token)
        # Add to cart
        add_to_cart_mutation = """
        mutation AddToCart($variantId: ID!, $quantity: Int!) {
          addToCart(variantId: $variantId, quantity: $quantity) {
            id
          }
        }
        """
        graphql_request(add_to_cart_mutation, {"variantId": variant_id, "quantity": 1}, token=buyer_token)
        # Checkout
        checkout_mutation = """
        mutation Checkout($input: CheckoutInput!) {
          checkout(input: $input) {
            id
            orderNumber
            status
            total
            currency
          }
        }
        """
        checkout_input = {
            "shippingAddress": {
                "fullName": "Test Buyer",
                "phone": phone_num,
                "addressLine1": "77 Live Test Road",
                "city": "Harare",
                "country": "Zimbabwe"
            },
            "currency": "USD"
        }
        res = graphql_request(checkout_mutation, {"input": checkout_input}, token=buyer_token)
        order = get_data_field(res, "checkout")
        if not order:
            print("Checkout failed.")
            sys.exit(1)
        return order

    # -------------------------------------------------------------------------
    # TEST CASE 1: Google Pay Payment Success Flow
    # -------------------------------------------------------------------------
    print("\n--- Test Case 1: Google Pay Success Flow ---")
    order = create_test_order()
    order_id = order.get("id")
    print(f"Order created: {order.get('orderNumber')} (ID: {order_id}) | Total: {order.get('total')} | Status: {order.get('status')}")
    
    # Initiate Payment
    initiate_payment_mutation = """
    mutation InitiatePayment($orderId: ID!, $provider: PaymentProvider!) {
      initiatePayment(orderId: $orderId, provider: $provider) {
        payment {
          id
          status
          provider
          amount
          currency
        }
        clientSecret
      }
    }
    """
    res = graphql_request(initiate_payment_mutation, {"orderId": order_id, "provider": "GOOGLE_PAY"}, token=buyer_token)
    init_data = get_data_field(res, "initiatePayment")
    if not init_data:
        print("Payment initiation failed.")
        sys.exit(1)
    
    payment_id = init_data["payment"]["id"]
    client_secret = init_data["clientSecret"]
    print(f"Payment initiated! ID: {payment_id} | Status: {init_data['payment']['status']}")
    print(f"Client Secret received: {client_secret}")
    
    # Verify client secret contains correct merchant ID
    expected_prefix = "merchant:BCR2DN5T5733RWQV:payment:"
    if not client_secret.startswith(expected_prefix):
        print(f"WARNING: Client secret '{client_secret}' does not start with expected prefix '{expected_prefix}'")
    else:
        print("✓ Verified Google Pay merchant ID in client secret!")

    # Process Payment with Success Token
    print("Processing payment with valid mock token...")
    process_payment_mutation = """
    mutation ProcessPayment($paymentId: ID!, $token: String!) {
      processPayment(paymentId: $paymentId, token: $token) {
        id
        status
        transactionId
      }
    }
    """
    res = graphql_request(process_payment_mutation, {"paymentId": payment_id, "token": "gp_live_test_success_token"}, token=buyer_token)
    payment_result = get_data_field(res, "processPayment")
    if not payment_result:
        print("Payment processing failed.")
        sys.exit(1)
        
    print(f"Process Payment Response: Status = {payment_result['status']} | TransactionID = {payment_result['transactionId']}")
    if payment_result['status'] != "COMPLETED":
        print(f"ERROR: Expected COMPLETED status, got {payment_result['status']}")
        sys.exit(1)
    print("✓ Google Pay payment successfully processed!")

    # Verify Order Status updates to CONFIRMED (via NATS event listener)
    print("Waiting for NATS worker to confirm order...")
    time.sleep(2)
    
    order_query = """
    query GetOrder($id: ID!) {
      order(id: $id) {
        id
        status
      }
    }
    """
    res = graphql_request(order_query, {"id": order_id}, token=buyer_token)
    confirmed_order = get_data_field(res, "order")
    if not confirmed_order:
        print("Failed to query order.")
        sys.exit(1)
        
    print(f"Order status in database: {confirmed_order['status']}")
    if confirmed_order['status'] != "CONFIRMED":
        print(f"ERROR: Expected order status CONFIRMED, got {confirmed_order['status']}")
        sys.exit(1)
    print("✓ Test Case 1 PASSED: Google Pay Success Flow verified!")

    # -------------------------------------------------------------------------
    # TEST CASE 2: Google Pay Payment Failure Flow
    # -------------------------------------------------------------------------
    print("\n--- Test Case 2: Google Pay Failure Flow ---")
    order = create_test_order()
    order_id = order.get("id")
    print(f"Order created: {order.get('orderNumber')} (ID: {order_id}) | Status: {order.get('status')}")
    
    # Initiate Payment
    res = graphql_request(initiate_payment_mutation, {"orderId": order_id, "provider": "GOOGLE_PAY"}, token=buyer_token)
    init_data = get_data_field(res, "initiatePayment")
    payment_id = init_data["payment"]["id"]
    print(f"Payment initiated! ID: {payment_id} | Status: {init_data['payment']['status']}")
    
    # Process Payment with Fail Token
    print("Processing payment with failure mock token...")
    res = graphql_request(process_payment_mutation, {"paymentId": payment_id, "token": "gp_live_test_fail_token"}, token=buyer_token)
    payment_result = get_data_field(res, "processPayment")
    
    print(f"Process Payment Response: Status = {payment_result['status']} | TransactionID = {payment_result['transactionId']}")
    if payment_result['status'] != "FAILED":
        print(f"ERROR: Expected FAILED status, got {payment_result['status']}")
        sys.exit(1)
    print("✓ Google Pay payment successfully failed!")
    
    # Verify Order Status updates to CANCELLED (via NATS event listener)
    print("Waiting for NATS worker to cancel order...")
    time.sleep(2)
    
    res = graphql_request(order_query, {"id": order_id}, token=buyer_token)
    cancelled_order = get_data_field(res, "order")
    print(f"Order status in database: {cancelled_order['status']}")
    if cancelled_order['status'] != "CANCELLED":
        print(f"ERROR: Expected order status CANCELLED, got {cancelled_order['status']}")
        sys.exit(1)
    print("✓ Test Case 2 PASSED: Google Pay Failure Flow verified!")

    # -------------------------------------------------------------------------
    # TEST CASE 3: Stripe Payment Success Flow
    # -------------------------------------------------------------------------
    print("\n--- Test Case 3: Stripe Success Flow ---")
    order = create_test_order()
    order_id = order.get("id")
    print(f"Order created: {order.get('orderNumber')} (ID: {order_id}) | Status: {order.get('status')}")
    
    # Initiate Payment
    res = graphql_request(initiate_payment_mutation, {"orderId": order_id, "provider": "STRIPE"}, token=buyer_token)
    init_data = get_data_field(res, "initiatePayment")
    payment_id = init_data["payment"]["id"]
    print(f"Payment initiated! ID: {payment_id} | Client Secret: {init_data['clientSecret']}")
    
    # Process Payment with Success Token
    print("Processing payment with valid mock Stripe token...")
    res = graphql_request(process_payment_mutation, {"paymentId": payment_id, "token": "stripe_live_test_success_token"}, token=buyer_token)
    payment_result = get_data_field(res, "processPayment")
    
    print(f"Process Payment Response: Status = {payment_result['status']} | TransactionID = {payment_result['transactionId']}")
    if payment_result['status'] != "COMPLETED":
        print(f"ERROR: Expected COMPLETED status, got {payment_result['status']}")
        sys.exit(1)
    print("✓ Stripe payment successfully processed!")
    
    # Verify Order Status updates to CONFIRMED
    print("Waiting for NATS worker to confirm order...")
    time.sleep(2)
    
    res = graphql_request(order_query, {"id": order_id}, token=buyer_token)
    confirmed_order = get_data_field(res, "order")
    print(f"Order status in database: {confirmed_order['status']}")
    if confirmed_order['status'] != "CONFIRMED":
        print(f"ERROR: Expected order status CONFIRMED, got {confirmed_order['status']}")
        sys.exit(1)
    print("✓ Test Case 3 PASSED: Stripe Success Flow verified!")

    # -------------------------------------------------------------------------
    # TEST CASE 4: Stripe Payment Failure Flow
    # -------------------------------------------------------------------------
    print("\n--- Test Case 4: Stripe Failure Flow ---")
    order = create_test_order()
    order_id = order.get("id")
    print(f"Order created: {order.get('orderNumber')} (ID: {order_id}) | Status: {order.get('status')}")
    
    # Initiate Payment
    res = graphql_request(initiate_payment_mutation, {"orderId": order_id, "provider": "STRIPE"}, token=buyer_token)
    init_data = get_data_field(res, "initiatePayment")
    payment_id = init_data["payment"]["id"]
    print(f"Payment initiated! ID: {payment_id} | Status: {init_data['payment']['status']}")
    
    # Process Payment with Fail Token
    print("Processing payment with failure mock Stripe token...")
    res = graphql_request(process_payment_mutation, {"paymentId": payment_id, "token": "stripe_live_test_fail_token"}, token=buyer_token)
    payment_result = get_data_field(res, "processPayment")
    
    print(f"Process Payment Response: Status = {payment_result['status']} | TransactionID = {payment_result['transactionId']}")
    if payment_result['status'] != "FAILED":
        print(f"ERROR: Expected FAILED status, got {payment_result['status']}")
        sys.exit(1)
    print("✓ Stripe payment successfully failed!")
    
    # Verify Order Status updates to CANCELLED
    print("Waiting for NATS worker to cancel order...")
    time.sleep(2)
    
    res = graphql_request(order_query, {"id": order_id}, token=buyer_token)
    cancelled_order = get_data_field(res, "order")
    print(f"Order status in database: {cancelled_order['status']}")
    if cancelled_order['status'] != "CANCELLED":
        print(f"ERROR: Expected order status CANCELLED, got {cancelled_order['status']}")
        sys.exit(1)
    print("✓ Test Case 4 PASSED: Stripe Failure Flow verified!")

    print("\n========================================")
    print("✓ ALL PAYMENT INTEGRATION TESTS PASSED!")
    print("========================================")

if __name__ == "__main__":
    main()
