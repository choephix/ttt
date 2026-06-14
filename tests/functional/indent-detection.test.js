import { describe, it, expect, afterEach } from "vitest";
import * as tui from "./tui.js";
import { createTempDir, createTempFile, cleanupDir } from "./helpers.js";

let dir;

afterEach(() => {
  tui.kill();
  if (dir) cleanupDir(dir);
});

describe("indent detection", () => {
  it("should detect 2-space indentation in mjs file", () => {
    dir = createTempDir();
    const content = [
      "import { defineConfig } from 'astro/config';",
      "",
      "export default defineConfig({",
      "  site: 'https://example.com',",
      "  integrations: [",
      "    starlight({",
      "      title: 'TTT',",
      "    }),",
      "  ],",
      "});",
    ].join("\n");
    const file = createTempFile(dir, "astro.config.mjs", content);

    tui.start(file);
    tui.waitFor("astro.config.mjs");
    tui.waitStable();

    const snap = tui.snapshot();
    expect(snap).toContain("Spaces: 2");
  });

  it("should detect tab indentation in go file", () => {
    dir = createTempDir();
    const content = [
      "package main",
      "",
      "func main() {",
      "\tfmt.Println()",
      "\tif true {",
      "\t\treturn",
      "\t}",
      "}",
    ].join("\n");
    const file = createTempFile(dir, "main.go", content);

    tui.start(file);
    tui.waitFor("main.go");
    tui.waitStable();

    const snap = tui.snapshot();
    expect(snap).toContain("Tab Size:");
  });

  it("should respect editorconfig over detection", () => {
    dir = createTempDir();
    createTempFile(dir, ".editorconfig", "root = true\n\n[*]\nindent_style = tab\nindent_size = 4\n");
    const content = [
      "export default {",
      "  foo: 'bar',",
      "  baz: 'qux',",
      "};",
    ].join("\n");
    const file = createTempFile(dir, "config.mjs", content);

    tui.start(file);
    tui.waitFor("config.mjs");
    tui.waitStable();

    const snap = tui.snapshot();
    expect(snap).toContain("Tab Size: 4");
  });

  it("should detect 4-space indentation in python file", () => {
    dir = createTempDir();
    const content = [
      "def main():",
      "    print('hello')",
      "    if True:",
      "        return",
    ].join("\n");
    const file = createTempFile(dir, "app.py", content);

    tui.start(file);
    tui.waitFor("app.py");
    tui.waitStable();
    // Dismiss LSP notification if present
    tui.press("Escape");
    tui.waitStable();

    const snap = tui.snapshot();
    expect(snap).toContain("Spaces: 4");
  });
});
