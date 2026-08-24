// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
package bridge

import (
	"strings"
	"testing"
)

func TestValidateBridgeURL(t *testing.T) {
	ok := []string{
		"http://127.0.0.1:9097/?action=version",
		"http://localhost:8080/control/status", // localhost перезаписывается
		"http://127.0.0.1:8080/control/status",
	}
	for _, u := range ok {
		if _, err := ValidateBridgeURL(u, ""); err != nil {
			t.Errorf("валидный %q отклонён: %v", u, err)
		}
	}

	bad := map[string]string{
		"":                                       "пустой",
		"https://127.0.0.1:9097/":                "https схема",
		"ftp://127.0.0.1/x":                      "ftp схема",
		"http://192.168.3.1:8087/":               "не loopback",
		"http://evil.example.com/":               "внешний хост",
		"http://admin:pass@127.0.0.1:9097/":      "userinfo",
		"http://127.0.0.1:9097/#frag":            "фрагмент",
		"http://127.0.0.1:0/":                    "порт 0",
		"http://127.0.0.1:99999/":                "порт 99999",
		"http://127.0.0.1:x/":                    "порт не число",
		strings.Repeat("http://127.0.0.1/", 300): "слишком длинный",
	}
	for u, why := range bad {
		if _, err := ValidateBridgeURL(u, ""); err == nil {
			t.Errorf("принят запрещённый URL (%s): %q", why, u)
		}
	}

	// localhost → 127.0.0.1
	res, err := ValidateBridgeURL("http://LOCALhost:8080/control/status", "")
	if err != nil {
		t.Fatal(err)
	}
	if res.Hostname() != "127.0.0.1" {
		t.Errorf("hostname = %q, хочу 127.0.0.1", res.Hostname())
	}
}

func TestValidateBridgeURLRelative(t *testing.T) {
	base := "http://127.0.0.1:9097/"

	good := []string{"?action=status", "/control/stats", "action=status"}
	for _, rel := range good {
		if _, err := ValidateBridgeURL(rel, base); err != nil {
			t.Errorf("относительный %q отклонён: %v", rel, err)
		}
	}

	// относительный путь, уводящий на другой host через // — должен пасть
	if _, err := ValidateBridgeURL("//evil.example.com/x", base); err == nil {
		t.Error("protocol-relative URL на внешний хост принят")
	}

	// без базы — падает
	if _, err := ValidateBridgeURL("?action=status", ""); err == nil {
		t.Error("относительный без базы принят")
	}
}

func TestValidateBridgeURLResolvedRechecked(t *testing.T) {
	// резолв от валидной базы не должен позволить сменить host
	if _, err := ValidateBridgeURL("http://10.0.0.5/x", "http://127.0.0.1:9097/"); err == nil {
		t.Error("абсолютный не-loopback в status/action принят")
	}
}
