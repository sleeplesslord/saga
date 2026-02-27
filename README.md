# Saga

Task management for agent workflows. Track work, capture knowledge, coordinate between humans and agents.

## Why Saga

Most task trackers are built for human teams. Saga is built for **human-agent collaboration**:

- **Sagas track tasks** — what needs to be done, by whom, in what order
- **Runes capture knowledge** — solutions, patterns, lessons learned (separate but integrated)
- **Hierarchical structure** — parent/child relationships mirror real work breakdown
- **Claim system** — agents coordinate without stepping on each other
- **Context command** — agents understand full picture before acting

## Quick Start

```bash
# Install
go install github.com/sleeplesslord/saga/cmd/sg@latest

# Initialize project storage
cd my-project
sg init

# Create a saga
sg new "Implement feature X" --desc "Details here"

# See what's ready to work on
sg ready

# Claim and work
sg claim <id>
sg log <id> "Started implementation"

# Mark done
sg done <id>
```

## Core Concepts

### Sagas

A saga is a task or project. It has:
- **Title** — short description
- **Description** — details, requirements
- **Status** — active, paused, done
- **Priority** — high, normal (default), low
- **Labels** — tags for filtering
- **Hierarchy** — parent/child relationships

### Sub-Sagas

Break large work into pieces:

```bash
sg new "Parent task"                              # Creates abc123
sg new "Sub-task 1" --parent abc123              # Creates abc123.1
sg new "Sub-task 2" --parent abc123              # Creates abc123.2
```

Hierarchical IDs make relationships obvious: `parent.1`, `parent.2`, etc.

### Dependencies

Hard dependencies block completion:

```bash
sg depend abc123 add def456    # abc123 blocked until def456 done
sg done def456                 # Now abc123 can be completed
```

Soft relationships (informational only):

```bash
sg relate abc123 add def456    # Link related work
```

### Claims

Prevent duplicate work:

```bash
sg claim abc123                # Mark as "in progress"
sg list --unclaimed            # Find available work
sg unclaim abc123              # Release claim
```

Claims expire after 24 hours (auto-cleanup of stale claims).

## Commands

### Essential

| Command | Description |
|---------|-------------|
| `sg new <title>` | Create saga |
| `sg ready` | List unblocked, unclaimed sagas |
| `sg list` | Show all sagas |
| `sg status <id>` | Show saga details |
| `sg context <id>` | Full context (hierarchy, deps, runes) |
| `sg done <id>` | Mark complete |

### Organization

| Command | Description |
|---------|-------------|
| `sg sub <parent> <title>` | Create sub-saga |
| `sg label <id> add <label>` | Add label |
| `sg priority <id> high` | Set priority |
| `sg log <id> <message>` | Log progress |

### Coordination

| Command | Description |
|---------|-------------|
| `sg claim <id>` | Claim for work |
| `sg depend <id> add <target>` | Add dependency |
| `sg relate <id> add <target>` | Add relationship |
| `sg search <query>` | Find sagas |

### Storage Scopes

Both global and project-local storage:

```bash
sg init                        # Create local .saga/ in project
sg new "Local task"            # Saved in ./.saga/
sg new "Global task" --global  # Saved in ~/.saga/
sg list --local               # Project only
sg list --global              # Global only
sg list                       # Both (default if in project)
```

## Agent Workflow

Saga shines when agents use it:

### Before Starting

1. **Check ready**: `sg ready` — find available work
2. **Read context**: `sg context <id>` — understand the full picture
3. **Check runes**: `runes search "problem"` — have we solved this?

### During Work

1. **Claim**: `sg claim <id>` — prevent duplicate work
2. **Log**: `sg log <id> "progress"` — track decisions
3. **Create sub-sagas**: break down complex work

### After Solving

1. **Mark done**: `sg done <id>`
2. **Capture knowledge**: `runes add "Solution" --saga <id>`

## Integration with Runes

[Runes](https://github.com/sleeplesslord/runes) captures knowledge. Linked to sagas:

```bash
# In saga: see linked knowledge
sg context <id>
# KNOWLEDGE (Runes)
#   • xr5h - Fixed auth timeout [auth-timeout-retry]

# In runes: link to saga
runes add "Auth fix" --saga <id>
```

Pattern: Saga tracks *doing*, Runes tracks *knowing*.

## Architecture

```
saga/
├── cmd/sg/           # CLI commands
│   └── cmd/
│       ├── new.go
│       ├── list.go
│       ├── claim.go
│       └── ...
├── internal/
│   ├── saga/         # Core types
│   └── store/        # Storage layer
└── skills/           # Agent skills

Storage:
- Global: ~/.saga/sagas.jsonl
- Local: ./.saga/sagas.jsonl (if sg init)
- Format: JSON Lines (append-only)

Dependencies:
- github.com/spf13/cobra (CLI)
- Standard library only for core
```

## Naming

**Saga** — from Old Norse, a long story of heroic achievement. Fitting for tracking epic work.

**Hierarchical IDs** — `parent.1`, `parent.2` — like IP addresses or legal document numbering. Clear, sortable, human-readable.

## Philosophy

- **Explicit over implicit** — dependencies are declared, not inferred
- **Local over global** — project context stays with the project
- **Human and machine readable** — structured but not rigid
- **Compound improvement** — each solution makes future work easier

## See Also

- [Runes](https://github.com/sleeplesslord/runes) — Knowledge management
- [Agent Skill](skills/saga-agent/) — Teach agents to use Saga

## License

MIT
