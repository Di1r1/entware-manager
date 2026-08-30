// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
package bridge

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
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
	resp, err := authedDo(client, dir, "svc", http.MethodGet, srv.URL+"/status", "")
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
	resp, err := authedDo(client, dir, "b", http.MethodGet, srv.URL+"/", "")
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

// Регрессия: большие тела (>256КБ) должны читаться полностью — контекст
// запроса не должен отменяться до завершения чтения тела.
func TestAuthedDoLargeBody(t *testing.T) {
	points := make([]string, 1200) // ~600КБ JSON
	for i := range points {
		points[i] = `{"t":` + itoa2(i) + `,"rx":1000,"tx":2000}`
	}
	payload := "[" + strings.Join(points, ",") + "]"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(payload))
	}))
	defer srv.Close()

	client := clientBridge()
	resp, err := authedDo(client, t.TempDir(), "svc2", http.MethodGet, srv.URL+"/", "")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxExtraBody))
	if len(body) < len(payload) {
		t.Fatalf("тело обрезано: %d из %d байт", len(body), len(payload))
	}
	var arr []interface{}
	if json.Unmarshal(body, &arr) != nil {
		t.Error("полное тело не парсится")
	} else if len(arr) != 1200 {
		t.Errorf("точек = %d, хочу 1200", len(arr))
	}
}

func itoa2(i int) string { return strconv.Itoa(i) }

// Регрессия MAJOR-A: 409-флоу Transmission — ретрай обязан нести тело
// (раньше strings.NewReader исчерпывался и ретрай уходил пустым).
func TestAuthedDo409RetriesWithBody(t *testing.T) {
	var sizes []int
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		sizes = append(sizes, len(b))
		t.Logf("handler: hdr=%q body=%d", r.Header.Get("X-Transmission-Session-Id"), len(b))
		gotMethod = r.Method
		if r.Header.Get("X-Transmission-Session-Id") == "" {
			w.Header().Set("X-Transmission-Session-Id", "tok42")
			w.WriteHeader(http.StatusConflict)
			return
		}
		w.Write([]byte(`{"result":"success"}`))
	}))
	defer srv.Close()

	dir := t.TempDir()
	os.Remove(sessionPath(dir, "tr")) // глобальный tmpfs — чистим сессию прошлых прогонов
	client := clientBridge()
	resp, err := authedDo(client, dir, "tr", http.MethodPost, srv.URL+"/", `{"method":"session-stats"}`)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 || !strings.Contains(string(body), "success") {
		t.Errorf("финал: %d %q", resp.StatusCode, string(body))
	}
	if len(sizes) != 2 || sizes[0] == 0 || sizes[1] == 0 {
		t.Errorf("размеры тел попыток: %v — обе должны быть непустыми", sizes)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("метод = %q", gotMethod)
	}
}

func TestValidateAuthCreds(t *testing.T) {
	// cookie_login: обязателен относительный путь
	if err := ValidateAuthCreds("cookie_login", "/api/login"); err != nil {
		t.Errorf("валидный path → %v", err)
	}
	if err := ValidateAuthCreds("cookie_login", ""); err == nil {
		t.Errorf("пустой login_url должен отклоняться")
	}
	if err := ValidateAuthCreds("cookie_login", "http://127.0.0.1/api/login"); err == nil {
		t.Errorf("абсолютный URL должен отклоняться")
	}
	if err := ValidateAuthCreds("cookie_login", "//evil.com/login"); err == nil {
		t.Errorf("network-path reference должен отклоняться")
	}
	// basic/api_key: login_url не используется
	if err := ValidateAuthCreds("basic", ""); err != nil {
		t.Errorf("basic без login_url → %v", err)
	}
	if err := ValidateAuthCreds("basic", "/x"); err == nil {
		t.Errorf("basic с login_url должен отклоняться")
	}
	if err := ValidateAuthCreds("api_key", "/x"); err == nil {
		t.Errorf("api_key с login_url должен отклоняться")
	}
}
