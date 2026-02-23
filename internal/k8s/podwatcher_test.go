package k8s

import (
	"testing"
	"time"
)

func TestNormalizeImageRef(t *testing.T) {
	cases := []struct {
		input  string
		expect string
	}{
		{"nginx", "docker.io/library/nginx:latest"},
		{"nginx:1.27", "docker.io/library/nginx:1.27"},
		{"redis:7-alpine", "docker.io/library/redis:7-alpine"},
		{"library/nginx", "docker.io/library/nginx:latest"},
		{"library/nginx:1.27", "docker.io/library/nginx:1.27"},
		{"ghcr.io/foo/bar:v1.0", "ghcr.io/foo/bar:v1.0"},
		{"ghcr.io/foo/bar", "ghcr.io/foo/bar:latest"},
		{"nginx@sha256:abc", "docker.io/library/nginx@sha256:abc"},
	}
	for _, c := range cases {
		got := normalizeImageRef(c.input)
		if got != c.expect {
			t.Errorf("normalizeImageRef(%q)\n  got  %q\n  want %q", c.input, got, c.expect)
		}
	}
}

func TestParseImageFromPullingMessage(t *testing.T) {
	cases := []struct {
		msg    string
		expect string
	}{
		{`Pulling image "nginx:1.27"`, "nginx:1.27"},
		{`Pulling image "ghcr.io/foo/bar:v1.0"`, "ghcr.io/foo/bar:v1.0"},
		{`Pulling image "redis:7-alpine"`, "redis:7-alpine"},
		{"no image here", ""},
		{`Pulling image ""`, ""},
	}
	for _, c := range cases {
		got := parseImageFromPullingMessage(c.msg)
		if got != c.expect {
			t.Errorf("parseImageFromPullingMessage(%q) = %q, want %q", c.msg, got, c.expect)
		}
	}
}

func TestParseImageFromPulledMessage(t *testing.T) {
	cases := []struct {
		msg    string
		expect string
	}{
		{`Successfully pulled image "nginx:1.27" in 5.2s`, "nginx:1.27"},
		{`Successfully pulled image "ghcr.io/foo/bar:v1.0" in 2.1s (image size: 15MB)`, "ghcr.io/foo/bar:v1.0"},
		{"no image", ""},
	}
	for _, c := range cases {
		got := parseImageFromPulledMessage(c.msg)
		if got != c.expect {
			t.Errorf("parseImageFromPulledMessage(%q) = %q, want %q", c.msg, got, c.expect)
		}
	}
}

func TestInNamespaces(t *testing.T) {
	pw := &PodWatcher{namespaces: []string{"default", "kube-system"}}
	if !pw.inNamespaces("default") {
		t.Error("default should match")
	}
	if !pw.inNamespaces("kube-system") {
		t.Error("kube-system should match")
	}
	if pw.inNamespaces("production") {
		t.Error("production should not match")
	}
}

func TestInNamespaces_Empty(t *testing.T) {
	// Empty slice means watch all namespaces; the caller skips the inNamespaces check.
	pw := &PodWatcher{namespaces: []string{}}
	// inNamespaces itself returns false for empty slice, but updatePod checks
	// len(pw.namespaces) > 0 first, so this path is never hit in practice.
	if pw.inNamespaces("any") {
		t.Error("empty slice should not match anything")
	}
}

func TestCleanupStalePulling(t *testing.T) {
	pw := &PodWatcher{
		pullingByNode: map[string]map[string]time.Time{
			"node1": {
				"nginx:latest": time.Now().Add(-2 * pullingImageTTL), // stale
				"redis:7":      time.Now(),                           // fresh
			},
		},
	}
	pw.cleanupStalePulling()

	if _, ok := pw.pullingByNode["node1"]["nginx:latest"]; ok {
		t.Error("stale entry should have been removed")
	}
	if _, ok := pw.pullingByNode["node1"]["redis:7"]; !ok {
		t.Error("fresh entry should remain")
	}
}

func TestCleanupStalePulling_RemovesEmptyNode(t *testing.T) {
	pw := &PodWatcher{
		pullingByNode: map[string]map[string]time.Time{
			"node1": {
				"nginx:latest": time.Now().Add(-2 * pullingImageTTL),
			},
		},
	}
	pw.cleanupStalePulling()

	if _, ok := pw.pullingByNode["node1"]; ok {
		t.Error("node with no remaining images should be removed")
	}
}
