.PHONY: dev build test lint migrate-up docker-up docker-down

# ─── Development ─────────────────────────
dev:
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api

# ─── Docker ──────────────────────────────
docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

# ─── Database ────────────────────────────
migrate-up:
	@echo "Running migrations..."
	@for f in migrations/*.up.sql; do \
		echo "Running $$f"; \
		PGPASSWORD=$(POSTGRES_PASSWORD) psql -h localhost -U $(POSTGRES_USER) -d $(POSTGRES_DB) -f $$f; \
	done

migrate-seed:
	PGPASSWORD=newspass123 psql -h localhost -U newsadmin -d newsplatform -f migrations/007_seed_data.up.sql

# ─── Testing ─────────────────────────────
test:
	go test ./... -v -race

test-cover:
	go test ./... -coverprofile=coverage.out -race
	go tool cover -html=coverage.out -o coverage.html

# ─── Linting ─────────────────────────────
lint:
	golangci-lint run ./...

# ─── Env setup ───────────────────────────
env:
	@if not exist .env copy .env.example .env
