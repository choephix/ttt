#!/usr/bin/env node

import {
  existsSync,
  mkdirSync,
  readFileSync,
  readdirSync,
  realpathSync,
  renameSync,
  unlinkSync,
  writeFileSync,
} from "node:fs";
import { homedir } from "node:os";
import { basename, isAbsolute, join, resolve } from "node:path";
import { setTimeout as delay } from "node:timers/promises";
import { fileURLToPath } from "node:url";
import { randomUUID } from "node:crypto";

export function sanitize(value) {
  return value.replace(/[^A-Za-z0-9._-]/g, "_");
}

export function instanceKey(env) {
  if (env.TTT_REMOTE_NAME) {
    return `name_${sanitize(env.TTT_REMOTE_NAME)}`;
  }
  if (env.HERDR_SESSION && env.HERDR_PANE_ID) {
    return `herdr_${sanitize(env.HERDR_SESSION)}_${sanitize(env.HERDR_PANE_ID)}`;
  }
  return null;
}

function parseNullSeparated(data) {
  return data.toString().split("\0").filter(Boolean);
}

function parseEnvironment(data) {
  return Object.fromEntries(
    parseNullSeparated(data).map((entry) => {
      const separator = entry.indexOf("=");
      return separator === -1
        ? [entry, ""]
        : [entry.slice(0, separator), entry.slice(separator + 1)];
    }),
  );
}

export function discoverInstances(procRoot = "/proc") {
  const instances = [];
  let entries = [];
  try {
    entries = readdirSync(procRoot, { withFileTypes: true });
  } catch {
    return instances;
  }

  for (const entry of entries) {
    if (!entry.isDirectory() || !/^\d+$/.test(entry.name)) continue;
    try {
      const command = parseNullSeparated(readFileSync(join(procRoot, entry.name, "cmdline")));
      if (command.length === 0 || basename(command[0]) !== "ttt") continue;
      const env = parseEnvironment(readFileSync(join(procRoot, entry.name, "environ")));
      const key = instanceKey(env);
      if (!key) continue;
      const selector = env.TTT_REMOTE_NAME
        ? `name:${env.TTT_REMOTE_NAME}`
        : `${env.HERDR_SESSION}:${env.HERDR_PANE_ID}`;
      instances.push({ pid: Number(entry.name), key, selector, cwd: env.PWD || "" });
    } catch {
      // Processes can exit or become inaccessible while /proc is being scanned.
    }
  }

  return instances.sort((a, b) => a.selector.localeCompare(b.selector));
}

function aliasesPath(pluginDir) {
  return join(pluginDir, "aliases.json");
}

function loadAliases(pluginDir) {
  try {
    return JSON.parse(readFileSync(aliasesPath(pluginDir), "utf8"));
  } catch {
    return {};
  }
}

export function saveAlias(pluginDir, alias, targetKey) {
  mkdirSync(pluginDir, { recursive: true });
  const aliases = { ...loadAliases(pluginDir), [alias]: targetKey };
  const path = aliasesPath(pluginDir);
  const temporaryPath = `${path}.tmp`;
  writeFileSync(temporaryPath, `${JSON.stringify(aliases, null, 2)}\n`, { mode: 0o600 });
  renameSync(temporaryPath, path);
}

export function resolveTarget(target, pluginDir, instances = discoverInstances()) {
  const aliases = loadAliases(pluginDir);
  const resolved = aliases[target] || target;
  return (
    instances.find(
      (instance) =>
        instance.key === resolved ||
        instance.selector === resolved ||
        String(instance.pid) === resolved,
    ) || null
  );
}

export function writeCommand({ pluginDir, targetKey, files }) {
  if (!targetKey) throw new Error("A target instance is required");
  if (!Array.isArray(files) || files.length === 0) throw new Error("At least one file is required");
  for (const file of files) {
    if (!file?.path || !isAbsolute(file.path)) {
      throw new Error(`File paths must be absolute: ${file?.path ?? ""}`);
    }
  }

  const mailboxes = join(pluginDir, "mailboxes");
  mkdirSync(mailboxes, { recursive: true });
  const token = `${Date.now()}-${process.pid}-${randomUUID()}`;
  const finalPath = join(mailboxes, `${targetKey}--${token}.json`);
  const temporaryPath = `${finalPath}.tmp`;
  writeFileSync(temporaryPath, `${JSON.stringify({ files })}\n`, { mode: 0o600 });
  renameSync(temporaryPath, finalPath);
  return finalPath;
}

export async function waitForAcknowledgement(commandPath, timeoutMs = 3000) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    if (existsSync(commandPath) && readFileSync(commandPath, "utf8") === "") {
      unlinkSync(commandPath);
      return;
    }
    await delay(50);
  }
  throw new Error("Target TTT instance did not acknowledge the command");
}

function defaultPluginDir() {
  if (process.env.TTT_REMOTE_OPEN_DIR) return process.env.TTT_REMOTE_OPEN_DIR;
  const configRoot = process.env.XDG_CONFIG_HOME || join(homedir(), ".config");
  return join(configRoot, "ttt", "plugins", "remote-open");
}

function printUsage() {
  console.log(`Usage:
  ttt-open list
  ttt-open set-main <target>
  ttt-open alias <name> <target>
  ttt-open <target-or-alias> <file> [file ...]

Targets shown by 'ttt-open list' may be a PID, name:<name>, or session:pane.`);
}

async function main(args = process.argv.slice(2)) {
  const pluginDir = defaultPluginDir();
  const instances = discoverInstances();
  const [command, ...rest] = args;

  if (!command || command === "--help" || command === "-h") {
    printUsage();
    return;
  }

  if (command === "list") {
    if (instances.length === 0) {
      console.log("No targetable TTT instances found.");
      return;
    }
    for (const instance of instances) {
      console.log(`${instance.selector}\tpid=${instance.pid}\t${instance.cwd}`);
    }
    return;
  }

  if (command === "set-main" || command === "alias") {
    const alias = command === "set-main" ? "main" : rest.shift();
    const selector = rest.shift();
    if (!alias || !selector) throw new Error(`${command} requires a target`);
    const instance = resolveTarget(selector, pluginDir, instances);
    if (!instance) throw new Error(`No live TTT instance matches '${selector}'`);
    saveAlias(pluginDir, alias, instance.key);
    console.log(`${alias} -> ${instance.selector} (${instance.cwd})`);
    return;
  }

  if (rest.length === 0) throw new Error("At least one file is required");
  const instance = resolveTarget(command, pluginDir, instances);
  if (!instance) throw new Error(`No live TTT instance matches '${command}'`);
  const files = rest.map((path) => ({ path: realpathSync(resolve(path)) }));
  const commandPath = writeCommand({ pluginDir, targetKey: instance.key, files });
  await waitForAcknowledgement(commandPath);
  console.log(`Opened ${files.length} file(s) in ${instance.selector}`);
}

function isMainModule() {
  if (!process.argv[1]) return false;
  try {
    return realpathSync(process.argv[1]) === realpathSync(fileURLToPath(import.meta.url));
  } catch {
    return false;
  }
}

if (isMainModule()) {
  main().catch((error) => {
    console.error(`ttt-open: ${error.message}`);
    process.exitCode = 1;
  });
}
