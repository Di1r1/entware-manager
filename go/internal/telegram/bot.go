// Интерактивный режим Telegram-бота (Entware Manager).
//
// Long-polling getUpdates: бот отвечает ТОЛЬКО в chat_id из конфига
// (чужие сообщения молча игнорируются — иначе нашедший бот получил бы
// доступ к роутеру). Команды уровня «только чтение»: статус, температуры,
// IP, службы, SMART, лог.
package telegram

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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
	"sync"
	"time"

	"entware-manager/internal/services"
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
	CallbackQuery *struct {
		ID      string `json:"id"`
		Data    string `json:"data"`
		Message struct {
			Chat struct {
				ID int64 `json:"id"`
			} `json:"chat"`
		} `json:"message"`
	} `json:"callback_query"`
}

// tgReply — ответ бота: текст и опциональные inline-кнопки [label, callbackData].
type tgReply struct {
	text    string
	buttons [][2]string
}

// pendingAction — действие, ожидающее подтверждения через inline-кнопку.
type pendingAction struct {
	desc    string
	run     func() string // выполняется после подтверждения; возвращает результат
	expires time.Time
}

var (
	pendingMu      sync.Mutex
	pendingActions = map[string]pendingAction{}
)

// newNonce — короткий случайный идентификатор подтверждения.
func newNonce() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// putPending регистрирует действие (заодно вычищает просроченные).
func putPending(nonce string, act pendingAction) {
	pendingMu.Lock()
	defer pendingMu.Unlock()
	now := time.Now()
	for k, v := range pendingActions {
		if now.After(v.expires) {
			delete(pendingActions, k)
		}
	}
	pendingActions[nonce] = act
}

// takePending достаёт действие (одноразовое). Просроченное — удаляется
// и считается отсутствующим.
func takePending(nonce string) (pendingAction, bool) {
	pendingMu.Lock()
	defer pendingMu.Unlock()
	act, ok := pendingActions[nonce]
	if !ok {
		return pendingAction{}, false
	}
	delete(pendingActions, nonce)
	if time.Now().After(act.expires) {
		return pendingAction{}, false
	}
	return act, true
}

const confirmTTL = 5 * time.Minute

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
			if u.CallbackQuery != nil {
				handleCallback(cfg, u.CallbackQuery)
				continue
			}
			if !allowedChat(u, cfg.ChatID) {
				continue
			}
			rep := replyFor(u.Message.Text, cmds)
			if rep.text == "" && len(rep.buttons) == 0 {
				continue
			}
			var kb string
			if len(rep.buttons) > 0 {
				kb = buildInlineKB(rep.buttons)
			}
			if kb != "" {
				SendMessageMarkup(cfg, rep.text, kb)
			} else {
				SendMessage(cfg, rep.text)
			}
		}
		time.Sleep(pollInterval)
	}
}

// allowedChat — сообщение или нажатие кнопки должно прийти ровно
// из настроенного chat_id.
func allowedChat(u tgUpdate, chatID string) bool {
	var id int64
	switch {
	case u.CallbackQuery != nil:
		id = u.CallbackQuery.Message.Chat.ID
	case u.Message.Text != "":
		id = u.Message.Chat.ID
	default:
		return false
	}
	return strconv.FormatInt(id, 10) == chatID
}

// replyFor — маршрутизация команды. Возвращает ответ ("" — молчать).
func replyFor(text string, cmds map[string]func([]string) tgReply) tgReply {
	fields := strings.Fields(strings.TrimSpace(text))
	if len(fields) == 0 {
		return tgReply{}
	}
	cmd := strings.ToLower(fields[0])
	// "/status@MyBot" → "/status"
	if i := strings.IndexByte(cmd, '@'); i > 0 {
		cmd = cmd[:i]
	}
	if fn, ok := cmds[cmd]; ok {
		return fn(fields[1:])
	}
	if strings.HasPrefix(cmd, "/") {
		return tgReply{text: "Неизвестная команда. Список: /help"}
	}
	return tgReply{}
}

