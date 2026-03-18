.PHONY: run build test test-integration test-all migrate-up migrate-down migrate-create migrate-force migrate-version

# Load .env file if it exists
ifneq (,$(wildcard .env))
    include .env
    export
endif

# Application
run:
	CGO_ENABLED=0 go run ./cmd/api

build:
	CGO_ENABLED=0 go build -o bin/b2b-orders-api ./cmd/api

test:
	CGO_ENABLED=0 go test ./...

test-integration:
	CGO_ENABLED=0 go test -tags=integration -v ./internal/integration/...

test-all:
	CGO_ENABLED=0 go test -tags=integration ./...

# Database migrations (requires: brew install golang-migrate)
migrate-up:
	migrate -path ./migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path ./migrations -database "$(DATABASE_URL)" down 1

migrate-force:
	@read -p "Version to force: " version; \
	migrate -path ./migrations -database "$(DATABASE_URL)" force $$version

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir ./migrations -seq $$name

migrate-version:
	migrate -path ./migrations -database "$(DATABASE_URL)" version
