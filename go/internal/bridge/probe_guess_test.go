// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
package bridge

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGuessNumTypeKbs(t *testing.T) {
	cases := []struct {
		path string
		v    float64
		want string
	}{
		{"download_speed", 153600, "kbs"},
		{"upload_rate", 51200, "kbs"},
		{"bw_traffic", 2048, "kbs"},
		{"bw_throughput", 100, "kbs"},
		{"throughput_bytes", 100, "bytes"},
		{"dns.time", 8, "ms"},
		{"mem.size", 1073741824, "bytes"},
		{"num_dns_queries", 96461, "num"},
		{"uptime", 545879, "dur"},
	}
	for _, c := range cases {
		if got := guessNumType(c.path, c.v); got != c.want {
			t.Errorf("guessNumType(%q)=%q, хотим %q", c.path, got, c.want)
		}
	}
}

func TestLooksLikeTopArray(t *testing.T) {
	if !looksLikeTopArray([]interface{}{map[string]interface{}{"ya.ru": float64(500)}}, 4) {
		t.Error("одно-ключевой числовой объект должен быть top")
	}
	if looksLikeTopArray([]interface{}{map[string]interface{}{"site": "ya.ru"}}, 4) {
		t.Error("строковое значение — не top")
	}
	if looksLikeTopArray([]interface{}{map[string]interface{}{"site": "ya.ru", "n": float64(5)}}, 4) {
		t.Error("объект из 2 ключей — не top")
	}
	if looksLikeTopArray(nil, 4) {
		t.Error("пустой массив — не top")
	}
}

func TestPreviewTopArray(t *testing.T) {
	in := []interface{}{
		map[string]interface{}{"ytlh": float64(300)},
		map[string]interface{}{"vk": float64(120)},
		map[string]interface{}{"ya.ru": float64(50)},
	}
	got := previewTopArray(in)
	if !strings.HasPrefix(got, "ytlh (300), vk (120)") {
		t.Errorf("previewTopArray=%q", got)
	}
}

func TestManifestWarnings(t *testing.T) {
	if ws := ManifestWarnings(&Manifest{}); ws != nil {
		t.Errorf("пустой манифест: warnings=%v", ws)
	}
	if ws := ManifestWarnings(&Manifest{Init: "xray", Process: []string{"xray"}}); ws != nil {
		t.Errorf("init == процесс: warnings=%v", ws)
	}
	if ws := ManifestWarnings(&Manifest{Init: "syncthing", Process: []string{"xray"}}); len(ws) != 1 || !strings.Contains(ws[0], "syncthing") {
		t.Errorf("расхождение init/процесс: warnings=%v", ws)
	}
	if ws := ManifestWarnings(&Manifest{Init: "xray", Process: []string{"xray", "syncthing"}}); ws != nil {
		t.Errorf("init среди процессов: warnings=%v", ws)
	}
	if ws := ManifestWarnings(&Manifest{Init: "foo", Process: []string{"bar", "baz"}}); len(ws) != 1 {
		t.Errorf("ожидалось 1 предупреждение, got %v", ws)
	}
}

func TestProbeResultWarningsPopulated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"version": "1.0"}`))
	}))
	defer srv.Close()
	data := []byte(`{"id":"t","name":"T","base":"` + srv.URL + `","probe":{"url":"/"},"process":["xray"],"init":"syncthing"}`)
	res := ProbeManifestData(data)
	if len(res.Warnings) != 1 {
		t.Errorf("ожидалось предупреждение init≠process, got %+v", res.Warnings)
	}
	dataOK := []byte(`{"id":"t","name":"T","base":"` + srv.URL + `","probe":{"url":"/"},"process":["xray"],"init":"xray"}`)
	if res := ProbeManifestData(dataOK); len(res.Warnings) != 0 {
		t.Errorf("init==process не должно давать warnings, got %+v", res.Warnings)
	}
}
