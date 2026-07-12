// Repro tests for confirmed bugs from audit.md (branch audit/bug-hunt).
// Each test asserts the CORRECT behavior and is declared with `it.fails`,
// so it passes while the bug exists and goes red the moment the bug is
// fixed — at that point remove the `.fails` marker and the audit entry.
//
// Common root cause (BUG-005..008): the multiExec* keyboard paths keep
// e.Multi.Cursors consistent, but line commands, transforms, paste, and
// undo operate only on the primary cursor/selection and never touch
// e.Multi — leaving stale cursors that corrupt the buffer on the next
// multicursor keystroke.
import { describe, it, expect, afterEach } from "vitest";
import * as tui from "./tui.js";
import { createTempDir, createTempFile, cleanupDir, readFile } from "./helpers.js";

// 4 occurrences of "foo": two on line0, one each on line1/line2
const FOO_LINES = "foo bar foo baz\nfoo qux\nbar foo end\n";

let dir;

afterEach(() => {
  tui.kill();
  if (dir) cleanupDir(dir);
});

describe("BUG-005: line commands under multicursor corrupt the buffer", () => {
  it.fails("typing after Duplicate Line edits at consistent cursor positions", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "dup.txt", FOO_LINES);

    tui.start(file);
    tui.waitFor("foo");

    tui.pressChord("ctrl+k", "l"); // Select All Occurrences (4 cursors)
    tui.exec("Duplicate Line");
    tui.type("Y");
    tui.waitStable();

    tui.press("ctrl+s");
    tui.waitStable();
    tui.run();

    // Minimal correct behavior: cursors shift with the inserted line and
    // typing replaces each selected occurrence. (A fix that instead
    // collapses multicursor on line commands would need this expectation
    // adjusted — the non-negotiable part is no corruption of text that
    // no cursor touched.) Buggy behavior today: "Yar foo baz\nfoo Y\n...".
    expect(readFile(file)).toBe(
      "Y bar Y baz\nfoo bar foo baz\nY qux\nbar Y end\n",
    );
  });
});
