import { describe, it, expect, afterEach } from "vitest";
import * as tui from "./tui.js";
import { createTempDir, createTempFile, cleanupDir } from "./helpers.js";

let dir;

afterEach(() => {
  tui.kill();
  if (dir) cleanupDir(dir);
});

describe("default settings", () => {
  it("should open default settings via command palette", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "test.txt", "hello");

    tui.start(file);
    tui.waitFor("test.txt");

    tui.exec("Preferences: Open Default Settings");
    tui.waitFor("Default Settings");

    const snap = tui.snapshot();
    expect(snap).toContain("Default Settings");
    expect(snap).toContain("Default Settings Reference");
  });

  it("should show settings sections in the reference", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "test.txt", "hello");

    tui.start(file);
    tui.waitFor("test.txt");

    tui.exec("Preferences: Open Default Settings");
    tui.waitFor("Default Settings");

    const snap = tui.snapshot();
    expect(snap).toContain("editor");
    expect(snap).toContain("tabSize");
  });
});
