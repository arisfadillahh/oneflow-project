import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { parseCoexistenceEvent, validateCoexistenceConfig } from "../lib/meta-coexistence.mjs";

const config = { enabled: true, onboardingMode: "coexistence", appId: "123", configurationId: "456", graphVersion: "v23.0" };

test("connection UI offers official WhatsApp only without legacy QR or group binding", () => {
  const source = readFileSync(new URL("../components/dashboard/views/WhatsAppConnectionView.jsx", import.meta.url), "utf8");
  assert.doesNotMatch(source, /value="whatsmeow"|Scan QR|qrModalOpen|bindModalOpen/);
  assert.match(source, /Gunakan koneksi resmi Meta/);
});

test("official channel forms have a single-column mobile layout", () => {
  const css = readFileSync(new URL("../app/(application)/globals.css", import.meta.url), "utf8");
  assert.match(css, /\.channel-connect-form,\s*\.channel-connect-form-wa \{ grid-template-columns: 1fr;/);
  assert.match(css, /\.topbar \.company-selector\s*\{\s*display: none;/);
});

test("WhatsApp management includes disconnected sessions for recovery", () => {
  const source = readFileSync(new URL("../components/dashboard/DashboardApp.jsx", import.meta.url), "utf8");
  const view = source.split("<WhatsAppConnectionView")[1].split("/>")[0];
  assert.match(view, /sessions=\{waSessions\}/);
  const create = source.split("async function createWhatsAppSession")[1].split("async function connectOfficialWhatsApp")[0];
  const recovery = create.split("catch (sessionError)")[1];
  assert.doesNotMatch(recovery, /method: "DELETE"/);
  assert.match(recovery, /await loadDashboardData/);
});

test("mobile dashboard does not block the WhatsApp connection view", () => {
  const source = readFileSync(new URL("../components/dashboard/DashboardApp.jsx", import.meta.url), "utf8");
  const allowed = source.match(/const mobileOperationsViews = \{([\s\S]*?)\n\};/)[1];
  for (const role of ["owner", "super_admin", "admin", "operator"]) {
    const views = allowed.match(new RegExp(`${role}: \\[([^\\]]*)\\]`))[1];
    assert.ok(views.includes('"whatsapp"'), `${role} must be able to open the WhatsApp view on mobile`);
  }
});
const event = (kind, data = { waba_id: "123", phone_number_id: "456" }) => ({ origin: "https://www.facebook.com", data: { type: "WA_EMBEDDED_SIGNUP", event: kind, data } });

test("coexistence requires explicit, complete server configuration", () => {
  assert.doesNotThrow(() => validateCoexistenceConfig(config));
  for (const patch of [{ enabled: false }, { onboardingMode: "cloud" }, { onboardingMode: undefined }, { appId: "" }, { configurationId: "" }, { graphVersion: "" }]) {
    assert.throws(() => validateCoexistenceConfig({ ...config, ...patch }));
  }
});

test("only trusted coexistence completion supplies connection IDs", () => {
  const valid = event("FINISH_WHATSAPP_BUSINESS_APP_ONBOARDING");
  assert.deepEqual(parseCoexistenceEvent(valid), { event: valid.data.event, wabaId: "123", phoneNumberId: "456" });
  assert.equal(parseCoexistenceEvent({ ...valid, origin: "https://www.facebook.com.attacker.test" }), null);
  assert.deepEqual(parseCoexistenceEvent({ ...valid, origin: "https://web.facebook.com", data: JSON.stringify(valid.data) }), parseCoexistenceEvent(valid));
});

test("invalid data and unrelated messages are ignored", () => {
  assert.equal(parseCoexistenceEvent({ origin: "https://www.facebook.com", data: "invalid JSON" }), null);
  assert.equal(parseCoexistenceEvent({ origin: "https://www.facebook.com", data: null }), null);
  assert.equal(parseCoexistenceEvent(event("STEP_PROGRESS")), null);
});

test("cancel, error, migration and incomplete completion stop signup", () => {
  for (const kind of ["CANCEL", "ERROR", "FINISH", "FINISH_ONLY_WABA"]) {
    assert.throws(() => parseCoexistenceEvent(event(kind)));
  }
  for (const data of [{}, { waba_id: "123" }, { waba_id: "123", phone_number_id: "../456" }]) {
    assert.throws(() => parseCoexistenceEvent(event("FINISH_WHATSAPP_BUSINESS_APP_ONBOARDING", data)));
  }
  assert.throws(() => parseCoexistenceEvent(event("ERROR", { error_message: "PRIVATE_PROVIDER_DETAIL" })), (error) => !error.message.includes("PRIVATE_PROVIDER_DETAIL"));
});
