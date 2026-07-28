package stats

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

type SysInfo struct {
	Model    string
	Hostname string
	Arch     string
	Kernel   string
	Uptime   string
}

type MemInfo struct {
	Info    string
	Percent int
	Class   string
	TextCls string
}

type TopProc struct {
	Cmd string
	RSS int
}

type DiskInfo struct {
	Size    string
	Used    string
	Avail   string
	UsePct  string
	Mount   string
	UseNum  int
	Class   string
	TextCls string
}

type DFEntry struct {
	FS     string
	Size   string
	Used   string
	Avail  string
	UsePct string
	UseNum int
	Class  string
	Mount  string
}

type PkgChange struct {
	Icon   string
	Cls    string
	Pkg    string
	Action string
	TS     string
}

func Handle() {
	if os.Getenv("REQUEST_METHOD") != "GET" {
		writeHTML(`<p class="error">Method not allowed</p>`)
		return
	}

	sys := collectSysInfo()
	mem, topProcs := collectMemInfo()
	pkgInstalled, pkgAvail := collectPkgCounts()
	changes := collectPkgChanges()
	disk := collectDiskInfo()
	tmpfs, block := collectDF()

	html := renderHTML(sys, mem, topProcs, pkgInstalled, pkgAvail, changes, disk, tmpfs, block)
	writeHTML(html)
}

func collectSysInfo() SysInfo {
	model := readNullTerminated("/proc/device-tree/model")
	if model == "" {
		if b, err := os.ReadFile("/tmp/sysinfo/model"); err == nil {
			model = strings.TrimSpace(string(b))
		}
	}
	if model == "" {
		if b, err := os.ReadFile("/etc/openwrt_release"); err == nil {
			for _, line := range strings.Split(string(b), "\n") {
				if strings.HasPrefix(line, "DISTRIB_DESCRIPTION=") {
					model = strings.Trim(line[20:], "'\" \t\r\n")
					break
				}
			}
		}
	}
	if model == "" {
		if b, err := os.ReadFile("/proc/cpuinfo"); err == nil {
			for _, line := range strings.Split(string(b), "\n") {
				if strings.HasPrefix(line, "machine") || strings.HasPrefix(line, "system type") || strings.HasPrefix(line, "model name") {
					parts := strings.SplitN(line, ":", 2)
					if len(parts) == 2 {
						model = strings.TrimSpace(parts[1])
						break
					}
				}
			}
		}
	}
	if model == "" {
		if b, err := os.ReadFile("/etc/product"); err == nil {
			model = strings.TrimSpace(string(b))
		}
	}
	if model == "" {
		if resp, err := http.Get("http://127.0.0.1:79/rci/show/system/"); err == nil {
			if b, err := io.ReadAll(resp.Body); err == nil {
				resp.Body.Close()
				var parsed map[string]any
				if json.Unmarshal(b, &parsed) == nil {
					if m, ok := parsed["model"].(string); ok && m != "" {
						model = m
					}
				}
			} else {
				resp.Body.Close()
			}
		}
	}
	if model == "" {
		model = "н/д"
	}

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "н/д"
	}

	arch := detectArch()
	if arch == "" {
		arch = "н/д"
	}

	kernel := runCmd("uname", "-r")
	if kernel == "" {
		kernel = "н/д"
	}

	uptime := parseUptime()
	return SysInfo{Model: model, Hostname: hostname, Arch: arch, Kernel: kernel, Uptime: uptime}
}

func detectArch() string {
	out := runCmd("opkg", "print-architecture")
	if out != "" {
		for _, line := range strings.Split(out, "\n") {
			parts := strings.Fields(line)
			if len(parts) >= 2 && parts[0] == "arch" {
				a := parts[1]
				if a == "all" || a == "noarch" {
					continue
				}
				switch a {
				case "aarch64":
					return "arm64"
				case "x86_64":
					return "amd64"
				case "i386", "i486", "i586", "i686":
					return "386"
				}
				return a
			}
		}
	}
	a := runCmd("uname", "-m")
	if a == "mips" {
		elf := "/opt/bin/opkg"
		if _, err := os.Stat(elf); err != nil {
			elf = "/bin/sh"
		}
		if data, err := os.ReadFile(elf); err == nil && len(data) > 5 {
			if data[5] == 1 {
				return "mipsel"
			}
		}
		return "mips"
	}
	return a
}

