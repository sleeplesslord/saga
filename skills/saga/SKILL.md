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

## Quick Reference

### Commands

```bash
# Read context (DO THIS FIRST)
sg context <id>                    # Full human-readable context
sg context <id> --format json      # Machine-readable for agents

# Search and list
sg list                            # Active sagas
sg list --all                      # Include done
sg list --label bug                # Filter by label
sg search "query"                  # Search titles/IDs/descriptions
sg status <id>                     # Brief status

# Create and update
sg new "title"                     # Create saga
sg new "title" --parent <id>     # Create sub-saga
sg new "title" --label bug --priority high --desc "details"
sg new "title" --deadline 20250415 # Set deadline (YYYYMMDD)
sg done <id>                       # Mark complete
sg done <id> --reason "message"    # Mark done with reason
sg done <id> --force               # Force completion
sg log <id> "progress note"        # Log work
sg log <id> --file notes.md       # Log from file

# Modify
sg edit <id> --title "new title"   # Update title
sg edit <id> --desc "new desc"     # Update description
sg edit <id> --deadline 20250415   # Set/edit deadline
sg edit <id> --deadline ""         # Clear deadline
sg label <id> add|remove <label>
sg priority <id> high|normal|low
sg depend <id> add|remove <target>
sg relate <id> add|remove <target>

# Claim/assign work
sg claim <id>                      # Claim for 24h (prevents others)
sg claim <id> --agent <name>       # Claim as specific agent
sg claim <id> --duration 4h        # Custom expiry
sg unclaim <id>                    # Release claim
sg list --unclaimed                # Find available work

# Find work
sg ready                           # List unblocked, unclaimed sagas
```

## Agent Workflow

### Before Starting Work

1. **Find available work** (or check if saga exists):
   ```bash
   sg ready                           # See what's ready to work on
   sg search "task name"              # Or search for specific task
   ```

2. **If saga exists**, read context:
   ```bash
   sg context <id> --format json
   ```
   - Check status (if done, ask user)
   - Check dependencies (any blocking?)
   - Check parent/child relationships
   - Read description for requirements
   - **Check if claimed** (if claimed by another agent, ask before proceeding)

3. **If no saga exists**, ask user to create one or create it:
   ```bash
   sg new "Implement feature X" --desc "Details from user"
   ```

4. **Claim the saga** to prevent duplicate work:
   ```bash
   sg claim <id> --agent <your-name>
   ```

### During Work

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
# - Blocking dependencies (show ✗ BLOCKING)
# - Active sub-sagas that need completion
```

**Create sub-sagas for large work:**
```bash
sg new "Sub-task 1" --parent <parent-id>
sg new "Sub-task 2" --parent <parent-id>
```

### Before Marking Complete

1. **Check all sub-sagas are done:**
   ```bash
   sg context <id>
   # Verify no active children
   ```

2. **Check all dependencies are done:**
   ```bash
   sg context <id>
   # Verify no blocking dependencies
   ```

3. **Mark as done:**
   ```bash
   sg done <id>
   ```
   
   With completion reason (logs to history):
   ```bash
   sg done <id> --reason "Implemented and tested"
   sg done <id> --reason "No longer needed - requirements changed"
   ```
   
   If blocked but user wants to force:
   ```bash
   sg done <id> --force
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
2. **Log Early and Often** - Use `sg log` for decisions and progress
3. **Dependencies Explicit** - Add blocking dependencies so completion checks work
4. **Sub-sagas for Detail** - Break large work into hierarchical sub-tasks
5. **Human Coordination** - Saga is the bridge between human planning and agent execution

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

Complete sub-sagas first, then:
  sg done abc123
```

### Incomplete Dependencies
```
Error: cannot mark "abc123" as done: 1 incomplete dependencie(s): [def456]

Complete these first:
  sg done def456
```

## Reference Files

- `references/saga-cli.md` - Full CLI reference
