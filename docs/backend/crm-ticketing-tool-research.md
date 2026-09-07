# CRM & Ticketing Tool Research

Status: study only, no implementation yet.

## External References Studied

- SuiteCRM: mature enterprise CRM. Useful concepts: Accounts, Contacts, Leads, Opportunities, Cases, Tasks, Notes, Campaigns, Workflows, process audit, assignment, and case escalation.
- Frappe CRM: simpler modern CRM. Useful concepts: unified Lead/Deal page, Kanban stages, custom views, tasks, notes, call logs, lead-to-deal conversion, reminders, and unified timeline.

Both repositories are AGPL-3.0. Do not copy code or UI directly into Oneflow unless we are prepared to comply with AGPL obligations. Use them as product references only.

## What Oneflow Already Has

Oneflow is not starting from zero. Current foundation already includes:

- Contacts with lifecycle status, owner, notes, tags, import, memory notes, and detail side panel.
- Follow-up tasks tied to contacts.
- Message templates and broadcasts.
- Deals / sales pipeline with pipelines, stages, Kanban board, deal activities, owner, priority, value, expected close date, and stage move.
- Business tools ledger with installable modules: `prospects`, `commerce`, `booking`, `payments`.
- AI business-tool router already supports `prospects`, `commerce`, and `booking`.
- AI prospect tool can create a prospect/deal when the prospects module is active.
- Billing path already logs AI usage and deducts credits with a minimum of 1 credit per AI call when cost is non-zero or tiny.

So the gap is not missing tables. The gap is product cohesion: CRM, support problems, customer stage, and admin action tracking are split across Contacts, Follow-up, Inbox, and Deals.

## Recommendation

Do not import SuiteCRM or Frappe CRM as a dependency. Their stacks and licenses do not match Oneflow's current Next.js + Go + FastAPI + Postgres architecture, and a full CRM clone would add too much surface area.

Recommended direction:

1. Keep and upgrade `prospects` into `crm` / `CRM & Follow-up`.
2. Add a new business tool: `tickets` / `Customer Tickets`.
3. Make CRM and Tickets share one customer timeline model in the UI, while keeping separate database tables for sales pipeline vs support/problem cases.
4. Add AI only as optional actions with visible credit impact.

## Planning With WhatsApp / Meta Rules

Follow-up automation must respect the WhatsApp Business Platform customer-service window.

Core rule to implement:

- If customer sent the last inbound WhatsApp message less than 24 hours ago, Oneflow may send normal free-form follow-up replies from the WhatsApp session.
- If the last inbound WhatsApp message is more than 24 hours old, Oneflow must not send free-form text. It should require an approved Meta WhatsApp template.
- Outside the 24-hour window, show the admin that Meta/BSP message charges can apply. Since pricing is category, country, and delivery dependent, Oneflow should show this as an external WhatsApp charge estimate/warning, not as a guaranteed fixed price unless the active BSP/rate card is integrated.
- AI credits are separate from Meta WhatsApp messaging fees. AI draft/triage/suggestion consumes Oneflow credits; actual WhatsApp template delivery can incur Meta/BSP charges.

Recommended UX:

- Every follow-up task has a `wa_window_status`:
  - `open`: free-form message allowed.
  - `closing_soon`: less than 2 hours left, nudge admin to respond now.
  - `expired`: template required.
  - `unknown`: no reliable last inbound timestamp; block automation and ask admin to verify.
- In `open`, show CTA: `Kirim follow-up sekarang`.
- In `closing_soon`, show CTA: `Kirim sebelum window tutup` plus countdown.
- In `expired`, show CTA: `Pilih template WhatsApp` and explain that free text is blocked by WhatsApp policy.
- Scheduled follow-up must preview whether it will still be inside the 24-hour window at send time.
- If scheduled send time is outside the window, the scheduler must force template selection before saving.

Recommended backend guardrail:

- Store `last_customer_message_at` per conversation/contact/session.
- Before sending any free-form outbound message through official WhatsApp API, check `now - last_customer_message_at <= 24h`.
- If expired, return a typed error such as `WA_TEMPLATE_REQUIRED`.
- For template sends, require `template_name`, `language`, category, components/variables, and approved/synced status.
- Log every scheduled or automated outbound attempt with `send_mode`: `freeform`, `template_utility`, `template_marketing`, `manual_only`, or `blocked`.
- Add `meta_charge_expected` boolean and `meta_charge_reason`.

Recommended billing display:

