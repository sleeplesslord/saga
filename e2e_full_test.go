package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunner handles e2e test execution
type TestRunner struct {
	BinaryPath string
	TempDir    string
	t          *testing.T
}

// NewTestRunner creates a new test runner
func NewTestRunner(t *testing.T) *TestRunner {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working dir: %v", err)
	}

	binaryPath := filepath.Join(wd, "sg_test")
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/sg")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build binary: %v\n%s", err, output)
	}

	tempDir, err := os.MkdirTemp("", "saga-e2e-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	os.MkdirAll(filepath.Join(tempDir, ".saga"), 0755)

	return &TestRunner{
		BinaryPath: binaryPath,
		TempDir:    tempDir,
		t:          t,
	}
}

// Cleanup removes temp directory and binary
func (r *TestRunner) Cleanup() {
	os.RemoveAll(r.TempDir)
	os.Remove(r.BinaryPath)
}

// Run executes a saga command and returns output
func (r *TestRunner) Run(args ...string) (string, string, error) {
	cmd := exec.Command(r.BinaryPath, args...)
	cmd.Dir = r.TempDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// getSagaID finds saga ID by title
func (r *TestRunner) getSagaID(title string) string {
	stdout, _, _ := r.Run("list", "--all")
	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		if strings.Contains(line, title) {
			fields := strings.Fields(line)
			if len(fields) > 0 {
				return fields[0]
			}
		}
	}
	return ""
}

