import { createPortal } from "react-dom";
import { Icons, formatDateTime, truncateText } from "../../../lib/dashboard-core";

const emptyPositionForm = {
  id: "",
  aiAgentId: "",
  title: "",
  location: "",
  workType: "",
  shortDescription: "",
  applyLink: "",
  status: "published",
  isActive: true,
};

function documentTypeLabel(document) {
  if (document?.mediaMime?.includes("pdf")) return "PDF";
  if (document?.knowledgeType === "file") return "Dokumen";
  if (document?.knowledgeType === "faq") return "FAQ";
  return document?.knowledgeType ? document.knowledgeType : "Teks";
}

function isBusinessToolInstalled(tool) {
  return Boolean(tool?.installed ?? tool?.enabled);
}

function hasBusinessCatalogTools(businessTools = []) {
  return businessTools.some((tool) => ["commerce", "booking"].includes(tool?.key) && isBusinessToolInstalled(tool));
}

function businessKnowledgeRuleText(businessTools = []) {
  const commerceInstalled = businessTools.some((tool) => tool?.key === "commerce" && isBusinessToolInstalled(tool));
  const bookingInstalled = businessTools.some((tool) => tool?.key === "booking" && isBusinessToolInstalled(tool));
  if (commerceInstalled || bookingInstalled) {
    return "Plugin bisnis yang aktif jadi sumber data operasional. Pakai Knowledge Produk/Layanan untuk benefit, syarat, FAQ, atau link; harga, stok, order, slot, dan booking tetap dari Alat Bisnis.";
  }
  return "Kalau nanti plugin Produk & Stok atau Booking dipakai, pindahkan harga, stok, order, slot, dan jadwal ke Alat Bisnis supaya AI tidak membaca dua sumber yang berbeda.";
}

function documentEditText(document) {
  return document?.rawText || document?.content || document?.text || document?.chunkPreview || document?.description || "";
}

function documentStatus(document) {
  if (document?.status === "draft") return { tone: "gray", label: "Draft" };
  if (document?.embeddingStatus === "failed") return { tone: "gray", label: "Perlu diproses ulang" };
  if (document?.embeddingStatus === "pending") return { tone: "orange", label: "Sedang diproses" };
  if (document?.embeddingStatus === "partial") return { tone: "orange", label: "Sebagian siap" };
  if (document?.embeddingStatus === "not_chunked") return { tone: "orange", label: "Menunggu proses" };
  return { tone: "green", label: document?.status === "published" ? "Aktif" : document?.status || "Aktif" };
}

function publishedCount(items, predicate) {
  return items.filter(predicate).length;
}

function KnowledgeStat({ value, label, color }) {
  return (
    <div className="card knowledge-stat">
      <div className="knowledge-stat-value" style={{ color }}>{value}</div>
      <div className="text-sm text-muted">{label}</div>
    </div>
  );
}

function KnowledgeHealthRow({ label, value, color }) {
  return (
    <div className="flex-between text-sm">
      <span className="text-muted">{label}</span>
      <strong className="text-bold" style={{ color }}>{value}</strong>
    </div>
  );
}

function CoverageBar({ label, value, color }) {
  return (
    <div className="knowledge-coverage-row">
      <div className="flex-between mb-4">
        <span className="text-sm">{label}</span>
        <span className="text-sm text-bold">{value}%</span>
      </div>
      <div className="knowledge-coverage-track">
        <div className="knowledge-coverage-fill" style={{ width: `${value}%`, background: color }} />
      </div>
    </div>
  );
}

