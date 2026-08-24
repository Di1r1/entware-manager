// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
package bridge

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// RunAction выполняет действие коннектора (URL уже провалидирован манифестом).
// Ответ upstream не проксируется в панель (изоляция от чужого HTML) — только код.
func RunAction(m *Manifest, a *Action) (*StatusProxy, error) {
	u, err := ValidateBridgeURL(a.URL, m.Base)
	if err != nil {
		return nil, err
	}
	method := a.Method
	if method == "" {
		method = http.MethodPost
	}
	client := clientBridge()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, u.String(), nil)
	if err != nil {
		return nil, err
	}
	applyAuth(req, LoadAuth(dirForManifest(m), m.ID))
	resp, err := client.Do(req)
	if err != nil {
		return &StatusProxy{HTTPCode: 0, Error: "сервис не отвечает"}, nil
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, io.LimitReader(resp.Body, maxStatusBody))

	return &StatusProxy{
		HTTPCode: resp.StatusCode,
		Raw:      fmt.Sprintf("HTTP %d", resp.StatusCode),
	}, nil
}

// dirForManifest — каталог, из которого загружен манифест (для auth-файла).
// В v1 манифесты лежат в единственном каталоге bridgeDir.
func dirForManifest(m *Manifest) string { return bridgeDirVar }
