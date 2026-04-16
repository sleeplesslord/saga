package saga

import (
	"crypto/rand"
	"encoding/binary"
	"time"
)

// Status represents the current state of a saga
type Status string

const (
	StatusActive Status = "active"
	StatusPaused Status = "paused"
	StatusDone   Status = "done"
	StatusWontDo Status = "wontdo"
)

// Priority represents the priority level of a saga
type Priority string

const (
	PriorityHigh   Priority = "high"
	PriorityNormal Priority = "normal"
	PriorityLow    Priority = "low"
)

// Saga represents a task or project
type Saga struct {
	ID          string    `json:"id"`
	ParentID    string    `json:"parent_id,omitempty"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Status      Status    `json:"status"`
	Priority    Priority  `json:"priority,omitempty"`
	Labels      []string  `json:"labels,omitempty"`
	DependsOn   []string  `json:"depends_on,omitempty"` // Hard dependencies (blocking)
	RelatedTo   []string  `json:"related_to,omitempty"` // Soft relationships (informational)
	ClaimedBy   string    `json:"claimed_by,omitempty"`
	ClaimedAt   time.Time `json:"claimed_at,omitempty"`
	Deadline    string    `json:"deadline,omitempty"` // YYYYMMDD format
	// IndentLevel is computed, not stored
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	History   []HistoryEntry `json:"history"`
}

// HistoryEntry tracks what happened and when
type HistoryEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Action    string    `json:"action"`
	Note      string    `json:"note,omitempty"`
}

// NewSaga creates a new saga with the given title
func NewSaga(title string) *Saga {
	now := time.Now()
	return &Saga{
		ID:        generateID(),
		Title:     title,
		Status:    StatusActive,
		Priority:  PriorityNormal,
		CreatedAt: now,
		UpdatedAt: now,
		History: []HistoryEntry{
			{
				Timestamp: now,
				Action:    "created",
				Note:      "Saga created",
			},
		},
	}
}

// NewSubSaga creates a new saga as a child of parentID with explicit ID
func NewSubSaga(title string, id string, parentID string) *Saga {
	now := time.Now()
	return &Saga{
		ID:        id,
		ParentID:  parentID,
		Title:     title,
		Status:    StatusActive,
		Priority:  PriorityNormal,
		CreatedAt: now,
		UpdatedAt: now,
		History: []HistoryEntry{
			{
				Timestamp: now,
				Action:    "created",
				Note:      "Sub-saga created",
			},
		},
	}
}

// IsSubSaga returns true if this saga has a parent
func (s *Saga) IsSubSaga() bool {
	return s.ParentID != ""
}

// HasLabel returns true if saga has the given label
func (s *Saga) HasLabel(label string) bool {
	for _, l := range s.Labels {
		if l == label {
			return true
		}
	}
	return false
}

// AddLabel adds a label to the saga if not already present
func (s *Saga) AddLabel(label string) {
	if s.HasLabel(label) {
		return
	}
	s.Labels = append(s.Labels, label)
	s.UpdatedAt = time.Now()
}

// RemoveLabel removes a label from the saga
func (s *Saga) RemoveLabel(label string) {
	for i, l := range s.Labels {
		if l == label {
			s.Labels = append(s.Labels[:i], s.Labels[i+1:]...)
			s.UpdatedAt = time.Now()
			return
		}
	}
}

// SetPriority changes the saga priority and logs it
func (s *Saga) SetPriority(priority Priority) {
	if s.Priority == priority {
		return
	}
	oldPriority := s.Priority
	s.Priority = priority
	s.AddHistory("priority_changed", string(oldPriority)+" -> "+string(priority))
}

// AddDependency adds a hard dependency (blocks completion until target is done)
func (s *Saga) AddDependency(targetID string) {
	for _, id := range s.DependsOn {
		if id == targetID {
			return
		}
	}
	s.DependsOn = append(s.DependsOn, targetID)
	s.UpdatedAt = time.Now()
}

// RemoveDependency removes a hard dependency
func (s *Saga) RemoveDependency(targetID string) {
	for i, id := range s.DependsOn {
		if id == targetID {
			s.DependsOn = append(s.DependsOn[:i], s.DependsOn[i+1:]...)
			s.UpdatedAt = time.Now()
			return
		}
	}
}

// HasDependency returns true if saga depends on target
func (s *Saga) HasDependency(targetID string) bool {
	for _, id := range s.DependsOn {
		if id == targetID {
			return true
		}
	}
	return false
}

// AddRelationship adds a soft relationship (informational only)
func (s *Saga) AddRelationship(targetID string) {
	for _, id := range s.RelatedTo {
		if id == targetID {
			return
		}
	}
	s.RelatedTo = append(s.RelatedTo, targetID)
	s.UpdatedAt = time.Now()
}

// RemoveRelationship removes a soft relationship
func (s *Saga) RemoveRelationship(targetID string) {
	for i, id := range s.RelatedTo {
		if id == targetID {
			s.RelatedTo = append(s.RelatedTo[:i], s.RelatedTo[i+1:]...)
			s.UpdatedAt = time.Now()
			return
		}
	}
}

// HasRelationship returns true if saga is related to target
func (s *Saga) HasRelationship(targetID string) bool {
	for _, id := range s.RelatedTo {
		if id == targetID {
			return true
		}
	}
	return false
}

// IsClaimed returns true if saga is currently claimed
func (s *Saga) IsClaimed() bool {
	if s.ClaimedBy == "" {
		return false
	}
	// Check if claim expired (24 hours default)
	expiry := s.ClaimedAt.Add(24 * time.Hour)
	return time.Now().Before(expiry)
}

// Claim marks saga as claimed by agent
func (s *Saga) Claim(agent string) {
	s.ClaimedBy = agent
	s.ClaimedAt = time.Now()
	s.UpdatedAt = time.Now()
	s.AddHistory("claimed", "Claimed by "+agent)
}

// Unclaim releases the claim
func (s *Saga) Unclaim() {
	if s.ClaimedBy != "" {
		s.AddHistory("unclaimed", "Released by "+s.ClaimedBy)
	}
	s.ClaimedBy = ""
	s.ClaimedAt = time.Time{}
	s.UpdatedAt = time.Now()
}

// ClaimExpiry returns when claim expires
func (s *Saga) ClaimExpiry() time.Time {
	if s.ClaimedAt.IsZero() {
		return time.Time{}
	}
	return s.ClaimedAt.Add(24 * time.Hour)
}

// AddHistory adds a new entry to the saga's history
func (s *Saga) AddHistory(action, note string) {
	s.History = append(s.History, HistoryEntry{
		Timestamp: time.Now(),
		Action:    action,
		Note:      note,
	})
	s.UpdatedAt = time.Now()
}

// SetStatus changes the saga status and logs it
func (s *Saga) SetStatus(status Status) {
	if s.Status == status {
		return
	}
	oldStatus := s.Status
	s.Status = status
	s.AddHistory("status_changed", string(oldStatus)+" -> "+string(status))
}

// generateID creates a short unique identifier using cryptographic randomness.
// Produces 6-character base36 IDs (36^6 ≈ 2.2 billion combinations).
func generateID() string {
	const alphabet = "0123456789abcdefghijklmnopqrstuvwxyz"
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		// Fallback: mix timestamp with whatever entropy is available
		binary.LittleEndian.PutUint64(buf[:], uint64(time.Now().UnixNano()))
	}
	n := binary.LittleEndian.Uint64(buf[:])
	result := make([]byte, 6)
	for i := range result {
		result[i] = alphabet[n%uint64(len(alphabet))]
		n /= uint64(len(alphabet))
	}
	return string(result)
}
