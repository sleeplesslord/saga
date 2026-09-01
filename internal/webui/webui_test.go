package webui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sleeplesslord/saga/internal/saga"
	"github.com/sleeplesslord/saga/internal/store"
)

func TestBuildSnapshotComputesRoutes(t *testing.T) {
	now := time.Now()
	done := &saga.Saga{ID: "done", Title: "Finished prerequisite", Status: saga.StatusDone, UpdatedAt: now}
	working := &saga.Saga{ID: "work", Title: "Claimed prerequisite", Status: saga.StatusActive, ClaimedBy: "agent@123", ClaimedAt: now, UpdatedAt: now}
	blocked := &saga.Saga{ID: "blocked", Title: "Blocked task", Status: saga.StatusActive, DependsOn: []string{"work"}, UpdatedAt: now}
	ready := &saga.Saga{ID: "ready", Title: "Ready task", Status: saga.StatusActive, DependsOn: []string{"done"}, UpdatedAt: now}
	parent := &saga.Saga{ID: "parent", Title: "Parent", Status: saga.StatusActive, UpdatedAt: now}
	child := &saga.Saga{ID: "parent.1", ParentID: "parent", Title: "Child", Status: saga.StatusActive, UpdatedAt: now}

	got := buildSnapshot([]*saga.Saga{done, working, blocked, ready, parent, child}, 24*time.Hour, Options{ProjectName: "example", Scope: store.ScopeLocal})
	if got.Project != "example" || got.Scope != "project" || got.Total != 6 {
		t.Fatalf("unexpected snapshot identity: %#v", got)
	}
	if got.Counts.Ready != 2 || got.Counts.Blocked != 2 || got.Counts.Claimed != 1 {
		t.Fatalf("unexpected counts: %#v", got.Counts)
	}

	byID := make(map[string]task)
	for _, item := range got.Sagas {
		byID[item.ID] = item
	}
	if !byID["blocked"].Blocked || len(byID["blocked"].BlockingDependencies) != 1 {
		t.Fatalf("blocked task did not expose its blocker: %#v", byID["blocked"])
	}
	if !byID["ready"].Ready {
		t.Fatalf("completed dependency should leave task ready: %#v", byID["ready"])
	}
	if !byID["parent"].Blocked {
		t.Fatalf("active child should block parent: %#v", byID["parent"])
	}
	if byID["parent.1"].Depth != 1 {
		t.Fatalf("child depth = %d, want 1", byID["parent.1"].Depth)
	}
}

func TestHandlerServesEmbeddedAppAndAPI(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "sagas.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Save(&saga.Saga{ID: "abc123", Title: "Test route", Description: "Use **Markdown** safely.", Plan: "# Steps\n\n1. Render it", Status: saga.StatusActive, Priority: saga.PriorityHigh, CreatedAt: time.Now(), UpdatedAt: time.Now()}, store.ScopeGlobal); err != nil {
		t.Fatal(err)
	}
	handler, err := Handler(st, Options{ProjectName: "test-project", Scope: store.ScopeGlobal})
	if err != nil {
		t.Fatal(err)
	}

	index := httptest.NewRecorder()
	handler.ServeHTTP(index, httptest.NewRequest(http.MethodGet, "/", nil))
	if index.Code != http.StatusOK || index.Header().Get("Content-Security-Policy") == "" {
		t.Fatalf("index response status=%d headers=%v", index.Code, index.Header())
	}
	if !strings.Contains(index.Body.String(), `<script src="/assets/markdown.js"></script>`) {
		t.Fatal("index does not load the dependency-free Markdown renderer")
	}

	markdown := httptest.NewRecorder()
	handler.ServeHTTP(markdown, httptest.NewRequest(http.MethodGet, "/assets/markdown.js", nil))
	if markdown.Code != http.StatusOK || !strings.Contains(markdown.Body.String(), "SagaMarkdown") {
		t.Fatalf("Markdown asset status=%d body=%q", markdown.Code, markdown.Body.String())
	}

	api := httptest.NewRecorder()
	handler.ServeHTTP(api, httptest.NewRequest(http.MethodGet, "/api/sagas", nil))
	if api.Code != http.StatusOK {
		t.Fatalf("api status=%d body=%s", api.Code, api.Body.String())
	}
	var response snapshot
	if err := json.Unmarshal(api.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Total != 1 || len(response.Sagas) != 1 || response.Sagas[0].ID != "abc123" {
		t.Fatalf("unexpected API response: %#v", response)
	}
	if response.Sagas[0].Description != "Use **Markdown** safely." || !strings.HasPrefix(response.Sagas[0].Plan, "# Steps") {
		t.Fatalf("Markdown source was not preserved in API response: %#v", response.Sagas[0])
	}

	post := httptest.NewRecorder()
	handler.ServeHTTP(post, httptest.NewRequest(http.MethodPost, "/api/sagas", nil))
	if post.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST status=%d, want %d", post.Code, http.StatusMethodNotAllowed)
	}
}

func TestAppAssetSyncsNavigationWithHistory(t *testing.T) {
	app, err := files.ReadFile("assets/app.js")
	if err != nil {
		t.Fatalf("reading embedded app.js: %v", err)
	}
	script := string(app)
	// Selecting a saga pushes its hash, so browser back/forward change the URL.
	// The app must also listen for hashchange and re-render from the URL,
	// otherwise the address bar and the displayed saga drift apart.
	for _, fragment := range []string{
		"addEventListener('hashchange',syncFromHash)",
		"function syncFromHash()",
		"location.hash=encodeURIComponent(id)",
	} {
		if !strings.Contains(script, fragment) {
			t.Fatalf("app.js is missing history-navigation wiring %q", fragment)
		}
	}
}

func TestListenAddressRejectsPublicBindings(t *testing.T) {
	tests := []struct {
		input string
		want  string
		ok    bool
	}{
		{"", "127.0.0.1:7331", true},
		{":9000", "127.0.0.1:9000", true},
		{"localhost:8123", "127.0.0.1:8123", true},
		{"127.0.0.1:0", "127.0.0.1:0", true},
		{"0.0.0.0:7331", "", false},
		{"192.168.1.5:7331", "", false},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			got, err := ListenAddress(test.input)
			if (err == nil) != test.ok || got != test.want {
				t.Fatalf("ListenAddress(%q) = %q, %v", test.input, got, err)
			}
		})
	}
}
