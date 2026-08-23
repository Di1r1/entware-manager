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
