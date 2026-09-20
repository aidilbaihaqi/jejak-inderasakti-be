# Postman

## Setup
1. `make dev-up` (API on `http://localhost:8080`).
2. In Postman: **Import** both files in this folder, then select the environment **Jejak - Local (Docker)**.
3. Set `host_email` / `host_password` in the environment to a seeded host account.

## What is automated
| Feature | Where |
|---|---|
| Auto-login: fetches `host_token` when it is empty or has < 60 s left; decodes `exp` from the JWT | collection pre-request script |
| Bearer auth on every request except those marked *No Auth* | collection auth uses `{{host_token}}` |
| Every response: < 300 ms, and errors must carry `{code, message}` | collection test script |
| `Login` saves `host_token`, `host_token_exp` | Auth folder |
| `Create room` saves `room_id`, `pin` | Rooms folder |
| `Join room` generates a random `nickname`, saves `player_token` | Rooms folder |
| `Search schools` saves `school_id` | Schools folder |
| Negative cases (wrong password, no auth, unknown PIN, nickname taken) assert the contract error codes | each folder |

## Run the whole flow
Collection Runner: run **Auth**, then **Rooms** top to bottom. Or with Newman:

```bash
npx newman run postman/jejak-inderasakti.postman_collection.json -e postman/jejak-local.postman_environment.json
```

## Status
Only `Healthz` works today. The other requests follow `contracts/openapi.yaml` and will pass as the endpoints are implemented (host auth and rooms/join/schools are planned for Sunday 20/9). No host account is seeded yet.

## Keeping it in sync
When an endpoint is added or changed, update the matching request, its test script and any saved variables here (see `CLAUDE.md`, Workflow rules).
