import { describe, it, expect, afterEach } from "vitest";
import { execFileSync } from "node:child_process";
import { mkdtempSync, mkdirSync, writeFileSync, readFileSync, rmSync } from "node:fs";
import { join, dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { tmpdir } from "node:os";

// Drives the real binary directly (not tui.js) so it can git-init the temp dir
// and open it as a folder — the ]c/[c hunk-nav bindings need a real repo with
// changes so that the git gutter (LineChanges) is populated.
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

function git(cwd, args) {
  execFileSync("git", args, { cwd, stdio: "pipe" });
}

// Sets up a git repo with a 20-line file, commits, then changes lines 5 and 15
// so there are two hunks. Runs the given --exec steps and returns the screenshot.
function runInRepo(steps) {
  dir = mkdtempSync(join(tmpdir(), "ttt-vimdiff-"));
  const cfg = join(dir, "config");
  mkdirSync(cfg, { recursive: true });
  writeFileSync(join(cfg, "settings.json"), JSON.stringify({ version: 1 }));

  git(dir, ["init", "-q"]);
  git(dir, ["config", "user.email", "t@t.co"]);
  git(dir, ["config", "user.name", "t"]);
  const file = join(dir, "f.txt");
  const base = Array.from({ length: 20 }, (_, i) => `line ${String(i + 1).padStart(2, "0")}`);
  writeFileSync(file, base.join("\n") + "\n");
  git(dir, ["add", "f.txt"]);
  git(dir, ["commit", "-qm", "init"]);
  const changed = base.slice();
  changed[4] = "line 05 CHANGED";
  changed[14] = "line 15 CHANGED";
  writeFileSync(file, changed.join("\n") + "\n");

  const shot = join(dir, "shot.txt");
  const script = [...steps, `screenshot ${shot}`, "quit"].join(SEP);
  try {
    execFileSync(
      BINARY,
      ["--size", "100x30", "--exec-split-on", SEP, "--exec", script, "--plugin", VIM_PLUGIN, dir, file],
      { encoding: "utf8", timeout: 15000, stdio: "pipe", env: { ...process.env, TTT_CONFIG_DIR: cfg } },
    );
  } catch {
    // non-zero exit from quit is fine
  }
  return readFileSync(shot, "utf8");
}

describe("vim ]c / [c hunk navigation", () => {
  it("]c jumps to the next changed hunk", () => {
    // Wait for the async git gutter to populate, then ]c from the top of file.
    const screen = runInRepo(["wait 1500", "key ] c", "wait 200"]);
    expect(screen).toContain("Ln 5,");
  });

  it("]c twice reaches the second hunk", () => {
    const screen = runInRepo(["wait 1500", "key ] c", "wait 150", "key ] c", "wait 200"]);
    expect(screen).toContain("Ln 15,");
  });
});
