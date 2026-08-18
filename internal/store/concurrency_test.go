package store

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sleeplesslord/saga/internal/saga"
)

// newTestStore builds a store over a throwaway global path with no local scope.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "sagas.jsonl")
	if err := os.WriteFile(path, nil, 0644); err != nil {
		t.Fatalf("seeding store: %v", err)
	}
	return &Store{globalPath: path}
}

func seed(t *testing.T, s *Store, title string) *saga.Saga {
	t.Helper()
	sg := saga.NewSaga(title)
	if err := s.Save(sg, ScopeGlobal); err != nil {
		t.Fatalf("saving saga: %v", err)
	}
	return sg
}

func TestUpdateBumpsRev(t *testing.T) {
	s := newTestStore(t)
	sg := seed(t, s, "first")

	fresh, err := s.GetByID(sg.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if fresh.Rev != 0 {
		t.Fatalf("a freshly created saga should start at rev 0, got %d", fresh.Rev)
	}

	fresh.AddHistory("log", "one")
	if err := s.Update(fresh); err != nil {
		t.Fatalf("update: %v", err)
	}
	if fresh.Rev != 1 {
		t.Fatalf("rev should advance to 1 after one update, got %d", fresh.Rev)
	}

	reloaded := &Store{globalPath: s.globalPath}
	got, err := reloaded.GetByID(sg.ID)
	if err != nil {
		t.Fatalf("get after update: %v", err)
	}
	if got.Rev != 1 {
		t.Fatalf("persisted rev should be 1, got %d", got.Rev)
	}
}

func TestUpdateRejectsStaleWrite(t *testing.T) {
	// The lost-update scenario: two readers hold the same revision, both apply a
	// different change. The second write must not silently discard the first.
	s := newTestStore(t)
	sg := seed(t, s, "contended")

	readerA, err := (&Store{globalPath: s.globalPath}).GetByID(sg.ID)
	if err != nil {
		t.Fatalf("reader A: %v", err)
	}
	readerB, err := (&Store{globalPath: s.globalPath}).GetByID(sg.ID)
	if err != nil {
		t.Fatalf("reader B: %v", err)
	}

	readerA.AddHistory("log", "from A")
	if err := (&Store{globalPath: s.globalPath}).Update(readerA); err != nil {
		t.Fatalf("first write should succeed: %v", err)
	}

	readerB.SetStatus(saga.StatusDone)
	err = (&Store{globalPath: s.globalPath}).Update(readerB)
	var conflict *ErrConflict
	if !errors.As(err, &conflict) {
		t.Fatalf("second write built on a stale read should conflict, got %v", err)
	}
	if conflict.DiskRev != 1 || conflict.WriteRev != 0 {
		t.Fatalf("conflict should report disk rev 1 vs write rev 0, got %+v", conflict)
	}

	// A's change survived.
	got, err := (&Store{globalPath: s.globalPath}).GetByID(sg.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status == saga.StatusDone {
		t.Fatal("the rejected write must not have been applied")
	}
	if len(got.History) != 2 || got.History[1].Note != "from A" {
		t.Fatalf("A's history entry should be intact, got %+v", got.History)
	}
}

func TestMutateSeesConcurrentChange(t *testing.T) {
	// Mutate re-reads under the lock, so it composes with a change another
	// process committed after the caller last looked.
	s := newTestStore(t)
	sg := seed(t, s, "sequenced")

	statusWriter, err := (&Store{globalPath: s.globalPath}).GetByID(sg.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	statusWriter.SetStatus(saga.StatusDone)
	if err := (&Store{globalPath: s.globalPath}).Update(statusWriter); err != nil {
		t.Fatalf("status write: %v", err)
	}

	if _, err := (&Store{globalPath: s.globalPath}).Mutate(sg.ID, func(fresh *saga.Saga) error {
		fresh.AddHistory("log", "note")
		return nil
	}); err != nil {
		t.Fatalf("mutate: %v", err)
	}

	got, err := (&Store{globalPath: s.globalPath}).GetByID(sg.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != saga.StatusDone {
		t.Fatalf("the status change must survive the log append, got %q", got.Status)
	}
	var found bool
	for _, h := range got.History {
		if h.Note == "note" {
			found = true
		}
	}
	if !found {
		t.Fatalf("the log entry must be present, got %+v", got.History)
	}
}

func TestMutateNotFoundIsSentinel(t *testing.T) {
	s := newTestStore(t)
	_, err := s.Mutate("nope00", func(*saga.Saga) error { return nil })
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestMutatePropagatesCallbackError(t *testing.T) {
	s := newTestStore(t)
	sg := seed(t, s, "callback")
	sentinel := errors.New("boom")

	if _, err := s.Mutate(sg.ID, func(*saga.Saga) error { return sentinel }); !errors.Is(err, sentinel) {
		t.Fatalf("callback error should surface, got %v", err)
	}

	got, err := (&Store{globalPath: s.globalPath}).GetByID(sg.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Rev != 0 {
		t.Fatalf("a failed callback must not persist a rev bump, got %d", got.Rev)
	}
}

func TestConcurrentMutatesAllLand(t *testing.T) {
	// Every appended entry must be present: this is the property the file lock
	// buys. Without it, whole-file rewrites clobber each other.
	s := newTestStore(t)
	sg := seed(t, s, "parallel")

	const writers = 12
	var wg sync.WaitGroup
	errs := make(chan error, writers)
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			// A separate Store per goroutine stands in for a separate process:
			// no shared in-process mutex, no shared index.
			st := &Store{globalPath: s.globalPath}
			if _, err := st.Mutate(sg.ID, func(fresh *saga.Saga) error {
				fresh.AddHistory("log", "entry")
				return nil
			}); err != nil {
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent mutate failed: %v", err)
	}

	got, err := (&Store{globalPath: s.globalPath}).GetByID(sg.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	// 1 "created" entry plus one per writer.
	if len(got.History) != writers+1 {
		t.Fatalf("expected %d history entries, got %d — writes were lost", writers+1, len(got.History))
	}
	if got.Rev != writers {
		t.Fatalf("expected rev %d, got %d", writers, got.Rev)
	}
}

// newTwoScopeStore builds a store with both a local and a global path, which is
// the layout every real invocation inside a project directory uses. The
// scope-loop in Update/Mutate only has more than one iteration here.
func newTwoScopeStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	global := filepath.Join(dir, "global", "sagas.jsonl")
	local := filepath.Join(dir, "local", ".saga", "sagas.jsonl")
	for _, p := range []string{global, local} {
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(p, nil, 0644); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	return &Store{globalPath: global, localPath: local}
}

func TestMutateFindsGlobalRecordWhenLocalScopeExists(t *testing.T) {
	s := newTwoScopeStore(t)
	sg := saga.NewSaga("lives in global")
	if err := s.Save(sg, ScopeGlobal); err != nil {
		t.Fatalf("save: %v", err)
	}

	if _, err := (&Store{globalPath: s.globalPath, localPath: s.localPath}).
		Mutate(sg.ID, func(fresh *saga.Saga) error {
			fresh.AddHistory("log", "found in global")
			return nil
		}); err != nil {
		t.Fatalf("mutate should search past the local scope: %v", err)
	}

	got, err := (&Store{globalPath: s.globalPath}).GetByID(sg.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(got.History) != 2 {
		t.Fatalf("expected the entry to land, got history %+v", got.History)
	}
}

func TestUpdateFindsGlobalRecordWhenLocalScopeExists(t *testing.T) {
	s := newTwoScopeStore(t)
	sg := saga.NewSaga("lives in global")
	if err := s.Save(sg, ScopeGlobal); err != nil {
		t.Fatalf("save: %v", err)
	}

	fresh, err := (&Store{globalPath: s.globalPath, localPath: s.localPath}).GetByID(sg.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	fresh.SetStatus(saga.StatusDone)
	if err := (&Store{globalPath: s.globalPath, localPath: s.localPath}).Update(fresh); err != nil {
		t.Fatalf("update should search past the local scope: %v", err)
	}

	got, err := (&Store{globalPath: s.globalPath}).GetByID(sg.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != saga.StatusDone {
		t.Fatalf("status not persisted, got %q", got.Status)
	}
}

// A validation error from the callback is the answer for the scope that holds the
// record. Carrying on to the next scope would let a rejected change be applied to
// a second copy of the same ID, and would let that scope's own failure (a lock
// timeout, say) replace the real message.
func TestMutateCallbackErrorDoesNotFallThroughToNextScope(t *testing.T) {
	s := newTwoScopeStore(t)
	sg := saga.NewSaga("lives in local")
	if err := s.Save(sg, ScopeLocal); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Hold the global lock so that reaching the global scope is unmistakable: it
	// would block until the lock timeout and report that instead.
	originalTimeout := lockTimeout
	lockTimeout = 300 * time.Millisecond
	defer func() { lockTimeout = originalTimeout }()

	held := make(chan struct{})
	release := make(chan struct{})
	go func() {
		_ = withStoreLock(s.globalPath, func(bool) error {
			close(held)
			<-release
			return nil
		})
	}()
	<-held
	defer close(release)

	sentinel := errors.New("already has that label")
	start := time.Now()
	_, err := (&Store{globalPath: s.globalPath, localPath: s.localPath}).
		Mutate(sg.ID, func(*saga.Saga) error { return sentinel })
	elapsed := time.Since(start)

	if !errors.Is(err, sentinel) {
		t.Fatalf("the callback's error must be returned verbatim, got %v", err)
	}
	if elapsed >= lockTimeout {
		t.Fatalf("took %s: the global scope was attempted after local already answered", elapsed)
	}
}

func TestUpdateConflictDoesNotFallThroughToNextScope(t *testing.T) {
	s := newTwoScopeStore(t)
	sg := saga.NewSaga("lives in local")
	if err := s.Save(sg, ScopeLocal); err != nil {
		t.Fatalf("save: %v", err)
	}

	stale, err := (&Store{globalPath: s.globalPath, localPath: s.localPath}).GetByID(sg.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	// Advance the stored revision so `stale` is now behind.
	if _, err := (&Store{globalPath: s.globalPath, localPath: s.localPath}).
		Mutate(sg.ID, func(fresh *saga.Saga) error {
			fresh.AddHistory("log", "someone else")
			return nil
		}); err != nil {
		t.Fatalf("mutate: %v", err)
	}

	stale.SetStatus(saga.StatusDone)
	err = (&Store{globalPath: s.globalPath, localPath: s.localPath}).Update(stale)
	var conflict *ErrConflict
	if !errors.As(err, &conflict) {
		t.Fatalf("expected ErrConflict from the owning scope, got %v", err)
	}
}

func TestMutateNotFoundAcrossBothScopes(t *testing.T) {
	s := newTwoScopeStore(t)
	if _, err := s.Mutate("nope00", func(*saga.Saga) error { return nil }); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestWithStoreLockTimesOutRatherThanHanging(t *testing.T) {
	// A stuck holder must surface as an error. A blocking flock would just trade
	// lost updates for a wedged command.
	dir := t.TempDir()
	path := filepath.Join(dir, "sagas.jsonl")

	originalTimeout := lockTimeout
	lockTimeout = 300 * time.Millisecond
	defer func() { lockTimeout = originalTimeout }()

	held := make(chan struct{})
	release := make(chan struct{})
	go func() {
		_ = withStoreLock(path, func(locked bool) error {
			if !locked {
				t.Error("the first holder should have acquired the lock")
			}
			close(held)
			<-release
			return nil
		})
	}()
	<-held
	defer close(release)

	start := time.Now()
	err := withStoreLock(path, func(bool) error {
		t.Error("the second holder must not enter while the first holds the lock")
		return nil
	})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected a timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("error should name the timeout, got %v", err)
	}
	if elapsed < lockTimeout {
		t.Fatalf("returned after %s, before the %s timeout", elapsed, lockTimeout)
	}
}

func TestConcurrentSavesAllLand(t *testing.T) {
	s := newTestStore(t)

	const writers = 10
	var wg sync.WaitGroup
	errs := make(chan error, writers)
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			st := &Store{globalPath: s.globalPath}
			if err := st.Save(saga.NewSaga("concurrent"), ScopeGlobal); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent save failed: %v", err)
	}

	all, err := (&Store{globalPath: s.globalPath}).LoadAll(ScopeGlobal)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(all) != writers {
		t.Fatalf("expected %d sagas, got %d — appends interleaved or were lost", writers, len(all))
	}
}
