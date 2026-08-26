// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
//
// bridge_watch — фоновый мониторинг модулей моста для Telegram-алертов.
//
// Вызывается шлюзом раз в цикл (без сессии — только локальные пробы и свой лог).
// Для каждого модуля с prefs.enabled && prefs.notifications:
//   - проба статуса;
//   - анти-дребезг: 2 подряд неудачи → «упал», 2 подряд успеха после падения
//     → «поднялся» (auth_required считается живым);
//   - ПЕРЕХОДЫ пишутся в суточный лог тегом [bridge] → их подхватывает
//     telegram_gateway (источник bridge 🔐🧩) и рассылает в Telegram.
package bridge

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type moduleWatch struct {
	Fails int    `json:"fails"`
	Oks   int    `json:"oks"`
	Up    bool   `json:"up"` // последнее известное состояние
	Name  string `json:"name"`
}

type watchState struct {
	Modules map[string]*moduleWatch `json:"modules"`
}

// Каталоги мониторинга (переопределяются в тестах).
var (
	watchStateDir = "/tmp/entware/bridge"
	watchLogDir   = "/tmp/entware/logs"
)

// Состояние мониторинга живёт в tmpfs (переживать ребут не нужно:
// после него первый проход молча создаёт базу) — берегём flash.
func watchStatePath() string { return filepath.Join(watchStateDir, ".state.json") }

// withWatchLock — межпроцессное исключение для read-modify-write состояния.
// Lock-файл O_EXCL; зависший (старше 15с) удаляется. Возврат false = не удалось.
func withWatchLock(fn func()) bool {
	lockDir := filepath.Dir(watchStatePath())
	os.MkdirAll(lockDir, 0700)
	lockPath := filepath.Join(lockDir, ".watch.lock")
	if fi, err := os.Stat(lockPath); err == nil && time.Since(fi.ModTime()) > 15*time.Second {
		os.Remove(lockPath) // зависший lock
	}
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return false // занято другим процессом — этот цикл пропускаем
	}
	f.Close()
	defer os.Remove(lockPath)
	fn()
	return true
}

func loadWatchState() watchState {
	st := watchState{Modules: map[string]*moduleWatch{}}
	data, err := os.ReadFile(watchStatePath())
	if err != nil {
		return st
	}
	var p watchState
	if json.Unmarshal(data, &p) == nil && p.Modules != nil {
		return p
	}
	return st
}

func saveWatchState(st watchState) error {
	out, err := json.MarshalIndent(st, "", "    ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	tmp := watchStatePath() + ".tmp"
	if err := os.WriteFile(tmp, out, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmp, watchStatePath()); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

const (
	watchFailsToDown = 2
	watchOksToUp     = 2
)

// RunWatch — основная логика мониторинга. Возвращает список событий-переходов
// (уже записанных в лог).
func RunWatch(dir string) []string {
	prefs := LoadPrefs()
	client := clientBridge()

	var (
		wg      syncWatch
		mu      syncMutex
		results = map[string]string{} // id → состояние пробы
	)

	// Один снимок процессов на проход, если хоть один модуль на process-детекте
	// (иначе пустой probe давал ложные «упал» — P1 кворума v1.15.7).
	var procSnap []procEntry
	for id := range prefs.Modules {
		if mf, err := LoadManifest(dir, id); err == nil && len(mf.Process) > 0 {
			procSnap = snapshotProcs()
			break
		}
	}

	for id, m := range prefs.Modules {
		if !m.Enabled || !m.Notifications {
			continue // выключенные или без уведомлений не мониторим
		}
		if !ValidID(id) {
			continue
		}
		wg.add(1)
		go func(id string, snap []procEntry) {
			defer wg.done()
			mf, err := LoadManifest(dir, id)
			if err != nil {
				mu.lock()
				results[id] = "absent"
				mu.unlock()
				return
			}
			// process-детект: как в discovery — probe игнорируется полностью.
			if len(mf.Process) > 0 {
				state := "absent"
				if len(matchProcs(snap, mf.Process)) > 0 {
					state = "running"
				}
				mu.lock()
				results[id] = state
				mu.unlock()
				return
			}
			u, err := ValidateBridgeURL(mf.Probe.URL, mf.Base)
			if err != nil {
				mu.lock()
				results[id] = "absent"
				mu.unlock()
				return
			}
			resp, err := authedDo(client, dir, id, http.MethodGet, u.String(), "")
			if err != nil {
				mu.lock()
				results[id] = "absent"
				mu.unlock()
				return
			}
			ioCopyDiscard(resp)
			code := resp.StatusCode
			state := "running"
			if code == 401 || code == 403 {
				state = "auth_required" // жив, просто заперт — не алерт
			} else if code >= 400 || code < 200 {
				state = "absent"
			}
			mu.lock()
			results[id] = state
			mu.unlock()
		}(id, procSnap)
	}
	wg.wait()

	var events []string
	withWatchLock(func() {
		st := loadWatchState()
		for id, state := range results {
			name := id
			if mf, err := LoadManifest(dir, id); err == nil && mf.Name != "" {
				name = mf.Name
			}
			up := state == "running" || state == "auth_required"

			w := st.Modules[id]
			if w == nil {
				w = &moduleWatch{Up: up, Name: name} // первый проход — база, без алертов
			}
			w.Name = name

			var evType string
			if up {
				w.Oks++
				w.Fails = 0
				if !w.Up && w.Oks >= watchOksToUp {
					evType = "recovery"
					w.Up = true
				}
			} else {
				w.Fails++
				w.Oks = 0
				if w.Up && w.Fails >= watchFailsToDown {
					evType = "down"
					w.Up = false
				}
			}
			st.Modules[id] = w

			switch evType {
			case "down":
				line := fmt.Sprintf("[%s] [WARN] [127.0.0.1] [%d] [bridge] Модуль «%s» перестал отвечать",
					time.Now().Format("2006-01-02 15:04:05"), os.Getpid(), name)
				appendDailyLog(line)
				events = append(events, line)
			case "recovery":
				line := fmt.Sprintf("[%s] [INFO] [127.0.0.1] [%d] [bridge] Модуль «%s» восстановился",
					time.Now().Format("2006-01-02 15:04:05"), os.Getpid(), name)
				appendDailyLog(line)
				events = append(events, line)
			}
		}

		// Модуль удалён из prefs → чистим его состояние
		for id := range st.Modules {
			if _, ok := prefs.Modules[id]; !ok {
				delete(st.Modules, id)
			}
		}

		_ = saveWatchState(st)
	})
	return events
}
