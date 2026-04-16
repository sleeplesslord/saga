# Command Reference

Full reference for all `sg` commands.

## Creating Sagas

### `sg new`

Create a new saga.

```bash
sg new <title> [flags]
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
sg new "Implement auth"
sg new "Add OAuth" --parent abc123
sg new "Fix bug" --label bug --label urgent --priority high
sg new "Refactor" --desc "Clean up the auth module" --deadline 20260415
```

## Viewing Sagas

### `sg list`

List sagas with optional filters.

```bash
sg list [flags]
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

When a local `.saga/` exists, `sg list` shows local sagas by default. Use `--global` to include global. Without a local store, shows global by default.

### `sg status`

Show brief details and history for a saga.

```bash
sg status <id>
```

### `sg context`

Show full context: hierarchy, dependencies, claims, linked runes, and history.

```bash
sg context <id> [--format json]
```

`--format json` outputs machine-readable JSON for agent consumption.

### `sg search`

Search sagas by title, ID, or description.

```bash
sg search <query> [flags]
```

| Flag | Description |
|------|-------------|
| `--label <label>` | Filter by label (repeatable) |
| `--status <status>` | Filter: `active`, `paused`, `done`, `wontdo` |
| `--priority <level>` | Filter: `high`, `normal`, `low` |

### `sg ready`

List sagas ready for work (unclaimed, unblocked, no active children).

```bash
sg ready [--take]
```

`--take` claims the top ready saga automatically.

## Modifying Sagas

### `sg edit`

Edit title, description, deadline, or priority after creation.

```bash
sg edit <id> [flags]
```

| Flag | Description |
|------|-------------|
| `--title <text>` | New title |
| `--desc <text>` | New description |
| `--deadline <YYYYMMDD>` | Set or edit deadline |
| `--deadline ""` | Clear deadline |
| `--priority <level>` | Set priority: `high`, `normal`, `low` |

At least one flag is required.

### `sg label`

Add or remove labels.

```bash
sg label <id> add <label>
sg label <id> remove <label>
```

### `sg priority`

Set priority directly.

```bash
sg priority <id> <high|normal|low>
```

### `sg log`

Add a work log entry to a saga's history.

```bash
sg log <id> <message>
sg log <id> --file notes.md
```

## Completing Sagas

### `sg done`

Mark saga(s) as complete.

```bash
sg done <id> [id...] [flags]
```

| Flag | Description |
|------|-------------|
| `--reason <text>` | Reason logged in history |
| `--cascade` | Also mark all active sub-sagas as done |
| `--force` | Complete despite active children or incomplete dependencies |
| `--quiet` | Suppress hints and non-essential output |

Multiple IDs can be provided: `sg done abc123 def456`

By default, cannot complete a saga that has active sub-sagas or incomplete dependencies. Use `--cascade` to complete children first, or `--force` to override.

### `sg wontdo`

Mark saga(s) as won't-do — abandoned, rejected, or obsoleted.

```bash
sg wontdo <id> [id...] [flags]
```

| Flag | Description |
|------|-------------|
| `--reason <text>` | Reason logged in history (recommended) |
| `--cascade` | Also mark all active sub-sagas as wontdo |
| `--quiet` | Suppress hints and non-essential output |

Wontdo is a terminal state like `done`, but semantically distinct. It is non-blocking in dependency checks — other sagas that depend on a wontdo saga can still be completed.

### `sg reopen`

Reopen a saga that was previously marked as done.

```bash
sg reopen <id> [--reason <text>]
```

| Flag | Description |
|------|-------------|
| `--reason <text>` | Reason logged in history |

Only `done` sagas can be reopened. Sets status back to `active`.

## Status Transitions

```
sg new ──→ active
             │
    sg pause ─┤ (set via edit or external tool)
             │
   sg continue ←┘
             │
     sg done ──┤──→ done ──sg reopen──→ active
             │
  sg wontdo ──┘──→ wontdo (terminal)
```

## Dependencies and Relationships

### `sg depend`

Manage hard (blocking) dependencies.

```bash
sg depend <id> add <target-id>     # id blocked until target is done
sg depend <id> remove <target-id>  # remove blocking dependency
```

Incomplete dependencies block completion (`sg done` fails). Wontdo dependencies are non-blocking (shown as ⊘).

### `sg relate`

Manage soft (informational) relationships.

```bash
sg relate <id> add <target-id>
sg relate <id> remove <target-id>
```

Relationships don't block anything — they're for cross-referencing related work.

## Claiming

### `sg claim`

Claim saga(s) for your session. Prevents duplicate work.

```bash
sg claim <id> [id...] [--duration <dur>]
```

| Flag | Description |
|------|-------------|
| `--agent <name>` | Agent name (default: `$USER`) |
| `--duration <dur>` | Claim duration (e.g. `4h`, `30m`, `72h`) |

Identity is `user@ppid` — same process = same session = "mine". Different process = "claimed by other".

### `sg unclaim`

Release claim(s).

```bash
sg unclaim <id> [id...]
```

Claims expire after the configured duration (default 24h). See `sg config` to change.

## Configuration

### `sg config`

View or set configuration.

```bash
sg config                              # Show current config
sg config --claim-duration 4h          # Set local default
sg config --scope global --claim-duration 4h  # Set global default
```

| Flag | Description |
|------|-------------|
| `--claim-duration <dur>` | Default claim duration |
| `--scope <scope>` | `local` (default) or `global` |

Config resolution for claim duration: `--duration` flag > local config (`.saga/config.json`) > global config (`~/.saga/config.json`) > 24h default.

## Storage

### `sg init`

Initialize local `.saga/` storage in the current directory.

```bash
sg init
```

Creates `.saga/sagas.jsonl` and `.saga/config.json`. Without `sg init`, all sagas are stored globally in `~/.saga/`.

### Storage Scopes

| Scope | Location | When Used |
|-------|----------|-----------|
| Local | `./.saga/sagas.jsonl` | When `.saga/` exists (auto-detected) |
| Global | `~/.saga/sagas.jsonl` | Always (fallback if no local) |

Sagas are saved to local by default when a local store exists. `sg list` shows local by default; use `--global` to include global sagas.
