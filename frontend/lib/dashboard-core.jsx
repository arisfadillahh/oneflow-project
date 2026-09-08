import {
  resolveBrowserApiBase,
  resolveBrowserWsBase,
  resolveWaGatewayBase,
} from "./runtime-config.mjs";

const browserLocation = () => (typeof window === "undefined" ? undefined : window.location);
const dashboardPublicEnv = {
  NEXT_PUBLIC_API_BASE_URL: process.env.NEXT_PUBLIC_API_BASE_URL,
  NEXT_PUBLIC_WS_BASE_URL: process.env.NEXT_PUBLIC_WS_BASE_URL,
  NEXT_PUBLIC_WA_GATEWAY_URL: process.env.NEXT_PUBLIC_WA_GATEWAY_URL,
};

export const apiBase = resolveBrowserApiBase({ env: dashboardPublicEnv, location: browserLocation() });
export const wsBase = resolveBrowserWsBase({ env: dashboardPublicEnv, location: browserLocation() });
export const waBase = resolveWaGatewayBase({ env: dashboardPublicEnv });

export const roleNav = {
  owner: [
    { id: "overview", label: "Ringkasan", icon: "overview" },
    { id: "wallet", label: "Kredit", icon: "wallet", group: "more" },
    { id: "commerceProducts", label: "Produk & Stok", icon: "products", moduleKey: "commerce", group: "tools" },
    { id: "commerceOrders", label: "Pesanan", icon: "orders", moduleKey: "commerce", group: "tools", parentId: "commerceProducts" },
    { id: "usage", label: "Laporan Pemakaian", icon: "usage", group: "more" },
    { id: "health", label: "Status Sistem", icon: "health", group: "more" },
    { id: "appsettings", label: "Model & Harga", icon: "settings", group: "more" },
  ],
  super_admin: [
    { id: "operations", label: "Inbox", icon: "inbox", group: "ops" },
    { id: "agents", label: "AI CS", icon: "robot" },
    { id: "team", label: "Tim", icon: "team", group: "more" },
    { id: "playground", label: "Coba AI", icon: "spark" },
    { id: "contacts", label: "Kontak", icon: "contacts", group: "ops" },
    { id: "whatsapp", label: "Channels", icon: "whatsapp" },
    { id: "businessTools", label: "Alat Bisnis", icon: "settings", group: "ops" },
    { id: "analytics", label: "Laporan", icon: "analytics", group: "more" },
    { id: "upgrade", label: "Tagihan", icon: "wallet", group: "more" },
    { id: "deals", label: "Tindak Lanjut", icon: "analytics", moduleKey: "prospects", group: "tools" },
    { id: "tickets", label: "Tiket", icon: "ticket", moduleKey: "tickets", group: "tools" },
    { id: "commerceProducts", label: "Produk & Stok", icon: "products", moduleKey: "commerce", group: "tools" },
    { id: "commerceOrders", label: "Pesanan", icon: "orders", moduleKey: "commerce", group: "tools", parentId: "commerceProducts" },
    { id: "booking", label: "Booking", icon: "calendar", moduleKey: "booking", group: "tools" },
  ],
  admin: [
    { id: "operations", label: "Inbox", icon: "inbox", group: "ops" },
    { id: "agents", label: "AI CS", icon: "robot" },
    { id: "team", label: "Tim", icon: "team", group: "more" },
    { id: "playground", label: "Coba AI", icon: "spark" },
    { id: "contacts", label: "Kontak", icon: "contacts", group: "ops" },
    { id: "whatsapp", label: "Channels", icon: "whatsapp" },
    { id: "businessTools", label: "Alat Bisnis", icon: "settings", group: "ops" },
    { id: "analytics", label: "Laporan", icon: "analytics", group: "more" },
    { id: "upgrade", label: "Tagihan", icon: "wallet", group: "more" },
    { id: "deals", label: "Tindak Lanjut", icon: "analytics", moduleKey: "prospects", group: "tools" },
    { id: "tickets", label: "Tiket", icon: "ticket", moduleKey: "tickets", group: "tools" },
    { id: "commerceProducts", label: "Produk & Stok", icon: "products", moduleKey: "commerce", group: "tools" },
    { id: "commerceOrders", label: "Pesanan", icon: "orders", moduleKey: "commerce", group: "tools", parentId: "commerceProducts" },
    { id: "booking", label: "Booking", icon: "calendar", moduleKey: "booking", group: "tools" },
  ],
  operator: [
    { id: "operations", label: "Inbox", icon: "inbox", group: "ops" },
    { id: "agents", label: "AI CS", icon: "robot" },
    { id: "playground", label: "Coba AI", icon: "spark" },
    { id: "contacts", label: "Kontak", icon: "contacts", group: "ops" },
    { id: "whatsapp", label: "Channels", icon: "whatsapp" },
    { id: "analytics", label: "Laporan", icon: "analytics", group: "more" },
    { id: "upgrade", label: "Tagihan", icon: "wallet", group: "more" },
    { id: "deals", label: "Tindak Lanjut", icon: "analytics", moduleKey: "prospects", group: "tools" },
    { id: "tickets", label: "Tiket", icon: "ticket", moduleKey: "tickets", group: "tools" },
    { id: "commerceProducts", label: "Produk & Stok", icon: "products", moduleKey: "commerce", group: "tools" },
    { id: "commerceOrders", label: "Pesanan", icon: "orders", moduleKey: "commerce", group: "tools", parentId: "commerceProducts" },
    { id: "booking", label: "Booking", icon: "calendar", moduleKey: "booking", group: "tools" },
  ],
};

