# Instagram Channel

Oneflow connects professional Instagram accounts through the official Instagram API with Instagram Login. The browser never receives or stores provider tokens.

## User flow

1. An organization admin opens `/channels`, selects an active AI agent and clicks **Hubungkan Instagram**.
2. App-backend creates a signed, one-time OAuth state and returns Meta's authorization URL.
3. Instagram redirects to `/api/instagram/oauth/callback`.
4. App-backend exchanges the code, encrypts the long-lived token with `META_CREDENTIAL_KEY`, and stores the connected account.
5. Signed Meta webhook events create or update an Instagram conversation in the shared Inbox.
6. After takeover, an agent can send a text reply from the dashboard. WhatsApp's 24-hour/template rule is not applied to Instagram.

## Server configuration

Keep these values only in the app-backend production environment:

```dotenv
INSTAGRAM_ENABLED=true
INSTAGRAM_APP_ID=
INSTAGRAM_APP_SECRET=
INSTAGRAM_REDIRECT_URI=https://oneflow.id/api/instagram/oauth/callback
INSTAGRAM_WEBHOOK_VERIFY_TOKEN=
INSTAGRAM_GRAPH_API_BASE_URL=https://graph.instagram.com
META_CREDENTIAL_KEY=
```

`META_CREDENTIAL_KEY` must be a base64-encoded 32-byte key. The webhook callback is `https://oneflow.id/api/instagram/webhook`. Meta must be configured with exactly the same verify token.

## API contract

- `GET /api/instagram/sessions`: list connected accounts for the signed-in organization.
- `POST /api/instagram/oauth/start`: organization admin starts login with `{ "aiAgentId": "..." }`.
- `GET /api/instagram/oauth/callback`: public provider callback protected by signed, expiring, one-time state.
- `GET|POST /api/instagram/webhook`: Meta verification and signed event receiver.
- `POST /api/instagram/sessions/{id}/validate`: organization admin verifies the stored profile and messaging permissions without exposing the encrypted token.
- `DELETE /api/instagram/sessions/{id}`: organization admin disconnects an account.

Tokens and app secrets must never be returned by these endpoints or committed to Git.

## Current supported surface

- Inbound and outbound text DM
- Shared Inbox, assignment, takeover and resolve flow
- Per-organization session isolation
- Sender profile name lookup when Meta permits it

Attachments, reactions, story replies and automatic AI replies require separate QA before enablement.
