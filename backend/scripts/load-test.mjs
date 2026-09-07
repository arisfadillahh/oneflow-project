const backendURL = stripTrailingSlash(process.env.LOAD_BACKEND_URL || "http://localhost:8080");
const aiURL = stripTrailingSlash(process.env.LOAD_AI_URL || "http://localhost:8000");
const token = process.env.LOAD_TOKEN || "";
const aiAgentId = process.env.LOAD_AI_AGENT_ID || "";
const includePlayground = truthy(process.env.LOAD_INCLUDE_PLAYGROUND);
const profile = (process.env.LOAD_PROFILE || "standard").toLowerCase();
const timeoutMs = Number(process.env.LOAD_TIMEOUT_MS || 30000);
const playgroundTimeoutMs = Number(process.env.LOAD_PLAYGROUND_TIMEOUT_MS || 180000);
const scenarioFilter = new Set(
  String(process.env.LOAD_SCENARIOS || "")
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean),
);

const profileScale = {
  quick: 0.25,
  standard: 1,
  heavy: 2,
  brutal: 4,
}[profile] || 1;

const scenarios = [
  scenario("backend_health", "GET", `${backendURL}/health`, 1200, 120, { p95BudgetMs: 1000 }),
  scenario("ai_health", "GET", `${aiURL}/healthz`, 800, 80, { p95BudgetMs: 1000 }),
  token && scenario("api_me", "GET", `${backendURL}/api/me`, 500, 50, { auth: true, p95BudgetMs: 1500 }),
  token && scenario("api_agents", "GET", `${backendURL}/api/ai-agents`, 300, 30, { auth: true, p95BudgetMs: 2500 }),
  token && scenario("api_dashboard_summary", "GET", `${backendURL}/api/dashboard/summary`, 300, 30, { auth: true, p95BudgetMs: 2500 }),
  token && scenario("api_business_tools", "GET", `${backendURL}/api/business-tools`, 240, 24, { auth: true, p95BudgetMs: 2500 }),
  token && scenario("api_commerce_products", "GET", `${backendURL}/api/commerce/products`, 240, 24, { auth: true, p95BudgetMs: 2500 }),
  token && scenario("api_billing_wallet", "GET", `${backendURL}/api/billing/wallet`, 300, 30, { auth: true, p95BudgetMs: 2500 }),
  includePlayground && token && aiAgentId && scenario("playground_ai_limited", "POST", `${backendURL}/api/playground/run`, 12, 3, {
    auth: true,
    timeoutMs: playgroundTimeoutMs,
    p95BudgetMs: 90000,
    maxErrorRate: 0.1,
    bodyFactory: (index) => ({
      aiAgentId,
      customerName: `Load Tester ${Date.now()} ${index}`,
      messageText: [
        "Halo, QA Coffee buka jam berapa?",
        "Ada produk apa aja dan harganya?",
        "Aku mau Kopi Susu QA pickup.",
      ][index % 3],
      history: [],
    }),
  }),
].filter(Boolean).filter((item) => !scenarioFilter.size || scenarioFilter.has(item.name));

function stripTrailingSlash(value) {
  return String(value || "").replace(/\/+$/, "");
}

function truthy(value) {
  return ["1", "true", "yes", "y", "on"].includes(String(value || "").toLowerCase());
}

function scaled(value) {
  return Math.max(1, Math.round(value * profileScale));
}

function scenario(name, method, url, total, concurrency, options = {}) {
  return {
    name,
    method,
    url,
    total: scaled(total),
    concurrency: Math.min(scaled(concurrency), scaled(total)),
    maxErrorRate: options.maxErrorRate ?? 0.01,
    p95BudgetMs: options.p95BudgetMs ?? 3000,
    timeoutMs: options.timeoutMs ?? timeoutMs,
    auth: Boolean(options.auth),
    bodyFactory: options.bodyFactory,
  };
}

function percentile(values, p) {
  if (!values.length) return 0;
  const sorted = [...values].sort((a, b) => a - b);
  const index = Math.min(sorted.length - 1, Math.ceil((p / 100) * sorted.length) - 1);
  return sorted[index];
}

