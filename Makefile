ifneq (,$(wildcard ./.env))
    include .env
    export
endif

.PHONY: db-migrate db-rollback db-status db-seed db-seed-strict db-refresh dev-up dev-down dev-logs dev-reset dev test seed seed-strict build-fe up down backup

# --- Database commands (run inside the Docker network, so no host port needed) ---
DB_TOOLS = docker compose -f deploy/docker-compose.dev.yml --profile tools run --rm --build tools
SEED_ARGS = -questions /seed/questions.json -schools /seed/schools.csv

db-show: 
	docker compose -f deploy/docker-compose.dev.yml exec postgres psql -U jejak -d jejak_inderasakti

db-migrate: ## apply all pending migrations
	$(DB_TOOLS) /app/migrate up

db-rollback: ## roll back the last migration only
	$(DB_TOOLS) /app/migrate down

db-status: ## show applied/pending migrations
	$(DB_TOOLS) /app/migrate status

db-seed: ## load questions + schools (idempotent)
	$(DB_TOOLS) /app/seed $(SEED_ARGS)

db-seed-strict: ## same, but require the full 66-question bank
	$(DB_TOOLS) /app/seed $(SEED_ARGS) -strict

db-refresh: ## DESTRUCTIVE (dev only): roll back everything, migrate, seed
	$(DB_TOOLS) /app/migrate reset
	$(DB_TOOLS) /app/migrate up
	$(DB_TOOLS) /app/seed $(SEED_ARGS) -strict

dev-up:
	docker compose -f deploy/docker-compose.dev.yml up -d --build

dev-down:
	docker compose -f deploy/docker-compose.dev.yml down

dev-logs:
	docker compose -f deploy/docker-compose.dev.yml logs -f api

dev-reset:
	docker compose -f deploy/docker-compose.dev.yml down -v

dev:
	cd api && go run ./cmd/server

test:
	cd api && go test -race ./...

seed:
	cd api && go run ./cmd/seed

seed-strict:
	cd api && go run ./cmd/seed -strict

build-fe:
	cd web && pnpm build

up:
	cd deploy && docker compose up -d --build

down:
	cd deploy && docker compose down

backup:
	mkdir -p backups/pg
	docker compose -f deploy/docker-compose.yml exec -T postgres pg_dump -U $${POSTGRES_USER:-jejak} -Fc $${POSTGRES_DB:-jejak_inderasakti} > backups/pg/jejak_$$(date +%Y%m%d_%H%M%S).dump
