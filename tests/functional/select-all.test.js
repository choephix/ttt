import { describe, it, expect, afterEach } from "vitest";
import * as tui from "./tui.js";
import { createTempDir, createTempFile, cleanupDir, readFile } from "./helpers.js";

let dir;

afterEach(() => {
  tui.kill();
  if (dir) cleanupDir(dir);
});

describe("select all and overwrite", () => {
  it("should select all text and replace with new content", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "selectall.txt", "old content\nthat spans\nmultiple lines");

    tui.start(file);
    tui.waitFor("old content");

    tui.press("ctrl+a");
    tui.waitStable();

    tui.type("replaced");
    tui.waitStable();

    tui.press("ctrl+s");
    tui.waitStable();

    const content = readFile(file);
    expect(content).toBe("replaced");
  });

  it("should undo select-all overwrite to restore original", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "undosel.txt", "preserve this\nand this");

    tui.start(file);
    tui.waitFor("preserve");

    tui.press("ctrl+a");
    tui.waitStable();

    tui.type("x");
    tui.waitStable();

    const snap = tui.snapshot();
    expect(snap).not.toContain("preserve");

    // undo the typed char, then undo the deletion
    tui.press("ctrl+z");
    tui.press("ctrl+z");
    tui.waitStable();

    const snap2 = tui.snapshot();
    expect(snap2).toContain("preserve");
  });
});
