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
    createTempFile(dir, "side.txt", "Sidebar test");

    tui.start(dir);
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

  it("should reset sidebar width after being narrowed below minimum", () => {
    dir = createTempDir();
    createTempFile(dir, "width.txt", "Width reset test");

    tui.start(dir);
    tui.waitFor("Explore");

    // Narrow sidebar to near-zero width using the command palette
    for (let i = 0; i < 25; i++) {
      tui.exec("Decrease Sidebar Width");
      tui.waitStable(50);
    }

    // Toggle sidebar off
    tui.press("ctrl+b");
    tui.waitStable();
    const hidden = tui.snapshot();
    expect(hidden).not.toContain("Explore");

    // Toggle sidebar back on — should reset to usable default width
    tui.press("ctrl+b");
    tui.waitFor("Explore");

    const restored = tui.snapshot();
    expect(restored).toContain("Explore");
  });

  it("should switch sidebar panels with chord keys", () => {
    dir = createTempDir();
    createTempFile(dir, "panels.txt", "Panel test");

    tui.start(dir);
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
