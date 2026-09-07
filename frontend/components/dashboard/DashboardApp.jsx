"use client";

import { useCallback, useDeferredValue, useEffect, useMemo, useRef, useState } from "react";
import { usePathname, useRouter } from "next/navigation";
import QRCode from "qrcode";
import { parseCoexistenceEvent, validateCoexistenceConfig } from "../../lib/meta-coexistence.mjs";
import { LoginScreen, Sidebar, Topbar } from "./shell";
import { Notice } from "./ui";
import {
  AccountSettingsView,
  AgentsView,
  AISettingsView,
  AnalyticsView,
  BusinessToolsView,
  BookingView,
  ContactsView,
  CommerceOrdersView,
  CommerceProductsView,
  DealsView,
  HealthView,
  InboxView,
  KnowledgeView,
  OwnerOverviewView,
  PerformanceDashboardView,
  PlaygroundView,
  PricingSettingsView,
  TeamManagementView,
  TicketsView,
  UsageLogsView,
  WalletView,
  WhatsAppConnectionView
} from "./views";
import {
  apiBase,
  wsBase,
  waBase,
  roleNav,
  viewMeta,
  defaultModelAliases,
  defaultAvailableModelIds,
  analyticsRangeOptions,
  composerEmojis,
  defaultSettingsDraft,
  simulationScenarios,
  createPlaygroundWelcome,
  buildPlaygroundHistory,
  Icons,
  formatDateTime,
  formatTime,
  formatNumber,
  formatPercent,
  formatCurrencyIDR,
  formatCostIDR,
  truncateText,
  fileToBase64,
  mediaKindFromMime,
  isEditingFormControl,
  toRoleLabel,
  roleInitials,
  renderChatMessage,
  renderMessageMedia,
  renderChatBlock,
  renderInlineRichText,
  getInitials,
  getAvatarColor,
  settingsApiToDraft,
  dashboardBadge,
  authStorageKey,
  activeViewStorageKey,
  loginNextStorageKey,
  legacyAuthStorageKey,
  legacyActiveViewStorageKey,
  readAndMigrateStorageValue,
  viewRouteSegments,
  segmentViewIds,
  canonicalViewId,
  isDashboardAppPath,
  segmentForView,
  setupRouteConfigFromPath,
  pathForView,
  viewFromPath,
  requestedDashboardPath,
  defaultViewForRole,
  isViewAllowedForRole,
  storedViewForRole
} from "../../lib/dashboard-core";

const fallbackWaitingMessage = "Pesan Anda sudah kami teruskan ke tim kami. Mohon tunggu sebentar ya.";
const legacyOperatorRole = "hr_agent";
const firstRunOnboardingStorageKeyBase = "oneflow-first-run-onboarding-v1";
const sessionExpiredStorageKey = "oneflow-session-expired";
const dashboardThemeStorageKey = "oneflow-dashboard-theme";
const midtransSnapScripts = {
  sandbox: "https://app.sandbox.midtrans.com/snap/snap.js",
  production: "https://app.midtrans.com/snap/snap.js",
};
const mobileOperationsViews = {
  owner: ["overview", "wallet", "usage", "health", "account", "whatsapp"],
  super_admin: ["operations", "contacts", "agents", "playground", "upgrade", "deals", "tickets", "commerceOrders", "booking", "account", "whatsapp"],
  admin: ["operations", "contacts", "agents", "playground", "upgrade", "deals", "tickets", "commerceOrders", "booking", "account", "whatsapp"],
  operator: ["operations", "contacts", "agents", "playground", "upgrade", "deals", "tickets", "commerceOrders", "booking", "account", "whatsapp"],
};
const mobileRestrictedGuidance = {
  agents: {
    title: "Setup lengkap agent tersedia di desktop",
    text: "Mode mobile tetap bisa cek agent dan status cepat. Upload knowledge besar dan setup panjang lebih aman dilakukan dari desktop.",
  },
  playground: {
    title: "Debug Playground lengkap tersedia di desktop",
    text: "Mode mobile dipakai untuk test chat ringan. Batch test dan audit detail lebih nyaman dari desktop.",
  },
  whatsapp: {
    title: "Koneksi WhatsApp tersedia di desktop",
    text: "Hubungkan WhatsApp Business melalui Meta Coexistence.",
  },
  businessTools: {
    title: "Setup Alat Bisnis tersedia di desktop",
    text: "Pemasangan alat, izin AI, dan pemetaan data butuh ruang review yang lebih lengkap.",
  },
  commerceProducts: {
    title: "Produk & Stok lengkap tersedia di desktop",
    text: "Kelola katalog dan stok besar dari desktop. Di mobile, fokusnya cek pesanan dan update status cepat.",
  },
  team: {
    title: "Team management tersedia di desktop",
    text: "Perubahan role, invite, dan reset password memerlukan ruang review yang lebih lengkap.",
  },
  analytics: {
    title: "Analytics lengkap tersedia di desktop",
    text: "Chart dan tabel analitik tetap dijaga utuh di desktop agar pembacaan data tidak terpotong.",
  },
  usage: {
    title: "Usage log lengkap tersedia di desktop",
    text: "Audit pemakaian, detail cost, dan inspeksi run lebih aman dibuka dengan tabel penuh.",
  },
  health: {
    title: "Health panel lengkap tersedia di desktop",
    text: "Status teknis dan alert sistem tetap bisa direview lebih jelas lewat desktop.",
  },
  appsettings: {
    title: "Pricing tersedia di desktop",
    text: "Perubahan model, pricing, dan adjustment kredit butuh ruang validasi yang lebih lengkap.",
  },
  settings: {
    title: "AI settings tersedia di desktop",
    text: "Policy dan guardrail AI lebih aman diedit dari desktop supaya konfigurasi tidak terlewat.",
  },
};
const firstRunOnboardingSlides = [
  {
    id: "setup-flow",
    icon: "robot",
    title: "Mulai dari AI agent",
    text: "Buat satu AI CS, pilih model, lalu tentukan gaya jawab dan aturan eskalasi. Akun baru punya kredit trial untuk tes di Playground sebelum masuk WhatsApp produksi.",
    points: ["Buat AI CS utama", "Atur behavior jawaban", "Tes dengan kredit trial"],
    visual: "agent",
  },
  {
    id: "knowledge",
    icon: "knowledge",
    title: "Isi knowledge bisnis",
    text: "Tambahkan FAQ dan dokumen supaya agent menjawab dari informasi yang benar, bukan menebak. Produk, stok, pesanan, layanan, slot, dan booking masuk dari Alat Bisnis.",
    points: ["FAQ untuk pertanyaan berulang", "Dokumen manual atau upload", "Produk dan layanan real-time di Alat Bisnis"],
    visual: "knowledge",
  },
  {
    id: "playground",
    icon: "spark",
    title: "Tes sebelum dipakai pelanggan",
    text: "Pakai Playground untuk simulasi customer. Tes ini tidak mengirim WhatsApp, tapi bisa membuat data test seperti draft pesanan atau booking jika izin AI untuk alat bisnis aktif.",
    points: ["Coba skenario customer", "Cek jawaban dan aksi bisnis", "Bersihkan data test sebelum live"],
    visual: "playground",
  },
  {
    id: "business-tools",
    icon: "settings",
    title: "Install Alat Bisnis",
    text: "Buka Alat Bisnis untuk install Follow-up, Produk/Stok/Pesanan, atau Booking. Setelah installed, menunya muncul terpisah dari fitur utama.",
    points: ["Klik Detail untuk cek fitur", "Klik Install untuk munculkan menu", "Customize alat per bisnis"],
    visual: "businessTools",
  },
  {
    id: "production",
    icon: "whatsapp",
    title: "Aktifkan operasional",
    text: "Setelah siap, upgrade paket untuk membuka WhatsApp production, inbox tim, kontak, eskalasi, dan analytics.",
    points: ["Connect WhatsApp dengan QR", "Pantau Inbox dan Contacts", "Lihat performa di Analytics"],
    visual: "operations",
  },
];

function rememberLoginNextPath(pathname) {
  if (typeof window === "undefined" || !isDashboardAppPath(pathname)) return;
  try {
    window.sessionStorage.setItem(loginNextStorageKey, pathname);
  } catch {}
}

function normalizeStoredAuthRole(auth) {
  if (auth?.user?.role !== legacyOperatorRole) return auth;
  return { ...auth, user: { ...auth.user, role: "operator" } };
}

function clearLoginNextPath() {
  if (typeof window === "undefined") return;
  try {
    window.sessionStorage.removeItem(loginNextStorageKey);
  } catch {}
}

function useMobileOperationsMode() {
  const [isMobile, setIsMobile] = useState(false);

  useEffect(() => {
    if (typeof window === "undefined") return undefined;
    const query = window.matchMedia("(max-width: 760px)");
    const update = () => setIsMobile(query.matches);
    update();
    if (typeof query.addEventListener === "function") {
      query.addEventListener("change", update);
      return () => query.removeEventListener("change", update);
    }
    query.addListener(update);
    return () => query.removeListener(update);
  }, []);

  return isMobile;
}

function isMobileOperationsView(role, viewId) {
  const allowed = mobileOperationsViews[role] || mobileOperationsViews.operator;
  return allowed.includes(canonicalViewId(viewId));
}

function mobileNavForRole(role, items) {
  return items.filter((item) => isMobileOperationsView(role, item.id));
}

function onboardingStartViewForRole(role) {
  if (["super_admin", "admin", "operator"].includes(role)) return "agents";
  return defaultViewForRole(role);
}

const pluginViewModules = {
  deals: "prospects",
  tickets: "tickets",
  commerceProducts: "commerce",
  commerceOrders: "commerce",
  booking: "booking",
};

function emptyCommerceProductForm() {
  return { id: "", sku: "", name: "", description: "", unitPrice: "", stockQuantity: "", lowStockThreshold: "", status: "active" };
}

function emptyPackageForm(overrides = {}) {
  return {
    id: "",
    name: "",
    description: "",
    creditAmount: "5000",
    price: "2500000",
    billingPeriod: "monthly",
    maxWhatsAppSessions: "1",
    maxAiAgents: "1",
    maxHumanUsers: "1",
    isActive: true,
    isPopular: false,
    ...overrides,
  };
}

function packageFormFromItem(item = {}) {
  return emptyPackageForm({
    id: item.id || "",
    name: item.name || "",
    description: item.description || "",
    creditAmount: String(item.creditAmount ?? "5000"),
    price: String(item.price ?? "2500000"),
    billingPeriod: item.billingPeriod || "monthly",
    maxWhatsAppSessions: String(item.maxWhatsAppSessions ?? "1"),
    maxAiAgents: String(item.maxAiAgents ?? "1"),
    maxHumanUsers: String(item.maxHumanUsers ?? "1"),
    isActive: item.isActive !== false,
    isPopular: item.isPopular === true || item.is_popular === true,
  });
}

function emptyCommerceOrderDraftForm() {
  return {
    customerName: "",
    customerPhone: "",
    stageId: "",
    fulfillmentType: "pickup",
    recipientName: "",
    recipientPhone: "",
    addressLine: "",
    addressArea: "",
    addressNotes: "",
    notes: "",
    items: [{ productId: "", quantity: "1" }],
  };
}

function defaultCommerceOrderSettings() {
  return { enabledFulfillmentTypes: ["pickup", "delivery", "shipping", "digital", "onsite_service"] };
}

function emptyBookingServiceForm() {
  return {
    id: "",
    name: "",
    description: "",
    durationMinutes: "30",
    bufferMinutes: "0",
    price: "0",
    timezone: "Asia/Jakarta",
    availabilityDays: "1,2,3,4,5",
    availabilityStart: "09:00",
    availabilityEnd: "17:00",
    status: "active",
  };
}

function emptyBookingAppointmentForm() {
  return { serviceId: "", customerName: "", customerPhone: "", scheduledStart: "", locationType: "business_location", locationAddress: "", locationNotes: "", notes: "" };
}

function availabilityDaysFromInput(value) {
  const days = String(value || "")
    .split(",")
    .map((item) => Number.parseInt(item.trim(), 10))
    .filter((item) => Number.isInteger(item) && item >= 0 && item <= 6);
  return days.length ? Array.from(new Set(days)) : [1, 2, 3, 4, 5];
}

function bookingServiceFormFromItem(item) {
  const availability = item?.availability || {};
  const days = Array.isArray(availability.days) ? availability.days.join(",") : "1,2,3,4,5";
  return {
    id: item?.id || "",
    name: item?.name || "",
    description: item?.description || "",
    durationMinutes: String(item?.durationMinutes ?? 30),
    bufferMinutes: String(item?.bufferMinutes ?? 0),
    price: String(item?.price ?? 0),
    timezone: item?.timezone || "Asia/Jakarta",
    availabilityDays: days,
    availabilityStart: availability.start || "09:00",
    availabilityEnd: availability.end || "17:00",
    status: item?.status || "active",
  };
}

function mobileGateShortcutIds(viewId) {
  const canonical = canonicalViewId(viewId);
  if (canonical === "businessTools") return ["deals", "tickets", "commerceOrders", "booking", "operations", "contacts"];
  if (canonical === "commerceProducts") return ["commerceOrders", "booking", "tickets", "deals", "operations", "contacts"];
  return [];
}

function MobileDesktopGate({ activeView, activeMeta, navItems, onNavigate }) {
  const canonical = canonicalViewId(activeView);
  const preferredShortcutIds = mobileGateShortcutIds(canonical);
  const preferredShortcuts = preferredShortcutIds
    .map((id) => navItems.find((item) => item.id === id))
    .filter(Boolean);
  const shortcutItems = [
    ...preferredShortcuts,
    ...navItems.filter((item) => !preferredShortcutIds.includes(item.id)),
  ].slice(0, 6);
  const guidance = mobileRestrictedGuidance[canonical] || {
    title: `${activeMeta?.title || "Halaman ini"} tersedia di desktop`,
    text: "Mode mobile difokuskan untuk aksi cepat. Pengaturan lengkap tetap tersedia dari desktop.",
  };

  return (
    <section className="mobile-ops-gate">
      <div className="mobile-ops-gate-mark">{Icons.lock}</div>
      <div className="mobile-ops-gate-copy">
        <span>Mode mobile</span>
        <h2>{guidance.title}</h2>
        <p>{guidance.text}</p>
      </div>
      <div className="mobile-ops-shortcuts">
        {shortcutItems.map((item) => (
          <button key={item.id} type="button" onClick={() => onNavigate(item.id)}>
            {Icons[item.icon]}
            <span>{item.label}</span>
          </button>
        ))}
      </div>
    </section>
  );
}

function midtransEnvironment(value) {
  return String(value || "sandbox").toLowerCase() === "production" ? "production" : "sandbox";
}

function loadMidtransSnapScript(clientKey, environment) {
  if (typeof window === "undefined") {
    return Promise.reject(new Error("Midtrans hanya tersedia di browser."));
  }
  const env = midtransEnvironment(environment);
  const src = midtransSnapScripts[env];
  const existing = document.getElementById("midtrans-snap-script");
  if (window.snap && existing?.dataset.clientKey === clientKey && existing?.dataset.environment === env) {
    return Promise.resolve(window.snap);
  }
  if (window.__oneflowMidtransSnapPromise && existing?.dataset.clientKey === clientKey && existing?.dataset.environment === env) {
    return window.__oneflowMidtransSnapPromise;
  }
  if (existing) existing.remove();

  window.__oneflowMidtransSnapPromise = new Promise((resolve, reject) => {
    const script = document.createElement("script");
    const nonce = document.querySelector("script[nonce]")?.nonce || "";
    script.id = "midtrans-snap-script";
    script.src = src;
    script.async = true;
    if (nonce) script.nonce = nonce;
    script.dataset.clientKey = clientKey;
    script.dataset.environment = env;
    script.setAttribute("data-client-key", clientKey);
    script.onload = () => {
      if (window.snap) resolve(window.snap);
      else reject(new Error("Layanan pembayaran belum siap."));
    };
    script.onerror = () => reject(new Error("Gagal memuat layanan pembayaran."));
    document.body.appendChild(script);
  });
  return window.__oneflowMidtransSnapPromise;
}

function loadMetaSDK(appId, graphVersion) {
  if (typeof window === "undefined") {
    return Promise.reject(new Error("Koneksi WhatsApp resmi hanya tersedia di browser."));
  }
  if (window.FB) {
    window.FB.init({ appId, cookie: true, xfbml: true, version: graphVersion });
    return Promise.resolve(window.FB);
  }
  if (window.__oneflowMetaSDKPromise) return window.__oneflowMetaSDKPromise;
  window.__oneflowMetaSDKPromise = new Promise((resolve, reject) => {
    window.fbAsyncInit = () => {
      window.FB.init({ appId, cookie: true, xfbml: true, version: graphVersion });
      resolve(window.FB);
    };
    const script = document.createElement("script");
    script.id = "facebook-jssdk";
    script.src = "https://connect.facebook.net/id_ID/sdk.js";
    script.async = true;
    script.defer = true;
    script.onerror = () => reject(new Error("Gagal memuat koneksi WhatsApp resmi."));
    document.body.appendChild(script);
  });
  return window.__oneflowMetaSDKPromise;
}

function firstRunOnboardingStorageKeyForAuth(authLike) {
  const user = authLike?.user ?? authLike ?? {};
  const organizationId = user.organizationId || user.organization?.id || "no-org";
  const userId = user.id || user.username || "no-user";
  return `${firstRunOnboardingStorageKeyBase}:${organizationId}:${userId}`;
}

function setupSessionConnected(session) {
  return ["connected", "open", "ready"].includes(String(session?.status || "").toLowerCase());
}

function FirstRunOnboardingVisual({ type }) {
  const rows = {
    agent: ["Nama agent", "Model AI", "Aturan eskalasi"],
    knowledge: ["FAQ", "Dokumen", "Alat Bisnis"],
    playground: ["Pertanyaan test", "Jawaban AI", "Catatan revisi"],
    businessTools: ["Detail alat", "Install", "Menu terpisah"],
    operations: ["WhatsApp", "Inbox", "Analytics"],
  }[type] || [];

  return (
    <div className={`first-run-visual ${type}`} aria-hidden="true">
      <div className="first-run-visual-top">
        <span />
        <span />
        <span />
      </div>
      <div className="first-run-visual-body">
        <div className="first-run-visual-rail">
          {rows.map((item, index) => <span key={item} className={index === 0 ? "active" : ""}>{item}</span>)}
        </div>
        <div className="first-run-visual-panel">
          <div className="first-run-visual-icon">{Icons[type === "agent" ? "robot" : type === "knowledge" ? "knowledge" : type === "playground" ? "spark" : type === "businessTools" ? "settings" : "whatsapp"]}</div>
          <div className="first-run-visual-line strong" />
          <div className="first-run-visual-line" />
          <div className="first-run-visual-line short" />
        </div>
      </div>
    </div>
  );
}

function FirstRunOnboardingModal({ open, slides, index, onBack, onNext, onSkip, onStart }) {
  if (!open) return null;
  const slide = slides[index] || slides[0];
  const isLast = index >= slides.length - 1;

  return (
    <div className="first-run-overlay" role="presentation">
      <section className="first-run-modal" role="dialog" aria-modal="true" aria-labelledby="first-run-title">
        <div className="first-run-copy">
          <div className="first-run-kicker">Panduan awal</div>
          <h2 id="first-run-title">{slide.title}</h2>
          <p>{slide.text}</p>
          <ul className="first-run-points">
            {slide.points.map((point) => (
              <li key={point}>
                <span>{Icons.check}</span>
                {point}
              </li>
            ))}
          </ul>
        </div>

        <FirstRunOnboardingVisual type={slide.visual} />

        <div className="first-run-footer">
          <div className="first-run-progress" aria-label={`Langkah ${index + 1} dari ${slides.length}`}>
            {slides.map((item, itemIndex) => (
              <span key={item.id} className={itemIndex === index ? "active" : ""} />
            ))}
          </div>
          <div className="first-run-actions">
            <button className="btn btn-secondary" type="button" onClick={onSkip}>Lewati</button>
            <button className="btn btn-secondary" type="button" onClick={onBack} disabled={index === 0}>Kembali</button>
            <button className="btn btn-primary" type="button" onClick={isLast ? onStart : onNext}>
              {isLast ? "Mulai setup" : "Lanjut"} {Icons.arrowRight}
            </button>
          </div>
        </div>
      </section>
    </div>
  );
}

function defaultAIAgentForm() {
  return {
    id: "",
    name: "",
    modelName: "openai/gpt-4o-mini",
    systemPrompt: "",
    escalationPrompt: "",
    fallbackWaitingMessage,
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
    allowAutoUpdateContactName: true,
    onlyFillNameIfEmpty: true,
    ...customerMemoryFormBundle(false),
    customerMemoryMaxItems: 5,
    customerMemoryMaxChars: 800,
    customerMemoryRetentionDays: 180,
    isActive: true,
  };
}

