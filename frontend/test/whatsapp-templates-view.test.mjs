import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const viewSource = readFileSync(new URL("../components/dashboard/views/WhatsAppTemplatesView.jsx", import.meta.url), "utf8");
const appSource = readFileSync(new URL("../components/dashboard/DashboardApp.jsx", import.meta.url), "utf8");

test("WhatsApp templates are presented as a synced status list without wizard steps", () => {
  assert.doesNotMatch(viewSource, /Langkah\s+[123]/);
  assert.match(viewSource, /Siap dipakai/);
  assert.match(viewSource, /Sedang ditinjau/);
  assert.match(viewSource, /Ditolak/);
  assert.match(viewSource, /Pakai template/);
  assert.match(viewSource, /Tambah template/);
  assert.match(viewSource, /List template/);
});

test("WhatsApp templates sync automatically when the page opens", () => {
  assert.match(appSource, /activeView !== "whatsappTemplates"/);
  assert.match(appSource, /loadMetaTemplates\(activeWaSessionId\)/);
});

test("a successful template submission clears the draft and reloads Meta data", () => {
  assert.match(appSource, /setMetaTemplateForm\(\{[\s\S]*?name:\s*""[\s\S]*?body:\s*""/);
  assert.match(appSource, /await loadMetaTemplates\(activeWaSessionId\)/);
});
