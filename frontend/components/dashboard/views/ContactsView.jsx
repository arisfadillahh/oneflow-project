import { useEffect, useMemo, useState } from "react";
import { formatDateTime, formatNumber, toLabel, truncateText } from "../../../lib/dashboard-core";

const avatarColors = ["blue", "green", "orange", "purple", "red"];
const lifecycleOptions = [
  { value: "lead", label: "Lead" },
  { value: "prospect", label: "Prospect" },
  { value: "customer", label: "Customer" },
  { value: "inactive", label: "Nonaktif" },
];
const priorityOptions = [
  { value: "low", label: "Low" },
  { value: "normal", label: "Normal" },
  { value: "high", label: "High" },
  { value: "urgent", label: "Urgent" },
];

function contactInitial(contact) {
  const source = String(contact?.name || contact?.phone || "?").trim();
  return source ? source[0].toUpperCase() : "?";
}

function contactStatus(contact) {
  if (contact.status === "resolved") return "resolved";
  if (contact.status === "pending_human" || contact.mode === "human" || contact.assignedToName) return "human";
  if (contact.conversationId || contact.mode === "ai") return "ai";
  return "new";
}

function statusLabel(status) {
  if (status === "ai") return "AI aktif";
  if (status === "human") return "Ditangani admin";
  if (status === "resolved") return "Selesai";
  return "Baru";
}

function statusBadge(status) {
  if (status === "ai") return "green";
  if (status === "human") return "blue";
  if (status === "resolved") return "gray";
  return "orange";
}

function priorityBadge(priority) {
  if (priority === "urgent") return "red";
  if (priority === "high") return "orange";
  if (priority === "low") return "gray";
  return "blue";
}

function lifecycleBadge(lifecycle) {
  if (lifecycle === "customer") return "green";
  if (lifecycle === "prospect") return "blue";
  if (lifecycle === "inactive") return "gray";
  return "orange";
}

function wasEscalated(contact) {
  return Boolean(contact.escalationReason || contact.assignedToName || contact.status === "pending_human" || contact.mode === "human");
}

function positionValue(contact) {
  return contact.position || contact.positionTitle || contact.appliedPosition || contact.jobTitle || "";
}

function safeArray(value) {
  return Array.isArray(value) ? value : [];
}

function toDateTimeLocal(value) {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  const shifted = new Date(date.getTime() - date.getTimezoneOffset() * 60000);
  return shifted.toISOString().slice(0, 16);
}

function dateTimeLocalToISO(value) {
  if (!value) return null;
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return null;
  return date.toISOString();
}

function parseImportRows(rawText) {
  const lines = rawText.split(/\r?\n/).map((line) => line.trim()).filter(Boolean);
  if (!lines.length) return [];
  const delimiter = lines[0].includes("\t") ? "\t" : lines[0].includes(";") ? ";" : ",";
  const first = lines[0].split(delimiter).map((item) => item.trim().toLowerCase());
  const hasHeader = first.some((item) => ["phone", "nomor", "name", "nama", "email", "notes", "catatan", "tags"].includes(item));
  const headers = hasHeader ? first : ["phone", "name", "email", "notes", "tags"];
  const body = hasHeader ? lines.slice(1) : lines;
  return body.map((line) => {
    const cells = line.split(delimiter).map((item) => item.trim());
    const row = {};
    headers.forEach((header, index) => {
      row[header] = cells[index] || "";
    });
    return {
      phone: row.phone || row.nomor || row.no || "",
      name: row.name || row.nama || "",
      email: row.email || "",
      notes: row.notes || row.catatan || "",
      tags: String(row.tags || row.tag || "").split("|").map((tag) => tag.trim()).filter(Boolean),
    };
  }).filter((item) => item.phone);
}

function ContactKpi({ icon, tone, value, label, meta, trend = "neutral" }) {
  return (
    <div className="kpi-card">
      <div className={`kpi-icon ${tone}`}>{icon}</div>
      <div className="kpi-value">{value}</div>
      <div className="kpi-label">{label}</div>
      <div className={`kpi-trend ${trend}`}>{meta}</div>
    </div>
  );
}

function MiniEmpty({ title, copy }) {
  return (
    <div className="crm-mini-empty">
      <strong>{title}</strong>
      <span>{copy}</span>
    </div>
  );
}

