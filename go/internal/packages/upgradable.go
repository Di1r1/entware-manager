package packages

import (
	"strings"
)

type UpgradablePkg struct {
	Package string `json:"package"`
	Current string `json:"current"`
	New     string `json:"new"`
}

func Upgradable() {
	if !isGET() {
		methodNotAllowed()
		return
	}

	out, code := runOpkg("list-upgradable")
	if code != 0 {
		writeJSON([]UpgradablePkg{})
		return
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	pkgs := make([]UpgradablePkg, 0)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 5 {
			pkg := parts[0]
			cur := parts[2]
			newVer := parts[4]
			if pkg != "" && cur != "" && newVer != "" && newVer != "-" {
				pkgs = append(pkgs, UpgradablePkg{
					Package: pkg,
					Current: cur,
					New:     newVer,
				})
			}
		}
	}

	writeJSON(pkgs)
}
