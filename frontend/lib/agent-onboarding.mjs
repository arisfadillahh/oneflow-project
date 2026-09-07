export const AGENT_SETUP_STEP_ORDER = [
  "business",
  "template",
  "business-name",
  "agent-name",
  "details",
  "handoff",
  "tools",
  "review",
];

const templateSignals = [
  {
    key: "clinic-admin",
    signals: ["klinik", "dokter", "pasien", "kesehatan", "dental", "gigi", "medis"],
  },
  {
    key: "education",
    signals: ["kursus", "sekolah", "kelas", "les", "belajar", "bootcamp", "pelatihan", "murid", "siswa"],
  },
  {
    key: "property",
    signals: ["properti", "rumah", "apartemen", "tanah", "kost", "kos", "unit", "sewa", "kpr"],
  },
  {
    key: "travel-hospitality",
    signals: ["travel", "wisata", "hotel", "villa", "tour", "trip", "penginapan", "reservasi kamar"],
  },
  {
    key: "b2b-sales",
    signals: ["b2b", "saas", "agency", "agensi", "software", "vendor", "proposal", "demo", "calon klien"],
  },
  {
    key: "booking-service",
    signals: ["salon", "bengkel", "studio", "servis", "appointment", "booking", "reservasi", "jadwal", "konsultasi"],
  },
  {
    key: "ecommerce",
    signals: ["toko", "jual", "retail", "produk", "online shop", "ecommerce", "e-commerce", "skincare", "fashion", "stok", "pesanan"],
  },
];

function normalizeDescription(value) {
  return String(value || "")
    .toLowerCase()
    .normalize("NFKD")
    .replace(/[\u0300-\u036f]/g, "")
    .replace(/[^a-z0-9-]+/g, " ")
    .trim();
}

function includesSignal(description, signal) {
  return ` ${description} `.includes(` ${signal} `);
}

export function inferAgentTemplateKey(value) {
  const description = normalizeDescription(value);
  if (!description) return "general-cs";

  let best = { key: "general-cs", score: 0 };
  for (const template of templateSignals) {
    const score = template.signals.reduce(
      (total, signal) => total + (includesSignal(description, signal) ? 1 : 0),
      0
    );
    if (score > best.score) best = { key: template.key, score };
  }
  return best.score > 0 ? best.key : "general-cs";
}

export function getAgentSetupProgress(step) {
  const index = AGENT_SETUP_STEP_ORDER.indexOf(step);
  return {
    current: index >= 0 ? index + 1 : 1,
    total: AGENT_SETUP_STEP_ORDER.length,
  };
}

export function getNextAgentSetupStep(step) {
  const index = AGENT_SETUP_STEP_ORDER.indexOf(step);
  if (index < 0) return AGENT_SETUP_STEP_ORDER[0];
  return AGENT_SETUP_STEP_ORDER[Math.min(index + 1, AGENT_SETUP_STEP_ORDER.length - 1)];
}

export function getPreviousAgentSetupStep(step) {
  const index = AGENT_SETUP_STEP_ORDER.indexOf(step);
  if (index <= 0) return AGENT_SETUP_STEP_ORDER[0];
  return AGENT_SETUP_STEP_ORDER[index - 1];
}
