import test from "node:test";
import assert from "node:assert/strict";

import {
  AGENT_SETUP_STEP_ORDER,
  getAgentSetupProgress,
  getNextAgentSetupStep,
  getPreviousAgentSetupStep,
  inferAgentTemplateKey,
} from "../lib/agent-onboarding.mjs";

test("agent onboarding suggests a relevant template from a natural business description", () => {
  const cases = [
    ["Aku jual skincare lewat toko online dan sering ditanya stok.", "ecommerce"],
    ["Saya punya salon perempuan yang menerima reservasi jadwal.", "booking-service"],
    ["Klinik gigi untuk informasi layanan dan jadwal dokter.", "clinic-admin"],
    ["Tempat les bahasa Inggris dan kelas persiapan IELTS.", "education"],
    ["Jual rumah, tanah, dan apartemen untuk disewakan.", "property"],
    ["Bisnis hotel, villa, dan paket wisata Bali.", "travel-hospitality"],
    ["Agency SaaS B2B yang menjadwalkan demo untuk calon klien.", "b2b-sales"],
    ["Saya butuh CS untuk menjawab pertanyaan pelanggan.", "general-cs"],
  ];

  for (const [description, expected] of cases) {
    assert.equal(inferAgentTemplateKey(description), expected, description);
  }
});

test("agent onboarding uses a deterministic one-question sequence", () => {
  assert.deepEqual(AGENT_SETUP_STEP_ORDER, [
    "business",
    "template",
    "business-name",
    "agent-name",
    "details",
    "handoff",
    "tools",
    "review",
  ]);

  assert.equal(getNextAgentSetupStep("business"), "template");
  assert.equal(getNextAgentSetupStep("tools"), "review");
  assert.equal(getNextAgentSetupStep("review"), "review");
  assert.equal(getPreviousAgentSetupStep("business"), "business");
  assert.equal(getPreviousAgentSetupStep("review"), "tools");
  assert.deepEqual(getAgentSetupProgress("details"), { current: 5, total: 8 });
});
