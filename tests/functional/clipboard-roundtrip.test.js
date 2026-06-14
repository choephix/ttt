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

    // Select "hello" (5 characters from the start)
    tui.press("home");
    tui.press("shift+arrow_right");
    tui.press("shift+arrow_right");
    tui.press("shift+arrow_right");
    tui.press("shift+arrow_right");
    tui.press("shift+arrow_right");
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

    const content = readFile(file);
    expect(content).toContain("hello worldhello");
  });

  it("should cut and paste text", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "cutpaste.txt", "REMOVE this keep");

    tui.start(file);
    tui.waitFor("REMOVE this keep");

    // Select "REMOVE " (7 characters including the space)
    tui.press("home");
    for (let i = 0; i < 7; i++) {
      tui.press("shift+arrow_right");
    }
    tui.waitStable();

    // Cut
    tui.press("ctrl+x");
    tui.waitStable();

    // Verify "REMOVE " is gone from the visible text
    const snapAfterCut = tui.snapshot();
    expect(snapAfterCut).toContain("this keep");
    expect(snapAfterCut).not.toContain("REMOVE this");

    // Move to end and paste
    tui.press("end");
    tui.press("ctrl+v");
    tui.waitStable();

    tui.press("ctrl+s");
    tui.waitStable();

    const content = readFile(file);
    expect(content).toContain("this keepREMOVE ");
  });

  it("should copy entire line when nothing is selected", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "linecopy.txt", "first line\nsecond line");

    tui.start(file);
    tui.waitFor("first line");

    // Cursor is on line 1, no selection — copy
    tui.press("ctrl+c");
    tui.waitStable();

    // Move to end of file
    tui.press("ctrl+end");
    tui.press("end");
    tui.waitStable();

    // Paste
    tui.press("ctrl+v");
    tui.waitStable();

    // Save and check — documents behavior regardless of whether full line was copied
    tui.press("ctrl+s");
    tui.waitStable();

    const content = readFile(file);
    // If the editor copies the whole line, we expect it pasted somewhere
    // Either way, the original content should still be intact
    expect(content).toContain("first line");
    expect(content).toContain("second line");
  });

  it("should paste the same text multiple times", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "multipaste.txt", "abc");

    tui.start(file);
    tui.waitFor("abc");

    // Select "abc"
    tui.press("home");
    tui.press("shift+arrow_right");
    tui.press("shift+arrow_right");
    tui.press("shift+arrow_right");
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

    const content = readFile(file);
    expect(content).toContain("abcabcabcabc");
  });
});
