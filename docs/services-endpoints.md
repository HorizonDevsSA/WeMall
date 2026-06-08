# WeMall gRPC Service Endpoints

This document catalogs all service endpoints exposed by the downstream microservices in the WeMall platform.

---

## 1. User Service (`UserService`)
Manages buyer/seller accounts, JWT authentication, and user addresses.

* **`BuyerGoogleAuth`** (`GoogleAuthRequest`) → `AuthResponse`
  Initializes or resolves Google OAuth sign-in for buyers.
* **`BuyerSendOTP`** (`PhoneOTPRequest`) → `PhoneOTPResponse`
  Sends SMS OTP verification codes to mobile devices.
* **`BuyerVerifyOTP`** (`VerifyOTPRequest`) → `AuthResponse`
  Validates phone OTP codes and returns authentication credentials.
* **`SellerRegister`** (`SellerRegisterRequest`) → `AuthResponse`
  Registers a new seller account with an email and password.
* **`SellerLogin`** (`SellerLoginRequest`) → `AuthResponse`
  Authenticates a seller and returns JWT access tokens.
* **`RefreshToken`** (`RefreshTokenRequest`) → `AuthResponse`
  Refreshes an expired JWT access token using a refresh token.
* **`ValidateToken`** (`ValidateTokenRequest`) → `ValidateTokenResponse`
  Validates an access token and returns payload information.
* **`GetUser`** (`GetUserRequest`) → `User`
  Retrieves a user profile by ID.
* **`GetUserBatch`** (`GetUserBatchRequest`) → `GetUserBatchResponse`
  Batch fetches profiles for a collection of user IDs.
* **`UpdateProfile`** (`UpdateProfileRequest`) → `User`
  Updates a user's full name and avatar URL.
* **`ListAddresses`** (`ListAddressesRequest`) → `ListAddressesResponse`
  Retrieves saved delivery addresses for a buyer.
* **`CreateAddress`** (`CreateAddressRequest`) → `Address`
  Creates a new delivery address.
* **`DeleteAddress`** (`DeleteAddressRequest`) → `google.protobuf.Empty`
  Deletes an address by ID.
* **`SendReviewStatusEmail`** (`SendReviewStatusEmailRequest`) → `google.protobuf.Empty`
  Sends email updates regarding seller registration verification outcomes.

---

## 2. Seller Service (`SellerService`)
Manages store profiles, follower metrics, and payout summaries.

* **`GetSeller`** (`GetSellerRequest`) → `Seller`
  Retrieves seller store details by ID.
* **`GetSellerByUserID`** (`GetSellerByUserIDRequest`) → `Seller`
  Retrieves store details for a seller by their User ID.
* **`GetSellerBatch`** (`GetSellerBatchRequest`) → `GetSellerBatchResponse`
  Batch fetches seller profiles.
* **`CreateStore`** (`CreateStoreRequest`) → `Seller`
  Registers a new storefront.
* **`UpdateStore`** (`UpdateStoreRequest`) → `Seller`
  Updates store profiles (slug, logo, banner, desc, coordinates).
* **`VerifySeller`** (`VerifySellerRequest`) → `Seller`
  Verifies or suspends a seller store profile.
* **`UpdateSellerStatus`** (`UpdateSellerStatusRequest`) → `Seller`
  Updates a store's lifecycle status (`pending`, `verified`, etc.).
* **`FollowStore`** (`FollowStoreRequest`) → `google.protobuf.Empty`
  Registers a follower association between a buyer and store.
* **`UnfollowStore`** (`UnfollowStoreRequest`) → `google.protobuf.Empty`
  Removes a follower association.
* **`IsFollowingStore`** (`IsFollowingStoreRequest`) → `IsFollowingStoreResponse`
  Checks if a user follows a store.
* **`ListFollowedStores`** (`ListFollowedStoresRequest`) → `ListFollowedStoresResponse`
  Lists stores followed by a buyer.
* **`ListStoreFollowers`** (`ListStoreFollowersRequest`) → `ListStoreFollowersResponse`
  Lists the user IDs of buyers following a store.
* **`ListPayouts`** (`ListPayoutsRequest`) → `ListPayoutsResponse`
  Lists payment settlement payouts to sellers.
* **`GetPayout`** (`GetPayoutRequest`) → `Payout`
  Retrieves a payout record.
* **`CreatePayout`** (`CreatePayoutRequest`) → `Payout`
  Registers a new payout record.

---

## 3. Product Service (`ProductService`)
Manages category schemas, product variants, inventory, and geo-searches.

* **`ListCategories`** (`ListCategoriesRequest`) → `ListCategoriesResponse`
  Retrieves hierarchical product categories.
* **`GetCategory`** (`GetCategoryRequest`) → `Category`
  Retrieves a specific category.
* **`ListProducts`** (`ListProductsRequest`) → `ListProductsResponse`
  Lists and filters products using search keywords, tags, or fields.
* **`GetProduct`** (`GetProductRequest`) → `Product`
  Gets details of a single product.
* **`GetProductBatch`** (`GetProductBatchRequest`) → `GetProductBatchResponse`
  Batch fetches product details.
* **`GetVariantBatch`** (`GetVariantBatchRequest`) → `GetVariantBatchResponse`
  Batch fetches product variant details.
* **`CreateProduct`** (`CreateProductRequest`) → `Product`
  Creates a new product with custom variants.
* **`UpdateProduct`** (`UpdateProductRequest`) → `Product`
  Updates product details and lifecycle status.
