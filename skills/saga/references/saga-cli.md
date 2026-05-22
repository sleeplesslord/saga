# Saga CLI Reference

Complete reference for all `saga` commands.

## Global

All commands work with both global (`~/.saga/`) and local (`./.saga/`) stores.

## Commands

### init

Initialize local saga storage in current directory.

```bash
saga init
```

Creates `./.saga/sagas.jsonl` for project-scoped sagas.

### new

Create a new saga or sub-saga.

```bash
# Root saga
saga new "Implement feature"

# With options
saga new "Fix bug" --desc "Detailed description" --label bug --priority high

# Sub-saga (hierarchical ID: parent.1, parent.2)
saga new "Sub-task" --parent abc123

# With deadline
saga new "Ship feature" --deadline 20250415
```

**Flags:**
- `--parent <id>` - Create as sub-saga under parent (blocked if parent is done or wontdo)
- `--label <name>` - Add label (can use multiple)
- `--priority high|normal|low` - Set priority (default: normal)
- `--desc "text"` - Add description
- `--deadline YYYYMMDD` - Set deadline

### list

List sagas with filtering. Default scope is local-only when `.saga/` exists; use `--global` to include global sagas.

```bash
saga list                            # Active sagas (local-only if .saga/ exists)
saga list --global                   # Include global sagas
saga list --all                      # Include done/wontdo sagas
saga list --status active            # Filter by status
saga list --status paused
saga list --status done
saga list --status wontdo
saga list --priority high            # Filter by priority
saga list --priority normal
saga list --priority low
saga list --mine                     # Your claimed sagas (same ppid session)
saga list --unclaimed                # Unclaimed only
saga list --label bug                # Filter by label
```

**Sorting:** By deadline (soonest first), then priority (high → normal → low), then updated time.

**Output:** Shows claim expiry time in listings when a saga is claimed.

### ready

Show sagas that are ready to work on. Excludes:
- Sagas claimed by other sessions (your own claims shown as [mine])
- Sagas blocked by incomplete dependencies
- Sagas with active children

```bash
saga ready                           # Show ready sagas
saga ready --take                    # Claim the top ready saga
```

**Sorting:** By deadline (soonest first), then priority (high → normal → low), then updated time.

**`--take`:** Claims the top result from the ready queue for your session.

### status

Show saga details.

```bash
saga status abc123
```

Shows: ID, title, description, status, priority, labels, parent/children counts, claim info, recent history.

### context ⭐

**Most important command for agents.**

Show full context including hierarchy, dependencies, and relationships.

```bash
saga context abc123              # Human-readable
saga context abc123 --format json   # Machine-readable
```

**Output includes:**
- Saga details (description converts literal `\n` to real newlines)
- Parent info (if sub-saga)
- Children list with status
- Dependencies with status indicators:
  - ✓ done (complete, non-blocking)
  - ⊘ wontdo (terminal, non-blocking)
  - ✗ BLOCKING (incomplete, blocks completion)
- Related sagas
- Claim info (who owns it, expiry)
- Recent history (last 10 entries)
- Summary: can-complete status (wontdo treated as terminal)

### done

Mark saga(s) as complete.

```bash
saga done abc123                          # Mark done (checks children/deps)
saga done abc123 def456                   # Mark multiple done
saga done abc123 --cascade               # Mark all active sub-sagas as done first
saga done abc123 --reason "Verified"     # Log reason in history
saga done abc123 --quiet                 # Suppress runes hint (auto-suppressed in non-TTY)
saga done abc123 --force                 # Force completion despite blockers
```

**Flags:**
- `--cascade` - Mark all active sub-sagas as done before completing parent
- `--reason "text"` - Log reason in history
- `--quiet` - Suppress runes hint (auto-suppressed in non-TTY environments)
- `--force` - Force completion even with active children or incomplete dependencies

**Blocks if (without --force):**
- Has active children (unless --cascade)
- Has incomplete dependencies (wontdo dependencies are OK)

### wontdo

Mark saga(s) as abandoned/rejected/obsoleted. Distinct terminal state from "done".

```bash
saga wontdo abc123 --reason "Requirements changed"
saga wontdo abc123 def456               # Mark multiple as wontdo
saga wontdo abc123 --cascade            # Mark all active sub-sagas as wontdo
saga wontdo abc123 --quiet              # Suppress runes hint
```

