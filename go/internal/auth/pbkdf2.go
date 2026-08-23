// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
//
// Хеширование пароля панели: PBKDF2-HMAC-SHA256 с солью (RFC 8018).
// Ручная реализация без внешних зависимостей (go.mod проекта чистый,
// CGO_ENABLED=0). Формат хеша в auth_config.json:
//
//	pbkdf2-sha256$<итерации>$<salt_hex>$<hash_hex>
//
// Число итераций читается из самой строки (VerifyPassword) — изменение
// DefaultIterations не ломает проверку старых хешей. Для обратной
// совместимости принимается и legacy-формат: голый sha256-hex (64 символа),
// такие хеши при следующем успешном входе мигрируют (NeedsRehash).
package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"runtime"
	"strings"
)

// pbkdf2Prefix — префикс нового формата хеша.
const pbkdf2Prefix = "pbkdf2-sha256$"

// DefaultIterations — число итераций PBKDF2 при создании НОВОГО хеша.
// Выбор по архитектуре: софтверный SHA256 на MIPS (~500–880 МГц) медленный —
// 210000 итераций заняли бы 1–3 сек на каждый запрос; для 32-битных роутеров
// берём компромиссные 60000. Проверка существующих хешей всегда использует
// итерации из stored-строки, поэтому значение можно менять безопасно.
func DefaultIterations() int {
	switch runtime.GOARCH {
	case "amd64", "arm64":
		return 210000
	default:
		return 60000
	}
}

// HashPassword создаёт хеш нового формата для пароля.
func HashPassword(password string) string {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		// crypto/rand недоступен — крайне маловероятно; fail-closed наверху.
		return ""
	}
	iter := DefaultIterations()
	hash := pbkdf2SHA256([]byte(password), salt, iter, 32)
	return fmt.Sprintf("%s%d$%s$%s", pbkdf2Prefix, iter, hex.EncodeToString(salt), hex.EncodeToString(hash))
}

// NeedsRehash — true если хеш старого формата (legacy sha256-hex или пустой/битый)
// и его стоит перехешировать при следующем успешном входе.
func NeedsRehash(stored string) bool {
	return !strings.HasPrefix(stored, pbkdf2Prefix)
}

// VerifyPassword проверяет пароль против stored-хеша (fail-closed):
// новый формат — PBKDF2 с числом итераций из строки; legacy — голый sha256-hex.
// Битый/пустой stored → false. Сравнение constant-time.
func VerifyPassword(password, stored string) bool {
	if strings.HasPrefix(stored, pbkdf2Prefix) {
		iter, salt, want, ok := parsePBKDF2(stored)
		if !ok || iter < 1 {
			return false
		}
		got := pbkdf2SHA256([]byte(password), salt, iter, len(want))
		return subtle.ConstantTimeCompare(got, want) == 1
	}
	// legacy: 64-hex sha256 без соли
	if len(stored) != 64 {
		return false
	}
	if _, err := hex.DecodeString(stored); err != nil {
		return false
	}
	sum := sha256.Sum256([]byte(password))
	got := fmt.Sprintf("%x", sum)
	return subtle.ConstantTimeCompare([]byte(got), []byte(stored)) == 1
}

// parsePBKDF2 разбирает "pbkdf2-sha256$N$salt_hex$hash_hex".
func parsePBKDF2(stored string) (iter int, salt, hash []byte, ok bool) {
	parts := strings.Split(strings.TrimPrefix(stored, pbkdf2Prefix), "$")
	if len(parts) != 3 {
		return 0, nil, nil, false
	}
	n := 0
	for _, ch := range parts[0] {
		if ch < '0' || ch > '9' {
			return 0, nil, nil, false
		}
		n = n*10 + int(ch-'0')
		if n > 10_000_000 { // защита от абсурдных значений
			return 0, nil, nil, false
		}
	}
	salt, err1 := hex.DecodeString(parts[1])
	hash, err2 := hex.DecodeString(parts[2])
	if err1 != nil || err2 != nil || len(salt) == 0 || len(hash) == 0 {
		return 0, nil, nil, false
	}
	return n, salt, hash, true
}

// pbkdf2SHA256 — PBKDF2-HMAC-SHA256 (RFC 8018), keyLen ≤ 64 байт.
func pbkdf2SHA256(password, salt []byte, iter, keyLen int) []byte {
	prf := hmac.New(sha256.New, password)
	hLen := prf.Size()
	numBlocks := (keyLen + hLen - 1) / hLen
	dk := make([]byte, 0, numBlocks*hLen)
	blockBuf := make([]byte, 4)
	u := make([]byte, 0, hLen)
	for block := 1; block <= numBlocks; block++ {
		prf.Reset()
		prf.Write(salt)
		binary.BigEndian.PutUint32(blockBuf, uint32(block))
		prf.Write(blockBuf)
		u = prf.Sum(u[:0])
		t := make([]byte, len(u))
		copy(t, u)
		for i := 2; i <= iter; i++ {
			prf.Reset()
			prf.Write(u)
			u = prf.Sum(u[:0])
			for j := range t {
				t[j] ^= u[j]
			}
		}
		dk = append(dk, t...)
	}
	return dk[:keyLen]
}
