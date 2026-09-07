# Official-Only WhatsApp and Dark Mode QA

## Scope

User requested removal of unofficial WhatsApp, retention of official Meta Business App Coexistence, dark-mode QA, and deployment to the Ubuntu testing server. Docker builds and service restarts were performed on Ubuntu, not on the Windows PC.

## Implemented

- Gateway initialization no longer starts the legacy real or mock transport, even when WA_MODE is set to real/mock. Existing unofficial sessions cannot reconnect or send messages through it.
- Legacy QR, QR-image, phone-pairing and connect endpoints return HTTP 410.
- Backend session creation defaults to meta_cloud and rejects unofficial provider requests. Inbound session resolution and WhatsApp plan capacity only include official sessions.
- Legacy group binding is disabled. Its QR/group modals and provider selector are removed from the connection screen. Inbox remains available for team handling; group delivery is not implemented for this official integration.
- Four legacy session rows were marked disconnected and non-default on the testing server. Conversation history, ownership and stored credentials were preserved; no customer data was deleted.
- Disconnected official sessions are displayed for recovery. They previously consumed plan capacity while being hidden from the table. Failed Meta signup no longer automatically deletes a potentially completed session.
- Dark dashboard/login muted colors were raised to readable contrast. Primary gradient buttons use dark text; select options use dark surfaces. Mobile WhatsApp access is enabled.
- The WhatsApp form uses three desktop columns and one mobile/tablet column, removing the retired provider column and its horizontal overflow. The informational workspace name is hidden in the tablet header to keep account controls on screen.

## Verification

- Frontend: 33 tests passed locally and on Ubuntu, including contrast, official-only UI, mobile layout, coexistence event validation, and session recovery source regressions. Source tests do not substitute for end-to-end pairing.
- Backend and gateway: go test ./... passed. Database-dependent backend tests are skipped without APP_BACKEND_TEST_DSN; this is not a complete database integration pass.
- Remote smoke: public page/login/health return 200; unauthenticated Meta config returns 401; invalid webhook verification returns 403; unsigned webhook returns 401; correctly signed empty webhook returns 200.
- Authenticated Meta config reports coexistence enabled with the expected app and configuration IDs. An unofficial session creation request returns 410 before creating data.
- All four retired gateway pairing endpoints return 410.
- Browser: dark mobile first-run guide, chat-first agent setup and Playground were visually checked. Setup dialog width was 374px with 372px scroll width at a 390px viewport. Chat timestamp computed color is #9aacc4 on #142238. Light-mode Playground was also inspected.
- Deployed WhatsApp browser QA: at 390px, main content client/scroll width is 385/385; at 820px, main content and header client/scroll width is 584/584. No page-level horizontal overflow remains in these tested states. The table retains its own horizontal scrolling; bottom notes remain above the mobile navigation.
- Testing accounts contain dummy QA data only. No paid model inference, real customer messages or real phone pairing were performed.

## Remaining Gates

- Real Meta popup completion, user consent, number eligibility, and phone pairing are NOT verified end-to-end. Do not advertise full customer onboarding readiness on these smoke tests alone.
- Last Meta account inspection showed Development mode and an outstanding App Review gate. Review/live access must be verified before broad customer rollout.
- Existing Business App history preservation and importing history into Oneflow are different. Do not promise full history import without actual consent/sync verification.
- The Ubuntu root disk is nearly full (about 95% at the last check). Expand storage and establish backup/monitoring before production reliance.
- Changes have not been committed or pushed to GitHub during this work.

## Recheck

Run `python3 /home/goffath/oneflow-coexistence-20260905/qa-meta-testing-deploy.py` on Ubuntu for non-destructive deployed endpoint checks. Its explicit --retire-legacy option changes only legacy session state and must not be confused with the read-only default QA run.
