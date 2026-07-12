// Repro test for confirmed bug from audit/2026-07-12-ux-bug-audit.md (branch audit/bug-hunt).
// Asserts the CORRECT behavior with `it.fails` — passes while the bug
// exists, goes red when fixed. Remove `.fails` + audit entry when fixing.
import { describe, it, expect, afterEach } from "vitest";
import { execFileSync } from "node:child_process";
import { writeFileSync, mkdirSync } from "node:fs";
import { join } from "node:path";
import * as tui from "./tui.js";
import { createTempDir, cleanupDir } from "./helpers.js";

let dir;

afterEach(() => {
  tui.kill();
  if (dir) cleanupDir(dir);
});

describe("BUG-044: git branch indicator missing when the opened file is below the repo root", () => {
  it.fails("status bar shows the branch for a file in a repo subdirectory", () => {
    dir = createTempDir();
    // Make `dir` a git repo with a subdirectory and a distinctively-named
    // branch so the assertion can't match incidental "main" text elsewhere.
    const git = (...args) =>
      execFileSync("git", ["-C", dir, ...args], { stdio: "ignore" });
    git("init", "-q", "-b", "auditbranch");
    git("config", "user.email", "a@a.com");
    git("config", "user.name", "a");
    mkdirSync(join(dir, "sub"));
    const subfile = join(dir, "sub", "f.txt");
    writeFileSync(subfile, "content\n");
    git("add", "-A");
    git("commit", "-qm", "init");

    // Open the subdir file by absolute path → workspace folder becomes the
    // file's parent (dir/sub), which has no .git of its own.
    tui.start(subfile);
    tui.waitFor("content");
    const s = tui.snapshot();
    const { snapshots } = tui.run();

    // Correct: isGitRepo should walk up and find dir/.git (as the Changes
    // panel does via `git rev-parse --show-toplevel`), so the branch shows.
    // Buggy: no walk-up, so the status bar has no branch segment.
    expect(snapshots[s]).toContain("auditbranch");
  });
});
