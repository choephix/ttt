import { describe, it, expect, afterEach } from "vitest";
import * as tui from "./tui.js";
import { createTempDir, createTempFile, cleanupDir } from "./helpers.js";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

// Exercise the real shipped plugin, not an inline copy.
const VIM_PLUGIN = join(dirname(fileURLToPath(import.meta.url)), "..", "..", "plugins", "vim", "init.lua");

// Line 1 is three plain words. Line 2 is indented so linewise change can be
// shown to preserve indentation. Line 3 ends the "paragraph" for { and }.
const FIXTURE =
  "alpha beta gamma\n" + "  indented two\n" + "count 41 and 007\n" + "temp -5 degrees\n" + "last line\n";

// Short, unambiguous line markers for the linewise matrix.
const LINES = "L1\nL2\nL3\nL4\nL5\n";

const PARA = "one\ntwo\n\nthree\nfour\n";

const CODE = "function foo() {\n" + "bar()\n" + "baz()\n" + "}\n";

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
  tui.waitStable();
  const s = tui.snapshot();
  const { snapshots } = tui.run();
  return snapshots[s];
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

describe("vim operators: exclusive vs inclusive motions", () => {
  // The single most common Vim-emulation bug: dw must not eat the character it
  // lands on, de must.
  it("dw is exclusive and de is inclusive", () => {
    expect(screenAfter(["dw"])).toContain("beta gamma");
    expect(screenAfter(["de"])).toContain(" beta gamma");
  });

  it("d$ is inclusive and deletes the last character", () => {
    const s = screenAfter(["5l", "d$"]);
    expect(line(s, 1)).toBe("alpha");
  });

  it("d0 deletes back to the start of the line", () => {
    expect(line(screenAfter(["6l", "d0"]), 1)).toBe("beta gamma");
  });

  it("dfx includes the target character, dtx does not", () => {
    expect(line(screenAfter(["dfb"]), 1)).toBe("eta gamma");
    expect(line(screenAfter(["dtb"]), 1)).toBe("beta gamma");
  });

  it("dFx and dTx are exclusive of the character under the cursor", () => {
    // Cursor on the "g" of gamma (col 12); dFb deletes back to the "b".
    expect(line(screenAfter(["11l", "dFb"]), 1)).toBe("alpha gamma");
    expect(line(screenAfter(["11l", "dTb"]), 1)).toBe("alpha bgamma");
  });

  it("dh and dl work in both directions", () => {
    expect(line(screenAfter(["dl"]), 1)).toBe("lpha beta gamma");
    expect(line(screenAfter(["3l", "dh"]), 1)).toBe("alha beta gamma");
  });

  it("d% is inclusive of both brackets", () => {
    expect(line(screenAfter(["f(", "d%"], "a (b c) d\n"), 1)).toBe("a  d");
  });
});

describe("vim operators: word motions at line ends", () => {
  // Vim: "the last word moved over is at the end of a line" -> the operated
  // text stops there instead of swallowing the newline and the next indent.
  it("dw on the last word of a line stops at the line end", () => {
    const s = screenAfter(["$", "b", "dw"]);
    expect(line(s, 1)).toBe("alpha beta");
    expect(line(s, 2)).toBe("  indented two");
  });

  it("cw behaves like ce on a non-blank", () => {
    // Real Vim special case: cw does not reach the start of the next word.
    expect(line(screenAfter(["cwZZ", "<escape>"]), 1)).toBe("ZZ beta gamma");
  });

  it("cw on the last word of a line does not join lines", () => {
    const s = screenAfter(["$", "b", "cwZZ", "<escape>"]);
    expect(line(s, 1)).toBe("alpha beta ZZ");
    expect(line(s, 2)).toBe("  indented two");
  });
});

