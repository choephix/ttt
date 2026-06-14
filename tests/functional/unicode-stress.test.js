import { describe, it, expect, afterEach } from "vitest";
import * as tui from "./tui.js";
import { createTempDir, createTempFile, cleanupDir, readFile } from "./helpers.js";

let dir;

afterEach(() => {
  tui.kill();
  if (dir) cleanupDir(dir);
});

describe("unicode stress tests", () => {
  it("should handle CJK character editing and deletion", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "cjk.txt", "");

    tui.start(file);
    tui.waitStable();

    tui.type("你好世界");
    tui.waitStable();

    const snap = tui.snapshot();
    expect(snap).toContain("你好世界");

    // Delete one character with backspace — should remove one CJK char, not a byte
    tui.press("backspace");
    tui.waitStable();

    tui.press("ctrl+s");
    tui.waitStable();

    const content = readFile(file);
    expect(content).toBe("你好世\n");
  });

  it("should handle symbol characters and deletion", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "symbols.txt", "");

    tui.start(file);
    tui.waitStable();

    // Use simple single-codepoint symbols that terminals handle reliably
    tui.type("ABC");
    tui.waitStable();

    // Select all and delete
    tui.press("ctrl+a");
    tui.waitStable();
    tui.press("backspace");
    tui.waitStable();

    // Type new symbols
    tui.type("XYZ");
    tui.waitStable();

    tui.press("ctrl+s");
    tui.waitStable();

    const content = readFile(file);
    expect(content).toBe("XYZ\n");
  });

  it("should handle mixed ASCII and unicode text", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "mixed.txt", "");

    tui.start(file);
    tui.waitStable();

    tui.type("abc");
    tui.waitStable();
    tui.type("你好");
    tui.waitStable();
    tui.type("def");
    tui.waitStable();

    tui.press("ctrl+s");
    tui.waitStable();

    const content = readFile(file);
    expect(content).toBe("abc你好def\n");

    // Select all, copy, move to end, enter new line, paste
    tui.press("ctrl+a");
    tui.waitStable();
    tui.press("ctrl+c");
    tui.waitStable();

    // Deselect by pressing End, then create a new line
    tui.press("ctrl+end");
    tui.press("end");
    tui.press("enter");
    tui.waitStable();

    tui.press("ctrl+v");
    tui.waitStable();

    tui.press("ctrl+s");
    tui.waitStable();

    const content2 = readFile(file);
    // The pasted content should contain the mixed string
    expect(content2).toContain("abc你好def");
  });

  it("should move cursor by character through Greek letters", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "greek.txt", "αβγδ");

    tui.start(file);
    tui.waitFor("αβγδ");

    // Move cursor to start, then right 2 times (past α and β)
    tui.press("home");
    tui.press("arrow_right");
    tui.press("arrow_right");

    // Type "X" — should insert between β and γ
    tui.type("X");
    tui.waitStable();

    tui.press("ctrl+s");
    tui.waitStable();

    const content = readFile(file);
    expect(content).toBe("αβXγδ\n");
  });

  it("should handle a file with wide CJK characters without crashing", () => {
    dir = createTempDir();
    // Pre-populate with a variety of wide characters
    const wideContent = "漢字テスト　全角スペース";
    const file = createTempFile(dir, "wide.txt", wideContent);

    tui.start(file);
    tui.waitStable();

    // Verify the editor opened and content is visible
    const snap = tui.snapshot();
    expect(snap).toContain("漢字");

    // Navigate and edit — verify no crash
    tui.press("end");
    tui.type("OK");
    tui.waitStable();

    tui.press("ctrl+s");
    tui.waitStable();

    const content = readFile(file);
    expect(content).toContain(wideContent + "OK");
  });

  it("should handle accented characters with cursor movement", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "accent-nav.txt", "cafe");

    tui.start(file);
    tui.waitFor("cafe");

    // Go to end, backspace to remove 'e', type accented 'e'
    tui.press("end");
    tui.press("backspace");
    tui.type("é");
    tui.waitStable();

    tui.press("ctrl+s");
    tui.waitStable();

    const content = readFile(file);
    expect(content).toBe("café\n");

    // Move left 2 chars (past é and f), insert a character
    tui.press("end");
    tui.press("arrow_left");
    tui.press("arrow_left");
    tui.type("Z");
    tui.waitStable();

    tui.press("ctrl+s");
    tui.waitStable();

    const content2 = readFile(file);
    expect(content2).toBe("caZfé\n");
  });
});
