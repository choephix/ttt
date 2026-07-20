import { describe, it, expect, afterEach } from "vitest";
import * as tui from "./tui.js";
import { createTempDir, createTempFile, cleanupDir } from "./helpers.js";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

// Exercise the real shipped plugin, not an inline copy.
const VIM_PLUGIN = join(
  dirname(fileURLToPath(import.meta.url)),
  "..",
  "..",
  "plugins",
  "vim",
  "init.lua",
);

// Four "alpha"s spread over the buffer, so wrapping in both directions is
// observable, plus a word that only differs by a suffix for whole-word tests.
const FIXTURE = "alpha one\nbeta two\nalpha three\nalphabet four\nlast alpha\n";

let dir;

afterEach(() => {
  tui.kill();
  if (dir) cleanupDir(dir);
});

function startVim(content = FIXTURE, name = "test.txt") {
  dir = createTempDir();
  const file = createTempFile(dir, name, content);
  tui.start("--plugin", VIM_PLUGIN, file);
  tui.waitStable(300);
  return file;
}

// Drive a keystroke script. "<name>" is a key press, everything else is typed.
// The pattern is deliberately strict: Vim scripts contain "<<" and "<G", which
// a bare startsWith("<") check would swallow as key names.
const KEY_TOKEN = /^<[a-z0-9+]+>$/i;

function send(keys) {
  for (const k of keys) {
    if (KEY_TOKEN.test(k)) {
      tui.press(k.slice(1, -1));
    } else {
      tui.type(k);
    }
  }
}

function screenAfter(keys, content = FIXTURE) {
  startVim(content);
  send(keys);
  tui.waitStable(300);
  const s = tui.snapshot();
  const { snapshots } = tui.run();
  return snapshots[s];
}

// A snapshot taken mid-script, so an incremental preview can be observed while
// the command line is still open.
function screensAfter(scripts, content = FIXTURE) {
  startVim(content);
  const idx = [];
  for (const keys of scripts) {
    send(keys);
    tui.waitStable(250);
    idx.push(tui.snapshot());
  }
  const { snapshots } = tui.run();
  return idx.map((i) => snapshots[i]);
}

function pos(snapshot) {
  const m = snapshot.match(/Ln (\d+), Col (\d+)/);
  return m ? `${m[1]}:${m[2]}` : "none";
}

function posAfter(keys, content = FIXTURE) {
  return pos(screenAfter(keys, content));
}

// Assert that a numbered gutter line holds exactly `text`.
function line(snapshot, n) {
  const m = snapshot.match(new RegExp(`\\u2502\\s+${n}\\s{2}(.*?)\\s*\\u2502`));
  return m ? m[1] : null;
}

describe("vim search: / and ?", () => {
  it("/ jumps forward to the next match on submit", () => {
    expect(posAfter(["/alpha", "<enter>"])).toBe("3:1");
  });

  it("? jumps backward to the previous match", () => {
    expect(posAfter(["G", "?alpha", "<enter>"])).toBe("5:6");
  });

  it("/ previews the match incrementally, before Enter is pressed", () => {
    const [typing, submitted] = screensAfter([["/three"], ["<enter>"]]);
    // The command line is still open here, and the cursor has already moved.
    expect(typing).toContain("/three");
    expect(pos(typing)).toBe("3:7");
    expect(pos(submitted)).toBe("3:7");
  });

  it("the preview retreats when the pattern stops matching", () => {
    const [matching, broken] = screensAfter([["/alpha"], ["zzz"]]);
    expect(pos(matching)).toBe("3:1");
    // No match for "alphazzz": the cursor falls back to where it started.
    expect(pos(broken)).toBe("1:1");
  });

  it("Esc cancels the search and restores the original cursor", () => {
    expect(posAfter(["2G", "/last", "<esc>"])).toBe("2:1");
  });

  it("a submitted search with no match leaves the cursor alone", () => {
    const s = screenAfter(["/nothinghere", "<enter>"]);
    expect(s).toContain("Pattern not found");
  });

  it("search is case sensitive by default and \\c makes it insensitive", () => {
    expect(posAfter(["/ALPHA", "<enter>"], "alpha\nALPHA\n")).toBe("2:1");
    expect(posAfter(["/ALPHA\\c", "<enter>"], "one\nalpha\n")).toBe("2:1");
  });

  it("matches a regex subset: . * \\d \\+ and character classes", () => {
    expect(posAfter(["/b.ta", "<enter>"])).toBe("2:1");
    expect(posAfter(["/\\d\\+", "<enter>"], "no digits\nx 4711 y\n")).toBe(
      "2:3",
    );
    expect(posAfter(["/[Bb]eta", "<enter>"])).toBe("2:1");
  });

  it("rejects an unsupported regex construct instead of misreading it", () => {
    expect(screenAfter(["/a\\|b", "<enter>"])).toContain("alternation");
  });
});

