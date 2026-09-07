const baseURL = stripTrailingSlash(process.env.SECURITY_BASE_URL || "http://localhost:8080");
const token = process.env.SECURITY_TOKEN || "";
const timeoutMs = Number(process.env.SECURITY_TIMEOUT_MS || 15000);

function stripTrailingSlash(value) {
  return String(value || "").replace(/\/+$/, "");
}

async function request(path, options = {}) {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), timeoutMs);
  try {
    const response = await fetch(`${baseURL}${path}`, {
      method: options.method || "GET",
      headers: options.headers || {},
      body: options.body,
      signal: controller.signal,
      redirect: "manual",
    });
    const text = await response.text().catch(() => "");
    return { response, text };
  } finally {
    clearTimeout(timeout);
  }
}

function hasHeader(response, key, expectedPattern) {
  const value = response.headers.get(key) || "";
  return expectedPattern ? expectedPattern.test(value) : Boolean(value);
}

function pass(name, details = {}) {
  return { name, pass: true, ...details };
}

function fail(name, details = {}) {
  return { name, pass: false, ...details };
}

async function main() {
  const checks = [];

  const root = await request("/");
  checks.push(root.response.status !== 200 ? pass("root_not_public") : fail("root_not_public", { status: root.response.status }));

  const health = await request("/health");
  checks.push(health.response.status === 200 ? pass("health_returns_200") : fail("health_returns_200", { status: health.response.status }));
  checks.push(hasHeader(health.response, "x-content-type-options", /nosniff/i) ? pass("header_x_content_type_options") : fail("header_x_content_type_options"));
  checks.push(hasHeader(health.response, "x-frame-options", /DENY|SAMEORIGIN/i) ? pass("header_x_frame_options") : fail("header_x_frame_options"));
  checks.push(hasHeader(health.response, "referrer-policy") ? pass("header_referrer_policy") : fail("header_referrer_policy"));

  for (const path of ["/api/me", "/api/ai-agents", "/api/billing/wallet"]) {
    const result = await request(path);
    checks.push(
      [401, 403].includes(result.response.status)
        ? pass(`auth_required_${path}`)
        : fail(`auth_required_${path}`, { status: result.response.status }),
    );
  }

  const invalidOrigin = await request("/api/me", {
    headers: { Origin: "https://evil.example" },
  });
  checks.push(
    invalidOrigin.response.status === 403
      ? pass("cors_rejects_untrusted_origin")
      : fail("cors_rejects_untrusted_origin", { status: invalidOrigin.response.status }),
  );

  const internalWA = await request("/api/internal/wa/inbound", { method: "POST", headers: { "Content-Type": "application/json" }, body: "{}" });
  checks.push(
    [401, 403].includes(internalWA.response.status)
      ? pass("internal_wa_requires_internal_auth")
      : fail("internal_wa_requires_internal_auth", { status: internalWA.response.status }),
  );

  const publicModelCatalog = await request("/api/openrouter-models");
  checks.push(
    [401, 403, 404].includes(publicModelCatalog.response.status)
      ? pass("openrouter_model_catalog_not_public")
      : fail("openrouter_model_catalog_not_public", { status: publicModelCatalog.response.status, body: publicModelCatalog.text.slice(0, 120) }),
  );

  checks.push(
    health.response.status === 200 && !/"service"\s*:/.test(health.text) && !/"error"\s*:/.test(health.text)
      ? pass("health_omits_service_and_error_details")
      : fail("health_omits_service_and_error_details", { status: health.response.status, body: health.text.slice(0, 120) }),
  );

  const loginInjection = await request("/api/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username: "' OR 1=1 --", password: "not-the-password" }),
  });
  checks.push(
    [400, 401].includes(loginInjection.response.status) && !/"token"\s*:/.test(loginInjection.text)
      ? pass("login_sqlish_payload_rejected")
      : fail("login_sqlish_payload_rejected", { status: loginInjection.response.status, body: loginInjection.text.slice(0, 120) }),
  );

  for (const path of ["/.env", "/.git/config", "/docker-compose.yml", "/server-status"]) {
    const result = await request(path);
    checks.push(
      result.response.status !== 200
        ? pass(`sensitive_path_not_served_${path}`)
        : fail(`sensitive_path_not_served_${path}`, { status: result.response.status }),
    );
  }

  if (token) {
    const authed = await request("/api/me", { headers: { Authorization: `Bearer ${token}` } });
    checks.push(authed.response.status === 200 ? pass("valid_token_me_works") : fail("valid_token_me_works", { status: authed.response.status }));
  }

  console.table(checks.map(({ name, pass, status }) => ({ check: name, pass, status: status ?? "" })));
  const failed = checks.filter((item) => !item.pass);
  console.log(JSON.stringify({ baseURL, total: checks.length, passed: checks.length - failed.length, failed: failed.length, failures: failed }, null, 2));
  if (failed.length) process.exitCode = 1;
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
