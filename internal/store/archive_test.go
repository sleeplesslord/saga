package store

import (
	"errors"
	"testing"
	"time"

	"github.com/sleeplesslord/saga/internal/saga"
)

// archiveOldSaga seeds a terminal saga that predates the archive cutoff.
func archiveOldSaga(t *testing.T, s *Store, title, plan string) *saga.Saga {
	t.Helper()
	sg := seed(t, s, title)
	sg.Status = saga.StatusDone
	sg.Plan = plan
	sg.UpdatedAt = time.Now().AddDate(0, 0, -60)
	if err := s.Update(sg); err != nil {
		t.Fatalf("marking %q done: %v", title, err)
	}
	return sg
}

func TestArchivedSagasStayReadable(t *testing.T) {
	s := newTestStore(t)
	old := archiveOldSaga(t, s, "long done", "step one")
	recent := seed(t, s, "still active")

	count, err := s.Archive(ScopeGlobal, time.Now().AddDate(0, 0, -30), false)
	if err != nil {
		t.Fatalf("archive: %v", err)
	}
	if count != 1 {
		t.Fatalf("only the stale terminal saga should archive, got %d", count)
	}

	// A fresh store proves the state is on disk, not in a warm index.
	reloaded := &Store{globalPath: s.globalPath}

	if _, err := reloaded.GetByID(old.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("archived saga should be gone from the active store, got %v", err)
	}
	if _, err := reloaded.GetByID(recent.ID); err != nil {
		t.Fatalf("active saga should survive archiving: %v", err)
	}

	archived, err := reloaded.LoadArchived(ScopeGlobal)
	if err != nil {
		t.Fatalf("load archived: %v", err)
	}
	if len(archived) != 1 {
		t.Fatalf("archive should hold exactly the archived saga, got %d", len(archived))
	}
	if archived[0].ID != old.ID {
		t.Fatalf("archive holds %s, want %s", archived[0].ID, old.ID)
	}
	// Archiving moves plan files to archive/plans/, so reading the archive
	// has to look there rather than at the active plans/ directory.
	if archived[0].Plan != "step one" {
		t.Fatalf("archived plan should be recovered, got %q", archived[0].Plan)
	}
}

func TestGetArchivedByID(t *testing.T) {
	s := newTestStore(t)
	old := archiveOldSaga(t, s, "long done", "")
	recent := seed(t, s, "still active")

	if _, err := s.Archive(ScopeGlobal, time.Now().AddDate(0, 0, -30), false); err != nil {
		t.Fatalf("archive: %v", err)
	}

	reloaded := &Store{globalPath: s.globalPath}

	got, err := reloaded.GetArchivedByID(old.ID)
	if err != nil {
		t.Fatalf("archived saga should be findable by ID: %v", err)
	}
	if got.Title != "long done" {
		t.Fatalf("got %q, want %q", got.Title, "long done")
	}

	if _, err := reloaded.GetArchivedByID(recent.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("an active saga is not in the archive, got %v", err)
	}
}

func TestLoadArchivedWithoutArchiveFile(t *testing.T) {
	s := newTestStore(t)
	seed(t, s, "still active")

	archived, err := s.LoadArchived(ScopeGlobal)
	if err != nil {
		t.Fatalf("a store that never archived should not error: %v", err)
	}
	if len(archived) != 0 {
		t.Fatalf("expected an empty archive, got %d", len(archived))
	}
}

func TestArchiveAppendsAcrossRuns(t *testing.T) {
	s := newTestStore(t)
	first := archiveOldSaga(t, s, "first done", "")
	if _, err := s.Archive(ScopeGlobal, time.Now().AddDate(0, 0, -30), false); err != nil {
		t.Fatalf("first archive: %v", err)
	}

	second := archiveOldSaga(t, s, "second done", "")
	if _, err := s.Archive(ScopeGlobal, time.Now().AddDate(0, 0, -30), false); err != nil {
		t.Fatalf("second archive: %v", err)
	}

	reloaded := &Store{globalPath: s.globalPath}
	archived, err := reloaded.LoadArchived(ScopeGlobal)
	if err != nil {
		t.Fatalf("load archived: %v", err)
	}
	if len(archived) != 2 {
		t.Fatalf("a second archive run must not drop the first batch, got %d", len(archived))
	}

	ids := map[string]bool{}
	for _, sg := range archived {
		ids[sg.ID] = true
	}
	if !ids[first.ID] || !ids[second.ID] {
		t.Fatalf("archive should hold both %s and %s, got %v", first.ID, second.ID, ids)
	}
}
