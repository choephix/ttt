import { describe, it, expect, afterEach } from "vitest";
import * as tui from "./tui.js";
import { createTempDir, createTempFile, cleanupDir } from "./helpers.js";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

// Exercise the real shipped plugin, not an inline copy.
const VIM_PLUGIN = join(dirname(fileURLToPath(import.meta.url)), "..", "..", "plugins", "vim", "init.lua");

// Line 1 is 16 columns of plain words. Line 2 is indented so I/^ differ from 0.
// Line 3 carries a number for Ctrl-A/Ctrl-X and a zero-padded one for width
// preservation. Line 4 has a negative number.
const FIXTURE =
  "alpha beta gamma\n" + "  indented two\n" + "count 41 and 007\n" + "temp -5 degrees\n" + "last line\n";

const CODE = "function foo() {\n" + "bar()\n" + "}\n";

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
function send(keys) {
  for (const k of keys) {
    if (k.startsWith("<")) {
      tui.press(k.slice(1, -1));
    } else {
      tui.type(k);
    }
  }
}

// Run a script against `content` and return the rendered screen.
function screenAfter(keys, content = FIXTURE) {
  startVim(content);
  send(keys);
  tui.waitStable();
  const s = tui.snapshot();
  const { snapshots } = tui.run();
  return snapshots[s];
}

// The status bar is the cleanest cursor readout the harness has.
function pos(snapshot) {
  const m = snapshot.match(/Ln (\d+), Col (\d+)/);
  return m ? `${m[1]}:${m[2]}` : "none";
}

function posAfter(keys, content = FIXTURE) {
  return pos(screenAfter(keys, content));
}

describe("vim edits: entering insert mode", () => {
  it("inserts before the cursor with i", () => {
    expect(screenAfter(["iX"])).toContain("Xalpha beta gamma");
  });

  it("inserts after the cursor with a", () => {
    expect(screenAfter(["aX"])).toContain("aXlpha beta gamma");
  });

  it("inserts at the first non-blank with I", () => {
    expect(screenAfter(["j", "$", "IX"])).toContain("  Xindented two");
  });

  it("appends at end of line with A", () => {
    expect(screenAfter(["AX"])).toContain("alpha beta gammaX");
  });

  it("opens a line below with o", () => {
    const s = screenAfter(["oNEW"]);
    expect(s).toContain("alpha beta gamma");
    expect(s).toMatch(/2\s+NEW/);
    expect(s).toMatch(/3\s+ {2}indented two/);
  });

  it("opens a line above with O", () => {
    const s = screenAfter(["jONEW"]);
    expect(s).toMatch(/2\s+NEW/);
    expect(s).toMatch(/3\s+ {2}indented two/);
  });

  // Insert rejects line >= #Lines on the Go side, so `o` on the last line has
  // to append the newline to that line instead of addressing the line after.
  it("opens a line below even on the last line", () => {
    const s = screenAfter(["G", "oEND"]);
    expect(s).toMatch(/7\s+END/);
  });

  it("resumes the last insert position with gi", () => {
    // Insert X at the start, leave, jump away, then gi must come back.
    const s = screenAfter(["iX", "<escape>", "G", "giY"]);
    expect(s).toContain("XYalpha beta gamma");
  });

  it("steps the cursor one column left on Esc, like Vim", () => {
    // i at col 1, type 5 chars -> col 6 in insert; Esc pulls back to col 5.
    startVim();
    tui.type("ihello");
    tui.waitStable(100);
    const inInsert = tui.snapshot();
    tui.press("escape");
    tui.waitStable();
    const inNormal = tui.snapshot();
    const { snapshots } = tui.run();

    expect(snapshots[inInsert]).toContain("-- INSERT --");
    expect(pos(snapshots[inInsert])).toBe("1:6");
    expect(snapshots[inNormal]).toContain("-- NORMAL --");
    expect(pos(snapshots[inNormal])).toBe("1:5");
  });

  it("does not step left when insert ended at column 1", () => {
    expect(posAfter(["i", "<escape>"])).toBe("1:1");
  });
});