export const viewMeta = {
  overview: { title: "Ringkasan", description: "Ringkasan pemakaian, kredit, dan status koneksi." },
  dashboard: { title: "Laporan Performa", description: "Statistik performa AI, balasan tim, dan volume chat." },
  operations: { title: "Inbox", description: "Semua percakapan dalam satu tempat." },
  contacts: { title: "Kontak", description: "Data dan riwayat pelanggan." },
  deals: { title: "Tindak Lanjut", description: "Peluang penjualan yang perlu ditindaklanjuti." },
  tickets: { title: "Tiket", description: "Keluhan pelanggan dan penanganannya." },
  commerceProducts: { title: "Produk & Stok", description: "Kelola produk, harga, dan stok." },
  commerceOrders: { title: "Pesanan", description: "Buat dan pantau draf pesanan dari produk aktif." },
  booking: { title: "Booking", description: "Kelola layanan, jadwal, dan janji temu pelanggan." },
  agents: { title: "AI CS", description: "AI yang membalas chat pelanggan otomatis." },
  knowledge: { title: "Pengetahuan AI", description: "Materi yang dipakai AI untuk menjawab." },
  businessTools: { title: "Alat Bisnis", description: "Tambahkan kemampuan AI: produk, pesanan, dan booking." },
  settings: { title: "Aturan AI", description: "Atur batas jawaban AI dan kapan diteruskan ke admin." },
  analytics: { title: "Laporan", description: "Ringkasan percakapan dan performa AI." },
  playground: { title: "Coba AI", description: "Coba AI seperti pelanggan. Tidak mengirim WhatsApp." },
  wallet: { title: "Kredit", description: "Pantau dan kelola kredit." },
  upgrade: { title: "Tagihan", description: "Pantau paket, kredit, dan pembayaran." },
  account: { title: "Akun", description: "Kelola nama akun dan kata sandi." },
  team: { title: "Tim", description: "Tambah anggota tim dan atur aksesnya." },
  usage: { title: "Laporan Pemakaian", description: "Ringkasan pemakaian AI. Isi chat tetap privat." },
  health: { title: "Status Sistem", description: "Status sistem dan koneksi." },
  whatsapp: { title: "Channels", description: "Hubungkan channel bisnis ke AI kamu." },
  whatsappTemplates: { title: "Template WhatsApp", description: "Kelola template pesan resmi WhatsApp." },
  appsettings: { title: "Model & Harga", description: "Atur model AI dan perhitungan kredit." },
};

export const modelCreditEstimates = {
  "openai/gpt-4o-mini": { averageAnswer: 1, complexAnswer: 2, imageAnswer: 3, escalation: 1 },
  "openai/gpt-4.1-mini": { averageAnswer: 3, complexAnswer: 6, imageAnswer: 8, escalation: 3 },
  "anthropic/claude-haiku-4.5": { averageAnswer: 3, complexAnswer: 6, imageAnswer: 8, escalation: 3 },
  "deepseek/deepseek-v4-flash": { averageAnswer: 3, complexAnswer: 6, imageAnswer: 8, escalation: 3 },
  "deepseek/deepseek-v4-pro": { averageAnswer: 6, complexAnswer: 12, imageAnswer: 16, escalation: 6 },
  "deepseek/deepseek-v3.2": { averageAnswer: 3, complexAnswer: 6, imageAnswer: 8, escalation: 3 },
  "deepseek/deepseek-chat-v3.1": { averageAnswer: 3, complexAnswer: 6, imageAnswer: 8, escalation: 3 },
  "deepseek/deepseek-r1": { averageAnswer: 3, complexAnswer: 6, imageAnswer: 8, escalation: 3 },
};

export const defaultModelAliases = {
  "openai/gpt-4o-mini": "Oneflow.id Basic Model",
  "openai/gpt-4.1-mini": "Oneflow.id Advance Model",
  "deepseek/deepseek-v4-flash": "DeepSeek V4 Flash",
  "deepseek/deepseek-v4-pro": "DeepSeek V4 Pro",
  "deepseek/deepseek-v3.2": "DeepSeek V3.2",
  "deepseek/deepseek-chat-v3.1": "DeepSeek V3.1",
  "deepseek/deepseek-r1": "DeepSeek R1",
};

export const defaultAvailableModelIds = ["openai/gpt-4o-mini", "openai/gpt-4.1-mini"];