* **`DeleteProduct`** (`DeleteProductRequest`) → `google.protobuf.Empty`
  Deletes a product entry.
* **`ListNearbyProducts`** (`ListNearbyProductsRequest`) → `ListNearbyProductsResponse`
  Searches for products near specific latitude and longitude coordinates.
* **`ListRecommendedProducts`** (`ListRecommendedProductsRequest`) → `ListRecommendedProductsResponse`
  Returns product recommendations.

---

## 4. Order Service (`OrderService`)
Manages buyer shopping carts and processes order checkouts.

* **`GetCart`** (`GetCartRequest`) → `Cart`
  Retrieves active items inside a buyer's cart.
* **`AddToCart`** (`AddToCartRequest`) → `Cart`
  Adds an item to the shopping cart.
* **`UpdateCartItem`** (`UpdateCartItemRequest`) → `Cart`
  Updates cart item quantities.
* **`RemoveCartItem`** (`RemoveCartItemRequest`) → `Cart`
  Removes an item from the cart.
* **`ClearCart`** (`ClearCartRequest`) → `Cart`
  Empties the shopping cart.
* **`Checkout`** (`CheckoutRequest`) → `Order`
  Converts cart items to an order (pending payment).
* **`GetOrder`** (`GetOrderRequest`) → `Order`
  Retrieves details of an order.
* **`ListOrders`** (`ListOrdersRequest`) → `ListOrdersResponse`
  Lists historical orders for a user.
* **`CancelOrder`** (`CancelOrderRequest`) → `Order`
  Cancels a pending order.

---

## 5. Notification Service (`NotificationService`)
Coordinates messaging (SMS, Email, Push) and device registers.

* **`RegisterDeviceToken`** (`RegisterDeviceTokenRequest`) → `google.protobuf.Empty`
  Registers client device push notification tokens.
* **`DeregisterDeviceToken`** (`DeregisterDeviceTokenRequest`) → `google.protobuf.Empty`
  Deregisters client device push tokens.
* **`GetNotificationPreferences`** (`GetNotificationPreferencesRequest`) → `GetNotificationPreferencesResponse`
  Lists category notification options for a user.
* **`UpdateNotificationPreferences`** (`UpdateNotificationPreferencesRequest`) → `NotificationPreference`
  Updates notification categories (opt-in/opt-out).
* **`ListNotifications`** (`ListNotificationsRequest`) → `ListNotificationsResponse`
  Lists transactional and promotional notification history logs.

---

## 6. Media Service (`MediaService`)
Manages file uploads, S3 presigned URLs, and responsive CDN variants.

* **`RequestUploadUrl`** (`RequestUploadUrlRequest`) → `RequestUploadUrlResponse`
  Generates a presigned S3 PUT URL for uploading media.
* **`ConfirmUpload`** (`ConfirmUploadRequest`) → `ConfirmUploadResponse`
  Confirms raw file upload completion to queue it for transcoding/resizing.
* **`GetMediaAsset`** (`GetMediaAssetRequest`) → `GetMediaAssetResponse`
  Gets CDN-signed URLs for media variants.
* **`BatchGetMediaAssets`** (`BatchGetMediaAssetsRequest`) → `BatchGetMediaAssetsResponse`
  Batch fetches media CDN variant URLs.
* **`ListMediaAssets`** (`ListMediaAssetsRequest`) → `ListMediaAssetsResponse`
  Lists files uploaded by a specific user/seller.

---

## 7. Review Service (`ReviewService`)
Manages product reviews, ratings, NLP tags, and seller reply comments.

* **`CreateReview`** (`CreateReviewRequest`) → `Review`
  Submits a buyer rating and review for a purchased order item.
* **`AppendReview`** (`AppendReviewRequest`) → `AppendReviewResponse`
  Appends additional content or images to an existing review.
* **`UpdateReview`** (`UpdateReviewRequest`) → `Review`
  Edits review contents within the 30-day window.
* **`DeleteReview`** (`DeleteReviewRequest`) → `google.protobuf.Empty`
  Deletes a review comment.
* **`CreateSellerReply`** (`CreateSellerReplyRequest`) → `SellerReply`
  Submits a seller response to a buyer's review.
* **`GetReview`** (`GetReviewRequest`) → `Review`
  Retrieves a specific review.
* **`ListProductReviews`** (`ListProductReviewsRequest`) → `ListProductReviewsResponse`
  Lists and filters reviews for a product.
* **`ListSellerReviews`** (`ListSellerReviewsRequest`) → `ListSellerReviewsResponse`
  Lists reviews for a seller.
* **`GetProductRatingStats`** (`GetProductRatingStatsRequest`) → `ProductRatingStats`
  Gets aggregate rating details (averages, counts, tags).
* **`GetSellerDSR`** (`GetSellerDSRRequest`) → `SellerDSR`
  Retrieves Detailed Seller Ratings (DSR) metrics.

---

## 8. Payment Service (`PaymentService` - *New*)
Orchestrates payments supporting Google Pay (primary) and Stripe (secondary).

* **`CreatePayment`** (`CreatePaymentRequest`) → `CreatePaymentResponse`
  Creates a new payment intent tracking record and returns provider secret keys.
* **`ProcessPayment`** (`ProcessPaymentRequest`) → `ProcessPaymentResponse`
  Completes a payment using Google Pay client tokenized data or Stripe methods.
* **`GetPayment`** (`GetPaymentRequest`) → `Payment`
  Retrieves a payment record by ID.
* **`RefundPayment`** (`RefundPaymentRequest`) → `RefundPaymentResponse`
  Processes refunds for completed payments.
