package telegram

import (
	"os"
	"path/filepath"
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
