// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
package bridge

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func findPath(paths []ProbePath, path string) *ProbePath {
	for i := range paths {
		if paths[i].Path == path {
			return &paths[i]
		}
	}
	return nil
}

func TestFlattenJSON(t *testing.T) {
	pc := &pathCollector{}
	body := map[string]interface{}{
		"version":             "1.2.3",
		"protection_enabled":  true,
		"num_dns_queries":     float64(96461),
		"avg_processing_time": float64(0.015),
		"inBytesTotal":        float64(1073741824),
		"total":               map[string]interface{}{"queries": float64(42)},
		"top_clients":         []interface{}{map[string]interface{}{"ya.ru": float64(500)}},
	}
	flattenJSON("", body, 0, pc)
	out := pc.paths
	if len(out) < 8 {
		t.Fatalf("мало путей: %d", len(out))
	}

	if p := findPath(out, "version"); p == nil || p.Preview != "1.2.3" || p.Guess != "" {
		t.Errorf("version: %+v", p)
	}
	if p := findPath(out, "protection_enabled"); p == nil || p.Guess != "bool" {
		t.Errorf("protection_enabled: %+v", p)
	}
	if p := findPath(out, "num_dns_queries"); p == nil || p.Guess != "num" {
		t.Errorf("num_dns_queries: %+v", p)
	}
	if p := findPath(out, "avg_processing_time"); p == nil || p.Guess != "ms" {
		t.Errorf("avg_processing_time: %+v (ожидался ms)", p)
	}
	if p := findPath(out, "inBytesTotal"); p == nil || p.Guess != "bytes" {
		t.Errorf("inBytesTotal: %+v (ожидался bytes)", p)
	}
	if p := findPath(out, "total.queries"); p == nil || p.Preview != "42" {
		t.Errorf("total.queries: %+v", p)
	}
	if p := findPath(out, "top_clients"); p == nil || p.Guess != "count" {
		t.Errorf("top_clients: %+v", p)
	}
	if p := findPath(out, "top_clients.0.ya.ru"); p == nil || p.Preview != "500" {
		t.Errorf("top_clients.0.ya.ru: %+v", p)
	}
}

func TestFlattenJSONLimits(t *testing.T) {
	big := map[string]interface{}{}
	for i := 0; i < maxProbePaths+20; i++ {
		big["k"+string(rune('a'+i%26))+itoa(i)] = float64(i)
	}
	pc := &pathCollector{}
	flattenJSON("", big, 0, pc)
	if len(pc.paths) > maxProbePaths {
		t.Errorf("лимит %d превышен: %d", maxProbePaths, len(pc.paths))
	}
	if pc.total <= len(pc.paths) {
		t.Errorf("total должен считать все пути: total=%d paths=%d", pc.total, len(pc.paths))
	}
}

// TestProbeManifestData — сканер по тексту: валидный манифест с живым
// httptest-сервером на 127.0.0.1 + мягкий разбор невалидного текста.
func TestProbeManifestData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/stats":
			w.Write([]byte(`{"num_blocked_filtering": 7}`))
		default:
			w.Write([]byte(`{"protection_enabled": true, "version": "1.0"}`))
		}
	}))
	defer srv.Close()

	valid := `{"id":"t","name":"T","base":"` + srv.URL + `","probe":{"url":"/"},"status":{"url":"/"},"stats":{"url":"/stats"},
		"extra":{"hist":{"url":"/history"}}}`
	res := ProbeManifestData([]byte(valid))
	if !res.Valid {
		t.Fatalf("валидный манифест помечен невалидным: %s", res.ValidationError)
	}
	if res.ValidationError != "" {
		t.Fatalf("лишняя ошибка: %s", res.ValidationError)
	}
	if len(res.Sources) != 3 { // status, stats, extra.hist
		t.Fatalf("источников: %d (%v)", len(res.Sources), res.Sources)
	}
	st := res.Sources[0]
	if st.Name != "status" || st.HTTPCode != 200 || st.Error != "" {
		t.Fatalf("status: %+v", st)
	}
	if p := findPath(st.Paths, "protection_enabled"); p == nil || p.Guess != "bool" {
		t.Errorf("status.protection_enabled: %+v", p)
	}
	statsSrc := res.Sources[1]
	if statsSrc.Name != "stats" || findPath(statsSrc.Paths, "num_blocked_filtering") == nil {
		t.Errorf("stats: %+v", statsSrc)
	}
	ex := res.Sources[2]
	if ex.Name != "extra.hist" || ex.Error != "" {
		t.Errorf("extra.hist: %+v", ex)
	}

	// Невалидный текст (плохой type поля, нет actions и т.д.), но URL извлечь можно
	lenient := `{"id":"x","name":"X","base":"` + srv.URL + `","probe":{"url":"/"},"bogus_field":1}`
	res2 := ProbeManifestData([]byte(lenient))
	if res2.Valid {
		t.Error("невалидный манифест помечен валидным")
	}
	if res2.ValidationError == "" {
		t.Error("нет ошибки валидации для битого манифеста")
	}
	if len(res2.Sources) != 1 || res2.Sources[0].Name != "status" || res2.Sources[0].Error != "" {
		t.Errorf("lenient-скан не сработал: %+v", res2.Sources)
	}

	// Совсем битый JSON — только ошибка
	res3 := ProbeManifestData([]byte("{nope"))
	if res3.Valid || len(res3.Sources) != 0 || res3.ValidationError == "" {
		t.Errorf("битый JSON: %+v", res3)
	}

	// SSRF-URL — источник с ошибкой, не запрос
	ssrf := `{"id":"y","name":"Y","base":"http://8.8.8.8","probe":{"url":"/"},"status":{"url":"/"}}`
	res4 := ProbeManifestData([]byte(ssrf))
	if res4.Valid {
		t.Error("SSRF-манифест помечен валидным")
	}
	if len(res4.Sources) != 1 || res4.Sources[0].Error == "" {
		t.Errorf("SSRF-URL должен дать ошибку источника: %+v", res4.Sources)
	}
}

