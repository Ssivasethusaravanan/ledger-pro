.PHONY: sqlc migrate-up migrate-down run build clean help

# Default target
help: ## Show this help message
	@echo "LedgerPro — Available targets:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

sqlc: ## Generate Go code from SQL queries via SQLC
	sqlc generate

migrate-up: ## Run all UP migrations against DATABASE_URL
	migrate -path db/migrations -database "$(DATABASE_URL)" up

migrate-down: ## Roll back the last migration
	migrate -path db/migrations -database "$(DATABASE_URL)" down 1

migrate-down-all: ## Roll back ALL migrations (destructive)
	migrate -path db/migrations -database "$(DATABASE_URL)" down

migrate-force: ## Force migration version (usage: make migrate-force V=1)
	migrate -path db/migrations -database "$(DATABASE_URL)" force $(V)

run: ## Run the server in development mode
	go run ./cmd/server/main.go

build: ## Build the production binary
	go build -ldflags="-s -w" -o bin/ledger_pro ./cmd/server/main.go

clean: ## Remove build artifacts
	rm -rf bin/

tidy: ## Tidy Go modules
	go mod tidy

lint: ## Run golangci-lint
	golangci-lint run ./...

test: ## Run all tests
	go test -v -race -count=1 ./...

docker-up: ## Start local infrastructure (Redis + RabbitMQ)
	docker-compose up -d

docker-down: ## Stop local infrastructure
	docker-compose down

docker-logs: ## Tail local infrastructure logs
	docker-compose logs -f
