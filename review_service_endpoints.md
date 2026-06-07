# WeMall Review Service Endpoints Documentation

This document describes all endpoints exposed by the WeMall Review Service. It is divided into:
1. **Public/Gateway GraphQL Endpoints** (exposed via `http://localhost:8080/graphql`)
2. **Internal gRPC Endpoints** (exposed internally on `review-service:9009`)

---

## 1. GraphQL Gateway Endpoints (Public HTTP)

All public requests must be sent as JSON payloads to `http://localhost:8080/graphql`.

### Authentication Headers
For mutations (write operations), you must authenticate as either a Buyer or Seller using a JWT Bearer token:
```http
Authorization: Bearer <your_token>
```

---

### Mutations

#### A. Create a Review (Buyer only)
Allows a buyer to submit a product and seller rating after purchasing.
* **GraphQL Mutation**:
  ```graphql
  mutation CreateReview($input: CreateReviewInput!) {
    createReview(input: $input) {
      id
      ratingDescription
      content
      reviewType
      isAnonymous
    }
  }
  ```
* **curl Command**:
  ```bash
  curl -X POST http://localhost:8080/graphql \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer <BUYER_TOKEN>" \
    -d '{
      "query": "mutation CreateReview($input: CreateReviewInput!) { createReview(input: $input) { id ratingDescription content reviewType } }",
      "variables": {
        "input": {
          "orderId": "c5ea1c07-9c98-4605-94b1-af93bb7394ba",
          "productId": "ff5dfbf6-5672-45f5-8248-505625dba35d",
          "variantId": "51e703c2-2d90-45bb-b253-0b2d40a96d7d",
          "ratingDescription": 3,
          "ratingService": 4,
          "ratingDelivery": 3,
          "content": "Original review: decent product, fast shipping.",
          "isAnonymous": false,
          "mediaUrls": ["http://example.com/image1.png"]
        }
      }
    }'
  ```

#### B. Update a Review (Buyer only)
Buyers can update Bad/Neutral reviews (<= 3 stars) to positive reviews (4 or 5 stars) within a 30-day editing window.
* **GraphQL Mutation**:
  ```graphql
  mutation UpdateReview($input: UpdateReviewInput!) {
    updateReview(input: $input) {
      id
      ratingDescription
      content
      reviewType
    }
  }
  ```
* **curl Command**:
  ```bash
  curl -X POST http://localhost:8080/graphql \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer <BUYER_TOKEN>" \
    -d '{
      "query": "mutation UpdateReview($input: UpdateReviewInput!) { updateReview(input: $input) { id ratingDescription content reviewType } }",
      "variables": {
        "input": {
          "reviewId": "a456651e-c0a3-4f98-bab6-bc6f87e7fbfd",
          "ratingDescription": 5,
          "ratingService": 5,
          "ratingDelivery": 5,
          "content": "Updated review: actually this is extremely good!"
        }
      }
    }'
  ```

#### C. Append/Add Additional Content to a Review (Buyer only)
Enables a buyer to add further feedback or more media/images to an existing review.
* **GraphQL Mutation**:
  ```graphql
  mutation AppendReview($input: AppendReviewInput!) {
    appendReview(input: $input) {
      id
      content
      hasMedia
    }
  }
  ```
* **curl Command**:
  ```bash
  curl -X POST http://localhost:8080/graphql \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer <BUYER_TOKEN>" \
    -d '{
      "query": "mutation AppendReview($input: AppendReviewInput!) { appendReview(input: $input) { id content hasMedia } }",
      "variables": {
        "input": {
          "reviewId": "a456651e-c0a3-4f98-bab6-bc6f87e7fbfd",
          "content": "Additional feedback: still loving it after 3 days!",
          "mediaUrls": ["http://example.com/image2.png"]
        }
      }
    }'
  ```

#### D. Create Seller Reply (Seller only)
Allows a store/seller to reply to a buyer's review.
* **GraphQL Mutation**:
  ```graphql
  mutation CreateSellerReply($input: SellerReplyInput!) {
    createSellerReply(input: $input) {
      id
      content
      replyType
    }
  }
  ```
