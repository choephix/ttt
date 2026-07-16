#!/usr/bin/env node

import {
  chmodSync,
  existsSync,
  lstatSync,
  mkdirSync,
  symlinkSync,
  unlinkSync,
} from "node:fs";
import { homedir } from "node:os";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const sourceDir = dirname(fileURLToPath(import.meta.url));
const configRoot = process.env.XDG_CONFIG_HOME || join(homedir(), ".config");
const pluginDir = join(configRoot, "ttt", "plugins", "remote-open");
const binDir = join(homedir(), ".local", "bin");

function ensureRealDirectory(path) {
  if (existsSync(path) && lstatSync(path).isSymbolicLink()) {
    throw new Error(`${path} is a directory symlink; remove it before installing`);
  }
  mkdirSync(path, { recursive: true });
}

function link(source, destination) {
  if (existsSync(destination)) {
    const stat = lstatSync(destination);
    if (!stat.isSymbolicLink()) {
      throw new Error(`Refusing to replace non-symlink ${destination}`);
    }
    unlinkSync(destination);
  }
  symlinkSync(source, destination, "file");
}

try {
  ensureRealDirectory(pluginDir);
  ensureRealDirectory(join(pluginDir, "mailboxes"));
  ensureRealDirectory(binDir);

  link(join(sourceDir, "init.lua"), join(pluginDir, "init.lua"));
  link(join(sourceDir, "plugin.ttt.json"), join(pluginDir, "plugin.ttt.json"));
  chmodSync(join(sourceDir, "ttt-open.mjs"), 0o755);
  link(join(sourceDir, "ttt-open.mjs"), join(binDir, "ttt-open"));

  console.log(`Installed Remote Open plugin in ${pluginDir}`);
  console.log(`Installed ttt-open in ${join(binDir, "ttt-open")}`);
} catch (error) {
  console.error(`remote-open install: ${error.message}`);
  process.exitCode = 1;
}