export const fallbackOpenRouterModels = [
  { id: "openai/gpt-4o-mini", name: defaultModelAliases["openai/gpt-4o-mini"], alias: defaultModelAliases["openai/gpt-4o-mini"], upstreamName: "gpt-4o-mini", provider: "GPT", available: true, inputPricePer1m: 0.15, outputPricePer1m: 0.6, creditEstimate: modelCreditEstimates["openai/gpt-4o-mini"] },
  { id: "openai/gpt-4.1-mini", name: defaultModelAliases["openai/gpt-4.1-mini"], alias: defaultModelAliases["openai/gpt-4.1-mini"], upstreamName: "gpt-4.1-mini", provider: "GPT", available: true, inputPricePer1m: 0.4, outputPricePer1m: 1.6, creditEstimate: modelCreditEstimates["openai/gpt-4.1-mini"] },
  { id: "deepseek/deepseek-v4-flash", name: defaultModelAliases["deepseek/deepseek-v4-flash"], alias: defaultModelAliases["deepseek/deepseek-v4-flash"], upstreamName: "DeepSeek V4 Flash", provider: "DeepSeek", available: false, inputPricePer1m: 0.1, outputPricePer1m: 0.2, creditEstimate: modelCreditEstimates["deepseek/deepseek-v4-flash"] },
  { id: "deepseek/deepseek-v4-pro", name: defaultModelAliases["deepseek/deepseek-v4-pro"], alias: defaultModelAliases["deepseek/deepseek-v4-pro"], upstreamName: "DeepSeek V4 Pro", provider: "DeepSeek", available: false, inputPricePer1m: 0.435, outputPricePer1m: 0.87, creditEstimate: modelCreditEstimates["deepseek/deepseek-v4-pro"] },
  { id: "deepseek/deepseek-v3.2", name: defaultModelAliases["deepseek/deepseek-v3.2"], alias: defaultModelAliases["deepseek/deepseek-v3.2"], upstreamName: "DeepSeek V3.2", provider: "DeepSeek", available: false, inputPricePer1m: 0.252, outputPricePer1m: 0.378, creditEstimate: modelCreditEstimates["deepseek/deepseek-v3.2"] },
  { id: "deepseek/deepseek-chat-v3.1", name: defaultModelAliases["deepseek/deepseek-chat-v3.1"], alias: defaultModelAliases["deepseek/deepseek-chat-v3.1"], upstreamName: "DeepSeek V3.1", provider: "DeepSeek", available: false, inputPricePer1m: 0.21, outputPricePer1m: 0.79, creditEstimate: modelCreditEstimates["deepseek/deepseek-chat-v3.1"] },
  { id: "deepseek/deepseek-r1", name: defaultModelAliases["deepseek/deepseek-r1"], alias: defaultModelAliases["deepseek/deepseek-r1"], upstreamName: "DeepSeek R1", provider: "DeepSeek", available: false, inputPricePer1m: 0.7, outputPricePer1m: 2.5, creditEstimate: modelCreditEstimates["deepseek/deepseek-r1"] },
];

export const fallbackOpenAIModels = fallbackOpenRouterModels;

export const analyticsRangeOptions = [
  { key: "daily", label: "1D" },
  { key: "five_day", label: "5D" },
  { key: "monthly", label: "1M" },
  { key: "yearly", label: "1Y" },
  { key: "five_year", label: "5Y" },
  { key: "max", label: "Max" },
];

export const composerEmojis = ["😊", "🙏", "👍", "💪", "✅", "📌", "📄", "🔗", "⏳", "🙇", "🙂", "✨"];

