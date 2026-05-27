import { describe, it, expect, afterEach } from "vitest";
import * as tui from "./tui.js";
import { createTempDir, cleanupDir, readFile, fileExists } from "./helpers.js";
import { join } from "node:path";

let dir;

afterEach(() => {
  tui.kill();
  if (dir) cleanupDir(dir);
});

describe("save file", () => {
  it("should type text, save to disk via Save As, and verify contents", () => {
    dir = createTempDir();
    const filePath = join(dir, "hello.txt");

    tui.start(filePath);
    tui.waitFor("untitled");

    tui.type("Hello World");
    tui.waitFor("Hello World");

    const snap = tui.snapshot();
    expect(snap).toContain("Hello World");

    tui.press("ctrl+s");
    tui.waitFor("Save As");

    tui.type(filePath);
    tui.press("enter");
    tui.waitStable();

    tui.press("ctrl+q");
    tui.waitStable(2000);

    expect(fileExists(filePath)).toBe(true);
    expect(readFile(filePath)).toBe("Hello World");
  });
});
