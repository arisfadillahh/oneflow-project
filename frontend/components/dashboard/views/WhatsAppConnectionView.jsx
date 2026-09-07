import { formatDateTime, toLabel } from "../../../lib/dashboard-core";

export function WhatsAppConnectionView({
  waStatus,
  sessions = [],
  aiAgents = [],
  businessTools = [],
  navigateToAgents,
  activeSessionId,
  setActiveSessionId,
  sessionForm,
  setSessionForm,
  createSession,
  officialWhatsAppEnabled,
  connectOfficialWhatsApp,
  canManageOfficialTemplates,
  metaTemplates = [],
  metaTemplateForm,
  setMetaTemplateForm,
  metaTemplateSendForm,
  setMetaTemplateSendForm,
  loadMetaTemplates,
  createMetaTemplate,
  sendMetaTemplate,
  pendingSessionId,
  renameSession,
  setDefaultSession,
  deleteSession,
  qrImageSrc,
  qrData,
  escalationGroup,
  groupNotificationRules = [],
  canManageEscalationGroup,
  generateEscalationGroupCode,
  removeEscalationGroup,
  sendGroupNotificationTest,
  busyKey,
  requestQr,
  disconnectWhatsApp,
  refreshAll,
  cancelPendingSession,
  billingPlan,
  navigateToUpgrade,
  navigateToBusinessTools
}) {
  const hasSessions = sessions.length > 0;
  const hasAIAgents = aiAgents.some((agent) => agent.isActive !== false);
  const isWaBusy = String(busyKey || "").startsWith("wa-");
  const activeSession = sessions.find((item) => item.id === activeSessionId) || sessions[0];
  const isConnected = activeSession?.status === "connected";
  const showOfficialTemplateTools = Boolean(canManageOfficialTemplates && isConnected);
  const qrActionLabel = "Sambungkan Resmi";
  const maxWhatsAppSessions = Number(billingPlan?.maxWhatsAppSessions ?? billingPlan?.max_whatsapp_sessions ?? 0);
  const planName = billingPlan?.planName || billingPlan?.name || "Trial";
  const isWhatsAppIncluded = maxWhatsAppSessions > 0;
  const isWhatsAppAtLimit = isWhatsAppIncluded && sessions.length >= maxWhatsAppSessions;
  const canCreateWhatsApp = officialWhatsAppEnabled && isWhatsAppIncluded && !isWhatsAppAtLimit && hasAIAgents;
  const handleRefreshStatus = () => refreshAll?.();
  const openQrForSession = (sessionId) => {
    if (sessionId && officialWhatsAppEnabled) connectOfficialWhatsApp?.(sessionId);
  };

  return (
    <>
      <div className="page-title-row">
        <div>
          <div className="page-title">Channels</div>
            <div className="page-desc">Hubungkan channel bisnis ke AI kamu dari satu tempat.</div>
        </div>
        <div className="page-actions">
          <button className="btn btn-secondary" onClick={handleRefreshStatus} disabled={isWaBusy}>
            Cek Status
          </button>
          <button className="btn btn-primary" onClick={() => openQrForSession(activeSessionId)} disabled={isWaBusy || !hasSessions || !isWhatsAppIncluded}>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/><path d="M3.51 9a9 9 0 0 1 14.13-3.36L23 10"/><path d="M20.49 15a9 9 0 0 1-14.13 3.36L1 14"/></svg>
            {busyKey === "wa-session-auto" ? "Menyiapkan Session" : qrActionLabel}
          </button>
        </div>
      </div>

      <div className="wa-flow-strip">
        {[
          { label: "Pilih agent", done: Boolean(sessionForm?.aiAgentId || hasSessions) },
          { label: "Isi nama perangkat", done: Boolean(sessionForm?.label || hasSessions) },
          { label: "Konfirmasi di Meta", done: isConnected },
        ].map((step, index) => (
          <div key={step.label} className={`wa-flow-step ${step.done ? "done" : ""}`}>
            <span>{step.done ? "OK" : index + 1}</span>
            <strong>{step.label}</strong>
          </div>
        ))}
      </div>

      <div className="card channel-center-card" style={{ marginBottom: "24px" }}>
        <div className="card-header">
          <div>
            <div className="card-title">Pusat channel</div>
            <div className="card-subtitle">Hubungkan channel bisnis dari satu tempat. Semua pesan akan masuk ke Inbox dan memakai agent yang kamu pilih.</div>
          </div>
        </div>
        <div className="channel-center-grid">
          <div className="channel-center-item is-active">
            <div className="channel-center-icon channel-icon-wa">WA</div>
            <div className="channel-center-copy">
              <strong>WhatsApp</strong>
              <span>Channel utama untuk chat pelanggan.</span>
            </div>
            <span className="badge green">Tersedia</span>
          </div>
          <div className="channel-center-item">
            <div className="channel-center-icon channel-icon-ig">IG</div>
            <div className="channel-center-copy">
              <strong>Instagram</strong>
              <span>Balas DM dari Inbox Oneflow.</span>
            </div>
            <span className="badge gray">Segera hadir</span>
          </div>
          <div className="channel-center-item">
            <div className="channel-center-icon channel-icon-tt">TT</div>
            <div className="channel-center-copy">
              <strong>TikTok</strong>
              <span>Siapkan channel sosial berikutnya.</span>
            </div>
            <span className="badge gray">Segera hadir</span>
          </div>
        </div>
        <p className="text-sm text-muted channel-center-note">Untuk channel baru, Oneflow akan memandu login akun bisnis, izin akses, dan tes koneksi tanpa istilah teknis.</p>
      </div>

      {!hasAIAgents ? (
        <div className="wa-agent-empty">
          <div>
            <strong>Belum ada agent</strong>
            <p>Buat agent dulu di halaman Agents, lalu balik ke sini untuk tempel ke nomor WhatsApp.</p>
          </div>
          <button className="btn btn-primary" type="button" onClick={navigateToAgents}>Buat Agent</button>
        </div>
      ) : null}

      {!isWhatsAppIncluded ? (
        <div className="notice warning" style={{ marginBottom: "16px" }}>
          <div>
            <strong>WhatsApp tersedia setelah paket aktif</strong>
            <p>Aktifkan paket untuk menghubungkan WhatsApp. Sebelum itu, coba AI dulu di Coba AI.</p>
          </div>
          <button className="btn btn-primary" type="button" onClick={navigateToUpgrade}>Pilih Paket</button>
        </div>
      ) : isWhatsAppAtLimit ? (
        <div className="notice warning" style={{ marginBottom: "16px" }}>
          <div>
            <strong>Limit WhatsApp paket {planName} sudah penuh</strong>
            <p>Paket ini mendukung maksimal {maxWhatsAppSessions} WhatsApp. Ubah paket untuk menambah koneksi.</p>
          </div>
          <button className="btn btn-primary" type="button" onClick={navigateToUpgrade}>Ubah Paket</button>
        </div>
      ) : null}

      <div className="card" style={{ marginBottom: "24px" }}>
        <div className="card-header">
          <div>
            <div className="card-title">Koneksi WhatsApp</div>
            <div className="card-subtitle">WhatsApp Business resmi melalui Meta Coexistence. Aplikasi WhatsApp Business tetap dapat digunakan; tidak perlu menghapus akun atau mereset chat.</div>
          </div>
        </div>
        <form className="inline-form wa-device-form" onSubmit={createSession}>
          <input className="form-input" placeholder="Nama perangkat, contoh: CS Jakarta" value={sessionForm?.label || ""} onChange={(event) => setSessionForm?.({ ...(sessionForm || {}), label: event.target.value })} disabled={!canCreateWhatsApp} />
          <select className="form-select" value={sessionForm?.aiAgentId || ""} onChange={(event) => setSessionForm?.({ ...(sessionForm || {}), aiAgentId: event.target.value })} disabled={!canCreateWhatsApp}>
            <option value="">Pilih AI Agent</option>
            {aiAgents.filter((agent) => agent.isActive !== false).map((agent) => (
              <option key={agent.id} value={agent.id}>{agent.name}</option>
            ))}
          </select>
          <button className="btn btn-primary" type="submit" disabled={busyKey === "wa-session-create" || String(busyKey || "").startsWith("wa-meta-") || !canCreateWhatsApp} data-tour="connect-whatsapp">Hubungkan WhatsApp</button>
        </form>
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Session</th>
                <th>Status</th>
                <th>Koneksi</th>
                <th>Updated</th>
                <th>Aksi</th>
              </tr>
            </thead>
            <tbody>
              {sessions.map((session) => (
                <tr key={session.id} className={session.id === activeSessionId ? "is-selected-row" : ""}>
                  <td>
                    <button className="link-button" type="button" onClick={() => setActiveSessionId?.(session.id)}>
                      <strong>{session.label}</strong>
                    </button>
                    <div className="text-xs text-muted">{session.aiAgentName || "AI agent belum dipilih"}{session.isDefault ? " - Default session" : ""}</div>
                  </td>
                  <td><span className={`badge ${session.status === "connected" ? "green" : session.status === "qr_pending" ? "orange" : "gray"}`}>{toLabel(session.status)}</span></td>
                  <td>{session.provider === "meta_cloud" ? <span className="badge green">Resmi</span> : <span className="badge gray">Legacy</span>}</td>
                  <td className="text-muted text-sm">{formatDateTime(session.updatedAt)}</td>
                  <td>
                    <div style={{ display: "flex", gap: "6px", flexWrap: "wrap" }}>
                      <button className="btn btn-secondary btn-sm" type="button" onClick={() => renameSession?.(session.id, session.label)}>Rename</button>
                      {session.status !== "connected" ? (
                        <button className="btn btn-secondary btn-sm" type="button" onClick={() => openQrForSession(session.id)} disabled={isWaBusy || String(busyKey || "").startsWith("wa-meta-") || !isWhatsAppIncluded}>{session.provider === "meta_cloud" ? "Connect" : "QR"}</button>
                      ) : null}
                      <button className="btn btn-danger btn-sm" type="button" onClick={() => deleteSession?.(session.id)}>Delete</button>
                    </div>
                  </td>
                </tr>
              ))}
              {!sessions.length ? (
                <tr><td colSpan={5} style={{ textAlign: "center", padding: "24px", color: "var(--gray-500)" }}>Belum ada koneksi resmi. Isi nama perangkat dan pilih AI agent untuk melanjutkan ke Meta.</td></tr>
              ) : null}
            </tbody>
          </table>
        </div>
      </div>

      {showOfficialTemplateTools ? (
        <div className="card wa-meta-template-card" style={{ marginBottom: "24px" }}>
          <div className="card-header">
            <div>
              <div className="card-title">Template Resmi Meta</div>
              <div className="card-subtitle">Template milik WhatsApp Business Account yang dipilih.</div>
            </div>
            <button className="btn btn-secondary btn-sm" type="button" onClick={() => loadMetaTemplates?.(activeSession.id)} disabled={busyKey === "wa-meta-template-list"}>
              {busyKey === "wa-meta-template-list" ? "Memuat..." : "Refresh"}
            </button>
          </div>
          <div className="wa-meta-template-grid">
            <form className="wa-meta-template-form" onSubmit={createMetaTemplate}>
              <strong>Ajukan Template</strong>
              <input className="form-input" aria-label="Nama template" placeholder="nama_template" value={metaTemplateForm?.name || ""} onChange={(event) => setMetaTemplateForm?.({ ...(metaTemplateForm || {}), name: event.target.value })} />
              <div className="wa-meta-template-row">
                <select className="form-select" aria-label="Kategori template" value={metaTemplateForm?.category || "UTILITY"} onChange={(event) => setMetaTemplateForm?.({ ...(metaTemplateForm || {}), category: event.target.value })}>
                  <option value="UTILITY">Utility</option>
                  <option value="MARKETING">Marketing</option>
                </select>
                <input className="form-input" aria-label="Bahasa template" placeholder="id" value={metaTemplateForm?.language || ""} onChange={(event) => setMetaTemplateForm?.({ ...(metaTemplateForm || {}), language: event.target.value })} />
              </div>
              <textarea className="form-input wa-meta-template-body" aria-label="Isi template" value={metaTemplateForm?.body || ""} onChange={(event) => setMetaTemplateForm?.({ ...(metaTemplateForm || {}), body: event.target.value })} />
              <button className="btn btn-primary" type="submit" disabled={busyKey === "wa-meta-template-create"}>{busyKey === "wa-meta-template-create" ? "Mengirim..." : "Ajukan ke Meta"}</button>
            </form>
            <div className="wa-meta-template-list">
              <div className="wa-meta-template-list-header">
                <strong>Daftar Template</strong>
                <span className="badge gray">{metaTemplates.length}</span>
              </div>
              {metaTemplates.length ? metaTemplates.map((template) => (
                <div className="wa-meta-template-item" key={template.id || `${template.name}-${template.language}`}>
                  <div>
                    <strong>{template.name}</strong>
                    <span>{template.language || "-"} - {template.category || "-"}</span>
                  </div>
                  <span className={`badge ${template.status === "APPROVED" ? "green" : template.status === "REJECTED" ? "red" : "orange"}`}>{template.status || "PENDING"}</span>
                </div>
              )) : <p className="text-sm text-muted">Belum ada data template yang dimuat.</p>}
              <form className="wa-meta-template-send" onSubmit={sendMetaTemplate}>
                <strong>Kirim Template Uji</strong>
                <input className="form-input" aria-label="Nomor tujuan template" placeholder="62812..." value={metaTemplateSendForm?.phone || ""} onChange={(event) => setMetaTemplateSendForm?.({ ...(metaTemplateSendForm || {}), phone: event.target.value })} />
                <div className="wa-meta-template-row">
                  <input className="form-input" aria-label="Nama template kirim" placeholder="nama_template" value={metaTemplateSendForm?.name || ""} onChange={(event) => setMetaTemplateSendForm?.({ ...(metaTemplateSendForm || {}), name: event.target.value })} />
                  <input className="form-input" aria-label="Bahasa pengiriman" placeholder="id" value={metaTemplateSendForm?.language || ""} onChange={(event) => setMetaTemplateSendForm?.({ ...(metaTemplateSendForm || {}), language: event.target.value })} />
                </div>
                <button className="btn btn-secondary" type="submit" disabled={busyKey === "wa-meta-template-send"}>{busyKey === "wa-meta-template-send" ? "Mengirim..." : "Kirim Uji"}</button>
              </form>
            </div>
          </div>
        </div>
      ) : null}

      {!officialWhatsAppEnabled ? <div className="notice warning">Koneksi resmi belum tersedia. Hubungi admin Oneflow.</div> : null}
      <p className="text-sm text-muted">Notifikasi grup belum tersedia pada integrasi resmi ini. Tim tetap dapat menangani pelanggan melalui Inbox.</p>
    </>
  );
}
