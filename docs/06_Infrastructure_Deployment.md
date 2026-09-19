# 06 — Infrastructure & Deployment
## Jejak Inderasakti: Jelajah Pulau Penyengat

> **Dokumen**: Infra Topology · Docker Compose · Caddyfile · CI/CD Pipeline · Environment Matrix · IaC Plan  
> **Versi**: 1.0 · **Tanggal**: 2026-09-19

---

## 1. Infrastructure Topology

```
Internet
    |
    | HTTPS (443) + HTTP (80)
    v
+---+------------------------------------------+
|              VPS Ubuntu LTS                   |
|         (2 vCPU / 4 GB RAM)                  |
|         Singapura atau Jakarta                |
|                                               |
|  +------------------------------------------+|
|  |           Docker Network: jejak_net       ||
|  |                                           ||
|  |  +-----------+     +------------------+  ||
|  |  |  Caddy    |     |    Go API        |  ||
|  |  |  :80, 443 +---->|    :8080         |  ||
|  |  |  - TLS    |     |  /api/* /ws      |  ||
|  |  |  - static |     |                  |  ||
|  |  |  /srv/web |     +---+----------+---+  ||
|  |  +-----------+         |          |       ||
|  |                        v          v       ||
|  |           +-----------+  +-------+------+ ||
|  |           | PostgreSQL|  |  Redis 7     | ||
|  |           |   17      |  |  :6379       | ||
|  |           | :5432     |  |              | ||
|  |           | Volume:   |  | (no persist) | ||
|  |           | pg_data   |  +--------------+ ||
|  |           +-----------+                   ||
|  +------------------------------------------+|
|                                               |
|  Volumes: pg_data (persistent), caddy_data   |
|  (TLS certs)                                  |
+-----------------------------------------------+

Firewall (UFW):
  ALLOW 80/tcp   (HTTP → redirect ke HTTPS)
  ALLOW 443/tcp  (HTTPS + WSS)
  ALLOW 22/tcp   (SSH, key-based only)
  DENY semua yang lain
```

---

## 2. Docker Compose

```yaml
# deploy/docker-compose.yml
version: "3.9"

networks:
  jejak_net:
    driver: bridge

volumes:
  pg_data:
  caddy_data:

services:
  caddy:
    image: caddy:2-alpine
    restart: always
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
      - ../web/dist:/srv/web:ro
      - caddy_data:/data
    networks:
      - jejak_net
    depends_on:
      - api

  api:
    build:
      context: ../api
      dockerfile: Dockerfile
    restart: always
    env_file: .env
    environment:
      - PORT=8080
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_started
    networks:
      - jejak_net
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:8080/api/healthz"]
      interval: 30s
      timeout: 5s
      retries: 3

  postgres:
    image: postgres:17-alpine
    restart: always
    env_file: .env
    volumes:
      - pg_data:/var/lib/postgresql/data
    networks:
      - jejak_net
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U $POSTGRES_USER -d $POSTGRES_DB"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    restart: always
    command: redis-server --save "" --appendonly no  # no persistence needed
    networks:
      - jejak_net
```

---

## 3. Caddyfile

```caddyfile
# deploy/Caddyfile
jejak.example.id {
    # Route WebSocket (harus sebelum /api agar tidak di-prefix-match dua kali)
    handle /ws {
        reverse_proxy api:8080
    }

    # Route REST API
    handle /api/* {
        reverse_proxy api:8080
    }

    # Semua route lain → SPA (React Router)
    handle {
        root * /srv/web
        try_files {path} /index.html
        file_server
    }

    # Security headers
    header {
        Strict-Transport-Security "max-age=31536000; includeSubDomains"
        X-Content-Type-Options nosniff
        X-Frame-Options DENY
        Referrer-Policy strict-origin-when-cross-origin
    }

    # Gzip encoding
    encode gzip
}
```

---

## 4. Dockerfile (Go API)

```dockerfile
# api/Dockerfile
FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

# --- Runtime image ---
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

COPY --from=builder /server /app/server

EXPOSE 8080
ENTRYPOINT ["/app/server"]
```

