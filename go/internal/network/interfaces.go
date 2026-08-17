package network

import (
	"encoding/json"
	"entware-manager/internal/cgiutil"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Iface struct {
	Name  string `json:"name"`
	State string `json:"state"`
	IP    string `json:"ip"`
	MAC   string `json:"mac"`
	Type  string `json:"type"`
	Speed string `json:"speed"`
	SSID  string `json:"ssid"`
}

// linkTypeNames — перевод типов каналов из ip link в читаемые названия
var linkTypeNames = map[string]string{
	"ether":    "Ethernet",
	"loopback": "Loopback",
	"ppp":      "PPP",
	"ipip":     "IPIP-туннель",
	"tunnel6":  "IPv6-туннель",
	"sit":      "SIT-туннель",
	"gre":      "GRE-туннель",
	"gretap":   "GRE-туннель",
	"bridge":   "Мост",
	"vlan":     "VLAN",
	"none":     "",
}

// rciIfaceURL — переопределяется в тестах
const rciBase = "http://127.0.0.1:79"

var rciIfaceURL = rciBase + "/rci/show/interface/"

// fetchRCIWifiSSIDs возвращает map[mac]ssid из RCI Keenetic
// (блоки AccessPoint*. При недоступности RCI — пустая map.
func fetchRCIWifiSSIDs() map[string]string {
	ssids := make(map[string]string)

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(rciIfaceURL)
	if err != nil {
		return ssids
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return ssids
	}

	var parsed map[string]any
	if json.Unmarshal(data, &parsed) != nil {
		return ssids
	}

	for _, v := range parsed {
		obj, ok := v.(map[string]any)
		if !ok {
			continue
		}
		mac, _ := obj["mac"].(string)
		ssid, _ := obj["ssid"].(string)
		if mac == "" || ssid == "" {
			continue
		}
		if _, exists := ssids[mac]; !exists {
			ssids[mac] = ssid
		}
	}
	return ssids
}

// ifaceSpeed возвращает скорость интерфейса из sysfs (Мбит/с) или "".
func ifaceSpeed(name string) string {
	b, err := os.ReadFile("/sys/class/net/" + name + "/speed")
	if err != nil {
		return ""
	}
	sp := strings.TrimSpace(string(b))
	n, err := strconv.Atoi(sp)
	if err != nil || n <= 0 {
		return ""
	}
	return strconv.Itoa(n) + " Мбит/с"
}

// parseLinkType извлекает тип канала и MAC из строки ip -o link show.
func parseLinkType(rest string) (typ, mac string) {
	idx := strings.Index(rest, "link/")
	if idx < 0 {
		return "", ""
	}
	rest = rest[idx+5:]
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return "", ""
	}
	typ = fields[0]
	if len(fields) >= 2 && isMAC(fields[1]) {
		mac = fields[1]
	}
	return
}

// isMAC проверяет, что строка похожа на MAC-адрес (AA:BB:CC:DD:EE:FF)
func isMAC(s string) bool {
	if len(s) != 17 {
		return false
	}
	for i, c := range s {
		if i%3 == 2 {
			if c != ':' {
				return false
			}
			continue
		}
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func HandleInterfaces() {
	if !cgiutil.IsGET() {
		cgiutil.NotAllowed()
		return
	}

	data, err := exec.Command("ip", "-o", "link", "show").Output()
	if err != nil {
		cgiutil.WriteError(err.Error())
		return
	}

	wifiSSIDs := fetchRCIWifiSSIDs()

	var list []Iface
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ": ", 3)
		if len(parts) < 3 {
			continue
		}
		name := strings.TrimSpace(parts[1])
		rest := parts[2]

		state := "UNKNOWN"
		if strings.Contains(rest, "DOWN") {
			state = "DOWN"
		} else if strings.Contains(rest, "UP") {
			state = "UP"
		}

		ipAddr := "--"
		if out, err := exec.Command("ip", "-4", "addr", "show", name).Output(); err == nil {
			for _, aline := range strings.Split(string(out), "\n") {
				aline = strings.TrimSpace(aline)
				if strings.HasPrefix(aline, "inet ") {
					fields := strings.Fields(aline)
					if len(fields) >= 2 {
						ipAddr = strings.SplitN(fields[1], "/", 2)[0]
					}
					break
				}
			}
		}

		linkType, mac := parseLinkType(rest)
		typ := linkTypeNames[linkType]

		// Wi-Fi интерфейсы (ra*, rai*, apcli*) — тип Wi-Fi + SSID по MAC
		isWiFi := strings.HasPrefix(name, "ra") || strings.HasPrefix(name, "rai") || strings.HasPrefix(name, "apcli")
		if isWiFi && (typ == "" || typ == "Ethernet") {
			typ = "Wi-Fi"
		}
		if mac == "" {
			mac = "--"
		}
		speed := ifaceSpeed(name)
		if speed == "" {
			speed = "--"
		}

		ssid := ""
		if isWiFi {
			ssid = wifiSSIDs[strings.ToLower(mac)]
		}

		list = append(list, Iface{
			Name:  name,
			State: state,
			IP:    ipAddr,
			MAC:   mac,
			Type:  typ,
			Speed: speed,
			SSID:  ssid,
		})
	}

	cgiutil.WriteJSON(map[string][]Iface{"interfaces": list})
}
