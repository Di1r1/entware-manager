// Di1r1
package rdp

import (
	"bufio"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// LighttpdConf — путь к конфигу lighttpd-режима (создаётся install.sh).
const LighttpdConf = "/opt/etc/lighttpd/conf.d/90-entware-manager.conf"

// portIsBusy проверяет, занят ли TCP-порт на loopback или всех интерфейсах
// процессом, отличным от grdp-proxy. Использует netstat/ss (BusyBox-совместимо).
func portIsBusy(port int) bool {
	portStr := ":" + strconv.Itoa(port)
	probe := []string{"netstat", "-ltn"}
	if _, err := exec.LookPath("ss"); err == nil {
		probe = []string{"ss", "-ltn"}
	}
	out, err := exec.Command(probe[0], probe[1:]...).Output()
	if err != nil {
		return false
	}
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		line := sc.Text()
		if !strings.Contains(line, portStr) || !strings.Contains(line, "LISTEN") {
			continue
		}
		// Наш grdp-proxy может слушать этот порт (если уже сконфигурирован).
		pid := pidFromNetstat(line)
		if pid > 0 && processIsProxy(pid) {
			continue
		}
		return true
	}
	return false
}

// pidFromNetstat извлекает PID из последнего поля netstat ("pid/program").
func pidFromNetstat(line string) int {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return 0
	}
	last := fields[len(fields)-1]
	pidStr := strings.Split(last, "/")[0]
	pid, _ := strconv.Atoi(pidStr)
	return pid
}

// processIsProxy проверяет, что PID принадлежит grdp-proxy.
func processIsProxy(pid int) bool {
	if pid <= 0 {
		return false
	}
	cmdline, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/cmdline")
	if err != nil {
		return false
	}
	return strings.Contains(string(cmdline), "grdp-proxy")
}

// applyProxyPortChange применяет смену порта grdp-proxy:
//  1. перезапускает прокси через init-скрипт (он сам читает новый порт);
//  2. в lighttpd-режиме обновляет proxy.server для /rdp/ и /ws в
//     90-entware-manager.conf, чтобы reverse-proxy указывал на новый порт.
//
// В go-режиме reverse-proxy (entware-server) читает порт из конфига
// динамически — ничего делать не нужно.
func applyProxyPortChange(port int) {
	if port < 1 || port > 65535 {
		return
	}
	// Перезапуск прокси на новом порту (идемпотентно).
	if _, err := os.Stat(ProxyInitScript); err == nil {
		exec.Command(ProxyInitScript, "restart").Run()
	}
	// lighttpd-режим: заменить порт в proxy.server блоков /rdp/ и /ws.
	updateLighttpdRDPPort(port)
}

var rdpPortRe = regexp.MustCompile(`("port" => )\d+`)

// updateLighttpdRDPPort заменяет порт в строках proxy.server, находящихся
// в блоках $HTTP["url"] =~ "^/rdp/" и "^/ws" конфига lighttpd.
func updateLighttpdRDPPort(port int) {
	data, err := os.ReadFile(LighttpdConf)
	if err != nil {
		return // go-режим или конфига нет
	}
	lines := strings.Split(string(data), "\n")
	inRDP := false
	for i, ln := range lines {
		if strings.Contains(ln, `$HTTP["url"]`) {
			// Новый блок: смотрим, RDP ли это.
			inRDP = strings.Contains(ln, `^/rdp/`) || strings.Contains(ln, `^/ws`)
		} else if inRDP && strings.Contains(ln, "proxy.server") {
			lines[i] = rdpPortRe.ReplaceAllString(ln, "${1}"+strconv.Itoa(port))
			inRDP = false
		}
	}
	// Атомарная запись (temp + mv), чтобы не оставить усечённый конфиг при сбое.
	if err := os.WriteFile(LighttpdConf+".tmp", []byte(strings.Join(lines, "\n")), 0644); err != nil {
		return
	}
	os.Rename(LighttpdConf+".tmp", LighttpdConf)
	// Применить конфиг: перезапуск lighttpd-режима (S80lighttpd).
	if _, err := os.Stat("/opt/etc/init.d/S80lighttpd"); err == nil {
		exec.Command("/opt/etc/init.d/S80lighttpd", "restart").Run()
	}
}
