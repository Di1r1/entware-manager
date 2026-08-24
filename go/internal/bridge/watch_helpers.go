// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
package bridge

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type syncWatch struct{ wg sync.WaitGroup }

func (w *syncWatch) add(n int) { w.wg.Add(n) }
func (w *syncWatch) done()     { w.wg.Done() }
func (w *syncWatch) wait()     { w.wg.Wait() }

type syncMutex struct{ mu sync.Mutex }

func (m *syncMutex) lock()   { m.mu.Lock() }
func (m *syncMutex) unlock() { m.mu.Unlock() }

func ioCopyDiscard(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
	}
}

// appendDailyLog пишет строку в суточный лог /tmp/entware/logs/<дата>.log
// (тот же файл, что читает telegram_gateway).
func appendDailyLog(line string) {
	os.MkdirAll(watchLogDir, 0755)
	path := filepath.Join(watchLogDir, time.Now().Format("2006-01-02")+".log")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		f.WriteString(line + "\n")
		f.Close()
	}
}
