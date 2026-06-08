# WeMall Services Endpoints Documentation

All public APIs are exposed through the single GraphQL Gateway endpoint:
`http://localhost:8080/graphql`

---

## 1. Authentication & User Profile Endpoints

### Mutations

#### A. Buyer Google Authentication
Authenticates a buyer using a Google OAuth authorization code.
* **Mutation**:
  ```graphql
  mutation BuyerGoogleAuth($code: String!, $redirectUri: String) {
    buyerGoogleAuth(code: $code, redirectUri: $redirectUri) {
      accessToken
      refreshToken
      user { id email fullName role }
    }
  }
  ```

#### B. Buyer Send OTP
Sends a one-time verification password to a buyer's mobile number.
* **Mutation**:
  ```graphql
  mutation BuyerSendOTP($phone: String!) {
    buyerSendOTP(phone: $phone) {
      message
      requestId
    }
  }
  ```

#### C. Buyer Verify OTP
Verifies the OTP sent to the buyer's phone and returns a session JWT.
* **Mutation**:
  ```graphql
  mutation BuyerVerifyOTP($phone: String!, $otp: String!) {
    buyerVerifyOTP(phone: $phone, otp: $otp) {
      accessToken
      refreshToken
      user { id email fullName role }
    }
  }
  ```

#### D. Seller Registration
Registers a new store owner/seller account.
* **Mutation**:
  ```graphql
  mutation SellerRegister($email: String!, $password: String!, $fullName: String!) {
    sellerRegister(email: $email, password: $password, fullName: $fullName) {
      accessToken
      refreshToken
      user { id email fullName role }
    }
  }
  ```

#### E. Seller Login
Authenticates an existing seller using credentials.
* **Mutation**:
  ```graphql
  mutation SellerLogin($email: String!, $password: String!) {
    sellerLogin(email: $email, password: $password) {
      accessToken
      refreshToken
      user { id email fullName role }
    }
  }
  ```

#### F. Refresh Token
Exchanges a valid refresh token for a new short-lived access token.
* **Mutation**:
  ```graphql
  mutation RefreshToken($refreshToken: String!) {
    refreshToken(refreshToken: $refreshToken) {
      accessToken
      refreshToken
    }
  }
  ```

#### G. Update Profile
Updates user profile information (Requires `BUYER` role).
* **Mutation**:
  ```graphql
  mutation UpdateProfile($fullName: String, $avatarUrl: String) {
    updateProfile(fullName: $fullName, avatarUrl: $avatarUrl) {
      id
      fullName
      avatarUrl
    }
  }
  ```

#### H. Create Address
Adds a shipping/delivery address for the authenticated buyer (Requires `BUYER` role).
* **Mutation**:
  ```graphql
  mutation CreateAddress($input: AddressInput!) {
    createAddress(input: $input) {
      id
      label
      fullName
      phone
      addressLine1
      city
      country
      isDefault
    }
  }
  ```

#### I. Delete Address
Removes a buyer's saved address (Requires `BUYER` role).
* **Mutation**:
  ```graphql
  mutation DeleteAddress($addressId: ID!) {
    deleteAddress(addressId: $addressId)
  }
  ```

### Queries

#### A. Fetch Current User (`me`)
Returns current authenticated user details (Requires `BUYER` role).
* **Query**:
  ```graphql
  query {
    me {
      id
      email
      phone
      fullName
      role
      isVerified
    }
  }
  ```

#### B. Fetch User Address Book
Lists all shipping addresses for the logged-in buyer (Requires `BUYER` role).
* **Query**:
  ```graphql
  query {
    addresses {
      id
      label
      fullName
      addressLine1
      city
      country
      isDefault
    }
  }
  ```

---

## 2. Product Catalog & Store Endpoints

### Queries

#### A. Fetch Categories
Lists top-level categories and nested sub-categories.
* **Query**:
  ```graphql
  query GetCategories($language: String) {
    categories(language: $language) {
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
  ```

#### B. Fetch Category Details
Retrieves details of a category by slug.
* **Query**:
  ```graphql
  query GetCategory($slug: String!) {
    category(slug: $slug) {
      id
      name
      slug
      attributeSchema
    }
  }
  ```

#### C. Search & Filter Products
Queries products using full-text search, categories, seller filters, and ratings.
* **Query**:
  ```graphql
  query GetProducts($filter: ProductFilterInput, $pageSize: Int, $pageToken: String) {
    products(filter: $filter, pageSize: $pageSize, pageToken: $pageToken) {
      products {
        id
        title
        brand
        status
        minPrice
        variants { id sku price }
      }
      nextPageToken
      total
    }
  }
  ```

