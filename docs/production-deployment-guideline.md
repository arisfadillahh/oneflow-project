# Oneflow Frontend Deployment Guideline

Use this file for every dashboard deployment.

## Source of truth

- Repository: `https://github.com/arisfadillahh/oneflow-frontend`
- Local path: `C:/Users/Aris Fadillah/Documents/Projects/Oneflow/oneflow-frontend`
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
8. Verify container health, build ID, public `/dashboard/inbox`, `/agents/setup`, `/whatsapp`, and `/api/health` after deployment.
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
