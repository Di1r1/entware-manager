// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
//
// Управление сервисом модуля через init.d (manifest.init): start/stop/restart.
// Три рубежа защиты: имя за regex, скрипт обязан лежать в каталоге init.d
// и быть исполняемым, выполнение разрешено только при prefs.control=true
// (галочка «управление» на карточке модуля).
package bridge

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"time"
)

// initDirVar — каталог init.d (инъекция для тестов).
var initDirVar = "/opt/etc/init.d"

var initNameRe = regexp.MustCompile(`^[a-z0-9._-]{1,64}$`)

const ctlTimeout = 15 * time.Second

// ValidInitName — имя сервиса манифеста проходит regex.
func ValidInitName(name string) bool { return initNameRe.MatchString(name) }

// FindInitScript — исполняемый скрипт сервиса в каталоге init.d:
// S??name / K??name / name. Пустая строка = не найден.
func FindInitScript(name string) string {
	if !initNameRe.MatchString(name) {
		return ""
	}
	entries, err := os.ReadDir(initDirVar)
	if err != nil {
		return ""
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
		if base != name {
			continue
		}
		p := filepath.Join(initDirVar, n)
		if fi, err := os.Stat(p); err == nil && fi.Mode().IsRegular() && fi.Mode().Perm()&0111 != 0 {
			return p
		}
	}
	return ""
}

// ControlAllowed — разрешил ли пользователь управление модулем (fail-closed:
// нет записи в prefs или галочка не включена → false).
func ControlAllowed(id string) bool {
	pf := LoadPrefs()
	m, ok := pf.Modules[id]
	return ok && m.Control
}

// RunInitAction — запуск скрипта с фиксированным аргументом (op приходит
// только из whitelist вызывающего обработчика). Вывод обрезан до 512 байт,
// таймаут 15с — зависший init-скрипт не блокирует CGI.
func RunInitAction(script, op string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), ctlTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, script, op)
	cmd.Env = append(os.Environ(), "HOME=/opt/root", "PATH=/opt/sbin:/opt/bin:/sbin:/bin:/usr/sbin:/usr/bin")
	out, err := cmd.CombinedOutput()
	if len(out) > 512 {
		out = out[:512]
	}
	return string(out), err
}
