package packages

import (
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

// writePidFileAtomic пишет pid-файл атомарно (temp + rename, RULES п.10).
func writePidFileAtomic(path string, pid int) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	tmp := fmt.Sprintf("%s.tmp.%d", path, os.Getpid())
	if err := os.WriteFile(tmp, []byte(strconv.Itoa(pid)), 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func Update() {
	if !isPOST() {
		methodNotAllowed()
		return
	}

	// Origin-чек для POST уже есть в cmd/entware-pkg/main.go:19-22
	// (auth.IsCrossSiteOrigin) — здесь не дублируем.

	if opkgUpdateRunning() {
		WriteError("Обновление списков пакетов уже выполняется")
		return
	}
	if err := writePidFileAtomic(opkgUpdatePidFile, os.Getpid()); err != nil {
		WriteError("Не удалось создать файл блокировки: " + err.Error())
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
