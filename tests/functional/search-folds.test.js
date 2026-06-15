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

});
