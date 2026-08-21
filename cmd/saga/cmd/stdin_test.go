package cmd

import (
	"errors"
	"os"
	"syscall"
	"testing"
	"time"
)

// withStdin swaps os.Stdin for the duration of the test.
func withStdin(t *testing.T, f *os.File) {
	t.Helper()
	original := os.Stdin
	os.Stdin = f
	t.Cleanup(func() { os.Stdin = original })
}

// withShortStdinWaits shrinks both stdin budgets so tests that exercise them stay
// fast.
func withShortStdinWaits(t *testing.T, producer, idle time.Duration) {
	t.Helper()
	origProducer, origIdle := stdinProducerWait, stdinIdleProbe
	stdinProducerWait, stdinIdleProbe = producer, idle
	t.Cleanup(func() { stdinProducerWait, stdinIdleProbe = origProducer, origIdle })
}

// socketPair returns a connected pair of unix sockets as *os.File, mirroring the
// stdin an agent harness attaches. The peer stays open until closed, so the near
// end sees neither data nor EOF unless the test acts.
func socketPair(t *testing.T) (near, peer *os.File) {
	t.Helper()
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		t.Skipf("socketpair unavailable on this platform: %v", err)
	}
	near = os.NewFile(uintptr(fds[0]), "stdin-socket")
	peer = os.NewFile(uintptr(fds[1]), "stdin-socket-peer")
	t.Cleanup(func() {
		near.Close()
		peer.Close()
	})
	return near, peer
}

// The classification is asserted directly. Going through readStdinIfReady cannot
// distinguish the branches for /dev/null — it reads as empty either way — so a
// guard that never fires would pass unnoticed, which is what happened.
func TestClassifyStdin(t *testing.T) {
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer devNull.Close()
	devNullInfo, err := devNull.Stat()
	if err != nil {
		t.Fatalf("stat: %v", err)
	}

	path := t.TempDir() + "/notes.md"
	if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	regular, err := os.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer regular.Close()
	regularInfo, err := regular.Stat()
	if err != nil {
		t.Fatalf("stat: %v", err)
	}

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	defer r.Close()
	defer w.Close()
	pipeInfo, err := r.Stat()
	if err != nil {
		t.Fatalf("stat: %v", err)
	}

	near, _ := socketPair(t)
	sockInfo, err := near.Stat()
	if err != nil {
		t.Fatalf("stat: %v", err)
	}

	cases := []struct {
		name string
		mode os.FileMode
		want stdinSource
	}{
		{"char device", devNullInfo.Mode(), stdinNone},
		{"regular file", regularInfo.Mode(), stdinWholeFile},
		{"pipe", pipeInfo.Mode(), stdinPipe},
		{"unix socket", sockInfo.Mode(), stdinSocket},
	}
	for _, tc := range cases {
		if got := classifyStdin(tc.mode); got != tc.want {
			t.Errorf("%s (mode=%v): got %v, want %v", tc.name, tc.mode, got, tc.want)
		}
	}
}

// Pins the reason classifyStdin must not use syscall.S_IFMT: masking a Go
// os.FileMode with it is always zero, because FileMode's type bits live above bit
// 19 while S_IFMT covers bits 12-15. A guard written that way never fires — which
// is how every stdin, including a tty, got treated as piped.
func TestFileModeCannotBeMaskedWithSyscallIFMT(t *testing.T) {
	f, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if fi.Mode()&os.ModeCharDevice == 0 {
		t.Fatal("os.DevNull should report ModeCharDevice")
	}
	if uint32(fi.Mode())&uint32(syscall.S_IFMT) != 0 {
		t.Skip("os.FileMode layout changed; the S_IFMT caveat may no longer apply")
	}
}

func TestReadStdinIfReadyFromPipe(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	defer r.Close()

	if _, err := w.WriteString("line one\nline two\n"); err != nil {
		t.Fatalf("write: %v", err)
	}
	w.Close()

	withStdin(t, r)
	got, err := readStdinIfReady()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got != "line one\nline two" {
		t.Fatalf("piped heredoc content should come through whole, got %q", got)
	}
}

// A pipe has a writer by construction, so a producer that takes a while to say
// anything must still be heard. Dropping it would silently discard the input of
// anything slower than a probe window.
func TestReadStdinIfReadyWaitsForSlowFirstByte(t *testing.T) {
	withShortStdinWaits(t, 2*time.Second, 50*time.Millisecond)
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	defer r.Close()

	go func() {
		time.Sleep(400 * time.Millisecond)
		w.WriteString("late but real\n")
		w.Close()
	}()

	withStdin(t, r)
	got, err := readStdinIfReady()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got != "late but real" {
		t.Fatalf("a slow producer's input must not be dropped, got %q", got)
	}
}

func TestReadStdinIfReadyCollectsBurstyInput(t *testing.T) {
	withShortStdinWaits(t, 2*time.Second, 50*time.Millisecond)
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	defer r.Close()

	go func() {
		w.WriteString("part one\n")
		time.Sleep(300 * time.Millisecond)
		w.WriteString("part two\n")
		w.Close()
	}()

	withStdin(t, r)
	got, err := readStdinIfReady()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got != "part one\npart two" {
		t.Fatalf("bursty input must arrive whole, got %q", got)
	}
}

