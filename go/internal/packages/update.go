package packages

import (
	"entware-manager/internal/cgiutil"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"entware-manager/internal/cache"
)

var opkgUpdatePidFile = "/tmp/entware/opkg_update.pid"

// opkgUpdateRunning — жив ли другой процесс обновления списков (по pidfile).
// По образцу updateWorkerRunning() в stats/update.go: файл-пид + проверка
// /proc/<PID>/cmdline. Битый или протухший pidfile не блокирует запуск.
func opkgUpdateRunning() bool {
	return opkgUpdateRunningIn("/proc", opkgUpdatePidFile)
}

func opkgUpdateRunningIn(procDir, pidFile string) bool {
	pidData, err := os.ReadFile(pidFile)
	if err != nil {
		return false
	}
	pid := strings.TrimSpace(string(pidData))
	if pid == "" {
		return false
	}
	var p int
	if _, err := fmt.Sscanf(pid, "%d", &p); err != nil || p <= 0 {
		os.Remove(pidFile)
		return false
	}
	cmdline, err := os.ReadFile(filepath.Join(procDir, pid, "cmdline"))
	if err != nil {
		os.Remove(pidFile)
		return false
	}
	return strings.Contains(string(cmdline), "entware-pkg")
}

// acquireUpdateLock берёт блокировку обновления атомарно через O_EXCL
// (TOCTOU-мутекс): создаёт pidfile только если его нет. При существующем
// файле проверяет жив ли процесс — если протух, убирает и пробует ещё раз.
// Возвращает true, если блокировка получена.
func acquireUpdateLock(path string, pid int) bool {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return false
	}
	for attempt := 0; attempt < 2; attempt++ {
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err == nil {
			_, werr := f.WriteString(strconv.Itoa(pid))
			f.Close()
			return werr == nil
		}
		// файл уже существует — жив ли владелец?
		if opkgUpdateRunning() {
			return false // уже выполняется
		}
		os.Remove(path) // протухший — убрать и повторить
	}
	return false
}

func Update() {
	if !cgiutil.IsPOST() {
		cgiutil.NotAllowed()
		return
	}

	// Origin-чек для POST уже есть в cmd/entware-pkg/main.go:19-22
	// (auth.IsCrossSiteOrigin) — здесь не дублируем.

	if !acquireUpdateLock(opkgUpdatePidFile, os.Getpid()) {
		cgiutil.WriteError("Обновление списков пакетов уже выполняется")
		return
	}
	defer os.Remove(opkgUpdatePidFile)

	html := "<h2>Обновление списков пакетов</h2>\n<pre>\n"

	out, code := runOpkgTimed("update")
	html += htmlEscape(out) + "</pre>\n"

	if code == 0 {
		html += `<p class="success">Списки пакетов успешно обновлены.</p>`
		cache.Invalidate("opkg_list")
	} else {
		html += `<p class="error">Ошибка обновления списков пакетов</p>`
	}

	writeHTML(html)
}
