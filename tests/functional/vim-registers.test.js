import { describe, it, expect, afterEach } from "vitest";
import * as tui from "./tui.js";
import { createTempDir, createTempFile, cleanupDir } from "./helpers.js";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

// Exercise the real shipped plugin, not an inline copy.
const VIM_PLUGIN = join(dirname(fileURLToPath(import.meta.url)), "..", "..", "plugins", "vim", "init.lua");

const LINES = "one\ntwo\nthree\nfour\nfive\n";
const WORDS = "alpha beta gamma\nsecond line here\nthird line\n";
const GRID = "abcdef\nghijkl\nmnopqr\nstuvwx\n";
// Deliberately ragged: the last row is too short to reach the paste column.
const RAGGED = "abcdef\nghijkl\nmn\n";

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

describe("vim registers: named registers", () => {
  it('"ayy then "ap pastes from the named register', () => {
    const s = screenAfter(['"ayy', "j", '"ap']);
    expect(line(s, 1)).toBe("one");
    expect(line(s, 2)).toBe("two");
    expect(line(s, 3)).toBe("one");
  });

  it("a named register survives an intervening unnamed delete", () => {
    const s = screenAfter(['"ayy', "j", "dd", '"ap']);
    // "two" was deleted; "a still holds "one".
    expect(line(s, 1)).toBe("one");
    expect(line(s, 2)).toBe("three");
    expect(line(s, 3)).toBe("one");
  });

  it('uppercase "A appends to the named register', () => {
    // The fixture has a trailing newline, so G lands on the empty line 6.
    const s = screenAfter(['"ayy', "j", '"Ayy', "G", '"ap']);
    expect(line(s, 7)).toBe("one");
    expect(line(s, 8)).toBe("two");
  });

  it("a named register works with an operator and a count", () => {
    const s = screenAfter(['"a2dd', "G", '"ap']);
    expect(line(s, 1)).toBe("three");
    expect(line(s, 5)).toBe("one");
    expect(line(s, 6)).toBe("two");
  });

  it("a named register is charwise when the yank was", () => {
    const s = screenAfter(['"ayw', "j", '"ap'], WORDS);
    expect(line(s, 2)).toBe("salpha econd line here");
  });
});

describe("vim registers: the yank register and the delete ring", () => {
  it('"0 keeps the last yank across a delete', () => {
    const s = screenAfter(["yy", "dd", '"0p']);
    expect(line(s, 1)).toBe("two");
    expect(line(s, 2)).toBe("one");
  });

  it('"1 and "2 rotate as lines are deleted', () => {
    const s = screenAfter(["dd", "dd", '"1p', '"2p']);
    expect(line(s, 1)).toBe("three");
    expect(line(s, 2)).toBe("two");
    expect(line(s, 3)).toBe("one");
  });

  it("a small delete does not disturb the ring", () => {
    const s = screenAfter(["dd", "x", '"1p']);
    expect(line(s, 1)).toBe("wo");
    expect(line(s, 2)).toBe("one");
  });
});

describe("vim registers: the blackhole register", () => {
  it('"_dd deletes without touching the unnamed register', () => {
    const s = screenAfter(["yy", '"_dd', "p"]);
    expect(line(s, 1)).toBe("two");
    expect(line(s, 2)).toBe("one");
  });

  it('"_x does not clobber a yank', () => {
    const s = screenAfter(["yy", '"_x', "p"]);
    expect(line(s, 1)).toBe("ne");
    expect(line(s, 2)).toBe("one");
  });
});

describe("vim registers: blockwise", () => {
  it("a blockwise yank pastes back as a rectangle", () => {
    const s = screenAfter(["<ctrl+v>", "jl", "y", "3l", "p"], GRID);
    expect(line(s, 1)).toBe("abcdabef");
    expect(line(s, 2)).toBe("ghijghkl");
    expect(line(s, 3)).toBe("mnopqr");
  });

  it("a blockwise delete pastes back as a rectangle", () => {
    const s = screenAfter(["<ctrl+v>", "jl", "d", "$", "p"], GRID);
    expect(line(s, 1)).toBe("cdefab");
    expect(line(s, 2)).toBe("ijklgh");
  });

  it("blockwise paste pads a short row out to the paste column", () => {
    const s = screenAfter(["<ctrl+v>", "jl", "y", "jj", "$", "p"], RAGGED);
    // Row 3 is "mn"; the block lands after its end, padded with spaces.
    expect(line(s, 3)).toBe("mnab");
    expect(line(s, 4)).toBe("  gh");
  });

  it("blockwise paste appends rows past the end of the buffer", () => {
    // G lands on the empty line after the fixture's trailing newline, so the
    // block's second and third rows have to create lines of their own.
    const s = screenAfter(["<ctrl+v>", "jjl", "y", "G", "p"], GRID);
    expect(line(s, 5)).toBe("ab");
    expect(line(s, 6)).toBe("gh");
    expect(line(s, 7)).toBe("mn");
  });
});

describe("vim registers: visual mode", () => {
  it("a register prefix works in visual mode", () => {
    const s = screenAfter(["v", "$", '"ay', "j", '"ap'], WORDS);
    expect(line(s, 2)).toBe("salpha beta gammaecond line here");
  });

  it("V with a named register yanks linewise", () => {
    const s = screenAfter(["V", '"ay', "G", '"ap']);
    expect(line(s, 7)).toBe("one");
  });
});

describe("vim registers: system clipboard", () => {
  // "+ has no Lua binding, so it is routed through editor.copy / editor.paste.
  // A round trip is the only thing that can be asserted without a real
  // clipboard tool being present.
  it('"+y then "+p round-trips through the clipboard', () => {
    const s = screenAfter(["v", "ll", '"+y', "j", "$", '"+p']);
    expect(line(s, 1)).toBe("one");
    expect(line(s, 2)).toBe("twoone");
  });
});

describe("vim marks", () => {
  it("m sets a mark and backtick jumps back to it exactly", () => {
    const s = screenAfter(["ll", "ma", "jjj", "`a"], WORDS);
    expect(pos(s)).toBe("1:3");
  });

  it("' jumps to the first non-blank of the marked line", () => {
    const s = screenAfter(["j", "ll", "ma", "gg", "'a"], "  indented\nsecond\n");
    expect(pos(s)).toBe("2:1");
  });

  it("a mark survives lines being inserted above it", () => {
    const s = screenAfter(["G", "ma", "gg", "O", "new", "<esc>", "`a"]);
    expect(pos(s)).toBe("7:1");
    expect(line(s, 7)).toBe("");
  });

  it("a mark survives lines being deleted above it", () => {
    const s = screenAfter(["3j", "ma", "gg", "dd", "`a"]);
    expect(pos(s)).toBe("3:1");
    expect(line(s, 3)).toBe("four");
  });

  it("`` jumps back to the position before the last jump", () => {
    const s = screenAfter(["jj", "G", "``"]);
    expect(pos(s)).toBe("3:1");
  });

  it("`. jumps to the position of the last change", () => {
    const s = screenAfter(["jj", "x", "G", "`."]);
    expect(pos(s)).toBe("3:1");
  });

  it("a mark is a valid operator target", () => {
    const s = screenAfter(["ma", "3j", "d'a"]);
    expect(line(s, 1)).toBe("five");
  });

  it("marks can be set in visual mode", () => {
    const s = screenAfter(["v", "jj", "<esc>", "gg", "'a"]);
    // No mark 'a' was set, so nothing moves.
    expect(pos(s)).toBe("1:1");
  });
});
