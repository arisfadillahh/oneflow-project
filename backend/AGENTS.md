# Repository Guidelines

## Project Structure & Module Organization

This repository is the backend side of Oneflow.id. The dashboard/public frontend lives in `https://github.com/arisfadillahh/oneflow-frontend`. `apps/app-backend` is the Go API for auth, tenant data, billing, WhatsApp ownership, dashboard-facing APIs, and business tools. `apps/ai-service` is the FastAPI AI runtime. `apps/wa-gateway` is the Go WhatsApp gateway, and `apps/worker` handles scheduled Python jobs. SQL migrations live in `db/migrations`; deployment and support files live in `ops`, `scripts`, `docs`, and root `docker-compose*.yml` files.

## Build, Test, and Development Commands

- `docker compose up -d --build`: build and run the local backend stack.
- `docker compose build app-backend ai-service wa-gateway worker`: validate service Docker builds.
- `cd apps/app-backend && go test ./...`: run Go API tests.
- `cd apps/wa-gateway && go test ./...`: run WhatsApp gateway tests.
- `python -m py_compile apps/ai-service/app/main.py`: quick syntax check for the AI service.
- `python -m py_compile apps/worker/app/main.py`: quick syntax check for the worker.
- `node scripts/verify-backend-standalone.mjs`: ensure the backend repo has no frontend service/build leftovers.
- `node scripts/security-smoke.mjs`: run non-destructive backend security smoke checks against a running stack.
- `node scripts/load-test.mjs`: run load tests; use `LOAD_PROFILE=quick` for a shorter run.

## Coding Style & Naming Conventions

Use existing service conventions and keep changes scoped to the owning app. Format Go with `gofmt`; use idiomatic package names and table-driven tests. Python code uses 4-space indentation, type hints where helpful, and explicit config/env handling. JavaScript utility scripts use modern Node APIs, clear failure messages, and no committed secrets.

## Testing Guidelines

Add focused tests or smoke coverage for behavioral changes. Name Go tests `*_test.go`, Python tests `test_*.py`, and JS utility tests `*.test.mjs` if a runner is introduced. For AI, billing, WhatsApp, or business-tool changes, verify both the answer and persisted side effects through playground/API flows. Avoid high-volume AI playground or WhatsApp tests without explicit caps because they call providers and consume credits.

## Commit & Pull Request Guidelines

Use short imperative subjects such as `Prepare backend standalone repository`. Keep subjects concise, describe the user-visible or operational impact, and group related changes. Pull requests should include a summary, affected services, environment or migration notes, verification commands, and screenshots only when a frontend-facing behavior contract changes in a way the frontend repo must consume.

## Security & Configuration Tips

Never commit real secrets. Start from the `.env.*.example` files and keep production-only values outside images and source control. Tenant isolation, billing/credit logging, internal WhatsApp auth, and AI provider usage are security-sensitive paths; include smoke or targeted verification when touching them. The frontend should call `app-backend`; do not expose `ai-service` or internal gateway credentials to browser code.