export function normalizeModelOptions(items, selectedModel) {
  const seen = new Set();
  const options = [];
  const sourceItems = (items || []).length ? items : fallbackOpenRouterModels;
  sourceItems.forEach((item) => {
    const id = String(item?.id || "").trim();
    if (!id || seen.has(id)) return;
    seen.add(id);
    const alias = item.alias || defaultModelAliases[id] || "";
    const upstreamName = item.upstreamName || item.upstream_name || (alias && item.name === alias ? "" : item.name) || id.replace(/^~?(openai|anthropic)\//, "");
    const displayName = alias || item.name || upstreamName || id;
    options.push({
      id,
      name: displayName,
      alias,
      upstreamName,
      provider: item.provider || (id.includes("anthropic/") ? "Claude" : id.includes("deepseek/") ? "DeepSeek" : "GPT"),
      contextLength: Number(item.contextLength ?? item.context_length ?? 0),
      available: Boolean(item.available ?? defaultAvailableModelIds.includes(id)),
      inputPricePer1m: Number(item.inputPricePer1m ?? 0),
      outputPricePer1m: Number(item.outputPricePer1m ?? 0),
      pricingSource: item.pricingSource || "fallback",
      creditEstimate: item.creditEstimate || modelCreditEstimates[id] || modelCreditEstimates["openai/gpt-4o-mini"],
    });
  });
  if (selectedModel && !seen.has(selectedModel)) {
    options.unshift({
      id: selectedModel,
      name: defaultModelAliases[selectedModel] || selectedModel,
      alias: defaultModelAliases[selectedModel] || "",
      upstreamName: selectedModel,
      provider: selectedModel.includes("anthropic/") ? "Claude" : selectedModel.includes("deepseek/") ? "DeepSeek" : "GPT",
      available: true,
      inputPricePer1m: 0,
      outputPricePer1m: 0,
      pricingSource: "current",
      creditEstimate: modelCreditEstimates[selectedModel] || modelCreditEstimates["openai/gpt-4o-mini"],
    });
  }
  return options;
}

export const defaultSettingsDraft = {
  systemPrompt: "",
  escalationPrompt: "",
  fallbackWaitingMessage: "",
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
  allowAutoUpdateContactName: false,
  onlyFillNameIfEmpty: true,
  customerMemoryEnabled: false,
  customerMemoryAutoSaveEnabled: false,
  customerMemoryAdminNotesEnabled: false,
  customerMemoryAiExtractionEnabled: false,
  customerMemoryVerifierEnabled: true,
  customerMemoryMaxItems: 5,
  customerMemoryMaxChars: 800,
  customerMemoryRetentionDays: 180,
};

export const simulationScenarios = [
  { id: "answer_followup", label: "Customer Follow-up", text: "Halo kak, saya mau tanya status pesanan saya dan apakah bisa dibantu hari ini?", forceDecision: "answer" },
  { id: "escalate_salary", label: "Pricing Escalation", text: "Saya mau tanya diskon khusus dan minta kepastian harga final untuk paket ini.", forceDecision: "escalate" },
  { id: "neutral_auto", label: "Neutral Auto AI", text: "Halo, saya ingin tahu detail produk, harga, dan cara order.", forceDecision: "" },
];

export function createPlaygroundWelcome() {
  return {
    id: "playground-welcome",
    role: "assistant",
    text: "Halo. Saya AI Playground dashboard. Ketik pertanyaan sebagai pelanggan, nanti jawaban AI muncul di sini tanpa mengirim apa pun ke WhatsApp.",
    createdAt: new Date().toISOString(),
    meta: "Dashboard-only test",
  };
}

export function buildPlaygroundHistory(messages) {
  return messages
    .filter((message) => message.id !== "playground-welcome")
    .filter((message) => message.role === "user" || message.role === "assistant")
    .filter((message) => message.decision !== "error")
    .map((message) => ({
      role: message.role,
      text: String(message.text || "").slice(0, 1200),
    }))
    .filter((message) => message.text.trim())
    .slice(-10);
}

export const Icons = {
  overview: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><rect x="3" y="3" width="7" height="7" /><rect x="14" y="3" width="7" height="7" /><rect x="14" y="14" width="7" height="7" /><rect x="3" y="14" width="7" height="7" /></svg>,
  inbox: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z" /><polyline points="22,6 12,13 2,6" /></svg>,
  contacts: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" /><circle cx="9" cy="7" r="4" /><path d="M22 21v-2a4 4 0 0 0-3-3.87" /><path d="M16 3.13a4 4 0 0 1 0 7.75" /></svg>,
  knowledge: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2z" /><path d="M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z" /></svg>,
  settings: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="12" r="3" /><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z" /></svg>,
  analytics: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><line x1="18" y1="20" x2="18" y2="10" /><line x1="12" y1="20" x2="12" y2="4" /><line x1="6" y1="20" x2="6" y2="14" /></svg>,
  spark: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="m12 3-1.9 5.1L5 10l5.1 1.9L12 17l1.9-5.1L19 10l-5.1-1.9L12 3Z" /><path d="M5 3v4" /><path d="M19 17v4" /><path d="M3 5h4" /><path d="M17 19h4" /></svg>,
  team: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" /><circle cx="9" cy="7" r="4" /><path d="M23 21v-2a4 4 0 0 0-3-3.87" /><path d="M16 3.13a4 4 0 0 1 0 7.75" /></svg>,
  products: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M21 8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16Z" /><path d="m3.3 7 8.7 5 8.7-5" /><path d="M12 22V12" /></svg>,
  orders: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M9 5h6" /><path d="M9 12h6" /><path d="M9 19h6" /><path d="M5 5h.01" /><path d="M5 12h.01" /><path d="M5 19h.01" /><rect x="3" y="2" width="18" height="20" rx="2" /></svg>,
  ticket: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M3 9a3 3 0 0 0 0 6v3a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-3a3 3 0 0 0 0-6V6a2 2 0 0 0-2-2H5a2 2 0 0 0-2 2v3Z" /><path d="M13 5v2" /><path d="M13 17v2" /><path d="M13 11v2" /></svg>,
  calendar: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><rect x="3" y="4" width="18" height="18" rx="2" /><path d="M16 2v4" /><path d="M8 2v4" /><path d="M3 10h18" /></svg>,
  wallet: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M21 12V7H5a2 2 0 0 1 0-4h14v4" /><path d="M3 5v14a2 2 0 0 0 2 2h16v-5" /><path d="M18 12a2 2 0 0 0 0 4h4v-4Z" /></svg>,
  purchase: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="9" cy="21" r="1" /><circle cx="20" cy="21" r="1" /><path d="M1 1h4l2.68 13.39a2 2 0 0 0 2 1.61h9.72a2 2 0 0 0 2-1.61L23 6H6" /></svg>,
  usage: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" /><polyline points="14 2 14 8 20 8" /><line x1="16" y1="13" x2="8" y2="13" /><line x1="16" y1="17" x2="8" y2="17" /></svg>,
  health: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M22 12h-4l-3 9L9 3l-3 9H2" /></svg>,
  whatsapp: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M21 11.5a8.38 8.38 0 0 1-.9 3.8 8.5 8.5 0 0 1-7.6 4.7 8.38 8.38 0 0 1-3.8-.9L3 21l1.9-5.7a8.38 8.38 0 0 1-.9-3.8 8.5 8.5 0 0 1 4.7-7.6 8.38 8.38 0 0 1 3.8-.9h.5a8.48 8.48 0 0 1 8 8v.5z" /></svg>,
  account: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="8" r="4" /><path d="M4 21a8 8 0 0 1 16 0" /></svg>,
  more: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="5" cy="12" r="1.5" /><circle cx="12" cy="12" r="1.5" /><circle cx="19" cy="12" r="1.5" /></svg>,
  refresh: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="23 4 23 10 17 10" /><polyline points="1 20 1 14 7 14" /><path d="M3.51 9a9 9 0 0 1 14.13-3.36L23 10" /><path d="M20.49 15a9 9 0 0 1-14.13 3.36L1 14" /></svg>,
  sun: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="12" r="4" /><path d="M12 2v2" /><path d="M12 20v2" /><path d="m4.93 4.93 1.41 1.41" /><path d="m17.66 17.66 1.41 1.41" /><path d="M2 12h2" /><path d="M20 12h2" /><path d="m6.34 17.66-1.41 1.41" /><path d="m19.07 4.93-1.41 1.41" /></svg>,
  moon: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M12 3a6.7 6.7 0 0 0 8.7 8.7 9 9 0 1 1-8.7-8.7Z" /></svg>,
  search: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="11" cy="11" r="8" /><line x1="21" y1="21" x2="16.65" y2="16.65" /></svg>,
  plus: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><line x1="12" y1="5" x2="12" y2="19" /><line x1="5" y1="12" x2="19" y2="12" /></svg>,
  send: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><line x1="22" y1="2" x2="11" y2="13" /><polygon points="22 2 15 22 11 13 2 9 22 2" /></svg>,
  upload: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" /><polyline points="17 8 12 3 7 8" /><line x1="12" y1="3" x2="12" y2="15" /></svg>,
  external: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6" /><polyline points="15 3 21 3 21 9" /><line x1="10" y1="14" x2="21" y2="3" /></svg>,
  bell: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M18 8a6 6 0 0 0-12 0c0 7-3 8-3 8h18s-3-1-3-8" /><path d="M13.73 21a2 2 0 0 1-3.46 0" /></svg>,
  alert: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="12" r="10" /><line x1="12" y1="8" x2="12" y2="12" /><line x1="12" y1="16" x2="12.01" y2="16" /></svg>,
  check: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M20 6 9 17l-5-5" /></svg>,
  close: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" /></svg>,
  edit: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M12 20h9" /><path d="M16.5 3.5a2.12 2.12 0 0 1 3 3L7 19l-4 1 1-4Z" /></svg>,
  trash: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="3 6 5 6 21 6" /><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6" /><path d="M10 11v6" /><path d="M14 11v6" /><path d="M9 6V4a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2" /></svg>,
  lock: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><rect x="3" y="11" width="18" height="11" rx="2" /><path d="M7 11V7a5 5 0 0 1 10 0v4" /></svg>,
  robot: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><rect x="3" y="11" width="18" height="10" rx="2" /><circle cx="12" cy="5" r="2" /><path d="M12 7v4" /><line x1="8" y1="16" x2="8" y2="16" /><line x1="16" y1="16" x2="16" y2="16" /></svg>,
  help: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="12" r="10" /><path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3" /><line x1="12" y1="17" x2="12.01" y2="17" /></svg>,
  arrowRight: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><line x1="5" y1="12" x2="19" y2="12" /><polyline points="12 5 19 12 12 19" /></svg>,
  chevronDown: <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polyline points="6 9 12 15 18 9" /></svg>,
};

export function formatDateTime(dateString) {
  if (!dateString) return "-";
  const d = new Date(dateString);
  return d.toLocaleString("id-ID", {
    day: "2-digit",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }) + " WIB";
}

export function formatTime(dateString) {
  if (!dateString) return "-";
  return new Date(dateString).toLocaleString("id-ID", { hour: "2-digit", minute: "2-digit" }) + " WIB";
}

export function formatNumber(value) {
  return new Intl.NumberFormat("id-ID").format(Number(value) || 0);
}

export function formatPercent(numerator, denominator) {
  const base = Number(denominator) || 0;
  if (!base) return "0%";
  const percent = Math.round(((Number(numerator) || 0) / base) * 100);
  return `${Math.min(100, Math.max(0, percent))}%`;
}

export function formatCurrencyIDR(value) {
  return new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(Number(value) || 0);
}

export function formatCostIDR(value) {
  return new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 2 }).format(Number(value) || 0);
}

