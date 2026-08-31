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

// mkProcStatm — резидентная память процесса (в страницах) для тестов ProcessDetails.
func mkProcStatm(t *testing.T, pid, pages int) {
	t.Helper()
	path := filepath.Join(procRoot, strconv.Itoa(pid), "statm")
	if err := os.WriteFile(path, []byte("1024 "+strconv.Itoa(pages)+" 0\n"), 0644); err != nil {
		t.Fatal(err)
	}
}

func manyProcNames(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = "p" + strconv.Itoa(i)
	}
	return out
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
		{"больше 24 имён", &Manifest{ID: "a", Name: "A",
			Process: manyProcNames(25)}},
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

// Валидация ключа init и поиск скрипта в init.d.
func TestInitKeyAndScript(t *testing.T) {
	ok := &Manifest{ID: "xray", Name: "Xray", Process: []string{"xray"}, Init: "xray"}
	if err := ValidateManifest(ok); err != nil {
		t.Fatalf("init отклонён: %v", err)
	}
	bad := &Manifest{ID: "a", Name: "A", Process: []string{"a"}, Init: "../evil"}
	if err := ValidateManifest(bad); err == nil {
		t.Error("traversal-имя init принято")
	}

	old := initDirVar
	initDirVar = t.TempDir()
	t.Cleanup(func() { initDirVar = old })
	writeExec := func(name string) {
		p := filepath.Join(initDirVar, name)
		os.WriteFile(p, []byte("#!/bin/sh\n"), 0755)
	}
	writeExec("S99koffe-api")
	writeExec("not-executable")
	os.WriteFile(filepath.Join(initDirVar, "K01dead"), []byte("#!/bin/sh\n"), 0644) // не исполняемый

	if s := FindInitScript("koffe-api"); !strings.HasSuffix(s, "S99koffe-api") {
		t.Errorf("FindInitScript = %q, want S99koffe-api", s)
	}
	if s := FindInitScript("net-sushestvuet"); s != "" {
		t.Errorf("несуществующий сервис найден: %q", s)
	}
	if s := FindInitScript("../evil"); s != "" {
		t.Errorf("traversal прошёл: %q", s)
	}
	if s := FindInitScript("dead"); s != "" {
		t.Errorf("неисполняемый скрипт принят: %q", s)
	}

	// ControlAllowed fail-closed: нет записи → false
	SetBridgeDir(t.TempDir())
	defer SetBridgeDir("/opt/web_entware/bridge")
	if ControlAllowed("koffe-api") {
		t.Error("control без записи в prefs должен быть false")
	}
	pf := LoadPrefs()
	pf.Modules["koffe-api"] = ModulePrefs{Enabled: true, Notifications: true, Control: true}
	SavePrefs(pf)
	if !ControlAllowed("koffe-api") {
		t.Error("control=true не подхватился")
	}
}

// ListProcesses: агрегация по basename(argv[0]), ядро-треды и зомби исключены.
func TestListProcesses(t *testing.T) {
	withProcRoot(t, t.TempDir())
	mkProc(t, 600, "xray", `600 (xray) S 1`, "/opt/bin/xray")
	mkProc(t, 601, "xray", `601 (xray) S 1`, "/opt/bin/xray")
	mkProc(t, 602, "lighttpd", `602 (lighttpd) S 1`, "/opt/sbin/lighttpd")
	mkProc(t, 603, "kthreadd", `603 (kthreadd) S 1`, "")          // ядро-тред
	mkProc(t, 604, "dead", `604 (dead) Z 1`, "/opt/bin/deadmond") // зомби
	list := ListProcesses()
	got := map[string]int{}
	for _, p := range list {
		got[p.Name] = p.Pids
	}
	if got["xray"] != 2 {
		t.Errorf("xray: want 2 pid, got %v", got)
	}
	if got["lighttpd"] != 1 {
		t.Errorf("lighttpd: want 1, got %v", got)
	}
	for _, bad := range []string{"kthreadd", "dead"} {
		if _, ok := got[bad]; ok {
			t.Errorf("%s не должен попадать в список (ядро-тред/зомби)", bad)
		}
	}
}

// ProcessDetails: PID-список и память по каждому имени манифеста;
// имена без процесса пропускаются.
func TestProcessDetails(t *testing.T) {
	withProcRoot(t, t.TempDir())
	mkProc(t, 700, "xray", `700 (xray) S 1`, "/opt/bin/xray")
	mkProc(t, 701, "xray", `701 (xray) S 1`, "/opt/bin/xray")
	mkProcStatm(t, 700, 256) // 256 страниц × 4096 = 1 МБ
	mkProcStatm(t, 701, 128)
	rows := ProcessDetails([]string{"xray", "net-takogo"})
	if len(rows) != 1 {
		t.Fatalf("want 1 row (несуществующее имя пропускается), got %v", rows)
	}
	r := rows[0]
	if r.Label != "xray" {
		t.Errorf("label = %q", r.Label)
	}
	if !strings.Contains(r.Value, "2 проц.") || !strings.Contains(r.Value, "PID 700, 701") {
		t.Errorf("value = %q, want PID-список и число процессов", r.Value)
	}
	if !strings.Contains(r.Value, "1.5 МБ") {
		t.Errorf("value = %q, want суммарную память 1.5 МБ", r.Value)
	}
}

