#!/bin/sh
# ==============================================
# Получение/сохранение конфигурации логирования
# Версия: 1.7 (добавлены заголовки Content-Type для GET)
# Дата: 2026-04-01
# ==============================================

. /opt/web_entware/lib/common.sh

CONFIG_FILE="/opt/web_entware/logger/config.json"
SYSTEM_LOG="/opt/var/log/entware/system.log"

log_system() {
    level="$1"
    message="$2"
    timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    ip="${REMOTE_ADDR:-localhost}"
    mkdir -p "$(dirname "$SYSTEM_LOG")" 2>/dev/null
    echo "[$timestamp] [$level] [$ip] $message" >> "$SYSTEM_LOG" 2>/dev/null
}

show_pretty_config() {
    echo "Content-type: text/html"
    echo ""
    
    cat <<'HEAD'
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Настройки логирования</title>
    <link rel="stylesheet" href="/entware-manager/style.css">
    <style>
        body { padding: 20px; background: var(--bg-primary, #1a1a2e); color: var(--text-primary, #e0e0e0); }
        .config-card { background: var(--card-bg, #16213e); border-radius: 12px; padding: 24px; max-width: 500px; margin: 0 auto; }
        .config-item { display: flex; justify-content: space-between; padding: 12px 0; border-bottom: 1px solid var(--border-color, #333); }
        .config-item:last-child { border-bottom: none; }
        .config-label { font-weight: 500; }
        .config-value { font-family: monospace; color: var(--accent-color, #8b5cf6); }
        .status-on { color: #2ecc71; }
        .status-off { color: #e74c3c; }
        .btn-back { display: inline-block; margin-top: 20px; padding: 10px 20px; background: var(--accent-color, #8b5cf6); color: #fff; border-radius: 8px; text-decoration: none; }
    </style>
</head>
<body>
<div class="config-card">
HEAD

    if [ -f "$CONFIG_FILE" ]; then
        if /opt/bin/jq --version >/dev/null 2>&1; then
            ENABLED=$(/opt/bin/jq -r '.enabled' "$CONFIG_FILE" 2>/dev/null)
            
            echo "<h2>Настройки логирования</h2>"
            echo "<div class='config-item'>"
            echo "  <span class='config-label'>Состояние</span>"
            if [ "$ENABLED" = "true" ]; then
                echo "  <span class='config-value status-on'>● Включено</span>"
            else
                echo "  <span class='config-value status-off'>● Отключено</span>"
            fi
            echo "</div>"
            
            echo "<div class='config-item'>"
            echo "  <span class='config-label'>Файлы логов</span>"
            echo "  <span class='config-value'>/tmp/entware/logs/</span>"
            echo "</div>"
            
            echo "<div class='config-item'>"
            echo "  <span class='config-label'>Архив логов</span>"
            echo "  <span class='config-value'>/opt/var/log/entware/</span>"
            echo "</div>"
            
            echo "<div class='config-item'>"
            echo "  <span class='config-label'>Системные события</span>"
            echo "  <span class='config-value'>/opt/var/log/entware/system.log</span>"
            echo "</div>"
            
            echo "<div class='config-item'>"
            echo "  <span class='config-label'>Конфиг</span>"
            echo "  <span class='config-value'>/opt/web_entware/logger/config.json</span>"
            echo "</div>"
        else
            echo "<h2>Ошибка</h2><p>JSON parser (jq) не найден</p>"
        fi
    else
        echo "<h2>Настройки логирования</h2>"
        echo "<div class='config-item'>"
        echo "  <span class='config-label'>Состояние</span>"
        echo "  <span class='config-value status-on'>● По умолчанию (включено)</span>"
        echo "</div>"
    fi
    
    cat <<'TAIL'
    <a href="/entware-cgi/logger/view.cgi" class="btn-back">← К логам</a>
</div>
</body>
</html>
TAIL
}

if [ "$REQUEST_METHOD" = "GET" ]; then
    echo "Content-type: application/json; charset=utf-8"
    echo ""
    if [ -n "$(echo "$QUERY_STRING" | grep -o 'pretty')" ]; then
        show_pretty_config
    else
        if [ -f "$CONFIG_FILE" ]; then
            cat "$CONFIG_FILE"
        else
            echo '{"enabled":true}'
        fi
    fi
    exit 0
fi

if [ "$REQUEST_METHOD" = "POST" ]; then
    POST_DATA=$(cat)
    
    if [ -z "$POST_DATA" ]; then
        json_out '{"status":"error","message":"Empty request"}'
        exit 0
    fi
    
    if ! echo "$POST_DATA" | /opt/bin/jq empty 2>/dev/null; then
        json_out '{"status":"error","message":"Invalid JSON"}'
        exit 0
    fi
    
    NEW_ENABLED=$(echo "$POST_DATA" | /opt/bin/jq -r '.enabled' 2>/dev/null)
    OLD_ENABLED=$(/opt/bin/jq -r '.enabled' "$CONFIG_FILE" 2>/dev/null || echo "unknown")
    
    if [ "$NEW_ENABLED" != "$OLD_ENABLED" ]; then
        if [ "$NEW_ENABLED" = "true" ]; then
            log_system "INFO" "Логирование ВКЛЮЧЕНО (было: enabled=$OLD_ENABLED)"
        else
            log_system "INFO" "Логирование ОТКЛЮЧЕНО (было: enabled=$OLD_ENABLED)"
        fi
    fi
    
    echo "$POST_DATA" > "$CONFIG_FILE"
    json_out '{"status":"ok"}'
    exit 0
fi

json_out '{"error":"Method not allowed"}'