// defaultCommands — карта команд. Аргументы — слова после команды.
func defaultCommands() map[string]func(args []string) tgReply {
	return map[string]func([]string) tgReply{
		"/start":    func([]string) tgReply { return tgReply{text: cmdHelp()} },
		"/help":     func([]string) tgReply { return tgReply{text: cmdHelp()} },
		"/status":   func([]string) tgReply { return tgReply{text: cmdStatus()} },
		"/temp":     func([]string) tgReply { return tgReply{text: cmdTemp()} },
		"/ip":       func([]string) tgReply { return tgReply{text: cmdIP()} },
		"/services": func([]string) tgReply { return tgReply{text: cmdServices()} },
		"/smart":    func([]string) tgReply { return tgReply{text: cmdSmart()} },
		"/log":      cmdLogCmd,
		// Уровень 2 — управление (подтверждение inline-кнопкой для опасных).
		"/service": cmdService,
		"/pkg":     cmdPkg,
		"/rotate":  func([]string) tgReply { return tgReply{text: execRotate()} },
		"/reboot":  cmdReboot,
	}
}

func cmdHelp() string {
	return "🤖 Entware Manager — команды:\n" +
		"📖 Информация:\n" +
		"/status — аптайм, нагрузка, память, диск\n" +
		"/temp — температуры CPU/WiFi\n" +
		"/ip — внешний IP и интерфейс\n" +
		"/services — статусы служб\n" +
		"/smart — здоровье дисков\n" +
		"/log [N] — последние N строк лога (по умолчанию 15)\n" +
		"⚙️ Управление (с подтверждением):\n" +
		"/service <имя> start|stop|restart — управление службой\n" +
		"/pkg update — обновить списки пакетов\n" +
		"/rotate — ротация логов сейчас\n" +
		"/reboot — перезагрузка роутера"
}

// cmdService — /service <имя> start|stop|restart.
func cmdService(args []string) tgReply {
	if len(args) != 2 {
		return tgReply{text: "Формат: /service <имя> start|stop|restart\nПример: /service cron restart"}
	}
	name, action := args[0], strings.ToLower(args[1])
	nonce := newNonce()
	putPending(nonce, pendingAction{
		desc:    action + " службы " + name,
		expires: time.Now().Add(confirmTTL),
		run: func() string {
			if err := services.ServiceAction(name, action); err != nil {
				return "❌ " + err.Error()
			}
			return fmt.Sprintf("✅ Служба %s: %s — выполнено", name, action)
		},
	})
	return tgReply{
		text:    fmt.Sprintf("⚠️ Подтвердите: %s службы «%s»", action, name),
		buttons: [][2]string{{"✅ Да, выполнить", "ok:" + nonce}, {"🚫 Отмена", "no:" + nonce}},
	}
}

// cmdPkg — /pkg update (долгая операция: результат отдельным сообщением).
func cmdPkg(args []string) tgReply {
	if len(args) == 0 || args[0] != "update" {
		return tgReply{text: "Формат: /pkg update"}
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logErr("pkg update panic: %v", r)
			}
		}()
		res := execPkgUpdate()
		SendMessage(LoadConfig(), res)
	}()
	return tgReply{text: "⏳ Запускаю opkg update (до 2 минут). Результат пришлю отдельным сообщением."}
}

// execPkgUpdate — opkg update с таймаутом; возвращает хвост вывода.
func execPkgUpdate() string {
	opkg, err := exec.LookPath("opkg")
	if err != nil {
		opkg = "/opt/bin/opkg"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, opkg, "update").CombinedOutput()
	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	if len(lines) > 6 {
		lines = lines[len(lines)-6:]
	}
	if err != nil {
		return "❌ Ошибка обновления:\n" + strings.Join(lines, "\n")
	}
	return "✅ opkg update готов:\n" + strings.Join(lines, "\n")
}

