import { describe, it, expect, afterEach } from "vitest";
import * as tui from "./tui.js";
import { createTempDir, createTempFile, cleanupDir } from "./helpers.js";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

// Exercise the real shipped plugin, not an inline copy.
const VIM_PLUGIN = join(dirname(fileURLToPath(import.meta.url)), "..", "..", "plugins", "vim", "init.lua");

// Columns of interest in FIXTURE line 1, "alpha beta gamma delta":
//   a1 l2 p3 h4 a5 _6 b7 e8 t9 a10 _11 g12 a13 m14 m15 a16 _17 d18 e19 l20 t21 a22
// Line 2 is indented and has trailing blanks so ^/$/g_ are all distinguishable.
// Line 3 mixes punctuation and words so w/W and e/E differ. Line 4 is blank.
const FIXTURE = "alpha beta gamma delta\n" + "  indented line two   \n" + "foo.bar(baz) qux\n" + "\n" + "last line here\n";

// 200 numbered lines, for screen-position and scrolling motions.
const LONG = Array.from({ length: 200 }, (_, i) => `line ${i + 1}`).join("\n") + "\n";

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

// The status bar is the cleanest cursor readout the harness has.
function pos(snapshot) {
  const m = snapshot.match(/Ln (\d+), Col (\d+)/);
  return m ? `${m[1]}:${m[2]}` : "none";
}

// Run one keystroke script against FIXTURE and return "line:col".
function posAfter(keys, content = FIXTURE) {
  startVim(content);
  for (const k of keys) {
    if (k.startsWith("<")) {
      tui.press(k.slice(1, -1));
    } else {
      tui.type(k);
    }
  }
  tui.waitStable();
  const s = tui.snapshot();
  const { snapshots } = tui.run();
  return pos(snapshots[s]);
}

describe("vim motions: characters and lines", () => {
  it("moves with h and l", () => {
    expect(posAfter(["l"])).toBe("1:2");
    expect(posAfter(["5l"])).toBe("1:6");
    expect(posAfter(["5l", "2h"])).toBe("1:4");
  });

  it("clamps h and l inside the line", () => {
    // Normal mode puts the cursor ON a character, so col never exceeds #line.
    expect(posAfter(["99l"])).toBe("1:22");
    expect(posAfter(["h"])).toBe("1:1");
  });

  it("moves with j and k, including counts", () => {
    expect(posAfter(["2j"])).toBe("3:1");
    expect(posAfter(["3j", "2k"])).toBe("2:1");
    expect(posAfter(["9j"])).toBe("6:1"); // clamped to the last line
  });

  it("keeps the goal column across a short line", () => {
    const short = "abcdefghij\nxy\nabcdefghij\n";
    expect(posAfter(["9l", "j"], short)).toBe("2:2");
    expect(posAfter(["9l", "2j"], short)).toBe("3:10");
  });

  it("moves to line starts and ends with 0 ^ $ g_", () => {
    expect(posAfter(["j", "0"])).toBe("2:1");
    expect(posAfter(["j", "^"])).toBe("2:3");
    expect(posAfter(["j", "$"])).toBe("2:22");
    expect(posAfter(["j", "g_"])).toBe("2:19");
  });

  it("keeps the cursor at end-of-line after $ then j", () => {
    expect(posAfter(["$", "j"])).toBe("2:22");
  });

  it("moves to first non-blank of neighbouring lines with + and -", () => {
    expect(posAfter(["+"])).toBe("2:3");
    expect(posAfter(["2+"])).toBe("3:1");
    expect(posAfter(["G", "-"])).toBe("5:1");
  });
});

