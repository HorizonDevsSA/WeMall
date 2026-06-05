# WeMall — Phase 1 Build Tasks

Legend: ✅ Done · ⬜ Not started

---

## Infrastructure & Shared

- [x] `docker-compose.yml` — PostgreSQL ×3, Redis, NATS, Meilisearch ✅
- [x] `Makefile` — all dev commands ✅
- [x] `buf.yaml` + `buf.gen.yaml` — proto linting + codegen config ✅
- [x] `.env.example` — all env vars documented ✅
- [x] `proto/user/v1/user.proto` ✅
- [x] `proto/product/v1/product.proto` ✅
- [x] `proto/order/v1/order.proto` ✅
- [x] `pkg/auth/` — JWT helpers ✅
- [x] `pkg/logger/` — zerolog setup ✅
- [x] `pkg/errors/` — gRPC error mapping ✅

---

## User Service (`services/user-service/`)

- [x] `go.mod` ✅
- [x] `cmd/main.go` — gRPC server bootstrap ✅
- [x] `db/migrations/` — users, refresh_tokens, addresses, otps ✅ (4 migration pairs)
- [x] `db/queries/users.sql` — sqlc query files ✅
- [x] `sqlc.yaml` ✅
- [x] `internal/db/` — DB layer (sqlc generated: db.go, models.go, users.sql.go) ✅
- [x] `internal/service/auth.go` — Google OAuth, Phone OTP, Email/Password ✅
- [x] `internal/service/user.go` — profile, address management ✅
- [x] `internal/handler/user_handler.go` — gRPC handler ✅
- [x] `Dockerfile` ✅

---

## Product Service (`services/product-service/`)

- [x] `go.mod` ✅
- [x] `cmd/main.go` ✅
- [x] `db/migrations/` — categories, products, variants, images, tags ✅ (2 migration pairs)
- [x] `db/queries/` — sqlc query files ✅ (categories.sql + products.sql)
- [x] `sqlc.yaml` ✅
- [x] `internal/db/` — DB layer (sqlc generated: db.go, models.go, products.sql.go) ✅
- [x] `internal/service/category.go` — category tree, i18n, tree builder ✅
- [x] `internal/service/product.go` ✅
- [x] `internal/handler/product_handler.go` — gRPC handler ✅
- [x] `scripts/seed/main.go` — categories + subcategories + attribute schemas ✅
- [x] `Dockerfile` ✅

---

## Order Service (`services/order-service/`)

- [x] `go.mod` ✅
- [x] `cmd/main.go` ✅
- [x] `db/migrations/` — carts, cart_items, orders, order_items, promotions, coupons ✅ (2 migration pairs)
- [x] `db/queries/orders.sql` — sqlc query files ✅
- [x] `sqlc.yaml` ✅
- [x] `internal/db/` — DB layer (sqlc generated: db.go, models.go, orders.sql.go) ✅
- [x] `internal/service/cart.go` ✅
- [x] `internal/service/order.go` ✅
- [x] `internal/handler/order_handler.go` — gRPC handler ✅
- [x] `Dockerfile` ✅

---

## API Gateway (`services/api-gateway/`)

- [x] `go.mod` — updated with gqlgen + vektah/gqlparser deps ✅
- [x] `cmd/main.go` — gqlgen server, JWT middleware, @hasRole directive, graceful shutdown ✅
- [x] `gqlgen.yml` ✅
- [x] `internal/graph/schema/schema.graphql` — full schema (queries, mutations, all types) ✅
- [x] `internal/graph/generated/generated.go` — gqlgen wiring (resolver interfaces, schema) ✅
- [x] `internal/graph/resolver/query.resolvers.go` — all query resolvers ✅
- [x] `internal/graph/resolver/mutation.resolvers.go` — all mutation resolvers ✅
- [x] `internal/graph/resolver/subscription.resolvers.go` — Phase 4 placeholder ✅
- [x] `internal/graph/model/models.go` — all GraphQL model types ✅
- [x] `internal/graph/resolver/mappers.go` — proto → model mapping helpers ✅
- [x] `internal/clients/clients.go` — gRPC client wrappers ✅
- [x] `internal/middleware/auth.go` — JWT validation middleware ✅
- [x] `Dockerfile` ✅

---

## Summary

| Area | Done | Remaining |
|---|---|---|
| Infrastructure & Shared | 10 / 10 | 0 |
| User Service | 10 / 10 | 0 |
| Product Service | 11 / 11 | 0 |
| Order Service | 10 / 10 | 0 |
| API Gateway | 13 / 13 | 0 |
| **Total** | **54 / 54** | **0** |

---

## Phase 1 Complete ✅

All tasks done. Run `make dev` to start the full stack.

**Next: Phase 2** — seller-service, inventory-service, DataLoaders, Redis caching.
