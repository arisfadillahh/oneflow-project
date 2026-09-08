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

- Change: replaced the clipped horizontal Inbox status strip with a stable three-plus-two mobile grid.
- Local checks: frontend tests passed 41/41 and the Next.js production build passed.
- Frontend image: `oneflow-dashboard-web:inbox-mobile-filters-20260908`, image ID `sha256:a6954a234202ddd076e65a8568b376f9013087e2ee977718762c5f95febcff47`.
- Compose backup: `/home/goffath/oneflow-deploy/docker-compose.before-inbox-mobile-filters-20260908.yml`.
- Production QA: dashboard container healthy, `/dashboard/inbox` returned HTTP 200, and all five filters were visible without horizontal overflow in the narrow dark-mode viewport.

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