// A pipe whose writer never produces is an anomaly, not "no input": reporting it
// keeps a caller from treating the silence as success.
func TestReadStdinIfReadyReportsSilentPipe(t *testing.T) {
	withShortStdinWaits(t, 150*time.Millisecond, 50*time.Millisecond)
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	defer r.Close()
	defer w.Close() // held open: a writer exists but says nothing

	withStdin(t, r)
	got, err := readStdinIfReady()
	if !errors.Is(err, errStdinStalled) {
		t.Fatalf("expected a stalled-stdin error, got value %q err %v", got, err)
	}
}

func TestReadStdinIfReadyFromRegularFile(t *testing.T) {
	path := t.TempDir() + "/notes.md"
	if err := os.WriteFile(path, []byte("from a file\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()

	withStdin(t, f)
	got, err := readStdinIfReady()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got != "from a file" {
		t.Fatalf("got %q", got)
	}
}

// A character device must short-circuit before any probing: probing would put
// O_NONBLOCK on a descriptor the shell shares with us.
func TestReadStdinIfReadyLeavesCharDeviceFlagsAlone(t *testing.T) {
	f, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()

	fd := int(f.Fd())
	before, err := fcntlGetFlags(fd)
	if err != nil {
		t.Fatalf("F_GETFL: %v", err)
	}

	withStdin(t, f)
	got, err := readStdinIfReady()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got != "" {
		t.Fatalf("/dev/null should read as no input, got %q", got)
	}

	after, err := fcntlGetFlags(fd)
	if err != nil {
		t.Fatalf("F_GETFL: %v", err)
	}
	if before != after {
		t.Fatalf("a character device must not be probed: flags changed %#o -> %#o", before, after)
	}
}

// The hang this exists to prevent: an agent harness attaches a socket it keeps
// open for the whole session. No data, no EOF — and, unlike a pipe, no promise
// that a producer exists, so this reads as "no input" rather than an error.
func TestReadStdinIfReadyDoesNotBlockOnIdleSocket(t *testing.T) {
	near, _ := socketPair(t)
	withStdin(t, near)

	done := make(chan struct{})
	var got string
	var readErr error
	go func() {
		got, readErr = readStdinIfReady()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("readStdinIfReady blocked on an idle socket")
	}

	if readErr != nil {
		t.Fatalf("read: %v", readErr)
	}
	if got != "" {
		t.Fatalf("an idle socket should read as no input, got %q", got)
	}
}

func TestReadStdinIfReadyRestoresNonblockFlag(t *testing.T) {
	near, _ := socketPair(t)
	fd := int(near.Fd())
	before, err := fcntlGetFlags(fd)
	if err != nil {
		t.Fatalf("F_GETFL: %v", err)
	}
	if before&syscall.O_NONBLOCK != 0 {
		t.Skip("socket already non-blocking; nothing to restore")
	}

	withStdin(t, near)
	if _, err := readStdinIfReady(); err != nil {
		t.Fatalf("read: %v", err)
	}

	after, err := fcntlGetFlags(fd)
	if err != nil {
		t.Fatalf("F_GETFL: %v", err)
	}
	if after&syscall.O_NONBLOCK != 0 {
		t.Fatal("O_NONBLOCK left set — it lives on the shared open file description")
	}
	if before != after {
		t.Fatalf("stdin flags not restored: %#o -> %#o", before, after)
	}
}

func TestReadStdinIfReadyReadsSocketPayloadThenEOF(t *testing.T) {
	near, peer := socketPair(t)
	if _, err := peer.Write([]byte("socket payload\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	peer.Close()

	withStdin(t, near)
	got, err := readStdinIfReady()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got != "socket payload" {
		t.Fatalf("got %q", got)
	}
}

// Input that starts and then stops without end-of-input is an error: storing the
// bytes read so far would record a truncated message as if it were whole.
func TestReadStdinIfReadyRejectsStalledProducer(t *testing.T) {
	withShortStdinWaits(t, 150*time.Millisecond, 150*time.Millisecond)
	near, peer := socketPair(t)
	if _, err := peer.Write([]byte("first half")); err != nil {
		t.Fatalf("write: %v", err)
	}

	withStdin(t, near)
	got, err := readStdinIfReady()
	if !errors.Is(err, errStdinStalled) {
		t.Fatalf("a stalled producer must be an error, got value %q err %v", got, err)
	}
	if got != "" {
		t.Fatalf("no partial content may be returned, got %q", got)
	}
}

func TestReadStdinReadsToCompletion(t *testing.T) {
	// The `-` path: the caller named stdin, so blocking until EOF is correct.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	defer r.Close()

	go func() {
		w.WriteString("explicit stdin\n")
		time.Sleep(300 * time.Millisecond)
		w.WriteString("still going\n")
		w.Close()
	}()

	withStdin(t, r)
	got, err := readStdin()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got != "explicit stdin\nstill going" {
		t.Fatalf("got %q", got)
	}
}