**Flags:**
- `--reason "text"` - Log reason in history (recommended)
- `--cascade` - Mark all active sub-sagas as wontdo first
- `--quiet` - Suppress runes hint (auto-suppressed in non-TTY environments)

**Key behaviors:**
- Terminal state (like done)
- Non-blocking in dependency checks (shown as ⊘ wontdo, not ✗ BLOCKING)
- `canComplete` treats wontdo as terminal
- Cannot create sub-sagas under wontdo parents

### continue

Resume a paused saga.

```bash
saga continue abc123
```

### log

Add work log entry.

```bash
saga log abc123 "Progress note"
saga log abc123 --file notes.md
```

Appears in saga history.

### label

Manage labels.

```bash
saga label abc123 add bug
saga label abc123 remove bug
```

### priority

Change priority.

```bash
saga priority abc123 high
saga priority abc123 normal
saga priority abc123 low
```

### depend

Manage hard dependencies (blocking).

```bash
saga depend abc123 add def456     # abc123 now depends on def456
saga depend abc123 remove def456  # Remove dependency
```

**Blocks completion** until target is done or wontdo.

**Prevents** circular dependencies (A→B→A).

### relate

Manage soft relationships (informational).

```bash
saga relate abc123 add def456     # Mark as related
saga relate abc123 remove def456  # Remove relationship
```

**Does not block** completion. For reference only.

### claim

Claim saga(s) for your session. Claim identity is `user@ppid` (session-based).

```bash
saga claim abc123                       # Claim for your session
saga claim abc123 def456                # Claim multiple sagas
saga claim abc123 --duration 4h         # Custom claim duration
```

**Flags:**
- `--duration <time>` - Claim duration (e.g., 4h, 30m). Default: configured value or 24h.

**Claim duration resolution:** `--duration` flag > local config (.saga/config.json) > global config (~/.saga/config.json) > 24h default

**Ownership:** Based on ppid (process session ID). Same ppid = same session = "mine". Different ppid = different session = "claimed by other".

### unclaim

Release claim(s) on saga(s).

```bash
saga unclaim abc123                    # Release claim
saga unclaim abc123 def456             # Release multiple claims
```

### edit

Edit saga properties.

```bash
saga edit abc123 --title "New title"
saga edit abc123 --desc "New description"
saga edit abc123 --deadline 20250415   # Set/edit deadline
saga edit abc123 --deadline ""          # Clear deadline
saga edit abc123 --priority high        # Set priority
```

**Flags:**
- `--title "text"` - Change title
- `--desc "text"` - Change description
- `--deadline YYYYMMDD` - Set deadline (empty string to clear)
- `--priority high|normal|low` - Change priority

### config

View and set configuration.

```bash
saga config                            # Show current config
saga config --claim-duration 4h        # Set default claim duration (local)
saga config --scope global --claim-duration 4h  # Set in ~/.saga/config.json
```

**Flags:**
- `--claim-duration <time>` - Default claim duration (e.g., 4h, 30m, 24h)
- `--scope global` - Write to global config (~/.saga/config.json) instead of local (.saga/config.json)

**Config resolution for claim duration:**
`--duration` flag > local config (.saga/config.json) > global config (~/.saga/config.json) > 24h default

### search

Search sagas.

```bash
# Text search
saga search "auth"                    # Search titles/IDs/descriptions

# Filter only
saga search "" --label bug            # All with bug label
saga search "" --status active        # All active
saga search "" --priority high        # All high priority

# Combined
saga search "fix" --label urgent --status active
```

## Status Values

| Status | Terminal | Blocks Parents | Blocks Dependants |
|--------|----------|----------------|-------------------|
| active | No | Yes | Yes (✗ BLOCKING) |
| paused | No | Yes | Yes (✗ BLOCKING) |
| done | Yes | No | No (✓ done) |
| wontdo | Yes | No | No (⊘ wontdo) |

## Exit Codes

- `0` - Success
- `1` - Error (with helpful message)

## File Locations

- Global: `~/.saga/sagas.jsonl`
- Local: `./.saga/sagas.jsonl` (if `saga init` run)
- Local config: `./.saga/config.json`
- Global config: `~/.saga/config.json`

All data files are JSON Lines format, human-readable.