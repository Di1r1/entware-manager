#!/bin/sh
# shellcheck disable=SC3043,SC3037,SC3057,SC1090,SC1091,SC2155,SC2046,SC2086,SC2034,SC2001
# ==============================================
# Entware Manager - шлюз Telegram-уведомлений
# Версия: 1.0 (независимый демон, читает существующие логи)
# Дата: 2026-08-19
#
# Читает источники событий (system.log + /tmp/entware/logs/*.log) по offset,
# фильтрует по уровню/источнику и шлёт в Telegram через Bot API.
# Независим от панели: отказ/недоступность Telegram не влияет на остальное.
# ==============================================

export PATH=/opt/sbin:/opt/bin:/sbin:/bin:/usr/sbin:/usr/bin
export HOME=/tmp

. /opt/web_entware/lib/common.sh

CONFIG_FILE="/opt/web_entware/telegram_config.json"
PID_FILE="/tmp/entware/pid/telegram_gateway.pid"
LOG_FILE="/tmp/entware/logs/telegram.log"
STATE_DIR="/tmp/entware/telegram"
OFFSET_FILE="$STATE_DIR/offsets"
API="https://api.telegram.org/bot"
CURL_TIMEOUT=8
CURL=/opt/bin/curl
[ -x "$CURL" ] || CURL=curl

# --- Загрузка конфига ---
load_config() {
    if [ -f "$CONFIG_FILE" ] && command -v jq >/dev/null 2>&1; then
        i=0
        while IFS= read -r _v; do
            i=$((i + 1))
            case "$i" in
                1) ENABLED=$_v ;;
                2) BOT_TOKEN=$_v ;;
                3) CHAT_ID=$_v ;;
                4) LEVEL=$_v ;;
                5) SOURCES=$_v ;;
                6) AUTOSTART=$_v ;;
                7) PROXY_URL=$_v ;;
            esac
        done << EOF
$(jq -r '.enabled // false, (.bot_token // ""), (.chat_id // ""), (.level // "ERROR"), (if (.sources|type)=="array" then (.sources|join("|")) else "system|monitor" end), (.autostart // false), (.proxy_url // "")' "$CONFIG_FILE" 2>/dev/null)
EOF
    else
        ENABLED=false
        BOT_TOKEN=""
        CHAT_ID=""
        LEVEL="ERROR"
        SOURCES="system|monitor"
        AUTOSTART=false
        PROXY_URL=""
    fi
    [ -z "$LEVEL" ] && LEVEL="ERROR"
    [ -z "$SOURCES" ] && SOURCES="system|monitor"
}

# --- Числовой ранг уровня (ERROR=3, WARN=2, INFO=1, OFF=0) ---
level_rank() {
    case "$1" in
        ERROR) echo 3 ;;
        WARN)  echo 2 ;;
        INFO)  echo 1 ;;
        *)     echo 0 ;;
    esac
}

# --- Отправка в Telegram. Возвращает 0 при успехе. ---
send_tg() {
    local text="$1"
    [ -z "$text" ] && return 1
    [ -n "$BOT_TOKEN" ] || return 1
    [ -n "$CHAT_ID" ] || return 1
    # Текст передаём через --data-urlencode (безопасно для спецсимволов).
    # -f: возвращает ненулевой код при HTTP >= 400 (неверный токен/chat_id) —
    # иначе curl вернул бы 0 и ошибка была бы залогирована как «sent».
    # Прокси (http/socks5) — если провайдер блокирует Telegram напрямую.
    local proxy_args=""
    [ -n "$PROXY_URL" ] && proxy_args="-x $PROXY_URL"
    # shellcheck disable=SC2086
    "$CURL" -s -f -o /dev/null -m "$CURL_TIMEOUT" $proxy_args \
        --data-urlencode "chat_id=$CHAT_ID" \
        --data-urlencode "text=$text" \
        --data-urlencode "disable_web_page_preview=true" \
        "https://api.telegram.org/bot${BOT_TOKEN}/sendMessage" 2>/dev/null
}

# --- Извлечение источника из строки лога "[tag]" или "[network] ..." ---
detect_source() {
    local line="$1"
    case "$line" in
        *"[network]"*) echo "network" ;;
        *"[service]"*) echo "service" ;;
        *"[monitor]"*|*"[watchdog]"*) echo "monitor" ;;
        *"[smart]"*) echo "monitor" ;;
        *) echo "system" ;;
    esac
}

# --- Уровень из строки лога "[INFO]"/"[WARN]"/"[ERROR]" ---
detect_level() {
    case "$1" in
        *"[ERROR]"*|*"[FATAL]"*) echo "ERROR" ;;
        *"[WARN]"*) echo "WARN" ;;
        *) echo "INFO" ;;
    esac
}

# --- Проверка, проходит ли строка фильтр (уровень + источник) ---
passes_filter() {
    local line="$1"
    local lvl src
    lvl=$(detect_level "$line")
    src=$(detect_source "$line")
    [ "$LEVEL" = "OFF" ] && return 1
    [ "$(level_rank "$lvl")" -lt "$(level_rank "$LEVEL")" ] && return 1
    case "|$SOURCES|" in
        *"|$src|"*) return 0 ;;
    esac
    return 1
}

