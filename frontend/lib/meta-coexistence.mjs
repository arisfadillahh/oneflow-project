const isMetaID = (value) => typeof value === "string" && /^[0-9]+$/.test(value);

export function validateCoexistenceConfig(config) {
  if (!config?.enabled) throw new Error("Koneksi WhatsApp resmi belum diaktifkan di server.");
  if (config.onboardingMode !== "coexistence") {
    throw new Error("Koneksi coexistence belum tersedia. Jangan hapus akun atau reset WhatsApp Business Anda.");
  }
  if (!isMetaID(config.appId) || !isMetaID(config.configurationId) || !/^v[0-9]+\.[0-9]+$/.test(config.graphVersion || "")) {
    throw new Error("Konfigurasi koneksi WhatsApp resmi belum lengkap. Hubungi admin Oneflow.");
  }
}

export function parseCoexistenceEvent(event) {
  if (!["https://www.facebook.com", "https://web.facebook.com"].includes(event.origin)) return null;
  let payload;
  try {
    payload = typeof event.data === "string" ? JSON.parse(event.data) : event.data;
  } catch {
    return null;
  }
  if (payload?.type !== "WA_EMBEDDED_SIGNUP") return null;
  if (payload.event === "CANCEL") throw new Error("Koneksi WhatsApp resmi dibatalkan.");
  if (payload.event === "ERROR") {
    throw new Error("Meta belum dapat menyelesaikan koneksi coexistence. Periksa kelayakan nomor dan izin akun, lalu coba lagi. Jangan hapus akun atau reset WhatsApp Business Anda.");
  }
  if (["FINISH", "FINISH_ONLY_WABA"].includes(payload.event)) {
    throw new Error("Hasil koneksi bukan WhatsApp Business coexistence. Koneksi tidak disimpan. Hubungi admin tanpa menghapus akun WhatsApp Business Anda.");
  }
  if (payload.event !== "FINISH_WHATSAPP_BUSINESS_APP_ONBOARDING") return null;
  const wabaId = payload.data?.waba_id;
  const phoneNumberId = payload.data?.phone_number_id;
  if (!isMetaID(wabaId) || !isMetaID(phoneNumberId)) {
    throw new Error("Meta belum memberikan detail nomor yang lengkap. Coba hubungkan kembali tanpa mereset WhatsApp Business Anda.");
  }
  return { event: payload.event, wabaId, phoneNumberId };
}
