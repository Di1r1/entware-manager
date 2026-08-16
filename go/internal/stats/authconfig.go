package stats

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"entware-manager/internal/auth"
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
		fmt.Println(`{"enabled":false,"configured":false}`)
		return
	}
	var cfg struct {
		Enabled bool   `json:"enabled"`
		Hash    string `json:"password_hash"`
		Pass    string `json:"password"`
	}
	if json.Unmarshal(data, &cfg) != nil {
		fmt.Print("Content-type: application/json; charset=utf-8\n\n")
		fmt.Println(`{"enabled":false,"configured":false}`)
		return
	}
	configured := cfg.Hash != "" || cfg.Pass != ""
	fmt.Print("Content-type: application/json; charset=utf-8\n\n")
	json.NewEncoder(os.Stdout).Encode(map[string]bool{"enabled": cfg.Enabled, "configured": configured})
}

func handleAuthConfigPost() {
	if auth.IsCrossSiteOrigin() {
		fmt.Print("Content-type: application/json; charset=utf-8\n\n")
		fmt.Println(`{"status":"error","message":"` + auth.CrossSiteDeny + `"}`)
		return
	}
	body, _ := io.ReadAll(os.Stdin)
	params := parsePostForm(string(body))

	enabled := params["enabled"] == "true"
	password := params["password"]
	currentPassword := params["current_password"]

	// Read existing config for current hash
	var oldHash string
	var oldEnabled bool
	var oldPlain string
	existingCfg := false
	if data, err := os.ReadFile(authConfigPath); err == nil {
		var old struct {
			Enabled  bool   `json:"enabled"`
			Hash     string `json:"password_hash"`
			Password string `json:"password"`
		}
		if json.Unmarshal(data, &old) == nil {
			existingCfg = true
			oldEnabled = old.Enabled
			oldHash = old.Hash
			oldPlain = old.Password
		}
	}

	// Защита от несанкционированной смены/отключения: если авторизация уже
	// включена (или был настроен пароль) — требуется текущий пароль.
	if existingCfg {
		authNeedsCheck := oldEnabled || oldHash != "" || oldPlain != ""
		if authNeedsCheck && currentPassword == "" {
			fmt.Print("Content-type: application/json; charset=utf-8\n\n")
			fmt.Println(`{"status":"error","message":"Введите текущий пароль"}`)
			return
		}
		if authNeedsCheck {
			h := sha256.Sum256([]byte(currentPassword))
			hashOk := oldHash != "" && fmt.Sprintf("%x", h) == oldHash
			plainOk := oldHash == "" && oldPlain != "" && oldPlain == currentPassword
			if !hashOk && !plainOk {
				fmt.Print("Content-type: application/json; charset=utf-8\n\n")
				fmt.Println(`{"status":"error","message":"Неверный текущий пароль"}`)
				return
			}
		}
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
	data = append(data, '\n')
	if err := os.WriteFile(authConfigPath, data, 0600); err != nil {
		fmt.Print("Content-type: application/json; charset=utf-8\n\n")
		fmt.Println(`{"status":"error","message":"Не удалось сохранить настройки"}`)
		return
	}

	// Смена/отключение пароля инвалидирует все существующие сессии:
	// файл /opt/var/run/panel_session удаляется, старые cookie умирают
	// (в обоих режимах гейт проверяет именно этот файл).
	auth.DestroySession()

	fmt.Print("Content-type: application/json; charset=utf-8\n\n")
	fmt.Println(`{"status":"ok","message":"Настройки сохранены"}`)
}
