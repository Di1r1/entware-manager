package network

import (
	"os/exec"
	"strings"
)

type Iface struct {
	Name  string `json:"name"`
	State string `json:"state"`
	IP    string `json:"ip"`
}

func HandleInterfaces() {
	if !IsGET() { NotAllowed(); return }

	data, err := exec.Command("ip", "-o", "link", "show").Output()
	if err != nil {
		WriteError(err.Error())
		return
	}

	var list []Iface
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" { continue }

		parts := strings.SplitN(line, ": ", 3)
		if len(parts) < 3 { continue }
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

		list = append(list, Iface{Name: name, State: state, IP: ipAddr})
	}

	WriteJSON(map[string][]Iface{"interfaces": list})
}
