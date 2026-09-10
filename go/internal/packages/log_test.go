package packages

import (
	"os"
	"path/filepath"
	"testing"
)

// CollectPackageLog: новые сверху, пусто при отсутствии файла.
func TestCollectPackageLog(t *testing.T) {
	dir := t.TempDir()
	old := PackageChangesLog
	PackageChangesLog = filepath.Join(dir, "package_changes.log")
	defer func() { PackageChangesLog = old }()

	data := "2026-08-28 19:44:20 | remove | 7-zip | success\n" +
		"2026-08-28 19:45:01 | install | htop | success\n" +
		"2026-08-28 19:46:12 | upgrade | curl | failed\n"
	if err := os.WriteFile(PackageChangesLog, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	entries := CollectPackageLog(50)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[0].Package != "curl" || entries[0].Action != "upgrade" || entries[0].Status != "failed" {
		t.Errorf("newest first broken, got %+v", entries[0])
	}

	PackageChangesLog = filepath.Join(dir, "missing.log")
	if got := CollectPackageLog(50); len(got) != 0 {
		t.Errorf("expected empty slice for missing file, got %d", len(got))
	}
}