* **curl Command**:
  ```bash
  curl -X POST http://localhost:8080/graphql \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer <SELLER_TOKEN>" \
    -d '{
      "query": "mutation CreateSellerReply($input: SellerReplyInput!) { createSellerReply(input: $input) { id content replyType } }",
      "variables": {
        "input": {
          "reviewId": "a456651e-c0a3-4f98-bab6-bc6f87e7fbfd",
          "replyType": "initial",
          "content": "Thank you for the positive review! We hope to serve you again."
        }
      }
    }'
  ```

#### E. Delete Review (Buyer only)
Allows a buyer to remove their own review within a 30-day window.
* **GraphQL Mutation**:
  ```graphql
  mutation {
    deleteReview(reviewId: "a456651e-c0a3-4f98-bab6-bc6f87e7fbfd")
  }
  ```
* **curl Command**:
  ```bash
  curl -X POST http://localhost:8080/graphql \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer <BUYER_TOKEN>" \
    -d '{"query": "mutation { deleteReview(reviewId: \"a456651e-c0a3-4f98-bab6-bc6f87e7fbfd\") }"}'
  ```

---

### Queries

#### A. Query Product Reviews & Aggregate Rating Stats
Fetches the rating summary (good/neutral/bad counts, tags, average) and the list of individual reviews with replies.
* **GraphQL Query**:
  ```graphql
  query GetProductReviews($productId: String!) {
    product(id: $productId) {
      rating
      reviewCount
      reviews {
        edges {
          id
          content
          ratingDescription
          buyerName
          isAnonymous
          appendReview {
            content
          }
          replies {
            content
          }
        }
      }
      reviewStats {
        averageRating
        totalReviews
        goodCount
        neutralCount
        badCount
        topTags
      }
    }
  }
  ```
* **curl Command**:
  ```bash
  curl -X POST http://localhost:8080/graphql \
    -H "Content-Type: application/json" \
    -d '{
      "query": "query { product(id: \"ff5dfbf6-5672-45f5-8248-505625dba35d\") { rating reviewCount reviews { edges { id content ratingDescription buyerName isAnonymous appendReview { content } replies { content } } } reviewStats { averageRating totalReviews goodCount neutralCount badCount topTags } } }"
    }'
  ```

#### B. Query Detailed Seller Rating (DSR)
Fetches a detailed breakdown of a store's performance ratings (Description accuracy, Service quality, Delivery speed, and overall Reputation Score).
* **GraphQL Query**:
  ```graphql
  query GetSellerDsr($sellerId: String!) {
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
* **curl Command**:
  ```bash
  curl -X POST http://localhost:8080/graphql \
    -H "Content-Type: application/json" \
    -d '{
      "query": "query { seller(id: \"117f5472-ac34-44ab-8e5d-9687dfdfc443\") { storeName rating dsr { avgDescription avgService avgDelivery reputationScore } } }"
    }'
  ```

---

## 2. gRPC Endpoints (Internal IPC)

These endpoints are exposed by the microservice internally via the gRPC network listener. Below is the mapping from `proto/review/v1/review.proto`.

| Service / RPC Method | Request Message Type | Response Message Type | Description |
|---|---|---|---|
| `CreateReview` | `CreateReviewRequest` | `Review` | Inserts a new buyer review in the DB. |
| `AppendReview` | `AppendReviewRequest` | `AppendReviewResponse` | Appends text/media feedback to a review. |
| `UpdateReview` | `UpdateReviewRequest` | `Review` | Modifies star ratings & content of bad/neutral reviews. |
| `DeleteReview` | `DeleteReviewRequest` | `google.protobuf.Empty` | Marks a review as deleted in the DB. |
| `CreateSellerReply` | `CreateSellerReplyRequest`| `SellerReply` | Stores a seller response to a specific review. |
| `GetReview` | `GetReviewRequest` | `Review` | Retrieves a single review by its UUID. |
| `ListProductReviews` | `ListProductReviewsRequest` | `ListProductReviewsResponse`| Lists and filters reviews for a given product ID. |
| `ListSellerReviews` | `ListSellerReviewsRequest` | `ListSellerReviewsResponse` | Lists reviews for all products sold by a seller ID. |
| `GetProductRatingStats`| `GetProductRatingStatsRequest`| `ProductRatingStats` | Computes aggregate counts, stars, and tags for a product. |
| `GetSellerDSR` | `GetSellerDSRRequest` | `SellerDSR` | Computes rolling avg DSR metrics for a seller. |
