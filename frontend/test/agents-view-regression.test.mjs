import test from "node:test";
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { join } from "node:path";
import { fileURLToPath } from "node:url";

const dashboardRoot = fileURLToPath(new URL("..", import.meta.url));

test("agent instructions memory toggle has a local component definition", async () => {
  const source = await readFile(join(dashboardRoot, "components", "dashboard", "views", "AgentsView.jsx"), "utf8");
  const definitionIndex = source.indexOf("function AgentRuleToggle");
  const usageIndex = source.indexOf("<AgentRuleToggle");

  assert.notEqual(definitionIndex, -1, "AgentRuleToggle must be defined before it is rendered.");
  assert.notEqual(usageIndex, -1, "The customer memory rule should keep rendering through AgentRuleToggle.");
  assert.ok(definitionIndex < usageIndex, "AgentRuleToggle must be declared before the JSX usage.");
});

test("guided agent system prompts define natural opening rules without forcing a salutation", async () => {
  const source = await readFile(join(dashboardRoot, "components", "dashboard", "views", "AgentsView.jsx"), "utf8");

  for (const instruction of [
    "Pilih fungsi pembuka sesuai intent",
    "memberi arah jawaban, bukan sekadar basa-basi generik.",
    "Jangan langsung membuka dengan fakta mentah",
    "setelah pembuka gunakan subjudul singkat",
    "Jangan memakai stock phrase sebagai awalan universal.",
    "Jangan mengulang konstruksi pembuka atau 2-4 kata awal yang sama.",
    "Pembuka harus langsung merespons maksud customer",
    "Jangan memulai dengan judul Knowledge",
    "Jangan mengulang pola pembuka yang sama pada dua balasan berurutan.",
    "Jangan mengarang nama atau memaksakan sapaan tertentu.",
    "langsung jawab tanpa pembuka tambahan.",
  ]) {
    assert.ok(source.includes(instruction), `Guided agent prompts should include: ${instruction}`);
  }

  for (const stockPhrase of ["Jadi gini", "Baik, Kak Aris", "Pertanyaan bagus, Kak"]) {
    assert.ok(!source.includes(stockPhrase), `Guided agent prompts must not anchor on: ${stockPhrase}`);
  }
});

