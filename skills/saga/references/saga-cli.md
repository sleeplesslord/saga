# Saga CLI Reference

Complete reference for all `sg` commands.

## Global

All commands work with both global (`~/.saga/`) and local (`./.saga/`) stores.

## Commands

### init

Initialize local saga storage in current directory.

```bash
sg init
```

Creates `./.saga/sagas.jsonl` for project-scoped sagas.

### new

Create a new saga or sub-saga.

```bash
# Root saga
sg new "Implement feature"

# With options
sg new "Fix bug" --desc "Detailed description" --label bug --priority high

# Sub-saga (hierarchical ID: parent.1, parent.2)
sg new "Sub-task" --parent abc123
```

**Flags:**
- `--parent <id>` - Create as sub-saga under parent
- `--label <name>` - Add label (can use multiple)
- `--priority high|normal|low` - Set priority (default: normal)
- `--desc "text"` - Add description

### list

List sagas with filtering.

```bash
sg list                      # Active sagas (default)
sg list --all               # Include done sagas
sg list --local            # Project sagas only
sg list --global           # Global sagas only
sg list --label bug        # Filter by label
sg list --unclaimed        # Show only unclaimed (available work)
```

**Sorting:** By priority (high → normal → low), then by updated time.

### status

Show saga details.

```bash
sg status abc123
```

Shows: ID, title, description, status, priority, labels, parent/children counts, recent history.

### context ⭐

**Most important command for agents.**

Show full context including hierarchy, dependencies, and relationships.

```bash
sg context abc123              # Human-readable
sg context abc123 --format json   # Machine-readable
```

**Output includes:**
- Saga details
- Parent info (if sub-saga)
- Children list with status
- Dependencies with ✓ done / ✗ BLOCKING status
- Related sagas
- Recent history (last 10 entries)
- Summary: can-complete status

### done

Mark saga as complete.

```bash
sg done abc123          # Mark done (checks children/deps)
sg done abc123 --force  # Force completion
```

**Blocks if:**
- Has active children
- Has incomplete dependencies

### continue

Resume a paused saga.

```bash
sg continue abc123
```

### log

Add work log entry.

```bash
sg log abc123 "Progress note"
sg log abc123 --file notes.md
```

Appears in saga history.

### label

Manage labels.

```bash
sg label abc123 add bug
sg label abc123 remove bug
```

### priority

Change priority.

```bash
sg priority abc123 high
sg priority abc123 normal
sg priority abc123 low
```

### depend

Manage hard dependencies (blocking).

```bash
sg depend abc123 add def456     # abc123 now depends on def456
sg depend abc123 remove def456  # Remove dependency
```

**Blocks completion** until target is done.

**Prevents** circular dependencies (A→B→A).

### relate

Manage soft relationships (informational).

```bash
sg relate abc123 add def456     # Mark as related
sg relate abc123 remove def456  # Remove relationship
```

**Does not block** completion. For reference only.

### claim

Claim a saga to prevent others from working on it.

```bash
sg claim abc123                    # Claim for 24h
sg claim abc123 --agent claude     # Claim as specific agent
sg claim abc123 --duration 4h      # Custom duration
sg unclaim abc123                  # Release claim
sg list --unclaimed               # Find available work
```

**Use case:** Multiple agents coordinating—claim before starting work.
**Expiry:** Claims auto-expire after 24h (or custom duration).

### search

Search sagas.

```bash
# Text search
sg search "auth"                    # Search titles/IDs/descriptions

# Filter only
sg search "" --label bug            # All with bug label
sg search "" --status active        # All active
sg search "" --priority high        # All high priority

# Combined
sg search "fix" --label urgent --status active
```

## Exit Codes

- `0` - Success
- `1` - Error (with helpful message)

## File Locations

- Global: `~/.saga/sagas.jsonl`
- Local: `./.saga/sagas.jsonl` (if `sg init` run)

Both are JSON Lines format, human-readable.
