// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
package bridge

import "testing"

func TestPortLabel(t *testing.T) {
	cases := map[int]string{
		22:    "SSH",
		51413: "BitTorrent (Transmission)",
		7681:  "ttyd (терминал)",
		8087:  "Панель Entware Manager",
		9097:  "Веб-панель (Koffe)",
		19999: "Netdata",
	}
	for port, want := range cases {
		if got := PortLabel(port); got != want {
			t.Errorf("PortLabel(%d)=%q, хотим %q", port, got, want)
		}
	}
	if got := PortLabel(54322); got != "" {
		t.Errorf("неизвестный порт должен давать '', got %q", got)
	}
}

func TestPortLabelsDictIsCopy(t *testing.T) {
	d := PortLabelsDict()
	d[22] = "взлом"
	if PortLabel(22) != "SSH" {
		t.Error("мутация копии изменила исходник базы")
	}
	if _, ok := PortLabelsDict()[19999]; !ok {
		t.Error("база должна содержать Netdata 19999")
	}
}