test("dark-mode first-run onboarding and portaled agent modal keep readable surfaces", async () => {
  const css = await readFile(join(dashboardRoot, "app", "(application)", "globals.css"), "utf8");

  for (const selector of [
    'html[data-dashboard-theme="dark"] .first-run-modal',
    'html[data-dashboard-theme="dark"] .first-run-copy h2',
    'html[data-dashboard-theme="dark"] .first-run-visual',
  ]) {
    assert.match(css, new RegExp(selector.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")), `${selector} should have explicit dark-mode coverage.`);
  }

  assert.ok(css.includes(".agent-create-title-block .modal-title"), "Portaled agent modal title should be covered by dark-mode text selectors.");
  assert.ok(css.includes(".agent-business-writing-guide"), "Agent setup helper card should be covered by dark-mode surface selectors.");
  assert.ok(css.includes('html[data-dashboard-theme="dark"] .agent-conversation-panel'), "Conversational setup panel should have explicit dark-mode surface coverage.");
});

test("mobile chat-first agent setup keeps one message scroll surface and a stable composer", async () => {
  const css = await readFile(join(dashboardRoot, "app", "(application)", "globals.css"), "utf8");

  assert.ok(css.includes(".agent-setup-messages"), "The conversation should own the scrollable message history.");
  assert.ok(css.includes(".agent-setup-composer"), "Text questions should keep a dedicated composer.");
  assert.ok(css.includes(".agent-setup-draft-mobile"), "Mobile users should be able to inspect the structured draft.");
  assert.ok(css.includes("padding-bottom: max(12px, env(safe-area-inset-bottom));"), "The mobile action area should clear the device safe area.");
});

test("agent setup onboarding is chat-first while keeping a structured editable draft", async () => {
  const source = await readFile(join(dashboardRoot, "components", "dashboard", "views", "AgentsView.jsx"), "utf8");
  const css = await readFile(join(dashboardRoot, "app", "(application)", "globals.css"), "utf8");

  for (const snippet of [
    "function ConversationalAgentSetup",
    'aria-label="Percakapan membuat AI CS"',
    "Satu pertanyaan setiap langkah",
    "Kamu cukup jawab seperti ngobrol biasa",
    "inferAgentTemplateKey",
    "AgentSetupDraftSummary",
    "messagesRef.current.scrollTop",
    "Tool tetap mode draf dan keputusan akhir ada di admin.",
    "onEditStep",
  ]) {
    assert.ok(source.includes(snippet), `Conversational setup should include: ${snippet}`);
  }

  const modalStart = source.indexOf("const createAgentModal");
  const modalEnd = source.indexOf("const isDeletingAgent", modalStart);
  const modalSource = source.slice(modalStart, modalEnd);
  assert.ok(modalSource.includes("<ConversationalAgentSetup"), "The modal should render the conversation as its main setup surface.");
  assert.ok(!modalSource.includes("agent-create-progress"), "The old four-card wizard progress should be removed.");
  assert.ok(!modalSource.includes("agent-create-template-column"), "The old duplicate template form should be removed.");
  assert.ok(css.includes(".agent-chat-first-layout"), "The chat-first modal should have a dedicated responsive layout.");
  assert.ok(css.includes('html[data-dashboard-theme="dark"] .agent-setup-chat'), "The chat surface should have explicit dark-mode coverage.");
  assert.ok(!source.includes("Jawab tiga pertanyaan singkat"), "The old wizard framing should not remain in the modal.");
  assert.ok(source.includes("Konfirmasi setup yang disarankan"), "The empty state should describe the new conversation order.");
});

test("new agents start business tools in safe draft mode", async () => {
  const source = await readFile(join(dashboardRoot, "components", "dashboard", "views", "AgentsView.jsx"), "utf8");
  const modeFunction = source.slice(source.indexOf("function businessToolSetupMode"), source.indexOf("function defaultSetupToolModes"));
  const detailDefaults = source.slice(source.indexOf("function defaultToolSetupDetails"), source.indexOf("function buildAgentInstructionDraft"));

  assert.ok(modeFunction.includes('return "draft";'), "Recommended business tools should start in draft mode.");
  assert.ok(!modeFunction.includes('return "action";'), "Agent onboarding must not silently grant action mode.");
  assert.ok(detailDefaults.includes('bookingServiceName: ""'), "Example booking data must not be saved as real business data.");
  assert.ok(source.includes("Semua alat dimulai sebagai draf."), "The tool step should explain the draft-mode impact.");
});

test("agent setup only counts and saves business tools available to the user", async () => {
  const source = await readFile(join(dashboardRoot, "components", "dashboard", "views", "AgentsView.jsx"), "utf8");

  assert.ok(source.includes("selectedAvailableSetupToolKeys = useMemo"), "The setup summary should derive selections from the visible tool catalog.");
  assert.ok(source.includes("const toolConfigs = selectedAvailableSetupToolKeys.map"), "Unavailable recommended tools must not be saved silently.");
});

test("agent creation refresh cannot stall on the external model catalog", async () => {
  const dashboardSource = await readFile(join(dashboardRoot, "components", "dashboard", "DashboardApp.jsx"), "utf8");
  const agentSource = await readFile(join(dashboardRoot, "components", "dashboard", "views", "AgentsView.jsx"), "utf8");

  assert.ok(dashboardSource.includes("new AbortController()"), "Dashboard requests should support an explicit cancellation deadline.");
  assert.ok(dashboardSource.includes("timeoutMs: 5000"), "Model catalog refresh should have a short frontend timeout.");
  assert.ok(!agentSource.includes("isReviewCreateArmed"), "Review should not use a hidden timer before creation is enabled.");
  assert.ok(agentSource.includes('isSavingAgent ? "Membuat..." : "Buat AI CS"'), "The final action should stay explicit and immediate.");
  assert.ok(!agentSource.includes('"Cek review dulu"'), "Review should not imply a hidden action is required.");
});

test("payment-required errors distinguish plan limits from playground credits", async () => {
  const source = await readFile(join(dashboardRoot, "components", "dashboard", "DashboardApp.jsx"), "utf8");

  assert.ok(source.includes('normalized.includes("limit reached")'), "Plan-capacity errors should be detected before the generic credit message.");
  assert.ok(source.includes('normalized.includes("is not included")'), "Unavailable plan features should use plan guidance.");
  assert.ok(source.includes("Batas AI CS untuk paket saat ini sudah terpakai."), "AI-agent capacity should not be described as exhausted Playground credit.");
  assert.ok(source.indexOf('normalized.includes("limit reached")') < source.indexOf('normalized.includes("insufficient")'), "Plan-capacity mapping must run before credit mapping.");
});
