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

func TestIsPanelPort(t *testing.T) {
	for _, p := range []int{8086, 8087, 8089, 8443, 9089, 9099} {
		if !isPanelPort(p) {
			t.Errorf("isPanelPort(%d) = false, хочу true (внутренний порт панели)", p)
		}
	}
	for _, p := range []int{22, 9097, 8080, 8384, 19999} {
		if isPanelPort(p) {
			t.Errorf("isPanelPort(%d) = true, хочу false (порт сервиса)", p)
		}
	}
}

// TestLoopbackListeningPortsExcludesPanel — внутренние порты панели не должны
// попадать в список-подсказку (клик по ним ставил base на панель → 302/404).
func TestLoopbackListeningPortsExcludesPanel(t *testing.T) {
	ports := LoopbackListeningPorts()
	for _, p := range ports {
		if isPanelPort(p) {
			t.Errorf("внутренний порт %d попал в подсказку", p)
		}
	}
}
