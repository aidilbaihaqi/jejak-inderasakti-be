ifneq (,$(wildcard ./.env))
    include .env
    export
endif

# Stop Git Bash on Windows from rewriting /app/... container paths.
export MSYS_NO_PATHCONV = 1

.PHONY: lint deploy-check deploy-network logs ps test-docker db-create-host db-migrate db-rollback db-status db-seed db-seed-strict db-refresh dev-up dev-down dev-logs dev-reset dev test seed seed-strict build-fe up down backup

# --- Database commands (run inside the Docker network, so no host port needed) ---
# STACK_FILE selects the stack: dev by default, deploy/docker-compose.yml for staging/production.
STACK_FILE ?= deploy/docker-compose.dev.yml
DB_TOOLS = docker compose -f $(STACK_FILE) --profile tools run --rm --build tools
SEED_ARGS = -questions /seed/questions.json -schools /seed/schools.csv

db-show: 
	docker compose -f deploy/docker-compose.dev.yml exec postgres psql -U jejak -d jejak_inderasakti

db-migrate: ## apply all pending migrations
	$(DB_TOOLS) /app/migrate up

db-rollback: ## roll back the last migration only
	$(DB_TOOLS) /app/migrate down

db-status: ## show applied/pending migrations
	$(DB_TOOLS) /app/migrate status

db-create-host: ## create/update a host login: make db-create-host EMAIL=a@b.c NAME="Host" PASSWORD=secret-pass
	$(DB_TOOLS) /app/createhost -email "$(EMAIL)" -name "$(NAME)" -password "$(PASSWORD)"

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

# Same as `make test` but inside a Go container that has gcc, for machines without CGO (needed by -race).
test-docker:
	docker run --rm -v "$(CURDIR)":/repo -v jejak-gomod:/go/pkg/mod -v jejak-gobuild:/root/.cache/go-build -w /repo/api golang:1.26 go test -race -count=1 ./...

lint:
	docker run --rm -v "$(CURDIR)/api":/app -v jejak-gomod:/go/pkg/mod -v jejak-golangci:/root/.cache -w /app golangci/golangci-lint:latest golangci-lint run

# Validate the staging/production compose file and Caddyfile without starting anything (needs deploy/.env).
# `compose config` only checks YAML + interpolation, so jejak_net (external) need not exist yet.
deploy-check:
	docker compose -f deploy/docker-compose.yml config --quiet
	docker run --rm -e APP_DOMAIN=app.check.example.com -e API_DOMAIN=api.check.example.com -v "$(CURDIR)/deploy/Caddyfile":/etc/caddy/Caddyfile:ro caddy:2-alpine caddy validate --config /etc/caddy/Caddyfile

# Create the Docker network shared by this stack and the frontend (jejak-inderasakti-fe) stack.
# Run once per host before the first `make up`.
deploy-network:
	docker network inspect jejak_net >/dev/null 2>&1 || docker network create jejak_net

logs:
	docker compose -f $(STACK_FILE) logs -f --tail 100

ps:
	docker compose -f $(STACK_FILE) ps

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
