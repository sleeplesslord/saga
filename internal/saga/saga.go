package saga

import (
	"time"
)

// Status represents the current state of a saga
type Status string

const (
	StatusActive Status = "active"
	StatusPaused Status = "paused"
	StatusDone   Status = "done"
)

// Saga represents a task or project
type Saga struct {
	ID       string `json:"id"`
	ParentID string `json:"parent_id,omitempty"`
	Title    string `json:"title"`
	Status   Status `json:"status"`
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

// NewSubSaga creates a new saga as a child of parentID
func NewSubSaga(title string, parentID string) *Saga {
	sg := NewSaga(title)
	sg.ParentID = parentID
	sg.History[0].Note = "Sub-saga created"
	return sg
}

// IsSubSaga returns true if this saga has a parent
func (s *Saga) IsSubSaga() bool {
	return s.ParentID != ""
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

// generateID creates a short unique identifier
func generateID() string {
	// Simple implementation: first 4 chars of timestamp hash
	// Replace with better solution later
	const alphabet = "0123456789abcdefghijklmnopqrstuvwxyz"
	now := time.Now().UnixNano()
	result := make([]byte, 4)
	for i := 0; i < 4; i++ {
		result[i] = alphabet[now%int64(len(alphabet))]
		now /= int64(len(alphabet))
	}
	return string(result)
}
