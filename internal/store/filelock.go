package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// lockTimeout bounds how long we wait for another process to finish its
// read-modify-write cycle. Long enough to absorb a normal write, short enough
// that a wedged holder surfaces as an error instead of a hang — a blocking
// flock would just trade lost updates for a stuck command.
//
// Variable so tests can shorten it; nothing else reassigns it.
var lockTimeout = 5 * time.Second

// lockPollInterval is how often we retry a non-blocking flock while waiting.
const lockPollInterval = 20 * time.Millisecond

// withStoreLock runs fn while holding an exclusive advisory lock on the store at
// path.
//
// Every mutation is a read-modify-write of the whole JSONL file, and the Store's
// sync.RWMutex only serializes goroutines inside one process. Two concurrent
// invocations would each load the file, apply their own change, and write the
// whole thing back — the second write silently discarding the first. Holding
// this lock across the entire load→save cycle is what makes the cycle atomic
// between processes.
//
// fn receives whether the lock was actually acquired. On a filesystem with no
// flock support it runs unlocked, and callers that can detect interference (see
// Store.Mutate's revision check) should do so in that case.
//
// The lock lives beside the store as `<path>.lock`; it is never deleted, so the
// inode stays stable for everyone contending on it.
func withStoreLock(path string, fn func(locked bool) error) error {
	if path == "" {
		return fn(false)
	}

	lockPath := path + ".lock"
	if dir := filepath.Dir(lockPath); dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating store directory: %w", err)
		}
	}

	// 0666 (less umask) so a store shared between users stays writable by all of
	// them. The lock carries no data; restricting it only turns a shared
	// checkout into a permanent failure for whoever didn't create it.
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		// The lock is not the data. Losing it costs cross-process safety, which
		// is worth a warning, not a refusal to work.
		warnLockUnavailable(lockPath, err)
		return fn(false)
	}
	defer file.Close()

	deadline := time.Now().Add(lockTimeout)
	for {
		err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			break
		}
		// Some filesystems (a few network mounts) have no flock at all.
		if errors.Is(err, syscall.ENOTSUP) || errors.Is(err, syscall.ENOLCK) ||
			errors.Is(err, syscall.EOPNOTSUPP) {
			warnLockUnavailable(lockPath, err)
			return fn(false)
		}
		if !errors.Is(err, syscall.EWOULDBLOCK) {
			return fmt.Errorf("locking store %s: %w", lockPath, err)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf(
				"timed out after %s waiting for the store lock (%s); another saga process may be stuck",
				lockTimeout, lockPath,
			)
		}
		time.Sleep(lockPollInterval)
	}
	defer syscall.Flock(int(file.Fd()), syscall.LOCK_UN)

	return fn(true)
}

var lockUnavailableOnce sync.Once

// warnLockUnavailable reports a missing lock a single time per process, so a
// mutation loop doesn't repeat it per saga.
func warnLockUnavailable(lockPath string, cause error) {
	lockUnavailableOnce.Do(func() {
		fmt.Fprintf(os.Stderr,
			"warning: cannot lock %s (%v); concurrent saga processes are detected by revision instead\n",
			lockPath, cause)
	})
}
