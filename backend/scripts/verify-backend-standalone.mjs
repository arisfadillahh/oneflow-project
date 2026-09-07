import assert from "node:assert/strict";
import { existsSync, readdirSync, readFileSync } from "node:fs";
import { dirname, join, relative } from "node:path";
import { fileURLToPath } from "node:url";

const root = dirname(dirname(fileURLToPath(import.meta.url)));
const formerService = ["dashboard", "web"].join("-");
const formerAppPath = ["apps", formerService].join("/");

const forbiddenPaths = [
  formerAppPath,
  [".env.prod", `${formerService}.example`].join("."),
  ["scripts", `smoke-${formerService.replace("-web", "")}.mjs`].join("/"),
  "docs/dashboard-pages-design-brief.md",
  "docs/ui-ux-dashboard-redesign-prompt.md",
];

for (const item of forbiddenPaths) {
  assert.equal(existsSync(join(root, item)), false, `${item} must not exist in backend repo`);
}

const ignoredDirs = new Set([".git", "node_modules", ".next", "dist", "build", "coverage", ".venv", "__pycache__"]);
const scannedExtensions = new Set([".md", ".mjs", ".js", ".json", ".yml", ".yaml", ".ps1", ".go", ".py", ".example", ".toml"]);
const violations = [];
const forbiddenContent = new RegExp(
  [
    `\\bapps/${formerService}\\b`,
    `\\bapps\\\\${formerService}\\b`,
    `\\b${formerService}\\b`,
    `\\.env\\.prod\\.${formerService}`,
    `smoke-${formerService.replace("-web", "")}`,
  ].join("|"),
);

function extensionOf(fileName) {
  if (fileName.endsWith(".example")) return ".example";
  const dot = fileName.lastIndexOf(".");
  return dot === -1 ? "" : fileName.slice(dot);
}

function walk(dir) {
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    if (entry.isDirectory()) {
      if (!ignoredDirs.has(entry.name)) walk(join(dir, entry.name));
      continue;
    }
    if (!entry.isFile() || !scannedExtensions.has(extensionOf(entry.name))) continue;
    const file = join(dir, entry.name);
    const rel = relative(root, file).replaceAll("\\", "/");
    const text = readFileSync(file, "utf8");
    if (forbiddenContent.test(text)) {
      violations.push(rel);
    }
  }
}

walk(root);

assert.deepEqual(violations, [], `frontend-only references remain:\n${violations.join("\n")}`);
console.log("Backend standalone verification passed.");
