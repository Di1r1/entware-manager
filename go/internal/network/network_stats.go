package network

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type ifaceIP struct {
	Iface string `json:"iface"`
	IP    string `json:"ip"`
}

type wifiBridge struct {
	Name string `json:"name"`
	G2g  string `json:"2g"`
	G5g  string `json:"5g"`
	Rx   string `json:"rx"`
	Tx   string `json:"tx"`
}

type portInfo struct {
	Iface   string `json:"iface"`
	Speed   string `json:"speed"`
	Carrier string `json:"carrier"`
}

type networkInfo struct {
	Name    string `json:"name"`
	Bridge  string `json:"bridge"`
	Members string `json:"members"`
}

type statsResponse struct {
	Interfaces []ifaceIP    `json:"interfaces"`
	LAN        string       `json:"lan"`
	WiFi       string       `json:"wifi"`
	WiFiInfo   []wifiBridge `json:"wifi_info"`
	WAN        string       `json:"wan"`
	Ports      []portInfo   `json:"ports"`
	Networks   []networkInfo `json:"networks"`
}

func HandleNetworkStats() {
	if !IsGET() {
		NotAllowed()
		return
	}

	resp := statsResponse{
		Interfaces: getInterfacesWithIPs(),
		WiFiInfo:   getWiFiInfo(),
		Ports:      getPhysicalPorts(),
		Networks:   getNetworksStatus(),
	}

	resp.LAN = buildLAN(resp.Interfaces)
	resp.WiFi = getWiFiStatus()
	resp.WAN = getWANStatus()

	WriteJSON(resp)
}

func getInterfacesWithIPs() []ifaceIP {
	data, err := exec.Command("ip", "-4", "addr", "show").Output()
	if err != nil {
		return []ifaceIP{{Iface: "-", IP: "--"}}
	}

	var list []ifaceIP
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "inet ") {
			continue
		}
		ip := strings.Fields(line)[1]
		ip = strings.SplitN(ip, "/", 2)[0]
		if ip == "127.0.0.1" {
			continue
		}

		lastField := strings.Fields(line)
		iface := lastField[len(lastField)-1]

		list = append(list, ifaceIP{Iface: iface, IP: ip})
	}

	if len(list) == 0 {
		return []ifaceIP{{Iface: "-", IP: "--"}}
	}
	return list
}

func buildLAN(list []ifaceIP) string {
	var ips []string
	for _, iface := range list {
		if iface.IP != "--" {
			ips = append(ips, iface.IP)
		}
	}
	if len(ips) == 0 {
		return "--"
	}
	return strings.Join(ips, ", ")
}

func getWiFiStatus() string {
	data, err := exec.Command("ip", "link", "show").Output()
	if err != nil {
		return "--"
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := strings.TrimSpace(fields[1])
		name = strings.TrimSuffix(name, ":")
		if name == "br0" || strings.HasPrefix(name, "ra") {
			if strings.Contains(line, "UP") {
				return "подключено"
			}
		}
	}
	return "--"
}

func getWiFiInfo() []wifiBridge {
	data, err := exec.Command("brctl", "show").Output()
	if err != nil {
		return []wifiBridge{{Name: "--", G2g: "--", G5g: "--", Rx: "--", Tx: "--"}}
	}

	devData := readProcNetDev()

	var bridges []wifiBridge
	var currentBridge string
	var currentIfaces []string

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line == "bridge name" || strings.HasPrefix(line, "bridge") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		first := fields[0]

		if strings.HasPrefix(first, "b") {
			if currentBridge != "" && len(currentIfaces) > 0 {
				b := buildWiFiBridge(currentBridge, currentIfaces, devData)
				bridges = append(bridges, b)
			}
			currentBridge = first
			currentIfaces = nil
			continue
		}

		currentIfaces = append(currentIfaces, first)
	}

	if currentBridge != "" && len(currentIfaces) > 0 {
		b := buildWiFiBridge(currentBridge, currentIfaces, devData)
		bridges = append(bridges, b)
	}

	if len(bridges) == 0 {
		return []wifiBridge{{Name: "--", G2g: "--", G5g: "--", Rx: "--", Tx: "--"}}
	}
	return bridges
}

func readProcNetDev() string {
	data, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return ""
	}
	return string(data)
}

func calcTrafficMB(devData, iface string) int64 {
	for _, line := range strings.Split(devData, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, iface+":") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 10 {
			return 0
		}
		rx, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			return 0
		}
		return rx / 1024 / 1024
	}
	return 0
}

func formatTraffic(mb int64) string {
	if mb > 1024 {
		return fmt.Sprintf("%d GB", mb/1024)
	}
	return fmt.Sprintf("%d MB", mb)
}

