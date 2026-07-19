import { describe, it, expect, afterEach } from "vitest";
import * as tui from "./tui.js";
import { createTempDir, createTempFile, cleanupDir } from "./helpers.js";

let dir;

afterEach(() => {
  tui.kill();
  if (dir) cleanupDir(dir);
});

function openEditor() {
  dir = createTempDir();
  const file = createTempFile(dir, "sample.txt", "alpha\nbeta\ngamma\n");
  tui.start(file);
  tui.waitFor("alpha");
  tui.waitStable();
  return file;
}

describe("settings editor", () => {
  it("opens as an editor tab with every category", () => {
    openEditor();

    tui.exec("Settings: Open Editor Settings");
    tui.waitStable();
    const s0 = tui.snapshot();

    const { snapshots } = tui.run();
    for (const want of [
      "Settings",
      "Editor",
      "Appearance",
      "Completion",
      "Explorer",
      "Terminal",
      "Search",
      "Advanced",
      "Tab size",
      "Word wrap",
      "Apply",
    ]) {
      expect(snapshots[s0]).toContain(want);
    }
  });

  it("shows current values", () => {
    openEditor();

    tui.exec("Settings: Open Editor Settings");
    tui.waitStable();
    const s0 = tui.snapshot();

    const { snapshots } = tui.run();
    // One row per setting: label left, control right.
    expect(snapshots[s0]).toMatch(/Insert spaces\s+\[x\]/);
    expect(snapshots[s0]).toMatch(/Word wrap\s+\[ \]/);
    expect(snapshots[s0]).toContain("Tab size");
    expect(snapshots[s0]).toMatch(/Tab size\s+❯ 4/);
  });

  it("defers edits until Apply, then live-applies them", () => {
    openEditor();

    tui.exec("Settings: Open Editor Settings");
    tui.waitStable();

    // Tab past the tab strip and scroll view onto the first controls, then
    // flip Word wrap.
    tui.press("tab");
    tui.press("tab");
    tui.press("tab");
    tui.press("tab");
    tui.press("space");
    tui.waitStable();
    const beforeApply = tui.snapshot();

    tui.exec("Settings: Apply Changes");
    tui.waitStable();
    const afterApply = tui.snapshot();

    const { snapshots } = tui.run();
    expect(snapshots[beforeApply]).not.toBe(snapshots[afterApply]);
    expect(snapshots[afterApply]).toContain("Settings applied");
  });

  it("keeps a single settings tab when reopened", () => {
    openEditor();

    tui.exec("Settings: Open Editor Settings");
    tui.waitStable();
    tui.exec("Settings: Open Editor Settings");
    tui.waitStable();
    const s0 = tui.snapshot();

    const { snapshots } = tui.run();
    const tabBar = snapshots[s0].split("\n").find((l) => l.includes("Settings x"));
    expect(tabBar).toBeTruthy();
    expect(tabBar.match(/Settings x/g)).toHaveLength(1);
  });

  it("still offers the raw JSON path", () => {
    openEditor();

    tui.exec("Settings: Open settings.json");
    tui.waitStable();
    const s0 = tui.snapshot();

    const { snapshots } = tui.run();
    expect(snapshots[s0]).toContain("settings.json");
  });
});
