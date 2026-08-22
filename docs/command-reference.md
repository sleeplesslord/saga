# Command Reference

Full reference for all `saga` commands.

## Creating Sagas

### `saga new`

Create a new saga.

```bash
saga new <title> [flags]
```

| Flag | Description |
|------|-------------|
| `--parent <id>` | Create as sub-saga under parent |
| `--label <label>` | Add label (repeatable) |
| `--priority <level>` | Set priority: `high`, `normal`, `low` |
| `--desc <text>` | Add description |
| `--deadline <YYYYMMDD>` | Set deadline |

Sagas are saved to the local `.saga/` if it exists, otherwise to the global `~/.saga/`.

```bash
saga new "Implement auth"
saga new "Add OAuth" --parent abc123
saga new "Fix bug" --label bug --label urgent --priority high
saga new "Refactor" --desc "Clean up the auth module" --deadline 20260415
```

## Viewing Sagas

### `saga list`

List sagas with optional filters.

```bash
saga list [flags]
```

| Flag | Description |
|------|-------------|
| `-a`, `--all` | Show all sagas including done/wontdo |
| `-l`, `--local` | Project-local sagas only |
| `-g`, `--global` | Include global sagas |
| `--status <status>` | Filter: `active`, `paused`, `done`, `wontdo` |
| `--priority <level>` | Filter: `high`, `normal`, `low` |
| `--label <label>` | Filter by label |
| `--mine` | Show only your claimed sagas |
| `--unclaimed` | Show only unclaimed sagas |

When a local `.saga/` exists, `saga list` shows local sagas by default. Use `--global` to include global. Without a local store, shows global by default.

### `saga status`

Show brief details and history for a saga.

```bash
saga status <id>
```

### `saga context`

Show full context: hierarchy, dependencies, claims, linked runes, and history.

```bash
saga context <id> [--format json]
```

`--format json` outputs machine-readable JSON for agent consumption.

A saga carries a `rev` field that increments on every stored change. It exists so
a write built on a stale read can be rejected instead of silently overwriting a
concurrent one; treat it as opaque and don't set it by hand.

### `saga search`

Search sagas by title, ID, or description.

```bash
saga search <query> [flags]
```

| Flag | Description |
|------|-------------|
| `--label <label>` | Filter by label (repeatable) |
| `--status <status>` | Filter: `active`, `paused`, `done`, `wontdo` |
| `--priority <level>` | Filter: `high`, `normal`, `low` |

### `saga ready`

List sagas ready for work (unclaimed, unblocked, no active children).

```bash
saga ready [--take]
```

`--take` claims the top ready saga automatically.

### `saga web`

Open a local, read-only dashboard for visualizing status, hierarchy, claims, and hard dependencies.

```bash
saga web [--address 127.0.0.1:7331] [--no-open] [--global]
```

| Flag | Description |
|------|-------------|
| `--address <host:port>` | Localhost listen address (default `127.0.0.1:7331`) |
| `--no-open` | Start the server without opening a browser |
| `--global` | Show the global store instead of the current project store |

The server only accepts localhost bindings. Data refreshes automatically; task changes remain CLI-only.

## Modifying Sagas

### `saga edit`

Edit title, description, deadline, or priority after creation.

```bash
saga edit <id> [flags]
```

| Flag | Description |
|------|-------------|
| `--title <text>` | New title |
| `--desc <text>` | New description |
| `--deadline <YYYYMMDD>` | Set or edit deadline |
| `--deadline ""` | Clear deadline |
| `--priority <level>` | Set priority: `high`, `normal`, `low` |

At least one flag is required.

### `saga label`

Add or remove labels.

```bash
saga label <id> add <label>
saga label <id> remove <label>
```

### `saga priority`

Set priority directly.

```bash
saga priority <id> <high|normal|low>
```

### `saga log`

Add a work log entry to a saga's history.

```bash
saga log <id> <message>
saga log <id> --file notes.md
saga log <id> -            # read the message from stdin
cmd | saga log <id>        # same, when no message is given
```

