import { useState } from "react";

const statusRank = { APPROVED: 0, PENDING: 1, REJECTED: 2 };

function templateStatus(template) {
  const status = String(template?.status || "PENDING").toUpperCase();
  if (status === "APPROVED") return { label: "Siap dipakai", badge: "green" };
  if (status === "REJECTED") return { label: "Ditolak", badge: "red" };
  return { label: "Sedang ditinjau", badge: "orange" };
}

export function WhatsAppTemplatesView({ activeSession, metaTemplates = [], metaTemplateForm, setMetaTemplateForm, metaTemplateSendForm, setMetaTemplateSendForm, loadMetaTemplates, createMetaTemplate, sendMetaTemplate, busyKey, navigateToChannels }) {
  const [composerOpen, setComposerOpen] = useState(false);
  const templates = [...metaTemplates].sort((left, right) => {
    const rank = (statusRank[String(left?.status || "PENDING").toUpperCase()] ?? 1) - (statusRank[String(right?.status || "PENDING").toUpperCase()] ?? 1);
    return rank || String(left?.name || "").localeCompare(String(right?.name || ""));
  });
  const approvedCount = templates.filter((item) => String(item.status).toUpperCase() === "APPROVED").length;
  const pendingCount = templates.filter((item) => !["APPROVED", "REJECTED"].includes(String(item.status).toUpperCase())).length;
  const rejectedCount = templates.filter((item) => String(item.status).toUpperCase() === "REJECTED").length;
  const selectedApprovedTemplate = templates.find((item) => String(item.status).toUpperCase() === "APPROVED" && item.name === metaTemplateSendForm?.name && (item.language || "id") === (metaTemplateSendForm?.language || "id"));

  function useTemplate(template) {
    setMetaTemplateSendForm?.({ ...(metaTemplateSendForm || {}), name: String(template.name || ""), language: String(template.language || "id") });
    window.setTimeout(() => document.getElementById("wa-template-test-send")?.scrollIntoView({ behavior: "smooth", block: "center" }), 0);
  }

  return <>
    <div className="page-title-row">
      <div><div className="page-title">Template WhatsApp</div><div className="page-desc">Buat dan gunakan template resmi yang tersinkron langsung dengan WhatsApp Manager.</div></div>
      <div className="page-actions">
        <button className="btn btn-secondary" type="button" onClick={navigateToChannels}>Kembali ke Channels</button>
        <button className="btn btn-secondary" type="button" onClick={() => loadMetaTemplates?.(activeSession?.id)} disabled={busyKey === "wa-meta-template-list"}>{busyKey === "wa-meta-template-list" ? "Menyinkronkan..." : "Sinkronkan"}</button>
        <button className="btn btn-primary" type="button" onClick={() => setComposerOpen((current) => !current)}>{composerOpen ? "Tutup form" : "Tambah template"}</button>
      </div>
    </div>

    <div className="wa-template-list-toolbar">
      <div><span className="badge green">{activeSession?.label || "WhatsApp"}</span><strong>List template</strong><small>Data langsung dari WhatsApp Manager</small></div>
      <div className="wa-template-list-counts"><span><b>{approvedCount}</b> siap dipakai</span><span><b>{pendingCount}</b> ditinjau</span>{rejectedCount ? <span><b>{rejectedCount}</b> ditolak</span> : null}</div>
    </div>

    <div className={`wa-template-workspace ${composerOpen ? "with-composer" : ""}`}>
      <section className="card wa-template-library">
        <div className="card-header"><div><div className="card-title">Template dari Meta</div><div className="card-subtitle">Template approved ditampilkan lebih dulu dan siap dipakai.</div></div><span className="badge gray">{templates.length} template</span></div>
        <div className="wa-template-status-list">{templates.length ? templates.map((template) => {
          const status = templateStatus(template);
          const approved = String(template.status).toUpperCase() === "APPROVED";
          const body = template.components?.find?.((component) => String(component?.type).toUpperCase() === "BODY")?.text;
          return <article className={`wa-meta-template-item ${approved ? "is-approved" : ""}`} key={template.id || `${template.name}-${template.language}`}>
            <div className="wa-template-item-copy"><div className="wa-template-item-heading"><strong>{template.name}</strong><span className={`badge ${status.badge}`}>{status.label}</span></div><span>{template.language || "-"} · {template.category || "-"}</span>{body ? <p>{body}</p> : null}{template.status === "REJECTED" && template.rejected_reason ? <span className="text-xs text-muted">Alasan Meta: {template.rejected_reason}</span> : null}</div>
            {approved ? <button className="btn btn-secondary btn-sm" type="button" onClick={() => useTemplate(template)}>Pakai template</button> : null}
          </article>;
        }) : <div className="wa-template-empty"><strong>Belum ada data template</strong><p>{busyKey === "wa-meta-template-list" ? "Sedang mengambil daftar terbaru dari Meta..." : "Sinkronkan untuk mengambil template dari WhatsApp Manager."}</p></div>}</div>
      </section>

      {composerOpen ? <form className="card wa-meta-template-form wa-template-editor" onSubmit={async (event) => { if (await createMetaTemplate?.(event)) setComposerOpen(false); }}>
        <div className="card-header"><div><div className="card-title">Buat template baru</div><div className="card-subtitle">Setelah diajukan, status review akan tampil di daftar Meta.</div></div></div>
        <label className="text-sm text-muted" htmlFor="meta-template-name">Nama template</label>
        <input id="meta-template-name" className="form-input" aria-label="Nama template" required pattern="[a-z0-9_]+" title="Gunakan huruf kecil, angka, dan underscore saja." placeholder="pesanan_dikonfirmasi" value={metaTemplateForm?.name || ""} onChange={(event) => setMetaTemplateForm?.({ ...(metaTemplateForm || {}), name: event.target.value.toLowerCase().replace(/[^a-z0-9_]/g, "_") })} />
        <span className="text-xs text-muted">Gunakan huruf kecil, angka, dan underscore.</span>
        <div className="wa-meta-template-row"><label className="text-sm text-muted">Kategori<select className="form-select" aria-label="Kategori template" value={metaTemplateForm?.category || "UTILITY"} onChange={(event) => setMetaTemplateForm?.({ ...(metaTemplateForm || {}), category: event.target.value })}><option value="UTILITY">Utility</option><option value="MARKETING">Marketing</option></select></label><label className="text-sm text-muted">Bahasa<select className="form-select" aria-label="Bahasa template" value={metaTemplateForm?.language || "id"} onChange={(event) => setMetaTemplateForm?.({ ...(metaTemplateForm || {}), language: event.target.value })}><option value="id">Indonesia</option><option value="en_US">English</option></select></label></div>
        <label className="text-sm text-muted" htmlFor="meta-template-body">Isi pesan</label>
        <textarea id="meta-template-body" className="form-input wa-meta-template-body" aria-label="Isi template" required maxLength={1024} placeholder="Halo {{1}}, pesanan kamu sudah dikonfirmasi." value={metaTemplateForm?.body || ""} onChange={(event) => setMetaTemplateForm?.({ ...(metaTemplateForm || {}), body: event.target.value })} />
        <div className="wa-template-form-help"><span>Gunakan {"{{1}}"}, {"{{2}}"}, dan seterusnya untuk data dinamis.</span><span>{String(metaTemplateForm?.body || "").length}/1024</span></div>
        <button className="btn btn-primary" type="submit" disabled={busyKey === "wa-meta-template-create"}>{busyKey === "wa-meta-template-create" ? "Mengajukan ke Meta..." : "Ajukan template"}</button>
      </form> : null}
    </div>

    <form id="wa-template-test-send" className="card wa-meta-template-send wa-template-send-panel" onSubmit={sendMetaTemplate}>
      <div className="card-header"><div><div className="card-title">Kirim template</div><div className="card-subtitle">Pilih template approved dari daftar, lalu masukkan nomor pelanggan.</div></div>{selectedApprovedTemplate ? <span className="badge green">Siap dikirim</span> : <span className="badge gray">Pilih template</span>}</div>
      <div className="wa-template-send-fields"><label className="text-sm text-muted">Nomor tujuan<input className="form-input" aria-label="Nomor tujuan template" required pattern="[0-9]{8,15}" placeholder="6281212022628" value={metaTemplateSendForm?.phone || ""} onChange={(event) => setMetaTemplateSendForm?.({ ...(metaTemplateSendForm || {}), phone: event.target.value.replace(/\D/g, "") })} /></label><label className="text-sm text-muted">Template terpilih<input className="form-input" aria-label="Nama template kirim" readOnly required value={metaTemplateSendForm?.name || ""} placeholder="Pilih dari daftar di atas" /></label><label className="text-sm text-muted">Bahasa<input className="form-input" aria-label="Bahasa pengiriman" readOnly value={metaTemplateSendForm?.language || "id"} /></label></div>
      <button className="btn btn-primary" type="submit" disabled={!selectedApprovedTemplate || busyKey === "wa-meta-template-send"}>{busyKey === "wa-meta-template-send" ? "Mengirim..." : "Kirim template"}</button>
    </form>
  </>;
}
