import { describe, it, expect, afterEach } from "vitest";
import * as tui from "./tui.js";
import {
  createTempDir,
  createTempFile,
  cleanupDir,
  readFile,
  fileExists,
} from "./helpers.js";
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

const FIXTURE = "foo one\nbar two\nfoo three\nbaz foo four\n";

// Ten identical lines, so a :%s across the file is unambiguously many edits.
const MANY =
  Array.from({ length: 10 }, (_, i) => `aa line ${i + 1} aa`).join("\n") + "\n";

let dir;

afterEach(() => {
  tui.kill();
  if (dir) cleanupDir(dir);
});

// The whole folder is opened, not just the file: the plugin filesystem API is
// scoped to the workspace roots, so `:w {file}` only has somewhere to write
// when a folder is part of the workspace.
function startVim(content = FIXTURE, name = "test.txt") {
  dir = createTempDir();
  const file = createTempFile(dir, name, content);
  tui.start("--plugin", VIM_PLUGIN, dir, file);
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
  tui.waitStable(350);
  const s = tui.snapshot();
  const { snapshots } = tui.run();
  return snapshots[s];
}

function pos(snapshot) {
  const m = snapshot.match(/Ln (\d+), Col (\d+)/);
  return m ? `${m[1]}:${m[2]}` : "none";
}

// Assert that a numbered gutter line holds exactly `text`.
function line(snapshot, n) {
  const m = snapshot.match(new RegExp(`\\u2502\\s+${n}\\s{2}(.*?)\\s*\\u2502`));
  return m ? m[1] : null;
}

describe("vim ex: the command line itself", () => {
  it(": opens the command line and shows what is typed", () => {
    startVim();
    tui.type(":noh");
    tui.waitStable(250);
    const s = tui.snapshot();
    const { snapshots } = tui.run();
    expect(snapshots[s]).toContain(":noh");
  });

  it("Esc closes the command line without running anything", () => {
    const s = screenAfter([":%s/foo/ZZZ/g", "<esc>"]);
    expect(line(s, 1)).toBe("foo one");
    expect(s).toContain("-- NORMAL --");
  });

  it("an unknown command is reported, not silently ignored", () => {
    expect(screenAfter([":frobnicate", "<enter>"])).toContain(
      "Not an editor command",
    );
  });
});

describe("vim ex: line addressing", () => {
  it(":{n} jumps to that line", () => {
    expect(pos(screenAfter([":3", "<enter>"]))).toBe("3:1");
  });

  it(":$ jumps to the last line", () => {
    // The trailing newline makes line 5 the (empty) last line.
    expect(pos(screenAfter([":$", "<enter>"]))).toBe("5:1");
  });

  it(":1 jumps back to the top", () => {
    expect(pos(screenAfter(["G", ":1", "<enter>"]))).toBe("1:1");
  });

  it("a line past the end clamps to the last line", () => {
    expect(pos(screenAfter([":999", "<enter>"]))).toBe("5:1");
  });
});

describe("vim ex: :w, :q and friends", () => {
  it(":w saves the buffer", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "test.txt", FIXTURE);
    tui.start("--plugin", VIM_PLUGIN, dir, file);
    tui.waitStable(300);
    send(["ihello", "<esc>", ":w", "<enter>"]);
    tui.waitStable(400);
    tui.run();
    expect(readFile(file)).toContain("hellofoo one");
  });

  it(":w {file} writes to another path, leaving the original alone", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "test.txt", FIXTURE);
    const other = join(dir, "copy.txt");
    tui.start("--plugin", VIM_PLUGIN, dir, file);
    tui.waitStable(300);
    send(["iX", "<esc>", ":w copy.txt", "<enter>"]);
    tui.waitStable(400);
    tui.run();
    expect(fileExists(other)).toBe(true);
    expect(readFile(other)).toContain("Xfoo one");
    // The buffer was never saved to its own path.
    expect(readFile(file)).toBe(FIXTURE);
  });

  it(":q! quits without prompting, even with unsaved changes", () => {
    startVim();
    send(["x", ":q!", "<enter>"]);
    tui.waitStable(300);
    const s = tui.snapshot();
    const { snapshots } = tui.run();
    // The editor is gone, so the screenshot after it never happened.
    expect(snapshots[s]).toBe("");
  });

  it(":q prompts when the buffer is dirty", () => {
    const s = screenAfter(["x", ":q", "<enter>"]);
    expect(s).toContain("Unsaved changes");
  });

  it(":wq saves and then quits", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "test.txt", FIXTURE);
    tui.start("--plugin", VIM_PLUGIN, dir, file);
    tui.waitStable(300);
    send(["iQ", "<esc>", ":wq", "<enter>"]);
    tui.waitStable(400);
    const s = tui.snapshot();
    const { snapshots } = tui.run();
    expect(readFile(file)).toContain("Qfoo one");
    expect(snapshots[s]).toBe("");
  });

  it(":x behaves like :wq", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "test.txt", FIXTURE);
    tui.start("--plugin", VIM_PLUGIN, dir, file);
    tui.waitStable(300);
    send(["iQ", "<esc>", ":x", "<enter>"]);
    tui.waitStable(400);
    tui.run();
    expect(readFile(file)).toContain("Qfoo one");
  });

  it(":e {file} opens another file in a new tab", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "test.txt", FIXTURE);
    createTempFile(dir, "other.txt", "other content here\n");
    tui.start("--plugin", VIM_PLUGIN, dir, file);
    tui.waitStable(300);
    send([":e other.txt", "<enter>"]);
    tui.waitStable(400);
    const s = tui.snapshot();
    const { snapshots } = tui.run();
    expect(snapshots[s]).toContain("other content here");
    expect(snapshots[s]).toContain("other.txt");
  });

  it(":e with no file name complains", () => {
    expect(screenAfter([":e", "<enter>"])).toContain("needs a file name");
  });
});

