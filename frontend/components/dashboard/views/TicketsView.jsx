import { useEffect, useMemo, useState } from "react";
import { EmptyState, Notice } from "../ui";
import { Icons, formatDateTime, formatNumber, truncateText } from "../../../lib/dashboard-core";

const statusOptions = [
  { value: "new", label: "Baru" },
  { value: "triage", label: "Triage" },
  { value: "waiting_customer", label: "Menunggu customer" },
  { value: "waiting_internal", label: "Menunggu internal" },
  { value: "in_progress", label: "Diproses" },
  { value: "resolved", label: "Resolved" },
  { value: "closed", label: "Closed" },
  { value: "cancelled", label: "Batal" },
];

const priorityOptions = [
  { value: "low", label: "Rendah" },
  { value: "normal", label: "Normal" },
  { value: "high", label: "Tinggi" },
  { value: "urgent", label: "Urgent" },
];

const severityOptions = [
  { value: "minor", label: "Minor" },
  { value: "moderate", label: "Moderate" },
  { value: "major", label: "Major" },
  { value: "critical", label: "Critical" },
];

const issueTypeOptions = [
  { value: "complaint", label: "Komplain" },
  { value: "product_question", label: "Pertanyaan produk" },
  { value: "payment", label: "Pembayaran" },
  { value: "delivery", label: "Delivery" },
  { value: "booking", label: "Booking" },
  { value: "technical", label: "Teknis" },
  { value: "refund", label: "Refund" },
  { value: "other", label: "Lainnya" },
];

function emptyTicketDraft() {
  return {
    title: "",
    description: "",
    issueType: "other",
    status: "new",
    stageId: "",
    priority: "normal",
    severity: "minor",
    contactId: "",
    conversationId: "",
    assignedToId: "",
    slaDueAt: "",
    source: "dashboard",
  };
}

function safeArray(value) {
  return Array.isArray(value) ? value : [];
}

function optionLabel(options, value) {
  return options.find((item) => item.value === value)?.label || value || "-";
}

function stageForStatus(stages, status) {
  return safeArray(stages).find((stage) => stage.status === status) || safeArray(stages)[0] || null;
}

function stageName(ticket) {
  return ticket?.stageName || optionLabel(statusOptions, ticket?.status);
}

function priorityTone(value) {
  if (value === "urgent") return "red";
  if (value === "high") return "orange";
  if (value === "low") return "gray";
  return "blue";
}

function severityTone(value) {
  if (value === "critical") return "red";
  if (value === "major") return "orange";
  if (value === "moderate") return "blue";
  return "gray";
}

function statusTone(value) {
  if (value === "resolved" || value === "closed") return "green";
  if (value === "cancelled") return "gray";
  if (value === "waiting_customer" || value === "waiting_internal") return "orange";
  return "blue";
}

function dateTimeLocalToISO(value) {
  if (!value) return null;
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return null;
  return date.toISOString();
}

function dateTimeLocalValue(value) {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  const shifted = new Date(date.getTime() - date.getTimezoneOffset() * 60000);
  return shifted.toISOString().slice(0, 16);
}

function aiModeLabel(mode) {
  if (mode === "draft") return "AI triage aktif";
  if (mode === "action") return "AI action aktif";
  if (mode === "read") return "AI baca saja";
  return "AI mati";
}

function aiModeAllowsTriage(mode) {
  return mode === "draft" || mode === "action";
}

function contactName(contact) {
  return contact?.name || contact?.phone || contact?.email || "";
}

function conversationLabel(item) {
  if (!item) return "";
  return `${item.contactName || item.phone || "Customer"} - ${truncateText(item.lastMessageText || "Belum ada pesan", 42)}`;
}

function windowCopy(windowInfo, provider = "") {
  const status = windowInfo?.status || "unknown";
  const prefix = provider === "meta_cloud" ? "WhatsApp official" : "WhatsApp";
  if (status === "open") {
    return `${prefix}: masih di bawah 24 jam, free-form follow-up boleh dikirim. Sisa sekitar ${windowInfo.hoursRemaining || 0} jam.`;
  }
  if (status === "expired") {
    return `${prefix}: sudah di atas 24 jam. Pakai approved template; biaya kategori Meta/BSP terpisah dari credit AI Oneflow.`;
  }
  return `${prefix}: belum ada inbound customer tercatat. Untuk official API, mulai dengan approved template.`;
}

