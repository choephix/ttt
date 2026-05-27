import { describe, it, expect, afterEach } from "vitest";
import * as tui from "./tui.js";
import { createTempDir, createTempFile, cleanupDir, readFile } from "./helpers.js";

let dir;

afterEach(() => {
  tui.kill();
  if (dir) cleanupDir(dir);
});

describe("line manipulation", () => {
  it("should delete a line with ctrl+k k chord", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "lines.txt", "Line 1\nLine 2\nLine 3");

    tui.start(file);
    tui.waitFor("Line 1");

    // Move to line 2
    tui.press("arrow_down");
    tui.waitStable();

    // Delete line (ctrl+k k chord)
    tui.pressChord("ctrl+k", "k");
    tui.waitStable();

    const snap = tui.snapshot();
    expect(snap).toContain("Line 1");
    expect(snap).toContain("Line 3");
    expect(snap).not.toContain("Line 2");

    tui.press("ctrl+s");
    tui.waitStable();

    const content = readFile(file);
    expect(content).not.toContain("Line 2");
    expect(content).toContain("Line 1");
    expect(content).toContain("Line 3");
  });

  it("should undo line deletion with ctrl+z", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "undoline.txt", "Alpha\nBeta\nGamma");

    tui.start(file);
    tui.waitFor("Alpha");

    tui.press("arrow_down");
    tui.waitStable();

    tui.pressChord("ctrl+k", "k");
    tui.waitStable();

    expect(tui.snapshot()).not.toContain("Beta");

    tui.press("ctrl+z");
    tui.waitStable();

    expect(tui.snapshot()).toContain("Beta");
  });
});
