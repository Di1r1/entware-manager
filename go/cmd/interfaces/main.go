package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type iface struct {
	Name  string `json:"name"`
	State string `json:"state"`
	IP    string `json:"ip"`
}

func main() {
	if os.Getenv("REQUEST_METHOD") != "GET" {
		json.NewEncoder(os.Stdout).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	fmt.Println("Content-type: application/json; charset=utf-8\n")

	data, err := exec.Command("ip", "-o", "link", "show").Output()
	if err != nil {
		json.NewEncoder(os.Stdout).Encode(map[string]string{"error": err.Error()})
		return
	}

	var list []iface
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

		list = append(list, iface{Name: name, State: state, IP: ipAddr})
	}

	json.NewEncoder(os.Stdout).Encode(map[string][]iface{"interfaces": list})
}
