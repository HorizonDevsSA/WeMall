FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy workspace files for dependency resolution
COPY go.work go.work.sum ./
COPY pkg/go.mod             ./pkg/
COPY gen/go.mod             ./gen/
COPY services/user-service/go.mod    ./services/user-service/
COPY services/product-service/go.mod ./services/product-service/
COPY services/order-service/go.mod   ./services/order-service/
COPY services/seller-service/go.mod  ./services/seller-service/
COPY services/api-gateway/go.mod     ./services/api-gateway/
COPY services/notification-service/go.mod ./services/notification-service/
COPY services/media-service/go.mod    ./services/media-service/
COPY services/documentation-service/go.mod ./services/documentation-service/
COPY services/review-service/go.mod ./services/review-service/

RUN go work sync

# Copy source
COPY pkg/    ./pkg/
COPY gen/    ./gen/
COPY services/notification-service/ ./services/notification-service/

RUN go build -ldflags="-s -w" -o /notification-service ./services/notification-service/cmd/main.go

# ── Runtime ───────────────────────────────────────────────────────────────────
FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /notification-service /notification-service
EXPOSE 9007
CMD ["/notification-service"]
