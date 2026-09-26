package backup

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"entware-manager/internal/auth"
)

// withAuth включает «панель под паролем» на временном конфиге.
func withAuth(t *testing.T, password string) {
	t.Helper()
	dir := t.TempDir()
	old := auth.ConfigPath
	auth.ConfigPath = dir + "/auth_config.json"
	t.Cleanup(func() { auth.ConfigPath = old })
	if err := os.WriteFile(auth.ConfigPath,
		[]byte(`{"enabled":true,"password":"`+password+`"}`), 0600); err != nil {
		t.Fatal(err)
	}
}

// callBackup дёргает HandleCreate с заданными query и cookie.
func callBackup(t *testing.T, query, cookie string) string {
	t.Helper()
	t.Setenv("REQUEST_METHOD", "GET")
	t.Setenv("QUERY_STRING", query)
	t.Setenv("HTTP_COOKIE", cookie)
	var out bytes.Buffer
	captureStdout(t, HandleCreate, &out)
	return out.String()
}

// Регрессия (аудит P1): в lighttpd-режиме authGate go-сервера нет, и
// backup.cgi отдавал архив с хешем пароля, токеном бота и bridge/*.auth.json
// любому в сети. Теперь требуется сессия панели ИЛИ верный пароль.
func TestBackupCreateRequiresAuth(t *testing.T) {
	withAuth(t, "verysecret")

	// Аноним, пароля нет → отказ.
	out := callBackup(t, "", "")
	if !strings.Contains(out, "Требуется вход в панель") {
		t.Errorf("анонимное скачивание не отклонено: %s", firstLines(out, 3))
	}
	// Неверный пароль → отказ.
	out = callBackup(t, "password=wrong", "")
	if !strings.Contains(out, "Требуется вход в панель") {
		t.Errorf("неверный пароль не отклонён: %s", firstLines(out, 3))
	}
	// Верный пароль → архив.
	out = callBackup(t, "password=verysecret", "")
	if !strings.Contains(out, "Content-type: application/gzip") {
		t.Errorf("верный пароль не дал архив: %s", firstLines(out, 3))
	}
}

// Обычный вход в панель: сессионная cookie из браузера (lighttpd её тоже
// передаёт в HTTP_COOKIE) — пароль не нужен, интерфейс не меняется.
func TestBackupCreateAllowsSession(t *testing.T) {
	withAuth(t, "verysecret")
	oldSess := auth.SessionFile
	auth.SessionFile = t.TempDir() + "/panel_session"
	t.Cleanup(func() { auth.SessionFile = oldSess })
	tok, err := auth.CreateSession()
	if err != nil {
		t.Fatal(err)
	}
	out := callBackup(t, "", auth.SessionCookieName+"="+tok)
	if !strings.Contains(out, "Content-type: application/gzip") {
		t.Errorf("валидная сессия не дала архив: %s", firstLines(out, 3))
	}
}

// Список состава бэкапа — тоже под тем же гейтом (иначе аноним видит
// имена конфигов и дисков).
func TestBackupListRequiresAuth(t *testing.T) {
	withAuth(t, "verysecret")
	out := callBackup(t, "list=1", "")
	if strings.Contains(out, `"filename"`) {
		t.Errorf("аноним получил список бэкапа: %s", firstLines(out, 3))
	}
	if !strings.Contains(out, "Требуется вход в панель") {
		t.Errorf("ожидался отказ, получено: %s", firstLines(out, 3))
	}
}

// Пароль не настроен (панель открыта) — поведение прежнее, отказа нет.
func TestBackupCreateOpenPanel(t *testing.T) {
	dir := t.TempDir()
	old := auth.ConfigPath
	auth.ConfigPath = dir + "/auth_config.json" // файла нет → пароль не настроен
	t.Cleanup(func() { auth.ConfigPath = old })

	out := callBackup(t, "", "")
	if !strings.Contains(out, "Content-type: application/gzip") {
		t.Errorf("на открытой панели скачивание должно работать: %s", firstLines(out, 3))
	}
}

// Регрессия: анонимное ВОССТАНОВЛЕНИЕ позволяло подменить auth_config.json
// (то есть забрать панель под свой пароль). Теперь и оно под гейтом.
func TestBackupRestoreRequiresAuth(t *testing.T) {
	withAuth(t, "verysecret")
	archive := buildArchive(t, []struct {
		name     string
		typeflag byte
		data     string
		size     int64
	}{
		{"auth_config.json", '0', `{"enabled":true,"password":"hacked"}`, 0},
	})

	t.Setenv("REQUEST_METHOD", "POST")
	t.Setenv("QUERY_STRING", "")
	t.Setenv("HTTP_COOKIE", "")

	sandbox := t.TempDir()
	t.Setenv("TMPDIR", sandbox)
	oldWebRoot := webRoot
	webRoot = sandbox + "/web"
	os.MkdirAll(webRoot, 0755)
	t.Cleanup(func() { webRoot = oldWebRoot })

	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	w.Write(archive)
	w.Close()
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })

	var out bytes.Buffer
	captureStdout(t, HandleRestore, &out)
	if !strings.Contains(out.String(), "Требуется вход в панель") {
		t.Fatalf("анонимное восстановление не отклонено: %s", firstLines(out.String(), 3))
	}
	// Файл не создан — конфиг не тронут.
	if _, err := os.Stat(webRoot + "/auth_config.json"); err == nil {
		t.Error("auth_config.json создан анонимным восстановлением — это и есть захват панели")
	}
}

// Отказ должен быть честным: Status 401 (его понимает lighttpd) и ключ
// message -- именно его читает интерфейс (entware.js:3655).
func TestBackupDenyShape(t *testing.T) {
	withAuth(t, "verysecret")
	out := callBackup(t, "", "")
	if !strings.Contains(out, "Status: 401 Unauthorized") {
		t.Errorf("нет заголовка 401: %s", firstLines(out, 3))
	}
	if !strings.Contains(out, `"message":"Требуется вход в панель"`) {
		t.Errorf("нет ключа message (фронт читает его): %s", firstLines(out, 3))
	}
}
