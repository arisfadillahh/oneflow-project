import test from "node:test";
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { join } from "node:path";
import { fileURLToPath } from "node:url";

const dashboardRoot = fileURLToPath(new URL("..", import.meta.url));

test("inbox badge counts unread messages instead of open conversations", async () => {
  const core = await readFile(join(dashboardRoot, "lib", "dashboard-core.jsx"), "utf8");
  assert.match(core, /item\.unreadCount/, "Inbox navigation badge must use server unread counts.");
  assert.doesNotMatch(core, /viewId === "operations"\) return summary\?\.open/, "Open conversations must not be presented as unread.");
});

test("opening a conversation records a read receipt and updates the row immediately", async () => {
  const app = await readFile(join(dashboardRoot, "components", "dashboard", "DashboardApp.jsx"), "utf8");
  assert.match(app, /\/api\/conversations\/\$\{id\}\/read/, "Dashboard must persist a read receipt.");
  assert.match(app, /unreadCount:\s*0/, "The selected row should clear without waiting for the next poll.");
});

test("inbox exposes unread state without relying on color alone", async () => {
  const view = await readFile(join(dashboardRoot, "components", "dashboard", "views", "InboxView.jsx"), "utf8");
  const css = await readFile(join(dashboardRoot, "app", "(application)", "globals.css"), "utf8");
  assert.match(view, /inbox-unread-dot/, "Unread rows need an explicit visual marker.");
  assert.match(view, /aria-label=.*inbox\.readCount/, "Unread state needs an accessible text label.");
  assert.match(css, /\.inbox-item\.is-unread/, "Unread rows need dedicated styling.");
  assert.match(css, /\.inbox-tabs\s*\{[\s\S]*?grid-template-columns:\s*repeat\(6, minmax\(0, 1fr\)\)/, "Mobile inbox filters must fit without horizontal scrolling.");
});

test("account language supports Indonesian and English with server persistence", async () => {
  const i18n = await readFile(join(dashboardRoot, "lib", "i18n.js"), "utf8");
  const account = await readFile(join(dashboardRoot, "components", "dashboard", "views", "AccountSettingsView.jsx"), "utf8");
  const app = await readFile(join(dashboardRoot, "components", "dashboard", "DashboardApp.jsx"), "utf8");
  assert.match(i18n, /id:\s*\{[\s\S]*en:\s*\{/, "Both locale dictionaries must exist.");
  assert.match(account, /supportedLocales\.map/, "Account settings must expose the supported locale selector.");
  assert.match(app, /preferredLocale:\s*accountProfileForm\.preferredLocale/, "Locale changes must be persisted through the profile API.");
  assert.match(app, /<AccountSettingsView[\s\S]*?locale=\{locale\}/, "Account settings must render with the active locale.");
});

test("WhatsApp window guidance stays compact and has dark-mode coverage", async () => {
  const view = await readFile(join(dashboardRoot, "components", "dashboard", "views", "InboxView.jsx"), "utf8");
  const css = await readFile(join(dashboardRoot, "app", "(application)", "globals.css"), "utf8");
  assert.match(view, /whatsapp-window-status/, "Composer should use the compact window status.");
  assert.doesNotMatch(view, /Aturan follow-up WhatsApp/, "The old dominant follow-up banner must be removed.");
  assert.match(css, /\.dashboard-shell\.theme-dark \.whatsapp-window-status/, "Window status needs explicit dark-mode styling.");
});
