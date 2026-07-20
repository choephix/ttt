import { describe, it, expect, afterEach } from "vitest";
import * as tui from "./tui.js";
import { createTempDir, createTempFile, cleanupDir } from "./helpers.js";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

// Exercise the real shipped plugin, not an inline copy.
const VIM_PLUGIN = join(dirname(fileURLToPath(import.meta.url)), "..", "..", "plugins", "vim", "init.lua");

const LINES = "one\ntwo\nthree\nfour\nfive\n";
const WORDS = "alpha beta gamma\nsecond line here\nthird word list\n";
const CODE = "foo(a)\nbar(b)\nbaz(c)\n";

let dir;

afterEach(() => {
  tui.kill();
  if (dir) cleanupDir(dir);
});

function startVim(content) {
  dir = createTempDir();
  const file = createTempFile(dir, "test.txt", content);
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

function screenAfter(keys, content = LINES) {
  startVim(content);
  send(keys);
  tui.waitStable();
  const s = tui.snapshot();
  const { snapshots } = tui.run();
  return snapshots[s];
}

// Assert that a numbered gutter line holds exactly `text`.
function line(snapshot, n) {
  const m = snapshot.match(new RegExp(`\\u2502\\s+${n}\\s{2}(.*?)\\s*\\u2502`));
  return m ? m[1] : null;
}

function pos(snapshot) {
  const m = snapshot.match(/Ln (\d+), Col (\d+)/);
  return m ? `${m[1]}:${m[2]}` : "none";
}

describe("vim macros: recording and replay", () => {
  it("q{a} records and @{a} replays", () => {
    const s = screenAfter(["qa", "dd", "q", "@a"]);
    expect(line(s, 1)).toBe("three");
  });

  it("the status bar shows the recording register", () => {
    startVim(LINES);
    send(["qa", "dd"]);
    tui.waitStable();
    const recording = tui.snapshot();
    send(["q"]);
    tui.waitStable();
    const stopped = tui.snapshot();
    const { snapshots } = tui.run();
    expect(snapshots[recording]).toContain("recording @a");
    expect(snapshots[stopped]).not.toContain("recording @a");
  });

  it("{count}@{a} replays the macro count times", () => {
    const s = screenAfter(["qa", "dd", "q", "3@a"]);
    expect(line(s, 1)).toBe("five");
  });

  it("@@ repeats the last played macro", () => {
    const s = screenAfter(["qa", "dd", "q", "@a", "@@"]);
    expect(line(s, 1)).toBe("four");
  });

  it("a macro replays text typed in insert mode", () => {
    const s = screenAfter(["qa", "A", "!", "<esc>", "j", "q", "@a"], WORDS);
    expect(line(s, 1)).toBe("alpha beta gamma!");
    expect(line(s, 2)).toBe("second line here!");
  });

  it("a macro replays an operator with a text object", () => {
    const s = screenAfter(["qa", "f(", "ci(", "X", "<esc>", "j", "0", "q", "@a"], CODE);
    expect(line(s, 1)).toBe("foo(X)");
    expect(line(s, 2)).toBe("bar(X)");
  });

  it("q{A} appends to an existing macro", () => {
    // "a deletes one line; appending a second dd makes @a delete two.
    const s = screenAfter(["qa", "dd", "q", "qA", "dd", "q", "@a"]);
    // Recording itself already deleted two lines, then @a deletes two more.
    expect(line(s, 1)).toBe("five");
  });

  it("an empty register replays as a no-op", () => {
    const s = screenAfter(["@z"]);
    expect(line(s, 1)).toBe("one");
  });

  it("a self-recursive macro is capped instead of hanging", () => {
    // qa records "dd@a", which calls itself. The depth cap has to stop it.
    const many = Array.from({ length: 40 }, (_, i) => `L${String(i + 1).padStart(2, "0")}`).join("\n") + "\n";
    const s = screenAfter(["qa", "dd", "@a", "q", "@a"], many);
    expect(s).toContain("L40");
    expect(s).not.toContain("L01");
    expect(s).toContain("-- NORMAL --");
  });

  it("a macro replay produces one undo step per operation", () => {
    // @a deletes two lines; two undos restore exactly those two.
    const s = screenAfter(["qa", "dd", "q", "2@a", "u", "u"]);
    expect(line(s, 1)).toBe("two");
    expect(line(s, 2)).toBe("three");
  });
});

describe("vim dot repeat: operators", () => {
  it(". repeats a delete over a motion", () => {
    const s = screenAfter(["dw", "j", "0", "."], WORDS);
    expect(line(s, 1)).toBe("beta gamma");
    expect(line(s, 2)).toBe("line here");
  });

  it(". repeats a doubled operator", () => {
    const s = screenAfter(["dd", "."]);
    expect(line(s, 1)).toBe("three");
  });

  it(". repeats an operator over a text object", () => {
    // `ci(` needs the cursor on or inside the parens, in the repeat as well.
    const s = screenAfter(["f(", "ci(", "X", "<esc>", "j", "f(", "."], CODE);
    expect(line(s, 1)).toBe("foo(X)");
    expect(line(s, 2)).toBe("bar(X)");
  });

  it(". repeats an operator over a find motion", () => {
    const s = screenAfter(["dta", "j", "0", "."], "xxabc\nyyabc\n");
    expect(line(s, 1)).toBe("abc");
    expect(line(s, 2)).toBe("abc");
  });

  it("{count}. replaces the original count", () => {
    const s = screenAfter(["dd", "3."]);
    expect(line(s, 1)).toBe("five");
  });

  it(". uses the register the original command used", () => {
    const s = screenAfter(['"add', ".", "G", '"ap']);
    // "a was overwritten by the repeat, so it holds "two".
    expect(line(s, 1)).toBe("three");
    expect(line(s, 5)).toBe("two");
  });

  it(". repeats an operator with a mark target", () => {
    const s = screenAfter(["ma", "3j", "d'a", "."], "L1\nL2\nL3\nL4\nL5\nL6\nL7\nL8\n");
    // d'a removes L1..L4. The mark collapsed onto the start of the deleted
    // span, so the repeat operates on the single line it now points at.
    expect(line(s, 1)).toBe("L6");
  });

  it("a yank is not repeatable and leaves . armed", () => {
    const s = screenAfter(["x", "j", "0", "yy", "."]);
    expect(line(s, 1)).toBe("ne");
    expect(line(s, 2)).toBe("wo");
  });
});

describe("vim dot repeat: single-key edits", () => {
  it(". repeats x with its count", () => {
    const s = screenAfter(["2x", "j", "0", "."]);
    expect(line(s, 1)).toBe("e");
    expect(line(s, 2)).toBe("o");
  });

  it(". repeats r{char}", () => {
    const s = screenAfter(["rZ", "j", "0", "."]);
    expect(line(s, 1)).toBe("Zne");
    expect(line(s, 2)).toBe("Zwo");
  });

  it(". repeats p", () => {
    const s = screenAfter(["yy", "j", "p", "j", "."]);
    expect(line(s, 3)).toBe("one");
    expect(line(s, 4)).toBe("three");
    expect(line(s, 5)).toBe("one");
  });

  it(". repeats J", () => {
    const s = screenAfter(["J", "."]);
    expect(line(s, 1)).toBe("one two three");
  });
});

describe("vim dot repeat: insert mode", () => {
  it(". repeats text typed after i", () => {
    const s = screenAfter(["i", "ab", "<esc>", "j", "0", "."]);
    expect(line(s, 1)).toBe("abone");
    expect(line(s, 2)).toBe("abtwo");
  });

  it(". repeats an o with its typed text", () => {
    const s = screenAfter(["o", "new", "<esc>", "."]);
    expect(line(s, 2)).toBe("new");
    expect(line(s, 3)).toBe("new");
  });

  it(". repeats a change with its typed text", () => {
    const s = screenAfter(["cw", "X", "<esc>", "j", "0", "."], WORDS);
    expect(line(s, 1)).toBe("X beta gamma");
    expect(line(s, 2)).toBe("X line here");
  });

  it(". is one undo step for an insert repeat", () => {
    const s = screenAfter(["i", "ab", "<esc>", "j", "0", ".", "u"]);
    expect(line(s, 1)).toBe("abone");
    expect(line(s, 2)).toBe("two");
  });

  it(". is one undo step for a change", () => {
    const s = screenAfter(["cw", "X", "<esc>", "j", "0", ".", "u"], WORDS);
    expect(line(s, 1)).toBe("X beta gamma");
    expect(line(s, 2)).toBe("second line here");
  });
});

describe("vim counts on insert-entry commands", () => {
  it("3i repeats the typed text three times", () => {
    const s = screenAfter(["3i", "ab", "<esc>"]);
    expect(line(s, 1)).toBe("abababone");
  });

  it("5a repeats the typed text after the cursor", () => {
    const s = screenAfter(["5a", "-", "<esc>"]);
    expect(line(s, 1)).toBe("o-----ne");
  });

  it("3o opens three lines with the same text", () => {
    const s = screenAfter(["3o", "X", "<esc>"]);
    expect(line(s, 2)).toBe("X");
    expect(line(s, 3)).toBe("X");
    expect(line(s, 4)).toBe("X");
    expect(line(s, 5)).toBe("two");
  });

  it("2O opens two lines above", () => {
    const s = screenAfter(["j", "2O", "X", "<esc>"]);
    expect(line(s, 1)).toBe("one");
    expect(line(s, 2)).toBe("X");
    expect(line(s, 3)).toBe("X");
    expect(line(s, 4)).toBe("two");
  });

  it("a counted insert is a single undo step", () => {
    const s = screenAfter(["3i", "ab", "<esc>", "u"]);
    expect(line(s, 1)).toBe("one");
  });

  it("3s deletes three characters but types its text once", () => {
    const s = screenAfter(["3s", "X", "<esc>"], "abcdef\n");
    expect(line(s, 1)).toBe("Xdef");
  });
});

describe("vim replace mode: backspace restores", () => {
  it("backspace puts back the overwritten characters", () => {
    // "xyz" overwrote "abc"; two backspaces put "b" and "c" back.
    const s = screenAfter(["R", "xyz", "<backspace>", "<backspace>", "<esc>"], "abcdef\n");
    expect(line(s, 1)).toBe("xbcdef");
  });

  it("backspace past the start of the session only walks left", () => {
    const s = screenAfter(["ll", "R", "x", "<backspace>", "<backspace>", "<esc>"], "abcdef\n");
    expect(line(s, 1)).toBe("abcdef");
    expect(pos(s)).toBe("1:1");
  });

  it("R is a single undo step", () => {
    const s = screenAfter(["R", "xyz", "<esc>", "u"], "abcdef\n");
    expect(line(s, 1)).toBe("abcdef");
  });

  it(". repeats a replace-mode overtype", () => {
    const s = screenAfter(["R", "XY", "<esc>", "j", "0", "."], "abcdef\nghijkl\n");
    expect(line(s, 1)).toBe("XYcdef");
    expect(line(s, 2)).toBe("XYijkl");
  });
});
