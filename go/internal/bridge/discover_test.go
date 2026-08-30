// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
package bridge

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestIsBuiltin(t *testing.T) {
	for _, id := range []string{"koffe", "adguard", "ttyd", "transmission", "syncthing"} {
		if !IsBuiltin(id) {
			t.Errorf("IsBuiltin(%q) = false, хочу true", id)
		}
	}
	for _, id := range []string{"xray", "koffe1", "myapp"} {
		if IsBuiltin(id) {
			t.Errorf("IsBuiltin(%q) = true, хочу false", id)
		}
	}
}

// Редиректы (3xx) — признак живого сервиса: транспорт моста не следует им
// (redirects disabled), но ответ получен → running. Регрессия для a3-заявки.
func TestClassifyRedirect(t *testing.T) {
	for _, code := range []int{300, 301, 302, 304, 307, 308, 309} {
		if st := classify(code, "text/html", []byte("<html/>")); st.State != "running" {
			t.Errorf("classify(%d) = %q, хочу running", code, st.State)
		}
	}
	if st := classify(http.StatusOK, "application/json", []byte(`{}`)); st.State != "running" {
		t.Errorf("200 → %q, хочу running", st.State)
	}
	if st := classify(http.StatusUnauthorized, "", nil); st.State != "auth_required" {
		t.Errorf("401 → %q, хочу auth_required", st.State)
	}
	if st := classify(http.StatusMethodNotAllowed, "", nil); st.State != "running" {
		t.Errorf("405 → %q, хочу running (эвристика Transmission)", st.State)
	}
	if st := classify(http.StatusNotFound, "", nil); st.State != "absent" {
		t.Errorf("404 → %q, хочу absent", st.State)
	}
}

// DiscoverState для манифестного веб-модуля зеркалит Discover: живой probe →
// running, недоступный → absent (раньше возвращал «running, раз файл есть»).
func TestDiscoverStateManifestWebProbe(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "svc1.json"), []byte(`{
		"id":"svc1","name":"Svc","base":"`+srv.URL+`",
		"probe":{"url":"/status","expect":"json"}
	}`), 0644); err != nil {
		t.Fatal(err)
	}

	if got := DiscoverState(dir, "svc1"); got != "running" {
		t.Fatalf("живой сервис → %q, хочу running", got)
	}
	srv.Close()
	if got := DiscoverState(dir, "svc1"); got != "absent" {
		t.Fatalf("упавший сервис → %q, хочу absent", got)
	}
}

// Процесс-модуль: живость только по процессу (probe игнорируется).
func TestDiscoverStateManifestProcess(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "dem.json"), []byte(`{
		"id":"dem","name":"Dem",
		"process":["__no_such_proc_ewm__"]
	}`), 0644); err != nil {
		t.Fatal(err)
	}
	if got := DiscoverState(dir, "dem"); got != "absent" {
		t.Fatalf("процесс не найден → %q, хочу absent", got)
	}
}
