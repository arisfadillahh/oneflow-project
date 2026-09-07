# Frontend API Reference

This document is the frontend-facing API reference for Oneflow.id. The machine-readable companion file is [`contracts/frontend-api.json`](../contracts/frontend-api.json).

Source of truth: `apps/app-backend`. Frontend must call `app-backend` only.

## Runtime Bases

| Environment | HTTP API base | WebSocket base |
| --- | --- | --- |
| Local backend | `http://localhost:8080` | `ws://localhost:8080/ws` |
| Production | `https://oneflow.id` | `wss://oneflow.id/ws` |

Frontend env:

| Variable | Purpose |
| --- | --- |
| `NEXT_PUBLIC_API_BASE_URL` | Optional browser API base. Empty means same-origin proxy in production and localhost backend on local loopback. |
| `NEXT_PUBLIC_WS_BASE_URL` | Optional browser WebSocket base. Empty means same-origin `/ws` in production and local backend WebSocket on local loopback. |
| `NEXT_PUBLIC_WA_GATEWAY_URL` | Legacy optional gateway base. Prefer `/api/whatsapp/*` through `app-backend`. |
| `API_PROXY_TARGET` | Server-side Next rewrite target for `/api/*` and `/health`. |

## Auth Contract

Most endpoints require:

```http
Authorization: Bearer <token>
Content-Type: application/json
```

Login and register return the same envelope:

```json
{
  "token": "jwt",
  "user": {
    "id": "agent-id",
    "name": "Owner",
    "username": "owner@example.com",
    "role": "super_admin",
    "organizationId": "org-id",
    "organizationRole": "org_owner",
    "organizationName": "Business",
    "organizationSlug": "business"
  }
}
```

Errors use:

```json
{ "error": "message" }
```

Common status codes: `400` validation, `401` invalid token, `402` plan or credit requirement, `403` forbidden, `404` missing route/resource, `409` conflict, `429` rate limited, `500/502` server/upstream failure.

## Boundary Rules

- Browser code may call `app-backend` and `/ws` only.
- Browser code must not call `ai-service`, `wa-gateway` internal routes, database, Redis, worker, `/api/internal/*`, or `/api/payments/midtrans/notification`.
- AI runtime calls must go through `app-backend` so tenant isolation, AI run logs, and credit billing are preserved.
- WhatsApp operations go through `/api/whatsapp/*`; do not expose internal gateway credentials.

## Public Endpoints

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/health` | Backend health: `{ "status": "ok" }`. |
| `GET` | `/healthz` | Same as `/health`. |
| `GET` | `/api/public/billing/packages` | Public landing pricing catalog. |
| `POST` | `/api/auth/login` | Login with `{ username, password }`. |
| `POST` | `/api/auth/register` | Register org owner with `{ name, username, password, phone?, organizationName, organizationSlug? }`. |
| `POST` | `/api/team/invites/accept` | Accept invite and receive login token. |
| `GET` | `/ws` | Realtime status snapshot stream. |

## Core Authenticated Endpoints

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/me` | Current user and active organization. |
| `GET` | `/api/organizations` | Organizations available to the current user. |
| `POST` | `/api/organizations/switch` | Owner-only organization switch. Returns replacement token. |
| `PUT` | `/api/account/profile` | Update profile. |
| `POST` | `/api/account/password` | Update password. |
| `GET` | `/api/dashboard/summary` | Dashboard overview. |
| `GET` | `/api/analytics/overview` | Analytics overview. |
| `GET` | `/api/notifications` | Notification feed. |
| `POST` | `/api/support/report` | Submit support report. |

## Team

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/team/members` | List team members. |
| `PUT` | `/api/team/members/{id}` | Update member. |
| `DELETE` | `/api/team/members/{id}` | Deactivate/remove member. |
| `GET` | `/api/team/invites` | List invites. |
| `POST` | `/api/team/invites` | Create invite. |
| `DELETE` | `/api/team/invites/{id}` | Cancel invite. |

## AI Configuration And Playground

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/agents` | Human/admin users. |
| `POST` | `/api/agents` | Create human/admin user. |
| `PUT` | `/api/agents/{id}` | Update human/admin user. |
| `POST` | `/api/agents/{id}/reset-password` | Reset password. |
| `GET` | `/api/ai-agents` | AI agent list. |
| `POST` | `/api/ai-agents` | Create AI agent. |
| `PUT` | `/api/ai-agents/{id}` | Update AI agent. |
| `DELETE` | `/api/ai-agents/{id}` | Delete AI agent. |
| `GET` | `/api/ai-settings` | Owner/admin AI settings. |
| `POST` | `/api/ai-settings` | Update AI settings. |
| `GET` | `/api/ai-models` | Allowed OpenRouter model choices. |
| `GET` | `/api/ai-runs` | AI run logs. |
| `GET` | `/api/ai-runs/{id}` | AI run detail. |
| `GET` | `/api/playground/shortcuts` | Playground shortcuts. |
| `POST` | `/api/playground/shortcuts` | Create shortcut. |
| `PUT` | `/api/playground/shortcuts/{id}` | Update shortcut. |
| `DELETE` | `/api/playground/shortcuts/{id}` | Delete shortcut. |
| `POST` | `/api/playground/run` | Run AI playground. This is billed through Oneflow credits. |

