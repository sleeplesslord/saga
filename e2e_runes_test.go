package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunner for runes integration
type RunesTestRunner struct {
	SagaBinary  string
	RunesBinary string
	TempDir     string
	t           *testing.T
}

// NewRunesTestRunner creates test runner with both binaries
func NewRunesTestRunner(t *testing.T) *RunesTestRunner {
	wd, _ := os.Getwd()

	// Build saga binary
	sagaBin := filepath.Join(wd, "sg_test")
	exec.Command("go", "build", "-o", sagaBin, "./cmd/sg").Run()

	// Build runes binary (from runes repo) - name it 'runes' so sg context can find it
	runesBin := filepath.Join(wd, "runes")
	exec.Command("go", "build", "-o", runesBin, "../runes/cmd/runes").Run()

	tempDir, _ := os.MkdirTemp("", "saga-runes-e2e-*")
	os.MkdirAll(filepath.Join(tempDir, ".saga"), 0755)
	os.MkdirAll(filepath.Join(tempDir, ".runes"), 0755)

	return &RunesTestRunner{
		SagaBinary:  sagaBin,
		RunesBinary: runesBin,
		TempDir:     tempDir,
		t:           t,
	}
}

func (r *RunesTestRunner) Cleanup() {
	os.RemoveAll(r.TempDir)
	os.Remove(r.SagaBinary)
	os.Remove(r.RunesBinary)
}

func (r *RunesTestRunner) RunSaga(args ...string) (string, error) {
	cmd := exec.Command(r.SagaBinary, args...)
	cmd.Dir = r.TempDir
	// Add runes binary directory to PATH so sg context can find it
	cmd.Env = append(os.Environ(), "PATH="+filepath.Dir(r.RunesBinary)+":"+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (r *RunesTestRunner) RunRunes(args ...string) (string, error) {
	cmd := exec.Command(r.RunesBinary, args...)
	cmd.Dir = r.TempDir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// TestRunesIntegration tests saga-runes workflow
func TestRunesIntegration(t *testing.T) {
	runner := NewRunesTestRunner(t)
	defer runner.Cleanup()

	var sagaID string
	var runeTitle = "Auth timeout fix"

	t.Run("CreateSagaAndRune", func(t *testing.T) {
		// Create a saga
		out, err := runner.RunSaga("new", "Implement auth")
		if err != nil {
			t.Fatalf("Failed to create saga: %v\n%s", err, out)
		}

		// Get saga ID
		out, _ = runner.RunSaga("list")
		for _, line := range strings.Split(out, "\n") {
			if strings.Contains(line, "Implement auth") {
				fields := strings.Fields(line)
				if len(fields) > 0 {
					sagaID = fields[0]
					break
				}
			}
		}
		if sagaID == "" {
			t.Fatal("Could not find saga ID")
		}

		// Create a rune linked to the saga
		out, err = runner.RunRunes("add", runeTitle,
			"--problem", "OAuth timing out",
			"--solution", "Increase timeout to 30s",
			"--saga", sagaID,
			"--tag", "auth",
			"--learned", "Always buffer network timeouts")
		if err != nil {
			t.Fatalf("Failed to create rune: %v\n%s", err, out)
		}

		if !strings.Contains(out, "Created rune") {
			t.Errorf("Expected rune creation: %s", out)
		}
	})

	t.Run("SagaContextShowsRunes", func(t *testing.T) {
		if sagaID == "" {
			t.Fatal("No saga ID from previous test")
		}



		// Check context shows rune
		// Both saga and runes use the temp directory with .saga/ and .runes/
		out, err := runner.RunSaga("context", sagaID)
		if err != nil {
			t.Fatalf("Failed to get context: %v\n%s", err, out)
		}

		// Should show knowledge section with rune
		if !strings.Contains(out, "KNOWLEDGE") {
			t.Errorf("Expected KNOWLEDGE section in context. Output:\n%s", out)
		}
		if !strings.Contains(out, runeTitle) {
			t.Errorf("Expected rune title '%s' in context. Output:\n%s", runeTitle, out)
		}
	})

	t.Run("SearchRunesBySaga", func(t *testing.T) {
		// Search for runes containing saga reference
		out, err := runner.RunRunes("search", "auth")
		if err != nil {
			t.Fatalf("Failed to search: %v\n%s", err, out)
		}

		if !strings.Contains(out, runeTitle) {
			t.Errorf("Expected to find rune '%s': %s", runeTitle, out)
		}
	})

	t.Run("CompleteWorkflow", func(t *testing.T) {
		if sagaID == "" {
			t.Fatal("No saga ID")
		}

		// Mark saga done
		out, err := runner.RunSaga("done", sagaID)
		if err != nil {
			t.Fatalf("Failed to complete: %v\n%s", err, out)
		}
		if !strings.Contains(out, "Marked saga") {
			t.Errorf("Expected completion: %s", out)
		}

		// Rune remains as knowledge for future reference
		out, _ = runner.RunRunes("list")
		if !strings.Contains(out, runeTitle) {
			t.Errorf("Rune should still exist after saga done: %s", out)
		}
	})

	fmt.Println("✅ Saga-Runes integration tests passed!")
}
