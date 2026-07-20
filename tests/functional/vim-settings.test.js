import { describe, it, expect, afterEach } from "vitest";
import { execFileSync } from "node:child_process";
import { mkdtempSync, mkdirSync, writeFileSync, readFileSync, rmSync } from "node:fs";
import { join, dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { tmpdir } from "node:os";

// This test drives the real binary directly (not via tui.js) because it needs
// to seed a settings.json into the config dir, which the shared tui.js harness
// creates privately per run and does not expose. Same --exec protocol as tui.js.
const __dirname = dirname(fileURLToPath(import.meta.url));
const ROOT = resolve(__dirname, "..", "..");
const BINARY = join(ROOT, "bin", "ttt");
const VIM_PLUGIN = join(ROOT, "plugins", "vim", "init.lua");
const SEP = "\x1f";

let dir;

afterEach(() => {
  if (dir) rmSync(dir, { recursive: true, force: true });
  dir = "";
});

// Boots ttt with the vim plugin, a seeded settings.json, and the given --exec
// steps, then returns the resulting screenshot text.
function runWithSettings(settings, steps) {
  dir = mkdtempSync(join(tmpdir(), "ttt-vimset-"));
  const cfg = join(dir, "config");
  mkdirSync(cfg, { recursive: true });
  writeFileSync(join(cfg, "settings.json"), JSON.stringify(settings));
  const file = join(dir, "test.txt");
  writeFileSync(file, "hello\n");
  const shot = join(dir, "shot.txt");

  const script = [...steps, `screenshot ${shot}`, "quit"].join(SEP);
  try {
    execFileSync(
      BINARY,
      ["--size", "100x30", "--exec-split-on", SEP, "--exec", script, "--plugin", VIM_PLUGIN, file],
      { encoding: "utf8", timeout: 15000, stdio: "pipe", env: { ...process.env, TTT_CONFIG_DIR: cfg } },
    );
  } catch {
    // A non-zero exit (e.g. from quit) is fine.
  }
  return readFileSync(shot, "utf8");
}

describe("vim settings", () => {
  it("vim.enabled=false starts with vim mode off (keys type into the buffer)", () => {
    const screen = runWithSettings({ version: 1, vim: { enabled: false } }, ["wait 600", "key x", "wait 150"]);
    // No modal indicator, and 'x' inserts literally rather than deleting a char.
    expect(screen).not.toContain("-- NORMAL --");
    expect(screen).toContain("xhello");
  });

  it("default (vim.enabled unset) starts in normal mode (load assertion)", () => {
    const screen = runWithSettings({ version: 1 }, ["wait 600", "key x", "wait 150"]);
    // Plugin loaded and active: the indicator is present and 'x' deleted the 'h'.
    expect(screen).toContain("-- NORMAL --");
    expect(screen).toContain("ello");
    expect(screen).not.toContain("xhello");
  });
});