describe("vim operators: linewise motions", () => {
  it("dj and dk take whole lines in both directions", () => {
    expect(line(screenAfter(["jj", "dj"], LINES), 3)).toBe("L5");
    const up = screenAfter(["jj", "dk"], LINES);
    expect(line(up, 1)).toBe("L1");
    expect(line(up, 2)).toBe("L4");
  });

  it("d2j takes three lines", () => {
    const s = screenAfter(["jj", "d2j"], LINES);
    expect(line(s, 1)).toBe("L1");
    expect(line(s, 2)).toBe("L2");
    expect(line(s, 3)).toBe("");
  });

  it("dG deletes to the end of the buffer", () => {
    expect(line(screenAfter(["jj", "dG"], LINES), 1)).toBe("L1");
  });

  it("dgg deletes to the start of the buffer", () => {
    const s = screenAfter(["jj", "dgg"], LINES);
    expect(line(s, 1)).toBe("L4");
  });

  it("d3G deletes from the cursor line through line 3", () => {
    const s = screenAfter(["d3G"], LINES);
    expect(line(s, 1)).toBe("L4");
  });

  it("d} takes the paragraph but leaves the blank line", () => {
    const s = screenAfter(["d}"], PARA);
    expect(line(s, 1)).toBe("");
    expect(line(s, 2)).toBe("three");
  });
});

describe("vim operators: doubled linewise forms", () => {
  it("dd deletes the current line", () => {
    expect(line(screenAfter(["dd"], LINES), 1)).toBe("L2");
  });

  it("2dd and d2d delete two lines", () => {
    expect(line(screenAfter(["2dd"], LINES), 1)).toBe("L3");
    expect(line(screenAfter(["d2d"], LINES), 1)).toBe("L3");
  });

  it("dd on the last line leaves no stray blank", () => {
    // LINES ends with a newline, so line 6 is the empty final line.
    const s = screenAfter(["G", "k", "dd"], LINES);
    expect(line(s, 4)).toBe("L4");
    expect(line(s, 5)).toBe("");
    expect(line(s, 6)).toBe(null);
  });

  it("deleting every line leaves one empty line", () => {
    const s = screenAfter(["10dd"], LINES);
    expect(line(s, 1)).toBe("");
    expect(line(s, 2)).toBe(null);
  });

  it("yy and cc are doubled too", () => {
    expect(line(screenAfter(["yyp"], LINES), 2)).toBe("L1");
    expect(line(screenAfter(["ccZZ", "<escape>"], LINES), 1)).toBe("ZZ");
  });

  it("cc preserves the line's indentation", () => {
    expect(line(screenAfter(["j", "ccZZ", "<escape>"]), 2)).toBe("  ZZ");
  });

  it("S is the same operation as cc", () => {
    expect(line(screenAfter(["j", "SZZ", "<escape>"]), 2)).toBe("  ZZ");
  });
});

describe("vim operators: counts multiply", () => {
  it("2d3w deletes six words", () => {
    // alpha beta gamma / indented two / count -> leaves "41 and 007".
    expect(line(screenAfter(["2d3w"]), 1)).toBe("41 and 007");
  });

  it("3dw and d3w agree", () => {
    expect(line(screenAfter(["3dw"]), 1)).toBe(line(screenAfter(["d3w"]), 1));
  });

  it("2d2d deletes four lines", () => {
    expect(line(screenAfter(["2d2d"], LINES), 1)).toBe("L5");
  });
});

describe("vim operators: change", () => {
  it("c$ changes to the end of the line", () => {
    expect(line(screenAfter(["5l", "c$ZZ", "<escape>"]), 1)).toBe("alphaZZ");
  });

  it("c with a linewise motion collapses the lines", () => {
    const s = screenAfter(["cjZZ", "<escape>"], LINES);
    expect(line(s, 1)).toBe("ZZ");
    expect(line(s, 2)).toBe("L3");
  });

  it("c leaves you in insert mode, d and y do not", () => {
    startVim();
    tui.type("cw");
    tui.waitStable(100);
    const afterC = tui.snapshot();
    tui.press("escape");
    tui.type("dw");
    tui.waitStable(100);
    const afterD = tui.snapshot();
    tui.type("yw");
    tui.waitStable(100);
    const afterY = tui.snapshot();
    const { snapshots } = tui.run();

    expect(snapshots[afterC]).toContain("-- INSERT --");
    expect(snapshots[afterD]).toContain("-- NORMAL --");
    expect(snapshots[afterY]).toContain("-- NORMAL --");
  });
});

