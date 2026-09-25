package monitor

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
)

// captureMonitorOut перехватывает stdout CGI-ответа.
func captureMonitorOut(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	os.Stdout = old
	return buf.String()
}

// postKillPID вызывает HandleKillPID с телом pid=<pid>.
func postKillPID(t *testing.T, pid int) string {
	t.Helper()
	t.Setenv("REQUEST_METHOD", "POST")
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	w.WriteString("pid=" + strconv.Itoa(pid))
	w.Close()
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })
	return captureMonitorOut(t, HandleKillPID)
}

// Регрессия: принимался любой положительный PID, включая init (1).
func TestPidKillableRejectsCritical(t *testing.T) {
	deny := []int{0, -1, 1, os.Getpid(), os.Getppid()}
	for _, pid := range deny {
		if pidKillable(pid) {
			t.Errorf("pidKillable(%d) = true, want false", pid)
		}
	}
	if !pidKillable(4242) {
		t.Error("pidKillable(4242) = false, want true (обычный процесс)")
	}
}

// Сквозной сценарий: попытка убить сам CGI-процесс должна быть отклонена.
// Без защиты тест-процесс был бы убит (падает весь go test).
func TestHandleKillPID_RejectsSelf(t *testing.T) {
	self := os.Getpid()
	out := postKillPID(t, self)
	if !strings.Contains(out, `"status":"error"`) {
		t.Fatalf("ожидался отказ, получено: %s", out)
	}
	if !pidAlive(self) {
		t.Fatal("тест-процесс убит — защита PID не сработала")
	}
}

// Регресс: обычный чужой процесс по-прежнему завершается.
func TestHandleKillPID_KillsForeignProcess(t *testing.T) {
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Skipf("не удалось запустить sleep: %v", err)
	}
	pid := cmd.Process.Pid
	t.Cleanup(func() {
		cmd.Process.Kill()
		cmd.Wait()
	})

	waitAlive := func() bool {
		for i := 0; i < 100; i++ {
			if pidAlive(pid) {
				return true
			}
			time.Sleep(20 * time.Millisecond)
		}
		return false
	}
	if !waitAlive() {
		t.Skipf("sleep (pid %d) не появился в /proc", pid)
	}

	out := postKillPID(t, pid)
	if !strings.Contains(out, `"status":"ok"`) {
		t.Fatalf("ожидался ok, получено: %s", out)
	}
	for i := 0; i < 100 && pidAlive(pid); i++ {
		time.Sleep(20 * time.Millisecond)
	}
	if pidAlive(pid) {
		t.Errorf("процесс %d не завершён", pid)
	}
}
