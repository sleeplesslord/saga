---
name: saga
description: Integration with Saga task management system for agents. Use when working with sagas to track work, check context, log progress, and coordinate with human planning. Triggers on saga-related tasks like reading context, creating sagas, updating status, logging work, searching sagas, and checking dependencies.
---

# Saga Agent Skill

Integration with [Saga](https://github.com/sleeplesslord/saga) task management system.

## What is Saga

Saga is a hierarchical task tracker with:
- **Sagas** - Tasks/projects with titles, descriptions, plans, status, priority
- **Plans** - Implementation details separate from description (what/why vs how)
- **Sub-sagas** - Parent/child relationships (parent.1, parent.2)
- **Labels** - Tags for filtering
- **Dependencies** - Hard blocking dependencies
- **Relationships** - Soft informational links
- **Claims** - Session-based ownership with expiry (identity = user@ppid)
- **Config** - Local and global configuration for defaults

## Quick Reference


## Commands

```bash
# Read context (DO THIS FIRST)
saga context <id>                    # Full human-readable context
saga context <id> --format json      # Machine-readable for agents

# Search and list
saga list                            # Active sagas (local-only if .saga/ exists)
saga list --global                   # Include global sagas
saga list --status active|paused|done|wontdo
saga list --priority high|normal|low
saga list --mine                     # Your claimed sagas
saga list --unclaimed                # Unclaimed only
saga search "query"                  # Search titles/IDs/descriptions
saga status <id>                     # Brief status

# Ready queue (ready-to-work)
saga ready                           # Unclaimed, unblocked, no active children
saga ready --take                    # Claim the top ready saga

# Create and update
saga new "title"                     # Create saga
saga new "title" --parent <id>       # Create sub-saga (blocked if parent done/wontdo)
saga new "title" --label bug --priority high --desc "details"
saga new "title" --plan "1. Step one\n2. Step two"
saga new "title" --deadline 20250415  # Set deadline (YYYYMMDD)

# Complete or abandon
saga done <id> [<id> ...]           # Mark complete (multiple IDs)
saga done <id> --cascade             # Mark all active sub-sagas done first
saga done <id> --reason "why"        # Log reason in history
saga done <id> --quiet               # Suppress runes hint
saga done <id> --force               # Force completion despite blockers
saga wontdo <id> [--id ...] --reason "why"  # Abandon/reject/obsolete
saga wontdo <id> --cascade           # Mark all active sub-sagas as wontdo
saga wontdo <id> --quiet             # Suppress runes hint

# Reopen
saga reopen <id>                     # Reopen a done saga (sets back to active)
saga reopen <id> --reason "why"     # Reopen with reason logged in history

# Claiming
saga claim <id> [<id> ...]           # Claim saga(s) for your session
saga claim <id> --duration 4h        # Custom claim duration
saga unclaim <id> [<id> ...]         # Release claim(s)

# Modify
saga label <id> add|remove <label>
saga depend <id> add|remove <target>
saga relate <id> add|remove <target>
saga edit <id> --title "New title"
saga edit <id> --desc "New description"
saga edit <id> --plan "Implementation plan"  # Set/edit plan
saga edit <id> --plan ""                      # Clear plan
saga edit <id> --deadline 20250415   # Set/edit deadline
saga edit <id> --deadline ""          # Clear deadline
saga edit <id> --priority high|normal|low
saga plan <id>                       # View plan
saga plan <id> "Implementation steps" # Set plan
saga plan <id> --file plan.md        # Set plan from file
saga plan <id> - < plan.md           # Set plan from stdin
saga plan <id> --clear               # Remove plan
saga log <id> "progress note"        # Log work
saga log <id> --file notes.md        # Log from file
saga log <id> -                      # Log from stdin
# Resolution order: --file > "-" > argument > piped stdin.
# An argument wins over a pipe, so pass "-" when you mean stdin.

# Configuration
saga config                          # Show current config
saga config --claim-duration 4h      # Set local default claim duration
saga config --scope global --claim-duration 4h  # Set global default
```

## Agent Workflow

### Before Starting Work

1. **Check if saga exists** for this task:
   ```bash
   saga search "task name"
   ```

2. **If saga exists**, read context:
   ```bash
   saga context <id> --format json
   ```
   - Check status (if done, can `saga reopen`; if wontdo, ask user)
   - Check dependencies (wontdo = ⊘ non-blocking, incomplete = ✗ BLOCKING)
   - Check parent/child relationships
   - Check claim status (yours = [mine], other session = claimed by other)
   - Read description for requirements

3. **If no saga exists**, ask user to create one or create it:
   ```bash
   saga new "Implement feature X" --desc "Details from user"
   ```

### Finding Ready Work

```bash
saga ready              # Show sagas ready for you to work on
saga ready --take       # Claim the top ready saga automatically
```

"Ready" means: not claimed by another session, not blocked by incomplete dependencies, no active children. Your own claims show as [mine].

### During Work

**Claim the saga first:**
```bash
saga claim <id>           # Claims for your session (user@ppid)
saga claim <id> --duration 4h  # Custom duration
```

**Log progress regularly:**
```bash
saga log <id> "Started implementation"
saga log <id> "Decided on approach Y due to Z"
saga log <id> "Hit blocker: waiting for API"
```

**Check if blocked:**
```bash
saga context <id>
# Look for:
# - ✗ BLOCKING dependencies (incomplete)
# - ⊘ wontdo dependencies (non-blocking, terminal)
# - Active sub-sagas that need completion
```

**Create sub-sagas for large work:**
```bash
saga new "Sub-task 1" --parent <parent-id>
saga new "Sub-task 2" --parent <parent-id>
```
Note: Cannot create sub-sagas under done or wontdo parents.

### Before Marking Complete

1. **Check all sub-sagas are done:**
   ```bash
   saga context <id>
   # Verify no active children
   ```

2. **Check all dependencies are done:**
   ```bash
   saga context <id>
   # Verify no ✗ BLOCKING dependencies (⊘ wontdo is OK)
   ```

3. **Mark as done:**
   ```bash
   saga done <id>
   ```
   Or mark multiple at once:
   ```bash
   saga done abc123 def456
   ```
   With cascade (completes all sub-sagas first):
   ```bash
   saga done <id> --cascade
   ```

   If blocked but user wants to force:
   ```bash
   saga done <id> --force
   ```

### Abandoning Work

For sagas that are abandoned, rejected, or obsoleted (not "completed"):
```bash
saga wontdo <id> --reason "Requirements changed"
saga wontdo <id> --cascade            # Also marks active sub-sagas as wontdo
saga wontdo <id> --quiet               # Suppress runes hint
```

Wontdo is a terminal state (like done) but semantically distinct. It is non-blocking in dependency checks (shown as ⊘ wontdo).

### Reopening Completed Work

For sagas that were marked done but need more work:
```bash
saga reopen <id>                     # Sets status back to active
saga reopen <id> --reason "Bug found in implementation"
```

Only `done` sagas can be reopened (not `wontdo`). The reason is logged in history.

## Claim System

Claims are session-based using `user@ppid` identity:
- Same ppid = same session = "mine"
- Different ppid = different session = "claimed by other"
- Claims have an expiry time (shown in listings)
- Default duration: configured value (see `saga config`) or 24h fallback

**Config resolution for claim duration:**
`--duration` flag > local config (.saga/config.json) > global config (~/.saga/config.json) > 24h default

```bash
saga config                            # View current config
saga config --claim-duration 4h        # Set local default
saga config --scope global --claim-duration 4h  # Set global default
```

## Common Patterns

### Pattern: Dependency Chain

```bash
# Task B depends on Task A
saga new "Task A"                              # Creates abc123
saga new "Task B"                              # Creates def456
saga depend def456 add abc123                  # B depends on A

# Later, mark A done first
saga done abc123
saga done def456                               # Now works
```

### Pattern: Dependency with Abandonment

```bash
# Task B depends on Task A, but A is abandoned
saga new "Task A"                              # Creates abc123
saga new "Task B"                              # Creates def456
saga depend def456 add abc123                  # B depends on A

saga wontdo abc123 --reason "No longer needed"
# abc123 shows as ⊘ wontdo in def456's context (non-blocking)
saga done def456                               # Works because wontdo is terminal
```

### Pattern: Sub-task Decomposition

```bash
# Parent saga
saga new "Build auth system"                   # Creates abc123

# Sub-tasks
saga new "OAuth integration" --parent abc123   # Creates abc123.1
saga new "Session management" --parent abc123  # Creates abc123.2
saga new "Password reset" --parent abc123      # Creates abc123.3

# Work on sub-tasks
saga done abc123.1
saga done abc123.2
saga done abc123.3

# Complete parent
saga done abc123                               # Works when all children done
```

### Pattern: Cascade Completion

```bash
# Complete parent and all sub-sagas at once
saga done abc123 --cascade --reason "All work verified"
```

### Pattern: Claim and Ready Queue

```bash
# Find available work
saga ready                    # Shows unclaimed, unblocked sagas
saga ready --take             # Claim the top one and start working

# Claim specific sagas
saga claim abc123 def456      # Claim multiple at once
saga claim abc123 --duration 2h  # Short claim for quick task
```

### Pattern: Label-based Filtering

```bash
# Tag sagas
saga label abc123 add urgent
saga label def456 add urgent

# View urgent only
saga search "" --label urgent
```

## Plan Field: Description vs Plan

The `Plan` field stores *how* you'll implement a task, separate from `Description` which stores *what* and *why*. This separation keeps task context clean while giving agents a natural place to track implementation strategy.

**Description**: Problem statement, requirements, acceptance criteria, user-facing context
**Plan**: Implementation steps, technical approach, architecture decisions, execution order

### When to set a plan

- **On creation** for well-scoped tasks where the approach is known:
  ```bash
  saga new "Add password reset" --plan "1. Add reset token model\n2. POST /auth/reset endpoint\n3. Email integration\n4. Tests"
  ```
- **After reading context** for tasks where you need to explore first:
  ```bash
  saga context abc123        # Understand the task
  saga plan abc123 "Use existing email service, add token table to auth schema"
  ```
- **When approach changes** — update the plan, don't edit the description:
  ```bash
  saga plan abc123 "Switched to OTP approach after discovering email rate limits"
  ```

### When NOT to set a plan

- Simple/trivial tasks where the approach is obvious (just `saga log` progress)
- Tasks where you're still exploring and don't have a clear approach yet (log findings instead)
- When the description already fully captures the implementation (no duplication)

### Plan vs log

- **`saga plan`** — *intended* approach, persists across sessions, updated when strategy changes
- **`saga log`** — *actual* progress and decisions, append-only timeline

Plans can be revised. Logs are history. Use both: set the plan, then log as you execute it.

### Reading plans

Plans appear in `saga status`, `saga context`, and `saga context --format json` (as `saga.plan`). The dedicated `saga plan <id>` command is the quickest way to view just the plan.

### Clearing plans

Once a task is done, plans are kept for reference (part of the saga record). If a plan becomes stale during active work, clear it with `saga plan <id> --clear` or set a new one.

## Key Principles

1. **Context First** - Always read `saga context` before working
2. **Claim Your Work** - Use `saga claim` so other agents know you're on it
3. **Log Early and Often** - Use `saga log` for decisions and progress
4. **Dependencies Explicit** - Add blocking dependencies so completion checks work
5. **Wontdo for Abandonment** - Use `saga wontdo` (not `saga done`) for rejected/obsoleted work
6. **Sub-sagas for Detail** - Break large work into hierarchical sub-tasks
7. **Human Coordination** - Saga is the bridge between human planning and agent execution
8. **Plan Separately from Description** - Description = what & why; Plan = how. Use `saga plan` for implementation details so they don't clutter the task description.

## Error Handling

### Saga Not Found
```
Error: saga "abc123" not found

To see all sagas:
  saga list
```

### Has Active Children
```
Error: cannot mark "abc123" as done: has active sub-sagas

Complete sub-sagas first, use --cascade, or:
  saga done abc123 --force
```

### Incomplete Dependencies
```
Error: cannot mark "abc123" as done: 1 incomplete dependencie(s): [def456]

Complete these first, use --force, or mark as wontdo:
  saga done def456
  saga wontdo def456 --reason "No longer needed"
```

### Sub-saga Under Terminal Parent
```
Error: cannot create sub-saga under "abc123": parent is done

Parent must be active or paused to add sub-sagas.
```

## Reference Files

- `references/saga-cli.md` - Full CLI reference