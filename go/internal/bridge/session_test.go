// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
package bridge

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Сценарий: защищённый сервис требует cookie → авто-логин по creds → retry.
func TestAuthedDoCookieLogin(t *testing.T) {
	logins := 0
	var inner http.Handler
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			logins++
			var in struct {
				Password string `json:"password"`
			}
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Password != "svc-pass" {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			http.SetCookie(w, &http.Cookie{Name: "app_session", Value: "tok123", Path: "/"})
			w.Write([]byte(`{"status":"ok"}`))
		default:
			ck, errC := r.Cookie("app_session")
			if errC == nil && ck.Value == "tok123" {
				inner.ServeHTTP(w, r)
				return
			}
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	defer srv.Close()
	inner = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"secret":"data"}`))
	})

	dir := t.TempDir()
	// глобальный путь сессии мог остаться от прошлого запуска тестов
	os.RemoveAll(filepath.Join("/tmp/entware/bridge", sanitize("svc")+".session"))
	os.WriteFile(filepath.Join(dir, "svc.auth.json"),
		[]byte(`{"type":"cookie_login","password":"svc-pass","login_url":"/login"}`), 0600)

	client := clientBridge()
	resp, err := authedDo(client, dir, "svc", http.MethodGet, srv.URL+"/status")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body := readAllStr(resp)
	if resp.StatusCode != 200 || !strings.Contains(body, `"secret":"data"`) {
		t.Errorf("после авто-логина ожидался 200 с данными, got %d %q", resp.StatusCode, body)
	}
	if logins != 1 {
		t.Errorf("логинов = %d, хочу 1", logins)
	}
	// cookie сохранена для следующего CGI-процесса
	if loadSessionCookie(dir, "svc") != "app_session=tok123" {
		t.Errorf("cookie не сохранена: %q", loadSessionCookie(dir, "svc"))
	}
}

func TestAuthedDoBasic(t *testing.T) {
	var gotUser, gotPass string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser, gotPass, _ = r.BasicAuth()
		if gotUser != "u1" || gotPass != "p1" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Write([]byte(`ok`))
	}))
	defer srv.Close()

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "b.auth.json"),
		[]byte(`{"type":"basic","username":"u1","password":"p1"}`), 0600)

	client := clientBridge()
	resp, err := authedDo(client, dir, "b", http.MethodGet, srv.URL+"/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("basic-auth: код %d", resp.StatusCode)
	}
	if gotUser != "u1" || gotPass != "p1" {
		t.Error("basic-заголовок не передан")
	}
}

func readAllStr(resp *http.Response) string {
	b := make([]byte, 4096)
	n, _ := resp.Body.Read(b)
	return string(b[:n])
}
