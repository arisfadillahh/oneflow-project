import { useEffect, useMemo, useState } from "react";
import { Icons, formatCurrencyIDR, formatDateTime, truncateText } from "../../../lib/dashboard-core";

const priorityOptions = [
  { value: "low", label: "Rendah" },
  { value: "normal", label: "Normal" },
  { value: "high", label: "Tinggi" },
  { value: "urgent", label: "Mendesak" },
];

const emptyBoard = { activePipelineId: "", pipelines: [], stages: [], deals: [], summary: {} };

function priorityTone(priority) {
  if (priority === "urgent") return "red";
  if (priority === "high") return "orange";
  if (priority === "low") return "gray";
  return "blue";
}

function priorityLabel(priority) {
  return priorityOptions.find((item) => item.value === priority)?.label || priority || "-";
}

function statusTone(status) {
  if (status === "won") return "green";
  if (status === "lost") return "red";
  if (status === "archived") return "gray";
  return "blue";
}

function statusLabel(status) {
  if (status === "won") return "Menang";
  if (status === "lost") return "Kalah";
  if (status === "archived") return "Arsip";
  return "Terbuka";
}

function contactLabel(contact) {
  if (!contact) return "";
  return contact.name || contact.phone || contact.email || "";
}

function dealContactLabel(deal) {
  return deal.contactName || deal.contactPhone || "Belum pilih contact";
}

function makeDealDraft(stages, selectedDeal) {
  const firstStage = stages.find((stage) => stage.type === "open") || stages[0];
  return {
    title: "",
    contactId: "",
    stageId: selectedDeal?.stageId || firstStage?.id || "",
    valueAmount: "",
    ownerAgentId: "",
    expectedCloseDate: "",
    priority: "normal",
    source: "WhatsApp",
    notes: "",
  };
}

function aiModeLabel(mode) {
  if (mode === "read") return "AI baca data";
  if (mode === "draft") return "AI bisa buat draf prospek";
  if (mode === "action") return "AI aksi langsung";
  return "AI mati";
}

function prospectAIToolText(toolState) {
  const mode = toolState?.aiMode || "off";
  if (mode === "off") return "AI belum memakai data prospek. Aktifkan izin AI di Alat Bisnis kalau ingin AI mencatat minat pelanggan.";
  if (mode === "read") return "AI boleh membaca konteks prospek untuk menjawab follow-up. Pembuatan prospek tetap manual.";
  return "AI boleh membuat prospek dari chat pelanggan. Tahap, PIC, dan prioritas tetap bisa dikoreksi tim penjualan.";
}

function draftFromDeal(deal) {
  if (!deal) return null;
  return {
    title: deal.title || "",
    contactId: deal.contactId || "",
    stageId: deal.stageId || "",
    valueAmount: String(deal.valueAmount || ""),
    ownerAgentId: deal.ownerAgentId || "",
    expectedCloseDate: deal.expectedCloseDate || "",
    priority: deal.priority || "normal",
    source: deal.source || "",
    notes: deal.notes || "",
    lossReason: deal.lossReason || "",
  };
}

function toAmount(value) {
  const parsed = Number(String(value || "").replace(/[^\d]/g, ""));
  return Number.isFinite(parsed) ? parsed : 0;
}

