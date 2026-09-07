import test from "node:test";
import assert from "node:assert/strict";
import { readdir, readFile } from "node:fs/promises";
import { join, relative } from "node:path";
import { fileURLToPath } from "node:url";

const dashboardRoot = fileURLToPath(new URL("..", import.meta.url));
const sourceRoots = ["app", "components", "lib"].map((part) => join(dashboardRoot, part));
const ignoredSegments = new Set([".next", "content", "node_modules", "public"]);
const textExtensions = new Set([".js", ".jsx", ".mjs", ".ts", ".tsx"]);

const extensionOf = (filePath) => {
  const dotIndex = filePath.lastIndexOf(".");
  return dotIndex >= 0 ? filePath.slice(dotIndex) : "";
};

const collectSourceFiles = async (directory) => {
  const entries = await readdir(directory, { withFileTypes: true });
  const files = [];
  for (const entry of entries) {
    if (ignoredSegments.has(entry.name)) continue;
    const entryPath = join(directory, entry.name);
    if (entry.isDirectory()) {
      files.push(...await collectSourceFiles(entryPath));
    } else if (entry.isFile() && textExtensions.has(extensionOf(entry.name))) {
      files.push(entryPath);
    }
  }
  return files;
};

const readDashboardSources = async () => {
  const files = (await Promise.all(sourceRoots.map(collectSourceFiles))).flat();
  return Promise.all(files.map(async (filePath) => ({
    filePath,
    relativePath: relative(dashboardRoot, filePath),
    content: await readFile(filePath, "utf8"),
  })));
};

test("public dashboard navigation stays relative for local and split-repo frontend runs", async () => {
  const sources = await readDashboardSources();
  const offenders = sources
    .filter(({ content }) => content.includes("https://oneflow.id/dashboard"))
    .map(({ relativePath }) => relativePath);

  assert.deepEqual(offenders, [], "Use /dashboard or an env-derived base instead of hardcoding the production dashboard URL.");
});

test("browser frontend does not call the internal AI runtime directly", async () => {
  const sources = await readDashboardSources();
  const forbiddenPatterns = [
    /NEXT_PUBLIC_AI_SERVICE_URL/,
    /ai-service(?::|\/)/,
    /localhost:8000/,
    /127\.0\.0\.1:8000/,
    /\/api\/decide\b/,
    /\/api\/ticket-triage\b/,
  ];
  const offenders = sources
    .filter(({ content }) => forbiddenPatterns.some((pattern) => pattern.test(content)))
    .map(({ relativePath }) => relativePath);

  assert.deepEqual(offenders, [], "Frontend must call app-backend only; app-backend owns AI billing, tenant isolation, and guardrails.");
});
