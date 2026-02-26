package store

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/hbn/saga/internal/saga"
)

// Scope defines where sagas are stored
type Scope int

const (
	ScopeGlobal Scope = iota
	ScopeLocal
)

// Store handles persistence of sagas
type Store struct {
	globalPath string
	localPath  string
	mu         sync.RWMutex
}

// New creates a new Store with global path, auto-detects local
func New(globalPath string) (*Store, error) {
	// Ensure global directory exists
	globalDir := filepath.Dir(globalPath)
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		return nil, fmt.Errorf("creating global store directory: %w", err)
	}

	s := &Store{globalPath: globalPath}

	// Check for local .saga directory
	if localPath := findLocalSagaDir(); localPath != "" {
		s.localPath = localPath
	}

	return s, nil
}

// findLocalSagaDir searches for .saga/ directory in current or parent directories
func findLocalSagaDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
		sagaDir := filepath.Join(dir, ".saga")
		// Check if .saga directory exists
		if info, err := os.Stat(sagaDir); err == nil && info.IsDir() {
			return filepath.Join(sagaDir, "sagas.jsonl")
		}

		// Check if we can go up
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return ""
}

// HasLocal returns true if local store exists
func (s *Store) HasLocal() bool {
	return s.localPath != ""
}

// LocalPath returns the local store path (empty if none)
func (s *Store) LocalPath() string {
	return s.localPath
}

// DefaultPath returns the default storage path
func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".saga/sagas.jsonl"
	}
	return filepath.Join(home, ".saga", "sagas.jsonl")
}

// LoadAll reads sagas from specified scopes
func (s *Store) LoadAll(scopes ...Scope) ([]*saga.Saga, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(scopes) == 0 {
		scopes = []Scope{ScopeGlobal}
		if s.HasLocal() {
			scopes = append(scopes, ScopeLocal)
		}
	}

	var allSagas []*saga.Saga

	for _, scope := range scopes {
		var path string
		switch scope {
		case ScopeGlobal:
			path = s.globalPath
		case ScopeLocal:
			path = s.localPath
		}

		if path == "" {
			continue
		}

		sagas, err := s.loadFromPath(path)
		if err != nil {
			return nil, fmt.Errorf("loading from %v: %w", scope, err)
		}
		allSagas = append(allSagas, sagas...)
	}

	return allSagas, nil
}

// loadFromPath loads sagas from a specific file path
func (s *Store) loadFromPath(path string) ([]*saga.Saga, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []*saga.Saga{}, nil
		}
		return nil, err
	}
	defer file.Close()

	var sagas []*saga.Saga
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var sg saga.Saga
		if err := json.Unmarshal(scanner.Bytes(), &sg); err != nil {
			continue // Skip malformed lines
		}
		sagas = append(sagas, &sg)
	}

	return sagas, scanner.Err()
}

// Save appends a saga to storage (default: local if in project, else global)
func (s *Store) Save(sg *saga.Saga, scope ...Scope) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Determine scope
	targetScope := ScopeGlobal
	if s.HasLocal() && (len(scope) == 0 || scope[0] == ScopeLocal) {
		targetScope = ScopeLocal
	}
	if len(scope) > 0 {
		targetScope = scope[0]
	}

	path := s.globalPath
	if targetScope == ScopeLocal && s.localPath != "" {
		path = s.localPath
	}

	// Ensure directory exists for local
	if targetScope == ScopeLocal {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating local store directory: %w", err)
		}
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening store: %w", err)
	}
	defer file.Close()

	data, err := json.Marshal(sg)
	if err != nil {
		return fmt.Errorf("encoding saga: %w", err)
	}

	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("writing saga: %w", err)
	}
	if _, err := file.WriteString("\n"); err != nil {
		return fmt.Errorf("writing newline: %w", err)
	}

	return nil
}

