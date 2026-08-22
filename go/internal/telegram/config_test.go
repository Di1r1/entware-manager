package telegram

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRedactToken(t *testing.T) {
	if RedactToken("") != "" {
		t.Error("empty token should return empty")
	}
	if RedactToken("abc") != "***" {
		t.Error("short token should be fully masked")
	}
	r := RedactToken("1234567890")
	if r == "1234567890" {
		t.Error("token must not be returned in full")
	}
	if r != "123…890" {
		t.Errorf("masked token should be '123…890', got %q", r)
	}
}

func TestIsValidChatID(t *testing.T) {
	if !IsValidChatID("123456789") {
		t.Error("numeric chat_id should be valid")
	}
	if !IsValidChatID("-1001234567890") {
		t.Error("negative supergroup chat_id should be valid")
	}
	if IsValidChatID("123-456") {
		t.Error("non-numeric chat_id should be invalid")
	}
	if IsValidChatID("") {
		t.Error("empty chat_id should be invalid")
	}
	if IsValidChatID("-") {
		t.Error("lone minus should be invalid")
	}
}

func TestIsValidLevel(t *testing.T) {
	for _, l := range []string{"ERROR", "WARN", "INFO", "OFF"} {
		if !IsValidLevel(l) {
			t.Errorf("%s should be valid level", l)
		}
	}
	if IsValidLevel("DEBUG") {
		t.Error("DEBUG should be invalid")
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	orig := os.Getenv(WebRootEnv)
	dir := t.TempDir()
	os.Setenv(WebRootEnv, dir)
	defer os.Setenv(WebRootEnv, orig)

	cfg := LoadConfig()
	if cfg.Enabled {
		t.Error("default should be disabled")
	}
	if cfg.Level != "ERROR" {
		t.Errorf("default level should be ERROR, got %q", cfg.Level)
	}
	if cfg.Configured {
		t.Error("default should not be configured (no token)")
	}
}

func TestSaveLoadConfigSecretNotLeaked(t *testing.T) {
	orig := os.Getenv(WebRootEnv)
	dir := t.TempDir()
	os.Setenv(WebRootEnv, dir)
	defer os.Setenv(WebRootEnv, orig)

	cfg := DefaultConfig()
	cfg.BotToken = "SECRET:TOKEN"
	cfg.ChatID = "123456789"
	cfg.Enabled = true
	if err := SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}

	// Файл 0600
	fi, err := os.Stat(filepath.Join(dir, ConfigName))
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0600 {
		t.Errorf("config should be 0600, got %v", fi.Mode().Perm())
	}

	loaded := LoadConfig()
	if loaded.BotToken != "SECRET:TOKEN" {
		t.Error("token should be loaded internally")
	}
	if !loaded.Configured {
		t.Error("should be configured when token set")
	}
	if loaded.ChatID != "123456789" {
		t.Errorf("chat_id mismatch: %q", loaded.ChatID)
	}
}

func TestDefaultThresholds(t *testing.T) {
	cfg := DefaultConfig()
	th := cfg.Thresholds
	if !th.CPUTemp.Enabled || th.CPUTemp.Value != 90 {
		t.Errorf("cpu_temp default wrong: %+v", th.CPUTemp)
	}
	if !th.CPULoad.Enabled || th.CPULoad.Value != 95 {
		t.Errorf("cpu_load default wrong: %+v", th.CPULoad)
	}
	if th.DiskTemp.Enabled || th.DiskTemp.Value != 60 {
		t.Errorf("disk_temp default wrong: %+v", th.DiskTemp)
	}
}

func TestFillDefaultsOldConfigCompat(t *testing.T) {
	// Старый конфиг без поля thresholds → после загрузки дефолты применятся.
	orig := os.Getenv(WebRootEnv)
	dir := t.TempDir()
	os.Setenv(WebRootEnv, dir)
	defer os.Setenv(WebRootEnv, orig)

	os.WriteFile(filepath.Join(dir, ConfigName), []byte(`{"enabled":true,"bot_token":"T","chat_id":"1","level":"INFO","sources":["system"],"proxy_url":"http://127.0.0.1:10871"}`), 0600)

	cfg := LoadConfig()
	if !cfg.Thresholds.CPUTemp.Enabled || cfg.Thresholds.CPUTemp.Value != 90 {
		t.Errorf("old config should get default cpu_temp, got %+v", cfg.Thresholds.CPUTemp)
	}
}

func TestFillDefaultsPreservesUserChoice(t *testing.T) {
	// Пользователь отключил метрику (value!=0, enabled=false) — не перезаписывать дефолтом.
	cfg := DefaultConfig()
	cfg.Thresholds.CPUTemp = Threshold{Enabled: false, Value: 88}
	cfg.fillDefaults()
	if cfg.Thresholds.CPUTemp.Enabled {
		t.Error("user-disabled metric should stay disabled")
	}
	if cfg.Thresholds.CPUTemp.Value != 88 {
		t.Errorf("user-set value should stay, got %d", cfg.Thresholds.CPUTemp.Value)
	}
}

func TestValidateThresholds(t *testing.T) {
	th := DefaultConfig().Thresholds
	if !ValidateThresholds(th) {
		t.Error("default thresholds should be valid")
	}
	bad := th
	bad.CPUTemp.Value = 999
	if ValidateThresholds(bad) {
		t.Error("cpu_temp 999 should be invalid")
	}
	bad2 := th
	bad2.RAMUsed.Value = 101
	if ValidateThresholds(bad2) {
		t.Error("ram_used 101 should be invalid")
	}
}

// POST с пустым chat_id не должен стирать сохранённый chat_id
// (GET ранее не отдавал chat_id — пересохранение настроек теряло его).
func TestHandleConfigPostKeepsEmptyChatID(t *testing.T) {
	origWebRoot := os.Getenv(WebRootEnv)
	dir := t.TempDir()
	os.Setenv(WebRootEnv, dir)
	defer os.Setenv(WebRootEnv, origWebRoot)

	cfg := DefaultConfig()
	cfg.BotToken = "SECRET:TOKEN"
	cfg.ChatID = "241544715"
	if err := SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}

	// CGI-окружение: POST с ПУСТЫМ chat_id.
	os.Setenv("REQUEST_METHOD", "POST")
	os.Unsetenv("HTTP_ORIGIN")
	os.Unsetenv("HTTP_HOST")
	defer os.Unsetenv("REQUEST_METHOD")

	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.WriteString("enabled=true&autostart=true&chat_id=&level=INFO"); err != nil {
		t.Fatal(err)
	}
	w.Close()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	oldStdout := os.Stdout
	or, ow, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = ow

	handleConfigPost()

	ow.Close()
	os.Stdout = oldStdout
	out, _ := io.ReadAll(or)
	or.Close()

	if !strings.Contains(string(out), `"status":"ok"`) {
		t.Fatalf("POST должен пройти успешно, stdout: %s", string(out))
	}
	loaded := LoadConfig()
	if loaded.ChatID != "241544715" {
		t.Errorf("пустой chat_id затёр сохранённое значение: got %q", loaded.ChatID)
	}
}
