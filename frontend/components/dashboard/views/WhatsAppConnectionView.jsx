import { useState } from "react";
import { formatDateTime, toLabel } from "../../../lib/dashboard-core";

export function WhatsAppConnectionView({
  waStatus,
  sessions = [],
  instagramSessions = [],
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
  connectInstagram,
  validateInstagram,
  disconnectInstagram,
  canManageInstagram = false,
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
  navigateToBusinessTools,
  navigateToTemplates,
  templateToolsOnSeparatePage = false
}) {
  const [instagramAgentId, setInstagramAgentId] = useState("");
  const [selectedChannel, setSelectedChannel] = useState("whatsapp");
  const hasSessions = sessions.length > 0;
  const hasAIAgents = aiAgents.some((agent) => agent.isActive !== false);
  const isWaBusy = String(busyKey || "").startsWith("wa-");
  const activeSession = sessions.find((item) => item.id === activeSessionId) || sessions[0];
  const isConnected = activeSession?.status === "connected";
  const showOfficialTemplateTools = Boolean(canManageOfficialTemplates && isConnected && !templateToolsOnSeparatePage);
  const qrActionLabel = "Sambungkan Resmi";
  const maxWhatsAppSessions = Number(billingPlan?.maxWhatsAppSessions ?? billingPlan?.max_whatsapp_sessions ?? 0);
  const planName = billingPlan?.planName || billingPlan?.name || "Trial";
  const isWhatsAppIncluded = maxWhatsAppSessions > 0;
  const isWhatsAppAtLimit = isWhatsAppIncluded && sessions.length >= maxWhatsAppSessions;
  const canCreateWhatsApp = officialWhatsAppEnabled && isWhatsAppIncluded && !isWhatsAppAtLimit && hasAIAgents;
  const activeInstagramAgents = aiAgents.filter((agent) => agent.isActive !== false);
  const selectedInstagramAgentId = instagramAgentId || activeInstagramAgents[0]?.id || "";
  const isInstagramBusy = String(busyKey || "").startsWith("instagram-");
  const handleRefreshStatus = () => refreshAll?.();
  const openQrForSession = (sessionId) => {
    if (sessionId && officialWhatsAppEnabled) connectOfficialWhatsApp?.(sessionId);
  };

  return (
    <>
      <div className="page-title-row">
        <div>
          <div className="page-title">Channels</div>
          <div className="page-desc">Pilih tempat pelanggan menghubungi bisnismu. Oneflow akan memandu sisanya.</div>
        </div>
        <div className="page-actions">
          <button className="btn btn-secondary" onClick={handleRefreshStatus} disabled={isWaBusy}>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/><path d="M3.51 9a9 9 0 0 1 14.13-3.36L23 10"/><path d="M20.49 15a9 9 0 0 1-14.13 3.36L1 14"/></svg>
            Perbarui status
          </button>
        </div>
      </div>

      <div className="channel-picker" aria-label="Pilih channel bisnis">
        <div className="card-header">
          <div>
            <div className="card-title">Mau menghubungkan channel apa?</div>
            <div className="card-subtitle">Pilih satu. Kamu bisa kembali dan menambahkan channel lain kapan saja.</div>
          </div>
        </div>
        <div className="channel-center-grid">
          <button type="button" className={`channel-center-item ${selectedChannel === "whatsapp" ? "is-selected" : ""}`} onClick={() => setSelectedChannel("whatsapp")} aria-pressed={selectedChannel === "whatsapp"}>
            <div className="channel-center-icon channel-icon-wa">WA</div>
            <div className="channel-center-copy">
              <strong>WhatsApp</strong>
              <span>Terima dan balas chat WhatsApp Business.</span>
            </div>
            <span className={`badge ${sessions.some((item) => item.status === "connected") ? "green" : "blue"}`}>{sessions.some((item) => item.status === "connected") ? "Terhubung" : "Hubungkan"}</span>
          </button>
          <button type="button" className={`channel-center-item ${selectedChannel === "instagram" ? "is-selected" : ""}`} onClick={() => setSelectedChannel("instagram")} aria-pressed={selectedChannel === "instagram"}>
            <div className="channel-center-icon channel-icon-ig">IG</div>
            <div className="channel-center-copy">
              <strong>Instagram</strong>
              <span>Terima dan balas DM akun profesional.</span>
            </div>
            <span className={`badge ${instagramSessions.length ? "green" : "blue"}`}>{instagramSessions.length ? "Terhubung" : "Hubungkan"}</span>
          </button>
          <div className="channel-center-item is-disabled" aria-disabled="true">
            <div className="channel-center-icon channel-icon-tt">TT</div>
            <div className="channel-center-copy">
              <strong>TikTok</strong>
              <span>Pesan TikTok Business.</span>
            </div>
            <span className="badge gray">Segera hadir</span>
          </div>
        </div>
      </div>

      {selectedChannel === "instagram" ? <div className="card channel-setup-card" style={{ marginBottom: "24px" }}>
        <div className="card-header">
          <div>
            <div className="channel-setup-eyebrow">Instagram</div>
            <div className="card-title">{instagramSessions.length ? "Instagram sudah terhubung" : "Hubungkan Instagram"}</div>
            <div className="card-subtitle">{instagramSessions.length ? "DM pelanggan akan masuk ke Inbox Oneflow." : "Pilih AI yang akan menjawab, lalu login ke Instagram. Tidak perlu menyalin token atau mengatur webhook."}</div>
          </div>
        </div>
        {canManageInstagram && !instagramSessions.length ? (
          <div className="channel-connect-form">
            <label className="form-group">
              <span className="form-label">AI yang membalas pelanggan</span>
              <select className="form-select" value={selectedInstagramAgentId} onChange={(event) => setInstagramAgentId(event.target.value)} disabled={!hasAIAgents || isInstagramBusy} aria-label="AI agent untuk Instagram">
                <option value="">Pilih AI Agent</option>
                {activeInstagramAgents.map((agent) => <option key={agent.id} value={agent.id}>{agent.name}</option>)}
              </select>
            </label>
            <button className="btn btn-primary channel-connect-action" type="button" onClick={() => connectInstagram?.(selectedInstagramAgentId)} disabled={!selectedInstagramAgentId || isInstagramBusy}>
              {busyKey === "instagram-connect" ? "Membuka Instagram..." : "Lanjutkan dengan Instagram"}
            </button>
            <p className="channel-connect-help">Setelah login, izinkan akses pesan. Kamu akan kembali otomatis ke halaman ini.</p>
          </div>
        ) : !canManageInstagram && !instagramSessions.length ? <p className="text-sm text-muted">Minta admin organisasi untuk menghubungkan akun Instagram.</p> : null}
        {instagramSessions.map((session) => {
          const agent = aiAgents.find((item) => item.id === session.aiAgentId);
          return <div className="channel-connection-summary" key={session.id}>
            <div className="channel-account-mark channel-icon-ig">IG</div>
            <div className="channel-connection-main"><strong>@{session.username || session.instagramUserId}</strong><span>Dijawab oleh {agent?.name || "AI belum dipilih"}</span></div>
            <div className="channel-connection-state"><span className={`badge ${session.webhookConnected ? "green" : "orange"}`}>{session.webhookConnected ? "Siap menerima DM" : "Menunggu pesan pertama"}</span><span>Terhubung {formatDateTime(session.connectedAt)}</span></div>
            {canManageInstagram ? <div className="row-actions"><button className="btn btn-secondary btn-sm" type="button" onClick={() => validateInstagram?.(session.id)} disabled={isInstagramBusy}>Cek koneksi</button><button className="btn btn-danger btn-sm" type="button" onClick={() => disconnectInstagram?.(session.id)} disabled={isInstagramBusy}>Putuskan</button></div> : null}
          </div>;
        })}
      </div> : null}

      {!hasAIAgents ? (
        <div className="wa-agent-empty">
          <div>
            <strong>Belum ada agent</strong>
            <p>Buat agent dulu di halaman Agents, lalu balik ke sini untuk tempel ke nomor WhatsApp.</p>
          </div>
          <button className="btn btn-primary" type="button" onClick={navigateToAgents}>Buat Agent</button>
        </div>
      ) : null}

      {selectedChannel === "whatsapp" ? <>
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

      <div className="card channel-setup-card" style={{ marginBottom: "24px" }}>
        <div className="card-header">
          <div>
            <div className="channel-setup-eyebrow">WhatsApp</div>
            <div className="card-title">{sessions.some((item) => item.status === "connected") ? "WhatsApp sudah terhubung" : "Hubungkan WhatsApp Business"}</div>
            <div className="card-subtitle">{sessions.some((item) => item.status === "connected") ? "Chat pelanggan akan masuk ke Inbox Oneflow. Aplikasi WhatsApp Business tetap bisa digunakan." : "Gunakan koneksi resmi Meta. Riwayat chat dan aplikasi WhatsApp Business tidak perlu dihapus."}</div>
          </div>
          {canManageOfficialTemplates && isConnected ? <button className="btn btn-secondary btn-sm" type="button" onClick={navigateToTemplates}>Template pesan</button> : null}
        </div>
        {!isWhatsAppAtLimit ? <form className="channel-connect-form channel-connect-form-wa" onSubmit={createSession}>
          <label className="form-group"><span className="form-label">Nama koneksi</span><input className="form-input" placeholder="Contoh: WhatsApp Toko Jakarta" value={sessionForm?.label || ""} onChange={(event) => setSessionForm?.({ ...(sessionForm || {}), label: event.target.value })} disabled={!canCreateWhatsApp} /></label>
          <label className="form-group"><span className="form-label">AI yang membalas pelanggan</span><select className="form-select" value={sessionForm?.aiAgentId || ""} onChange={(event) => setSessionForm?.({ ...(sessionForm || {}), aiAgentId: event.target.value })} disabled={!canCreateWhatsApp}>
              <option value="">Pilih AI Agent</option>
              {aiAgents.filter((agent) => agent.isActive !== false).map((agent) => <option key={agent.id} value={agent.id}>{agent.name}</option>)}
            </select></label>
          <button className="btn btn-primary channel-connect-action" type="submit" disabled={busyKey === "wa-session-create" || String(busyKey || "").startsWith("wa-meta-") || !canCreateWhatsApp} data-tour="connect-whatsapp">{busyKey === "wa-session-create" ? "Menyiapkan..." : "Lanjutkan dengan Meta"}</button>
          <p className="channel-connect-help">Meta akan meminta konfirmasi akun WhatsApp Business. Ikuti petunjuk sampai selesai.</p>
        </form> : null}
        {sessions.length ? <details className="channel-advanced" open={sessions.length === 1 && !isConnected}>
          <summary>Kelola koneksi WhatsApp ({sessions.length})</summary>
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
            </tbody>
          </table>
          </div>
        </details> : null}
      </div>

      {showOfficialTemplateTools ? (
        <div className="card wa-meta-template-card" style={{ marginBottom: "24px" }}>
          <div className="card-header">
            <div>
              <div className="card-title">Template Resmi Meta</div>
              <div className="card-subtitle">Buat template untuk pesan di luar jendela chat 24 jam. Meta tetap meninjau setiap pengajuan.</div>
            </div>
            <button className="btn btn-secondary btn-sm" type="button" onClick={() => loadMetaTemplates?.(activeSession.id)} disabled={busyKey === "wa-meta-template-list"}>
              {busyKey === "wa-meta-template-list" ? "Memuat..." : "Refresh"}
            </button>
          </div>
          <div className="wa-meta-template-grid">
            <form className="wa-meta-template-form" onSubmit={createMetaTemplate}>
              <strong>1. Ajukan template baru</strong>
              <label className="text-sm text-muted" htmlFor="meta-template-name">Nama internal</label>
              <input id="meta-template-name" className="form-input" aria-label="Nama template" required pattern="[a-z0-9_]+" title="Gunakan huruf kecil, angka, dan underscore saja." placeholder="pesanan_dikonfirmasi" value={metaTemplateForm?.name || ""} onChange={(event) => setMetaTemplateForm?.({ ...(metaTemplateForm || {}), name: event.target.value })} />
              <span className="text-xs text-muted">Gunakan huruf kecil, angka, dan underscore.</span>
              <div className="wa-meta-template-row">
                <select className="form-select" aria-label="Kategori template" value={metaTemplateForm?.category || "UTILITY"} onChange={(event) => setMetaTemplateForm?.({ ...(metaTemplateForm || {}), category: event.target.value })}>
                  <option value="UTILITY">Utility</option>
                  <option value="MARKETING">Marketing</option>
                </select>
                <select className="form-select" aria-label="Bahasa template" value={metaTemplateForm?.language || "id"} onChange={(event) => setMetaTemplateForm?.({ ...(metaTemplateForm || {}), language: event.target.value })}><option value="id">Indonesia (id)</option><option value="en_US">English (en_US)</option></select>
              </div>
              <label className="text-sm text-muted" htmlFor="meta-template-body">Isi pesan</label>
              <textarea id="meta-template-body" className="form-input wa-meta-template-body" aria-label="Isi template" required maxLength={1024} placeholder="Halo {{1}}, pesanan kamu sudah dikonfirmasi." value={metaTemplateForm?.body || ""} onChange={(event) => setMetaTemplateForm?.({ ...(metaTemplateForm || {}), body: event.target.value })} />
              <span className="text-xs text-muted">Maksimal 1.024 karakter. Gunakan {"{{1}}"}, {"{{2}}"}, dan seterusnya untuk nilai dinamis.</span>
              <button className="btn btn-primary" type="submit" disabled={busyKey === "wa-meta-template-create"}>{busyKey === "wa-meta-template-create" ? "Mengirim..." : "Ajukan ke Meta"}</button>
            </form>
            <div className="wa-meta-template-list">
              <div className="wa-meta-template-list-header">
                <strong>2. Pantau hasil review</strong>
                <span className="badge gray">{metaTemplates.length}</span>
              </div>
              {metaTemplates.length ? metaTemplates.map((template) => (
                <div className="wa-meta-template-item" key={template.id || `${template.name}-${template.language}`}>
                  <div>
                    <strong>{template.name}</strong>
                    <span>{template.language || "-"} - {template.category || "-"}</span>
                    {template.status === "REJECTED" && template.rejected_reason ? <span className="text-xs text-muted">Alasan Meta: {template.rejected_reason}</span> : null}
                  </div>
                  <span className={`badge ${template.status === "APPROVED" ? "green" : template.status === "REJECTED" ? "red" : "orange"}`}>{template.status || "PENDING"}</span>
                </div>
              )) : <p className="text-sm text-muted">Belum ada template yang dimuat. Klik Refresh untuk mengecek template terbaru dari Meta.</p>}
              <form className="wa-meta-template-send" onSubmit={sendMetaTemplate}>
                <strong>3. Kirim uji setelah Approved</strong>
                <span className="text-xs text-muted">Template hanya dapat dikirim jika statusnya Approved. Nomor tanpa tanda + atau spasi.</span>
                <input className="form-input" aria-label="Nomor tujuan template" required pattern="[0-9]{8,15}" placeholder="6281212022628" value={metaTemplateSendForm?.phone || ""} onChange={(event) => setMetaTemplateSendForm?.({ ...(metaTemplateSendForm || {}), phone: event.target.value })} />
                <div className="wa-meta-template-row">
                  <input className="form-input" aria-label="Nama template kirim" placeholder="nama_template" value={metaTemplateSendForm?.name || ""} onChange={(event) => setMetaTemplateSendForm?.({ ...(metaTemplateSendForm || {}), name: event.target.value })} />
                  <select className="form-select" aria-label="Bahasa pengiriman" value={metaTemplateSendForm?.language || "id"} onChange={(event) => setMetaTemplateSendForm?.({ ...(metaTemplateSendForm || {}), language: event.target.value })}><option value="id">Indonesia (id)</option><option value="en_US">English (en_US)</option></select>
                </div>
                <button className="btn btn-secondary" type="submit" disabled={busyKey === "wa-meta-template-send"}>{busyKey === "wa-meta-template-send" ? "Mengirim..." : "Kirim Uji"}</button>
              </form>
            </div>
          </div>
        </div>
      ) : null}

      {!officialWhatsAppEnabled ? <div className="notice warning">Koneksi resmi belum tersedia. Hubungi admin Oneflow.</div> : null}
      <p className="text-sm text-muted">Notifikasi grup belum tersedia pada integrasi resmi ini. Tim tetap dapat menangani pelanggan melalui Inbox.</p>
      </> : null}
    </>
  );
}
