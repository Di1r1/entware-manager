// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
package bridge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func jsonMarshal(v interface{}) ([]byte, error) { return json.Marshal(v) }

func TestValidateManifest(t *testing.T) {
	good := &Manifest{
		ID:     "koffe",
		Name:   "Koffe VPN",
		Base:   "http://127.0.0.1:9097/",
		Probe:  Endpoint{URL: "?action=version", Expect: "json"},
		Status: &Endpoint{URL: "?action=status"},
		Actions: []Action{
			{ID: "restart", Label: "Перезапуск", URL: "?action=restart", Method: "POST", Confirm: true},
		},
	}
	if err := ValidateManifest(good); err != nil {
		t.Fatalf("валидный манифест отклонён: %v", err)
	}

	bad := []struct {
		name string
		m    *Manifest
	}{
		{"плохой id", &Manifest{ID: "../evil", Name: "x", Probe: Endpoint{URL: "http://127.0.0.1:1/"}}},
		{"длинный id", &Manifest{ID: strings.Repeat("a", 40), Name: "x", Probe: Endpoint{URL: "http://127.0.0.1:1/"}}},
		{"SSRF в probe", &Manifest{ID: "koffe", Name: "x", Probe: Endpoint{URL: "http://8.8.8.8/"}}},
		{"SSRF в action", &Manifest{ID: "koffe", Name: "x",
			Probe:   Endpoint{URL: "http://127.0.0.1:9097/"},
			Actions: []Action{{ID: "a", Label: "a", URL: "http://10.0.0.1/x"}}},
		},
		{"слишком много действий", &Manifest{ID: "koffe", Name: "x",
			Probe:   Endpoint{URL: "http://127.0.0.1:9097/"},
			Actions: manyActions(MaxActions + 1)},
		},
	}
	for _, c := range bad {
		if err := ValidateManifest(c.m); err == nil {
			t.Errorf("%s: принят", c.name)
		}
	}
}

func manyActions(n int) []Action {
	out := make([]Action, n)
	for i := range out {
		out[i] = Action{ID: string(rune('a'+i%26)) + itoa(i), Label: "a", URL: "http://127.0.0.1:9097/"}
	}
	return out
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}

func TestLoadManifestSecurity(t *testing.T) {
	dir := t.TempDir()

	// path traversal через id
	if _, err := LoadManifest(dir, "../../etc/passwd"); err == nil {
		t.Error("traversal-id принят")
	}

	// битый JSON
	os.WriteFile(filepath.Join(dir, "broken.json"), []byte("{nope"), 0644)
	if _, err := LoadManifest(dir, "broken"); err == nil {
		t.Error("битый манифест принят")
	}

	// неизвестное поле → reject (DisallowUnknownFields)
	os.WriteFile(filepath.Join(dir, "extra.json"),
		[]byte(`{"id":"extra","name":"x","probe":{"url":"http://127.0.0.1:1/"},"hacker_field":true}`), 0644)
	if _, err := LoadManifest(dir, "extra"); err == nil {
		t.Error("манифест с неизвестным полем принят")
	}

	// oversize
	big := `{"id":"big","name":"` + strings.Repeat("x", MaxManifestSize) + `"}`
	os.WriteFile(filepath.Join(dir, "big.json"), []byte(big), 0644)
	if _, err := LoadManifest(dir, "big"); err == nil {
		t.Error("oversize манифест принят")
	}

	// валидный читается
	os.WriteFile(filepath.Join(dir, "ok.json"),
		[]byte(`{"id":"ok","name":"Ok","probe":{"url":"http://127.0.0.1:9097/?action=version"}}`), 0644)
	m, err := LoadManifest(dir, "ok")
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "Ok" {
		t.Errorf("name = %q", m.Name)
	}
}

func TestLoadAuthSeparation(t *testing.T) {
	dir := t.TempDir()
	// секреты лежат отдельно и читаются только из .auth.json
	os.WriteFile(filepath.Join(dir, "svc.json"),
		[]byte(`{"id":"svc","name":"S","probe":{"url":"http://127.0.0.1:1/"},"password":"LEAK"}`), 0644)
	m, err := LoadManifest(dir, "svc")
	if err == nil && strings.Contains(string(mustJSON(t, m)), "LEAK") {
		t.Error("секрет из манифеста попал в структуру")
	}

	os.WriteFile(filepath.Join(dir, "svc.auth.json"), []byte(`{"type":"basic","username":"u","password":"p"}`), 0600)
	a := LoadAuth(dir, "svc")
	if a == nil || a.Username != "u" {
		t.Fatal("auth.json не прочитан")
	}

	// ListManifests не подхватывает *.auth.json как манифест
	ms := ListManifests(dir)
	for _, mm := range ms {
		if strings.HasSuffix(mm.ID, ".auth") {
			t.Error(".auth.json подхвачен как манифест")
		}
	}
}

func mustJSON(t *testing.T, v interface{}) string {
	t.Helper()
	b, _ := jsonMarshal(v)
	return string(b)
}
