export const normalizePublicBase = (value) => String(value || "").trim().replace(/\/+$/, "");

export const isLoopbackHostname = (hostname) =>
  hostname === "localhost" || hostname === "127.0.0.1" || hostname === "::1";

export const hostForUrl = (hostname) => (hostname === "::1" ? "[::1]" : hostname);

export const resolveBrowserLocalBase = ({ location, port, protocolOverride } = {}) => {
  const hostname = location?.hostname || "";
  if (!isLoopbackHostname(hostname)) return "";
  const protocol = protocolOverride || location?.protocol || "http:";
  return `${protocol}//${hostForUrl(hostname)}:${port}`;
};

const originForLocation = (location) => {
  if (!location?.hostname) return "";
  return `${location.protocol || "http:"}//${location.host || hostForUrl(location.hostname)}`;
};

const isLoopbackBase = (base) => {
  try {
    return isLoopbackHostname(new URL(base).hostname);
  } catch {
    return false;
  }
};

const isLocalDevDashboardPort = (location) => {
  const port = String(location?.port || location?.host?.split(":").pop() || "");
  return port === "3000" || port === "3001";
};

export const resolveBrowserApiBase = ({ env = {}, location } = {}) => {
  const configuredBase = normalizePublicBase(env.NEXT_PUBLIC_API_BASE_URL);
  if (location && isLoopbackHostname(location.hostname)) {
    if (configuredBase && !isLoopbackBase(configuredBase)) return originForLocation(location);
    if (!configuredBase && !isLocalDevDashboardPort(location)) return originForLocation(location);
    return configuredBase || resolveBrowserLocalBase({ location, port: 8080 });
  }
  return configuredBase || resolveBrowserLocalBase({ location, port: 8080 });
};

export const resolveBrowserWsBase = ({ env = {}, location } = {}) => {
  const configuredBase = normalizePublicBase(env.NEXT_PUBLIC_WS_BASE_URL);
  if (configuredBase) return configuredBase;
  if (!location) return "ws://localhost:8080/ws";

  const localBase = resolveBrowserLocalBase({
    location,
    port: 8080,
    protocolOverride: location.protocol === "https:" ? "wss:" : "ws:",
  });
  if (localBase) return `${localBase}/ws`;

  const protocol = location.protocol === "https:" ? "wss:" : "ws:";
  return `${protocol}//${location.host}/ws`;
};

export const resolveWaGatewayBase = ({ env = {} } = {}) =>
  normalizePublicBase(env.NEXT_PUBLIC_WA_GATEWAY_URL);

export const resolveServerApiProxyTarget = ({ env = {} } = {}) =>
  normalizePublicBase(env.API_PROXY_TARGET) || "http://app-backend:8080";
