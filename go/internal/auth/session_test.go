package auth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSessionLifecycle(t *testing.T) {
	orig := SessionFile
	defer func() { SessionFile = orig }()
	SessionFile = filepath.Join(t.TempDir(), "panel_session")

	// нет сессии → невалидна
	if SessionValid() {
		t.Error("SessionValid() = true без сессии")
	}

	token, err := CreateSession()
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if token == "" {
		t.Fatal("empty token")
	}
	if len(token) != 64 { // 32 байта hex
		t.Errorf("token length = %d, want 64", len(token))
	}

	restore := setEnv(t, "HTTP_COOKIE", SessionCookieName+"="+token)
	defer restore()
	if !SessionValid() {
		t.Error("SessionValid() = false с корректным cookie")
	}

	// подделка токена → невалидна
	restore = setEnv(t, "HTTP_COOKIE", SessionCookieName+"=ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	defer restore()
	if SessionValid() {
		t.Error("SessionValid() = true с подделанным токеном")
	}

	// выход → невалидна
	restore = setEnv(t, "HTTP_COOKIE", SessionCookieName+"="+token)
	defer restore()
	DestroySession()
	if SessionValid() {
		t.Error("SessionValid() = true после DestroySession")
	}
}

func TestSessionTokenFromCookie(t *testing.T) {
	restore := setEnv(t, "HTTP_COOKIE", "a=1; "+SessionCookieName+"=abc123; b=2")
	defer restore()
	if got := SessionTokenFromCookie(); got != "abc123" {
		t.Errorf("SessionTokenFromCookie() = %q, want abc123", got)
	}

	restore = setEnv(t, "HTTP_COOKIE", "")
	defer restore()
	if got := SessionTokenFromCookie(); got != "" {
		t.Errorf("SessionTokenFromCookie() = %q с пустой cookie, want empty", got)
	}
}

func TestTokenFromHeader(t *testing.T) {
	if got := TokenFromHeader("a=1; panel_session=xyz; b=2"); got != "xyz" {
		t.Errorf("TokenFromHeader() = %q, want xyz", got)
	}
	if got := TokenFromHeader(""); got != "" {
		t.Errorf("TokenFromHeader() = %q с пустым, want empty", got)
	}
}

func TestSessionValidCookie(t *testing.T) {
	orig := SessionFile
	defer func() { SessionFile = orig }()
	SessionFile = filepath.Join(t.TempDir(), "panel_session")

	tok, err := CreateSession()
	if err != nil {
		t.Fatal(err)
	}
	if !SessionValidCookie(tok) {
		t.Error("SessionValidCookie(valid) = false")
	}
	if SessionValidCookie("deadbeef") {
		t.Error("SessionValidCookie(wrong) = true")
	}
	if SessionValidCookie("") {
		t.Error("SessionValidCookie(empty) = true")
	}
}

func TestEnabled(t *testing.T) {
	orig := ConfigPath
	defer func() { ConfigPath = orig }()
	dir := t.TempDir()
	ConfigPath = filepath.Join(dir, "auth_config.json")

	// файл отсутствует → защита не настроена → панель открыта (совпадает с go.cgi)
	os.Remove(ConfigPath)
	if Enabled() {
		t.Error("Enabled() = true при отсутствии конфига, want false")
	}

	// enabled=false → панель открыта
	os.WriteFile(ConfigPath, []byte(`{"enabled":false}`), 0644)
	if Enabled() {
		t.Error("Enabled() = true при enabled=false, want false")
	}

	// enabled=true с hash → защита включена
	h := SHA256Hex("secret")
	os.WriteFile(ConfigPath, []byte(`{"enabled":true,"password_hash":"`+h+`"}`), 0644)
	if !Enabled() {
		t.Error("Enabled() = false при enabled=true, want true")
	}
}
