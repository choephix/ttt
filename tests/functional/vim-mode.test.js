import { describe, it, expect, afterEach } from "vitest";
import * as tui from "./tui.js";
import { createTempDir, createTempFile, cleanupDir } from "./helpers.js";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

// Exercise the real shipped plugin, not an inline copy.
const VIM_PLUGIN = join(dirname(fileURLToPath(import.meta.url)), "..", "..", "plugins", "vim", "init.lua");

let dir;

afterEach(() => {
  tui.kill();
  if (dir) cleanupDir(dir);
});

function startVim(content = "hello world\nsecond line\n") {
  dir = createTempDir();
  const file = createTempFile(dir, "test.txt", content);
  tui.start("--plugin", VIM_PLUGIN, file);
  tui.waitStable(300);
  return file;
}

describe("vim mode", () => {
  it("starts in normal mode", () => {
    startVim();
    const s = tui.snapshot();
    const { snapshots } = tui.run();

    expect(snapshots[s]).toContain("-- NORMAL --");
  });

  it("swallows printable keys in normal mode instead of typing them", () => {
    startVim();
    tui.type("jjxx");
    tui.waitStable();
    const s = tui.snapshot();
    const { snapshots } = tui.run();

    expect(snapshots[s]).toContain("hello world");
    expect(snapshots[s]).not.toContain("jjxx");
    expect(snapshots[s]).toContain("-- NORMAL --");
  });

  it("enters insert mode on i and types text", () => {
    startVim();
    tui.type("iABC");
    tui.waitStable();
    const s = tui.snapshot();
    const { snapshots } = tui.run();

    expect(snapshots[s]).toContain("-- INSERT --");
    expect(snapshots[s]).toContain("ABChello world");
    // the `i` that entered insert mode must not itself be typed
    expect(snapshots[s]).not.toContain("iABC");
  });

  it("returns to normal mode on escape", () => {
    startVim();
    tui.type("iABC");
    tui.press("escape");
    tui.waitStable();
    const s = tui.snapshot();
    const { snapshots } = tui.run();

    expect(snapshots[s]).toContain("-- NORMAL --");
    expect(snapshots[s]).toContain("ABChello world");
  });

  // The interceptor now sits above handleChord, so a chord's printable
  // continuation key (the `s` of `ctrl+k s`) must not be eaten by normal mode.
  it("does not break ctrl+k chords", () => {
    startVim();
    tui.pressChord("ctrl+k", "s");
    tui.waitStable(200);
    const s = tui.snapshot();
    const { snapshots } = tui.run();

    expect(snapshots[s].toLowerCase()).toContain("save as");
  });

  it("stops swallowing keys once vim mode is disabled", () => {
    startVim();
    tui.exec("Vim: Disable Vim Mode");
    tui.waitStable();
    tui.type("zzz");
    tui.waitStable();
    const s = tui.snapshot();
    const { snapshots } = tui.run();

    expect(snapshots[s]).toContain("zzz");
    expect(snapshots[s]).not.toContain("-- NORMAL --");
  });
});
