#!/bin/sh
# ==============================================
# Entware Manager - управление защитой файлового менеджера
# GET: возвращает {enabled}
# POST: сохраняет {enabled, password}
# ==============================================

. /opt/web_entware/lib/common.sh

CONFIG="/opt/web_entware/auth_config.json"

if [ "$REQUEST_METHOD" = "POST" ]; then
    _POST_BODY=$(cat); export _POST_BODY
    enabled=$(post_param "enabled" "")
    password=$(post_param "password" "")

    if [ "$enabled" = "true" ]; then
        if [ -n "$password" ]; then
            if [ ${#password} -lt 4 ]; then
                json_out '{"status":"error","message":"Пароль должен быть минимум 4 символа"}'
            fi
            if command -v sha256sum >/dev/null 2>&1; then
                password_hash=$(echo -n "$password" | sha256sum 2>/dev/null | cut -d' ' -f1)
            elif command -v openssl >/dev/null 2>&1; then
                password_hash=$(echo -n "$password" | openssl dgst -sha256 2>/dev/null | cut -d' ' -f2)
            else
                json_out '{"status":"error","message":"Нет sha256sum или openssl для хэширования пароля"}'
            fi
        else
            # Сохраняем старый хэш, если файл существует
            if [ -f "$CONFIG" ]; then
                password_hash=$(jq -r '.password_hash // ""' "$CONFIG" 2>/dev/null)
                [ -z "$password_hash" ] && json_out '{"status":"error","message":"Введите новый пароль"}'
            else
                json_out '{"status":"error","message":"Введите пароль"}'
            fi
        fi
    else
        password_hash=""
    fi

    printf '{"enabled":%s,"password_hash":"%s"}\n' \
        "$([ "$enabled" = "true" ] && echo "true" || echo "false")" \
        "$password_hash" > "$CONFIG"
    json_out '{"status":"ok","message":"Настройки сохранены"}'
fi

# GET — возвращаем только enabled
echo "Content-type: application/json; charset=utf-8"
echo ""
if [ -f "$CONFIG" ] && jq -e '.' "$CONFIG" >/dev/null 2>&1; then
    jq '{enabled}' "$CONFIG" 2>/dev/null || echo '{"enabled":false}'
else
    echo '{"enabled":false}'
fi