describe("vim motions: words", () => {
  it("moves forward by word with w and counts", () => {
    expect(posAfter(["w"])).toBe("1:7");
    expect(posAfter(["3w"])).toBe("1:18");
  });

  it("treats punctuation as its own word for w but not W", () => {
    // line 3 is "foo.bar(baz) qux"
    expect(posAfter(["2j", "w"])).toBe("3:4");
    expect(posAfter(["2j", "3w"])).toBe("3:8");
    expect(posAfter(["2j", "W"])).toBe("3:14");
  });

  it("moves backward by word with b and B", () => {
    expect(posAfter(["3w", "b"])).toBe("1:12");
    expect(posAfter(["3w", "2b"])).toBe("1:7");
    expect(posAfter(["2j", "$", "B"])).toBe("3:14");
  });

  it("moves to word ends with e and E", () => {
    expect(posAfter(["e"])).toBe("1:5");
    expect(posAfter(["3e"])).toBe("1:16");
    expect(posAfter(["2j", "e"])).toBe("3:3");
    expect(posAfter(["2j", "E"])).toBe("3:12");
  });

  it("moves to previous word ends with ge and gE", () => {
    // 4w lands on line 2's first word, so ge/gE walk back into line 1.
    expect(posAfter(["4w", "ge"])).toBe("1:22");
    expect(posAfter(["4w", "gE"])).toBe("1:22");
    expect(posAfter(["3w", "ge"])).toBe("1:16");
  });

  it("crosses lines and stops on blank lines", () => {
    // line 3 has 5 words ("foo" "." "bar" "(" "baz" ...); walking off its end
    // lands on the blank line 4, which Vim counts as a word.
    expect(posAfter(["2j", "$", "w"])).toBe("4:1");
    expect(posAfter(["2j", "$", "2w"])).toBe("5:1");
  });
});

describe("vim motions: buffer positions", () => {
  it("jumps with gg and G", () => {
    expect(posAfter(["3j", "gg"])).toBe("1:1");
    expect(posAfter(["G"])).toBe("6:1"); // trailing newline makes a 6th, empty line
    expect(posAfter(["3G"])).toBe("3:1");
    expect(posAfter(["2gg"])).toBe("2:3"); // first non-blank of the indented line
    expect(posAfter(["99G"])).toBe("6:1");
  });

  it("moves by paragraph with { and }", () => {
    expect(posAfter(["}"])).toBe("4:1");
    expect(posAfter(["2}"])).toBe("6:1");
    expect(posAfter(["G", "{"])).toBe("4:1");
    expect(posAfter(["G", "2{"])).toBe("1:1");
  });

  it("jumps between brackets with %", () => {
    expect(posAfter(["2j", "f(", "%"])).toBe("3:12");
    expect(posAfter(["2j", "f)", "%"])).toBe("3:8");
  });
});

describe("vim motions: character search on the line", () => {
  it("finds forward with f and t", () => {
    expect(posAfter(["fg"])).toBe("1:12");
    expect(posAfter(["fa"])).toBe("1:5");
    expect(posAfter(["2fa"])).toBe("1:10");
    expect(posAfter(["3fa"])).toBe("1:13");
    expect(posAfter(["ta"])).toBe("1:4");
    expect(posAfter(["2ta"])).toBe("1:9");
  });

  it("finds backward with F and T", () => {
    expect(posAfter(["$", "Fa"])).toBe("1:16");
    expect(posAfter(["$", "2Fa"])).toBe("1:13");
    expect(posAfter(["$", "Ta"])).toBe("1:17");
  });

  it("does not move when the character is absent on the line", () => {
    expect(posAfter(["fz"])).toBe("1:1");
    expect(posAfter(["$", "Fz"])).toBe("1:22");
  });

  it("never leaves the current line", () => {
    // "second"-style characters exist on other lines but f must stay put.
    expect(posAfter(["j", "fq"])).toBe("2:1");
  });

  // Sending a literal ";" relies on tui.js driving --exec-split-on with a
  // non-printable separator; with the default ";" separator these keystrokes
  // would be swallowed as command boundaries by the harness itself.
  it("repeats the last find with ;", () => {
    expect(posAfter(["fa", ";"])).toBe("1:10");
    expect(posAfter(["fa", ";", ";"])).toBe("1:13");
    expect(posAfter(["$", "Fa", ";"])).toBe("1:13");
  });

  it("advances instead of standing still when repeating a till-find", () => {
    expect(posAfter(["ta", ";"])).toBe("1:9");
    expect(posAfter(["$", "Ta", ";"])).toBe("1:14");
  });

  it("repeats a find in the opposite direction with ,", () => {
    expect(posAfter(["2fa", ","])).toBe("1:5");
    expect(posAfter(["$", "2Fa", ","])).toBe("1:16");
    expect(posAfter(["fa", ";", ",", ";"])).toBe("1:10");
  });
});

describe("vim motions: screen positions", () => {
  it("moves to top, middle and bottom of the screen with H M L", () => {
    // 120x40 gives a 34-line editor viewport.
    expect(posAfter(["H"], LONG)).toBe("1:1");
    expect(posAfter(["M"], LONG)).toBe("17:1");
    expect(posAfter(["L"], LONG)).toBe("34:1");
  });

  it("accepts counts for H and L", () => {
    expect(posAfter(["5H"], LONG)).toBe("5:1");
    expect(posAfter(["3L"], LONG)).toBe("32:1");
  });
});