describe("vim ex: :noh, :reg and :marks", () => {
  it(":noh runs without reporting a missing command", () => {
    const s = screenAfter([":noh", "<enter>"]);
    expect(s).not.toContain("is not available");
    expect(s).toContain("-- NORMAL --");
  });

  it(":reg lists the registers", () => {
    const s = screenAfter(["yy", ":reg", "<enter>"]);
    expect(s).toContain("Registers");
    expect(s).toContain("foo one");
  });

  it(":marks lists the marks that have been set", () => {
    const s = screenAfter(["3G", "ma", ":marks", "<enter>"]);
    expect(s).toContain("Marks");
    expect(s).toContain("foo three");
  });
});

describe("vim ex: substitution", () => {
  it(":s replaces the first match on the current line only", () => {
    const s = screenAfter([":s/foo/X/", "<enter>"], "foo foo\nfoo foo\n");
    expect(line(s, 1)).toBe("X foo");
    expect(line(s, 2)).toBe("foo foo");
  });

  it("the g flag replaces every match on the line", () => {
    const s = screenAfter([":s/foo/X/g", "<enter>"], "foo foo foo\n");
    expect(line(s, 1)).toBe("X X X");
  });

  it(":%s covers the whole file", () => {
    const s = screenAfter([":%s/foo/X/g", "<enter>"]);
    expect(line(s, 1)).toBe("X one");
    expect(line(s, 3)).toBe("X three");
    expect(line(s, 4)).toBe("baz X four");
  });

  it("the i flag ignores case", () => {
    const s = screenAfter([":%s/foo/X/i", "<enter>"], "FOO\nFoo\nfoo\n");
    expect(line(s, 1)).toBe("X");
    expect(line(s, 2)).toBe("X");
    expect(line(s, 3)).toBe("X");
  });

  it("without the i flag the case must match", () => {
    const s = screenAfter([":%s/foo/X/", "<enter>"], "FOO\nfoo\n");
    expect(line(s, 1)).toBe("FOO");
    expect(line(s, 2)).toBe("X");
  });

  it("an explicit {n},{m} range limits the substitution", () => {
    const s = screenAfter([":2,3s/o/0/g", "<enter>"]);
    expect(line(s, 1)).toBe("foo one");
    expect(line(s, 2)).toBe("bar tw0");
    expect(line(s, 3)).toBe("f00 three");
    expect(line(s, 4)).toBe("baz foo four");
  });

  it(".,$ runs from the cursor to the end", () => {
    const s = screenAfter(["3G", ":.,$s/foo/X/g", "<enter>"]);
    expect(line(s, 1)).toBe("foo one");
    expect(line(s, 3)).toBe("X three");
    expect(line(s, 4)).toBe("baz X four");
  });

  it("an alternate delimiter works", () => {
    const s = screenAfter([":%s#foo#X#g", "<enter>"]);
    expect(line(s, 1)).toBe("X one");
    expect(line(s, 4)).toBe("baz X four");
  });

  it("& and \\1 refer to the match and its groups", () => {
    expect(line(screenAfter([":s/foo/[&]/", "<enter>"]), 1)).toBe("[foo] one");
    expect(
      line(screenAfter([":s/\\(f\\)\\(oo\\)/\\2\\1/", "<enter>"]), 1),
    ).toBe("oof one");
  });

  it("reports when the pattern is not found", () => {
    expect(screenAfter([":%s/nothing/X/", "<enter>"])).toContain(
      "Pattern not found",
    );
  });

  it("rejects an unsupported regex construct instead of misreading it", () => {
    const s = screenAfter([":%s/a\\|b/X/", "<enter>"]);
    expect(s).toContain("alternation");
    expect(line(s, 1)).toBe("foo one");
  });

  it("rejects a replacement that would need a line break", () => {
    expect(screenAfter([":s/foo/a\\rb/", "<enter>"])).toContain("line break");
  });
});

