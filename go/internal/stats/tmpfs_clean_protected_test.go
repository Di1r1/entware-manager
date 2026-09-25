package stats

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// captureCleanOut перехватывает stdout CGI-ответа.
func captureCleanOut(t *testing.T, fn func()) string {
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

// disableFilemgrAuth подменяет auth_config.json на «выключенный» —
// проверяем логику очистки, а не авторизацию.
func disableFilemgrAuth(t *testing.T) {
	t.Helper()
	oldCfg, oldMarker := authConfigFile, authMarkerFile
	dir := t.TempDir()
	authConfigFile = filepath.Join(dir, "auth_config.json")
	authMarkerFile = filepath.Join(dir, ".auth_configured")
	if err := os.WriteFile(authConfigFile, []byte(`{"enabled":false}`), 0600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { authConfigFile, authMarkerFile = oldCfg, oldMarker })
}

// postClean вызывает deleteTmpClean с urlencoded-телом.
func postClean(t *testing.T, body string) string {
	t.Helper()
	t.Setenv("REQUEST_METHOD", "POST")
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	w.WriteString(body)
	w.Close()
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })
	return captureCleanOut(t, deleteTmpClean)
}

// Регрессия: защита смотрела только на basename, поэтому потомки
// защищённых каталогов (/tmp/entware/logs, /tmp/koffe/cache) проходили
// валидацию и удалялись вместе с логами и состоянием демонов.
func TestIsProtectedPath(t *testing.T) {
	deny := []string{
		"/tmp/entware",
		"/tmp/entware/logs",
		"/tmp/entware/bridge",
		"/tmp/entware/logs/2026-09-25.log",
		"/tmp/koffe",
		"/tmp/koffe/cache/data",
		"/dev/shm/nginx/logs",
	}
	allow := []string{
		"/tmp/foo",
		"/tmp/foo/bar",
		"/tmp/entware-offline-20260806",
		"/tmp/entware-data.tar.gz",
		"/dev/shm/cache.bin",
		"/tmp/koffee/x",
	}
	for _, p := range deny {
		if !isProtectedPath(p) {
			t.Errorf("isProtectedPath(%q) = false, want true", p)
		}
	}
	for _, p := range allow {
		if isProtectedPath(p) {
			t.Errorf("isProtectedPath(%q) = true, want false", p)
		}
	}
}

// Сквозной сценарий: попытка удалить потомка защищённого каталога.
func TestDeleteTmpClean_ProtectsDescendants(t *testing.T) {
	disableFilemgrAuth(t)
	root := t.TempDir()
	protected := filepath.Join(root, "entware", "logs")
	if err := os.MkdirAll(protected, 0755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(protected, "2026-09-25.log")
	if err := os.WriteFile(marker, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	out := postClean(t, "paths="+protected+"&password=")
	if !strings.Contains(out, "Доступ запрещен") {
		t.Errorf("ожидался отказ, получено: %s", out)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Errorf("лог защищённого каталога не должен удаляться: %v", err)
	}
}

// Регресс защиты: обычный пользовательский путь удаляется как раньше.
func TestDeleteTmpClean_DeletesUserPath(t *testing.T) {
	disableFilemgrAuth(t)
	root := t.TempDir()
	userDir := filepath.Join(root, "foo", "bar")
	if err := os.MkdirAll(userDir, 0755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(userDir, "cache.bin")
	if err := os.WriteFile(marker, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	out := postClean(t, "paths="+userDir+"&password=")
	if !strings.Contains(out, `"status":"ok"`) {
		t.Fatalf("ожидался ok, получено: %s", out)
	}
	if _, err := os.Stat(userDir); !os.IsNotExist(err) {
		t.Errorf("каталог не удалён: %v", err)
	}
}

// Сканирование не должно предлагать к очистке содержимое защищённых
// каталогов (иначе UI показывает их как кандидатов).
func TestScanTmpClean_HidesProtectedDescendants(t *testing.T) {
	root := t.TempDir()
	protectedRoot := filepath.Join(root, "entware")
	for _, sub := range []string{"logs", "bridge"} {
		dir := filepath.Join(protectedRoot, sub)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "big.bin"), make([]byte, 2<<20), 0644); err != nil {
			t.Fatal(err)
		}
	}
	userDir := filepath.Join(root, "userdata")
	if err := os.MkdirAll(userDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(userDir, "big.bin"), make([]byte, 2<<20), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("REQUEST_METHOD", "GET")
	t.Setenv("QUERY_STRING", "path="+protectedRoot+"&min_bytes=1048576")
	out := captureCleanOut(t, scanTmpClean)

	if strings.Contains(out, `"name":"logs"`) || strings.Contains(out, `"name":"bridge"`) {
		t.Errorf("содержимое защищённого каталога не должно предлагаться: %s", out)
	}

	// Контроль: сканирование /tmp корректного уровня показывает userdata.
	t.Setenv("QUERY_STRING", "path="+root+"&min_bytes=1048576")
	out = captureCleanOut(t, scanTmpClean)
	if !strings.Contains(out, `"name":"userdata"`) {
		t.Errorf("пользовательский каталог должен предлагаться: %s", out)
	}
	if strings.Contains(out, `"name":"entware"`) {
		t.Errorf("защищённый каталог не должен предлагаться: %s", out)
	}
}
