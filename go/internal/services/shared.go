package services

import (
	"regexp"
)

// serviceNameRe — допустимое имя службы (защита от path traversal в findScript).
var serviceNameRe = regexp.MustCompile(`^[0-9A-Za-z_-]+$`)
