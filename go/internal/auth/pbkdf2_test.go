// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Известные тест-векторы PBKDF2-HMAC-SHA256 (RFC 7914 §11 / общепринятые).
func TestPBKDF2Vectors(t *testing.T) {
	cases := []struct {
		pass, salt string
		iter       int
		want       string
	}{
		{"password", "salt", 1, "120fb6cffcf8b32c43e7225256c4f837a86548c92ccc35480805987cb70be17b"},
		{"password", "salt", 2, "ae4d0c95af6b46d32d0adff928f06dd02a303f8ef3c251dfd6e2d85a95474c43"},
		{"password", "salt", 4096, "c5e478d59288c841aa530db6845c4c8d962893a001ce4e11a4963873aa98134a"},
	}
	for _, c := range cases {
		got := hex.EncodeToString(pbkdf2SHA256([]byte(c.pass), []byte(c.salt), c.iter, 32))
		if got != c.want {
			t.Errorf("pbkdf2(pass=%q salt=%q iter=%d) = %s, want %s", c.pass, c.salt, c.iter, got, c.want)
		}
	}
}

func TestHashPasswordFormat(t *testing.T) {
	h := HashPassword("secret123")
	parts := strings.Split(h, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2-sha256" {
		t.Fatalf("неожиданный формат: %q", h)
	}
	if parts[1] != fmt.Sprintf("%d", DefaultIterations()) {
		t.Errorf("итерации %s, ожидалось %d", parts[1], DefaultIterations())
	}
	// два вызова дают разные соли
	if h == HashPassword("secret123") {
		t.Error("соль повторяется — не случайная")
	}
}

func TestVerifyPassword(t *testing.T) {
	h := HashPassword("secret123")
	if !VerifyPassword("secret123", h) {
		t.Error("верный пароль отклонён (pbkdf2)")
	}
	if VerifyPassword("wrongpass", h) {
		t.Error("неверный пароль принят (pbkdf2)")
	}

	// legacy sha256-hex
	sum := sha256.Sum256([]byte("oldsecret"))
	legacy := hex.EncodeToString(sum[:])
	if !VerifyPassword("oldsecret", legacy) {
		t.Error("верный пароль отклонён (legacy)")
	}
	if VerifyPassword("nope", legacy) {
		t.Error("неверный пароль принят (legacy)")
	}

	// битые форматы → false (fail-closed)
	broken := []string{"", "not-a-hash", "pbkdf2-sha256$abc$00$00",
		"pbkdf2-sha256$210000$zz$zz", "pbkdf2-sha256$0$aabb$ccdd",
		"pbkdf2-sha256$20000000$" + strings.Repeat("ab", 16) + "$" + strings.Repeat("cd", 32)}
	for _, b := range broken {
		if VerifyPassword("anything", b) {
			t.Errorf("битый формат принят: %q", b)
		}
	}
}

func TestNeedsRehash(t *testing.T) {
	if !NeedsRehash(hex.EncodeToString(make([]byte, 32))) {
		t.Error("legacy-хеш должен требовать перехеша")
	}
	if NeedsRehash(HashPassword("x")) {
		t.Error("pbkdf2-хеш не должен требовать перехеша")
	}
}

func TestDefaultIterations(t *testing.T) {
	n := DefaultIterations()
	switch {
	case n >= 100000:
		if n != 210000 {
			t.Errorf("быстрая архитектура: неожиданно %d", n)
		}
	default:
		if n != 60000 {
			t.Errorf("медленная архитектура: неожиданно %d", n)
		}
	}
}

func TestCheckPasswordPBKDF2(t *testing.T) {
	dir := t.TempDir()
	ConfigPath = filepath.Join(dir, "auth_config.json")
	SessionFile = filepath.Join(dir, "panel_session")

	hash := HashPassword("strongpass99")
	if err := os.WriteFile(ConfigPath, []byte(`{"enabled":true,"password_hash":"`+hash+`"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if !CheckPassword("strongpass99") {
		t.Error("CheckPassword отклонил верный pbkdf2-пароль")
	}
	if CheckPassword("wrong") {
		t.Error("CheckPassword принял неверный пароль")
	}

	// enabled=false → доступ без проверки
	if err := os.WriteFile(ConfigPath, []byte(`{"enabled":false,"password_hash":"`+hash+`"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if !CheckPassword("") {
		t.Error("enabled=false должен открывать доступ")
	}
}