function ticketPayloadFromDraft(draft) {
  return {
    title: draft.title,
    description: draft.description,
    issueType: draft.issueType,
    status: draft.status,
    stageId: draft.stageId || undefined,
    priority: draft.priority,
    severity: draft.severity,
    contactId: draft.contactId || undefined,
    conversationId: draft.conversationId || undefined,
    assignedToId: draft.assignedToId || undefined,
    source: draft.source || "dashboard",
    slaDueAt: dateTimeLocalToISO(draft.slaDueAt),
    metadata: {},
  };
}

export function TicketsView({
  contacts = [],
  inbox = [],
  teamMembers = [],
  apiBase = "",
  requestJSON,
  toolState,
  onRefreshDashboard,
}) {
  const installed = Boolean(toolState?.installed ?? toolState?.enabled);
  const aiMode = toolState?.aiMode || "off";
  const canRunTriage = installed && aiModeAllowsTriage(aiMode);
  const [tickets, setTickets] = useState([]);
  const [stages, setStages] = useState([]);
  const [stageDrafts, setStageDrafts] = useState({});
  const [newStageName, setNewStageName] = useState("");
  const [newStageStatus, setNewStageStatus] = useState("new");
  const [newStageColor, setNewStageColor] = useState("#2196F3");
  const [summary, setSummary] = useState({});
  const [filterStatus, setFilterStatus] = useState("all");
  const [query, setQuery] = useState("");
  const [selectedTicketId, setSelectedTicketId] = useState("");
  const [detail, setDetail] = useState(null);
  const [draft, setDraft] = useState(emptyTicketDraft());
  const [patchDraft, setPatchDraft] = useState(emptyTicketDraft());
  const [commentDraft, setCommentDraft] = useState("");
  const [busyKey, setBusyKey] = useState("");
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [aiResult, setAiResult] = useState(null);

  const selectedTicket = detail?.item || tickets.find((item) => item.id === selectedTicketId) || null;
  const selectedConversation = useMemo(
    () => inbox.find((item) => item.id === draft.conversationId) || null,
    [draft.conversationId, inbox]
  );
  const selectedTicketConversation = useMemo(
    () => inbox.find((item) => item.id === selectedTicket?.conversationId) || null,
    [inbox, selectedTicket?.conversationId]
  );

  useEffect(() => {
    if (!installed) return;
    loadTickets();
  }, [installed, filterStatus]);

  useEffect(() => {
    if (!tickets.length || selectedTicketId) return;
    setSelectedTicketId(tickets[0].id);
  }, [tickets, selectedTicketId]);

  useEffect(() => {
    if (!selectedTicketId || !installed) {
      setDetail(null);
      return;
    }
    loadTicketDetail(selectedTicketId);
  }, [selectedTicketId, installed]);

  useEffect(() => {
    if (!selectedTicket) return;
    setPatchDraft({
      title: selectedTicket.title || "",
      description: selectedTicket.description || "",
      issueType: selectedTicket.issueType || "other",
      status: selectedTicket.status || "new",
      priority: selectedTicket.priority || "normal",
      severity: selectedTicket.severity || "minor",
      stageId: selectedTicket.stageId || stageForStatus(stages, selectedTicket.status)?.id || "",
      contactId: selectedTicket.contactId || "",
      conversationId: selectedTicket.conversationId || "",
      assignedToId: selectedTicket.assignedToId || "",
      slaDueAt: dateTimeLocalValue(selectedTicket.slaDueAt),
      source: selectedTicket.source || "dashboard",
    });
  }, [selectedTicket?.id, stages]);

  useEffect(() => {
    setStageDrafts(Object.fromEntries(stages.map((stage) => [stage.id, {
      name: stage.name || "",
      status: stage.status || "new",
      position: String(stage.position || 0),
      color: stage.color || "#2196F3",
    }])));
  }, [stages]);

  useEffect(() => {
    if (!selectedConversation) return;
    setDraft((current) => ({
      ...current,
      contactId: current.contactId || selectedConversation.contactId || "",
      title: current.title || `Problem ${selectedConversation.contactName || selectedConversation.phone || "customer"}`,
    }));
  }, [selectedConversation?.id]);

  async function loadTickets(nextQuery = query) {
    if (!requestJSON) return;
    setBusyKey("tickets-load");
    setError("");
    try {
      const params = new URLSearchParams();
      if (filterStatus !== "all") params.set("status", filterStatus);
      if (nextQuery.trim()) params.set("query", nextQuery.trim());
      const payload = await requestJSON(`${apiBase}/api/tickets${params.toString() ? `?${params}` : ""}`);
      setTickets(safeArray(payload.items));
      setStages(safeArray(payload.stages));
      setSummary(payload.summary || {});
    } catch (err) {
      setError(err.message || "Gagal memuat tiket.");
    } finally {
      setBusyKey("");
    }
  }

  async function loadTicketDetail(ticketId) {
    setBusyKey("ticket-detail");
    setError("");
    try {
      const payload = await requestJSON(`${apiBase}/api/tickets/${encodeURIComponent(ticketId)}`);
      setDetail(payload);
      if (payload.stages) setStages(safeArray(payload.stages));
    } catch (err) {
      setError(err.message || "Gagal memuat detail tiket.");
    } finally {
      setBusyKey("");
    }
  }

  async function createTicket() {
    setBusyKey("ticket-create");
    setError("");
    setNotice("");
    try {
      const payload = await requestJSON(`${apiBase}/api/tickets`, {
        method: "POST",
        body: JSON.stringify(ticketPayloadFromDraft(draft)),
      });
      const item = payload.item;
      setDraft(emptyTicketDraft());
      setAiResult(null);
      setNotice(`Tiket ${item?.ticketNumber || ""} dibuat.`);
      await loadTickets("");
      if (item?.id) setSelectedTicketId(item.id);
      onRefreshDashboard?.();
    } catch (err) {
      setError(err.message || "Gagal membuat tiket.");
    } finally {
      setBusyKey("");
    }
  }

  async function saveSelectedTicket() {
    if (!selectedTicket?.id) return;
    setBusyKey("ticket-save");
    setError("");
    try {
      const payload = await requestJSON(`${apiBase}/api/tickets/${encodeURIComponent(selectedTicket.id)}`, {
        method: "PATCH",
        body: JSON.stringify(ticketPayloadFromDraft(patchDraft)),
      });
      setNotice(`Tiket ${payload.item?.ticketNumber || ""} diperbarui.`);
      await loadTickets();
      await loadTicketDetail(selectedTicket.id);
      onRefreshDashboard?.();
    } catch (err) {
      setError(err.message || "Gagal menyimpan tiket.");
    } finally {
      setBusyKey("");
    }
  }

  async function addComment() {
    if (!selectedTicket?.id || !commentDraft.trim()) return;
    setBusyKey("ticket-comment");
    setError("");
    try {
      const payload = await requestJSON(`${apiBase}/api/tickets/${encodeURIComponent(selectedTicket.id)}/comments`, {
        method: "POST",
        body: JSON.stringify({ body: commentDraft.trim(), isInternal: true }),
      });
      setCommentDraft("");
      setDetail((current) => ({ ...(current || {}), item: payload.item, comments: payload.comments || [] }));
      setNotice("Komentar internal ditambahkan.");
      await loadTickets();
    } catch (err) {
      setError(err.message || "Gagal menambah komentar.");
    } finally {
      setBusyKey("");
    }
  }

  async function runAITriage(mode = "draft") {
    const conversationId = mode === "selected" ? selectedTicket?.conversationId : draft.conversationId;
    if (!conversationId || !canRunTriage) return;
    setBusyKey(`ticket-ai-${mode}`);
    setError("");
    setNotice("");
    try {
      const payload = await requestJSON(`${apiBase}/api/tickets/ai/triage`, {
        method: "POST",
        body: JSON.stringify({
          conversationId,
          ticketId: mode === "selected" ? selectedTicket?.id : "",
        }),
      });
      const suggestion = payload.suggestion || {};
      const patch = {
        title: suggestion.title || "",
        description: suggestion.description || suggestion.summary || "",
        issueType: suggestion.issueType || "other",
        status: suggestion.status || "triage",
        stageId: stageForStatus(stages, suggestion.status || "triage")?.id || "",
        priority: suggestion.priority || "normal",
        severity: suggestion.severity || "minor",
      };
      if (mode === "selected") {
        setPatchDraft((current) => ({ ...current, ...patch }));
      } else {
        setDraft((current) => ({ ...current, ...patch }));
      }
      setAiResult(payload);
      setNotice(`AI triage siap. Credit terpakai: ${payload.usage?.creditsUsed ?? payload.usage?.credits_used ?? 1}.`);
    } catch (err) {
      setError(err.message || "AI triage gagal.");
    } finally {
      setBusyKey("");
    }
  }

  async function addStage(event) {
    event.preventDefault();
    if (!newStageName.trim()) return;
    setBusyKey("ticket-stage-new");
    setError("");
    try {
      await requestJSON(`${apiBase}/api/tickets/stages`, {
        method: "POST",
        body: JSON.stringify({
          name: newStageName.trim(),
          status: newStageStatus,
          color: newStageColor,
        }),
      });
      setNewStageName("");
      await loadTickets();
      setNotice("Stage ticket ditambahkan.");
    } catch (err) {
      setError(err.message || "Gagal menambah stage ticket.");
    } finally {
      setBusyKey("");
    }
  }

  async function saveStage(stage) {
    const draftStage = stageDrafts[stage.id];
    if (!stage?.id || !draftStage?.name?.trim()) return;
    setBusyKey(`ticket-stage-${stage.id}`);
    setError("");
    try {
      await requestJSON(`${apiBase}/api/tickets/stages/${encodeURIComponent(stage.id)}`, {
        method: "PATCH",
        body: JSON.stringify({
          name: draftStage.name.trim(),
          status: draftStage.status || stage.status,
          position: Number(draftStage.position) || stage.position,
          color: draftStage.color || stage.color,
        }),
      });
      await loadTickets();
      setNotice("Stage ticket diperbarui.");
    } catch (err) {
      setError(err.message || "Gagal menyimpan stage ticket.");
    } finally {
      setBusyKey("");
    }
  }

  async function deleteStage(stage) {
    if (!stage?.id) return;
    if (!window.confirm(`Hapus stage "${stage.name}"? Tiket di stage ini akan dipindah ke stage default.`)) return;
    setBusyKey(`ticket-stage-delete-${stage.id}`);
    setError("");
    try {
      await requestJSON(`${apiBase}/api/tickets/stages/${encodeURIComponent(stage.id)}`, { method: "DELETE" });
      await loadTickets();
      setNotice("Stage ticket dihapus.");
    } catch (err) {
      setError(err.message || "Gagal menghapus stage ticket.");
    } finally {
      setBusyKey("");
    }
  }

  const visibleTickets = tickets;
  const openCount = Number(summary.open || 0);
  const overdueCount = Number(summary.overdue || 0);
  const urgentCount = Number(summary.urgent || 0);
  const createDisabled = !draft.title.trim() || busyKey === "ticket-create";

  return (
    <div className="tickets-page">
      <div className="page-title-row">
        <div>
          <div className="page-title">Tiket</div>
          <div className="page-desc">Keluhan pelanggan dan penanganannya dalam satu daftar.</div>
        </div>
        <div className="page-actions">
          <button className="btn btn-secondary" type="button" onClick={() => loadTickets()} disabled={!installed || busyKey === "tickets-load"}>
            {Icons.refresh} Muat ulang
          </button>
        </div>
      </div>

      {!installed ? (
        <Notice tone="warn" title="Tool Ticketing belum dipasang" text="Pasang Ticketing dari Alat Bisnis dulu. Fitur utama CRM, onboarding, dan inbox tetap tidak berubah." />
      ) : (
        <Notice
          tone={canRunTriage ? "success" : "warn"}
          title={`Mode AI: ${aiModeLabel(aiMode)}`}
          text="Manual create/update/comment tidak memakai credit AI. Tombol AI triage memakai billing backend dan minimal 1 credit per action sesuai token aktual/estimasi."
        />
      )}

      {(error || notice) ? (
        <div className={`crm-alert ${error ? "error" : "success"}`}>
          {error || notice}
          <button type="button" onClick={() => { setError(""); setNotice(""); }}>Tutup</button>
        </div>
      ) : null}

      <div className="tickets-kpi-grid">
        <div className="kpi-card"><div className="kpi-icon blue">{Icons.ticket}</div><div><div className="kpi-value">{formatNumber(summary.total || 0)}</div><div className="kpi-label">Total tiket</div></div></div>
        <div className="kpi-card"><div className="kpi-icon orange" /><div><div className="kpi-value">{formatNumber(openCount)}</div><div className="kpi-label">Masih aktif</div></div></div>
        <div className="kpi-card"><div className="kpi-icon red" /><div><div className="kpi-value">{formatNumber(overdueCount)}</div><div className="kpi-label">SLA lewat</div></div></div>
        <div className="kpi-card"><div className="kpi-icon purple" /><div><div className="kpi-value">{formatNumber(urgentCount)}</div><div className="kpi-label">Urgent</div></div></div>
      </div>

      {installed ? (
        <details className="business-pipeline-settings tickets-stage-settings">
          <summary>Customize stage ticket</summary>
          <div className="business-stage-editor-list">
            {stages.map((stage) => (
              <div className="business-stage-editor-row" key={stage.id}>
                <div className="business-stage-name-field">
                  <input className="form-input" value={stageDrafts[stage.id]?.name || ""} onChange={(event) => setStageDrafts((current) => ({ ...current, [stage.id]: { ...(current[stage.id] || {}), name: event.target.value } }))} />
                </div>
                <select className="form-select" value={stageDrafts[stage.id]?.status || "new"} onChange={(event) => setStageDrafts((current) => ({ ...current, [stage.id]: { ...(current[stage.id] || {}), status: event.target.value } }))}>
                  {statusOptions.map((item) => <option key={item.value} value={item.value}>{item.label}</option>)}
                </select>
                <input className="form-input" type="number" min="1" value={stageDrafts[stage.id]?.position || "0"} onChange={(event) => setStageDrafts((current) => ({ ...current, [stage.id]: { ...(current[stage.id] || {}), position: event.target.value } }))} aria-label={`Urutan ${stage.name}`} />
                <input className="form-input business-color-input" type="color" value={stageDrafts[stage.id]?.color || "#2196F3"} onChange={(event) => setStageDrafts((current) => ({ ...current, [stage.id]: { ...(current[stage.id] || {}), color: event.target.value } }))} aria-label={`Warna ${stage.name}`} />
                <button className="btn btn-secondary btn-sm" type="button" onClick={() => saveStage(stage)} disabled={busyKey === `ticket-stage-${stage.id}`}>Simpan</button>
                <button className="btn btn-danger btn-sm" type="button" onClick={() => deleteStage(stage)} disabled={stages.length <= 1 || busyKey === `ticket-stage-delete-${stage.id}`}>Hapus</button>
              </div>
            ))}
          </div>
          <form className="business-stage-add-row" onSubmit={addStage}>
            <input className="form-input" value={newStageName} onChange={(event) => setNewStageName(event.target.value)} placeholder="Nama stage baru" />
            <select className="form-select" value={newStageStatus} onChange={(event) => setNewStageStatus(event.target.value)}>
              {statusOptions.map((item) => <option key={item.value} value={item.value}>{item.label}</option>)}
            </select>
            <input className="form-input business-color-input" type="color" value={newStageColor} onChange={(event) => setNewStageColor(event.target.value)} aria-label="Warna stage baru" />
            <button className="btn btn-primary btn-sm" type="submit" disabled={!newStageName.trim() || busyKey === "ticket-stage-new"}>Tambah stage</button>
          </form>
        </details>
      ) : null}

      <div className="tickets-workspace">
        <section className="card tickets-create-card">
          <div className="card-header">
            <div>
              <div className="card-title">Buat Tiket</div>
              <div className="card-subtitle">Ambil dari chat WhatsApp atau input manual.</div>
            </div>
          </div>
          <div className="tickets-form-grid">
            <label>Conversation
              <select className="form-select" value={draft.conversationId} onChange={(event) => setDraft((current) => ({ ...current, conversationId: event.target.value, contactId: inbox.find((item) => item.id === event.target.value)?.contactId || current.contactId }))} disabled={!installed}>
                <option value="">Manual / tanpa chat</option>
                {inbox.map((item) => <option key={item.id} value={item.id}>{conversationLabel(item)}</option>)}
              </select>
            </label>
            <label>Kontak
              <select className="form-select" value={draft.contactId} onChange={(event) => setDraft((current) => ({ ...current, contactId: event.target.value }))} disabled={!installed}>
                <option value="">Pilih kontak</option>
                {contacts.map((contact) => <option key={contact.id} value={contact.id}>{contactName(contact)}</option>)}
              </select>
            </label>
            <label>Judul
              <input className="form-input" value={draft.title} onChange={(event) => setDraft((current) => ({ ...current, title: event.target.value }))} placeholder="Contoh: Customer belum menerima order" disabled={!installed} />
            </label>
            <label>PIC
              <select className="form-select" value={draft.assignedToId} onChange={(event) => setDraft((current) => ({ ...current, assignedToId: event.target.value }))} disabled={!installed}>
                <option value="">Belum assign</option>
                {teamMembers.map((member) => <option key={member.id} value={member.id}>{member.name || member.username}</option>)}
              </select>
            </label>
            <label>Tipe issue
              <select className="form-select" value={draft.issueType} onChange={(event) => setDraft((current) => ({ ...current, issueType: event.target.value }))} disabled={!installed}>{issueTypeOptions.map((item) => <option key={item.value} value={item.value}>{item.label}</option>)}</select>
            </label>
            <label>Stage
              <select className="form-select" value={draft.stageId || stageForStatus(stages, draft.status)?.id || ""} onChange={(event) => {
                const stage = stages.find((item) => item.id === event.target.value);
                setDraft((current) => ({ ...current, stageId: event.target.value, status: stage?.status || current.status }));
              }} disabled={!installed || !stages.length}>
                {stages.map((stage) => <option key={stage.id} value={stage.id}>{stage.name}</option>)}
              </select>
            </label>
            <label>Priority
              <select className="form-select" value={draft.priority} onChange={(event) => setDraft((current) => ({ ...current, priority: event.target.value }))} disabled={!installed}>{priorityOptions.map((item) => <option key={item.value} value={item.value}>{item.label}</option>)}</select>
            </label>
            <label>Severity
              <select className="form-select" value={draft.severity} onChange={(event) => setDraft((current) => ({ ...current, severity: event.target.value }))} disabled={!installed}>{severityOptions.map((item) => <option key={item.value} value={item.value}>{item.label}</option>)}</select>
            </label>
            <label>SLA due
              <input className="form-input" type="datetime-local" value={draft.slaDueAt} onChange={(event) => setDraft((current) => ({ ...current, slaDueAt: event.target.value }))} disabled={!installed} />
            </label>
          </div>
          <label className="tickets-field-block">Deskripsi internal
            <textarea className="form-textarea" value={draft.description} onChange={(event) => setDraft((current) => ({ ...current, description: event.target.value }))} placeholder="Ringkasan problem, bukti, dan konteks operasional." disabled={!installed} />
          </label>

          {selectedConversation ? (
            <div className={`tickets-window-panel ${selectedConversation.whatsappWindow?.requiresTemplate ? "warn" : "ok"}`}>
              <strong>Aturan follow-up WhatsApp</strong>
              <span>{windowCopy(selectedConversation.whatsappWindow, selectedConversation.whatsappProvider)}</span>
            </div>
          ) : (
            <div className="tickets-window-panel">
              <strong>Aturan follow-up WhatsApp</strong>
              <span>Di bawah 24 jam sejak inbound customer: free-form. Di atas 24 jam: gunakan approved template; biaya Meta/BSP terpisah dari credit AI.</span>
            </div>
          )}

          {aiResult?.suggestion ? (
            <div className="tickets-ai-result">
              <strong>AI summary</strong>
              <p>{aiResult.suggestion.summary || aiResult.suggestion.description}</p>
              <span>{aiResult.suggestion.nextAction}</span>
            </div>
          ) : null}

          <div className="tickets-actions-row">
            <button className="btn btn-secondary" type="button" onClick={() => runAITriage("draft")} disabled={!draft.conversationId || !canRunTriage || busyKey === "ticket-ai-draft"}>
              {Icons.spark} AI triage
            </button>
            <button className="btn btn-primary" type="button" onClick={createTicket} disabled={!installed || createDisabled}>Buat tiket</button>
          </div>
        </section>

        <section className="card tickets-list-card">
          <div className="card-header tickets-list-header">
            <div>
              <div className="card-title">Queue Problem</div>
              <div className="card-subtitle">Urut berdasarkan priority dan SLA.</div>
            </div>
            <div className="tickets-filter-row">
              <input className="form-input" value={query} onChange={(event) => setQuery(event.target.value)} onKeyDown={(event) => { if (event.key === "Enter") loadTickets(event.currentTarget.value); }} placeholder="Cari tiket/customer..." disabled={!installed} />
              <select className="form-select" value={filterStatus} onChange={(event) => setFilterStatus(event.target.value)} disabled={!installed}>
                <option value="all">Semua status</option>
                {statusOptions.map((item) => <option key={item.value} value={item.value}>{item.label}</option>)}
              </select>
            </div>
          </div>
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Tiket</th>
                  <th>Status</th>
                  <th>Customer</th>
                  <th>Priority</th>
                  <th>SLA</th>
                  <th>PIC</th>
                </tr>
              </thead>
              <tbody>
                {visibleTickets.map((ticket) => (
                  <tr key={ticket.id} className={ticket.id === selectedTicketId ? "crm-selected-row" : ""} onClick={() => setSelectedTicketId(ticket.id)}>
                    <td><strong>{ticket.ticketNumber}</strong><div className="text-xs text-muted">{truncateText(ticket.title, 54)}</div></td>
                    <td><span className={`badge ${statusTone(ticket.status)}`} style={ticket.stageColor ? { borderColor: ticket.stageColor } : undefined}>{stageName(ticket)}</span></td>
                    <td><div className="text-sm">{ticket.contactName || "-"}</div><div className="text-xs text-muted">{ticket.contactPhone || "-"}</div></td>
                    <td><span className={`badge ${priorityTone(ticket.priority)}`}>{optionLabel(priorityOptions, ticket.priority)}</span></td>
                    <td><div className="text-xs text-muted">{ticket.slaDueAt ? formatDateTime(ticket.slaDueAt) : "-"}</div></td>
                    <td><div className="text-sm">{ticket.assignedToName || "Unassigned"}</div></td>
                  </tr>
                ))}
                {!visibleTickets.length ? (
                  <tr><td colSpan={6} className="contacts-empty-cell">Belum ada tiket.</td></tr>
                ) : null}
              </tbody>
            </table>
          </div>
        </section>

        <aside className="card tickets-detail-card">
          {!selectedTicket ? (
            <EmptyState title={busyKey === "ticket-detail" ? "Memuat detail" : "Pilih tiket"} copy="Detail problem dan komentar internal muncul di sini." compact />
          ) : (
            <>
              <div className="tickets-detail-head">
                <div>
                  <div className="card-title">{selectedTicket.ticketNumber}</div>
                  <div className="card-subtitle">{selectedTicket.title}</div>
                </div>
                <span className={`badge ${severityTone(selectedTicket.severity)}`}>{optionLabel(severityOptions, selectedTicket.severity)}</span>
              </div>

              {selectedTicketConversation || selectedTicket.whatsappWindow ? (
                <div className={`tickets-window-panel ${selectedTicket.whatsappWindow?.requiresTemplate ? "warn" : "ok"}`}>
                  <strong>Follow-up window</strong>
                  <span>{windowCopy(selectedTicket.whatsappWindow || selectedTicketConversation?.whatsappWindow, selectedTicket.whatsappProvider || selectedTicketConversation?.whatsappProvider)}</span>
                </div>
              ) : null}

              <div className="tickets-form-grid compact">
                <label>Stage<select className="form-select" value={patchDraft.stageId || stageForStatus(stages, patchDraft.status)?.id || ""} onChange={(event) => {
                  const stage = stages.find((item) => item.id === event.target.value);
                  setPatchDraft((current) => ({ ...current, stageId: event.target.value, status: stage?.status || current.status }));
                }}>{stages.map((stage) => <option key={stage.id} value={stage.id}>{stage.name}</option>)}</select></label>
                <label>Priority<select className="form-select" value={patchDraft.priority} onChange={(event) => setPatchDraft((current) => ({ ...current, priority: event.target.value }))}>{priorityOptions.map((item) => <option key={item.value} value={item.value}>{item.label}</option>)}</select></label>
                <label>Severity<select className="form-select" value={patchDraft.severity} onChange={(event) => setPatchDraft((current) => ({ ...current, severity: event.target.value }))}>{severityOptions.map((item) => <option key={item.value} value={item.value}>{item.label}</option>)}</select></label>
                <label>PIC<select className="form-select" value={patchDraft.assignedToId} onChange={(event) => setPatchDraft((current) => ({ ...current, assignedToId: event.target.value }))}><option value="">Unassigned</option>{teamMembers.map((member) => <option key={member.id} value={member.id}>{member.name || member.username}</option>)}</select></label>
              </div>
              <label className="tickets-field-block">Judul<input className="form-input" value={patchDraft.title} onChange={(event) => setPatchDraft((current) => ({ ...current, title: event.target.value }))} /></label>
              <label className="tickets-field-block">Deskripsi<textarea className="form-textarea" value={patchDraft.description} onChange={(event) => setPatchDraft((current) => ({ ...current, description: event.target.value }))} /></label>
              <div className="tickets-actions-row">
                <button className="btn btn-secondary btn-sm" type="button" onClick={() => runAITriage("selected")} disabled={!selectedTicket.conversationId || !canRunTriage || busyKey === "ticket-ai-selected"}>{Icons.spark} AI triage ulang</button>
                <button className="btn btn-primary btn-sm" type="button" onClick={saveSelectedTicket} disabled={busyKey === "ticket-save"}>Simpan</button>
              </div>

              <div className="tickets-meta-list">
                <div><span>Issue</span><strong>{optionLabel(issueTypeOptions, selectedTicket.issueType)}</strong></div>
                <div><span>Stage</span><strong>{stageName(selectedTicket)}</strong></div>
                <div><span>Customer</span><strong>{selectedTicket.contactName || "-"}</strong></div>
                <div><span>WhatsApp</span><strong>{selectedTicket.whatsappSession || "-"}</strong></div>
                <div><span>Update</span><strong>{formatDateTime(selectedTicket.updatedAt)}</strong></div>
              </div>

              <div className="tickets-comments">
                <div className="card-title">Komentar Internal</div>
                <textarea className="form-textarea" value={commentDraft} onChange={(event) => setCommentDraft(event.target.value)} placeholder="Catatan untuk tim..." />
                <button className="btn btn-secondary btn-sm" type="button" onClick={addComment} disabled={!commentDraft.trim() || busyKey === "ticket-comment"}>Tambah komentar</button>
                <div className="crm-list-stack">
                  {safeArray(detail?.comments).map((comment) => (
                    <div className="crm-note-item" key={comment.id}>
                      <p>{comment.body}</p>
                      <span>{comment.authorName || "Tim"} - {formatDateTime(comment.createdAt)}</span>
                    </div>
                  ))}
                  {!safeArray(detail?.comments).length ? <EmptyState title="Belum ada komentar" copy="Catatan internal akan tersimpan di tiket ini." compact /> : null}
                </div>
              </div>
            </>
          )}
        </aside>
      </div>
    </div>
  );
}
