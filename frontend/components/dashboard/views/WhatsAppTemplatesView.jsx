export function WhatsAppTemplatesView({
  activeSession,
  metaTemplates = [],
  metaTemplateForm,
  setMetaTemplateForm,
  metaTemplateSendForm,
  setMetaTemplateSendForm,
  loadMetaTemplates,
  createMetaTemplate,
  sendMetaTemplate,
  busyKey,
  navigateToChannels,
}) {
  return (
    <>
      <div className="page-title-row">
        <div>
          <div className="page-title">Template WhatsApp</div>
          <div className="page-desc">Buat, pantau, dan uji template pesan resmi untuk pelanggan.</div>
        </div>
        <div className="page-actions">
          <button className="btn btn-secondary" type="button" onClick={navigateToChannels}>Kembali ke Channels</button>
          <button className="btn btn-primary" type="button" onClick={() => loadMetaTemplates?.(activeSession?.id)} disabled={busyKey === "wa-meta-template-list"}>
            {busyKey === "wa-meta-template-list" ? "Memuat..." : "Refresh template"}
          </button>
        </div>
      </div>
      <div className="wa-template-page-intro">
        <div><span className="badge green">WhatsApp resmi terhubung</span><h2>{activeSession?.label || "Koneksi WhatsApp"}</h2><p>Template dipakai saat chat customer sudah melewati 24 jam. Semua template tetap harus disetujui Meta sebelum dikirim.</p></div>
        <div className="wa-template-page-summary"><strong>{metaTemplates.length}</strong><span>template terdaftar</span></div>
      </div>
      <div className="wa-template-management-grid">
        <form className="card wa-meta-template-form wa-template-editor" onSubmit={createMetaTemplate}>
          <div className="card-header"><div><div className="card-title">Ajukan template baru</div><div className="card-subtitle">Isi sekali, lalu tunggu review dari Meta.</div></div><span className="badge gray">Langkah 1</span></div>
          <label className="text-sm text-muted" htmlFor="meta-template-name">Nama internal</label>
          <input id="meta-template-name" className="form-input" aria-label="Nama template" required pattern="[a-z0-9_]+" title="Gunakan huruf kecil, angka, dan underscore saja." placeholder="pesanan_dikonfirmasi" value={metaTemplateForm?.name || ""} onChange={(event) => setMetaTemplateForm?.({ ...(metaTemplateForm || {}), name: event.target.value })} />
          <span className="text-xs text-muted">Huruf kecil, angka, dan underscore saja.</span>
          <div className="wa-meta-template-row"><label className="text-sm text-muted">Kategori<select className="form-select" aria-label="Kategori template" value={metaTemplateForm?.category || "UTILITY"} onChange={(event) => setMetaTemplateForm?.({ ...(metaTemplateForm || {}), category: event.target.value })}><option value="UTILITY">Utility</option><option value="MARKETING">Marketing</option></select></label><label className="text-sm text-muted">Bahasa<select className="form-select" aria-label="Bahasa template" value={metaTemplateForm?.language || "id"} onChange={(event) => setMetaTemplateForm?.({ ...(metaTemplateForm || {}), language: event.target.value })}><option value="id">Indonesia</option><option value="en_US">English</option></select></label></div>
          <label className="text-sm text-muted" htmlFor="meta-template-body">Isi pesan</label>
          <textarea id="meta-template-body" className="form-input wa-meta-template-body" aria-label="Isi template" required maxLength={1024} placeholder="Halo {{1}}, pesanan kamu sudah dikonfirmasi." value={metaTemplateForm?.body || ""} onChange={(event) => setMetaTemplateForm?.({ ...(metaTemplateForm || {}), body: event.target.value })} />
          <span className="text-xs text-muted">Gunakan {"{{1}}"}, {"{{2}}"}, dan seterusnya untuk nilai dinamis.</span>
          <button className="btn btn-primary" type="submit" disabled={busyKey === "wa-meta-template-create"}>{busyKey === "wa-meta-template-create" ? "Mengirim ke Meta..." : "Ajukan ke Meta"}</button>
        </form>
        <div className="card wa-template-status-panel">
          <div className="card-header"><div><div className="card-title">Status template</div><div className="card-subtitle">Pantau hasil review dari Meta.</div></div><span className="badge gray">Langkah 2</span></div>
          <div className="wa-template-status-list">{metaTemplates.length ? metaTemplates.map((template) => (
            <div className="wa-meta-template-item" key={template.id || `${template.name}-${template.language}`}><div><strong>{template.name}</strong><span>{template.language || "-"} - {template.category || "-"}</span>{template.status === "REJECTED" && template.rejected_reason ? <span className="text-xs text-muted">Alasan Meta: {template.rejected_reason}</span> : null}</div><span className={`badge ${template.status === "APPROVED" ? "green" : template.status === "REJECTED" ? "red" : "orange"}`}>{template.status || "PENDING"}</span></div>
          )) : <p className="text-sm text-muted">Belum ada template. Klik Refresh template untuk memuat data dari Meta.</p>}</div>
        </div>
        <form className="card wa-meta-template-send wa-template-send-panel" onSubmit={sendMetaTemplate}>
          <div className="card-header"><div><div className="card-title">Kirim template uji</div><div className="card-subtitle">Hanya template dengan status Approved yang bisa dikirim.</div></div><span className="badge gray">Langkah 3</span></div>
          <input className="form-input" aria-label="Nomor tujuan template" required pattern="[0-9]{8,15}" placeholder="6281212022628" value={metaTemplateSendForm?.phone || ""} onChange={(event) => setMetaTemplateSendForm?.({ ...(metaTemplateSendForm || {}), phone: event.target.value })} />
          <div className="wa-meta-template-row"><input className="form-input" aria-label="Nama template kirim" required pattern="[a-z0-9_]+" placeholder="nama_template" value={metaTemplateSendForm?.name || ""} onChange={(event) => setMetaTemplateSendForm?.({ ...(metaTemplateSendForm || {}), name: event.target.value })} /><select className="form-select" aria-label="Bahasa pengiriman" value={metaTemplateSendForm?.language || "id"} onChange={(event) => setMetaTemplateSendForm?.({ ...(metaTemplateSendForm || {}), language: event.target.value })}><option value="id">Indonesia</option><option value="en_US">English</option></select></div>
          <button className="btn btn-secondary" type="submit" disabled={busyKey === "wa-meta-template-send"}>{busyKey === "wa-meta-template-send" ? "Mengirim..." : "Kirim template uji"}</button>
        </form>
      </div>
    </>
  );
}