- Oneflow credit estimate:
  - Manual follow-up send: 0 Oneflow AI credits.
  - AI draft follow-up: minimum 1 Oneflow credit, actual from token cost.
  - AI triage/summary/next-action: minimum 1 Oneflow credit, actual from token cost.
- WhatsApp/Meta cost display:
  - Inside 24h with free-form: `Tidak perlu template. Biaya WhatsApp mengikuti session/service window aktif.`
  - Outside 24h: `Wajib template Meta. Bisa kena biaya WhatsApp/BSP per pesan terkirim sesuai kategori template dan negara nomor tujuan.`
  - Utility template inside active customer-service window: show as likely not charged by Meta under current 2025 pricing updates, but still label as external policy/rate dependent.

Automation rule:

- Default automation should never auto-send outside 24h without explicit template configuration.
- For first MVP, auto-follow-up outside 24h should create a task/reminder and draft template payload, not send automatically.
- Admin can later enable auto-send for selected approved templates after seeing cost warning and opt-out rules.

## Tool Split

### CRM & Follow-up

Purpose: track sales/customer journey.

Keep existing concepts:

- Contacts
- Lifecycle status: lead, prospect, customer, inactive
- Deals/pipelines/stages
- Tasks/reminders
- Notes
- Tags
- Broadcast/message templates
- AI prospect creation from chat

Improve with:

- Unified customer timeline: conversations, notes, tasks, deals, tickets, orders, bookings.
- Custom saved views: by owner, lifecycle, stage, priority, stale follow-up, tag.
- Better lead/deal conversion language for Indonesian business users.
- AI summary and next action suggestion per contact/deal.
- AI stale-follow-up detection.

### Customer Tickets

Purpose: track customer problems, complaints, support requests, unresolved handoffs, and admin work.

This should be a separate installable business tool because support/case workflows are different from sales follow-up.

Core fields:

- `ticket_number`
- `title`
- `description`
- `issue_type`: complaint, product_question, payment, delivery, booking, technical, refund, other
- `status`: new, triage, waiting_customer, waiting_internal, in_progress, resolved, closed, cancelled
- `priority`: low, normal, high, urgent
- `severity`: minor, moderate, major, critical
- `stage_id` or fixed status pipeline
- `contact_id`
- `conversation_id`
- `whatsapp_session_id`
- `ai_agent_id`
- `assigned_to`
- `source`: inbox, whatsapp, playground, dashboard, ai_tool
- `sla_due_at`
- `resolved_at`
- `closed_at`
- `created_by`
- `organization_id`

Supporting tables:

- `ticket_comments`
- `ticket_events`
- `ticket_tasks`
- `ticket_attachments` later if needed
- `ticket_ai_suggestions`
- `ticket_sla_policies` later

Dashboard surfaces:

- Tickets board/list with filters: status, priority, owner, issue type, SLA, source.
- Ticket detail with customer profile, linked conversation, AI summary, timeline, internal notes, next action, and related deal/order/booking.
- Inbox action: "Buat ticket" and "AI buat draft ticket".
- Contacts panel tab: Tickets.
- Analytics: open tickets, overdue SLA, average resolution time, issue types, AI-created vs manual.

## AI Behavior

AI should not silently mutate CRM/ticket data unless the business tool mode allows it.

Modes should follow current business-tool pattern:

- `off`: AI cannot use the tool.
- `read`: AI can read relevant CRM/ticket context.
- `draft`: AI can suggest ticket/deal/task changes for admin approval.
- `action`: AI can create ticket/prospect/task when deterministic validation passes.

AI actions:

- Summarize conversation into issue/problem statement.
- Classify issue type, priority, severity, sentiment.
- Suggest stage/status.
- Suggest owner based on rules/team role.
- Extract missing customer info.
- Draft internal note.
- Draft customer reply.
- Detect duplicate open ticket for same contact/topic.
- Suggest next best action.
- Mark stale ticket/follow-up candidates.

Guardrails:

- AI cannot resolve/close a ticket without admin confirmation.
- AI cannot promise refund, payment, delivery, booking, stock, or policy outcomes.
- AI cannot infer private facts from memory; business facts must come from Knowledge or official tools.
- Every AI mutation must write an audit event and usage log.

## Credit Model

Use the existing backend credit billing path. Do not create a separate credit calculator.

Suggested usage types:

- `crm_ai_summary`
- `crm_ai_next_action`
- `crm_ai_stage_suggestion`
- `ticket_ai_triage`
- `ticket_ai_reply_draft`
- `ticket_ai_duplicate_check`
- `ticket_ai_bulk_triage`

Charging rules:

