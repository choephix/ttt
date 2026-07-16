# TTT Remote Open

Open files in one specific running TTT process from another shell or agent.

The Lua plugin polls a per-instance mailbox inside its installation directory. `ttt-open` discovers targetable TTT processes through `/proc`, atomically queues a command, waits for the plugin to acknowledge it, and then removes the command file.

## Install

```bash
node plugins/remote-open/install.mjs
```

The installer creates a real global plugin directory at `~/.config/ttt/plugins/remote-open/`, symlinks the versioned manifest and Lua source into it, and installs `~/.local/bin/ttt-open`. The directory itself intentionally is not a symlink because TTT's filesystem sandbox resolves child symlinks when validating allowed roots.

Restart TTT, approve the plugin's `fs.read`, `fs.write`, and `system.env` permissions, then list instances:

```bash
ttt-open list
```

Choose the persistent default once:

```bash
ttt-open set-main hustles:w3:p2
```

Open one or more existing files as pinned tabs:

```bash
ttt-open main path/to/one.ts path/to/two.ts
```

Other aliases are supported:

```bash
ttt-open alias vault hustles:w3:p2
ttt-open vault README.md
```

A TTT process outside Herdr can opt in with a name:

```bash
TTT_REMOTE_NAME=scratch ttt .
ttt-open name:scratch README.md
```

## Development

```bash
make build
node --test plugins/remote-open/test/remote-open.test.mjs
```

The tests cover instance discovery, atomic queueing, aliases, installation, and a headless TTT run that consumes a mailbox and opens multiple tabs.

## Constraints

- Linux only: discovery uses `/proc`.
- Requested paths must already exist; the CLI resolves them to absolute paths before queueing.
- The target instance must have loaded and approved the plugin.
- TTT currently has one editor group, so this opens tabs but cannot arrange arbitrary files side by side.