func buildWiFiBridge(bridge string, ifaces []string, devData string) wifiBridge {
	b := wifiBridge{Name: "--", G2g: "--", G5g: "--", Rx: "--", Tx: "--"}

	switch bridge {
	case "br0":
		b.Name = "LAN"
	case "br1":
		b.Name = "Guest"
	case "br2":
		b.Name = "Guest2"
	default:
		b.Name = bridge
	}

	var maxRX, maxTX int64
	for _, iface := range ifaces {
		switch {
		case strings.HasPrefix(iface, "ra") && !strings.HasPrefix(iface, "rai"):
			if b.G2g == "--" {
				b.G2g = iface
			}
		case strings.HasPrefix(iface, "rai"):
			if b.G5g == "--" {
				b.G5g = iface
			}
		}
	}

	for _, iface := range ifaces {
		rxMB := calcTrafficMB(devData, iface)
		txMB := calcTrafficTXMB(devData, iface)
		if rxMB > maxRX {
			maxRX = rxMB
		}
		if txMB > maxTX {
			maxTX = txMB
		}
	}

	b.Rx = formatTraffic(maxRX)
	b.Tx = formatTraffic(maxTX)

	return b
}

func calcTrafficTXMB(devData, iface string) int64 {
	for _, line := range strings.Split(devData, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, iface+":") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 10 {
			return 0
		}
		tx, err := strconv.ParseInt(fields[9], 10, 64)
		if err != nil {
			return 0
		}
		return tx / 1024 / 1024
	}
	return 0
}

func getWANStatus() string {
	for _, iface := range []string{"ppp0", "ppoe0", "wwan0"} {
		data, err := exec.Command("ip", "link", "show", iface).Output()
		if err != nil {
			continue
		}
		if strings.Contains(string(data), "UP") {
			return iface + " (up)"
		}
	}

	data, err := exec.Command("ip", "route", "show", "default").Output()
	if err != nil {
		return "--"
	}
	fields := strings.Fields(string(data))
	if len(fields) < 5 {
		return "down"
	}
	iface := fields[4]
	data2, err := exec.Command("ip", "link", "show", iface).Output()
	if err != nil {
		return "down"
	}
	if strings.Contains(string(data2), "UP") {
		return iface + " (up)"
	}
	return "down"
}

func getPhysicalPorts() []portInfo {
	matches, err := filepath.Glob("/sys/class/net/eth*")
	if err != nil {
		return []portInfo{{Iface: "—", Speed: "—", Carrier: "—"}}
	}

	var list []portInfo
	for _, path := range matches {
		name := filepath.Base(path)
		if strings.Contains(name, ".") {
			continue
		}

		p := portInfo{Iface: name, Speed: "—", Carrier: "—"}

		carrier, err := os.ReadFile(filepath.Join(path, "carrier"))
		if err == nil && strings.TrimSpace(string(carrier)) == "1" {
			p.Carrier = "✓"

			speed, err := os.ReadFile(filepath.Join(path, "speed"))
			if err == nil {
				s := strings.TrimSpace(string(speed))
				if s != "" && s != "0" && s != "-1" {
					p.Speed = s + "Mbps"
				}
			}
		}

		list = append(list, p)
	}

	if len(list) == 0 {
		return []portInfo{{Iface: "—", Speed: "—", Carrier: "—"}}
	}
	return list
}

func getNetworksStatus() []networkInfo {
	var list []networkInfo

	data, err := exec.Command("brctl", "show").Output()
	if err == nil {
		var currentBridge string
		var currentMembers []string

		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || line == "bridge name" || strings.HasPrefix(line, "bridge") {
				continue
			}

			fields := strings.Fields(line)
			if len(fields) == 0 {
				continue
			}

			first := fields[0]

			if strings.HasPrefix(first, "b") {
				if currentBridge != "" {
					list = append(list, buildNetworkInfo(currentBridge, currentMembers))
				}
				currentBridge = first
				currentMembers = nil
				continue
			}

			currentMembers = append(currentMembers, first)
		}

		if currentBridge != "" {
			list = append(list, buildNetworkInfo(currentBridge, currentMembers))
		}
	}

	wanIface := getWANInterface()
	wanName := "WAN"
	list = append(list, networkInfo{Name: wanName, Bridge: wanIface, Members: ""})

	return list
}

func getWANInterface() string {
	for _, iface := range []string{"ppp0", "ppoe0", "wwan0"} {
		data, err := exec.Command("ip", "link", "show", iface).Output()
		if err != nil {
			continue
		}
		if strings.Contains(string(data), "UP") {
			return iface
		}
	}

	data, err := exec.Command("ip", "route", "show", "default").Output()
	if err != nil {
		return "—"
	}
	fields := strings.Fields(string(data))
	if len(fields) >= 5 {
		return fields[4]
	}
	return "—"
}

func buildNetworkInfo(bridge string, members []string) networkInfo {
	name := bridge
	switch bridge {
	case "br0":
		name = "LAN"
	case "br1":
		name = "Guest"
	case "br2":
		name = "Guest2"
	}

	sort.Strings(members)
	return networkInfo{
		Name:    name,
		Bridge:  bridge,
		Members: strings.TrimSpace(strings.Join(members, " ")),
	}
}