describe("vim ex: substitution undo atomicity", () => {
  it(":%s across ten lines is a single undo step", () => {
    const s = screenAfter([":%s/aa/ZZ/g", "<enter>", "u"], MANY);
    for (let i = 1; i <= 10; i++) {
      expect(line(s, i)).toBe(`aa line ${i} aa`);
    }
  });

  it("the substitution itself touched every line", () => {
    const s = screenAfter([":%s/aa/ZZ/g", "<enter>"], MANY);
    expect(line(s, 1)).toBe("ZZ line 1 ZZ");
    expect(line(s, 10)).toBe("ZZ line 10 ZZ");
  });

  it("one undo is enough: a second undo has nothing left to revert", () => {
    // If the run had been many undo steps, the first u would only have
    // restored the last line and the buffer would still be half-substituted.
    const s = screenAfter([":%s/aa/ZZ/g", "<enter>", "u"], MANY);
    expect(s).not.toContain("ZZ");
  });

  it("redo brings the whole substitution back in one step", () => {
    const s = screenAfter([":%s/aa/ZZ/g", "<enter>", "u", "<ctrl+r>"], MANY);
    expect(line(s, 1)).toBe("ZZ line 1 ZZ");
    expect(line(s, 10)).toBe("ZZ line 10 ZZ");
  });
});

describe("vim ex: substitution with the c flag", () => {
  it("prompts for each match and honours y, n and a", () => {
    startVim("a1\na2\na3\na4\n");
    tui.type(":%s/a/Z/gc");
    tui.press("enter");
    tui.waitStable(350);
    const prompt = tui.snapshot();
    tui.type("y");
    tui.waitStable(300);
    tui.type("n");
    tui.waitStable(300);
    tui.type("a");
    tui.waitStable(400);
    const done = tui.snapshot();
    const { snapshots } = tui.run();

    expect(snapshots[prompt]).toContain("(y/n/a/q)?");
    expect(line(snapshots[done], 1)).toBe("Z1");
    expect(line(snapshots[done], 2)).toBe("a2");
    expect(line(snapshots[done], 3)).toBe("Z3");
    expect(line(snapshots[done], 4)).toBe("Z4");
  });

  it("q stops the run, keeping only what was already accepted", () => {
    startVim("a1\na2\na3\n");
    tui.type(":%s/a/Z/gc");
    tui.press("enter");
    tui.waitStable(350);
    tui.type("y");
    tui.waitStable(300);
    tui.type("q");
    tui.waitStable(400);
    const done = tui.snapshot();
    const { snapshots } = tui.run();

    expect(line(snapshots[done], 1)).toBe("Z1");
    expect(line(snapshots[done], 2)).toBe("a2");
    expect(line(snapshots[done], 3)).toBe("a3");
  });

  it("a confirmed run is still a single undo step", () => {
    startVim("a1\na2\na3\n");
    tui.type(":%s/a/Z/gc");
    tui.press("enter");
    tui.waitStable(350);
    tui.type("a");
    tui.waitStable(400);
    tui.type("u");
    tui.waitStable(300);
    const done = tui.snapshot();
    const { snapshots } = tui.run();

    expect(line(snapshots[done], 1)).toBe("a1");
    expect(line(snapshots[done], 2)).toBe("a2");
    expect(line(snapshots[done], 3)).toBe("a3");
  });
});

describe("vim: editor integration keys", () => {
  it("gt and gT move between tabs", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "test.txt", FIXTURE);
    createTempFile(dir, "other.txt", "other content here\n");
    tui.start("--plugin", VIM_PLUGIN, dir, file);
    tui.waitStable(300);
    send([":e other.txt", "<enter>"]);
    tui.waitStable(400);
    send(["gT"]);
    tui.waitStable(350);
    const back = tui.snapshot();
    send(["gt"]);
    tui.waitStable(350);
    const fwd = tui.snapshot();
    const { snapshots } = tui.run();
    expect(snapshots[back]).toContain("foo one");
    expect(snapshots[fwd]).toContain("other content here");
  });

  it("za, zR and zM reach the fold commands without complaining", () => {
    for (const keys of [["za"], ["zR"], ["zM"]]) {
      const s = screenAfter(keys);
      expect(s).not.toContain("is not available");
      expect(s).toContain("-- NORMAL --");
    }
  });

  it("Ctrl-W w moves the focus and does not close the tab", () => {
    // Core binds Ctrl-W to tab.close; normal mode takes it as a prefix instead.
    const s = screenAfter(["<ctrl+w>", "w"]);
    expect(s).toContain("test.txt");
    expect(s).not.toContain("is not available");
  });
});