describe("vim operators: yank and paste", () => {
  it("yy then p puts the line below", () => {
    const s = screenAfter(["yyp"], LINES);
    expect(line(s, 1)).toBe("L1");
    expect(line(s, 2)).toBe("L1");
    expect(line(s, 3)).toBe("L2");
  });

  it("yy then P puts the line above", () => {
    const s = screenAfter(["j", "yyP"], LINES);
    expect(line(s, 2)).toBe("L2");
    expect(line(s, 3)).toBe("L2");
  });

  it("a linewise paste lands on its own line, not inline", () => {
    const s = screenAfter(["yy", "j", "p"], LINES);
    expect(line(s, 2)).toBe("L2");
    expect(line(s, 3)).toBe("L1");
  });

  it("yw then P pastes charwise before the cursor", () => {
    expect(line(screenAfter(["ywP"]), 1)).toBe("alpha alpha beta gamma");
  });

  it("yw then p pastes charwise after the cursor", () => {
    expect(line(screenAfter(["yw", "p"]), 1)).toBe("aalpha lpha beta gamma");
  });

  it("a counted paste repeats the register", () => {
    const s = screenAfter(["yy", "3p"], LINES);
    expect(line(s, 2)).toBe("L1");
    expect(line(s, 3)).toBe("L1");
    expect(line(s, 4)).toBe("L1");
    expect(line(s, 5)).toBe("L2");
  });

  it("dd then p moves a line down", () => {
    const s = screenAfter(["dd", "p"], LINES);
    expect(line(s, 1)).toBe("L2");
    expect(line(s, 2)).toBe("L1");
  });

  it("x then p transposes two characters", () => {
    expect(line(screenAfter(["xp"]), 1)).toBe("lapha beta gamma");
  });

  it("y2j yanks three lines linewise", () => {
    const s = screenAfter(["y2j", "G", "p"], LINES);
    expect(line(s, 7)).toBe("L1");
    expect(line(s, 8)).toBe("L2");
    expect(line(s, 9)).toBe("L3");
  });

  it("Y yanks to the end of the line", () => {
    // ttt follows Neovim's Y = y$ rather than Vim's Y = yy.
    expect(line(screenAfter(["6l", "YP"]), 1)).toBe("alpha beta gammabeta gamma");
  });

  it("y leaves the cursor at the start of the range", () => {
    expect(posAfter(["$", "yb"])).toBe("1:12");
  });
});

describe("vim operators: indent", () => {
  it(">> and >j agree on two lines", () => {
    const a = screenAfter(["2>>"], LINES);
    const b = screenAfter([">j"], LINES);
    expect(line(a, 1)).toBe(line(b, 1));
    expect(line(a, 2)).toBe(line(b, 2));
    expect(line(a, 1)).toBe("    L1");
  });

  it("> with a linewise motion indents the range", () => {
    const s = screenAfter([">G"], LINES);
    expect(line(s, 1)).toBe("    L1");
    expect(line(s, 5)).toBe("    L5");
  });

  it("< dedents and stops at column 1", () => {
    const s = screenAfter([">>", ">>", "<<"], LINES);
    expect(line(s, 1)).toBe("    L1");
  });

  it("> is always linewise even with a charwise motion", () => {
    expect(line(screenAfter([">w"], LINES), 1)).toBe("    L1");
  });

  it("= reindents a range", () => {
    const s = screenAfter(["j", "=j"], CODE);
    expect(line(s, 2)).toBe("    bar()");
    expect(line(s, 3)).toBe("    baz()");
  });
});

describe("vim operators: case", () => {
  it("gUw and guw change case over a word", () => {
    expect(line(screenAfter(["gUw"]), 1)).toBe("ALPHA beta gamma");
    expect(line(screenAfter(["gUw", "guw"]), 1)).toBe("alpha beta gamma");
  });

  it("g~w swaps case", () => {
    expect(line(screenAfter(["gUw", "g~w"]), 1)).toBe("alpha beta gamma");
  });

  it("gUU, gUgU and gU$ act on the line", () => {
    expect(line(screenAfter(["gUU"]), 1)).toBe("ALPHA BETA GAMMA");
    expect(line(screenAfter(["gUgU"]), 1)).toBe("ALPHA BETA GAMMA");
    expect(line(screenAfter(["gU$"]), 1)).toBe("ALPHA BETA GAMMA");
  });

  it("guu and g~~ act on the line", () => {
    expect(line(screenAfter(["gUU", "guu"]), 1)).toBe("alpha beta gamma");
    expect(line(screenAfter(["g~~"]), 1)).toBe("ALPHA BETA GAMMA");
  });

  it("gU with a linewise motion covers every line", () => {
    const s = screenAfter(["gUj"], LINES);
    expect(line(s, 1)).toBe("L1");
    expect(line(s, 2)).toBe("L2");
    // Already uppercase; use the word fixture for a visible change.
    const t = screenAfter(["gUj"]);
    expect(line(t, 1)).toBe("ALPHA BETA GAMMA");
    expect(line(t, 2)).toBe("  INDENTED TWO");
  });

  it("2gUU covers two lines", () => {
    const s = screenAfter(["2gUU"]);
    expect(line(s, 1)).toBe("ALPHA BETA GAMMA");
    expect(line(s, 2)).toBe("  INDENTED TWO");
  });
});

