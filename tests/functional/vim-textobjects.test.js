import { describe, it, expect, afterEach } from "vitest";
import * as tui from "./tui.js";
import { createTempDir, createTempFile, cleanupDir } from "./helpers.js";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

// Exercise the real shipped plugin, not an inline copy.
const VIM_PLUGIN = join(dirname(fileURLToPath(import.meta.url)), "..", "..", "plugins", "vim", "init.lua");

const WORDS = "alpha beta gamma delta\n" + "foo.bar baz\n";

const QUOTES = "say \"hello there\" now\n" + "call 'one two' end\n" + "tick `a b` done\n";

const BRACKETS = "xs = arr[one two] end\n" + "fn(a, b) tail\n" + "map{k: v} rest\n" + "cmp<T, U> post\n";

const CODE = "function foo(a, b) {\n" + "    bar()\n" + "    baz()\n" + "}\n";

const NESTED = "outer(one, inner(two, three), four)\n";

const HTML = "<div class=\"x\">\n" + "  <p>hello world</p>\n" + "</div>\n";

const PARA = "one\ntwo\n\nthree\nfour\n\nfive\n";

const SENT = "One two. Three four. Five six.\n";

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
// Strict on purpose: text-object scripts contain "<" and ">" as object keys.
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

function screenAfter(keys, content) {
  startVim(content);
  send(keys);
  tui.waitStable();
  const s = tui.snapshot();
  const { snapshots } = tui.run();
  return snapshots[s];
}

// Assert on a numbered gutter line's exact content.
function line(snapshot, n) {
  const m = snapshot.match(new RegExp(`\\u2502\\s+${n}\\s{2}(.*?)\\s*\\u2502`));
  return m ? m[1] : null;
}

// Same, but keeping trailing whitespace out of the picture by matching a prefix.
function lineOf(keys, content, n = 1) {
  return line(screenAfter(keys, content), n);
}

describe("text objects: words", () => {
  it("diw deletes the word under the cursor", () => {
    expect(lineOf(["w", "diw"], WORDS)).toBe("alpha  gamma delta");
  });

  it("daw takes the trailing whitespace too", () => {
    expect(lineOf(["w", "daw"], WORDS)).toBe("alpha gamma delta");
  });

  it("daw falls back to leading whitespace at the end of a line", () => {
    expect(lineOf(["$", "daw"], WORDS)).toBe("alpha beta gamma");
  });

  it("ciw replaces the word and leaves insert mode clean", () => {
    expect(lineOf(["w", "ciwZZ", "<escape>"], WORDS)).toBe("alpha ZZ gamma delta");
  });

  it("diw on punctuation takes only the punctuation run", () => {
    // Line 2 is "foo.bar baz"; the "." is its own word for iw.
    expect(lineOf(["j", "3l", "diw"], WORDS, 2)).toBe("foobar baz");
  });

  it("diW takes the whole WORD including punctuation", () => {
    expect(lineOf(["j", "diW"], WORDS, 2)).toBe(" baz");
  });

  it("daW takes the WORD and its trailing space", () => {
    expect(lineOf(["j", "daW"], WORDS, 2)).toBe("baz");
  });

  it("d2iw counts chunks, d2aw counts words", () => {
    // iw chunk 2 is the space after "alpha".
    expect(lineOf(["d2iw"], WORDS)).toBe("beta gamma delta");
    expect(lineOf(["d2aw"], WORDS)).toBe("gamma delta");
  });

  it("yiw then P duplicates a word", () => {
    expect(lineOf(["w", "yiwP"], WORDS)).toBe("alpha betabeta gamma delta");
  });

  it("gUiw uppercases a word in place", () => {
    expect(lineOf(["w", "gUiw"], WORDS)).toBe("alpha BETA gamma delta");
  });
});

describe("text objects: quotes", () => {
  it('di" empties a double-quoted string', () => {
    expect(lineOf(["f h", "di\""], QUOTES)).toBe('say "" now');
  });

  it('di" works from the start of the line, searching forward', () => {
    expect(lineOf(['di"'], QUOTES)).toBe('say "" now');
  });

  it('da" takes the quotes and the trailing space', () => {
    expect(lineOf(['da"'], QUOTES)).toBe("say now");
  });

  it("di' and da' handle single quotes", () => {
    expect(lineOf(["j", "di'"], QUOTES, 2)).toBe("call '' end");
    expect(lineOf(["j", "da'"], QUOTES, 2)).toBe("call end");
  });

  it("di` and da` handle backticks", () => {
    expect(lineOf(["jj", "di`"], QUOTES, 3)).toBe("tick `` done");
    expect(lineOf(["jj", "da`"], QUOTES, 3)).toBe("tick done");
  });

  it('ci" replaces the string contents', () => {
    expect(lineOf(['ci"ZZ', "<escape>"], QUOTES)).toBe('say "ZZ" now');
  });
});