function stageValue(deals) {
  return deals.reduce((sum, deal) => sum + Number(deal.valueAmount || 0), 0);
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

function isOverdueActivity(item) {
  if (!item?.dueAt || item.completedAt) return false;
  const due = new Date(item.dueAt);
  return !Number.isNaN(due.getTime()) && due.getTime() < Date.now();
}

function DealSummaryCard({ label, value, meta, tone = "blue" }) {
  return (
    <div className="kpi-card deals-kpi-card">
      <div className={`kpi-icon ${tone}`} />
      <div>
        <div className="kpi-value">{value}</div>
        <div className="kpi-label">{label}</div>
        {meta ? <div className="kpi-meta">{meta}</div> : null}
      </div>
    </div>
  );
}

export function DealsView({ contacts = [], teamMembers = [], apiBase = "", requestJSON, focusDealId = "", onFocusDealHandled, toolState }) {
  const [board, setBoard] = useState(emptyBoard);
  const [activePipelineId, setActivePipelineId] = useState("");
  const [selectedDealId, setSelectedDealId] = useState("");
  const [showCreate, setShowCreate] = useState(false);
  const [dealDraft, setDealDraft] = useState(makeDealDraft([], null));
  const [editDraft, setEditDraft] = useState(null);
  const [filters, setFilters] = useState({ query: "", ownerId: "all", status: "open" });
  const [pipelineDraft, setPipelineDraft] = useState({ name: "", description: "" });
  const [stageDrafts, setStageDrafts] = useState({});
  const [activities, setActivities] = useState([]);
  const [activityBody, setActivityBody] = useState("");
  const [taskDraft, setTaskDraft] = useState({ title: "", dueAt: "", body: "", assignedToId: "" });
  const [busyKey, setBusyKey] = useState("");
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  const stages = board.stages || [];
  const deals = board.deals || [];
  const summary = board.summary || {};
  const selectedDeal = useMemo(() => deals.find((deal) => deal.id === selectedDealId) || deals[0] || null, [deals, selectedDealId]);
  const activePipeline = useMemo(() => (board.pipelines || []).find((pipeline) => pipeline.id === activePipelineId) || null, [board.pipelines, activePipelineId]);
  const contactsForSelect = useMemo(
    () => contacts.filter((contact) => contact?.id && contact.phone !== "__playground__").slice(0, 300),
    [contacts]
  );
  const filteredDeals = useMemo(() => {
    const query = filters.query.trim().toLowerCase();
    return deals.filter((deal) => {
      if (filters.status !== "all" && deal.status !== filters.status) return false;
      if (filters.ownerId !== "all" && (deal.ownerAgentId || "") !== filters.ownerId) return false;
      if (!query) return true;
      return [deal.title, deal.contactName, deal.contactPhone, deal.ownerName, deal.source]
        .filter(Boolean)
        .some((value) => String(value).toLowerCase().includes(query));
    });
  }, [deals, filters]);
  const dealsByStage = useMemo(() => {
    const groups = new Map(stages.map((stage) => [stage.id, []]));
    filteredDeals.forEach((deal) => {
      const group = groups.get(deal.stageId) || [];
      group.push(deal);
      groups.set(deal.stageId, group);
    });
    return groups;
  }, [filteredDeals, stages]);

  async function loadDeals(nextPipelineId = activePipelineId) {
    if (!requestJSON) return;
    const suffix = nextPipelineId ? `?pipelineId=${encodeURIComponent(nextPipelineId)}` : "";
    const payload = await requestJSON(`${apiBase}/api/deals${suffix}`);
    setBoard(payload || emptyBoard);
    setActivePipelineId(payload?.activePipelineId || "");
    setDealDraft((current) => ({ ...makeDealDraft(payload?.stages || [], selectedDeal), ...current, stageId: current.stageId || payload?.stages?.[0]?.id || "" }));
    if (!selectedDealId && payload?.deals?.length) {
      setSelectedDealId(payload.deals[0].id);
    }
  }

  useEffect(() => {
    loadDeals("").catch((loadError) => setError(loadError.message));
  }, [requestJSON]);

  useEffect(() => {
    if (!focusDealId) return;
    setSelectedDealId(focusDealId);
    onFocusDealHandled?.();
  }, [focusDealId, onFocusDealHandled]);

  useEffect(() => {
    setPipelineDraft({ name: activePipeline?.name || "", description: activePipeline?.description || "" });
    setStageDrafts(Object.fromEntries(stages.map((stage) => [stage.id, {
      name: stage.name || "",
      probability: String(stage.probability ?? 0),
      type: stage.type || "open",
      color: stage.color || "#2196F3",
      position: String(stage.position || 0),
    }])));
  }, [activePipeline?.id, activePipeline?.name, activePipeline?.description, stages]);

  useEffect(() => {
    setEditDraft(draftFromDeal(selectedDeal));
    if (!selectedDeal?.id || !requestJSON) {
      setActivities([]);
      return;
    }
    requestJSON(`${apiBase}/api/deals/${encodeURIComponent(selectedDeal.id)}/activities`)
      .then((payload) => setActivities(payload.items || []))
      .catch((activityError) => setError(activityError.message));
  }, [selectedDeal?.id, requestJSON, apiBase]);

  async function reloadAfterChange(nextSelectedId = selectedDealId) {
    await loadDeals(activePipelineId);
    setSelectedDealId(nextSelectedId);
  }

  async function createDeal() {
    setBusyKey("deal-create");
    setError("");
    setNotice("");
    try {
      const payload = await requestJSON(`${apiBase}/api/deals`, {
        method: "POST",
        body: JSON.stringify({
          ...dealDraft,
          pipelineId: activePipelineId,
          valueAmount: toAmount(dealDraft.valueAmount),
        }),
      });
      setShowCreate(false);
      setDealDraft(makeDealDraft(stages, null));
      setNotice("Follow-up baru berhasil dibuat.");
      await reloadAfterChange(payload.item?.id || selectedDealId);
    } catch (createError) {
      setError(createError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function saveDeal() {
    if (!selectedDeal || !editDraft) return;
    setBusyKey("deal-save");
    setError("");
    setNotice("");
    try {
      await requestJSON(`${apiBase}/api/deals/${encodeURIComponent(selectedDeal.id)}`, {
        method: "PATCH",
        body: JSON.stringify({
          ...editDraft,
          valueAmount: toAmount(editDraft.valueAmount),
        }),
      });
      setNotice("Follow-up berhasil disimpan.");
      await reloadAfterChange(selectedDeal.id);
    } catch (saveError) {
      setError(saveError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function moveDeal(deal, stageId) {
    if (!deal?.id || !stageId || deal.stageId === stageId) return;
    setBusyKey(`deal-move-${deal.id}`);
    setError("");
    try {
      await requestJSON(`${apiBase}/api/deals/${encodeURIComponent(deal.id)}`, {
        method: "PATCH",
        body: JSON.stringify({ stageId }),
      });
      await reloadAfterChange(deal.id);
    } catch (moveError) {
      setError(moveError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function addActivity() {
    if (!selectedDeal?.id || !activityBody.trim()) return;
    setBusyKey("deal-activity");
    setError("");
    try {
      const payload = await requestJSON(`${apiBase}/api/deals/${encodeURIComponent(selectedDeal.id)}/activities`, {
        method: "POST",
        body: JSON.stringify({ type: "note", title: "Catatan deal", body: activityBody.trim() }),
      });
      setActivities((current) => [payload.item, ...current]);
      setActivityBody("");
      await reloadAfterChange(selectedDeal.id);
    } catch (activityError) {
      setError(activityError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function addTaskActivity() {
    if (!selectedDeal?.id || !taskDraft.title.trim()) return;
    setBusyKey("deal-task");
    setError("");
    try {
      const payload = await requestJSON(`${apiBase}/api/deals/${encodeURIComponent(selectedDeal.id)}/activities`, {
        method: "POST",
        body: JSON.stringify({
          type: "task",
          title: taskDraft.title.trim(),
          body: taskDraft.body.trim(),
          dueAt: dateTimeLocalToISO(taskDraft.dueAt),
          assignedToId: taskDraft.assignedToId,
        }),
      });
      setActivities((current) => [payload.item, ...current]);
      setTaskDraft({ title: "", dueAt: "", body: "", assignedToId: "" });
      await reloadAfterChange(selectedDeal.id);
    } catch (taskError) {
      setError(taskError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function completeActivity(item, done = true) {
    if (!item?.id) return;
    setBusyKey(`deal-activity-${item.id}`);
    setError("");
    try {
      const payload = await requestJSON(`${apiBase}/api/deal-activities/${encodeURIComponent(item.id)}`, {
        method: "PATCH",
        body: JSON.stringify({ done }),
      });
      setActivities((current) => current.map((activity) => activity.id === item.id ? payload.item : activity));
      await reloadAfterChange(selectedDeal?.id);
    } catch (activityError) {
      setError(activityError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function savePipeline() {
    if (!activePipelineId || !pipelineDraft.name.trim()) return;
    setBusyKey("pipeline-save");
    setError("");
    try {
      await requestJSON(`${apiBase}/api/deal-pipelines/${encodeURIComponent(activePipelineId)}`, {
        method: "PATCH",
        body: JSON.stringify(pipelineDraft),
      });
      setNotice("Pipeline berhasil disimpan.");
      await loadDeals(activePipelineId);
    } catch (pipelineError) {
      setError(pipelineError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function saveStage(stage) {
    const draft = stageDrafts[stage.id];
    if (!draft?.name?.trim()) return;
    setBusyKey(`stage-save-${stage.id}`);
    setError("");
    try {
      await requestJSON(`${apiBase}/api/deal-stages/${encodeURIComponent(stage.id)}`, {
        method: "PATCH",
        body: JSON.stringify({
          name: draft.name,
          probability: Number(draft.probability) || 0,
          type: draft.type || "open",
          color: draft.color || stage.color,
          position: Number(draft.position) || stage.position,
        }),
      });
      setNotice("Tahap berhasil disimpan.");
      await loadDeals(activePipelineId);
    } catch (stageError) {
      setError(stageError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function deleteStage(stage) {
    if (!stage?.id) return;
    if (!window.confirm(`Hapus tahap "${stage.name}"? Prospek di tahap ini akan dipindah ke tahap pertama yang tersisa.`)) return;
    setBusyKey(`stage-delete-${stage.id}`);
    setError("");
    try {
      await requestJSON(`${apiBase}/api/deal-stages/${encodeURIComponent(stage.id)}`, {
        method: "DELETE",
      });
      setNotice("Tahap berhasil dihapus.");
      await loadDeals(activePipelineId);
    } catch (stageError) {
      setError(stageError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function addStage() {
    setBusyKey("stage-create");
    setError("");
    try {
      await requestJSON(`${apiBase}/api/deal-stages`, {
        method: "POST",
        body: JSON.stringify({ pipelineId: activePipelineId, name: "Tahap Baru", probability: 20, type: "open", color: "#2196F3" }),
      });
      setNotice("Tahap baru ditambahkan.");
      await loadDeals(activePipelineId);
    } catch (stageError) {
      setError(stageError.message);
    } finally {
      setBusyKey("");
    }
  }

  return (
    <div className="deals-page">
      {error ? <div className="crm-alert error"><span>{error}</span><button type="button" onClick={() => setError("")}>Tutup</button></div> : null}
      {notice ? <div className="crm-alert"><span>{notice}</span><button type="button" onClick={() => setNotice("")}>Tutup</button></div> : null}

      <div className="business-ai-banner">
        {Icons.robot}
        <div>
          <strong>{aiModeLabel(toolState?.aiMode)}</strong>
          <span>{prospectAIToolText(toolState)}</span>
        </div>
      </div>

      <div className="deals-summary-grid">
        <DealSummaryCard label="Prospek terbuka" value={summary.openCount || 0} meta={formatCurrencyIDR(summary.openValue || 0)} />
        <DealSummaryCard label="Forecast tertimbang" value={formatCurrencyIDR(summary.weightedValue || 0)} meta="Berdasarkan peluang tahap" tone="purple" />
        <DealSummaryCard label="Menang bulan ini" value={formatCurrencyIDR(summary.wonValue || 0)} meta={`${summary.wonCount || 0} prospek closing`} tone="green" />
        <DealSummaryCard label="Perlu closing" value={summary.closingSoon || 0} meta="Jatuh tempo <= 14 hari" tone="orange" />
        <DealSummaryCard label="Reminder lewat tempo" value={summary.overdueActivities || 0} meta={`${summary.dueTodayActivities || 0} reminder hari ini`} tone="red" />
      </div>

      <div className="card deals-toolbar">
        <div>
          <div className="card-title">Prospek & Follow-up</div>
          <div className="card-subtitle">Prospek terhubung ke kontak, PIC, nilai, dan tahap closing.</div>
        </div>
        <div className="deals-toolbar-actions">
          <input
            className="form-input"
            placeholder="Cari prospek/kontak"
            value={filters.query}
            onChange={(event) => setFilters((current) => ({ ...current, query: event.target.value }))}
          />
          <select className="form-select" value={filters.ownerId} onChange={(event) => setFilters((current) => ({ ...current, ownerId: event.target.value }))}>
            <option value="all">Semua PIC</option>
            <option value="">Belum diassign</option>
            {teamMembers.map((member) => <option key={member.id} value={member.id}>{member.name || member.username}</option>)}
          </select>
          <select className="form-select" value={filters.status} onChange={(event) => setFilters((current) => ({ ...current, status: event.target.value }))}>
            <option value="open">Terbuka</option>
            <option value="won">Menang</option>
            <option value="lost">Kalah</option>
            <option value="all">Semua status</option>
          </select>
          <select
            className="form-select"
            value={activePipelineId}
            onChange={(event) => {
              setSelectedDealId("");
              loadDeals(event.target.value).catch((loadError) => setError(loadError.message));
            }}
          >
            {board.pipelines.map((pipeline) => <option key={pipeline.id} value={pipeline.id}>{pipeline.name}</option>)}
          </select>
          <button className="btn btn-secondary" type="button" onClick={() => loadDeals(activePipelineId)} disabled={busyKey === "deals-load"}>Muat ulang</button>
          <button className="btn btn-primary" type="button" onClick={() => setShowCreate((current) => !current)}>{showCreate ? "Tutup Form" : "Prospek Baru"}</button>
        </div>
      </div>

      <details className="card deals-config-panel">
        <summary>Pengaturan alur & tahap</summary>
        <div className="deal-form-grid">
          <label>Nama alur<input className="form-input" value={pipelineDraft.name} onChange={(event) => setPipelineDraft((current) => ({ ...current, name: event.target.value }))} /></label>
          <label>Deskripsi<input className="form-input" value={pipelineDraft.description} onChange={(event) => setPipelineDraft((current) => ({ ...current, description: event.target.value }))} /></label>
        </div>
        <div className="crm-actions-row">
          <button className="btn btn-secondary btn-sm" type="button" onClick={savePipeline} disabled={busyKey === "pipeline-save"}>Simpan alur</button>
          <button className="btn btn-secondary btn-sm" type="button" onClick={addStage} disabled={busyKey === "stage-create"}>Tambah tahap</button>
        </div>
        <div className="business-stage-editor-list deals-stage-settings-list">
          {stages.map((stage, index) => (
            <div className="business-stage-editor-row deals-stage-editor-row" key={stage.id}>
              <div className="business-stage-name-field">
                <input className="form-input" value={stageDrafts[stage.id]?.name || ""} onChange={(event) => setStageDrafts((current) => ({ ...current, [stage.id]: { ...(current[stage.id] || {}), name: event.target.value } }))} />
                <small>Urutan {index + 1}</small>
              </div>
              <input className="form-input" type="number" min="0" max="100" value={stageDrafts[stage.id]?.probability || "0"} onChange={(event) => setStageDrafts((current) => ({ ...current, [stage.id]: { ...(current[stage.id] || {}), probability: event.target.value } }))} aria-label={`Peluang ${stage.name}`} />
              <select className="form-select" value={stageDrafts[stage.id]?.type || "open"} onChange={(event) => setStageDrafts((current) => ({ ...current, [stage.id]: { ...(current[stage.id] || {}), type: event.target.value } }))}>
                <option value="open">Terbuka</option>
                <option value="won">Menang</option>
                <option value="lost">Kalah</option>
              </select>
              <input className="form-input crm-color-input" type="color" value={stageDrafts[stage.id]?.color || stage.color} onChange={(event) => setStageDrafts((current) => ({ ...current, [stage.id]: { ...(current[stage.id] || {}), color: event.target.value } }))} aria-label={`Warna ${stage.name}`} />
              <button className="btn btn-secondary btn-sm" type="button" onClick={() => saveStage(stage)} disabled={busyKey === `stage-save-${stage.id}`}>Simpan</button>
              <button className="btn btn-danger btn-sm" type="button" onClick={() => deleteStage(stage)} disabled={stages.length <= 1 || busyKey === `stage-delete-${stage.id}`}>Hapus</button>
            </div>
          ))}
        </div>
      </details>

      <div className="deals-layout">
        <div className="deals-main">
          <div className="deals-kanban" style={{ "--deal-stage-count": Math.max(stages.length, 1) }}>
            {stages.length ? stages.map((stage) => {
              const stageDeals = dealsByStage.get(stage.id) || [];
              return (
                <section
                  className="deal-stage-column"
                  key={stage.id}
                  onDragOver={(event) => event.preventDefault()}
                  onDrop={(event) => {
                    event.preventDefault();
                    const dealId = event.dataTransfer.getData("text/deal-id");
                    const draggedDeal = deals.find((item) => item.id === dealId);
                    if (draggedDeal) moveDeal(draggedDeal, stage.id);
                  }}
                >
                  <div className="deal-stage-header">
                    <div>
                      <div className="deal-stage-title"><span style={{ background: stage.color }} />{stage.name}</div>
                      <div className="deal-stage-meta">{stageDeals.length} prospek - {formatCurrencyIDR(stageValue(stageDeals))}</div>
                    </div>
                    <span className={`badge ${stage.type === "won" ? "green" : stage.type === "lost" ? "red" : "blue"}`}>{stage.probability}%</span>
                  </div>
                  <div className="deal-card-stack">
                    {stageDeals.map((deal) => (
                      <button
                        className={`deal-card ${selectedDeal?.id === deal.id ? "selected" : ""}`}
                        type="button"
                        key={deal.id}
                        draggable
                        onDragStart={(event) => event.dataTransfer.setData("text/deal-id", deal.id)}
                        onClick={() => setSelectedDealId(deal.id)}
                      >
                        <div className="deal-card-top">
                          <strong>{deal.title}</strong>
                          <span className={`badge ${statusTone(deal.status)}`}>{statusLabel(deal.status)}</span>
                        </div>
                        <div className="deal-card-value">{formatCurrencyIDR(deal.valueAmount || 0)}</div>
                        <div className="deal-card-meta">
                          <span>{dealContactLabel(deal)}</span>
                          <span>{deal.ownerName || "Belum ditugaskan"}</span>
                        </div>
                        <div className="deal-card-footer">
                          <span className={`badge ${priorityTone(deal.priority)}`}>{priorityLabel(deal.priority)}</span>
                          {deal.openActivityCount ? <span className="badge orange">{deal.openActivityCount} task</span> : null}
                          {deal.expectedCloseDate ? <span>{deal.expectedCloseDate}</span> : <span>Belum ada target</span>}
                        </div>
                      </button>
                    ))}
                    {!stageDeals.length ? <div className="crm-mini-empty"><strong>Kosong</strong><span>Belum ada prospek di tahap ini.</span></div> : null}
                  </div>
                </section>
              );
            }) : (
              <div className="crm-mini-empty deals-kanban-empty">
                <strong>Belum ada alur follow-up</strong>
                <span>Muat ulang data atau buat tahap pipeline sebelum tim mulai memindahkan prospek.</span>
              </div>
            )}
          </div>
        </div>

        <aside className="deals-side">
          {showCreate ? (
            <div className="card crm-section-card">
                  <div className="crm-section-header">
                    <div>
                      <div className="card-title">Prospek Baru</div>
                      <div className="card-subtitle">Masukkan peluang penjualan yang perlu ditindaklanjuti tim.</div>
                    </div>
                  </div>
              <div className="deal-form-grid">
                <label>Judul<input className="form-input" value={dealDraft.title} onChange={(event) => setDealDraft((current) => ({ ...current, title: event.target.value }))} placeholder="Contoh: Paket Pro - PT Sinar" /></label>
                <label>Kontak<select className="form-select" value={dealDraft.contactId} onChange={(event) => setDealDraft((current) => ({ ...current, contactId: event.target.value }))}><option value="">Tanpa kontak</option>{contactsForSelect.map((contact) => <option key={contact.id} value={contact.id}>{contactLabel(contact)}</option>)}</select></label>
                <label>Tahap<select className="form-select" value={dealDraft.stageId} onChange={(event) => setDealDraft((current) => ({ ...current, stageId: event.target.value }))}>{stages.map((stage) => <option key={stage.id} value={stage.id}>{stage.name}</option>)}</select></label>
                <label>Nilai<input className="form-input" inputMode="numeric" value={dealDraft.valueAmount} onChange={(event) => setDealDraft((current) => ({ ...current, valueAmount: event.target.value }))} placeholder="2500000" /></label>
                <label>Owner<select className="form-select" value={dealDraft.ownerAgentId} onChange={(event) => setDealDraft((current) => ({ ...current, ownerAgentId: event.target.value }))}><option value="">Belum diassign</option>{teamMembers.map((member) => <option key={member.id} value={member.id}>{member.name || member.username}</option>)}</select></label>
                <label>Estimasi closing<input className="form-input" type="date" value={dealDraft.expectedCloseDate} onChange={(event) => setDealDraft((current) => ({ ...current, expectedCloseDate: event.target.value }))} /></label>
                <label>Prioritas<select className="form-select" value={dealDraft.priority} onChange={(event) => setDealDraft((current) => ({ ...current, priority: event.target.value }))}>{priorityOptions.map((item) => <option key={item.value} value={item.value}>{item.label}</option>)}</select></label>
                <label>Sumber<input className="form-input" value={dealDraft.source} onChange={(event) => setDealDraft((current) => ({ ...current, source: event.target.value }))} /></label>
              </div>
              <label className="crm-field-block">Catatan<textarea className="form-textarea" value={dealDraft.notes} onChange={(event) => setDealDraft((current) => ({ ...current, notes: event.target.value }))} /></label>
              <div className="crm-actions-row">
                <button className="btn btn-primary" type="button" onClick={createDeal} disabled={busyKey === "deal-create" || !dealDraft.title.trim()}>Simpan prospek</button>
                <button className="btn btn-secondary" type="button" onClick={() => setShowCreate(false)}>Batal</button>
              </div>
            </div>
          ) : null}

          <div className="card crm-section-card deals-detail-card">
            {selectedDeal && editDraft ? (
              <>
                <div className="crm-section-header">
                  <div>
                    <div className="card-title">{selectedDeal.title}</div>
                    <div className="card-subtitle">{dealContactLabel(selectedDeal)} - {formatDateTime(selectedDeal.updatedAt)}</div>
                  </div>
                  <span className={`badge ${statusTone(selectedDeal.status)}`}>{statusLabel(selectedDeal.status)}</span>
                </div>
                <div className="deal-form-grid">
                  <label>Judul<input className="form-input" value={editDraft.title} onChange={(event) => setEditDraft((current) => ({ ...current, title: event.target.value }))} /></label>
                  <label>Tahap<select className="form-select" value={editDraft.stageId} onChange={(event) => setEditDraft((current) => ({ ...current, stageId: event.target.value }))}>{stages.map((stage) => <option key={stage.id} value={stage.id}>{stage.name}</option>)}</select></label>
                  <label>Kontak<select className="form-select" value={editDraft.contactId} onChange={(event) => setEditDraft((current) => ({ ...current, contactId: event.target.value }))}><option value="">Tanpa kontak</option>{contactsForSelect.map((contact) => <option key={contact.id} value={contact.id}>{contactLabel(contact)}</option>)}</select></label>
                  <label>Nilai<input className="form-input" inputMode="numeric" value={editDraft.valueAmount} onChange={(event) => setEditDraft((current) => ({ ...current, valueAmount: event.target.value }))} /></label>
                  <label>Owner<select className="form-select" value={editDraft.ownerAgentId} onChange={(event) => setEditDraft((current) => ({ ...current, ownerAgentId: event.target.value }))}><option value="">Belum diassign</option>{teamMembers.map((member) => <option key={member.id} value={member.id}>{member.name || member.username}</option>)}</select></label>
                  <label>Estimasi closing<input className="form-input" type="date" value={editDraft.expectedCloseDate} onChange={(event) => setEditDraft((current) => ({ ...current, expectedCloseDate: event.target.value }))} /></label>
                  <label>Prioritas<select className="form-select" value={editDraft.priority} onChange={(event) => setEditDraft((current) => ({ ...current, priority: event.target.value }))}>{priorityOptions.map((item) => <option key={item.value} value={item.value}>{item.label}</option>)}</select></label>
                  <label>Sumber<input className="form-input" value={editDraft.source} onChange={(event) => setEditDraft((current) => ({ ...current, source: event.target.value }))} /></label>
                </div>
                <label className="crm-field-block">Catatan<textarea className="form-textarea" value={editDraft.notes} onChange={(event) => setEditDraft((current) => ({ ...current, notes: event.target.value }))} /></label>
                <div className="crm-actions-row">
                  <button className="btn btn-primary" type="button" onClick={saveDeal} disabled={busyKey === "deal-save"}>Simpan perubahan</button>
                  {stages.filter((stage) => stage.id !== selectedDeal.stageId).slice(0, 3).map((stage) => (
                    <button className="btn btn-secondary btn-sm" type="button" key={stage.id} onClick={() => moveDeal(selectedDeal, stage.id)} disabled={busyKey === `deal-move-${selectedDeal.id}`}>Ke {stage.name}</button>
                  ))}
                </div>
                <div className="deals-activity-box">
                  <div className="crm-section-header">
                    <div>
                      <div className="card-title">Aktivitas Deal</div>
                      <div className="card-subtitle">Catatan dan reminder tim penjualan yang melekat di peluang.</div>
                    </div>
                  </div>
                  <div className="deal-task-form">
                    <input className="form-input" value={taskDraft.title} onChange={(event) => setTaskDraft((current) => ({ ...current, title: event.target.value }))} placeholder="Langkah berikutnya, contoh: Follow-up proposal" />
                    <input className="form-input" type="datetime-local" value={taskDraft.dueAt} onChange={(event) => setTaskDraft((current) => ({ ...current, dueAt: event.target.value }))} />
                    <select className="form-select" value={taskDraft.assignedToId} onChange={(event) => setTaskDraft((current) => ({ ...current, assignedToId: event.target.value }))}>
                      <option value="">PIC reminder</option>
                      {teamMembers.map((member) => <option key={member.id} value={member.id}>{member.name || member.username}</option>)}
                    </select>
                    <textarea className="form-textarea" value={taskDraft.body} onChange={(event) => setTaskDraft((current) => ({ ...current, body: event.target.value }))} placeholder="Catatan reminder" />
                    <button className="btn btn-secondary" type="button" onClick={addTaskActivity} disabled={busyKey === "deal-task" || !taskDraft.title.trim()}>Tambah reminder</button>
                  </div>
                  <textarea className="form-textarea" value={activityBody} onChange={(event) => setActivityBody(event.target.value)} placeholder="Tulis catatan rapat, negosiasi harga, atau langkah berikutnya." />
                  <button className="btn btn-secondary" type="button" onClick={addActivity} disabled={busyKey === "deal-activity" || !activityBody.trim()}>Tambah catatan</button>
                  <div className="crm-list-stack">
                    {activities.map((item) => (
                      <div className={`crm-note-item deal-activity-item ${isOverdueActivity(item) ? "overdue" : ""}`} key={item.id}>
                        <strong>{item.title} {item.type === "task" ? <span className={`badge ${item.completedAt ? "green" : isOverdueActivity(item) ? "red" : "orange"}`}>{item.completedAt ? "Selesai" : "Task"}</span> : null}</strong>
                        {item.body ? <span>{truncateText(item.body, 180)}</span> : null}
                        <small>{item.assignedToName ? `${item.assignedToName} - ` : ""}{item.createdByName || "Sistem"} - {item.dueAt ? `Jatuh tempo ${formatDateTime(item.dueAt)}` : formatDateTime(item.createdAt)}</small>
                        {item.type === "task" ? (
                          <button className="btn btn-secondary btn-sm" type="button" onClick={() => completeActivity(item, !item.completedAt)} disabled={busyKey === `deal-activity-${item.id}`}>
                            {item.completedAt ? "Buka lagi" : "Selesai"}
                          </button>
                        ) : null}
                      </div>
                    ))}
                    {!activities.length ? <div className="crm-mini-empty"><strong>Belum ada aktivitas</strong><span>Catatan akan muncul setelah deal diproses.</span></div> : null}
                  </div>
                </div>
              </>
            ) : (
              <div className="crm-mini-empty"><strong>Belum ada deal</strong><span>Tambahkan deal pertama untuk mulai tracking sales pipeline.</span></div>
            )}
          </div>
        </aside>
      </div>
    </div>
  );
}