async function requestOnce(currentScenario, index) {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), currentScenario.timeoutMs);
  const headers = {};
  let body;
  if (currentScenario.auth) headers.Authorization = `Bearer ${token}`;
  if (currentScenario.bodyFactory) {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify(currentScenario.bodyFactory(index));
  }
  const started = performance.now();
  try {
    const response = await fetch(currentScenario.url, {
      method: currentScenario.method,
      headers,
      body,
      signal: controller.signal,
    });
    await response.arrayBuffer();
    return {
      ok: response.ok,
      status: response.status,
      ms: performance.now() - started,
      error: response.ok ? "" : `HTTP ${response.status}`,
    };
  } catch (error) {
    return {
      ok: false,
      status: 0,
      ms: performance.now() - started,
      error: error?.name === "AbortError" ? "timeout" : String(error?.message || error),
    };
  } finally {
    clearTimeout(timeout);
  }
}

async function runScenario(currentScenario) {
  let nextIndex = 0;
  const results = [];
  async function worker() {
    for (;;) {
      const index = nextIndex;
      nextIndex += 1;
      if (index >= currentScenario.total) return;
      results.push(await requestOnce(currentScenario, index));
    }
  }
  const started = performance.now();
  await Promise.all(Array.from({ length: currentScenario.concurrency }, () => worker()));
  const durationMs = performance.now() - started;
  const latencies = results.map((item) => item.ms);
  const failures = results.filter((item) => !item.ok);
  const statusCounts = {};
  const errorCounts = {};
  for (const item of results) {
    statusCounts[item.status] = (statusCounts[item.status] || 0) + 1;
    if (!item.ok) errorCounts[item.error] = (errorCounts[item.error] || 0) + 1;
  }
  const errorRate = failures.length / Math.max(1, results.length);
  const summary = {
    name: currentScenario.name,
    method: currentScenario.method,
    url: currentScenario.url,
    total: currentScenario.total,
    concurrency: currentScenario.concurrency,
    durationMs: Math.round(durationMs),
    rps: Number((results.length / (durationMs / 1000)).toFixed(2)),
    ok: results.length - failures.length,
    failed: failures.length,
    errorRate: Number(errorRate.toFixed(4)),
    p50Ms: Math.round(percentile(latencies, 50)),
    p95Ms: Math.round(percentile(latencies, 95)),
    p99Ms: Math.round(percentile(latencies, 99)),
    maxMs: Math.round(Math.max(...latencies, 0)),
    statusCounts,
    errorCounts,
    pass: errorRate <= currentScenario.maxErrorRate && percentile(latencies, 95) <= currentScenario.p95BudgetMs,
    budgets: {
      maxErrorRate: currentScenario.maxErrorRate,
      p95BudgetMs: currentScenario.p95BudgetMs,
    },
  };
  return summary;
}

async function main() {
  console.log(JSON.stringify({
    profile,
    backendURL,
    aiURL,
    authenticated: Boolean(token),
    includePlayground: includePlayground && Boolean(token && aiAgentId),
    scenarios: scenarios.map(({ name, total, concurrency }) => ({ name, total, concurrency })),
  }, null, 2));

  const summaries = [];
  for (const currentScenario of scenarios) {
    console.log(`\nRunning ${currentScenario.name}: total=${currentScenario.total}, concurrency=${currentScenario.concurrency}`);
    const summary = await runScenario(currentScenario);
    summaries.push(summary);
    console.log(JSON.stringify(summary, null, 2));
  }

  const failed = summaries.filter((item) => !item.pass);
  console.log("\nSUMMARY");
  console.table(summaries.map((item) => ({
    scenario: item.name,
    pass: item.pass,
    total: item.total,
    failed: item.failed,
    errorRate: item.errorRate,
    rps: item.rps,
    p95Ms: item.p95Ms,
    maxMs: item.maxMs,
  })));
  if (failed.length) {
    process.exitCode = 1;
  }
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
