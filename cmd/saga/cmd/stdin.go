package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
	"time"
)

// stdinProducerWait is how long a stream with a real producer behind it may take
// to deliver its first byte. A shell pipe or heredoc always has a writer
// attached, so waiting is correct: the alternative is dropping the input of any
// producer slower than the window (`curl ... | saga log <id>`).
//
// Variable so tests can shorten it; nothing else reassigns it.
var stdinProducerWait = 5 * time.Second

// stdinIdleProbe is how long a stream that may have no producer at all gets. An
// agent harness attaches a socket it holds open for the whole session, with no
// data and no EOF, so this cannot be open-ended — but nothing has been consumed
// when it expires, so giving up costs no data.
var stdinIdleProbe = 250 * time.Millisecond

// stdinPoll is how often a stream with nothing available yet is re-checked.
const stdinPoll = 5 * time.Millisecond

// errStdinStalled reports that a stream with a producer behind it delivered
// nothing, or stopped part-way, rather than reaching end of input. Surfacing this
// is what keeps a half-read message from being stored as though it were whole.
var errStdinStalled = errors.New("stdin did not deliver a complete message")

// readStdinIfReady returns whatever is waiting on stdin, or "" when stdin is a
// terminal, is already at end of input, or is a stream with no producer.
//
// It never blocks indefinitely, and it never returns a partial read.
func readStdinIfReady() (string, error) {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return "", nil
	}

	switch classifyStdin(fi.Mode()) {
	case stdinNone:
		return "", nil
	case stdinWholeFile:
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", err
		}
		return strings.TrimRight(string(data), "\n"), nil
	case stdinPipe:
		// A writer is attached by construction, so nothing arriving is an
		// anomaly worth reporting rather than silently treating as "no input".
		return readStreamWithoutBlocking(int(os.Stdin.Fd()), stdinProducerWait, true)
	default:
		return readStreamWithoutBlocking(int(os.Stdin.Fd()), stdinIdleProbe, false)
	}
}

// readStdin reads stdin to completion because the caller asked for it explicitly
// (the `-` argument). Blocking is correct here: the user named stdin as the
// source, so waiting for it is what they asked for.
func readStdin() (string, error) {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(data), "\n"), nil
}

// stdinSource is how stdin should be consumed.
type stdinSource int

const (
	// stdinNone: a character device — an interactive terminal or /dev/null. There
	// is no piped input to collect, and probing it would put O_NONBLOCK on a
	// descriptor the shell shares.
	stdinNone stdinSource = iota
	// stdinWholeFile: a regular file (`saga log id < notes.md`). Known length,
	// cannot stall, so read it whole.
	stdinWholeFile
	// stdinPipe: a pipe or FIFO — how a shell delivers `cmd | saga ...` and
	// heredocs. A writer is attached, so end of input will arrive.
	stdinPipe
	// stdinSocket: a socket or anything else. This is the shape an agent harness
	// attaches, where there may be no producer and no end of input at all.
	stdinSocket
)

// classifyStdin decides how to consume a stdin with the given mode.
//
// Split out as a pure function because the branches are otherwise unobservable
// for /dev/null — it reads as empty either way — which is how a guard that never
// fired went unnoticed.
//
// The character-device test must use os.ModeCharDevice, not syscall.S_IFMT:
// FileInfo.Mode returns Go's portable os.FileMode, whose type bits live above bit
// 19, while S_IFMT covers bits 12-15. Masking a FileMode with S_IFMT is always
// zero — which is exactly how the previous check reported "input is piped" for
// every stdin, including a tty.
func classifyStdin(mode os.FileMode) stdinSource {
	switch {
	case mode&os.ModeCharDevice != 0:
		return stdinNone
	case mode.IsRegular():
		return stdinWholeFile
	case mode&os.ModeNamedPipe != 0:
		return stdinPipe
	default:
		return stdinSocket
	}
}

// readStreamWithoutBlocking drains a pipe, FIFO or socket with non-blocking
// reads, waiting up to firstByteWait for input to start and then up to
// stdinProducerWait for each subsequent chunk.
//
// When nothing arrives at all: requireInput decides whether that is an error (a
// pipe, which has a writer) or simply "no input" (a socket, which may not).
// Input that starts and then stops is always an error — a partially consumed
// message must never be stored as if it were whole.
//
// O_NONBLOCK lives on the open file description, which is shared with whoever
// handed us this descriptor, so the original flags are restored on every path.
func readStreamWithoutBlocking(fd int, firstByteWait time.Duration, requireInput bool) (string, error) {
	original, err := fcntlGetFlags(fd)
	if err != nil {
		// Can't probe safely; treat stdin as empty rather than risk a hang.
		return "", nil
	}
	if original&syscall.O_NONBLOCK == 0 {
		if err := fcntlSetFlags(fd, original|syscall.O_NONBLOCK); err != nil {
			return "", nil
		}
		defer func() { _ = fcntlSetFlags(fd, original) }()
	}

	var buf bytes.Buffer
	deadline := time.Now().Add(firstByteWait)
	chunk := make([]byte, 32*1024)

	for {
		n, readErr := syscall.Read(fd, chunk)
		if n > 0 {
			buf.Write(chunk[:n])
			// Input is flowing; allow time for the rest of it.
			deadline = time.Now().Add(stdinProducerWait)
			continue
		}
		if readErr == nil {
			// n == 0 with no error is end of input.
			return strings.TrimRight(buf.String(), "\n"), nil
		}
		if errors.Is(readErr, syscall.EINTR) {
			continue
		}
		if errors.Is(readErr, syscall.EAGAIN) || errors.Is(readErr, syscall.EWOULDBLOCK) {
			if time.Now().After(deadline) {
				if buf.Len() > 0 {
					return "", fmt.Errorf("%w: read %d bytes then stalled for %s",
						errStdinStalled, buf.Len(), stdinProducerWait)
				}
				if requireInput {
					return "", fmt.Errorf("%w: nothing arrived within %s",
						errStdinStalled, firstByteWait)
				}
				// Not an input source.
				return "", nil
			}
			time.Sleep(stdinPoll)
			continue
		}
		return "", readErr
	}
}

func fcntlGetFlags(fd int) (int, error) {
	flags, _, errno := syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), syscall.F_GETFL, 0)
	if errno != 0 {
		return 0, errno
	}
	return int(flags), nil
}

func fcntlSetFlags(fd, flags int) error {
	_, _, errno := syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), syscall.F_SETFL, uintptr(flags))
	if errno != 0 {
		return errno
	}
	return nil
}
