import { describe, it, expect, afterEach } from "vitest";
import { join } from "node:path";
import * as tui from "./tui.js";
import { createTempDir, createTempFile, cleanupDir, readFile, fileExists } from "./helpers.js";

let dir;

afterEach(() => {
  tui.kill();
  if (dir) cleanupDir(dir);
});

describe("save behavior edge cases", () => {
  it("save on clean file does not alter content", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "clean.txt", "original content\n");

    tui.start(file);
    tui.waitFor("original content");
    tui.waitStable();

    // Save without making any edits
    tui.press("ctrl+s");
    tui.waitStable(500);

    // Content must remain identical
    const content = readFile(file);
    expect(content).toBe("original content\n");
  });

  it("save adds final newline to file missing one", () => {
    dir = createTempDir();
    // Write file without trailing newline
    const file = createTempFile(dir, "noeol.txt", "no newline here");

    tui.start(file);
    tui.waitFor("no newline here");

    // Make a small edit then save to ensure the buffer is dirty and a write occurs
    tui.press("end");
    tui.type("!");
    tui.waitFor("no newline here!");

    tui.press("ctrl+s");
    tui.waitStable();

    const content = readFile(file);
    expect(content).toBe("no newline here!\n");
  });

  it("dirty indicator clears after save", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "dirty-clear.txt", "Clean start");

    tui.start(file);
    tui.waitFor("dirty-clear.txt");

    // Type something to make the buffer dirty
    tui.type("x");
    tui.waitStable();

    const dirtySnap = tui.snapshot();
    expect(dirtySnap).toContain("●"); // dirty dot

    // Save the file
    tui.press("ctrl+s");
    tui.waitStable(500);

    const savedSnap = tui.snapshot();
    expect(savedSnap).not.toContain("●"); // dirty dot should be gone
  });

  it("save on new buffer shows Save As dialog", () => {
    dir = createTempDir();
    const newFilePath = join(dir, "created.txt");

    // Start ttt with a non-existent file path to get an untitled buffer
    tui.start(newFilePath);
    tui.waitFor("untitled");

    tui.type("new content here");
    tui.waitFor("new content here");

    // Ctrl+S on an untitled buffer should show Save As
    tui.press("ctrl+s");
    tui.waitFor("Save As");

    const snap = tui.snapshot();
    expect(snap).toContain("Save As");

    // Complete the save to verify it works
    tui.type(newFilePath);
    tui.press("enter");
    tui.waitStable();

    expect(fileExists(newFilePath)).toBe(true);
    expect(readFile(newFilePath)).toContain("new content here");
  });
});
