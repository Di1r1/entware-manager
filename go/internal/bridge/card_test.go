// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
package bridge

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestBuildCardUniversal(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Write([]byte(`{"protection_enabled":true,"num_dns_queries":1000,
			"num_blocked_filtering":50,"top_clients":[{"192.168.3.5":900}]}`))
	}))
	defer srv.Close()

	dir := t.TempDir()
	SetBridgeDir(dir)
	defer SetBridgeDir("/opt/web_entware/bridge")

	os.WriteFile(dir+"/adg.json", []byte(`{
		"id":"adg","name":"AdGuard Test","base":"`+srv.URL+`",
		"probe":{"url":"/"},
		"status":{"url":"/control/status"},
		"stats":{"url":"/control/stats"},
		"fields":[
			{"path":"protection_enabled","label":"Защита","type":"bool","tile":true},
			{"path":"num_dns_queries","label":"Запросов","from":"stats","tile":true},
			{"path":"num_blocked_filtering","label":"Блокировано","from":"stats","tile":true,"color":"#d69e2e"}
		]}`), 0644)

	card, err := BuildCard(dir, "adg")
	if err != nil {
		t.Fatal(err)
	}
	if len(card.Tiles) != 3 {
		t.Fatalf("плиток = %d, хочу 3", len(card.Tiles))
	}
	if card.Tiles[0].Value != "да" {
		t.Errorf("bool → %q, хочу «да»", card.Tiles[0].Value)
	}
	if card.Tiles[1].Value != "1 000" {
		t.Errorf("число → %q", card.Tiles[1].Value)
	}
	if len(card.Rows) != 0 {
		t.Errorf("лишние ряды: %v", card.Rows)
	}
}

func TestLookupPath(t *testing.T) {
	src := map[string]interface{}{
		"a": map[string]interface{}{"b": map[string]interface{}{"c": float64(42)}},
	}
	v, ok := lookupPath(src, stringsSplit("a.b.c"))
	if !ok || v != float64(42) {
		t.Errorf("lookupPath = %v %v", v, ok)
	}
	if _, ok := lookupPath(src, []string{"x"}); ok {
		t.Error("несуществующий путь найден")
	}
}

func stringsSplit(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == '.' {
			out = append(out, cur)
			cur = ""
			continue
		}
		cur += string(r)
	}
	out = append(out, cur)
	return out
}

func TestFormatValueStringNumbers(t *testing.T) {
	// Netdata отдаёт числа строками: "1039433728" → 991.3 МБ, "3" → 3
	if got := formatValue("1039433728", "bytes"); got != "991.3 МБ" {
		t.Errorf("bytes из строки: %q (ожидался \"991.3 МБ\")", got)
	}
	if got := formatValue("3", "num"); got != "3" {
		t.Errorf("num из строки: %q", got)
	}
	if got := formatValue("12", "count"); got != "12" {
		t.Errorf("count из строки: %q", got)
	}
	if got := formatValue("abc", "bytes"); got != "" {
		t.Errorf("нестрока-число должна дать пусто: %q", got)
	}
}