export function truncateText(text, max = 120) {
  if (!text) return "";
  return text.length > max ? `${text.slice(0, max)}...` : text;
}

export function fileToBase64(file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      const value = String(reader.result || "");
      resolve(value.includes(",") ? value.split(",", 2)[1] : value);
    };
    reader.onerror = () => reject(reader.error || new Error("Gagal membaca file."));
    reader.readAsDataURL(file);
  });
}

export function mediaKindFromMime(mimeType) {
  const normalized = String(mimeType || "").toLowerCase();
  if (normalized.startsWith("image/")) return "image";
  if (normalized.startsWith("audio/")) return "audio";
  return "document";
}

export function isEditingFormControl() {
  if (typeof document === "undefined") return false;
  const active = document.activeElement;
  if (!active) return false;
  const tagName = active.tagName?.toLowerCase();
  return tagName === "input" || tagName === "textarea" || tagName === "select" || active.isContentEditable;
}

export function toLabel(value) {
  if (!value) return "-";
  const key = String(value).toLowerCase();
  const labels = {
    active: "Aktif",
    inactive: "Nonaktif",
    connected: "Terhubung",
    disconnected: "Terputus",
    qr_pending: "Menunggu scan QR",
    open: "Terbuka",
    pending: "Menunggu",
    pending_human: "Menunggu admin",
    ready: "Tersedia",
    requested: "Menunggu approval",
    confirmed: "Terkonfirmasi",
    cancelled: "Batal",
    completed: "Selesai",
    draft: "Draft",
    done: "Selesai",
    resolved: "Selesai",
    won: "Menang",
    lost: "Kalah",
    archived: "Arsip",
    published: "Aktif",
  };
  return labels[key] || String(value).replace(/_/g, " ").replace(/\b\w/g, (char) => char.toUpperCase());
}