const knowledgeStarterTemplates = [
  {
    key: "faq-hours",
    kind: "faq",
    title: "Jam operasional",
    description: "Untuk pertanyaan kapan bisnis buka, libur, atau bisa dihubungi.",
    question: "Jam operasional [ubah bagian ini: nama bisnis] kapan?",
    answer: [
      "[ubah bagian ini: nama bisnis] buka pada [ubah bagian ini: hari dan jam operasional].",
      "",
      "Untuk hari libur atau perubahan jadwal, tim kami akan menginformasikan lewat [ubah bagian ini: WhatsApp/Instagram/website].",
      "Kalau customer butuh bantuan di luar jam operasional, jawab bahwa tim akan membalas saat jam kerja berikutnya.",
    ].join("\n"),
  },
  {
    key: "faq-price",
    kind: "faq",
    title: "Harga / paket",
    description: "Untuk menjelaskan kisaran harga tanpa membuat janji yang keliru.",
    question: "Berapa harga atau paket yang tersedia?",
    answer: [
      "Harga/paket yang tersedia:",
      "- [ubah bagian ini: nama paket/produk 1] - [ubah bagian ini: harga atau keterangan]",
      "- [ubah bagian ini: nama paket/produk 2] - [ubah bagian ini: harga atau keterangan]",
      "",
      "Catatan penting:",
      "- Harga bisa berubah jika ada [ubah bagian ini: custom request/ongkir/tambahan layanan].",
      "- Untuk harga final, tim akan cek detail kebutuhan customer terlebih dahulu.",
    ].join("\n"),
  },
  {
    key: "faq-register",
    kind: "faq",
    title: "Cara daftar / order",
    description: "Untuk pendaftaran, order, booking, atau langkah awal customer.",
    question: "Bagaimana cara daftar atau order?",
    answer: [
      "Cara daftar/order:",
      "1. Pilih [ubah bagian ini: produk/layanan/program/jadwal].",
      "2. Kirim data: nama, nomor WhatsApp, dan [ubah bagian ini: data wajib lain].",
      "3. Tim akan cek ketersediaan dan mengonfirmasi langkah berikutnya.",
      "",
      "Kalau customer sudah siap, kumpulkan data wajib dulu lalu arahkan ke admin bila butuh konfirmasi.",
    ].join("\n"),
  },
  {
    key: "doc-business-profile",
    kind: "document",
    title: "Profil bisnis & cara kerja",
    description: "Dokumen dasar agar AI tahu bisnis ini siapa dan cara melayani customer.",
    documentTitle: "Profil bisnis dan cara kerja",
    rawText: [
      "PROFIL BISNIS",
      "Nama bisnis: [ubah bagian ini: nama bisnis]",
      "Jenis bisnis: [ubah bagian ini: jenis bisnis]",
      "Lokasi/area layanan: [ubah bagian ini: lokasi atau area layanan]",
      "Customer utama: [ubah bagian ini: siapa target customer]",
      "",
      "LAYANAN / PRODUK UTAMA",
      "- [ubah bagian ini: produk/layanan utama 1]",
      "- [ubah bagian ini: produk/layanan utama 2]",
      "- [ubah bagian ini: produk/layanan utama 3]",
      "",
      "CARA MELAYANI CUSTOMER",
      "- Sapaan yang diinginkan: [ubah bagian ini: formal/santai/ramah]",
      "- Data yang perlu dikumpulkan: [ubah bagian ini: nama, nomor, kebutuhan, jadwal, alamat, dll]",
      "- Hal yang harus dikonfirmasi admin: [ubah bagian ini: harga final, slot, stok, refund, custom request]",
      "",
      "BATASAN",
      "- AI tidak boleh menjanjikan [ubah bagian ini: stok/jadwal/refund/hasil/harga final] sebelum dikonfirmasi.",
      "- AI harus panggil admin jika [ubah bagian ini: kondisi khusus].",
    ].join("\n"),
  },
  {
    key: "doc-policy",
    kind: "document",
    title: "Kebijakan layanan",
    description: "Untuk refund, reschedule, garansi, pembayaran, dan batasan layanan.",
    documentTitle: "Kebijakan layanan customer",
    rawText: [
      "KEBIJAKAN LAYANAN",
      "",
      "Pembayaran:",
      "- Metode pembayaran: [ubah bagian ini: transfer/QRIS/cash/payment link]",
      "- Bukti pembayaran dikirim ke: [ubah bagian ini: WhatsApp/admin/link]",
      "",
      "Refund / pembatalan:",
      "- Kebijakan refund: [ubah bagian ini: boleh/tidak/syarat refund]",
      "- Batas waktu pembatalan: [ubah bagian ini: contoh H-1, 24 jam, tidak bisa dibatalkan]",
      "",
      "Reschedule / perubahan:",
      "- Reschedule bisa dilakukan jika [ubah bagian ini: syarat reschedule].",
      "- Perubahan data/order/jadwal harus dikonfirmasi admin.",
      "",
      "Komplain:",
      "- Customer bisa menyampaikan komplain dengan menyebutkan [ubah bagian ini: nomor order/jadwal/nama layanan/bukti].",
      "- AI harus meneruskan komplain ke admin dan tidak menjanjikan hasil sebelum dicek.",
    ].join("\n"),
  },
  {
    key: "position-service",
    kind: "position",
    title: "Info produk/layanan",
    description: "Untuk bisnis tanpa plugin Produk/Booking. Jangan isi stok atau jadwal real-time.",
    positionTitle: "[ubah bagian ini: nama produk/layanan]",
    location: "[ubah bagian ini: kategori/lokasi]",
    workType: "Layanan",
    shortDescription: [
      "Ringkasan: [ubah bagian ini: jelaskan produk/layanan dalam 1-2 kalimat].",
      "Cocok untuk: [ubah bagian ini: target customer].",
      "Benefit utama: [ubah bagian ini: benefit 1, benefit 2, benefit 3].",
      "Syarat/catatan: [ubah bagian ini: syarat yang perlu diketahui customer].",
      "Jangan jawab stok, slot, atau jadwal real-time dari info ini.",
    ].join("\n"),
    applyLink: "",
  },
];

function InlineSpinner() {
  return <span className="inline-spinner" aria-hidden="true" />;
}

function ModalShell({ title, onClose, children, footer }) {
  const modal = (
    <div className="modal-backdrop" role="presentation" onMouseDown={(event) => {
      if (event.target === event.currentTarget) onClose?.();
    }}>
      <div className="modal-card knowledge-modal-card" role="dialog" aria-modal="true" aria-labelledby="knowledge-modal-title">
        <div className="modal-header">
          <div className="modal-title" id="knowledge-modal-title">{title}</div>
          <button type="button" className="icon-btn" onClick={onClose}>{Icons.close}</button>
        </div>
        <div className="modal-body">
          {children}
        </div>
        <div className="modal-footer">
          {footer}
        </div>
      </div>
    </div>
  );
  return typeof document !== "undefined" ? createPortal(modal, document.body) : modal;
}

