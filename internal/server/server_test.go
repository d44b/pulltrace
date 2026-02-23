package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/d44b/pulltrace/internal/model"
)

func newTestServer() *Server {
	return New(Config{
		HTTPAddr:    ":0",
		MetricsAddr: ":0",
		LogLevel:    "error",
		HistoryTTL:  30 * time.Minute,
	}, nil)
}

func postReport(t *testing.T, s *Server, report model.AgentReport, token string) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshaling report: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/report", bytes.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	s.handleReport(w, req)
	return w
}

// ── handleReport ─────────────────────────────────────────────────────────────

func TestHandleReport_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/report", nil)
	w := httptest.NewRecorder()
	s.handleReport(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHandleReport_InvalidJSON(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/report", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()
	s.handleReport(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleReport_EmptyNodeName(t *testing.T) {
	s := newTestServer()
	w := postReport(t, s, model.AgentReport{NodeName: ""}, "")
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty nodeName, got %d", w.Code)
	}
}

func TestHandleReport_NodeNameTooLong(t *testing.T) {
	s := newTestServer()
	long := make([]byte, 254)
	for i := range long {
		long[i] = 'a'
	}
	w := postReport(t, s, model.AgentReport{NodeName: string(long)}, "")
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for oversized nodeName, got %d", w.Code)
	}
}

func TestHandleReport_RateLimited(t *testing.T) {
	s := newTestServer()
	report := model.AgentReport{NodeName: "test-node", Timestamp: time.Now()}

	w := postReport(t, s, report, "")
	if w.Code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", w.Code)
	}

	w = postReport(t, s, report, "")
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("immediate second request: expected 429, got %d", w.Code)
	}
}

func TestHandleReport_TokenAuth(t *testing.T) {
	s := newTestServer()
	s.config.AgentToken = "secret"
	report := model.AgentReport{NodeName: "node1", Timestamp: time.Now()}

	if w := postReport(t, s, report, ""); w.Code != http.StatusUnauthorized {
		t.Errorf("no token: expected 401, got %d", w.Code)
	}
	if w := postReport(t, s, report, "wrong"); w.Code != http.StatusUnauthorized {
		t.Errorf("wrong token: expected 401, got %d", w.Code)
	}
	if w := postReport(t, s, report, "secret"); w.Code != http.StatusOK {
		t.Errorf("correct token: expected 200, got %d", w.Code)
	}
}

func TestHandleReport_TokenDisabled(t *testing.T) {
	s := newTestServer() // no token configured
	report := model.AgentReport{NodeName: "node1", Timestamp: time.Now()}
	// Any request without a token should be accepted
	if w := postReport(t, s, report, ""); w.Code != http.StatusOK {
		t.Errorf("expected 200 when token auth is disabled, got %d", w.Code)
	}
}

// ── handlePulls ───────────────────────────────────────────────────────────────

