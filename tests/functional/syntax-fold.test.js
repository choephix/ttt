import { describe, it, expect, afterEach } from "vitest";
import * as tui from "./tui.js";
import { createTempDir, createTempFile, cleanupDir } from "./helpers.js";

let dir;

afterEach(() => {
  tui.kill();
  if (dir) cleanupDir(dir);
  dir = null;
});

const goContent = `package main

import "fmt"

func main() {
\tx := "hello"
\tfmt.Println(x)
}
`;

describe("syntax highlighting consistency after fold/unfold", () => {
  it("should preserve content after fold and unfold cycle", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "main.go", goContent);

    tui.start(file);
    tui.waitFor("hello");

    // Verify initial content is visible
    let snap = tui.snapshot();
    expect(snap).toContain("hello");
    expect(snap).toContain("Println");

    // Go to the func main() line and fold it
    tui.press("ctrl+g");
    tui.waitStable();
    tui.type("5");
    tui.press("enter");
    tui.waitStable();

    tui.pressChord("ctrl+k", "[");
    tui.waitStable();

    // Content inside the fold should be hidden
    snap = tui.snapshot();
    expect(snap).not.toContain("hello");

    // Unfold using toggle
    tui.pressChord("ctrl+k", "[");
    tui.waitStable();

    // Content should be restored exactly
    snap = tui.snapshot();
    expect(snap).toContain("hello");
    expect(snap).toContain("Println");
    expect(snap).toContain("package main");
    expect(snap).toContain('import "fmt"');
  });

  it("should survive multiple fold/unfold cycles without content corruption", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "main.go", goContent);

    tui.start(file);
    tui.waitFor("hello");

    // Perform 3 fold-all / unfold-all cycles
    for (let i = 0; i < 3; i++) {
      tui.pressChord("ctrl+k", "0");
      tui.waitStable();

      let snap = tui.snapshot();
      expect(snap).not.toContain("hello");

      tui.pressChord("ctrl+k", "9");
      tui.waitStable();

      snap = tui.snapshot();
      expect(snap).toContain("hello");
    }

    // Final verification that all content is intact
    const snap = tui.snapshot();
    expect(snap).toContain("package main");
    expect(snap).toContain('import "fmt"');
    expect(snap).toContain("func main()");
    expect(snap).toContain("hello");
    expect(snap).toContain("Println");
  });

  it("should preserve edits made after unfolding", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "main.go", goContent);

    tui.start(file);
    tui.waitFor("hello");

    // Go to the func main() line and fold
    tui.press("ctrl+g");
    tui.waitStable();
    tui.type("5");
    tui.press("enter");
    tui.waitStable();

    tui.pressChord("ctrl+k", "[");
    tui.waitStable();

    let snap = tui.snapshot();
    expect(snap).not.toContain("hello");

    // Unfold to edit inside
    tui.pressChord("ctrl+k", "[");
    tui.waitStable();

    snap = tui.snapshot();
    expect(snap).toContain("hello");

    // Navigate to the "hello" line and add text
    tui.press("ctrl+g");
    tui.waitStable();
    tui.type("6");
    tui.press("enter");
    tui.waitStable();

    // Go to end of line and add a comment
    tui.press("end");
    tui.waitStable();
    tui.type(" // edited");
    tui.waitStable();

    snap = tui.snapshot();
    expect(snap).toContain("edited");

    // Fold again
    tui.press("ctrl+g");
    tui.waitStable();
    tui.type("5");
    tui.press("enter");
    tui.waitStable();

    tui.pressChord("ctrl+k", "[");
    tui.waitStable();

    snap = tui.snapshot();
    expect(snap).not.toContain("hello");
    expect(snap).not.toContain("edited");

    // Unfold and verify the edit persisted
    tui.pressChord("ctrl+k", "[");
    tui.waitStable();

    snap = tui.snapshot();
    expect(snap).toContain("hello");
    expect(snap).toContain("edited");
  });

  it("should show fold indicator when folded and hide it when unfolded", () => {
    dir = createTempDir();
    const file = createTempFile(dir, "main.go", goContent);

    tui.start(file);
    tui.waitFor("hello");

    // Initially no fold indicator
    let snap = tui.snapshot();
    expect(snap).not.toContain("⋯");

    // Go to func main() line and fold
    tui.press("ctrl+g");
    tui.waitStable();
    tui.type("5");
    tui.press("enter");
    tui.waitStable();

    tui.pressChord("ctrl+k", "[");
    tui.waitStable();

    // Fold indicator should appear
    snap = tui.snapshot();
    expect(snap).toContain("⋯");

    // Unfold
    tui.pressChord("ctrl+k", "[");
    tui.waitStable();

    // Fold indicator should disappear
    snap = tui.snapshot();
    expect(snap).not.toContain("⋯");
  });
});
