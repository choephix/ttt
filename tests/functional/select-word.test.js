import { describe, it, expect, afterEach } from "vitest";
import * as tui from "./tui.js";
import { createTempDir, createTempFile, cleanupDir, readFile } from "./helpers.js";

let dir;

afterEach(() => {
  tui.kill();
  if (dir) cleanupDir(dir);
});

describe("word-level cursor movement and selection", () => {
  it("should move word right", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "moveright.txt", "hello world foo");

    tui.start(file);
    tui.waitFor("hello");

    tui.press("home");
    tui.waitStable();

    tui.press("ctrl+right");
    tui.waitStable();

    tui.type("X");
    tui.waitStable();

    tui.press("ctrl+s");
    tui.waitStable();

    tui.run();

    const content = readFile(file);
    expect(content).toBe("helloX world foo\n");
  });

  it("should move word left", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "moveleft.txt", "hello world");

    tui.start(file);
    tui.waitFor("hello");

    tui.press("end");
    tui.waitStable();

    tui.press("ctrl+left");
    tui.waitStable();

    tui.type("X");
    tui.waitStable();

    tui.press("ctrl+s");
    tui.waitStable();

    tui.run();

    const content = readFile(file);
    expect(content).toBe("hello Xworld\n");
  });

  it("should select word right and delete", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "selright.txt", "hello world foo");

    tui.start(file);
    tui.waitFor("hello");

    tui.press("home");
    tui.waitStable();

    tui.press("ctrl+right");
    tui.waitStable();

    tui.exec("Select Word Right");
    tui.exec("Select Word Right");
    tui.waitStable();

    tui.press("backspace");
    tui.waitStable();

    tui.press("ctrl+s");
    tui.waitStable();

    tui.run();

    const content = readFile(file);
    expect(content).toBe("hello foo\n");
  });

  it("should select word right and replace", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "selreplace.txt", "hello world");

    tui.start(file);
    tui.waitFor("hello");

    tui.press("home");
    tui.waitStable();

    tui.exec("Select Word Right");
    tui.waitStable();

    tui.type("HI");
    tui.waitStable();

    tui.press("ctrl+s");
    tui.waitStable();

    tui.run();

    const content = readFile(file);
    expect(content).toBe("HI world\n");
  });

  it("should move word right across multiple words", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "multiword.txt", "one two three");

    tui.start(file);
    tui.waitFor("one");

    tui.press("home");
    tui.waitStable();

    tui.press("ctrl+right");
    tui.press("ctrl+right");
    tui.press("ctrl+right");
    tui.waitStable();

    tui.type("X");
    tui.waitStable();

    tui.press("ctrl+s");
    tui.waitStable();

    tui.run();

    const content = readFile(file);
    expect(content).toBe("one twoX three\n");
  });
});
