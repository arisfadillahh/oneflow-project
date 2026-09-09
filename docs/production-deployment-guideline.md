# Oneflow Frontend Deployment Guideline

Use this file for every dashboard deployment.

## Source of truth

- Repository: `https://github.com/arisfadillahh/oneflow-project`
- Local path: `C:/Users/Aris Fadillah/Documents/Projects/Oneflow/oneflow-platform`
- Dashboard build context: `frontend/`; backend build context: `backend/`.
- The frontend copy under `../archive/urus.in/apps/dashboard-web` is legacy and must not be used for dashboard deployment.
- Preserve the working tree. Review `git status --short` and include intentional uncommitted changes in the release manifest before building.

## Required checks

1. Record the current production image ID, image digest and compose image reference.
2. Run `npm ci` and `npm test` from this repository.
3. Run `npm run build` and confirm the build completes without errors.
4. Build with the repository `Dockerfile`, which also uses `npm ci`.
5. Use a new immutable image tag containing the date and release purpose. Never overwrite an existing rollback tag.
6. Back up the active compose file before changing its dashboard image.
7. Recreate only the dashboard container. Keep backend, AI, worker, gateway and data volumes unchanged.
8. Verify container health, build ID, public `/dashboard/inbox`, `/agents/setup`, `/whatsapp`, and `/health` after deployment.
9. Test desktop and mobile layout, dark mode, chat-first onboarding, official WhatsApp coexistence entry point, and Meta template controls before calling the release ready.
10. Record the deployed image ID and rollback compose path in the release notes.

## Release record

Every deployment must record:

- source repository and exact commit or working-tree manifest;
- Node/npm versions and lockfile hash;
- test and build results;
- image tag, image ID and digest;
- remote compose backup path;
- health and route verification results;
- remaining known differences from the previous production image.

Do not call a release synchronized only because the page returns HTTP 200. Compiled assets, feature behavior and responsive UI must be checked.

## Release record: Instagram add-account discoverability 2026-09-09

- Scope: kept `Tambah akun Instagram` visible to organization admins after an Instagram account is already connected; the AI assignment and Meta login form opens inline and can be cancelled without navigation.
- UX verification: account `qwe` showed the action with connected `@itsoneflow`; opening the action displayed the AI selector, `Login dan tambahkan akun`, and `Batal`; cancelling restored the connected-account list.
- Verification: frontend tests passed 45/45, production build passed with 25 routes, and no backend service was changed or recreated.
- Frontend image: `oneflow-dashboard-web:channels-instagram-add-v10-20260909`, image ID and local digest `sha256:ca960ff05ca11ba26ee4ca09df9338eae9132e903cd88981abd18399615f85ef`.
- Compose backup: `/home/goffath/oneflow-deploy/docker-compose.before-channels-instagram-add-v10-20260909.yml`.
- Production checks: container healthy; `/channels` and `/health` returned HTTP 200.

## Release record: single-page Channels overview 2026-09-09

- Scope: replaced the channel-tab interaction with one connection overview that shows WhatsApp, Instagram and TikTok statuses together. No setup panel is open by default.
- Beginner flow: users only need to understand whether a channel is active; clicking WhatsApp or Instagram expands its settings inline on the same page, and `Tutup` returns to the overview.
- Behavior preserved: official Meta connection, AI assignment, Instagram validation/disconnect, WhatsApp session management, templates, package limits and recovery actions remain available. No backend service was changed or recreated.
- Verification: frontend tests passed 45/45, production build passed with 25 routes, and `npm audit --audit-level=low` reported zero vulnerabilities.
- Frontend image: `oneflow-dashboard-web:channels-overview-v8-20260909`, image ID and local digest `sha256:10dd9314cabe2b8683c299ab1b0a427dccdaa6f577537e91fc27e4f4a2919347`.
- Compose backup: `/home/goffath/oneflow-deploy/docker-compose.before-channels-overview-v8-20260909.yml`.
- Production QA: container healthy; `/health`, `/channels`, and `/dashboard/inbox` returned HTTP 200; account `qwe` showed both connected channels in one overview; Instagram settings expanded inline and collapsed through `Tutup` without navigation.

## Release record: novice-friendly Channels UX 2026-09-09

- Scope: replaced simultaneous WhatsApp/Instagram setup panels and the global WhatsApp progress strip with an explicit channel picker and one setup flow at a time.
- Behavior preserved: official Meta connection, AI assignment, Instagram validation/disconnect, WhatsApp session management, templates, package limits and recovery actions remain available.
- UX changes: plain-language labels, a single primary action, connected-account summaries, technical WhatsApp session controls inside an expandable management section, and responsive stacked controls on narrow screens.
- Verification: frontend tests passed 45/45, production build passed with 25 routes, and `npm audit --audit-level=low` reported zero vulnerabilities.
- Frontend image: `oneflow-dashboard-web:channels-ux-v7-20260909`, image ID `sha256:c588346fcd1f4025a5e020cb301aa718d871c32e0315279271f0ff9e59c95982`.
- Compose backup: `/home/goffath/oneflow-deploy/docker-compose.before-channels-ux-v7-20260909.yml`.
- Production QA: container healthy; `/channels` and `/health` returned HTTP 200; account `qwe` could switch between WhatsApp and Instagram panels; connected `@itsoneflow` rendered correctly; light and dark themes were visually checked.

