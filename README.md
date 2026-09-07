# Oneflow Platform

Oneflow is maintained as one monorepo with clear service boundaries.

## Layout

- `frontend/`: Next.js public site and dashboard. Browser calls the backend API only.
- `backend/`: Go API, AI service, WhatsApp gateway, worker, migrations and backend checks.
- `infra/`: Docker Compose, Caddy and testing deployment support. Secrets stay outside Git.
- `docs/`: cross-service API, architecture, Meta coexistence, QA and deployment documentation.

The old standalone repositories remain available as migration references. The monorepo is published at `https://github.com/arisfadillahh/oneflow-project`.

## Channel UX

The dashboard uses a single Channel Center pattern at `/channels`. WhatsApp is the active production connector; Instagram and TikTok are represented as upcoming connectors until their official OAuth, permissions and webhook adapters are implemented. `/whatsapp` remains a compatibility route for existing bookmarks. Do not label an upcoming channel as connected or route messages to it before its backend adapter is live.

## Rules

- Frontend changes stay under `frontend/`.
- Backend changes stay under `backend/`.
- Frontend must not call AI, gateway, database or worker internals directly.
- Production dashboard deployments follow `docs/production-deployment-guideline.md`.
- Do not commit `.env` files, tokens, private keys, generated caches or node modules.

## Checks

```powershell
cd frontend
npm ci
npm test
npm run build

cd ..\backend
node scripts/verify-backend-standalone.mjs
python -m py_compile apps/ai-service/app/main.py apps/worker/app/main.py
```
