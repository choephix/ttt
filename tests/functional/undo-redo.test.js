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

    // " Added" is one group (space joins next word) → 1 undo
    tui.press("ctrl+z");
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

    // " Extra" is one group → 1 undo
    tui.press("ctrl+z");
    tui.waitStable();
    expect(tui.snapshot()).not.toContain("Extra");

    tui.press("ctrl+y");
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
    expect(readFile(file)).toBe("First Second\n");

    tui.type(" Third");
    tui.waitFor("First Second Third");

    // " Third" is one group → 1 undo
    tui.press("ctrl+z");
    tui.waitStable();

    tui.press("ctrl+s");
    tui.waitStable();
    expect(readFile(file)).toBe("First Second\n");
  });

  it("should undo word by word, not char by char", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "group.txt", "");

    tui.start(file);
    tui.waitStable();

    tui.type("hello world");
    tui.waitFor("hello world");

    // One undo removes " world" (space belongs with next word)
    tui.press("ctrl+z");
    tui.waitStable();
    expect(tui.snapshot()).toContain("hello");
    expect(tui.snapshot()).not.toContain("hello world");

    // Next undo removes "hello"
    tui.press("ctrl+z");
    tui.waitStable();
    expect(tui.snapshot()).not.toContain("hello");
  });

  it("should break undo group on cursor movement", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "cursor.txt", "");

    tui.start(file);
    tui.waitStable();

    tui.type("ab");
    tui.press("arrow_left");
    tui.press("arrow_right");
    tui.type("cd");
    tui.waitFor("abcd");

    // First undo removes "cd" (typed after cursor movement)
    tui.press("ctrl+z");
    tui.waitStable();
    expect(tui.snapshot()).toContain("ab");
    expect(tui.snapshot()).not.toContain("abcd");
  });
});