## Release record: frontend security patches 2026-09-09

- Scope: upgraded Next.js from `15.5.18` to `15.5.25`, PostCSS to `8.5.23`, and the transitive Sharp runtime to `0.35.4` without changing application behavior.
- Security verification: `npm audit --audit-level=low` reports zero vulnerabilities, down from one critical and three high-severity findings.
- Functional verification: frontend tests passed 44/44 and the production build passed with 25 routes.
- Frontend image: `oneflow-dashboard-web:security-patches-v6-20260909`, image ID `sha256:8e060eb4967bd3cede6f44a4be418c3087a79abac57bfc640cd9a245e2dac7d5`.
- Compose backup: `/home/goffath/oneflow-deploy/docker-compose.before-security-patches-v6-20260909.yml`.
- Production checks: dashboard container healthy; `/`, `/dashboard/inbox`, `/channels`, and `/health` returned HTTP 200. Backend and other services were not recreated.

## Release record: Instagram Inbox and permission validation 2026-09-09

- Repository: `https://github.com/arisfadillahh/oneflow-project`; deployment used the intentional Instagram working tree from the monorepo source of truth.
- Scope: official Instagram Login, encrypted session storage, signed webhooks, shared multi-channel Inbox, Instagram-only Inbox readiness, and an admin connection check for `instagram_business_basic` plus `instagram_business_manage_messages`.
- Local checks: frontend tests passed 44/44; frontend production build passed with 25 routes.
- Ubuntu checks: Go formatting and `go test ./apps/app-backend/internal/httpapi` passed; backend and frontend Docker builds passed.
- Backend image: `oneflow-app-backend:instagram-permission-validation-20260908`, image ID `sha256:6ec4f3d309784a1cffe3e51c7378bb53f3b065409cbaff63f54e76ae8f2c06d6`.
- Frontend image: `oneflow-dashboard-web:instagram-inbox-v5-20260909`, image ID `sha256:91fd99e1019664085eb587d32a41268b2c4f014525fcaa1435d20484563b2396`.
- Compose backups: `/home/goffath/oneflow-deploy/docker-compose.before-instagram-permission-validation-20260909.yml` and `/home/goffath/oneflow-deploy/docker-compose.before-instagram-inbox-v5-20260909.yml`.
- Production checks: containers healthy; `/dashboard/inbox`, `/channels`, and `/health` returned HTTP 200; account `qwe` listed connected `@itsoneflow`; server-side profile and messaging validation returned HTTP 200.
- Meta status: both Instagram permissions remain at standard access while the existing WhatsApp submission is in review. Meta's API-call counter still shows zero immediately after successful calls and may update asynchronously; real non-role Instagram DMs remain unavailable while the app is in Development mode.
- Known risk: `npm ci` reports four high-severity dependency advisories. They were not auto-fixed during this release because dependency changes require a separate compatibility review.

## Release record: Inbox read state and bilingual shell 2026-09-08

- Repository: `https://github.com/arisfadillahh/oneflow-project`.
- Scope: per-agent Inbox read receipts, unread filter and badge, Indonesian/English account preference, compact WhatsApp 24-hour status, and dark-mode Inbox contrast.
- Local checks: frontend tests passed 41/41; frontend production build passed with 25 routes.
- Ubuntu checks: Go formatting and `go test ./...` passed in `golang:1.24`; both production Docker builds passed.
- Backend image: `oneflow-app-backend:inbox-i18n-20260908`, image ID `sha256:3d2cd66bae2f3a962e05a541c9bea4cec2b0484538a5bc8541c215bb17c5a6eb`.
- Frontend image: `oneflow-dashboard-web:inbox-i18n-v2-20260908`, image ID `sha256:9e1cfe28411cc2ee78db0774061be06175bbe35c395c62a27a4aa272d29b1d20`.
- Compose backups: `/home/goffath/oneflow-deploy/docker-compose.before-20260908-inbox-i18n.yml` and `/home/goffath/oneflow-deploy/docker-compose.before-inbox-i18n-v2-20260908.yml`.
- Database backup: `/home/goffath/oneflow-deploy/oneflow-before-20260908-inbox-i18n.dump`.
- Migration verification: `agents.preferred_locale` and `conversation_reads` exist.
- Browser QA: opening a chat cleared its per-agent unread badge; English persisted after reload; the test account was restored to Indonesian; narrow-screen dark Inbox and the compact WhatsApp window status rendered without overlap.
- Public route checks: `/`, `/dashboard/inbox`, `/dashboard/account`, `/channels`, `/whatsapp/templates`, and `/health` returned HTTP 200; unauthenticated `/api/me` correctly returned HTTP 401.

