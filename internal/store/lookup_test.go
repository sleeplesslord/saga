package store

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sleeplesslord/saga/internal/saga"
)

// newTestStoreWithLocal builds a store with both a global and a project scope.
func newTestStoreWithLocal(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	global := filepath.Join(dir, "global", "sagas.jsonl")
	local := filepath.Join(dir, "project", ".saga", "sagas.jsonl")
	for _, path := range []string{global, local} {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("creating %s: %v", path, err)
		}
		if err := os.WriteFile(path, nil, 0644); err != nil {
			t.Fatalf("seeding %s: %v", path, err)
		}
	}
	return &Store{globalPath: global, localPath: local}
}

func TestNotFoundNamesGlobalStore(t *testing.T) {
	s := newTestStore(t)

	_, err := s.GetByID("missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if !strings.Contains(err.Error(), s.globalPath) {
		t.Fatalf("error should name the store it searched, got %q", err.Error())
	}
}

func TestNotFoundNamesBothStores(t *testing.T) {
	s := newTestStoreWithLocal(t)

	_, err := s.GetByID("missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	// Whether an ID resolves depends on which stores are in scope, so both
	// have to be named — otherwise "not found" reads as "does not exist".
	if !strings.Contains(err.Error(), s.globalPath) {
		t.Fatalf("error should name the global store, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), s.localPath) {
		t.Fatalf("error should name the project store, got %q", err.Error())
	}
}

func TestMutateNotFoundNamesSearchedStores(t *testing.T) {
	s := newTestStoreWithLocal(t)

	_, err := s.Mutate("missing", func(*saga.Saga) error { return nil })
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if !strings.Contains(err.Error(), s.localPath) {
		t.Fatalf("mutation errors should name the stores too, got %q", err.Error())
	}
}
