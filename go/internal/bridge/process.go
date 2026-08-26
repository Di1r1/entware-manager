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
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// procRoot — корень procfs (инъекция для тестов).
var procRoot = "/proc"

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
