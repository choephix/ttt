import test from "node:test";
import assert from "node:assert/strict";
import {
  cpSync,
  mkdirSync,
  mkdtempSync,
  readFileSync,
  realpathSync,
  writeFileSync,
} from "node:fs";
import { spawnSync } from "node:child_process";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import {
  discoverInstances,
  instanceKey,
  resolveTarget,
  saveAlias,
  writeCommand,
} from "../ttt-open.mjs";

const here = dirname(fileURLToPath(import.meta.url));
const pluginRoot = resolve(here, "..");
const repoRoot = resolve(pluginRoot, "../..");
const binary = join(repoRoot, "bin", "ttt");

test("instanceKey matches the plugin's stable target naming", () => {
  assert.equal(instanceKey({ TTT_REMOTE_NAME: "Main TTT" }), "name_Main_TTT");
  assert.equal(
    instanceKey({ HERDR_SESSION: "hustles", HERDR_PANE_ID: "w3:p2" }),
    "herdr_hustles_w3_p2",
  );
  assert.equal(instanceKey({}), null);
});

test("writeCommand atomically queues absolute files for one instance", async (t) => {
  const root = mkdtempSync(join(tmpdir(), "ttt-remote-open-sender-"));
  t.after(() => import("node:fs").then(({ rmSync }) => rmSync(root, { recursive: true, force: true })));
  mkdirSync(join(root, "mailboxes"));

  const commandPath = writeCommand({
    pluginDir: root,
    targetKey: "name_test",
    files: [{ path: "/tmp/one.txt" }, { path: "/tmp/two.txt", line: 7 }],
  });

  assert.match(commandPath, /name_test--.+\.json$/);
  assert.deepEqual(JSON.parse(readFileSync(commandPath, "utf8")), {
    files: [{ path: "/tmp/one.txt" }, { path: "/tmp/two.txt", line: 7 }],
  });
});

test("discoverInstances finds targetable TTT processes and ignores other commands", async (t) => {
  const root = mkdtempSync(join(tmpdir(), "ttt-remote-open-proc-"));
  t.after(() => import("node:fs").then(({ rmSync }) => rmSync(root, { recursive: true, force: true })));
  mkdirSync(join(root, "101"));
  mkdirSync(join(root, "202"));
  writeFileSync(join(root, "101", "cmdline"), "/home/cx/.local/bin/ttt\0/mnt/vault\0");
  writeFileSync(
    join(root, "101", "environ"),
    "HERDR_SESSION=hustles\0HERDR_PANE_ID=w3:p2\0PWD=/mnt/vault\0",
  );
  writeFileSync(join(root, "202", "cmdline"), "bash\0");
  writeFileSync(join(root, "202", "environ"), "PWD=/tmp\0");

  assert.deepEqual(discoverInstances(root), [
    {
      pid: 101,
      key: "herdr_hustles_w3_p2",
      selector: "hustles:w3:p2",
      cwd: "/mnt/vault",
    },
  ]);
});

test("a persisted main alias resolves to the same live instance", async (t) => {
  const root = mkdtempSync(join(tmpdir(), "ttt-remote-open-alias-"));
  t.after(() => import("node:fs").then(({ rmSync }) => rmSync(root, { recursive: true, force: true })));
  const instance = {
    pid: 101,
    key: "herdr_hustles_w3_p2",
    selector: "hustles:w3:p2",
    cwd: "/mnt/vault",
  };

  saveAlias(root, "main", instance.key);
  assert.deepEqual(resolveTarget("main", root, [instance]), instance);
  assert.deepEqual(resolveTarget("hustles:w3:p2", root, [instance]), instance);
  assert.equal(resolveTarget("missing", root, [instance]), null);
});

test("installer links versioned sources into a real global plugin directory", async (t) => {
  const root = mkdtempSync(join(tmpdir(), "ttt-remote-open-install-"));
  t.after(() => import("node:fs").then(({ rmSync }) => rmSync(root, { recursive: true, force: true })));
  const configRoot = join(root, "config");

  const result = spawnSync("node", [join(pluginRoot, "install.mjs")], {
    env: { ...process.env, HOME: root, XDG_CONFIG_HOME: configRoot },
    encoding: "utf8",
  });
  assert.equal(result.status, 0, result.stderr);

  const installed = join(configRoot, "ttt", "plugins", "remote-open");
  assert.equal(realpathSync(join(installed, "init.lua")), join(pluginRoot, "init.lua"));
  assert.equal(
    realpathSync(join(installed, "plugin.ttt.json")),
    join(pluginRoot, "plugin.ttt.json"),
  );
  assert.equal(realpathSync(join(root, ".local", "bin", "ttt-open")), join(pluginRoot, "ttt-open.mjs"));
  assert.equal(realpathSync(join(installed, "mailboxes")), join(installed, "mailboxes"));
});

test("plugin consumes its instance mailbox and opens every requested file as a tab", (t) => {
  const root = mkdtempSync(join(tmpdir(), "ttt-remote-open-plugin-"));
  t.after(() => import("node:fs").then(({ rmSync }) => rmSync(root, { recursive: true, force: true })));

  const copiedPlugin = join(root, "remote-open");
  const mailboxes = join(copiedPlugin, "mailboxes");
  mkdirSync(mailboxes, { recursive: true });
  cpSync(join(pluginRoot, "init.lua"), join(copiedPlugin, "init.lua"));

  const first = join(root, "first.txt");
  const second = join(root, "second.txt");
  const initial = join(root, "initial.txt");
  writeFileSync(first, "first\n");
  writeFileSync(second, "one\ntwo\nthree\n");
  writeFileSync(initial, "initial\n");
  writeFileSync(
    join(mailboxes, "name_test--command.json"),
    JSON.stringify({ files: [{ path: first }, { path: second, line: 2 }] }),
  );

  const debugPath = join(root, "debug.json");
  const result = spawnSync(
    binary,
    [
      "--plugin",
      join(copiedPlugin, "init.lua"),
      initial,
      "--exec",
      `wait 500; debug ${debugPath}; quit`,
    ],
    {
      env: {
        ...process.env,
        HOME: root,
        TTT_CONFIG_DIR: join(root, "config"),
        TTT_REMOTE_NAME: "test",
      },
      encoding: "utf8",
      timeout: 5000,
    },
  );
  assert.equal(result.status, 0, result.stderr);

  const state = JSON.parse(readFileSync(debugPath, "utf8"));
  assert.deepEqual(
    state.tabs.map((tab) => tab.path),
    [initial, first, second],
    `stderr: ${result.stderr}\nplugin output: ${JSON.stringify(state.output)}`,
  );
  assert.equal(state.buffer.path, second);
  assert.equal(state.cursor.line, 1);
  assert.equal(readFileSync(join(mailboxes, "name_test--command.json"), "utf8"), "");
});
