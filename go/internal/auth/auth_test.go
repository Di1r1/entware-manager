package auth

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// setEnv — временная установка env.
func setEnv(t *testing.T, key, val string) func() {
	t.Helper()
	old, existed := os.LookupEnv(key)
	os.Setenv(key, val)
	return func() {
		if existed {
			os.Setenv(key, old)
		} else {
			os.Unsetenv(key)
		}
	}
}

func TestOriginCheck(t *testing.T) {
	cases := []struct {
		name    string
		origin  string
		host    string
		sfs     string
		wantXSS bool
	}{
		{"same origin", "http://192.168.3.1:8087", "192.168.3.1:8087", "", false},
		{"same port-agnostic", "http://192.168.3.1:8087", "192.168.3.1:8087", "same-origin", false},
		{"empty origin allows", "", "192.168.3.1:8087", "", false},
		{"cross host", "http://evil.com", "192.168.3.1:8087", "", true},
		{"cross port", "http://192.168.3.1:9999", "192.168.3.1:8087", "", true},
		{"null origin", "null", "192.168.3.1:8087", "", true},
		{"sec-fetch cross-site", "http://192.168.3.1:8087", "192.168.3.1:8087", "cross-site", true},
		{"sec-fetch same-site", "http://192.168.3.1:8087", "192.168.3.1:8087", "same-site", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			restore1 := setEnv(t, "HTTP_ORIGIN", tc.origin)
			defer restore1()
			restore2 := setEnv(t, "HTTP_HOST", tc.host)
			defer restore2()
			restore3 := setEnv(t, "HTTP_SEC_FETCH_SITE", tc.sfs)
			defer restore3()
			if got := IsCrossSiteOrigin(); got != tc.wantXSS {
				t.Errorf("IsCrossSiteOrigin() = %v, want %v", got, tc.wantXSS)
			}
		})
	}
}

func TestEnabledReportsFailClosed(t *testing.T) {
	orig := ConfigPath
	defer func() { ConfigPath = orig }()

	// Файл отсутствует → deny
	dir := t.TempDir()
	ConfigPath = filepath.Join(dir, "auth_config.json")
	if allow, reason := EnabledReports(); allow || reason == "" {
		t.Errorf("missing file: allow=%v, want deny with reason", allow)
	}

	// Битый файл → deny
	os.WriteFile(ConfigPath, []byte("{broken"), 0644)
	if allow, _ := EnabledReports(); allow {
		t.Errorf("broken config: allow=true, want false")
	}

	// enabled=true, нет hash → deny
	os.WriteFile(ConfigPath, []byte(`{"enabled":true}`), 0644)
	if allow, _ := EnabledReports(); allow {
		t.Errorf("enabled without hash: allow=true, want false")
	}

	// enabled=false валидный → allow
	os.WriteFile(ConfigPath, []byte(`{"enabled":false}`), 0644)
	if allow, _ := EnabledReports(); !allow {
		t.Errorf("enabled=false valid: allow=false, want true")
	}
}

func TestCheckPassword(t *testing.T) {
	orig := ConfigPath
	defer func() { ConfigPath = orig }()
	dir := t.TempDir()
	ConfigPath = filepath.Join(dir, "auth_config.json")

	t.Run("disabled allows any", func(t *testing.T) {
		os.WriteFile(ConfigPath, []byte(`{"enabled":false}`), 0644)
		if !CheckPassword("") {
			t.Errorf("disabled config: CheckPassword(\"\") = false, want true")
		}
	})
	t.Run("wrong hash denies", func(t *testing.T) {
		h := testHash("correct")
		os.WriteFile(ConfigPath, []byte(`{"enabled":true,"password_hash":"`+h+`"}`), 0644)
		if CheckPassword("wrong") {
			t.Errorf("wrong password accepted")
		}
		if !CheckPassword("correct") {
			t.Errorf("correct password denied")
		}
	})
	t.Run("missing file denies", func(t *testing.T) {
		os.Remove(ConfigPath)
		if CheckPassword("anything") {
			t.Errorf("missing file: password accepted")
		}
	})
}

// testHash — SHA-256 hex (аналог удалённого SHA256Hex).
func testHash(password string) string {
	h := sha256.Sum256([]byte(password))
	return fmt.Sprintf("%x", h)
}