Playground body:

```json
{
  "aiAgentId": "agent-id",
  "customerName": "Tester",
  "messageText": "Halo, buka jam berapa?",
  "history": []
}
```

## Inbox And Conversations

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/inbox` | Conversation list. |
| `GET` | `/api/conversations/{id}` | Conversation detail, messages, events. |
| `POST` | `/api/conversations/{id}/takeover` | Human takeover. |
| `POST` | `/api/conversations/{id}/force-takeover` | Force human takeover. |
| `POST` | `/api/conversations/{id}/assign` | Assign conversation. |
| `POST` | `/api/conversations/{id}/return-to-ai` | Return conversation to AI. |
| `POST` | `/api/conversations/{id}/resolve` | Resolve conversation. |
| `POST` | `/api/conversations/{id}/manual-message` | Send manual WhatsApp message. |
| `POST` | `/api/conversations/{id}/workflow` | Update workflow/stage metadata. |

Manual free-form WhatsApp follow-up must respect Meta's 24-hour customer-service window. After that, the UI should guide admin to approved template send and show that Meta/BSP charges are external to Oneflow credits.

## Contacts, Follow-Up, Templates

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/contacts` | Contact list. |
| `GET` | `/api/contacts/{id}` | Contact detail. |
| `PATCH`/`PUT` | `/api/contacts/{id}` | Update contact. |
| `GET` | `/api/contact-tags` | Tags. |
| `POST` | `/api/contact-tags` | Create tag. |
| `GET` | `/api/follow-up-tasks` | Follow-up tasks. |
| `POST` | `/api/follow-up-tasks` | Create follow-up task. |
| `PATCH`/`PUT` | `/api/follow-up-tasks/{id}` | Update follow-up task. |
| `GET` | `/api/message-templates` | Internal CRM templates. |
| `POST` | `/api/message-templates` | Create internal template. |
| `PATCH`/`PUT` | `/api/message-templates/{id}` | Update internal template. |
| `DELETE` | `/api/message-templates/{id}` | Delete internal template. |
| `GET` | `/api/broadcasts` | Broadcast list. |
| `POST` | `/api/broadcasts` | Create broadcast. |

## Business Tools

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/business-tools` | Installed/available modules. |
| `POST` | `/api/business-tools/{moduleKey}/install` | Install a module. |
| `PATCH` | `/api/business-tools/{moduleKey}` | Update module status, AI mode, or settings. |

Known module keys: `prospects`, `commerce`, `booking`, `tickets`.

AI modes: `off`, `read`, `draft`, `action`. AI-powered assists may consume credits; normal CRUD does not.

## Commerce

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/commerce/products` | Products; query supports `query`, `status`. |
| `POST` | `/api/commerce/products` | Create product. |
| `PATCH` | `/api/commerce/products/{id}` | Update product. |
| `DELETE` | `/api/commerce/products/{id}` | Delete product. |
| `GET` | `/api/commerce/order-pipelines` | Order pipelines. |
| `POST` | `/api/commerce/order-pipelines` | Create order pipeline. |
| `PATCH` | `/api/commerce/order-pipelines/{id}` | Update order pipeline. |
| `POST` | `/api/commerce/order-stages` | Create order stage. |
| `PATCH` | `/api/commerce/order-stages/{id}` | Update order stage. |
| `DELETE` | `/api/commerce/order-stages/{id}` | Delete order stage. |
| `GET` | `/api/commerce/order-settings` | Order settings. |
| `PATCH` | `/api/commerce/order-settings` | Update order settings. |
| `GET` | `/api/commerce/order-drafts` | Order drafts. |
| `POST` | `/api/commerce/order-drafts` | Create order draft. |
| `PATCH` | `/api/commerce/order-drafts/{id}` | Update order draft. |