---

## 5. Environment Variables

```bash
# deploy/.env (JANGAN commit ke Git)

# PostgreSQL
POSTGRES_USER=jejak
POSTGRES_PASSWORD=<strong-random-password>
POSTGRES_DB=jejak_inderasakti
DATABASE_URL=postgres://jejak:<password>@postgres:5432/jejak_inderasakti?sslmode=disable

# Redis
REDIS_URL=redis://redis:6379/0

# JWT
JWT_SECRET=<min-256-bit-random-hex>

# App
PORT=8080
ENV=production
LOG_LEVEL=info
```

`.gitignore` wajib mengandung:
```
deploy/.env
*.env
```

---

## 6. Makefile (Developer Commands)

```makefile
# Makefile di root repo

.PHONY: dev test seed build deploy backup

# Development
dev:
	cd api && go run ./cmd/server

# Run semua tests
test:
	cd api && go test -race ./...
	cd web && pnpm test

# Seed database (questions.json + schools.csv)
seed:
	cd api && go run ./cmd/seed

# Build FE
build-fe:
	cd web && pnpm build

# Build dan start semua container
up:
	cd deploy && docker compose up -d --build

# Stop semua container
down:
	cd deploy && docker compose down

# Backup PostgreSQL
backup:
	docker compose -f deploy/docker-compose.yml exec postgres \
		pg_dump -Fc -U $$POSTGRES_USER $$POSTGRES_DB \
		> backups/pg/manual_$(shell date +%Y%m%d_%H%M%S).dump

# Deploy ke produksi
deploy:
	git pull origin main
	make build-fe
	make up
	make seed  # idempotent: skip if already seeded

# Lihat log API
logs:
	docker compose -f deploy/docker-compose.yml logs -f api
```

---

## 7. CI/CD Pipeline (GitHub Actions)

### 7.1 Pipeline Overview

```
PR ke main:
  1. Lint & Test (BE + FE)
  2. Build check

Merge ke main:
  (Manual deploy via SSH)
  1. git pull
  2. pnpm build
  3. docker compose up --build
```

### 7.2 Workflow File: `.github/workflows/ci.yml`

```yaml
name: CI

on:
  pull_request:
    branches: [main]
  push:
    branches: [main]

jobs:
  backend:
    name: Go Lint & Test
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:17-alpine
        env:
          POSTGRES_PASSWORD: test
          POSTGRES_DB: jejak_test
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    env:
      DATABASE_URL: postgres://postgres:test@localhost:5432/jejak_test?sslmode=disable
      REDIS_URL: redis://localhost:6379/0

    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.23"
          cache: true

      - name: golangci-lint
        uses: golangci/golangci-lint-action@v6
        with:
          working-directory: api
          version: latest

      - name: Run tests with race detector
        working-directory: api
        run: go test -race -count=1 ./...

  frontend:
    name: FE Lint & Build
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4

      - name: Setup pnpm
        uses: pnpm/action-setup@v3
        with:
          version: 9

      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: "20"
          cache: "pnpm"
          cache-dependency-path: web/pnpm-lock.yaml

      - name: Install dependencies
        working-directory: web
        run: pnpm install --frozen-lockfile

      - name: Lint
        working-directory: web
        run: pnpm lint

      - name: Type check
        working-directory: web
        run: pnpm tsc --noEmit

      - name: Build
        working-directory: web
        run: pnpm build
```

### 7.3 Deploy Manual (Versi Kamis)

```bash
# Di VPS, sebagai user dengan akses Docker
cd /home/deploy/jejak-inderasakti

# Pull latest
git pull origin main

# Build FE
cd web && pnpm install && pnpm build && cd ..

# Restart services
cd deploy && docker compose up -d --build

# Jalankan migrasi (otomatis via goose di startup)
# Seed questions jika belum
docker compose exec api /app/server seed

# Verifikasi
curl https://jejak.example.id/api/healthz
```

---

## 8. Environment Matrix

