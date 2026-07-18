package network

import (
	"os/exec"
	"strings"
)

type ARPEntry struct {
	IP        string `json:"ip"`
	MAC       string `json:"mac"`
	Interface string `json:"interface"`
	State     string `json:"state"`
	Name      string `json:"name"`
}

func HandleARP() {
	if !IsGET() { NotAllowed(); return }

	data, err := exec.Command("ip", "neigh", "show").Output()
	if err != nil {
		WriteError(err.Error())
		return
	}

	var list []ARPEntry
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" { continue }

		ip := strings.Fields(line)[0]
		// Фильтруем IPv6 (shell: grep -v "^fe80" | grep -v "^::")
		if strings.HasPrefix(ip, "fe80") || strings.HasPrefix(ip, "::") {
			continue
		}

		iface := "--"
		mac := "--"
		state := "UNKNOWN"

		fields := strings.Fields(line)
		for i, f := range fields {
			if f == "dev" && i+1 < len(fields) {
				iface = fields[i+1]
			}
			if f == "lladdr" && i+1 < len(fields) {
				mac = fields[i+1]
			}
		}

		// Последнее поле — состояние (REACHABLE, STALE, DELAY, FAILED...)
		if last := fields[len(fields)-1]; last != "dev" && last != "lladdr" {
			state = last
		}

		list = append(list, ARPEntry{
			IP:        ip,
			MAC:       mac,
			Interface: iface,
			State:     state,
			Name:      "",
		})
	}

	WriteJSON(map[string][]ARPEntry{"entries": list})
}
