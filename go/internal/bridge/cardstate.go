package bridge

import (
	"fmt"
	"io"
	"net/http"
)

func mkURL(port int, path string) string {
	return fmt.Sprintf("http://127.0.0.1:%d%s", port, path)
}

func readSmall(resp *http.Response) []byte {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxProbeBody))
	resp.Body.Close()
	return body
}

// DiscoverState — состояние одного сервиса через существующий discovery-проб.
func DiscoverState(dir, id string) string {
	for _, e := range BuiltInCatalog() {
		if e.ID != id {
			continue
		}
		client := clientBridge()
		var best ServiceState
		for _, port := range e.Ports {
			resp, err := authedDo(client, dir, id, http.MethodGet,
				mkURL(port, e.Path), "")
			if err != nil {
				continue
			}
			body := readSmall(resp)
			best = classify(resp.StatusCode, resp.Header.Get("Content-Type"), body)
			if best.State != "absent" {
				break
			}
		}
		if best.State == "" {
			best.State = "absent"
		}
		return best.State
	}
	// манифестный сервис: процесс = источник истины; иначе веб-проба
	// зеркалит Discover (бюджет/классификация), чтобы статус карточки
	// Статистики не расходился с вкладкой Модули.
	m, err := LoadManifest(dir, id)
	if err != nil {
		return "absent"
	}
	if len(m.Process) > 0 {
		if pids := matchProcs(snapshotProcs(), m.Process); len(pids) > 0 {
			return "running"
		}
		return "absent"
	}
	if m.Probe.URL == "" {
		// ни процесса, ни адреса: зеркалим Discover (который на пустой probe
		// вернёт absent) — статусы карточки и вкладки не расходятся.
		return "absent"
	}
	u, err := ValidateBridgeURL(m.Probe.URL, m.Base)
	if err != nil {
		return "absent"
	}
	client := clientBridge()
	if len(m.Ports) > 0 {
		best := ServiceState{State: "absent"}
		for _, port := range m.Ports {
			resp, err := authedDo(client, dir, id, m.Probe.MethodOrGET(),
				fmt.Sprintf("http://127.0.0.1:%d%s?%s", port, u.Path, u.RawQuery),
				m.Probe.Body)
			if err != nil {
				continue
			}
			body := readSmall(resp)
			best = classify(resp.StatusCode, resp.Header.Get("Content-Type"), body)
			if best.State != "absent" {
				break
			}
		}
		return best.State
	}
	resp, err := authedDo(client, dir, id, m.Probe.MethodOrGET(), u.String(), m.Probe.Body)
	if err != nil {
		return "absent"
	}
	body := readSmall(resp)
	return classify(resp.StatusCode, resp.Header.Get("Content-Type"), body).State
}
