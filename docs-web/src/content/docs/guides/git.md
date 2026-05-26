---
title: Git Integration
description: Source control features built into TTT.
---

The changes panel in the sidebar (**Ctrl+K C**) provides a full staging workflow.

## Staging

- **Spacebar** toggles stage/unstage on the selected file
- **`a`** stages all unstaged files
- **`u`** unstages all staged files
- **`+` button** on the "Changes" section header stages all files in that section
- **`-` button** on the "Staged" section header unstages all files in that section

## Committing

- Inline commit message input at the top of each group
- Type a message and press Enter to commit all staged files

## Remote Operations

- **Pull**, **Push**, **Sync** (pull then push) from the sidebar actions button
- Per-repo actions via the group header menu button in multi-root workspaces

## Diff View

- Select a changed file to open a diff with syntax highlighting layered on diff backgrounds
- Untracked files open directly in the editor

## Multi-Root Support

When working with multiple folders:

- Changes are grouped by repository, each with its own collapsible Staged/Changes sections
- Each group has a commit input and a menu button for pull/push/sync on that specific repo
- File status badges: **M** (modified), **A** (added), **D** (deleted), **R** (renamed), **U** (untracked)

## Git Blame

The status bar shows inline blame info for the current line: author, relative time, and commit summary.
