// Di1r1
package rdp

import (
	"os"
	"strconv"
	"strings"
)

// Instance — статус прокси.
type Instance struct {
	State string `json:"state"`
	PID   string `json:"pid"`
	Port  int    `json:"port"`
}

// Status собирает текущий статус прокси: PID из pid-файла (+ проверка
// что процесс жив и это grdp-proxy), порт из конфига.
func Status() Instance {
	inst := Instance{State: "stopped", PID: "", Port: defaultProxyPort}
	if cfg, err := LoadConfig(); err == nil {
		inst.Port = cfg.ProxyPort
	}
	if pid, ok := aliveProxyPID(); ok {
		inst.State = "running"
		inst.PID = pid
	}
	return inst
}

// aliveProxyPID возвращает PID живого grdp-proxy из pid-файла.
// Проверка: /proc/<pid> существует и cmdline содержит grdp-proxy.
func aliveProxyPID() (string, bool) {
	data, err := os.ReadFile(ProxyPidFile)
	if err != nil {
		return "", false
	}
	pid := strings.TrimSpace(string(data))
	if pid == "" {
		return "", false
	}
	if _, ok := isProxyAlive(pid); ok {
		return pid, true
	}
	return "", false
}

// isProxyAlive проверяет /proc/<pid>/cmdline на grdp-proxy.
func isProxyAlive(pid string) (int, bool) {
	n, err := strconv.Atoi(pid)
	if err != nil || n <= 0 {
		return 0, false
	}
	cmd, err := os.ReadFile("/proc/" + pid + "/cmdline")
	if err != nil {
		return 0, false
	}
	return n, strings.Contains(string(cmd), "grdp-proxy")
}
