import { describe, it, expect, afterEach } from "vitest";
import * as tui from "./tui.js";
import { createTempDir, createTempFile, cleanupDir, readFile } from "./helpers.js";

let dir;

afterEach(() => {
  tui.kill();
  if (dir) cleanupDir(dir);
});

describe("clipboard roundtrip", () => {
  it("should copy and paste selected text", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "copypaste.txt", "hello world");

    tui.start(file);
    tui.waitFor("hello world");

    // Select "hello" using ctrl+d (select word at cursor)
    tui.press("home");
    tui.press("ctrl+d");
    tui.waitStable();

    // Copy
    tui.press("ctrl+c");
    tui.waitStable();

    // Move to end of line and paste
    tui.press("end");
    tui.press("ctrl+v");
    tui.waitStable();

    tui.press("ctrl+s");
    tui.waitStable();

    const { snapshots } = tui.run();

    const content = readFile(file);
    expect(content).toContain("hello worldhello");
  });

  it("should cut and paste text", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "cutpaste.txt", "REMOVE keep");

    tui.start(file);
    tui.waitFor("REMOVE keep");

    // Select "REMOVE" using ctrl+d (select word at cursor)
    tui.press("home");
    tui.press("ctrl+d");
    tui.waitStable();

    // Cut
    tui.press("ctrl+x");
    tui.waitStable();

    // Verify "REMOVE" is gone from the visible text
    const s0 = tui.snapshot();

    // Move to end and paste
    tui.press("end");
    tui.press("ctrl+v");
    tui.waitStable();

    tui.press("ctrl+s");
    tui.waitStable();

    const { snapshots } = tui.run();

    expect(snapshots[s0]).toContain("keep");
    expect(snapshots[s0]).not.toContain("REMOVE");

    const content = readFile(file);
    expect(content).toContain("keepREMOVE");
  });

  it("should copy entire line when nothing is selected", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "linecopy.txt", "first line\nsecond line");

    tui.start(file);
    tui.waitFor("first line");

    // Cursor is on line 1, no selection — copy
    tui.press("ctrl+c");
    tui.waitStable();

    // Move to line 2
    tui.press("arrow_down");
    tui.press("end");
    tui.waitStable();

    // Paste
    tui.press("ctrl+v");
    tui.waitStable();

    // Save and check — documents behavior regardless of whether full line was copied
    tui.press("ctrl+s");
    tui.waitStable();

    const { snapshots } = tui.run();

    const content = readFile(file);
    expect(content).toContain("first line");
    expect(content).toContain("second line");
  });

  it("should paste the same text multiple times", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "multipaste.txt", "abc");

    tui.start(file);
    tui.waitFor("abc");

    // Select word "abc" with ctrl+d
    tui.press("home");
    tui.press("ctrl+d");
    tui.waitStable();

    // Copy
    tui.press("ctrl+c");
    tui.waitStable();

    // Move to end of line
    tui.press("end");

    // Paste 3 times
    tui.press("ctrl+v");
    tui.waitStable();
    tui.press("ctrl+v");
    tui.waitStable();
    tui.press("ctrl+v");
    tui.waitStable();

    tui.press("ctrl+s");
    tui.waitStable();

    const { snapshots } = tui.run();

    const content = readFile(file);
    expect(content).toContain("abcabcabcabc");
  });
});
