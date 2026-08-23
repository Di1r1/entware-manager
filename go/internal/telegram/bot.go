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

	"entware-manager/internal/backup"
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

// Ежедневный дайджест: час отправки и state-файл (чтобы после рестарта
// не продублировать сводку в тот же день).
const (
	digestHour      = 9
	digestStateFile = "/tmp/entware/telegram/digest.last"
)

func readDigestDate() string {
	b, err := os.ReadFile(digestStateFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// scanServicesUp — какие службы сейчас работают (по pid-файлам + /proc).
func scanServicesUp() map[string]bool {
	up := map[string]bool{}
	matches, _ := filepath.Glob("/opt/var/run/*.pid")
	for _, m := range matches {
		name := strings.TrimSuffix(filepath.Base(m), ".pid")
		pid := strings.TrimSpace(readLine(m))
		if pid == "" {
			continue
		}
		if _, err := os.Stat("/proc/" + pid); err == nil {
			up[name] = true
		}
	}
	return up
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
	svcPrev := map[string]bool{}
	svcBase := false
	lastDigest := readDigestDate()
	netPrev := scanNetDevices()
	netBase := false
	bgIt := 0
	diskAlerted := false
	wanPrev, wanSet := "", false

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

		// Ежедневный дайджест (в 09:00; после рестарта не дублируется —
		// дата хранится в state-файле).
		if today := time.Now().Format("2006-01-02"); time.Now().Hour() >= digestHour && lastDigest != today {
			lastDigest = today
			os.MkdirAll(filepath.Dir(digestStateFile), 0755)
			os.WriteFile(digestStateFile, []byte(today), 0644)
			SendMessage(cfg, buildDigest())
		}

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

		// Слежение за службами: падение → алерт с кнопкой перезапуска,
		// восстановление → уведомление. Первый проход — база (без алертов).
		cur := scanServicesUp()
		if svcBase {
			for name, wasUp := range svcPrev {
				if !wasUp {
					continue
				}
				if cur[name] {
					continue
				}
				n := newNonce()
				putPending(n, pendingAction{
					desc:    "перезапуск службы " + name,
					expires: time.Now().Add(10 * time.Minute),
					run: func() string {
						if err := services.ServiceAction(name, "start"); err != nil {
							return "❌ " + err.Error()
						}
						return fmt.Sprintf("✅ Служба %s запущена", name)
					},
				})
				SendMessageMarkup(cfg,
					fmt.Sprintf("🔴 Служба «%s» остановилась!", name),
					buildInlineKB([][2]string{
						{"🔄 Перезапустить", "ok:" + n},
						{"Игнорировать", "no:" + n},
					}))
			}
			for name := range cur {
				if !svcPrev[name] && svcPrev != nil {
					SendMessage(cfg, fmt.Sprintf("🟢 Служба «%s» запущена", name))
				}
			}
		}
		svcPrev = cur
		svcBase = true

		// Раз в ~60 сек — фоновые проверки безопасности.
		bgIt++
		if bgIt%12 == 0 {
			// Новое устройство в сети
			devs := scanNetDevices()
			if netBase {
				for k, h := range devs {
					if _, known := netPrev[k]; !known {
						SendMessage(cfg, fmt.Sprintf("🔴 Новое устройство в сети!\n📱 %s\n🌐 IP: %s\n🔖 MAC: %s",
							cleanHostDisplay(h), h.IP, strings.ToUpper(h.MAC)))
					}
				}
			}
			netPrev = devs
			netBase = true

			// Мало места на /opt (Entware)
			pct := dfUsedPct(runDF(), "/opt")
			if pct >= 90 && !diskAlerted {
				diskAlerted = true
				SendMessage(cfg, fmt.Sprintf("🟡 Мало места на /opt: занято %d%%", pct))
			} else if pct >= 0 && pct < 85 {
				diskAlerted = false
			}

			// Смена внешнего IP
			ip := ifaceIPv4(defaultIface(readLine("/proc/net/route")))
			if wanSet && ip != "" && ip != wanPrev {
				SendMessage(cfg, fmt.Sprintf("🌐 Внешний IP изменился: %s → %s", wanPrev, ip))
			}
			if ip != "" {
				wanPrev = ip
			} else if !wanSet {
				wanSet = true
				wanPrev = ip
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
		"/find":     cmdFind,
		"/digest":   func([]string) tgReply { return tgReply{text: buildDigest()} },
		// Группа A — расширенная информация
		"/top":     cmdTop,
		"/ports":   func([]string) tgReply { return tgReply{text: cmdPorts()} },
		"/devices": func([]string) tgReply { return tgReply{text: cmdDevices()} },
		"/wifi":    func([]string) tgReply { return tgReply{text: cmdWifi()} },
		"/updates": func([]string) tgReply { return tgReply{text: cmdUpdates()} },
		"/cron":    func([]string) tgReply { return tgReply{text: cmdCron()} },
		// Уровень 2 — управление (подтверждение inline-кнопкой для опасных).
		"/service": cmdService,
		"/pkg":     cmdPkg,
		"/rotate":  func([]string) tgReply { return tgReply{text: execRotate()} },
		"/backup":  cmdBackup,
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
		"/find <текст> — поиск по логу\n" +
		"/digest — сводка за сутки сейчас\n" +
		"/top [N] — процессы по нагрузке CPU\n" +
		"/ports — слушающие порты\n" +
		"/devices — устройства в сети\n" +
		"/wifi — клиенты Wi-Fi\n" +
		"/updates — доступные обновления пакетов\n" +
		"/cron — содержимое crontab\n" +
		"⚙️ Управление (с подтверждением):\n" +
		"/service <имя> start|stop|restart — управление службой\n" +
		"/pkg update — обновить списки пакетов\n" +
		"/rotate — ротация логов сейчас\n" +
		"/reboot — перезагрузка роутера"
}

// buildDigest — ежедневная сводка (и команда /digest).
func buildDigest() string {
	svcUp, svcTotal := 0, 0
	matches, _ := filepath.Glob("/opt/var/run/*.pid")
	for _, m := range matches {
		svcTotal++
		pid := strings.TrimSpace(readLine(m))
		if pid != "" {
			if _, err := os.Stat("/proc/" + pid); err == nil {
				svcUp++
			}
		}
	}
	errCount := 0
	if path := resolveLogPath(); path != "" {
		if data, err := os.ReadFile(path); err == nil {
			for _, ln := range strings.Split(string(data), "\n") {
				if strings.Contains(ln, "[ERROR]") {
					errCount++
				}
			}
		}
	}
	return "🌅 Сводка за сутки:\n" +
		strings.TrimRight(cmdStatus(), "\n") +
		fmt.Sprintf("\n🟢 Службы: %d/%d работает", svcUp, svcTotal) +
		fmt.Sprintf("\n❗ Ошибок в логе: %d", errCount)
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
	var results []string
	for _, dev := range []string{"/dev/sda", "/dev/sdb", "/dev/sdc", "/dev/sdd"} {
		if _, err := os.Stat(dev); err != nil {
			continue
		}
		cctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		b, err := exec.CommandContext(cctx, sm, "-H", dev).Output()
		cancel()
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
		results = append(results, fmt.Sprintf("%s — %s", dev, res))
	}
	if len(results) == 0 {
		return "SMART: диски не найдены"
	}
	return "💾 SMART:\n" + strings.Join(results, "\n")
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

// resolveLogPath — суточный лог; если за сегодня записей ещё нет —
// самый свежий существующий файл.
func resolveLogPath() string {
	dir := "/tmp/entware/logs"
	path := filepath.Join(dir, time.Now().Format("2006-01-02")+".log")
	if _, err := os.Stat(path); err == nil {
		return path
	}
	best, bestMod := "", time.Time{}
	matches, _ := filepath.Glob(filepath.Join(dir, "20*.log"))
	for _, m := range matches {
		if fi, err := os.Stat(m); err == nil && fi.ModTime().After(bestMod) {
			best, bestMod = m, fi.ModTime()
		}
	}
	return best
}

// tailLog — хвост лога EM (N последних строк).
func tailLog(n int) string {
	path := resolveLogPath()
	if path == "" {
		return "Логи отсутствуют"
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

// cmdFind — /find <текст>: поиск по логу без учёта регистра (до 10 строк).
func cmdFind(args []string) tgReply {
	if len(args) == 0 {
		return tgReply{text: "Формат: /find <текст>"}
	}
	query := strings.ToLower(strings.Join(args, " "))
	path := resolveLogPath()
	if path == "" {
		return tgReply{text: "Логи отсутствуют"}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return tgReply{text: "Логи отсутствуют"}
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	var hits []string
	total := 0
	for _, ln := range lines {
		if strings.Contains(strings.ToLower(ln), query) {
			total++
			if len(hits) < 10 {
				hits = append(hits, ln)
			}
		}
	}
	if total == 0 {
		return tgReply{text: fmt.Sprintf("🔍 «%s»: не найдено в %s", query, filepath.Base(path))}
	}
	head := fmt.Sprintf("🔍 «%s»: найдено %d (показаны первые %d):\n", query, total, len(hits))
	if total <= len(hits) {
		head = fmt.Sprintf("🔍 «%s»: найдено %d:\n", query, total)
	}
	return tgReply{text: head + strings.Join(hits, "\n")}
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

// --- Группа A: расширенная информация ---

// procSample — снимок процесса для замера CPU.
type procSample struct {
	name string
	cpu  int64 // utime+stime в тиках
	rss  int64 // КБ
}

// procSnapshot — CPU-тики и RSS всех процессов + суммарные тики системы.
func procSnapshot() (map[string]procSample, int64) {
	res := map[string]procSample{}
	var total int64
	if f := strings.Fields(readLine("/proc/stat")); len(f) > 4 {
		for _, v := range f[1:] {
			x, _ := strconv.ParseInt(v, 10, 64)
			total += x
		}
	}
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return res, total
	}
	for _, e := range entries {
		pid, perr := strconv.Atoi(e.Name())
		if perr != nil || pid <= 0 {
			continue
		}
		raw := readLine("/proc/" + e.Name() + "/stat")
		open, close := strings.Index(raw, "("), strings.LastIndex(raw, ")")
		if open < 0 || close < open || close+2 > len(raw) {
			continue
		}
		f := strings.Fields(raw[close+2:])
		if len(f) < 22 {
			continue
		}
		ut, _ := strconv.ParseInt(f[11], 10, 64)
		st, _ := strconv.ParseInt(f[12], 10, 64)
		rssPages, _ := strconv.ParseInt(f[21], 10, 64)
		res[e.Name()] = procSample{name: raw[open+1 : close], cpu: ut + st, rss: rssPages * 4}
	}
	return res, total
}

// cmdTop — /top [N]: процессы по приросту CPU (двухточечный замер, 1с).
func cmdTop(args []string) tgReply {
	n := 5
	if len(args) > 0 {
		if v, err := strconv.Atoi(args[0]); err == nil && v > 0 && v <= 15 {
			n = v
		}
	}
	s1, t1 := procSnapshot()
	time.Sleep(time.Second)
	s2, t2 := procSnapshot()
	dt := t2 - t1
	if dt <= 0 {
		return tgReply{text: "Нет данных для замера"}
	}
	type row struct {
		name string
		pct  float64
		rss  int64
	}
	var rows []row
	for pid, s := range s2 {
		prev, ok := s1[pid]
		if !ok {
			continue
		}
		d := s.cpu - prev.cpu
		if d < 0 {
			d = 0
		}
		pct := 100 * float64(d) / float64(dt)
		rows = append(rows, row{s.name, pct, s.rss})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].pct != rows[j].pct {
			return rows[i].pct > rows[j].pct
		}
		return rows[i].rss > rows[j].rss
	})
	if len(rows) > n {
		rows = rows[:n]
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("🔝 Топ-%d по CPU (за 1 сек):\n", len(rows)))
	for _, r := range rows {
		b.WriteString(fmt.Sprintf("  %s — %.1f%% CPU, %d МБ RAM\n", r.name, r.pct, r.rss/1024))
	}
	return tgReply{text: b.String()}
}

// knownPortNames — подписи известных портов (для /ports).
var knownPortNames = map[int]string{
	22: "SSH", 53: "DNS (dnsmasq)", 80: "HTTP", 443: "HTTPS",
	8086: "lighttpd/koffe", 8087: "Панель EM", 8089: "htop (ttyd)",
	9089: "терминал (ttyd)", 9097: "koffe-api", 10871: "прокси (xray)",
}

// decodeHexSockaddr — hex-адрес из /proc/net/tcp → человекочитаемый хост.
func decodeHexSockaddr(hexAddr string) string {
	if len(hexAddr) == 8 { // IPv4 little-endian по байтам
		b := make([]byte, 4)
		for i := 0; i < 4; i++ {
			v, _ := strconv.ParseUint(hexAddr[i*2:i*2+2], 16, 8)
			b[i] = byte(v)
		}
		return fmt.Sprintf("%d.%d.%d.%d", b[3], b[2], b[1], b[0])
	}
	return hexAddr // IPv6 — как есть
}

// cmdPorts — слушающие TCP-порты (/proc/net/tcp+tcp6).
func cmdPorts() string {
	type listen struct {
		addr string
		port int
	}
	seen := map[int]map[string]bool{}
	for _, file := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		for i, ln := range strings.Split(string(data), "\n") {
			if i == 0 {
				continue // заголовок
			}
			f := strings.Fields(ln)
			if len(f) < 4 || f[3] != "0A" { // 0A = LISTEN
				continue
			}
			hexAddr, portHex := "", ""
			if j := strings.IndexByte(f[1], ':'); j > 0 {
				hexAddr, portHex = f[1][:j], f[1][j+1:]
			}
			port, _ := strconv.ParseInt(portHex, 16, 32)
			host := decodeHexSockaddr(hexAddr)
			p := int(port)
			if seen[p] == nil {
				seen[p] = map[string]bool{}
			}
			seen[p][host] = true
		}
	}
	if len(seen) == 0 {
		return "Слушающие порты не найдены"
	}
	ports := make([]int, 0, len(seen))
	for p := range seen {
		ports = append(ports, p)
	}
	sort.Ints(ports)
	var b strings.Builder
	b.WriteString("🔌 Слушающие порты:\n")
	for _, p := range ports {
		note := knownPortNames[p]
		addrs := make([]string, 0, len(seen[p]))
		for a := range seen[p] {
			addrs = append(addrs, a)
		}
		sort.Strings(addrs)
		line := fmt.Sprintf("  :%d", p)
		if note != "" {
			line += " (" + note + ")"
		}
		line += " ← " + strings.Join(addrs, ", ")
		b.WriteString(line + "\n")
	}
	return b.String()
}

// rciHost — устройство домашней сети из RCI Keenetic.
type rciHost struct {
	MAC       string `json:"mac"`
	IP        string `json:"ip"`
	Hostname  string `json:"hostname"`
	Name      string `json:"name"`
	Interface struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"interface"`
	Link   string `json:"link"`
	Active bool   `json:"active"`
	SSID   string `json:"ssid"`
	AP     string `json:"ap"`
	Mode   string `json:"mode"`
	RSSI   int    `json:"rssi"`
	TXRate int    `json:"txrate"`
}

// cleanHostDisplay — имя устройства без служебного хвоста Keenetic
// («Имя - Сегмент - дата регистрации»).
func cleanHostDisplay(h rciHost) string {
	name := hostDisplayName(h)
	if i := strings.Index(name, " - "); i > 0 {
		tail := strings.ToLower(name[i:])
		if strings.Contains(tail, "network") && strings.Contains(tail, "- 20") {
			name = name[:i]
		}
	}
	return name
}

// isWiFiClient — клиент подключён по Wi-Fi (есть ssid/ap в записи RCI).
func isWiFiClient(h rciHost) bool {
	return h.SSID != "" || h.AP != ""
}

// fetchRCIHosts — список устройств через RCI (как arp.go).
func fetchRCIHosts() []rciHost {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://127.0.0.1:79/rci/show/ip/hotspot/host")
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	var hosts []rciHost
	if json.NewDecoder(resp.Body).Decode(&hosts) != nil {
		return nil
	}
	return hosts
}

func hostDisplayName(h rciHost) string {
	if h.Name != "" {
		return h.Name
	}
	if h.Hostname != "" {
		return h.Hostname
	}
	return h.MAC
}

// cmdDevices — активные устройства домашней сети (RCI hotspot/host).
// Офлайн-записи (IP пуст или 0.0.0.0) скрываются.
func cmdDevices() string {
	hosts := fetchRCIHosts()
	var online []rciHost
	for _, h := range hosts {
		if h.IP == "" || h.IP == "0.0.0.0" {
			continue
		}
		online = append(online, h)
	}
	if len(online) == 0 {
		return "Устройства: нет данных от RCI"
	}
	sort.Slice(online, func(i, j int) bool {
		return ipToNum(online[i].IP) < ipToNum(online[j].IP)
	})
	var b strings.Builder
	fmt.Fprintf(&b, "📱 Устройства в сети (%d):\n", len(online))
	for _, h := range online {
		mark := ""
		if isWiFiClient(h) {
			mark = " 📶"
		}
		fmt.Fprintf(&b, "  %s%s — %s\n", cleanHostDisplay(h), mark, h.IP)
	}
	return b.String()
}

// ipToNum — числовое значение IPv4 для человекочитаемой сортировки.
func ipToNum(ip string) int64 {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return -1
	}
	var n int64
	for i := 0; i < 4; i++ {
		v, _ := strconv.ParseInt(parts[i], 10, 64)
		n = n*256 + v
	}
	return n
}

// cmdWifi — клиенты Wi-Fi: SSID, уровень сигнала (RSSI), стандарт и скорость.
func cmdWifi() string {
	hosts := fetchRCIHosts()
	var wifi []rciHost
	for _, h := range hosts {
		if h.IP == "" || h.IP == "0.0.0.0" {
			continue
		}
		if isWiFiClient(h) {
			wifi = append(wifi, h)
		}
	}
	if len(wifi) == 0 {
		return "📶 Клиентов Wi-Fi сейчас нет"
	}
	sort.Slice(wifi, func(i, j int) bool { return wifi[i].RSSI > wifi[j].RSSI })
	var b strings.Builder
	fmt.Fprintf(&b, "📶 Клиенты Wi-Fi (%d), сильнейший сигнал первым:\n", len(wifi))
	for _, h := range wifi {
		line := fmt.Sprintf("  %s — %s", cleanHostDisplay(h), h.IP)
		if h.RSSI != 0 {
			line += fmt.Sprintf(", сигнал %d dBm", h.RSSI)
		}
		if h.Mode != "" {
			line += ", " + h.Mode
		}
		if h.TXRate > 0 {
			line += fmt.Sprintf(", %d Мбит/с", h.TXRate)
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

// cmdUpdates — доступные обновления пакетов (opkg list-upgradable).
func cmdUpdates() string {
	opkg, err := exec.LookPath("opkg")
	if err != nil {
		opkg = "/opt/bin/opkg"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, opkg, "list-upgradable").CombinedOutput()
	if err != nil {
		return "❌ Не удалось получить список обновлений"
	}
	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return "✅ Все пакеты актуальны"
	}
	head := fmt.Sprintf("📦 Доступно обновлений: %d\n", len(lines))
	if len(lines) > 10 {
		lines = append(lines[:10], "… и ещё "+strconv.Itoa(len(lines)-10))
	}
	return head + strings.Join(lines, "\n")
}

// cmdCron — содержимое crontab (crontab -l с фолбэком на файл).
func cmdCron() string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "crontab", "-l").Output()
	if err == nil && len(strings.TrimSpace(string(out))) > 0 {
		return "⏰ Crontab:\n" + strings.TrimRight(string(out), "\n")
	}
	data, rerr := os.ReadFile("/opt/etc/crontab")
	if rerr == nil && len(strings.TrimSpace(string(data))) > 0 {
		return "⏰ Crontab (/opt/etc/crontab):\n" + strings.TrimRight(string(data), "\n")
	}
	return "⏰ Crontab пуст или отсутствует"
}

// scanNetDevices — активные устройства по MAC-ключу (для диффа «новое устройство»).
func scanNetDevices() map[string]rciHost {
	m := map[string]rciHost{}
	for _, h := range fetchRCIHosts() {
		if h.IP == "" || h.IP == "0.0.0.0" {
			continue
		}
		key := strings.ToLower(h.MAC)
		if key == "" {
			key = "ip:" + h.IP
		}
		m[key] = h
	}
	return m
}

// dfUsedPct — процент занятого места для точки монтирования (-1: не найдено).
func dfUsedPct(dfOut, mount string) int {
	for _, ln := range strings.Split(dfOut, "\n") {
		f := strings.Fields(ln)
		if len(f) >= 5 && f[len(f)-1] == mount {
			v, err := strconv.Atoi(strings.TrimSuffix(f[4], "%"))
			if err != nil {
				return -1
			}
			return v
		}
	}
	return -1
}

// cmdBackup — /backup: архив конфигурации файлом в чат.
func cmdBackup([]string) tgReply {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logErr("backup panic: %v", r)
			}
		}()
		cfgNow := LoadConfig()
		data, err := backup.BuildArchive()
		if err != nil || len(data) == 0 {
			msg := "❌ Не удалось создать бэкап"
			if err != nil {
				msg += ": " + err.Error()
			}
			SendMessage(cfgNow, msg)
			return
		}
		fn := "entware-backup-" + time.Now().Format("2006-01-02_1504") + ".tar.gz"
		if SendDocumentBytes(cfgNow, fn, data, "📦 Бэкап Entware Manager ("+fmtKB(len(data))+")") {
			SendMessage(cfgNow, fmt.Sprintf("✅ Бэкап отправлен (%s)", fmtKB(len(data))))
		} else {
			SendMessage(cfgNow, "❌ Не удалось отправить файл (лимит Telegram — 50 МБ)")
		}
	}()
	return tgReply{text: "⏳ Создаю бэкап конфигурации... Архив пришлю отдельным сообщением."}
}

// fmtKB — человекочитаемый размер.
func fmtKB(n int) string {
	switch {
	case n >= 1024*1024:
		return fmt.Sprintf("%.1f МБ", float64(n)/(1024*1024))
	default:
		return fmt.Sprintf("%d КБ", n/1024)
	}
}
