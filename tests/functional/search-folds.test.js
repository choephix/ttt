import { describe, it, expect, afterEach } from "vitest";
import * as tui from "./tui.js";
import { createTempDir, createTempFile, cleanupDir } from "./helpers.js";

let dir;

afterEach(() => {
  tui.kill();
  if (dir) cleanupDir(dir);
  dir = null;
});

const goContent = `package main

func alpha() {
\tsecretAlpha := "hidden"
\tfmt.Println(secretAlpha)
}

func beta() {
\tsecretBeta := "also hidden"
\treturn
}
`;

describe("search interaction with code folding", () => {
  it("should expand the second collapsed fold when search matches inside it", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "main.go", goContent);

    tui.start(file);
    tui.waitFor("secretAlpha");

    // Fold all functions
    tui.pressChord("ctrl+k", "0");
    tui.waitStable();

    // Verify both folds are collapsed
    let snap = tui.snapshot();
    expect(snap).not.toContain("secretAlpha");
    expect(snap).not.toContain("secretBeta");

    // Search for text in the second fold
    tui.press("ctrl+f");
    tui.waitStable();
    tui.type("secretBeta");
    tui.waitStable();
    tui.press("enter");
    tui.waitStable();
    tui.press("escape");
    tui.waitStable();

    // The second fold should have expanded to reveal the match
    snap = tui.snapshot();
    expect(snap).toContain("secretBeta");
  });

  it("should find matches in folded content and show correct match count", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "main.go", goContent);

    tui.start(file);
    tui.waitFor("secretAlpha");

    // Fold all functions
    tui.pressChord("ctrl+k", "0");
    tui.waitStable();

    let snap = tui.snapshot();
    expect(snap).not.toContain("secretAlpha");
    expect(snap).not.toContain("secretBeta");

    // Search for "secret" which appears in both functions
    tui.press("ctrl+f");
    tui.waitStable();
    tui.type("secret");
    tui.waitStable();

    // Should show matches found (2 matches across both folds)
    snap = tui.snapshot();
    expect(snap).toContain("1/2");

    // Navigate to second match
    tui.press("enter");
    tui.waitStable();

    snap = tui.snapshot();
    expect(snap).toContain("2/2");

    // Close search
    tui.press("escape");
    tui.waitStable();

    // Both folds should have expanded since we visited matches in each
    snap = tui.snapshot();
    expect(snap).toContain("secretAlpha");
    expect(snap).toContain("secretBeta");
  });

  it("should allow folding after search and then unfolding to restore content", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "main.go", goContent);

    tui.start(file);
    tui.waitFor("secretAlpha");

    // Search for Println
    tui.press("ctrl+f");
    tui.waitStable();
    tui.type("Println");
    tui.waitStable();

    let snap = tui.snapshot();
    expect(snap).toContain("Println");

    // Fold all while search is open
    tui.pressChord("ctrl+k", "0");
    tui.waitStable();

    // Content should be folded
    snap = tui.snapshot();
    expect(snap).not.toContain("secretAlpha");

    // Close search
    tui.press("escape");
    tui.waitStable();

    // Unfold all
    tui.pressChord("ctrl+k", "9");
    tui.waitStable();

    // All content should be visible again
    snap = tui.snapshot();
    expect(snap).toContain("secretAlpha");
    expect(snap).toContain("secretBeta");
    expect(snap).toContain("Println");
  });
});
