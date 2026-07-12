// Repro test for confirmed bug from audit.md (branch audit/bug-hunt).
// Asserts the CORRECT behavior with `it.fails` — passes while the bug
// exists, goes red when fixed. Remove `.fails` + audit entry when fixing.
import { describe, it, expect, afterEach } from "vitest";
import { writeFileSync } from "node:fs";
import { join } from "node:path";
import * as tui from "./tui.js";
import { createTempDir, cleanupDir } from "./helpers.js";

let dir;

afterEach(() => {
  tui.kill();
  if (dir) cleanupDir(dir);
});

describe("BUG-047: global-search navigation ignores the match column (lands at col 0)", () => {
  it.fails("activating a result places the cursor at the match column", () => {
    dir = createTempDir();
    writeFileSync(
      join(dir, "alpha.txt"),
      "line one\nneedle here in alpha\nanother needle line\nlast line no match\n",
    );

    tui.start(dir);
    tui.waitStable(300);
    tui.pressChord("ctrl+k", "f"); // open global search
    tui.type("needle");
    tui.waitStable(700); // debounce + rg
    tui.press("arrow_down");
    tui.press("arrow_down"); // to the "another needle line" match
    tui.press("enter"); // navigate
    tui.waitStable();
    tui.type("Z"); // marker at the cursor's landing column
    tui.waitStable();
    const s = tui.snapshot();
    const { snapshots } = tui.run();

    // "another " is 8 chars, so the match column is 8. Correct: the marker
    // lands at the match → "another Zneedle line". Buggy: NavigateToSearchMatch
    // ignores col and GoToLine forces col 0 → "Zanother needle line".
    expect(snapshots[s]).toContain("another Zneedle line");
  });
});
