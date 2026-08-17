package packages

import (
	"strconv"
	"strings"
	"time"

	"entware-manager/internal/cgiutil"
)

func HandleAPI() {
	if !cgiutil.IsGET() {
		cgiutil.NotAllowed()
		return
	}

	action := cgiutil.GetQueryParam("action")
	pkgRaw := cgiutil.GetQueryParam("package")

	if action != "info" {
		cgiutil.WriteJSON(map[string]string{"error": "Invalid action"})
		return
	}

	pkg := sanitizePkg(pkgRaw)
	if pkg == "" {
		cgiutil.WriteJSON(map[string]string{"error": "Invalid package name"})
		return
	}

	info, _ := runOpkg("-f", "/opt/etc/opkg.conf", "info", pkg)
	if info == "" {
		cgiutil.WriteJSON(map[string]string{"error": "No information returned by opkg"})
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

	cgiutil.WriteJSON(map[string]string{"info": info})
}
