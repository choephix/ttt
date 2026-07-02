import { describe, it, expect, afterEach } from "vitest";
import * as tui from "./tui.js";
import { createTempDir, createTempFile, cleanupDir } from "./helpers.js";

let dir;

afterEach(() => {
  tui.kill();
  if (dir) cleanupDir(dir);
});

describe("command palette", () => {
  it("should open command palette with ctrl+p", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "palette.txt", "Palette test");

    tui.start(file);
    tui.waitFor("palette.txt");

    tui.press("ctrl+p");
    tui.waitStable();

    const s0 = tui.snapshot();
    const { snapshots } = tui.run();
    expect(snapshots[s0]).toContain(">");
  });

  it("should execute a command from the palette", () => {
    dir = createTempDir();
    createTempFile(dir, "exec.txt", "Exec test");

    tui.start(dir);
    tui.waitFor("Explore");

    tui.exec("View: Toggle Sidebar");
    tui.waitStable();

    const s0 = tui.snapshot();
    const { snapshots } = tui.run();
    expect(snapshots[s0]).not.toContain("Explore");
  });

  it("should dismiss palette with escape", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "dismiss.txt", "Dismiss test");

    tui.start(file);
    tui.waitFor("dismiss.txt");

    tui.press("ctrl+p");
    tui.waitStable();

    tui.press("escape");
    tui.waitStable();

    const s0 = tui.snapshot();
    const { snapshots } = tui.run();
    expect(snapshots[s0]).toContain("Dismiss test");
  });
});
