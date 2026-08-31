// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
//
// Детект сервисов без веб-порта по запущенному процессу.
// Читается только /proc/<pid>/comm и первый аргумент cmdline (basename);
// symlink'и (exe, cwd) не трогаем — нет гонок с исчезающим процессом.
//
// Особенность ядра: comm усекается до 15 символов (TASK_COMM_LEN=16 с NUL),
// поэтому «transmission-daemon» виден как «transmission-da». Матч считаем
// успешным при любом из условий:
//   - name == comm;
//   - comm обрезан (15 симв.) и является префиксом name;
//   - basename(argv[0]) == name.
package bridge

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// procRoot — корень procfs (инъекция для тестов).
var procRoot = "/proc"

// timeNow — точка времени (инъекция для тестов).
var timeNow = time.Now().Unix

const maxProcScan = 4096 // защита от аномально огромного /proc

// procEntry — один живой (не-зомби) процесс: имя из comm, basename argv[0], pid.
type procEntry struct {
	comm string
	base string
	pid  int
}

// snapshotProcs — единый снимок процессов. Вызывается ОДИН раз на проход
// discovery/watch; горутины манифестов читают готовый слайс (write-before-wait).
func snapshotProcs() []procEntry {
	entries, err := os.ReadDir(procRoot)
	if err != nil {
		return nil
	}
	out := make([]procEntry, 0, len(entries))
	scanned := 0
	for _, e := range entries {
		if scanned >= maxProcScan {
			break
		}
		pid, err := strconv.Atoi(e.Name())
		if err != nil || pid <= 0 {
			continue
		}
		scanned++
		if isZombie(pid) {
			continue
		}
		comm := readProcFile(filepath.Join(procRoot, e.Name(), "comm"))
		pe := procEntry{comm: strings.TrimSpace(comm), pid: pid}
		if pe.comm == "" {
			continue
		}
		out = append(out, pe)
	}
	return out
}

// matchProcs — pids процессов, соответствующих любому имени из names.
// argv[0] читается лениво: только если comm не совпал точно/префиксом.
func matchProcs(snap []procEntry, names []string) []int {
	var pids []int
	for _, pe := range snap {
		for _, name := range names {
			if procNameMatch(pe, name) {
				pids = append(pids, pe.pid)
				break
			}
		}
	}
	return pids
}

func procNameMatch(pe procEntry, name string) bool {
	if name == "" {
		return false
	}
	if pe.comm == name {
		return true
	}
	// comm усечён ядром до 15 символов — сравниваем как префикс полного имени
	if len(pe.comm) == 15 && strings.HasPrefix(name, pe.comm) {
		return true
	}
	base := filepath.Base(procArgv0(pe.pid))
	return base == name
}

// procArgv0 — первый токен cmdline (полный путь бинарника); пусто при ошибке.
func procArgv0(pid int) string {
	data := readProcFile(filepath.Join(procRoot, strconv.Itoa(pid), "cmdline"))
	for i, b := range data {
		if b == 0 {
			return string(data[:i])
		}
	}
	return string(data)
}

// isZombie — состояние 'Z' в /proc/<pid>/stat (поле после последнего ')').
func isZombie(pid int) bool {
	stat := readProcFile(filepath.Join(procRoot, strconv.Itoa(pid), "stat"))
	if i := strings.LastIndex(stat, ")"); i >= 0 && i+2 < len(stat) {
		return stat[i+2] == 'Z'
	}
	return false
}

func readProcFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// ProcInfo — агрегированный процесс для сканера манифеста.
type ProcInfo struct {
	Name string `json:"name"`
	Pids int    `json:"pids,omitempty"`
	Init string `json:"init,omitempty"` // имя init-скрипта, если найден в /opt/etc/init.d/
}

// ProcStat — числа живых процессов одного имени из manifest.process
// для карточки: аптайм старейшего процесса и суммарные CPU-тики
// (utime+stime; клиент считает % по дельте между опросами).
type ProcStat struct {
	Name     string `json:"name"`
	Pids     int    `json:"pids"`
	UptimeS  int64  `json:"uptime_s,omitempty"`
	CPUTicks int64  `json:"cpu_ticks,omitempty"`
	MemKB    int64  `json:"mem_kb,omitempty"` // суммарная резидентная память (КБ)
}

// hz — тактовая частота ядра для /proc/<pid>/stat (стандарт CONFIG_HZ=100).
const hz = 100

// procStatCore — starttime (поле 22) и utime+stime (поля 14+15) процесса.
func procStatCore(pid int) (starttime uint64, cpuTicks int64) {
	stat := readProcFile(filepath.Join(procRoot, strconv.Itoa(pid), "stat"))
	if i := strings.LastIndex(stat, ")"); i >= 0 {
		fields := strings.Fields(stat[i+1:])
		// fields[0]=state(3), fields[11]=utime(14), fields[12]=stime(15),
		// fields[19]=starttime(22)
		if len(fields) > 19 {
			utime, _ := strconv.ParseInt(fields[11], 10, 64)
			stime, _ := strconv.ParseInt(fields[12], 10, 64)
			start, _ := strconv.ParseUint(fields[19], 10, 64)
			return start, utime + stime
		}
	}
	return 0, 0
}

