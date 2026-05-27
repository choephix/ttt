import { describe, it, expect, afterEach } from "vitest";
import * as tui from "./tui.js";
import { createTempDir, createTempFile, cleanupDir } from "./helpers.js";

let dir;

afterEach(() => {
  tui.kill();
  if (dir) cleanupDir(dir);
});

describe("sidebar toggle", () => {
  it("should toggle sidebar with ctrl+b", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "side.txt", "Sidebar test");

    tui.start(file);
    tui.waitFor("Explore");

    tui.press("ctrl+b");
    tui.waitStable();

    const snap = tui.snapshot();
    expect(snap).not.toContain("Explore");

    tui.press("ctrl+b");
    tui.waitFor("Explore");

    const snap2 = tui.snapshot();
    expect(snap2).toContain("Explore");
  });

  it("should switch sidebar panels with chord keys", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "panels.txt", "Panel test");

    tui.start(file);
    tui.waitFor("Explore");

    tui.pressChord("ctrl+k", "f");
    tui.waitFor("Find");

    const snap = tui.snapshot();
    expect(snap).toContain("Find");

    tui.pressChord("ctrl+k", "e");
    tui.waitFor("Explore");

    const snap2 = tui.snapshot();
    expect(snap2).toContain("Explore");
  });
});
