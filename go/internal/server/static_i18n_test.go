package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStaticWhitelistI18nAssets(t *testing.T) {
	oldRoot := webRoot
	webRoot = t.TempDir()
	t.Cleanup(func() { webRoot = oldRoot })

	files := map[string]string{
		"i18n.js":         "window.I18n = {};",
		"locales/ru.json": `{"Статистика":"Статистика"}`,
		"locales/en.json": `{"Статистика":"Statistics"}`,
		"locales/tr.json": `{"Статистика":"İstatistikler"}`,
	}
	for name, content := range files {
		full := filepath.Join(webRoot, name)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	for _, p := range []string{
		"/entware-manager/i18n.js",
		"/entware-manager/locales/ru.json",
		"/entware-manager/locales/en.json",
		"/entware-manager/locales/tr.json",
	} {
		req := httptest.NewRequest(http.MethodGet, p, nil)
		rec := httptest.NewRecorder()
		handleStatic(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", p, rec.Code)
		}
		if got := rec.Header().Get("Cache-Control"); !strings.Contains(got, "no-store") {
			t.Errorf("GET %s Cache-Control = %q, want no-store", p, got)
		}
	}

	// Keep the whitelist explicit: arbitrary locale files must not become public.
	req := httptest.NewRequest(http.MethodGet, "/entware-manager/locales/secret.json", nil)
	rec := httptest.NewRecorder()
	handleStatic(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("GET unknown locale = %d, want 404", rec.Code)
	}
}