export function KnowledgeView({
  knowledgeTab,
  setKnowledgeTab,
  faqs,
  documents,
  positions,
  businessTools = [],
  aiAgents = [],
  activeAIAgentId,
  setActiveAIAgentId,
  faqForm,
  setFaqForm,
  documentForm,
  setDocumentForm,
  documentSourceMode,
  setDocumentSourceMode,
  positionForm,
  setPositionForm,
  uploadTitle,
  setUploadTitle,
  setUploadFile,
  canManageKnowledge,
  submitFaq,
  submitDocument,
  submitPosition,
  deleteFaq,
  deleteDocument,
  reindexDocument,
  deletePosition,
  refreshVectorIndex,
  uploadKnowledgeDocument,
  busyKey,
  embedded = false,
}) {
  const readOnly = canManageKnowledge === false;
  const hasSelectedAgent = Boolean(activeAIAgentId);
  const publishedFaqs = publishedCount(faqs, (item) => item.status !== "draft");
  const publishedDocs = publishedCount(documents, (item) => item.status !== "draft" && item.embeddingStatus !== "failed");
  const activePositions = publishedCount(positions, (item) => item.isActive !== false && item.status !== "inactive");
  const usesBusinessCatalogTools = hasBusinessCatalogTools(businessTools);
  const visibleKnowledgeTabs = [
    { id: "faqs", label: `FAQ (${faqs.length})` },
    { id: "documents", label: `Dokumen (${documents.length})` },
    ...(!usesBusinessCatalogTools ? [{ id: "positions", label: `Info Produk/Layanan (${positions.length})` }] : []),
  ];
  const currentKnowledgeTab = visibleKnowledgeTabs.some((tab) => tab.id === knowledgeTab) ? knowledgeTab : "faqs";
  const documentMode = documentForm?.mode || documentSourceMode || (documentForm?.id ? "text" : "upload");
  const documentIsEdit = Boolean(documentForm?.id);
  const documentIsUpload = Boolean(documentForm) && documentMode === "upload" && !documentIsEdit;
  const documentIsText = Boolean(documentForm) && (documentMode === "text" || documentIsEdit);
  const isDocumentBusy = busyKey === "document" || busyKey === "document-upload" || String(busyKey || "").startsWith("document-reindex-");
  const inlineEditorOpen = embedded && Boolean(faqForm || documentForm || (!usesBusinessCatalogTools && positionForm));
  const businessRuleText = businessKnowledgeRuleText(businessTools);
  const availableKnowledgeTemplates = knowledgeStarterTemplates.filter((template) => template.kind !== "position" || !usesBusinessCatalogTools);

  const closeFaqForm = () => setFaqForm?.(null);
  const closeDocumentForm = () => {
    setDocumentForm?.(null);
    setDocumentSourceMode?.("upload");
    setUploadTitle?.("");
    setUploadFile?.(null);
  };
  const closePositionForm = () => setPositionForm?.(null);

  const openFaqCreate = () => {
    setDocumentForm?.(null);
    setPositionForm?.(null);
    setFaqForm?.({ id: "", question: "", answer: "", status: "published" });
  };

  const openDocumentUpload = () => {
    setFaqForm?.(null);
    setPositionForm?.(null);
    setDocumentSourceMode?.("upload");
    setUploadTitle?.("");
    setUploadFile?.(null);
    setDocumentForm?.({ id: "", title: "", file: null, mode: "upload" });
  };

  const openDocumentText = () => {
    setFaqForm?.(null);
    setPositionForm?.(null);
    setDocumentSourceMode?.("text");
    setUploadTitle?.("");
    setUploadFile?.(null);
    setDocumentForm?.({
      id: "",
      title: "",
      rawText: "",
      knowledgeType: "text",
      status: "published",
      mode: "text",
    });
  };

  const openDocumentEdit = (document) => {
    setFaqForm?.(null);
    setPositionForm?.(null);
    setDocumentSourceMode?.("text");
    setUploadFile?.(null);
    setDocumentForm?.({
      id: document.id,
      title: document.title || "",
      rawText: documentEditText(document),
      knowledgeType: document.knowledgeType || "text",
      status: document.status || "published",
      mode: "text",
    });
  };

  const openPositionCreate = () => {
    if (usesBusinessCatalogTools) return;
    setFaqForm?.(null);
    setDocumentForm?.(null);
    setPositionForm?.({ ...emptyPositionForm, aiAgentId: activeAIAgentId || "" });
  };

  const openPositionEdit = (position) => {
    if (usesBusinessCatalogTools) return;
    setFaqForm?.(null);
    setDocumentForm?.(null);
    setPositionForm?.({
      id: position.id,
      aiAgentId: position.aiAgentId || activeAIAgentId || "",
      title: position.title || "",
      location: position.location || "",
      workType: position.workType || position.employmentType || "",
      shortDescription: position.shortDescription || position.description || "",
      applyLink: position.applyLink || position.link || "",
      status: position.status || "published",
      isActive: position.isActive !== false && position.status !== "inactive",
    });
  };

  const applyKnowledgeTemplate = (template) => {
    if (!template || readOnly || !hasSelectedAgent) return;
    if (template.kind === "faq") {
      setKnowledgeTab?.("faqs");
      setDocumentForm?.(null);
      setPositionForm?.(null);
      setFaqForm?.({
        id: "",
        question: template.question,
        answer: template.answer,
        status: "published",
      });
      return;
    }
    if (template.kind === "document") {
      setKnowledgeTab?.("documents");
      setFaqForm?.(null);
      setPositionForm?.(null);
      setDocumentSourceMode?.("text");
      setUploadTitle?.("");
      setUploadFile?.(null);
      setDocumentForm?.({
        id: "",
        title: template.documentTitle,
        rawText: template.rawText,
        knowledgeType: "text",
        status: "published",
        mode: "text",
      });
      return;
    }
    if (template.kind === "position" && !usesBusinessCatalogTools) {
      setKnowledgeTab?.("positions");
      setFaqForm?.(null);
      setDocumentForm?.(null);
      setPositionForm?.({
        ...emptyPositionForm,
        aiAgentId: activeAIAgentId || "",
        title: template.positionTitle,
        location: template.location,
        workType: template.workType,
        shortDescription: template.shortDescription,
        applyLink: template.applyLink,
      });
    }
  };

  const faqActions = (
    <>
      <button type="button" className="btn btn-secondary" onClick={closeFaqForm}>Batal</button>
      <button type="submit" form="knowledge-faq-form" className="btn btn-primary" disabled={!hasSelectedAgent || !faqForm?.question || !faqForm?.answer || busyKey === "faq"}>
        {busyKey === "faq" ? "Menyimpan..." : "Simpan FAQ"}
      </button>
    </>
  );

  const uploadActions = (
    <>
      <button type="button" className="btn btn-secondary" onClick={closeDocumentForm}>Batal</button>
      <button type="submit" form="knowledge-document-upload-form" className="btn btn-primary" disabled={!hasSelectedAgent || !documentForm?.title || !documentForm?.file || busyKey === "document-upload"}>
        {busyKey === "document-upload" ? "Mengupload..." : "Upload"}
      </button>
    </>
  );

  const documentTextActions = (
    <>
      <button type="button" className="btn btn-secondary" onClick={closeDocumentForm}>Batal</button>
      <button type="submit" form="knowledge-document-edit-form" className="btn btn-primary" disabled={!hasSelectedAgent || !documentForm?.title || !documentForm?.rawText || busyKey === "document"}>
        {busyKey === "document" ? "Menyimpan..." : "Simpan Dokumen"}
      </button>
    </>
  );

  const positionActions = (
    <>
      <button type="button" className="btn btn-secondary" onClick={closePositionForm}>Batal</button>
      <button type="submit" form="knowledge-position-form" className="btn btn-primary" disabled={!positionForm?.title || busyKey === "position"}>
        {busyKey === "position" ? "Menyimpan..." : "Simpan"}
      </button>
    </>
  );

  function renderFaqForm(inline = false) {
    if (!faqForm) return null;
    return (
      <form id="knowledge-faq-form" className={inline ? "knowledge-inline-form" : ""} onSubmit={submitFaq}>
        <div className="form-group">
          <label className="form-label">Pertanyaan</label>
          <input className="form-input" value={faqForm.question || ""} onChange={(event) => setFaqForm?.({ ...faqForm, question: event.target.value })} placeholder="Contoh: Jam operasional bisnis ini kapan?" />
        </div>
        <div className="form-group">
          <label className="form-label">Jawaban</label>
          <textarea className="form-textarea" rows="6" value={faqForm.answer || ""} onChange={(event) => setFaqForm?.({ ...faqForm, answer: event.target.value })} placeholder="Tulis jawaban yang boleh dipakai AI. Kalau ada [ubah bagian ini: ...], ganti dulu sebelum publish." />
          <span className="form-field-hint">FAQ: jawaban pendek yang sering ditanya.</span>
        </div>
        <div className="form-group">
          <label className="form-label">Status</label>
          <select className="form-select" value={faqForm.status || "published"} onChange={(event) => setFaqForm?.({ ...faqForm, status: event.target.value })}>
            <option value="published">Published</option>
            <option value="draft">Draft</option>
          </select>
        </div>
        {inline ? <div className="knowledge-inline-actions">{faqActions}</div> : null}
      </form>
    );
  }

  function renderUploadForm(inline = false) {
    if (!documentIsUpload) return null;
    return (
      <form id="knowledge-document-upload-form" className={inline ? "knowledge-inline-form" : ""} onSubmit={uploadKnowledgeDocument}>
        <div className="form-group">
          <label className="form-label">Judul Dokumen</label>
          <input
            className="form-input"
            value={documentForm.title || ""}
            onChange={(event) => {
              setUploadTitle?.(event.target.value);
              setDocumentForm?.({ ...documentForm, title: event.target.value });
            }}
            placeholder="Nama dokumen..."
          />
        </div>
        <div className="form-group">
          <label className="form-label">Upload File</label>
          <div className="knowledge-upload-box">
            {Icons.upload}
            <div>Klik untuk upload atau drag & drop</div>
            <span>PDF, DOCX, TXT - max 10MB</span>
            <input
              type="file"
              id="doc-upload"
              accept=".pdf,.docx,.txt,text/plain,application/pdf,application/vnd.openxmlformats-officedocument.wordprocessingml.document"
              className="sr-only-file"
              onChange={(event) => {
                const file = event.target.files?.[0];
                if (!file) return;
                setUploadFile?.(file);
                setUploadTitle?.(documentForm.title || file.name);
                setDocumentForm?.({ ...documentForm, file, title: documentForm.title || file.name });
              }}
            />
            <button type="button" className="btn btn-secondary btn-sm" onClick={() => document.getElementById("doc-upload")?.click()}>Pilih File</button>
            {documentForm.file && <strong className="knowledge-selected-file">Terpilih: {documentForm.file.name}</strong>}
          </div>
        </div>
        {busyKey === "document-upload" ? (
          <div className="knowledge-busy-inline"><InlineSpinner /> File sedang diproses. Embedding lanjut di background.</div>
        ) : null}
        {inline ? <div className="knowledge-inline-actions">{uploadActions}</div> : null}
      </form>
    );
  }

  function renderDocumentTextForm(inline = false) {
    if (!documentIsText) return null;
    return (
      <form id="knowledge-document-edit-form" className={inline ? "knowledge-inline-form" : ""} onSubmit={submitDocument}>
        <div className="form-group">
          <label className="form-label">Judul Dokumen</label>
          <input className="form-input" value={documentForm.title || ""} onChange={(event) => setDocumentForm?.({ ...documentForm, title: event.target.value })} placeholder="Nama dokumen..." />
        </div>
        <div className="form-group">
          <label className="form-label">Isi Knowledge</label>
          <textarea className="form-textarea knowledge-guided-textarea" rows="10" value={documentForm.rawText || ""} onChange={(event) => setDocumentForm?.({ ...documentForm, rawText: event.target.value })} placeholder="Tulis isi dokumen manual di sini. Pakai template kalau bingung mulai dari mana." />
          <span className="form-field-hint">Dokumen: info panjang seperti profil, kebijakan, atau SOP.</span>
        </div>
        <div className="knowledge-form-grid">
          <div className="form-group">
            <label className="form-label">Tipe</label>
            <select className="form-select" value={documentForm.knowledgeType || "text"} onChange={(event) => setDocumentForm?.({ ...documentForm, knowledgeType: event.target.value })}>
              <option value="text">Text</option>
              <option value="file">Document</option>
              <option value="faq">FAQ</option>
            </select>
          </div>
          <div className="form-group">
            <label className="form-label">Status</label>
            <select className="form-select" value={documentForm.status || "published"} onChange={(event) => setDocumentForm?.({ ...documentForm, status: event.target.value })}>
              <option value="published">Published</option>
              <option value="draft">Draft</option>
            </select>
          </div>
        </div>
        {busyKey === "document" ? (
          <div className="knowledge-busy-inline"><InlineSpinner /> Menyimpan dan menyiapkan embedding.</div>
        ) : null}
        {inline ? <div className="knowledge-inline-actions">{documentTextActions}</div> : null}
      </form>
    );
  }

  function renderPositionForm(inline = false) {
    if (usesBusinessCatalogTools || !positionForm) return null;
    return (
      <form id="knowledge-position-form" className={inline ? "knowledge-inline-form" : ""} onSubmit={submitPosition}>
        <div className="form-group">
          <label className="form-label">Nama Info Produk/Layanan</label>
          <input className="form-input" value={positionForm.title || ""} onChange={(event) => setPositionForm?.({ ...positionForm, title: event.target.value })} placeholder="Contoh: Benefit Paket Konsultasi, Cara Pakai Produk A" />
        </div>
        <div className="knowledge-form-grid">
          <div className="form-group">
            <label className="form-label">Kategori / Lokasi</label>
            <input className="form-input" value={positionForm.location || ""} onChange={(event) => setPositionForm?.({ ...positionForm, location: event.target.value })} placeholder="Contoh: Digital, Jakarta, Remote" />
          </div>
          <div className="form-group">
            <label className="form-label">Tipe</label>
            <select className="form-select" value={positionForm.workType || ""} onChange={(event) => setPositionForm?.({ ...positionForm, workType: event.target.value })}>
              <option value="">Pilih tipe</option>
              <option value="Produk">Produk</option>
              <option value="Layanan">Layanan</option>
              <option value="Paket">Paket</option>
              <option value="Promo">Promo</option>
              <option value="Full-time">Full-time</option>
              <option value="Part-time">Part-time</option>
              <option value="Contract">Contract</option>
              <option value="Internship">Internship</option>
            </select>
          </div>
        </div>
        <div className="form-group">
          <label className="form-label">Deskripsi Singkat</label>
          <textarea className="form-textarea" rows="5" value={positionForm.shortDescription || ""} onChange={(event) => setPositionForm?.({ ...positionForm, shortDescription: event.target.value })} placeholder="Isi benefit, syarat, cara pakai, atau FAQ. Jangan taruh stok, slot, atau jadwal yang harus real-time." />
          <span className="form-field-hint">Untuk stok, jadwal, atau pesanan terkini, pakai Alat Bisnis.</span>
        </div>
        <div className="form-group">
          <label className="form-label">Link Detail / Order</label>
          <input className="form-input" value={positionForm.applyLink || ""} onChange={(event) => setPositionForm?.({ ...positionForm, applyLink: event.target.value })} placeholder="https://..." />
        </div>
        <div className="knowledge-form-grid">
          <div className="form-group">
            <label className="form-label">Status</label>
            <select className="form-select" value={positionForm.status || "published"} onChange={(event) => setPositionForm?.({ ...positionForm, status: event.target.value })}>
              <option value="published">Aktif</option>
              <option value="draft">Draft</option>
              <option value="inactive">Nonaktif</option>
            </select>
          </div>
          <label className="knowledge-toggle-row">
            <span>
              <strong>Aktif di AI</strong>
              <small>Dipakai untuk jawaban pelanggan</small>
            </span>
            <span className="toggle">
              <input type="checkbox" checked={positionForm.isActive !== false} onChange={(event) => setPositionForm?.({ ...positionForm, isActive: event.target.checked })} />
              <span className="toggle-slider" />
            </span>
          </label>
        </div>
        {busyKey === "position" ? (
          <div className="knowledge-busy-inline"><InlineSpinner /> Menyimpan info produk/layanan ke knowledge.</div>
        ) : null}
        {inline ? <div className="knowledge-inline-actions">{positionActions}</div> : null}
      </form>
    );
  }

  function renderInlineEditor() {
    if (!inlineEditorOpen) return null;
    let title = "";
    let body = null;
    let onClose = null;

    if (faqForm) {
      title = faqForm.id ? "Edit FAQ" : "Tambah FAQ";
      body = renderFaqForm(true);
      onClose = closeFaqForm;
    } else if (documentIsUpload) {
      title = "Upload Dokumen";
      body = renderUploadForm(true);
      onClose = closeDocumentForm;
    } else if (documentIsText) {
      title = documentForm?.id ? "Edit Dokumen" : "Tulis Dokumen Manual";
      body = renderDocumentTextForm(true);
      onClose = closeDocumentForm;
    } else if (!usesBusinessCatalogTools && positionForm) {
      title = positionForm.id ? "Edit Info Produk/Layanan" : "Tambah Info Produk/Layanan";
      body = renderPositionForm(true);
      onClose = closePositionForm;
    }

    return (
      <div className="embedded-knowledge-editor">
        <div className="embedded-knowledge-editor-head">
          <div>
            <strong>{title}</strong>
            <p>Perubahan masuk ke knowledge agent aktif. Kalau embedding butuh waktu, statusnya muncul di daftar.</p>
          </div>
          <button type="button" className="icon-btn" onClick={onClose} aria-label="Tutup editor">{Icons.close}</button>
        </div>
        {isDocumentBusy ? (
          <div className="knowledge-busy-inline"><InlineSpinner /> Proses dokumen / embedding berjalan. Tunggu di sini, tidak perlu popup.</div>
        ) : null}
        {body}
      </div>
    );
  }

  return (
    <div className={embedded ? "knowledge-view embedded-knowledge-view" : "knowledge-view"}>
      {!embedded ? (
        <div className="page-title-row">
          <div>
            <div className="page-title">Pengetahuan AI</div>
            <div className="page-desc">Materi yang dipakai AI untuk menjawab.</div>
          </div>
          <div className="page-actions">
            <select className="form-select" value={activeAIAgentId || ""} onChange={(event) => setActiveAIAgentId?.(event.target.value)}>
              <option value="">Pilih AI Agent</option>
              {aiAgents.filter((agent) => agent.isActive !== false).map((agent) => (
                <option key={agent.id} value={agent.id}>{agent.name}</option>
              ))}
            </select>
            <div className="topbar-search knowledge-search">
              {Icons.search}
              <input type="text" placeholder="Cari knowledge..." aria-label="Cari knowledge" />
            </div>
          </div>
        </div>
      ) : null}

      {!hasSelectedAgent ? (
        <div className="alert orange">
          {Icons.alert}
          <span>Pilih AI agent dulu supaya knowledge tidak tercampur antar agent.</span>
        </div>
      ) : null}

      {usesBusinessCatalogTools ? (
        <div className="alert blue knowledge-alert">
          {Icons.alert}
          <span>
            Produk, stok, pesanan, layanan, slot, dan booking sekarang dikelola dari Alat Bisnis. Knowledge agent dipakai untuk FAQ dan dokumen supaya AI tidak membaca dua sumber.
            {positions.length ? ` ${positions.length} info produk/layanan lama disembunyikan dari alur utama; pindahkan yang masih relevan ke plugin bisnis.` : ""}
          </span>
        </div>
      ) : null}

      <div className="knowledge-summary-grid">
        <KnowledgeStat value={faqs.length} label="Total FAQ" color="var(--blue)" />
        <KnowledgeStat value={documents.length} label="Dokumen" color="var(--orange)" />
        {!usesBusinessCatalogTools ? <KnowledgeStat value={activePositions} label="Info Produk/Layanan Aktif" color="var(--green)" /> : null}
      </div>

      {!readOnly ? (
        <div className="knowledge-template-panel">
          <div className="knowledge-template-head">
            <strong>Bingung mulai dari mana?</strong>
            <span>Pilih template, lalu ganti bagian bertanda [ubah bagian ini: ...].</span>
          </div>
          <div className="knowledge-template-actions">
            {availableKnowledgeTemplates.map((template) => (
              <button
                key={template.key}
                type="button"
                className="knowledge-template-card"
                disabled={!hasSelectedAgent}
                onClick={() => applyKnowledgeTemplate(template)}
              >
                <strong>{template.title}</strong>
                <span>{template.description}</span>
              </button>
            ))}
          </div>
        </div>
      ) : null}

      {renderInlineEditor()}

      <div className="knowledge-layout">
        <div className="card knowledge-main-card">
          <div className="tabs knowledge-tabs">
            {visibleKnowledgeTabs.map((tab) => (
              <button key={tab.id} type="button" className={`tab ${currentKnowledgeTab === tab.id ? "active" : ""}`} onClick={() => setKnowledgeTab?.(tab.id)}>
                {tab.label}
              </button>
            ))}
          </div>

          {currentKnowledgeTab === "faqs" && (
            <>
              <div className="card-header knowledge-card-header">
                <div className="card-title">FAQ Knowledge Base</div>
                {!readOnly && (
                  <button className="btn btn-primary btn-sm" disabled={!hasSelectedAgent} onClick={openFaqCreate} data-tour="knowledge">
                    {Icons.plus}
                    Tambah FAQ
                  </button>
                )}
              </div>
              <div className="table-wrap">
                <table>
                  <thead><tr><th>Pertanyaan</th><th>Preview Jawaban</th><th>Status</th>{!readOnly && <th>Aksi</th>}</tr></thead>
                  <tbody>
                    {faqs.map((faq) => (
                      <tr key={faq.id}>
                        <td><strong>{faq.question}</strong></td>
                        <td className="text-muted knowledge-preview-cell">{truncateText(faq.answer, 60)}</td>
                        <td><span className={`badge ${faq.status === "draft" ? "gray" : "green"}`}>{faq.status === "draft" ? "Draft" : "Aktif"}</span></td>
                        {!readOnly && (
                          <td>
                            <div className="knowledge-row-actions">
                              <button className="btn btn-secondary btn-sm" onClick={() => setFaqForm?.({ id: faq.id, question: faq.question || "", answer: faq.answer || "", status: faq.status || "published" })}>Edit</button>
                              <button className="btn btn-danger btn-sm" onClick={() => deleteFaq(faq.id)} disabled={busyKey === `faq-delete-${faq.id}`}>{busyKey === `faq-delete-${faq.id}` ? "..." : "Hapus"}</button>
                            </div>
                          </td>
                        )}
                      </tr>
                    ))}
                    {!faqs.length && <tr><td colSpan={readOnly ? 3 : 4} className="knowledge-empty-cell">Belum ada data FAQ.</td></tr>}
                  </tbody>
                </table>
              </div>
            </>
          )}

          {currentKnowledgeTab === "documents" && (
            <>
              <div className="card-header knowledge-card-header">
                <div>
                  <div className="card-title">Dokumen Knowledge</div>
                  <p className="knowledge-card-subtitle">Tambah dokumen untuk agent ini.</p>
                </div>
                {!readOnly && (
                  <div className="knowledge-header-actions">
                    <button className="btn btn-primary btn-sm" disabled={!hasSelectedAgent} onClick={openDocumentText} data-tour="knowledge-document">
                      {Icons.plus}
                      Tulis Manual
                    </button>
                    <button className="btn btn-secondary btn-sm" disabled={!hasSelectedAgent} onClick={openDocumentUpload}>
                      {Icons.upload}
                      Upload File
                    </button>
                  </div>
                )}
              </div>
              <div className="table-wrap">
                <table>
                  <thead><tr><th>Judul</th><th>Tipe</th><th>Diperbarui</th><th>Status</th>{!readOnly && <th>Aksi</th>}</tr></thead>
                  <tbody>
                    {documents.map((document) => {
                      const status = documentStatus(document);
                      const isSyncing = busyKey === `document-reindex-${document.id}`;
                      return (
                        <tr key={document.id}>
                          <td>
                            <strong>{document.title}</strong>
                            {document.pendingEmbeddingCount > 0 ? <small className="knowledge-row-note">{document.pendingEmbeddingCount} bagian masih diproses</small> : null}
                          </td>
                          <td><span className="badge gray">{documentTypeLabel(document)}</span></td>
                          <td className="text-muted text-sm">{formatDateTime(document.updatedAt || document.createdAt)}</td>
                          <td><span className={`badge ${status.tone}`}>{isSyncing ? "Sedang diproses" : status.label}</span></td>
                          {!readOnly && (
                            <td>
                              <div className="knowledge-row-actions">
                                <button className="btn btn-secondary btn-sm" onClick={() => openDocumentEdit(document)}>Edit</button>
                                <button className="btn btn-secondary btn-sm" onClick={() => reindexDocument(document.id)} disabled={isSyncing}>{isSyncing ? "Memproses..." : "Proses Ulang"}</button>
                                <button className="btn btn-danger btn-sm" onClick={() => deleteDocument(document.id)} disabled={busyKey === `document-delete-${document.id}`}>{busyKey === `document-delete-${document.id}` ? "..." : "Hapus"}</button>
                              </div>
                            </td>
                          )}
                        </tr>
                      );
                    })}
                    {!documents.length && <tr><td colSpan={readOnly ? 4 : 5} className="knowledge-empty-cell">Belum ada dokumen. Pakai Tulis Manual atau Upload File.</td></tr>}
                  </tbody>
                </table>
              </div>
            </>
          )}

          {currentKnowledgeTab === "positions" && !usesBusinessCatalogTools && (
            <>
              <div className="card-header knowledge-card-header">
                <div>
                  <div className="card-title">Info Produk & Layanan</div>
                  <p className="knowledge-card-subtitle">Untuk penjelasan umum yang stabil. Data operasional real-time tetap dikelola dari plugin bisnis.</p>
                </div>
                {!readOnly && (
                  <div className="knowledge-header-actions">
                    <button className="btn btn-primary btn-sm" disabled={!hasSelectedAgent} onClick={openPositionCreate} data-tour="knowledge-product">
                      {Icons.plus}
                      Tambah Info
                    </button>
                    <button className="btn btn-secondary btn-sm" onClick={refreshVectorIndex}>{Icons.refresh} Sync ATS</button>
                  </div>
                )}
              </div>
              <div className="alert blue knowledge-alert">
                {Icons.alert}
                <span>{businessRuleText}</span>
              </div>
              <div className="table-wrap">
                <table>
                  <thead><tr><th>Item</th><th>Kategori</th><th>Tipe</th><th>Link</th><th>Status</th>{!readOnly && <th>Aksi</th>}</tr></thead>
                  <tbody>
                    {positions.map((position) => {
                      const active = position.isActive !== false && position.status !== "inactive";
                      return (
                        <tr key={position.id}>
                          <td>
                            <strong>{position.title}</strong>
                            {position.shortDescription ? <small className="knowledge-row-note">{truncateText(position.shortDescription, 72)}</small> : null}
                          </td>
                          <td>{position.location || "-"}</td>
                          <td><span className="badge blue">{position.workType || position.employmentType || "-"}</span></td>
                          <td>{position.applyLink ? <a href={position.applyLink} target="_blank" rel="noreferrer" className="knowledge-link">{position.applyLink}</a> : <span className="text-muted">-</span>}</td>
                          <td><span className={`badge ${active ? "green" : "gray"}`}>{active ? "Aktif" : "Nonaktif"}</span></td>
                          {!readOnly && (
                            <td>
                              <div className="knowledge-row-actions">
                                <button className="btn btn-secondary btn-sm" onClick={() => openPositionEdit(position)}>Edit</button>
                                <button className="btn btn-danger btn-sm" onClick={() => deletePosition(position.id)} disabled={busyKey === `position-delete-${position.id}`}>{busyKey === `position-delete-${position.id}` ? "..." : "Hapus"}</button>
                              </div>
                            </td>
                          )}
                        </tr>
                      );
                    })}
                    {!positions.length && <tr><td colSpan={readOnly ? 5 : 6} className="knowledge-empty-cell">Belum ada info produk/layanan. Klik Tambah Info.</td></tr>}
                  </tbody>
                </table>
              </div>
            </>
          )}
        </div>

        <div className="knowledge-side">
          <div className="card">
            <div className="card-title mb-12">AI Health</div>
            <div className="knowledge-health-list">
              <KnowledgeHealthRow label="FAQ Published" value={`${publishedFaqs}/${faqs.length}`} color="var(--green)" />
              <KnowledgeHealthRow label="Docs Published" value={`${publishedDocs}/${documents.length}`} color="var(--blue)" />
              {!usesBusinessCatalogTools ? <KnowledgeHealthRow label="Active Items" value={`${activePositions}/${positions.length}`} color="var(--orange)" /> : null}
            </div>
            <div className="divider" />
            <div className="alert blue knowledge-alert">
              {Icons.alert}
              <span>{usesBusinessCatalogTools ? "AI memakai Pengetahuan untuk FAQ dan dokumen. Data produk, stok, order, slot, dan booking dibaca dari Alat Bisnis sesuai izin AI." : "AI memakai pengetahuan yang sudah dipublish untuk jawaban umum; pertanyaan stok, harga, order, slot, dan booking diprioritaskan ke alat bisnis kalau izin AI aktif."}</span>
            </div>
          </div>
          <div className="card">
            <div className="card-title mb-12">Knowledge Coverage</div>
            <CoverageBar label="FAQ" value={faqs.length ? Math.round((publishedFaqs / faqs.length) * 100) : 0} color="var(--green)" />
            <CoverageBar label="Dokumen" value={documents.length ? Math.round((publishedDocs / documents.length) * 100) : 0} color="var(--blue)" />
            {!usesBusinessCatalogTools ? <CoverageBar label="Info Produk/Layanan" value={positions.length ? Math.round((activePositions / positions.length) * 100) : 0} color="var(--orange)" /> : null}
          </div>
        </div>
      </div>

      {!embedded && faqForm ? (
        <ModalShell
          title={faqForm.id ? "Edit FAQ" : "Tambah FAQ"}
          onClose={closeFaqForm}
          footer={faqActions}
        >
          {renderFaqForm(false)}
        </ModalShell>
      ) : null}

      {!embedded && documentIsUpload ? (
        <ModalShell
          title="Upload Dokumen"
          onClose={closeDocumentForm}
          footer={uploadActions}
        >
          {renderUploadForm(false)}
        </ModalShell>
      ) : null}

      {!embedded && documentIsText ? (
        <ModalShell
          title={documentForm?.id ? "Edit Dokumen" : "Tulis Dokumen Manual"}
          onClose={closeDocumentForm}
          footer={documentTextActions}
        >
          {renderDocumentTextForm(false)}
        </ModalShell>
      ) : null}

      {!embedded && positionForm ? (
        <ModalShell
          title={positionForm.id ? "Edit Info Produk/Layanan" : "Tambah Info Produk/Layanan"}
          onClose={closePositionForm}
          footer={positionActions}
        >
          {renderPositionForm(false)}
        </ModalShell>
      ) : null}
    </div>
  );
}

export { emptyPositionForm };
