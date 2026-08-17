package network

import (
	"encoding/json"
	"entware-manager/internal/cgiutil"
	"io"
	"os"

	"entware-manager/internal/auth"
)

const defaultConfig = `{
  "enabled": true,
  "interval": 30,
  "watch_interfaces": ["eth0"],
  "watch_internet": true,
  "ping_host": "8.8.8.8",
  "ping_timeout": 5,
  "autostart": false,
  "notify_on": ["interface_down", "no_internet", "ip_changed"]
}`

func HandleConfig() {
	switch os.Getenv("REQUEST_METHOD") {
	case "GET":
		handleConfigGet()
	case "POST":
		if auth.IsCrossSiteOrigin() {
			cgiutil.WriteJSON(map[string]string{"status": "error", "message": auth.CrossSiteDeny})
			return
		}
		handleConfigPost()
	default:
		cgiutil.NotAllowed()
	}
}

func handleConfigGet() {
	data, err := os.ReadFile(ConfigFile)
	if err != nil || !json.Valid(data) {
		data = []byte(defaultConfig)
	} else {
		var cfg map[string]interface{}
		if json.Unmarshal(data, &cfg) == nil {
			if _, ok := cfg["autostart"]; !ok {
				cfg["autostart"] = false
				if patched, err := json.MarshalIndent(cfg, "", "  "); err == nil {
					data = patched
				}
			}
		}
	}
	os.Stdout.WriteString("Content-type: application/json; charset=utf-8\n\n")
	os.Stdout.Write(data)
}

func handleConfigPost() {
	body, err := io.ReadAll(os.Stdin)
	if err != nil {
		cgiutil.WriteJSON(map[string]string{"status": "error", "message": "Failed to read request"})
		return
	}

	data := string(body)
	if !json.Valid([]byte(data)) {
		cgiutil.WriteJSON(map[string]string{"status": "error", "message": "Invalid JSON"})
		return
	}

	if err := os.WriteFile(ConfigFile, []byte(data), 0644); err != nil {
		cgiutil.WriteJSON(map[string]string{"status": "error", "message": "Failed to write config"})
		return
	}

	cgiutil.WriteJSON(map[string]string{"status": "ok"})
}