func TestProbeSourceJSONShape(t *testing.T) {
	b, _ := json.Marshal(ProbeSource{Name: "status", HTTPCode: 200,
		Paths: []ProbePath{{Path: "a.b", Preview: "x", Guess: "bool"}}})
	s := string(b)
	for _, want := range []string{`"path":"a.b"`, `"preview":"x"`, `"guess":"bool"`, `"http_code":200`} {
		if !jsonContains(s, want) {
			t.Errorf("в %s нет %s", s, want)
		}
	}
}

func jsonContains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestLoopbackListeningPorts(t *testing.T) {
	ports := LoopbackListeningPorts()
	if len(ports) == 0 {
		t.Skip("нет слушающих портов в окружении теста")
	}
	for i := 1; i < len(ports); i++ {
		if ports[i] <= ports[i-1] {
			t.Fatalf("порты не отсортированы/дубликат: %v", ports)
		}
	}
	if len(ports) > 48 {
		t.Errorf("слишком много портов: %d", len(ports))
	}
}

func TestDescribeDialError(t *testing.T) {
	cases := []struct {
		msg, want string
	}{
		{"Get http://127.0.0.1:1/: dial tcp 127.0.0.1:1: connect: connection refused", "соединение отклонено"},
		{"Get http://127.0.0.1:1/: dial tcp: i/o timeout", "таймаут"},
		{"context deadline exceeded", "таймаут"},
		{"что-то иное", "не отвечает"},
	}
	for _, c := range cases {
		got := describeDialError(errString(c.msg))
		if !strings.Contains(got, c.want) {
			t.Errorf("describeDialError(%q) = %q, ждём подстроку %q", c.msg, got, c.want)
		}
	}
}

type errString string

func (e errString) Error() string { return string(e) }

func TestReachableFromLoopback(t *testing.T) {
	cases := []struct {
		hex  string
		want bool
	}{
		{"0100007F", true},                          // 127.0.0.1
		{"00000000", true},                          // 0.0.0.0
		{"256433C6", false},                         // внешний адрес
		{"00000000000000000000000000000000", true},  // ::
		{"00000000000000000000000000000001", true},  // ::1 (big-endian ядро)
		{"00000000000000000000000001000000", true},  // ::1 (little-endian ядро: перевёрнутые слова)
		{"020001200000000000000000256433C6", false}, // внешний v6
	}
	for _, c := range cases {
		if got := reachableFromLoopback(c.hex); got != c.want {
			t.Errorf("reachableFromLoopback(%q) = %v, ждём %v", c.hex, got, c.want)
		}
	}
}

// TestProbeSuggestions — порт отвечает (404 на мусорный путь), но данные
// не получены → сканер предлагает известный API-путь Netdata.
func TestProbeSuggestions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/info" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"version":"v1.33.1","ram_total":"1024"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	tmpl := `{"id":"t","name":"T","base":"` + srv.URL + `","probe":{"url":"/"},"status":{"url":"/api/status"}}`
	res := ProbeManifestData([]byte(tmpl))
	if len(res.Suggestions) == 0 {
		t.Fatalf("подсказок нет: %+v", res)
	}
	found := false
	for _, s := range res.Suggestions {
		if s.Service == "Netdata" && s.Path == "/api/v1/info" {
			found = true
		}
	}
	if !found {
		t.Errorf("нет подсказки Netdata: %+v", res.Suggestions)
	}

	// рабочий манифест — подсказки не нужны
	ok := `{"id":"t2","name":"T","base":"` + srv.URL + `","probe":{"url":"/api/v1/info"},"status":{"url":"/api/v1/info"}}`
	res2 := ProbeManifestData([]byte(ok))
	if len(res2.Suggestions) != 0 {
		t.Errorf("лишние подсказки при рабочих источниках: %+v", res2.Suggestions)
	}

	// мёртвый порт — подсказки не ищем
	dead := `{"id":"t3","name":"T","base":"http://127.0.0.1:1","probe":{"url":"/"},"status":{"url":"/x"}}`
	res3 := ProbeManifestData([]byte(dead))
	if len(res3.Suggestions) != 0 {
		t.Errorf("подсказки для недоступного порта: %+v", res3.Suggestions)
	}
}