## Release record: Mobile Inbox filters 2026-09-08

- Change: replaced the clipped horizontal Inbox status strip with a stable three-plus-two grid based on the Inbox column width, including narrow desktop panels.
- Local checks: frontend tests passed 41/41 and the Next.js production build passed.
- Frontend image: `oneflow-dashboard-web:inbox-mobile-filters-20260908`, image ID `sha256:a6954a234202ddd076e65a8568b376f9013087e2ee977718762c5f95febcff47`.
- Compose backup: `/home/goffath/oneflow-deploy/docker-compose.before-inbox-mobile-filters-20260908.yml`.
- Production QA: dashboard container healthy, `/dashboard/inbox` returned HTTP 200, and all five filters were visible without horizontal overflow in the narrow dark-mode viewport.
- Final frontend image: `oneflow-dashboard-web:inbox-panel-filters-20260908`, image ID `sha256:5673f5407f9b` (full digest recorded by Docker on the host).

## Release record: 2026-09-07

- Source: `oneflow-frontend` working tree used for the verified local build.
- Local checks: `npm test` passed 33/33; `npm run build` passed.
- Remote Docker build: passed with Next.js 15.5.18 and `npm ci`.
- Active image: `oneflow-dashboard-web:release-20260907`.
- Image ID: `sha256:7af50e4c56586b7ee8d6cef7cb3b582988a6c6cc9094324d9dc994f3f6e8c837`.
- Build ID: `76Qr9WdvKpzPsnZFX42JX/`.
- Compose backup: `/home/goffath/oneflow-deploy/docker-compose.before-release-20260907.yml`.
- Container health: healthy.
- Public route checks: `/`, `/dashboard/inbox`, `/agents/setup`, and `/whatsapp` returned HTTP 200.
- Previous image reference remains in the compose backup for rollback.

## Release record: monorepo 2026-09-07

- Repository: `https://github.com/arisfadillahh/oneflow-project`
- Commit: `9293042bb019dec9d60e9c1ac53661f51ecfe20a`
- Build context: `frontend/` inside the monorepo.
- Remote image: `oneflow-dashboard-web:monorepo-20260907`.
- Image ID: `sha256:3f14b9767b5caee6766c83bf0f7d6164f4be0e50146b0d6844d60744a8e6e73c`.
- Container health: healthy.
- Compose backup: `/home/goffath/oneflow-deploy/docker-compose.before-monorepo-20260907.yml`.
- Public route checks: `/`, `/dashboard/inbox`, `/agents/setup`, and `/whatsapp` returned HTTP 200.
- Backend, AI, gateway, worker, database, Redis and MinIO were not recreated.

## Release record: channel navigation fix 2026-09-07

- Repository: `https://github.com/arisfadillahh/oneflow-project`
- Commit: `92599d5`
- Change: fixed Channels navigation permissions and replaced generic setup copy that incorrectly told users to connect WhatsApp.
- Local checks: `npm test` passed 33/33; `npm run build` passed with 24 routes.
- Remote image: `oneflow-dashboard-web:channel-fix-20260907`.
- Remote image ID: `sha256:c03ba9069ce2e40b0e40deab3a5e69ef639b773e7a5ad5a5080bf58d2427ebb2`.
- Container health: healthy.
- Public route checks: `/channels`, `/whatsapp`, and `/dashboard/inbox` returned HTTP 200; `/channels` page marker was present.
- Compose backup: `/home/goffath/oneflow-deploy/docker-compose.before-channel-fix-20260907.yml`.

## Release record: channels center 2026-09-07

- Repository: `https://github.com/arisfadillahh/oneflow-project`
- Commit: `f206eda`
- Change: renamed the dashboard connection experience to `/channels`, added Instagram and TikTok upcoming states, and kept `/whatsapp` as a compatibility route.
- Local checks: `npm test` passed 33/33; `npm run build` passed with 24 routes.
- Remote image: `oneflow-dashboard-web:channels-route-20260907`.
- Remote image ID: `sha256:f27aaaa37b2f387afc585f94da362ff95b84c7c80c5832a0455763a0f03a183b`.
- Container health: healthy.
- Compose backup: `/home/goffath/oneflow-deploy/docker-compose.before-channels-route-20260907.yml`.
- Public route checks: `/`, `/dashboard/inbox`, `/channels`, `/whatsapp`, and `/agents/setup` returned HTTP 200.
- Backend, AI, gateway, worker, database, Redis and MinIO were not recreated.
