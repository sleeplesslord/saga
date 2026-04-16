---
name: saga
description: Integration with Saga task management system for agents. Use when working with sagas to track work, check context, log progress, and coordinate with human planning. Triggers on saga-related tasks like reading context, creating sagas, updating status, logging work, searching sagas, and checking dependencies.
---

# Saga Agent Skill

Integration with [Saga](https://github.com/sleeplesslord/saga) task management system.

## What is Saga

Saga is a hierarchical task tracker with:
- **Sagas** - Tasks/projects with titles, descriptions, status, priority
- **Sub-sagas** - Parent/child relationships (parent.1, parent.2)
- **Labels** - Tags for filtering
- **Dependencies** - Hard blocking dependencies
- **Relationships** - Soft informational links
- **Claims** - Session-based ownership with expiry (identity = user@ppid)
- **Config** - Local and global configuration for defaults

## Quick Reference

### Commands

```bash
# Read context (DO THIS FIRST)
sg context <id>                    # Full human-readable context
sg context <id> --format json      # Machine-readable for agents

# Search and list
sg list                            # Active sagas (local-only if .saga/ exists)
sg list --global                   # Include global sagas
sg list --status active|paused|done|wontdo
sg list --priority high|normal|low
sg list --mine                     # Your claimed sagas
sg list --unclaimed                # Unclaimed only
sg search "query"                  # Search titles/IDs/descriptions
sg status <id>                     # Brief status

# Ready queue (ready-to-work)
sg ready                           # Unclaimed, unblocked, no active children
sg ready --take                    # Claim the top ready saga

# Create and update
sg new "title"                     # Create saga
sg new "title" --parent <id>       # Create sub-saga (blocked if parent done/wontdo)
sg new "title" --label bug --priority high --desc "details"
sg new "title" --deadline 20250415  # Set deadline (YYYYMMDD)

# Complete or abandon
sg done <id> [<id> ...]           # Mark complete (multiple IDs)
sg done <id> --cascade             # Mark all active sub-sagas done first
sg done <id> --reason "why"        # Log reason in history
sg done <id> --quiet               # Suppress runes hint
sg done <id> --force               # Force completion despite blockers
sg wontdo <id> [--id ...] --reason "why"  # Abandon/reject/obsolete
sg wontdo <id> --cascade           # Mark all active sub-sagas as wontdo
sg wontdo <id> --quiet             # Suppress runes hint

# Reopen
sg reopen <id>                     # Reopen a done saga (sets back to active)
sg reopen <id> --reason "why"     # Reopen with reason logged in history

# Claiming
sg claim <id> [<id> ...]           # Claim saga(s) for your session
sg claim <id> --duration 4h        # Custom claim duration
sg unclaim <id> [<id> ...]         # Release claim(s)

# Modify
sg label <id> add|remove <label>
sg depend <id> add|remove <target>
sg relate <id> add|remove <target>
sg edit <id> --title "New title"
sg edit <id> --desc "New description"
sg edit <id> --deadline 20250415   # Set/edit deadline
sg edit <id> --deadline ""          # Clear deadline
sg edit <id> --priority high|normal|low
sg log <id> "progress note"        # Log work
sg log <id> --file notes.md        # Log from file

# Configuration
sg config                          # Show current config
sg config --claim-duration 4h      # Set local default claim duration
sg config --scope global --claim-duration 4h  # Set global default
```

## Agent Workflow

### Before Starting Work

1. **Check if saga exists** for this task:
   ```bash
   sg search "task name"
   ```

2. **If saga exists**, read context:
   ```bash
   sg context <id> --format json
   ```
   - Check status (if done, can `sg reopen`; if wontdo, ask user)
   - Check dependencies (wontdo = ⊘ non-blocking, incomplete = ✗ BLOCKING)
   - Check parent/child relationships
   - Check claim status (yours = [mine], other session = claimed by other)
   - Read description for requirements

3. **If no saga exists**, ask user to create one or create it:
   ```bash
   sg new "Implement feature X" --desc "Details from user"
   ```

### Finding Ready Work

```bash
sg ready              # Show sagas ready for you to work on
sg ready --take       # Claim the top ready saga automatically
```

"Ready" means: not claimed by another session, not blocked by incomplete dependencies, no active children. Your own claims show as [mine].

### During Work

**Claim the saga first:**
```bash
sg claim <id>           # Claims for your session (user@ppid)
sg claim <id> --duration 4h  # Custom duration
```

**Log progress regularly:**
```bash
sg log <id> "Started implementation"
sg log <id> "Decided on approach Y due to Z"
sg log <id> "Hit blocker: waiting for API"
```

**Check if blocked:**
```bash
sg context <id>
# Look for:
# - ✗ BLOCKING dependencies (incomplete)
# - ⊘ wontdo dependencies (non-blocking, terminal)
# - Active sub-sagas that need completion
```

**Create sub-sagas for large work:**
```bash
sg new "Sub-task 1" --parent <parent-id>
sg new "Sub-task 2" --parent <parent-id>
```
Note: Cannot create sub-sagas under done or wontdo parents.

### Before Marking Complete

1. **Check all sub-sagas are done:**
   ```bash
   sg context <id>
   # Verify no active children
   ```

2. **Check all dependencies are done:**
   ```bash
   sg context <id>
   # Verify no ✗ BLOCKING dependencies (⊘ wontdo is OK)
   ```

3. **Mark as done:**
   ```bash
   sg done <id>
   ```
   Or mark multiple at once:
   ```bash
   sg done abc123 def456
   ```
   With cascade (completes all sub-sagas first):
   ```bash
   sg done <id> --cascade
   ```

   If blocked but user wants to force:
   ```bash
   sg done <id> --force
   ```

### Abandoning Work

For sagas that are abandoned, rejected, or obsoleted (not "completed"):
```bash
sg wontdo <id> --reason "Requirements changed"
sg wontdo <id> --cascade            # Also marks active sub-sagas as wontdo
sg wontdo <id> --quiet               # Suppress runes hint
```

Wontdo is a terminal state (like done) but semantically distinct. It is non-blocking in dependency checks (shown as ⊘ wontdo).

### Reopening Completed Work

For sagas that were marked done but need more work:
```bash
sg reopen <id>                     # Sets status back to active
sg reopen <id> --reason "Bug found in implementation"
```

Only `done` sagas can be reopened (not `wontdo`). The reason is logged in history.

## Claim System

Claims are session-based using `user@ppid` identity:
- Same ppid = same session = "mine"
- Different ppid = different session = "claimed by other"
- Claims have an expiry time (shown in listings)
- Default duration: configured value (see `sg config`) or 24h fallback

**Config resolution for claim duration:**
`--duration` flag > local config (.saga/config.json) > global config (~/.saga/config.json) > 24h default

```bash
sg config                            # View current config
sg config --claim-duration 4h        # Set local default
sg config --scope global --claim-duration 4h  # Set global default
```

## Common Patterns

### Pattern: Dependency Chain

```bash
# Task B depends on Task A
sg new "Task A"                              # Creates abc123
sg new "Task B"                              # Creates def456
sg depend def456 add abc123                  # B depends on A

# Later, mark A done first
sg done abc123
sg done def456                               # Now works
```

### Pattern: Dependency with Abandonment

```bash
# Task B depends on Task A, but A is abandoned
sg new "Task A"                              # Creates abc123
sg new "Task B"                              # Creates def456
sg depend def456 add abc123                  # B depends on A

sg wontdo abc123 --reason "No longer needed"
# abc123 shows as ⊘ wontdo in def456's context (non-blocking)
sg done def456                               # Works because wontdo is terminal
```

### Pattern: Sub-task Decomposition

```bash
# Parent saga
sg new "Build auth system"                   # Creates abc123

# Sub-tasks
sg new "OAuth integration" --parent abc123   # Creates abc123.1
sg new "Session management" --parent abc123  # Creates abc123.2
sg new "Password reset" --parent abc123      # Creates abc123.3

# Work on sub-tasks
sg done abc123.1
sg done abc123.2
sg done abc123.3

# Complete parent
sg done abc123                               # Works when all children done
```

### Pattern: Cascade Completion

```bash
# Complete parent and all sub-sagas at once
sg done abc123 --cascade --reason "All work verified"
```

### Pattern: Claim and Ready Queue

```bash
# Find available work
sg ready                    # Shows unclaimed, unblocked sagas
sg ready --take             # Claim the top one and start working

# Claim specific sagas
sg claim abc123 def456      # Claim multiple at once
sg claim abc123 --duration 2h  # Short claim for quick task
```

### Pattern: Label-based Filtering

```bash
# Tag sagas
sg label abc123 add urgent
sg label def456 add urgent

# View urgent only
sg search "" --label urgent
```

## Key Principles

1. **Context First** - Always read `sg context` before working
2. **Claim Your Work** - Use `sg claim` so other agents know you're on it
3. **Log Early and Often** - Use `sg log` for decisions and progress
4. **Dependencies Explicit** - Add blocking dependencies so completion checks work
5. **Wontdo for Abandonment** - Use `sg wontdo` (not `sg done`) for rejected/obsoleted work
6. **Sub-sagas for Detail** - Break large work into hierarchical sub-tasks
7. **Human Coordination** - Saga is the bridge between human planning and agent execution

## Error Handling

### Saga Not Found
```
Error: saga "abc123" not found

To see all sagas:
  sg list
```

### Has Active Children
```
Error: cannot mark "abc123" as done: has active sub-sagas

Complete sub-sagas first, use --cascade, or:
  sg done abc123 --force
```

### Incomplete Dependencies
```
Error: cannot mark "abc123" as done: 1 incomplete dependencie(s): [def456]

Complete these first, use --force, or mark as wontdo:
  sg done def456
  sg wontdo def456 --reason "No longer needed"
```

### Sub-saga Under Terminal Parent
```
Error: cannot create sub-saga under "abc123": parent is done

Parent must be active or paused to add sub-sagas.
```

## Reference Files

- `references/saga-cli.md` - Full CLI reference