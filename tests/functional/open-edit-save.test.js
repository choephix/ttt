import { describe, it, expect, afterEach } from "vitest";
import * as tui from "./tui.js";
import { createTempDir, createTempFile, cleanupDir, readFile } from "./helpers.js";

let dir;

afterEach(() => {
  tui.kill();
  if (dir) cleanupDir(dir);
});

describe("open, edit, save", () => {
  it("should open an existing file and display its contents", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "test.txt", "Hello from existing file");

    tui.start(file);
    tui.waitFor("test.txt");
    tui.waitFor("Hello from existing file");

    const snap = tui.snapshot();
    expect(snap).toContain("Hello from existing file");
  });

  it("should edit an existing file and save changes", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "edit.txt", "Original content");

    tui.start(file);
    tui.waitFor("Original content");

    tui.press("escape");
    tui.press("end");
    tui.type(" Modified");
    tui.waitFor("Original content Modified");

    tui.press("ctrl+s");
    tui.waitStable();

    const content = readFile(file);
    expect(content).toBe("Original content Modified");
  });

  it("should show dirty indicator after editing", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "dirty.txt", "Clean content");

    tui.start(file);
    tui.waitFor("dirty.txt");

    tui.type("x");
    tui.waitStable();

    const snap = tui.snapshot();
    expect(snap).toContain("●");
  });
});