export function toRoleLabel(role) {
  if (!role) return "-";
  if (role === "super_admin") return "Admin Utama";
  if (role === "operator") return "Agent";
  return toLabel(role);
}

export function roleInitials(role) {
  if (role === "owner") return "OW";
  if (role === "super_admin") return "AU";
  if (role === "admin") return "AD";
  if (role === "operator") return "AG";
  return "BP";
}

export function renderChatMessage(text) {
  const normalized = String(text || "")
    .replace(/\r\n/g, "\n")
    .replace(/\s+(\d+)\.\s+/g, "\n$1. ")
    .trim();
  if (!normalized) return <p>(No text payload)</p>;

  const blocks = normalized.split(/\n{2,}/).filter(Boolean);
  return blocks.flatMap((block, blockIndex) => renderChatBlock(block, blockIndex));
}

export function renderMessageMedia(message) {
  const media = message?.rawPayload?.media || message?.rawPayload?.image;
  if (!media?.url) return null;
  const kind = media.kind || message.contentType;
  const label = media.fileName || (kind === "image" ? "image" : "attachment");
  if (kind === "image" || String(media.mimeType || "").startsWith("image/")) {
    return (
      <a href={media.url} target="_blank" rel="noreferrer" className="message-media-link">
        <img src={media.url} alt={label} className="message-media-image" />
      </a>
    );
  }
  return (
    <a href={media.url} target="_blank" rel="noreferrer" className="message-media-document">
      <span>{Icons.upload}</span>
      <span>{label}</span>
    </a>
  );
}

export function renderChatBlock(block, blockIndex) {
  const lines = block.split("\n").map((line) => line.trim()).filter(Boolean);
  if (!lines.length) return [];

  const output = [];
  let listItems = [];
  const flushList = () => {
    if (!listItems.length) return;
    const listStart = listItems[0].number;
    output.push(
      <ol
        key={`ol-${blockIndex}-${output.length}`}
        className="chat-rich-list"
        start={listStart}
      >
        {listItems.map((item, index) => (
          <li key={`${blockIndex}-li-${index}`}>{renderInlineRichText(item.text)}</li>
        ))}
      </ol>
    );
    listItems = [];
  };

  lines.forEach((line, lineIndex) => {
    const numbered = line.match(/^(\d+)\.\s+(.+)$/);
    if (numbered) {
      listItems.push({ number: Number(numbered[1]), text: numbered[2] });
      return;
    }
    flushList();
    output.push(
      <p key={`p-${blockIndex}-${lineIndex}`}>
        {renderInlineRichText(line)}
      </p>
    );
  });
  flushList();
  return output;
}

