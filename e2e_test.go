package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestRunner handles e2e test execution
type TestRunner struct {
	BinaryPath string
	TempDir    string
	t          *testing.T
}

// NewTestRunner creates a new test runner
func NewTestRunner(t *testing.T) *TestRunner {
	// Get current working directory
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working dir: %v", err)
	}

	// Build binary with absolute path
	binaryPath := filepath.Join(wd, "sg_test")
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/sg")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build binary: %v\n%s", err, output)
	}

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "saga-e2e-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Initialize saga in temp dir
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

// TestEndToEnd runs the full e2e test suite
func TestEndToEnd(t *testing.T) {
	runner := NewTestRunner(t)
	defer runner.Cleanup()

	t.Run("CreateSaga", func(t *testing.T) {
		stdout, _, err := runner.Run("new", "Test saga")
		if err != nil {
			t.Fatalf("Failed to create saga: %v", err)
		}
		if !strings.Contains(stdout, "Created saga") {
			t.Errorf("Expected 'Created saga' in output, got: %s", stdout)
		}
	})

	t.Run("ListSagas", func(t *testing.T) {
		stdout, _, err := runner.Run("list")
		if err != nil {
			t.Fatalf("Failed to list sagas: %v", err)
		}
		if !strings.Contains(stdout, "Test saga") {
			t.Errorf("Expected saga in list, got: %s", stdout)
		}
	})

	t.Run("CreateSubSaga", func(t *testing.T) {
		// Get the saga ID first
		stdout, _, _ := runner.Run("list")
		lines := strings.Split(stdout, "\n")
		var parentID string
		for _, line := range lines {
			if strings.Contains(line, "Test saga") {
				fields := strings.Fields(line)
				if len(fields) > 0 {
					parentID = fields[0]
					break
				}
			}
		}
		if parentID == "" {
			t.Fatal("Could not find parent saga ID")
		}

		stdout, _, err := runner.Run("new", "Sub task", "--parent", parentID)
		if err != nil {
			t.Fatalf("Failed to create sub-saga: %v", err)
		}
		if !strings.Contains(stdout, "Created sub-saga") {
			t.Errorf("Expected 'Created sub-saga' in output, got: %s", stdout)
		}
		if !strings.Contains(stdout, parentID+".") {
			t.Errorf("Expected hierarchical ID, got: %s", stdout)
		}
	})

	t.Run("ClaimSaga", func(t *testing.T) {
		stdout, _, _ := runner.Run("list")
		lines := strings.Split(stdout, "\n")
		var sagaID string
		for _, line := range lines {
			if strings.Contains(line, "Test saga") {
				fields := strings.Fields(line)
				if len(fields) > 0 {
					sagaID = fields[0]
					break
				}
			}
		}

		stdout, _, err := runner.Run("claim", sagaID, "--agent", "test-agent")
		if err != nil {
			t.Fatalf("Failed to claim saga: %v", err)
		}
		if !strings.Contains(stdout, "Claimed saga") {
			t.Errorf("Expected 'Claimed saga' in output, got: %s", stdout)
		}
		if !strings.Contains(stdout, "test-agent") {
			t.Errorf("Expected agent name in output, got: %s", stdout)
		}
	})

	t.Run("ListShowsClaim", func(t *testing.T) {
		stdout, _, err := runner.Run("list")
		if err != nil {
			t.Fatalf("Failed to list sagas: %v", err)
		}
		if !strings.Contains(stdout, "claimed by test-agent") {
			t.Errorf("Expected claim status in list, got: %s", stdout)
		}
	})

	t.Run("UnclaimSaga", func(t *testing.T) {
		stdout, _, _ := runner.Run("list")
		lines := strings.Split(stdout, "\n")
		var sagaID string
		for _, line := range lines {
			if strings.Contains(line, "Test saga") {
				fields := strings.Fields(line)
				if len(fields) > 0 {
					sagaID = fields[0]
					break
				}
			}
		}

		stdout, _, err := runner.Run("unclaim", sagaID)
		if err != nil {
			t.Fatalf("Failed to unclaim saga: %v", err)
		}
		if !strings.Contains(stdout, "Released claim") {
			t.Errorf("Expected 'Released claim' in output, got: %s", stdout)
		}
	})

	t.Run("MarkDone", func(t *testing.T) {
		// First complete sub-saga
		stdout, _, _ := runner.Run("list")
		lines := strings.Split(stdout, "\n")
		var subSagaID string
		for _, line := range lines {
			if strings.Contains(line, "Sub task") {
				fields := strings.Fields(line)
				if len(fields) > 0 {
					subSagaID = fields[0]
					break
				}
			}
		}

		if subSagaID != "" {
			_, _, err := runner.Run("done", subSagaID)
			if err != nil {
				t.Fatalf("Failed to complete sub-saga: %v", err)
			}
		}

		// Now complete parent
		stdout, _, _ = runner.Run("list")
		lines = strings.Split(stdout, "\n")
		var parentID string
		for _, line := range lines {
			if strings.Contains(line, "Test saga") {
				fields := strings.Fields(line)
				if len(fields) > 0 {
					parentID = fields[0]
					break
				}
			}
		}

		stdout, _, err := runner.Run("done", parentID)
		if err != nil {
			t.Fatalf("Failed to complete saga: %v", err)
		}
		if !strings.Contains(stdout, "Marked saga") {
			t.Errorf("Expected 'Marked saga' in output, got: %s", stdout)
		}
	})

	t.Run("ListAllShowsDone", func(t *testing.T) {
		stdout, _, err := runner.Run("list", "--all")
		if err != nil {
			t.Fatalf("Failed to list all sagas: %v", err)
		}
		if !strings.Contains(stdout, "done") {
			t.Errorf("Expected done status in list, got: %s", stdout)
		}
	})

	fmt.Println("✅ All E2E tests passed!")
}

// Test specific scenarios
func TestClaimExpiry(t *testing.T) {
	runner := NewTestRunner(t)
	defer runner.Cleanup()

	// Create and claim saga
	runner.Run("new", "Expiry test")
	stdout, _, _ := runner.Run("list")
	lines := strings.Split(stdout, "\n")
	var sagaID string
	for _, line := range lines {
		if strings.Contains(line, "Expiry test") {
			fields := strings.Fields(line)
			if len(fields) > 0 {
				sagaID = fields[0]
				break
			}
		}
	}

	// Claim with 1 second duration
	_, _, err := runner.Run("claim", sagaID, "--agent", "test", "--duration", "1s")
	if err != nil {
		t.Fatalf("Failed to claim: %v", err)
	}

	// Wait for expiry
	time.Sleep(2 * time.Second)

	// Should be able to claim again
	stdout, _, err = runner.Run("claim", sagaID, "--agent", "test2")
	if err != nil {
		t.Fatalf("Should be able to claim after expiry: %v", err)
	}
	if !strings.Contains(stdout, "test2") {
		t.Errorf("Expected new agent in claim output: %s", stdout)
	}
}
