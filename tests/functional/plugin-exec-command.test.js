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

describe("ttt.exec_command plugin API", () => {
  it("executes a built-in command and returns true", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "test.txt", "line one\nline two\nline three\n");
    const plugin = writePlugin(dir, "test.lua", `
      local ttt = require("ttt")
      ttt.register({
        commands = {
          { id = "test.run", title = "Test Run Exec", handler = function()
              local ok = ttt.exec_command("editor.undo")
              ttt.set_status_item("left", "result", ok and "OK" or "FAIL", { priority = 10 })
            end
          }
        }
      })
    `);

    tui.start("--plugin", plugin, file);
    tui.waitStable(300);
    tui.type("hello");
    tui.waitStable();
    tui.exec("Test Run Exec");
    tui.waitStable();
    const s = tui.snapshot();
    const { snapshots } = tui.run();

    const lastLine = snapshots[s].split("\n").pop();
    expect(lastLine).toContain("OK");
  });

  it("returns false for unknown command ID", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "test.txt", "hello\n");
    const plugin = writePlugin(dir, "test.lua", `
      local ttt = require("ttt")
      ttt.register({
        commands = {
          { id = "test.run", title = "Test Run Bad", handler = function()
              local ok = ttt.exec_command("nonexistent.command")
              ttt.set_status_item("left", "result", ok and "FOUND" or "NOT_FOUND", { priority = 10 })
            end
          }
        }
      })
    `);

    tui.start("--plugin", plugin, file);
    tui.waitStable(300);
    tui.exec("Test Run Bad");
    tui.waitStable();
    const s = tui.snapshot();
    const { snapshots } = tui.run();

    const lastLine = snapshots[s].split("\n").pop();
    expect(lastLine).toContain("NOT_FOUND");
  });
});
