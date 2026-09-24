# Deploying to staging / production

One VPS (Ubuntu LTS, 2 vCPU / 4 GB) runs two independent Docker Compose stacks that share one
Docker network:

- **This repo** (`jejak-inderasakti-be/deploy/`): **caddy** (TLS, reverse proxy for both domains),
  **api**, **postgres**, **redis**.
- **The frontend repo** (`jejak-inderasakti-fe/deploy/`): **web**, the Next.js app.

Two domains, one certificate each, both served by the single Caddy instance above:

| Domain | Points to | Purpose |
|---|---|---|
| `penyengatadventure.tech` (`APP_DOMAIN`) | `web` container (frontend stack) | the Next.js frontend |
| `api.penyengatadventure.tech` (`API_DOMAIN`) | `api` container (this stack) | REST + WebSocket API |

Design: `docs/06_Infrastructure_Deployment.md`. Local development uses `docker-compose.dev.yml` instead.

## 1. Prepare the server (once)

```bash
# Docker + compose plugin, git, make
sudo apt update && sudo apt install -y git make ca-certificates curl
curl -fsSL https://get.docker.com | sudo sh && sudo usermod -aG docker "$USER"   # log out and in again

# Firewall: SSH, HTTP (certificate + redirect), HTTPS/WSS only
sudo ufw allow 22/tcp && sudo ufw allow 80/tcp && sudo ufw allow 443/tcp && sudo ufw enable
```

DNS: create an **A record** for each domain pointing at the server's public IP —
`penyengatadventure.tech` and `api.penyengatadventure.tech`. Caddy cannot obtain certificates
until both resolve and ports 80/443 are reachable.

Clone both repos as siblings (the frontend stack's compose file expects this layout, and so does
`make deploy-network` below):

```bash
git clone https://github.com/aidilbaihaqi/jejak-inderasakti-be.git
git clone https://github.com/aidilbaihaqi/jejak-inderasakti-fe.git
# -> ~/jejak-inderasakti-be and ~/jejak-inderasakti-fe
```

Create the Docker network the two stacks share (once per host):

```bash
cd jejak-inderasakti-be && make deploy-network   # docker network create jejak_net
```

## 2. Configure secrets

```bash
cd jejak-inderasakti-be
cp deploy/.env.example deploy/.env
```

Edit `deploy/.env`: `APP_DOMAIN` and `API_DOMAIN` already default to the real domains above, then
generate the secrets on the server so they never leave it:

```bash
echo "POSTGRES_PASSWORD=$(openssl rand -hex 24)"
echo "JWT_SECRET=$(openssl rand -hex 32)"
```

`deploy/.env` is git-ignored. Use different secrets for staging and production.

Do the equivalent for the frontend: `cd ../jejak-inderasakti-fe && cp deploy/.env.example deploy/.env`
(see that repo's `deploy/README.md` — it only needs `API_DOMAIN`, used as a build arg).

## 3. Start and initialise

```bash
cd jejak-inderasakti-be
make deploy-check                       # validates compose + Caddyfile
make up                                 # builds and starts caddy, api, postgres, redis

# STACK_FILE points the db-* commands at this stack instead of the dev one
make db-migrate STACK_FILE=deploy/docker-compose.yml      # (the API also migrates on start)
make db-seed-strict STACK_FILE=deploy/docker-compose.yml  # loads the 66-question bank
make db-create-host STACK_FILE=deploy/docker-compose.yml EMAIL=panitia@example.id NAME="Panitia" PASSWORD='<at least 8 chars>'

# then start the frontend stack (builds the Next.js image; see its own deploy/README.md)
cd ../jejak-inderasakti-fe && make up
```

Start the API stack first — Caddy's `depends_on: api: condition: service_healthy` needs it, and
the frontend's `web` container is only reachable once it joins `jejak_net`.

## 4. Verify

```bash
curl -i https://api.penyengatadventure.tech/api/healthz   # 200 {"status":"ok"} with a valid certificate
curl -i https://penyengatadventure.tech                   # the frontend, also with a valid certificate
make ps STACK_FILE=deploy/docker-compose.yml
make logs STACK_FILE=deploy/docker-compose.yml
```

Then import `postman/` and set `base_url` to `https://api.penyengatadventure.tech` to run the
collection against staging. WebSocket check:
`wss://api.penyengatadventure.tech/ws?token=...` (see `contracts/ws.md`) — the browser's `Origin`
header must be `https://penyengatadventure.tech`, which `CORS_ALLOWED_ORIGINS` allow-lists both
for REST CORS and for the WebSocket upgrade's origin check.

## 5. Operate

| Task | Command |
|---|---|
| Deploy a new backend version | `git pull && make up` |
| Deploy a new frontend version | `cd ../jejak-inderasakti-fe && git pull && make up` |
| Backup the database | `make backup` (writes `backups/pg/*.dump`; it reads `POSTGRES_USER` / `POSTGRES_DB` from the shell, defaults `jejak` / `jejak_inderasakti`) |
| Restore | `docker compose -f deploy/docker-compose.yml exec -T postgres pg_restore -U jejak -d jejak_inderasakti --clean < backups/pg/<file>.dump` |
| Roll back migrations by one | `make db-rollback STACK_FILE=deploy/docker-compose.yml` |
| Stop the backend stack | `make down` |
| Stop the frontend stack | `cd ../jejak-inderasakti-fe && make down` |

Take a `make backup` before every production deploy. Never run `make db-refresh` or `make dev-reset` against a real stack: they erase data.

## Notes

- Rate limit: 10 join attempts per IP per minute (`RATE_LIMIT_JOIN_PER_MIN`). The API trusts `X-Forwarded-For` only because it is reachable exclusively through Caddy.
- Redis is disposable. If it restarts, rankings and rate limits rebuild from memory and Postgres.
- Postgres, Redis and the API publish no ports; only Caddy is exposed.
- The certificate and account key live in the `caddy_data` volume. Do not delete it, or Caddy will request a new certificate (Let's Encrypt has rate limits).
- `jejak_net` is an externally-created Docker network (`make deploy-network`), not owned by either
  compose file, so `docker compose down` in either repo never deletes it and the two stacks can be
  deployed and rolled back independently.
