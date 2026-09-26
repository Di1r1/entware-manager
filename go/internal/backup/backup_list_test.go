package backup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupBackupTree создаёт типовой набор конфигов в отдельном webRoot.
func setupBackupTree(t *testing.T) {
	t.Helper()
	old := webRoot
	webRoot = filepath.Join(t.TempDir(), "web")
	t.Cleanup(func() { webRoot = old })

	os.MkdirAll(filepath.Join(webRoot, "bridge"), 0755)
	os.MkdirAll(filepath.Join(webRoot, "logger"), 0755)
	os.WriteFile(filepath.Join(webRoot, "links.json"), []byte(`{"links":[]}`), 0644)
	os.WriteFile(filepath.Join(webRoot, "logger", "config.json"), []byte(`{"enabled":true}`), 0644)
	os.WriteFile(filepath.Join(webRoot, "logger", "system_sources.json"), []byte(`[]`), 0644)
	os.WriteFile(filepath.Join(webRoot, "auth_config.json"), []byte(`{"enabled":true}`), 0600)
	os.WriteFile(filepath.Join(webRoot, "telegram_config.json"), []byte(`{"token":"t"}`), 0600)
	os.WriteFile(filepath.Join(webRoot, "version.json"), []byte(`{"version":"test"}`), 0644)
	os.WriteFile(filepath.Join(webRoot, "bridge", "myapp.json"), []byte(`{"id":"myapp"}`), 0644)
	os.WriteFile(filepath.Join(webRoot, "bridge", "myapp.auth.json"), []byte(`{}`), 0600)
	os.WriteFile(filepath.Join(webRoot, "bridge", "_prefs.json"), []byte(`{}`), 0600)
	os.WriteFile(filepath.Join(webRoot, "bridge", "broken.tmp"), []byte("x"), 0644)
}

// listResponse — разбор JSON из backup.cgi?list=1.
type listResponse struct {
	Filename   string `json:"filename"`
	Count      int    `json:"count"`
	TotalBytes int    `json:"total_bytes"`
	Entries    []struct {
		Name  string `json:"name"`
		Bytes int    `json:"bytes"`
	} `json:"entries"`
}

func runList(t *testing.T) listResponse {
	t.Helper()
	t.Setenv("REQUEST_METHOD", "GET")
	t.Setenv("QUERY_STRING", "list=1")
	var out bytes.Buffer
	captureStdout(t, HandleCreate, &out)
	s := out.String()
	i := strings.Index(s, "\n\n")
	if i < 0 {
		t.Fatalf("нет JSON в ответе: %s", s)
	}
	var r listResponse
	if err := json.Unmarshal([]byte(s[i+2:]), &r); err != nil {
		t.Fatalf("невалидный JSON: %v (%s)", err, s[i+2:])
	}
	return r
}

// G3: список — чтение, поэтому только GET.
func TestHandleListRejectsNonGET(t *testing.T) {
	setupBackupTree(t)
	t.Setenv("REQUEST_METHOD", "POST")
	t.Setenv("QUERY_STRING", "list=1")
	var out bytes.Buffer
	captureStdout(t, HandleCreate, &out)
	if strings.Contains(out.String(), `"filename"`) {
		t.Errorf("POST не должен отдавать список: %s", out.String())
	}
	if !strings.Contains(out.String(), "Method not allowed") {
		t.Errorf("ожидался Method not allowed, получено: %s", out.String())
	}
}

// backup.cgi?list=1 отдаёт имя файла и список записей.
func TestHandleListReturnsEntries(t *testing.T) {
	setupBackupTree(t)
	r := runList(t)

	if r.Filename != backupFileName {
		t.Errorf("filename = %q, want %q", r.Filename, backupFileName)
	}
	if r.Count != len(r.Entries) {
		t.Errorf("count = %d, записей %d", r.Count, len(r.Entries))
	}
	want := map[string]bool{
		"links.json": true, "logger_config.json": true, "logger_system_sources.json": true,
		"auth_config.json": true, "telegram_config.json": true, "version.json": true,
		"backup.json": true, "bridge_myapp.json": true, "bridge_myapp.auth.json": true,
		"bridge__prefs.json": true,
	}
	got := map[string]int{}
	for _, e := range r.Entries {
		got[e.Name] = e.Bytes
	}
	for n := range want {
		if _, ok := got[n]; !ok {
			t.Errorf("в списке нет %s (есть: %v)", n, got)
		}
	}
	if _, ok := got["broken.tmp"]; ok {
		t.Error(".tmp не должен попадать в список")
	}
	// Размеры соответствуют содержимому файлов.
	if got["version.json"] != len(`{"version":"test"}`) {
		t.Errorf("version.json: bytes = %d, want %d", got["version.json"], len(`{"version":"test"}`))
	}
	if r.TotalBytes <= 0 {
		t.Error("total_bytes должен быть больше нуля")
	}
}

// Главное свойство: список в панели и реальный архив строятся из одного
// кода, поэтому разойтись не могут.
func TestListMatchesArchive(t *testing.T) {
	setupBackupTree(t)

	data, err := BuildArchive()
	if err != nil {
		t.Fatal(err)
	}
	var archiveNames []string
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(gr)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		archiveNames = append(archiveNames, h.Name)
	}

	listed := runList(t)
	var listNames []string
	for _, e := range listed.Entries {
		listNames = append(listNames, e.Name)
	}

	if len(archiveNames) != len(listNames) {
		t.Fatalf("в архиве %d записей, в списке %d\nархив: %v\nсписок: %v",
			len(archiveNames), len(listNames), archiveNames, listNames)
	}
	for i := range archiveNames {
		if archiveNames[i] != listNames[i] {
			t.Errorf("запись %d: архив %q, список %q", i, archiveNames[i], listNames[i])
		}
	}
}

// Без ?list=1 эндпоинт по-прежнему отдаёт архив, а не JSON.
func TestHandleCreateStillServesArchive(t *testing.T) {
	setupBackupTree(t)
	t.Setenv("QUERY_STRING", "")
	var out bytes.Buffer
	captureStdout(t, HandleCreate, &out)
	s := out.String()
	if !strings.Contains(s, "Content-type: application/gzip") {
		t.Fatalf("ожидался архив, получено: %s", firstLines(s, 3))
	}
	if !strings.Contains(s, "filename="+backupFileName) {
		t.Errorf("имя файла в заголовке не согласовано: %s", firstLines(s, 4))
	}

	// Тело ответа после заголовков — gzip-tar: распаковываем и ищем version.json.
	i := strings.Index(s, "\n\n")
	if i < 0 {
		t.Fatal("нет заголовков CGI")
	}
	gr, err := gzip.NewReader(strings.NewReader(s[i+2:]))
	if err != nil {
		t.Fatalf("ответ не gzip: %v", err)
	}
	tr := tar.NewReader(gr)
	found := false
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("битый tar: %v", err)
		}
		if h.Name == "version.json" {
			data, _ := io.ReadAll(tr)
			if !strings.Contains(string(data), `"version":"test"`) {
				t.Errorf("содержимое version.json не то: %q", string(data))
			}
			found = true
		}
	}
	if !found {
		t.Error("version.json нет в архиве")
	}
}

func firstLines(s string, n int) string {
	parts := strings.Split(s, "\n")
	if len(parts) > n {
		parts = parts[:n]
	}
	return strings.Join(parts, " | ")
}
