package stats

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindCronPIDIn(t *testing.T) {
	root := t.TempDir()
	mkProc := func(name, cmdline string) {
		dir := filepath.Join(root, name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if cmdline != "" {
			if err := os.WriteFile(filepath.Join(dir, "cmdline"), []byte(cmdline), 0644); err != nil {
				t.Fatal(err)
			}
		}
	}

	mkProc("1", "\x00")
	mkProc("100", "crontab_update.cgi")
	mkProc("200", "sh -c curl crontab_update.cgi")
	mkProc("500", "/opt/sbin/crond\x00-f")
	mkProc("999", "someother")

	if got := findCronPIDIn(root); got != 500 {
		t.Fatalf("expected PID 500 (crond), got %d", got)
	}
}
