.PHONY: dev dev-d dev-down dev-clean proto migrate-up migrate-down seed generate generate-gql test build tools help

SERVICES := user-service seller-service product-service order-service notification-service media-service api-gateway

## ── Infrastructure ─────────────────────────────────────────────────────────
dev: ## Start all services (docker compose up --build)
	docker compose up --build

dev-d: ## Start all services in background
	docker compose up --build -d

dev-down: ## Stop all services
	docker compose down

dev-clean: ## Stop and remove all volumes (fresh start)
	docker compose down -v

logs: ## Tail logs for a service: make logs svc=api-gateway
	docker compose logs -f $(svc)

## ── Proto Generation ────────────────────────────────────────────────────────
proto: ## Generate Go code from proto files (requires buf)
	buf generate

proto-lint: ## Lint proto files
	buf lint

## ── Database ────────────────────────────────────────────────────────────────
migrate-up: ## Run all migrations (or: make migrate-up svc=user-service)
	@if [ -z "$(svc)" ]; then \
		for s in user-service seller-service product-service order-service notification-service media-service; do \
			echo "\n▶ Migrating $$s..."; \
			$(MAKE) migrate-up svc=$$s; \
		done; \
	else \
		cd services/$(svc) && \
		migrate -path db/migrations \
		        -database "$$(grep DB_URL .env.local 2>/dev/null | cut -d= -f2-)" \
		        up; \
	fi

migrate-down: ## Roll back one migration: make migrate-down svc=user-service
	cd services/$(svc) && \
	migrate -path db/migrations \
	        -database "$$(grep DB_URL .env.local 2>/dev/null | cut -d= -f2-)" \
	        down 1

migrate-create: ## Create a new migration: make migrate-create svc=user-service name=add_column
	cd services/$(svc) && \
	migrate create -ext sql -dir db/migrations -seq $(name)

seed: ## Seed categories into product-service DB
	cd services/product-service && go run ./scripts/seed/...

## ── Code Generation ─────────────────────────────────────────────────────────
generate: ## Run sqlc generation for all services
	@for s in user-service seller-service product-service order-service notification-service media-service; do \
		echo "\n▶ sqlc generate for $$s..."; \
		cd services/$$s && sqlc generate && cd ../..; \
	done

generate-gql: ## Regenerate gqlgen code (run after schema changes)
	cd services/api-gateway && go run github.com/99designs/gqlgen generate

## ── Testing ─────────────────────────────────────────────────────────────────
test: ## Run all tests across all services
	@for s in $(SERVICES); do \
		echo "\n▶ Testing $$s..."; \
		(cd services/$$s && CGO_ENABLED=0 go test ./...); \
	done

test-svc: ## Test a specific service: make test-svc svc=user-service
	cd services/$(svc) && CGO_ENABLED=0 go test ./... -v

## ── Build ────────────────────────────────────────────────────────────────────
build: ## Build all service binaries into bin/
	@mkdir -p bin
	@for s in $(SERVICES); do \
		echo "\n▶ Building $$s..."; \
		(cd services/$$s && go build -ldflags="-s -w" -o ../../bin/$$s ./cmd); \
	done

build-svc: ## Build one service: make build-svc svc=api-gateway
	mkdir -p bin
	cd services/$(svc) && go build -ldflags="-s -w" -o ../../bin/$(svc) ./cmd

## ── Tools ────────────────────────────────────────────────────────────────────
tools: ## Install required dev tools
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/bufbuild/buf/cmd/buf@latest
	go install github.com/air-verse/air@latest
	go install github.com/99designs/gqlgen@latest

help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-22s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