describe("vim edits: deleting characters", () => {
  it("deletes forward with x and a count", () => {
    expect(screenAfter(["x"])).toContain("lpha beta gamma");
    expect(screenAfter(["3x"])).toContain("ha beta gamma");
  });

  it("clamps x at the end of the line", () => {
    const s = screenAfter(["$", "5x"]);
    expect(s).toContain("alpha beta gamm");
    expect(pos(s)).toBe("1:15");
  });

  it("deletes backward with X", () => {
    expect(screenAfter(["3lX"])).toContain("alha beta gamma");
    expect(screenAfter(["3l2X"])).toContain("aha beta gamma");
  });

  it("does nothing for X at column 1", () => {
    const s = screenAfter(["X"]);
    expect(s).toContain("alpha beta gamma");
    expect(pos(s)).toBe("1:1");
  });

  it("deletes to end of line with D", () => {
    expect(screenAfter(["5lD"])).toMatch(/1\s+alpha\b/);
  });

  it("deletes across lines with a counted D", () => {
    // 2D removes to the end of the line below, joining what is left.
    const s = screenAfter(["5l2D"]);
    expect(s).toMatch(/1\s+alpha\s*│/);
    expect(s).toMatch(/2\s+count 41 and 007/);
  });
});

describe("vim edits: change and replace", () => {
  it("changes to end of line with C", () => {
    expect(screenAfter(["5lCXYZ"])).toContain("alphaXYZ");
  });

  it("substitutes characters with s", () => {
    expect(screenAfter(["sX"])).toContain("Xlpha beta gamma");
    expect(screenAfter(["3sX"])).toContain("Xha beta gamma");
  });

  it("substitutes whole lines with S", () => {
    const s = screenAfter(["SX"]);
    expect(s).toMatch(/1\s+X\s*│/);
    expect(s).toMatch(/2\s+ {2}indented two/);
  });

  it("replaces one character with r", () => {
    const s = screenAfter(["rZ"]);
    expect(s).toContain("Zlpha beta gamma");
    expect(pos(s)).toBe("1:1");
  });

  it("replaces a counted run with r", () => {
    const s = screenAfter(["3rZ"]);
    expect(s).toContain("ZZZha beta gamma");
    expect(pos(s)).toBe("1:3");
  });

  it("refuses r when the count runs past the end of the line", () => {
    expect(screenAfter(["$", "3rZ"])).toContain("alpha beta gamma");
  });

  it("overtypes in replace mode until Esc", () => {
    startVim();
    tui.type("R");
    tui.waitStable(100);
    const inReplace = tui.snapshot();
    tui.type("XYZ");
    tui.press("escape");
    tui.waitStable();
    const done = tui.snapshot();
    const { snapshots } = tui.run();

    expect(snapshots[inReplace]).toContain("-- REPLACE --");
    expect(snapshots[done]).toContain("XYZha beta gamma");
    expect(snapshots[done]).toContain("-- NORMAL --");
  });

  it("extends the line when replace mode runs past its end", () => {
    expect(screenAfter(["$", "RXYZ", "<escape>"])).toContain("alpha beta gammXYZ");
  });
});

describe("vim edits: case and joins", () => {
  it("toggles case with ~ and advances", () => {
    const s = screenAfter(["~"]);
    expect(s).toContain("Alpha beta gamma");
    expect(pos(s)).toBe("1:2");
  });

  it("toggles a counted run with ~", () => {
    expect(screenAfter(["5~"])).toContain("ALPHA beta gamma");
  });

  it("joins two lines with a space using J", () => {
    const s = screenAfter(["J"]);
    expect(s).toContain("alpha beta gamma indented two");
    expect(pos(s)).toBe("1:17"); // cursor sits on the inserted space
  });

  it("joins a counted run of lines with J", () => {
    expect(screenAfter(["3J"])).toContain("alpha beta gamma indented two count 41 and 007");
  });

  it("joins without a space using gJ", () => {
    expect(screenAfter(["gJ"])).toContain("alpha beta gamma  indented two");
  });
});