func collectMemInfo() (MemInfo, []TopProc) {
	total, avail := 0, 0
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return MemInfo{Info: "н/д"}, nil
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "MemTotal:") {
			total = parseIntField(line)
		} else if strings.HasPrefix(line, "MemAvailable:") {
			avail = parseIntField(line)
		} else if avail == 0 && strings.HasPrefix(line, "MemFree:") {
			avail = parseIntField(line)
		}
	}
	if total == 0 {
		return MemInfo{Info: "н/д"}, nil
	}
	used := total - avail
	usedMB := used / 1024
	totalMB := total / 1024
	pct := used * 100 / total
	cls := percentClass(pct)

	topProcs := collectTopProcs(6)

	return MemInfo{
		Info:    fmt.Sprintf("%d MB / %d MB", usedMB, totalMB),
		Percent: pct,
		Class:   cls,
		TextCls: "stat-value-" + cls,
	}, topProcs
}

func collectTopProcs(n int) []TopProc {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	var procs []TopProc
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		name, rss := readProcStatus(pid)
		if rss > 0 {
			procs = append(procs, TopProc{Cmd: name, RSS: rss})
		}
	}
	sort.Slice(procs, func(i, j int) bool {
		return procs[i].RSS > procs[j].RSS
	})
	if len(procs) > n {
		procs = procs[:n]
	}
	return procs
}

func collectPkgCounts() (installed, avail int) {
	out := runCmd("/opt/bin/opkg", "list-installed")
	if out != "" {
		installed = len(strings.Split(out, "\n"))
	}
	out = runCmd("/opt/bin/opkg", "list")
	if out != "" {
		avail = len(strings.Split(out, "\n"))
	}
	return
}

func collectPkgChanges() []PkgChange {
	data, err := os.ReadFile("/opt/var/log/package_changes.log")
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		return nil
	}
	if len(lines) > 6 {
		lines = lines[len(lines)-6:]
	}
	var changes []PkgChange
	for _, line := range lines {
		parts := strings.SplitN(line, " | ", 4)
		if len(parts) < 4 {
			continue
		}
		ts, act, pkg, st := parts[0], parts[1], parts[2], parts[3]
		icon, cls := "?", ""
		actRu := "обновлён"
		if st == "success" {
			icon = "✓"
			cls = "pkg-change-ok"
		} else if st == "error" {
			icon = "✗"
			cls = "pkg-change-error"
		}
		switch act {
		case "install":
			if st == "error" {
				actRu = "ошибка установки"
			} else {
				actRu = "установлен"
			}
		case "remove":
			if st == "error" {
				actRu = "ошибка удаления"
			} else {
				actRu = "удалён"
			}
		default:
			if st == "error" {
				actRu = "ошибка обновления"
			} else {
				actRu = "обновлён"
			}
		}
		changes = append(changes, PkgChange{Icon: icon, Cls: cls, Pkg: pkg, Action: actRu, TS: ts})
	}
	return changes
}

func collectDiskInfo() DiskInfo {
	out := runCmd("df", "-h", "/opt")
	lines := strings.Split(out, "\n")
	if len(lines) < 2 {
		return DiskInfo{Size: "н/д", Used: "н/д", Avail: "н/д", Mount: "/opt"}
	}
	parts := strings.Fields(lines[len(lines)-1])
	if len(parts) < 6 {
		return DiskInfo{Size: "н/д", Used: "н/д", Mount: "/opt"}
	}
	useStr := strings.TrimSuffix(parts[4], "%")
	useNum, _ := strconv.Atoi(useStr)
	cls := percentClass(useNum)
	return DiskInfo{
		Size: parts[1], Used: parts[2], Avail: parts[3],
		UsePct: parts[4], Mount: parts[5], UseNum: useNum,
		Class: cls, TextCls: "stat-value-" + cls,
	}
}

func collectDF() (tmpfs, block []DFEntry) {
	out := runCmd("df", "-h")
	if out == "" {
		return nil, nil
	}
	for _, line := range strings.Split(out, "\n") {
		parts := strings.Fields(line)
		if len(parts) < 6 {
			continue
		}
		fs := parts[0]
		if !strings.HasPrefix(fs, "tmpfs") && !strings.HasPrefix(fs, "/dev/") {
			continue
		}
		useStr := strings.TrimSuffix(parts[4], "%")
		useNum, _ := strconv.Atoi(useStr)
		cls := percentClass(useNum)
		e := DFEntry{
			FS: fs, Size: parts[1], Used: parts[2],
			Avail: parts[3], UsePct: parts[4], UseNum: useNum,
			Class: cls, Mount: parts[5],
		}
		if strings.HasPrefix(fs, "tmpfs") {
			tmpfs = append(tmpfs, e)
		} else {
			block = append(block, e)
		}
	}
	return
}

