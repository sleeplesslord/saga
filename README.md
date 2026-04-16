# Saga

Task management for agent workflows. Track work, capture knowledge, coordinate between humans and agents.

## Why Saga

Most task trackers are built for human teams. Saga is built for **human-agent collaboration**:

- **Sagas track tasks** — what needs to be done, by whom, in what order
- **Runes capture knowledge** — solutions, patterns, lessons learned ([separate project](https://github.com/sleeplesslord/runes))
- **Hierarchical structure** — parent/child relationships mirror real work breakdown
- **Claim system** — agents coordinate without stepping on each other
- **Context command** — agents understand the full picture before acting

## Quick Start

```bash
# Install
go install github.com/sleeplesslord/saga/cmd/sg@latest

# Initialize project storage
cd my-project
sg init

# Create a saga
sg new "Implement feature X" --desc "Details here" --priority high

# Break it down
sg new "Write tests" --parent <id>
sg new "Handle edge cases" --parent <id>

# See what's ready to work on
sg ready

# Claim and work
sg ready --take              # Claim the top ready saga
sg log <id> "Started implementation"

# Mark done
sg done <id> --reason "All tests passing"
```

## Core Concepts

### Sagas

A saga is a task or project. It has:

- **Title** — short description
- **Description** — details, requirements
- **Status** — `active`, `paused`, `done`, or `wontdo`
- **Priority** — `high`, `normal` (default), or `low`
- **Labels** — tags for filtering
- **Deadline** — optional due date (YYYYMMDD format)
- **Hierarchy** — parent/child relationships

### Statuses

| Status | Meaning | Transitions |
|--------|---------|-------------|
| `active` | Work in progress | → `done`, `wontdo`, `paused` |
| `paused` | Temporarily set aside | → `active` (via `sg continue`) |
| `done` | Completed | → `active` (via `sg reopen`) |
| `wontdo` | Abandoned/rejected/obsoleted | Terminal — no reversal |

Key distinction: `done` means "completed successfully." `wontdo` means "we're not doing this." Both are terminal, but `wontdo` is **non-blocking** — other sagas that depend on a wontdo saga can still be completed.

### Sub-Sagas

Break large work into pieces:

```bash
sg new "Parent task"                # Creates abc123
sg new "Sub-task 1" --parent abc123 # Creates abc123.1
sg new "Sub-task 2" --parent abc123 # Creates abc123.2
```

Hierarchical IDs make relationships obvious: `parent.1`, `parent.2`, etc.

- Cannot create sub-sagas under `done` or `wontdo` parents
- Cannot complete a parent while it has `active` children (use `--cascade` or `--force`)

### Dependencies

Hard dependencies block completion:

```bash
sg depend abc123 add def456    # abc123 blocked until def456 is done
sg done def456                 # Now abc123 can be completed
```

- Incomplete dependencies block `sg done` (shown as ✗ BLOCKING)
- Wontdo dependencies are non-blocking (shown as ⊘ wontdo)
- Done dependencies are satisfied (shown as ✓ done)

Soft relationships (informational only, no blocking):

```bash
sg relate abc123 add def456    # Link related work
```

### Claims

Prevent duplicate work across agents:

```bash
sg claim abc123                # Mark as "in progress" for your session
sg claim abc123 --duration 4h  # Custom duration
sg list --unclaimed            # Find available work
sg unclaim abc123              # Release claim
```

Claims are session-based using `user@ppid` identity:
- Same process = same session = "mine"
- Different process = different session = "claimed by other"
- Claims expire after a configurable duration (default 24h)

### Storage Scopes

Saga supports both global and project-local storage:

| Scope | Location | When Used |
|-------|----------|-----------|
| Local | `./.saga/` | When `sg init` has been run in the project |
| Global | `~/.saga/` | Always available as fallback |

Sagas are saved to local by default when a local store exists. `sg list` shows local sagas by default; use `--global` to include global sagas.

```bash
sg init                        # Create local .saga/ in project
sg new "Local task"            # Saved in ./.saga/
sg list --local               # Project only
sg list --global              # Global only
sg list                       # Both (default when in project)
```

## Agent Workflow

Saga shines when agents use it systematically:

### Before Starting

1. **Find work**: `sg ready` — see what's unblocked and unclaimed
2. **Read context**: `sg context <id>` — understand hierarchy, dependencies, claims
3. **Check knowledge** *(optional)*: `runes search "problem"` — has this been solved before?

### During Work

1. **Claim**: `sg claim <id>` — prevent duplicate work
2. **Log**: `sg log <id> "progress"` — track decisions and progress
3. **Decompose**: `sg new "Sub-task" --parent <id>` — break down complex work

### Completing Work

1. **Mark done**: `sg done <id>` — complete the saga
2. **Or abandon**: `sg wontdo <id> --reason "why"` — for rejected/obsoleted work
3. **Capture knowledge** *(optional)*: `runes add "Solution" --saga <id>`

### Reopening Work

If a done saga needs more work:

```bash
sg reopen <id> --reason "Bug found in implementation"
```

Only `done` sagas can be reopened (not `wontdo`).

## Commands

Full command reference with all flags and examples: **[Command Reference](docs/command-reference.md)**

Quick overview:

| Category | Commands |
|----------|----------|
| Create | `sg new`, `sg init` |
| View | `sg list`, `sg status`, `sg context`, `sg search`, `sg ready` |
| Modify | `sg edit`, `sg label`, `sg priority`, `sg log` |
| Complete | `sg done`, `sg wontdo`, `sg reopen` |
| Coordinate | `sg claim`, `sg unclaim`, `sg depend`, `sg relate` |
| Configure | `sg config` |
| Status change | `sg continue` |

Run `sg <command> --help` for detailed usage of any command.

## Integration with Runes

[Runes](https://github.com/sleeplesslord/runes) is a separate knowledge management tool that integrates with Saga:

```bash
# In saga: see linked knowledge
sg context <id>
# KNOWLEDGE (Runes)
#   • xr5h - Fixed auth timeout [auth-timeout-retry]

# In runes: link to saga
runes add "Auth fix" --saga <id>
```

Pattern: Saga tracks *doing*, Runes tracks *knowing*. When `sg done` detects runes is installed, it suggests capturing knowledge.

## Architecture

```
saga/
├── cmd/sg/           # CLI (cobra commands)
│   └── cmd/          # One file per command
├── internal/
│   ├── saga/         # Core types (Status, Priority, Saga struct)
│   └── store/        # Storage layer (JSONL, config, scoping)
└── skills/saga/      # Agent skill (SKILL.md + references)

Storage:
- Global: ~/.saga/sagas.jsonl
- Local: ./.saga/sagas.jsonl (if sg init)
- Format: JSON Lines (append-only)
- Config: .saga/config.json (local), ~/.saga/config.json (global)

Dependencies:
- github.com/spf13/cobra (CLI framework)
- Standard library only for core logic
```

## Naming

**Saga** — from Old Norse, a long story of heroic achievement. Fitting for tracking epic work.

**Hierarchical IDs** — `parent.1`, `parent.2` — like IP addresses or legal document numbering. Clear, sortable, human-readable.

## Philosophy

- **Explicit over implicit** — dependencies are declared, not inferred
- **Local over global** — project context stays with the project
- **Human and machine readable** — structured but not rigid
- **Compound improvement** — each solution makes future work easier
- **Done vs won't-do** — completion and abandonment are distinct outcomes

## See Also

- [Command Reference](docs/command-reference.md) — Full command docs with flags and examples
- [Runes](https://github.com/sleeplesslord/runes) — Knowledge management
- [Agent Skill](skills/saga/) — Teach agents to use Saga

## License

MIT
