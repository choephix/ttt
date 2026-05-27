import { describe, it, expect, afterEach } from "vitest";
import * as tui from "./tui.js";
import { createTempDir, createTempFile, cleanupDir, readFile } from "./helpers.js";

let dir;

afterEach(() => {
  tui.kill();
  if (dir) cleanupDir(dir);
});

describe("undo and redo", () => {
  it("should undo typed text", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "undo.txt", "Base");

    tui.start(file);
    tui.waitFor("Base");

    tui.press("end");
    tui.type(" Added");
    tui.waitFor("Base Added");

    for (let i = 0; i < 6; i++) tui.press("ctrl+z");
    tui.waitStable();

    const snap = tui.snapshot();
    expect(snap).not.toContain("Added");
    expect(snap).toContain("Base");
  });

  it("should redo after undo", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "redo.txt", "Base");

    tui.start(file);
    tui.waitFor("Base");

    tui.press("end");
    tui.type(" Extra");
    tui.waitFor("Base Extra");

    for (let i = 0; i < 6; i++) tui.press("ctrl+z");
    tui.waitStable();
    expect(tui.snapshot()).not.toContain("Extra");

    for (let i = 0; i < 6; i++) tui.press("ctrl+y");
    tui.waitStable();
    expect(tui.snapshot()).toContain("Base Extra");
  });

  it("should persist undo state through save", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "persist.txt", "First");

    tui.start(file);
    tui.waitFor("First");

    tui.press("end");
    tui.type(" Second");
    tui.waitFor("First Second");

    tui.press("ctrl+s");
    tui.waitStable();
    expect(readFile(file)).toBe("First Second");

    tui.type(" Third");
    tui.waitFor("First Second Third");

    for (let i = 0; i < 6; i++) tui.press("ctrl+z");
    tui.waitStable();

    tui.press("ctrl+s");
    tui.waitStable();
    expect(readFile(file)).toBe("First Second");
  });
});
