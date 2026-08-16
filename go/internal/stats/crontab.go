package stats

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"entware-manager/internal/auth"
)

func HandleCrontab() {
	qs := os.Getenv("QUERY_STRING")
	typ := getQueryParam(qs, "type")

	content := ""
	switch typ {
	case "system":
		out, err := exec.Command("crontab", "-l").CombinedOutput()
		if err == nil {
			content = string(out)
		}
	case "opt", "":
		data, err := os.ReadFile("/opt/etc/crontab")
		if err == nil {
			content = string(data)
		}
	default:
		fmt.Print("Content-type: application/json; charset=utf-8\n\n")
		json.NewEncoder(os.Stdout).Encode(map[string]string{"error": "Invalid type"})
		return
	}

	fmt.Print("Content-type: application/json; charset=utf-8\n\n")
	json.NewEncoder(os.Stdout).Encode(map[string]string{"crontab": content})
}

func HandleCrontabUpdate() {
	if os.Getenv("REQUEST_METHOD") != "POST" {
		fmt.Print("Content-type: application/json; charset=utf-8\n\n")
		json.NewEncoder(os.Stdout).Encode(map[string]string{"error": "POST required"})
		return
	}

	if auth.IsCrossSiteOrigin() {
		fmt.Print("Content-type: application/json; charset=utf-8\n\n")
		json.NewEncoder(os.Stdout).Encode(map[string]string{"status": "error", "message": auth.CrossSiteDeny})
		return
	}

	body, _ := io.ReadAll(os.Stdin)
	params := parsePostForm(string(body))
	typ := params["type"]
	crontab := params["crontab"]

	// crontab requires trailing newline
	if !strings.HasSuffix(crontab, "\n") {
		crontab += "\n"
	}

	switch typ {
	case "system":
		cmd := exec.Command("crontab", "-")
		cmd.Stdin = strings.NewReader(crontab)
		if err := cmd.Run(); err != nil {
			fmt.Print("Content-type: application/json; charset=utf-8\n\n")
			json.NewEncoder(os.Stdout).Encode(map[string]string{"status": "error", "message": "Invalid crontab"})
			return
		}
		logCrontabAction("INFO", "Сохранён crontab (system)")
		fmt.Print("Content-type: application/json; charset=utf-8\n\n")
		json.NewEncoder(os.Stdout).Encode(map[string]string{"status": "ok"})
	case "opt", "":
		dir := "/opt/etc"
		os.MkdirAll(dir, 0755)
		if err := os.WriteFile(dir+"/crontab", []byte(crontab), 0644); err != nil {
			fmt.Print("Content-type: application/json; charset=utf-8\n\n")
			json.NewEncoder(os.Stdout).Encode(map[string]string{"status": "error", "message": "Failed to write file"})
			return
		}
		if pid := findCronPID(); pid > 0 {
			if proc, err := os.FindProcess(pid); err == nil {
				proc.Signal(syscall.SIGHUP)
			}
		}
		logCrontabAction("INFO", "Сохранён crontab (opt)")
		fmt.Print("Content-type: application/json; charset=utf-8\n\n")
		json.NewEncoder(os.Stdout).Encode(map[string]string{"status": "ok"})
	default:
		fmt.Print("Content-type: application/json; charset=utf-8\n\n")
		json.NewEncoder(os.Stdout).Encode(map[string]string{"status": "error", "message": "Invalid type"})
	}
}

// findCronPID ищет PID демона cron/crond по имени процесса (argv[0]),
// а не по подстроке "cron" во всём cmdline (иначе может убить чужой
// процесс — например, curl на собственный эндпоинт).
func findCronPID() int {
	return findCronPIDIn("/proc")
}

func findCronPIDIn(procDir string) int {
	entries, err := os.ReadDir(procDir)
	if err != nil {
		return 0
	}
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		cmdline, err := os.ReadFile(fmt.Sprintf("%s/%d/cmdline", procDir, pid))
		if err != nil {
			continue
		}
		args := strings.Split(strings.TrimRight(string(cmdline), "\x00"), "\x00")
		if len(args) == 0 || args[0] == "" {
			continue
		}
		name := filepath.Base(args[0])
		if name == "cron" || name == "crond" {
			return pid
		}
	}
	return 0
}

func logCrontabAction(level, msg string) {
	logFile := fmt.Sprintf("/tmp/entware/logs/%s.log", time.Now().Format("2006-01-02"))
	ts := time.Now().Format("2006-01-02 15:04:05")
	ip := os.Getenv("REMOTE_ADDR")
	if ip == "" {
		ip = "0.0.0.0"
	}
	entry := fmt.Sprintf("[%s] [%s] [%s] [%d] [crontab_update.cgi] %s\n", ts, level, ip, os.Getpid(), msg)
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		f.WriteString(entry)
		f.Close()
	}
}
