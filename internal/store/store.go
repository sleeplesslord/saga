package store

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	
	// In-memory indexes (rebuilt from JSONL)
	indexByID       map[string]*saga.Saga
	indexByParent   map[string][]*saga.Saga
	indexByStatus   map[saga.Status][]*saga.Saga
	indexByLabel    map[string][]*saga.Saga
	indexLoaded     bool
}

// loadIndexes builds in-memory indexes from JSONL files
func (s *Store) loadIndexes() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.indexLoaded {
		return nil
	}
	
	// Initialize indexes
	s.indexByID = make(map[string]*saga.Saga)
	s.indexByParent = make(map[string][]*saga.Saga)
	s.indexByStatus = make(map[saga.Status][]*saga.Saga)
	s.indexByLabel = make(map[string][]*saga.Saga)
	
	// Load all sagas
	sagas, err := s.loadAllUnlocked()
	if err != nil {
		return err
	}
	
	// Build indexes
	for _, sg := range sagas {
		s.indexSaga(sg)
	}
	
	s.indexLoaded = true
	return nil
}

// indexSaga adds a saga to all indexes
func (s *Store) indexSaga(sg *saga.Saga) {
	s.indexByID[sg.ID] = sg
	
	// Index by parent
	if sg.IsSubSaga() {
		s.indexByParent[sg.ParentID] = append(s.indexByParent[sg.ParentID], sg)
	}
	
	// Index by status
	s.indexByStatus[sg.Status] = append(s.indexByStatus[sg.Status], sg)
	
	// Index by labels
	for _, label := range sg.Labels {
		s.indexByLabel[label] = append(s.indexByLabel[label], sg)
	}
}

// updateIndex replaces a saga in indexes after update
func (s *Store) updateIndex(old, updated *saga.Saga) {
	// Remove old from status index
	if old != nil {
		s.removeFromStatusIndex(old)
		s.removeFromLabelIndex(old)
	}
	
	// Add updated
	s.indexByID[updated.ID] = updated
	s.indexByStatus[updated.Status] = append(s.indexByStatus[updated.Status], updated)
	for _, label := range updated.Labels {
		s.indexByLabel[label] = append(s.indexByLabel[label], updated)
	}
}

func (s *Store) removeFromStatusIndex(sg *saga.Saga) {
	var filtered []*saga.Saga
	for _, s := range s.indexByStatus[sg.Status] {
		if s.ID != sg.ID {
			filtered = append(filtered, s)
		}
	}
	s.indexByStatus[sg.Status] = filtered
}

func (s *Store) removeFromLabelIndex(sg *saga.Saga) {
	for _, label := range sg.Labels {
		var filtered []*saga.Saga
		for _, s := range s.indexByLabel[label] {
			if s.ID != sg.ID {
				filtered = append(filtered, s)
			}
		}
		s.indexByLabel[label] = filtered
	}
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
// Also checks for git worktrees to find .saga in the main repo
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

		// Check for git worktree (.git file points to main repo)
		if worktreeDir := findWorktreeSagaDir(dir); worktreeDir != "" {
			return worktreeDir
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

// findWorktreeSagaDir checks if dir is a git worktree and returns path to main repo's .saga
func findWorktreeSagaDir(dir string) string {
	gitFile := filepath.Join(dir, ".git")
	info, err := os.Stat(gitFile)
	if err != nil || info.IsDir() {
		// Not a worktree (either no .git or it's a directory)
		return ""
	}

	// .git is a file - this is a worktree, read it to find main repo
	content, err := os.ReadFile(gitFile)
	if err != nil {
		return ""
	}

	// Parse "gitdir: /path/to/main/.git/worktrees/..." format
	contentStr := string(content)
	const prefix = "gitdir: "
	if idx := len(prefix); len(contentStr) > idx {
		gitDir := strings.TrimSpace(contentStr[idx:])
		// gitDir points to .git/worktrees/<name>, go up to find main repo
		mainRepoDir := filepath.Dir(filepath.Dir(gitDir))
		sagaDir := filepath.Join(mainRepoDir, ".saga")
		if info, err := os.Stat(sagaDir); err == nil && info.IsDir() {
			return filepath.Join(sagaDir, "sagas.jsonl")
		}
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

// loadAllUnlocked loads all sagas without locking (caller must hold lock)
func (s *Store) loadAllUnlocked() ([]*saga.Saga, error) {
	var allSagas []*saga.Saga
	
	// Load global
	sagas, err := s.loadFromPathUnlocked(s.globalPath)
	if err != nil {
		return nil, err
	}
	allSagas = append(allSagas, sagas...)
	
	// Load local if exists
	if s.localPath != "" {
		sagas, err = s.loadFromPathUnlocked(s.localPath)
		if err != nil {
			return nil, err
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

// Save appends a saga to storage (default: local if in project, else global).
// Returns an error if a saga with the same ID already exists.
func (s *Store) Save(sg *saga.Saga, scope ...Scope) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicate ID across all scopes
	existing, _ := s.loadAllUnlocked()
	for _, e := range existing {
		if e.ID == sg.ID {
			return fmt.Errorf("duplicate saga ID: %s", sg.ID)
		}
	}

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

	var old *saga.Saga
	if s.indexLoaded {
		old = s.indexByID[updated.ID]
	}

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
			// Update indexes
			if s.indexLoaded {
				s.updateIndex(old, updated)
			}
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

// GetByID finds a saga by its ID (uses index if available)
func (s *Store) GetByID(id string) (*saga.Saga, error) {
	// Ensure indexes are loaded
	if err := s.loadIndexes(); err != nil {
		return nil, err
	}
	
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if sg, ok := s.indexByID[id]; ok {
		return sg, nil
	}
	
	return nil, fmt.Errorf("saga not found: %s", id)
}

// GetChildren returns all direct children of a saga (uses index)
func (s *Store) GetChildren(parentID string) ([]*saga.Saga, error) {
	// Ensure indexes are loaded
	if err := s.loadIndexes(); err != nil {
		return nil, err
	}
	
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return s.indexByParent[parentID], nil
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

// HasIncompleteDependencies returns true if any dependency is not done
func (s *Store) HasIncompleteDependencies(sagaID string) (bool, []string, error) {
	sg, err := s.GetByID(sagaID)
	if err != nil {
		return false, nil, err
	}

	var incomplete []string
	for _, depID := range sg.DependsOn {
		dep, err := s.GetByID(depID)
		if err != nil {
			// Dependency not found - treat as incomplete
			incomplete = append(incomplete, depID)
			continue
		}
		if dep.Status != saga.StatusDone {
			incomplete = append(incomplete, depID)
		}
	}

	return len(incomplete) > 0, incomplete, nil
}

// WouldCreateCircularDependency checks if adding a dependency would create a cycle
func (s *Store) WouldCreateCircularDependency(sagaID string, targetID string) (bool, error) {
	// Check if target depends on saga (directly or transitively)
	visited := make(map[string]bool)
	var check func(string) (bool, error)
	check = func(id string) (bool, error) {
		if id == sagaID {
			return true, nil
		}
		if visited[id] {
			return false, nil
		}
		visited[id] = true

		sg, err := s.GetByID(id)
		if err != nil {
			return false, nil // Orphan dependency
		}

		for _, depID := range sg.DependsOn {
			if circular, err := check(depID); err != nil || circular {
				return circular, err
			}
		}
		return false, nil
	}

	return check(targetID)
}
