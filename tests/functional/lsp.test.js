import { describe, it, expect, afterEach, beforeEach } from "vitest";
import {
  cpSync,
  readFileSync,
  readdirSync,
  unlinkSync,
  existsSync,
} from "node:fs";
import { execSync } from "node:child_process";
import { resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import * as tui from "./tui.js";
import { createTempDir, cleanupDir } from "./helpers.js";

const __dirname = dirname(fileURLToPath(import.meta.url));
const LSP_DIR = resolve(__dirname, "lsp");
const LOG_FILE = resolve(
  process.env.HOME || process.env.USERPROFILE,
  ".config",
  "ttt",
  "ttt.log"
);

function sleep(ms) {
  execSync(`sleep ${ms / 1000}`);
}

function waitForLog(pattern, timeoutMs = 10000) {
  const re = new RegExp(pattern);
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    if (existsSync(LOG_FILE)) {
      const log = readFileSync(LOG_FILE, "utf8");
      if (re.test(log)) return log;
    }
    sleep(200);
  }
  return existsSync(LOG_FILE) ? readFileSync(LOG_FILE, "utf8") : "";
}

function navigateTo(pos) {
  const [line, col] = pos.split(":").map(Number);
  tui.press("ctrl+g");
  tui.waitStable();
  tui.type(String(line));
  tui.press("enter");
  tui.waitStable();
  tui.press("home");
  for (let i = 0; i < col; i++) {
    tui.press("arrow_right");
  }
}

function lspServerAvailable(fixtureDir) {
  const settings = JSON.parse(
    readFileSync(resolve(fixtureDir, "settings.json"), "utf8")
  );
  for (const server of Object.values(settings.lsp.servers)) {
    try {
      execSync(`which ${server.command[0]}`, { stdio: "ignore" });
    } catch {
      return false;
    }
  }
  return true;
}

const SKIP_LANGUAGES = ["svelte"];

const languages = readdirSync(LSP_DIR, { withFileTypes: true })
  .filter((d) => d.isDirectory())
  .filter((d) => !SKIP_LANGUAGES.includes(d.name))
  .filter((d) => existsSync(resolve(LSP_DIR, d.name, "spec.json")))
  .map((d) => d.name);

describe("lsp", () => {
  let dir;

  beforeEach(() => {
    if (existsSync(LOG_FILE)) unlinkSync(LOG_FILE);
  });

  afterEach(() => {
    tui.kill();
    if (dir) cleanupDir(dir);
    dir = null;
  });

  for (const lang of languages) {
    const fixtureDir = resolve(LSP_DIR, lang);
    const spec = JSON.parse(
      readFileSync(resolve(fixtureDir, "spec.json"), "utf8")
    );
    const available = lspServerAvailable(fixtureDir);

    describe(lang, () => {
      const testFn = available ? it : it.skip;

      testFn("diagnostics", () => {
        dir = createTempDir();
        cpSync(fixtureDir, dir, { recursive: true });

        const files = readdirSync(dir).filter(
          (f) => !["spec.json", "settings.json", "install.sh"].includes(f)
        );
        const testFile = resolve(
          dir,
          files.find((f) => f.startsWith("test."))
        );
        const configFile = resolve(fixtureDir, "settings.json");

        tui.start("--config", configFile, testFile);
        tui.waitFor(spec.waitFor);
        waitForLog("lsp initialized");

        const log = waitForLog(spec.diagnostic);
        tui.press("ctrl+q");

        expect(log).toContain("lsp starting server");
        expect(log).toMatch(new RegExp(spec.diagnostic));
      });

      testFn("hover", () => {
        if (!spec.hover) return;
        dir = createTempDir();
        cpSync(fixtureDir, dir, { recursive: true });

        const files = readdirSync(dir).filter(
          (f) => !["spec.json", "settings.json", "install.sh"].includes(f)
        );
        const testFile = resolve(
          dir,
          files.find((f) => f.startsWith("test."))
        );
        const configFile = resolve(fixtureDir, "settings.json");

        tui.start("--config", configFile, testFile);
        tui.waitFor(spec.waitFor);
        waitForLog("lsp initialized");

        navigateTo(spec.hover.goto);
        tui.waitStable();
        tui.pressChord("ctrl+k", "i");

        const log = waitForLog(spec.hover.log);
        tui.press("ctrl+q");

        expect(log).toMatch(new RegExp(spec.hover.log));
      });

      testFn("completion", () => {
        if (!spec.completion) return;
        dir = createTempDir();
        cpSync(fixtureDir, dir, { recursive: true });

        const files = readdirSync(dir).filter(
          (f) => !["spec.json", "settings.json", "install.sh"].includes(f)
        );
        const testFile = resolve(
          dir,
          files.find((f) => f.startsWith("test."))
        );
        const configFile = resolve(fixtureDir, "settings.json");

        tui.start("--config", configFile, testFile);
        tui.waitFor(spec.waitFor);
        waitForLog("lsp initialized");

        navigateTo(spec.completion.goto);
        tui.waitStable();
        tui.type(spec.completion.type);

        const log = waitForLog(spec.completion.log);
        tui.press("escape");
        tui.press("ctrl+q");
        sleep(200);
        tui.press("arrow_right");
        tui.press("enter");

        expect(log).toMatch(new RegExp(spec.completion.log));
      });

      testFn("signature help", () => {
        if (!spec.signatureHelp) return;
        dir = createTempDir();
        cpSync(fixtureDir, dir, { recursive: true });

        const files = readdirSync(dir).filter(
          (f) => !["spec.json", "settings.json", "install.sh"].includes(f)
        );
        const testFile = resolve(
          dir,
          files.find((f) => f.startsWith("test."))
        );
        const configFile = resolve(fixtureDir, "settings.json");

        tui.start("--config", configFile, testFile);
        tui.waitFor(spec.waitFor);
        waitForLog("lsp initialized");

        navigateTo(spec.signatureHelp.goto);
        tui.waitStable();
        tui.type(spec.signatureHelp.type);

        const log = waitForLog(spec.signatureHelp.log);
        tui.press("escape");
        tui.press("ctrl+q");
        sleep(200);
        tui.press("arrow_right");
        tui.press("enter");

        expect(log).toMatch(new RegExp(spec.signatureHelp.log));
      });
    });
  }
});
