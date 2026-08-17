package packages

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpkgUpdateRunning(t *testing.T) {
	root := t.TempDir()
	pidFile := filepath.Join(root, "opkg_update.pid")
	procDir := filepath.Join(root, "proc")
	mkProc := func(pid, cmdline string) {
		dir := filepath.Join(procDir, pid)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if cmdline != "" {
			if err := os.WriteFile(filepath.Join(dir, "cmdline"), []byte(cmdline), 0644); err != nil {
				t.Fatal(err)
			}
		}
	}

	if opkgUpdateRunningIn(procDir, pidFile) {
		t.Fatal("no pidfile should be not running")
	}

	mkProc("100", "/opt/web_entware/cgi-bin/go/entware-pkg")
	os.WriteFile(pidFile, []byte("100"), 0644)
	if !opkgUpdateRunningIn(procDir, pidFile) {
		t.Fatal("expected running for live entware-pkg")
	}

	mkProc("100", "/opt/bin/something-else")
	if opkgUpdateRunningIn(procDir, pidFile) {
		t.Fatal("different binary with same pid should be not running")
	}

	mkProc("200", "/opt/web_entware/cgi-bin/go/entware-pkg")
	os.WriteFile(pidFile, []byte("200"), 0644)
	os.RemoveAll(filepath.Join(procDir, "200"))
	if opkgUpdateRunningIn(procDir, pidFile) {
		t.Fatal("dead pid should be not running")
	}
	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Fatal("stale pidfile should be removed")
	}
}
