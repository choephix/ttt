import { describe, it, expect, afterEach } from "vitest";
import * as tui from "./tui.js";

afterEach(() => {
  tui.kill();
});

describe("tab close button hit test (#354)", () => {
  it("closes the active tab when its × is clicked in the MoreButton overflow window", () => {
    tui.start();
    // At 83 cols the tab strip overflows the inner zone (the ⋮ MoreButton
    // reserves 4 cols) but NOT the full width — the window where Render draws
    // no overflow-arrow gutter. The click handler used to assume a 3-col gutter
    // anyway, shifting the close-X hit test by 3 cells so the × was a dead
    // click. This puts the active tab's × at row 2, col 75.
    tui.setSize(83, 24);

    tui.press("ctrl+n");
    tui.press("ctrl+n");
    tui.press("ctrl+n");
    tui.waitStable();
    const before = tui.snapshot();

    tui.click(75, 2);
    tui.waitStable();
    const after = tui.snapshot();

    const { snapshots } = tui.run();

    // Sanity: the × really is under the click (guards against geometry drift
    // silently turning this into a no-op that always passes).
    expect(snapshots[before].split("\n")[2][75]).toBe("x");
    expect(snapshots[before]).toContain("untitled-4");

    // The click must close the active tab.
    expect(snapshots[after]).not.toContain("untitled-4");
  });
});
