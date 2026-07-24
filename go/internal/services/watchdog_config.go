package services

import (
	"encoding/json"
	"io"
	"os"
	"syscall"
	"time"
)

func HandleWatchdogConfig() {
	switch method := os.Getenv("REQUEST_METHOD"); method {
	case "GET":
		handleWrapperConfigGet()
	case "POST":
		handleWrapperConfigPost()
	default:
		NotAllowed()
	}
}

func handleWrapperConfigGet() {
	data, err := os.ReadFile(wrapperConfig)
	if err != nil || !json.Valid(data) {
		data = []byte(defaultServiceConfig)
	}
	os.Stdout.WriteString("Content-type: application/json; charset=utf-8\n\n")
	os.Stdout.Write(data)
}

func handleWrapperConfigPost() {
	body, err := io.ReadAll(os.Stdin)
	if err != nil {
		WriteJSON(map[string]string{"status": "error", "message": "Failed to read request"})
		return
	}

	data := string(body)
	if !json.Valid([]byte(data)) {
		WriteJSON(map[string]string{"status": "error", "message": "Invalid JSON configuration"})
		return
	}

	if err := os.WriteFile(wrapperConfig, []byte(data), 0644); err != nil {
		WriteJSON(map[string]string{"status": "error", "message": "Failed to write config"})
		return
	}

	if pid := readWrapperPID(); pid > 0 && pidAlive(pid) {
		syscall.Kill(pid, syscall.SIGHUP)
		time.Sleep(time.Second)
		WriteJSON(map[string]string{"status": "ok", "message": "Конфигурация сохранена, демон перезагружен"})
	} else {
		WriteJSON(map[string]string{"status": "ok", "message": "Конфигурация сохранена (демон не запущен)"})
	}
}
