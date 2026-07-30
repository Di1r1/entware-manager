package stats

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

const authConfigPath = "/opt/web_entware/auth_config.json"

func HandleAuthConfig() {
	if os.Getenv("REQUEST_METHOD") == "POST" {
		handleAuthConfigPost()
		return
	}
	handleAuthConfigGet()
}

func handleAuthConfigGet() {
	data, err := os.ReadFile(authConfigPath)
	if err != nil {
		fmt.Print("Content-type: application/json; charset=utf-8\n\n")
		fmt.Println(`{"enabled":false}`)
		return
	}
	var cfg struct {
		Enabled bool `json:"enabled"`
	}
	if json.Unmarshal(data, &cfg) != nil {
		fmt.Print("Content-type: application/json; charset=utf-8\n\n")
		fmt.Println(`{"enabled":false}`)
		return
	}
	fmt.Print("Content-type: application/json; charset=utf-8\n\n")
	json.NewEncoder(os.Stdout).Encode(map[string]bool{"enabled": cfg.Enabled})
}

func handleAuthConfigPost() {
	body, _ := io.ReadAll(os.Stdin)
	params := parsePostForm(string(body))

	enabled := params["enabled"] == "true"
	password := params["password"]

	// Read existing config for current hash
	var oldHash string
	if data, err := os.ReadFile(authConfigPath); err == nil {
		var old struct {
			Hash string `json:"password_hash"`
		}
		json.Unmarshal(data, &old)
		oldHash = old.Hash
	}

	var passwordHash string

	if enabled {
		if password != "" {
			if len(password) < 4 {
				fmt.Print("Content-type: application/json; charset=utf-8\n\n")
				fmt.Println(`{"status":"error","message":"Пароль должен быть минимум 4 символа"}`)
				return
			}
			h := sha256.Sum256([]byte(password))
			passwordHash = fmt.Sprintf("%x", h)
		} else if oldHash != "" {
			passwordHash = oldHash
		} else {
			fmt.Print("Content-type: application/json; charset=utf-8\n\n")
			fmt.Println(`{"status":"error","message":"Введите пароль"}`)
			return
		}
	}

	newCfg := map[string]interface{}{
		"enabled":       enabled,
		"password_hash": passwordHash,
	}

	data, _ := json.MarshalIndent(newCfg, "", "    ")
	if err := os.WriteFile(authConfigPath, data, 0644); err != nil {
		fmt.Print("Content-type: application/json; charset=utf-8\n\n")
		fmt.Println(`{"status":"error","message":"Не удалось сохранить настройки"}`)
		return
	}

	fmt.Print("Content-type: application/json; charset=utf-8\n\n")
	fmt.Println(`{"status":"ok","message":"Настройки сохранены"}`)
}
