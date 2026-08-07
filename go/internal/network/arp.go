package network

import (
	"encoding/json"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

type ARPEntry struct {
	IP        string `json:"ip"`
	MAC       string `json:"mac"`
	Interface string `json:"interface"`
	State     string `json:"state"`
	Name      string `json:"name"`
}

type rciHost struct {
	IP       string `json:"ip"`
	Hostname string `json:"hostname"`
	Name     string `json:"name"`
}

// rciHotspotURL — переопределяется в тестах
var rciHotspotURL = rciBase + "/rci/show/ip/hotspot/host"

// fetchRCIHostnames возвращает map[ip]имя устройства по данным RCI Keenetic
// (/rci/show/ip/hotspot/host). Приоритет: name, иначе hostname.
// При недоступности RCI возвращается пустая map — ARP не должен падать.
func fetchRCIHostnames() map[string]string {
	names := make(map[string]string)

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(rciHotspotURL)
	if err != nil {
		return names
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return names
	}

	var hosts []rciHost
	if json.Unmarshal(data, &hosts) != nil {
		return names
	}

	for _, h := range hosts {
		if h.IP == "" {
			continue
		}
		name := h.Name
		if name == "" {
			name = h.Hostname
		}
		names[h.IP] = name
	}
	return names
}

func HandleARP() {
	if !IsGET() {
		NotAllowed()
		return
	}

	data, err := exec.Command("ip", "neigh", "show").Output()
	if err != nil {
		WriteError(err.Error())
		return
	}

	hostnames := fetchRCIHostnames()

	var list []ARPEntry
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

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
			Name:      hostnames[ip],
		})
	}

	WriteJSON(map[string][]ARPEntry{"entries": list})
}
