import test from "node:test";
import assert from "node:assert/strict";

import {
  normalizePublicBase,
  resolveBrowserApiBase,
  resolveBrowserLocalBase,
  resolveBrowserWsBase,
  resolveServerApiProxyTarget,
  resolveWaGatewayBase,
} from "../lib/runtime-config.mjs";

test("normalizes configured public bases", () => {
  assert.equal(normalizePublicBase(" https://api.staging.oneflow.id/// "), "https://api.staging.oneflow.id");
  assert.equal(normalizePublicBase(""), "");
  assert.equal(normalizePublicBase(undefined), "");
});

test("uses local app-backend when dashboard runs on loopback without explicit API env", () => {
  assert.equal(
    resolveBrowserApiBase({ env: {}, location: { hostname: "localhost", protocol: "http:", host: "localhost:3000" } }),
    "http://localhost:8080",
  );
  assert.equal(
    resolveBrowserApiBase({ env: {}, location: { hostname: "127.0.0.1", protocol: "http:", host: "127.0.0.1:3000" } }),
    "http://127.0.0.1:8080",
  );
  assert.equal(
    resolveBrowserLocalBase({ location: { hostname: "::1", protocol: "http:", host: "[::1]:3000" }, port: 8080 }),
    "http://[::1]:8080",
  );
});

test("keeps loopback production-build QA on the active origin instead of deployed API env", () => {
  assert.equal(
    resolveBrowserApiBase({
      env: { NEXT_PUBLIC_API_BASE_URL: "https://oneflow.id" },
      location: { hostname: "127.0.0.1", protocol: "http:", host: "127.0.0.1:18088" },
    }),
    "http://127.0.0.1:18088",
  );
  assert.equal(
    resolveBrowserApiBase({
      env: {},
      location: { hostname: "127.0.0.1", protocol: "http:", host: "127.0.0.1:18088", port: "18088" },
    }),
    "http://127.0.0.1:18088",
  );
});

test("keeps deployed dashboard on same origin unless an API env is configured", () => {
  assert.equal(
    resolveBrowserApiBase({ env: {}, location: { hostname: "oneflow.id", protocol: "https:", host: "oneflow.id" } }),
    "",
  );
  assert.equal(
    resolveBrowserApiBase({
      env: { NEXT_PUBLIC_API_BASE_URL: "https://api.oneflow.id/" },
      location: { hostname: "oneflow.id", protocol: "https:", host: "oneflow.id" },
    }),
    "https://api.oneflow.id",
  );
});

test("resolves websocket base for local, deployed, and configured frontend", () => {
  assert.equal(resolveBrowserWsBase({ env: {}, location: undefined }), "ws://localhost:8080/ws");
  assert.equal(
    resolveBrowserWsBase({ env: {}, location: { hostname: "localhost", protocol: "http:", host: "localhost:3000" } }),
    "ws://localhost:8080/ws",
  );
  assert.equal(
    resolveBrowserWsBase({ env: {}, location: { hostname: "oneflow.id", protocol: "https:", host: "oneflow.id" } }),
    "wss://oneflow.id/ws",
  );
  assert.equal(
    resolveBrowserWsBase({ env: { NEXT_PUBLIC_WS_BASE_URL: "wss://ws.oneflow.id/ws/" } }),
    "wss://ws.oneflow.id/ws",
  );
});

test("keeps WA gateway optional and trims server proxy targets", () => {
  assert.equal(resolveWaGatewayBase({ env: {} }), "");
  assert.equal(resolveWaGatewayBase({ env: { NEXT_PUBLIC_WA_GATEWAY_URL: "http://localhost:8090/" } }), "http://localhost:8090");
  assert.equal(resolveServerApiProxyTarget({ env: {} }), "http://app-backend:8080");
  assert.equal(resolveServerApiProxyTarget({ env: { API_PROXY_TARGET: " http://127.0.0.1:8080/ " } }), "http://127.0.0.1:8080");
});
