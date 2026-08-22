package webui

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sleeplesslord/saga/internal/saga"
	"github.com/sleeplesslord/saga/internal/store"
)

//go:embed index.html assets/*
var files embed.FS

// Options configures the read-only web interface.
type Options struct {
	ProjectName string
	Scope       store.Scope
}

// Handler returns the complete read-only Saga web application.
func Handler(st *store.Store, opts Options) (http.Handler, error) {
	assets, err := fs.Sub(files, "assets")
	if err != nil {
		return nil, fmt.Errorf("loading embedded web assets: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(assets))))
	mux.HandleFunc("GET /api/sagas", func(w http.ResponseWriter, r *http.Request) {
		serveSnapshot(w, st, opts)
	})
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		data, err := files.ReadFile("index.html")
		if err != nil {
			http.Error(w, "Saga web interface is unavailable", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(data)
	})

	return securityHeaders(mux), nil
}

type snapshot struct {
	GeneratedAt time.Time      `json:"generated_at"`
	Project     string         `json:"project"`
	Scope       string         `json:"scope"`
	Total       int            `json:"total"`
	Counts      snapshotCounts `json:"counts"`
	Sagas       []task         `json:"sagas"`
}

type snapshotCounts struct {
	Active  int `json:"active"`
	Paused  int `json:"paused"`
	Done    int `json:"done"`
	WontDo  int `json:"wontdo"`
	Ready   int `json:"ready"`
	Blocked int `json:"blocked"`
	Claimed int `json:"claimed"`
}

type task struct {
	ID                   string              `json:"id"`
	ParentID             string              `json:"parent_id,omitempty"`
	Title                string              `json:"title"`
	Description          string              `json:"description,omitempty"`
	Plan                 string              `json:"plan,omitempty"`
	Status               saga.Status         `json:"status"`
	Priority             saga.Priority       `json:"priority"`
	Labels               []string            `json:"labels"`
	DependsOn            []string            `json:"depends_on"`
	RelatedTo            []string            `json:"related_to"`
	ClaimedBy            string              `json:"claimed_by,omitempty"`
	Claimed              bool                `json:"claimed"`
	Deadline             string              `json:"deadline,omitempty"`
	CreatedAt            time.Time           `json:"created_at"`
	UpdatedAt            time.Time           `json:"updated_at"`
	History              []saga.HistoryEntry `json:"history"`
	Depth                int                 `json:"depth"`
	Children             []string            `json:"children"`
	BlockingDependencies []string            `json:"blocking_dependencies"`
	Blocked              bool                `json:"blocked"`
	Ready                bool                `json:"ready"`
}

func serveSnapshot(w http.ResponseWriter, st *store.Store, opts Options) {
	sagas, err := st.LoadAll(opts.Scope)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Could not read Saga storage. Check the terminal for storage errors, then refresh.",
		})
		return
	}

	claimDuration := st.ClaimDuration()
	result := buildSnapshot(sagas, claimDuration, opts)
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, result)
}