describe("text objects: brackets", () => {
  it("di( and da( work from inside", () => {
    expect(lineOf(["j", "f,", "di("], BRACKETS, 2)).toBe("fn() tail");
    expect(lineOf(["j", "f,", "da("], BRACKETS, 2)).toBe("fn tail");
  });

  it("dib and dab are synonyms for di( and da(", () => {
    expect(lineOf(["j", "f,", "dib"], BRACKETS, 2)).toBe("fn() tail");
    expect(lineOf(["j", "f,", "dab"], BRACKETS, 2)).toBe("fn tail");
  });

  it("di) closes the same object as di(", () => {
    expect(lineOf(["j", "f,", "di)"], BRACKETS, 2)).toBe("fn() tail");
  });

  it("di[ and da[ work on square brackets", () => {
    expect(lineOf(["fo", "di["], BRACKETS, 1)).toBe("xs = arr[] end");
    expect(lineOf(["fo", "da["], BRACKETS, 1)).toBe("xs = arr end");
  });

  it("di] is a synonym for di[", () => {
    expect(lineOf(["fo", "di]"], BRACKETS, 1)).toBe("xs = arr[] end");
  });

  it("di{ , di} and diB work on braces", () => {
    expect(lineOf(["jj", "fk", "di{"], BRACKETS, 3)).toBe("map{} rest");
    expect(lineOf(["jj", "fk", "di}"], BRACKETS, 3)).toBe("map{} rest");
    expect(lineOf(["jj", "fk", "diB"], BRACKETS, 3)).toBe("map{} rest");
    expect(lineOf(["jj", "fk", "da{"], BRACKETS, 3)).toBe("map rest");
  });

  it("di< and da< work on angle brackets", () => {
    expect(lineOf(["3j", "fT", "di<"], BRACKETS, 4)).toBe("cmp<> post");
    expect(lineOf(["3j", "fT", "da<"], BRACKETS, 4)).toBe("cmp post");
    expect(lineOf(["3j", "fT", "di>"], BRACKETS, 4)).toBe("cmp<> post");
  });

  it("a count selects the enclosing pair", () => {
    // "w" only occurs inside inner(...), so fw parks the cursor one level deep.
    expect(lineOf(["fw", "di("], NESTED)).toBe("outer(one, inner(), four)");
    expect(lineOf(["fw", "d2i("], NESTED)).toBe("outer()");
  });

  it("ci( leaves you inside the brackets in insert mode", () => {
    expect(lineOf(["j", "f,", "ci(ZZ", "<escape>"], BRACKETS, 2)).toBe("fn(ZZ) tail");
  });
});

describe("text objects: blocks spanning lines", () => {
  // Vim turns an inner block into a linewise one when the open brace ends a
  // line and the close brace starts one -- that is what makes di{ clear a body.
  it("di{ clears a multi-line block linewise", () => {
    const s = screenAfter(["3j", "di{"], CODE);
    expect(line(s, 1)).toBe("function foo(a, b) {");
    expect(line(s, 2)).toBe("}");
  });

  it("da{ takes the braces with it", () => {
    expect(line(screenAfter(["3j", "da{"], CODE), 1)).toBe("function foo(a, b)");
  });

  it("di{ works from inside the block too", () => {
    const s = screenAfter(["j", "di{"], CODE);
    expect(line(s, 1)).toBe("function foo(a, b) {");
    expect(line(s, 2)).toBe("}");
  });

  it(">i{ indents the block body", () => {
    const s = screenAfter(["j", ">i{"], CODE);
    expect(line(s, 2)).toBe("        bar()");
    expect(line(s, 3)).toBe("        baz()");
  });

  it("di( on the signature only touches the signature", () => {
    const s = screenAfter(["f,", "di("], CODE);
    expect(line(s, 1)).toBe("function foo() {");
    expect(line(s, 2)).toBe("    bar()");
  });
});

describe("text objects: tags", () => {
  it("dit empties the tag body", () => {
    const s = screenAfter(["j", "fh", "dit"], HTML);
    expect(line(s, 2)).toBe("  <p></p>");
  });

  it("dat removes the tag as well", () => {
    const s = screenAfter(["j", "fh", "dat"], HTML);
    expect(line(s, 2)).toBe("");
    expect(line(s, 1)).toBe('<div class="x">');
  });

  it("cit replaces the tag body", () => {
    const s = screenAfter(["j", "fh", "citZZ", "<escape>"], HTML);
    expect(line(s, 2)).toBe("  <p>ZZ</p>");
  });

  it("dit picks the innermost enclosing tag", () => {
    // On the "p" of <p>, the innermost pair containing the cursor is <p>...</p>.
    const s = screenAfter(["j", "fp", "dit"], HTML);
    expect(line(s, 2)).toBe("  <p></p>");
  });
});

