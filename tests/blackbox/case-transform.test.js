import { describe, it, expect, afterEach } from "vitest";
import * as tui from "./tui.js";
import { createTempDir, createTempFile, cleanupDir } from "./helpers.js";

let dir;

afterEach(() => {
  tui.kill();
  if (dir) cleanupDir(dir);
});

describe("case transforms", () => {
  it("should transform to upper case", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "test.txt", "hello world\n");

    tui.start(file);
    tui.waitFor("hello");

    tui.press("ctrl+a");
    tui.waitStable();

    tui.exec("Transform to Uppercase");
    tui.waitStable();

    const s0 = tui.snapshot();
    const { snapshots } = tui.run();
    expect(snapshots[s0]).toContain("HELLO WORLD");
  });

  it("should transform to lower case", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "test.txt", "HELLO WORLD\n");

    tui.start(file);
    tui.waitFor("HELLO");

    tui.press("ctrl+a");
    tui.waitStable();

    tui.exec("Transform to Lowercase");
    tui.waitStable();

    const s0 = tui.snapshot();
    const { snapshots } = tui.run();
    expect(snapshots[s0]).toContain("hello world");
  });

  it("should transform to title case", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "test.txt", "hello world\n");

    tui.start(file);
    tui.waitFor("hello");

    tui.press("ctrl+a");
    tui.waitStable();

    tui.exec("Transform to Titlecase");
    tui.waitStable();

    const s0 = tui.snapshot();
    const { snapshots } = tui.run();
    expect(snapshots[s0]).toContain("Hello World");
  });

  it("should undo upper case transform", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "test.txt", "hello world\n");

    tui.start(file);
    tui.waitFor("hello");

    tui.press("ctrl+a");
    tui.waitStable();

    tui.exec("Transform to Uppercase");
    tui.waitStable();

    const s0 = tui.snapshot();

    tui.press("ctrl+z");
    tui.waitStable();

    const s1 = tui.snapshot();
    const { snapshots } = tui.run();
    expect(snapshots[s0]).toContain("HELLO WORLD");
    expect(snapshots[s1]).toContain("hello world");
  });
});
