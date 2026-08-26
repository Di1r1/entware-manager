// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
package bridge

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// mkProc — синтетический процесс в tmpdir-корне procfs.
func mkProc(t *testing.T, pid int, comm, stat string, argv0 string) {
	t.Helper()
	dir := filepath.Join(procRoot, strconv.Itoa(pid))
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "comm"), []byte(comm+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "stat"), []byte(stat), 0644); err != nil {
		t.Fatal(err)
	}
	if argv0 != "" {
		if err := os.WriteFile(filepath.Join(dir, "cmdline"), []byte(argv0+"\x00"+argv0+"\x00"), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func withProcRoot(t *testing.T, dir string) {
	t.Helper()
	old := procRoot
	procRoot = dir
	t.Cleanup(func() { procRoot = old })
}

// Тесты с настоящим /proc: минимум один процесс гарантированно жив (сам тест).
func TestSnapshotProcsReal(t *testing.T) {
	snap := snapshotProcs()
	if len(snap) == 0 {
		t.Fatal("снимок пуст — в /proc нет ни одного живого процесса")
	}
	for _, pe := range snap {
		if pe.pid <= 0 || pe.comm == "" {
			t.Errorf("битая запись: %+v", pe)
		}
	}
}

func TestMatchProcsExactComm(t *testing.T) {
	self := os.Getpid()
	selfComm := readProcFile(filepath.Join(procRoot, strconv.Itoa(self), "comm"))
	selfComm = strings.TrimSpace(selfComm)
	snap := []procEntry{{comm: selfComm, pid: self}}
	got := matchProcs(snap, []string{selfComm})
	if len(got) != 1 || got[0] != self {
		t.Fatalf("точное совпадение comm не найдено: %v", got)
	}
	if n := matchProcs(snap, []string{"точно-несуществующий"}); len(n) != 0 {
		t.Fatalf("чужое имя совпало: %v", n)
	}
}

// P1 кворума: comm усекается до 15 символов — «transmission-daemon» виден
// как «transmission-da», матч должен сработать по префиксу.
func TestMatchProcsTruncatedComm(t *testing.T) {
	snap := []procEntry{
		{comm: "transmission-da", pid: 100},
		{comm: "xray", pid: 200},
	}
	cases := []struct {
		name string
		want int
	}{
		{"transmission-daemon", 100}, // усечённый префикс (len==15)
		{"transmission-da", 100},     // точное совпадение с усечённым comm
		{"xray", 200},
		{"transmission", 0}, // имя короче comm и не совпадает
	}
	for _, c := range cases {
		got := matchProcs(snap, []string{c.name})
		if c.want == 0 {
			if len(got) != 0 {
				t.Errorf("%q: неожиданное совпадение %v", c.name, got)
			}
			continue
		}
		if len(got) != 1 || got[0] != c.want {
			t.Errorf("%q: got %v, want pid %d", c.name, got, c.want)
		}
	}
}

// Матч по basename(argv[0]), когда comm не помог (переименованный процесс).
func TestMatchProcsArgv0Basename(t *testing.T) {
	withProcRoot(t, t.TempDir())
	mkProc(t, 300, "weird-name", "300 (weird-name) S 1", "/opt/bin/frpc")
	snap := snapshotProcs()
	got := matchProcs(snap, []string{"frpc"})
	if len(got) != 1 || got[0] != 300 {
		t.Fatalf("basename argv[0] не сматчен: %v", got)
	}
}

// Зомби не должны выдаваться за живые (P2 кворума).
func TestSnapshotSkipsZombie(t *testing.T) {
	withProcRoot(t, t.TempDir())
	mkProc(t, 400, "mydaemon", "400 (mydaemon) Z 1", "")
	mkProc(t, 401, "mydaemon", "401 (mydaemon) S 1", "")
	snap := snapshotProcs()
	pids := matchProcs(snap, []string{"mydaemon"})
	if len(pids) != 1 || pids[0] != 401 {
		t.Fatalf("зомби отфильтрован неправильно: %v", pids)
	}
}

// Процесс со скобками в имени comm: парсер stat должен брать состояние после
// последней ')' (имя может содержать скобки и пробелы).
func TestZombieParseWithBrackets(t *testing.T) {
	withProcRoot(t, t.TempDir())
	mkProc(t, 500, "a(b) c", `500 (a(b) c) Z 1`, "")
	if !isZombie(500) {
		t.Error("зомби со скобками в имени не распознан")
	}
	mkProc(t, 501, "plain", "501 (plain) R 1", "")
	if isZombie(501) {
		t.Error("живой процесс распознан как зомби")
	}
}

func TestMatchProcsMultiplePids(t *testing.T) {
	snap := []procEntry{
		{comm: "worker", pid: 10},
		{comm: "worker", pid: 20},
		{comm: "other", pid: 30},
	}
	got := matchProcs(snap, []string{"worker"})
	if len(got) != 2 {
		t.Fatalf("want 2 pid, got %v", got)
	}
}

// Валидация ключа process в манифесте.
func TestValidateManifestProcess(t *testing.T) {
	ok := &Manifest{ID: "xray", Name: "Xray", Process: []string{"xray", "xray-core"}}
	if err := ValidateManifest(ok); err != nil {
		t.Fatalf("process-манифест без probe отклонён: %v", err)
	}

	bad := []struct {
		name string
		m    *Manifest
	}{
		{"пустой probe без process", &Manifest{ID: "a", Name: "A"}},
		{"плохое имя процесса", &Manifest{ID: "a", Name: "A", Process: []string{"x ray"}}},
		{"пустое имя процесса", &Manifest{ID: "a", Name: "A", Process: []string{""}}},
		{"слишком длинное имя", &Manifest{ID: "a", Name: "A", Process: []string{strings.Repeat("x", 65)}}},
		{"больше 4 имён", &Manifest{ID: "a", Name: "A",
			Process: []string{"p1", "p2", "p3", "p4", "p5"}}},
	}
	for _, c := range bad {
		if err := ValidateManifest(c.m); err == nil {
			t.Errorf("%s: принят", c.name)
		}
	}
}

// LoadManifest читает process-only манифест end-to-end.
func TestLoadManifestProcessOnly(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "frpc.json"),
		[]byte(`{"id":"frpc","name":"frp client","process":["frpc"]}`), 0644)
	m, err := LoadManifest(dir, "frpc")
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Process) != 1 || m.Process[0] != "frpc" {
		t.Errorf("process = %v", m.Process)
	}
}
