import { describe, it, expect, afterEach } from "vitest";
import * as tui from "./tui.js";
import { createTempDir, createTempFile, cleanupDir } from "./helpers.js";
import { writeFileSync } from "node:fs";
import { join } from "node:path";

let dir;

afterEach(() => {
  tui.kill();
  if (dir) cleanupDir(dir);
});

function writePlugin(dir, name, lua) {
  const path = join(dir, name);
  writeFileSync(path, lua, "utf8");
  return path;
}

describe("editor.get_line plugin API", () => {
  it("returns the text of a specific line (1-based)", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "test.txt", "alpha\nbeta\ngamma\n");
    const plugin = writePlugin(dir, "test.lua", `
      local ttt = require("ttt")
      local editor = require("ttt.editor")
      ttt.register({
        commands = {
          { id = "test.getline", title = "Test GetLine", handler = function()
              local line2 = editor.get_line(2)
              ttt.set_status_item("left", "result", "L2:" .. line2, { priority = 10 })
            end
          }
        }
      })
    `);

    tui.start("--plugin", plugin, file);
    tui.waitStable(300);
    tui.exec("Test GetLine");
    tui.waitStable();
    const s = tui.snapshot();
    const { snapshots } = tui.run();

    const lastLine = snapshots[s].split("\n").pop();
    expect(lastLine).toContain("L2:beta");
  });
});

describe("editor.line_count plugin API", () => {
  it("returns the total number of lines", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "test.txt", "one\ntwo\nthree\nfour\n");
    const plugin = writePlugin(dir, "test.lua", `
      local ttt = require("ttt")
      local editor = require("ttt.editor")
      ttt.register({
        commands = {
          { id = "test.count", title = "Test Count", handler = function()
              local n = editor.line_count()
              ttt.set_status_item("left", "result", "LINES:" .. n, { priority = 10 })
            end
          }
        }
      })
    `);

    tui.start("--plugin", plugin, file);
    tui.waitStable(300);
    tui.exec("Test Count");
    tui.waitStable();
    const s = tui.snapshot();
    const { snapshots } = tui.run();

    const lastLine = snapshots[s].split("\n").pop();
    expect(lastLine).toContain("LINES:");
  });
});

describe("editor.set_line plugin API", () => {
  it("replaces a line's content", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "test.txt", "aaa\nbbb\nccc\n");
    const plugin = writePlugin(dir, "test.lua", `
      local ttt = require("ttt")
      local editor = require("ttt.editor")
      ttt.register({
        commands = {
          { id = "test.setline", title = "Test SetLine", handler = function()
              editor.set_line(2, "REPLACED")
            end
          }
        }
      })
    `);

    tui.start("--plugin", plugin, file);
    tui.waitStable(300);
    tui.exec("Test SetLine");
    tui.waitStable();
    const s = tui.snapshot();
    const { snapshots } = tui.run();

    expect(snapshots[s]).toContain("REPLACED");
    expect(snapshots[s]).not.toContain("bbb");
  });
});