describe("vim operators: shorthands", () => {
  it("D deletes to the end of the line", () => {
    expect(line(screenAfter(["5l", "D"]), 1)).toBe("alpha");
  });

  it("C changes to the end of the line", () => {
    expect(line(screenAfter(["5l", "CZZ", "<escape>"]), 1)).toBe("alphaZZ");
  });

  it("D fills the register", () => {
    // D from col 6 takes " beta gamma"; Esc-style clamping leaves the cursor on
    // col 5, so P puts the text back before the final "a".
    expect(line(screenAfter(["5l", "D", "P"]), 1)).toBe("alph beta gammaa");
  });
});

// Every Vim operation must collapse to exactly one undo step.
describe("vim operators: undo atomicity", () => {
  const original = "alpha beta gamma";

  it("undoes d with a motion as ONE step", () => {
    expect(line(screenAfter(["d2w", "u"]), 1)).toBe(original);
  });

  it("undoes dd as ONE step", () => {
    expect(line(screenAfter(["dd", "u"], LINES), 1)).toBe("L1");
  });

  it("undoes a counted dd as ONE step", () => {
    const s = screenAfter(["3dd", "u"], LINES);
    expect(line(s, 1)).toBe("L1");
    expect(line(s, 3)).toBe("L3");
  });

  it("undoes c plus typing as ONE step", () => {
    expect(line(screenAfter(["cwZZZ", "<escape>", "u"]), 1)).toBe(original);
  });

  it("undoes cc plus typing as ONE step", () => {
    expect(line(screenAfter(["ccZZZ", "<escape>", "u"]), 1)).toBe(original);
  });

  it("undoes a linewise c plus typing as ONE step", () => {
    const s = screenAfter(["cjZZZ", "<escape>", "u"], LINES);
    expect(line(s, 1)).toBe("L1");
    expect(line(s, 2)).toBe("L2");
  });

  it("undoes > as ONE step", () => {
    expect(line(screenAfter([">G", "u"], LINES), 1)).toBe("L1");
  });

  it("undoes < as ONE step", () => {
    const s = screenAfter([">G", "<G", "u"], LINES);
    expect(line(s, 1)).toBe("    L1");
  });

  it("undoes = as ONE step", () => {
    expect(line(screenAfter(["j", "=j", "u"], CODE), 2)).toBe("bar()");
  });

  it("undoes gU as ONE step", () => {
    expect(line(screenAfter(["gUj", "u"]), 1)).toBe(original);
  });

  it("undoes gu as ONE step", () => {
    expect(line(screenAfter(["gUj", "guj", "u"]), 1)).toBe("ALPHA BETA GAMMA");
  });

  it("undoes g~ as ONE step", () => {
    expect(line(screenAfter(["g~~", "u"]), 1)).toBe(original);
  });

  it("undoes p as ONE step", () => {
    expect(line(screenAfter(["yy", "3p", "u"], LINES), 2)).toBe("L2");
  });

  it("undoes P as ONE step", () => {
    expect(line(screenAfter(["ywP", "u"]), 1)).toBe(original);
  });

  it("y alone pushes nothing onto the undo stack", () => {
    // The undo after `yw` must reach the earlier `x`, not a no-op yank entry.
    expect(line(screenAfter(["x", "yw", "u"]), 1)).toBe(original);
  });
});

describe("vim operators: pending state", () => {
  it("Esc cancels a pending operator without editing", () => {
    expect(line(screenAfter(["d", "<escape>", "w"]), 1)).toBe("alpha beta gamma");
  });

  it("an operator followed by a meaningless key is a no-op", () => {
    expect(line(screenAfter(["dz", "w"]), 1)).toBe("alpha beta gamma");
  });

  it("the operator does not leak into the next keystroke", () => {
    // d then Esc, then a plain dd must still delete exactly one line.
    expect(line(screenAfter(["d", "<escape>", "dd"], LINES), 1)).toBe("L2");
  });
});