# --- Чтение offset файла для источника. ---
read_offset() {
    local key="$1" val="0"
    [ -f "$OFFSET_FILE" ] || { echo 0; return; }
    val=$(grep "^$key=" "$OFFSET_FILE" 2>/dev/null | sed "s/^$key=//")
    [ -n "$val" ] || val=0
    echo "$val"
}

# --- Сохранение offset (перезапись файла, без дублей). ---
save_offset() {
    local key="$1" val="$2" tmp
    mkdir -p "$STATE_DIR" 2>/dev/null
    tmp="$OFFSET_FILE.tmp"
    # Убираем старую запись key=..., добавляем новую.
    grep -v "^$key=" "$OFFSET_FILE" 2>/dev/null > "$tmp"
    echo "$key=$val" >> "$tmp"
    mv -f "$tmp" "$OFFSET_FILE"
}

# --- Чтение новых строк файла от offset. Возвращает новые строки. ---
read_new_lines() {
    local file="$1" key="$2"
    local off
    off=$(read_offset "$key")
    [ -f "$file" ] || return 0
    # Печатаем строки, начиная с (off+1)-й (1-индексация).
    tail -n +$((off + 1)) "$file" 2>/dev/null
}

# --- Обновление offset до текущего числа строк файла. ---
update_offset() {
    local file="$1" key="$2" count
    [ -f "$file" ] || { save_offset "$key" 0; return; }
    count=$(wc -l < "$file" 2>/dev/null)
    [ -z "$count" ] && count=0
    save_offset "$key" "$count"
}

# --- Обработка одного файла-источника. ---
process_file() {
    local file="$1" key="$2"
    [ -f "$file" ] || { save_offset "$key" 0; return; }
    local lines
    lines=$(read_new_lines "$file" "$key")
    [ -z "$lines" ] && { update_offset "$file" "$key"; return; }
    echo "$lines" | while IFS= read -r l; do
        [ -z "$l" ] && continue
        if passes_filter "$l"; then
            if send_tg "$l"; then
                log_message "INFO" "[telegram] sent: $l"
            else
                log_message "WARN" "[telegram] failed to send: $l"
            fi
        fi
    done
    update_offset "$file" "$key"
}

# --- Обработка всех источников. ---
process_all() {
    local syslog="/opt/var/log/entware/system.log"
    local daylog="/tmp/entware/logs/$(date '+%Y-%m-%d').log"
    process_file "$syslog" "system"
    process_file "$daylog" "monitor"
}

daemon_loop() {
    load_config
    mkdir -p "$STATE_DIR" "$(dirname "$LOG_FILE")" 2>/dev/null
    # mtime конфига — для автоперечитывания изменений из панели.
    # date -r +%s — BusyBox-совместимо (stat -c %Y на роутере недоступен).
    CFG_MTIME=$(date -r "$CONFIG_FILE" +%s 2>/dev/null || echo 0)

    trap 'log_message "INFO" "[telegram] gateway stopped (pid=$$)"; rm -f "$PID_FILE"; exit 0' TERM
    trap 'load_config; log_message "INFO" "[telegram] config reloaded (level=$LEVEL, sources=$SOURCES)"' HUP

    log_message "INFO" "[telegram] gateway started (level=$LEVEL, sources=$SOURCES)"

    local it=0
    while true; do
        # Раз в 5 итераций (~50с) проверяем, не изменился ли конфиг.
        it=$((it + 1))
        if [ $((it % 5)) -eq 0 ]; then
            NEW_MTIME=$(date -r "$CONFIG_FILE" +%s 2>/dev/null || echo 0)
            if [ -n "$NEW_MTIME" ] && [ "$NEW_MTIME" != "$CFG_MTIME" ]; then
                CFG_MTIME=$NEW_MTIME
                load_config
                log_message "INFO" "[telegram] config reloaded (level=$LEVEL, sources=$SOURCES)"
            fi
        fi
        if [ "$ENABLED" = "true" ] && [ -n "$BOT_TOKEN" ] && [ -n "$CHAT_ID" ]; then
            process_all
        fi
        sleep 10
    done
}

case "$1" in
    start)
        daemon_start "telegram" "$PID_FILE" "$LOG_FILE" "(^|[/ ])telegram_gateway\.sh daemon"
        ;;
    stop)
        daemon_stop "$PID_FILE" "(^|[/ ])telegram_gateway\.sh daemon"
        log_message "INFO" "[telegram] gateway stopped"
        ;;
    restart)
        "$0" stop
        sleep 1
        "$0" start
        ;;
    status)
        daemon_status "$PID_FILE" "(^|[/ ])telegram_gateway\.sh daemon"
        ;;
    daemon)
        daemon_loop
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|status|daemon}"
        exit 1
        ;;
esac
