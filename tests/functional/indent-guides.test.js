import { describe, it, expect, afterEach } from "vitest";
import * as tui from "./tui.js";
import { createTempDir, createTempFile, cleanupDir } from "./helpers.js";

let dir;

afterEach(() => {
  tui.kill();
  if (dir) cleanupDir(dir);
});

describe("indent guides", () => {
  it("should render indent guides in indented code", () => {
    dir = createTempDir();
    const content = [
      "func main() {",
      "    if true {",
      "        fmt.Println()",
      "    }",
      "}",
    ].join("\n");
    const file = createTempFile(dir, "guides.go", content);

    tui.start(file);
    tui.waitFor("fmt.Println");
    tui.waitStable();

    const snap = tui.snapshot();
    // The fmt.Println line has 8-space indent; at column 4 there should be a guide
    const lines = snap.split("\n");
    const fmtLine = lines.find((l) => l.includes("fmt.Println"));
    expect(fmtLine).toBeDefined();
    // The indent area before "fmt" should contain the guide character
    const idx = fmtLine.indexOf("fmt");
    const indentArea = fmtLine.substring(0, idx);
    expect(indentArea).toContain("│"); // │ character
  });

  it("should toggle indent guides off and on", () => {
    dir = createTempDir();
    const content = [
      "func main() {",
      "    if true {",
      "        fmt.Println()",
      "    }",
      "}",
    ].join("\n");
    const file = createTempFile(dir, "toggle.go", content);

    tui.start(file);
    tui.waitFor("fmt.Println");
    tui.waitStable();

    // Verify guides are on by default
    let snap = tui.snapshot();
    let lines = snap.split("\n");
    let fmtLine = lines.find((l) => l.includes("fmt.Println"));
    let idx = fmtLine.indexOf("fmt");
    let indentArea = fmtLine.substring(0, idx);
    expect(indentArea).toContain("│");

    // Toggle off
    tui.exec("Toggle Indent Guides");
    tui.waitStable();

    snap = tui.snapshot();
    lines = snap.split("\n");
    fmtLine = lines.find((l) => l.includes("fmt.Println"));
    idx = fmtLine.indexOf("fmt");
    // After line number, the indent area should be all spaces now
    // Find the content area (after gutter) by locating the line number
    const numIdx = fmtLine.indexOf("3");
    const afterNum = fmtLine.substring(numIdx + 1, idx);
    const guideCount = (afterNum.match(/│/g) || []).length;
    expect(guideCount).toBe(0);
  });
});
