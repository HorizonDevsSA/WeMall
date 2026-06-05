# WeMall API Documentation

Welcome to the WeMall backend API documentation. All API calls go through the GraphQL gateway at `http://localhost:8080/graphql`.

## Guides

| Guide | Description |
|-------|-------------|
| [Cart & Orders API](./cart-and-orders-api.md) | Complete buyer journey: OTP login → browse → add to cart → checkout → order history |

## Quick Start

```bash
# 1. Start all services
docker compose up -d

# 2. Open GraphQL Playground
open http://localhost:8080/playground

# 3. Run the integration test
bash test_cart_and_orders.sh
```

## Service Ports

| Service | Port | Protocol |
|---------|------|----------|
| API Gateway (GraphQL) | `8080` | HTTP / WebSocket |
| User Service | `9001` | gRPC |
| Seller Service | `9002` | gRPC |
| Product Service | `9003` | gRPC |
| Order Service | `9005` | gRPC |
| Notification Service | `9007` | gRPC |
| Media Service | `8087` / `50057` | HTTP / gRPC |

## Database Ports (Local Dev)

| Database | Port | DB Name |
|----------|------|---------|
| Users | `5432` | `wemall_users` |
| Products | `5433` | `wemall_products` |
| Orders | `5434` | `wemall_orders` |
| Sellers | `5435` | `wemall_sellers` |
| Notifications | `5436` | `wemall_notifications` |
| Media | `5437` | `wemall_media` |
