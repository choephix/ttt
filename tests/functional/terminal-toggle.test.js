import { describe, it, expect, afterEach } from "vitest";
import * as tui from "./tui.js";
import { createTempDir, createTempFile, cleanupDir } from "./helpers.js";

let dir;

afterEach(() => {
  tui.kill();
  if (dir) cleanupDir(dir);
});

describe("terminal toggle", () => {
  it("should show terminal panel with ctrl+t", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "term.txt", "Editor content");

    tui.start(file);
    tui.waitFor("term.txt");

    tui.press("ctrl+t");
    tui.waitFor("TERMINAL");

    const snap = tui.snapshot();
    expect(snap).toContain("TERMINAL");
  });

  it("should toggle terminal off and on", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "toggle.txt", "Toggle test");

    tui.start(file);
    tui.waitFor("toggle.txt");

    tui.press("ctrl+t");
    tui.waitFor("TERMINAL");

    tui.press("ctrl+t");
    tui.waitStable();

    const snap = tui.snapshot();
    expect(snap).not.toContain("TERMINAL");

    tui.press("ctrl+t");
    tui.waitFor("TERMINAL");

    const snap2 = tui.snapshot();
    expect(snap2).toContain("TERMINAL");
  });
});