function customerMemoryFormBundle(enabled) {
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

function aiAgentToForm(agent) {
  const memoryBundle = customerMemoryFormBundle(agent?.customerMemoryEnabled === true);
  return {
    ...defaultAIAgentForm(),
    id: agent?.id || "",
    name: agent?.name || "",
    modelName: agent?.modelName || "openai/gpt-4o-mini",
    systemPrompt: agent?.systemPrompt || "",
    escalationPrompt: agent?.escalationPrompt || "",
    fallbackWaitingMessage: agent?.fallbackWaitingMessage || fallbackWaitingMessage,
    allowClarification: agent?.allowClarification !== false,
    maxClarificationCount: Number(agent?.maxClarificationCount ?? 1),
    answerOnlyFromKnowledge: agent?.answerOnlyFromKnowledge !== false,
    dontBroadenTopic: agent?.dontBroadenTopic !== false,
    forbidPromises: agent?.forbidPromises !== false,
    forbidSensitiveAnswers: agent?.forbidSensitiveAnswers !== false,
    requireActionConfirmation: agent?.requireActionConfirmation !== false,
    escalateLowConfidence: agent?.escalateLowConfidence !== false,
    guideNextStep: agent?.guideNextStep !== false,
    conciseResponse: agent?.conciseResponse !== false,
    allowAutoUpdateContactName: agent?.allowAutoUpdateContactName !== false,
    onlyFillNameIfEmpty: agent?.onlyFillNameIfEmpty !== false,
    ...memoryBundle,
    customerMemoryMaxItems: Number(agent?.customerMemoryMaxItems ?? 5),
    customerMemoryMaxChars: Number(agent?.customerMemoryMaxChars ?? 800),
    customerMemoryRetentionDays: Number(agent?.customerMemoryRetentionDays ?? 180),
    isActive: agent?.isActive !== false,
  };
}

export default function DashboardApp() {
  const router = useRouter();
  const pathname = usePathname();
  const [auth, setAuth] = useState(null);
  const [authReady, setAuthReady] = useState(false);
  const [activeView, setActiveView] = useState("operations");
  const [analyticsRange, setAnalyticsRange] = useState("monthly");

  const [summary, setSummary] = useState(null);
  const [analytics, setAnalytics] = useState(null);
  const [health, setHealth] = useState(null);
  const [settings, setSettings] = useState(null);
  const [wallet, setWallet] = useState(null);
  const [billingPlan, setBillingPlan] = useState(null);
  const [packages, setPackages] = useState([]);
  const [purchases, setPurchases] = useState([]);
  const autoSyncedPurchaseIdsRef = useRef(new Set());
  const [faqs, setFaqs] = useState([]);
  const [documents, setDocuments] = useState([]);
  const [positions, setPositions] = useState([]);
  const [aiRuns, setAiRuns] = useState([]);
  const [inbox, setInbox] = useState([]);
  const [contacts, setContacts] = useState([]);
  const [notifications, setNotifications] = useState([]);
  const [notificationReadIds, setNotificationReadIds] = useState([]);
  const [selectedConversation, setSelectedConversation] = useState(null);
  const [focusContactId, setFocusContactId] = useState("");
  const [focusDealId, setFocusDealId] = useState("");
  const [waStatus, setWaStatus] = useState({ status: "disconnected", details: {} });
  const [organizations, setOrganizations] = useState([]);
  const [teamMembers, setTeamMembers] = useState([]);
  const [teamInvites, setTeamInvites] = useState([]);
  const [waSessions, setWaSessions] = useState([]);
  const [activeWaSessionId, setActiveWaSessionId] = useState("");
  const [pendingWaSessionId, setPendingWaSessionId] = useState("");
  const [metaTemplates, setMetaTemplates] = useState([]);
  const [metaCloudConfig, setMetaCloudConfig] = useState({ enabled: false });
  const [metaTemplateForm, setMetaTemplateForm] = useState({
    name: "oneflow_status_update",
    category: "UTILITY",
    language: "id",
    body: "Halo kak, ada pembaruan layanan dari tim kami."
  });
  const [metaTemplateSendForm, setMetaTemplateSendForm] = useState({ phone: "", name: "", language: "id" });
  const waSessionIdsRef = useRef(new Set());
  const [aiAgents, setAiAgents] = useState([]);
  const [activeKnowledgeAIAgentId, setActiveKnowledgeAIAgentId] = useState("");
  const activeKnowledgeAIAgentIdRef = useRef("");
  const appliedSetupRouteRef = useRef("");
  const [escalationGroup, setEscalationGroup] = useState(null);
  const [groupNotificationRules, setGroupNotificationRules] = useState([]);
  const [groupNotificationVariables, setGroupNotificationVariables] = useState([]);
  const [realtimeSnapshot, setRealtimeSnapshot] = useState(null);

  const [qrData, setQrData] = useState(null);
  const [qrImageSrc, setQrImageSrc] = useState("");

  const [error, setError] = useState("");
  const [loadWarnings, setLoadWarnings] = useState([]);
  const [loading, setLoading] = useState(false);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [lastSyncedAt, setLastSyncedAt] = useState("");
  const [busyKey, setBusyKey] = useState("");
  const [toast, setToast] = useState(null);
  const [dialog, setDialog] = useState(null);
  const [dialogInput, setDialogInput] = useState("");
  const [aiAgentSaveFeedback, setAIAgentSaveFeedback] = useState(null);
  const dialogResolverRef = useRef(null);
  const toastTimerRef = useRef(null);
  const handledExpiredTokenRef = useRef("");

  const [inboxTab, setInboxTab] = useState("all");
  const [inboxSearch, setInboxSearch] = useState("");
  const [activeInboxWaSessionId, setActiveInboxWaSessionId] = useState("");
  const [activeInboxAIAgentId, setActiveInboxAIAgentId] = useState("");
  const deferredInboxSearch = useDeferredValue(inboxSearch);
  const visibleWaSessions = useMemo(
    () => waSessions.filter((session) => session.status === "connected" && session.id !== pendingWaSessionId),
    [pendingWaSessionId, waSessions]
  );
  const humanHandoffNotificationRule = useMemo(
    () => groupNotificationRules.find((rule) => rule?.triggerKey === "human_handoff") || null,
    [groupNotificationRules]
  );
  const [manualMessage, setManualMessage] = useState("");
  const [manualMedia, setManualMedia] = useState(null);
  const [takeoverNote, setTakeoverNote] = useState("");

  const [knowledgeTab, setKnowledgeTab] = useState("faqs");
  const [faqForm, setFaqForm] = useState(null);
  const [documentForm, setDocumentForm] = useState(null);
  const [documentSourceMode, setDocumentSourceMode] = useState("upload");
  const [positionForm, setPositionForm] = useState(null);
  const [uploadTitle, setUploadTitle] = useState("");
  const [uploadFile, setUploadFile] = useState(null);

  const [businessTools, setBusinessTools] = useState([]);
  const [businessToolsLoadState, setBusinessToolsLoadState] = useState("loading");
  const [commerceProducts, setCommerceProducts] = useState([]);
  const [commerceOrderDrafts, setCommerceOrderDrafts] = useState([]);
  const [commerceOrderPipelines, setCommerceOrderPipelines] = useState([]);
  const [commerceOrderStages, setCommerceOrderStages] = useState([]);
  const [commerceOrderSettings, setCommerceOrderSettings] = useState(() => defaultCommerceOrderSettings());
  const [commerceProductForm, setCommerceProductForm] = useState(() => emptyCommerceProductForm());
  const [commerceOrderDraftForm, setCommerceOrderDraftForm] = useState(() => emptyCommerceOrderDraftForm());
  const [bookingServices, setBookingServices] = useState([]);
  const [bookingAppointments, setBookingAppointments] = useState([]);
  const [bookingServiceForm, setBookingServiceForm] = useState(() => emptyBookingServiceForm());
  const [bookingAppointmentForm, setBookingAppointmentForm] = useState(() => emptyBookingAppointmentForm());

  const [settingsDraft, setSettingsDraft] = useState(defaultSettingsDraft);
  const [playgroundForm, setPlaygroundForm] = useState({ customerName: "Pelanggan Test", messageText: "", aiAgentId: "" });
  const [playgroundMessages, setPlaygroundMessages] = useState([createPlaygroundWelcome()]);
  const [playgroundShortcuts, setPlaygroundShortcuts] = useState([]);
  const [playgroundShortcutForm, setPlaygroundShortcutForm] = useState({ id: "", label: "", question: "", expectedBehavior: "", isActive: true, sortOrder: 100 });
  const [playgroundBatchResults, setPlaygroundBatchResults] = useState([]);
  const [simulatedInboundText, setSimulatedInboundText] = useState("Halo, saya mau tanya status pesanan saya.");
  const [simulateAutoReply, setSimulateAutoReply] = useState(true);
  const [simulateForceDecision, setSimulateForceDecision] = useState("");
  const [newSimulationCustomerName, setNewSimulationCustomerName] = useState("Pelanggan Test");
  const [newSimulationPhone, setNewSimulationPhone] = useState("628982628875");
  const [newSimulationText, setNewSimulationText] = useState("Halo kak, saya mau tanya produk dan cara order.");
  const [simulationResults, setSimulationResults] = useState([]);

  const [purchaseForm, setPurchaseForm] = useState({ packageId: "", billingPeriod: "monthly", paymentMethod: "manual_transfer", notes: "Requested from dashboard" });
  const [packageForm, setPackageForm] = useState(() => emptyPackageForm());
  const [accountProfileForm, setAccountProfileForm] = useState({ name: "" });
  const [accountPasswordForm, setAccountPasswordForm] = useState({ currentPassword: "", newPassword: "", confirmPassword: "" });
  const [usageLogs, setUsageLogs] = useState([]);
  const [billingAnalytics, setBillingAnalytics] = useState(null);
  const [usageLogDetail, setUsageLogDetail] = useState(null);
  const [pricing, setPricing] = useState(null);
  const [aiModels, setAiModels] = useState([]);
  const [pricingForm, setPricingForm] = useState({
    creditUnitIdr: "500",
    usdToIdrRate: "16000",
    chatModelName: "openai/gpt-4o-mini",
    chatInputPricePer1m: "0.15",
    chatOutputPricePer1m: "0.60",
    embeddingModelName: "text-embedding-3-small",
    embeddingPricePer1m: "0.02",
    annualDiscountPercent: "10",
    monthlyCreditLimit: "100000",
    modelAliases: defaultModelAliases,
    availableModelIds: defaultAvailableModelIds,
  });
  const pricingFormDirtyRef = useRef(false);
  const [creditAdjustments, setCreditAdjustments] = useState([]);
  const [adjustmentForm, setAdjustmentForm] = useState({ adjustmentType: "add_additional", amount: "1000", notes: "" });
  const [agents, setAgents] = useState([]);
  const [agentForm, setAgentForm] = useState(null);
  const [aiAgentForm, setAIAgentForm] = useState(defaultAIAgentForm);
  const [agentTab, setAgentTab] = useState("overview");
  const [inviteForm, setInviteForm] = useState({ email: "", phone: "", role: "operator" });
  const [waSessionForm, setWaSessionForm] = useState({ label: "", aiAgentId: "", provider: "meta_cloud" });
  const [firstRunOnboardingOpen, setFirstRunOnboardingOpen] = useState(false);
  const [firstRunOnboardingIndex, setFirstRunOnboardingIndex] = useState(0);
  const [dashboardTheme, setDashboardTheme] = useState("light");

  const role = auth?.user?.role ?? "";
  const firstRunOnboardingStorageKey = useMemo(
    () => firstRunOnboardingStorageKeyForAuth(auth),
    [auth?.user?.id, auth?.user?.username, auth?.user?.organizationId, auth?.user?.organization?.id]
  );
  const isOwner = role === "owner";
  const isSuperAdmin = role === "super_admin";
  const isAdmin = role === "admin";
  const isOperator = role === "operator";
  const canManageSettings = isAdmin || isSuperAdmin;
  const canManageKnowledge = isAdmin || isSuperAdmin;
  const canRequestPurchases = isAdmin || isSuperAdmin;
  const canManageGroupNotifications = isOwner || isAdmin || isSuperAdmin;
  const canManagePackages = isOwner;
  const planMaxWhatsAppSessions = Number(billingPlan?.maxWhatsAppSessions ?? billingPlan?.max_whatsapp_sessions ?? 0);
  const planAllowsWhatsApp = planMaxWhatsAppSessions > 0;
  const hasConnectedWhatsAppSession = visibleWaSessions.some(setupSessionConnected);
  const canOpenChatOps = planAllowsWhatsApp && hasConnectedWhatsAppSession;
  const displayWaStatus = useMemo(() => (
    canOpenChatOps
      ? { status: "connected", details: visibleWaSessions.find(setupSessionConnected)?.details ?? {} }
      : { status: "disconnected", details: {} }
  ), [canOpenChatOps, visibleWaSessions]);
  const enabledBusinessToolKeys = useMemo(
    () => new Set(businessTools.filter((item) => item.installed ?? item.enabled).map((item) => item.key)),
    [businessTools]
  );
  const businessToolByKey = useMemo(
    () => new Map(businessTools.map((item) => [item.key, item])),
    [businessTools]
  );
  const businessToolsReady = businessTools.length > 0;
  const canUseProspects = enabledBusinessToolKeys.has("prospects");

  const isMobileOpsMode = useMobileOperationsMode();
  const baseNavItems = roleNav[role] ?? [];
  const availableBaseNavItems = useMemo(() => baseNavItems.filter((item) => {
    if (!item.moduleKey) return true;
    if (!businessToolsReady) return false;
    return enabledBusinessToolKeys.has(item.moduleKey);
  }), [baseNavItems, businessToolsReady, enabledBusinessToolKeys]);
  const navItems = useMemo(
    () => (isMobileOpsMode ? mobileNavForRole(role, availableBaseNavItems) : availableBaseNavItems),
    [availableBaseNavItems, isMobileOpsMode, role]
  );
  const activeMeta = viewMeta[activeView] ?? viewMeta.operations;

  function isViewAvailableForCurrentState(viewId) {
    const canonical = canonicalViewId(viewId);
    if (!isViewAllowedForRole(role, canonical)) return false;
    const moduleKey = pluginViewModules[canonical];
    if (!moduleKey) return true;
    if (!businessToolsReady) return businessToolsLoadState === "loading";
    return enabledBusinessToolKeys.has(moduleKey);
  }

  function showToast(nextToast) {
    if (toastTimerRef.current) window.clearTimeout(toastTimerRef.current);
    setToast({
      tone: nextToast.tone || "success",
      title: nextToast.title || "Berhasil",
      message: nextToast.message || "",
    });
    toastTimerRef.current = window.setTimeout(() => setToast(null), nextToast.duration || 3200);
  }

  function resolveDialog(value) {
    const resolve = dialogResolverRef.current;
    dialogResolverRef.current = null;
    setDialog(null);
    setDialogInput("");
    resolve?.(value);
  }

  function askConfirm({ title, message, confirmLabel = "Ya, lanjut", cancelLabel = "Batal", tone = "default" }) {
    return new Promise((resolve) => {
      dialogResolverRef.current = resolve;
      setDialog({ type: "confirm", title, message, confirmLabel, cancelLabel, tone });
      setDialogInput("");
    });
  }

  function askPrompt({ title, message, label, initialValue = "", placeholder = "", inputType = "text", multiline = false, confirmLabel = "Simpan" }) {
    return new Promise((resolve) => {
      dialogResolverRef.current = resolve;
      setDialog({ type: "prompt", title, message, label, placeholder, inputType, multiline, confirmLabel, cancelLabel: "Batal" });
      setDialogInput(initialValue || "");
    });
  }

  function showNotice({ title, message, confirmLabel = "Mengerti", tone = "success" }) {
    dialogResolverRef.current = null;
    setDialog({ type: "notice", title, message, confirmLabel, tone });
    setDialogInput("");
  }

  function resetWhatsAppViewState() {
    setWaStatus({ status: "disconnected", details: {} });
    setWaSessions([]);
    setActiveWaSessionId("");
    setPendingWaSessionId("");
    setMetaTemplates([]);
    setMetaCloudConfig({ enabled: false });
    setMetaTemplateSendForm({ phone: "", name: "", language: "id" });
    setQrData(null);
    setQrImageSrc("");
    setEscalationGroup(null);
    setGroupNotificationRules([]);
    setGroupNotificationVariables([]);
  }

  const pendingPurchases = purchases.filter((item) => ["requested", "pending"].includes(item.paymentStatus));
  const ownerWallet = summary?.wallet ?? wallet;
  const notificationStorageKey = auth?.user?.id ? `oneflow-notifications-read:${auth.user.id}` : "";
  const legacyNotificationStorageKey = auth?.user?.id ? `hrchat-notifications-read:${auth.user.id}` : "";
  const unreadNotificationCount = notifications.filter((item) => !notificationReadIds.includes(item.id)).length;

  const displayInbox = useMemo(() => {
    const query = deferredInboxSearch.trim().toLowerCase();
    let items = [...inbox];
    if (inboxTab === "ai") items = items.filter((item) => item.mode === "ai");
    if (inboxTab === "human") items = items.filter((item) => item.mode === "human");
    if (inboxTab === "pending") items = items.filter((item) => item.status === "pending_human");
    if (inboxTab === "resolved") items = items.filter((item) => item.status === "resolved");
    if (activeInboxWaSessionId) items = items.filter((item) => item.whatsappSessionId === activeInboxWaSessionId);
    if (activeInboxAIAgentId) items = items.filter((item) => item.aiAgentId === activeInboxAIAgentId);
    if (!query) return items;
    return items.filter((item) => {
      const haystack = [
        item.contactName,
        item.phone,
        item.lastMessageText,
        item.assignedToName,
        item.escalationReason,
        item.whatsappSession,
        item.aiAgentName,
      ].join(" ").toLowerCase();
      return haystack.includes(query);
    });
  }, [activeInboxAIAgentId, activeInboxWaSessionId, deferredInboxSearch, inbox, inboxTab]);

  useEffect(() => {
    try {
      const storedTheme = window.localStorage.getItem(dashboardThemeStorageKey);
      if (storedTheme === "light" || storedTheme === "dark") {
        setDashboardTheme(storedTheme);
        return;
      }
      if (window.matchMedia?.("(prefers-color-scheme: dark)").matches) {
        setDashboardTheme("dark");
      }
    } catch {}
  }, []);

  useEffect(() => {
    try {
      window.localStorage.setItem(dashboardThemeStorageKey, dashboardTheme);
      if (auth?.token) {
        document.documentElement.dataset.dashboardTheme = dashboardTheme;
      } else {
        delete document.documentElement.dataset.dashboardTheme;
      }
    } catch {}
    return () => {
      try {
        if (document.documentElement.dataset.dashboardTheme === dashboardTheme) {
          delete document.documentElement.dataset.dashboardTheme;
        }
      } catch {}
    };
  }, [auth?.token, dashboardTheme]);

  const toggleDashboardTheme = useCallback(() => {
    setDashboardTheme((current) => (current === "dark" ? "light" : "dark"));
  }, []);

  useEffect(() => {
    activeKnowledgeAIAgentIdRef.current = activeKnowledgeAIAgentId;
  }, [activeKnowledgeAIAgentId]);

  useEffect(() => {
    waSessionIdsRef.current = new Set(waSessions.map((session) => String(session.id || "")));
  }, [waSessions]);

  useEffect(() => {
    try {
      const stored = readAndMigrateStorageValue(window.localStorage, authStorageKey, legacyAuthStorageKey);
      if (stored) {
        const parsed = normalizeStoredAuthRole(JSON.parse(stored));
        window.localStorage.setItem(authStorageKey, JSON.stringify(parsed));
        const requestedPath = requestedDashboardPath();
        const routeConfig = setupRouteConfigFromPath(requestedPath);
        const nextView = viewFromPath(requestedPath, parsed.user?.role) || storedViewForRole(parsed.user?.role);
        setAuth(parsed);
        setActiveView(nextView);
        if (routeConfig?.tab) setAgentTab(routeConfig.tab);
        if (!isDashboardAppPath(window.location.pathname)) {
          router.replace(pathForView(nextView));
        }
      } else {
        let sessionExpired = new URLSearchParams(window.location.search).get("reason") === "session-expired";
        try {
          sessionExpired = sessionExpired || window.sessionStorage.getItem(sessionExpiredStorageKey) === "1";
        } catch {}
        if (window.location.pathname === "/login" && sessionExpired) {
          setError("Sesi login sudah berakhir. Silakan login ulang.");
        }
        if (isDashboardAppPath(window.location.pathname)) {
          rememberLoginNextPath(window.location.pathname);
          const loginQuery = new URLSearchParams({ next: window.location.pathname });
          if (sessionExpired) loginQuery.set("reason", "session-expired");
          router.replace(`/login?${loginQuery.toString()}`);
        }
      }
    } catch {
      window.localStorage.removeItem(authStorageKey);
      window.localStorage.removeItem(legacyAuthStorageKey);
      if (isDashboardAppPath(window.location.pathname)) {
        rememberLoginNextPath(window.location.pathname);
        router.replace(`/login?next=${encodeURIComponent(window.location.pathname)}`);
      }
    } finally {
      setAuthReady(true);
    }
  }, [router]);

  useEffect(() => {
    if (!auth?.token) return;
    try {
      if (window.localStorage.getItem(firstRunOnboardingStorageKey) === "pending") {
        setFirstRunOnboardingIndex(0);
        setFirstRunOnboardingOpen(true);
      }
    } catch {
      setFirstRunOnboardingOpen(false);
    }
  }, [auth?.token, firstRunOnboardingStorageKey]);

  useEffect(() => {
    if (!role || isViewAvailableForCurrentState(activeView)) return;
    const nextView = navItems[0]?.id ?? defaultViewForRole(role);
    setActiveView(nextView);
    router.replace(pathForView(nextView));
  }, [activeView, businessToolsLoadState, businessToolsReady, enabledBusinessToolKeys, navItems, role, router]);

  useEffect(() => {
    if (!role || !isViewAvailableForCurrentState(activeView)) return;
    window.localStorage.setItem(activeViewStorageKey, activeView);
  }, [activeView, businessToolsReady, enabledBusinessToolKeys, role]);

  useEffect(() => {
    setAccountProfileForm({ name: auth?.user?.name ?? "" });
    if (!notificationStorageKey) {
      setNotificationReadIds([]);
      return;
    }
    try {
      const stored = readAndMigrateStorageValue(window.localStorage, notificationStorageKey, legacyNotificationStorageKey);
      setNotificationReadIds(JSON.parse(stored || "[]"));
    } catch {
      setNotificationReadIds([]);
    }
  }, [auth?.user?.name, legacyNotificationStorageKey, notificationStorageKey]);

  useEffect(() => {
    if (!role || !isDashboardAppPath(pathname)) return;
    const routeConfig = setupRouteConfigFromPath(pathname);
    const setupRouteKey = routeConfig ? `${pathname}:${routeConfig.tab || ""}:${routeConfig.anchor || ""}` : "";
    const shouldApplySetupRoute = Boolean(routeConfig) && appliedSetupRouteRef.current !== setupRouteKey;
    if (!routeConfig) appliedSetupRouteRef.current = "";
    if (shouldApplySetupRoute) {
      appliedSetupRouteRef.current = setupRouteKey;
    }
    if (shouldApplySetupRoute && routeConfig?.tab) setAgentTab(routeConfig.tab);
    if (shouldApplySetupRoute && routeConfig?.anchor) {
      window.setTimeout(() => {
        document.querySelector(`[data-tour="${routeConfig.anchor}"]`)?.scrollIntoView({ block: "center", inline: "center", behavior: "smooth" });
      }, 260);
    }
    if (!pathname?.startsWith("/dashboard")) {
      const setupView = viewFromPath(pathname, role);
      if (setupView && setupView !== activeView) setActiveView(setupView);
      return;
    }
    const segment = pathname.split("/").filter(Boolean)[1] || "";
    const resolvedView = segmentViewIds[segment] ?? segment;
    const nextView = viewFromPath(pathname, role);
    if (!segment) {
      if (nextView && nextView !== activeView) setActiveView(nextView);
      return;
    }
    if (!isViewAvailableForCurrentState(resolvedView)) {
      const fallbackView = isViewAvailableForCurrentState(nextView) ? nextView : (navItems[0]?.id ?? defaultViewForRole(role));
      const nextPath = pathForView(fallbackView);
      if (pathname !== nextPath) router.replace(nextPath);
      return;
    }
    if (nextView && nextView !== activeView) setActiveView(nextView);
  }, [activeView, businessToolsLoadState, businessToolsReady, enabledBusinessToolKeys, navItems, pathname, role, router]);

  useEffect(() => {
    setSettingsDraft(settingsApiToDraft(settings));
  }, [settings]);

  useEffect(() => {
    if (!waSessions.length || !activeWaSessionId || activeWaSessionId === pendingWaSessionId) return;
    const selected = waSessions.find((item) => item.id === activeWaSessionId);
    if (!selected) return;
    setWaStatus({ status: selected.status ?? "disconnected", details: selected.details ?? {} });
    if (selected.status === "connected" && selected.id === pendingWaSessionId) {
      setPendingWaSessionId("");
    }
  }, [activeWaSessionId, pendingWaSessionId, waSessions]);

  useEffect(() => {
    if (!pricing) return;
    if (pricingFormDirtyRef.current) return;
    setPricingForm({
      creditUnitIdr: String(pricing.creditUnitIdr ?? "500"),
      usdToIdrRate: String(pricing.usdToIdrRate ?? "16000"),
      chatModelName: pricing.chatModelName ?? "openai/gpt-4o-mini",
      chatInputPricePer1m: String(pricing.chatInputPricePer1m ?? "0.15"),
      chatOutputPricePer1m: String(pricing.chatOutputPricePer1m ?? "0.60"),
      embeddingModelName: pricing.embeddingModelName ?? "text-embedding-3-small",
      embeddingPricePer1m: String(pricing.embeddingPricePer1m ?? "0.02"),
      annualDiscountPercent: String(pricing.annualDiscountPercent ?? "10"),
      monthlyCreditLimit: String(pricing.monthlyCreditLimit ?? "100000"),
      modelAliases: { ...defaultModelAliases, ...(pricing.modelAliases ?? {}) },
      availableModelIds: pricing.availableModelIds?.length ? pricing.availableModelIds : defaultAvailableModelIds,
    });
  }, [pricing]);

  const updatePricingForm = useCallback((nextPricingForm) => {
    pricingFormDirtyRef.current = true;
    setPricingForm((current) => (typeof nextPricingForm === "function" ? nextPricingForm(current) : nextPricingForm));
  }, []);

  useEffect(() => {
    const connectedSessionId = String(waStatus?.details?.sessionId || "");
    if (waStatus?.status === "connected" && (!pendingWaSessionId || connectedSessionId === pendingWaSessionId)) {
      setQrData(null);
      setQrImageSrc("");
      return;
    }
    if (waStatus?.details?.qr) {
      setQrData({
        qr: waStatus.details.qr,
        expiresIn: waStatus.details.expiresIn ?? 60,
        mode: waStatus.details.mode ?? "real",
      });
    }
  }, [pendingWaSessionId, waStatus]);

  useEffect(() => {
    if (!qrData?.qr) {
      setQrImageSrc("");
      return;
    }
    let cancelled = false;
    QRCode.toDataURL(qrData.qr, { width: 320, margin: 2 })
      .then((src) => {
        if (!cancelled) setQrImageSrc(src);
      })
      .catch(() => {
        if (!cancelled) setQrImageSrc("");
      });
    return () => {
      cancelled = true;
    };
  }, [qrData]);

  useEffect(() => {
    if (!auth?.token) return;
    let active = true;
    async function sync() {
      if (!active) return;
      await loadDashboardData(auth, { preserveDrafts: isEditingFormControl() });
    }
    sync().catch(() => {});
    const timer = window.setInterval(() => {
      sync().catch(() => {});
    }, 8000);
    return () => {
      active = false;
      window.clearInterval(timer);
    };
  }, [auth, analyticsRange]);

  useEffect(() => {
    if (!auth?.token || !["agents", "knowledge"].includes(activeView) || !activeKnowledgeAIAgentId) return;
    loadDashboardData(auth, { preserveDrafts: true, refreshKnowledge: true }).catch(() => {});
  }, [activeKnowledgeAIAgentId]);

  useEffect(() => {
    if (!auth?.token || !selectedConversation?.conversation?.id) return;
    const timer = window.setInterval(() => {
      refreshConversation(selectedConversation.conversation.id).catch(() => {});
    }, 6000);
    return () => window.clearInterval(timer);
  }, [auth?.token, selectedConversation?.conversation?.id]);

  useEffect(() => {
    if (!auth?.token) return;
    const socket = new WebSocket(wsBase);
    socket.onmessage = (event) => {
      try {
        const payload = JSON.parse(event.data);
        if (payload?.wa) {
          const payloadSessionId = String(payload.wa?.details?.sessionId || "");
          if (payloadSessionId && (payloadSessionId === activeWaSessionId || waSessionIdsRef.current.has(payloadSessionId))) {
            setWaStatus(payload.wa);
          }
        }
        if (payload?.wallet) setWallet((current) => current ?? payload.wallet);
        if (payload?.services || payload?.systemAlerts) setHealth((current) => ({ ...(current || {}), services: payload.services ?? current?.services, systemAlerts: payload.systemAlerts ?? current?.systemAlerts }));
        setRealtimeSnapshot(payload);
      } catch {}
    };
    return () => socket.close();
  }, [auth?.token, activeWaSessionId]);

  function expireAuthenticatedSession(expiredToken) {
    if (!expiredToken || handledExpiredTokenRef.current === expiredToken) return;
    try {
      const stored = readAndMigrateStorageValue(window.localStorage, authStorageKey, legacyAuthStorageKey);
      const storedToken = stored ? JSON.parse(stored)?.token : "";
      if (storedToken && storedToken !== expiredToken) return;
    } catch {
      // Invalid persisted auth state should be cleared through the same login flow.
    }
    handledExpiredTokenRef.current = expiredToken;
    logout({ sessionExpired: true });
  }

  async function requestJSON(url, options = {}, config = {}) {
    const { needsAuth = true, isForm = false, token = auth?.token, timeoutMs = 0 } = config;
    const headers = { ...(options.headers ?? {}) };
    if (needsAuth && token) headers.Authorization = `Bearer ${token}`;
    if (!isForm && options.body && !headers["Content-Type"]) headers["Content-Type"] = "application/json";

    const requestOptions = { ...options, headers };
    let timeoutId = null;
    if (timeoutMs > 0 && !requestOptions.signal) {
      const controller = new AbortController();
      requestOptions.signal = controller.signal;
      timeoutId = window.setTimeout(() => controller.abort(), timeoutMs);
    }
    let response;
    try {
      response = await fetch(url, requestOptions);
    } catch (requestError) {
      if (timeoutId !== null && requestError?.name === "AbortError") {
        const timeoutError = new Error("Permintaan terlalu lama. Coba lagi.");
        timeoutError.code = "request_timeout";
        throw timeoutError;
      }
      throw requestError;
    } finally {
      if (timeoutId !== null) window.clearTimeout(timeoutId);
    }
    let payload = {};
    try {
      payload = await response.json();
    } catch {}
    if (!response.ok) {
      const fetchError = new Error(userFriendlyError(payload?.error, response.status));
      fetchError.status = response.status;
      fetchError.payload = payload;
      fetchError.code = payload?.code;
      const failure = String(payload?.error || "").trim().toLowerCase();
      if (needsAuth && token && response.status === 401 && ["missing bearer token", "invalid token", "invalid organization access"].includes(failure)) {
        fetchError.sessionExpired = true;
        expireAuthenticatedSession(token);
      }
      throw fetchError;
    }
    return payload;
  }

  async function loadAIModelsCatalog(currentAuth = auth) {
    let backendError = null;
    try {
      const payload = await requestJSON(`${apiBase}/api/ai-models`, {}, { token: currentAuth?.token, timeoutMs: 5000 });
      const backendItems = payload.items ?? [];
      if (backendItems.length > 0) {
        return payload;
      }
      backendError = new Error(payload.warning || "backend returned empty model catalog");
    } catch (error) {
      backendError = error;
    }

    if (backendError) throw backendError;
    return { items: [], source: "empty" };
  }

  async function installBusinessTool(moduleKey) {
    try {
      setBusyKey(`business-tool-${moduleKey}`);
      setError("");
      const payload = await requestJSON(`${apiBase}/api/business-tools/${encodeURIComponent(moduleKey)}/install`, { method: "POST" });
      setBusinessTools(payload.items ?? []);
      await loadDashboardData(auth, { preserveDrafts: true });
      showToast({
        title: "Alat terinstall",
        message: "Menu alat sudah muncul di sidebar pada bagian Alat Terinstall.",
      });
    } catch (toolError) {
      setError(toolError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function removeBusinessTool(moduleKey) {
    const confirmed = await askConfirm({
      title: "Remove alat?",
      message: "Menu alat akan disembunyikan dari sidebar. Data yang sudah dibuat tetap tersimpan.",
      confirmLabel: "Remove",
      tone: "danger",
    });
    if (!confirmed) return;
    try {
      setBusyKey(`business-tool-${moduleKey}`);
      setError("");
      const payload = await requestJSON(`${apiBase}/api/business-tools/${encodeURIComponent(moduleKey)}`, {
        method: "PATCH",
        body: JSON.stringify({ enabled: false }),
      });
      setBusinessTools(payload.items ?? []);
      await loadDashboardData(auth, { preserveDrafts: true });
      showToast({
        title: "Alat diremove",
        message: "Menu alat sudah disembunyikan dari Alat Terinstall.",
      });
    } catch (toolError) {
      setError(toolError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function updateBusinessToolAI(moduleKey, aiMode) {
    try {
      setBusyKey(`business-tool-ai-${moduleKey}`);
      setError("");
      const payload = await requestJSON(`${apiBase}/api/business-tools/${encodeURIComponent(moduleKey)}`, {
        method: "PATCH",
        body: JSON.stringify({ aiMode }),
      });
      setBusinessTools(payload.items ?? []);
      showToast({
        title: "Akses AI diperbarui",
        message: aiMode === "off" ? "AI tidak memakai data alat ini." : "AI akan mengikuti mode yang dipilih untuk alat ini.",
      });
    } catch (toolError) {
      setError(toolError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function configureBusinessToolsForAgent(toolConfigs = [], setupDetails = {}) {
    const configs = (toolConfigs || []).filter((item) => item?.key);
    if (!configs.length) return true;
    try {
      setBusyKey("agent-business-tools-setup");
      setError("");
      const configuredKeys = new Set(configs.map((item) => item.key));
      let latestItems = businessTools;
      for (const item of configs) {
        const payload = await requestJSON(`${apiBase}/api/business-tools/${encodeURIComponent(item.key)}`, {
          method: "PATCH",
          body: JSON.stringify({ enabled: true, aiMode: item.aiMode || "draft" }),
        });
        latestItems = payload.items ?? latestItems;
      }
      setBusinessTools(latestItems ?? []);
      if (configuredKeys.has("commerce")) {
        const fulfillmentTypes = Array.isArray(setupDetails.commerceFulfillmentTypes) && setupDetails.commerceFulfillmentTypes.length
          ? setupDetails.commerceFulfillmentTypes
          : defaultCommerceOrderSettings().enabledFulfillmentTypes;
        await requestJSON(`${apiBase}/api/commerce/order-settings`, {
          method: "PATCH",
          body: JSON.stringify({ enabledFulfillmentTypes: fulfillmentTypes }),
        });
        const setupProducts = Array.isArray(setupDetails.commerceProducts) && setupDetails.commerceProducts.length
          ? setupDetails.commerceProducts
          : [{
              sku: "",
              name: setupDetails.commerceProductName || "",
              description: "Produk awal dari setup agent.",
              unitPrice: setupDetails.commerceProductPrice || 0,
              stockQuantity: setupDetails.commerceProductStock || 0,
              lowStockThreshold: 0,
            }];
        for (const product of setupProducts) {
          const productName = String(product.name || "").trim();
          if (!productName) continue;
          const fallbackSKU = productName
            .toUpperCase()
            .replace(/[^A-Z0-9]+/g, "-")
            .replace(/^-+|-+$/g, "")
            .slice(0, 24);
          await requestJSON(`${apiBase}/api/commerce/products`, {
            method: "POST",
            body: JSON.stringify({
              sku: String(product.sku || fallbackSKU).trim(),
              name: productName,
              description: product.description || "Produk awal dari setup agent.",
              unitPrice: Number(product.unitPrice || 0),
              stockQuantity: Number(product.stockQuantity || 0),
              lowStockThreshold: Number(product.lowStockThreshold || 0),
              status: "active",
            }),
          });
        }
      }
      if (configuredKeys.has("booking")) {
        const serviceName = String(setupDetails.bookingServiceName || "").trim();
        if (serviceName) {
          await requestJSON(`${apiBase}/api/booking/services`, {
            method: "POST",
            body: JSON.stringify({
              name: serviceName,
              description: "Layanan awal dari setup agent.",
              durationMinutes: Number(setupDetails.bookingDurationMinutes || 60),
              bufferMinutes: 0,
              price: Number(setupDetails.bookingPrice || 0),
              timezone: "Asia/Jakarta",
              availability: {
                days: availabilityDaysFromInput(setupDetails.bookingAvailabilityDays),
                start: setupDetails.bookingAvailabilityStart || "09:00",
                end: setupDetails.bookingAvailabilityEnd || "17:00",
              },
              status: "active",
            }),
          });
        }
      }
      await loadDashboardData(auth, { preserveDrafts: true });
      showToast({
        title: "Setup agent lengkap",
        message: "Alat bisnis, izin AI, dan data awal yang diisi sudah disimpan.",
      });
      return true;
    } catch (toolError) {
      setError(toolError.message);
      showToast({
        title: "Agent dibuat, alat belum aktif",
        message: "Buka Alat Bisnis untuk mengaktifkan tool secara manual.",
        tone: "warning",
      });
      return false;
    } finally {
      setBusyKey("");
    }
  }

  function editCommerceProduct(product) {
    setCommerceProductForm({
      id: product.id || "",
      sku: product.sku || "",
      name: product.name || "",
      description: product.description || "",
      unitPrice: String(product.unitPrice ?? 0),
      stockQuantity: String(product.stockQuantity ?? 0),
      lowStockThreshold: String(product.lowStockThreshold ?? 0),
      status: product.status || "active",
    });
  }

  async function submitCommerceProduct(event) {
    event.preventDefault();
    try {
      setBusyKey("commerce-product");
      setError("");
      const payload = {
        sku: commerceProductForm.sku,
        name: commerceProductForm.name,
        description: commerceProductForm.description,
        unitPrice: Number(commerceProductForm.unitPrice || 0),
        stockQuantity: Number(commerceProductForm.stockQuantity || 0),
        lowStockThreshold: Number(commerceProductForm.lowStockThreshold || 0),
        status: commerceProductForm.status || "active",
      };
      const id = commerceProductForm.id;
      await requestJSON(`${apiBase}/api/commerce/products${id ? `/${encodeURIComponent(id)}` : ""}`, {
        method: id ? "PATCH" : "POST",
        body: JSON.stringify(payload),
      });
      setCommerceProductForm(emptyCommerceProductForm());
      await loadDashboardData(auth, { preserveDrafts: true });
      showToast({ title: "Produk tersimpan", message: "Data produk dan stok sudah diperbarui." });
      return true;
    } catch (productError) {
      setError(productError.message);
      return false;
    } finally {
      setBusyKey("");
    }
  }

  async function deleteCommerceProduct(product) {
    if (!product?.id) return false;
    if (!(await askConfirm({
      title: "Hapus produk?",
      message: `${product.name || "Produk ini"} akan dihapus dari katalog dan tidak muncul lagi sebagai pilihan pesanan baru.`,
      confirmLabel: "Hapus produk",
      tone: "danger",
    }))) return false;
    try {
      setBusyKey(`commerce-product-delete-${product.id}`);
      setError("");
      await requestJSON(`${apiBase}/api/commerce/products/${encodeURIComponent(product.id)}`, { method: "DELETE" });
      setCommerceProductForm((current) => (current.id === product.id ? emptyCommerceProductForm() : current));
      await loadDashboardData(auth, { preserveDrafts: true });
      showToast({ title: "Produk dihapus", message: "Produk sudah dikeluarkan dari katalog aktif." });
      return true;
    } catch (productError) {
      setError(productError.message);
      return false;
    } finally {
      setBusyKey("");
    }
  }

  async function submitCommerceOrderDraft(event) {
    event.preventDefault();
    try {
      setBusyKey("commerce-order-draft");
      setError("");
      const items = (commerceOrderDraftForm.items || [])
        .filter((item) => item.productId)
        .map((item) => ({ productId: item.productId, quantity: Number(item.quantity || 1) }));
      await requestJSON(`${apiBase}/api/commerce/order-drafts`, {
        method: "POST",
        body: JSON.stringify({
          customerName: commerceOrderDraftForm.customerName,
          customerPhone: commerceOrderDraftForm.customerPhone,
          stageId: commerceOrderDraftForm.stageId,
          fulfillmentType: commerceOrderDraftForm.fulfillmentType,
          recipientName: commerceOrderDraftForm.recipientName,
          recipientPhone: commerceOrderDraftForm.recipientPhone,
          addressLine: commerceOrderDraftForm.addressLine,
          addressArea: commerceOrderDraftForm.addressArea,
          addressNotes: commerceOrderDraftForm.addressNotes,
          notes: commerceOrderDraftForm.notes,
          items,
        }),
      });
      setCommerceOrderDraftForm(emptyCommerceOrderDraftForm());
      await loadDashboardData(auth, { preserveDrafts: true });
      showToast({ title: "Draft dibuat", message: "Pesanan masih draft dan stok belum dikurangi." });
      return true;
    } catch (draftError) {
      setError(draftError.message);
      return false;
    } finally {
      setBusyKey("");
    }
  }

  async function updateCommerceOrderDraftStatus(id, status) {
    try {
      setBusyKey(`commerce-order-${id}-${status}`);
      setError("");
      await requestJSON(`${apiBase}/api/commerce/order-drafts/${encodeURIComponent(id)}`, {
        method: "PATCH",
        body: JSON.stringify({ status }),
      });
      await loadDashboardData(auth, { preserveDrafts: true });
      showToast({
        title: status === "confirmed" ? "Pesanan dikonfirmasi" : "Pesanan diperbarui",
        message: status === "confirmed" ? "Stok masuk reserved, belum final berkurang." : "Status draft pesanan sudah diperbarui.",
      });
    } catch (orderError) {
      setError(orderError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function updateCommerceOrderDraft(id, payload, toast = {}) {
    try {
      setBusyKey(`commerce-order-${id}-update`);
      setError("");
      await requestJSON(`${apiBase}/api/commerce/order-drafts/${encodeURIComponent(id)}`, {
        method: "PATCH",
        body: JSON.stringify(payload),
      });
      await loadDashboardData(auth, { preserveDrafts: true });
      if (toast.title) showToast(toast);
    } catch (orderError) {
      setError(orderError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function saveCommerceOrderStage(id, payload) {
    try {
      setBusyKey(`commerce-order-stage-${id || "new"}`);
      setError("");
      await requestJSON(`${apiBase}/api/commerce/order-stages${id ? `/${encodeURIComponent(id)}` : ""}`, {
        method: id ? "PATCH" : "POST",
        body: JSON.stringify(payload),
      });
      await loadDashboardData(auth, { preserveDrafts: true });
      showToast({ title: "Pipeline disimpan", message: "Stage pesanan sudah diperbarui." });
    } catch (stageError) {
      setError(stageError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function deleteCommerceOrderStage(id) {
    if (!id) return;
    if (!window.confirm("Hapus stage ini? Pesanan di stage ini akan dipindah ke stage pertama yang tersisa.")) return;
    try {
      setBusyKey(`commerce-order-stage-delete-${id}`);
      setError("");
      await requestJSON(`${apiBase}/api/commerce/order-stages/${encodeURIComponent(id)}`, {
        method: "DELETE",
      });
      await loadDashboardData(auth, { preserveDrafts: true });
      showToast({ title: "Stage dihapus", message: "Pesanan dari stage itu dipindah ke stage pertama yang tersisa." });
    } catch (stageError) {
      setError(stageError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function updateCommerceOrderSettings(enabledFulfillmentTypes) {
    try {
      setBusyKey("commerce-order-settings");
      setError("");
      const payload = await requestJSON(`${apiBase}/api/commerce/order-settings`, {
        method: "PATCH",
        body: JSON.stringify({ enabledFulfillmentTypes }),
      });
      const nextSettings = payload.settings ?? defaultCommerceOrderSettings();
      setCommerceOrderSettings(nextSettings);
      const enabledTypes = nextSettings.enabledFulfillmentTypes ?? [];
      setCommerceOrderDraftForm((current) => (
        enabledTypes.includes(current.fulfillmentType)
          ? current
          : { ...current, fulfillmentType: enabledTypes[0] || "pickup" }
      ));
      showToast({ title: "Cara terima pesanan disimpan", message: "Manual order dan AI akan mengikuti opsi yang aktif." });
    } catch (settingsError) {
      setError(settingsError.message);
    } finally {
      setBusyKey("");
    }
  }

  function editBookingService(item) {
    setBookingServiceForm(bookingServiceFormFromItem(item));
  }

  async function submitBookingService(event) {
    event.preventDefault();
    try {
      setBusyKey("booking-service");
      setError("");
      const payload = {
        name: bookingServiceForm.name,
        description: bookingServiceForm.description,
        durationMinutes: Number(bookingServiceForm.durationMinutes || 30),
        bufferMinutes: Number(bookingServiceForm.bufferMinutes || 0),
        price: Number(bookingServiceForm.price || 0),
        timezone: bookingServiceForm.timezone || "Asia/Jakarta",
        status: bookingServiceForm.status || "active",
        availability: {
          days: availabilityDaysFromInput(bookingServiceForm.availabilityDays),
          start: bookingServiceForm.availabilityStart || "09:00",
          end: bookingServiceForm.availabilityEnd || "17:00",
        },
      };
      const id = bookingServiceForm.id;
      await requestJSON(`${apiBase}/api/booking/services${id ? `/${encodeURIComponent(id)}` : ""}`, {
        method: id ? "PATCH" : "POST",
        body: JSON.stringify(payload),
      });
      setBookingServiceForm(emptyBookingServiceForm());
      await loadDashboardData(auth, { preserveDrafts: true });
      showToast({ title: "Layanan tersimpan", message: "Pengaturan booking bisnis sudah diperbarui." });
      return true;
    } catch (bookingError) {
      setError(bookingError.message);
      return false;
    } finally {
      setBusyKey("");
    }
  }

  async function submitBookingAppointment(event) {
    event.preventDefault();
    try {
      setBusyKey("booking-appointment");
      setError("");
      await requestJSON(`${apiBase}/api/booking/appointments`, {
        method: "POST",
        body: JSON.stringify({
          serviceId: bookingAppointmentForm.serviceId,
          customerName: bookingAppointmentForm.customerName,
          customerPhone: bookingAppointmentForm.customerPhone,
          scheduledStart: bookingAppointmentForm.scheduledStart,
          locationType: bookingAppointmentForm.locationType,
          locationAddress: bookingAppointmentForm.locationAddress,
          locationNotes: bookingAppointmentForm.locationNotes,
          notes: bookingAppointmentForm.notes,
        }),
      });
      setBookingAppointmentForm(emptyBookingAppointmentForm());
      await loadDashboardData(auth, { preserveDrafts: true });
      showToast({ title: "Booking dibuat", message: "Appointment customer sudah masuk jadwal." });
    } catch (bookingError) {
      setError(bookingError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function updateBookingAppointmentStatus(id, status) {
    try {
      setBusyKey(`booking-appointment-${id}-${status}`);
      setError("");
      await requestJSON(`${apiBase}/api/booking/appointments/${encodeURIComponent(id)}`, {
        method: "PATCH",
        body: JSON.stringify({ status }),
      });
      await loadDashboardData(auth, { preserveDrafts: true });
      showToast({ title: "Booking diperbarui", message: "Status appointment sudah diperbarui." });
    } catch (bookingError) {
      setError(bookingError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function syncMidtransPayment(payment) {
    if (!payment?.id) return null;
    return requestJSON(`${apiBase}/api/billing/purchases/${payment.id}/sync`, { method: "POST" });
  }

  async function cancelBillingPurchase(payment) {
    if (!payment?.id) return null;
    return requestJSON(`${apiBase}/api/billing/purchases/${payment.id}/cancel`, { method: "POST" });
  }

  async function refreshAfterMidtransCallback(payment, options = {}) {
    const retries = Number(options.retries ?? 2);
    try {
      let payload = null;
      for (let attempt = 0; attempt <= retries; attempt += 1) {
        payload = await syncMidtransPayment(payment);
        if (payload?.paymentStatus === "confirmed" || attempt === retries) break;
        await new Promise((resolve) => window.setTimeout(resolve, 1500));
      }
      await loadDashboardData().catch(() => {});
      return payload?.paymentStatus || "";
    } catch (syncError) {
      showNotice({
        title: "Menunggu verifikasi",
        message: userFriendlyError(syncError.message, syncError.status),
        tone: "warn",
      });
      await loadDashboardData().catch(() => {});
      return "";
    }
  }

  async function openMidtransPayment(payment) {
    if (!payment?.snapToken) return;
    const clientKey = payment.midtransClientKey;
    if (!clientKey) {
      if (payment.snapRedirectUrl) window.location.assign(payment.snapRedirectUrl);
      return;
    }
    try {
      const snap = await loadMidtransSnapScript(clientKey, payment.midtransEnvironment);
      snap.pay(payment.snapToken, {
        onSuccess: async () => {
          const paymentStatus = await refreshAfterMidtransCallback(payment, { retries: 3 });
          showNotice({
            title: paymentStatus === "confirmed" ? "Pembayaran terverifikasi" : "Pembayaran diproses",
            message: paymentStatus === "confirmed"
              ? "Paket dan kredit sudah aktif."
              : "Pembayaran sudah dicatat dan menunggu konfirmasi akhir.",
            tone: paymentStatus === "confirmed" ? "success" : "warn",
          });
        },
        onPending: async () => {
          await refreshAfterMidtransCallback(payment);
          showNotice({
            title: "Pembayaran menunggu",
            message: "Transaksi sudah dibuat. Selesaikan pembayaran lewat instruksi yang tersedia, lalu status akan diperbarui otomatis.",
            tone: "warn",
          });
        },
        onError: () => {
          setError("Pembayaran belum bisa diproses. Coba buka ulang dari riwayat pembayaran.");
          loadDashboardData().catch(() => {});
        },
        onClose: () => {
          refreshAfterMidtransCallback(payment).catch(() => {});
        },
      });
    } catch (snapError) {
      if (payment.snapRedirectUrl) {
        window.location.assign(payment.snapRedirectUrl);
        return;
      }
      throw snapError;
    }
  }

  async function cancelPurchase(payment) {
    if (!payment?.id) return null;
    try {
      setBusyKey(`purchase-cancel-${payment.id}`);
      const payload = await cancelBillingPurchase(payment);
      await loadDashboardData(auth, { preserveDrafts: true });
      showNotice({
        title: "Pembayaran dibatalkan",
        message: "Transaksi ini sudah dipindahkan dari pembayaran berjalan.",
        tone: "success",
      });
      return payload;
    } catch (cancelError) {
      setError(cancelError.message);
      return null;
    } finally {
      setBusyKey("");
    }
  }

  useEffect(() => {
    if (!auth?.token || (activeView !== "wallet" && activeView !== "upgrade")) return;
    const eligiblePurchases = purchases
      .filter((item) => item?.id && item?.midtransOrderId && ["requested", "pending"].includes(item.paymentStatus))
      .filter((item) => !autoSyncedPurchaseIdsRef.current.has(item.id))
      .slice(0, 4);
    if (!eligiblePurchases.length) return;

    let cancelled = false;
    (async () => {
      for (const item of eligiblePurchases) {
        autoSyncedPurchaseIdsRef.current.add(item.id);
        await syncMidtransPayment(item).catch(() => {});
      }
      if (!cancelled) await loadDashboardData(auth, { preserveDrafts: true }).catch(() => {});
    })();

    return () => {
      cancelled = true;
    };
  }, [auth?.token, activeView, purchases]);

  function userFriendlyError(message, status) {
    const raw = String(message || "").trim();
    const normalized = raw.toLowerCase();
    if (normalized === "invalid credentials") return "Username atau password belum sesuai. Silakan coba lagi.";
    if (status === 402 && (normalized.includes("limit reached") || normalized.includes("is not included"))) {
      if (normalized.includes("ai agent")) return "Batas AI CS untuk paket saat ini sudah terpakai. Upgrade paket untuk menambah agent.";
      if (normalized.includes("whatsapp number")) return "Batas nomor WhatsApp untuk paket saat ini sudah terpakai. Upgrade paket untuk menambah nomor.";
      if (normalized.includes("human user")) return "Batas anggota tim untuk paket saat ini sudah terpakai. Upgrade paket untuk menambah anggota.";
      return "Batas fitur untuk paket saat ini sudah terpakai. Upgrade paket untuk menambah kapasitas.";
    }
    if (status === 402 || normalized.includes("insufficient") || normalized.includes("credit")) {
      return "Kredit test sudah habis. Playground belum menjalankan tool atau membuat data baru. Tambah kredit di Billing untuk lanjut tes.";
    }
    if (normalized === "missing bearer token" || normalized === "invalid token" || normalized === "invalid organization access") return "Sesi login sudah berakhir. Silakan login ulang.";
    if (normalized.includes("current password is incorrect")) return "Password saat ini belum sesuai.";
    if (normalized.includes("forbidden") || normalized.includes("access denied") || normalized.startsWith("only ")) return "Akun ini belum punya akses untuk melakukan aksi tersebut.";
    if (normalized.includes("not found or expired") || normalized.includes("expired")) return "Tautan atau kode sudah tidak berlaku. Silakan minta yang baru.";
    if (normalized.includes("failed to sign token") || normalized.includes("request failed")) return "Sistem belum bisa memproses permintaan. Silakan coba lagi.";
    if (!raw && status === 401) return "Sesi login sudah berakhir. Silakan login ulang.";
    if (!raw && status === 403) return "Akun ini belum punya akses untuk melakukan aksi tersebut.";
    if (!raw) return "Sistem belum bisa memproses permintaan. Silakan coba lagi.";
    return raw;
  }

  async function loadDashboardData(currentAuth = auth, options = {}) {
    if (!currentAuth?.token) return;
    const preserveDrafts = Boolean(options.preserveDrafts);
    const requestedKnowledgeAgentId = activeKnowledgeAIAgentId;
    const currentRole = currentAuth.user?.role ?? "";
    const currentCanManagePricing = currentRole === "owner" || currentRole === "super_admin";
    const currentCanViewSystemHealth = currentRole === "owner";
    setIsRefreshing(true);
    const warnings = [];
    let loadedBusinessTools = businessTools;
    if (!loadedBusinessTools.length) setBusinessToolsLoadState("loading");
    try {
      const toolPayload = await requestJSON(`${apiBase}/api/business-tools`, {}, { token: currentAuth.token });
      loadedBusinessTools = toolPayload.items ?? [];
      setBusinessTools(loadedBusinessTools);
      setBusinessToolsLoadState("loaded");
    } catch (toolError) {
      if (toolError?.sessionExpired) {
        setIsRefreshing(false);
        return;
      }
      setBusinessToolsLoadState("error");
      if (![403, 404].includes(toolError?.status ?? 0)) {
        warnings.push(`Gagal memuat alat bisnis: ${toolError?.message ?? "unknown error"}`);
      }
    }
    const loadedBusinessToolKeys = new Set(
      (loadedBusinessTools || [])
        .filter((item) => item.installed ?? item.enabled)
        .map((item) => item.key)
    );
    const requests = [
      { key: "summary", label: "dashboard summary", run: () => requestJSON(`${apiBase}/api/dashboard/summary`, {}, { token: currentAuth.token }) },
      { key: "organizations", label: "organizations", run: () => requestJSON(`${apiBase}/api/organizations`, {}, { token: currentAuth.token }) },
      { key: "teamMembers", label: "team members", run: () => requestJSON(`${apiBase}/api/team/members`, {}, { token: currentAuth.token }) },
    { key: "teamInvites", label: "undangan tim", run: () => requestJSON(`${apiBase}/api/team/invites`, {}, { token: currentAuth.token }) },
    { key: "waSessions", label: "koneksi WhatsApp", run: () => requestJSON(`${apiBase}/api/whatsapp/sessions`, {}, { token: currentAuth.token }) },
      { key: "metaCloudConfig", label: "konfigurasi WhatsApp resmi", run: () => requestJSON(`${apiBase}/api/whatsapp/meta/config`, {}, { token: currentAuth.token }) },
      { key: "aiAgents", label: "AI agent", run: () => requestJSON(`${apiBase}/api/ai-agents`, {}, { token: currentAuth.token }) },
      loadedBusinessToolKeys.has("commerce") ? { key: "commerceProducts", label: "produk dan stok", run: () => requestJSON(`${apiBase}/api/commerce/products?status=all`, {}, { token: currentAuth.token }) } : null,
      loadedBusinessToolKeys.has("commerce") ? { key: "commerceOrderSettings", label: "setting pesanan", run: () => requestJSON(`${apiBase}/api/commerce/order-settings`, {}, { token: currentAuth.token }) } : null,
      loadedBusinessToolKeys.has("commerce") ? { key: "commerceOrderPipelines", label: "pipeline pesanan", run: () => requestJSON(`${apiBase}/api/commerce/order-pipelines`, {}, { token: currentAuth.token }) } : null,
      loadedBusinessToolKeys.has("commerce") ? { key: "commerceOrderDrafts", label: "draft pesanan", run: () => requestJSON(`${apiBase}/api/commerce/order-drafts`, {}, { token: currentAuth.token }) } : null,
      loadedBusinessToolKeys.has("booking") ? { key: "bookingServices", label: "layanan booking", run: () => requestJSON(`${apiBase}/api/booking/services?status=all`, {}, { token: currentAuth.token }) } : null,
      loadedBusinessToolKeys.has("booking") ? { key: "bookingAppointments", label: "appointment booking", run: () => requestJSON(`${apiBase}/api/booking/appointments`, {}, { token: currentAuth.token }) } : null,
      { key: "notifications", label: "notifications", run: () => requestJSON(`${apiBase}/api/notifications`, {}, { token: currentAuth.token }) },
      { key: "inbox", label: "inbox", run: () => requestJSON(`${apiBase}/api/inbox`, {}, { token: currentAuth.token }) },
      { key: "contacts", label: "contacts", run: () => requestJSON(`${apiBase}/api/contacts`, {}, { token: currentAuth.token }) },
      { key: "analytics", label: "analytics", run: () => requestJSON(`${apiBase}/api/analytics/overview?range=${analyticsRange}`, {}, { token: currentAuth.token }) },
      { key: "settings", label: "ai settings", run: () => requestJSON(`${apiBase}/api/ai-settings`, {}, { token: currentAuth.token }) },
      currentCanViewSystemHealth ? { key: "health", label: "system health", run: () => requestJSON(`${apiBase}/api/system/health`, {}, { token: currentAuth.token }) } : null,
      activeWaSessionId ? { key: "escalationGroup", label: "escalation group", run: () => requestJSON(`${apiBase}/api/escalation-group?sessionId=${encodeURIComponent(activeWaSessionId)}`, {}, { token: currentAuth.token }) } : null,
      { key: "groupNotificationRules", label: "group notification rules", run: () => requestJSON(`${apiBase}/api/group-notification-rules`, {}, { token: currentAuth.token }) },
      { key: "faqs", label: "faq knowledge", run: () => requestJSON(`${apiBase}/api/knowledge/faqs${activeKnowledgeAIAgentId ? `?aiAgentId=${encodeURIComponent(activeKnowledgeAIAgentId)}` : ""}`, {}, { token: currentAuth.token }) },
      { key: "documents", label: "document knowledge", run: () => requestJSON(`${apiBase}/api/knowledge/documents${activeKnowledgeAIAgentId ? `?aiAgentId=${encodeURIComponent(activeKnowledgeAIAgentId)}` : ""}`, {}, { token: currentAuth.token }) },
      { key: "positions", label: "job positions", run: () => requestJSON(`${apiBase}/api/job-positions${activeKnowledgeAIAgentId ? `?aiAgentId=${encodeURIComponent(activeKnowledgeAIAgentId)}` : ""}`, {}, { token: currentAuth.token }) },
    { key: "playgroundShortcuts", label: "shortcut playground", run: () => requestJSON(`${apiBase}/api/playground/shortcuts`, {}, { token: currentAuth.token }) },
    { key: "aiRuns", label: "riwayat AI", run: () => requestJSON(`${apiBase}/api/ai-runs`, {}, { token: currentAuth.token }) },
      { key: "wallet", label: "wallet", run: () => requestJSON(`${apiBase}/api/billing/wallet`, {}, { token: currentAuth.token }) },
      { key: "usageLogs", label: "usage logs", run: () => requestJSON(`${apiBase}/api/billing/usage-logs`, {}, { token: currentAuth.token }) },
      { key: "billingAnalytics", label: "billing analytics", run: () => requestJSON(`${apiBase}/api/billing/analytics`, {}, { token: currentAuth.token }) },
      currentCanManagePricing ? { key: "pricing", label: "pricing settings", run: () => requestJSON(`${apiBase}/api/billing/pricing`, {}, { token: currentAuth.token }) } : null,
      { key: "aiModels", label: "ai models", run: () => loadAIModelsCatalog(currentAuth) },
      currentRole === "owner" ? { key: "adjustments", label: "credit adjustments", run: () => requestJSON(`${apiBase}/api/billing/adjustments`, {}, { token: currentAuth.token }) } : null,
      { key: "agents", label: "agents", run: () => requestJSON(`${apiBase}/api/agents`, {}, { token: currentAuth.token }) },
      { key: "packages", label: "packages", run: () => requestJSON(`${apiBase}/api/billing/packages`, {}, { token: currentAuth.token }) },
      { key: "purchases", label: "purchases", run: () => requestJSON(`${apiBase}/api/billing/purchases`, {}, { token: currentAuth.token }) },
    ].filter(Boolean);

    const results = await Promise.allSettled(requests.map((item) => item.run()));
    if (results.some((result) => result.status === "rejected" && result.reason?.sessionExpired)) {
      setIsRefreshing(false);
      return;
    }

    const shouldPreserveDrafts = preserveDrafts || isEditingFormControl();
    const shouldPreservePricing = shouldPreserveDrafts || pricingFormDirtyRef.current;
    const shouldRefreshKnowledge = !shouldPreserveDrafts || Boolean(options.refreshKnowledge);
    const canApplyKnowledgeResult = requestedKnowledgeAgentId === activeKnowledgeAIAgentIdRef.current;
    results.forEach((result, index) => {
      const { key, label } = requests[index];
      if (result.status === "rejected") {
        if (![403, 404].includes(result.reason?.status ?? 0)) {
          warnings.push(`Gagal memuat ${label}: ${result.reason?.message ?? "unknown error"}`);
        }
        return;
      }

      const payload = result.value;
      if (key === "summary") setSummary({ ...(payload.summary ?? {}), crm: payload.crm ?? {}, wallet: payload.wallet ?? null });
      if (key === "organizations") setOrganizations(payload.items ?? []);
      if (key === "teamMembers") setTeamMembers(payload.items ?? []);
      if (key === "teamInvites") setTeamInvites(payload.items ?? []);
      if (key === "waSessions") {
        const items = (payload.items ?? []).filter((item) => item.provider === "meta_cloud");
        setWaSessions(items);
        const pending = pendingWaSessionId ? items.find((item) => item.id === pendingWaSessionId) : null;
        if (pending?.status === "connected") {
          setPendingWaSessionId("");
          setActiveWaSessionId(pending.id);
          setWaStatus({ status: pending.status ?? "connected", details: pending.details ?? {} });
          return;
        }
        if (pendingWaSessionId) return;
        const selectableItems = items.filter((item) => item.id !== pendingWaSessionId);
        const selected = selectableItems.find((item) => item.id === activeWaSessionId) || selectableItems.find((item) => item.isDefault) || selectableItems[0];
        setActiveWaSessionId(selected?.id ?? "");
        setWaStatus({
          status: selected?.status ?? "disconnected",
          details: selected?.details ?? {},
        });
      }
      if (key === "metaCloudConfig") {
        setMetaCloudConfig(payload ?? { enabled: false });
      }
      if (key === "aiAgents") {
        const items = payload.items ?? [];
        setAiAgents(items);
        const defaultAgentId = items[0]?.id || "";
        setWaSessionForm((current) => ({
          ...current,
          aiAgentId: items.some((agent) => agent.id === current.aiAgentId) ? current.aiAgentId : defaultAgentId,
        }));
        setActiveKnowledgeAIAgentId((current) => (current && items.some((agent) => agent.id === current) ? current : ""));
        setPlaygroundForm((current) => ({
          ...current,
          aiAgentId: items.some((agent) => agent.id === current.aiAgentId) ? current.aiAgentId : defaultAgentId,
        }));
      }
      if (key === "notifications") setNotifications(payload.items ?? []);
      if (key === "businessTools") setBusinessTools(payload.items ?? []);
      if (key === "commerceProducts") setCommerceProducts(payload.items ?? []);
      if (key === "commerceOrderSettings") setCommerceOrderSettings(payload.settings ?? defaultCommerceOrderSettings());
      if (key === "commerceOrderPipelines") {
        setCommerceOrderPipelines(payload.pipelines ?? []);
        setCommerceOrderStages(payload.stages ?? []);
      }
      if (key === "commerceOrderDrafts") setCommerceOrderDrafts(payload.items ?? []);
      if (key === "bookingServices") setBookingServices(payload.items ?? []);
      if (key === "bookingAppointments") setBookingAppointments(payload.items ?? []);
      if (key === "inbox") setInbox(payload.items ?? []);
      if (key === "contacts") setContacts(payload.items ?? []);
      if (key === "analytics") setAnalytics(payload.analytics ?? null);
      if (key === "settings" && !shouldPreserveDrafts) setSettings(payload.settings ?? null);
      if (key === "health") setHealth(payload ?? null);
      if (key === "escalationGroup") setEscalationGroup(payload.escalationGroup ?? null);
      if (key === "groupNotificationRules") {
        setGroupNotificationRules((payload.items ?? []).filter(Boolean));
        setGroupNotificationVariables(payload.variables ?? []);
      }
      if (key === "faqs" && shouldRefreshKnowledge && canApplyKnowledgeResult) setFaqs(payload.items ?? []);
      if (key === "documents" && shouldRefreshKnowledge && canApplyKnowledgeResult) setDocuments(payload.items ?? []);
      if (key === "positions" && shouldRefreshKnowledge && canApplyKnowledgeResult) setPositions(payload.items ?? []);
      if (key === "playgroundShortcuts" && !shouldPreserveDrafts) setPlaygroundShortcuts(payload.items ?? []);
      if (key === "aiRuns") setAiRuns(payload.items ?? []);
      if (key === "wallet") {
        setWallet(payload.wallet ?? null);
        setBillingPlan(payload.plan ?? null);
      }
      if (key === "usageLogs") setUsageLogs(payload.items ?? []);
      if (key === "billingAnalytics") setBillingAnalytics(payload.analytics ?? null);
      if (key === "pricing" && !shouldPreservePricing) setPricing(payload.pricing ?? null);
      if (key === "aiModels") setAiModels(payload.items ?? []);
      if (key === "adjustments") setCreditAdjustments(payload.items ?? []);
      if (key === "agents" && !shouldPreserveDrafts) setAgents(payload.items ?? []);
      if (key === "packages" && !shouldPreserveDrafts) {
        const annualDiscountPercent = Number(payload.annualDiscountPercent ?? 10);
        setPackages((payload.items ?? []).map((item) => ({ ...item, annualDiscountPercent })));
      }
      if (key === "purchases") setPurchases(payload.items ?? []);
    });

    setLoadWarnings(warnings);
    setLastSyncedAt(new Date().toISOString());
    setIsRefreshing(false);
  }

  async function login(username, password) {
    setLoading(true);
    setError("");
    try {
      const payload = await requestJSON(`${apiBase}/api/auth/login`, {
        method: "POST",
        body: JSON.stringify({ username, password }),
      }, { needsAuth: false });
      const nextAuth = { token: payload.token, user: payload.user };
      const requestedPath = requestedDashboardPath();
      const routeConfig = setupRouteConfigFromPath(requestedPath);
      const nextView = viewFromPath(requestedPath, payload.user.role) || storedViewForRole(payload.user.role);
      resetWhatsAppViewState();
      setWallet(null);
      setBillingPlan(null);
      setAuth(nextAuth);
      window.localStorage.setItem(authStorageKey, JSON.stringify(nextAuth));
      try {
        window.sessionStorage.removeItem(sessionExpiredStorageKey);
      } catch {}
      setActiveView(nextView);
      if (routeConfig?.tab) setAgentTab(routeConfig.tab);
      clearLoginNextPath();
      router.replace(isDashboardAppPath(requestedPath) ? requestedPath : pathForView(nextView));
    } catch (loginError) {
      setError(userFriendlyError(loginError.message, loginError.status));
    } finally {
      setLoading(false);
    }
  }

  async function registerAccount(payload) {
    setLoading(true);
    setError("");
    try {
      const response = await requestJSON(`${apiBase}/api/auth/register`, {
        method: "POST",
        body: JSON.stringify(payload),
      }, { needsAuth: false });
      const nextAuth = { token: response.token, user: response.user };
      const nextView = onboardingStartViewForRole(response.user.role);
      resetWhatsAppViewState();
      setWallet(null);
      setBillingPlan(null);
      setAuth(nextAuth);
      window.localStorage.setItem(authStorageKey, JSON.stringify(nextAuth));
      try {
        window.sessionStorage.removeItem(sessionExpiredStorageKey);
      } catch {}
      window.localStorage.setItem(firstRunOnboardingStorageKeyForAuth(nextAuth), "pending");
      window.localStorage.setItem(activeViewStorageKey, nextView);
      setFirstRunOnboardingIndex(0);
      setFirstRunOnboardingOpen(true);
      setActiveView(nextView);
      if (nextView === "agents") setAgentTab("instructions");
      clearLoginNextPath();
      router.replace(nextView === "agents" ? "/agents/setup" : pathForView(nextView));
    } catch (registerError) {
      setError(registerError.message);
    } finally {
      setLoading(false);
    }
  }

  function logout(options = {}) {
    const sessionExpired = options?.sessionExpired === true;
    const requestedPath = sessionExpired ? requestedDashboardPath() : "";
    const nextPath = isDashboardAppPath(requestedPath) ? requestedPath : "";
    if (!sessionExpired && auth?.token) handledExpiredTokenRef.current = auth.token;
    try {
      if (sessionExpired) {
        window.sessionStorage.setItem(sessionExpiredStorageKey, "1");
      } else {
        window.sessionStorage.removeItem(sessionExpiredStorageKey);
      }
    } catch {}
    window.localStorage.removeItem(authStorageKey);
    window.localStorage.removeItem(legacyAuthStorageKey);
    window.localStorage.removeItem(activeViewStorageKey);
    window.localStorage.removeItem(legacyActiveViewStorageKey);
    if (nextPath) {
      rememberLoginNextPath(nextPath);
    } else {
      clearLoginNextPath();
    }
    setAuth(null);
    setFirstRunOnboardingOpen(false);
    setFirstRunOnboardingIndex(0);
    setActiveView("operations");
    const loginQuery = new URLSearchParams();
    if (nextPath) loginQuery.set("next", nextPath);
    if (sessionExpired) loginQuery.set("reason", "session-expired");
    const loginSearch = loginQuery.toString();
    router.replace(loginSearch ? `/login?${loginSearch}` : "/login");
    setSummary(null);
    setAnalytics(null);
    setHealth(null);
    setSettings(null);
    setWallet(null);
    setBillingPlan(null);
    resetWhatsAppViewState();
    setNotifications([]);
    setNotificationReadIds([]);
    setUsageLogs([]);
    setContacts([]);
    setBillingAnalytics(null);
    setUsageLogDetail(null);
    setPricing(null);
    setCreditAdjustments([]);
    setAgents([]);
    setOrganizations([]);
    setTeamMembers([]);
    setTeamInvites([]);
    setAiAgents([]);
    setActiveKnowledgeAIAgentId("");
    setPackages([]);
    setPurchases([]);
    setFaqs([]);
    setDocuments([]);
    setPositions([]);
    setBusinessTools([]);
    setBusinessToolsLoadState("loading");
    setCommerceProducts([]);
    setCommerceOrderDrafts([]);
    setCommerceOrderPipelines([]);
    setCommerceOrderStages([]);
    setCommerceOrderSettings(defaultCommerceOrderSettings());
    setCommerceProductForm(emptyCommerceProductForm());
    setCommerceOrderDraftForm(emptyCommerceOrderDraftForm());
    setBookingServices([]);
    setBookingAppointments([]);
    setBookingServiceForm(emptyBookingServiceForm());
    setBookingAppointmentForm(emptyBookingAppointmentForm());
    setPlaygroundShortcuts([]);
    setPlaygroundBatchResults([]);
    setPlaygroundForm({ customerName: "Pelanggan Test", messageText: "", aiAgentId: "" });
    setAiRuns([]);
    setInbox([]);
    setSelectedConversation(null);
    setActiveInboxWaSessionId("");
    setActiveInboxAIAgentId("");
    setPlaygroundMessages([createPlaygroundWelcome()]);
    setError(sessionExpired ? "Sesi login sudah berakhir. Silakan login ulang." : "");
    setLoadWarnings([]);
    setLastSyncedAt("");
  }

  async function refreshConversation(id) {
    const payload = await requestJSON(`${apiBase}/api/conversations/${id}`);
    setSelectedConversation(payload);
  }

  async function fetchConversation(id) {
    try {
      setError("");
      await refreshConversation(id);
    } catch (loadError) {
      setError(loadError.message);
    }
  }

  function navigateToView(viewId, method = "push") {
    const canonical = canonicalViewId(viewId);
    if (role && !isViewAvailableForCurrentState(canonical)) return;
    if (role) window.localStorage.setItem(activeViewStorageKey, canonical);
    router[method](pathForView(canonical));
  }

  const completeFirstRunOnboarding = useCallback(() => {
    try {
      window.localStorage.setItem(firstRunOnboardingStorageKey, "seen");
    } catch {
      // Local storage can be blocked in private contexts; closing the modal still keeps the current session usable.
    }
    setFirstRunOnboardingOpen(false);
    setFirstRunOnboardingIndex(0);
  }, [firstRunOnboardingStorageKey]);

  const nextFirstRunOnboarding = useCallback(() => {
    setFirstRunOnboardingIndex((current) => Math.min(current + 1, firstRunOnboardingSlides.length - 1));
  }, []);

  const backFirstRunOnboarding = useCallback(() => {
    setFirstRunOnboardingIndex((current) => Math.max(current - 1, 0));
  }, []);

  const startFirstRunSetup = useCallback(() => {
    completeFirstRunOnboarding();
    setAgentTab("instructions");
    navigateToView("agents");
  }, [completeFirstRunOnboarding]);

  async function runConversationAction(id, action, body) {
    const isManualMessage = action === "manual-message";
    const previousManualMessage = manualMessage;
    const previousManualMedia = manualMedia;
    try {
      setBusyKey(`conversation-${action}`);
      setError("");
      if (isManualMessage) {
        setManualMessage("");
        setManualMedia(null);
      }
      await requestJSON(`${apiBase}/api/conversations/${id}/${action}`, {
        method: "POST",
        body: body ? JSON.stringify(body) : undefined,
      });
      await refreshConversation(id);
      if (isManualMessage) {
        loadDashboardData().catch((loadError) => setLoadWarnings((current) => [...current, `Gagal refresh dashboard setelah kirim pesan: ${loadError.message}`]));
      } else {
        await loadDashboardData();
      }
      if (action === "takeover") setTakeoverNote("");
    } catch (actionError) {
      if (isManualMessage) {
        setManualMessage(previousManualMessage);
        setManualMedia(previousManualMedia);
      }
      setError(actionError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function saveConversationWorkflow(id, body) {
    try {
      setBusyKey("conversation-workflow");
      setError("");
      const payload = await requestJSON(`${apiBase}/api/conversations/${id}/workflow`, {
        method: "POST",
        body: JSON.stringify(body),
      });
      setSelectedConversation(payload);
      await loadDashboardData(auth, { preserveDrafts: true });
      showToast("Workflow chat disimpan.", "success");
    } catch (workflowError) {
      setError(workflowError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function createDealFromConversation(conversation) {
    if (!conversation?.id) return;
    try {
      setBusyKey("deal-from-conversation");
      setError("");
      const payload = await requestJSON(`${apiBase}/api/deals`, {
        method: "POST",
        body: JSON.stringify({
          title: `Deal - ${conversation.contactName || conversation.phone || "WhatsApp"}`,
          contactId: conversation.contactId || "",
          conversationId: conversation.id,
          priority: conversation.priority || "normal",
          ownerAgentId: conversation.assignedToId || "",
          source: "WhatsApp Inbox",
          notes: conversation.lastMessageText || conversation.internalNote || "",
        }),
      });
      showToast({ title: "Follow-up dibuat", message: "Chat sudah masuk ke Prospek & Follow-up." });
      await loadDashboardData(auth, { preserveDrafts: true });
      if (payload?.item?.id) setFocusDealId(payload.item.id);
      navigateToView("deals");
      return payload.item;
    } catch (dealError) {
      setError(dealError.message);
      return null;
    } finally {
      setBusyKey("");
    }
  }

  function openDealFromConversation(deal) {
    if (deal?.id) setFocusDealId(deal.id);
    navigateToView("deals");
  }

  function openContactFromConversation(conversation) {
    if (conversation?.contactId) setFocusContactId(conversation.contactId);
    navigateToView("contacts");
  }

  async function ensureWhatsAppSession() {
    const existingSessionId = activeWaSessionId || visibleWaSessions.find((item) => item.isDefault)?.id || visibleWaSessions[0]?.id;
    if (existingSessionId) return existingSessionId;

    setBusyKey("wa-session-auto");
    const payload = await requestJSON(`${apiBase}/api/whatsapp/sessions`, {
      method: "POST",
      body: JSON.stringify({ label: "Primary WhatsApp", mode: "real" }),
    });
    const sessionId = payload.id ?? payload.session?.id ?? "";
    if (!sessionId) throw new Error("Gagal menyiapkan WhatsApp session pertama.");
    setActiveWaSessionId(sessionId);
    await loadDashboardData(auth, { preserveDrafts: true });
    return sessionId;
  }

  async function requestQr(sessionIdOverride = "") {
    let sessionId = sessionIdOverride || activeWaSessionId || visibleWaSessions.find((item) => item.isDefault)?.id || visibleWaSessions[0]?.id;
    if (!sessionId) {
      setError("Isi nama perangkat dan pilih AI agent dulu, lalu klik Hubungkan WhatsApp.");
      return;
    }
    try {
      setError("");
      setBusyKey(`wa-qr-${sessionId}`);
      setActiveWaSessionId(sessionId);
      const statusPayload = await requestJSON(`${apiBase}/api/whatsapp/sessions/${sessionId}/connect`, { method: "POST" });
      setWaStatus(statusPayload);
      if (statusPayload?.details?.qr) {
        setQrData({
          qr: statusPayload.details.qr,
          expiresIn: statusPayload.details.expiresIn ?? 60,
          mode: statusPayload.details.mode ?? "real",
        });
        return;
      }
      const qrPayload = await requestJSON(`${apiBase}/api/whatsapp/sessions/${sessionId}/qr`);
      setQrData(qrPayload);
    } catch (qrError) {
      setError(qrError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function disconnectWhatsApp() {
    const sessionId = activeWaSessionId || visibleWaSessions.find((item) => item.isDefault)?.id || visibleWaSessions[0]?.id;
    if (!sessionId) {
      setError("Belum ada WhatsApp session untuk organization aktif.");
      return;
    }
    if (!(await askConfirm({
      title: "Disconnect WhatsApp?",
      message: "Chat ops akan terblokir sampai QR discan ulang.",
      confirmLabel: "Disconnect",
      tone: "danger",
    }))) return;
    try {
      setBusyKey("wa-disconnect");
      setError("");
      const payload = await requestJSON(`${apiBase}/api/whatsapp/sessions/${sessionId}/disconnect`, { method: "POST" });
      setWaStatus(payload);
      setQrData(null);
      setQrImageSrc("");
      await loadDashboardData();
    } catch (disconnectError) {
      setError(disconnectError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function switchOrganization(organizationId) {
    if (!organizationId || organizationId === auth?.user?.organizationId) return;
    try {
      setBusyKey("organization-switch");
      setError("");
      const payload = await requestJSON(`${apiBase}/api/organizations/switch`, {
        method: "POST",
        body: JSON.stringify({ organizationId }),
      });
      const selected = organizations.find((item) => item.id === payload.organizationId);
      const nextAuth = {
        token: payload.token,
        user: {
          ...auth.user,
          organizationId: payload.organizationId,
          organizationRole: payload.organizationRole,
          organizationName: selected?.name ?? auth.user.organizationName,
          organizationSlug: selected?.slug ?? auth.user.organizationSlug,
        },
      };
      setAuth(nextAuth);
      window.localStorage.setItem(authStorageKey, JSON.stringify(nextAuth));
      resetWhatsAppViewState();
      setWallet(null);
      setBillingPlan(null);
      await loadDashboardData(nextAuth);
    } catch (switchError) {
      setError(switchError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function createTeamInvite(event) {
    event?.preventDefault?.();
    try {
      setBusyKey("team-invite");
      setError("");
      await requestJSON(`${apiBase}/api/team/invites`, {
        method: "POST",
        body: JSON.stringify(inviteForm),
      });
      setInviteForm({ email: "", phone: "", role: "operator" });
      await loadDashboardData();
    } catch (inviteError) {
      setError(inviteError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function revokeTeamInvite(id) {
    try {
      setBusyKey(`team-invite-${id}`);
      await requestJSON(`${apiBase}/api/team/invites/${id}`, { method: "DELETE" });
      await loadDashboardData();
    } catch (inviteError) {
      setError(inviteError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function updateTeamMember(id, roleValue, isActive = true) {
    try {
      setBusyKey(`team-member-${id}`);
      await requestJSON(`${apiBase}/api/team/members/${id}`, {
        method: "PUT",
        body: JSON.stringify({ role: roleValue, isActive }),
      });
      await loadDashboardData();
    } catch (memberError) {
      setError(memberError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function removeTeamMember(id) {
    if (!(await askConfirm({
      title: "Nonaktifkan anggota?",
      message: "Anggota ini tidak bisa lagi mengakses organization aktif.",
      confirmLabel: "Nonaktifkan",
      tone: "danger",
    }))) return;
    try {
      setBusyKey(`team-member-${id}`);
      await requestJSON(`${apiBase}/api/team/members/${id}`, { method: "DELETE" });
      await loadDashboardData();
    } catch (memberError) {
      setError(memberError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function createWhatsAppSession(event) {
    event?.preventDefault?.();
    const label = String(waSessionForm.label || "").trim();
    const aiAgentId = String(waSessionForm.aiAgentId || "").trim();
    const provider = "meta_cloud";
    let sessionId = "";
    if (!label || !aiAgentId) {
      setError("Nama perangkat dan AI agent wajib diisi.");
      return;
    }
    try {
      setBusyKey("wa-session-create");
      setError("");
      const payload = await requestJSON(`${apiBase}/api/whatsapp/sessions`, {
        method: "POST",
        body: JSON.stringify({ label, aiAgentId, mode: "real", provider }),
      });
      sessionId = payload.id ?? "";
      setActiveWaSessionId(sessionId);
      setPendingWaSessionId(sessionId);
      setBusyKey(`wa-meta-${sessionId}`);
      const connected = await connectOfficialWhatsApp(sessionId);
      setWaStatus(connected);
      setPendingWaSessionId("");
      setWaSessionForm({ label: "", aiAgentId, provider });
      await loadDashboardData(auth, { preserveDrafts: true });
    } catch (sessionError) {
      if (sessionId) {
        // Meta signup may have completed even if its browser callback failed.
        setPendingWaSessionId("");
        await loadDashboardData(auth, { preserveDrafts: true }).catch(() => {});
      }
      setError(sessionError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function connectOfficialWhatsApp(sessionId) {
    const configPayload = await requestJSON(`${apiBase}/api/whatsapp/meta/config`);
    validateCoexistenceConfig(configPayload);
    await loadMetaSDK(configPayload.appId, configPayload.graphVersion);
    const signup = await new Promise((resolve, reject) => {
      let sessionInfo = null;
      let code = "";
      let completed = false;
      const finish = (error, value) => {
        if (completed) return;
        completed = true;
        window.clearTimeout(timeout);
        window.removeEventListener("message", receiveSessionInfo);
        if (error) reject(error);
        else resolve(value);
      };
      const maybeFinish = () => {
        if (code && sessionInfo?.wabaId && sessionInfo?.phoneNumberId) {
          finish(null, { ...sessionInfo, code });
        }
      };
      const receiveSessionInfo = (event) => {
        try {
          const parsed = parseCoexistenceEvent(event);
          if (parsed) {
            sessionInfo = parsed;
            maybeFinish();
          }
        } catch (signupError) {
          finish(signupError);
        }
      };
      const timeout = window.setTimeout(() => finish(new Error("Proses koneksi WhatsApp resmi melewati batas waktu.")), 10 * 60 * 1000);
      window.addEventListener("message", receiveSessionInfo);
      try {
        window.FB.login((response) => {
          code = response?.authResponse?.code || "";
          if (!code) {
            finish(new Error("Meta tidak memberikan izin koneksi WhatsApp."));
            return;
          }
          maybeFinish();
        }, {
          config_id: configPayload.configurationId,
          response_type: "code",
          override_default_response_type: true,
          extras: {
            setup: {},
            feature: "whatsapp_embedded_signup",
            featureType: "whatsapp_business_app_onboarding",
            sessionInfoVersion: "3",
          },
        });
      } catch {
        finish(new Error("Jendela koneksi Meta tidak dapat dibuka. Coba kembali dari browser Anda."));
      }
    });
    return requestJSON(`${apiBase}/api/whatsapp/sessions/${encodeURIComponent(sessionId)}/meta-complete`, {
      method: "POST",
      body: JSON.stringify(signup),
    });
  }

  async function reconnectOfficialWhatsApp(sessionId) {
    try {
      setBusyKey(`wa-meta-${sessionId}`);
      setError("");
      const connected = await connectOfficialWhatsApp(sessionId);
      setWaStatus(connected);
      await loadDashboardData(auth, { preserveDrafts: true });
    } catch (connectError) {
      setError(connectError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function loadMetaTemplates(sessionId = activeWaSessionId) {
    if (!sessionId) return;
    try {
      setBusyKey("wa-meta-template-list");
      setError("");
      const payload = await requestJSON(`${apiBase}/api/whatsapp/sessions/${encodeURIComponent(sessionId)}/meta-templates`);
      const items = payload.items || [];
      setMetaTemplates(items);
      setMetaTemplateSendForm((current) => ({
        ...current,
        name: current.name || String(items[0]?.name || ""),
        language: current.language || String(items[0]?.language || "id"),
      }));
    } catch (templateError) {
      setError(templateError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function createMetaTemplate(event) {
    event?.preventDefault?.();
    if (!activeWaSessionId) return;
    try {
      setBusyKey("wa-meta-template-create");
      setError("");
      await requestJSON(`${apiBase}/api/whatsapp/sessions/${encodeURIComponent(activeWaSessionId)}/meta-templates`, {
        method: "POST",
        body: JSON.stringify(metaTemplateForm),
      });
      showToast({ title: "Template diajukan", message: "Template dikirim ke Meta untuk ditinjau." });
      await loadMetaTemplates(activeWaSessionId);
    } catch (templateError) {
      setError(templateError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function sendMetaTemplate(event) {
    event?.preventDefault?.();
    if (!activeWaSessionId) return;
    try {
      setBusyKey("wa-meta-template-send");
      setError("");
      await requestJSON(`${apiBase}/api/whatsapp/sessions/${encodeURIComponent(activeWaSessionId)}/meta-template-send`, {
        method: "POST",
        body: JSON.stringify(metaTemplateSendForm),
      });
      showToast({ title: "Template dikirim", message: "Pesan template resmi sudah dikirim." });
    } catch (templateError) {
      setError(templateError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function cancelPendingWhatsAppSession(sessionId = "") {
    const id = sessionId || pendingWaSessionId;
    if (!id || id !== pendingWaSessionId) return;
    const pendingSession = waSessions.find((item) => item.id === id);
    if (pendingSession?.status === "connected" || (waStatus?.status === "connected" && waStatus?.details?.sessionId === id)) {
      setPendingWaSessionId("");
      return;
    }
    try {
      setBusyKey(`wa-delete-${id}`);
      await requestJSON(`${apiBase}/api/whatsapp/sessions/${id}`, { method: "DELETE" });
      if (activeWaSessionId === id) setActiveWaSessionId("");
      setPendingWaSessionId("");
      setQrData(null);
      setQrImageSrc("");
      setWaStatus({ status: "disconnected", details: {} });
      await loadDashboardData(auth, { preserveDrafts: true });
    } catch (cleanupError) {
      setError(cleanupError.message);
    } finally {
      setBusyKey("");
    }
  }

  function workspaceAvailableAIModelIds() {
    const fromCatalog = (aiModels || [])
      .filter((model) => model?.available !== false)
      .map((model) => String(model?.id || "").trim())
      .filter(Boolean);
    if (fromCatalog.length) return fromCatalog;
    const fromPricing = (pricingForm.availableModelIds || [])
      .map((modelId) => String(modelId || "").trim())
      .filter(Boolean);
    return fromPricing.length ? fromPricing : defaultAvailableModelIds;
  }

  function resolveWorkspaceAIModelName(requestedModelName) {
    const availableIds = workspaceAvailableAIModelIds();
    const availableLookup = new Set(availableIds.map((modelId) => modelId.toLowerCase()));
    const requested = String(requestedModelName || "").trim();
    if (requested && availableLookup.has(requested.toLowerCase())) return requested;
    return availableIds[0] || "openai/gpt-4o-mini";
  }

  async function createAIAgent(event, formOverride) {
    event?.preventDefault?.();
    const sourceForm = formOverride || aiAgentForm;
    const name = String(sourceForm.name || "").trim();
    if (!name) {
      setError("Nama AI agent wajib diisi.");
      return false;
    }
    const agentId = String(sourceForm.id || "").trim();
    const memoryBundle = customerMemoryFormBundle(sourceForm.customerMemoryEnabled === true);
    const body = JSON.stringify({
      ...defaultAIAgentForm(),
      ...sourceForm,
      ...memoryBundle,
      name,
      modelName: resolveWorkspaceAIModelName(sourceForm.modelName),
      fallbackWaitingMessage: sourceForm.fallbackWaitingMessage || fallbackMessage,
    });
    try {
      setBusyKey(agentId ? "ai-agent-save" : "ai-agent-create");
      setAIAgentSaveFeedback({ status: "saving", message: "Menyimpan perubahan..." });
      setError("");
      const payload = await requestJSON(`${apiBase}/api/ai-agents${agentId ? `/${encodeURIComponent(agentId)}` : ""}`, {
        method: agentId ? "PUT" : "POST",
        body,
      });
      const id = payload.id ?? agentId ?? "";
      const savedAgentForm = { ...defaultAIAgentForm(), ...sourceForm, ...payload, id, name };
      setAIAgentForm(savedAgentForm);
      setActiveKnowledgeAIAgentId(id);
      setPlaygroundForm((current) => ({ ...current, aiAgentId: id || current.aiAgentId }));
      setWaSessionForm((current) => ({ ...current, aiAgentId: id || current.aiAgentId }));
      await loadDashboardData(auth, { preserveDrafts: true, refreshKnowledge: true });
      setAIAgentForm(savedAgentForm);
      setActiveKnowledgeAIAgentId(id);
      const isUpdatingAgent = Boolean(agentId);
      setAIAgentSaveFeedback({
        status: "saved",
        message: isUpdatingAgent ? "Pengaturan agent tersimpan" : "Agent baru dibuat",
      });
      return true;
    } catch (agentError) {
      setAIAgentSaveFeedback({ status: "error", message: "Gagal menyimpan agent." });
      setError(agentError.message);
      return false;
    } finally {
      setBusyKey("");
    }
  }

  async function deleteAIAgent(agentId) {
    const targetId = String(agentId || "").trim();
    if (!targetId) return false;
    const remainingAgents = aiAgents.filter((agent) => agent.id !== targetId);
    const nextAgent = remainingAgents[0] || null;
    try {
      setBusyKey(`ai-agent-delete-${targetId}`);
      setError("");
      await requestJSON(`${apiBase}/api/ai-agents/${encodeURIComponent(targetId)}`, { method: "DELETE" });
      setAiAgents(remainingAgents);
      if (activeKnowledgeAIAgentId === targetId) {
        setActiveKnowledgeAIAgentId(nextAgent?.id || "");
        setAIAgentForm(nextAgent ? aiAgentToForm(nextAgent) : defaultAIAgentForm());
      }
      setPlaygroundForm((current) => ({
        ...current,
        aiAgentId: current.aiAgentId === targetId ? nextAgent?.id || "" : current.aiAgentId,
      }));
      setWaSessionForm((current) => ({
        ...current,
        aiAgentId: current.aiAgentId === targetId ? nextAgent?.id || "" : current.aiAgentId,
      }));
      setAIAgentSaveFeedback({ status: "saved", message: "Agent dihapus" });
      showToast({ title: "Agent dihapus", message: "Knowledge khusus agent itu ikut dibersihkan." });
      await loadDashboardData(auth, { refreshKnowledge: true });
      if (activeKnowledgeAIAgentId === targetId) {
        setActiveKnowledgeAIAgentId(nextAgent?.id || "");
        setAIAgentForm(nextAgent ? aiAgentToForm(nextAgent) : defaultAIAgentForm());
      }
      return true;
    } catch (deleteError) {
      const linkedSessions = Array.isArray(deleteError.payload?.sessions) ? deleteError.payload.sessions : [];
      const message = deleteError.code === "AI_AGENT_LINKED_WHATSAPP"
        ? "Agent masih dipakai WhatsApp. Lepaskan session WhatsApp dulu, lalu hapus lagi."
        : deleteError.message;
      setAIAgentSaveFeedback({ status: "error", message: "Gagal menghapus agent." });
      setError(message);
      showToast({
        tone: "danger",
        title: "Agent belum bisa dihapus",
        message: linkedSessions.length ? `${linkedSessions.length} WhatsApp masih memakai agent ini.` : message,
      });
      return false;
    } finally {
      setBusyKey("");
    }
  }

  async function renameWhatsAppSession(id, currentLabel) {
    const label = await askPrompt({
      title: "Rename WhatsApp",
      label: "Nama perangkat",
      initialValue: currentLabel || "WhatsApp Session",
      placeholder: "WhatsApp Sales",
      confirmLabel: "Simpan nama",
    });
    if (label === null) return;
    const nextLabel = String(label || "").trim();
    if (!nextLabel) return;
    try {
      setBusyKey(`wa-session-${id}`);
      await requestJSON(`${apiBase}/api/whatsapp/sessions/${id}`, {
        method: "PUT",
        body: JSON.stringify({ label: nextLabel }),
      });
      await loadDashboardData();
    } catch (sessionError) {
      setError(sessionError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function setDefaultWhatsAppSession(id) {
    try {
      setBusyKey(`wa-session-default-${id}`);
      await requestJSON(`${apiBase}/api/whatsapp/sessions/${id}/default`, { method: "POST" });
      setActiveWaSessionId(id);
      await loadDashboardData();
    } catch (sessionError) {
      setError(sessionError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function deleteWhatsAppSession(id) {
    if (!(await askConfirm({
      title: "Hapus WhatsApp session?",
      message: "Kalau session masih connected, WhatsApp akan otomatis disconnect lalu session dihapus.",
      confirmLabel: "Hapus session",
      tone: "danger",
    }))) return;
    try {
      setBusyKey(`wa-session-delete-${id}`);
      await requestJSON(`${apiBase}/api/whatsapp/sessions/${id}`, { method: "DELETE" });
      await loadDashboardData();
    } catch (sessionError) {
      setError(sessionError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function generateEscalationGroupCode() {
    if (!activeWaSessionId) {
      setError("Pilih WhatsApp yang sudah terhubung dulu.");
      return;
    }
    try {
      setBusyKey("escalation-group-code");
      setError("");
      const payload = await requestJSON(`${apiBase}/api/escalation-group/bind-code?sessionId=${encodeURIComponent(activeWaSessionId)}`, {
        method: "POST",
        body: JSON.stringify({}),
      });
      setEscalationGroup(payload.escalationGroup ?? null);
      await loadDashboardData();
    } catch (groupError) {
      setError(groupError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function removeEscalationGroup() {
    if (!activeWaSessionId) {
      setError("Pilih WhatsApp yang sudah terhubung dulu.");
      return;
    }
    if (!(await askConfirm({
      title: "Hapus grup notifikasi?",
      message: "Notifikasi tim tidak akan dikirim ke grup sampai kamu hubungkan ulang.",
      confirmLabel: "Hapus grup",
      tone: "danger",
    }))) return;
    try {
      setBusyKey("escalation-group-remove");
      setError("");
      const payload = await requestJSON(`${apiBase}/api/escalation-group?sessionId=${encodeURIComponent(activeWaSessionId)}`, { method: "DELETE" });
      setEscalationGroup(payload.escalationGroup ?? null);
      await loadDashboardData();
    } catch (groupError) {
      setError(groupError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function saveGroupNotificationRule(nextRule) {
    try {
      setBusyKey("group-notification-rule-save");
      setError("");
      const payload = await requestJSON(`${apiBase}/api/group-notification-rules`, {
        method: "PUT",
        body: JSON.stringify({
          triggerKey: nextRule.triggerKey || "human_handoff",
          isEnabled: nextRule.isEnabled !== false,
          templateText: nextRule.templateText || "",
        }),
      });
      const savedRule = payload.rule;
      if (savedRule) {
        setGroupNotificationRules((current) => {
          const nextItems = current.filter((item) => item?.triggerKey !== savedRule.triggerKey);
          return [...nextItems, savedRule];
        });
      }
      showToast({
        title: "Template tersimpan",
        message: "Notifikasi grup akan memakai format baru untuk event ini.",
      });
    } catch (ruleError) {
      setError(ruleError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function sendGroupNotificationTest(input = "human_handoff") {
    const triggerKey = typeof input === "string" ? input : "human_handoff";
    if (!activeWaSessionId) {
      setError("Pilih WhatsApp yang sudah terhubung dulu.");
      return;
    }
    if (!escalationGroup?.bound) {
      setError("Hubungkan grup notifikasi dulu sebelum kirim test.");
      return;
    }
    try {
      setBusyKey("group-notification-test");
      setError("");
      await requestJSON(`${apiBase}/api/group-notification-rules/test?sessionId=${encodeURIComponent(activeWaSessionId)}`, {
        method: "POST",
        body: JSON.stringify({ triggerKey: triggerKey || "human_handoff" }),
      });
      showToast({
        title: "Test terkirim",
        message: "Cek grup WhatsApp tim untuk melihat contoh notifikasi.",
      });
    } catch (testError) {
      setError(testError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function sendAnswerCurationNotification(payload = {}) {
    if (!escalationGroup?.bound) {
      setError("Hubungkan grup notifikasi dulu sebelum kirim kurasi jawaban AI.");
      return;
    }
    const query = activeWaSessionId ? `?sessionId=${encodeURIComponent(activeWaSessionId)}` : "";
    try {
      setBusyKey("answer-curation-notification");
      setError("");
      await requestJSON(`${apiBase}/api/group-notification-rules/curation${query}`, {
        method: "POST",
        body: JSON.stringify(payload),
      });
      showToast({
        title: "Kurasi terkirim",
        message: "Ringkasan kurasi jawaban AI sudah dikirim ke grup tim.",
      });
    } catch (curationError) {
      setError(curationError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function reportDashboardIssue() {
    const detail = await askPrompt({
      title: "Laporkan masalah",
      message: "Tulis singkat masalah yang terjadi di dashboard.",
      label: "Detail masalah",
      multiline: true,
      placeholder: "Contoh: QR tidak muncul setelah klik Connect.",
      confirmLabel: "Kirim laporan",
    });
    if (detail === null) return;
    try {
      setBusyKey("support-report");
      setError("");
      await requestJSON(`${apiBase}/api/support/report`, {
        method: "POST",
        body: JSON.stringify({
          detail: detail.trim(),
          currentView: activeMeta.title || activeView,
          currentUrl: window.location.href,
        }),
      });
      showToast({ tone: "success", title: "Laporan terkirim", message: "Laporan masalah sudah dikirim lewat WhatsApp." });
    } catch (reportError) {
      setError(reportError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function saveSettings() {
    try {
      setBusyKey("settings");
      await requestJSON(`${apiBase}/api/ai-settings`, {
        method: "POST",
        body: JSON.stringify(settingsDraft),
      });
      await loadDashboardData();
    } catch (settingsError) {
      setError(settingsError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function submitFaq(event) {
    event.preventDefault();
    if (!faqForm) return;
    if (!activeKnowledgeAIAgentId) {
      setError("Pilih AI agent untuk knowledge dulu.");
      return;
    }
    try {
      setBusyKey("faq");
      const method = faqForm.id ? "PUT" : "POST";
      const url = faqForm.id ? `${apiBase}/api/knowledge/faqs/${faqForm.id}` : `${apiBase}/api/knowledge/faqs`;
      await requestJSON(url, {
        method,
        body: JSON.stringify({
          question: faqForm.question,
          answer: faqForm.answer,
          status: faqForm.status,
          aiAgentId: activeKnowledgeAIAgentId,
        }),
      });
      setFaqForm(null);
      await loadDashboardData();
    } catch (faqError) {
      setError(faqError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function deleteFaq(id) {
    if (!(await askConfirm({ title: "Hapus FAQ?", message: "FAQ ini akan hilang dari knowledge agent.", confirmLabel: "Hapus FAQ", tone: "danger" }))) return;
    try {
      setBusyKey(`faq-delete-${id}`);
      await requestJSON(`${apiBase}/api/knowledge/faqs/${id}`, { method: "DELETE" });
      if (faqForm?.id === id) setFaqForm(null);
      await loadDashboardData();
    } catch (faqError) {
      setError(faqError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function submitDocument(event) {
    event?.preventDefault?.();
    if (!documentForm) return;
    if (!activeKnowledgeAIAgentId) {
      setError("Pilih AI agent untuk knowledge dulu.");
      return;
    }
    try {
      setBusyKey("document");
      const method = documentForm.id ? "PUT" : "POST";
      const url = documentForm.id ? `${apiBase}/api/knowledge/documents/${documentForm.id}` : `${apiBase}/api/knowledge/documents`;
      await requestJSON(url, {
        method,
        body: JSON.stringify({
          title: documentForm.title,
          rawText: documentForm.rawText,
          knowledgeType: documentForm.knowledgeType,
          status: documentForm.status,
          aiAgentId: activeKnowledgeAIAgentId,
        }),
      });
      setDocumentForm(null);
      setDocumentSourceMode("upload");
      await loadDashboardData();
    } catch (documentError) {
      setError(documentError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function uploadKnowledgeDocument(event) {
    event?.preventDefault?.();
    if (!uploadFile) {
      setError("Pilih file knowledge dulu.");
      return;
    }
    if (!activeKnowledgeAIAgentId) {
      setError("Pilih AI agent untuk knowledge dulu.");
      return;
    }
    try {
      setBusyKey("document-upload");
      const formData = new FormData();
      formData.append("title", uploadTitle || uploadFile.name);
      formData.append("knowledgeType", "file");
      formData.append("aiAgentId", activeKnowledgeAIAgentId);
      formData.append("file", uploadFile);
      await requestJSON(`${apiBase}/api/knowledge/documents/upload`, {
        method: "POST",
        body: formData,
      }, { isForm: true });
      setUploadTitle("");
      setUploadFile(null);
      setDocumentForm(null);
      setDocumentSourceMode("upload");
      await loadDashboardData();
    } catch (uploadError) {
      setError(uploadError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function deleteDocument(id) {
    if (!(await askConfirm({ title: "Hapus dokumen?", message: "Dokumen ini akan hilang dari knowledge agent.", confirmLabel: "Hapus dokumen", tone: "danger" }))) return;
    try {
      setBusyKey(`document-delete-${id}`);
      await requestJSON(`${apiBase}/api/knowledge/documents/${id}`, { method: "DELETE" });
      if (documentForm?.id === id) setDocumentForm(null);
      if (documentForm?.id === id) setDocumentSourceMode("upload");
      await loadDashboardData();
    } catch (documentError) {
      setError(documentError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function reindexDocument(id) {
    try {
      setBusyKey(`document-reindex-${id}`);
      await requestJSON(`${apiBase}/api/knowledge/documents/${id}/reindex`, { method: "POST" });
      await loadDashboardData();
    } catch (documentError) {
      setError(documentError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function submitPosition(event) {
    event?.preventDefault?.();
    if (!positionForm) return;
    try {
      setBusyKey("position");
      const method = positionForm.id ? "PUT" : "POST";
      const url = positionForm.id ? `${apiBase}/api/job-positions/${positionForm.id}` : `${apiBase}/api/job-positions`;
      await requestJSON(url, {
        method,
        body: JSON.stringify({
          title: positionForm.title,
          location: positionForm.location,
          workType: positionForm.workType,
          shortDescription: positionForm.shortDescription,
          applyLink: positionForm.applyLink,
          status: positionForm.status,
          isActive: positionForm.isActive,
          aiAgentId: activeKnowledgeAIAgentId,
        }),
      });
      setPositionForm(null);
      await loadDashboardData();
    } catch (positionError) {
      setError(positionError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function deletePosition(id) {
    if (!(await askConfirm({ title: "Hapus produk/layanan?", message: "Data ini tidak akan dipakai lagi oleh agent.", confirmLabel: "Hapus", tone: "danger" }))) return;
    try {
      setBusyKey(`position-delete-${id}`);
      await requestJSON(`${apiBase}/api/job-positions/${id}`, { method: "DELETE" });
      if (positionForm?.id === id) setPositionForm(null);
      await loadDashboardData();
    } catch (positionError) {
      setError(positionError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function runPlayground(event) {
    event?.preventDefault?.();
    const messageText = playgroundForm.messageText.trim();
    if (!messageText) {
      setError("Tulis pertanyaan pelanggan dulu.");
      return;
    }
    if (!playgroundForm.aiAgentId) {
      setError("Pilih AI agent untuk playground dulu.");
      return;
    }

    const userMessage = {
      id: `playground-user-${Date.now()}`,
      role: "user",
      text: messageText,
      createdAt: new Date().toISOString(),
      meta: playgroundForm.customerName || "Pelanggan Test",
    };
    const history = buildPlaygroundHistory(playgroundMessages);
    setPlaygroundMessages((current) => [...current, userMessage]);
    setPlaygroundForm((current) => ({ ...current, messageText: "" }));

    try {
      setBusyKey("playground");
      const payload = await requestJSON(`${apiBase}/api/playground/run`, {
        method: "POST",
        body: JSON.stringify({ ...playgroundForm, messageText, history }),
      });
      const result = payload.result ?? {};
      const answerText = result.answerText ?? result.answer_text ?? "";
      const escalationReason = result.escalationReason ?? result.escalation_reason ?? "";
      const retrieval = result.retrieval_metadata ?? result.retrievalMetadata ?? null;
      const usage = result.usage_metadata ?? result.usageMetadata ?? null;
      if (payload.wallet) {
        setWallet((current) => ({ ...(current ?? {}), ...payload.wallet }));
      }
      setPlaygroundMessages((current) => [
        ...current,
        {
          id: `playground-ai-${Date.now()}`,
          role: "assistant",
          text: answerText || escalationReason || "AI tidak mengembalikan jawaban.",
          createdAt: new Date().toISOString(),
          decision: result.decision,
          confidence: result.confidenceScore ?? result.confidence_score,
          latencyMS: result.latencyMS ?? result.latency_ms,
          modelName: result.modelName ?? result.model_name,
          retrieval,
          usage,
        },
      ]);
      await loadDashboardData();
    } catch (playgroundError) {
      setError(playgroundError.message);
      setPlaygroundMessages((current) => [
        ...current,
        {
          id: `playground-error-${Date.now()}`,
          role: "assistant",
          text: playgroundError.status === 402
            ? playgroundError.message
            : `Playground gagal berjalan: ${playgroundError.message}`,
          createdAt: new Date().toISOString(),
          decision: "error",
        },
      ]);
    } finally {
      setBusyKey("");
    }
  }

  async function submitPlaygroundShortcut(event) {
    event.preventDefault();
    const payload = {
      label: playgroundShortcutForm.label.trim(),
      question: playgroundShortcutForm.question.trim(),
      expectedBehavior: playgroundShortcutForm.expectedBehavior.trim(),
      isActive: playgroundShortcutForm.isActive,
      sortOrder: Number(playgroundShortcutForm.sortOrder) || 100,
    };
    if (!payload.label || !payload.question) {
      setError("Label dan pertanyaan shortcut wajib diisi.");
      return;
    }
    try {
      setBusyKey("playground-shortcut-save");
      const url = playgroundShortcutForm.id
        ? `${apiBase}/api/playground/shortcuts/${playgroundShortcutForm.id}`
        : `${apiBase}/api/playground/shortcuts`;
      const response = await requestJSON(url, {
        method: playgroundShortcutForm.id ? "PUT" : "POST",
        body: JSON.stringify(payload),
      });
      setPlaygroundShortcuts(response.items ?? []);
      setPlaygroundShortcutForm({ id: "", label: "", question: "", expectedBehavior: "", isActive: true, sortOrder: 100 });
    } catch (shortcutError) {
      setError(shortcutError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function deletePlaygroundShortcut(id) {
    if (!(await askConfirm({ title: "Hapus shortcut?", message: "Shortcut test ini akan hilang dari Playground.", confirmLabel: "Hapus shortcut", tone: "danger" }))) return;
    try {
      setBusyKey(`playground-shortcut-delete-${id}`);
      const response = await requestJSON(`${apiBase}/api/playground/shortcuts/${id}`, { method: "DELETE" });
      setPlaygroundShortcuts(response.items ?? []);
      if (playgroundShortcutForm.id === id) {
        setPlaygroundShortcutForm({ id: "", label: "", question: "", expectedBehavior: "", isActive: true, sortOrder: 100 });
      }
    } catch (shortcutError) {
      setError(shortcutError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function runPlaygroundBatch() {
    const activeShortcuts = playgroundShortcuts.filter((item) => item.isActive);
    if (!activeShortcuts.length) {
      setError("Belum ada shortcut aktif untuk dites massal.");
      return;
    }
    if (!playgroundForm.aiAgentId) {
      setError("Pilih AI agent untuk playground dulu.");
      return;
    }
    setPlaygroundBatchResults([]);
    setBusyKey("playground-batch");
    try {
      const results = [];
      for (const shortcut of activeShortcuts) {
        const startedAt = Date.now();
        try {
          const payload = await requestJSON(`${apiBase}/api/playground/run`, {
            method: "POST",
            body: JSON.stringify({
              customerName: playgroundForm.customerName || "Pelanggan Test",
              aiAgentId: playgroundForm.aiAgentId,
              messageText: shortcut.question,
              history: [],
            }),
          });
          const result = payload.result ?? {};
          const usage = result.usage_metadata ?? result.usageMetadata ?? null;
          results.push({
            id: `${shortcut.id}-${Date.now()}`,
            shortcutId: shortcut.id,
            label: shortcut.label,
            question: shortcut.question,
            expectedBehavior: shortcut.expectedBehavior,
            decision: result.decision,
            answerText: result.answerText ?? result.answer_text ?? result.escalationReason ?? result.escalation_reason ?? "",
            confidence: result.confidenceScore ?? result.confidence_score,
            modelName: result.modelName ?? result.model_name,
            latencyMS: result.latencyMS ?? result.latency_ms ?? Date.now() - startedAt,
            usage,
            creditsUsed: usage?.creditsUsed ?? usage?.credits_used,
            status: "ok",
          });
        } catch (batchError) {
          results.push({
            id: `${shortcut.id}-${Date.now()}`,
            shortcutId: shortcut.id,
            label: shortcut.label,
            question: shortcut.question,
            expectedBehavior: shortcut.expectedBehavior,
            answerText: batchError.message,
            status: "error",
          });
        }
        setPlaygroundBatchResults([...results]);
      }
      await loadDashboardData();
    } finally {
      setBusyKey("");
    }
  }

  async function createSimulationConversation(payload) {
    try {
      setBusyKey("simulate-conversation");
      const response = await requestJSON(`${apiBase}/api/dev/simulate-conversation`, {
        method: "POST",
        body: JSON.stringify(payload),
      });
      setSimulationResults((current) => [
        {
          id: `${response.conversationId}-${Date.now()}`,
          title: payload.customerName || "Pelanggan Test",
          type: "conversation",
          detail: response,
          createdAt: new Date().toISOString(),
        },
        ...current,
      ].slice(0, 8));
      await loadDashboardData();
    } catch (simulationError) {
      setError(simulationError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function simulateInboundOnSelected() {
    if (!selectedConversation?.conversation?.id) {
      setError("Pilih percakapan dulu untuk simulate inbound.");
      return;
    }
    try {
      setBusyKey("simulate-inbound");
      const payload = await requestJSON(`${apiBase}/api/dev/simulate-inbound`, {
        method: "POST",
        body: JSON.stringify({
          conversationId: selectedConversation.conversation.id,
          text: simulatedInboundText,
          autoReply: simulateAutoReply,
          forceDecision: simulateForceDecision,
        }),
      });
      setSimulationResults((current) => [
        {
          id: `${payload.conversationId}-${Date.now()}`,
          title: selectedConversation.conversation.contactName || selectedConversation.conversation.phone,
          type: "inbound",
          detail: payload,
          createdAt: new Date().toISOString(),
        },
        ...current,
      ].slice(0, 8));
      await refreshConversation(selectedConversation.conversation.id);
      await loadDashboardData();
    } catch (simulationError) {
      setError(simulationError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function submitPurchaseRequest(event, overrideForm) {
    event.preventDefault();
    try {
      setBusyKey("purchase-request");
      const payment = await requestJSON(`${apiBase}/api/billing/purchases`, {
        method: "POST",
        body: JSON.stringify(overrideForm || purchaseForm),
      });
      if (payment?.snapToken) {
        await openMidtransPayment(payment);
      }
      if (payment?.reusedPurchase) {
        showNotice({
          title: "Pembayaran diperbarui",
          message: "Metode pembayaran diperbarui untuk transaksi paket yang belum selesai.",
          tone: "success",
        });
      }
      await loadDashboardData();
    } catch (purchaseError) {
      setError(purchaseError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function createPackage(event) {
    event.preventDefault();
    try {
      setBusyKey("package");
      const id = packageForm.id;
      const isActive = packageForm.isActive !== false && packageForm.isActive !== "false";
      const isPopular = packageForm.isPopular === true || packageForm.isPopular === "true";
      await requestJSON(`${apiBase}/api/billing/packages${id ? `/${encodeURIComponent(id)}` : ""}`, {
        method: id ? "PUT" : "POST",
        body: JSON.stringify({
          name: packageForm.name,
          description: packageForm.description || "",
          creditAmount: Number(packageForm.creditAmount),
          price: Number(packageForm.price),
          billingPeriod: packageForm.billingPeriod || "monthly",
          maxWhatsAppSessions: Number(packageForm.maxWhatsAppSessions || 1),
          maxAiAgents: Number(packageForm.maxAiAgents || 1),
          maxHumanUsers: Number(packageForm.maxHumanUsers || 1),
          isActive,
          isPopular,
        }),
      });
      setPackageForm(emptyPackageForm());
      await loadDashboardData();
      showToast({
        title: id ? "Paket diperbarui" : "Paket dibuat",
        message: packageForm.billingPeriod === "one_time" ? "Top-up credit sudah tersimpan." : "Paket layanan sudah tersimpan.",
      });
      return true;
    } catch (packageError) {
      setError(packageError.message);
      return false;
    } finally {
      setBusyKey("");
    }
  }

  function editPackage(item) {
    setPackageForm(packageFormFromItem(item));
  }

  function resetPackageForm(overrides) {
    setPackageForm(emptyPackageForm(overrides));
  }

  async function deletePackage(item) {
    if (!item?.id) return false;
    if (!(await askConfirm({
      title: item.billingPeriod === "one_time" ? "Hapus top-up?" : "Hapus paket?",
      message: `${item.name || "Paket ini"} akan dinonaktifkan dari katalog pembelian baru.`,
      confirmLabel: "Hapus",
      tone: "danger",
    }))) return false;
    try {
      setBusyKey(`package-delete-${item.id}`);
      setError("");
      await requestJSON(`${apiBase}/api/billing/packages/${encodeURIComponent(item.id)}`, { method: "DELETE" });
      setPackageForm((current) => (current.id === item.id ? emptyPackageForm() : current));
      await loadDashboardData();
      showToast({
        title: item.billingPeriod === "one_time" ? "Top-up dihapus" : "Paket dihapus",
        message: "Item tidak lagi tersedia di katalog aktif.",
      });
      return true;
    } catch (packageError) {
      setError(packageError.message);
      return false;
    } finally {
      setBusyKey("");
    }
  }

  function markNotificationsRead(ids = notifications.map((item) => item.id)) {
    const nextIds = Array.from(new Set([...notificationReadIds, ...ids]));
    setNotificationReadIds(nextIds);
    if (notificationStorageKey) {
      window.localStorage.setItem(notificationStorageKey, JSON.stringify(nextIds));
    }
  }

  async function saveAccountProfile(event) {
    event.preventDefault();
    try {
      setBusyKey("account-profile");
      setError("");
      const payload = await requestJSON(`${apiBase}/api/account/profile`, {
        method: "PUT",
        body: JSON.stringify({ name: accountProfileForm.name }),
      });
      const nextAuth = { ...auth, user: payload.user };
      setAuth(nextAuth);
      window.localStorage.setItem(authStorageKey, JSON.stringify(nextAuth));
      setAccountProfileForm({ name: payload.user?.name ?? "" });
      await loadDashboardData(nextAuth);
    } catch (accountError) {
      setError(accountError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function changeAccountPassword(event) {
    event.preventDefault();
    if (accountPasswordForm.newPassword !== accountPasswordForm.confirmPassword) {
      setError("Konfirmasi password baru tidak sama.");
      return;
    }
    try {
      setBusyKey("account-password");
      setError("");
      await requestJSON(`${apiBase}/api/account/password`, {
        method: "POST",
        body: JSON.stringify({
          currentPassword: accountPasswordForm.currentPassword,
          newPassword: accountPasswordForm.newPassword,
        }),
      });
      setAccountPasswordForm({ currentPassword: "", newPassword: "", confirmPassword: "" });
    } catch (accountError) {
      setError(accountError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function savePricing(event) {
    event.preventDefault();
    try {
      setBusyKey("pricing");
      const payload = isOwner
        ? {
            creditUnitIdr: Number(pricingForm.creditUnitIdr),
            usdToIdrRate: Number(pricingForm.usdToIdrRate),
            chatModelName: pricingForm.chatModelName,
            chatInputPricePer1m: Number(pricingForm.chatInputPricePer1m),
            chatOutputPricePer1m: Number(pricingForm.chatOutputPricePer1m),
            embeddingModelName: pricingForm.embeddingModelName,
            embeddingPricePer1m: Number(pricingForm.embeddingPricePer1m),
            annualDiscountPercent: Number(pricingForm.annualDiscountPercent),
            monthlyCreditLimit: Number(pricingForm.monthlyCreditLimit),
            modelAliases: pricingForm.modelAliases ?? {},
            availableModelIds: pricingForm.availableModelIds ?? defaultAvailableModelIds,
          }
        : {
            chatModelName: pricingForm.chatModelName,
          };
      await requestJSON(`${apiBase}/api/billing/pricing`, {
        method: "POST",
        body: JSON.stringify(payload),
      });
      pricingFormDirtyRef.current = false;
      await loadDashboardData();
    } catch (pricingError) {
      setError(pricingError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function submitCreditAdjustment(event) {
    event.preventDefault();
    try {
      setBusyKey("credit-adjustment");
      await requestJSON(`${apiBase}/api/billing/adjustments`, {
        method: "POST",
        body: JSON.stringify({
          adjustmentType: adjustmentForm.adjustmentType,
          amount: Number(adjustmentForm.amount),
          notes: adjustmentForm.notes,
        }),
      });
      setAdjustmentForm({ adjustmentType: "add_additional", amount: "1000", notes: "" });
      await loadDashboardData();
    } catch (adjustmentError) {
      setError(adjustmentError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function submitAgent(event) {
    event?.preventDefault?.();
    if (!agentForm) return;
    try {
      setBusyKey("agent");
      const payload = {
        name: agentForm.name,
        phone: agentForm.phone,
        role: agentForm.role,
        isActive: agentForm.isActive,
      };
      if (agentForm.id) {
        await requestJSON(`${apiBase}/api/agents/${agentForm.id}`, {
          method: "PUT",
          body: JSON.stringify(payload),
        });
      } else {
        await requestJSON(`${apiBase}/api/agents`, {
          method: "POST",
          body: JSON.stringify({
            ...payload,
            username: agentForm.username,
            password: agentForm.password,
          }),
        });
      }
      setAgentForm(null);
      await loadDashboardData();
    } catch (agentError) {
      setError(agentError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function resetAgentPassword(id, username) {
    const password = await askPrompt({
      title: "Reset password",
      message: `Password baru untuk ${username}.`,
      label: "Password baru",
      inputType: "password",
      confirmLabel: "Reset password",
    });
    if (!password) return;
    try {
      setBusyKey(`agent-reset-${id}`);
      await requestJSON(`${apiBase}/api/agents/${id}/reset-password`, {
        method: "POST",
        body: JSON.stringify({ password }),
      });
      await loadDashboardData();
    } catch (agentError) {
      setError(agentError.message);
    } finally {
      setBusyKey("");
    }
  }

  async function inspectUsageLog(id) {
    if (!id) return;
    try {
      setBusyKey(`usage-${id}`);
      const payload = await requestJSON(`${apiBase}/api/billing/usage-logs/${id}`);
      setUsageLogDetail(payload.log ?? null);
    } catch (usageError) {
      setError(usageError.message);
    } finally {
      setBusyKey("");
    }
  }

  function renderActiveView() {
    if (isMobileOpsMode && !isMobileOperationsView(role, activeView)) {
      return (
        <MobileDesktopGate
          activeView={activeView}
          activeMeta={activeMeta}
          navItems={navItems}
          onNavigate={navigateToView}
        />
      );
    }

    if (activeView === "overview") {
      return (
        <OwnerOverviewView
          analytics={analytics}
          summary={summary}
          wallet={ownerWallet}
          purchases={purchases}
          health={health}
          creditAdjustments={creditAdjustments}
        />
      );
    }

    if (activeView === "dashboard") {
      return (
        <PerformanceDashboardView
          role={role}
          analytics={analytics}
          analyticsRange={analyticsRange}
          setAnalyticsRange={setAnalyticsRange}
          summary={summary}
          aiRuns={aiRuns}
          inbox={inbox}
          contacts={contacts}
          agents={agents}
        />
      );
    }

    if (activeView === "operations") {
      return (
        <InboxView
          auth={auth}
          inbox={inbox}
          displayInbox={displayInbox}
          inboxTab={inboxTab}
          setInboxTab={setInboxTab}
          inboxSearch={inboxSearch}
          setInboxSearch={setInboxSearch}
          waSessions={waSessions}
          aiAgents={aiAgents}
          activeWaSessionId={activeInboxWaSessionId}
          setActiveWaSessionId={setActiveInboxWaSessionId}
          activeAIAgentId={activeInboxAIAgentId}
          setActiveAIAgentId={setActiveInboxAIAgentId}
          selectedConversation={selectedConversation}
          fetchConversation={fetchConversation}
          runConversationAction={runConversationAction}
          teamMembers={teamMembers}
          saveConversationWorkflow={saveConversationWorkflow}
          createDealFromConversation={createDealFromConversation}
          openDealFromConversation={openDealFromConversation}
          openContactFromConversation={openContactFromConversation}
          manualMessage={manualMessage}
          setManualMessage={setManualMessage}
          manualMedia={manualMedia}
          setManualMedia={setManualMedia}
          takeoverNote={takeoverNote}
          canOpenChatOps={canOpenChatOps}
          canForceTakeover={isAdmin || isSuperAdmin}
          canUseProspects={canUseProspects}
          isOwner={isOwner}
          isOperator={isOperator}
          navigateToWhatsApp={() => navigateToView("whatsapp")}
          aiTyping={realtimeSnapshot?.aiTyping}
          busyKey={busyKey}
          onNotify={showToast}
        />
      );
    }

    if (activeView === "contacts") {
      return (
        <ContactsView
          contacts={contacts}
          teamMembers={teamMembers}
          apiBase={apiBase}
          requestJSON={requestJSON}
          focusContactId={focusContactId}
          onFocusContactHandled={() => setFocusContactId("")}
          onRefresh={() => loadDashboardData(auth, { preserveDrafts: true })}
          canUseProspects={canUseProspects}
        />
      );
    }

    if (activeView === "deals") {
      return (
        <DealsView
          contacts={contacts}
          teamMembers={teamMembers}
          apiBase={apiBase}
          requestJSON={requestJSON}
          focusDealId={focusDealId}
          onFocusDealHandled={() => setFocusDealId("")}
          toolState={businessToolByKey.get("prospects")}
        />
      );
    }

    if (activeView === "tickets") {
      return (
        <TicketsView
          contacts={contacts}
          inbox={inbox}
          teamMembers={teamMembers}
          apiBase={apiBase}
          requestJSON={requestJSON}
          toolState={businessToolByKey.get("tickets")}
          onRefreshDashboard={() => loadDashboardData(auth, { preserveDrafts: true })}
        />
      );
    }

    if (activeView === "knowledge") {
      return (
        <KnowledgeView
          knowledgeTab={knowledgeTab}
          setKnowledgeTab={setKnowledgeTab}
          faqs={faqs}
          documents={documents}
          positions={positions}
          businessTools={businessTools}
          aiAgents={aiAgents}
          activeAIAgentId={activeKnowledgeAIAgentId}
          setActiveAIAgentId={setActiveKnowledgeAIAgentId}
          faqForm={faqForm}
          setFaqForm={setFaqForm}
          documentForm={documentForm}
          setDocumentForm={setDocumentForm}
          documentSourceMode={documentSourceMode}
          setDocumentSourceMode={setDocumentSourceMode}
          fetchPositions={loadDashboardData}
          positionForm={positionForm}
          setPositionForm={setPositionForm}
          uploadTitle={uploadTitle}
          setUploadTitle={setUploadTitle}
          setUploadFile={setUploadFile}
          submitFaq={submitFaq}
          submitDocument={submitDocument}
          submitPosition={submitPosition}
          uploadKnowledgeDocument={uploadKnowledgeDocument}
          deleteFaq={deleteFaq}
          deleteDocument={deleteDocument}
          reindexDocument={reindexDocument}
          deletePosition={deletePosition}
          refreshVectorIndex={loadDashboardData}
          canManageKnowledge={canManageKnowledge}
          busyKey={busyKey}
        />
      );
    }

    if (activeView === "businessTools") {
      return (
        <BusinessToolsView
          businessTools={businessTools}
          loadState={businessToolsLoadState}
          canManage={isOwner || isAdmin || isSuperAdmin}
          busyKey={busyKey}
          onInstallTool={installBusinessTool}
          onRemoveTool={removeBusinessTool}
          onUpdateToolAI={updateBusinessToolAI}
          onRefresh={() => loadDashboardData(auth, { preserveDrafts: true })}
          onOpenTool={(viewId) => viewId && navigateToView(viewId)}
          groupNotificationRules={groupNotificationRules}
          groupNotificationVariables={groupNotificationVariables}
          canManageGroupNotifications={canManageGroupNotifications}
          isNotificationGroupBound={Boolean(escalationGroup?.bound)}
          notificationGroupName={escalationGroup?.group?.groupName}
          saveGroupNotificationRule={saveGroupNotificationRule}
          sendGroupNotificationTest={sendGroupNotificationTest}
          onOpenNotificationGroup={() => navigateToView("whatsapp")}
        />
      );
    }

    if (activeView === "commerceProducts") {
      return (
        <CommerceProductsView
          products={commerceProducts}
          productForm={commerceProductForm}
          setProductForm={setCommerceProductForm}
          onSubmitProduct={submitCommerceProduct}
          onEditProduct={editCommerceProduct}
          onDeleteProduct={deleteCommerceProduct}
          onResetProduct={() => setCommerceProductForm(emptyCommerceProductForm())}
          onOpenOrders={() => navigateToView("commerceOrders")}
          canManage={isOwner || isAdmin || isSuperAdmin}
          busyKey={busyKey}
          toolState={businessToolByKey.get("commerce")}
        />
      );
    }

    if (activeView === "commerceOrders") {
      return (
        <CommerceOrdersView
          products={commerceProducts}
          orderDrafts={commerceOrderDrafts}
          orderPipelines={commerceOrderPipelines}
          orderStages={commerceOrderStages}
          orderSettings={commerceOrderSettings}
          orderDraftForm={commerceOrderDraftForm}
          setOrderDraftForm={setCommerceOrderDraftForm}
          onSubmitOrderDraft={submitCommerceOrderDraft}
          onUpdateOrderStatus={updateCommerceOrderDraftStatus}
          onUpdateOrderDraft={updateCommerceOrderDraft}
          onSaveOrderStage={saveCommerceOrderStage}
          onDeleteOrderStage={deleteCommerceOrderStage}
          onUpdateOrderSettings={updateCommerceOrderSettings}
          onOpenProducts={() => navigateToView("commerceProducts")}
          canManagePipeline={isOwner || isAdmin || isSuperAdmin}
          busyKey={busyKey}
          toolState={businessToolByKey.get("commerce")}
        />
      );
    }

    if (activeView === "booking") {
      return (
        <BookingView
          bookingServices={bookingServices}
          bookingAppointments={bookingAppointments}
          bookingServiceForm={bookingServiceForm}
          setBookingServiceForm={setBookingServiceForm}
          bookingAppointmentForm={bookingAppointmentForm}
          setBookingAppointmentForm={setBookingAppointmentForm}
          onSubmitBookingService={submitBookingService}
          onEditBookingService={editBookingService}
          onResetBookingService={() => setBookingServiceForm(emptyBookingServiceForm())}
          onSubmitBookingAppointment={submitBookingAppointment}
          onUpdateBookingAppointmentStatus={updateBookingAppointmentStatus}
          canManageServices={isOwner || isAdmin || isSuperAdmin}
          busyKey={busyKey}
          toolState={businessToolByKey.get("booking")}
        />
      );
    }

    if (activeView === "agents") {
      return (
        <AgentsView
          aiAgents={aiAgents}
          sessions={waSessions}
          aiAgentForm={aiAgentForm}
          setAIAgentForm={setAIAgentForm}
          saveAIAgent={createAIAgent}
          deleteAIAgent={deleteAIAgent}
          activeAIAgentId={activeKnowledgeAIAgentId}
          setActiveAIAgentId={setActiveKnowledgeAIAgentId}
          agentTab={agentTab}
          setAgentTab={setAgentTab}
          canManageKnowledge={canManageKnowledge}
          navigateToWhatsApp={() => navigateToView("whatsapp")}
          navigateToPlayground={() => navigateToView("playground")}
          busyKey={busyKey}
          aiAgentSaveFeedback={aiAgentSaveFeedback}
          humanHandoffNotificationRule={humanHandoffNotificationRule}
          groupNotificationVariables={groupNotificationVariables}
          saveGroupNotificationRule={saveGroupNotificationRule}
          sendGroupNotificationTest={sendGroupNotificationTest}
          canManageGroupNotifications={canManageGroupNotifications}
          isNotificationGroupBound={Boolean(escalationGroup?.bound)}
          notificationGroupName={escalationGroup?.group?.groupName}
          aiModels={aiModels}
          configureBusinessTools={configureBusinessToolsForAgent}
          knowledgeProps={{
            knowledgeTab,
            setKnowledgeTab,
            faqs,
            documents,
            positions,
            businessTools,
            businessToolDataCounts: {
              commerce: commerceProducts.filter((item) => item.status === "active").length,
              booking: bookingServices.filter((item) => item.status === "active").length,
            },
            aiAgents,
            faqForm,
            setFaqForm,
            documentForm,
            setDocumentForm,
            documentSourceMode,
            setDocumentSourceMode,
            fetchPositions: loadDashboardData,
            positionForm,
            setPositionForm,
            uploadTitle,
            setUploadTitle,
            setUploadFile,
            submitFaq,
            submitDocument,
            submitPosition,
            uploadKnowledgeDocument,
            deleteFaq,
            deleteDocument,
            reindexDocument,
            deletePosition,
            refreshVectorIndex: loadDashboardData,
            busyKey,
          }}
        />
      );
    }

    if (activeView === "settings") {
      return (
        <AISettingsView
          settingsDraft={settingsDraft}
          setSettingsDraft={setSettingsDraft}
          settings={settings}
          aiRuns={aiRuns}
          canManageSettings={canManageSettings}
          saveSettings={saveSettings}
          busyKey={busyKey}
        />
      );
    }

    if (activeView === "analytics") {
      return (
        <AnalyticsView
          role={role}
          analytics={analytics}
          analyticsRange={analyticsRange}
          setAnalyticsRange={setAnalyticsRange}
          aiRuns={aiRuns}
          inbox={inbox}
          contacts={contacts}
          agents={agents}
        />
      );
    }

    if (activeView === "playground") {
      return (
        <PlaygroundView
          playgroundForm={playgroundForm}
          setPlaygroundForm={setPlaygroundForm}
          playgroundMessages={playgroundMessages}
          aiAgents={aiAgents}
          setPlaygroundMessages={setPlaygroundMessages}
          playgroundShortcuts={playgroundShortcuts}
          playgroundShortcutForm={playgroundShortcutForm}
          setPlaygroundShortcutForm={setPlaygroundShortcutForm}
          playgroundBatchResults={playgroundBatchResults}
          businessTools={businessTools}
          wallet={wallet}
          sendPlaygroundMessage={runPlayground}
          clearPlaygroundChat={() => setPlaygroundMessages([createPlaygroundWelcome()])}
          runPlaygroundBatch={runPlaygroundBatch}
          sendAnswerCurationNotification={sendAnswerCurationNotification}
          answerCurationNotificationRule={groupNotificationRules.find((rule) => rule?.triggerKey === "ai_answer_curation")}
          isNotificationGroupBound={Boolean(escalationGroup?.bound)}
          notificationGroupName={escalationGroup?.group?.groupName}
          savePlaygroundShortcut={submitPlaygroundShortcut}
          deletePlaygroundShortcut={deletePlaygroundShortcut}
          busyKey={busyKey}
        />
      );
    }

    if (activeView === "wallet" || activeView === "upgrade") {
      return (
        <WalletView
          upgradeMode={activeView === "upgrade"}
          role={role}
          wallet={wallet}
          billingPlan={billingPlan}
          packages={packages}
          pricingForm={pricingForm}
          setPricingForm={updatePricingForm}
          savePricing={savePricing}
          packageForm={packageForm}
          setPackageForm={setPackageForm}
          createPackage={createPackage}
          editPackage={editPackage}
          deletePackage={deletePackage}
          resetPackageForm={resetPackageForm}
          canManagePackages={canManagePackages}
          purchases={purchases}
          purchaseForm={purchaseForm}
          setPurchaseForm={setPurchaseForm}
          submitPurchaseRequest={submitPurchaseRequest}
          cancelPurchase={cancelPurchase}
          canRequestPurchases={canRequestPurchases}
          busyKey={busyKey}
        />
      );
    }

    if (activeView === "account") {
      return (
        <AccountSettingsView
          auth={auth}
          accountProfileForm={accountProfileForm}
          setAccountProfileForm={setAccountProfileForm}
          accountPasswordForm={accountPasswordForm}
          setAccountPasswordForm={setAccountPasswordForm}
          saveAccountProfile={saveAccountProfile}
          changeAccountPassword={changeAccountPassword}
          busyKey={busyKey}
        />
      );
    }

    if (activeView === "health") {
      return <HealthView health={health} waStatus={waStatus} realtimeSnapshot={realtimeSnapshot} />;
    }

    if (activeView === "whatsapp") {
      return (
        <WhatsAppConnectionView
          waStatus={waStatus}
          sessions={waSessions}
          aiAgents={aiAgents}
          businessTools={businessTools}
          navigateToAgents={() => navigateToView("agents")}
          activeSessionId={activeWaSessionId}
          setActiveSessionId={setActiveWaSessionId}
          sessionForm={waSessionForm}
          setSessionForm={setWaSessionForm}
          createSession={createWhatsAppSession}
          officialWhatsAppEnabled={metaCloudConfig.enabled === true}
          connectOfficialWhatsApp={reconnectOfficialWhatsApp}
          canManageOfficialTemplates={role === "owner" || role === "super_admin" || role === "admin"}
          metaTemplates={metaTemplates}
          metaTemplateForm={metaTemplateForm}
          setMetaTemplateForm={setMetaTemplateForm}
          metaTemplateSendForm={metaTemplateSendForm}
          setMetaTemplateSendForm={setMetaTemplateSendForm}
          loadMetaTemplates={loadMetaTemplates}
          createMetaTemplate={createMetaTemplate}
          sendMetaTemplate={sendMetaTemplate}
          pendingSessionId={pendingWaSessionId}
          renameSession={renameWhatsAppSession}
          setDefaultSession={setDefaultWhatsAppSession}
          deleteSession={deleteWhatsAppSession}
          qrImageSrc={qrImageSrc}
          qrData={qrData}
          escalationGroup={escalationGroup}
          groupNotificationRules={groupNotificationRules}
          canManageEscalationGroup={canManageGroupNotifications}
          generateEscalationGroupCode={generateEscalationGroupCode}
          removeEscalationGroup={removeEscalationGroup}
          sendGroupNotificationTest={sendGroupNotificationTest}
          busyKey={busyKey}
          requestQr={requestQr}
          disconnectWhatsApp={disconnectWhatsApp}
          refreshAll={() => loadDashboardData()}
          cancelPendingSession={cancelPendingWhatsAppSession}
          billingPlan={billingPlan}
          navigateToUpgrade={() => navigateToView("upgrade")}
          navigateToBusinessTools={() => navigateToView("businessTools")}
        />
      );
    }

    if (activeView === "team") {
      return (
        <TeamManagementView
          agents={agents}
          members={teamMembers}
          invites={teamInvites}
          inviteForm={inviteForm}
          setInviteForm={setInviteForm}
          createInvite={createTeamInvite}
          revokeInvite={revokeTeamInvite}
          updateMember={updateTeamMember}
          removeMember={removeTeamMember}
          agentForm={agentForm}
          setAgentForm={setAgentForm}
          submitAgent={submitAgent}
          resetAgentPassword={resetAgentPassword}
          canManageAgentAccounts={isSuperAdmin}
          busyKey={busyKey}
        />
      );
    }

    if (activeView === "usage") {
      return (
        <UsageLogsView
          role={role}
          usageLogs={usageLogs}
          analytics={analytics}
          billingAnalytics={billingAnalytics}
          usageLogDetail={usageLogDetail}
          inspectUsageLog={inspectUsageLog}
          busyKey={busyKey}
        />
      );
    }

    return (
      <PricingSettingsView
        pricing={pricing}
        aiModels={aiModels}
        pricingForm={pricingForm}
        setPricingForm={updatePricingForm}
        savePricing={savePricing}
        role={role}
        adjustmentForm={adjustmentForm}
        setAdjustmentForm={setAdjustmentForm}
        submitCreditAdjustment={submitCreditAdjustment}
        creditAdjustments={creditAdjustments}
        busyKey={busyKey}
      />
    );
  }

  if (!authReady && pathname !== "/login") {
    return (
      <div className="auth-loading-screen" suppressHydrationWarning>
        <div className="auth-loading-mark" suppressHydrationWarning><img src="/brand/oneflow-icon.png" alt="" aria-hidden="true" /></div>
        <span>Memuat dashboard...</span>
      </div>
    );
  }

  if (!auth) {
    return (
      <LoginScreen
        error={error}
        loading={loading}
        theme={dashboardTheme}
        onLogin={login}
        onRegister={registerAccount}
      />
    );
  }

  return (
    <div className={`dashboard-shell app-shell theme-${dashboardTheme}`} data-theme={dashboardTheme}>
      <Sidebar
        role={role}
        navItems={navItems}
        activeView={activeView}
        onNavigate={navigateToView}
        summary={summary}
        inbox={inbox}
        purchases={purchases}
        health={health}
        wallet={ownerWallet || wallet}
        waConnected={canOpenChatOps}
        busyKey={busyKey}
        onReportIssue={reportDashboardIssue}
      />

      <div className="dashboard-main main-content">
        <Topbar
          title={activeMeta.title}
          auth={auth}
          theme={dashboardTheme}
          waStatus={displayWaStatus}
          notifications={notifications}
          unreadNotificationCount={unreadNotificationCount}
          notificationReadIds={notificationReadIds}
          onMarkNotificationsRead={markNotificationsRead}
          onNavigate={navigateToView}
          onToggleTheme={toggleDashboardTheme}
          logout={logout}
        />

        <div className={`dashboard-content page-body ${activeView === "operations" ? "inbox-page-body" : ""}`}>
          {error ? <Notice tone="danger" title="Ada aksi yang gagal" text={error} /> : null}
          {loadWarnings.length ? (
            <Notice tone="warn" title="Sebagian data belum sempurna" text={loadWarnings.join(" ")} />
          ) : null}
          {renderActiveView()}
        </div>
      </div>

      <FirstRunOnboardingModal
        open={firstRunOnboardingOpen}
        slides={firstRunOnboardingSlides}
        index={firstRunOnboardingIndex}
        onBack={backFirstRunOnboarding}
        onNext={nextFirstRunOnboarding}
        onSkip={completeFirstRunOnboarding}
        onStart={startFirstRunSetup}
      />

      {toast ? (
        <div className={`app-toast ${toast.tone || "success"}`} role="status">
          <div className="app-toast-icon">{toast.tone === "warn" ? Icons.alert : Icons.check}</div>
          <div>
            <strong>{toast.title}</strong>
            {toast.message ? <p>{toast.message}</p> : null}
          </div>
          <button className="icon-btn" type="button" onClick={() => setToast(null)} aria-label="Tutup notifikasi">
            {Icons.close}
          </button>
        </div>
      ) : null}

      {dialog ? (
        <div className="modal-backdrop app-dialog-backdrop">
          <div className={`modal-card app-dialog-card ${dialog.tone === "danger" ? "danger" : ""} ${dialog.tone === "success" ? "success" : ""}`}>
            <div className="modal-icon">{dialog.tone === "danger" ? Icons.alert : Icons.check}</div>
            <h2>{dialog.title}</h2>
            {dialog.message ? <p>{dialog.message}</p> : null}
            {dialog.type === "prompt" ? (
              <label className="form-group app-dialog-field">
                {dialog.label ? <span className="form-label">{dialog.label}</span> : null}
                {dialog.multiline ? (
                  <textarea
                    className="form-textarea"
                    rows={4}
                    value={dialogInput}
                    placeholder={dialog.placeholder || ""}
                    onChange={(event) => setDialogInput(event.target.value)}
                    autoFocus
                  />
                ) : (
                  <input
                    className="form-input"
                    type={dialog.inputType || "text"}
                    value={dialogInput}
                    placeholder={dialog.placeholder || ""}
                    onChange={(event) => setDialogInput(event.target.value)}
                    onKeyDown={(event) => {
                      if (event.key === "Enter") {
                        event.preventDefault();
                        resolveDialog(dialogInput);
                      }
                    }}
                    autoFocus
                  />
                )}
              </label>
            ) : null}
            <div className="modal-actions">
              {dialog.type !== "notice" ? (
                <button className="btn btn-secondary" type="button" onClick={() => resolveDialog(dialog.type === "prompt" ? null : false)}>
                  {dialog.cancelLabel || "Batal"}
                </button>
              ) : null}
              <button className={`btn ${dialog.tone === "danger" ? "btn-danger" : "btn-primary"}`} type="button" onClick={() => resolveDialog(dialog.type === "prompt" ? dialogInput : true)}>
                {dialog.confirmLabel || "Ya, lanjut"}
              </button>
            </div>
          </div>
        </div>
      ) : null}

    </div>
  );
}
