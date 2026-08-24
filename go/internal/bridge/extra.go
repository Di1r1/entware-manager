// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
package bridge

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ProxyExtra — прокси именованного extra-эндпоинта манифеста.
func ProxyExtra(dir, id, name string) (*StatusProxy, error) {
	m, err := LoadManifest(dir, id)
	if err != nil {
		return nil, err
	}
	ep := m.Extra[name]
	if ep == nil {
		return nil, fmt.Errorf("extra-эндпоинт %q не найден", name)
	}
	u, err := ValidateBridgeURL(ep.URL, m.Base)
	if err != nil {
		return nil, err
	}
	client := clientBridge()
	resp, err := authedDo(client, dir, id, http.MethodGet, u.String())
	if err != nil {
		return &StatusProxy{Error: "сервис не отвечает"}, nil
	}
	defer resp.Body.Close()
	// Читаем полностью (истории бывают >256КБ) — валидность проверит парсер.
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxExtraBody))
	sp := &StatusProxy{HTTPCode: resp.StatusCode}
	var v interface{}
	if json.Unmarshal(body, &v) != nil {
		if len(body) >= maxExtraBody {
			return &StatusProxy{Error: "ответ сервиса слишком большой"}, nil
		}
		sp.Raw = truncate(string(body), 512)
		return sp, nil
	}
	// slice_last: оставить последние N элементов массива
	if ep.SliceLast > 0 {
		if arr, ok := v.([]interface{}); ok && len(arr) > ep.SliceLast {
			v = arr[len(arr)-ep.SliceLast:]
		}
	}
	out, err2 := json.Marshal(v)
	if err2 != nil {
		sp.Raw = truncate(string(body), 512)
		return sp, nil
	}
	sp.Body = json.RawMessage(out)
	return sp, nil
}
