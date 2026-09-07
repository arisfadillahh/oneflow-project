import { resolveServerApiProxyTarget } from "./lib/runtime-config.mjs";

const apiProxyTarget = resolveServerApiProxyTarget({ env: process.env });
const isProduction = process.env.NODE_ENV === "production";

const securityHeaders = [
  {
    key: "Content-Security-Policy",
    value: [
      "default-src 'self'",
      "base-uri 'self'",
      "object-src 'none'",
      "frame-ancestors 'none'",
      "form-action 'self'",
      [
        "script-src 'self' 'unsafe-inline'",
        isProduction ? "" : "'unsafe-eval'",
        "https://connect.facebook.net",
        "https://app.sandbox.midtrans.com",
        "https://app.midtrans.com",
      ].filter(Boolean).join(" "),
      "style-src 'self' 'unsafe-inline' https://fonts.googleapis.com",
      "img-src 'self' data: blob: https:",
      "font-src 'self' data: https://fonts.gstatic.com",
      [
        "connect-src 'self'",
        "https://oneflow.id",
        "https://www.oneflow.id",
        "wss://oneflow.id",
        "wss://www.oneflow.id",
        "https://app.sandbox.midtrans.com",
        "https://app.midtrans.com",
        "https://www.facebook.com",
        "https://web.facebook.com",
        "https://graph.facebook.com",
        "http://localhost:*",
        "http://127.0.0.1:*",
        "ws://localhost:*",
        "ws://127.0.0.1:*",
      ].join(" "),
      "frame-src https://www.facebook.com https://web.facebook.com https://app.sandbox.midtrans.com https://app.midtrans.com",
    ].join("; "),
  },
  {
    key: "Strict-Transport-Security",
    value: "max-age=31536000; includeSubDomains",
  },
  {
    key: "X-Frame-Options",
    value: "DENY",
  },
  {
    key: "X-Content-Type-Options",
    value: "nosniff",
  },
  {
    key: "Referrer-Policy",
    value: "strict-origin-when-cross-origin",
  },
  {
    key: "Permissions-Policy",
    value: "camera=(), microphone=(), geolocation=(), payment=()",
  },
];

const nextConfig = {
  output: "standalone",
  poweredByHeader: false,
  async headers() {
    return [
      {
        source: "/:path*",
        headers: securityHeaders,
      },
    ];
  },
  async rewrites() {
    return [
      {
        source: "/api/:path*",
        destination: `${apiProxyTarget}/api/:path*`,
      },
      {
        source: "/health",
        destination: `${apiProxyTarget}/health`,
      },
    ];
  },
};

export default nextConfig;
