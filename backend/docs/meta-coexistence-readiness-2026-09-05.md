# Meta Coexistence Readiness

Historical inspection snapshot. Deployment and official-only retirement updates are recorded in [the 2026-09-06 QA report](official-only-dark-mode-qa-2026-09-06.md). Statements below about deployment status describe the earlier inspection, not the current testing server.

## Product Decision

User confirmed official WhatsApp Business App coexistence on 2026-09-05. Customers must not be instructed to delete their WhatsApp Business account, reset data, or unregister their existing number. Do not silently fall back to standalone Cloud API migration when coexistence is unavailable. Explain eligibility failures and leave the existing account untouched.

Preserving the Business App account is distinct from importing existing chats into Oneflow. History synchronization requires its own consent, eligibility checks and verified result; a successful connection must not imply that every old chat was imported.

## Account Inspection

Read-only inspection of the logged-in Meta Developers account:

- Oneflow app ID: `1460232068764576`.
- App mode: Development.
- Business verification: Approved.
- Tech Provider onboarding page: 1 of 2 steps complete; App Review remains a gate.
- Signup configuration ID: `1768328367945778`.
- Existing hosted signup link uses `featureType=whatsapp_business_app_onboarding`, session info version 3, and redirect `https://oneflow.id/dashboard`.
- Configuration access token: system-user, expires after 60 days. Confirmed from selected, disabled controls, not just the configuration name.
- JavaScript SDK domain: `https://oneflow.id/`; configured OAuth redirect: `https://oneflow.id/dashboard`.
- Business Login settings showed empty deauthorization and data-deletion callback fields. Basic app data-deletion settings were not inspected; do not infer that all deletion mechanisms are absent.
- No account settings were saved; no live-mode switch, permissions submission or real customer signup was performed.

## Local Changes And Verification

- Meta webhook subscription now requires a parsed `success: true` response, not merely a 2xx HTTP status.
- TDD: rejection, missing confirmation and malformed response tests failed before the fix and passed after it. Success and permission-denied responses are also covered.
- Booking writes serialize on the service row and enforce opening days/hours, future time and buffer conflicts. Updates re-read appointment state after waiting for the service lock.
- App backend Go run: 51 passing test/subtest events; 15 database-dependent tests skipped locally. This is not a full database integration pass.
- WhatsApp gateway `go test ./...`: passed locally.
- Required-database mode was verified to fail instead of skip when the DSN is absent.
- CI now provisions an isolated test database and executes offline persona tests. The updated GitHub workflow has not run yet.
- No Ubuntu changes, production deploy, live WhatsApp send or paid model calls in this continuation.

## Remaining Gates

1. Verify advanced access for whatsapp_business_messaging and whatsapp_business_management, prepare truthful messaging/template demo evidence, and complete App Review. Business verification alone is not full Tech Provider readiness.
2. Token-to-WABA-to-phone lookup is implemented and mock-tested before persistence. Cross-tenant phone binding and signup replay protections still need verification.
3. Track token expiration and reconnection before the configured 60-day expiry. Never store tokens in frontend code or commit secrets.
4. Verify actual webhook callbacks, signature validation, subscription fields, and coexistence history events end to end on a consenting test account.
5. Frontend completion is now restricted to FINISH_WHATSAPP_BUSINESS_APP_ONBOARDING. Cancel, error, incomplete IDs and non-coexistence completion reject promptly; timeout and synchronous SDK failure clean up the listener. Real Meta popup and eligibility behavior still require end-to-end verification.
6. Test preservation of Business App operation separately from optional Oneflow history import. Do not certify data preservation or complete history sync without the real end-to-end test.
7. Run the new database CI, including concurrent booking tests, before treating booking concurrency as production-verified.
8. Continue the earlier application audit fixes: onboarding partial-save recovery, organization-wide tool permission preservation, follow-up dispatch, password recovery and misleading demo CTA. These are not completed by this Meta inspection.

## Evidence

### Continuation Verification

- Frontend `npm test`: 25 passed, including 4 new coexistence configuration/event test groups; split contract checks passed.
- Frontend `npm run build`: passed. Local preview started on port 3015.
- Backend `go test ./...`: passed for runnable local tests; database tests still skip without the dedicated test DSN.
- Ownership tests cover matching and mismatched numbers, denied access, malformed responses, later-page matches and invalid pagination. Pagination reconstructs the configured Graph endpoint instead of following next URLs with a bearer token. Lookup is bounded by page count and timeout.
- Tests were introduced before the helper implementations; initial runs failed because helpers did not yet exist, then passed after implementation. This is not a real Meta API integration test.
- Backend graph asset IDs are restricted to numeric IDs before constructing Graph paths.
- No destructive WhatsApp API calls were added, and no provider account changes were submitted.

Endpoint reference: [Meta's official WhatsApp Cloud API collection](https://www.postman.com/meta/whatsapp-business-platform/documentation/wlk6lh4/whatsapp-cloud-api?entity=request-13382743-5a8389e6-5e86-48e6-85da-577d6e8d4474), WABA phone_numbers lookup.

Account pages inspected: https://developers.facebook.com/apps/1460232068764576/ and its WhatsApp Tech Provider / Facebook Login for Business settings and configuration editor. Account UI evidence is time-specific. No secrets are included in this document.