The message is resolved in this order: `--file`, then `-`, then the message
argument, then stdin. An argument therefore wins over piped input — pass `-`
when you mean stdin. `saga plan` resolves its plan text the same way.

## Completing Sagas

### `saga done`

Mark saga(s) as complete.

```bash
saga done <id> [id...] [flags]
```

| Flag | Description |
|------|-------------|
| `--reason <text>` | Reason logged in history |
| `--cascade` | Also mark all active sub-sagas as done |
| `--force` | Complete despite active children or incomplete dependencies |
| `--quiet` | Suppress hints and non-essential output |

Multiple IDs can be provided: `saga done abc123 def456`

By default, cannot complete a saga that has active sub-sagas or incomplete dependencies. Use `--cascade` to complete children first, or `--force` to override.

### `saga wontdo`

Mark saga(s) as won't-do — abandoned, rejected, or obsoleted.

```bash
saga wontdo <id> [id...] [flags]
```

| Flag | Description |
|------|-------------|
| `--reason <text>` | Reason logged in history (recommended) |
| `--cascade` | Also mark all active sub-sagas as wontdo |
| `--quiet` | Suppress hints and non-essential output |

Wontdo is a terminal state like `done`, but semantically distinct. It is non-blocking in dependency checks — other sagas that depend on a wontdo saga can still be completed.

### `saga reopen`

Reopen a saga that was previously marked as done.

```bash
saga reopen <id> [--reason <text>]
```

| Flag | Description |
|------|-------------|
| `--reason <text>` | Reason logged in history |

Only `done` sagas can be reopened. Sets status back to `active`.

## Status Transitions

```
saga new ──→ active
             │
    saga pause ─┤ (set via edit or external tool)
             │
   saga continue ←┘
             │
     saga done ──┤──→ done ──saga reopen──→ active
             │
  saga wontdo ──┘──→ wontdo (terminal)
```

## Dependencies and Relationships

### `saga depend`

Manage hard (blocking) dependencies.

```bash
saga depend <id> add <target-id>     # id blocked until target is done
saga depend <id> remove <target-id>  # remove blocking dependency
```

Incomplete dependencies block completion (`saga done` fails). Wontdo dependencies are non-blocking (shown as ⊘).

### `saga relate`

Manage soft (informational) relationships.

```bash
saga relate <id> add <target-id>
saga relate <id> remove <target-id>
```

Relationships don't block anything — they're for cross-referencing related work.

## Claiming

### `saga claim`

Claim saga(s) for your session. Prevents duplicate work.

```bash
saga claim <id> [id...] [--duration <dur>]
```

| Flag | Description |
|------|-------------|
| `--agent <name>` | Agent name (default: `$USER`) |
| `--duration <dur>` | Claim duration (e.g. `4h`, `30m`, `72h`) |

Identity is `user@ppid` — same process = same session = "mine". Different process = "claimed by other".

### `saga unclaim`

Release claim(s).

```bash
saga unclaim <id> [id...]
```

Claims expire after the configured duration (default 24h). See `saga config` to change.

## Configuration

### `saga config`

View or set configuration.

```bash
saga config                              # Show current config
saga config --claim-duration 4h          # Set local default
saga config --scope global --claim-duration 4h  # Set global default
```

| Flag | Description |
|------|-------------|
| `--claim-duration <dur>` | Default claim duration |
| `--scope <scope>` | `local` (default) or `global` |

Config resolution for claim duration: `--duration` flag > local config (`.saga/config.json`) > global config (`~/.saga/config.json`) > 24h default.

## Storage

### `saga init`

Initialize local `.saga/` storage in the current directory.

```bash
saga init
```

Creates `.saga/sagas.jsonl` and `.saga/config.json`. Without `saga init`, all sagas are stored globally in `~/.saga/`.

### Storage Scopes

| Scope | Location | When Used |
|-------|----------|-----------|
| Local | `./.saga/sagas.jsonl` | When `.saga/` exists (auto-detected) |
| Global | `~/.saga/sagas.jsonl` | Always (fallback if no local) |

Sagas are saved to local by default when a local store exists. `saga list` shows local by default; use `--global` to include global sagas.
