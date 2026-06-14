import { describe, it, expect, afterEach } from "vitest";
import * as tui from "./tui.js";
import { createTempDir, createMultiLineFile, cleanupDir } from "./helpers.js";

let dir;

afterEach(() => {
  tui.kill();
  if (dir) cleanupDir(dir);
});

describe("large file scroll stability", () => {
  it("should scroll to bottom with ctrl+g to last line", () => {
    dir = createTempDir();
    const file = createMultiLineFile(dir, "big.txt", 500);

    tui.start(file);
    tui.waitFor("big.txt");

    // Jump to the last line using go-to-line
    tui.press("ctrl+g");
    tui.waitStable();
    tui.type("500");
    tui.press("enter");
    tui.waitStable();

    const snap = tui.snapshot();
    expect(snap).toContain("Line 500");
    expect(snap).toContain("500");
    expect(snap).toContain("Ln 500");
  });

  it("should scroll to top with ctrl+g to line 1", () => {
    dir = createTempDir();
    const file = createMultiLineFile(dir, "big2.txt", 500);

    tui.start(file);
    tui.waitFor("big2.txt");

    // Jump to bottom first
    tui.press("ctrl+g");
    tui.waitStable();
    tui.type("500");
    tui.press("enter");
    tui.waitStable();
    tui.waitFor("Line 500");

    // Now jump back to top
    tui.press("ctrl+g");
    tui.waitStable();
    tui.type("1");
    tui.press("enter");
    tui.waitStable();

    const snap = tui.snapshot();
    expect(snap).toContain("Line 1");
    expect(snap).toContain("Ln 1");
  });

  it("should go to a middle line with ctrl+g", () => {
    dir = createTempDir();
    const file = createMultiLineFile(dir, "big3.txt", 500);

    tui.start(file);
    tui.waitFor("big3.txt");

    tui.press("ctrl+g");
    tui.waitStable();
    tui.type("250");
    tui.press("enter");
    tui.waitStable();

    const snap = tui.snapshot();
    expect(snap).toContain("Line 250");
    expect(snap).toContain("Ln 250");
  });

  it("should page down and page up through a large file", () => {
    dir = createTempDir();
    const file = createMultiLineFile(dir, "big4.txt", 500);

    tui.start(file);
    tui.waitFor("big4.txt");

    // Verify we start at the top
    const snapBefore = tui.snapshot();
    expect(snapBefore).toContain("Ln 1");

    // Press page down to scroll away from the top
    tui.press("page_down");
    tui.waitStable(200);
    tui.press("page_down");
    tui.waitStable(200);
    tui.press("page_down");
    tui.waitStable(200);

    // Cursor should have moved past line 1
    const snapMiddle = tui.snapshot();
    expect(snapMiddle).not.toContain("Ln 1");

    // Press page up the same number of times to return near the top
    tui.press("page_up");
    tui.waitStable(200);
    tui.press("page_up");
    tui.waitStable(200);
    tui.press("page_up");
    tui.waitStable(200);

    // Should be back at line 1
    const snapAfter = tui.snapshot();
    expect(snapAfter).toContain("Ln 1");
  });
});
