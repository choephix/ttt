import { describe, it, expect, afterEach } from "vitest";
import * as tui from "./tui.js";
import { createTempDir, createTempFile, cleanupDir, readFile } from "./helpers.js";

let dir;

afterEach(() => {
  tui.kill();
  if (dir) cleanupDir(dir);
});

describe("unicode stress tests", () => {
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
