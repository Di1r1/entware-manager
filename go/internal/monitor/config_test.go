package monitor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// useConfigFile подменяет monitorConfigFile на временный путь.
func useConfigFile(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "monitor_config.json")
	old := monitorConfigFile
	monitorConfigFile = p
	t.Cleanup(func() { monitorConfigFile = old })
	return p
}

// getConfig вызывает HandleConfig методом GET.
func getConfig(t *testing.T) string {
	t.Helper()
	t.Setenv("REQUEST_METHOD", "GET")
	return captureMonitorOut(t, HandleConfig)
}

// postConfig вызывает HandleConfig методом POST с JSON-телом.
func postConfig(t *testing.T, body string) string {
	t.Helper()
	t.Setenv("REQUEST_METHOD", "POST")
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	w.WriteString(body)
	w.Close()
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })
	return captureMonitorOut(t, HandleConfig)
}

// Регрессия G3: чтение не должно писать на диск.
// Файл без новых ключей + GET → в ответе дефолты, файл не тронут.
func TestConfigGETDoesNotWrite(t *testing.T) {
	p := useConfigFile(t)
	before := `{"enabled":true,"interval":30}`
	if err := os.WriteFile(p, []byte(before), 0644); err != nil {
		t.Fatal(err)
	}
	stBefore, _ := os.Stat(p)

	out := getConfig(t)
	if !strings.Contains(out, `"max_processes":200`) {
		t.Fatalf("в ответе нет дефолта max_processes: %q", out)
	}
	if !strings.Contains(out, `"autostart":false`) {
		t.Fatalf("в ответе нет дефолта autostart: %q", out)
	}

	after, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != before {
		t.Fatalf("GET изменил файл на диске:\nбыло: %s\nстало: %s", before, after)
	}
	stAfter, _ := os.Stat(p)
	if !stAfter.ModTime().Equal(stBefore.ModTime()) {
		t.Fatal("GET обновил mtime файла")
	}
}

// POST полного объекта сохраняет значения.
func TestConfigPOSTWrites(t *testing.T) {
	p := useConfigFile(t)
	out := postConfig(t, `{"enabled":true,"interval":15,"max_processes":100,"autostart":true}`)
	if !strings.Contains(out, `"status":"ok"`) {
		t.Fatalf("POST должен сохранить, получено: %q", out)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"max_processes": 100`, `"autostart": true`, `"interval": 15`} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("в файле нет %s: %s", want, data)
		}
	}
}

// POST не сносит бэкенд-ключи, которых нет в объекте фронта
// (log_file/log_max_size сегодня никто не потребляет, но сносить их нельзя).
func TestConfigPOSTPreservesBackendKeys(t *testing.T) {
	p := useConfigFile(t)
	if err := os.WriteFile(p, []byte(`{"enabled":true,"log_file":"/custom/monitor.log"}`), 0644); err != nil {
		t.Fatal(err)
	}
	out := postConfig(t, `{"enabled":false,"interval":10,"max_processes":200,"autostart":false}`)
	if !strings.Contains(out, `"status":"ok"`) {
		t.Fatalf("POST должен сохранить, получено: %q", out)
	}
	data, _ := os.ReadFile(p)
	if !strings.Contains(string(data), `"/custom/monitor.log"`) {
		t.Fatalf("POST снёс бэкенд-ключ log_file: %s", data)
	}
	if !strings.Contains(string(data), `"enabled": false`) {
		t.Fatalf("POST не применил новые значения: %s", data)
	}
}