describe("vim motions: scrolling", () => {
  function screenAfter(keys, content = LONG) {
    startVim(content, "long.txt");
    for (const k of keys) {
      if (k.startsWith("<")) {
        tui.press(k.slice(1, -1));
      } else {
        tui.type(k);
      }
      tui.waitStable(80);
    }
    tui.waitStable();
    const s = tui.snapshot();
    const { snapshots } = tui.run();
    return snapshots[s];
  }

  it("scrolls a half page with Ctrl-D and Ctrl-U", () => {
    const down = screenAfter(["<ctrl+d>"]);
    expect(pos(down)).toBe("18:1");
    expect(down).toMatch(/\bline 18\b/);
    expect(down).not.toMatch(/\bline 17\b/);

    expect(pos(screenAfter(["<ctrl+d>", "<ctrl+u>"]))).toBe("1:1");
  });

  it("scrolls a full page with Ctrl-F", () => {
    const down = screenAfter(["<ctrl+f>"]);
    expect(pos(down)).toBe("35:1");
    expect(down).toMatch(/\bline 35\b/);
    expect(down).not.toMatch(/\bline 34\b/);
  });

  // Ctrl-B is a core force key (sidebar.toggle) and outranks plugin key
  // interceptors, so PgUp is the reachable equivalent.
  it("pages with PgDn and PgUp", () => {
    expect(pos(screenAfter(["<pgdn>"]))).toBe("35:1");
    expect(pos(screenAfter(["<pgdn>", "<pgup>"]))).toBe("1:1");
  });

  it("scrolls one line without moving the cursor using Ctrl-Y", () => {
    const view = screenAfter(["<ctrl+f>", "<ctrl+y>"]);
    expect(pos(view)).toBe("35:1"); // cursor stayed
    expect(view).toMatch(/\bline 34\b/); // but the view moved up one
  });

  it("drags the cursor along with Ctrl-E when it would leave the screen", () => {
    const view = screenAfter(["<ctrl+e>"]);
    expect(pos(view)).toBe("2:1");
    expect(view).not.toMatch(/\bline 1\b/);
  });

  it("repositions the view around the cursor with zz, zt and zb", () => {
    const zt = screenAfter(["100G", "zt"]);
    expect(pos(zt)).toBe("100:1");
    expect(zt).toMatch(/\bline 100\b/);
    expect(zt).not.toMatch(/\bline 99\b/);

    const zz = screenAfter(["100G", "zz"]);
    expect(pos(zz)).toBe("100:1");
    expect(zz).toMatch(/\bline 83\b/);
    expect(zz).not.toMatch(/\bline 82\b/);

    const zb = screenAfter(["100G", "zb"]);
    expect(pos(zb)).toBe("100:1");
    expect(zb).toMatch(/\bline 67\b/);
    expect(zb).not.toMatch(/\bline 66\b/);
  });
});

describe("vim motions: counts and cursor semantics", () => {
  it("lands the cursor where the buffer says it should", () => {
    startVim();
    tui.type("3wiX");
    tui.waitStable();
    const s = tui.snapshot();
    const { snapshots } = tui.run();

    expect(snapshots[s]).toContain("alpha beta gamma Xdelta");
  });

  it("does not leak a count into the following motion", () => {
    expect(posAfter(["3l", "l"])).toBe("1:5");
  });

  it("does not treat a leading 0 as a count", () => {
    expect(posAfter(["5l", "0"])).toBe("1:1");
  });

  it("clamps the cursor back onto a character when leaving insert mode", () => {
    // `$` then `i` then Right puts the cursor one past the end (legal in
    // insert mode); Esc must pull it back onto the last character.
    startVim();
    tui.type("$");
    tui.type("i");
    tui.press("right");
    tui.waitStable(100);
    const insert = tui.snapshot();
    tui.press("escape");
    tui.waitStable();
    const normal = tui.snapshot();
    const { snapshots } = tui.run();

    expect(pos(snapshots[insert])).toBe("1:23");
    expect(pos(snapshots[normal])).toBe("1:22");
  });

  it("does not type motion keys into the buffer", () => {
    startVim();
    tui.type("3wbe0$Ggg");
    tui.waitStable();
    const s = tui.snapshot();
    const { snapshots } = tui.run();

    expect(snapshots[s]).toContain("alpha beta gamma delta");
    expect(snapshots[s]).not.toContain("3wbe");
  });
});
