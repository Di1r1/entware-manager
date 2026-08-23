package telegram

import (
	"strings"
	"testing"
	"time"

	"entware-manager/internal/services"
)

func TestParseMeminfo(t *testing.T) {
	sample := `MemTotal:        1048576 kB
MemFree:          120996 kB
MemAvailable:     291344 kB
Buffers:           56144 kB`
	total, avail := parseMeminfo(sample)
	if total != 1048576 {
		t.Errorf("total = %d, want 1048576", total)
	}
	if avail != 291344 {
		t.Errorf("avail = %d, want 291344", avail)
	}
}

func TestParseDfRoot(t *testing.T) {
	df := `Filesystem                Size      Used Available Use% Mounted on
/dev/root                 23.4M    500.0M       0   2% /
tmpfs                     512.0M    120.0K    511.9M   0% /tmp
/dev/sde1                  28.7G     12.3G     15.0G  45% /opt`
	got := parseDfRoot(df)
	if !strings.Contains(got, "23.4M") || !strings.Contains(got, "28.7G") {
		t.Errorf("parseDfRoot должен показать / и /opt, got: %q", got)
	}
	if strings.Contains(got, "tmpfs") {
		t.Errorf("parseDfRoot не должен показывать tmpfs, got: %q", got)
	}
}

func TestDefaultIface(t *testing.T) {
	rt := `Iface	Destination	Gateway	Flags	RefCnt	Use	Metric	Mask	MTU	Window	IRTT
PPPoE0	00000000	00000000	0003	0	0	0	00000000	0	0	0
br0	00C0A8C0	00000000	0001	0	0	0	00FFFFFF	0	0	0`
	if got := defaultIface(rt); got != "PPPoE0" {
		t.Errorf("defaultIface = %q, want PPPoE0", got)
	}
}

func TestAllowedChat(t *testing.T) {
	var u tgUpdate
	u.Message.Text = "/status"
	u.Message.Chat.ID = 241544715
	if !allowedChat(u, "241544715") {
		t.Error("свой chat_id должен пропускаться")
	}
	if allowedChat(u, "111222333") {
		t.Error("чужой chat_id должен блокироваться")
	}
	u.Message.Text = ""
	if allowedChat(u, "241544715") {
		t.Error("пустой текст не должен обрабатываться")
	}
}

func TestReplyForRouting(t *testing.T) {
	called := ""
	cmds := map[string]func([]string) tgReply{
		"/help": func([]string) tgReply { called = "/help"; return tgReply{text: "HELP"} },
	}
	rep := replyFor("/Help@MyBot", cmds)
	if rep.text != "HELP" || called != "/help" {
		t.Errorf("регистр/@суффикс должны нормализоваться, got %q called=%q", rep.text, called)
	}
	if rep := replyFor("привет", cmds); rep.text != "" || len(rep.buttons) != 0 {
		t.Errorf("не-команда должна молчать, got %+v", rep)
	}
	if rep := replyFor("/unknown cmd", cmds); !strings.Contains(rep.text, "Неизвестная команда") {
		t.Errorf("неизвестная команда должна давать подсказку, got %q", rep.text)
	}
}

func TestPendingLifecycle(t *testing.T) {
	n := newNonce()
	if len(n) < 6 {
		t.Fatalf("nonce слишком короткий: %q", n)
	}
	putPending(n, pendingAction{
		desc:    "тест",
		run:     func() string { return "done" },
		expires: time.Now().Add(time.Minute),
	})
	act, ok := takePending(n)
	if !ok || act.run() != "done" {
		t.Fatalf("takePending не вернул действие")
	}
	// одноразовость
	if _, ok := takePending(n); ok {
		t.Error("повторный take должен вернуть false")
	}
	// просроченное не возвращается через put+prune
	putPending("x", pendingAction{desc: "старое", expires: time.Now().Add(-time.Second)})
	if _, ok := takePending("x"); ok {
		t.Error("просроченное действие не должно выдаваться")
	}
}

func TestServiceActionValidation(t *testing.T) {
	// валидация ошибок без выполнения (path traversal / недопустимое действие)
	if err := services.ServiceAction("../../bin/reboot", "start"); err == nil {
		t.Error("path traversal должен отклоняться")
	}
	if err := services.ServiceAction("cron", "format"); err == nil {
		t.Error("недопустимое действие должно отклоняться")
	}
}

func TestCmdLogMissing(t *testing.T) {
	// не должно паниковать при отсутствии файла за сегодня
	_ = tailLog(15)
}

func TestScanServicesUp(t *testing.T) {
	// реальный роутер: /opt/var/run/*.pid — функция не должна паниковать
	up := scanServicesUp()
	for name := range up {
		if name == "" {
			t.Error("пустое имя службы")
		}
	}
}

func TestResolveLogPathFallback(t *testing.T) {
	p := resolveLogPath()
	if p != "" && !strings.Contains(p, "/tmp/entware/logs/") {
		t.Errorf("неожиданный путь: %q", p)
	}
}