describe("vim search: n and N", () => {
  it("n walks forward through the matches", () => {
    expect(posAfter(["/alpha", "<enter>", "n"])).toBe("4:1");
  });

  it("N walks back the other way", () => {
    expect(posAfter(["/alpha", "<enter>", "n", "n", "N"])).toBe("4:1");
  });

  it("n wraps past the end of the buffer and says so", () => {
    const s = screenAfter(["/alpha", "<enter>", "n", "n", "n"]);
    expect(s).toContain("search hit BOTTOM, continuing at TOP");
  });

  it("N wraps past the start of the buffer and says so", () => {
    // Line 1 is a match, so the first N lands there and the second wraps.
    const s = screenAfter(["/alpha", "<enter>", "N", "N"]);
    expect(s).toContain("search hit TOP, continuing at BOTTOM");
  });

  it("n after ? keeps searching backward", () => {
    expect(posAfter(["G", "?alpha", "<enter>", "n"])).toBe("4:1");
  });

  it("{count}n skips ahead", () => {
    expect(posAfter(["/alpha", "<enter>", "2n"])).toBe("5:6");
  });

  it("n without a previous search reports no pattern", () => {
    expect(screenAfter(["n"])).toContain("No previous regular expression");
  });
});

describe("vim search: * and #", () => {
  it("* searches forward for the whole word under the cursor", () => {
    // "alphabet" on line 4 must not match: * is whole-word.
    expect(posAfter(["*"])).toBe("3:1");
  });

  it("# searches backward for the whole word under the cursor", () => {
    // G lands on the empty line after the trailing newline; 5G is the last
    // line with text on it.
    expect(posAfter(["5G", "$", "#"])).toBe("3:1");
  });

  it("* is repeatable with n", () => {
    expect(posAfter(["*", "n"])).toBe("5:6");
  });

  it("* skips a substring match", () => {
    // Only line 3 has a standalone "foo"; line 1 and 2 embed it in a word.
    expect(posAfter(["*"], "foo\nfoobar\nbarfoo\nfoo\n")).toBe("4:1");
  });
});

describe("vim search: as an operator target", () => {
  it("d/pattern deletes up to the match, exclusive of it", () => {
    const s = screenAfter(["d/alpha three", "<enter>"]);
    expect(line(s, 1)).toBe("alpha three");
  });

  it("c/pattern changes up to the match and enters insert mode", () => {
    const s = screenAfter(["c/one", "<enter>", "X"]);
    expect(line(s, 1)).toBe("Xone");
    expect(s).toContain("-- INSERT --");
  });

  it("a search landing in column 1 becomes linewise, as in Vim", () => {
    // Vim's exclusive-motion adjustment: the motion started at the first
    // non-blank and ends in column 1, so c/ operates on whole lines.
    const s = screenAfter(["c/beta", "<enter>", "X"]);
    expect(line(s, 1)).toBe("X");
    expect(line(s, 2)).toBe("beta two");
  });

  it("y/pattern yanks up to the match", () => {
    const s = screenAfter(["y/beta", "<enter>", "P"]);
    expect(line(s, 1)).toBe("alpha one");
    expect(line(s, 2)).toBe("alpha one");
  });

  it("a cancelled search leaves the operator pending state clean", () => {
    // Esc drops the operator; the following x must delete a single character.
    const s = screenAfter(["d/alpha", "<esc>", "x"]);
    expect(line(s, 1)).toBe("lpha one");
  });

  it("d/pattern is one undo step", () => {
    const s = screenAfter(["d/alpha three", "<enter>", "u"]);
    expect(line(s, 1)).toBe("alpha one");
    expect(line(s, 2)).toBe("beta two");
    expect(line(s, 3)).toBe("alpha three");
  });

  it(". repeats d/pattern against the next match", () => {
    const s = screenAfter(["d/BB", "<enter>", "."], "x1\nBB\nx2\nBB\nx3\n");
    expect(line(s, 1)).toBe("BB");
    expect(line(s, 2)).toBe("x3");
  });
});