export function renderInlineRichText(text) {
  const pattern = /(https?:\/\/[^\s)]+)|(```[^`]+```)|(`[^`\n]+`)|(\*\*[^*]+\*\*)|(__[^_]+__)|(\*[^*\n]+\*)|(_[^_\n]+_)|(~[^~\n]+~)/g;
  const nodes = [];
  let cursor = 0;
  let match;

  while ((match = pattern.exec(text)) !== null) {
    if (match.index > cursor) nodes.push(text.slice(cursor, match.index));
    const raw = match[0];
    const key = `${match.index}-${raw}`;
    if (raw.startsWith("http")) {
      const trailing = raw.match(/[.,!?]+$/)?.[0] ?? "";
      const href = trailing ? raw.slice(0, -trailing.length) : raw;
      nodes.push(
        <a key={key} href={href} target="_blank" rel="noreferrer">
          {href}
        </a>
      );
      if (trailing) nodes.push(trailing);
    } else if (raw.startsWith("```")) {
      nodes.push(<code key={key}>{raw.slice(3, -3)}</code>);
    } else if (raw.startsWith("`")) {
      nodes.push(<code key={key}>{raw.slice(1, -1)}</code>);
    } else if (raw.startsWith("**") || raw.startsWith("__")) {
      nodes.push(<strong key={key}>{raw.slice(2, -2)}</strong>);
    } else if (raw.startsWith("*")) {
      nodes.push(<strong key={key}>{raw.slice(1, -1)}</strong>);
    } else if (raw.startsWith("_")) {
      nodes.push(<em key={key}>{raw.slice(1, -1)}</em>);
    } else if (raw.startsWith("~")) {
      nodes.push(<s key={key}>{raw.slice(1, -1)}</s>);
    }
    cursor = pattern.lastIndex;
  }

  if (cursor < text.length) nodes.push(text.slice(cursor));
  return nodes;
}

export function getInitials(name) {
  if (!name) return "?";
  return name.split(" ").filter(Boolean).map((item) => item[0]).join("").slice(0, 2).toUpperCase();
}

export function getAvatarColor(name) {
  const colors = [
    { background: "var(--avatar-red-bg)", color: "var(--avatar-red-text)" },
    { background: "var(--avatar-orange-bg)", color: "var(--avatar-orange-text)" },
    { background: "var(--avatar-yellow-bg)", color: "var(--avatar-yellow-text)" },
    { background: "var(--avatar-green-bg)", color: "var(--avatar-green-text)" },
    { background: "var(--avatar-blue-bg)", color: "var(--avatar-blue-text)" },
    { background: "var(--avatar-violet-bg)", color: "var(--avatar-violet-text)" },
  ];
  if (!name) return colors[0];
  let hash = 0;
  for (let index = 0; index < name.length; index += 1) {
    hash = name.charCodeAt(index) + ((hash << 5) - hash);
  }
  return colors[Math.abs(hash) % colors.length];
}

export function settingsApiToDraft(settings) {
  if (!settings) return defaultSettingsDraft;
  return {
    systemPrompt: settings.system_prompt ?? "",
    escalationPrompt: settings.escalation_prompt ?? "",
    fallbackWaitingMessage: settings.fallback_waiting_message ?? "",
    allowClarification: settings.allow_clarification !== false,
    maxClarificationCount: Number(settings.max_clarification_count ?? 1),
    answerOnlyFromKnowledge: settings.answer_only_from_knowledge !== false,
    dontBroadenTopic: settings.dont_broaden_topic !== false,
    forbidPromises: settings.forbid_promises !== false,
    forbidSensitiveAnswers: settings.forbid_sensitive_answers !== false,
    requireActionConfirmation: settings.require_action_confirmation !== false,
    escalateLowConfidence: settings.escalate_low_confidence !== false,
    guideNextStep: settings.guide_next_step !== false,
    conciseResponse: settings.concise_response !== false,
    allowAutoUpdateContactName: Boolean(settings.allow_auto_update_contact_name),
    onlyFillNameIfEmpty: settings.only_fill_name_if_empty !== false,
    customerMemoryEnabled: Boolean(settings.customer_memory_enabled),
    customerMemoryAutoSaveEnabled: Boolean(settings.customer_memory_auto_save_enabled),
    customerMemoryAdminNotesEnabled: Boolean(settings.customer_memory_admin_notes_enabled),
    customerMemoryAiExtractionEnabled: Boolean(settings.customer_memory_ai_extraction_enabled),
    customerMemoryVerifierEnabled: settings.customer_memory_verifier_enabled !== false,
    customerMemoryMaxItems: Number(settings.customer_memory_max_items ?? 5),
    customerMemoryMaxChars: Number(settings.customer_memory_max_chars ?? 800),
    customerMemoryRetentionDays: Number(settings.customer_memory_retention_days ?? 180),
  };
}

export function dashboardBadge(viewId, { summary, inbox, purchases, health }) {
  if (viewId === "operations") return inbox.reduce((total, item) => total + Number(item.unreadCount || 0), 0);
  if (viewId === "wallet" || viewId === "upgrade") return purchases.filter((item) => ["requested", "pending"].includes(item.paymentStatus)).length;
  if (viewId === "health") return health?.systemAlerts?.length ?? health?.alerts?.length ?? 0;
  return 0;
}

