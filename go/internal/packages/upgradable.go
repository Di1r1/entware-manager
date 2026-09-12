package packages

import (
	"entware-manager/internal/cgiutil"
	"strings"
)

type UpgradablePkg struct {
	Package string `json:"package"`
	Current string `json:"current"`
	New     string `json:"new"`
}

func Upgradable() {
	if !cgiutil.IsGET() {
		cgiutil.NotAllowed()
		return
	}

	cgiutil.WriteJSON(CollectUpgradable())
}

// CollectUpgradable — чистое значение для dashboard.cgi.
// Форма ровно та, что отдаёт upgradable.cgi.
func CollectUpgradable() []UpgradablePkg {
	out, code := runOpkgList("list-upgradable")
	if code != 0 {
		return []UpgradablePkg{}
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

	return pkgs
}
