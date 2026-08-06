// Package localtime выставляет процессу часовой пояс роутера.
//
// На BusyBox-роутерах (Entware) таймзона хранится в виде POSIX-строки
// в /etc/TZ (или /var/TZ, на который симлинком указывает /etc/localtime),
// например "MSK-3". Go не умеет читать такую строку: при пустой переменной
// окружения TZ он пытается открыть /etc/localtime как бинарный tzfile,
// получает текст "MSK-3" и молча откатывается к UTC. Из-за этого время
// Go-процессов расходится с временем shell-скриптов (date), и логи пишутся/
// читаются за соседний день (обычно около полуночи по локальному времени).
//
// Пакет парсит offset из POSIX-строки и делает time.Local фиксированным
// локальным поясом. Подключается пустым импортом в каждой cmd/*/main.go.
package localtime

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func init() {
	tz := routerTZ()
	if tz == "" {
		return
	}
	os.Setenv("TZ", tz)
	if loc := posixLocation(tz); loc != nil {
		time.Local = loc
	}
}

// routerTZ возвращает строку таймзоны роутера: переменную окружения TZ,
// либо содержимое /etc/TZ, /var/TZ, /etc/timezone — первое непустое.
func routerTZ() string {
	if tz := os.Getenv("TZ"); strings.TrimSpace(tz) != "" {
		return strings.TrimSpace(tz)
	}
	for _, p := range []string{"/etc/TZ", "/var/TZ", "/etc/timezone"} {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		if s := strings.TrimSpace(string(b)); s != "" {
			return s
		}
	}
	return ""
}

// posixLocation строит *time.Location из POSIX-строки "std offset".
//
// В POSIX offset — значение, прибавляемое к локальному времени, чтобы
// получить UTC. "MSK-3" → UTC+3 (сдвиг +3ч), "EST5" → UTC-5 (сдвиг -5ч),
// "UTC0" → UTC. DST-часть ("EDT", второй offset) не разбирается: на
// роутерах используется фиксированный сдвиг.
func posixLocation(tz string) *time.Location {
	t := strings.TrimSpace(tz)
	if t == "" {
		return nil
	}

	// Ищем начало offset: первый символ из "+-0123456789".
	idx := -1
	for i := 0; i < len(t); i++ {
		c := t[i]
		if c == '+' || c == '-' || (c >= '0' && c <= '9') {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil
	}

	abbr := strings.ToUpper(t[:idx])
	rest := t[idx:]

	// Знак POSIX-offset. Внутреннее значение V: '-' → отрицательное.
	negative := false
	if rest[0] == '-' {
		negative = true
		rest = rest[1:]
	} else if rest[0] == '+' {
		rest = rest[1:]
	}

	// Части offset: hh[:mm[:ss]], за которыми могут идти буквы DST.
	m := reOffset.FindStringSubmatch(rest)
	if m == nil {
		return nil
	}
	vM, _ := strconv.Atoi(m[1])
	vMin, _ := strconv.Atoi(m[2])
	vSec, _ := strconv.Atoi(m[3])
	vSec = vM*3600 + vMin*60 + vSec
	if negative {
		vSec = -vSec
	}
	// Сдвиг местного времени от UTC = -(POSIX offset).
	offset := -vSec
	if abbr == "" {
		abbr = "LMT"
	}
	return time.FixedZone(abbr, offset)
}

var reOffset = regexp.MustCompile(`^(\d{1,2})(?::(\d{1,2}))?(?::(\d{1,2}))?`)
