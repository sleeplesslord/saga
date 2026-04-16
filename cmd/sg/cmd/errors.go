package cmd

import (
	"fmt"
)

// Error helpers for consistent, helpful error messages

func sagaNotFound(id string) error {
	return fmt.Errorf(`saga "%s" not found

To see all sagas:
  sg list

To search for a saga:
  sg search "%s"`, id, id)
}

func parentNotFound(id string) error {
	return fmt.Errorf(`parent saga "%s" not found

To see available parent sagas:
  sg list --status active

To create a root saga (no parent):
  sg new "title"`, id)
}

func parentDone(id string) error {
	return fmt.Errorf(`cannot create sub-saga: parent "%s" is already done or won't-do

Active parents:
  sg list --status active

Or use a different parent saga`, id)
}

func incompleteDependencies(id string, deps []string) error {
	return fmt.Errorf(`cannot mark "%s" as done: %d incomplete dependencie(s): %v

Complete these first:
  sg done <id>

Or force completion (not recommended):
  sg done "%s" --force`, id, len(deps), deps, id)
}

func activeChildren(id string) error {
	return fmt.Errorf(`cannot mark "%s" as done: has active sub-sagas

Complete sub-sagas first, then:
  sg done "%s"

Or force completion (not recommended):
  sg done "%s" --force`, id, id, id)
}

func circularDependency() error {
	return fmt.Errorf(`cannot add dependency: would create circular reference

Dependencies must form a directed acyclic graph.
Check existing dependencies with:
  sg status <id>`)
}

func alreadyHasLabel(id, label string) error {
	return fmt.Errorf(`saga "%s" already has label "%s"`, id, label)
}

func missingLabel(id, label string) error {
	return fmt.Errorf(`saga "%s" does not have label "%s"

Current labels:
  sg status "%s"`, id, label, id)
}

func invalidPriority(p string) error {
	return fmt.Errorf(`invalid priority "%s"

Valid priorities: high, normal, low

Examples:
  sg new "title" --priority high
  sg priority abc123 high`, p)
}

func invalidStatus(s string) error {
	return fmt.Errorf(`invalid status "%s"

Valid statuses: active, paused, done, wontdo`, s)
}

func storeError(err error) error {
	return fmt.Errorf(`store error: %w

Try:
  sg init`, err)
}
