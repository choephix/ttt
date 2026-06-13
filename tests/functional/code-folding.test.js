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

    let snap = tui.snapshot();
    expect(snap).not.toContain("hello");
    expect(snap).toContain("⋯");

    tui.exec("Toggle Fold");
    tui.waitStable();

    snap = tui.snapshot();
    expect(snap).toContain("hello");
  });

  it("should fold all and unfold all", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "main.go", goContent);

    tui.start(file);
    tui.waitFor("hello");

    tui.exec("Fold All");
    tui.waitStable();

    let snap = tui.snapshot();
    expect(snap).not.toContain("hello");
    expect(snap).not.toContain("return");

    tui.exec("Unfold All");
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

    tui.exec("Toggle Fold");
    tui.waitStable();

    const snap = tui.snapshot();
    expect(snap).toContain("⏵");
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
