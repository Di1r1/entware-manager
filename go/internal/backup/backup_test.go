package backup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, fn func(), out *bytes.Buffer) {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	w.Close()
	io.Copy(out, r)
	os.Stdout = old
}

// buildArchive собирает gzip-tar с заданными записями (name → Typeflag).
// data != nil → запись с данными; nil → запись с типом Typeflag.
func buildArchive(t *testing.T, entries []struct {
	name     string
	typeflag byte
	data     string
	size     int64
}) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	for _, e := range entries {
		size := e.size
		if e.data != "" {
			size = int64(len(e.data))
		}
		tw.WriteHeader(&tar.Header{
			Name:     e.name,
			Typeflag: e.typeflag,
			Size:     size,
			Mode:     0644,
		})
		if e.data != "" {
			tw.Write([]byte(e.data))
		}
	}
	tw.Close()
	gw.Close()
	return buf.Bytes()
}

// runRestore выполняет HandleRestore с телом архива и санитарной папкой TMPDIR.
// webRoot переопределяется на отдельную подпапку, чтобы не писать в /opt.
func runRestore(t *testing.T, archive []byte) string {
	t.Helper()
	sandbox := t.TempDir()
	t.Setenv("TMPDIR", sandbox)
	t.Setenv("REQUEST_METHOD", "POST")
	oldWebRoot := webRoot
	webRoot = filepath.Join(sandbox, "web")
	os.MkdirAll(webRoot, 0755)
	t.Cleanup(func() { webRoot = oldWebRoot })

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
	return out.String()
}

func TestRestore_ValidFiles(t *testing.T) {
	archive := buildArchive(t, []struct {
		name     string
		typeflag byte
		data     string
		size     int64
	}{
		{"links.json", tar.TypeReg, `{"links":[]}`, 0},
		{"logger_config.json", tar.TypeReg, `{"enabled":true}`, 0},
		{"packages.txt", tar.TypeRegA, "package1\n", 0},
		{"backup.json", tar.TypeReg, `{"version":"1.09.10"}`, 0},
	})
	out := runRestore(t, archive)
	if !strings.Contains(out, `"status":"ok"`) {
		t.Fatalf("expected ok status, got: %s", out)
	}
}

func TestRestore_TraversalRejected(t *testing.T) {
	archive := buildArchive(t, []struct {
		name     string
		typeflag byte
		data     string
		size     int64
	}{
		{"../../etc/cron.d/evil", tar.TypeReg, "evil", 0},
		{"../evil", tar.TypeReg, "evil", 0},
		{"sub/dir/f", tar.TypeReg, "f", 0},
		{"./links.json", tar.TypeReg, "x", 0},
		{"/etc/cron.d/evil2", tar.TypeReg, "evil2", 0},
		{".", tar.TypeReg, "x", 0},
		{"..", tar.TypeReg, "x", 0},
	})
	out := runRestore(t, archive)
	_ = out
}

func TestRestore_SymlinkAttackRejected(t *testing.T) {
	archive := buildArchive(t, []struct {
		name     string
		typeflag byte
		data     string
		size     int64
	}{
		{"x", tar.TypeSymlink, "", 0},
		{"x/passwd", tar.TypeReg, "root:root", 0},
	})
	_ = runRestore(t, archive)
}

func TestRestore_DirAndHardlinkRejected(t *testing.T) {
	archive := buildArchive(t, []struct {
		name     string
		typeflag byte
		data     string
		size     int64
	}{
		{"somedir/", tar.TypeDir, "", 0},
		{"hardlink", tar.TypeLink, "", 0},
	})
	_ = runRestore(t, archive)
}

func TestRestore_EntryTooLargeRejected(t *testing.T) {
	archive := buildArchive(t, []struct {
		name     string
		typeflag byte
		data     string
		size     int64
	}{
		{"huge.txt", tar.TypeReg, "", maxEntrySize + 1},
	})
	_ = runRestore(t, archive)
}

func TestBuildArchive(t *testing.T) {
	data, err := BuildArchive()
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 20 {
		t.Fatal("архив подозрительно мал")
	}
	if data[0] != 0x1f || data[1] != 0x8b {
		t.Error("архив не gzip")
	}
}

