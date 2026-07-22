package packages

import (
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

func HandleAPI() {
	if !isGET() {
		methodNotAllowed()
		return
	}

	qs := os.Getenv("QUERY_STRING")
	action := getQueryParam(qs, "action")
	pkgRaw := getQueryParam(qs, "package")

	if action != "info" {
		writeJSON(map[string]string{"error": "Invalid action"})
		return
	}

	pkg := sanitizePkg(pkgRaw)
	if pkg == "" {
		writeJSON(map[string]string{"error": "Invalid package name"})
		return
	}

	info, _ := runOpkg("-f", "/opt/etc/opkg.conf", "info", pkg)
	if info == "" {
		writeJSON(map[string]string{"error": "No information returned by opkg"})
		return
	}

	// Parse Installed-Time and add Installed-Date
	lines := strings.Split(info, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "Installed-Time:") {
			tsStr := strings.TrimSpace(strings.TrimPrefix(line, "Installed-Time:"))
			ts, err := strconv.ParseInt(tsStr, 10, 64)
			if err == nil && ts > 0 {
				dateStr := time.Unix(ts, 0).Format("2006-01-02 15:04:05")
				lines[i] = line
				// Insert Installed-Date after Installed-Time
				rest := append([]string{}, lines[i+1:]...)
				lines = append(lines[:i+1], "Installed-Date: "+dateStr)
				lines = append(lines, rest...)
			}
			break
		}
	}
	info = strings.Join(lines, "\n")

	writeJSON(map[string]string{"info": info})
}

func getQueryParam(qs, key string) string {
	for _, part := range strings.Split(qs, "&") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 && kv[0] == key {
			val, err := url.QueryUnescape(kv[1])
			if err != nil {
				return kv[1]
			}
			return val
		}
	}
	return ""
}
