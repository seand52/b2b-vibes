.PHONY: dev dev-backend dev-frontend build build-backend build-frontend \
        test test-backend test-frontend test-integration test-all \
        lint lint-backend lint-frontend \
        migrate-up migrate-down migrate-create migrate-force migrate-version \
        install clean

# Load .env file from root if it exists
ifneq (,$(wildcard .env))
    include .env
    export
endif

# ============================================================================
# Development
# ============================================================================

# Run both backend and frontend in parallel
dev:
	@echo "Starting backend and frontend..."
	@$(MAKE) -j2 dev-backend dev-frontend

dev-backend:
	cd backend && $(MAKE) run

dev-frontend:
	cd apps/web && PORT=3000 npm run dev

# ============================================================================
# Build
# ============================================================================

build: build-backend build-frontend

build-backend:
	cd backend && $(MAKE) build

build-frontend:
	cd apps/web && npm run build

# ============================================================================
# Testing
# ============================================================================

test: test-backend test-frontend

test-backend:
	cd backend && $(MAKE) test

test-frontend:
	cd apps/web && npm test

test-integration:
	cd backend && $(MAKE) test-integration

test-all:
	cd backend && $(MAKE) test-all
	cd apps/web && npm test

# ============================================================================
# Linting
# ============================================================================

lint: lint-backend lint-frontend

lint-backend:
	cd backend && golangci-lint run

lint-frontend:
	cd apps/web && npm run lint

# ============================================================================
# Database Migrations (delegates to backend)
# ============================================================================

migrate-up:
	cd backend && $(MAKE) migrate-up

migrate-down:
	cd backend && $(MAKE) migrate-down

migrate-create:
	cd backend && $(MAKE) migrate-create

migrate-force:
	cd backend && $(MAKE) migrate-force

migrate-version:
	cd backend && $(MAKE) migrate-version

# ============================================================================
# Setup & Cleanup
# ============================================================================

install:
	@echo "Installing dependencies..."
	cd apps/web && npm install
	cd backend && go mod download
	@echo "Done!"

clean:
	rm -rf backend/bin
	rm -rf apps/web/.next
	rm -rf apps/web/node_modules/.cache
