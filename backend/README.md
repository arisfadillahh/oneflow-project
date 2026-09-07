# Oneflow Backend

Backend services for Oneflow.id. The customer-facing dashboard and public site live in the separate frontend repository:

https://github.com/arisfadillahh/oneflow-frontend

This repository owns the API, AI runtime, WhatsApp gateway, worker jobs, database migrations, deployment support, and backend verification scripts.

## Services

| Path | Service | Responsibility |
| --- | --- | --- |
| `apps/app-backend` | Go API | Auth/RBAC, tenant isolation, billing and credits, dashboard APIs, WhatsApp ownership, AI run logging, business tool CRUD |
| `apps/ai-service` | FastAPI AI runtime | OpenRouter calls, knowledge retrieval/reranking, answer generation, tool-intent extraction, AI guardrails |
| `apps/wa-gateway` | Go WhatsApp gateway | Official Meta Cloud transport, coexistence lifecycle, verified webhooks, outbound text/media boundary |
| `apps/worker` | Python worker | Scheduled jobs, knowledge document processing, embedding sync, wallet and entitlement checks |
| `db/migrations` | SQL migrations | Schema, defaults, pricing/model/billing/business-tool source of truth |
| `ops` and `scripts` | Operations | Caddy routing, preflight, backup/restore, load/security checks, WhatsApp QA helpers |

Customer-facing tone, salutation, pronouns, and opening style belong to each agent's system prompt. Configured opening-style rules are treated as generation requirements, but the AI runtime must not inject fixed opening phrases or a fixed persona such as `Kak`, `Anda`, `aku`, or `saya`. When response generation cannot safely produce a customer-facing answer, including provider quota or configuration failures, the configured agent `fallbackWaitingMessage` is used. Provider diagnostics stay in internal logs and customer/API response metadata only exposes a stable error type; factual grounding and business-action validation remain deterministic runtime guardrails.

Grounded generation must preserve source entity/role labels, limits, units, and relationships instead of semantically relabeling them. High-confidence sources can skip the provider factual verifier only when the generated answer stays within the source scope; materially expanded answers are verified, and a verifier-requested regeneration gets one grounded revision pass before escalation.

Mixed-scope knowledge requests are preflighted against the configured agent system prompt. The supported business portion continues through knowledge retrieval, while explicitly unsupported portions are carried into generation and verification so the final answer does not leak code, tutorials, or other unsupported detail. When an approved scoped refusal is available, the final mixed-scope answer uses the selected primary source directly plus that refusal to prevent model-added business claims. Vague or overbroad refusals are rejected and rewritten through the configured agent tone so they name only the unsupported scope without platform-hardcoded customer copy. This boundary follows each agent's system prompt rather than globally disabling technical-support agents.

Knowledge retrieval supplements semantic candidates with lexical candidates when the semantic result set does not contain the user's preferred knowledge intent. Deterministic source selection then prioritizes the most specific matching intent, such as an explicit feature overview, before generation.

Short intent keywords are matched as complete tokens so unrelated words cannot trigger another business intent, and factual verification cannot approve an answer while reporting issues, missing facts, unsupported claims, or contradictions.

## Official WhatsApp Onboarding

The customer onboarding target is official WhatsApp Business App coexistence, not a destructive migration to standalone Cloud API. Keep the existing Business App account usable; never ask customers to delete their account, reset app data, or unregister their number as an onboarding workaround. Ineligible numbers must receive an actionable explanation, not an automatic migration fallback. Existing chat data in the Business App and optional history synchronization into Oneflow are separate concerns; do not promise complete history import without verifying Meta eligibility, consent, and sync results.

Account verification and remaining production gates are recorded in `docs/meta-coexistence-readiness-2026-09-05.md`.

As of 2026-09-06, unofficial WhatsApp is retired. The gateway does not initialize the legacy real or mock transport, regardless of WA_MODE. Old QR/pairing endpoints return HTTP 410. New backend sessions default to and only accept meta_cloud; inbound session resolution and WhatsApp plan capacity only use that provider. Historical session rows and conversations remain stored. Legacy group binding is disabled; group notification delivery is not supported by this integration. Run `scripts/qa-meta-testing-deploy.py` on the Ubuntu testing host for non-destructive official-only endpoint and Meta webhook checks. Full phone pairing remains a separate user-consent QA step.

