// Di1r1
package main

import (
	"os"

	"entware-manager/internal/auth"
	_ "entware-manager/internal/localtime"
	"entware-manager/internal/packages"
)

func main() {
	ep := os.Getenv("ENDPOINT")
	if ep == "" {
		packages.WriteError("ENDPOINT not set")
		return
	}

	if os.Getenv("REQUEST_METHOD") == "POST" && auth.IsCrossSiteOrigin() {
		packages.WriteError(auth.CrossSiteDeny)
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
		packages.WriteError("unknown endpoint: " + ep)
	}
}
