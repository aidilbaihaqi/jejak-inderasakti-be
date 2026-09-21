# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Status: foundation scaffolded

Done: scaffold + migrations + seed (19 Sep); host login, rooms/join/schools REST, WS hub + room goroutine, question selector (20 Sep); `Score()`, `q.next`/`q.answer` with per-player pacing, Redis leaderboard + throttled `lb.update`, reconnect, auto room end (21 Sep). 22 Sep: join rate limit (Redis + memory fallback), results.csv, school leaderboard, staging stack (`deploy/docker-compose.yml`, `Caddyfile`, `deploy/README.md`), CI. Not done: the actual VPS deploy, k6 load test (see `docs/10_Team_Workflow.md` §7). `docs/` holds 10 design documents (written in Indonesian) generated from three source documents; they are the specification for the backend that will live here. Read the relevant doc before writing code; do not re-derive decisions already recorded there.

| Doc | Read it when |
|---|---|
| `docs/00_INDEX.md` | Orientation, ADR summary, tech stack, NFR targets |
| `docs/01_PRD_BRD.md` | Functional requirements (FR-xx), acceptance criteria |
| `docs/03_System_Architecture.md` | REST + WebSocket contract, package layout, 6 ADRs, `Score()` reference impl |
| `docs/04_Data_Modelling.md` | DDL for the 6 tables, Redis keys, sqlc queries, question-selector composition table |
| `docs/05_NFR_Security.md` | STRIDE model, rate limits, capacity, ops runbook commands |
| `docs/06_Infrastructure_Deployment.md` | Docker Compose, Caddyfile, Dockerfile, Makefile, CI workflow, env vars |
| `docs/09_Testing_Plan.md` | Numbered test cases (UT-SC-xx, UT-RM-xx, …) that new tests should map to |
| `docs/10_Team_Workflow.md` | Definition of Done, coding standards, branching, commit format |

## What is being built

**Jejak Inderasakti** — a mobile-first, real-time multiplayer quiz about five heritage sites on Pulau Penyengat. Bilingual ID/EN. Hard limits that shape the design: max 15 players/room, max 5 concurrent rooms (75 WS connections), 15 questions (or 10 in short session), room auto-ends after 12 minutes.

Target: single Go binary (REST + WebSocket + game engine) behind Caddy, with PostgreSQL 17 as source of truth and Redis 7 as volatile cache — 4 Docker Compose services on one VPS.

## Commands

The Makefile from `docs/06_Infrastructure_Deployment.md` §6 is the intended interface once the repo is scaffolded:

```bash
make dev    # cd api && go run ./cmd/server
make test   # cd api && go test -race ./...  (plus web tests)
make seed   # cd api && go run ./cmd/seed  — loads questions.json + schools.csv, idempotent
make up     # cd deploy && docker compose up -d --build
make backup # pg_dump -Fc into backups/pg/
```

Run a single Go test: `cd api && go test -race -run TestScore ./internal/game/`

Lint: `cd api && golangci-lint run` (errcheck, govet, staticcheck, gofmt, gosec all enabled).

Migrations run automatically via `goose.Up()` in `cmd/server/main.go` before the server listens — there is no separate migrate command.

## Workflow rules (mandatory)

- **After every new feature or fix, run `make test` and make sure it passes before reporting the work as done.** Add or update tests for the change first. Report failures with their output; never claim done on a red or unrun suite. (`-race` needs gcc/CGO: if `make test` fails with "-race requires cgo", run `make test-docker` instead, which runs the identical suite with `-race` inside a Go container.)
- **Run `make lint` too; it must report 0 issues.**
- **Every new or changed REST endpoint must also update the Postman collection** in `postman/` (request, test scripts, saved variables) so it stays in sync with `contracts/openapi.yaml`.

## Local Docker & Postman

```bash
make dev-up      # postgres:5432, redis:6379, api:8080 (docker-compose.dev.yml); migrations auto-run
make db-migrate # apply pending migrations (inside Docker network)
make db-rollback # undo the last migration
make db-status   # applied/pending migrations
make db-seed     # load seed/ (idempotent); db-seed-strict requires 66 questions
make db-refresh  # DESTRUCTIVE dev only: reset -> migrate -> seed-strict
make db-create-host EMAIL=host@example.com NAME="Host Dev" PASSWORD=change-me   # host login (dev)
make test-docker # `go test -race` inside a Go container (use when the machine has no gcc)
make lint        # golangci-lint (errcheck, govet, staticcheck, gosec, gofmt) in a container; must be clean
make deploy-check # validate deploy/docker-compose.yml + Caddyfile
# staging/production: add STACK_FILE=deploy/docker-compose.yml to make up/db-*/ps/logs (see deploy/README.md)
make seed        # host-side seed via .env (needs host port to reach Postgres)
make dev-logs    # tail API logs
make dev-down    # stop (keep data);  make dev-reset  # stop and wipe DB volume
```

