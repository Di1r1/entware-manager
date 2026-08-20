// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
package main

import (
	"entware-manager/internal/cgiutil"
	"os"

	"entware-manager/internal/auth"
	_ "entware-manager/internal/localtime"
	"entware-manager/internal/packages"
)

func main() {
	ep := os.Getenv("ENDPOINT")
	if ep == "" {
		cgiutil.WriteError("ENDPOINT not set")
		return
	}

	if os.Getenv("REQUEST_METHOD") == "POST" && auth.IsCrossSiteOrigin() {
		cgiutil.WriteError(auth.CrossSiteDeny)
		return
	}

	switch ep {
	case "available":
		packages.Available()
	case "packages":
		packages.Installed()
	case "installed":
		packages.InstalledJSON()
	case "install":
		packages.Install()
	case "remove":
		packages.Remove()
	case "upgrade":
		packages.Upgrade()
	case "update":
		packages.Update()
	case "upgradable":
		packages.Upgradable()
	case "api":
		packages.HandleAPI()
	default:
		cgiutil.WriteError("unknown endpoint: " + ep)
	}
}
