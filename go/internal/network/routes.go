package network

import (
	"os/exec"
	"strings"
)

type Route struct {
	Destination string `json:"destination"`
	Gateway     string `json:"gateway"`
	Interface   string `json:"interface"`
	Metric      string `json:"metric,omitempty"`
}

func HandleRoutes() {
	if !IsGET() { NotAllowed(); return }

	data, err := exec.Command("ip", "route", "show").Output()
	if err != nil {
		WriteError(err.Error())
		return
	}

	var list []Route
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" { continue }

		dest := strings.Fields(line)[0]
		gateway := "0.0.0.0"
		iface := "--"
		metric := ""

		fields := strings.Fields(line)
		for i, f := range fields {
			if f == "via" && i+1 < len(fields) {
				gateway = fields[i+1]
			}
			if f == "dev" && i+1 < len(fields) {
				iface = fields[i+1]
			}
			if f == "metric" && i+1 < len(fields) {
				metric = fields[i+1]
			}
		}

		r := Route{Destination: dest, Gateway: gateway, Interface: iface}
		if metric != "" {
			r.Metric = metric
		}
		list = append(list, r)
	}

	WriteJSON(map[string][]Route{"routes": list})
}
