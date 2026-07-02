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

    tui.exec("Toggle Fold");
    tui.waitStable();

    const s0 = tui.snapshot();

    tui.exec("Toggle Fold");
    tui.waitStable();

    const s1 = tui.snapshot();
    const { snapshots } = tui.run();

    expect(snapshots[s0]).not.toContain("hello");
    expect(snapshots[s0]).toContain("⋯");
    expect(snapshots[s1]).toContain("hello");
  });

  it("should fold all and unfold all", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "main.go", goContent);

    tui.start(file);
    tui.waitFor("hello");

    tui.pressChord("ctrl+k", "0");
    tui.waitStable();

    const s0 = tui.snapshot();

    tui.pressChord("ctrl+k", "9");
    tui.waitStable();

    const s1 = tui.snapshot();
    const { snapshots } = tui.run();

    expect(snapshots[s0]).not.toContain("hello");
    expect(snapshots[s0]).not.toContain("return");
    expect(snapshots[s1]).toContain("hello");
    expect(snapshots[s1]).toContain("return");
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

    tui.exec("Toggle Fold");
    tui.waitStable();

    const s0 = tui.snapshot();
    const { snapshots } = tui.run();
    expect(snapshots[s0]).toContain("⋯");
  });

  it("should expand fold when search finds match inside collapsed section", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "main.go", goContent);

    tui.start(file);
    tui.waitFor("hello");

    // Collapse all folds using keybinding
    tui.pressChord("ctrl+k", "0");
    tui.waitStable();

    const s0 = tui.snapshot();

    // Search for text inside collapsed fold
    tui.press("ctrl+f");
    tui.waitStable();
    tui.type("hello");
    tui.waitStable();
    tui.press("enter");
    tui.waitStable();
    tui.press("escape");
    tui.waitStable();

    const s1 = tui.snapshot();
    const { snapshots } = tui.run();

    expect(snapshots[s0]).not.toContain("hello");
    expect(snapshots[s0]).not.toContain("return");
    expect(snapshots[s1]).toContain("hello");
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

    const s0 = tui.snapshot();
    const { snapshots } = tui.run();
    expect(snapshots[s0]).not.toContain("hello");
  });
});
