// Интерактивный режим Telegram-бота (Entware Manager).
//
// Long-polling getUpdates: бот отвечает ТОЛЬКО в chat_id из конфига
// (чужие сообщения молча игнорируются — иначе нашедший бот получил бы
// доступ к роутеру). Команды уровня «только чтение»: статус, температуры,
// IP, службы, SMART, лог.
package telegram

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// tgUpdate — одно обновление из getUpdates.
type tgUpdate struct {
	UpdateID int64 `json:"update_id"`
	Message  struct {
		Text string `json:"text"`
		Chat struct {
			ID int64 `json:"id"`
		} `json:"chat"`
	} `json:"message"`
}

// pollInterval — пауза между опросами, когда нет новых обновлений.
const pollInterval = 3 * time.Second

// BotRun запускает интерактивный режим (блокирующий цикл long-polling).
// Возвращает ошибку только если бот не может стартовать (нет токена/chat_id).
func logInfo(format string, args ...interface{}) {
	ts := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(os.Stderr, "[%s] [INFO] "+format+"\n", append([]interface{}{ts}, args...)...)
}
func BotRun() error {
	cfg := LoadConfig()
	if !cfg.Configured || cfg.ChatID == "" || !cfg.BotEnabled {
		return fmt.Errorf("bot disabled or not configured (enabled/bot_enabled/token/chat_id)")
	}
	logInfo("bot started (chat=%s)", cfg.ChatID)
	cmds := defaultCommands()
	offset := int64(0)
	idle := false
	for {
		// Перечитываем конфиг каждый цикл: смена токена/галочки «Чат-бот»
		// применяется на лету, без перезапуска.
		cfg = LoadConfig()
		if !cfg.Configured || cfg.ChatID == "" || !cfg.BotEnabled {
			if !idle {
				logInfo("bot idle (disabled)")
				idle = true
			}
			time.Sleep(15 * time.Second)
			continue
		}
		idle = false
		updates, err := fetchUpdates(cfg, offset)
		if err != nil {
			logErr("getUpdates: %s", redactURL(err.Error(), cfg.BotToken))
			time.Sleep(10 * time.Second)
			continue
		}
		for _, u := range updates {
			if u.UpdateID >= offset {
				offset = u.UpdateID + 1
			}
			if !allowedChat(u, cfg.ChatID) {
				continue
			}
			if reply := replyFor(u.Message.Text, cmds); reply != "" {
				SendMessage(cfg, reply)
			}
		}
		time.Sleep(pollInterval)
	}
}

// allowedChat — сообщение должно прийти ровно из настроенного chat_id.
func allowedChat(u tgUpdate, chatID string) bool {
	if u.Message.Text == "" {
		return false
	}
	return strconv.FormatInt(u.Message.Chat.ID, 10) == chatID
}

// replyFor — маршрутизация команды. Возвращает текст ответа ("" — молчать).
func replyFor(text string, cmds map[string]func() string) string {
	fields := strings.Fields(strings.TrimSpace(text))
	if len(fields) == 0 {
		return ""
	}
	cmd := strings.ToLower(fields[0])
	// "/status@MyBot" → "/status"
	if i := strings.IndexByte(cmd, '@'); i > 0 {
		cmd = cmd[:i]
	}
	if fn, ok := cmds[cmd]; ok {
		return fn()
	}
	if strings.HasPrefix(cmd, "/") {
		return "Неизвестная команда. Список: /help"
	}
	return ""
}

// defaultCommands — карта команд уровня «только чтение».
func defaultCommands() map[string]func() string {
	return map[string]func() string{
		"/start":    cmdHelp,
		"/help":     cmdHelp,
		"/status":   cmdStatus,
		"/temp":     cmdTemp,
		"/ip":       cmdIP,
		"/services": cmdServices,
		"/smart":    cmdSmart,
		"/log":      cmdLog,
	}
}

func cmdHelp() string {
	return "🤖 Entware Manager — команды:\n" +
		"/status — аптайм, нагрузка, память, диск\n" +
		"/temp — температуры CPU/WiFi\n" +
		"/ip — внешний IP и интерфейс\n" +
		"/services — статусы служб\n" +
		"/smart — здоровье дисков\n" +
		"/log [N] — последние N строк лога (по умолчанию 15)"
}

