---
title: Git Integration
description: Source control features built into TTT.
---

The changes panel in the sidebar (**Ctrl+K C**) provides a full staging workflow.

## Staging & Unstaging

- **Spacebar** toggles stage/unstage on the selected file
- **`a`** stages all unstaged files
- **`u`** unstages all staged files
- **`+` button** on each unstaged file stages that file
- **`−` button** on each staged file unstages that file
- **`+` button** on the "Changes" section header stages all files
- **`−` button** on the "Staged" section header unstages all files

## Discarding Changes

- **`d`** discards changes to the selected unstaged file (with confirmation)
- **`D`** discards all unstaged changes in the current group (with confirmation)
- **`✕` button** on the "Changes" section header discards all unstaged changes
- Untracked files are deleted; modified files are restored to HEAD

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

## Pull Request Review

Open a GitHub pull request directly from the command line:

```sh
ttt https://github.com/owner/repo/pull/123

# Review a PR with the repo tree open
ttt . https://github.com/owner/repo/pull/123
```

Changed files appear in the changes panel with diff view.

## Git Blame

The status bar shows inline blame info for the current line: author, relative time, and commit summary.