func renderHTML(sys SysInfo, mem MemInfo, topProcs []TopProc, pkgInstalled, pkgAvail int, changes []PkgChange, disk DiskInfo, tmpfs, block []DFEntry) string {
	var b strings.Builder

	b.WriteString(`<h2 style="display: flex; align-items: center; gap: 10px;">
    <span class="stat-icon" style="width: 32px; height: 32px;">
        <svg class="icon" width="32" height="32">
            <use href="/entware-manager/icons.svg?v=2#icon-stats"/>
        </svg>
    </span>
    Статистика системы
</h2>
<div class="stats-grid">`)

	renderSysCard(&b, sys)
	renderMemCard(&b, mem, topProcs)
	renderPkgCard(&b, pkgInstalled, pkgAvail, changes)
	renderDiskCard(&b, disk)

	b.WriteString(`</div>`)

	renderNetworkPlaceholder(&b)
	renderTable(&b, "tmpfs", "tmpfs", "folder", tmpfs)
	renderTable(&b, "storage", "Блочные устройства", "disk", block)

	return b.String()
}

func renderSysCard(b *strings.Builder, sys SysInfo) {
	b.WriteString(fmt.Sprintf(`
    <div class="stat-card system">
        <h3><span class="stat-icon"><svg class="icon" width="24" height="24"><use href="/entware-manager/icons.svg?v=2#icon-stats"/></svg></span>Система</h3>
        <table class="stat-table">
            <tr><td>Модель:</td><td>%s</td></tr>
            <tr><td>Имя хоста:</td><td>%s</td></tr>
            <tr><td>Архитектура:</td><td>%s</td></tr>
            <tr><td>Версия ядра:</td><td>%s</td></tr>
            <tr><td>Время работы:</td><td>%s</td></tr>
        </table>
    </div>`, esc(sys.Model), esc(sys.Hostname), esc(sys.Arch), esc(sys.Kernel), esc(sys.Uptime)))
}

func renderMemCard(b *strings.Builder, mem MemInfo, topProcs []TopProc) {
	var topHTML string
	if len(topProcs) > 0 {
		var t strings.Builder
		t.WriteString(`<div class="top-mem-wrapper top-mem-` + mem.Class + `"><table class="top-mem"><tr><th colspan="2">Топ по памяти</th></tr>`)
		for _, p := range topProcs {
			if p.RSS >= 1024 {
				t.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%d MB</td></tr>\n", esc(p.Cmd), p.RSS/1024))
			} else {
				t.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%d KB</td></tr>\n", esc(p.Cmd), p.RSS))
			}
		}
		t.WriteString(`</table></div>`)
		topHTML = t.String()
	}

	b.WriteString(fmt.Sprintf(`
    <div class="stat-card memory">
        <h3><span class="stat-icon"><svg class="icon" width="24" height="24"><use href="/entware-manager/icons.svg?v=2#icon-memory"/></svg></span>Память (RAM)</h3>
        <table class="stat-table">
            <tr><td>Использовано / Всего:</td><td>%s</td></tr>
            <tr><td>Загрузка:</td><td><span class="%s">%d%%</span></td></tr>
        </table>
        <div class="progress-bar"><div class="progress-bar-fill fill-%s" style="width: %d%%;"></div></div>
        %s
    </div>`, esc(mem.Info), mem.TextCls, mem.Percent, mem.Class, mem.Percent, topHTML))
}

func renderPkgCard(b *strings.Builder, installed, avail int, changes []PkgChange) {
	instStr := strconv.Itoa(installed)
	availStr := strconv.Itoa(avail)
	if installed == 0 {
		instStr = "н/д"
		availStr = "н/д"
	}

	var changesHTML string
	if len(changes) > 0 {
		var t strings.Builder
		t.WriteString(`<div class="top-mem-wrapper"><table class="top-mem"><tr><th colspan="4">Последние изменения</th></tr>`)
		for _, c := range changes {
			t.WriteString(fmt.Sprintf(`<tr class="%s"><td class="pkg-change-icon">%s</td><td class="pkg-change-pkg">%s</td><td class="pkg-change-act">%s</td><td class="pkg-change-ts">%s</td></tr>`, c.Cls, c.Icon, esc(c.Pkg), esc(c.Action), esc(c.TS)))
		}
		t.WriteString(`</table></div>`)
		changesHTML = t.String()
	}

	b.WriteString(fmt.Sprintf(`
    <div class="stat-card packages">
        <h3><span class="stat-icon"><svg class="icon" width="24" height="24"><use href="/entware-manager/icons.svg?v=2#icon-package"/></svg></span>Пакеты Entware</h3>
        <table class="stat-table">
            <tr><td>Установлено:</td><td>%s</td></tr>
            <tr><td>Доступно:</td><td>%s</td></tr>
        </table>
        %s
    </div>`, instStr, availStr, changesHTML))
}

