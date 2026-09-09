import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

const appSource = await readFile(new URL("../components/dashboard/DashboardApp.jsx", import.meta.url), "utf8");
const channelSource = await readFile(new URL("../components/dashboard/views/WhatsAppConnectionView.jsx", import.meta.url), "utf8");

test("Instagram connector is loaded and managed through app-backend", () => {
  assert.match(appSource, /\/api\/instagram\/sessions/);
  assert.match(appSource, /\/api\/instagram\/oauth\/start/);
  assert.match(appSource, /\/api\/instagram\/sessions\/\$\{encodeURIComponent\(sessionId\)\}\/validate/);
  assert.match(appSource, /disconnectInstagram/);
});

test("Instagram-only accounts can use the shared inbox without faking WhatsApp status", () => {
  assert.match(appSource, /const canOpenWhatsAppOps = planAllowsWhatsApp && hasConnectedWhatsAppSession/);
  assert.match(appSource, /const canOpenChatOps = canOpenWhatsAppOps \|\| hasConnectedInstagramSession/);
  assert.match(appSource, /canOpenWhatsAppOps\s*\? \{ status: "connected"/);
});

test("Channels gives Instagram a real connect flow instead of coming-soon copy", () => {
  assert.match(channelSource, /Hubungkan Instagram/);
  assert.match(channelSource, /Lanjutkan dengan Instagram/);
  assert.match(channelSource, /Tambah akun Instagram/);
  assert.match(channelSource, /Login dan tambahkan akun/);
  assert.match(channelSource, /!instagramSessions\.length \|\| instagramConnectOpen/);
  assert.match(channelSource, /Cek koneksi/);
  assert.match(channelSource, /instagramSessions\.length \? "Aktif · Kelola" : "Belum aktif · Hubungkan"/);
  assert.doesNotMatch(channelSource, /<strong>Instagram<\/strong>[\s\S]{0,250}Segera hadir/);
});

test("Channels reveals one novice-friendly setup flow at a time", () => {
  assert.match(channelSource, /setupChannel === "instagram"/);
  assert.match(channelSource, /setupChannel === "whatsapp"/);
  assert.match(channelSource, /Koneksi bisnismu/);
  assert.match(channelSource, /Status "Aktif" berarti pesan pelanggan sudah bisa masuk/);
  assert.doesNotMatch(channelSource, /className="wa-flow-strip"/);
});
