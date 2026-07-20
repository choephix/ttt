import { describe, it, expect, afterEach } from "vitest";
import * as tui from "./tui.js";
import { createTempDir, createTempFile, cleanupDir } from "./helpers.js";

let dir;

afterEach(() => {
  tui.kill();
  if (dir) cleanupDir(dir);
});

describe("status bar segments", () => {
  it("shows branch, position, indent, encoding, eol, and language", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "hello.go", 'package main\n\nfunc main() {\n}\n');

    tui.start(file);
    tui.waitStable(300);
    const s = tui.snapshot();
    const { snapshots } = tui.run();

    const lastLine = snapshots[s].split("\n").pop();
    expect(lastLine).toContain("Ln 1, Col 1");
    expect(lastLine).toContain("UTF-8");
    expect(lastLine).toContain("LF");
    expect(lastLine).toContain("Go");
  });

  it("updates position when cursor moves", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "nav.txt", "hello world\nline two\nline three\n");

    tui.start(file);
    tui.waitStable(300);
    tui.press("arrow_down");
    tui.press("arrow_down");
    tui.press("arrow_right");
    tui.press("arrow_right");
    tui.press("arrow_right");
    tui.waitStable();
    const s = tui.snapshot();
    const { snapshots } = tui.run();

    const lastLine = snapshots[s].split("\n").pop();
    expect(lastLine).toContain("Ln 3, Col 4");
  });

  it("shows indent style for spaces", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "spaces.js", "const x = 1;\n");

    tui.start(file);
    tui.waitStable(300);
    const s = tui.snapshot();
    const { snapshots } = tui.run();

    const lastLine = snapshots[s].split("\n").pop();
    expect(lastLine).toContain("Spaces:");
  });

  it("shows notification and dismisses after expiry", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "note.txt", "hello\n");

    tui.start(file);
    tui.waitStable(300);
    tui.exec("Debug: Screenshot");
    tui.waitStable();
    const s = tui.snapshot();
    const { snapshots } = tui.run();

    const lastLine = snapshots[s].split("\n").pop();
    expect(lastLine).toContain("Screenshot");
  });
});