func buildSnapshot(sagas []*saga.Saga, claimDuration time.Duration, opts Options) snapshot {
	byID := make(map[string]*saga.Saga, len(sagas))
	children := make(map[string][]string)
	for _, sg := range sagas {
		byID[sg.ID] = sg
		if sg.ParentID != "" {
			children[sg.ParentID] = append(children[sg.ParentID], sg.ID)
		}
	}
	for id := range children {
		sort.Strings(children[id])
	}

	out := snapshot{
		GeneratedAt: time.Now(),
		Project:     opts.ProjectName,
		Scope:       scopeName(opts.Scope),
		Total:       len(sagas),
		Sagas:       make([]task, 0, len(sagas)),
	}
	for _, sg := range sagas {
		blocking := make([]string, 0)
		for _, dependencyID := range sg.DependsOn {
			dependency := byID[dependencyID]
			if dependency == nil || (dependency.Status != saga.StatusDone && dependency.Status != saga.StatusWontDo) {
				blocking = append(blocking, dependencyID)
			}
		}
		hasActiveChildren := false
		for _, childID := range children[sg.ID] {
			if child := byID[childID]; child != nil && child.Status == saga.StatusActive {
				hasActiveChildren = true
				break
			}
		}
		blocked := len(blocking) > 0 || hasActiveChildren || parentBlocked(sg, byID)
		claimed := sg.IsClaimedWithDuration(claimDuration)
		ready := sg.Status == saga.StatusActive && !blocked && !claimed

		t := task{
			ID:                   sg.ID,
			ParentID:             sg.ParentID,
			Title:                sg.Title,
			Description:          sg.Description,
			Plan:                 sg.Plan,
			Status:               sg.Status,
			Priority:             normalizedPriority(sg.Priority),
			Labels:               nonNil(sg.Labels),
			DependsOn:            nonNil(sg.DependsOn),
			RelatedTo:            nonNil(sg.RelatedTo),
			ClaimedBy:            sg.ClaimedBy,
			Claimed:              claimed,
			Deadline:             sg.Deadline,
			CreatedAt:            sg.CreatedAt,
			UpdatedAt:            sg.UpdatedAt,
			History:              sg.History,
			Depth:                taskDepth(sg, byID),
			Children:             nonNil(children[sg.ID]),
			BlockingDependencies: nonNil(blocking),
			Blocked:              blocked,
			Ready:                ready,
		}
		out.Sagas = append(out.Sagas, t)

		switch sg.Status {
		case saga.StatusActive:
			out.Counts.Active++
		case saga.StatusPaused:
			out.Counts.Paused++
		case saga.StatusDone:
			out.Counts.Done++
		case saga.StatusWontDo:
			out.Counts.WontDo++
		}
		if ready {
			out.Counts.Ready++
		}
		if sg.Status == saga.StatusActive && blocked {
			out.Counts.Blocked++
		}
		if claimed && sg.Status == saga.StatusActive {
			out.Counts.Claimed++
		}
	}

	sort.SliceStable(out.Sagas, func(i, j int) bool {
		if out.Sagas[i].Status != out.Sagas[j].Status {
			return statusRank(out.Sagas[i].Status) < statusRank(out.Sagas[j].Status)
		}
		if out.Sagas[i].Priority != out.Sagas[j].Priority {
			return priorityRank(out.Sagas[i].Priority) < priorityRank(out.Sagas[j].Priority)
		}
		return out.Sagas[i].UpdatedAt.After(out.Sagas[j].UpdatedAt)
	})
	return out
}

func parentBlocked(sg *saga.Saga, byID map[string]*saga.Saga) bool {
	if sg.ParentID == "" {
		return false
	}
	parent := byID[sg.ParentID]
	if parent == nil || parent.Status != saga.StatusActive {
		return true
	}
	for _, id := range parent.DependsOn {
		dependency := byID[id]
		if dependency == nil || (dependency.Status != saga.StatusDone && dependency.Status != saga.StatusWontDo) {
			return true
		}
	}
	return false
}

func taskDepth(sg *saga.Saga, byID map[string]*saga.Saga) int {
	depth := 0
	seen := map[string]bool{sg.ID: true}
	parentID := sg.ParentID
	for parentID != "" && depth < 50 && !seen[parentID] {
		seen[parentID] = true
		parent := byID[parentID]
		if parent == nil {
			break
		}
		depth++
		parentID = parent.ParentID
	}
	return depth
}

func normalizedPriority(priority saga.Priority) saga.Priority {
	if priority == "" {
		return saga.PriorityNormal
	}
	return priority
}

func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func scopeName(scope store.Scope) string {
	if scope == store.ScopeLocal {
		return "project"
	}
	return "global"
}

func statusRank(status saga.Status) int {
	switch status {
	case saga.StatusActive:
		return 0
	case saga.StatusPaused:
		return 1
	case saga.StatusDone:
		return 2
	default:
		return 3
	}
}

func priorityRank(priority saga.Priority) int {
	switch priority {
	case saga.PriorityHigh:
		return 0
	case saga.PriorityLow:
		return 2
	default:
		return 1
	}
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self'; script-src 'self'; img-src 'self' data:; connect-src 'self'; font-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

// ProjectName derives a compact project label from a local store path.
func ProjectName(st *store.Store) string {
	if st.HasLocal() {
		return filepath.Base(filepath.Dir(filepath.Dir(st.LocalPath())))
	}
	return "global"
}

// ListenAddress normalizes a host:port value and prevents accidental public binding.
func ListenAddress(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "127.0.0.1:7331", nil
	}
	if strings.HasPrefix(value, ":") {
		return "127.0.0.1" + value, nil
	}
	if strings.HasPrefix(value, "localhost:") {
		return "127.0.0.1:" + strings.TrimPrefix(value, "localhost:"), nil
	}
	if !strings.HasPrefix(value, "127.0.0.1:") && !strings.HasPrefix(value, "[::1]:") {
		return "", fmt.Errorf("web address must bind to localhost (for example 127.0.0.1:7331)")
	}
	return value, nil
}