describe("vim edits: undo and redo", () => {
  it("undoes with u and redoes with Ctrl-R", () => {
    const undone = screenAfter(["x", "u"]);
    expect(undone).toContain("alpha beta gamma");

    const redone = screenAfter(["x", "u", "<ctrl+r>"]);
    expect(redone).toContain("lpha beta gamma");
  });

  it("undoes a counted delete as ONE step", () => {
    const s = screenAfter(["3x", "u"]);
    expect(s).toContain("alpha beta gamma");
  });

  it("undoes a counted join as ONE step", () => {
    const s = screenAfter(["J", "u"]);
    expect(s).toMatch(/1\s+alpha beta gamma\s*│/);
    expect(s).toMatch(/2\s+ {2}indented two/);
  });

  it("undoes a whole insert session as ONE step", () => {
    expect(screenAfter(["ihello world", "<escape>", "u"])).toContain("alpha beta gamma");
  });

  it("undoes an o-plus-typing session as ONE step", () => {
    const s = screenAfter(["oNEW LINE", "<escape>", "u"]);
    expect(s).not.toContain("NEW LINE");
    expect(s).toMatch(/2\s+ {2}indented two/);
  });

  it("undoes a counted change as ONE step", () => {
    expect(screenAfter(["5lCXYZ", "<escape>", "u"])).toContain("alpha beta gamma");
  });

  it("undoes a counted indent as ONE step", () => {
    const s = screenAfter(["2>>", "u"]);
    expect(s).toMatch(/1\s+alpha beta gamma/);
    expect(s).not.toMatch(/1\s+ {4}alpha/);
  });

  it("undoes a counted increment as ONE step", () => {
    expect(screenAfter(["2j", "10", "<ctrl+a>", "u"])).toContain("count 41 and 007");
  });

  it("applies a count to u", () => {
    // Three separate single-char deletes, then one 3u puts them all back.
    expect(screenAfter(["x", "x", "x", "3u"])).toContain("alpha beta gamma");
  });
});

describe("vim edits: indenting", () => {
  it("indents and dedents with >> and <<", () => {
    expect(screenAfter([">>"])).toMatch(/1\s+ {4}alpha beta gamma/);
    expect(screenAfter([">>", "<<"])).toMatch(/1\s+alpha beta gamma/);
  });

  it("indents a counted run of lines", () => {
    const s = screenAfter(["2>>"]);
    expect(s).toMatch(/1\s+ {4}alpha beta gamma/);
    expect(s).toMatch(/2\s+ {6}indented two/);
  });

  it("dedents only as far as the existing indent goes", () => {
    expect(screenAfter(["j<<"])).toMatch(/2\s+indented two/);
  });

  it("leaves the cursor on the first non-blank after >>", () => {
    expect(posAfter([">>"])).toBe("1:5");
  });

  it("reindents with == using the surrounding block", () => {
    const s = screenAfter(["j2=="], CODE);
    expect(s).toMatch(/2\s+ {4}bar\(\)/);
    expect(s).toMatch(/3\s+\}/);
  });
});

describe("vim edits: numbers", () => {
  it("increments and decrements with Ctrl-A and Ctrl-X", () => {
    expect(screenAfter(["2j", "<ctrl+a>"])).toContain("count 42 and 007");
    expect(screenAfter(["2j", "<ctrl+x>"])).toContain("count 40 and 007");
  });

  it("applies a count to Ctrl-A", () => {
    expect(screenAfter(["2j", "10", "<ctrl+a>"])).toContain("count 51 and 007");
  });

  it("finds the number after the cursor, not before it", () => {
    expect(screenAfter(["2j", "$", "<ctrl+a>"])).toContain("count 41 and 008");
  });

  it("preserves zero padding", () => {
    expect(screenAfter(["2j", "$", "<ctrl+x>"])).toContain("count 41 and 006");
  });

  it("treats a leading minus as a sign", () => {
    expect(screenAfter(["3j", "<ctrl+a>"])).toContain("temp -4 degrees");
    expect(screenAfter(["3j", "<ctrl+x>"])).toContain("temp -6 degrees");
  });

  it("does nothing when the line has no number after the cursor", () => {
    expect(screenAfter(["<ctrl+a>"])).toContain("alpha beta gamma");
  });

  it("leaves the cursor on the last digit", () => {
    // "count 41 and 007": 9 Ctrl-A turns 41 into 50, last digit at col 8.
    expect(posAfter(["2j", "9", "<ctrl+a>"])).toBe("3:8");
  });
});
