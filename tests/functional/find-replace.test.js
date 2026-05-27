import { describe, it, expect, afterEach } from "vitest";
import * as tui from "./tui.js";
import { createTempDir, createTempFile, cleanupDir, readFile } from "./helpers.js";

let dir;

afterEach(() => {
  tui.kill();
  if (dir) cleanupDir(dir);
});

describe("find and replace", () => {
  it("should find text and show match count", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "find.txt", "foo bar foo baz foo");

    tui.start(file);
    tui.waitFor("foo bar foo baz foo");

    tui.press("ctrl+f");
    tui.waitStable();

    tui.type("foo");
    tui.waitStable();

    const snap = tui.snapshot();
    expect(snap).toContain("1/3");
  });

  it("should navigate between find matches", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "nav.txt", "apple banana apple cherry apple");

    tui.start(file);
    tui.waitFor("apple banana");

    tui.press("ctrl+f");
    tui.waitStable();

    tui.type("apple");
    tui.waitStable();

    const snap1 = tui.snapshot();
    expect(snap1).toContain("1/3");

    tui.press("enter");
    tui.waitStable();

    const snap2 = tui.snapshot();
    expect(snap2).toContain("2/3");
  });

  it("should open find and replace dialog", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "replace.txt", "hello world hello");

    tui.start(file);
    tui.waitFor("hello world hello");

    tui.press("ctrl+r");
    tui.waitStable();

    const snap = tui.snapshot();
    expect(snap).toContain("Replace");
  });

  it("should replace text and save to verify on disk", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "disk.txt", "old value old again");

    tui.start(file);
    tui.waitFor("old value old again");

    tui.press("ctrl+r");
    tui.waitStable();

    tui.type("old");
    tui.waitStable();

    tui.press("tab");
    tui.type("new");
    tui.waitStable();

    // Replace first match
    tui.press("enter");
    tui.waitStable();

    // Replace second match
    tui.press("enter");
    tui.waitStable();

    // Close replace bar and save
    tui.press("escape");
    tui.waitStable();
    tui.press("ctrl+s");
    tui.waitStable(500);

    const content = readFile(file);
    expect(content).toContain("new");
    expect(content).not.toContain("old");
  });
});