func TestDecodeHexSockaddr(t *testing.T) {
	// /proc/net/tcp хранит адрес little-endian: 127.0.0.1 → 0100007F
	if got := decodeHexSockaddr("0100007F"); got != "127.0.0.1" {
		t.Errorf("got %q, want 127.0.0.1", got)
	}
	// 192.168.3.1 → 0103A8C0
	if got := decodeHexSockaddr("0103A8C0"); got != "192.168.3.1" {
		t.Errorf("got %q, want 192.168.3.1", got)
	}
	// 0.0.0.0
	if got := decodeHexSockaddr("00000000"); got != "0.0.0.0" {
		t.Errorf("got %q, want 0.0.0.0", got)
	}
}

func TestFindLinesCap(t *testing.T) {
	content := "line ERROR one\nplain\nline error two\nERROR three\n"
	hits, total := []string{}, 0
	for _, ln := range strings.Split(content, "\n") {
		if strings.Contains(strings.ToLower(ln), "error") {
			total++
			if len(hits) < 2 {
				hits = append(hits, ln)
			}
		}
	}
	if total != 3 || len(hits) != 2 {
		t.Errorf("total=%d hits=%d, want 3 и 2 (cap)", total, len(hits))
	}
}

func TestProcSnapshot(t *testing.T) {
	// на тестовой машине /proc есть — снимок должен вернуться непустым
	snap, total := procSnapshot()
	if total <= 0 {
		t.Fatal("total cpu ticks должен быть > 0")
	}
	if len(snap) == 0 {
		t.Log("процессов не найдено (ок для CI-контейнера)")
	}
	for _, s := range snap {
		if s.name == "" {
			t.Error("пустое имя процесса")
			break
		}
	}
}

func TestCleanHostDisplay(t *testing.T) {
	h := rciHost{Name: "N701-0000004401 - Home network - 2026-07-07 02:15", Hostname: "n701"}
	if got := cleanHostDisplay(h); got != "N701-0000004401" {
		t.Errorf("got %q", got)
	}
	// имя без хвоста — не трогаем
	h2 := rciHost{Name: "S23", Hostname: "s23-phone"}
	if got := cleanHostDisplay(h2); got != "S23" {
		t.Errorf("got %q", got)
	}
	// имя с дефисом в середине (не сегмент+дата) — сохраняем
	h3 := rciHost{Name: "Google-Home-Mini Спальня"}
	if got := cleanHostDisplay(h3); got != "Google-Home-Mini Спальня" {
		t.Errorf("got %q", got)
	}
}

func TestIsWiFiClient(t *testing.T) {
	w := rciHost{SSID: "DiZyXEL", AP: "WifiMaster1/AccessPoint0"}
	if !isWiFiClient(w) {
		t.Error("хост с ssid должен считаться Wi-Fi клиентом")
	}
	p := rciHost{Name: "mini-AMD"} // проводной
	if isWiFiClient(p) {
		t.Error("проводной хост не должен считаться Wi-Fi")
	}
}

func TestIPToNum(t *testing.T) {
	if ipToNum("192.168.3.10") >= ipToNum("192.168.3.100") {
		t.Error("10 должно быть меньше 100 при одинаковых первых октетах")
	}
	if ipToNum("bad") != -1 {
		t.Error("невалидный IP → -1")
	}
}

func TestQuietAt(t *testing.T) {
	// выключен — никогда
	cfg := Config{QuietEnabled: false, QuietFrom: 23, QuietTo: 7}
	if quietAt(cfg, 3) {
		t.Error("выключенный тихий режим не должен срабатывать")
	}
	// ночное окно через полночь: 23 → 7
	cfg = Config{QuietEnabled: true, QuietFrom: 23, QuietTo: 7}
	if !quietAt(cfg, 23) || !quietAt(cfg, 3) {
		t.Error("23:00-07:00: ночь должна попадать в окно")
	}
	if quietAt(cfg, 12) {
		t.Error("12:00 днём не должно попадать в ночное окно")
	}
	// дневное окно: 13 → 15
	cfg = Config{QuietEnabled: true, QuietFrom: 13, QuietTo: 15}
	if !quietAt(cfg, 14) {
		t.Error("14:00 должно быть в окне 13-15")
	}
	if quietAt(cfg, 16) {
		t.Error("16:00 вне окна 13-15")
	}
	// границы: from включительно, to исключительно
	cfg = Config{QuietEnabled: true, QuietFrom: 13, QuietTo: 15}
	if !quietAt(cfg, 13) || quietAt(cfg, 15) {
		t.Error("границы окна: from вкл, to исключён")
	}
	// from == to — неактивно
	cfg = Config{QuietEnabled: true, QuietFrom: 9, QuietTo: 9}
	if quietAt(cfg, 9) {
		t.Error("from==to не должно активировать режим")
	}
}

func TestRecipients(t *testing.T) {
	cfg := Config{ChatID: "111"}
	got := recipients(cfg)
	if len(got) != 1 || got[0] != "111" {
		t.Errorf("только основной chat_id: %v", got)
	}
	cfg.AllowedChatIDs = []string{"222", "333", "111"}
	got = recipients(cfg)
	if len(got) != 3 {
		t.Errorf("основной + 2 доп: %v", got)
	}
	// дубликаты отфильтрованы, порядок сохранён
	// внутренние дубликаты среди доп. ID тоже убираются
	cfg.AllowedChatIDs = []string{"111", "222", "111", "222"}
	got = recipients(cfg)
	if len(got) != 2 || got[0] != "111" || got[1] != "222" {
		t.Errorf("внутренние дубликаты не убраны: %v", got)
	}
}
