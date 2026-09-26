package smart

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// fakeProbe подменяет smartctlRunTimeout: на каждый -d возвращает заранее
// заданный вывод/ошибку и пишет порядок вызовов.
type fakeProbe struct {
	byDriver   map[string]string // вывод по значению -d
	errs       map[string]error  // ошибка по значению -d
	calls      []string          // порядок вызовов в виде "-d <type>"
	delay      time.Duration     // задержка на вызов
	defaultOut string
}

func (f *fakeProbe) run(dev string, timeout time.Duration, args ...string) (string, error) {
	typ := ""
	for i, a := range args {
		if a == "-d" && i+1 < len(args) {
			typ = args[i+1]
		}
	}
	f.calls = append(f.calls, typ)
	if f.delay > 0 {
		time.Sleep(f.delay)
	}
	if err, ok := f.errs[typ]; ok {
		return "", err
	}
	if out, ok := f.byDriver[typ]; ok {
		return out, nil
	}
	return f.defaultOut, nil
}

func withProbe(t *testing.T, f *fakeProbe) {
	t.Helper()
	old := smartctlRunTimeoutFn
	smartctlRunTimeoutFn = f.run
	t.Cleanup(func() { smartctlRunTimeoutFn = old })
}

// Регрессия (аудит P1-04/P1-09): USB-мост может ответить ПУСТЫМ выводом.
// Раньше `if out == "" { return }` обрывал цепочку, и следующий -d
// (usbjmicron/sntjmicron) не проверялся — диск оставался без данных.
func TestFallback_EmptyOutputTriesNextDriver(t *testing.T) {
	f := &fakeProbe{byDriver: map[string]string{
		"sat":        "", // мост ответил пустотой — не значит «подвешен»
		"usbjmicron": "SMART overall-health self-assessment test result: PASSED",
		"sntjmicron": "SMART overall-health self-assessment test result: PASSED",
		"scsi":       "SMART overall-health self-assessment test result: PASSED",
	}}
	withProbe(t, f)

	out, used := smartctlWithFallback("sda", "/dev/sda", time.Second, "-a")
	if !strings.Contains(out, "PASSED") {
		t.Fatalf("ожидался пригодный вывод, получен: %q", out)
	}
	if used != "usbjmicron" {
		t.Errorf("сработавший драйвер = %q, want usbjmicron (вызовы: %v)", used, f.calls)
	}
	if len(f.calls) < 2 {
		t.Errorf("цепочка не перешла к следующему кандидату: %v", f.calls)
	}
}

// Диск подвешен (busy) — дальше идти бессмысленно, и тип не выдаём.
func TestFallback_BusyStopsChain(t *testing.T) {
	f := &fakeProbe{errs: map[string]error{"sat": errDeviceBusy}}
	withProbe(t, f)

	out, used := smartctlWithFallback("sda", "/dev/sda", time.Second, "-a")
	if used != "" {
		t.Errorf("usedType должен быть пуст при busy, получен %q", used)
	}
	if out != "" {
		t.Errorf("вывод при busy ожидается пустым, получен %q", out)
	}
	if len(f.calls) != 1 {
		t.Errorf("при busy должен быть ровно один вызов, было %d: %v", len(f.calls), f.calls)
	}
}

// Не вернуть тип, который ничего не дал: UI не должен показывать «sat» как
// рабочий драйвер на диске, который не ответил ни одним -d.
func TestFallback_AllFailReturnsNoDriver(t *testing.T) {
	f := &fakeProbe{defaultOut: "Read Device Identity failed: scsi command failed"}
	withProbe(t, f)

	out, used := smartctlWithFallback("sda", "/dev/sda", time.Second, "-a")
	if used != "" {
		t.Errorf("usedType должен быть пустым при полном провале, получен %q", used)
	}
	if len(f.calls) != len(candidatesFor("sda")) {
		t.Errorf("перебрано %d кандидатов из %d: %v", len(f.calls), len(candidatesFor("sda")), f.calls)
	}
	_ = out
}

// Таймаут-ошибка тоже завершает цепочку: следующие -d уже не уложатся в бюджет.
func TestFallback_TimeoutStopsChain(t *testing.T) {
	f := &fakeProbe{errs: map[string]error{"sat": fmt.Errorf("timeout after 1s")}}
	withProbe(t, f)
	_, used := smartctlWithFallback("sda", "/dev/sda", time.Second, "-a")
	if used != "" {
		t.Errorf("usedType при таймауте должен быть пуст, получен %q", used)
	}
	if len(f.calls) != 1 {
		t.Errorf("при таймауте один вызов, было %d: %v", len(f.calls), f.calls)
	}
}

// Главный защитный тест: цепочка обязана укладываться в бюджет независимо
// от числа кандидатов, иначе entware-server (таймаут 300 с) рвёт соединение,
// CGI-процесс убивается, а smartctl остаются сиротами.
func TestFallback_TotalTimeBounded(t *testing.T) {
	const perCall = 40 * time.Millisecond
	// timeout=20ms → бюджет = 2× = 40ms; каждый вызов спит 40ms → успеет
	// пройти максимум 2 кандидата вместо 4.
	f := &fakeProbe{defaultOut: "Read Device Identity failed", delay: perCall}
	withProbe(t, f)

	start := time.Now()
	_, used := smartctlWithFallback("sda", "/dev/sda", 20*time.Millisecond, "-a")
	elapsed := time.Since(start)

	if used != "" {
		t.Errorf("usedType = %q, want \"\"", used)
	}
	if len(f.calls) >= len(candidatesFor("sda")) {
		t.Errorf("бюджет не ограничил цепочку: вызовов %d из %d (%v)", len(f.calls), len(candidatesFor("sda")), f.calls)
	}
	// Бюджет 40ms + допуск на один «висящий» вызов.
	if elapsed > 2*smartFallbackBudget {
		t.Errorf("заняло %v — бюджет не соблюдён", elapsed)
	}
}

// Бюджет = 2× намерения вызывающего (точные действия не должны висеть).
func TestFallback_BudgetFollowsCallerTimeout(t *testing.T) {
	f := &fakeProbe{defaultOut: "Read Device Identity failed", delay: 15 * time.Millisecond}
	withProbe(t, f)
	start := time.Now()
	smartctlWithFallback("sda", "/dev/sda", 10*time.Millisecond, "-a")
	elapsed := time.Since(start)
	if elapsed > 2*time.Second {
		t.Errorf("точечное действие заняло %v при таймауте 10ms", elapsed)
	}
}

func TestIsTimeoutErr(t *testing.T) {
	yes := []error{fmt.Errorf("timeout after 1s"), fmt.Errorf("TIMEOUT"), fmt.Errorf("operation timed out")}
	no := []error{nil, errDeviceBusy, fmt.Errorf("exit status 1"), fmt.Errorf("permission denied")}
	for _, e := range yes {
		if !isTimeoutErr(e) {
			t.Errorf("isTimeoutErr(%v) = false, want true", e)
		}
	}
	for _, e := range no {
		if isTimeoutErr(e) {
			t.Errorf("isTimeoutErr(%v) = true, want false", e)
		}
	}
}
