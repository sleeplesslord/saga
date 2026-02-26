# Saga Roadmap

Task management for agent workflows.

---

## Phase 1: Core (MVP)

**Goal:** Basic task tracking that actually works.

- [ ] `sg new <title>` - Create saga
- [ ] `sg list` - Show active sagas
- [ ] `sg status <id>` - Current state + recent history
- [ ] `sg done <id>` - Mark complete
- [ ] `sg continue <id>` - Resume context

**Storage:** JSON file(s) in `.saga/` or similar

---

## Phase 2: Structure

**Goal:** Handle real complexity.

- [ ] `sg sub <parent-id> <title>` - Sub-sagas
- [ ] Labels: `--label bug`, `--label feature`
- [ ] `sg list --label bug --status active`
- [ ] Dependencies: `sg block <id> --on <other-id>`
- [ ] Priority: `--priority high|normal|low`

---

## Phase 3: Linking & Context

**Goal:** Connect sagas to related resources.

- [ ] Attach files/refs: `sg link <id> /path/to/file`
- [ ] Link Runes entries: `sg link <id> --runes auth-notes`
- [ ] View all links: `sg links <id>`
- [ ] Optional context fetch: `sg status <id> --with-context` (pulls Runes, shows linked files)

---

## Phase 4: Agent Friendly

**Goal:** Agents work naturally with Saga.

- [ ] Clean human-readable format (no forced structure)
- [ ] Agents can read/write saga state directly
- [ ] `sg log <id> "agent did X"` - manual checkpoint when useful
- [ ] Track child sessions: `sg spawn <id> --session <session-key>`

---

## Phase 5: Collaboration

**Goal:** Multiple agents, same saga.

- [ ] Concurrent saga access
- [ ] Conflict resolution
- [ ] Saga assignment: `--assign agent-name`
- [ ] Cross-session saga sharing

---

## Future / Maybe

- [ ] Time tracking
- [ ] Sprint/milestone grouping
- [ ] Visual graph view (`sg graph`)
- [ ] Git integration (branches per saga)
- [ ] Notification rules (Huginn integration)

---

## Design Principles

1. **Plain language** — sub-saga, not "chapter"
2. **Works without agents** — human-usable CLI
3. **Agent-native** — structured for LLM consumption
4. **Grows with need** — simple now, powerful later
