package network

import (
	"encoding/json"
	"fmt"
	"os"
)

func WriteJSON(v any) {
	fmt.Println("Content-type: application/json; charset=utf-8\n")
	json.NewEncoder(os.Stdout).Encode(v)
}

func WriteError(msg string) {
	WriteJSON(map[string]string{"error": msg})
}

func NotAllowed() {
	WriteJSON(map[string]string{"error": "Method not allowed"})
}

func IsGET() bool {
	return os.Getenv("REQUEST_METHOD") == "GET"
}
