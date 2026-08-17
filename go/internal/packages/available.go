package packages

import (
	"entware-manager/internal/cgiutil"
	"strings"
)

type PkgInfo struct {
	Package     string `json:"package"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

func Available() {
	if !cgiutil.IsGET() {
		cgiutil.NotAllowed()
		return
	}

	out, code := runOpkg("list")
	if code != 0 {
		cgiutil.WriteJSON([]PkgInfo{})
		return
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	pkgs := make([]PkgInfo, 0)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		pos := strings.Index(line, " - ")
		if pos < 0 {
			continue
		}
		pkg := line[:pos]
		rest := line[pos+3:]
		pos2 := strings.Index(rest, " - ")
		ver := rest
		desc := ""
		if pos2 >= 0 {
			ver = rest[:pos2]
			desc = rest[pos2+3:]
		}
		pkgs = append(pkgs, PkgInfo{Package: pkg, Version: ver, Description: desc})
	}

	cgiutil.WriteJSON(pkgs)
}