Product request fields: `sku`, `name`, `description`, `unitPrice`, `stockQuantity`, `lowStockThreshold`, `status`.

Order draft items:

```json
{
  "contactId": "contact-id",
  "customerName": "Customer",
  "fulfillmentType": "delivery",
  "items": [{ "productId": "product-id", "quantity": 2 }]
}
```

## Booking

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/booking/services` | Booking services; query supports `status`. |
| `POST` | `/api/booking/services` | Create service. |
| `PATCH` | `/api/booking/services/{id}` | Update service. |
| `GET` | `/api/booking/appointments` | Appointments. |
| `POST` | `/api/booking/appointments` | Create appointment. |
| `PATCH` | `/api/booking/appointments/{id}` | Update appointment. |

Appointment body includes `serviceId`, `customerName`, `customerPhone`, `scheduledStart`, `status`, `locationType`, `locationAddress`, and optional contact/conversation references.

## Deals / CRM

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/deals` | Deal board. |
| `POST` | `/api/deals` | Create deal. |
| `GET` | `/api/deals/{id}` | Deal detail. |
| `PATCH` | `/api/deals/{id}` | Update deal. |
| `POST` | `/api/deals/{id}/activities` | Add deal activity. |
| `PATCH` | `/api/deal-activities/{id}` | Update activity. |
| `GET` | `/api/deal-pipelines` | Deal pipelines. |
| `POST` | `/api/deal-pipelines` | Create pipeline. |
| `PATCH` | `/api/deal-pipelines/{id}` | Update pipeline. |
| `POST` | `/api/deal-stages` | Create stage. |
| `PATCH` | `/api/deal-stages/{id}` | Update stage. |
| `DELETE` | `/api/deal-stages/{id}` | Delete stage. |

## Tickets

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/tickets` | Ticket list. |
| `POST` | `/api/tickets` | Create ticket. |
| `GET` | `/api/tickets/{id}` | Ticket detail and comments. |
| `PATCH` | `/api/tickets/{id}` | Update ticket. |
| `POST` | `/api/tickets/{id}/comments` | Add ticket comment. |
| `GET` | `/api/tickets/stages` | Custom ticket stages. |
| `POST` | `/api/tickets/stages` | Create stage. |
| `PATCH` | `/api/tickets/stages/{id}` | Update stage. |
| `POST` | `/api/tickets/ai/triage` | AI triage from conversation/contact/ticket context. This is billed. |

Ticket AI triage body:

```json
{
  "conversationId": "conversation-id",
  "contactId": "contact-id",
  "ticketId": "ticket-id"
}
```

Only one of these IDs may be enough depending on available context, but frontend should pass the richest known context.

## Knowledge

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/knowledge/faqs` | FAQ knowledge. |
| `POST` | `/api/knowledge/faqs` | Create FAQ. |
| `PUT` | `/api/knowledge/faqs/{id}` | Update FAQ. |
| `DELETE` | `/api/knowledge/faqs/{id}` | Delete FAQ. |
| `GET` | `/api/knowledge/documents` | Documents. |
| `POST` | `/api/knowledge/documents` | Create text document. |
| `PUT` | `/api/knowledge/documents/{id}` | Update document metadata/text. |
| `DELETE` | `/api/knowledge/documents/{id}` | Delete document. |
| `POST` | `/api/knowledge/documents/{id}/process` | Queue/reprocess document. |
| `POST` | `/api/knowledge/documents/upload` | Upload file with `multipart/form-data`. |
| `GET` | `/api/knowledge-download?id={documentId}` | Download document. |
| `GET` | `/api/job-positions` | Job/position records. |
| `POST` | `/api/job-positions` | Create job/position record. |
| `PUT` | `/api/job-positions/{id}` | Update job/position record. |
| `DELETE` | `/api/job-positions/{id}` | Delete job/position record. |

