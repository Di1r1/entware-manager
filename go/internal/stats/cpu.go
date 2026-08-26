// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
//
// Карточка CPU на странице Статистики: двухточечный замер загрузки
// (/proc/stat, окно 350 мс — паттерн /top Telegram-бота), средняя нагрузка
// и топ-5 процессов по CPU за то же окно. Чистый /proc, без внешних команд.
package stats

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const cpuSampleInterval = 350 * time.Millisecond

type CPUInfo struct {
	Percent int
	Class   string
	TextCls string
	Load    string // «0,42 / 0,38 / 0,35» (LA 1/5/15)
	Top     []TopProcCPU
}

type TopProcCPU struct {
	Cmd  string
	Pcts string // «12%» или «<1%»
}

// parseCPUTotal — сумма всех счётчиков и idle(+iowait) первой строки /proc/stat.
func parseCPUTotal(line string) (total, idle uint64) {
	if !strings.HasPrefix(line, "cpu ") {
		return 0, 0
	}
	fields := strings.Fields(line)[1:]
	var vals []uint64
	for _, f := range fields {
		v, err := strconv.ParseUint(f, 10, 64)
		if err != nil {
			return 0, 0
		}
		vals = append(vals, v)
	}
	if len(vals) < 4 {
		return 0, 0
	}
	for _, v := range vals {
		total += v
	}
	idle = vals[3]
	if len(vals) > 4 {
		idle += vals[4]
	}
	return total, idle
}

// formatLoadavg — первые три поля /proc/loadavg через « / ».
func formatLoadavg(data string) string {
	f := strings.Fields(data)
	if len(f) < 3 {
		return "н/д"
	}
	return f[0] + " / " + f[1] + " / " + f[2]
}

// parseProcStatTicks — utime+stime из /proc/<pid>/stat (поля 14+15,
// после последней ')' — имя процесса может содержать скобки).
func parseProcStatTicks(data []byte) uint64 {
	s := string(data)
	i := strings.LastIndex(s, ")")
	if i < 0 {
		return 0
	}
	f := strings.Fields(s[i+1:])
	if len(f) <= 12 {
		return 0
	}
	u, _ := strconv.ParseUint(f[11], 10, 64)
	st, _ := strconv.ParseUint(f[12], 10, 64)
	return u + st
}

type procTick struct {
	cmd   string
	ticks uint64
}

// readCPUTicksAll — суммарные счётчики ядра.
func readCPUTicksAll() (total, idle uint64) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, 0
	}
	return parseCPUTotal(strings.SplitN(string(data), "\n", 2)[0])
}

// snapshotProcTicks — utime+stime всех процессов с осмысленным cmdline
// (ядро-треды исключены). Один проход по /proc.
func snapshotProcTicks() map[int]procTick {
	out := map[int]procTick{}
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return out
	}
	scanned := 0
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil || pid <= 0 || scanned >= 4096 {
			continue
		}
		scanned++
		argv0 := readNullTerminated(fmt.Sprintf("/proc/%d/cmdline", pid))
		if argv0 == "" {
			continue
		}
		data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
		if err != nil {
			continue
		}
		ticks := parseProcStatTicks(data)
		if ticks == 0 {
			continue
		}
		out[pid] = procTick{cmd: filepath.Base(argv0), ticks: ticks}
	}
	return out
}