| Parameter | Development | Staging | Production |
|---|---|---|---|
| **Domain** | localhost:5173 (FE), :8080 (BE) | staging.jejak.example.id | jejak.example.id |
| **TLS** | Tidak (HTTP) | Let's Encrypt (Caddy) | Let's Encrypt (Caddy) |
| **Database** | Local PostgreSQL / Docker | PostgreSQL di VPS staging | PostgreSQL di VPS produksi |
| **Redis** | Local Redis / Docker | Redis di VPS staging | Redis di VPS produksi |
| **Seed data** | `make seed` manual | Termasuk dalam deploy pipeline | Manual sebelum acara |
| **Log level** | debug | info | info |
| **JWT expiry** | 24 jam (untuk kemudahan dev) | 8 jam | 8 jam |
| **Rate limit join** | Dinonaktifkan | 10/menit per IP | 10/menit per IP |
| **Docker** | `docker compose up` lokal | `docker compose up` di staging VPS | `docker compose up` di prod VPS |
| **CI** | Tidak perlu | GitHub Actions (manual trigger) | GitHub Actions pada PR ke main |
| **Backup** | Tidak perlu | pg_dump manual | Cron harian 02:00 |

---

## 9. Struktur Direktori Deploy

```
deploy/
├── .env                    # JANGAN commit; berisi semua secrets
├── .env.example            # Template tanpa nilai; commit ini
├── docker-compose.yml      # Konfigurasi 4 service
├── docker-compose.dev.yml  # Override untuk development (optional)
├── Caddyfile               # Reverse proxy + TLS config
└── backups/
    └── pg/                 # pg_dump files (lokal di VPS)
        ├── 20260922.dump
        ├── manual_20260923_150000.dump
        └── ...
```

---

## 10. IaC Plan (Post-Launch)

> Versi Kamis menggunakan setup manual. Setelah launch, jika traffic meningkat, pertimbangkan:

| Komponen | Tool | Kapan |
|---|---|---|
| VPS provisioning | Terraform + DigitalOcean/Hetzner provider | Setelah volume pengguna terbukti |
| Secret management | HashiCorp Vault atau Doppler | Saat tim bertambah |
| Container orchestration | Kubernetes (GKE/EKS) atau Docker Swarm | Saat butuh high-availability |
| CDN untuk aset statis | Cloudflare R2 + CDN | Saat pengguna lintas kota/pulau |
| Database managed | Cloud SQL / RDS | Saat butuh automated backup + failover |
| Monitoring | Prometheus + Grafana | Setelah MVP terbukti |

**Prioritas untuk versi Kamis**: Zero infrastructure complexity. Satu VPS, satu `docker compose up`, deploy selesai.

---

## 11. Pre-Deploy Checklist

### Development → Staging
- [ ] `go test -race ./...` lulus
- [ ] `golangci-lint` tidak ada error
- [ ] `pnpm lint && pnpm tsc --noEmit && pnpm build` berhasil
- [ ] Migrasi baru sudah dicek memiliki `-- +goose Down`
- [ ] `seed/questions.json` valid JSON; semua 66 soal ada (60 situs + 6 cadangan)
- [ ] `.env.example` terupdate dengan variabel baru (jika ada)

### Staging → Production (Selasa 22/9 malam)
- [ ] Uji end-to-end di staging: daftar → 15 soal → podium (ID + EN)
- [ ] Uji reconnect: putus di soal ke-7, sambung lagi, lanjut dari soal ke-8
- [ ] Uji host: buat room → mulai → monitor → akhiri → download CSV
- [ ] `pg_dump` manual staging tersimpan
- [ ] DNS A record mengarah ke VPS produksi
- [ ] TLS sertifikat valid di browser
- [ ] `/api/healthz` → 200 OK

### Pre-Event (Rabu 23/9 malam)
- [ ] `pg_dump` manual produksi tersimpan
- [ ] k6 load test lulus: 5 room × 15 pemain, p95 < 300ms, 0 error
- [ ] Semua akun host di-seed dan bisa login
- [ ] Poster QR room siap cetak (PDF siap di printer)
- [ ] Tim BE/FE on-call tersedia Kamis 24/9 08:00–17:00