// --- сбор данных ---

func readLine(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func cmdStatus() string {
	var b strings.Builder

	upSec := 0
	fmt.Sscanf(readLine("/proc/uptime"), "%d", &upSec)
	days := upSec / 86400
	hours := (upSec % 86400) / 3600
	mins := (upSec % 3600) / 60
	b.WriteString(fmt.Sprintf("⏱ Аптайм: %dд %dч %dм\n", days, hours, mins))

	load := readLine("/proc/loadavg")
	if parts := strings.Fields(load); len(parts) >= 3 {
		b.WriteString(fmt.Sprintf("📊 Нагрузка: %s %s %s\n", parts[0], parts[1], parts[2]))
	}

	memTotal, memAvail := parseMeminfo(readLine("/proc/meminfo"))
	if memTotal > 0 {
		used := memTotal - memAvail
		pct := 100 * used / memTotal
		b.WriteString(fmt.Sprintf("🧠 RAM: %d/%d МБ (%d%%)\n", used/1024, memTotal/1024, pct))
	}

	b.WriteString("💾 Диски:\n" + parseDfRoot(runDF()))

	return b.String()
}

// parseMeminfo — MemTotal и MemAvailable в КБ.
func parseMeminfo(meminfo string) (total, avail int) {
	for _, ln := range strings.Split(meminfo, "\n") {
		f := strings.Fields(ln)
		if len(f) < 2 {
			continue
		}
		v, _ := strconv.Atoi(f[1])
		switch f[0] {
		case "MemTotal:":
			total = v
		case "MemAvailable:":
			avail = v
		}
	}
	return total, avail
}

// runDF — вывод df -h для корня.
func runDF() string {
	out, err := exec.Command("df", "-h").Output()
	if err != nil {
		return ""
	}
	return string(out)
}

// parseDfRoot — строки df для корня "/" и /opt (Entware).
// Внутренний / у Keenetic — read-only squashfs, часто 100%: это норма.
func parseDfRoot(dfOut string) string {
	var out []string
	for _, ln := range strings.Split(dfOut, "\n") {
		f := strings.Fields(ln)
		if len(f) >= 6 && (f[len(f)-1] == "/" || f[len(f)-1] == "/opt") {
			out = append(out, fmt.Sprintf("  %s: занято %s из %s (%s)\n", f[len(f)-1], f[2], f[1], f[4]))
		}
	}
	if len(out) == 0 {
		return "  нет данных\n"
	}
	return strings.Join(out, "")
}

// cmdTemp — максимум по всем hwmon-датчикам (mili°C).
func cmdTemp() string {
	best := -1000
	var names []string
	matches, _ := filepath.Glob("/sys/class/hwmon/hwmon*/temp*_input")
	for _, m := range matches {
		b, err := os.ReadFile(m)
		if err != nil {
			continue
		}
		if v, err := strconv.Atoi(strings.TrimSpace(string(b))); err == nil {
			c := v / 1000
			if c > best {
				best = c
			}
			names = append(names, fmt.Sprintf("%d°C", c))
		}
	}
	if best == -1000 {
		return "Температуры: нет датчиков hwmon"
	}
	sort.Strings(names)
	return fmt.Sprintf("🌡 Датчики: %s (макс %d°C)", strings.Join(names, ", "), best)
}

// cmdIP — интерфейс с дефолтным маршрутом + его IPv4.
func cmdIP() string {
	iface := defaultIface(readLine("/proc/net/route"))
	if iface == "" {
		return "Сеть: дефолтный маршрут не найден"
	}
	ip := ifaceIPv4(iface)
	if ip == "" {
		return fmt.Sprintf("Сеть: %s (адрес не определён)", iface)
	}
	return fmt.Sprintf("🌐 IP: %s (интерфейс %s)", ip, iface)
}

// defaultIface — имя интерфейса с destination 00000000 из /proc/net/route.
func defaultIface(routeTable string) string {
	for i, ln := range strings.Split(routeTable, "\n") {
		if i == 0 {
			continue // заголовок
		}
		f := strings.Fields(ln)
		if len(f) > 1 && f[1] == "00000000" {
			return f[0]
		}
	}
	return ""
}

// ifaceIPv4 — первый IPv4-адрес интерфейса.
func ifaceIPv4(name string) string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, ifc := range ifaces {
		if ifc.Name != name {
			continue
		}
		addrs, err := ifc.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			s := a.String()
			if i := strings.IndexByte(s, '/'); i > 0 {
				s = s[:i]
			}
			if strings.Count(s, ".") == 3 {
				return s
			}
		}
	}
	return ""
}