// collectCPU — двухточечный замер: % загрузки ядра, LA и топ процессов.
// При недоступности /proc возвращает нулевой CPUInfo (карточка покажет «н/д»).
func collectCPU() CPUInfo {
	info := CPUInfo{Load: "н/д"}
	if data, err := os.ReadFile("/proc/loadavg"); err == nil {
		info.Load = formatLoadavg(string(data))
	}

	t1, i1 := readCPUTicksAll()
	p1 := snapshotProcTicks()
	time.Sleep(cpuSampleInterval)
	t2, i2 := readCPUTicksAll()
	p2 := snapshotProcTicks()

	dt := t2 - t1
	if t1 == 0 || t2 == 0 || dt == 0 {
		return info
	}
	busy := int64(dt) - int64(i2-i1)
	if busy < 0 {
		busy = 0
	}
	info.Percent = int(busy * 100 / int64(dt))
	info.Class = percentClass(info.Percent)
	info.TextCls = "stat-value-" + info.Class

	type procDelta struct {
		name string
		pct  float64
	}
	var deltas []procDelta
	for pid, s2 := range p2 {
		s1, ok := p1[pid]
		if !ok || s2.ticks < s1.ticks {
			continue
		}
		deltas = append(deltas, procDelta{s2.cmd, float64(s2.ticks-s1.ticks) * 100 / float64(dt)})
	}
	sort.Slice(deltas, func(i, j int) bool { return deltas[i].pct > deltas[j].pct })
	for _, d := range deltas {
		if len(info.Top) >= 5 {
			break
		}
		if d.pct < 0.5 {
			break // дальше только нули — список отсортирован
		}
		pcts := strconv.Itoa(int(d.pct+0.5)) + "%"
		if d.pct < 1 {
			pcts = "<1%"
		}
		info.Top = append(info.Top, TopProcCPU{Cmd: d.name, Pcts: pcts})
	}
	return info
}

// HandleCPU — GET: HTML-фрагмент содержимого карточки (без обёртки) для
// живого обновления каждые 5 секунд. Сессия проверяется гейтом go.cgi/server.
func HandleCPU() {
	if os.Getenv("REQUEST_METHOD") != "GET" {
		fmt.Print("Content-type: text/plain; charset=utf-8\n\nMethod not allowed")
		return
	}
	cpu := collectCPU()
	fmt.Print("Content-type: text/html; charset=utf-8\n\n")
	var b strings.Builder
	renderCPUCardInner(&b, cpu)
	fmt.Print(b.String())
}

func renderCPUCard(b *strings.Builder, cpu CPUInfo) {
	b.WriteString(`<div class="stat-card cpu" id="cpuCard">`)
	renderCPUCardInner(b, cpu)
	b.WriteString(`</div>`)
}

// renderCPUCardInner — содержимое карточки без внешнего div: то же самое
// отдаёт эндпоинт cpu.cgi для живого обновления каждые 5 секунд.
func renderCPUCardInner(b *strings.Builder, cpu CPUInfo) {
	b.WriteString(fmt.Sprintf(`
        <h3><span class="stat-icon"><svg class="icon" width="24" height="24"><use href="/entware-manager/icons.svg?v=3#icon-chart"/></svg></span>Процессор (CPU)</h3>
        <table class="stat-table">
            <tr><td>Загрузка:</td><td><span class="%s">%s</span></td></tr>
            <tr><td>Средняя нагрузка:</td><td>%s</td></tr>
        </table>
        %s
        %s`, cls(cpu), percentStr(cpu), esc(cpu.Load), bar(cpu), top(cpu)))
}

func cls(cpu CPUInfo) string {
	if cpu.Percent > 0 || cpu.Load != "н/д" {
		return cpu.TextCls
	}
	return "normal"
}

func percentStr(cpu CPUInfo) string {
	if cpu.Percent > 0 || cpu.Load != "н/д" {
		return strconv.Itoa(cpu.Percent) + "%"
	}
	return "н/д"
}

func bar(cpu CPUInfo) string {
	if cpu.Percent > 0 || cpu.Load != "н/д" {
		return `<div class="progress-bar"><div class="progress-bar-fill fill-` + cpu.Class + `" style="width: ` + strconv.Itoa(cpu.Percent) + `%;"></div></div>`
	}
	return ""
}

func top(cpu CPUInfo) string {
	if len(cpu.Top) == 0 {
		return ""
	}
	var t strings.Builder
	t.WriteString(`<div class="top-mem-wrapper top-mem-` + cpu.Class + `"><table class="top-mem"><tr><th colspan="2">Топ по CPU</th></tr>`)
	for _, p := range cpu.Top {
		t.WriteString("<tr><td>" + esc(p.Cmd) + "</td><td>" + p.Pcts + "</td></tr>\n")
	}
	t.WriteString(`</table></div>`)
	return t.String()
}
