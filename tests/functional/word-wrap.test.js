import { describe, it, expect, afterEach } from "vitest";
import * as tui from "./tui.js";
import { createTempDir, createTempFile, cleanupDir } from "./helpers.js";

let dir;

afterEach(() => {
  tui.kill();
  if (dir) cleanupDir(dir);
});

function ensureWordWrapOff() {
  const snap = tui.snapshot();
  const statusLine = snap.split("\n").find((l) => l.includes("Ln "));
  if (!statusLine) return;
  // Toggle word wrap to make sure it's off, then check if the line got longer or shorter
  // Since we can't directly read the setting, we just toggle it off explicitly
  // by checking the options menu
}

describe("word wrap", () => {
  it("should wrap long lines making off-screen content visible", () => {
    dir = createTempDir();
    // Make line long enough that it definitely exceeds any terminal width
    const longText = "VISIBLE_" + "x".repeat(500) + "_ENDMARK";
    const file = createTempFile(dir, "wrap.txt", longText);

    tui.start(file);
    tui.waitFor("VISIBLE_");
    tui.waitStable();

    // First ensure word wrap is off: toggle twice if needed
    let snap = tui.snapshot();
    if (snap.includes("ENDMARK")) {
      // Word wrap might be on from a previous test, toggle it off
      tui.exec("Toggle Word Wrap");
      tui.waitStable();
      snap = tui.snapshot();
    }

    // Now ENDMARK should not be visible (line is too long)
    expect(snap).not.toContain("ENDMARK");

    // Toggle word wrap on
    tui.exec("Toggle Word Wrap");
    tui.waitStable();

    const after = tui.snapshot();
    expect(after).toContain("ENDMARK");
  });

  it("should show wrapped content on multiple screen rows", () => {
    dir = createTempDir();
    const longText = "HELLO".repeat(200);
    const file = createTempFile(dir, "wrap2.txt", longText);

    tui.start(file);
    tui.waitFor("HELLO");
    tui.waitStable();

    // Ensure word wrap is on
    let snap = tui.snapshot();
    let helloLines = snap.split("\n").filter((l) => l.includes("HELLO")).length;
    if (helloLines <= 1) {
      tui.exec("Toggle Word Wrap");
      tui.waitStable();
    }

    snap = tui.snapshot();
    helloLines = snap.split("\n").filter((l) => l.includes("HELLO")).length;
    expect(helloLines).toBeGreaterThan(1);
  });

  it("should toggle word wrap off and hide off-screen content again", () => {
    dir = createTempDir();
    const longText = "VISIBLE_" + "x".repeat(500) + "_ENDMARK";
    const file = createTempFile(dir, "wrap3.txt", longText);

    tui.start(file);
    tui.waitFor("VISIBLE_");
    tui.waitStable();

    // Ensure word wrap is on first
    let snap = tui.snapshot();
    if (!snap.includes("ENDMARK")) {
      tui.exec("Toggle Word Wrap");
      tui.waitStable();
    }

    snap = tui.snapshot();
    expect(snap).toContain("ENDMARK");

    // Toggle word wrap off
    tui.exec("Toggle Word Wrap");
    tui.waitStable();

    snap = tui.snapshot();
    expect(snap).not.toContain("ENDMARK");
  });
});
