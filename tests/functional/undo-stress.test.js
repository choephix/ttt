import { describe, it, expect, afterEach } from "vitest";
import * as tui from "./tui.js";
import { createTempDir, createTempFile, cleanupDir, readFile } from "./helpers.js";

let dir;

afterEach(() => {
  tui.kill();
  if (dir) cleanupDir(dir);
});

describe("undo/redo stress", () => {
  it("should fully roundtrip diverse edits via undo and redo", () => {
    dir = createTempDir();
    const original = "The quick brown fox";
    const file = createTempFile(dir, "stress.txt", original);

    tui.start(file);
    tui.waitFor("The quick brown fox");

    // Move to end of line and make diverse edits
    tui.press("end");

    // Edit 1-2: type " jumps"
    tui.type(" jumps");
    tui.waitStable();

    // Edit 3: press Enter to create a new line
    tui.press("enter");
    tui.waitStable();

    // Edit 4-5: type "over the"
    tui.type("over the");
    tui.waitStable();

    // Edit 6: press Enter again
    tui.press("enter");
    tui.waitStable();

    // Edit 7-8: type "lazy dog"
    tui.type("lazy dog");
    tui.waitStable();

    // Edit 9: backspace to delete "g"
    tui.press("backspace");
    tui.waitStable();

    // Edit 10: backspace to delete "o"
    tui.press("backspace");
    tui.waitStable();

    // Edit 11-12: type "og!"
    tui.type("og!");
    tui.waitStable();

    // Edit 13: go to beginning of "lazy" line and select the word
    tui.press("home");
    tui.press("ctrl+d");
    tui.waitStable();

    // Edit 14: delete the selection ("lazy")
    tui.press("backspace");
    tui.waitStable();

    // Edit 15: type replacement
    tui.type("LAZY");
    tui.waitStable();

    // Save the fully-edited state
    tui.press("ctrl+s");
    tui.waitStable();
    const editedContent = readFile(file);

    // Undo ALL edits (press ctrl+z many times)
    for (let i = 0; i < 20; i++) {
      tui.press("ctrl+z");
    }
    tui.waitStable();

    // Save and verify we are back to the original
    tui.press("ctrl+s");
    tui.waitStable();
    const restoredContent = readFile(file);
    expect(restoredContent).toBe(original + "\n");

    // Redo ALL edits (press ctrl+y many times)
    for (let i = 0; i < 20; i++) {
      tui.press("ctrl+y");
    }
    tui.waitStable();

    // Save and verify we match the fully-edited state
    tui.press("ctrl+s");
    tui.waitStable();
    const redoneContent = readFile(file);
    expect(redoneContent).toBe(editedContent);
  });

  it("should undo select-all delete to restore multi-line content", () => {
    dir = createTempDir();
    const original = "line one\nline two\nline three";
    const file = createTempFile(dir, "selectall.txt", original);

    tui.start(file);
    tui.waitFor("line one");

    // Navigate to end of last line (line 3)
    tui.press("ctrl+g");
    tui.waitStable();
    tui.type("3");
    tui.press("enter");
    tui.waitStable();
    tui.press("end");
    tui.press("enter");
    tui.type("line four");
    tui.press("enter");
    tui.type("line five");
    tui.waitStable();

    // Select all and delete
    tui.press("ctrl+a");
    tui.waitStable();
    tui.press("backspace");
    tui.waitStable();

    // Verify the buffer is empty (no original lines visible)
    const snapEmpty = tui.snapshot();
    expect(snapEmpty).not.toContain("line one");
    expect(snapEmpty).not.toContain("line five");

    // Undo the delete and the typed text to get back to original
    tui.press("ctrl+z");
    tui.waitStable();

    // After undoing the delete, all content (including typed lines) should be restored
    const snapRestored = tui.snapshot();
    expect(snapRestored).toContain("line one");

    // Keep undoing to remove the typed lines
    for (let i = 0; i < 10; i++) {
      tui.press("ctrl+z");
    }
    tui.waitStable();

    // Save and verify original content is restored
    tui.press("ctrl+s");
    tui.waitStable();
    const content = readFile(file);
    expect(content).toBe(original + "\n");
  });

  it("should discard redo stack when new edits are made after undo", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "interleave.txt", "start");

    tui.start(file);
    tui.waitFor("start");

    tui.press("end");

    // Make 5 edits (each separated by cursor movement to force separate undo groups)
    tui.type(" one");
    tui.waitStable();
    tui.press("arrow_left");
    tui.press("end");
    tui.type(" two");
    tui.waitStable();
    tui.press("arrow_left");
    tui.press("end");
    tui.type(" three");
    tui.waitStable();
    tui.press("arrow_left");
    tui.press("end");
    tui.type(" four");
    tui.waitStable();
    tui.press("arrow_left");
    tui.press("end");
    tui.type(" five");
    tui.waitStable();

    const snapAll = tui.snapshot();
    expect(snapAll).toContain("start one two three four five");

    // Undo 3 times
    tui.press("ctrl+z");
    tui.press("ctrl+z");
    tui.press("ctrl+z");
    tui.waitStable();

    const snapUndo3 = tui.snapshot();
    expect(snapUndo3).toContain("start one two");
    expect(snapUndo3).not.toContain("five");

    // Make 2 new edits (this should discard the redo stack)
    tui.type(" alpha");
    tui.waitStable();
    tui.press("arrow_left");
    tui.press("end");
    tui.type(" beta");
    tui.waitStable();

    const snapNew = tui.snapshot();
    expect(snapNew).toContain("start one two alpha beta");

    // Try to redo -- should do nothing since redo stack was discarded
    tui.press("ctrl+y");
    tui.press("ctrl+y");
    tui.press("ctrl+y");
    tui.waitStable();

    // Verify content is unchanged (redo did nothing)
    tui.press("ctrl+s");
    tui.waitStable();
    const content = readFile(file);
    expect(content).toBe("start one two alpha beta\n");
  });
});