export const authStorageKey = "oneflow-auth";
export const activeViewStorageKey = "oneflow-active-view";
export const loginNextStorageKey = "oneflow-login-next";
export const waModalDismissedStorageKey = "oneflow-wa-modal-dismissed";
export const legacyAuthStorageKey = "hrchat-auth";
export const legacyActiveViewStorageKey = "hrchat-active-view";
export const legacyLoginNextStorageKey = "hrchat-login-next";
export const legacyWaModalDismissedStorageKey = "hrchat-wa-modal-dismissed";

export function readAndMigrateStorageValue(storage, key, legacyKey) {
  const current = storage.getItem(key);
  if (current !== null) return current;
  const legacy = storage.getItem(legacyKey);
  if (legacy !== null) {
    storage.setItem(key, legacy);
    storage.removeItem(legacyKey);
  }
  return legacy;
}
export const viewRouteSegments = {
  operations: "inbox",
  knowledge: "knowledge",
  businessTools: "alat-bisnis",
  deals: "pipeline",
  tickets: "ticketing",
  commerceProducts: "produk-stok",
  commerceOrders: "pesanan",
  booking: "booking",
  team: "user-management",
  settings: "ai-settings",
  appsettings: "pricing",
  usage: "usage-logs",
  upgrade: "upgrade",
};
export const segmentViewIds = Object.fromEntries(Object.entries(viewRouteSegments).map(([viewId, segment]) => [segment, viewId]));
segmentViewIds.team = "team";
segmentViewIds.purchases = "wallet";
segmentViewIds.logs = "usage";
segmentViewIds["usage-log"] = "usage";
segmentViewIds["owner-log"] = "usage";
segmentViewIds["owner-logs"] = "usage";

export const setupRouteConfigs = {
  "/agents/setup": { view: "agents", tab: "instructions", anchor: "create-agent" },
  "/playground": { view: "playground" },
  "/whatsapp": { view: "whatsapp", anchor: "connect-whatsapp" },
  "/channels": { view: "whatsapp", anchor: "connect-whatsapp" },
  "/whatsapp/templates": { view: "whatsappTemplates" },
  "/escalation": { view: "whatsapp", anchor: "bind-escalation" },
};

export function normalizeDashboardPath(pathname) {
  if (!pathname) return "";
  const cleanPath = pathname.split("?")[0].replace(/\/+$/, "");
  return cleanPath || "/";
}

export function setupRouteConfigFromPath(pathname) {
  return setupRouteConfigs[normalizeDashboardPath(pathname)] ?? null;
}

export function canonicalViewId(viewId) {
  if (viewId === "purchase") return "wallet";
  if (viewId === "channels") return "whatsapp";
  return viewId;
}

export function segmentForView(viewId) {
  return viewRouteSegments[viewId] ?? viewId;
}

export function pathForView(viewId) {
  if (viewId === "playground") return "/playground";
  if (viewId === "whatsapp") return "/channels";
  if (viewId === "whatsappTemplates") return "/whatsapp/templates";
  if (viewId === "dashboard") return "/dashboard";
  return `/dashboard/${segmentForView(viewId)}`;
}

export function isDashboardAppPath(pathname) {
  const normalized = normalizeDashboardPath(pathname);
  return normalized.startsWith("/dashboard") || Boolean(setupRouteConfigFromPath(normalized));
}

export function viewFromPath(pathname, role) {
  const setupRoute = setupRouteConfigFromPath(pathname);
  if (setupRoute) {
    const viewId = canonicalViewId(setupRoute.view);
    return isViewAllowedForRole(role, viewId) ? viewId : defaultViewForRole(role);
  }
  if (!pathname?.startsWith("/dashboard")) return "";
  const segment = pathname.split("/").filter(Boolean)[1] || "";
  if (!segment) return defaultViewForRole(role);
  const viewId = canonicalViewId(segmentViewIds[segment] ?? segment);
  return isViewAllowedForRole(role, viewId) ? viewId : defaultViewForRole(role);
}

export function requestedDashboardPath() {
  if (typeof window === "undefined") return "";
  if (window.location.pathname === "/login") {
    const queryNext = new URLSearchParams(window.location.search).get("next") || "";
    if (queryNext) return queryNext;
    try {
      return readAndMigrateStorageValue(window.sessionStorage, loginNextStorageKey, legacyLoginNextStorageKey) || "";
    } catch {
      return "";
    }
  }
  return window.location.pathname;
}

export function defaultViewForRole(role) {
  return role === "owner" ? "overview" : "operations";
}

export function isViewAllowedForRole(role, viewId) {
  const canonical = canonicalViewId(viewId);
  if (canonical === "account") return Boolean(role);
  if (canonical === "playground") return ["super_admin", "admin", "operator"].includes(role);
  if (canonical === "whatsappTemplates") return ["super_admin", "admin", "operator"].includes(role);
  return (roleNav[role] ?? []).some((item) => item.id === canonical);
}

export function storedViewForRole(role) {
  const fallback = defaultViewForRole(role);
  if (typeof window === "undefined") return fallback;
  const stored = canonicalViewId(readAndMigrateStorageValue(window.localStorage, activeViewStorageKey, legacyActiveViewStorageKey));
  return isViewAllowedForRole(role, stored) ? stored : fallback;
}
