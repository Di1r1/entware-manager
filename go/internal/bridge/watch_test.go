// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
package bridge

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// setupWatch — мок-сервис (переключаемый healthy), манифест, prefs, изолир. каталоги.
func setupWatch(t *testing.T, healthy *bool) (string, *bool) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if *healthy {
			w.Write([]byte(`{"ok":true}`))
		} else {
			w.WriteHeader(http.StatusBadGateway)
		}
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	SetBridgeDir(dir)
	os.WriteFile(filepath.Join(dir, "svc.json"), []byte(
		`{"id":"svc","name":"Test Service","base":"`+srv.URL+`/","probe":{"url":"/ping"}}`), 0644)
	os.WriteFile(filepath.Join(dir, "_prefs.json"),
		[]byte(`{"modules":{"svc":{"enabled":true,"notifications":true}}}`), 0600)

	t.Cleanup(func() { SetBridgeDir("/opt/web_entware/bridge") })
	return dir, healthy
}

func readDailyLog() string {
	data, _ := os.ReadFile(filepath.Join(watchLogDir, time.Now().Format("2006-01-02")+".log"))
	return string(data)
}

func TestWatchDownAndRecovery(t *testing.T) {
	healthy := true
	dir, healthyPtr := setupWatch(t, &healthy)
	watchStateDir = filepath.Join(t.TempDir(), "st")
	watchLogDir = t.TempDir()
	t.Cleanup(func() {
		watchStateDir = "/tmp/entware/bridge"
		watchLogDir = "/tmp/entware/logs"
	})
	// лог за сегодня может содержать старые записи — считаем дельту
	before := strings.Count(readDailyLog(), "[bridge]")

	// 1: база (первый проход, без алерта)
	RunWatch(dir)
	// 2: сервис падает → fails=1 < 2, алерта нет
	*healthyPtr = false
	evs := RunWatch(dir)
	if len(evs) != 0 {
		t.Fatalf("fails=1 не должен давать событие, получено %d", len(evs))
	}
	// 3: fails=2 → DOWN
	evs = RunWatch(dir)
	if len(evs) != 1 || !strings.Contains(evs[0], "перестал отвечать") {
		t.Fatalf("ожидался down, получено: %v", evs)
	}
	// 4-5: пока сервис лежит — событий нет
	if evs := RunWatch(dir); len(evs) != 0 {
		t.Fatalf("лежащий сервис не должен генерить события: %v", evs)
	}
	// 6: восстановился → oks=1 < 2, алерта нет
	*healthyPtr = true
	if evs := RunWatch(dir); len(evs) != 0 {
		t.Fatalf("oks=1 не должен давать recovery: %v", evs)
	}
	// 7: oks=2 → RECOVERY
	evs = RunWatch(dir)
	if len(evs) != 1 || !strings.Contains(evs[0], "восстановился") {
		t.Fatalf("ожидался recovery, получено: %v", evs)
	}

	after := strings.Count(readDailyLog(), "[bridge]")
	if after-before != 2 {
		t.Errorf("[bridge] строк в логе: до=%d после=%d, хочу +2", before, after)
	}
}

func TestWatchDisabledSkipped(t *testing.T) {
	dir, _ := setupWatch(t, new(bool))
	watchStateDir = filepath.Join(t.TempDir(), "st")
	watchLogDir = t.TempDir()
	t.Cleanup(func() {
		watchStateDir = "/tmp/entware/bridge"
		watchLogDir = "/tmp/entware/logs"
	})
	// выключаем модуль
	pf := LoadPrefs()
	m := pf.Modules["svc"]
	m.Enabled = false
	pf.Modules["svc"] = m
	SavePrefs(pf)

	stBefore := func() string {
		b, _ := os.ReadFile(watchStatePath())
		return string(b)
	}
	os.Remove(watchStatePath())
	RunWatch(dir) // база: выключенный модуль не попадает в state
	if stBefore() != "" && strings.Contains(stBefore(), `"svc"`) {
		t.Error("выключенный модуль не должен мониториться")
	}
}

func TestWatchStatePersistsAcrossRuns(t *testing.T) {
	h := true
	dir, hp := setupWatch(t, &h)
	_ = hp
	watchStateDir = filepath.Join(t.TempDir(), "st")
	watchLogDir = t.TempDir()
	t.Cleanup(func() {
		watchStateDir = "/tmp/entware/bridge"
		watchLogDir = "/tmp/entware/logs"
	})
	RunWatch(dir)
	RunWatch(dir)
	st := loadWatchState()
	w := st.Modules["svc"]
	if w == nil || !w.Up {
		t.Fatalf("state не сохранился между вызовами: %+v", st.Modules)
	}
}