- Manual CRUD, manual stage move, manual notes, manual ticket creation: 0 credits.
- AI single-action call: billed by actual token cost through current credit formula, minimum 1 credit.
- AI bulk triage: bill per processed item or per batch with clear estimate before run.
- Read-only dashboard filtering/searching: 0 credits.
- If AI action only uses deterministic rules without provider call: 0 credits, but log event as non-billed automation if useful.

UI must show:

- "AI action uses credits" before execution.
- Estimated minimum: "mulai dari 1 credit".
- Actual credits after execution from `usage_metadata.creditsUsed`.
- Remaining credits after action.

## MVP Proposal

Build first as `tickets` business tool, then unify CRM.

MVP scope:

1. Migration for ticket tables.
2. Backend endpoints:
   - `GET /api/tickets`
   - `POST /api/tickets`
   - `GET /api/tickets/{id}`
   - `PATCH /api/tickets/{id}`
   - `POST /api/tickets/{id}/comments`
   - `POST /api/tickets/{id}/events`
   - `POST /api/tickets/ai/triage`
3. Business tool ledger adds `tickets`.
4. Dashboard `TicketsView`:
   - board/list toggle
   - filters
   - ticket detail panel
   - AI triage button
   - credit usage preview/result
5. Inbox integration:
   - create ticket from conversation
   - AI draft ticket from latest conversation context
6. Contact detail integration:
   - open tickets list
7. Billing:
   - log `ticket_ai_triage` through existing credit usage path.
8. Tests:
   - tenant isolation
   - role permissions
   - credit deduction
   - AI action blocked on insufficient credits
   - ticket linked to correct contact/conversation

## Implementation Plan

Phase 0 - tighten current CRM language:

- Rename/position existing `prospects` as `CRM & Follow-up` in Alat Bisnis while preserving module key compatibility.
- Add helper copy explaining: CRM = leads/deals/follow-up; Tickets = customer problems/support cases.
- Add WhatsApp 24-hour window status to Contacts, Inbox, and Follow-up surfaces.

Phase 1 - Follow-up Guardrail:

- Add `last_customer_message_at` exposure to contact/conversation payloads if not already available.
- Add frontend countdown badge for active WhatsApp window.
- Add backend send guard for official WhatsApp free-form follow-up sends.
- Add scheduled follow-up validation:
  - inside 24h: free-form allowed.
  - outside 24h: template required.
- Add explicit warnings for Meta/BSP charge risk.

Phase 2 - Tickets MVP:

- Add `tickets` module to business-tools ledger.
- Add ticket migrations, APIs, and dashboard view.
- Add Inbox -> Create Ticket.
- Add AI ticket triage as a credit-billed action.
- Keep auto-send disabled by default.

Phase 3 - AI Follow-up:

- Add AI draft follow-up message for CRM tasks.
- AI can recommend timing:
  - "send now" if inside 24h.
  - "schedule reminder" if close to expiry.
  - "use approved template" if outside 24h.
- AI can suggest template category and variables, but admin must approve before send.

Phase 4 - Template-aware Scheduler:

- Add scheduled follow-up jobs.
- Free-form scheduled jobs must re-check 24h window at send time.
- If expired at send time, job pauses and asks for template unless an approved template was configured.
- Template auto-send requires explicit opt-in per template/use case.

## Later Phases

Phase 2:

- Unified customer timeline across conversations, notes, tasks, deals, tickets, orders, bookings.
- Saved CRM/ticket views.
- Ticket SLA policies and overdue alerts.
- Duplicate detection.
- AI draft reply inside ticket detail.

Phase 3:

- Workflow rules inspired by SuiteCRM:
  - if issue type = refund and priority high, assign owner
  - if ticket stale for 2 days, notify group
  - if new lead from WhatsApp and qualified, create follow-up task
- Process audit page.
- Custom fields per business tool.

Phase 4:

- Advanced reporting, exports, and automation builder.

## Product Naming Options

- CRM & Follow-up: customer journey, sales, leads, prospects.
- Tickets: problems, complaints, admin cases, support work.
- Customer Ops: combined top-level concept that includes Contacts, CRM, Tickets, Orders, Booking, and Inbox.

Recommended UI naming:

- Keep menu `Follow-up` for existing prospects/deals.
- Add menu `Tickets` when the `tickets` tool is installed.
- In Alat Bisnis, show `CRM & Follow-up` and `Tickets` separately.

## Final Decision

Implement concepts, not code, from SuiteCRM and Frappe CRM.

Best next implementation is a Oneflow-native `tickets` business tool with AI triage and credit-aware billing, while gradually improving the existing `prospects` tool into a cleaner CRM & Follow-up experience.