## Billing

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/billing/wallet` | Wallet and plan. |
| `GET` | `/api/billing/usage-logs` | Usage logs; query supports `limit`. |
| `GET` | `/api/billing/usage-logs/{id}` | Usage log detail. |
| `GET` | `/api/billing/analytics` | Billing analytics. |
| `GET` | `/api/billing/pricing` | Pricing settings. |
| `POST` | `/api/billing/pricing` | Update pricing settings. |
| `GET` | `/api/billing/adjustments` | Credit adjustments. |
| `POST` | `/api/billing/adjustments` | Create adjustment. |
| `GET` | `/api/billing/packages` | Packages for dashboard checkout/admin. |
| `POST` | `/api/billing/packages` | Owner creates package. |
| `PUT` | `/api/billing/packages/{id}` | Owner updates package. |
| `DELETE` | `/api/billing/packages/{id}` | Owner deactivates package. |
| `GET` | `/api/billing/purchases` | Purchase history. |
| `POST` | `/api/billing/purchases` | Create checkout/purchase request. |
| `GET` | `/api/billing/purchases/{id}/invoice` | Invoice view. |
| `POST` | `/api/billing/purchases/{id}/confirm` | Owner confirms manual purchase. |
| `POST` | `/api/billing/purchases/{id}/cancel` | Cancel purchase. |

Checkout request:

```json
{
  "packageId": "package-id",
  "billingPeriod": "monthly",
  "paymentMethod": "virtual_account",
  "notes": "optional"
}
```

`billingPeriod` may be `monthly`, `annual`, or `one_time` when compatible with the selected package. Card checkout is not available.

## WhatsApp

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/whatsapp/sessions` | WhatsApp session list. |
| `POST` | `/api/whatsapp/sessions` | Create session. |
| `PUT` | `/api/whatsapp/sessions/{id}` | Rename session. |
| `DELETE` | `/api/whatsapp/sessions/{id}` | Soft-delete/disconnect session. |
| `POST` | `/api/whatsapp/sessions/{id}/default` | Set default session. |
| `GET` | `/api/whatsapp/sessions/{id}/status` | Proxied gateway status. |
| `GET` | `/api/whatsapp/sessions/{id}/qr` | Proxied gateway QR. |
| `POST` | `/api/whatsapp/sessions/{id}/connect` | Connect session. |
| `POST` | `/api/whatsapp/sessions/{id}/disconnect` | Disconnect session. |
| `GET` | `/api/whatsapp/sessions/{id}/meta-templates` | List Meta templates. |
| `POST` | `/api/whatsapp/sessions/{id}/meta-templates` | Submit Meta template. |
| `POST` | `/api/whatsapp/sessions/{id}/meta-template-send` | Send approved template. |
| `POST` | `/api/whatsapp/sessions/{id}/meta-complete` | Complete Meta Embedded Signup. |
| `GET` | `/api/whatsapp/meta/config` | Meta frontend config. |

Create session:

```json
{
  "label": "Admin WA",
  "mode": "real",
  "provider": "whatsmeow",
  "aiAgentId": "ai-agent-id"
}
```

Provider can be `whatsmeow` or `meta_cloud` when enabled.

## Escalation And Group Notification

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/escalation-group` | Current escalation group binding. |
| `DELETE` | `/api/escalation-group` | Remove binding. |
| `POST` | `/api/escalation-group/bind-code` | Generate binding code. |
| `GET` | `/api/group-notification-rules` | Notification rules. |
| `PUT` | `/api/group-notification-rules` | Replace/update rules. |
| `POST` | `/api/group-notification-rules/test` | Send test notification. |
| `POST` | `/api/group-notification-rules/curation` | Update curated defaults. |

## Dev/Internal Routes

Do not call these from production frontend:

| Method | Path | Intended caller |
| --- | --- | --- |
| `POST` | `/api/dev/simulate-conversation` | Local development only. |
| `POST` | `/api/dev/simulate-inbound` | Local development only. |
| `POST` | `/api/internal/wa/inbound` | `wa-gateway` with internal token. |
| `POST` | `/api/internal/wa/echo` | `wa-gateway` with internal token. |
| `POST` | `/api/internal/wa/group-binding` | `wa-gateway` with internal token. |
| `POST` | `/api/payments/midtrans/notification` | Midtrans webhook only. |

## Cost Notes For UI

Show a confirmation or cost hint before these actions:

| Action | Cost |
| --- | --- |
| `/api/playground/run` | Oneflow AI credits. |
| `/api/tickets/ai/triage` | Oneflow AI credits. |
| WhatsApp inbound AI auto-reply | Oneflow AI credits, logged by backend. |
| Meta template send after 24-hour window | External Meta/BSP fee, separate from Oneflow credits. |
| Midtrans checkout | Payment gateway fee included in quote. |

Manual CRUD actions for CRM, tickets, commerce, booking, knowledge, team, and settings do not consume Oneflow AI credits.

## Implementation Checklist For Frontend

- Keep API base resolution in one runtime config module.
- Use relative URLs or env-derived API base; do not hardcode production dashboard URLs.
- Store JWT securely according to the frontend app policy and attach it to authenticated calls.
- Handle `402`, `403`, and `409` as user-facing product states, not generic crashes.
- Never show provider/internal secrets in UI or browser logs.
