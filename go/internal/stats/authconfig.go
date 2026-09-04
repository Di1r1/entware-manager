package stats

import (
	"encoding/json"
	"entware-manager/internal/cgiutil"
	"fmt"
	"io"
	"os"

	"entware-manager/internal/auth"
)

const authConfigPath = "/opt/web_entware/auth_config.json"
const authMarkerPath = "/opt/web_entware/.auth_configured"

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
	params := cgiutil.ParseFormBody(string(body))

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
			hashOk := oldHash != "" && auth.VerifyPassword(currentPassword, oldHash)
			plainOk := oldHash == "" && oldPlain != "" && oldPlain == currentPassword
			if !hashOk && !plainOk {
				fmt.Print("Content-type: application/json; charset=utf-8\n\n")
				fmt.Println(`{"status":"error","message":"Неверный текущий пароль"}`)
				return
			}
		}
	} else {
		// Конфиг отсутствует или повреждён. Если панель уже настраивалась
		// (marker от install.sh) — блокируем установку нового пароля: это
		// защита от захвата панели после потери/повреждения конфига.
		if _, err := os.Stat(authMarkerPath); err == nil {
			fmt.Print("Content-type: application/json; charset=utf-8\n\n")
			fmt.Println(`{"status":"error","message":"Файл конфигурации авторизации повреждён или удалён. Восстановите его из бэкапа."}`)
			return
		}
	}

	var passwordHash string

	if enabled {
		if password != "" {
			if len(password) < 8 {
				fmt.Print("Content-type: application/json; charset=utf-8\n\n")
				fmt.Println(`{"status":"error","message":"Пароль должен быть минимум 8 символов"}`)
				return
			}
			passwordHash = auth.HashPassword(password)
			if passwordHash == "" {
				fmt.Print("Content-type: application/json; charset=utf-8\n\n")
				fmt.Println(`{"status":"error","message":"Не удалось сохранить настройки"}`)
				return
			}
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
	if err := cgiutil.WriteFileAtomic(authConfigPath, data, 0600); err != nil {
		fmt.Print("Content-type: application/json; charset=utf-8\n\n")
		fmt.Println(`{"status":"error","message":"Не удалось сохранить настройки"}`)
		return
	}

	// Смена/отключение пароля инвалидирует все существующие сессии:
	// файл /opt/var/run/panel_session удаляется, старые cookie умирают
	// (в обоих режимах гейт проверяет именно этот файл).
	auth.DestroySession()

	// Marker «панель защищалась» создаётся при первой установке пароля.
	// Используется для fail-closed: если auth_config.json позже будет
	// повреждён/удалён — установить новый пароль через UI станет нельзя
	// (защита от захвата панели), потребуется восстановление конфига.
	if enabled && password != "" {
		os.WriteFile(authMarkerPath, []byte("1"), 0600)
	}

	fmt.Print("Content-type: application/json; charset=utf-8\n\n")
	fmt.Println(`{"status":"ok","message":"Настройки сохранены"}`)
}
