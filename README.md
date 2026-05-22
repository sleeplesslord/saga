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
go install github.com/sleeplesslord/saga/cmd/saga@latest

# Initialize project storage
cd my-project
saga init

# Create a saga
saga new "Implement feature X" --desc "Details here" --priority high

# Break it down
saga new "Write tests" --parent <id>
saga new "Handle edge cases" --parent <id>

# See what's ready to work on
saga ready

# Claim and work
saga ready --take              # Claim the top ready saga
saga log <id> "Started implementation"

# Mark done
saga done <id> --reason "All tests passing"
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
| `paused` | Temporarily set aside | → `active` (via `saga continue`) |
| `done` | Completed | → `active` (via `saga reopen`) |
| `wontdo` | Abandoned/rejected/obsoleted | Terminal — no reversal |

Key distinction: `done` means "completed successfully." `wontdo` means "we're not doing this." Both are terminal, but `wontdo` is **non-blocking** — other sagas that depend on a wontdo saga can still be completed.

### Sub-Sagas

Break large work into pieces:

```bash
saga new "Parent task"                # Creates abc123
saga new "Sub-task 1" --parent abc123 # Creates abc123.1
saga new "Sub-task 2" --parent abc123 # Creates abc123.2
```

Hierarchical IDs make relationships obvious: `parent.1`, `parent.2`, etc.

- Cannot create sub-sagas under `done` or `wontdo` parents
- Cannot complete a parent while it has `active` children (use `--cascade` or `--force`)

### Dependencies

Hard dependencies block completion:

```bash
saga depend abc123 add def456    # abc123 blocked until def456 is done
saga done def456                 # Now abc123 can be completed
```

- Incomplete dependencies block `saga done` (shown as ✗ BLOCKING)
- Wontdo dependencies are non-blocking (shown as ⊘ wontdo)
- Done dependencies are satisfied (shown as ✓ done)

Soft relationships (informational only, no blocking):

```bash
saga relate abc123 add def456    # Link related work
```

### Claims

Prevent duplicate work across agents:

```bash
saga claim abc123                # Mark as "in progress" for your session
saga claim abc123 --duration 4h  # Custom duration
saga list --unclaimed            # Find available work
saga unclaim abc123              # Release claim
```

Claims are session-based using `user@ppid` identity:
- Same process = same session = "mine"
- Different process = different session = "claimed by other"
- Claims expire after a configurable duration (default 24h)

### Storage Scopes

Saga supports both global and project-local storage:

| Scope | Location | When Used |
|-------|----------|-----------|
| Local | `./.saga/` | When `saga init` has been run in the project |
| Global | `~/.saga/` | Always available as fallback |

Sagas are saved to local by default when a local store exists. `saga list` shows local sagas by default; use `--global` to include global sagas.

```bash
saga init                        # Create local .saga/ in project
saga new "Local task"            # Saved in ./.saga/
saga list --local               # Project only
saga list --global              # Global only
saga list                       # Both (default when in project)
```

## Agent Workflow

Saga shines when agents use it systematically:

### Before Starting

1. **Find work**: `saga ready` — see what's unblocked and unclaimed
2. **Read context**: `saga context <id>` — understand hierarchy, dependencies, claims
3. **Check knowledge** *(optional)*: `runes search "problem"` — has this been solved before?

### During Work

1. **Claim**: `saga claim <id>` — prevent duplicate work
2. **Log**: `saga log <id> "progress"` — track decisions and progress
3. **Decompose**: `saga new "Sub-task" --parent <id>` — break down complex work

### Completing Work

1. **Mark done**: `saga done <id>` — complete the saga
2. **Or abandon**: `saga wontdo <id> --reason "why"` — for rejected/obsoleted work
3. **Capture knowledge** *(optional)*: `runes add "Solution" --saga <id>`

### Reopening Work

If a done saga needs more work:

```bash
saga reopen <id> --reason "Bug found in implementation"
```

Only `done` sagas can be reopened (not `wontdo`).

## Commands

Full command reference with all flags and examples: **[Command Reference](docs/command-reference.md)**

Quick overview:

| Category | Commands |
|----------|----------|
| Create | `saga new`, `saga init` |
| View | `saga list`, `saga status`, `saga context`, `saga search`, `saga ready` |
| Modify | `saga edit`, `saga label`, `saga priority`, `saga log` |
| Complete | `saga done`, `saga wontdo`, `saga reopen` |
| Coordinate | `saga claim`, `saga unclaim`, `saga depend`, `saga relate` |
| Configure | `saga config` |
| Status change | `saga continue` |

Run `saga <command> --help` for detailed usage of any command.

Common aliases (LLM-friendly): `add`/`create`→`new`, `show`→`context`, `update`→`edit`, `complete`/`finish`→`done`, `cancel`/`skip`→`wontdo`, `assign`→`claim`, `unassign`/`release`→`unclaim`, `ls`→`list`, `todo`→`ready`, `resume`→`continue`, `comment`→`log`.

## Integration with Runes

[Runes](https://github.com/sleeplesslord/runes) is a separate knowledge management tool that integrates with Saga:

```bash
# In saga: see linked knowledge
saga context <id>
# KNOWLEDGE (Runes)
#   • xr5h - Fixed auth timeout [auth-timeout-retry]

# In runes: link to saga
runes add "Auth fix" --saga <id>
```

Pattern: Saga tracks *doing*, Runes tracks *knowing*. When `saga done` detects runes is installed, it suggests capturing knowledge.

## Architecture

```
saga/
├── cmd/saga/          # CLI (cobra commands)
│   └── cmd/          # One file per command
├── internal/
│   ├── saga/         # Core types (Status, Priority, Saga struct)
│   └── store/        # Storage layer (JSONL, config, scoping)
└── skills/saga/      # Agent skill (SKILL.md + references)

Storage:
- Global: ~/.saga/sagas.jsonl
- Local: ./.saga/sagas.jsonl (if saga init)
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
