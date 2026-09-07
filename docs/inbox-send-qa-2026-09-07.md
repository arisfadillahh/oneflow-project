# Inbox Takeover and Send QA

Production deployment: Ubuntu, 2026-09-07.

## Fixes

- Hide Take Over when the current user already owns a human conversation.
- Scope manual-message row locking to `FOR UPDATE OF c`. The previous unqualified lock failed on the lateral aggregate query.
- Remove the phone-based fallback lookup. Conversation ID and organization remain authoritative.

## Verification

- Frontend production build passed; frontend tests passed (33 tests).
- Corrected SQL executed against the production conversation in a rolled-back transaction.
- Chrome, qwe account: Return to AI displays Take Over and disables manual composition. Taking over restores composition and removes Take Over.
- Clicked Kirim in Inbox with text `p` to the authorized tester ending 2628 at 15:43:07 UTC. Message persisted as an agent outbound message with a Meta `wamid` and appeared in the Inbox timeline.
- Meta acceptance is verified; handset delivery is not independently confirmed in this report.

## Deployment

- Frontend image: `oneflow-dashboard-web:inbox-owner-v3`.
- Backend image: `oneflow-app-backend:inbox-fix2-20260907`.
- Compose: `/home/goffath/oneflow-deploy/docker-compose.yml`.
- Previous compose saved as `docker-compose.yml.before-inbox-owner-v3` beside the active file.

## Testing Credential Limitation

The old temporary Meta credential returned HTTP 401. It was refreshed through the logged-in Meta test setup and stored in server-only environment files with mode 0600, then encrypted through the existing test-connect endpoint for the qwe session. No credentials belong in Git.

Temporary test credentials can expire or be revoked. This successful test does not establish permanent production authorization or completion of Embedded Signup review.

## WhatsApp Template QA

- Template creation now surfaces the Meta Graph error message instead of hiding it behind a generic HTTP status.
- Body variables such as `{{1}}` receive generated example values in the Meta submission payload.
- The Channels form guides the user through submit, review, and test-send stages, with browser validation for names, language, body length, and recipient format.
- Production smoke test accepted `oneflow_dynamic_20260907` with status `PENDING` from the Oneflow Channels page.
- A Meta `REJECTED` status is a Meta policy/review result, not a transport failure; approved templates are required before test sending.
