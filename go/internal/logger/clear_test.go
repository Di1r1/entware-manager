package logger

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// setLogDir подменяет каталог логов на временный (в проде /opt/var/log/entware).
func setLogDir(t *testing.T, dir string) {
	t.Helper()
	old := logDir
	logDir = dir
	t.Cleanup(func() { logDir = old })
}

// captureOut перехватывает stdout CGI-ответа.
func captureOut(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stdout = old
	return buf.String()
}

func touchLog(t *testing.T, dir, name string, age time.Duration) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("log line\n"), 0644); err != nil {
		t.Fatal(err)
	}
	mt := time.Now().Add(-age)
	if err := os.Chtimes(p, mt, mt); err != nil {
		t.Fatal(err)
	}
	return p
}

// Регрессия: ошибки os.Remove игнорировались, ответ всегда был
// «Логи старше 30 дней удалены» — даже если ничего не удалилось.
func TestClearLogsSuccess(t *testing.T) {
	dir := t.TempDir()
	setLogDir(t, dir)

	old := touchLog(t, dir, "old.log", 40*24*time.Hour)
	fresh := touchLog(t, dir, "fresh.log", time.Hour)
	other := touchLog(t, dir, "notes.txt", 40*24*time.Hour)

	t.Setenv("REQUEST_METHOD", "POST")
	out := captureOut(t, HandleClear)

	if !strings.Contains(out, `"status":"ok"`) {
		t.Fatalf("ожидался ok, получено: %s", out)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Errorf("старый лог не удалён: %v", err)
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Errorf("свежий лог не должен удаляться: %v", err)
	}
	if _, err := os.Stat(other); err != nil {
		t.Errorf("файл не .log не должен удаляться: %v", err)
	}
}

// Регрессия: при неудачном удалении (read-only каталог) обработчик
// не должен рапортовать об успехе.
func TestClearLogsReportsFailure(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root игнорирует права на каталоге — сценарий не воспроизводится")
	}
	dir := t.TempDir()
	setLogDir(t, dir)

	old := touchLog(t, dir, "locked.log", 40*24*time.Hour)
	if err := os.Chmod(dir, 0555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0755) })

	t.Setenv("REQUEST_METHOD", "POST")
	out := captureOut(t, HandleClear)

	if strings.Contains(out, `"status":"ok"`) {
		t.Fatalf("при ошибке удаления ответ не должен быть ok: %s", out)
	}
	if !strings.Contains(out, `"failed":1`) {
		t.Errorf("ожидался счётчик failed=1, получено: %s", out)
	}
	if _, err := os.Stat(old); err != nil {
		t.Errorf("лог не должен был исчезнуть при read-only: %v", err)
	}
}
