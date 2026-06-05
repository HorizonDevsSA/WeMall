FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.work go.work.sum ./
COPY pkg/go.mod             ./pkg/
COPY gen/go.mod             ./gen/
COPY services/user-service/go.mod    ./services/user-service/
COPY services/product-service/go.mod ./services/product-service/
COPY services/order-service/go.mod   ./services/order-service/
COPY services/seller-service/go.mod  ./services/seller-service/
COPY services/api-gateway/go.mod     ./services/api-gateway/
COPY services/media-service/go.mod    ./services/media-service/
COPY services/notification-service/go.mod ./services/notification-service/
COPY services/documentation-service/go.mod ./services/documentation-service/


RUN go work sync

COPY pkg/    ./pkg/
COPY gen/    ./gen/
# order-service imports product-service proto types at runtime via gRPC client
COPY services/product-service/ ./services/product-service/
COPY services/order-service/   ./services/order-service/

RUN go build -ldflags="-s -w" -o /order-service ./services/order-service/cmd/main.go

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /order-service /order-service
EXPOSE 9005
CMD ["/order-service"]
