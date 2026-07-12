# Grievances

## 1. No middle-click tab close

Middle-clicking a tab should close that tab.

Current grievance: tabs cannot be closed with middle-click, which removes a common, fast tab-management interaction used in many editors and browsers.

Expected behavior:

- Middle-click on a tab closes that tab.
- The click should not activate the tab first unless activation is required internally.
- This should work consistently for pinned and unpinned tabs, with pinned-tab behavior following the app's existing close rules.

## 2. No preview-open tab mode

There is no preview-style file opening workflow where a single click opens a file temporarily in the same tab, and a double-click commits it as a normal tab.

Current grievance: every file open behaves as a committed open, or the app lacks the familiar preview-tab distinction. This makes browsing files heavier than necessary and causes tab clutter.

Expected behavior:

- Single-clicking a file opens it in a reusable preview tab.
- The preview tab is visually distinct, for example with an italic tab label.
- Single-clicking another file reuses the same preview tab instead of creating a new committed tab.
- Double-clicking a file opens it as a committed tab.
- Editing a preview tab, pinning it, or otherwise explicitly committing it should convert it into a normal tab.

## Summary

The missing interactions make file and tab management slower than expected:

- Middle-click should close tabs.
- Single-click should preview files in a reusable italicized preview tab.
- Double-click should commit files into normal tabs.
