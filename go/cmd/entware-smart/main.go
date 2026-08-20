// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
package main

import (
	_ "entware-manager/internal/localtime"
	"entware-manager/internal/smart"
)

func main() {
	smart.HandleSmart()
}