func renderDiskCard(b *strings.Builder, disk DiskInfo) {
	b.WriteString(fmt.Sprintf(`
    <div class="stat-card disk">
        <h3><span class="stat-icon"><svg class="icon" width="24" height="24"><use href="/entware-manager/icons.svg?v=2#icon-disk"/></svg></span>Диск (/opt)</h3>
        <table class="stat-table">
            <tr><td>Размер:</td><td>%s</td></tr>
            <tr><td>Использовано:</td><td>%s</td></tr>
            <tr><td>Доступно:</td><td>%s</td></tr>
            <tr><td>Загрузка:</td><td><span class="%s">%s</span></td></tr>
        </table>
        <div class="progress-bar"><div class="progress-bar-fill fill-%s" style="width: %d%%;"></div></div>
    </div>`, esc(disk.Size), esc(disk.Used), esc(disk.Avail), disk.TextCls, esc(disk.UsePct), disk.Class, disk.UseNum))
}

func renderNetworkPlaceholder(b *strings.Builder) {
	b.WriteString(`
<div class="stat-card network" id="networkCard">
    <h3><span class="stat-icon"><svg class="icon" width="24" height="24"><use href="/entware-manager/icons.svg?v=2#icon-router"/></svg></span>Сеть</h3>
    <div id="networkTable"><div style="padding: 0.5rem 1rem;">Загрузка...</div></div>
    <div style="margin-top: 8px; display: flex; gap: 8px;">
        <button id="network-refresh" class="packages-delete-btn" style="padding: 4px 8px; font-size: 12px;">
            <svg class="icon" width="14" height="14"><use href="/entware-manager/icons.svg?v=2#icon-refresh"/></svg>
        </button>
    </div>
</div>`)
}

func renderTable(b *strings.Builder, id, title, icon string, entries []DFEntry) {
	var rows strings.Builder
	if len(entries) == 0 {
		rows.WriteString("<tr><td colspan='6'>Нет данных</td></tr>")
	} else {
		for _, e := range entries {
			rows.WriteString(fmt.Sprintf(`<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td><span class="stat-value-%s">%s</span></td><td><a href="/entware-cgi/tmpfs.cgi?path=%s" style="text-decoration:none; color:inherit;">%s</a></td></tr>`,
				esc(e.FS), esc(e.Size), esc(e.Used), esc(e.Avail),
				e.Class, esc(e.UsePct), e.Mount, esc(e.Mount)))
		}
	}

	b.WriteString(fmt.Sprintf(`
<div class="stat-card %s">
    <h3><span class="stat-icon"><svg class="icon" width="24" height="24"><use href="/entware-manager/icons.svg?v=2#icon-%s"/></svg></span>%s</h3>
    <div class="table-wrapper">
        <table>
            <thead><tr><th>ФС</th><th>Размер</th><th>Использовано</th><th>Доступно</th><th>Загрузка</th><th>Точка монтирования</th></tr></thead>
            <tbody>%s</tbody>
        </table>
    </div>
</div>`, id, icon, title, rows.String()))
}

func esc(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

func writeHTML(html string) {
	fmt.Println("Content-type: text/html; charset=utf-8\n")
	fmt.Print(html)
}

func percentClass(pct int) string {
	if pct > 90 {
		return "critical"
	}
	if pct > 70 {
		return "warning"
	}
	return "normal"
}

func runCmd(name string, args ...string) string {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func readNullTerminated(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	idx := strings.IndexByte(string(data), 0)
	if idx >= 0 {
		return string(data[:idx])
	}
	return strings.TrimSpace(string(data))
}

func parseIntField(line string) int {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return 0
	}
	v, _ := strconv.Atoi(parts[1])
	return v
}

func readProcStatus(pid int) (name string, rss int) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return "", 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "Name:") {
			name = strings.TrimSpace(line[5:])
		} else if strings.HasPrefix(line, "VmRSS:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				rss, _ = strconv.Atoi(parts[1])
			}
		}
	}
	return
}

func parseUptime() string {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return "н/д"
	}
	parts := strings.Fields(string(data))
	if len(parts) == 0 {
		return "н/д"
	}
	secs, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return "н/д"
	}
	d := int(secs) / 86400
	h := (int(secs) % 86400) / 3600
	m := (int(secs) % 3600) / 60
	if d > 0 {
		return fmt.Sprintf("%d day %d:%02d", d, h, m)
	}
	return fmt.Sprintf("%d:%02d", h, m)
}