// bootTime — btime из /proc/stat (unix-время загрузки).
func bootTime() int64 {
	for _, line := range strings.Split(readProcFile(filepath.Join(procRoot, "stat")), "\n") {
		if strings.HasPrefix(line, "btime ") {
			v, _ := strconv.ParseInt(strings.TrimSpace(line[6:]), 10, 64)
			return v
		}
	}
	return 0
}

// ProcessStats — сводка по каждому имени из manifest.process.
// Имена без живого процесса пропускаются.
func ProcessStats(names []string) []ProcStat {
	snap := snapshotProcs()
	now := timeNow()
	bt := bootTime()
	var out []ProcStat
	for _, name := range names {
		if name == "" {
			continue
		}
		pids := matchProcs(snap, []string{name})
		if len(pids) == 0 {
			continue
		}
		ps := ProcStat{Name: name, Pids: len(pids)}
		for _, pid := range pids {
			start, ticks := procStatCore(pid)
			ps.CPUTicks += ticks
			ps.MemKB += procRSSkb(pid)
			if up := now - (bt + int64(start)/hz); up > ps.UptimeS && start > 0 {
				ps.UptimeS = up
			}
		}
		out = append(out, ps)
	}
	return out
}

const maxProcessList = 128

// ListInitScripts — базовые имена init-скриптов в /opt/etc/init.d/.
// Стрипает префиксы S##/K## и возвращает уникальные имена (для детекта
// в ListProcesses: если процесс совпадает по имени → auto-init).
func ListInitScripts() map[string]bool {
	names := make(map[string]bool)
	entries, err := os.ReadDir(initDirVar)
	if err != nil {
		return names
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		base := n
		if len(n) > 3 && (n[0] == 'S' || n[0] == 'K') &&
			n[1] >= '0' && n[1] <= '9' && n[2] >= '0' && n[2] <= '9' {
			base = n[3:]
		}
		if base != "" {
			names[base] = true
		}
	}
	return names
}

// procRSSkb — резидентная память процесса в КБ (поле 2 statm × размер страницы).
func procRSSkb(pid int) int64 {
	statm := readProcFile(filepath.Join(procRoot, strconv.Itoa(pid), "statm"))
	fields := strings.Fields(statm)
	if len(fields) < 2 {
		return 0
	}
	pages, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return 0
	}
	return pages * int64(os.Getpagesize()) / 1024
}

// ProcessDetails — строки карточки для process-модуля: по каждому имени
// из манифеста — живые PID (до 5) и суммарная память. Имена без процесса
// пропускаются (их отсутствие видно по статусу «не установлен»).
func ProcessDetails(names []string) []CardRow {
	snap := snapshotProcs()
	var rows []CardRow
	for _, name := range names {
		if name == "" {
			continue
		}
		pids := matchProcs(snap, []string{name})
		if len(pids) == 0 {
			continue
		}
		memKB := int64(0)
		for _, pid := range pids {
			memKB += procRSSkb(pid)
		}
		shown := pids
		more := ""
		if len(shown) > 5 {
			shown = shown[:5]
			more = fmt.Sprintf(" …+%d", len(pids)-5)
		}
		pidStrs := make([]string, len(shown))
		for i, pid := range shown {
			pidStrs[i] = strconv.Itoa(pid)
		}
		value := fmt.Sprintf("%d проц. · PID %s%s · %s",
			len(pids), strings.Join(pidStrs, ", "), more,
			humanBytesServer(float64(memKB*1024)))
		rows = append(rows, CardRow{Label: name, Value: value})
	}
	return rows
}

// ListProcesses — живые процессы роутера, агрегированные по имени
// (сканер «Процессы» в редакторе манифеста). Имя — basename(argv[0]),
// т.е. то самое полное имя, которое пишется в process[]; comm — фолбэк.
// Ядро-треды (пустой cmdline) и зомби исключены.
// Если в /opt/etc/init.d/ есть скрипт с таким именем — поле Init заполнено.
func ListProcesses() []ProcInfo {
	snap := snapshotProcs()
	agg := map[string]int{}
	for _, pe := range snap {
		if procArgv0(pe.pid) == "" {
			continue // ядро-тред: cmdline пуст
		}
		name := filepath.Base(procArgv0(pe.pid))
		if name == "" || name == "." || name == "/" {
			name = pe.comm
		}
		if name == "" {
			continue
		}
		agg[name]++
	}
	initScripts := ListInitScripts()
	out := make([]ProcInfo, 0, len(agg))
	for n, c := range agg {
		pi := ProcInfo{Name: n, Pids: c}
		if initScripts[n] {
			pi.Init = n
		}
		out = append(out, pi)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	if len(out) > maxProcessList {
		out = out[:maxProcessList]
	}
	return out
}
