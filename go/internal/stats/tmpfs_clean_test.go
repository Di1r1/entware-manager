package stats

import "testing"

func TestIsCleanablePath(t *testing.T) {
	allow := []string{"/tmp/foo", "/tmp/a/b", "/dev/shm/x", "/tmp/entware-offline-20260806"}
	deny := []string{
		"/", "/tmp", "/dev", "/dev/shm", "/opt", "/etc/passwd",
		"/tmp/../etc", "", "/opt/web_entware", "entware-offline-x",
	}
	for _, p := range allow {
		if !isCleanablePath(p) {
			t.Errorf("isCleanablePath(%q) = false, want true", p)
		}
	}
	for _, p := range deny {
		if isCleanablePath(p) {
			t.Errorf("isCleanablePath(%q) = true, want false", p)
		}
	}
}
