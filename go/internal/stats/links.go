package stats

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
)

type Link struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Icon string `json:"icon"`
}

var defaultLinks = []Link{
	{Name: "Роутер", URL: "http://192.168.3.1", Icon: "router"},
	{Name: "Entware Manager", URL: "http://192.168.3.1:8087/entware-manager/", Icon: "package"},
	{Name: "AdGuard Home", URL: "http://192.168.3.1:3000", Icon: "shield"},
	{Name: "Transmission", URL: "http://192.168.3.1:9091", Icon: "download"},
	{Name: "Netdata", URL: "http://192.168.3.1:19999", Icon: "chart"},
	{Name: "htop (ttyd)", URL: "http://192.168.3.1:8089", Icon: "process"},
	{Name: "Терминал (ttyd)", URL: "http://192.168.3.1:9089", Icon: "terminal"},
}

func HandleLinksLoad() {
	data, err := os.ReadFile("/opt/web_entware/links.json")
	if err == nil {
		fmt.Println("Content-type: application/json; charset=utf-8\n")
		os.Stdout.Write(data)
		return
	}

	routerIP := getRouterIP()
	links := make([]Link, len(defaultLinks))
	for i, l := range defaultLinks {
		links[i] = Link{
			Name: l.Name,
			Icon: l.Icon,
			URL:  strings.ReplaceAll(l.URL, "192.168.3.1", routerIP),
		}
	}

	fmt.Println("Content-type: application/json; charset=utf-8\n")
	json.NewEncoder(os.Stdout).Encode(links)
}

func getRouterIP() string {
	remote := os.Getenv("REMOTE_ADDR")
	if remote != "" {
		conn, err := net.Dial("udp", remote+":80")
		if err == nil {
			defer conn.Close()
			if addr := conn.LocalAddr().(*net.UDPAddr); addr != nil {
				return addr.IP.String()
			}
		}
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return "192.168.3.1"
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			if ipnet.IP.To4() != nil {
				if isPrivateIP(ipnet.IP) {
					return ipnet.IP.String()
				}
			}
		}
	}
	return "192.168.3.1"
}

func isPrivateIP(ip net.IP) bool {
	if ip4 := ip.To4(); ip4 != nil {
		switch {
		case ip4[0] == 10:
			return true
		case ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31:
			return true
		case ip4[0] == 192 && ip4[1] == 168:
			return true
		}
	}
	return false
}