// cmdServices — статусы по pid-файлам /opt/var/run/*.pid.
func cmdServices() string {
	matches, _ := filepath.Glob("/opt/var/run/*.pid")
	if len(matches) == 0 {
		return "Службы: pid-файлы не найдены"
	}
	var b strings.Builder
	for _, m := range matches {
		name := strings.TrimSuffix(filepath.Base(m), ".pid")
		pid := strings.TrimSpace(readLine(m))
		state := "🔴 остановлен"
		if pid != "" {
			if _, err := os.Stat("/proc/" + pid); err == nil {
				state = "🟢 работает (PID " + pid + ")"
			}
		}
		b.WriteString(fmt.Sprintf("%s — %s\n", name, state))
	}
	return b.String()
}

// cmdSmart — краткое здоровье дисков (best-effort, таймаут на каждый).
func cmdSmart() string {
	sm, err := exec.LookPath("smartctl")
	if err != nil {
		if _, e2 := os.Stat("/opt/sbin/smartctl"); e2 != nil {
			return "SMART: smartctl не установлен"
		}
		sm = "/opt/sbin/smartctl"
	}
	var out []string
	for _, dev := range []string{"/dev/sda", "/dev/sdb", "/dev/sdc", "/dev/sdd"} {
		if _, err := os.Stat(dev); err != nil {
			continue
		}
		ctx := exec.Command(sm, "-H", dev)
		ctx.Stdout = nil
		b, err := ctx.Output()
		res := "нет ответа"
		if err == nil {
			s := string(b)
			switch {
			case strings.Contains(s, "PASSED"):
				res = "✅ OK"
			case strings.Contains(s, "FAILED"):
				res = "❌ FAILED"
			default:
				res = "⚠️ см. панель"
			}
		}
		out = append(out, fmt.Sprintf("%s — %s", dev, res))
	}
	if len(out) == 0 {
		return "SMART: диски не найдены"
	}
	return "💾 SMART:\n" + strings.Join(out, "\n")
}

// cmdLog — хвост суточного лога EM; если за сегодня записей ещё нет —
// самый свежий существующий файл.
func cmdLog() string {
	n := 15
	dir := "/tmp/entware/logs"
	path := filepath.Join(dir, time.Now().Format("2006-01-02")+".log")
	if _, err := os.Stat(path); err != nil {
		best, bestMod := "", time.Time{}
		matches, _ := filepath.Glob(filepath.Join(dir, "20*.log"))
		for _, m := range matches {
			if fi, err := os.Stat(m); err == nil && fi.ModTime().After(bestMod) {
				best, bestMod = m, fi.ModTime()
			}
		}
		path = best
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "Логи отсутствуют"
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	name := filepath.Base(path)
	return "📜 " + name + " (последние записи):\n" + strings.Join(lines, "\n")
}

// --- Telegram API: getUpdates ---

func fetchUpdates(cfg Config, offset int64) ([]tgUpdate, error) {
	q := url.Values{}
	q.Set("timeout", "2")
	if offset > 0 {
		q.Set("offset", strconv.FormatInt(offset, 10))
	}
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?%s", cfg.BotToken, q.Encode())

	client := httpClient(cfg)
	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("%s", redactURL(err.Error(), cfg.BotToken))
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("getUpdates HTTP %d", resp.StatusCode)
	}
	var wrap struct {
		OK     bool       `json:"ok"`
		Result []tgUpdate `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrap); err != nil {
		return nil, err
	}
	if !wrap.OK {
		return nil, fmt.Errorf("getUpdates ok=false")
	}
	return wrap.Result, nil
}