// Мост и секреты панели входят в бэкап (v1.16.4), при восстановлении
// получают 0600.
func TestBackupBridgeAndSecrets(t *testing.T) {
	sandbox := t.TempDir()
	oldWebRoot := webRoot
	webRoot = filepath.Join(sandbox, "web")
	t.Cleanup(func() { webRoot = oldWebRoot })

	os.MkdirAll(filepath.Join(webRoot, "bridge"), 0755)
	os.MkdirAll(filepath.Join(webRoot, "logger"), 0755)
	os.WriteFile(filepath.Join(webRoot, "version.json"), []byte(`{"version":"test"}`), 0644)
	os.WriteFile(filepath.Join(webRoot, "links.json"), []byte(`[]`), 0644)
	os.WriteFile(filepath.Join(webRoot, "bridge", "myapp.json"), []byte(`{"id":"myapp"}`), 0644)
	os.WriteFile(filepath.Join(webRoot, "bridge", "myapp.auth.json"), []byte(`{}`), 0600)
	os.WriteFile(filepath.Join(webRoot, "bridge", "_prefs.json"), []byte(`{}`), 0600)
	os.WriteFile(filepath.Join(webRoot, "auth_config.json"), []byte(`{"enabled":true}`), 0600)
	os.WriteFile(filepath.Join(webRoot, "logger", "system_sources.json"), []byte(`[]`), 0644)
	// мусор не должен попасть в архив
	os.WriteFile(filepath.Join(webRoot, "bridge", "broken.tmp"), []byte("x"), 0644)

	data, err := BuildArchive()
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
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
		names[h.Name] = true
	}
	want := []string{"bridge_myapp.json", "bridge_myapp.auth.json",
		"bridge__prefs.json", "auth_config.json", "logger_system_sources.json"}
	for _, n := range want {
		if !names[n] {
			t.Errorf("в архиве нет %s (есть: %v)", n, names)
		}
	}
	if names["broken.tmp"] {
		t.Error(".tmp не должен попадать в архив")
	}
}

// Секретные назначения получают 0600 при восстановлении.
func TestSecretDest(t *testing.T) {
	yes := []string{"auth_config.json", "telegram_config.json", "bridge/_prefs.json",
		"bridge/x.auth.json", "bridge/a.b.auth.json"}
	no := []string{"links.json", "bridge/myapp.json", "logger/config.json", "bridge/x.json"}
	for _, d := range yes {
		if !secretDest(d) {
			t.Errorf("%q: ожидался secret", d)
		}
	}
	for _, d := range no {
		if secretDest(d) {
			t.Errorf("%q: не должен быть secret", d)
		}
	}
}

// Roundtrip restore: bridge_* уезжают в bridge/, секреты — 0600.
func TestRestoreBridgeFiles(t *testing.T) {
	archive := buildArchive(t, []struct {
		name     string
		typeflag byte
		data     string
		size     int64
	}{
		{"bridge_myapp.json", tar.TypeReg, `{"id":"myapp"}`, 0},
		{"bridge_myapp.auth.json", tar.TypeReg, `{"password":"p"}`, 0},
		{"bridge__prefs.json", tar.TypeReg, `{"modules":{}}`, 0},
	})
	out := runRestore(t, archive)
	if !strings.Contains(out, `"status":"ok"`) || !strings.Contains(out, "bridge/myapp.json") {
		t.Fatalf("restore failed: %s", out)
	}
	if _, err := os.Stat(filepath.Join(webRoot, "bridge", "myapp.json")); err != nil {
		t.Errorf("манифест не восстановлен: %v", err)
	}
	for _, secret := range []string{"bridge/myapp.auth.json", "bridge/_prefs.json"} {
		fi, err := os.Stat(filepath.Join(webRoot, filepath.FromSlash(secret)))
		if err != nil {
			t.Errorf("%s не восстановлен: %v", secret, err)
			continue
		}
		if fi.Mode().Perm() != 0600 {
			t.Errorf("%s: права %v, want 0600", secret, fi.Mode().Perm())
		}
	}
}