describe("text objects: paragraphs", () => {
  it("dip takes the block of non-blank lines", () => {
    const s = screenAfter(["dip"], PARA);
    expect(line(s, 1)).toBe("");
    expect(line(s, 2)).toBe("three");
  });

  it("dap takes the following blank line too", () => {
    const s = screenAfter(["dap"], PARA);
    expect(line(s, 1)).toBe("three");
    expect(line(s, 2)).toBe("four");
  });

  it("dap on a later paragraph works the same", () => {
    const s = screenAfter(["3j", "dap"], PARA);
    expect(line(s, 1)).toBe("one");
    expect(line(s, 2)).toBe("two");
    expect(line(s, 3)).toBe("");
    expect(line(s, 4)).toBe("five");
  });

  it("dip on a blank line removes the gap", () => {
    const s = screenAfter(["jj", "dip"], PARA);
    expect(line(s, 2)).toBe("two");
    expect(line(s, 3)).toBe("three");
  });

  it("yip then P duplicates a paragraph", () => {
    const s = screenAfter(["yipP"], PARA);
    expect(line(s, 1)).toBe("one");
    expect(line(s, 2)).toBe("two");
    expect(line(s, 3)).toBe("one");
    expect(line(s, 4)).toBe("two");
  });

  it("gUip uppercases a paragraph", () => {
    const s = screenAfter(["gUip"], PARA);
    expect(line(s, 1)).toBe("ONE");
    expect(line(s, 2)).toBe("TWO");
    expect(line(s, 4)).toBe("three");
  });
});

describe("text objects: sentences", () => {
  it("das removes a sentence and its trailing space", () => {
    expect(lineOf(["das"], SENT)).toBe("Three four. Five six.");
  });

  it("dis leaves the space that followed the sentence", () => {
    expect(lineOf(["dis"], SENT)).toBe(" Three four. Five six.");
  });

  it("das works on a later sentence", () => {
    expect(lineOf(["fT", "das"], SENT)).toBe("One two. Five six.");
  });

  it("cis replaces a sentence", () => {
    expect(lineOf(["cisZZ", "<escape>"], SENT)).toBe("ZZ Three four. Five six.");
  });

  it("d2as removes two sentences", () => {
    expect(lineOf(["d2as"], SENT)).toBe("Five six.");
  });
});

describe("text objects: undo atomicity", () => {
  it("undoes diw as ONE step", () => {
    expect(lineOf(["w", "diw", "u"], WORDS)).toBe("alpha beta gamma delta");
  });

  it("undoes ciw plus typing as ONE step", () => {
    expect(lineOf(["w", "ciwZZZ", "<escape>", "u"], WORDS)).toBe("alpha beta gamma delta");
  });

  it("undoes di{ as ONE step", () => {
    const s = screenAfter(["3j", "di{", "u"], CODE);
    expect(line(s, 2)).toBe("    bar()");
    expect(line(s, 3)).toBe("    baz()");
  });

  it("undoes dap as ONE step", () => {
    const s = screenAfter(["dap", "u"], PARA);
    expect(line(s, 1)).toBe("one");
    expect(line(s, 3)).toBe("");
    expect(line(s, 4)).toBe("three");
  });

  it("undoes dit as ONE step", () => {
    expect(line(screenAfter(["j", "fh", "dit", "u"], HTML), 2)).toBe("  <p>hello world</p>");
  });

  it("undoes gUip as ONE step", () => {
    expect(line(screenAfter(["gUip", "u"], PARA), 1)).toBe("one");
  });
});

describe("text objects: no-ops", () => {
  it("di( with no enclosing parens does nothing", () => {
    expect(lineOf(["di("], WORDS)).toBe("alpha beta gamma delta");
  });

  it("dit outside any tag does nothing", () => {
    expect(lineOf(["dit"], WORDS)).toBe("alpha beta gamma delta");
  });

  it("an unknown object key cancels the operator", () => {
    expect(lineOf(["diq"], WORDS)).toBe("alpha beta gamma delta");
    // ...and the cancelled operator must not swallow the next command.
    expect(lineOf(["diq", "x"], WORDS)).toBe("lpha beta gamma delta");
  });

  it("i and a still enter insert mode without a pending operator", () => {
    expect(lineOf(["iZZ", "<escape>"], WORDS)).toBe("ZZalpha beta gamma delta");
    expect(lineOf(["aZZ", "<escape>"], WORDS)).toBe("aZZlpha beta gamma delta");
  });
});
