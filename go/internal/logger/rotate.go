package logger

import (
	"os/exec"
)

const rotateScript = "/opt/web_entware/logger/scripts/rotate.sh"

func HandleRotate() {
	if !IsPOST() {
		NotAllowed()
		return
	}

	cmd := exec.Command(rotateScript)
	err := cmd.Run()
	if err != nil {
		WriteJSON(map[string]string{"status": "error", "message": "Ошибка при ротации"})
		return
	}
	WriteJSON(map[string]string{"status": "ok", "message": "Ротация выполнена"})
}
