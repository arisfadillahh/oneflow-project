# Oneflow Dashboard Web

Deployment source and mandatory release checks: [`docs/production-deployment-guideline.md`](docs/production-deployment-guideline.md).

This is the active standalone frontend repository. The archived frontend under `../archive/urus.in/apps/dashboard-web` is legacy and must not be used for deployment. Preserve pending changes here and follow the deployment guideline above.

## Boundary

- Browser code calls `app-backend` only.
- Browser code must not call `ai-service`, internal WA gateway routes, databases, or workers.
- `app-backend` owns auth, organization membership, tenant isolation, AI credit billing, WhatsApp rules, and AI service calls.
- API/WebSocket/WA base URL behavior lives in `lib/runtime-config.mjs`.
- Official WhatsApp signup is coexistence-only. `lib/meta-coexistence.mjs` validates server configuration and trusted completion events, and rejects non-coexistence completion without suggesting account deletion or data reset. Provider cancellation, errors and incomplete asset IDs terminate the pending flow. Tests run with `npm test`.
- WhatsApp management is official-only, including mobile. No unofficial provider selector, legacy QR modal, or legacy group-binding wizard is exposed. Disconnected official sessions remain visible for reconnect instead of silently consuming plan capacity. A failed signup does not automatically delete a session that may already have completed at Meta.
- Dark dashboard/login muted text and primary labels have contrast regression checks in `test/dark-mode-contrast.test.mjs`; browser QA must still check rendered states and viewport layouts.
- API route-family coverage lives in `contracts/dashboard-api.json`.
- Agent creation templates define customer-facing tone, salutation, and natural opening rules inside the generated agent system prompt; backend runtime guardrails must not replace that persona.
- Agent creation onboarding is a conversational guided wizard layered over the existing structured setup fields. It must keep writing template, business context, admin-escalation, and tool setup into the same create-agent flow instead of storing raw onboarding chat as agent truth.

## Local Run

```powershell
npm install
Copy-Item .env.example .env.local
npm run dev -- -p 3000
```

Set `NEXT_PUBLIC_API_BASE_URL` only when the browser should call a remote backend directly. Leaving it empty uses `http://localhost:8080` when the dashboard runs on localhost or `127.0.0.1`.

## Checks

```powershell
npm run test
npm run build
```

`npm run test` runs Node unit tests plus split-readiness checks that reject hardcoded production dashboard navigation, direct browser-to-AI runtime references, undocumented dashboard API path prefixes, and missing standalone env keys.
