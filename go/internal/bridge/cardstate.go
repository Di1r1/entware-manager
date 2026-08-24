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
	// манифестный сервис: считаем живым, если файл есть
	if HasManifestFile(dir, id) {
		return "running"
	}
	return "absent"
}
