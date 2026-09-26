package stats

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// postDeleteFile вызывает HandleDeleteFile с urlencoded-телом.
func postDeleteFile(t *testing.T, body string) string {
	t.Helper()
	t.Setenv("REQUEST_METHOD", "POST")
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	w.WriteString(body)
	w.Close()
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })
	out := captureCleanOut(t, HandleDeleteFile)
	if i := strings.Index(out, "\n\n"); i >= 0 {
		return out[i+2:]
	}
	return out
}

// Регрессия: файловый менеджер (delete_file.cgi) пропускал любой путь
// под /tmp и /dev/shm, поэтому через него удалялись pid-файлы демонов,
// логи и состояние менеджера в /tmp/entware — тот же класс дефекта, что
// закрыт для очистки tmpfs.
func TestAllowedDeletePathProtectsManagerDirs(t *testing.T) {
	deny := []string{
		"/tmp/entware",
		"/tmp/entware/logs",
		"/tmp/entware/logs/2026-09-25.log",
		"/tmp/entware/pid/service_watchdog.pid",
		"/tmp/entware/bridge",
		"/tmp/koffe/config.json",
		"/dev/shm/nginx/log",
	}
	allow := []string{
		"/tmp/foo",
		"/tmp/foo/bar.bin",
		"/tmp/entware-offline-20260806/x",
		"/tmp/entware-backup/config.json",
		"/dev/shm/cache.bin",
	}
	for _, p := range deny {
		if isAllowedDeletePath(p) {
			t.Errorf("isAllowedDeletePath(%q) = true, want false (защищённый каталог)", p)
		}
	}
	for _, p := range allow {
		if !isAllowedDeletePath(p) {
			t.Errorf("isAllowedDeletePath(%q) = false, want true (пользовательский путь)", p)
		}
	}
}

// Сквозной сценарий: файл внутри /tmp/entware не удаляется и остаётся на месте.
func TestDeleteFile_ProtectsManagerState(t *testing.T) {
	disableFilemgrAuth(t)
	root := t.TempDir()
	protected := filepath.Join(root, "entware", "pid")
	if err := os.MkdirAll(protected, 0755); err != nil {
		t.Fatal(err)
	}
	pidFile := filepath.Join(protected, "service_watchdog.pid")
	if err := os.WriteFile(pidFile, []byte("1234\n"), 0644); err != nil {
		t.Fatal(err)
	}

	out := postDeleteFile(t, "path="+pidFile+"&password=")
	if !strings.Contains(out, "Доступ запрещен") {
		t.Errorf("ожидался отказ, получено: %s", out)
	}
	if _, err := os.Stat(pidFile); err != nil {
		t.Errorf("pid-файл демона не должен удаляться: %v", err)
	}
}

// Регресс защиты: пользовательский файл под /tmp удаляется как раньше.
func TestDeleteFile_StillDeletesUserFile(t *testing.T) {
	disableFilemgrAuth(t)
	root := t.TempDir()
	userFile := filepath.Join(root, "mycache.bin")
	if err := os.WriteFile(userFile, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	out := postDeleteFile(t, "path="+userFile+"&password=")
	if !strings.Contains(out, `"status":"ok"`) {
		t.Fatalf("ожидался ok, получено: %s", out)
	}
	if _, err := os.Stat(userFile); !os.IsNotExist(err) {
		t.Errorf("пользовательский файл должен удаляться: %v", err)
	}
}