// InitLocal creates a local .saga directory in current working directory
func (s *Store) InitLocal() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	localDir := filepath.Join(cwd, ".saga")
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return fmt.Errorf("creating .saga directory: %w", err)
	}

	s.localPath = filepath.Join(localDir, "sagas.jsonl")
	return nil
}

// Update replaces an existing saga in storage (searches both scopes)
func (s *Store) Update(updated *saga.Saga) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Try local first, then global
	scopes := []Scope{ScopeLocal, ScopeGlobal}
	for _, scope := range scopes {
		path := s.localPath
		if scope == ScopeGlobal {
			path = s.globalPath
		}
		if path == "" {
			continue
		}

		sagas, err := s.loadFromPathUnlocked(path)
		if err != nil {
			return err
		}

		found := false
		for i, sg := range sagas {
			if sg.ID == updated.ID {
				sagas[i] = updated
				found = true
				break
			}
		}

		if found {
			return s.saveAllUnlocked(path, sagas)
		}
	}

	return fmt.Errorf("saga not found: %s", updated.ID)
}

// loadFromPathUnlocked loads sagas without locking (caller must hold lock)
func (s *Store) loadFromPathUnlocked(path string) ([]*saga.Saga, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []*saga.Saga{}, nil
		}
		return nil, err
	}
	defer file.Close()

	var sagas []*saga.Saga
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var sg saga.Saga
		if err := json.Unmarshal(scanner.Bytes(), &sg); err != nil {
			continue
		}
		sagas = append(sagas, &sg)
	}

	return sagas, scanner.Err()
}

// saveAllUnlocked writes all sagas to a specific path without locking
func (s *Store) saveAllUnlocked(path string, sagas []*saga.Saga) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating store: %w", err)
	}
	defer file.Close()

	for _, sg := range sagas {
		data, err := json.Marshal(sg)
		if err != nil {
			return fmt.Errorf("encoding saga: %w", err)
		}
		if _, err := file.Write(data); err != nil {
			return fmt.Errorf("writing saga: %w", err)
		}
		if _, err := file.WriteString("\n"); err != nil {
			return fmt.Errorf("writing newline: %w", err)
		}
	}

	return nil
}

// GetByID finds a saga by its ID (searches both scopes)
func (s *Store) GetByID(id string) (*saga.Saga, error) {
	// Try local first, then global
	scopes := []Scope{ScopeLocal, ScopeGlobal}
	for _, scope := range scopes {
		sagas, err := s.LoadAll(scope)
		if err != nil {
			return nil, err
		}
		for _, sg := range sagas {
			if sg.ID == id {
				return sg, nil
			}
		}
	}

	return nil, fmt.Errorf("saga not found: %s", id)
}

// GetChildren returns all direct children of a saga
func (s *Store) GetChildren(parentID string) ([]*saga.Saga, error) {
	sagas, err := s.LoadAll()
	if err != nil {
		return nil, err
	}

	var children []*saga.Saga
	for _, sg := range sagas {
		if sg.ParentID == parentID {
			children = append(children, sg)
		}
	}

	return children, nil
}

// HasActiveChildren returns true if saga has any children that aren't done
func (s *Store) HasActiveChildren(parentID string) (bool, error) {
	children, err := s.GetChildren(parentID)
	if err != nil {
		return false, err
	}

	for _, child := range children {
		if child.Status != saga.StatusDone {
			return true, nil
		}
	}

	return false, nil
}

// GetNextChildID returns the next available child ID for a parent
// Format: parent.1, parent.2, etc.
func (s *Store) GetNextChildID(parentID string) (string, error) {
	children, err := s.GetChildren(parentID)
	if err != nil {
		return "", err
	}

	// Find highest existing number
	maxNum := 0
	for _, child := range children {
		// Parse child ID format: parent.N
		var num int
		if _, err := fmt.Sscanf(child.ID, parentID+".%d", &num); err == nil {
			if num > maxNum {
				maxNum = num
			}
		}
	}

	return fmt.Sprintf("%s.%d", parentID, maxNum+1), nil
}