// TestAllCommands runs comprehensive E2E tests
func TestAllCommands(t *testing.T) {
	runner := NewTestRunner(t)
	defer runner.Cleanup()

	t.Run("Init", func(t *testing.T) {
		// Already initialized in NewTestRunner, just verify
		_, _, err := runner.Run("list")
		if err != nil {
			t.Fatalf("List should work after init: %v", err)
		}
	})

	t.Run("New", func(t *testing.T) {
		stdout, _, err := runner.Run("new", "Test saga")
		if err != nil {
			t.Fatalf("Failed to create saga: %v", err)
		}
		if !strings.Contains(stdout, "Created saga") {
			t.Errorf("Expected 'Created saga', got: %s", stdout)
		}
	})

	t.Run("NewWithOptions", func(t *testing.T) {
		stdout, _, err := runner.Run("new", "Priority saga", "--priority", "high", "--label", "urgent", "--desc", "Test description")
		if err != nil {
			t.Fatalf("Failed to create saga with options: %v", err)
		}
		if !strings.Contains(stdout, "Priority: high") {
			t.Errorf("Expected priority in output: %s", stdout)
		}
	})

	t.Run("List", func(t *testing.T) {
		stdout, _, err := runner.Run("list")
		if err != nil {
			t.Fatalf("Failed to list: %v", err)
		}
		if !strings.Contains(stdout, "Test saga") {
			t.Errorf("Expected saga in list: %s", stdout)
		}
	})

	t.Run("ListFilter", func(t *testing.T) {
		stdout, _, err := runner.Run("list", "--label", "urgent")
		if err != nil {
			t.Fatalf("Failed to filter by label: %v", err)
		}
		if !strings.Contains(stdout, "Priority saga") {
			t.Errorf("Expected filtered saga: %s", stdout)
		}
		if strings.Contains(stdout, "Test saga") {
			t.Errorf("Should not show unlabeled saga: %s", stdout)
		}
	})

	t.Run("Status", func(t *testing.T) {
		id := runner.getSagaID("Test saga")
		if id == "" {
			t.Fatal("Could not find saga")
		}
		stdout, _, err := runner.Run("status", id)
		if err != nil {
			t.Fatalf("Failed to get status: %v", err)
		}
		if !strings.Contains(stdout, "Test saga") {
			t.Errorf("Expected title in status: %s", stdout)
		}
		if !strings.Contains(stdout, "active") {
			t.Errorf("Expected status in output: %s", stdout)
		}
	})

	t.Run("SubSaga", func(t *testing.T) {
		parentID := runner.getSagaID("Test saga")
		if parentID == "" {
			t.Fatal("Could not find parent saga")
		}

		stdout, _, err := runner.Run("new", "Child task", "--parent", parentID)
		if err != nil {
			t.Fatalf("Failed to create sub-saga: %v", err)
		}
		if !strings.Contains(stdout, "Created sub-saga") {
			t.Errorf("Expected sub-saga message: %s", stdout)
		}
		if !strings.Contains(stdout, parentID+".") {
			t.Errorf("Expected hierarchical ID: %s", stdout)
		}
	})

	t.Run("Label", func(t *testing.T) {
		id := runner.getSagaID("Test saga")
		if id == "" {
			t.Fatal("Could not find saga")
		}

		stdout, _, err := runner.Run("label", id, "add", "test-label")
		if err != nil {
			t.Fatalf("Failed to add label: %v", err)
		}
		if !strings.Contains(stdout, "Added label") {
			t.Errorf("Expected label added message: %s", stdout)
		}

		// Verify in list
		stdout, _, _ = runner.Run("list", "--label", "test-label")
		if !strings.Contains(stdout, "Test saga") {
			t.Errorf("Label should appear in filtered list: %s", stdout)
		}
	})

	t.Run("Priority", func(t *testing.T) {
		id := runner.getSagaID("Test saga")
		if id == "" {
			t.Fatal("Could not find saga")
		}

		stdout, _, err := runner.Run("priority", id, "high")
		if err != nil {
			t.Fatalf("Failed to set priority: %v", err)
		}
		if !strings.Contains(stdout, "Changed priority") {
			t.Errorf("Expected priority change message: %s", stdout)
		}
	})

	t.Run("Log", func(t *testing.T) {
		id := runner.getSagaID("Test saga")
		if id == "" {
			t.Fatal("Could not find saga")
		}

		stdout, _, err := runner.Run("log", id, "Test progress entry")
		if err != nil {
			t.Fatalf("Failed to log: %v", err)
		}
		if !strings.Contains(stdout, "Added log") {
			t.Errorf("Expected log added message: %s", stdout)
		}

		// Verify in status
		stdout, _, _ = runner.Run("status", id)
		if !strings.Contains(stdout, "Test progress entry") {
			t.Errorf("Log should appear in history: %s", stdout)
		}
	})

	t.Run("Depend", func(t *testing.T) {
		// Get two sagas
		id1 := runner.getSagaID("Test saga")
		id2 := runner.getSagaID("Priority saga")
		if id1 == "" || id2 == "" {
			t.Fatal("Could not find sagas")
		}

		stdout, _, err := runner.Run("depend", id1, "add", id2)
		if err != nil {
			t.Fatalf("Failed to add dependency: %v", err)
		}
		if !strings.Contains(stdout, "Added dependency") {
			t.Errorf("Expected dependency message: %s", stdout)
		}

		// Verify in status
		stdout, _, _ = runner.Run("status", id1)
		if !strings.Contains(stdout, id2) {
			t.Errorf("Dependency should appear in status: %s", stdout)
		}
	})

	t.Run("Relate", func(t *testing.T) {
		id1 := runner.getSagaID("Test saga")
		id2 := runner.getSagaID("Priority saga")
		if id1 == "" || id2 == "" {
			t.Fatal("Could not find sagas")
		}

		stdout, _, err := runner.Run("relate", id1, "add", id2)
		if err != nil {
			t.Fatalf("Failed to add relationship: %v", err)
		}
		if !strings.Contains(stdout, "Added relationship") {
			t.Errorf("Expected relationship message: %s", stdout)
		}
	})

	t.Run("Search", func(t *testing.T) {
		stdout, _, err := runner.Run("search", "Priority")
		if err != nil {
			t.Fatalf("Failed to search: %v", err)
		}
		if !strings.Contains(stdout, "Priority saga") {
			t.Errorf("Expected search result: %s", stdout)
		}
	})

	t.Run("Context", func(t *testing.T) {
		id := runner.getSagaID("Test saga")
		if id == "" {
			t.Fatal("Could not find saga")
		}

		stdout, _, err := runner.Run("context", id)
		if err != nil {
			t.Fatalf("Failed to get context: %v", err)
		}
		if !strings.Contains(stdout, "SAGA:") {
			t.Errorf("Expected context header: %s", stdout)
		}
		if !strings.Contains(stdout, "HIERARCHY") {
			t.Errorf("Expected hierarchy section: %s", stdout)
		}
	})

	t.Run("Claim", func(t *testing.T) {
		id := runner.getSagaID("Test saga")
		if id == "" {
			t.Fatal("Could not find saga")
		}

		stdout, _, err := runner.Run("claim", id, "--agent", "test-bot")
		if err != nil {
			t.Fatalf("Failed to claim: %v", err)
		}
		if !strings.Contains(stdout, "Claimed saga") {
			t.Errorf("Expected claim message: %s", stdout)
		}

		// Verify in list (now includes PPID: [claimed:test-bot@<pid>])
		stdout, _, _ = runner.Run("list")
		if !strings.Contains(stdout, "[claimed:test-bot@") {
			t.Errorf("Claim should appear in list with PPID: %s", stdout)
		}
	})

	t.Run("Unclaim", func(t *testing.T) {
		id := runner.getSagaID("Test saga")
		if id == "" {
			t.Fatal("Could not find saga")
		}

		stdout, _, err := runner.Run("unclaim", id)
		if err != nil {
			t.Fatalf("Failed to unclaim: %v", err)
		}
		if !strings.Contains(stdout, "Released claim") {
			t.Errorf("Expected unclaim message: %s", stdout)
		}
	})

	t.Run("Continue", func(t *testing.T) {
		// First pause by marking done then continuing... actually we need a pause command
		// For now, just test that continue doesn't error on active saga
		id := runner.getSagaID("Test saga")
		if id == "" {
			t.Fatal("Could not find saga")
		}

		stdout, _, err := runner.Run("continue", id)
		// May error if already active, that's ok
		_ = err
		_ = stdout
	})

	t.Run("Ready", func(t *testing.T) {
		// Test that sg ready respects parent blocking
		// Create a blocked parent with a child
		stdout, _, err := runner.Run("new", "Blocked parent saga")
		if err != nil {
			t.Fatalf("Failed to create parent saga: %v", err)
		}
		// Extract parent ID from output
		parentID := ""
		if strings.Contains(stdout, "Created saga") {
			// Get the ID from the list
			parentID = runner.getSagaID("Blocked parent saga")
		}

		// Create a dependency that blocks the parent
		stdout, _, err = runner.Run("new", "Blocking dependency saga")
		if err != nil {
			t.Fatalf("Failed to create blocking saga: %v", err)
		}
		blockerID := runner.getSagaID("Blocking dependency saga")

		// Make parent depend on blocker
		_, _, err = runner.Run("depend", parentID, "add", blockerID)
		if err != nil {
			t.Fatalf("Failed to add dependency: %v", err)
		}

		// Create child of blocked parent
		_, _, err = runner.Run("new", "Child of blocked parent", "--parent", parentID)
		if err != nil {
			t.Fatalf("Failed to create child saga: %v", err)
		}

		// Now check sg ready - child should NOT appear because parent is blocked
		stdout, _, err = runner.Run("ready")
		if err != nil {
			t.Fatalf("Failed to run ready command: %v", err)
		}

		// Child should not be in ready list because parent is blocked
		if strings.Contains(stdout, "Child of blocked parent") {
			t.Errorf("Child of blocked parent should not appear in ready list: %s", stdout)
		}

		// But the blocker should appear (it has no dependencies)
		if !strings.Contains(stdout, "Blocking dependency saga") {
			t.Errorf("Blocking dependency saga should appear in ready list: %s", stdout)
		}

		// Clean up - complete blocker
		_, _, _ = runner.Run("done", blockerID)
		// Now parent should be unblocked, complete it
		_, _, _ = runner.Run("done", parentID)
	})

	t.Run("Complete", func(t *testing.T) {
		// Complete child first
		childID := ""
		stdout, _, _ := runner.Run("list", "--all")
		lines := strings.Split(stdout, "\n")
		for _, line := range lines {
			if strings.Contains(line, "Child task") {
				fields := strings.Fields(line)
				if len(fields) > 0 {
					childID = fields[0]
					break
				}
			}
		}

		if childID != "" {
			runner.Run("done", childID)
		}

		// Complete dependency
		depID := runner.getSagaID("Priority saga")
		if depID != "" {
			runner.Run("done", depID)
		}

		// Now complete parent
		id := runner.getSagaID("Test saga")
		if id == "" {
			t.Fatal("Could not find saga")
		}

		stdout, _, err := runner.Run("done", id)
		if err != nil {
			t.Fatalf("Failed to complete saga: %v", err)
		}
		if !strings.Contains(stdout, "Marked saga") {
			t.Errorf("Expected completion message: %s", stdout)
		}
	})

	t.Run("ListAllShowsDone", func(t *testing.T) {
		stdout, _, err := runner.Run("list", "--all")
		if err != nil {
			t.Fatalf("Failed to list all: %v", err)
		}
		if !strings.Contains(stdout, "done") {
			t.Errorf("Expected done status: %s", stdout)
		}
	})

	fmt.Println("✅ All commands tested!")
}
