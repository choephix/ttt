import { describe, it, expect, afterEach } from "vitest";
import * as tui from "./tui.js";
import { createTempDir, createTempFile, cleanupDir } from "./helpers.js";

let dir;

afterEach(() => {
  tui.kill();
  if (dir) cleanupDir(dir);
});

const goContent = `package main

func main() {
\tfmt.Println("hello")
\tfmt.Println("world")
}

func other() {
\treturn
}
`;

describe("code folding", () => {
  it("should collapse and expand a fold with command", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "main.go", goContent);

    tui.start(file);
    tui.waitFor("hello");

    tui.press("ctrl+g");
    tui.waitStable();
    tui.type("3");
    tui.press("enter");
    tui.waitStable();

    tui.exec("Fold: Toggle Fold");
    tui.waitStable();

    let snap = tui.snapshot();
    expect(snap).not.toContain("hello");
    expect(snap).toContain("⋯");

    tui.exec("Fold: Toggle Fold");
    tui.waitStable();

    snap = tui.snapshot();
    expect(snap).toContain("hello");
  });

  it("should fold all and unfold all", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "main.go", goContent);

    tui.start(file);
    tui.waitFor("hello");

    tui.pressChord("ctrl+k", "0");
    tui.waitStable();

    let snap = tui.snapshot();
    expect(snap).not.toContain("hello");
    expect(snap).not.toContain("return");

    tui.pressChord("ctrl+k", "9");
    tui.waitStable();

    snap = tui.snapshot();
    expect(snap).toContain("hello");
    expect(snap).toContain("return");
  });

  it("should show collapsed chevron on folded line", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "main.go", goContent);

    tui.start(file);
    tui.waitFor("hello");

    tui.press("ctrl+g");
    tui.waitStable();
    tui.type("3");
    tui.press("enter");
    tui.waitStable();

    tui.exec("Fold: Toggle Fold");
    tui.waitStable();

    const snap = tui.snapshot();
    expect(snap).toContain("▶");
  });

  it("should expand fold when search finds match inside collapsed section", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "main.go", goContent);

    tui.start(file);
    tui.waitFor("hello");

    // Collapse all folds using keybinding
    tui.pressChord("ctrl+k", "0");
    tui.waitStable();

    let snap = tui.snapshot();
    expect(snap).not.toContain("hello");
    expect(snap).not.toContain("return");

    // Search for text inside collapsed fold
    tui.press("ctrl+f");
    tui.waitStable();
    tui.type("hello");
    tui.waitStable();
    tui.press("enter");
    tui.waitStable();
    tui.press("escape");
    tui.waitStable();

    snap = tui.snapshot();
    expect(snap).toContain("hello");
  });

  it("should use keybinding ctrl+k [ to toggle fold", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "main.go", goContent);

    tui.start(file);
    tui.waitFor("hello");

    tui.press("ctrl+g");
    tui.waitStable();
    tui.type("3");
    tui.press("enter");
    tui.waitStable();

    tui.pressChord("ctrl+k", "[");
    tui.waitStable();

    const snap = tui.snapshot();
    expect(snap).not.toContain("hello");
  });
});
