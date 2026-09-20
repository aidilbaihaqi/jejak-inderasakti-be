# Deploying to staging / production

One VPS (Ubuntu LTS, 2 vCPU / 4 GB) runs four containers: **caddy** (HTTPS + static web), **api**, **postgres**, **redis**.
Design: `docs/06_Infrastructure_Deployment.md`. Local development uses `docker-compose.dev.yml` instead.

## 1. Prepare the server (once)

```bash
# Docker + compose plugin, git, make
sudo apt update && sudo apt install -y git make ca-certificates curl
curl -fsSL https://get.docker.com | sudo sh && sudo usermod -aG docker "$USER"   # log out and in again

# Firewall: SSH, HTTP (certificate + redirect), HTTPS/WSS only
sudo ufw allow 22/tcp && sudo ufw allow 80/tcp && sudo ufw allow 443/tcp && sudo ufw enable
```

DNS: create an **A record** for the hostname (for example `staging.jejak.example.id`) pointing at the server's public IP.
Caddy cannot obtain a certificate until it resolves and ports 80/443 are reachable.

## 2. Configure secrets

```bash
git clone https://github.com/aidilbaihaqi/jejak-inderasakti-be.git && cd jejak-inderasakti-be
cp deploy/.env.example deploy/.env
```

Edit `deploy/.env`: set `DOMAIN`, then generate the secrets on the server so they never leave it:

```bash
echo "POSTGRES_PASSWORD=$(openssl rand -hex 24)"
echo "JWT_SECRET=$(openssl rand -hex 32)"
```

`deploy/.env` is git-ignored. Use different secrets for staging and production.

## 3. Start and initialise

```bash
make deploy-check                       # validates compose + Caddyfile
make up                                 # builds and starts everything

# STACK_FILE points the db-* commands at this stack instead of the dev one
make db-migrate STACK_FILE=deploy/docker-compose.yml      # (the API also migrates on start)
make db-seed-strict STACK_FILE=deploy/docker-compose.yml  # loads the 66-question bank
make db-create-host STACK_FILE=deploy/docker-compose.yml EMAIL=panitia@example.id NAME="Panitia" PASSWORD='<at least 8 chars>'
```

The web build goes into `deploy/web-dist/` (or the folder in `WEB_DIST_DIR`); until then `/` shows a placeholder page.

## 4. Verify

```bash
curl -i https://$DOMAIN/api/healthz     # 200 {"status":"ok"} with a valid certificate
make ps STACK_FILE=deploy/docker-compose.yml
make logs STACK_FILE=deploy/docker-compose.yml
```

Then import `postman/` and set `base_url` to `https://<your domain>` to run the collection against staging.
WebSocket check: `wss://<your domain>/ws?token=...` (see `contracts/ws.md`).

## 5. Operate

| Task | Command |
|---|---|
| Deploy a new version | `git pull && make up` |
| Backup the database | `make backup` (writes `backups/pg/*.dump`; it reads `POSTGRES_USER` / `POSTGRES_DB` from the shell, defaults `jejak` / `jejak_inderasakti`) |
| Restore | `docker compose -f deploy/docker-compose.yml exec -T postgres pg_restore -U jejak -d jejak_inderasakti --clean < backups/pg/<file>.dump` |
| Roll back migrations by one | `make db-rollback STACK_FILE=deploy/docker-compose.yml` |
| Stop | `make down` |

Take a `make backup` before every production deploy. Never run `make db-refresh` or `make dev-reset` against a real stack: they erase data.

## Notes

- Rate limit: 10 join attempts per IP per minute (`RATE_LIMIT_JOIN_PER_MIN`). The API trusts `X-Forwarded-For` only because it is reachable exclusively through Caddy.
- Redis is disposable. If it restarts, rankings and rate limits rebuild from memory and Postgres.
- Postgres, Redis and the API publish no ports; only Caddy is exposed.
- The certificate and account key live in the `caddy_data` volume. Do not delete it, or Caddy will request a new certificate (Let's Encrypt has rate limits).
