import assert from "node:assert/strict";
import { readdir, readFile } from "node:fs/promises";
import { join, relative } from "node:path";
import { fileURLToPath } from "node:url";

const dashboardRoot = fileURLToPath(new URL("..", import.meta.url));
const contractPath = join(dashboardRoot, "contracts", "dashboard-api.json");
const sourceRoots = ["app", "components", "lib"].map((part) => join(dashboardRoot, part));
const ignoredSegments = new Set([".next", "content", "node_modules", "public"]);
const sourceExtensions = new Set([".js", ".jsx", ".mjs", ".ts", ".tsx"]);
const requiredEnvKeys = [
  "NEXT_PUBLIC_API_BASE_URL",
  "NEXT_PUBLIC_WS_BASE_URL",
  "NEXT_PUBLIC_WA_GATEWAY_URL",
  "API_PROXY_TARGET",
];

const extensionOf = (filePath) => {
  const dotIndex = filePath.lastIndexOf(".");
  return dotIndex >= 0 ? filePath.slice(dotIndex) : "";
};

const collectFiles = async (directory) => {
  const entries = await readdir(directory, { withFileTypes: true });
  const files = [];
  for (const entry of entries) {
    if (ignoredSegments.has(entry.name)) continue;
    const entryPath = join(directory, entry.name);
    if (entry.isDirectory()) {
      files.push(...await collectFiles(entryPath));
    } else if (entry.isFile() && sourceExtensions.has(extensionOf(entry.name))) {
      files.push(entryPath);
    }
  }
  return files;
};

const readSources = async () => {
  const files = (await Promise.all(sourceRoots.map(collectFiles))).flat();
  return Promise.all(files.map(async (filePath) => ({
    filePath,
    relativePath: relative(dashboardRoot, filePath),
    content: await readFile(filePath, "utf8"),
  })));
};

const extractApiPaths = (content) => {
  const paths = new Set();
  for (const match of content.matchAll(/\/api\/[A-Za-z0-9_-]+(?:\/[A-Za-z0-9_-]+)?(?:\/[A-Za-z0-9_-]+)?/g)) {
    paths.add(match[0]);
  }
  return paths;
};

const main = async () => {
  const contract = JSON.parse(await readFile(contractPath, "utf8"));
  const routeMatchers = contract.routeFamilies.map((family) => ({
    id: family.id,
    regex: new RegExp(family.pathPattern),
  }));
  const sources = await readSources();

  const forbiddenHits = [];
  for (const source of sources) {
    for (const pattern of contract.forbiddenBrowserPatterns) {
      if (source.content.includes(pattern)) {
        forbiddenHits.push(`${source.relativePath}: ${pattern}`);
      }
    }
  }
  assert.deepEqual(forbiddenHits, [], "Forbidden browser boundary references found.");

  const apiPaths = new Map();
  for (const source of sources) {
    for (const path of extractApiPaths(source.content)) {
      if (!apiPaths.has(path)) apiPaths.set(path, new Set());
      apiPaths.get(path).add(source.relativePath);
    }
  }

  const uncovered = [];
  for (const [path, files] of apiPaths.entries()) {
    if (!routeMatchers.some((matcher) => matcher.regex.test(path))) {
      uncovered.push(`${path} (${Array.from(files).join(", ")})`);
    }
  }
  assert.deepEqual(uncovered, [], "Dashboard API path is not covered by contracts/dashboard-api.json.");

  const envExample = await readFile(join(dashboardRoot, ".env.example"), "utf8");
  const missingEnvKeys = requiredEnvKeys.filter((key) => !envExample.includes(`${key}=`));
  assert.deepEqual(missingEnvKeys, [], "Standalone frontend env example is missing required split keys.");

  console.log(`Split readiness verified: ${apiPaths.size} API path prefixes, ${routeMatchers.length} contract families, ${sources.length} source files.`);
};

main().catch((error) => {
  console.error(error.message || error);
  process.exitCode = 1;
});
