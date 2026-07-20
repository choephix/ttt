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

// A rectangular grid, so blockwise column ranges are unambiguous.
const GRID = "abcdef\n" + "ghijkl\n" + "mnopqr\n" + "stuvwx\n";

const WORDS =
  "alpha beta gamma\n" + "second line here\n" + "third line\n" + "fourth\n";

const RAGGED = "abcdefgh\n" + "ij\n" + "klmnop\n";

let dir;

afterEach(() => {
  tui.kill();
  if (dir) cleanupDir(dir);
});

function startVim(content, name = "test.txt") {
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

function screenAfter(keys, content = GRID) {
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

// Lines 1..n of the buffer as an array, for the multi-line assertions.
function lines(snapshot, n) {
  const out = [];
  for (let i = 1; i <= n; i++) out.push(line(snapshot, i));
  return out;
}

describe("vim visual: entering, switching and leaving", () => {
  it("v, V and ctrl+v show their mode indicators", () => {
    expect(screenAfter(["v"])).toContain("-- VISUAL --");
    expect(screenAfter(["V"])).toContain("-- VISUAL LINE --");
    expect(screenAfter(["<ctrl+v>"])).toContain("-- VISUAL BLOCK --");
  });

  it("re-pressing the same key leaves visual mode", () => {
    expect(screenAfter(["v", "v"])).toContain("-- NORMAL --");
    expect(screenAfter(["V", "V"])).toContain("-- NORMAL --");
    expect(screenAfter(["<ctrl+v>", "<ctrl+v>"])).toContain("-- NORMAL --");
  });

  it("pressing a different mode key switches without losing the anchor", () => {
    // v then l then V then d: the anchor is still line 1, so V-d is linewise.
    const s = screenAfter(["v", "l", "V", "d"]);
    expect(line(s, 1)).toBe("ghijkl");
    expect(s).toContain("-- NORMAL --");
  });

  it("Esc leaves visual mode and keeps the cursor where the motion left it", () => {
    const s = screenAfter(["v", "ll", "<escape>"]);
    expect(s).toContain("-- NORMAL --");
    expect(pos(s)).toBe("1:3");
  });

  it("leaving visual mode does not modify the buffer", () => {
    const s = screenAfter(["v", "jl", "<escape>"]);
    expect(lines(s, 4)).toEqual(["abcdef", "ghijkl", "mnopqr", "stuvwx"]);
  });
});

describe("vim visual: selection movement", () => {
  it("charwise d is inclusive of the character under the cursor", () => {
    // v then two l = 3 characters selected.
    expect(line(screenAfter(["v", "ll", "d"]), 1)).toBe("def");
  });

  it("a bare v then d deletes exactly one character", () => {
    expect(line(screenAfter(["v", "d"]), 1)).toBe("bcdef");
  });

  it("word motions extend the selection", () => {
    expect(line(screenAfter(["v", "e", "d"], WORDS), 1)).toBe(" beta gamma");
    // Visual mode is always inclusive, so `vw` also takes the character it lands on.
    expect(line(screenAfter(["v", "w", "d"], WORDS), 1)).toBe("eta gamma");
  });

  it("counts apply to motions inside visual mode", () => {
    expect(line(screenAfter(["v", "3l", "d"]), 1)).toBe("ef");
    expect(line(screenAfter(["v", "2e", "d"], WORDS), 1)).toBe(" gamma");
  });

  it("$ selects to the end of the line inclusively", () => {
    expect(line(screenAfter(["l", "v", "$", "d"]), 1)).toBe("a");
  });

  it("f and t motions extend the selection", () => {
    expect(line(screenAfter(["v", "fd", "d"]), 1)).toBe("ef");
    expect(line(screenAfter(["v", "td", "d"]), 1)).toBe("def");
  });

  it("G and gg extend the selection linewise-ish across the buffer", () => {
    expect(line(screenAfter(["V", "G", "d"]), 1)).toBe("");
  });

  it("a backward selection includes the anchor character", () => {
    // Start on "d" (col 4), select back to "b" (col 2): b, c and d all go.
    expect(line(screenAfter(["3l", "v", "hh", "d"]), 1)).toBe("aef");
  });
});

describe("vim visual: o, O and gv", () => {
  it("o swaps the cursor and the anchor and keeps the same range", () => {
    const s = screenAfter(["v", "jj", "o", "d"]);
    expect(line(s, 1)).toBe("nopqr");
  });

  it("o moves the cursor to the other end", () => {
    expect(pos(screenAfter(["v", "jj", "o"]))).toBe("1:1");
  });

  it("O in blockwise swaps the corners horizontally", () => {
    // Block from 1:1 to 3:3, then O puts the cursor on the left edge and l
    // widens from the *left*, so the block becomes columns 1..4.
    const s = screenAfter(["<ctrl+v>", "jj", "ll", "O", "h", "d"]);
    expect(lines(s, 4)).toEqual(["def", "jkl", "pqr", "stuvwx"]);
  });

  it("gv reselects the previous visual range", () => {
    expect(line(screenAfter(["v", "ll", "<escape>", "gv", "d"]), 1)).toBe(
      "def",
    );
  });

  it("gv reselects a linewise range", () => {
    const s = screenAfter(["V", "j", "<escape>", "gv", "d"]);
    expect(line(s, 1)).toBe("mnopqr");
  });

  it("gv after an operator reselects the range that was operated on", () => {
    // yank two lines, move away, then gv + d removes the same two lines.
    const s = screenAfter(["V", "j", "y", "G", "gv", "d"]);
    expect(line(s, 1)).toBe("mnopqr");
  });
});

describe("vim visual: operators over a charwise selection", () => {
  it("d and x both delete the selection", () => {
    expect(line(screenAfter(["v", "ll", "d"]), 1)).toBe("def");
    expect(line(screenAfter(["v", "ll", "x"]), 1)).toBe("def");
  });

  it("c and s delete the selection and enter insert mode", () => {
    expect(screenAfter(["v", "ll", "c"])).toContain("-- INSERT --");
    const s = screenAfter(["v", "ll", "c", "ZZ", "<escape>"]);
    expect(line(s, 1)).toBe("ZZdef");
    expect(s).toContain("-- NORMAL --");
    const t = screenAfter(["v", "ll", "s", "Q", "<escape>"]);
    expect(line(t, 1)).toBe("Qdef");
  });

  it("y yanks the selection and p pastes it", () => {
    const s = screenAfter(["v", "ll", "y", "$", "p"]);
    expect(line(s, 1)).toBe("abcdefabc");
  });

  it("gu, gU and g~ change case over the selection", () => {
    expect(line(screenAfter(["v", "ll", "gU"]), 1)).toBe("ABCdef");
    expect(line(screenAfter(["v", "ll", "gU", "gv", "gu"]), 1)).toBe("abcdef");
    expect(line(screenAfter(["v", "ll", "g~"]), 1)).toBe("ABCdef");
  });

  it("u, U and ~ are the visual-mode case shorthands", () => {
    expect(line(screenAfter(["v", "ll", "U"]), 1)).toBe("ABCdef");
    expect(line(screenAfter(["v", "ll", "U", "gv", "u"]), 1)).toBe("abcdef");
    expect(line(screenAfter(["v", "ll", "~"]), 1)).toBe("ABCdef");
  });

  it("r replaces every selected character", () => {
    expect(line(screenAfter(["v", "ll", "rZ"]), 1)).toBe("ZZZdef");
  });

  it("r spans lines in a charwise selection", () => {
    const s = screenAfter(["v", "jl", "rZ"]);
    expect(lines(s, 2)).toEqual(["ZZZZZZ", "ZZijkl"]);
  });

  it("p replaces the selection with the register", () => {
    // yank "abc" charwise, then select "ghi" and replace it.
    const s = screenAfter(["v", "ll", "y", "j", "v", "ll", "p"]);
    expect(lines(s, 2)).toEqual(["abcdef", "abcjkl"]);
  });

  it("J joins the selected lines", () => {
    const s = screenAfter(["v", "j", "J"]);
    expect(line(s, 1)).toBe("abcdef ghijkl");
    expect(line(s, 2)).toBe("mnopqr");
  });

  it("> and < shift the lines the selection touches", () => {
    expect(line(screenAfter(["v", "j", ">"]), 1)).toBe("    abcdef");
    expect(line(screenAfter(["v", "j", ">"]), 2)).toBe("    ghijkl");
    expect(line(screenAfter(["v", "j", ">", "gv", "<"]), 1)).toBe("abcdef");
  });

  it("a count repeats the shift", () => {
    expect(line(screenAfter(["v", "2>"]), 1)).toBe("        abcdef");
  });

  it("= reindents the selected lines", () => {
    const s = screenAfter(["V", "j", "="], "function foo() {\n     bar()\n}\n");
    expect(line(s, 2)).toBe("    bar()");
  });
});

describe("vim visual: operators over a linewise selection", () => {
  it("V d removes whole lines", () => {
    expect(lines(screenAfter(["V", "j", "d"]), 2)).toEqual([
      "mnopqr",
      "stuvwx",
    ]);
  });

  it("V c collapses the lines and inserts", () => {
    const s = screenAfter(["V", "j", "c", "new", "<escape>"]);
    expect(lines(s, 2)).toEqual(["new", "mnopqr"]);
  });

  it("V y yanks linewise so p opens a new line", () => {
    const s = screenAfter(["V", "y", "j", "p"]);
    expect(lines(s, 4)).toEqual(["abcdef", "ghijkl", "abcdef", "mnopqr"]);
  });

  it("V p replaces the lines with the register", () => {
    const s = screenAfter(["V", "y", "j", "V", "p"]);
    expect(lines(s, 3)).toEqual(["abcdef", "abcdef", "mnopqr"]);
  });

  it("V gU uppercases whole lines regardless of the cursor column", () => {
    expect(line(screenAfter(["lll", "V", "gU"]), 1)).toBe("ABCDEF");
  });

  it("V r fills the lines", () => {
    expect(lines(screenAfter(["lll", "V", "rZ"]), 2)).toEqual([
      "ZZZZZZ",
      "ghijkl",
    ]);
  });

  it("X, D, S and C act linewise from a charwise selection", () => {
    expect(line(screenAfter(["v", "l", "X"]), 1)).toBe("ghijkl");
    expect(line(screenAfter(["v", "l", "D"]), 1)).toBe("ghijkl");
    expect(line(screenAfter(["v", "l", "S", "hi", "<escape>"]), 1)).toBe("hi");
    expect(line(screenAfter(["v", "l", "C", "hi", "<escape>"]), 1)).toBe("hi");
  });
});

describe("vim visual: text objects", () => {
  it("viw selects the word under the cursor", () => {
    expect(line(screenAfter(["viw", "d"], WORDS), 1)).toBe(" beta gamma");
  });

  it("vaw takes the trailing whitespace too", () => {
    expect(line(screenAfter(["vaw", "d"], WORDS), 1)).toBe("beta gamma");
  });

  it("vi( selects the parenthesised body", () => {
    expect(line(screenAfter(["f(", "vi(", "d"], "call(a, b) end\n"), 1)).toBe(
      "call() end",
    );
  });

  it("va( includes the brackets", () => {
    expect(line(screenAfter(["f(", "va(", "d"], "call(a, b) end\n"), 1)).toBe(
      "call end",
    );
  });

  it('vi" selects the quoted body', () => {
    expect(line(screenAfter(['vi"', "d"], 'x = "hello" ;\n'), 1)).toBe(
      'x = "" ;',
    );
  });

  it("text objects can be changed straight away", () => {
    const s = screenAfter(["viw", "c", "ZZ", "<escape>"], WORDS);
    expect(line(s, 1)).toBe("ZZ beta gamma");
  });

  it("vip selects a paragraph linewise", () => {
    const s = screenAfter(["vip", "d"], "one\ntwo\n\nthree\n");
    expect(s).toContain("-- NORMAL --");
    expect(lines(s, 2)).toEqual(["", "three"]);
  });

  it("a text object switches the mode indicator to VISUAL LINE when linewise", () => {
    expect(screenAfter(["vip"], "one\ntwo\n\nthree\n")).toContain(
      "-- VISUAL LINE --",
    );
  });

  it("counted bracket objects reach the enclosing pair", () => {
    expect(line(screenAfter(["fx", "v2i(", "d"], "a(b(x)c)d\n"), 1)).toBe(
      "a()d",
    );
  });
});

describe("vim visual block", () => {
  it("d deletes the column range on every row", () => {
    const s = screenAfter(["<ctrl+v>", "jj", "l", "d"]);
    expect(lines(s, 4)).toEqual(["cdef", "ijkl", "opqr", "stuvwx"]);
  });

  it("x is the same as d", () => {
    const s = screenAfter(["<ctrl+v>", "j", "l", "x"]);
    expect(lines(s, 3)).toEqual(["cdef", "ijkl", "mnopqr"]);
  });

  it("y yanks the block and p pastes the rows joined by newlines", () => {
    // G lands on the trailing empty line, so the two rows paste as new lines.
    const s = screenAfter(["<ctrl+v>", "j", "l", "y", "G", "p"]);
    expect(line(s, 5)).toBe("ab");
    expect(line(s, 6)).toBe("gh");
  });

  it("c replaces the block and types on every row", () => {
    const s = screenAfter(["<ctrl+v>", "jj", "l", "c", "Z", "<escape>"]);
    expect(lines(s, 4)).toEqual(["Zcdef", "Zijkl", "Zopqr", "stuvwx"]);
  });

  it("I inserts at the left edge of every row", () => {
    const s = screenAfter(["l", "<ctrl+v>", "jj", "I", "X", "<escape>"]);
    expect(lines(s, 4)).toEqual(["aXbcdef", "gXhijkl", "mXnopqr", "stuvwx"]);
  });

  it("A appends at the right edge of every row", () => {
    const s = screenAfter(["<ctrl+v>", "jj", "ll", "A", "X", "<escape>"]);
    expect(lines(s, 4)).toEqual(["abcXdef", "ghiXjkl", "mnoXpqr", "stuvwx"]);
  });

  it("r fills every cell of the block", () => {
    const s = screenAfter(["<ctrl+v>", "jj", "l", "rZ"]);
    expect(lines(s, 4)).toEqual(["ZZcdef", "ZZijkl", "ZZopqr", "stuvwx"]);
  });

  it("gU uppercases the block only", () => {
    const s = screenAfter(["<ctrl+v>", "jj", "l", "gU"]);
    expect(lines(s, 4)).toEqual(["ABcdef", "GHijkl", "MNopqr", "stuvwx"]);
  });

  it("$ gives a ragged right edge", () => {
    const s = screenAfter(["l", "<ctrl+v>", "jj", "$", "d"], RAGGED);
    expect(lines(s, 3)).toEqual(["a", "i", "k"]);
  });

  it("$ then A appends at each line's own end", () => {
    const s = screenAfter(
      ["l", "<ctrl+v>", "jj", "$", "A", "X", "<escape>"],
      RAGGED,
    );
    expect(lines(s, 3)).toEqual(["abcdefghX", "ijX", "klmnopX"]);
  });

  it("rows shorter than the block's left edge are left alone", () => {
    // Block over columns 4..5; line 2 ("ij") has no such columns.
    const s = screenAfter(["3l", "<ctrl+v>", "jj", "l", "d"], RAGGED);
    expect(lines(s, 3)).toEqual(["abcfgh", "ij", "klmp"]);
  });

  it("leaving block mode collapses the extra cursors", () => {
    // If the block cursors survived, typing "Z" in insert mode would land on
    // every row instead of just the current one.
    // Esc leaves the cursor on the row the block ended on, as in Vim.
    const s = screenAfter(["<ctrl+v>", "jj", "<escape>", "iZ", "<escape>"]);
    expect(lines(s, 4)).toEqual(["abcdef", "ghijkl", "Zmnopqr", "stuvwx"]);
  });
});

describe("vim visual: undo atomicity", () => {
  it("a charwise delete is one undo step", () => {
    const s = screenAfter(["v", "ll", "d", "u"]);
    expect(line(s, 1)).toBe("abcdef");
  });

  it("a linewise delete is one undo step", () => {
    const s = screenAfter(["V", "jj", "d", "u"]);
    expect(lines(s, 4)).toEqual(["abcdef", "ghijkl", "mnopqr", "stuvwx"]);
  });

  it("a blockwise delete across three lines is ONE undo step", () => {
    const s = screenAfter(["<ctrl+v>", "jj", "l", "d", "u"]);
    expect(lines(s, 4)).toEqual(["abcdef", "ghijkl", "mnopqr", "stuvwx"]);
  });

  it("a blockwise r across three lines is one undo step", () => {
    const s = screenAfter(["<ctrl+v>", "jj", "l", "rZ", "u"]);
    expect(lines(s, 4)).toEqual(["abcdef", "ghijkl", "mnopqr", "stuvwx"]);
  });

  it("a blockwise gU across three lines is one undo step", () => {
    const s = screenAfter(["<ctrl+v>", "jj", "l", "gU", "u"]);
    expect(lines(s, 4)).toEqual(["abcdef", "ghijkl", "mnopqr", "stuvwx"]);
  });

  it("blockwise I plus typing plus Esc is one undo step", () => {
    const s = screenAfter(["<ctrl+v>", "jj", "I", "XY", "<escape>", "u"]);
    expect(lines(s, 4)).toEqual(["abcdef", "ghijkl", "mnopqr", "stuvwx"]);
  });

  it("blockwise c plus typing plus Esc is one undo step", () => {
    const s = screenAfter(["<ctrl+v>", "jj", "l", "c", "Z", "<escape>", "u"]);
    expect(lines(s, 4)).toEqual(["abcdef", "ghijkl", "mnopqr", "stuvwx"]);
  });

  it("a visual change is one undo step", () => {
    const s = screenAfter(["v", "ll", "c", "ZZ", "<escape>", "u"]);
    expect(line(s, 1)).toBe("abcdef");
  });

  it("a multi-line visual indent is one undo step", () => {
    const s = screenAfter(["V", "jj", ">", "u"]);
    expect(lines(s, 3)).toEqual(["abcdef", "ghijkl", "mnopqr"]);
  });

  it("a visual J is one undo step", () => {
    const s = screenAfter(["V", "jj", "J", "u"]);
    expect(lines(s, 3)).toEqual(["abcdef", "ghijkl", "mnopqr"]);
  });

  it("a visual p is one undo step", () => {
    const s = screenAfter(["v", "ll", "y", "j", "v", "ll", "p", "u"]);
    expect(lines(s, 2)).toEqual(["abcdef", "ghijkl"]);
  });
});
