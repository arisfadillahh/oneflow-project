import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";

const css = readFileSync(new URL("../app/(application)/globals.css", import.meta.url), "utf8");
function luminance(hex) {
  const channels = hex.match(/[a-f0-9]{2}/gi).map(value => parseInt(value, 16) / 255).map(value => value <= 0.04045 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4);
  return channels[0] * 0.2126 + channels[1] * 0.7152 + channels[2] * 0.0722;
}
function ratio(a, b) {
  const values = [luminance(a), luminance(b)].sort((x, y) => y - x);
  return (values[0] + 0.05) / (values[1] + 0.05);
}
for (const scope of ["dashboard-shell", "login-shell"]) {
  test(`${scope} muted dark text meets normal-text contrast`, () => {
    const block = css.slice(css.indexOf(`.${scope}.theme-dark {`)).split("}")[0];
    for (const token of ["gray-400", "gray-500"]) {
      const color = block.match(new RegExp(`--${token}: (#[a-f0-9]{6})`, "i"))[1];
      for (const background of ["#101a2b", "#142238", "#1a2b45"]) {
        assert.ok(ratio(color, background) >= 4.5, `${token} on ${background}: ${ratio(color, background).toFixed(2)}`);
      }
    }
  });
}
test("dark primary button label contrasts against both gradient stops", () => {
  const rule = css.match(/\.dashboard-shell\.theme-dark \.btn-primary\s*\{([^}]+)\}/);
  assert.ok(rule, "explicit dark primary label rule required");
  const color = rule[1].match(/color:\s*(#[a-f0-9]{6})/i)[1];
  for (const stop of ["#38a4ff", "#9b7cff"]) assert.ok(ratio(color, stop) >= 4.5);
});

test("dark active navigation label contrasts against both gradient stops", () => {
  const rule = css.match(/\.dashboard-shell\.theme-dark \.desktop-nav-list \.nav-item\.active\s*\{([^}]+)\}/);
  assert.doesNotMatch(css, /\.dashboard-shell\.theme-dark \.nav-item\.active\s*\{/);
  assert.ok(rule, "explicit dark active navigation label rule required");
  const color = rule[1].match(/color:\s*(#[a-f0-9]{6})/i)[1];
  for (const stop of ["#38a4ff", "#9b7cff"]) assert.ok(ratio(color, stop) >= 4.5);
});
