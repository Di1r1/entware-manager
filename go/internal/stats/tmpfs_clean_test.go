package stats

import (
	"os"
	"strings"
	"testing"
)

func TestIsCleanablePath(t *testing.T) {
	allow := []string{"/tmp/foo", "/tmp/a/b", "/dev/shm/x", "/tmp/entware-offline-20260806", "/tmp/data.tar.gz", "/dev/shm/cache.bin"}
	deny := []string{
		"/", "/tmp", "/dev", "/dev/shm", "/opt", "/etc/passwd",
		"/tmp/../etc", "", "/opt/web_entware", "entware-offline-x",
	}
	for _, p := range allow {
		if !isCleanablePath(p) {
			t.Errorf("isCleanablePath(%q) = false, want true", p)
		}
	}
	for _, p := range deny {
		if isCleanablePath(p) {
			t.Errorf("isCleanablePath(%q) = true, want false", p)
		}
	}
}

func TestScanTmpClean_FindsLargeFile(t *testing.T) {
	dir := t.TempDir()
	// Крупный файл и мелкая папка в сканируемом корне.
	if err := os.WriteFile(dir+"/big.bin", make([]byte, 2<<20), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir+"/small", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir+"/small/x", []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("REQUEST_METHOD", "GET")
	os.Setenv("QUERY_STRING", "path="+dir+"&min_bytes=1048576")
	defer func() {
		os.Unsetenv("REQUEST_METHOD")
		os.Unsetenv("QUERY_STRING")
	}()

	// scanTmpClean пишет в stdout — перехватываем через pipe.
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	scanTmpClean()
	w.Close()
	os.Stdout = old

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	out := string(buf[:n])

	if !strings.Contains(out, `"name":"big.bin"`) {
		t.Errorf("expected big.bin in scan output, got: %s", out)
	}
	if !strings.Contains(out, `"type":"file"`) {
		t.Errorf("expected type file for big.bin, got: %s", out)
	}
	if strings.Contains(out, `"name":"small"`) {
		t.Errorf("small dir should be excluded (below threshold), got: %s", out)
	}
}
