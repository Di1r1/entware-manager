// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
package bridge

// ValidID — публичная проверка идентификатора модуля.
func ValidID(id string) bool { return idRe.MatchString(id) }