// Обрезка длинного PID-списка до 5 с хвостом «…+N».
func TestProcessDetailsPidCap(t *testing.T) {
	withProcRoot(t, t.TempDir())
	for i := 0; i < 7; i++ {
		pid := 800 + i
		mkProc(t, pid, "worker", strconv.Itoa(pid)+" (worker) S 1", "/opt/bin/worker")
	}
	rows := ProcessDetails([]string{"worker"})
	if len(rows) != 1 {
		t.Fatalf("want 1 row, got %v", rows)
	}
	v := rows[0].Value
	if !strings.Contains(v, "7 проц.") || !strings.Contains(v, "…+2") {
		t.Errorf("value = %q, want «7 проц.» и «…+2»", v)
	}
}

// mkProcFull — процесс с полным stat (starttime/utime/stime) для ProcessStats.
// starttime=100 тиков (1 сек после загрузки), utime+stime = cpuTicks.
func mkProcFull(t *testing.T, pid int, comm, argv0 string, cpuTicks int64) {
	t.Helper()
	stat := strconv.Itoa(pid) + " (" + comm + ") S 1 1 1 0 -1 4194560 1 0 0 0 " +
		strconv.FormatInt(cpuTicks/2, 10) + " " + strconv.FormatInt(cpuTicks-cpuTicks/2, 10) +
		" 0 0 20 0 1 0 " + strconv.Itoa(100) // starttime=100
	for i := 0; i < 22; i++ {
		stat += " 0"
	}
	mkProc(t, pid, comm, stat, argv0)
}

func TestProcessStats(t *testing.T) {
	withProcRoot(t, t.TempDir())
	oldNow := timeNow
	timeNow = func() int64 { return 100000 }
	t.Cleanup(func() { timeNow = oldNow })

	// btime: загрузка в t=50000; starttime=100 тиков → аптайм ≈ 100000-50000-1
	os.WriteFile(filepath.Join(procRoot, "stat"), []byte("cpu  1 2 3\nbtime 50000\n"), 0644)

	mkProcFull(t, 900, "xray", "/opt/bin/xray", 250) // 2.5 сек CPU
	mkProcFull(t, 901, "xray", "/opt/bin/xray", 150)
	mkProcStatm(t, 900, 256) // 256 страниц × 4096 = 1 МБ
	mkProcStatm(t, 901, 128) // 512 КБ
	stats := ProcessStats([]string{"xray", "net-takogo"})
	if len(stats) != 1 {
		t.Fatalf("want 1 ProcStat, got %v", stats)
	}
	ps := stats[0]
	if ps.Name != "xray" || ps.Pids != 2 {
		t.Errorf("name/pids = %+v", ps)
	}
	if ps.CPUTicks != 400 {
		t.Errorf("cpu_ticks = %d, want 400", ps.CPUTicks)
	}
	if ps.MemKB != 1536 { // (256+128) страниц × 4096 / 1024
		t.Errorf("mem_kb = %d, want 1536", ps.MemKB)
	}
	wantUp := int64(100000 - 50000 - 1)
	if ps.UptimeS != wantUp {
		t.Errorf("uptime_s = %d, want %d", ps.UptimeS, wantUp)
	}
}

// ListInitScripts: сканирует init.d, возвращает базовые имена без S##/K##.
func TestListInitScripts(t *testing.T) {
	old := initDirVar
	initDirVar = t.TempDir()
	t.Cleanup(func() { initDirVar = old })
	writeExec := func(name string) {
		os.WriteFile(filepath.Join(initDirVar, name), []byte("#!/bin/sh\n"), 0755)
	}
	writeExec("S92syncthing")
	writeExec("S99adguardhome")
	writeExec("K01dead")
	writeExec("bare") // без префикса
	names := ListInitScripts()
	for _, want := range []string{"syncthing", "adguardhome", "dead", "bare"} {
		if !names[want] {
			t.Errorf("init-скрипт %q не найден в ListInitScripts", want)
		}
	}
	// S## префиксы не должны попадать как отдельные имена
	if names["S92syncthing"] {
		t.Error("полное имя файла не должно попадать в map")
	}
}

// ListProcesses: поле Init заполнено для процесса с совпадающим init-скриптом.
func TestListProcessesInit(t *testing.T) {
	withProcRoot(t, t.TempDir())
	mkProc(t, 1000, "syncthing", `1000 (syncthing) S 1`, "/opt/bin/syncthing")
	mkProc(t, 1001, "lighttpd", `1001 (lighttpd) S 1`, "/opt/sbin/lighttpd")
	old := initDirVar
	initDirVar = t.TempDir()
	t.Cleanup(func() { initDirVar = old })
	os.WriteFile(filepath.Join(initDirVar, "S92syncthing"), []byte("#!/bin/sh\n"), 0755)
	// lighttpd НЕТ init-скрипта в тесте
	list := ListProcesses()
	for _, p := range list {
		if p.Name == "syncthing" && p.Init != "syncthing" {
			t.Errorf("syncthing: Init = %q, want \"syncthing\"", p.Init)
		}
		if p.Name == "lighttpd" && p.Init != "" {
			t.Errorf("lighttpd: Init = %q, want пусто (нет init-скрипта)", p.Init)
		}
	}
}
