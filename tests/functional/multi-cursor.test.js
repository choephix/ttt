import { describe, it, expect, afterEach } from "vitest";
import * as tui from "./tui.js";
import { createTempDir, createTempFile, cleanupDir, readFile } from "./helpers.js";

let dir;

afterEach(() => {
  tui.kill();
  if (dir) cleanupDir(dir);
});

describe("multi-cursor", () => {
  it("should select next occurrence with ctrl+d and type at all cursors", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "ctrld.txt", "foo bar foo baz foo");

    tui.start(file);
    tui.waitFor("foo bar foo");

    tui.press("home");
    tui.press("ctrl+d");
    tui.waitStable();
    tui.press("ctrl+d");
    tui.waitStable();
    tui.press("ctrl+d");
    tui.waitStable();

    tui.type("qux");
    tui.waitStable();

    tui.press("ctrl+s");
    tui.waitStable();

    const content = readFile(file);
    expect(content).toBe("qux bar qux baz qux\n");
  });

  it("should select all occurrences with command palette and replace", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "selall.txt", "cat dog cat bird cat");

    tui.start(file);
    tui.waitFor("cat dog");

    tui.press("home");
    tui.press("ctrl+d");
    tui.waitStable();

    tui.exec("Select All Occurrences");
    tui.waitStable();

    tui.type("pet");
    tui.waitStable();

    tui.press("ctrl+s");
    tui.waitStable();

    const content = readFile(file);
    expect(content).toBe("pet dog pet bird pet\n");
  });

  it("should undo multi-cursor edit as single step", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "undo.txt", "aa bb aa");

    tui.start(file);
    tui.waitFor("aa bb aa");

    tui.press("home");
    tui.press("ctrl+d");
    tui.waitStable();
    tui.press("ctrl+d");
    tui.waitStable();

    tui.type("cc");
    tui.waitStable();

    tui.press("ctrl+z");
    tui.waitStable();
    tui.press("ctrl+z");
    tui.waitStable();

    tui.press("ctrl+s");
    tui.waitStable();

    const content = readFile(file);
    expect(content).toContain("aa");
    expect(content).not.toContain("cc");
  });

  it("should collapse multi-cursor on escape", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "escape.txt", "xx yy xx");

    tui.start(file);
    tui.waitFor("xx yy xx");

    tui.press("home");
    tui.press("ctrl+d");
    tui.waitStable();
    tui.press("ctrl+d");
    tui.waitStable();

    tui.press("escape");
    tui.waitStable();

    tui.type("Z");
    tui.waitStable();

    tui.press("ctrl+s");
    tui.waitStable();

    const content = readFile(file);
    // Only one Z should be inserted (single cursor after escape)
    const zCount = (content.match(/Z/g) || []).length;
    expect(zCount).toBe(1);
  });

  it("should backspace at all cursors", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "bksp.txt", "ABC DEF ABC");

    tui.start(file);
    tui.waitFor("ABC DEF ABC");

    tui.press("home");
    tui.press("ctrl+d");
    tui.waitStable();
    tui.press("ctrl+d");
    tui.waitStable();

    // Cursors have "ABC" selected at both positions, type to replace
    tui.press("delete");
    tui.waitStable();

    tui.press("ctrl+s");
    tui.waitStable();

    const content = readFile(file);
    expect(content).toBe(" DEF \n");
  });

  it("should handle multi-cursor on multiline file", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "multiline.txt", "var x = 1;\nvar y = 2;\nvar z = 3;");

    tui.start(file);
    tui.waitFor("var x");

    tui.press("home");
    tui.press("ctrl+d");
    tui.waitStable();
    tui.press("ctrl+d");
    tui.waitStable();
    tui.press("ctrl+d");
    tui.waitStable();

    tui.type("let");
    tui.waitStable();

    tui.press("ctrl+s");
    tui.waitStable();

    const content = readFile(file);
    expect(content).toBe("let x = 1;\nlet y = 2;\nlet z = 3;\n");
  });

  it("should move all cursors with ctrl+right word movement", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "wordmove.txt", "test one\ntest two\ntest three");

    tui.start(file);
    tui.waitFor("test one");

    tui.press("home");
    tui.press("ctrl+d");
    tui.waitStable();
    tui.press("ctrl+d");
    tui.waitStable();
    tui.press("ctrl+d");
    tui.waitStable();

    // All cursors have "test" selected; word-right x2: skip space, then jump to end of number word
    tui.exec("Move Word Right");
    tui.waitStable();
    tui.exec("Move Word Right");
    tui.waitStable();

    tui.type("!");
    tui.waitStable();

    tui.press("ctrl+s");
    tui.waitStable();

    const content = readFile(file);
    expect(content).toBe("test one!\ntest two!\ntest three!\n");
  });
});