#### D. Fetch Product Detail
Fetches full details of a specific product by ID or slug.
* **Query**:
  ```graphql
  query GetProductDetail($id: ID, $slug: String) {
    product(id: $id, slug: $slug) {
      id
      title
      description
      brand
      rating
      reviewCount
      soldCount
      variants { id sku price options }
      images { url isPrimary }
      seller { id storeName }
    }
  }
  ```

#### E. Find Nearby Products
Returns products within a specific radius of coordinates (Geo-location).
* **Query**:
  ```graphql
  query NearbyProducts($latitude: Float!, $longitude: Float!, $radiusMeters: Float!) {
    nearbyProducts(latitude: $latitude, longitude: $longitude, radiusMeters: $radiusMeters) {
      product {
        id
        title
      }
      distance
    }
  }
  ```

#### F. Recommended Products
Fetches personalized recommended products for the user.
* **Query**:
  ```graphql
  query RecommendedProducts($pageSize: Int, $pageToken: String) {
    recommendedProducts(pageSize: $pageSize, pageToken: $pageToken) {
      products { id title minPrice }
    }
  }
  ```

### Mutations (Sellers / Admin)

#### A. Create Store
Allows an authenticated seller to create their storefront (Requires `SELLER` role).
* **Mutation**:
  ```graphql
  mutation CreateStore($input: CreateStoreInput!) {
    createStore(input: $input) {
      id
      storeName
      storeSlug
      status
    }
  }
  ```

#### B. Create Product
Adds a new product to the catalog (Requires `SELLER` role).
* **Mutation**:
  ```graphql
  mutation CreateProduct($input: CreateProductInput!) {
    createProduct(input: $input) {
      id
      title
      variants {
        id
        sku
        price
      }
    }
  }
  ```

#### C. Update Product
Modifies product attributes, status, and information (Requires `SELLER` role).
* **Mutation**:
  ```graphql
  mutation UpdateProduct($id: ID!, $input: UpdateProductInput!) {
    updateProduct(id: $id, input: $input) {
      id
      title
      status
    }
  }
  ```

#### D. Delete Product
Removes a product from the catalog (Requires `SELLER` role).
* **Mutation**:
  ```graphql
  mutation DeleteProduct($id: ID!) {
    deleteProduct(id: $id)
  }
  ```

---

## 3. Cart & Order Endpoints

### Queries

#### A. Fetch Cart
Gets items, counts, and subtotal for the active shopping cart (Requires `BUYER` role).
* **Query**:
  ```graphql
  query {
    cart {
      id
      items {
        id
        variantId
        productTitle
        quantity
        unitPrice
      }
      itemCount
      subtotal
    }
  }
  ```

#### B. Fetch Order Details
Retrieves details of a specific order by ID (Requires `BUYER` role).
* **Query**:
  ```graphql
  query GetOrder($id: ID!) {
    order(id: $id) {
      id
      orderNumber
      status
      total
      shippingAddress
      items {
        productId
        productTitle
        quantity
        unitPrice
      }
    }
  }
  ```

#### C. List Orders
Queries past orders for the buyer (Requires `BUYER` role).
* **Query**:
  ```graphql
  query GetOrders($pageSize: Int, $pageToken: String) {
    orders(pageSize: $pageSize, pageToken: $pageToken) {
      orders {
        id
        orderNumber
        status
        total
        createdAt
      }
    }
  }
  ```

### Mutations

#### A. Add to Cart
Adds a product variant to the buyer's cart (Requires `BUYER` role).
* **Mutation**:
  ```graphql
  mutation AddToCart($variantId: ID!, $quantity: Int!) {
    addToCart(variantId: $variantId, quantity: $quantity) {
      id
      itemCount
      subtotal
    }
  }
  ```

#### B. Update Cart Item
Changes the quantity of an item already in the cart (Requires `BUYER` role).
* **Mutation**:
  ```graphql
  mutation UpdateCartItem($itemId: ID!, $quantity: Int!) {
    updateCartItem(itemId: $itemId, quantity: $quantity) {
      id
      itemCount
      subtotal
    }
  }
  ```

#### C. Remove Cart Item
Removes a specific item from the cart (Requires `BUYER` role).
* **Mutation**:
  ```graphql
  mutation RemoveCartItem($itemId: ID!) {
    removeCartItem(itemId: $itemId) {
      id
      subtotal
    }
  }
  ```

#### D. Clear Cart
Removes all items from the active cart (Requires `BUYER` role).
* **Mutation**:
  ```graphql
  mutation {
    clearCart {
      id
      itemCount
      subtotal
    }
  }
  ```

#### E. Checkout Order
Places an order for items in the cart (Requires `BUYER` role).
* **Mutation**:
  ```graphql
  mutation Checkout($input: CheckoutInput!) {
    checkout(input: $input) {
      id
      orderNumber
      status
      total
    }
  }
  ```

#### F. Cancel Order
Cancels a pending order (Requires `BUYER` role).
* **Mutation**:
  ```graphql
  mutation CancelOrder($id: ID!) {
    cancelOrder(id: $id) {
      id
      status
    }
  }
  ```

