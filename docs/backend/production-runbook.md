# oneflow.id Production Runbook

## What is covered

This repo can be prepared for production without a public domain or live WhatsApp scan by tightening local Docker defaults, creating production env templates, and adding operational scripts. Public HTTPS URLs, reverse proxy routing, and real WhatsApp pairing still need the deployment target.

## First setup

1. Copy every production template:

```powershell
Copy-Item .env.prod.infra.example .env.prod.infra
Copy-Item .env.prod.app-backend.example .env.prod.app-backend
Copy-Item .env.prod.wa-gateway.example .env.prod.wa-gateway
Copy-Item .env.prod.ai-service.example .env.prod.ai-service
Copy-Item .env.prod.worker.example .env.prod.worker
```

2. Replace all placeholder values. Do not reuse the local `.env.*` secrets for production.
3. Set public frontend origins in `WEBSOCKET_ALLOWED_ORIGINS` and `WA_ALLOWED_ORIGINS` once the frontend URL exists.
4. Run preflight:

```powershell
.\scripts\preflight.ps1
```

For local rehearsal before the public URLs exist:

```powershell
.\scripts\preflight.ps1 -AllowLocalUrls
```

## Run

```powershell
docker compose -f docker-compose.prod.yml build
docker compose -f docker-compose.prod.yml up -d
```

The production compose binds backend and WA gateway to `127.0.0.1` only. Postgres, Redis, and MinIO are not published to the host. Put a reverse proxy or tunnel in front of ports `8080` and `8090`; deploy the dashboard from the frontend repository.

## Backup

Local Docker stack:

```powershell
.\scripts\backup-postgres.ps1
```

Production compose stack:

```powershell
.\scripts\backup-postgres.ps1 -Container oneflow-postgres -Username oneflow_app
```

Restore requires an explicit `-Force`:

```powershell
.\scripts\restore-postgres.ps1 -Path .\backups\oneflow-YYYYMMDD-HHMMSS.sql -Container oneflow-postgres -Username oneflow_app -Force
```

## Internal Naming Cutover

Migrations `051_oneflow_identity_values.sql` through `054_oneflow_object_storage_urls.sql` change active identity/resource vocabulary and stored object URLs to Oneflow names. Do not start the renamed Compose project against an existing legacy deployment without a stateful backup and restore plan.

1. Create a database dump and inventory object-storage and Redis persistence before stopping the legacy containers.
2. Start the `oneflow` infrastructure resources on new volumes, restore database/object data, and keep the previous volumes untouched as rollback material.
3. Recreate application services only after their env files point at the `oneflow` database and bucket.
4. Verify migrations `051` through `054`, service health, dashboard login, inbound customer messages, and object retrieval before retiring any legacy resource.

Legacy payload and browser-session reads remain compatible during this cutover; all new writes use the Oneflow names.

## Smoke test

After the backend stack is running, run:

```powershell
node .\scripts\security-smoke.mjs
```

The smoke test checks backend health, auth gates, CORS rejection, internal WA auth, and sensitive-path exposure.

## Remaining production gates

- Public HTTPS origins for frontend, API/WebSocket, WA gateway, and object file access.
- Secret rotation for any real secrets that have lived in local files.
- Real WhatsApp pairing and reconnect test from the deployed host.
- External backup storage and restore drill.
- Monitoring target for container restarts, failed jobs, and low wallet balance.