## Integration Test Database

GitHub Actions provisions a dedicated pgvector/Postgres database, runs `go run ./cmd/testdb` from `apps/app-backend`, then runs Go tests with `APP_BACKEND_REQUIRE_DB_TESTS=true`. The initializer accepts only localhost and database `oneflow_test`, using `APP_BACKEND_TEST_DSN`; never point it at production. Without the required flag, local database tests may skip when no test DSN exists. CI also runs the offline AI persona tests, not only Python compilation.

## Frontend Contract

- Frontend code must call `app-backend`; it must not call `ai-service` directly.
- `ai-service`, Redis, Postgres, MinIO, and internal WA endpoints stay private behind backend-owned boundaries.
- Browser origins are configured with `WEBSOCKET_ALLOWED_ORIGINS` and `WA_ALLOWED_ORIGINS`; include the deployed frontend origin and local development origins as needed.
- Backend API changes should be documented clearly so the frontend repo can update against a stable contract.
- Frontend developer API reference: `docs/frontend-api-reference.md`.
- Machine-readable frontend API contract: `contracts/frontend-api.json`.

## Local Development

Create local env files from the production examples or existing deployment templates, then run the stack:

```powershell
docker compose up -d --build
```

Useful checks:

```powershell
node scripts/verify-backend-standalone.mjs
python -m py_compile apps/ai-service/app/main.py
python -m py_compile apps/worker/app/main.py
python -m unittest discover -s apps/ai-service/tests -p "test_*.py" -v
cd apps/app-backend; go test ./...
cd ../wa-gateway; go test ./...
```

Use the frontend repo separately for dashboard development. Point its API variables to this backend, for example local `http://localhost:8080` and `ws://localhost:8080/ws`.

For real local WhatsApp answer-quality QA, `scripts/setup_local_oneflow_wa_agent.py` keeps the paired dummy session and seeds Oneflow knowledge. It defaults the dummy agent to `openai/gpt-oss-120b:free`; override it with `ONEFLOW_LOCAL_WA_TEST_MODEL` when needed. Keep this local-only and send tests through the paired WhatsApp tester. The WA gateway allows the app backend enough time to finish multi-step AI generation and logs inbound-forwarding failures instead of dropping them silently.

## Production

Copy and fill the production env templates:

```powershell
Copy-Item .env.prod.infra.example .env.prod.infra
Copy-Item .env.prod.app-backend.example .env.prod.app-backend
Copy-Item .env.prod.wa-gateway.example .env.prod.wa-gateway
Copy-Item .env.prod.ai-service.example .env.prod.ai-service
Copy-Item .env.prod.worker.example .env.prod.worker
```

Run preflight before deployment:

```powershell
.\scripts\preflight.ps1
```

Production compose builds backend services only:

```powershell
docker compose -f docker-compose.prod.yml build
docker compose -f docker-compose.prod.yml up -d
```

`ops/Caddyfile.pc-server` routes backend-owned HTTP surfaces only: `/health`, `/healthz`, `/api/*`, `/api/meta/webhook`, and `/ws`. The frontend deployment should route public pages and dashboard traffic from its own repository.

## Verification

Before pushing backend changes, run the narrowest relevant checks:

```powershell
node scripts/verify-backend-standalone.mjs
python -m py_compile apps/ai-service/app/main.py
python -m py_compile apps/worker/app/main.py
```

For backend behavior changes, also run Go tests for the touched service and Docker builds for affected services:

```powershell
cd apps/app-backend; go test ./...
cd ../wa-gateway; go test ./...
docker compose build app-backend ai-service wa-gateway worker
```

When the stack is running, run:

```powershell
node scripts/security-smoke.mjs
LOAD_PROFILE=quick node scripts/load-test.mjs
```

Avoid high-volume AI or WhatsApp tests unless the test plan includes explicit credit and provider-cost limits.
