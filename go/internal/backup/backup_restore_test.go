package backup

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runRestorePrep — как runRestore, но до HandleRestore создаёт файлы с
// заданными правами: восстановление поверх СУЩЕСТВУЮЩЕГО файла должно
// выставить правильный режим, а не оставить старый.
func runRestorePrep(t *testing.T, archive []byte, prep map[string]os.FileMode) {
	t.Helper()
	sandbox := t.TempDir()
	t.Setenv("TMPDIR", sandbox)
	t.Setenv("REQUEST_METHOD", "POST")
	oldWebRoot := webRoot
	webRoot = filepath.Join(sandbox, "web")
	os.MkdirAll(webRoot, 0755)
	t.Cleanup(func() { webRoot = oldWebRoot })

	for rel, mode := range prep {
		dst := filepath.Join(webRoot, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dst, []byte("stale"), mode); err != nil {
			t.Fatal(err)
		}
	}

	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	w.Write(archive)
	w.Close()
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	var out bytes.Buffer
	captureStdout(t, func() {
		HandleRestore()
	}, &out)
	if !strings.Contains(out.String(), `"status":"ok"`) {
		t.Fatalf("restore не прошёл: %s", out.String())
	}
}

// Регрессия: os.WriteFile не меняет права уже существующего файла,
// поэтому секреты, созданные с 0644 (или развернутые старой версией),
// после restore оставались читаемыми всеми. Restore обязан chmod-ить.
func TestRestoreChmodsExistingSecrets(t *testing.T) {
	archive := buildArchive(t, []struct {
		name     string
		typeflag byte
		data     string
		size     int64
	}{
		{"auth_config.json", tar.TypeReg, `{"enabled":true,"hash":"x"}`, 0},
		{"telegram_config.json", tar.TypeReg, `{"token":"t"}`, 0},
		{"bridge__prefs.json", tar.TypeReg, `{"modules":{}}`, 0},
		{"bridge_myapp.auth.json", tar.TypeReg, `{"password":"p"}`, 0},
		{"links.json", tar.TypeReg, `{"links":[]}`, 0},
	})

	runRestorePrep(t, archive, map[string]os.FileMode{
		"auth_config.json":       0644,
		"telegram_config.json":   0644,
		"bridge/_prefs.json":     0644,
		"bridge/myapp.auth.json": 0644,
		"links.json":             0666,
	})

	secrets := []string{
		"auth_config.json",
		"telegram_config.json",
		"bridge/_prefs.json",
		"bridge/myapp.auth.json",
	}
	for _, rel := range secrets {
		fi, err := os.Stat(filepath.Join(webRoot, filepath.FromSlash(rel)))
		if err != nil {
			t.Errorf("%s не восстановлен: %v", rel, err)
			continue
		}
		if fi.Mode().Perm() != 0600 {
			t.Errorf("%s: права %v, want 0600", rel, fi.Mode().Perm())
		}
	}
	// Обычный конфиг остаётся 0644 (не затягиваем права на лишнее).
	fi, err := os.Stat(filepath.Join(webRoot, "links.json"))
	if err != nil {
		t.Fatalf("links.json не восстановлен: %v", err)
	}
	if fi.Mode().Perm() != 0644 {
		t.Errorf("links.json: права %v, want 0644", fi.Mode().Perm())
	}
}
