package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	SetBaseDir(filepath.Join(os.TempDir(), "entware-cache-test"))
	os.RemoveAll(baseDir)
	code := m.Run()
	os.RemoveAll(baseDir)
	os.Exit(code)
}

func TestPutGet(t *testing.T) {
	if err := Put("a", []byte("hello")); err != nil {
		t.Fatalf("Put: %v", err)
	}
	data, ok := Get("a", time.Minute)
	if !ok || string(data) != "hello" {
		t.Fatalf("Get: got %q ok=%v", data, ok)
	}
}

func TestGetMissing(t *testing.T) {
	if _, ok := Get("nope", time.Minute); ok {
		t.Fatal("expected missing key to return ok=false")
	}
}

func TestGetExpired(t *testing.T) {
	if err := Put("old", []byte("x")); err != nil {
		t.Fatal(err)
	}
	os.Chtimes(path("old"), time.Now().Add(-time.Hour), time.Now().Add(-time.Hour))
	if _, ok := Get("old", time.Second); ok {
		t.Fatal("expected expired key to return ok=false")
	}
	// с TTL=0 всегда считаем свежим
	if _, ok := Get("old", 0); !ok {
		t.Fatal("expected ttl=0 to ignore expiry")
	}
}

func TestCorruptedFile(t *testing.T) {
	os.MkdirAll(baseDir, 0755)
	os.WriteFile(path("bad"), []byte("x"), 0644)
	// каталог вместо файла — ReadFile вернёт ошибку, Get должен вернуть ok=false
	os.MkdirAll(filepath.Join(baseDir, "dirkey"), 0755)
	if _, ok := Get("dirkey", time.Minute); ok {
		t.Fatal("expected dir key to return ok=false")
	}
	if _, ok := Get("bad", time.Minute); !ok {
		t.Fatal("expected normal file to be readable")
	}
}

func TestInvalidate(t *testing.T) {
	Put("k", []byte("v"))
	Invalidate("k")
	if _, ok := Get("k", time.Minute); ok {
		t.Fatal("expected key to be gone after Invalidate")
	}
	// не падает на отсутствующих ключах
	Invalidate("k", "k2")
}