func TestHandlePulls_MethodNotAllowed(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pulls", nil)
	w := httptest.NewRecorder()
	s.handlePulls(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHandlePulls_Empty(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/pulls", nil)
	w := httptest.NewRecorder()
	s.handlePulls(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp model.APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if len(resp.Pulls) != 0 {
		t.Errorf("expected 0 pulls, got %d", len(resp.Pulls))
	}
}

// ── processReport ─────────────────────────────────────────────────────────────

func TestProcessReport_TracksNewPull(t *testing.T) {
	s := newTestServer()
	s.processReport(model.AgentReport{
		NodeName:  "node1",
		Timestamp: time.Now(),
		Pulls: []model.PullState{
			{
				ImageRef:   "nginx:latest",
				StartedAt:  time.Now(),
				TotalKnown: true,
				Layers: []model.LayerState{
					{Digest: "sha256:layer1", TotalBytes: 1000, DownloadedBytes: 500, TotalKnown: true},
				},
			},
		},
	})

	s.mu.RLock()
	defer s.mu.RUnlock()

	pull := s.pulls["node1:nginx:latest"]
	if pull == nil {
		t.Fatal("pull not found at key node1:nginx:latest")
	}
	if pull.TotalBytes != 1000 {
		t.Errorf("TotalBytes: want 1000, got %d", pull.TotalBytes)
	}
	if pull.DownloadedBytes != 500 {
		t.Errorf("DownloadedBytes: want 500, got %d", pull.DownloadedBytes)
	}
	if len(pull.Layers) != 1 {
		t.Errorf("Layers: want 1, got %d", len(pull.Layers))
	}
}

func TestProcessReport_CompletesAbsentPull(t *testing.T) {
	s := newTestServer()

	s.processReport(model.AgentReport{
		NodeName: "node1",
		Pulls:    []model.PullState{{ImageRef: "nginx:latest", StartedAt: time.Now()}},
	})
	s.processReport(model.AgentReport{
		NodeName: "node1",
		Pulls:    nil, // pull no longer present → completed
	})

	s.mu.RLock()
	defer s.mu.RUnlock()

	pull := s.pulls["node1:nginx:latest"]
	if pull == nil {
		t.Fatal("pull should remain in history after completion")
	}
	if pull.CompletedAt == nil {
		t.Error("pull should be marked completed")
	}
}

func TestProcessReport_MergedDigestPull(t *testing.T) {
	s := newTestServer()
	s.processReport(model.AgentReport{
		NodeName: "node1",
		Pulls: []model.PullState{
			{ImageRef: "sha256:abc", Layers: []model.LayerState{{Digest: "sha256:abc"}}},
			{ImageRef: "sha256:def", Layers: []model.LayerState{{Digest: "sha256:def"}}},
		},
	})

	s.mu.RLock()
	defer s.mu.RUnlock()

	key := "node1" + mergedPullSuffix
	pull := s.pulls[key]
	if pull == nil {
		t.Fatalf("merged pull not found at key %s", key)
	}
	if pull.ImageRef != "__pulling__" {
		t.Errorf("expected __pulling__ imageRef, got %s", pull.ImageRef)
	}
	if len(pull.Layers) != 2 {
		t.Errorf("expected 2 layers, got %d", len(pull.Layers))
	}
}

// ── mergeDigestPulls ──────────────────────────────────────────────────────────

func TestMergeDigestPulls_Empty(t *testing.T) {
	if result := mergeDigestPulls(nil); len(result) != 0 {
		t.Errorf("expected 0, got %d", len(result))
	}
}

func TestMergeDigestPulls_NoDigests(t *testing.T) {
	pulls := []model.PullState{{ImageRef: "nginx:latest"}, {ImageRef: "redis:7"}}
	result := mergeDigestPulls(pulls)
	if len(result) != 2 {
		t.Errorf("expected 2 normal pulls unchanged, got %d", len(result))
	}
}

func TestMergeDigestPulls_AllDigests(t *testing.T) {
	pulls := []model.PullState{
		{ImageRef: "sha256:aaa", Layers: []model.LayerState{{Digest: "sha256:aaa"}}},
		{ImageRef: "sha256:bbb", Layers: []model.LayerState{{Digest: "sha256:bbb"}}},
	}
	result := mergeDigestPulls(pulls)
	if len(result) != 1 {
		t.Fatalf("expected 1 merged pull, got %d", len(result))
	}
	if result[0].ImageRef != "__pulling__" {
		t.Errorf("expected __pulling__, got %s", result[0].ImageRef)
	}
	if len(result[0].Layers) != 2 {
		t.Errorf("expected 2 layers merged, got %d", len(result[0].Layers))
	}
}

func TestMergeDigestPulls_Mixed(t *testing.T) {
	pulls := []model.PullState{
		{ImageRef: "nginx:latest"},
		{ImageRef: "sha256:abc", Layers: []model.LayerState{{Digest: "sha256:abc"}}},
	}
	result := mergeDigestPulls(pulls)
	if len(result) != 2 {
		t.Fatalf("expected 2 (1 normal + 1 merged), got %d", len(result))
	}
}

// ── isContentDigest ───────────────────────────────────────────────────────────

func TestIsContentDigest(t *testing.T) {
	cases := []struct {
		ref    string
		expect bool
	}{
		{"sha256:abc", true},
		{"layer-sha256:abc", true},
		{"config-sha256:abc", true},
		{"manifest-sha256:abc", true},
		{"index-sha256:abc", true},
		{"nginx:latest", false},
		{"ghcr.io/foo/bar:v1.0", false},
		{"__pulling__", false},
		{"", false},
	}
	for _, c := range cases {
		if got := isContentDigest(c.ref); got != c.expect {
			t.Errorf("isContentDigest(%q) = %v, want %v", c.ref, got, c.expect)
		}
	}
}

// ── rateLimiter ───────────────────────────────────────────────────────────────

func TestRateLimiter_Allow(t *testing.T) {
	rl := newRateLimiter()
	if !rl.allow("node1") {
		t.Error("first request should be allowed")
	}
	if rl.allow("node1") {
		t.Error("immediate second request should be denied")
	}
}

func TestRateLimiter_DifferentNodes(t *testing.T) {
	rl := newRateLimiter()
	if !rl.allow("node1") {
		t.Error("node1 should be allowed")
	}
	if !rl.allow("node2") {
		t.Error("node2 should be allowed independently")
	}
}

func TestRateLimiter_MapFull(t *testing.T) {
	rl := newRateLimiter()
	for i := 0; i < maxRateLimitEntries; i++ {
		rl.allow(fmt.Sprintf("node-%d", i)) //nolint:errcheck
	}
	if rl.allow("brand-new-node") {
		t.Error("should reject new node when map is full")
	}
}

func TestRateLimiter_Cleanup(t *testing.T) {
	rl := newRateLimiter()
	rl.allow("node1") //nolint:errcheck
	// Backdate the entry
	rl.mu.Lock()
	rl.nodes["node1"] = time.Now().Add(-2 * time.Minute)
	rl.mu.Unlock()
	rl.cleanup()
	rl.mu.Lock()
	_, exists := rl.nodes["node1"]
	rl.mu.Unlock()
	if exists {
		t.Error("stale entry should have been cleaned up")
	}
}