---

## 4. Payment Endpoints

### Queries

#### A. Get Payment Detail
Fetches payment record details (Requires `BUYER` role).
* **Query**:
  ```graphql
  query GetPayment($id: ID!) {
    payment(id: $id) {
      id
      orderId
      amount
      currency
      provider
      status
      transactionId
    }
  }
  ```

### Mutations

#### A. Initiate Payment
Prepares a payment record and generates provider-specific parameters (Stripe `client_secret` or Google Pay merchant options) (Requires `BUYER` role).
* **Mutation**:
  ```graphql
  mutation InitiatePayment($orderId: ID!, $provider: PaymentProvider!) {
    initiatePayment(orderId: $orderId, provider: $provider) {
      payment {
        id
        amount
        currency
        provider
        status
      }
      clientSecret
    }
  }
  ```

#### B. Process Payment
Completes verification and authorization of the payment using the token received from the payment UI (Requires `BUYER` role).
* **Mutation**:
  ```graphql
  mutation ProcessPayment($paymentId: ID!, $token: String!) {
    processPayment(paymentId: $paymentId, token: $token) {
      id
      status
      transactionId
    }
  }
  ```

---

## 5. Review & Rating Endpoints

### Queries

#### A. Fetch Seller Detailed Seller Rating (DSR)
Queries ratings breakdown for description accuracy, service quality, and delivery speed.
* **Query**:
  ```graphql
  query GetSellerDsr($sellerId: ID!) {
    seller(id: $sellerId) {
      storeName
      rating
      dsr {
        avgDescription
        avgService
        avgDelivery
        reputationScore
      }
    }
  }
  ```

### Mutations

#### A. Create Review
Submits a rating and comment for a product and transaction (Requires `BUYER` role).
* **Mutation**:
  ```graphql
  mutation CreateReview($input: CreateReviewInput!) {
    createReview(input: $input) {
      id
      ratingDescription
      content
    }
  }
  ```

#### B. Append Review Feedback
Appends additional text or images to an existing review (Requires `BUYER` role).
* **Mutation**:
  ```graphql
  mutation AppendReview($input: AppendReviewInput!) {
    appendReview(input: $input) {
      id
      content
      hasMedia
    }
  }
  ```

#### C. Update Review
Allows a buyer to update an existing review rating and text (Requires `BUYER` role).
* **Mutation**:
  ```graphql
  mutation UpdateReview($input: UpdateReviewInput!) {
    updateReview(input: $input) {
      id
      ratingDescription
      content
    }
  }
  ```

#### D. Delete Review
Deletes a review (Requires `BUYER` role).
* **Mutation**:
  ```graphql
  mutation DeleteReview($reviewId: ID!) {
    deleteReview(reviewId: $reviewId)
  }
  ```

#### E. Create Seller Reply
Enables a store owner to reply to a customer review (Requires `SELLER` role).
* **Mutation**:
  ```graphql
  mutation CreateSellerReply($input: SellerReplyInput!) {
    createSellerReply(input: $input) {
      id
      replyType
      content
    }
  }
  ```

---

## 6. Notification Endpoints

### Queries

#### A. Fetch Preferences
Gets notification category options and channels configured by the buyer (Requires `BUYER` role).
* **Query**:
  ```graphql
  query {
    notificationPreferences {
      category
      emailEnabled
      pushEnabled
    }
  }
  ```

#### B. My Notifications
Retrieves the logged-in user's historical notification delivery records (Requires `BUYER` role).
* **Query**:
  ```graphql
  query GetNotifications($limit: Int, $offset: Int) {
    myNotifications(limit: $limit, offset: $offset) {
      id
      category
      title
      content
      status
      createdAt
    }
  }
  ```

### Mutations

#### A. Register Device Token
Registers a push token (FCM/APNS) for the user's current device (Requires `BUYER` role).
* **Mutation**:
  ```graphql
  mutation RegisterDevice($token: String!, $platform: String!, $deviceName: String) {
    registerDeviceToken(token: $token, platform: $platform, deviceName: $deviceName)
  }
  ```

#### B. Deregister Device Token
Deregisters the device token to prevent sending notifications after logout (Requires `BUYER` role).
* **Mutation**:
  ```graphql
  mutation DeregisterDevice($token: String!) {
    deregisterDeviceToken(token: $token)
  }
  ```

#### C. Update Notification Preferences
Changes communication channel settings for a notification category (Requires `BUYER` role).
* **Mutation**:
  ```graphql
  mutation UpdatePreferences($category: NotificationCategory!, $email: Boolean!, $push: Boolean!) {
    updateNotificationPreferences(category: $category, emailEnabled: $email, pushEnabled: $push) {
      category
      emailEnabled
      pushEnabled
    }
  }
  ```
