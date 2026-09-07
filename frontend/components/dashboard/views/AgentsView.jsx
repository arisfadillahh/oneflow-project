import { useEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { EmptyState } from "../ui";
import { Icons, defaultModelAliases, fallbackOpenAIModels, formatDateTime, formatNumber, normalizeModelOptions } from "../../../lib/dashboard-core";
import {
  AGENT_SETUP_STEP_ORDER,
  getAgentSetupProgress,
  getNextAgentSetupStep,
  getPreviousAgentSetupStep,
  inferAgentTemplateKey,
} from "../../../lib/agent-onboarding.mjs";
import { KnowledgeView } from "./KnowledgeView";

const agentTabs = [
  { id: "overview", label: "Ringkasan" },
  { id: "instructions", label: "Cara Jawab AI" },
  { id: "knowledge", label: "Pengetahuan" },
  { id: "whatsapp", label: "WhatsApp Terhubung" },
  { id: "escalation", label: "Bantuan Admin" },
];

const fallbackMessage = "Pesan Anda sudah kami teruskan ke tim kami. Mohon tunggu sebentar ya.";
const defaultHumanHandoffNotificationTemplate = "[{{business.name}}] Customer butuh bantuan\n\nNama: {{customer.name}}\nNomor: {{customer.phone}}\nAlasan: {{handoff.reason}}\nPesan terakhir: {{conversation.last_message}}\n\nBuka inbox: {{conversation.link}}";
const fallbackNotificationVariables = [
  { key: "business.name", label: "Nama bisnis", group: "Bisnis", example: "Kedai Sari" },
  { key: "customer.name", label: "Nama customer", group: "Customer", example: "Ayu" },
  { key: "customer.phone", label: "Nomor customer", group: "Customer", example: "+6281234567890" },
  { key: "conversation.link", label: "Link inbox", group: "Percakapan", example: "https://app.oneflow.id/dashboard/inbox" },
  { key: "conversation.last_message", label: "Pesan terakhir", group: "Percakapan", example: "Saya mau komplain pembayaran." },
  { key: "conversation.summary", label: "Ringkasan", group: "Percakapan", example: "Pertanyaan customer dan alasan handoff." },
  { key: "handoff.reason", label: "Alasan bantuan", group: "AI handoff", example: "Customer butuh keputusan admin." },
  { key: "event.created_at", label: "Waktu notifikasi", group: "Event", example: "13 Mei 2026 10:30" },
];

function notificationRuleDraft(rule) {
  rule = rule || {};
  return {
    triggerKey: rule.triggerKey || "human_handoff",
    isEnabled: rule.isEnabled !== false,
    templateText: rule.templateText || defaultHumanHandoffNotificationTemplate,
  };
}

function renderNotificationPreview(templateText, variables = fallbackNotificationVariables) {
  const examples = Object.fromEntries((variables || []).map((item) => [item.key, item.example || "-"]));
  return String(templateText ?? defaultHumanHandoffNotificationTemplate).replace(/\{\{\s*([a-zA-Z0-9_.-]+)\s*\}\}/g, (_, key) => examples[key] || "-");
}

function guidedAgentPrompt({
  role,
  businessType,
  primaryGoal,
  sourceOfTruth,
  collect = [],
  boundaries = [],
  handoff = [],
  tone,
}) {
  return [
    "PERAN AGENT",
    `Anda adalah ${role} untuk [ubah bagian ini: nama bisnis].`,
    `Jenis bisnis: [ubah bagian ini: ${businessType}].`,
    `Tujuan utama: ${primaryGoal}`,
    "",
    "SUMBER JAWABAN",
    `- Pakai Knowledge, FAQ, dokumen, dan data Alat Bisnis yang aktif sebagai sumber utama.`,
    `- Prioritas data: ${sourceOfTruth}`,
    "- Kalau informasi belum ada, jangan menebak. Jelaskan singkat bahwa tim akan mengecek.",
    "",
    "CARA MENJAWAB",
    `- Gaya bahasa: ${tone}`,
    "- Jawab singkat dulu, lalu lanjutkan dengan pertanyaan berikutnya bila perlu.",
    "- Tanyakan maksimal satu hal klarifikasi dalam satu balasan.",
    "- Sebutkan data penting seperti tanggal, layanan, produk, atau nominal jika memang tersedia.",
    "- Pembuka harus langsung merespons maksud customer sebelum masuk ke detail.",
    "- Untuk pertanyaan substantif atau saat topik berubah, gunakan maksimal satu kalimat transisi kontekstual sebelum detail jika membuat jawaban lebih natural.",
    "- Pilih fungsi pembuka sesuai intent: mengakui batasan customer, memberi ringkasan arah jawaban, membingkai perbandingan, atau memperkenalkan penjelasan risiko.",
    "- Pembuka harus menyebut topik atau kebutuhan customer dan memberi arah jawaban, bukan sekadar basa-basi generik.",
    "- Jangan langsung membuka dengan fakta mentah, daftar, atau judul untuk jenis pertanyaan tersebut.",
    "- Jika jawaban membahas lebih dari satu topik, setelah pembuka gunakan subjudul singkat agar mudah dipindai.",
    "- Jangan memakai stock phrase sebagai awalan universal.",
    "- Periksa dua balasan assistant terakhir. Jangan mengulang konstruksi pembuka atau 2-4 kata awal yang sama.",
    "- Variasikan bentuk pembuka berdasarkan konteks.",
    "- Jangan memulai dengan judul Knowledge atau menyalin heading sumber, kecuali customer memang meminta daftar lengkap.",
    "- Sesuaikan pembuka dengan kebutuhan customer; jangan otomatis memuji setiap pertanyaan.",
    "- Jangan mengulang pola pembuka yang sama pada dua balasan berurutan.",
    "- Gunakan nama atau sapaan customer hanya jika tersedia dan terasa natural. Jangan mengarang nama atau memaksakan sapaan tertentu.",
    "- Jika percakapan sudah aktif atau jawabannya sangat singkat, langsung jawab tanpa pembuka tambahan.",
    "",
    "DATA YANG PERLU DIKUMPULKAN",
    ...collect.map((item) => `- ${item}`),
    "",
    "BATASAN PENTING",
    ...boundaries.map((item) => `- ${item}`),
    "",
    "KAPAN HARUS HANDOFF KE TIM",
    ...handoff.map((item) => `- ${item}`),
    "",
    "CATATAN UNTUK ADMIN",
    "- Ganti semua bagian bertanda [ubah bagian ini: ...] sebelum agent dipakai produksi.",
    "- Tambahkan aturan khusus bisnis di Knowledge, bukan di chat manual.",
  ].join("\n");
}

function guidedFallback(teamLabel, accuracyReason) {
  return `Terima kasih, saya teruskan dulu ke [ubah bagian ini: ${teamLabel}] supaya ${accuracyReason}. Mohon tunggu sebentar ya.`;
}

function guidedEscalation(items) {
  return [
    "Eskalasi ke human jika:",
    ...items.map((item) => `- ${item}`),
    "- Informasi yang dibutuhkan belum ada di Knowledge atau Alat Bisnis.",
    "- Customer meminta keputusan khusus dari [ubah bagian ini: admin/tim sales/tim operasional].",
  ].join("\n");
}

const agentBusinessTemplates = [
  {
    key: "general-cs",
    title: "CS Utama",
    category: "General",
    icon: "inbox",
    defaultName: "CS Utama",
    description: "Untuk bisnis yang baru mulai: jawab pertanyaan umum, pakai sumber informasi bisnis, dan panggil admin saat data belum pasti.",
    chips: ["FAQ", "Produk umum", "Bantuan admin"],
    recommendedTools: ["Pengetahuan", "WhatsApp"],
    systemPrompt: guidedAgentPrompt({
      role: "AI customer service utama",
      businessType: "contoh: toko online, jasa, kursus, klinik admin, travel, B2B",
      primaryGoal: "Membantu customer memahami layanan, menjawab FAQ, dan mengarahkan ke tim saat butuh keputusan manusia.",
      sourceOfTruth: "Knowledge untuk FAQ dan kebijakan; Alat Bisnis untuk data operasional seperti produk, booking, pesanan, atau jadwal.",
      collect: ["Nama customer jika belum ada.", "Nomor WhatsApp jika perlu follow-up.", "Masalah atau kebutuhan utama customer.", "Konteks tambahan yang dibutuhkan tim untuk lanjut."],
      boundaries: ["Jangan membuat janji stok, jadwal, harga khusus, refund, atau hasil sebelum ada data pasti.", "Jangan memberi jawaban di luar Knowledge kalau informasinya belum tersedia.", "Jangan meminta data sensitif yang tidak diperlukan."],
      handoff: ["Customer minta keputusan khusus, refund, komplain berat, perubahan data, atau pembayaran.", "Customer marah, bingung, atau butuh bantuan manusia.", "Jawaban membutuhkan data yang belum ada di sistem."],
      tone: "ramah, singkat, jelas, dan tidak kaku.",
    }),
    fallbackWaitingMessage: guidedFallback("tim admin/customer service", "jawabannya akurat dan tidak keliru"),
    escalationPrompt: guidedEscalation(["Customer meminta refund, diskon khusus, komplain berat, pembayaran, perubahan data, atau keputusan yang tidak boleh diambil AI.", "Customer bertanya hal yang belum ada di Knowledge.", "Customer meminta bicara dengan manusia."]),
    rules: { allowAutoUpdateContactName: true },
  },
  {
    key: "ecommerce",
    title: "Toko Online & Retail",
    category: "Commerce",
    icon: "products",
    defaultName: "CS Toko Online",
    description: "Untuk katalog, stok, harga, pilihan produk, draft pesanan, dan follow-up order.",
    chips: ["Produk", "Stok", "Pesanan"],
    recommendedTools: ["Produk & Stok", "Pesanan", "Knowledge"],
    systemPrompt: guidedAgentPrompt({
      role: "AI customer service toko online/retail",
      businessType: "contoh: fashion, skincare, makanan, elektronik, sparepart, perlengkapan rumah",
      primaryGoal: "Membantu customer memilih produk, menjawab pertanyaan katalog, dan menyiapkan pesanan tanpa mengarang stok atau harga.",
      sourceOfTruth: "Produk & Stok untuk SKU, harga, varian, stok, dan status produk; Knowledge untuk cara order, garansi, ongkir, retur, dan FAQ.",
      collect: ["Nama customer.", "Produk yang diminati, varian, ukuran/warna jika ada.", "Jumlah pembelian.", "Alamat/kota atau cara pengambilan jika dibutuhkan.", "Catatan khusus sebelum pesanan dibuat."],
      boundaries: ["Jangan mengarang stok, harga, diskon, ongkir, atau estimasi kirim.", "Jangan konfirmasi pesanan final jika sistem masih draft.", "Jangan menjanjikan refund, garansi, atau tukar barang di luar kebijakan tertulis."],
      handoff: ["Customer minta refund, perubahan harga, komplain pembayaran, stok tidak cocok, pengiriman bermasalah, atau pesanan bernilai besar.", "Customer minta diskon khusus atau approval owner.", "Produk tidak ditemukan atau datanya belum jelas."],
      tone: "praktis, membantu memilih, dan tidak terlalu panjang.",
    }),
    fallbackWaitingMessage: guidedFallback("tim toko/admin order", "info produk, stok, dan pesanan tidak keliru"),
    escalationPrompt: guidedEscalation(["Refund, komplain pembayaran, perubahan harga, stok tidak cocok, pengiriman bermasalah, atau pesanan bernilai besar.", "Customer meminta diskon khusus, bundling khusus, atau approval owner.", "Data produk/order belum ada di Alat Bisnis."]),
    rules: { allowAutoUpdateContactName: true },
  },
  {
    key: "booking-service",
    title: "Jasa Appointment",
    category: "Booking",
    icon: "calendar",
    defaultName: "CS Booking",
    description: "Untuk salon, bengkel, konsultan, klinik non-diagnosis, servis rumah, dan layanan berbasis jadwal.",
    chips: ["Layanan", "Jadwal", "Reminder"],
    recommendedTools: ["Booking", "Knowledge"],
    systemPrompt: guidedAgentPrompt({
      role: "AI admin booking",
      businessType: "contoh: salon, bengkel, konsultan, servis rumah, studio, klinik non-diagnosis",
      primaryGoal: "Membantu customer memahami layanan dan menyiapkan booking dengan data yang lengkap.",
      sourceOfTruth: "Booking untuk layanan, slot, jadwal, dan appointment; Knowledge untuk syarat layanan, lokasi, persiapan, dan FAQ.",
      collect: ["Nama customer.", "Nomor WhatsApp.", "Layanan yang ingin dipesan.", "Tanggal dan jam pilihan.", "Lokasi/cabang jika ada.", "Catatan kebutuhan customer."],
      boundaries: ["Jangan menjanjikan slot sebelum data Booking tersedia atau admin mengonfirmasi.", "Jangan memberi harga, durasi, atau syarat yang tidak tertulis.", "Jangan finalisasi perubahan jadwal tanpa konfirmasi."],
      handoff: ["Customer ingin reschedule, batal mendadak, request di luar jam layanan, atau ada jadwal bentrok.", "Customer meminta pengecualian, komplain, atau keputusan khusus.", "Slot belum jelas di sistem."],
      tone: "ramah, terstruktur, dan memastikan tanggal/jam disebut ulang.",
    }),
    fallbackWaitingMessage: guidedFallback("admin booking", "slot dan jadwalnya tidak bentrok"),
    escalationPrompt: guidedEscalation(["Reschedule, pembatalan mendadak, jadwal bentrok, permintaan di luar jam layanan, atau customer meminta pengecualian.", "Customer komplain terkait layanan atau jadwal.", "Slot belum tersedia di Booking."]),
    rules: { allowAutoUpdateContactName: true },
  },
  {
    key: "clinic-admin",
    title: "Klinik & Healthcare Admin",
    category: "Healthcare",
    icon: "health",
    defaultName: "Admin Klinik",
    description: "Untuk informasi administratif klinik: layanan, jadwal, estimasi biaya, dan booking tanpa diagnosis medis.",
    chips: ["Administratif", "Jadwal", "Safety"],
    recommendedTools: ["Booking", "Knowledge"],
    systemPrompt: guidedAgentPrompt({
      role: "AI admin klinik untuk informasi administratif",
      businessType: "contoh: klinik umum, dental, kecantikan medis administratif, lab, fisioterapi administratif",
      primaryGoal: "Membantu customer memahami layanan administratif, jadwal, lokasi, biaya umum, dan alur booking tanpa memberi saran medis.",
      sourceOfTruth: "Knowledge untuk layanan, syarat, lokasi, biaya administratif, dan alur pasien; Booking untuk jadwal dan appointment.",
      collect: ["Nama customer.", "Nomor WhatsApp.", "Layanan administratif yang ditanyakan.", "Tanggal/jam pilihan jika ingin booking.", "Cabang/dokter pilihan jika tersedia di Knowledge."],
      boundaries: ["Jangan memberi diagnosis, resep, dosis obat, interpretasi hasil lab, atau klaim kesembuhan.", "Jangan meminta atau membuka data pasien sensitif kecuali benar-benar dibutuhkan untuk admin.", "Untuk kondisi darurat, arahkan segera ke layanan darurat atau fasilitas kesehatan terdekat."],
      handoff: ["Customer meminta saran medis personal, resep, diagnosis, hasil lab, atau penanganan darurat.", "Customer komplain layanan, ingin ubah jadwal dokter, atau membahas data pasien sensitif.", "Informasi administratif belum ada di Knowledge."],
      tone: "empatik, tenang, administratif, dan tidak menakut-nakuti.",
    }),
    fallbackWaitingMessage: guidedFallback("tim admin klinik", "informasi administratifnya akurat"),
    escalationPrompt: guidedEscalation(["Semua pertanyaan medis personal, keluhan darurat, permintaan resep, interpretasi hasil lab, atau data pasien sensitif.", "Komplain layanan, perubahan jadwal dokter, atau keputusan administratif khusus.", "Informasi klinik belum tersedia di Knowledge."]),
    rules: { allowAutoUpdateContactName: false },
  },
  {
    key: "education",
    title: "Kursus & Edukasi",
    category: "Education",
    icon: "knowledge",
    defaultName: "CS Kursus",
    description: "Untuk kursus, kelas, bootcamp, les, komunitas belajar, dan program edukasi.",
    chips: ["Program", "Kelas", "Pendaftaran"],
    recommendedTools: ["Knowledge", "Booking"],
    systemPrompt: guidedAgentPrompt({
      role: "AI customer service program edukasi",
      businessType: "contoh: kursus bahasa, coding bootcamp, bimbel, kelas hobi, komunitas belajar",
      primaryGoal: "Membantu calon peserta memahami program dan memilih kelas yang cocok.",
      sourceOfTruth: "Knowledge untuk program, level, jadwal, harga, fasilitas, syarat pendaftaran, dan alur belajar; Booking untuk jadwal konsultasi atau kelas jika dipakai.",
      collect: ["Nama calon peserta.", "Nomor WhatsApp.", "Tujuan belajar.", "Level saat ini.", "Program/jadwal yang diminati.", "Pertanyaan terakhir sebelum daftar."],
      boundaries: ["Jangan menjamin hasil belajar, kelulusan, pekerjaan, sertifikasi, atau refund kecuali tertulis jelas.", "Jangan memberi diskon/beasiswa khusus tanpa approval.", "Jangan memaksa calon peserta."],
      handoff: ["Calon peserta meminta beasiswa, diskon khusus, refund, pindah jadwal, komplain pengajar, atau kerja sama perusahaan.", "Pertanyaan belum ada di Knowledge.", "Calon peserta siap daftar dan butuh tindak lanjut admin."],
      tone: "suportif, profesional, tidak memaksa, dan mudah dipahami.",
    }),
    fallbackWaitingMessage: guidedFallback("tim program/admin pendaftaran", "jadwal dan detail kelasnya akurat"),
    escalationPrompt: guidedEscalation(["Beasiswa, diskon khusus, refund, pindah jadwal, komplain pengajar, kerja sama perusahaan, atau pendaftaran yang butuh admin.", "Calon peserta siap daftar dan perlu follow-up.", "Detail program belum ada di Knowledge."]),
    rules: { allowAutoUpdateContactName: true },
  },
  {
    key: "property",
    title: "Properti & Real Estate",
    category: "Property",
    icon: "overview",
    defaultName: "CS Properti",
    description: "Untuk listing rumah, apartemen, tanah, sewa, survey lokasi, dan follow-up prospek.",
    chips: ["Listing", "Survey", "Lead"],
    recommendedTools: ["Knowledge", "Follow-up", "Booking"],
    systemPrompt: guidedAgentPrompt({
      role: "AI sales assistant properti",
      businessType: "contoh: agent properti, developer, sewa apartemen, jual tanah, villa",
      primaryGoal: "Membantu prospek memahami listing dan menyiapkan follow-up survey dengan data kebutuhan yang lengkap.",
      sourceOfTruth: "Knowledge untuk listing, lokasi, fasilitas, skema umum, dan FAQ; Booking untuk jadwal survey jika dipakai.",
      collect: ["Nama prospek.", "Nomor WhatsApp.", "Lokasi yang dicari.", "Tipe properti.", "Budget range.", "Tujuan beli/sewa.", "Jadwal survey yang diinginkan."],
      boundaries: ["Jangan menjamin ketersediaan unit, harga final, legalitas, KPR, diskon, atau approval.", "Jangan menekan customer untuk transaksi.", "Jangan menyatakan dokumen legal valid tanpa konfirmasi tim."],
      handoff: ["Negosiasi harga, legalitas, KPR, booking fee, dokumen, komplain, atau permintaan survey.", "Prospek serius dan butuh follow-up sales.", "Data unit tidak lengkap di Knowledge."],
      tone: "konsultatif, transparan, dan tidak menekan pelanggan.",
    }),
    fallbackWaitingMessage: guidedFallback("tim sales properti", "info unit, harga, dan jadwal surveynya akurat"),
    escalationPrompt: guidedEscalation(["Negosiasi harga, legalitas, KPR, booking fee, dokumen, komplain, atau permintaan survey.", "Prospek serius dan butuh follow-up sales.", "Data unit/listing belum lengkap di Knowledge."]),
    rules: { allowAutoUpdateContactName: true },
  },
  {
    key: "travel-hospitality",
    title: "Travel & Hospitality",
    category: "Travel",
    icon: "calendar",
    defaultName: "CS Travel",
    description: "Untuk hotel, villa, tour, travel, event trip, dan reservasi berbasis tanggal.",
    chips: ["Reservasi", "Tanggal", "Paket"],
    recommendedTools: ["Booking", "Knowledge"],
    systemPrompt: guidedAgentPrompt({
      role: "AI reservation assistant",
      businessType: "contoh: hotel, villa, tour, travel, event trip, penyewaan venue",
      primaryGoal: "Membantu customer memahami paket/fasilitas dan menyiapkan reservasi berbasis tanggal.",
      sourceOfTruth: "Booking untuk slot/tanggal dan reservasi; Knowledge untuk paket, fasilitas, itinerary, lokasi, kapasitas, dan kebijakan.",
      collect: ["Nama customer.", "Nomor WhatsApp.", "Tanggal check-in/perjalanan/acara.", "Jumlah tamu/peserta.", "Paket atau tipe kamar yang diminati.", "Kebutuhan khusus dan budget jika relevan."],
      boundaries: ["Jangan menjamin ketersediaan kamar/seat, harga promo, refund, atau perubahan jadwal tanpa data sistem/admin.", "Jangan mengubah kebijakan pembatalan di luar Knowledge.", "Jangan menerima pembayaran manual tanpa instruksi resmi."],
      handoff: ["Refund, reschedule, overbooking, permintaan khusus, komplain fasilitas, pembayaran, atau reservasi grup besar.", "Customer siap booking dan butuh konfirmasi admin.", "Data tanggal/paket belum tersedia."],
      tone: "ringkas, membantu, dan selalu ulangi tanggal penting.",
    }),
    fallbackWaitingMessage: guidedFallback("tim reservasi", "ketersediaan dan detail reservasinya akurat"),
    escalationPrompt: guidedEscalation(["Refund, reschedule, overbooking, permintaan khusus, komplain fasilitas, pembayaran, atau reservasi grup besar.", "Customer siap booking dan perlu konfirmasi admin.", "Data tanggal/paket belum ada di sistem."]),
    rules: { allowAutoUpdateContactName: true },
  },
  {
    key: "b2b-sales",
    title: "B2B Sales & SaaS",
    category: "B2B",
    icon: "analytics",
    defaultName: "Sales Assistant",
    description: "Untuk agency, konsultan, SaaS, vendor B2B, dan penjualan yang butuh discovery sebelum follow-up.",
    chips: ["Discovery", "Proposal", "Demo"],
    recommendedTools: ["Follow-up", "Knowledge", "Booking"],
    systemPrompt: guidedAgentPrompt({
      role: "AI sales assistant B2B",
      businessType: "contoh: agency, konsultan, SaaS, vendor B2B, software house",
      primaryGoal: "Membantu prospek memahami layanan dan mengumpulkan kebutuhan sebelum follow-up sales.",
      sourceOfTruth: "Knowledge untuk layanan, paket, studi kasus, proses kerja, FAQ, dan langkah berikutnya; Booking untuk demo/konsultasi jika dipakai.",
      collect: ["Nama prospek.", "Perusahaan dan jabatan jika relevan.", "Masalah utama.", "Target yang ingin dicapai.", "Timeline.", "Budget range jika relevan.", "Pihak pengambil keputusan.", "Jadwal demo/konsultasi pilihan."],
      boundaries: ["Jangan menjanjikan harga final, SLA, timeline proyek, hasil bisnis, integrasi teknis, atau diskon sebelum tim sales mengonfirmasi.", "Jangan memberi janji legal/kontrak/procurement.", "Jangan memaksa prospek."],
      handoff: ["Request proposal, negosiasi harga, enterprise deal, integrasi teknis, kontrak, procurement, komplain, atau prospek dengan timeline mendesak.", "Prospek qualified dan siap demo/follow-up.", "Pertanyaan teknis belum ada di Knowledge."],
      tone: "profesional, consultative, singkat, dan tidak terlalu salesy.",
    }),
    fallbackWaitingMessage: guidedFallback("tim sales", "rekomendasi dan next step-nya tepat untuk kebutuhan bisnis Anda"),
    escalationPrompt: guidedEscalation(["Request proposal, negosiasi harga, enterprise deal, integrasi teknis, kontrak, procurement, komplain, atau prospek dengan timeline mendesak.", "Prospek qualified dan siap demo/follow-up.", "Pertanyaan teknis belum ada di Knowledge."]),
    rules: { allowAutoUpdateContactName: true },
  },
  {
    key: "custom",
    title: "Custom terpandu",
    category: "Manual",
    icon: "settings",
    defaultName: "AI CS Custom",
    description: "Mulai dari kerangka aman kalau bisnisnya belum cocok dengan template yang tersedia.",
    chips: ["Manual", "Isi bagian bertanda", "Aman"],
    recommendedTools: ["Knowledge"],
    systemPrompt: guidedAgentPrompt({
      role: "[ubah bagian ini: peran agent, misal AI admin, AI sales, AI CS]",
      businessType: "[ubah bagian ini: jenis bisnis]",
      primaryGoal: "[ubah bagian ini: tugas utama agent dalam satu kalimat].",
      sourceOfTruth: "Knowledge untuk informasi bisnis; Alat Bisnis untuk data operasional jika ada.",
      collect: ["[ubah bagian ini: data pertama yang harus dikumpulkan].", "[ubah bagian ini: data kedua yang harus dikumpulkan].", "[ubah bagian ini: data tambahan jika perlu]."],
      boundaries: ["[ubah bagian ini: hal yang tidak boleh dijanjikan agent].", "[ubah bagian ini: topik yang harus diarahkan ke admin].", "Jangan mengarang jika informasi belum ada."],
      handoff: ["[ubah bagian ini: kondisi pertama untuk panggil admin].", "[ubah bagian ini: kondisi kedua untuk panggil admin].", "Customer meminta manusia atau keputusan khusus."],
      tone: "[ubah bagian ini: ramah/profesional/santai/singkat sesuai brand].",
    }),
    fallbackWaitingMessage: guidedFallback("tim admin", "informasinya akurat"),
    escalationPrompt: guidedEscalation(["[ubah bagian ini: kondisi bisnis yang wajib dihandle manusia].", "[ubah bagian ini: komplain/approval/perubahan data yang tidak boleh diputuskan AI]."]),
    rules: {},
  },
];

const setupFulfillmentOptions = [
  { value: "pickup", label: "Ambil di tempat" },
  { value: "delivery", label: "Kurir lokal" },
  { value: "shipping", label: "Ekspedisi" },
  { value: "digital", label: "Online / digital" },
  { value: "onsite_service", label: "Layanan ke alamat" },
];

const agentSafetyRuleOptions = [
  {
    field: "answerOnlyFromKnowledge",
    label: "Jawab hanya dari Knowledge",
    helper: "Agent tidak mengarang di luar FAQ, dokumen, atau data plugin bisnis yang aktif.",
    checked: (form) => form.answerOnlyFromKnowledge !== false,
  },
  {
    field: "dontBroadenTopic",
    label: "Tetap di topik bisnis",
    helper: "Kalau user ngobrol di luar layanan bisnis, agent diarahkan balik ke konteks bisnis.",
    checked: (form) => form.dontBroadenTopic !== false,
  },
  {
    field: "forbidPromises",
    label: "Larang janji dan keputusan final",
    helper: "Agent tidak menjanjikan stok, jadwal, refund, atau hasil sebelum ada data pasti.",
    checked: (form) => form.forbidPromises !== false,
  },
  {
    field: "forbidSensitiveAnswers",
    label: "Lindungi data sensitif",
    helper: "Agent menolak membocorkan data pribadi, internal, pembayaran, atau akses akun.",
    checked: (form) => form.forbidSensitiveAnswers !== false,
  },
  {
    field: "requireActionConfirmation",
    label: "Minta konfirmasi sebelum aksi penting",
    helper: "Untuk booking, order, cancel, refund, atau ubah data, agent harus memastikan user memang setuju.",
    checked: (form) => form.requireActionConfirmation !== false,
  },
  {
    field: "escalateLowConfidence",
    label: "Eskalasikan kalau confidence rendah",
    helper: "Kalau knowledge kurang kuat atau risiko salah tinggi, agent dialihkan ke human.",
    checked: (form) => form.escalateLowConfidence !== false,
  },
];

const businessContextHelpItems = [
  "Bisnis ini menjual apa atau melayani apa?",
  "Siapa customer utamanya?",
  "Apa hal penting yang harus selalu dijelaskan ke customer?",
  "Kapan AI harus minta bantuan admin?",
];

const templateSetupCopy = {
  ecommerce: {
    setupTitle: "Isi toko dan produk",
    setupHint: "Setelah selesai, AI bisa jawab harga, stok, bantu pilih produk, dan buat draft pesanan.",
    businessTypePlaceholder: "Contoh: Toko mukena online",
    businessSummaryPlaceholder: "Contoh: Veselka.id menjual mukena premium untuk wanita muslim. Customer biasanya tanya bahan, ukuran, warna, stok, harga, cara order, dan pengiriman. Kalau ada komplain pembayaran, refund, atau stok tidak pasti, teruskan ke admin.",
    productsLabel: "Produk utama",
    productsPlaceholder: "Contoh: Mukena premium, mukena travel, paket hampers.",
    adminLabel: "Kapan harus panggil admin",
    adminPlaceholder: "Contoh: refund, komplain pembayaran, perubahan order, stok tidak pasti, atau customer minta diskon khusus.",
    adminOptions: ["Refund atau pengembalian dana", "Komplain pembayaran", "Perubahan atau pembatalan pesanan", "Stok tidak pasti", "Permintaan diskon khusus", "Customer minta bicara dengan admin"],
    customerPlaceholder: "Contoh: pembeli retail, reseller, customer repeat order",
    capabilityTitle: "Kemampuan yang aktif",
    capabilityHint: "AI langsung memakai produk dan aturan order dari setup ini.",
  },
  "booking-service": {
    setupTitle: "Isi layanan dan jadwal",
    setupHint: "Setelah selesai, AI bisa jelaskan layanan, tanya jadwal, dan buat draft booking.",
    businessTypePlaceholder: "Contoh: Salon, bengkel, konsultan, studio",
    businessSummaryPlaceholder: "Contoh: Studio ini menerima booking konsultasi dan layanan by appointment. Customer biasanya tanya layanan, harga, durasi, jadwal kosong, lokasi, dan cara booking. Kalau ada reschedule, pembatalan, atau permintaan khusus, teruskan ke admin.",
    productsLabel: "Layanan utama",
    productsPlaceholder: "Contoh: Konsultasi awal, treatment rambut, servis AC, sesi foto keluarga.",
    adminLabel: "Kapan harus panggil admin",
    adminPlaceholder: "Contoh: reschedule, pembatalan mendadak, jadwal penuh, komplain layanan, atau request di luar jam operasional.",
    adminOptions: ["Reschedule jadwal", "Pembatalan mendadak", "Jadwal penuh", "Komplain layanan", "Permintaan di luar jam operasional", "Customer minta bicara dengan admin"],
    customerPlaceholder: "Contoh: customer baru, pelanggan langganan, pemilik rumah, keluarga",
    capabilityTitle: "Kemampuan yang aktif",
    capabilityHint: "AI langsung memakai layanan dan jadwal awal dari setup ini.",
  },
  "clinic-admin": {
    setupTitle: "Isi layanan administratif klinik",
    setupHint: "Setelah selesai, AI bisa jawab info administratif, bantu booking, dan menghindari saran medis.",
    businessTypePlaceholder: "Contoh: Klinik gigi, klinik kecantikan, lab, fisioterapi",
    businessSummaryPlaceholder: "Contoh: Klinik ini membantu pendaftaran, info layanan, jadwal dokter, estimasi biaya umum, dan alur booking. AI tidak boleh memberi diagnosis, resep, dosis obat, atau saran medis personal.",
    productsLabel: "Layanan administratif",
    productsPlaceholder: "Contoh: konsultasi dokter, scaling gigi, facial treatment, cek lab.",
    adminLabel: "Kapan harus panggil admin",
    adminPlaceholder: "Contoh: keluhan medis personal, kondisi darurat, hasil lab, komplain layanan, ubah jadwal dokter, atau data pasien sensitif.",
    adminOptions: ["Keluhan medis personal", "Kondisi darurat", "Hasil lab atau rekam medis", "Komplain layanan", "Perubahan jadwal dokter", "Data pasien sensitif"],
    customerPlaceholder: "Contoh: pasien baru, pasien kontrol, keluarga pasien",
    capabilityTitle: "Kemampuan yang aktif",
    capabilityHint: "AI membantu sisi administratif dan menjaga batas aman klinik.",
  },
  education: {
    setupTitle: "Isi program dan pendaftaran",
    setupHint: "Setelah selesai, AI bisa jelaskan program, kumpulkan minat calon peserta, dan bantu jadwalkan konsultasi.",
    businessTypePlaceholder: "Contoh: Kursus bahasa, bootcamp, bimbel, kelas hobi",
    businessSummaryPlaceholder: "Contoh: Kursus ini membantu calon peserta memilih program belajar sesuai tujuan dan level. Customer biasanya tanya program, jadwal, harga, metode belajar, sertifikat, dan cara daftar.",
    productsLabel: "Program utama",
    productsPlaceholder: "Contoh: kelas beginner, kelas private, bootcamp 3 bulan, konsultasi gratis.",
    adminLabel: "Kapan harus panggil admin",
    adminPlaceholder: "Contoh: diskon khusus, beasiswa, refund, pindah jadwal, komplain pengajar, atau calon peserta siap daftar.",
    adminOptions: ["Diskon khusus atau beasiswa", "Refund", "Pindah jadwal kelas", "Komplain pengajar", "Calon peserta siap mendaftar", "Customer minta bicara dengan admin"],
    customerPlaceholder: "Contoh: pelajar, mahasiswa, karyawan, orang tua murid",
    capabilityTitle: "Kemampuan yang aktif",
    capabilityHint: "AI membantu tanya kebutuhan dan menyiapkan follow-up calon peserta.",
  },
  property: {
    setupTitle: "Isi listing dan prospek",
    setupHint: "Setelah selesai, AI bisa jawab info properti, kumpulkan kebutuhan prospek, dan bantu jadwalkan survey.",
    businessTypePlaceholder: "Contoh: Agen properti, developer, sewa apartemen",
    businessSummaryPlaceholder: "Contoh: Bisnis ini membantu customer mencari rumah, apartemen, tanah, atau sewa properti. Customer biasanya tanya lokasi, harga, fasilitas, legalitas umum, dan jadwal survey.",
    productsLabel: "Listing / layanan utama",
    productsPlaceholder: "Contoh: rumah siap huni, apartemen studio, tanah kavling, survey lokasi.",
    adminLabel: "Kapan harus panggil admin",
    adminPlaceholder: "Contoh: negosiasi harga, legalitas, KPR, booking fee, dokumen, survey lokasi, atau prospek serius.",
    adminOptions: ["Negosiasi harga", "Legalitas atau dokumen", "KPR atau pembiayaan", "Booking fee", "Jadwal survey lokasi", "Prospek serius siap lanjut"],
    customerPlaceholder: "Contoh: pembeli rumah pertama, investor, penyewa, keluarga",
    capabilityTitle: "Kemampuan yang aktif",
    capabilityHint: "AI mengumpulkan kebutuhan prospek dan menyiapkan follow-up sales.",
  },
  "travel-hospitality": {
    setupTitle: "Isi paket dan reservasi",
    setupHint: "Setelah selesai, AI bisa jelaskan paket, tanya tanggal, dan bantu draft reservasi.",
    businessTypePlaceholder: "Contoh: Hotel, villa, tour, travel, venue",
    businessSummaryPlaceholder: "Contoh: Bisnis ini menerima reservasi kamar, villa, tour, atau paket perjalanan. Customer biasanya tanya tanggal tersedia, fasilitas, harga, kapasitas, itinerary, dan cara booking.",
    productsLabel: "Paket / layanan utama",
    productsPlaceholder: "Contoh: kamar deluxe, paket tour 3D2N, sewa villa, private trip.",
    adminLabel: "Kapan harus panggil admin",
    adminPlaceholder: "Contoh: refund, reschedule, overbooking, pembayaran, reservasi grup besar, atau permintaan khusus.",
    adminOptions: ["Refund", "Reschedule reservasi", "Overbooking", "Konfirmasi pembayaran", "Reservasi grup besar", "Permintaan khusus"],
    customerPlaceholder: "Contoh: keluarga, pasangan, rombongan kantor, wisatawan",
    capabilityTitle: "Kemampuan yang aktif",
    capabilityHint: "AI membantu cek kebutuhan reservasi dan menyiapkan follow-up.",
  },
  "b2b-sales": {
    setupTitle: "Isi layanan dan proses sales",
    setupHint: "Setelah selesai, AI bisa jelaskan layanan, discovery kebutuhan prospek, dan arahkan ke demo/follow-up.",
    businessTypePlaceholder: "Contoh: SaaS, agency, konsultan, software house",
    businessSummaryPlaceholder: "Contoh: Bisnis ini membantu perusahaan menyelesaikan masalah operasional lewat layanan B2B. Prospek biasanya tanya fitur, paket, proses kerja, studi kasus, demo, dan proposal.",
    productsLabel: "Layanan / paket utama",
    productsPlaceholder: "Contoh: paket implementasi, konsultasi, managed service, demo produk.",
    adminLabel: "Kapan harus panggil sales/admin",
    adminPlaceholder: "Contoh: request proposal, negosiasi harga, integrasi teknis, kontrak, enterprise deal, atau prospek siap demo.",
    adminOptions: ["Request proposal", "Negosiasi harga atau kontrak", "Integrasi teknis", "Enterprise deal", "Prospek siap demo", "Butuh keputusan manusia"],
    customerPlaceholder: "Contoh: founder, owner, manager operasional, procurement",
    capabilityTitle: "Kemampuan yang aktif",
    capabilityHint: "AI mengumpulkan kebutuhan prospek dan menyiapkan follow-up sales.",
  },
  custom: {
    setupTitle: "Isi kebutuhan bisnis",
    setupHint: "Setelah selesai, AI punya konteks dasar dan aturan kapan harus panggil admin.",
    businessTypePlaceholder: "Contoh: Jasa custom, komunitas, bisnis lokal",
    businessSummaryPlaceholder: "Ceritakan bisnis, hal yang sering ditanyakan customer, layanan/produk utama, dan kapan AI harus minta bantuan admin.",
    productsLabel: "Produk / layanan utama",
    productsPlaceholder: "Contoh: layanan utama, paket paling sering ditanya, produk unggulan.",
    adminLabel: "Kapan harus panggil admin",
    adminPlaceholder: "Contoh: pembayaran, komplain, refund, permintaan khusus, atau info belum pasti.",
    adminOptions: ["Pembayaran", "Komplain", "Refund", "Permintaan khusus", "Informasi belum pasti", "Customer minta bicara dengan admin"],
    customerPlaceholder: "Contoh: customer baru, pelanggan lama, reseller, komunitas",
    capabilityTitle: "Kemampuan yang aktif",
    capabilityHint: "AI memakai konteks bisnis dan aturan aman dari setup ini.",
  },
};

function setupCopyForTemplate(templateKey) {
  return templateSetupCopy[templateKey] || {
    setupTitle: "Isi bisnis dan layanan",
    setupHint: "Setelah selesai, AI bisa menjawab dari konteks bisnis dan mengarahkan customer ke admin saat perlu.",
    businessTypePlaceholder: "Contoh: toko online, jasa, kursus, klinik admin, travel, B2B",
    businessSummaryPlaceholder: "Ceritakan bisnis, hal yang sering ditanyakan customer, produk/layanan utama, dan kapan AI harus minta bantuan admin.",
    productsLabel: "Produk / layanan utama",
    productsPlaceholder: "Contoh: produk unggulan, layanan utama, paket populer.",
    adminLabel: "Kapan harus panggil admin",
    adminPlaceholder: "Contoh: refund, pembayaran, komplain, perubahan data, atau informasi belum pasti.",
    adminOptions: ["Refund atau pembayaran", "Komplain", "Perubahan data atau pesanan", "Informasi belum pasti", "Customer minta bicara dengan admin"],
    customerPlaceholder: "Contoh: customer baru, pelanggan lama, reseller, calon peserta",
    capabilityTitle: "Kemampuan yang aktif",
    capabilityHint: "AI memakai konteks bisnis dan alat yang cocok dengan template.",
  };
}
const agentMemoryRule = {
  field: "customerMemoryEnabled",
  label: "Ingat pelanggan",
  helper: "AI mengingat preferensi pelanggan antar chat. Default mati supaya biaya hemat.",
  checked: (form) => form.customerMemoryEnabled === true,
};

const memoryLimitOptions = {
  items: [3, 5, 8, 10],
  chars: [400, 800, 1200],
  retention: [90, 180, 365, 730],
};

function customerMemoryBundle(enabled) {
  const active = enabled === true;
  return {
    customerMemoryEnabled: active,
    customerMemoryAutoSaveEnabled: active,
    customerMemoryAdminNotesEnabled: active,
    customerMemoryAiExtractionEnabled: active,
    customerMemoryVerifierEnabled: true,
    customerMemoryLlmValidatorEnabled: false,
  };
}

function modelLabel(modelName) {
  const id = modelName || "openai/gpt-4o-mini";
  return defaultModelAliases[id] || id.replace("openai/", "");
}

function modelCardCopy(modelId) {
  const normalizedId = String(modelId || "").toLowerCase();
  if (normalizedId.includes("deepseek")) {
    return {
      badge: "Advance",
      note: "Alternatif efisien untuk percakapan panjang, FAQ, dan analisis instruksi dengan biaya tetap terkontrol.",
    };
  }
  if (normalizedId.includes("opus") || normalizedId.includes("pro")) {
    return {
      badge: "Premium",
      note: "Untuk kasus bernilai tinggi yang butuh reasoning lebih dalam dan toleransi biaya lebih besar.",
    };
  }
  if (normalizedId === "openai/gpt-4.1-mini" || normalizedId.includes("gpt-4.1") || normalizedId.includes("sonnet") || normalizedId.includes("gpt-5")) {
    return {
      badge: "Advance",
      note: "Lebih kuat untuk konteks panjang, instruksi detail, dan kasus customer yang butuh penalaran lebih teliti.",
    };
  }
  return {
    badge: "Hemat",
    note: "Efisien untuk percakapan customer harian, FAQ, follow-up ringan, dan operasional yang sudah punya data jelas.",
  };
}

function updateCardSpotlight(event) {
  const rect = event.currentTarget.getBoundingClientRect();
  event.currentTarget.style.setProperty("--mx", `${event.clientX - rect.left}px`);
  event.currentTarget.style.setProperty("--my", `${event.clientY - rect.top}px`);
}

function ModelChoiceCards({ value, onChange, disabled = false, compact = false, models = fallbackOpenAIModels }) {
  const selectedValue = value || "openai/gpt-4o-mini";
  const visibleModels = models.length ? models : fallbackOpenAIModels.filter((model) => model.available !== false);
  return (
    <div className={`agent-model-cards ${compact ? "compact" : ""}`} role="radiogroup" aria-label="Pilihan model AI">
      {visibleModels.map((model) => {
        const isSelected = selectedValue === model.id;
        const copy = modelCardCopy(model.id);
        return (
          <button
            key={model.id}
            className={`agent-model-card ${isSelected ? "active" : ""}`}
            type="button"
            role="radio"
            aria-checked={isSelected}
            onClick={() => onChange?.(model.id)}
            onMouseMove={updateCardSpotlight}
            disabled={disabled}
          >
            <span className="agent-model-card-head">
              <strong>{model.name || modelLabel(model.id)}</strong>
              <span>{copy.badge}</span>
            </span>
            <em>{copy.note}</em>
          </button>
        );
      })}
    </div>
  );
}

function agentFormFrom(agent = {}) {
  const memoryBundle = customerMemoryBundle(agent.customerMemoryEnabled === true);
  return {
    id: agent.id || "",
    name: agent.name || "",
    modelName: agent.modelName || "openai/gpt-4o-mini",
    systemPrompt: agent.systemPrompt || "",
    escalationPrompt: agent.escalationPrompt || "",
    fallbackWaitingMessage: agent.fallbackWaitingMessage || fallbackMessage,
    allowClarification: agent.allowClarification !== false,
    maxClarificationCount: Number(agent.maxClarificationCount ?? 1),
    answerOnlyFromKnowledge: agent.answerOnlyFromKnowledge !== false,
    dontBroadenTopic: agent.dontBroadenTopic !== false,
    forbidPromises: agent.forbidPromises !== false,
    forbidSensitiveAnswers: agent.forbidSensitiveAnswers !== false,
    requireActionConfirmation: agent.requireActionConfirmation !== false,
    escalateLowConfidence: agent.escalateLowConfidence !== false,
    guideNextStep: agent.guideNextStep !== false,
    conciseResponse: agent.conciseResponse !== false,
    allowAutoUpdateContactName: agent.allowAutoUpdateContactName !== false,
    onlyFillNameIfEmpty: agent.onlyFillNameIfEmpty !== false,
    ...memoryBundle,
    customerMemoryMaxItems: Number(agent.customerMemoryMaxItems ?? 5),
    customerMemoryMaxChars: Number(agent.customerMemoryMaxChars ?? 800),
    customerMemoryRetentionDays: Number(agent.customerMemoryRetentionDays ?? 180),
    isActive: agent.isActive !== false,
  };
}

function agentComparable(form = {}) {
  const memoryBundle = customerMemoryBundle(form.customerMemoryEnabled === true);
  return {
    name: form.name || "",
    modelName: form.modelName || "openai/gpt-4o-mini",
    systemPrompt: form.systemPrompt || "",
    escalationPrompt: form.escalationPrompt || "",
    fallbackWaitingMessage: form.fallbackWaitingMessage || fallbackMessage,
    allowClarification: form.allowClarification !== false,
    maxClarificationCount: Number(form.maxClarificationCount ?? 1),
    answerOnlyFromKnowledge: form.answerOnlyFromKnowledge !== false,
    dontBroadenTopic: form.dontBroadenTopic !== false,
    forbidPromises: form.forbidPromises !== false,
    forbidSensitiveAnswers: form.forbidSensitiveAnswers !== false,
    requireActionConfirmation: form.requireActionConfirmation !== false,
    escalateLowConfidence: form.escalateLowConfidence !== false,
    guideNextStep: form.guideNextStep !== false,
    conciseResponse: form.conciseResponse !== false,
    allowAutoUpdateContactName: form.allowAutoUpdateContactName !== false,
    onlyFillNameIfEmpty: form.onlyFillNameIfEmpty !== false,
    ...memoryBundle,
    customerMemoryMaxItems: Number(form.customerMemoryMaxItems ?? 5),
    customerMemoryMaxChars: Number(form.customerMemoryMaxChars ?? 800),
    customerMemoryRetentionDays: Number(form.customerMemoryRetentionDays ?? 180),
    isActive: form.isActive !== false,
  };
}

function isBusinessToolInstalled(tool) {
  return Boolean(tool?.installed ?? tool?.enabled);
}

function hasBusinessCatalogTools(businessTools = []) {
  return businessTools.some((tool) => ["commerce", "booking"].includes(tool?.key) && isBusinessToolInstalled(tool));
}

function installedBusinessToolDataCount(businessTools = [], counts = {}) {
  return businessTools
    .filter(isBusinessToolInstalled)
    .reduce((sum, tool) => sum + (Number(counts?.[tool.key]) || 0), 0);
}

function sessionStatusLabel(status) {
  if (status === "connected") return "Terhubung";
  if (status === "qr_pending") return "Menunggu QR";
  if (status === "disconnected") return "Terputus";
  return status || "Terputus";
}

function AgentAvatar({ large = false }) {
  return (
    <span className={`agent-avatar ${large ? "agent-avatar-lg" : ""}`} aria-hidden="true">
      {Icons.robot}
    </span>
  );
}

function AgentStatCard({ icon, label, value, tone = "blue" }) {
  return (
    <div className={`agent-stat-card ${tone}`}>
      <span className="agent-stat-icon">{icon}</span>
      <span>
        <small>{label}</small>
        <strong>{value}</strong>
      </span>
    </div>
  );
}

function ReadinessRow({ label, state = "ok" }) {
  return (
    <div className={`agent-readiness-row ${state}`}>
      <span>{state === "ok" ? Icons.check : Icons.alert}</span>
      <p>{label}</p>
    </div>
  );
}

function SummaryRow({ icon, label, value }) {
  return (
    <div className="agent-summary-row">
      <span>{icon}</span>
      <p>{label}</p>
      <strong>{value}</strong>
    </div>
  );
}

function AgentRuleToggle({ label, helper, checked, onChange, disabled = false }) {
  return (
    <label className={`agent-rule-toggle ${disabled ? "disabled" : ""}`}>
      <span>
        <strong>{label}</strong>
        {helper ? <small>{helper}</small> : null}
      </span>
      <span className="toggle">
        <input
          type="checkbox"
          checked={Boolean(checked)}
          onChange={(event) => onChange?.(event.target.checked)}
          disabled={disabled}
        />
        <span className="toggle-slider" />
      </span>
    </label>
  );
}

function agentSetupQuestion(step, selectedTemplate) {
  const questions = {
    business: {
      title: "Ceritain sedikit soal bisnismu.",
      body: "Cukup seperti ngobrol biasa. Contoh: saya punya salon perempuan dan customer sering tanya layanan, harga, dan jadwal.",
    },
    template: {
      title: `Dari ceritamu, setup ${selectedTemplate?.title || "CS Utama"} paling mendekati.`,
      body: "Ini hanya titik awal. Cara jawab dan semua pengaturan masih bisa diubah nanti.",
    },
    "business-name": {
      title: "Nama bisnismu apa?",
      body: "Nama ini akan dipakai AI saat memperkenalkan bisnis ke customer.",
    },
    "agent-name": {
      title: "AI CS ini mau dipanggil apa di dashboard?",
      body: "Customer tidak harus melihat nama ini. Kamu bisa menggantinya kapan saja.",
    },
    details: {
      title: "Produk, layanan, atau pertanyaan apa yang paling sering ditanyakan customer?",
      body: "Jawab singkat saja. Detail harga, stok, dan jadwal bisa ditambahkan dari Pengetahuan atau Alat Bisnis setelah agent dibuat.",
    },
    handoff: {
      title: "Kapan AI harus berhenti menjawab dan memanggil admin?",
      body: "Pilih semua kondisi yang penting. Kondisi umum sudah dipilih berdasarkan jenis bisnismu.",
    },
    tools: {
      title: "Pekerjaan apa yang boleh dibantu AI?",
      body: "Semua alat dimulai sebagai draf. Admin tetap memeriksa sebelum data dianggap final.",
    },
    review: {
      title: "Sudah sesuai?",
      body: "Periksa ringkasannya. Setelah dibuat, kamu tetap bisa mengubah cara jawab, sumber informasi, dan alat bisnis.",
    },
  };
  return questions[step] || questions.business;
}

function agentSetupAnswer(step, { businessDetails, newAgentName, selectedTemplate, selectedSetupToolKeys, setupToolOptions }) {
  const details = businessDetails || {};
  const toolNames = setupToolOptions
    .filter((tool) => selectedSetupToolKeys.includes(tool.key))
    .map((tool) => tool.name || tool.navLabel || tool.key);
  const answers = {
    business: details.businessSummary,
    template: selectedTemplate?.title,
    "business-name": details.businessName,
    "agent-name": newAgentName,
    details: details.productsServices || "Lewati dulu",
    handoff: [...(details.adminConditions || []), details.adminOther].filter(Boolean).join(", "),
    tools: toolNames.length ? toolNames.join(", ") : "Tanpa alat bisnis dulu",
  };
  return String(answers[step] || "").trim();
}

function AgentSetupDraftSummary({
  currentStep,
  businessDetails,
  newAgentName,
  selectedTemplate,
  selectedSetupToolKeys,
  setupToolOptions,
  onEditStep,
}) {
  const details = businessDetails || {};
  const currentStepIndex = AGENT_SETUP_STEP_ORDER.indexOf(currentStep);
  const toolNames = setupToolOptions
    .filter((tool) => selectedSetupToolKeys.includes(tool.key))
    .map((tool) => tool.name || tool.navLabel || tool.key);
  const rows = [
    { label: "Bisnis", value: details.businessName || "Belum diisi", step: "business-name" },
    { label: "Jenis setup", value: selectedTemplate?.title || "CS Utama", step: "template" },
    { label: "Nama AI CS", value: newAgentName || "Belum diisi", step: "agent-name" },
    { label: "Produk atau layanan", value: details.productsServices || "Bisa dilengkapi nanti", step: "details" },
    { label: "Alat bisnis", value: toolNames.length ? toolNames.join(", ") : "Tidak ada", step: "tools" },
  ];

  return (
    <div className="agent-setup-draft-summary">
      <div className="agent-setup-draft-head">
        <span>Draft agent</span>
        <strong>{newAgentName || selectedTemplate?.defaultName || "AI CS baru"}</strong>
      </div>
      <div className="agent-setup-draft-list">
        {rows.map((row) => (
          <div key={row.label}>
            <span><small>{row.label}</small><strong>{row.value}</strong></span>
            {AGENT_SETUP_STEP_ORDER.indexOf(row.step) <= currentStepIndex ? (
              <button type="button" className="icon-btn" title={`Ubah ${row.label}`} aria-label={`Ubah ${row.label}`} onClick={() => onEditStep(row.step)}>
                {Icons.edit}
              </button>
            ) : <span aria-hidden="true" />}
          </div>
        ))}
      </div>
      <p>{Icons.lock} Tool tetap mode draf dan keputusan akhir ada di admin.</p>
    </div>
  );
}

function ConversationalAgentSetup({
  step,
  reply,
  setReply,
  selectedTemplate,
  newAgentName,
  businessDetails,
  selectedSetupToolKeys,
  setupToolOptions,
  setupCopy,
  canEditAgent,
  isNewAgentReady,
  isSavingAgent,
  onSelectTemplate,
  onConfirmTemplate,
  onEditStep,
  onBack,
  onCancel,
  onSkipDetails,
  onToggleAdminCondition,
  onUpdateAdminOther,
  onToggleTool,
  onContinue,
  onCreate,
}) {
  const messagesRef = useRef(null);
  const progress = getAgentSetupProgress(step);
  const stepIndex = AGENT_SETUP_STEP_ORDER.indexOf(step);
  const completedSteps = AGENT_SETUP_STEP_ORDER.slice(0, Math.max(stepIndex, 0)).filter((item) => item !== "review");
  const question = agentSetupQuestion(step, selectedTemplate);
  const details = businessDetails || {};
  const isTextStep = ["business", "business-name", "agent-name", "details"].includes(step);
  const replyPlaceholder = {
    business: "Contoh: Saya punya toko skincare online...",
    "business-name": "Ketik nama bisnis",
    "agent-name": "Contoh: CS Cantika",
    details: "Contoh: layanan, harga, jam buka, cara pesan...",
  }[step];
  const replyLabel = {
    business: "Cerita bisnis",
    "business-name": "Nama bisnis",
    "agent-name": "Nama AI CS",
    details: "Produk, layanan, atau pertanyaan customer",
  }[step];

  useEffect(() => {
    if (!messagesRef.current) return;
    messagesRef.current.scrollTop = messagesRef.current.scrollHeight;
  }, [step]);

  return (
    <div className={`agent-chat-first-layout step-${step}`}>
      <section className="agent-setup-chat" aria-label="Percakapan membuat AI CS">
        <div className="agent-setup-progress" aria-label={`Langkah ${progress.current} dari ${progress.total}`}>
          <span>Langkah {progress.current} dari {progress.total}</span>
          <div><i style={{ width: `${(progress.current / progress.total) * 100}%` }} /></div>
          <small>Satu pertanyaan setiap langkah</small>
        </div>

        <div className="agent-setup-messages" aria-live="polite" ref={messagesRef}>
          <div className="agent-setup-message assistant intro">
            <span className="agent-conversation-avatar" aria-hidden="true">{Icons.robot}</span>
            <div>
              <strong>Halo, kita bikin AI CS-nya bareng ya.</strong>
              <p>Kamu cukup jawab seperti ngobrol biasa. Aku yang susun pengaturannya.</p>
            </div>
          </div>

          {completedSteps.map((completedStep) => {
            const completedQuestion = agentSetupQuestion(completedStep, selectedTemplate);
            const answer = agentSetupAnswer(completedStep, {
              businessDetails: details,
              newAgentName,
              selectedTemplate,
              selectedSetupToolKeys,
              setupToolOptions,
            });
            return (
              <div className="agent-setup-exchange" key={completedStep}>
                <div className="agent-setup-message assistant compact"><div><p>{completedQuestion.title}</p></div></div>
                <div className="agent-setup-message customer">
                  <div><p>{answer || "Belum diisi"}</p></div>
                  <button type="button" className="icon-btn" title="Ubah jawaban" aria-label={`Ubah jawaban: ${completedQuestion.title}`} onClick={() => onEditStep(completedStep)}>
                    {Icons.edit}
                  </button>
                </div>
              </div>
            );
          })}

          <div className="agent-setup-message assistant current">
            <span className="agent-conversation-avatar" aria-hidden="true">{Icons.robot}</span>
            <div><strong>{question.title}</strong><p>{question.body}</p></div>
          </div>

          {step === "template" ? (
            <div className="agent-setup-choice-panel">
              <button type="button" className="agent-template-suggestion" onClick={onConfirmTemplate} disabled={!canEditAgent || isSavingAgent}>
                <span className="agent-template-icon">{Icons[selectedTemplate.icon] || Icons.robot}</span>
                <span><small>Setup yang disarankan</small><strong>{selectedTemplate.title}</strong><em>{selectedTemplate.description}</em></span>
                {Icons.arrowRight}
              </button>
              <details className="agent-setup-other-templates">
                <summary>Pilih jenis bisnis lain</summary>
                <div>
                  {agentBusinessTemplates.filter((template) => template.key !== selectedTemplate.key).map((template) => (
                    <button type="button" key={template.key} onClick={() => onSelectTemplate(template.key)} disabled={!canEditAgent || isSavingAgent}>
                      <span className="agent-template-icon">{Icons[template.icon] || Icons.robot}</span>
                      <span><strong>{template.title}</strong><small>{template.description}</small></span>
                    </button>
                  ))}
                </div>
              </details>
            </div>
          ) : null}

          {step === "handoff" ? (
            <div className="agent-setup-choice-panel">
              <div className="agent-setup-chip-grid" role="group" aria-label="Kondisi panggil admin">
                {(setupCopy.adminOptions || []).map((option) => {
                  const selected = (details.adminConditions || []).includes(option);
                  return (
                    <button type="button" className={selected ? "selected" : ""} aria-pressed={selected} key={option} onClick={() => onToggleAdminCondition(option)} disabled={!canEditAgent || isSavingAgent}>
                      {selected ? Icons.check : Icons.plus}{option}
                    </button>
                  );
                })}
              </div>
              <label className="agent-setup-inline-field">
                <span>Kondisi lain <small>Opsional</small></span>
                <input className="form-input" value={details.adminOther || ""} onChange={(event) => onUpdateAdminOther(event.target.value)} placeholder="Contoh: pesanan di atas Rp5 juta" disabled={!canEditAgent || isSavingAgent} />
              </label>
            </div>
          ) : null}

          {step === "tools" ? (
            <div className="agent-setup-choice-panel">
              {setupToolOptions.length ? (
                <div className="agent-setup-tool-grid" role="group" aria-label="Alat bisnis untuk agent">
                  {setupToolOptions.map((tool) => {
                    const selected = selectedSetupToolKeys.includes(tool.key);
                    return (
                      <button type="button" className={selected ? "selected" : ""} aria-pressed={selected} key={tool.key} onClick={() => onToggleTool(tool.key)} disabled={!canEditAgent || isSavingAgent}>
                        <span>{selected ? Icons.check : Icons.plus}</span>
                        <span><strong>{tool.name || tool.navLabel || tool.key}</strong><small>{selected ? "Aktif sebagai draf" : "Tidak dipakai"}</small></span>
                      </button>
                    );
                  })}
                </div>
              ) : (
                <div className="agent-tool-empty-state">{Icons.info}<span><strong>Belum ada alat bisnis yang tersedia.</strong><small>Agent tetap bisa dibuat dan menjawab dari sumber informasi bisnis.</small></span></div>
              )}
            </div>
          ) : null}

          {step === "review" ? (
            <div className="agent-setup-review">
              <AgentSetupDraftSummary
                currentStep={step}
                businessDetails={details}
                newAgentName={newAgentName}
                selectedTemplate={selectedTemplate}
                selectedSetupToolKeys={selectedSetupToolKeys}
                setupToolOptions={setupToolOptions}
                onEditStep={onEditStep}
              />
              <div className="agent-setup-review-ready">{Icons.check}<span><strong>Siap dibuat</strong><small>Aturan aman dan pesan bantuan admin ikut disiapkan otomatis.</small></span></div>
            </div>
          ) : null}
        </div>

        {isTextStep ? (
          <div className="agent-setup-composer">
            <label>
              <span className="sr-only">{replyLabel}</span>
              {step === "business-name" || step === "agent-name" ? (
                <input className="form-input" value={reply} onChange={(event) => setReply(event.target.value)} placeholder={replyPlaceholder} aria-label={replyLabel} autoFocus disabled={!canEditAgent || isSavingAgent} />
              ) : (
                <textarea className="form-textarea" value={reply} onChange={(event) => setReply(event.target.value)} placeholder={replyPlaceholder} aria-label={replyLabel} rows={3} autoFocus disabled={!canEditAgent || isSavingAgent} />
              )}
            </label>
            <div className="agent-setup-composer-actions">
              <button className="btn btn-secondary" type="button" onClick={step === "business" ? onCancel : onBack} disabled={isSavingAgent}>{step === "business" ? "Batal" : "Kembali"}</button>
              {step === "details" ? <button className="btn btn-secondary" type="button" onClick={onSkipDetails} disabled={isSavingAgent}>Lewati</button> : null}
              <button className="btn btn-primary" type="submit" disabled={!canEditAgent || isSavingAgent || !String(reply || "").trim()}>{Icons.send} Kirim</button>
            </div>
          </div>
        ) : (
          <div className="agent-setup-action-bar">
            <button className="btn btn-secondary" type="button" onClick={onBack} disabled={isSavingAgent}>Kembali</button>
            {step === "template" ? null : step === "review" ? (
              <button className="btn btn-primary" type="button" onClick={onCreate} disabled={!canEditAgent || !isNewAgentReady || isSavingAgent}>{isSavingAgent ? "Membuat..." : "Buat AI CS"}</button>
            ) : (
              <button className="btn btn-primary" type="button" onClick={onContinue} disabled={!canEditAgent || isSavingAgent}>Lanjut</button>
            )}
          </div>
        )}
      </section>

      <aside className="agent-setup-draft-desktop" aria-label="Ringkasan draft agent">
        <AgentSetupDraftSummary
          currentStep={step}
          businessDetails={details}
          newAgentName={newAgentName}
          selectedTemplate={selectedTemplate}
          selectedSetupToolKeys={selectedSetupToolKeys}
          setupToolOptions={setupToolOptions}
          onEditStep={onEditStep}
        />
      </aside>
      {step !== "review" ? (
        <details className="agent-setup-draft-mobile">
          <summary>Lihat draft agent</summary>
          <AgentSetupDraftSummary
            currentStep={step}
            businessDetails={details}
            newAgentName={newAgentName}
            selectedTemplate={selectedTemplate}
            selectedSetupToolKeys={selectedSetupToolKeys}
            setupToolOptions={setupToolOptions}
            onEditStep={onEditStep}
          />
        </details>
      ) : null}
    </div>
  );
}

export function emptyAIAgentForm() {
  return agentFormFrom({});
}

function agentTemplateByKey(key) {
  return agentBusinessTemplates.find((template) => template.key === key) || agentBusinessTemplates[0];
}

function formFromAgentTemplate(template, current = {}) {
  return {
    ...emptyAIAgentForm(),
    ...current,
    id: "",
    name: template.defaultName || "",
    systemPrompt: template.systemPrompt || "",
    escalationPrompt: template.escalationPrompt || "",
    fallbackWaitingMessage: template.fallbackWaitingMessage || fallbackMessage,
    allowClarification: true,
    maxClarificationCount: 1,
    answerOnlyFromKnowledge: true,
    dontBroadenTopic: true,
    forbidPromises: true,
    forbidSensitiveAnswers: true,
    requireActionConfirmation: true,
    escalateLowConfidence: true,
    guideNextStep: true,
    conciseResponse: true,
    allowAutoUpdateContactName: template.rules?.allowAutoUpdateContactName !== false,
    onlyFillNameIfEmpty: true,
    ...customerMemoryBundle(false),
    customerMemoryMaxItems: 5,
    customerMemoryMaxChars: 800,
    customerMemoryRetentionDays: 180,
    isActive: true,
  };
}

function defaultBusinessDetails(template) {
  return {
    businessName: "",
    businessType: template?.category || "",
    businessSummary: "",
    productsServices: "",
    targetCustomers: "",
    adminConditions: [...(setupCopyForTemplate(template?.key).adminOptions || [])],
    adminOther: "",
  };
}

function recommendedToolKeysForTemplate(templateKey) {
  const map = {
    ecommerce: ["commerce", "prospects"],
    "booking-service": ["booking", "prospects"],
    "clinic-admin": ["booking"],
    education: ["booking", "prospects"],
    property: ["prospects", "booking"],
    "travel-hospitality": ["booking", "prospects"],
    "b2b-sales": ["prospects", "booking"],
  };
  return map[templateKey] || ["prospects"];
}

function businessToolSetupMode(toolKey) {
  if (["commerce", "booking", "prospects"].includes(toolKey)) return "draft";
  return "read";
}

function defaultSetupToolModes(templateKey = "general-cs") {
  return Object.fromEntries(recommendedToolKeysForTemplate(templateKey).map((key) => [key, businessToolSetupMode(key)]));
}

function defaultToolSetupDetails(template) {
  return {
    commerceProducts: [{ sku: "", name: "", description: "", unitPrice: "", stockQuantity: "", lowStockThreshold: "" }],
    commerceFulfillmentTypes: ["pickup", "delivery", "shipping"],
    bookingServiceName: "",
    bookingDurationMinutes: "60",
    bookingPrice: "",
    bookingAvailabilityDays: "1,2,3,4,5",
    bookingAvailabilityStart: "09:00",
    bookingAvailabilityEnd: "17:00",
  };
}

function emptySetupCommerceProduct() {
  return { sku: "", name: "", description: "", unitPrice: "", stockQuantity: "", lowStockThreshold: "" };
}

function escapeCSVCell(value) {
  const text = String(value ?? "");
  if (!/[",\n]/.test(text)) return text;
  return `"${text.replaceAll('"', '""')}"`;
}

function parseCSVRows(text) {
  const rows = [];
  let row = [];
  let cell = "";
  let quoted = false;
  const source = String(text || "").replace(/^\uFEFF/, "");
  for (let index = 0; index < source.length; index += 1) {
    const char = source[index];
    const next = source[index + 1];
    if (quoted) {
      if (char === '"' && next === '"') {
        cell += '"';
        index += 1;
      } else if (char === '"') {
        quoted = false;
      } else {
        cell += char;
      }
    } else if (char === '"') {
      quoted = true;
    } else if (char === "," || char === "\t") {
      row.push(cell);
      cell = "";
    } else if (char === "\n") {
      row.push(cell);
      rows.push(row);
      row = [];
      cell = "";
    } else if (char !== "\r") {
      cell += char;
    }
  }
  row.push(cell);
  rows.push(row);
  return rows.filter((item) => item.some((value) => String(value || "").trim()));
}

function productsFromCSV(text) {
  const rows = parseCSVRows(text);
  if (!rows.length) return [];
  const header = rows[0].map((item) => String(item || "").trim().toLowerCase());
  const indexOf = (...names) => header.findIndex((item) => names.includes(item));
  const indexes = {
    sku: indexOf("sku", "kode", "kode produk"),
    name: indexOf("name", "nama", "nama produk", "produk"),
    description: indexOf("description", "deskripsi"),
    unitPrice: indexOf("unit_price", "unitprice", "harga", "harga satuan"),
    stockQuantity: indexOf("stock_quantity", "stockquantity", "stok", "stock", "stok awal"),
    lowStockThreshold: indexOf("low_stock_threshold", "lowstockthreshold", "batas stok rendah", "restok"),
  };
  return rows.slice(1).map((row) => ({
    sku: indexes.sku >= 0 ? row[indexes.sku] || "" : "",
    name: indexes.name >= 0 ? row[indexes.name] || "" : row[0] || "",
    description: indexes.description >= 0 ? row[indexes.description] || "" : "",
    unitPrice: indexes.unitPrice >= 0 ? row[indexes.unitPrice] || "" : "",
    stockQuantity: indexes.stockQuantity >= 0 ? row[indexes.stockQuantity] || "" : "",
    lowStockThreshold: indexes.lowStockThreshold >= 0 ? row[indexes.lowStockThreshold] || "" : "",
  })).filter((item) => String(item.name || "").trim());
}

function composeOperatingNotes(details = {}) {
  const conditions = Array.isArray(details.adminConditions)
    ? details.adminConditions.map((item) => String(item || "").trim()).filter(Boolean)
    : [];
  const other = String(details.adminOther || "").trim();
  const composed = [...conditions, other].filter(Boolean).join("; ");
  return composed || String(details.operatingNotes || "").trim();
}

function normalizeBusinessDetails(details = {}) {
  return {
    businessName: String(details.businessName || "").trim(),
    businessType: String(details.businessType || "").trim(),
    businessSummary: String(details.businessSummary || "").trim(),
    productsServices: String(details.productsServices || "").trim(),
    targetCustomers: String(details.targetCustomers || "").trim(),
    operatingNotes: composeOperatingNotes(details),
  };
}

function buildBusinessContextBlock(details) {
  const normalized = normalizeBusinessDetails(details);
  return [
    "BUSINESS CONTEXT / PROFIL BISNIS",
    `Nama bisnis: ${normalized.businessName}`,
    normalized.businessType ? `Jenis bisnis: ${normalized.businessType}` : "",
    normalized.businessSummary ? `Ringkasan bisnis: ${normalized.businessSummary}` : "",
    normalized.productsServices ? `Produk/layanan utama: ${normalized.productsServices}` : "",
    normalized.targetCustomers ? `Target customer: ${normalized.targetCustomers}` : "",
    normalized.operatingNotes ? `Catatan operasional penting: ${normalized.operatingNotes}` : "",
    "",
    "Instruksi: perlakukan business context ini sebagai konteks resmi agent. Pakai ini sebelum meminta user mengubah Cara Jawab AI manual, dan tetap utamakan Knowledge serta data Alat Bisnis yang aktif untuk fakta operasional terbaru.",
  ].filter(Boolean).join("\n");
}

function enhanceAgentFormWithBusinessDetails(form, details) {
  const businessName = String(details.businessName || "").trim();
  const businessType = String(details.businessType || "").trim();
  const basePrompt = String(form.systemPrompt || "")
    .replaceAll("[ubah bagian ini: nama bisnis]", businessName || "nama bisnis")
    .replace(/Jenis bisnis: \[ubah bagian ini: [^\]]+\]\./, businessType ? `Jenis bisnis: ${businessType}.` : "Jenis bisnis: sesuai business context di bawah.")
    .replace("- Ganti semua bagian bertanda [ubah bagian ini: ...] sebelum agent dipakai produksi.", "- Business context sudah diisi dari setup awal. Lengkapi Knowledge dan Alat Bisnis sebelum agent dipakai produksi.");
  const businessBlock = buildBusinessContextBlock(details);
  return {
    ...form,
    systemPrompt: `${basePrompt}\n\n${businessBlock}`,
  };
}

export function AgentsView({
  aiAgents = [],
  sessions = [],
  aiAgentForm,
  setAIAgentForm,
  saveAIAgent,
  deleteAIAgent,
  activeAIAgentId,
  setActiveAIAgentId,
  agentTab,
  setAgentTab,
  canManageKnowledge,
  knowledgeProps,
  navigateToWhatsApp,
  navigateToPlayground,
  busyKey,
  aiAgentSaveFeedback,
  humanHandoffNotificationRule,
  groupNotificationVariables = [],
  saveGroupNotificationRule,
  sendGroupNotificationTest,
  canManageGroupNotifications,
  isNotificationGroupBound,
  notificationGroupName,
  aiModels = [],
  configureBusinessTools,
}) {
  const [isCreateAgentOpen, setIsCreateAgentOpen] = useState(false);
  const [newAgentForm, setNewAgentForm] = useState(() => emptyAIAgentForm());
  const [selectedTemplateKey, setSelectedTemplateKey] = useState("general-cs");
  const [agentQuery, setAgentQuery] = useState("");
  const [agentStatusFilter, setAgentStatusFilter] = useState("all");
  const [notificationDraft, setNotificationDraft] = useState(() => notificationRuleDraft(humanHandoffNotificationRule));
  const [isNotificationComposerOpen, setIsNotificationComposerOpen] = useState(false);
  const [createAgentStep, setCreateAgentStep] = useState("business");
  const [agentSetupReply, setAgentSetupReply] = useState("");
  const [newAgentBusinessDetails, setNewAgentBusinessDetails] = useState(() => defaultBusinessDetails(agentTemplateByKey("general-cs")));
  const [selectedSetupToolKeys, setSelectedSetupToolKeys] = useState(() => recommendedToolKeysForTemplate("general-cs"));
  const [selectedSetupToolModes, setSelectedSetupToolModes] = useState(() => defaultSetupToolModes("general-cs"));
  const [setupToolDetails, setSetupToolDetails] = useState(() => defaultToolSetupDetails(agentTemplateByKey("general-cs")));
  const [deleteDialogAgent, setDeleteDialogAgent] = useState(null);
  const notificationTemplateRef = useRef(null);

  const activeAgents = aiAgents.filter((agent) => agent.isActive !== false);
  const selectedAgent = activeAIAgentId ? aiAgents.find((agent) => agent.id === activeAIAgentId) || null : null;
  const selectedAgentId = selectedAgent?.id || "";
  const connectedSessions = useMemo(
    () => sessions.filter((session) => session.aiAgentId === selectedAgentId || session.aiAgentName === selectedAgent?.name),
    [sessions, selectedAgentId, selectedAgent?.name]
  );
  const connectedAgentIds = useMemo(() => {
    const ids = new Set();
    sessions.forEach((session) => {
      if (session.aiAgentId) ids.add(session.aiAgentId);
      if (session.aiAgentName) {
        const matched = aiAgents.find((agent) => agent.name === session.aiAgentName);
        if (matched?.id) ids.add(matched.id);
      }
    });
    return ids;
  }, [aiAgents, sessions]);
  const deleteDialogSessions = useMemo(() => {
    if (!deleteDialogAgent) return [];
    return sessions.filter((session) => session.aiAgentId === deleteDialogAgent.id || session.aiAgentName === deleteDialogAgent.name);
  }, [deleteDialogAgent?.id, deleteDialogAgent?.name, sessions]);
  const form = aiAgentForm || emptyAIAgentForm();
  const availableModelOptions = useMemo(() => {
    const normalized = normalizeModelOptions(aiModels, form.modelName || newAgentForm.modelName || "openai/gpt-4o-mini");
    const available = normalized.filter((model) => model.available !== false);
    return available.length ? available : fallbackOpenAIModels.filter((model) => model.available !== false);
  }, [aiModels, form.modelName, newAgentForm.modelName]);
  const selectedTemplate = agentTemplateByKey(selectedTemplateKey);
  const setupCopy = setupCopyForTemplate(selectedTemplate.key);
  const newAgentName = String(newAgentForm.name || "").trim();
  const newBusinessName = String(newAgentBusinessDetails.businessName || "").trim();
  const newBusinessSummary = String(newAgentBusinessDetails.businessSummary || "").trim();
  const isBusinessSetupReady = Boolean(newBusinessName && newBusinessSummary);
  const isNewAgentReady = Boolean(newAgentName && isBusinessSetupReady);
  const setupToolOptions = useMemo(() => {
    const readyTools = (knowledgeProps?.businessTools || []).filter((tool) => ["prospects", "commerce", "booking"].includes(tool?.key) && tool?.status === "ready");
    const byKey = new Map(readyTools.map((tool) => [tool.key, tool]));
    return ["commerce", "booking", "prospects"].map((key) => byKey.get(key)).filter(Boolean);
  }, [knowledgeProps?.businessTools]);
  const selectedAvailableSetupToolKeys = useMemo(() => {
    const availableKeys = new Set(setupToolOptions.map((tool) => tool.key));
    return selectedSetupToolKeys.filter((key) => availableKeys.has(key));
  }, [selectedSetupToolKeys, setupToolOptions]);
  const isCreateReviewStep = createAgentStep === "review";
  const selectedTab = agentTabs.some((tab) => tab.id === agentTab) ? agentTab : "overview";
  const canEditAgent = canManageKnowledge !== false;
  const hasActiveAgent = activeAgents.length > 0;
  const anyWhatsAppConnected = sessions.some((session) => session.status === "connected");
  const isStartGuideComplete = hasActiveAgent && anyWhatsAppConnected;
  const isSavingAgent = busyKey === "ai-agent-save" || busyKey === "ai-agent-create" || busyKey === "agent-business-tools-setup";
  const hasUnsavedChanges = form.id && selectedAgent
    ? JSON.stringify(agentComparable(form)) !== JSON.stringify(agentComparable(selectedAgent))
    : !form.id && Boolean((form.name || "").trim() || (form.systemPrompt || "").trim());
  const businessCatalogTools = (knowledgeProps?.businessTools || []).filter((tool) => ["commerce", "booking"].includes(tool?.key));
  const usesBusinessCatalogTools = hasBusinessCatalogTools(knowledgeProps?.businessTools || []);
  const notificationVariables = groupNotificationVariables.length ? groupNotificationVariables : fallbackNotificationVariables;
  const notificationPreview = renderNotificationPreview(notificationDraft.templateText, notificationVariables);
  const notificationDirty = JSON.stringify(notificationDraft) !== JSON.stringify(notificationRuleDraft(humanHandoffNotificationRule));
  const isNotificationSaving = busyKey === "group-notification-rule-save";
  const isNotificationTesting = busyKey === "group-notification-test";
  const notificationGroupLabel = isNotificationGroupBound ? notificationGroupName || "Grup notifikasi" : "Belum dipilih";
  const notificationPreviewText = notificationPreview.trim() || "Isi pesan belum diatur.";
  const notificationVariableGroups = useMemo(() => {
    return notificationVariables.reduce((groups, item) => {
      const groupName = item.group || "Umum";
      groups[groupName] = [...(groups[groupName] || []), item];
      return groups;
    }, {});
  }, [notificationVariables]);
  const visibleNotificationVariables = useMemo(() => {
    const priorityKeys = [
      "customer.name",
      "customer.phone",
      "conversation.summary",
      "handoff.reason",
      "conversation.link",
      "business.name",
      "conversation.last_message",
      "event.created_at",
    ];
    const byKey = new Map(notificationVariables.map((item) => [item.key, item]));
    const used = new Set();
    const prioritized = priorityKeys.map((key) => byKey.get(key)).filter(Boolean);
    prioritized.forEach((item) => used.add(item.key));
    return [...prioritized, ...notificationVariables.filter((item) => !used.has(item.key))];
  }, [notificationVariables]);

  useEffect(() => {
    setNotificationDraft(notificationRuleDraft(humanHandoffNotificationRule));
    setIsNotificationComposerOpen(false);
  }, [humanHandoffNotificationRule?.isEnabled, humanHandoffNotificationRule?.templateText, humanHandoffNotificationRule?.triggerKey]);

  useEffect(() => {
    if (!isCreateAgentOpen) return undefined;
    function onKeyDown(event) {
      if (event.key === "Escape") setIsCreateAgentOpen(false);
    }
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [isCreateAgentOpen]);

  useEffect(() => {
    if (!deleteDialogAgent) return undefined;
    function onKeyDown(event) {
      if (event.key === "Escape") setDeleteDialogAgent(null);
    }
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [deleteDialogAgent]);

  const businessToolDataCount = installedBusinessToolDataCount(businessCatalogTools, knowledgeProps?.businessToolDataCounts || {});
  const totalKnowledge =
    (knowledgeProps?.faqs?.length || 0) +
    (knowledgeProps?.documents?.length || 0) +
    (usesBusinessCatalogTools ? 0 : (knowledgeProps?.positions?.length || 0));
  const hasUsableKnowledge = totalKnowledge > 0 || businessToolDataCount > 0;
  const selectedKnowledgeProps = {
    ...knowledgeProps,
    embedded: true,
    activeAIAgentId: selectedAgentId,
    setActiveAIAgentId,
    canManageKnowledge,
  };
  const visibleAgents = useMemo(() => {
    const query = agentQuery.trim().toLowerCase();
    return aiAgents.filter((agent) => {
      const statusMatch =
        agentStatusFilter === "all" ||
        (agentStatusFilter === "active" && agent.isActive !== false) ||
        (agentStatusFilter === "inactive" && agent.isActive === false) ||
        (agentStatusFilter === "connected" && connectedAgentIds.has(agent.id));
      const queryMatch = !query || `${agent.name || ""} ${modelLabel(agent.modelName)}`.toLowerCase().includes(query);
      return statusMatch && queryMatch;
    });
  }, [agentQuery, agentStatusFilter, aiAgents, connectedAgentIds]);
  const readiness = [
    { label: "Nama agent sudah diisi", state: String(form.name || "").trim() ? "ok" : "warn" },
    { label: "Cara jawab AI sudah dibuat", state: String(form.systemPrompt || "").trim() ? "ok" : "warn" },
    {
      label: usesBusinessCatalogTools
        ? (businessToolDataCount > 0 ? "Data Alat Bisnis siap dipakai AI" : "Isi data di Alat Bisnis agar AI bisa pakai tool")
        : "Knowledge FAQ/dokumen sudah ditambahkan",
      state: hasUsableKnowledge ? "ok" : "warn",
    },
    { label: "Channel terhubung", state: connectedSessions.length > 0 ? "ok" : "warn" },
    { label: "Aturan panggil admin sudah diatur", state: String(form.escalationPrompt || form.fallbackWaitingMessage || "").trim() ? "ok" : "warn" },
  ];

  function selectAgent(agent) {
    setActiveAIAgentId?.(agent.id);
    setAIAgentForm?.(agentFormFrom(agent));
  }

  function startNewAgent() {
    const template = agentTemplateByKey("general-cs");
    setSelectedTemplateKey(template.key);
    setNewAgentForm(formFromAgentTemplate(template));
    setNewAgentBusinessDetails(defaultBusinessDetails(template));
    setSelectedSetupToolKeys(recommendedToolKeysForTemplate(template.key));
    setSelectedSetupToolModes(defaultSetupToolModes(template.key));
    setSetupToolDetails(defaultToolSetupDetails(template));
    setCreateAgentStep("business");
    setAgentSetupReply("");
    setIsCreateAgentOpen(true);
  }

  function updateField(field, value) {
    setAIAgentForm?.((current) => {
      const base = { ...emptyAIAgentForm(), ...(current || {}) };
      if (field === "customerMemoryEnabled") {
        return { ...base, ...customerMemoryBundle(value) };
      }
      return { ...base, [field]: value };
    });
  }

  function updateNewAgentField(field, value) {
    setNewAgentForm((current) => ({ ...emptyAIAgentForm(), ...(current || {}), id: "", [field]: value }));
  }

  function updateNewAgentBusinessDetail(field, value) {
    setNewAgentBusinessDetails((current) => ({ ...current, [field]: value }));
  }

  function toggleNewAgentAdminCondition(label) {
    setNewAgentBusinessDetails((current) => {
      const list = Array.isArray(current.adminConditions) ? current.adminConditions : [];
      const next = list.includes(label) ? list.filter((item) => item !== label) : [...list, label];
      return { ...current, adminConditions: next };
    });
  }

  function updateSetupToolDetail(field, value) {
    setSetupToolDetails((current) => ({ ...current, [field]: value }));
  }

  function toggleSetupFulfillmentType(value) {
    setSetupToolDetails((current) => {
      const currentTypes = Array.isArray(current.commerceFulfillmentTypes) ? current.commerceFulfillmentTypes : [];
      const nextTypes = currentTypes.includes(value)
        ? currentTypes.filter((item) => item !== value)
        : [...currentTypes, value];
      return { ...current, commerceFulfillmentTypes: nextTypes.length ? nextTypes : currentTypes };
    });
  }

  function toggleSetupTool(toolKey) {
    setSelectedSetupToolKeys((current) => (
      current.includes(toolKey)
        ? current.filter((key) => key !== toolKey)
        : [...current, toolKey]
    ));
    setSelectedSetupToolModes((current) => ({
      ...current,
      [toolKey]: current[toolKey] || businessToolSetupMode(toolKey),
    }));
  }

  function updateSetupCommerceProduct(index, field, value) {
    setSetupToolDetails((current) => {
      const products = Array.isArray(current.commerceProducts) && current.commerceProducts.length
        ? current.commerceProducts
        : [emptySetupCommerceProduct()];
      return {
        ...current,
        commerceProducts: products.map((item, itemIndex) => itemIndex === index ? { ...item, [field]: value } : item),
      };
    });
  }

  function addSetupCommerceProduct() {
    setSetupToolDetails((current) => ({
      ...current,
      commerceProducts: [...(current.commerceProducts || []), emptySetupCommerceProduct()],
    }));
  }

  function removeSetupCommerceProduct(index) {
    setSetupToolDetails((current) => {
      const products = (current.commerceProducts || []).filter((_, itemIndex) => itemIndex !== index);
      return { ...current, commerceProducts: products.length ? products : [emptySetupCommerceProduct()] };
    });
  }

  function downloadProductTemplate() {
    const rows = [
      ["sku", "name", "description", "unit_price", "stock_quantity", "low_stock_threshold"],
      ["KOPI-SUSU-250", "Kopi Susu 250ml", "Kopi susu botol siap minum", "25000", "100", "10"],
      ["HAM-001", "Paket Hampers", "Paket hadiah isi 3 produk", "150000", "25", "5"],
    ];
    const csv = rows.map((row) => row.map(escapeCSVCell).join(",")).join("\n");
    const blob = new Blob([csv], { type: "text/csv;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = "template-produk-oneflow.csv";
    document.body.appendChild(link);
    link.click();
    link.remove();
    URL.revokeObjectURL(url);
  }

  function uploadProductTemplate(event) {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => {
      const products = productsFromCSV(reader.result);
      if (!products.length) return;
      setSetupToolDetails((current) => ({ ...current, commerceProducts: products }));
    };
    reader.readAsText(file);
  }

  function insertNotificationVariable(variableKey) {
    const token = `{{${variableKey}}}`;
    const textarea = notificationTemplateRef.current;
    setNotificationDraft((current) => {
      const currentText = current.templateText || "";
      if (textarea && typeof textarea.selectionStart === "number") {
        const start = textarea.selectionStart;
        const end = textarea.selectionEnd;
        return {
          ...current,
          templateText: `${currentText.slice(0, start)}${token}${currentText.slice(end)}`,
        };
      }
      return {
        ...current,
        templateText: `${currentText}${currentText && !currentText.endsWith("\n") ? " " : ""}${token}`,
      };
    });
    window.requestAnimationFrame(() => notificationTemplateRef.current?.focus());
  }

  function submitNotificationRule(event) {
    event.preventDefault();
    saveGroupNotificationRule?.(notificationDraft);
  }

  function applyAgentTemplate(templateKey, { preserveBusinessDetails = false } = {}) {
    const template = agentTemplateByKey(templateKey);
    const previousTemplate = agentTemplateByKey(selectedTemplateKey);
    setSelectedTemplateKey(template.key);
    setNewAgentBusinessDetails((current) => preserveBusinessDetails
      ? {
          ...defaultBusinessDetails(template),
          ...current,
          businessType: template.category || current.businessType,
          adminConditions: [...(setupCopyForTemplate(template.key).adminOptions || [])],
        }
      : defaultBusinessDetails(template));
    setSelectedSetupToolKeys(recommendedToolKeysForTemplate(template.key));
    setSelectedSetupToolModes(defaultSetupToolModes(template.key));
    setSetupToolDetails(defaultToolSetupDetails(template));
    setNewAgentForm((current) => {
      const next = formFromAgentTemplate(template, { modelName: current?.modelName || "openai/gpt-4o-mini" });
      const hasCustomName = preserveBusinessDetails && current?.name && current.name !== previousTemplate.defaultName;
      return hasCustomName ? { ...next, name: current.name } : next;
    });
  }

  function replyForAgentSetupStep(step) {
    const values = {
      business: newAgentBusinessDetails.businessSummary,
      "business-name": newAgentBusinessDetails.businessName,
      "agent-name": newAgentForm.name,
      details: newAgentBusinessDetails.productsServices,
    };
    return String(values[step] || "");
  }

  function moveToAgentSetupStep(step, nextReply) {
    setCreateAgentStep(step);
    setAgentSetupReply(nextReply === undefined ? replyForAgentSetupStep(step) : String(nextReply || ""));
  }

  function closeAgentSetup() {
    setIsCreateAgentOpen(false);
    setAgentSetupReply("");
  }

  function handleCreateAgentModalSubmit(event) {
    event?.preventDefault?.();
    const value = String(agentSetupReply || "").trim();
    if (!value) return;

    if (createAgentStep === "business") {
      const inferredTemplateKey = inferAgentTemplateKey(value);
      applyAgentTemplate(inferredTemplateKey, { preserveBusinessDetails: true });
      setNewAgentBusinessDetails((current) => ({
        ...current,
        businessSummary: value,
        businessType: agentTemplateByKey(inferredTemplateKey).category,
      }));
      moveToAgentSetupStep("template", "");
    } else if (createAgentStep === "business-name") {
      updateNewAgentBusinessDetail("businessName", value);
      moveToAgentSetupStep("agent-name", newAgentForm.name);
    } else if (createAgentStep === "agent-name") {
      updateNewAgentField("name", value);
      moveToAgentSetupStep("details", newAgentBusinessDetails.productsServices);
    } else if (createAgentStep === "details") {
      updateNewAgentBusinessDetail("productsServices", value);
      moveToAgentSetupStep("handoff", "");
    }
  }

  function confirmSuggestedTemplate() {
    moveToAgentSetupStep("business-name", newAgentBusinessDetails.businessName);
  }

  function selectAgentSetupTemplate(templateKey) {
    applyAgentTemplate(templateKey, { preserveBusinessDetails: true });
    moveToAgentSetupStep("business-name", newAgentBusinessDetails.businessName);
  }

  function editAgentSetupStep(step) {
    moveToAgentSetupStep(step);
  }

  function goBackInAgentSetup() {
    moveToAgentSetupStep(getPreviousAgentSetupStep(createAgentStep));
  }

  function continueAgentSetup() {
    moveToAgentSetupStep(getNextAgentSetupStep(createAgentStep), "");
  }

  function skipAgentSetupDetails() {
    updateNewAgentBusinessDetail("productsServices", "");
    moveToAgentSetupStep("handoff", "");
  }

  async function createAgentFromReview(event) {
    event?.preventDefault?.();
    if (!isCreateReviewStep || !isNewAgentReady || isSavingAgent || !canEditAgent) return false;
    const configuredForm = enhanceAgentFormWithBusinessDetails({ ...emptyAIAgentForm(), ...newAgentForm, id: "" }, newAgentBusinessDetails);
    const saved = await saveAIAgent?.(event, configuredForm);
    if (saved) {
      const toolConfigs = selectedAvailableSetupToolKeys.map((key) => ({ key, aiMode: selectedSetupToolModes[key] || businessToolSetupMode(key) }));
      const toolSetupDetails = {
        ...setupToolDetails,
        commerceFulfillmentTypes: [...(setupToolDetails.commerceFulfillmentTypes || [])],
        commerceProducts: [...(setupToolDetails.commerceProducts || [])],
      };
      setIsCreateAgentOpen(false);
      setNewAgentForm(emptyAIAgentForm());
      setNewAgentBusinessDetails(defaultBusinessDetails(agentTemplateByKey("general-cs")));
      setSelectedSetupToolKeys(recommendedToolKeysForTemplate("general-cs"));
      setSelectedSetupToolModes(defaultSetupToolModes("general-cs"));
      setSetupToolDetails(defaultToolSetupDetails(agentTemplateByKey("general-cs")));
      setCreateAgentStep("business");
      setAgentSetupReply("");
      setAgentTab?.("instructions");
      if (toolConfigs.length) {
        void Promise.resolve(configureBusinessTools?.(toolConfigs, toolSetupDetails)).catch(() => {});
      }
    }
    return Boolean(saved);
  }

  function requestDeleteAgent() {
    if (!selectedAgent) return;
    setDeleteDialogAgent(selectedAgent);
  }

  async function confirmDeleteAgent() {
    if (!deleteDialogAgent || deleteDialogSessions.length) return;
    const deleted = await deleteAIAgent?.(deleteDialogAgent.id);
    if (deleted) setDeleteDialogAgent(null);
  }

  function openWhatsAppFromDeleteFlow() {
    setDeleteDialogAgent(null);
    navigateToWhatsApp?.();
  }

  const createAgentModal = isCreateAgentOpen ? (
    <div className="modal-backdrop agent-create-backdrop" role="presentation" onMouseDown={(event) => {
      if (event.target === event.currentTarget && !isSavingAgent) closeAgentSetup();
    }}>
      <form
        className="modal-card agent-create-modal agent-chat-first-modal"
        onSubmit={handleCreateAgentModalSubmit}
        role="dialog"
        aria-modal="true"
        aria-labelledby="agent-create-title"
        aria-describedby="agent-create-description"
      >
        <div className="modal-header agent-create-header">
          <div className="agent-create-title-block">
            <span className="agent-create-orb" aria-hidden="true">{Icons.robot}</span>
            <div>
              <span className="agent-create-kicker">Buat lewat percakapan</span>
              <div className="modal-title" id="agent-create-title">Bikin AI CS baru</div>
              <p id="agent-create-description">Jawab seperti ngobrol biasa. Oneflow akan menyusun pengaturannya tanpa menghilangkan kontrolmu.</p>
            </div>
          </div>
          <button className="icon-btn" type="button" aria-label="Tutup" onClick={closeAgentSetup} disabled={isSavingAgent}>
            {Icons.close}
          </button>
        </div>
        <div className="modal-body agent-chat-first-body">
          <ConversationalAgentSetup
            step={createAgentStep}
            reply={agentSetupReply}
            setReply={setAgentSetupReply}
            selectedTemplate={selectedTemplate}
            newAgentName={newAgentName}
            businessDetails={newAgentBusinessDetails}
            selectedSetupToolKeys={selectedAvailableSetupToolKeys}
            setupToolOptions={setupToolOptions}
            setupCopy={setupCopy}
            canEditAgent={canEditAgent}
            isNewAgentReady={isNewAgentReady}
            isSavingAgent={isSavingAgent}
            onSelectTemplate={selectAgentSetupTemplate}
            onConfirmTemplate={confirmSuggestedTemplate}
            onEditStep={editAgentSetupStep}
            onBack={goBackInAgentSetup}
            onCancel={closeAgentSetup}
            onSkipDetails={skipAgentSetupDetails}
            onToggleAdminCondition={toggleNewAgentAdminCondition}
            onUpdateAdminOther={(value) => updateNewAgentBusinessDetail("adminOther", value)}
            onToggleTool={toggleSetupTool}
            onContinue={continueAgentSetup}
            onCreate={createAgentFromReview}
          />
        </div>
      </form>
    </div>
  ) : null;

  const isDeletingAgent = deleteDialogAgent ? busyKey === `ai-agent-delete-${deleteDialogAgent.id}` : false;
  const deleteAgentModal = deleteDialogAgent ? (
    <div className="modal-backdrop agent-delete-backdrop" role="presentation" onMouseDown={(event) => {
      if (event.target === event.currentTarget && !isDeletingAgent) setDeleteDialogAgent(null);
    }}>
      <div className="modal-card agent-delete-modal" role="dialog" aria-modal="true" aria-labelledby="agent-delete-title">
        <div className={`agent-delete-icon ${deleteDialogSessions.length ? "warning" : "danger"}`} aria-hidden="true">
          {deleteDialogSessions.length ? Icons.whatsapp : Icons.trash}
        </div>
        <div className="agent-delete-copy">
          <h2 id="agent-delete-title">
            {deleteDialogSessions.length ? "Agent masih dipakai WhatsApp" : `Hapus ${deleteDialogAgent.name}?`}
          </h2>
          {deleteDialogSessions.length ? (
            <>
              <p>Agent ini masih terhubung ke WhatsApp. Lepaskan atau hapus session WhatsApp dulu, baru agent bisa dihapus.</p>
              <div className="agent-linked-session-list">
                {deleteDialogSessions.map((session) => (
                  <div key={session.id} className="agent-linked-session-item">
                    <span>
                      <strong>{session.label || "WhatsApp Session"}</strong>
                      <small>{session.aiAgentName || deleteDialogAgent.name}</small>
                    </span>
                    <span className={`badge ${session.status === "connected" ? "green" : session.status === "qr_pending" ? "orange" : "gray"}`}>
                      {session.status === "connected" ? "Connected" : session.status || "Disconnected"}
                    </span>
                  </div>
                ))}
              </div>
            </>
          ) : (
            <p>Materi dan pengaturan agent ini akan dihapus. Riwayat chat tetap aman.</p>
          )}
        </div>
        <div className="modal-actions agent-delete-actions">
          <button className="btn btn-secondary" type="button" onClick={() => setDeleteDialogAgent(null)} disabled={isDeletingAgent}>
            Batal
          </button>
          {deleteDialogSessions.length ? (
            <button className="btn btn-primary" type="button" onClick={openWhatsAppFromDeleteFlow}>
              {Icons.whatsapp} Buka WhatsApp
            </button>
          ) : (
            <button className="btn btn-danger" type="button" onClick={confirmDeleteAgent} disabled={isDeletingAgent}>
              {isDeletingAgent ? "Menghapus..." : "Hapus agent"}
            </button>
          )}
        </div>
      </div>
    </div>
  ) : null;

  return (
    <>
      <div className="agents-page">
        <div className="page-title-row agents-title-row">
          <div>
            <div className="page-title">AI CS</div>
            <div className="page-desc">AI yang membalas chat pelanggan otomatis.</div>
          </div>
          <div className="page-actions">
            {canEditAgent && aiAgents.length ? (
              <button className="btn btn-primary" type="button" onClick={startNewAgent} data-tour="create-agent">
                {Icons.plus} Agent Baru
              </button>
            ) : null}
          </div>
        </div>

        {!isStartGuideComplete ? (
          <div className="agents-startguide">
            <div className="agents-startguide-head">
              <strong>Mulai di sini</strong>
              <span>3 langkah supaya AI kamu siap melayani pelanggan.</span>
            </div>
            <div className="agents-startguide-steps">
              <button type="button" className={`agents-startguide-step ${hasActiveAgent ? "done" : "active"}`} onClick={startNewAgent} disabled={!canEditAgent || isSavingAgent}>
                <span className="agents-startguide-index">{hasActiveAgent ? Icons.check : "1"}</span>
                <span className="agents-startguide-copy">
                  <strong>Buat AI CS</strong>
                  <small>{hasActiveAgent ? "Selesai" : "Belum dibuat"}</small>
                </span>
              </button>
              <button type="button" className={`agents-startguide-step ${anyWhatsAppConnected ? "done" : hasActiveAgent ? "active" : ""}`} onClick={() => navigateToWhatsApp?.()}>
                <span className="agents-startguide-index">{anyWhatsAppConnected ? Icons.check : "2"}</span>
                <span className="agents-startguide-copy">
                  <strong>Hubungkan channel</strong>
                  <small>{anyWhatsAppConnected ? "Terhubung" : "Belum terhubung"}</small>
                </span>
              </button>
              <button type="button" className="agents-startguide-step" onClick={() => navigateToPlayground?.()}>
                <span className="agents-startguide-index">3</span>
                <span className="agents-startguide-copy">
                  <strong>Coba AI</strong>
                  <small>Tes tanpa kirim WhatsApp</small>
                </span>
              </button>
            </div>
          </div>
        ) : null}

        <div className="agents-stats-grid">
          <AgentStatCard icon={Icons.robot} label="Total Agent" value={aiAgents.length} tone="blue" />
          <AgentStatCard icon={Icons.check} label="Aktif" value={activeAgents.length} tone="green" />
          <AgentStatCard icon={Icons.whatsapp} label="Terhubung WhatsApp" value={connectedAgentIds.size} tone="teal" />
          <AgentStatCard icon={Icons.alert} label="Draft / Nonaktif" value={aiAgents.length - activeAgents.length} tone="orange" />
        </div>

        {aiAgents.length ? (
        <div className="agents-layout">
          <aside className="agents-list-panel">
            <div className="agents-list-header">
              <div>
                <strong>Daftar Agent</strong>
                <span>{visibleAgents.length} tampil</span>
              </div>
            </div>
            <div className="agent-list-tools">
              <label className="agent-search-field">
                {Icons.search}
                <input value={agentQuery} onChange={(event) => setAgentQuery(event.target.value)} placeholder="Cari agent..." />
              </label>
              <select className="form-select" value={agentStatusFilter} onChange={(event) => setAgentStatusFilter(event.target.value)}>
                <option value="all">Semua Status</option>
                <option value="active">Aktif</option>
                <option value="inactive">Draft / Nonaktif</option>
                <option value="connected">Terhubung WhatsApp</option>
              </select>
            </div>
            {visibleAgents.length ? (
              <div className="agents-list">
                {visibleAgents.map((agent) => {
                  const isConnected = connectedAgentIds.has(agent.id);
                  return (
                    <button
                      key={agent.id}
                      type="button"
                      className={`agent-list-item ${agent.id === selectedAgentId ? "active" : ""}`}
                      onClick={() => selectAgent(agent)}
                    >
                      <AgentAvatar />
                      <span className="agent-list-copy">
                        <strong>{agent.name}</strong>
                        <small>{modelLabel(agent.modelName)}</small>
                      </span>
                      <span className="agent-list-side">
                        <span className={`badge ${agent.isActive === false ? "gray" : "green"}`}>{agent.isActive === false ? "Draft" : "Aktif"}</span>
                        <span className={`agent-wa-indicator ${isConnected ? "connected" : ""}`}>{Icons.whatsapp}</span>
                      </span>
                      <span className="agent-list-updated">Diperbarui {agent.updatedAt ? formatDateTime(agent.updatedAt) : "-"}</span>
                    </button>
                  );
                })}
              </div>
            ) : (
              <EmptyState
                title="Agent tidak ditemukan"
                copy="Coba ubah pencarian atau filter status."
                compact
              />
            )}
          </aside>

          <section className="agents-main-panel">
            {selectedAgent ? (
              <>
                <div className="agent-detail-head">
                  <div className="agent-detail-identity">
                    <AgentAvatar large />
                    <div>
                      <div className="agent-detail-title-row">
                        <h2>{selectedAgent.name}</h2>
                        <span className={`badge ${form.isActive === false ? "gray" : "green"}`}>{form.isActive === false ? "Draft" : "Aktif"}</span>
                      </div>
                      <p>Agent ini siap menjawab dengan pengetahuan dan channel yang kamu atur.</p>
                    </div>
                  </div>
                  <div className="agent-detail-actions">
                    {navigateToPlayground ? (
                      <button className="btn btn-secondary btn-sm" type="button" onClick={navigateToPlayground}>
                        {Icons.spark} Test di Playground
                      </button>
                    ) : null}
                    <button className="btn btn-secondary btn-sm" type="button" onClick={navigateToWhatsApp}>
                      {Icons.whatsapp} WhatsApp
                    </button>
                    {canEditAgent && deleteAIAgent ? (
                      <button className="btn btn-danger btn-sm" type="button" onClick={requestDeleteAgent} disabled={isSavingAgent || String(busyKey || "").startsWith("ai-agent-delete-")}>
                        {Icons.trash} Hapus
                      </button>
                    ) : null}
                  </div>
                </div>

                <div className="agent-editor-band">
                  <form className="agent-editor-form" onSubmit={saveAIAgent}>
                    <input type="hidden" value={form.id || ""} readOnly />
                    <div className="agent-form-grid">
                      <label className="form-group">
                        <span className="form-label">Nama agent</span>
                        <input className="form-input" value={form.name || ""} onChange={(event) => updateField("name", event.target.value)} placeholder="Contoh: Sales Bot" disabled={!canEditAgent} />
                      </label>
                      <div className="form-group agent-model-field">
                        <span className="form-label">Paket AI</span>
                        <ModelChoiceCards value={form.modelName} onChange={(modelId) => updateField("modelName", modelId)} disabled={!canEditAgent} compact models={availableModelOptions} />
                      </div>
                      <div className="form-group agent-status-field">
                        <span className="form-label">Status</span>
                        <label className="agent-active-switch">
                          <input className="agent-active-input" type="checkbox" checked={form.isActive !== false} onChange={(event) => updateField("isActive", event.target.checked)} disabled={!canEditAgent} />
                          <span className="agent-active-control" aria-hidden="true" />
                          <span className="agent-active-copy">
                            <strong>{form.isActive !== false ? "Aktif" : "Nonaktif"}</strong>
                            <small>{form.isActive !== false ? "Bisa dipakai WhatsApp" : "Tidak dipakai WhatsApp"}</small>
                          </span>
                        </label>
                      </div>
                    </div>
                    <div className="agent-form-actions">
                      <p className={`agent-save-hint ${isSavingAgent ? "saving" : hasUnsavedChanges ? "dirty" : aiAgentSaveFeedback?.status === "saved" ? "saved" : ""}`}>
                        {isSavingAgent
                          ? "Menyimpan setting agent..."
                          : hasUnsavedChanges
                            ? "Ada perubahan belum disimpan."
                            : aiAgentSaveFeedback?.message || `Tersimpan${selectedAgent?.updatedAt ? `, update terakhir ${formatDateTime(selectedAgent.updatedAt)}` : ""}`}
                      </p>
                      <button className="btn btn-primary" type="submit" disabled={!canEditAgent || isSavingAgent || !hasUnsavedChanges}>
                        {isSavingAgent ? "Menyimpan..." : "Update setting"}
                      </button>
                    </div>
                  </form>
                </div>

                <div className="tabs agent-tabs">
                  {agentTabs.map((tab) => (
                    <button key={tab.id} type="button" className={`tab ${selectedTab === tab.id ? "active" : ""}`} onClick={() => setAgentTab?.(tab.id)} data-tour={tab.id === "knowledge" ? tab.id : undefined}>
                      {tab.label}
                    </button>
                  ))}
                </div>

                {selectedTab === "overview" ? (
                  <>
                    <div className="agent-overview-grid">
                      <div className="agent-overview-panel">
                        <div className="agent-section-title">Ringkasan</div>
                        <div className="agent-summary-list">
                          <SummaryRow icon={Icons.help} label="FAQ" value={knowledgeProps?.faqs?.length || 0} />
                          <SummaryRow icon={Icons.usage} label="Dokumen" value={knowledgeProps?.documents?.length || 0} />
                          {usesBusinessCatalogTools
                            ? <SummaryRow icon={Icons.knowledge} label="Produk / Layanan" value="Alat Bisnis" />
                            : <SummaryRow icon={Icons.knowledge} label="Produk / Layanan" value={knowledgeProps?.positions?.length || 0} />}
                          <SummaryRow icon={Icons.whatsapp} label="WhatsApp Terhubung" value={connectedSessions.length} />
                          <SummaryRow icon={Icons.health} label="Terakhir Aktif" value={selectedAgent.updatedAt ? formatDateTime(selectedAgent.updatedAt) : "-"} />
                        </div>
                      </div>
                      <div className="agent-overview-panel">
                        <div className="agent-section-title">Kesiapan Agent</div>
                        <div className="agent-readiness-list">
                          {readiness.map((item) => (
                            <ReadinessRow key={item.label} label={item.label} state={item.state} />
                          ))}
                        </div>
                      </div>
                    </div>
                    <div className="agent-performance-panel">
                      <div className="agent-section-title">Status Operasional</div>
                      <div className="agent-performance-grid">
                        <div className="agent-performance-item">
                          <span>WhatsApp</span>
                          <strong>{connectedSessions.length}</strong>
                          <small>session memakai agent ini</small>
                        </div>
                        <div className="agent-performance-item">
                          <span>Pengetahuan</span>
                          <strong>{totalKnowledge}</strong>
                          <small>{usesBusinessCatalogTools ? "FAQ dan dokumen" : "FAQ, dokumen, produk/layanan"}</small>
                        </div>
                        <div className="agent-performance-item">
                          <span>Keamanan jawaban</span>
                          <strong>{agentSafetyRuleOptions.filter((rule) => rule.checked(form)).length}/{agentSafetyRuleOptions.length}</strong>
                          <small>aturan aman aktif</small>
                        </div>
                        <div className="agent-performance-item">
                          <span>Paket AI</span>
                          <strong>{modelLabel(form.modelName)}</strong>
                          <small>aktif untuk jawaban AI</small>
                        </div>
                      </div>
                    </div>
                  </>
                ) : null}

                {selectedTab === "instructions" ? (
                  <div className="agent-rules-grid">
                    <div className="agent-edit-guide agent-span-2">
                      <strong>Edit cepat</strong>
                      <span>Isi bagian bertanda. Minimal: nama & jenis bisnis, data yang dikumpulkan, dan kapan AI panggil admin.</span>
                    </div>
                    <label className="form-group agent-span-2">
                      <span className="form-label">Cara AI menjawab</span>
                      <textarea className="form-textarea agent-guided-textarea" rows={10} value={form.systemPrompt || ""} onChange={(event) => updateField("systemPrompt", event.target.value)} placeholder="Tulis cara agent menjawab pelanggan." disabled={!canEditAgent} />
                      <span className="form-field-hint">Panduan utama AI. Ubah bagian bertanda saja, jangan hapus batasan penting.</span>
                    </label>
                    <label className="form-group">
                      <span className="form-label">Pesan saat AI minta bantuan admin</span>
                      <textarea className="form-textarea" rows={4} value={form.fallbackWaitingMessage || ""} onChange={(event) => updateField("fallbackWaitingMessage", event.target.value)} disabled={!canEditAgent} />
                      <span className="form-field-hint">Pesan ini dikirim ke customer saat AI perlu bantuan tim.</span>
                    </label>
                    <label className="form-group">
                      <span className="form-label">Kapan harus panggil admin</span>
                      <textarea className="form-textarea" rows={6} value={form.escalationPrompt || ""} onChange={(event) => updateField("escalationPrompt", event.target.value)} placeholder="Contoh: refund, komplain pembayaran, perubahan order, atau data belum pasti." disabled={!canEditAgent} />
                      <span className="form-help-row">
                        <span>Aturan ini menentukan kapan AI berhenti menjawab sendiri dan meminta bantuan tim.</span>
                        <button className="link-button" type="button" onClick={() => setAgentTab?.("escalation")}>Atur pesan grup</button>
                      </span>
                    </label>
                    <div className="agent-instructions-default-note agent-span-2">
                      {Icons.check}
                      <span>Batas aman jawaban, gaya percakapan, dan aturan kontak aktif otomatis dengan default aman.</span>
                    </div>
                    <div className="agent-rule-section agent-span-2">
                      <div className="agent-section-title">Customer Memory</div>
                      <div className="agent-rule-list compact single">
                        <AgentRuleToggle
                          label={agentMemoryRule.label}
                          helper={agentMemoryRule.helper}
                          checked={agentMemoryRule.checked(form)}
                          onChange={(value) => updateField(agentMemoryRule.field, value)}
                          disabled={!canEditAgent}
                        />
                      </div>
                      {form.customerMemoryEnabled ? (
                        <div className="alert orange agent-memory-credit-alert">
                          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
                          <span>AI mengingat konteks tiap pelanggan. Menambah biaya, jadi aktifkan hanya kalau perlu personalisasi.</span>
                        </div>
                      ) : null}
                      <div className="agent-form-grid mt-16">
                        <label className="form-group">
                          <span className="form-label">Maks item</span>
                          <select className="form-select" value={form.customerMemoryMaxItems ?? 5} onChange={(event) => updateField("customerMemoryMaxItems", Number(event.target.value))} disabled={!canEditAgent || !form.customerMemoryEnabled}>
                            {memoryLimitOptions.items.map((value) => <option key={value} value={value}>{value}</option>)}
                          </select>
                        </label>
                        <label className="form-group">
                          <span className="form-label">Maks karakter</span>
                          <select className="form-select" value={form.customerMemoryMaxChars ?? 800} onChange={(event) => updateField("customerMemoryMaxChars", Number(event.target.value))} disabled={!canEditAgent || !form.customerMemoryEnabled}>
                            {memoryLimitOptions.chars.map((value) => <option key={value} value={value}>{value}</option>)}
                          </select>
                        </label>
                        <label className="form-group">
                          <span className="form-label">Retensi</span>
                          <select className="form-select" value={form.customerMemoryRetentionDays ?? 180} onChange={(event) => updateField("customerMemoryRetentionDays", Number(event.target.value))} disabled={!canEditAgent || !form.customerMemoryEnabled}>
                            {memoryLimitOptions.retention.map((value) => <option key={value} value={value}>{value} hari</option>)}
                          </select>
                        </label>
                      </div>
                    </div>
                  </div>
                ) : null}

                {selectedTab === "knowledge" ? <KnowledgeView {...selectedKnowledgeProps} /> : null}

                {selectedTab === "whatsapp" ? (
                  <div className="agent-connected-wa">
                    {connectedSessions.length ? connectedSessions.map((session) => (
                      <div key={session.id} className="agent-wa-row">
                        <div>
                          <strong>{session.label}</strong>
                          <span>{session.sessionKey}</span>
                        </div>
                        <span className={`badge ${session.status === "connected" ? "green" : session.status === "qr_pending" ? "orange" : "gray"}`}>{sessionStatusLabel(session.status)}</span>
                      </div>
                    )) : (
                      <EmptyState title="Belum dipakai WhatsApp" copy="Tempel agent ini dari halaman WhatsApp." compact />
                    )}
                    <button className="btn btn-secondary" type="button" onClick={navigateToWhatsApp}>Ke WhatsApp</button>
                  </div>
                ) : null}

                {selectedTab === "escalation" ? (
                  <form className="agent-notification-editor" onSubmit={submitNotificationRule}>
                    <div className="agent-notification-head">
                      <div>
                        <span className="agent-section-kicker">Bantuan admin</span>
                        <h3>Notifikasi saat AI butuh bantuan</h3>
                        <p>Aturan panggil admin tetap di Cara Jawab AI. Bagian ini hanya mengatur pesan internal yang masuk ke grup tim.</p>
                      </div>
                      <div className="agent-notification-status">
                        <span className={`badge ${isNotificationGroupBound ? "green" : "orange"}`}>{isNotificationGroupBound ? "Grup siap" : "Grup belum dipilih"}</span>
                        <span className={`badge ${notificationDraft.isEnabled ? "green" : "gray"}`}>{notificationDraft.isEnabled ? "Aktif" : "Nonaktif"}</span>
                      </div>
                    </div>

                    <div className="agent-notification-grid">
                      <section className="agent-notification-main" aria-label="Notifikasi grup bantuan admin">
                        <label className="agent-notification-toggle">
                          <span>
                            <strong>Kirim notif ke grup kalau AI butuh bantuan</strong>
                            <small>Customer tetap menerima fallback message. Tim hanya menerima ringkasan untuk ditindaklanjuti.</small>
                          </span>
                          <span className="toggle">
                            <input
                              type="checkbox"
                              checked={notificationDraft.isEnabled}
                              onChange={(event) => setNotificationDraft((current) => ({ ...current, isEnabled: event.target.checked }))}
                              disabled={!canManageGroupNotifications || isNotificationSaving}
                            />
                            <span className="toggle-slider" />
                          </span>
                        </label>

                        <div className="agent-notification-summary-card">
                          <div className="agent-notification-summary-row">
                            <span>Grup tujuan</span>
                            <strong>{notificationGroupLabel}</strong>
                            <button className="link-button" type="button" onClick={navigateToWhatsApp}>
                              {isNotificationGroupBound ? "Ganti grup" : "Pilih grup"}
                            </button>
                          </div>
                          <div className="agent-notification-summary-row">
                            <span>Status pesan</span>
                            <strong>{notificationDraft.isEnabled ? "Akan dikirim" : "Tidak dikirim"}</strong>
                            {notificationDirty ? <em>Belum disimpan</em> : <em>Tersimpan</em>}
                          </div>
                        </div>

                        <div className="agent-notification-message-card">
                          <div className="agent-variable-title">
                            <strong>Pesan yang dikirim</strong>
                            <small>Ini contoh pesan yang akan muncul di grup.</small>
                          </div>
                          <pre>{notificationPreviewText}</pre>
                        </div>

                        <div className="agent-notification-actions">
                          <button className="btn btn-secondary" type="button" onClick={() => setIsNotificationComposerOpen((open) => !open)}>
                            {isNotificationComposerOpen ? "Tutup editor" : "Edit pesan"}
                          </button>
                          <button className="btn btn-primary" type="submit" disabled={!canManageGroupNotifications || isNotificationSaving || !notificationDirty || !String(notificationDraft.templateText || "").trim()}>
                            {isNotificationSaving ? "Menyimpan..." : "Simpan perubahan"}
                          </button>
                          <button className="btn btn-secondary" type="button" onClick={sendGroupNotificationTest} disabled={!canManageGroupNotifications || !isNotificationGroupBound || isNotificationTesting || notificationDirty} title={notificationDirty ? "Simpan perubahan dulu" : undefined}>
                            {isNotificationTesting ? "Mengirim test..." : "Kirim test"}
                          </button>
                        </div>

                        {isNotificationComposerOpen ? (
                          <div className="agent-message-composer">
                            <label className="form-group">
                              <span className="form-label">Isi pesan</span>
                              <textarea
                                ref={notificationTemplateRef}
                                className="form-textarea notification-template-textarea"
                                rows={8}
                                value={notificationDraft.templateText}
                                onChange={(event) => setNotificationDraft((current) => ({ ...current, templateText: event.target.value }))}
                                disabled={!canManageGroupNotifications || isNotificationSaving}
                              />
                            </label>

                            <div className="agent-auto-data-panel">
                              <div className="agent-variable-title">
                                <strong>Tambah data otomatis</strong>
                                <small>Klik data yang ingin dimasukkan ke posisi kursor.</small>
                              </div>
                              <div className="agent-variable-chip-row">
                                {visibleNotificationVariables.map((variable) => (
                                  <button key={variable.key} className="agent-variable-chip" type="button" onClick={() => insertNotificationVariable(variable.key)} disabled={!canManageGroupNotifications || isNotificationSaving} title={variable.example || variable.label}>
                                    {variable.label}
                                  </button>
                                ))}
                              </div>
                              <details className="agent-variable-details">
                                <summary>Lihat kategori data</summary>
                                <div className="agent-variable-groups">
                                  {Object.entries(notificationVariableGroups).map(([groupName, variables]) => (
                                    <div className="agent-variable-group" key={groupName}>
                                      <span>{groupName}</span>
                                      <div>
                                        {variables.map((variable) => (
                                          <button key={variable.key} className="agent-variable-chip" type="button" onClick={() => insertNotificationVariable(variable.key)} disabled={!canManageGroupNotifications || isNotificationSaving} title={variable.example || variable.label}>
                                            {variable.label}
                                          </button>
                                        ))}
                                      </div>
                                    </div>
                                  ))}
                                </div>
                              </details>
                            </div>
                          </div>
                        ) : null}
                      </section>

                      <aside className="agent-whatsapp-preview" aria-label="Preview WhatsApp">
                        <div className="agent-whatsapp-preview-head">
                          <div>
                            <strong>Preview WhatsApp</strong>
                            <span>Ilustrasi pesan di grup tim.</span>
                          </div>
                          <span className={`badge ${notificationDraft.isEnabled ? "green" : "gray"}`}>{notificationDraft.isEnabled ? "Aktif" : "Mati"}</span>
                        </div>
                        <div className="agent-whatsapp-phone">
                          <div className="agent-whatsapp-phone-bar">
                            <span className="agent-whatsapp-avatar">{Icons.whatsapp}</span>
                            <div>
                              <strong>{notificationGroupLabel}</strong>
                              <small>Notifikasi internal</small>
                            </div>
                          </div>
                          <div className="agent-whatsapp-chat-area">
                            <div className="agent-whatsapp-bubble">
                              <pre>{notificationPreviewText}</pre>
                              <span>10.30</span>
                            </div>
                          </div>
                        </div>
                        <p>Preview memakai contoh data. Isi asli mengikuti chat yang membuat AI meminta bantuan admin.</p>
                      </aside>
                    </div>
                  </form>
                ) : null}
              </>
            ) : (
              <EmptyState
                title="Pilih AI CS"
                copy="Pilih AI CS di daftar untuk lihat detailnya."
                compact
              />
            )}
          </section>
        </div>
        ) : (
          <div className="agents-onboarding">
            <span className="agents-onboarding-orb" aria-hidden="true">{Icons.robot}</span>
            <h2>Buat AI CS pertama kamu</h2>
            <p>Satu AI CS bisa balas chat customer otomatis di WhatsApp—dari tanya produk, harga, sampai booking. Ceritakan bisnismu seperti ngobrol biasa, lalu Oneflow yang menyusun setupnya.</p>
            <div className="agents-onboarding-steps">
              <span><b>1</b>Ceritakan bisnismu</span>
              <span><b>2</b>Konfirmasi setup yang disarankan</span>
              <span><b>3</b>Pilih bantuan AI</span>
              <span><b>4</b>Review &amp; buat</span>
            </div>
            {canEditAgent ? (
              <button className="btn btn-primary btn-lg" type="button" onClick={startNewAgent} data-tour="create-agent">
                {Icons.plus} Buat AI CS
              </button>
            ) : (
              <p className="agents-onboarding-locked">Hubungi admin workspace untuk membuat AI CS.</p>
            )}
          </div>
        )}
      </div>

      {createAgentModal && typeof document !== "undefined" ? createPortal(createAgentModal, document.body) : null}
      {deleteAgentModal && typeof document !== "undefined" ? createPortal(deleteAgentModal, document.body) : null}
    </>
  );
}
