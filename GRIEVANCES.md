# Grievances

## 1. Middle-click tab close — resolved

Middle-clicking any tab closes it. Clean background tabs close without being activated; dirty tabs activate first so the existing save confirmation can protect their contents.

Implemented behavior:

- Middle-click on a tab closes that tab.
- Clean tabs are not activated first.
- Dirty tabs use the existing Discard/Cancel/Save confirmation.

## 2. Preview-open tab mode — resolved

The explorer opens files in a reusable preview tab on single-click and commits them by editing or pressing Enter.

Implemented behavior:

- Single-clicking a file opens it in a reusable preview tab.
- Preview labels are italic.
- Single-clicking another file reuses the clean preview tab.
- Pressing Enter commits the file as a normal tab.
- Double-clicking starts an inline filesystem rename.
- Editing a preview tab converts it into a normal tab.

## Summary

Both requested file and tab management interactions are implemented.
