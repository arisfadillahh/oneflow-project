import test from "node:test";
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { join } from "node:path";
import { fileURLToPath } from "node:url";

const dashboardRoot = fileURLToPath(new URL("..", import.meta.url));

test("mobile dashboard nav uses quick actions with an overflow menu", async () => {
  const source = await readFile(join(dashboardRoot, "components", "dashboard", "shell", "AppShell.jsx"), "utf8");

  assert.match(source, /mobilePrimaryNavByRole/, "Mobile nav should define role-based quick actions.");
  assert.match(source, /mobile-more-trigger/, "Mobile nav should expose a dedicated overflow trigger.");
  assert.match(source, /mobile-more-menu/, "Mobile nav overflow items should stay accessible in a menu.");
  assert.match(source, /id:\s*"account"/, "Mobile nav should keep account settings reachable.");
});

test("mobile dashboard nav is a fixed grid instead of a horizontal scroller", async () => {
  const css = await readFile(join(dashboardRoot, "app", "(application)", "globals.css"), "utf8");

  assert.match(css, /\.mobile-nav-list/, "Mobile nav should have a dedicated list container.");
  assert.match(css, /grid-template-columns:\s*repeat\(5,\s*minmax\(0,\s*1fr\)\)/, "Mobile nav should fit five stable slots.");
  assert.match(css, /\.mobile-more-menu/, "Mobile overflow menu should have explicit mobile styling.");
});
