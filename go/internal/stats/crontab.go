package stats

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
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
		fmt.Println("Content-type: application/json; charset=utf-8\n")
		json.NewEncoder(os.Stdout).Encode(map[string]string{"error": "Invalid type"})
		return
	}

	fmt.Println("Content-type: application/json; charset=utf-8\n")
	json.NewEncoder(os.Stdout).Encode(map[string]string{"crontab": content})
}

func HandleCrontabUpdate() {
	if os.Getenv("REQUEST_METHOD") != "POST" {
		fmt.Println("Content-type: application/json; charset=utf-8\n")
		json.NewEncoder(os.Stdout).Encode(map[string]string{"error": "POST required"})
		return
	}

	body, _ := io.ReadAll(os.Stdin)
	params := parsePostForm(string(body))
	typ := params["type"]
	crontab := params["crontab"]

	switch typ {
	case "system":
		cmd := exec.Command("crontab")
		cmd.Stdin = strings.NewReader(crontab)
		if err := cmd.Run(); err != nil {
			fmt.Println("Content-type: application/json; charset=utf-8\n")
			json.NewEncoder(os.Stdout).Encode(map[string]string{"status": "error", "message": "Invalid crontab"})
			return
		}
		logCrontabAction("INFO", "Сохранён crontab (system)")
		fmt.Println("Content-type: application/json; charset=utf-8\n")
		json.NewEncoder(os.Stdout).Encode(map[string]string{"status": "ok"})
	case "opt", "":
		dir := "/opt/etc"
		os.MkdirAll(dir, 0755)
		if err := os.WriteFile(dir+"/crontab", []byte(crontab), 0644); err != nil {
			fmt.Println("Content-type: application/json; charset=utf-8\n")
			json.NewEncoder(os.Stdout).Encode(map[string]string{"status": "error", "message": "Failed to write file"})
			return
		}
		if pid := findCronPID(); pid > 0 {
			if proc, err := os.FindProcess(pid); err == nil {
				proc.Signal(syscall.SIGHUP)
			}
		}
		logCrontabAction("INFO", "Сохранён crontab (opt)")
		fmt.Println("Content-type: application/json; charset=utf-8\n")
		json.NewEncoder(os.Stdout).Encode(map[string]string{"status": "ok"})
	default:
		fmt.Println("Content-type: application/json; charset=utf-8\n")
		json.NewEncoder(os.Stdout).Encode(map[string]string{"status": "error", "message": "Invalid type"})
	}
}

func findCronPID() int {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0
	}
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		cmdline, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
		if err != nil {
			continue
		}
		args := strings.Split(strings.TrimRight(string(cmdline), "\x00"), "\x00")
		for _, arg := range args {
			if strings.Contains(arg, "cron") {
				return pid
			}
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