// execRotate — ротация логов сейчас.
func execRotate() string {
	script := "/opt/web_entware/logger/scripts/rotate.sh"
	if _, err := os.Stat(script); err != nil {
		return "❌ rotate.sh не найден"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, script).CombinedOutput()
	if err != nil {
		return "❌ Ошибка ротации: " + err.Error()
	}
	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	if len(lines) > 5 {
		lines = lines[len(lines)-5:]
	}
	return "🔄 Ротация логов выполнена:\n" + strings.Join(lines, "\n")
}

// cmdReboot — /reboot с подтверждением через inline-кнопку.
func cmdReboot([]string) tgReply {
	nonce := newNonce()
	putPending(nonce, pendingAction{
		desc:    "Перезагрузка роутера",
		expires: time.Now().Add(confirmTTL),
		run: func() string {
			go func() {
				time.Sleep(800 * time.Millisecond) // дать сообщению уйти
				_ = exec.Command("sync").Run()
				reb, err := exec.LookPath("reboot")
				if err != nil {
					reb = "/sbin/reboot"
				}
				_ = exec.Command(reb).Run()
			}()
			return "♻️ Перезагрузка через пару секунд..."
		},
	})
	return tgReply{
		text:    "⚠️ Точно перезагрузить роутер?",
		buttons: [][2]string{{"♻️ Да, перезагрузить", "ok:" + nonce}, {"🚫 Отмена", "no:" + nonce}},
	}
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

// cmdLogCmd — /log [N]: N строк хвоста (по умолчанию 15, макс 100).
func cmdLogCmd(args []string) tgReply {
	n := 15
	if len(args) > 0 {
		if v, err := strconv.Atoi(args[0]); err == nil && v > 0 && v <= 100 {
			n = v
		}
	}
	return tgReply{text: tailLog(n)}
}

// tailLog — хвост суточного лога EM; если за сегодня записей ещё нет —
// самый свежий существующий файл.
func tailLog(n int) string {
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

// handleCallback — нажатие inline-кнопки подтверждения (формат ok:<nonce> / no:<nonce>).
func handleCallback(cfg Config, cb *struct {
	ID      string `json:"id"`
	Data    string `json:"data"`
	Message struct {
		Chat struct {
			ID int64 `json:"id"`
		} `json:"chat"`
	} `json:"message"`
}) {
	if strconv.FormatInt(cb.Message.Chat.ID, 10) != cfg.ChatID {
		AnswerCallbackQuery(cfg, cb.ID, "")
		return
	}
	parts := strings.SplitN(cb.Data, ":", 2)
	if len(parts) != 2 {
		AnswerCallbackQuery(cfg, cb.ID, "")
		return
	}
	verb, nonce := parts[0], parts[1]
	act, ok := takePending(nonce)
	if !ok {
		AnswerCallbackQuery(cfg, cb.ID, "Подтверждение истекло")
		return
	}
	switch verb {
	case "ok":
		AnswerCallbackQuery(cfg, cb.ID, "Выполняю…")
		SendMessage(cfg, act.run())
	case "no":
		AnswerCallbackQuery(cfg, cb.ID, "Отменено")
		SendMessage(cfg, "🚫 Отменено: "+act.desc)
	default:
		AnswerCallbackQuery(cfg, cb.ID, "")
	}
}

// buildInlineKB — JSON reply_markup для sendMessage.
func buildInlineKB(buttons [][2]string) string {
	type btn struct {
		Text         string `json:"text"`
		CallbackData string `json:"callback_data"`
	}
	rows := [][]btn{{}}
	for _, b := range buttons {
		rows[0] = append(rows[0], btn{Text: b[0], CallbackData: b[1]})
	}
	m := map[string]interface{}{"inline_keyboard": rows}
	j, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return string(j)
}