export function ContactsView({ contacts = [], teamMembers = [], apiBase = "", requestJSON, focusContactId = "", onFocusContactHandled, onRefresh, canUseProspects = true }) {
  const [query, setQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  const [lifecycleFilter, setLifecycleFilter] = useState("all");
  const [selectedContactId, setSelectedContactId] = useState("");
  const [detail, setDetail] = useState(null);
  const [contactTags, setContactTags] = useState([]);
  const [tasks, setTasks] = useState([]);
  const [templates, setTemplates] = useState([]);
  const [broadcasts, setBroadcasts] = useState([]);
  const [dealStages, setDealStages] = useState([]);
  const [selectedTagIds, setSelectedTagIds] = useState([]);
  const [profileDraft, setProfileDraft] = useState({ name: "", email: "", notes: "", lifecycleStatus: "lead", ownerAgentId: "", customFieldsText: "{}", lastSummary: "", customerMemoryEnabled: true });
  const [newTag, setNewTag] = useState({ name: "", color: "#2563eb" });
  const [noteDraft, setNoteDraft] = useState("");
  const [memoryDraft, setMemoryDraft] = useState({ memoryType: "admin_note", value: "" });
  const [taskDraft, setTaskDraft] = useState({ title: "", notes: "", dueAt: "", priority: "normal", assignedToId: "" });
  const [importText, setImportText] = useState("");
  const [templateDraft, setTemplateDraft] = useState({ name: "", category: "follow_up", body: "", status: "draft" });
  const [broadcastDraft, setBroadcastDraft] = useState({ title: "", templateId: "", body: "", audience: "selected", tagId: "", dealStageId: "", noFollowUpDays: "7", lifecycleStatus: "prospect" });
  const [busyKey, setBusyKey] = useState("");
  const [crmError, setCrmError] = useState("");
  const [crmNotice, setCrmNotice] = useState("");

  const enrichedContacts = useMemo(() => contacts.map((contact, index) => ({
    ...contact,
    statusGroup: contactStatus(contact),
    escalated: wasEscalated(contact),
    positionName: positionValue(contact),
    avatarColor: avatarColors[index % avatarColors.length],
    lifecycleStatus: contact.lifecycleStatus || "lead",
    tags: safeArray(contact.tags),
  })), [contacts]);

  const filteredContacts = useMemo(() => {
    const needle = query.trim().toLowerCase();
    return enrichedContacts.filter((contact) => {
      if (statusFilter !== "all" && contact.statusGroup !== statusFilter) return false;
      if (lifecycleFilter !== "all" && contact.lifecycleStatus !== lifecycleFilter) return false;
      if (!needle) return true;
      return [
        contact.name,
        contact.phone,
        contact.email,
        contact.notes,
        contact.positionName,
        contact.lastMessageText,
        contact.ownerName,
        ...safeArray(contact.tags).map((tag) => tag.name),
      ].join(" ").toLowerCase().includes(needle);
    });
  }, [enrichedContacts, lifecycleFilter, query, statusFilter]);

  const selectedContact = useMemo(
    () => enrichedContacts.find((contact) => contact.id === selectedContactId) || enrichedContacts[0] || null,
    [enrichedContacts, selectedContactId],
  );

  const aiCount = enrichedContacts.filter((contact) => contact.statusGroup === "ai").length;
  const escalatedCount = enrichedContacts.filter((contact) => contact.escalated).length;
  const openTaskCount = enrichedContacts.reduce((sum, contact) => sum + Number(contact.openTasks || 0), 0);

  useEffect(() => {
    if (!selectedContactId && filteredContacts[0]?.id) {
      setSelectedContactId(filteredContacts[0].id);
    }
  }, [filteredContacts, selectedContactId]);

  useEffect(() => {
    if (!focusContactId) return;
    setSelectedContactId(focusContactId);
    onFocusContactHandled?.();
  }, [focusContactId, onFocusContactHandled]);

  useEffect(() => {
    if (!requestJSON) return;
    loadCrmResources().catch((error) => setCrmError(error.message));
  }, [requestJSON, apiBase, canUseProspects]);

  useEffect(() => {
    if (canUseProspects || broadcastDraft.audience !== "deal_stage") return;
    setBroadcastDraft((current) => ({ ...current, audience: "selected", dealStageId: "" }));
  }, [broadcastDraft.audience, canUseProspects]);

  useEffect(() => {
    if (!requestJSON || !selectedContact?.id) return;
    loadContactDetail(selectedContact.id).catch((error) => setCrmError(error.message));
  }, [requestJSON, apiBase, selectedContact?.id]);

  function hydrateDetail(payload) {
    const contact = payload?.contact || {};
    setDetail(payload);
    setSelectedTagIds(safeArray(payload?.tags).map((tag) => tag.id));
    setProfileDraft({
      name: contact.name || "",
      email: contact.email || "",
      notes: contact.notes || "",
      lifecycleStatus: contact.lifecycleStatus || "lead",
      ownerAgentId: contact.ownerAgentId || "",
      customFieldsText: JSON.stringify(contact.customFields || {}, null, 2),
      lastSummary: contact.lastSummary || payload?.aiSummary || "",
      customerMemoryEnabled: contact.customerMemoryEnabled !== false,
    });
  }

  async function loadCrmResources() {
    const [tagPayload, taskPayload, templatePayload, broadcastPayload, dealPayload] = await Promise.all([
      requestJSON(`${apiBase}/api/contact-tags`),
      requestJSON(`${apiBase}/api/follow-up-tasks?status=open`),
      requestJSON(`${apiBase}/api/message-templates`),
      requestJSON(`${apiBase}/api/broadcasts`),
      canUseProspects ? requestJSON(`${apiBase}/api/deals`) : Promise.resolve({ stages: [] }),
    ]);
    setContactTags(tagPayload.items || []);
    setTasks(taskPayload.items || []);
    setTemplates(templatePayload.items || []);
    setBroadcasts(broadcastPayload.items || []);
    setDealStages(canUseProspects ? (dealPayload.stages || []) : []);
  }

  async function loadContactDetail(contactId) {
    setBusyKey("contact-detail");
    setCrmError("");
    try {
      const payload = await requestJSON(`${apiBase}/api/contacts/${encodeURIComponent(contactId)}`);
      hydrateDetail(payload);
    } finally {
      setBusyKey("");
    }
  }

  async function saveProfile() {
    if (!detail?.contact?.id) return;
    let customFields = {};
    try {
      customFields = profileDraft.customFieldsText.trim() ? JSON.parse(profileDraft.customFieldsText) : {};
    } catch {
      setCrmError("Custom fields harus berupa JSON yang valid.");
      return;
    }
    setBusyKey("profile-save");
    setCrmError("");
    try {
      const payload = await requestJSON(`${apiBase}/api/contacts/${encodeURIComponent(detail.contact.id)}`, {
        method: "PATCH",
        body: JSON.stringify({
          name: profileDraft.name,
          email: profileDraft.email,
          notes: profileDraft.notes,
          lifecycleStatus: profileDraft.lifecycleStatus,
          ownerAgentId: profileDraft.ownerAgentId,
          customFields,
          lastSummary: profileDraft.lastSummary,
        }),
      });
      hydrateDetail(payload);
      setCrmNotice("Profil customer disimpan.");
      onRefresh?.();
    } catch (error) {
      setCrmError(error.message);
    } finally {
      setBusyKey("");
    }
  }

  async function saveTags() {
    if (!detail?.contact?.id) return;
    setBusyKey("tags-save");
    setCrmError("");
    try {
      await requestJSON(`${apiBase}/api/contacts/${encodeURIComponent(detail.contact.id)}/tags`, {
        method: "POST",
        body: JSON.stringify({ tagIds: selectedTagIds }),
      });
      await loadContactDetail(detail.contact.id);
      onRefresh?.();
      setCrmNotice("Tags customer diperbarui.");
    } catch (error) {
      setCrmError(error.message);
    } finally {
      setBusyKey("");
    }
  }

  async function createTag() {
    if (!newTag.name.trim()) return;
    setBusyKey("tag-create");
    setCrmError("");
    try {
      const payload = await requestJSON(`${apiBase}/api/contact-tags`, {
        method: "POST",
        body: JSON.stringify(newTag),
      });
      setContactTags((current) => [...current, payload.item].filter(Boolean));
      if (payload.item?.id) setSelectedTagIds((current) => Array.from(new Set([...current, payload.item.id])));
      setNewTag({ name: "", color: "#2563eb" });
      setCrmNotice("Tag baru dibuat.");
    } catch (error) {
      setCrmError(error.message);
    } finally {
      setBusyKey("");
    }
  }

  async function addNote() {
    if (!detail?.contact?.id || !noteDraft.trim()) return;
    setBusyKey("note-add");
    setCrmError("");
    try {
      await requestJSON(`${apiBase}/api/contacts/${encodeURIComponent(detail.contact.id)}/notes`, {
        method: "POST",
        body: JSON.stringify({ body: noteDraft }),
      });
      setNoteDraft("");
      await loadContactDetail(detail.contact.id);
      setCrmNotice("Catatan tersimpan.");
    } catch (error) {
      setCrmError(error.message);
    } finally {
      setBusyKey("");
    }
  }

  async function addMemory() {
    if (!detail?.contact?.id || !memoryDraft.value.trim()) return;
    setBusyKey("memory-add");
    setCrmError("");
    try {
      await requestJSON(`${apiBase}/api/contacts/${encodeURIComponent(detail.contact.id)}/memories`, {
        method: "POST",
        body: JSON.stringify(memoryDraft),
      });
      setMemoryDraft({ memoryType: "admin_note", value: "" });
      await loadContactDetail(detail.contact.id);
      setCrmNotice("Memory customer tersimpan.");
    } catch (error) {
      setCrmError(error.message);
    } finally {
      setBusyKey("");
    }
  }

  async function deleteMemory(memoryId) {
    if (!detail?.contact?.id || !memoryId) return;
    setBusyKey(`memory-${memoryId}`);
    setCrmError("");
    try {
      await requestJSON(`${apiBase}/api/contacts/${encodeURIComponent(detail.contact.id)}/memories/${encodeURIComponent(memoryId)}`, {
        method: "DELETE",
      });
      await loadContactDetail(detail.contact.id);
      setCrmNotice("Memory customer dihapus.");
    } catch (error) {
      setCrmError(error.message);
    } finally {
      setBusyKey("");
    }
  }

  async function createTask() {
    if (!detail?.contact?.id || !taskDraft.title.trim()) return;
    setBusyKey("task-create");
    setCrmError("");
    try {
      await requestJSON(`${apiBase}/api/follow-up-tasks`, {
        method: "POST",
        body: JSON.stringify({
          contactId: detail.contact.id,
          title: taskDraft.title,
          notes: taskDraft.notes,
          dueAt: dateTimeLocalToISO(taskDraft.dueAt),
          priority: taskDraft.priority,
          assignedToId: taskDraft.assignedToId,
        }),
      });
      setTaskDraft({ title: "", notes: "", dueAt: "", priority: "normal", assignedToId: "" });
      await Promise.all([loadContactDetail(detail.contact.id), loadCrmResources()]);
      onRefresh?.();
      setCrmNotice("Follow-up dibuat.");
    } catch (error) {
      setCrmError(error.message);
    } finally {
      setBusyKey("");
    }
  }

  async function updateTask(taskId, patch) {
    setBusyKey(`task-${taskId}`);
    setCrmError("");
    try {
      await requestJSON(`${apiBase}/api/follow-up-tasks/${encodeURIComponent(taskId)}`, {
        method: "PATCH",
        body: JSON.stringify(patch),
      });
      await Promise.all([detail?.contact?.id ? loadContactDetail(detail.contact.id) : Promise.resolve(), loadCrmResources()]);
      onRefresh?.();
    } catch (error) {
      setCrmError(error.message);
    } finally {
      setBusyKey("");
    }
  }

  async function importContacts() {
    const rows = parseImportRows(importText);
    if (!rows.length) {
      setCrmError("Isi minimal satu kontak. Format: phone,name,email,notes,tags");
      return;
    }
    setBusyKey("contacts-import");
    setCrmError("");
    try {
      const payload = await requestJSON(`${apiBase}/api/contacts/import`, {
        method: "POST",
        body: JSON.stringify({ contacts: rows }),
      });
      setImportText("");
      setCrmNotice(`${formatNumber(payload.imported || 0)} kontak diimport.`);
      await Promise.all([loadCrmResources(), onRefresh?.()]);
    } catch (error) {
      setCrmError(error.message);
    } finally {
      setBusyKey("");
    }
  }

  async function saveTemplate() {
    if (!templateDraft.name.trim() || !templateDraft.body.trim()) return;
    setBusyKey("template-save");
    setCrmError("");
    try {
      await requestJSON(`${apiBase}/api/message-templates`, {
        method: "POST",
        body: JSON.stringify(templateDraft),
      });
      setTemplateDraft({ name: "", category: "follow_up", body: "", status: "draft" });
      await loadCrmResources();
      setCrmNotice("Template WhatsApp disimpan.");
    } catch (error) {
      setCrmError(error.message);
    } finally {
      setBusyKey("");
    }
  }

  async function createBroadcast(sendNow = false) {
    const selectedTemplate = templates.find((template) => template.id === broadcastDraft.templateId);
    const body = (broadcastDraft.body || selectedTemplate?.body || "").trim();
    const contactIds = broadcastDraft.audience === "filtered"
      ? filteredContacts.map((contact) => contact.id)
      : broadcastDraft.audience === "selected"
        ? [detail?.contact?.id || selectedContact?.id].filter(Boolean)
        : [];
    if (!body || (["selected", "filtered"].includes(broadcastDraft.audience) && !contactIds.length)) {
      setCrmError("Pilih kontak dan isi pesan broadcast dulu.");
      return;
    }
    setBusyKey(sendNow ? "broadcast-send" : "broadcast-draft");
    setCrmError("");
    try {
      await requestJSON(`${apiBase}/api/broadcasts`, {
        method: "POST",
        body: JSON.stringify({
          title: broadcastDraft.title || selectedTemplate?.name || "Broadcast WhatsApp",
          templateId: broadcastDraft.templateId,
          body,
          contactIds,
          audience: broadcastDraft.audience,
          tagIds: broadcastDraft.tagId ? [broadcastDraft.tagId] : [],
          dealStageId: broadcastDraft.dealStageId,
          lifecycleStatus: broadcastDraft.lifecycleStatus,
          noFollowUpDays: Number(broadcastDraft.noFollowUpDays) || 7,
          sendNow,
        }),
      });
      setBroadcastDraft({ title: "", templateId: "", body: "", audience: "selected", tagId: "", dealStageId: "", noFollowUpDays: "7", lifecycleStatus: "prospect" });
      await loadCrmResources();
      setCrmNotice(sendNow ? "Broadcast dikirim." : "Draft broadcast disimpan.");
    } catch (error) {
      setCrmError(error.message);
    } finally {
      setBusyKey("");
    }
  }

  function toggleTag(tagId) {
    setSelectedTagIds((current) => current.includes(tagId) ? current.filter((id) => id !== tagId) : [...current, tagId]);
  }

  const detailContact = detail?.contact;
  const detailTags = safeArray(detail?.tags);
  const detailTasks = safeArray(detail?.tasks);
  const detailMessages = safeArray(detail?.messages);
  const detailNotes = safeArray(detail?.notes);
  const detailMemories = safeArray(detail?.memories);
  const recentTasks = tasks.slice(0, 5);

  return (
    <>
      <div className="page-title-row">
        <div>
          <div className="page-title">Kontak</div>
          <div className="page-desc">Data dan riwayat pelanggan.</div>
        </div>
        <div className="page-actions">
          <button className="btn btn-secondary" type="button" onClick={() => setImportText("phone,name,email,notes,tags\n628123456789,Budi,budi@email.com,Prospek paket Pro,prospect|hot")}>Contoh import</button>
          <button className="btn btn-primary" type="button" onClick={importContacts} disabled={busyKey === "contacts-import"}>Import contacts</button>
        </div>
      </div>

      <div className="contacts-kpi-grid">
        <ContactKpi tone="blue" value={formatNumber(enrichedContacts.length)} label="Total Kontak" meta={`${formatNumber(filteredContacts.length)} sesuai filter`} icon={<span>360</span>} />
        <ContactKpi tone="green" value={formatNumber(aiCount)} label="Aktif di AI" meta="Sedang otomatis" icon={<span>AI</span>} />
        <ContactKpi tone="orange" value={formatNumber(escalatedCount)} label="Pernah Eskalasi" meta="Butuh perhatian tim" icon={<span>!</span>} />
        <ContactKpi tone="purple" value={formatNumber(openTaskCount)} label="Follow-up Open" meta={`${formatNumber(recentTasks.length)} terdekat`} icon={<span>OK</span>} />
      </div>

      {(crmError || crmNotice) && (
        <div className={`crm-alert ${crmError ? "error" : "success"}`}>
          {crmError || crmNotice}
          <button type="button" onClick={() => { setCrmError(""); setCrmNotice(""); }}>Tutup</button>
        </div>
      )}

      <div className="contacts-layout crm-contacts-layout">
        <div className="card contacts-table-card">
          <div className="card-header contacts-card-header">
            <div>
              <div className="card-title">Daftar Kontak</div>
              <div className="card-subtitle">Klik detail untuk membuka profil customer.</div>
            </div>
            <div className="contacts-filter-row">
              <div className="contacts-search">
                <svg viewBox="0 0 24 24" fill="none" strokeWidth="2"><circle cx="11" cy="11" r="8" /><line x1="21" y1="21" x2="16.65" y2="16.65" /></svg>
                <input value={query} onChange={(event) => setQuery(event.target.value)} type="text" placeholder="Cari nama, nomor, tag..." />
              </div>
              <select className="form-select contacts-status-select" value={statusFilter} onChange={(event) => setStatusFilter(event.target.value)}>
                <option value="all">Semua chat</option>
                <option value="ai">AI Active</option>
                <option value="human">Human</option>
                <option value="resolved">Resolved</option>
                <option value="new">New</option>
              </select>
              <select className="form-select contacts-status-select" value={lifecycleFilter} onChange={(event) => setLifecycleFilter(event.target.value)}>
                <option value="all">Semua lifecycle</option>
                {lifecycleOptions.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
              </select>
            </div>
          </div>

          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Kontak</th>
                  <th>Lifecycle</th>
                  <th>Tags</th>
                  <th>Owner</th>
                  <th>Follow-up</th>
                  <th>Terakhir Aktif</th>
                  <th>Aksi</th>
                </tr>
              </thead>
              <tbody>
                {filteredContacts.map((contact) => (
                  <tr key={`${contact.id}-${contact.conversationId || "contact"}`} className={contact.id === selectedContactId ? "crm-selected-row" : ""}>
                    <td>
                      <div className="contacts-customer-cell">
                        <div className={`avatar avatar-sm ${contact.avatarColor}`}>{contactInitial(contact)}</div>
                        <div>
                          <strong>{contact.name || "Unknown"}</strong>
                          <div className="text-xs text-muted">{contact.phone}</div>
                        </div>
                      </div>
                    </td>
                    <td><span className={`badge ${lifecycleBadge(contact.lifecycleStatus)}`}>{lifecycleOptions.find((item) => item.value === contact.lifecycleStatus)?.label || "Lead"}</span></td>
                    <td>
                      <div className="crm-tag-row">
                        {safeArray(contact.tags).slice(0, 3).map((tag) => <span key={tag.id || tag.name} className="crm-tag-chip" style={{ borderColor: tag.color }}>{tag.name}</span>)}
                        {!safeArray(contact.tags).length && <span className="text-xs text-muted">Belum ada</span>}
                      </div>
                    </td>
                    <td className="text-sm">{contact.ownerName || "-"}</td>
                    <td>
                      {contact.openTasks ? <span className="badge orange">{contact.openTasks} open</span> : <span className="badge gray">Kosong</span>}
                    </td>
                    <td>
                      <div className="text-xs text-muted">{formatDateTime(contact.lastMessageAt || contact.updatedAt || contact.createdAt)}</div>
                      {contact.lastMessageText && <div className="contacts-last-message">{truncateText(contact.lastMessageText, 44)}</div>}
                      {contact.conversationId ? (
                        <span className={`badge ${contact.whatsappWindow?.requiresTemplate ? "orange" : "green"}`}>
                          {contact.whatsappWindow?.requiresTemplate ? "Template 24h+" : "Free-form 24h"}
                        </span>
                      ) : null}
                    </td>
                    <td>
                      <div className="contacts-row-actions">
                        <button className="btn btn-primary btn-sm" type="button" onClick={() => setSelectedContactId(contact.id)}>Detail</button>
                        <a href="/dashboard/inbox" className="btn btn-ghost btn-sm">Chat</a>
                      </div>
                    </td>
                  </tr>
                ))}
                {!filteredContacts.length && (
                  <tr>
                    <td colSpan={7} className="contacts-empty-cell">Tidak ada data kontak.</td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>

        <div className="contacts-side crm-contact-panel">
          {!detailContact ? (
            <div className="card"><MiniEmpty title={busyKey === "contact-detail" ? "Memuat detail" : "Pilih kontak"} copy="Profil customer akan tampil di sini." /></div>
          ) : (
            <>
              <div className="card crm-profile-card">
                <div className="crm-profile-header">
                  <div className={`avatar avatar-lg ${selectedContact?.avatarColor || "blue"}`}>{contactInitial(detailContact)}</div>
                  <div>
                    <div className="crm-profile-name">{detailContact.name || detailContact.phone}</div>
                    <div className="crm-profile-phone">{detailContact.phone}</div>
                    <div className="crm-tag-row mt-8">
                      <span className={`badge ${statusBadge(selectedContact?.statusGroup)}`}>{statusLabel(selectedContact?.statusGroup)}</span>
                      <span className={`badge ${lifecycleBadge(profileDraft.lifecycleStatus)}`}>{lifecycleOptions.find((item) => item.value === profileDraft.lifecycleStatus)?.label}</span>
                    </div>
                  </div>
                </div>

                <div className="crm-form-grid">
                  <label>Nama<input className="form-input" value={profileDraft.name} onChange={(event) => setProfileDraft((current) => ({ ...current, name: event.target.value }))} /></label>
                  <label>Email<input className="form-input" value={profileDraft.email} onChange={(event) => setProfileDraft((current) => ({ ...current, email: event.target.value }))} /></label>
                  <label>Lifecycle<select className="form-select" value={profileDraft.lifecycleStatus} onChange={(event) => setProfileDraft((current) => ({ ...current, lifecycleStatus: event.target.value }))}>{lifecycleOptions.map((item) => <option key={item.value} value={item.value}>{item.label}</option>)}</select></label>
                  <label>Owner<select className="form-select" value={profileDraft.ownerAgentId} onChange={(event) => setProfileDraft((current) => ({ ...current, ownerAgentId: event.target.value }))}><option value="">Belum diassign</option>{teamMembers.map((member) => <option key={member.id} value={member.id}>{member.name || member.username}</option>)}</select></label>
                </div>
                <label className="crm-field-block">Catatan profil<textarea className="form-textarea" value={profileDraft.notes} onChange={(event) => setProfileDraft((current) => ({ ...current, notes: event.target.value }))} /></label>
                <label className="crm-field-block">Custom fields JSON<textarea className="form-textarea crm-code-textarea" value={profileDraft.customFieldsText} onChange={(event) => setProfileDraft((current) => ({ ...current, customFieldsText: event.target.value }))} /></label>
                <label className="crm-field-block">Ringkasan AI<textarea className="form-textarea" value={profileDraft.lastSummary} onChange={(event) => setProfileDraft((current) => ({ ...current, lastSummary: event.target.value }))} /></label>
                <div className="crm-actions-row">
                  <button className="btn btn-primary" type="button" onClick={saveProfile} disabled={busyKey === "profile-save"}>Simpan profil</button>
                </div>
              </div>

              <div className="card crm-section-card">
                <div className="crm-section-header">
                  <div>
                    <div className="card-title">Customer Memory</div>
                    <div className="card-subtitle">Konteks personal customer, bukan sumber harga/stok/policy.</div>
                  </div>
                </div>
                <div className="crm-inline-form">
                  <select className="form-select" value={memoryDraft.memoryType} onChange={(event) => setMemoryDraft((current) => ({ ...current, memoryType: event.target.value }))}>
                    <option value="admin_note">Admin note</option>
                    <option value="preference">Preference</option>
                    <option value="interest">Interest</option>
                    <option value="communication_preference">Communication</option>
                    <option value="open_issue">Open issue</option>
                  </select>
                  <input className="form-input" placeholder="Contoh: suka rasa coklat" value={memoryDraft.value} onChange={(event) => setMemoryDraft((current) => ({ ...current, value: event.target.value }))} />
                  <button className="btn btn-secondary" type="button" onClick={addMemory} disabled={busyKey === "memory-add"}>Tambah</button>
                </div>
                <div className="crm-list-stack">
                  {detailMemories.map((memory) => (
                    <div className="crm-note-item" key={memory.id}>
                      <p>{memory.value}</p>
                      <span>{toLabel(memory.memoryType || memory.type)} - {memory.sourceType || "ai"} - {formatDateTime(memory.updatedAt || memory.createdAt)}</span>
                      <button className="btn btn-ghost btn-sm" type="button" onClick={() => deleteMemory(memory.id)} disabled={busyKey === `memory-${memory.id}`}>Hapus</button>
                    </div>
                  ))}
                  {!detailMemories.length && <MiniEmpty title="Belum ada memory" copy="Memory baru dipakai setelah long-term memory agent aktif." />}
                </div>
              </div>

              <div className="card crm-section-card">
                <div className="crm-section-header">
                  <div>
                    <div className="card-title">Tags</div>
                    <div className="card-subtitle">Segmentasi untuk follow-up, broadcast, dan laporan.</div>
                  </div>
                  <button className="btn btn-secondary btn-sm" type="button" onClick={saveTags} disabled={busyKey === "tags-save"}>Simpan tags</button>
                </div>
                <div className="crm-checkbox-grid">
                  {contactTags.map((tag) => (
                    <label key={tag.id} className={`crm-tag-option ${selectedTagIds.includes(tag.id) ? "selected" : ""}`}>
                      <input type="checkbox" checked={selectedTagIds.includes(tag.id)} onChange={() => toggleTag(tag.id)} />
                      <span style={{ background: tag.color }} />
                      {tag.name}
                    </label>
                  ))}
                  {!contactTags.length && <MiniEmpty title="Belum ada tag" copy="Buat tag pertama di bawah ini." />}
                </div>
                <div className="crm-inline-form">
                  <input className="form-input" placeholder="Nama tag baru" value={newTag.name} onChange={(event) => setNewTag((current) => ({ ...current, name: event.target.value }))} />
                  <input className="form-input crm-color-input" type="color" value={newTag.color} onChange={(event) => setNewTag((current) => ({ ...current, color: event.target.value }))} />
                  <button className="btn btn-secondary" type="button" onClick={createTag} disabled={busyKey === "tag-create"}>Tambah tag</button>
                </div>
              </div>

              <div className="card crm-section-card">
                <div className="crm-section-header">
                  <div>
                    <div className="card-title">Follow-up & Task</div>
                    <div className="card-subtitle">Reminder yang langsung nempel ke customer.</div>
                  </div>
                </div>
                <div className="crm-task-form">
                  <input className="form-input" placeholder="Contoh: Follow-up penawaran Pro" value={taskDraft.title} onChange={(event) => setTaskDraft((current) => ({ ...current, title: event.target.value }))} />
                  <input className="form-input" type="datetime-local" value={taskDraft.dueAt} onChange={(event) => setTaskDraft((current) => ({ ...current, dueAt: event.target.value }))} />
                  <select className="form-select" value={taskDraft.priority} onChange={(event) => setTaskDraft((current) => ({ ...current, priority: event.target.value }))}>{priorityOptions.map((item) => <option key={item.value} value={item.value}>{item.label}</option>)}</select>
                  <select className="form-select" value={taskDraft.assignedToId} onChange={(event) => setTaskDraft((current) => ({ ...current, assignedToId: event.target.value }))}><option value="">Assign nanti</option>{teamMembers.map((member) => <option key={member.id} value={member.id}>{member.name || member.username}</option>)}</select>
                  <textarea className="form-textarea" placeholder="Catatan task" value={taskDraft.notes} onChange={(event) => setTaskDraft((current) => ({ ...current, notes: event.target.value }))} />
                  <button className="btn btn-primary" type="button" onClick={createTask} disabled={busyKey === "task-create"}>Buat follow-up</button>
                </div>
                <div className="crm-list-stack">
                  {detailTasks.map((task) => (
                    <div className="crm-task-item" key={task.id}>
                      <div>
                        <strong>{task.title}</strong>
                        <div className="text-xs text-muted">{task.dueAt ? formatDateTime(task.dueAt) : "Tanpa deadline"} {task.assignedToName ? `- ${task.assignedToName}` : ""}</div>
                      </div>
                      <div className="crm-task-actions">
                        <span className={`badge ${priorityBadge(task.priority)}`}>{task.priority}</span>
                        {task.status === "open" ? <button className="btn btn-ghost btn-sm" type="button" onClick={() => updateTask(task.id, { status: "done" })}>Selesai</button> : <span className="badge gray">{toLabel(task.status)}</span>}
                      </div>
                    </div>
                  ))}
                  {!detailTasks.length && <MiniEmpty title="Belum ada follow-up" copy="Buat task supaya customer tidak lepas dari radar." />}
                </div>
              </div>

              <div className="card crm-section-card">
                <div className="crm-section-header">
                  <div>
                    <div className="card-title">Note Internal</div>
                    <div className="card-subtitle">Catatan ini tidak dikirim ke customer.</div>
                  </div>
                </div>
                <textarea className="form-textarea" placeholder="Tulis konteks penting customer..." value={noteDraft} onChange={(event) => setNoteDraft(event.target.value)} />
                <div className="crm-actions-row"><button className="btn btn-secondary" type="button" onClick={addNote} disabled={busyKey === "note-add"}>Tambah note</button></div>
                <div className="crm-list-stack">
                  {detailNotes.slice(0, 4).map((note) => (
                    <div className="crm-note-item" key={note.id}>
                      <p>{note.body}</p>
                      <span>{note.authorName || "Tim"} - {formatDateTime(note.createdAt)}</span>
                    </div>
                  ))}
                  {!detailNotes.length && <MiniEmpty title="Belum ada note" copy="Tambahkan konteks yang perlu diketahui tim." />}
                </div>
              </div>

              <div className="card crm-section-card">
                <div className="crm-section-header">
                  <div>
                    <div className="card-title">Riwayat Chat</div>
                    <div className="card-subtitle">80 pesan terakhir dari WhatsApp.</div>
                  </div>
                </div>
                <div className="crm-message-list">
                  {detailMessages.slice(0, 12).map((message, index) => (
                    <div className={`crm-message ${message.direction === "inbound" ? "inbound" : "outbound"}`} key={`${message.createdAt}-${index}`}>
                      <div>{truncateText(message.text || message.contentType, 160)}</div>
                      <span>{message.senderType} - {formatDateTime(message.createdAt)}</span>
                    </div>
                  ))}
                  {!detailMessages.length && <MiniEmpty title="Belum ada chat" copy="Riwayat akan muncul setelah customer mengirim pesan." />}
                </div>
              </div>
            </>
          )}
        </div>
      </div>

      <div className="crm-tools-grid">
        <div className="card crm-section-card">
          <div className="crm-section-header">
            <div>
              <div className="card-title">Import Kontak</div>
              <div className="card-subtitle">CSV sederhana: phone,name,email,notes,tags. Pisahkan tags dengan tanda |.</div>
            </div>
          </div>
          <textarea className="form-textarea crm-import-textarea" value={importText} onChange={(event) => setImportText(event.target.value)} placeholder="phone,name,email,notes,tags&#10;628123456789,Budi,budi@email.com,Minat paket Pro,prospect|hot" />
          <button className="btn btn-primary" type="button" onClick={importContacts} disabled={busyKey === "contacts-import"}>Import</button>
        </div>

        <div className="card crm-section-card">
          <div className="crm-section-header">
            <div>
              <div className="card-title">Template WhatsApp</div>
              <div className="card-subtitle">Simpan template pesan untuk follow-up dan broadcast.</div>
            </div>
          </div>
          <div className="crm-template-form">
            <input className="form-input" placeholder="Nama template" value={templateDraft.name} onChange={(event) => setTemplateDraft((current) => ({ ...current, name: event.target.value }))} />
            <select className="form-select" value={templateDraft.status} onChange={(event) => setTemplateDraft((current) => ({ ...current, status: event.target.value }))}>
              <option value="draft">Draft</option>
              <option value="active">Aktif</option>
            </select>
            <textarea className="form-textarea" placeholder="Isi pesan..." value={templateDraft.body} onChange={(event) => setTemplateDraft((current) => ({ ...current, body: event.target.value }))} />
            <button className="btn btn-secondary" type="button" onClick={saveTemplate} disabled={busyKey === "template-save"}>Simpan template</button>
          </div>
          <div className="crm-list-stack">
            {templates.slice(0, 3).map((template) => (
              <div className="crm-template-item" key={template.id}>
                <strong>{template.name}</strong>
                <span>{truncateText(template.body, 90)}</span>
              </div>
            ))}
          </div>
        </div>

        <div className="card crm-section-card">
          <div className="crm-section-header">
            <div>
              <div className="card-title">Broadcast WhatsApp</div>
              <div className="card-subtitle">Kirim ke kontak, tag, lifecycle, atau customer yang lama tidak difollow-up.</div>
            </div>
          </div>
          <div className="crm-template-form">
            <input className="form-input" placeholder="Judul broadcast" value={broadcastDraft.title} onChange={(event) => setBroadcastDraft((current) => ({ ...current, title: event.target.value }))} />
            <select className="form-select" value={broadcastDraft.templateId} onChange={(event) => {
              const template = templates.find((item) => item.id === event.target.value);
              setBroadcastDraft((current) => ({ ...current, templateId: event.target.value, body: template?.body || current.body }));
            }}>
              <option value="">Tanpa template</option>
              {templates.map((template) => <option key={template.id} value={template.id}>{template.name}</option>)}
            </select>
            <select className="form-select" value={broadcastDraft.audience} onChange={(event) => setBroadcastDraft((current) => ({ ...current, audience: event.target.value }))}>
              <option value="selected">Kontak terpilih</option>
              <option value="filtered">Semua hasil filter ({filteredContacts.length})</option>
              <option value="tag">Tag kontak</option>
              <option value="lifecycle">Lifecycle</option>
              {canUseProspects ? <option value="deal_stage">Stage follow-up</option> : null}
              <option value="stale_follow_up">Belum follow-up</option>
            </select>
            {broadcastDraft.audience === "tag" ? (
              <select className="form-select" value={broadcastDraft.tagId} onChange={(event) => setBroadcastDraft((current) => ({ ...current, tagId: event.target.value }))}>
                <option value="">Pilih tag</option>
                {contactTags.map((tag) => <option key={tag.id} value={tag.id}>{tag.name} ({tag.contactCount || 0})</option>)}
              </select>
            ) : null}
            {broadcastDraft.audience === "lifecycle" ? (
              <select className="form-select" value={broadcastDraft.lifecycleStatus} onChange={(event) => setBroadcastDraft((current) => ({ ...current, lifecycleStatus: event.target.value }))}>
                <option value="lead">Lead</option>
                <option value="prospect">Prospect</option>
                <option value="customer">Customer</option>
                <option value="inactive">Nonaktif</option>
              </select>
            ) : null}
            {canUseProspects && broadcastDraft.audience === "deal_stage" ? (
              <select className="form-select" value={broadcastDraft.dealStageId} onChange={(event) => setBroadcastDraft((current) => ({ ...current, dealStageId: event.target.value }))}>
                <option value="">Pilih stage deal</option>
                {dealStages.map((stage) => <option key={stage.id} value={stage.id}>{stage.name}</option>)}
              </select>
            ) : null}
            {broadcastDraft.audience === "stale_follow_up" ? (
              <input className="form-input" type="number" min="1" max="90" value={broadcastDraft.noFollowUpDays} onChange={(event) => setBroadcastDraft((current) => ({ ...current, noFollowUpDays: event.target.value }))} placeholder="Hari tanpa follow-up" />
            ) : null}
            <textarea className="form-textarea" placeholder="Isi pesan broadcast..." value={broadcastDraft.body} onChange={(event) => setBroadcastDraft((current) => ({ ...current, body: event.target.value }))} />
            <div className="crm-actions-row">
              <button className="btn btn-secondary" type="button" onClick={() => createBroadcast(false)} disabled={busyKey === "broadcast-draft"}>Simpan draft</button>
              <button className="btn btn-primary" type="button" onClick={() => createBroadcast(true)} disabled={busyKey === "broadcast-send"}>Kirim sekarang</button>
            </div>
          </div>
          <div className="crm-list-stack">
            {broadcasts.slice(0, 3).map((broadcast) => (
              <div className="crm-template-item" key={broadcast.id}>
                <strong>{broadcast.title}</strong>
                <span>{toLabel(broadcast.status)} - {broadcast.sentCount}/{broadcast.recipientCount} terkirim</span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </>
  );
}