`postman/` has the collection + `Jejak - Local (Docker)` environment (`base_url=http://localhost:8080`). The collection pre-request script auto-logs in as the host when `host_token` is missing/expiring; test scripts save `room_id`, `pin`, `player_token`, `school_id`. See `postman/README.md`.

## Architecture invariants

These are load-bearing decisions; changing any of them means revisiting an ADR:

- **One goroutine per room** (ADR-003). All room state mutation happens in that single goroutine, fed by a channel — no mutexes. `go test -race ./...` must pass; it is the primary guard for this design.
- **Per-player pacing, not synchronized rounds** (ADR-002). The server records `served_at` when it sends `q.show` and `answered_at` when `q.answer` arrives; there is no clock sync between devices. Each player may be on a different question index at any moment.
- **Scoring is server-side only** (ADR-006). `q.show` must never contain the `correct` field. The answer key stays server-side; `q.result` is sent only after an answer is received.
- **Postgres is the source of truth; Redis is disposable** (ADR-004). If Redis is lost, the leaderboard is rebuilt from `answers` + `room_players` and PINs from `rooms`. Never store something only in Redis.
- **Standard `net/http`, no framework** (ADR-001). Routing is manual; error responses and param parsing go through local helpers (`writeJSON`, `writeError`).
- Answer deadline is `served_at + limit_ms + 1000ms` (the extra 1000ms is network grace). `q.next` is only served once the previous question was answered or its deadline passed; a skipped question is recorded with 0 points.
- `lb.update` is throttled to at most once per second per room.

## Scoring formula

Reference implementation in `docs/03_System_Architecture.md` §2.4. Key details that are easy to get wrong:

- Base points by level: 1 → 500, 2 → 750, 3 → 1000.
- Reading grace (`GraceMs`) is subtracted before the speed factor: 2000ms, or 3000ms for SD. SD timers are also ×1.25.
- Speed factor is disabled entirely when `AccuracyMode` is on.
- Final: `round(base * (0.6 + 0.4*speed)) + min(50*streakBefore, 250)`.
- Use `math.Round`, not an int cast.

## Intended repo layout

Per `docs/03_System_Architecture.md` §9. The `-be` repo owns `api/`, `contracts/`, `seed/`, `deploy/`, `loadtest/`; `web/` is the frontend counterpart.

```
api/cmd/{server,seed}/       # entrypoints
api/internal/http/           # REST handlers, JWT middleware
api/internal/ws/             # hub, read/write pumps, {t,d} message parsing
api/internal/game/           # room goroutine, scoring, selector, registry
api/internal/store/          # pgx pool, sqlc-generated queries, redis wrapper
api/db/{migrations,queries}/ # goose SQL, sqlc source SQL
contracts/openapi.yaml       # REST contract — FE/BE source of truth
contracts/ws.md              # WebSocket message contract
```

**`contracts/` is frozen.** Do not modify `openapi.yaml` or `ws.md` unless the task is explicitly about changing the contract — the frontend is built against it in parallel.

## Data access

All SQL goes through sqlc: write queries in `api/db/queries/*.sql` with `-- name: X :one|:many|:exec` annotations and regenerate. Never build SQL by string concatenation.

Bilingual content lives in JSONB columns shaped `{"id": "...", "en": "..."}` on `questions` (prompt, options labels, explanation). The server sends only the player's language in `q.show` / `q.result` — it does not ship both.

Question IDs in the real bank (`seed/bank_soal.xlsx`) are `M{site}-{seq}` for every level (`M1-01`..`M5-12`; level is its own column) plus `X-01..X-06` reserve questions with `site = 0`. The selector must only pick `site` 1-5 for room sets. `seed/bank_soal.xlsx` is the editable source (also holds sumber/status_validasi, not stored in DB); regenerate `seed/questions.json` with `python seed/convert_bank.py`.

## Conventions

- Commits: `<type>(<scope>): <subject>` with types `feat|fix|test|refactor|docs|chore`, e.g. `feat(game): implement q.answer handler with deadline validation`.
- Branches: `feature/BE-{num}-{slug}` or `fix/BUG-{num}-{slug}`. Never commit directly to `main`.
- Every I/O function takes `ctx context.Context`. Never discard errors with `_ = err`. Log with `log/slog` in JSON.
- Migrations must include a `-- +goose Down` section.
