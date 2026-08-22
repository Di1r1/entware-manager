package telegram

import (
	"strings"
	"testing"
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
	cmds := map[string]func() string{
		"/help": func() string { called = "/help"; return "HELP" },
	}
	if got := replyFor("/Help@MyBot", cmds); got != "HELP" || called != "/help" {
		t.Errorf("регистр/@суффикс должны нормализоваться, got %q called=%q", got, called)
	}
	if got := replyFor("привет", cmds); got != "" {
		t.Errorf("не-команда должна молчать, got %q", got)
	}
	if got := replyFor("/unknown cmd", cmds); !strings.Contains(got, "Неизвестная команда") {
		t.Errorf("неизвестная команда должна давать подсказку, got %q", got)
	}
}

func TestCmdLogMissing(t *testing.T) {
	// не должно паниковать при отсутствии файла за сегодня
	_ = cmdLog()
}
