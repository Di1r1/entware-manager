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
THRESHOLD_STATE="$STATE_DIR/thresholds.state"
API="https://api.telegram.org/bot"
CURL_TIMEOUT=8
CURL=/opt/bin/curl
[ -x "$CURL" ] || CURL=curl

# --- Загрузка конфига (поля бота + критические пороги) ---
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
                8) T_CPU_TEMP_EN=$_v ;;
                9) T_CPU_TEMP_VAL=$_v ;;
                10) T_WIFI0_EN=$_v ;;
                11) T_WIFI0_VAL=$_v ;;
                12) T_WIFI1_EN=$_v ;;
                13) T_WIFI1_VAL=$_v ;;
                14) T_CPU_LOAD_EN=$_v ;;
                15) T_CPU_LOAD_VAL=$_v ;;
                16) T_RAM_EN=$_v ;;
                17) T_RAM_VAL=$_v ;;
                18) T_DISK_EN=$_v ;;
                19) T_DISK_VAL=$_v ;;
            esac
        done << EOF
$(jq -r '.enabled // false, (.bot_token // ""), (.chat_id // ""), (.level // "ERROR"), (if (.sources|type)=="array" then (.sources|join("|")) else "system|monitor" end), (.autostart // false), (.proxy_url // ""),
  (.thresholds.cpu_temp.enabled // false), (.thresholds.cpu_temp.value // 90),
  (.thresholds.wifi0_temp.enabled // false), (.thresholds.wifi0_temp.value // 100),
  (.thresholds.wifi1_temp.enabled // false), (.thresholds.wifi1_temp.value // 100),
  (.thresholds.cpu_load.enabled // false), (.thresholds.cpu_load.value // 95),
  (.thresholds.ram_used.enabled // false), (.thresholds.ram_used.value // 90),
  (.thresholds.disk_temp.enabled // false), (.thresholds.disk_temp.value // 60)' "$CONFIG_FILE" 2>/dev/null)
EOF
    else
        ENABLED=false
        BOT_TOKEN=""
        CHAT_ID=""
        LEVEL="ERROR"
        SOURCES="system|monitor"
        AUTOSTART=false
        PROXY_URL=""
        T_CPU_TEMP_EN=false; T_CPU_TEMP_VAL=90
        T_WIFI0_EN=false;    T_WIFI0_VAL=100
        T_WIFI1_EN=false;    T_WIFI1_VAL=100
        T_CPU_LOAD_EN=false; T_CPU_LOAD_VAL=95
        T_RAM_EN=false;      T_RAM_VAL=90
        T_DISK_EN=false;     T_DISK_VAL=60
    fi
    [ -z "$LEVEL" ] && LEVEL="ERROR"
    [ -z "$SOURCES" ] && SOURCES="system|monitor"
    [ -z "$T_CPU_TEMP_VAL" ] && T_CPU_TEMP_VAL=90
    [ -z "$T_WIFI0_VAL" ] && T_WIFI0_VAL=100
    [ -z "$T_WIFI1_VAL" ] && T_WIFI1_VAL=100
    [ -z "$T_CPU_LOAD_VAL" ] && T_CPU_LOAD_VAL=95
    [ -z "$T_RAM_VAL" ] && T_RAM_VAL=90
    [ -z "$T_DISK_VAL" ] && T_DISK_VAL=60
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

# --- Эмодзи уровня ---
level_emoji() {
    case "$1" in
        ERROR) echo "🔴" ;;
        WARN)  echo "🟠" ;;
        INFO)  echo "🔵" ;;
        *)     echo "⚪" ;;
    esac
}

# --- Эмодзи источника ---
source_emoji() {
    case "$1" in
        network) echo "🌐" ;;
        service) echo "⚙️" ;;
        monitor) echo "📊" ;;
        smart)   echo "💾" ;;
        *)       echo "🖥️" ;;
    esac
}

# --- Эскейп HTML-спецсимволов для parse_mode=HTML (безопасно для <,>,&) ---
html_escape_tg() {
    local s="$1"
    s=$(printf '%s' "$s" | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g')
    printf '%s' "$s"
}

# --- Форматирование строки лога в эстетичное Telegram-сообщение ---
format_message() {
    local line="$1"
    local lvl src
    lvl=$(detect_level "$line")
    src=$(detect_source "$line")
    # Убираем [тег] [уровень] [ip] из начала, оставляем суть.
    local body
    body=$(printf '%s' "$line" | sed -E 's/^\[[^]]*\] \[(ERROR|WARN|INFO|FATAL)\] \[[^]]*\]( \[[0-9]+\])? //')
    [ -z "$body" ] && body=$(printf '%s' "$line" | sed -E 's/^\[[^]]*\] \[(ERROR|WARN|INFO|FATAL)\] //')
    # Чистим повторные [тег источника] в начале тела.
    body=$(printf '%s' "$body" | sed -E 's/^\[(network|service|monitor|smart)\] //')
    printf '%s %s <b>%s</b>\n%s' "$(level_emoji "$lvl")" "$(source_emoji "$src")" "$(html_escape_tg "$src")" "$(html_escape_tg "$body")"
}

# --- Отправка в Telegram (parse_mode=HTML). Возвращает 0 при успехе. ---
send_tg() {
    local text="$1"
    [ -z "$text" ] && return 1
    [ -n "$BOT_TOKEN" ] || return 1
    [ -n "$CHAT_ID" ] || return 1
    local msg
    msg=$(format_message "$text")
    # Прокси (http/socks5) — если провайдер блокирует Telegram напрямую.
    local proxy_args=""
    [ -n "$PROXY_URL" ] && proxy_args="-x $PROXY_URL"
    # shellcheck disable=SC2086
    "$CURL" -s -f -o /dev/null -m "$CURL_TIMEOUT" $proxy_args \
        --data-urlencode "chat_id=$CHAT_ID" \
        --data-urlencode "text=$msg" \
        --data-urlencode "parse_mode=HTML" \
        --data-urlencode "disable_web_page_preview=true" \
        "https://api.telegram.org/bot${BOT_TOKEN}/sendMessage" 2>/dev/null
}

# --- Отправка готового HTML-сообщения как есть (без format_message) ---
# Используется для событий порогов, где текст уже отформатирован с эмодзи.
send_raw() {
    local msg="$1"
    [ -z "$msg" ] && return 1
    [ -n "$BOT_TOKEN" ] || return 1
    [ -n "$CHAT_ID" ] || return 1
    local proxy_args=""
    [ -n "$PROXY_URL" ] && proxy_args="-x $PROXY_URL"
    # shellcheck disable=SC2086
    "$CURL" -s -f -o /dev/null -m "$CURL_TIMEOUT" $proxy_args \
        --data-urlencode "chat_id=$CHAT_ID" \
        --data-urlencode "text=$msg" \
        --data-urlencode "parse_mode=HTML" \
        --data-urlencode "disable_web_page_preview=true" \
        "https://api.telegram.org/bot${BOT_TOKEN}/sendMessage" 2>/dev/null
}

# --- Извлечение источника из строки лога "[tag]" или "[network] ..." ---
detect_source() {
    local line="$1"
    case "$line" in
        *"[service_action]"*) echo "service" ;;
        *"[service]"*) echo "service" ;;
        *"[monitor_action]"*|*"[ACTION]"*) echo "monitor" ;;
        *"[monitor]"*|*"[watchdog]"*) echo "monitor" ;;
        *"[smart]"*) echo "monitor" ;;
        *"[network]"*) echo "network" ;;
        *"[login.cgi]"*|*"[links_save.cgi]"*|*"[delete_file.cgi]"*|*"[crontab_update.cgi]"*|*"[logger"*) echo "system" ;;
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
    # Не обрабатываем собственные записи демона — иначе бесконечный цикл
    # (демон пишет в суточный лог, который сам же читает, и шлёт себе же).
    case "$line" in
        *"[telegram]"*) return 1 ;;
    esac
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

# ================= КРИТИЧЕСКИЕ ПОРОГИ =================

# --- Температура CPU из /sys/class/thermal (в °C). Пусто при отсутствии. ---
read_cpu_temp() {
    local z f t
    for f in /sys/class/thermal/thermal_zone*/temp; do
        [ -f "$f" ] || continue
        t=$(cat "$f" 2>/dev/null | tr -d ' ')
        [ -n "$t" ] || continue
        # Убираем миллисекунды (если len>3) — как в Go temperature.go.
        case "${#t}" in
            4|5|6) z=$((t / 1000)) ;;
            *) z=$t ;;
        esac
        echo "$z"
        return 0
    done
    return 1
}

# --- Занятость RAM в % из /proc/meminfo. Пусто при недоступности. ---
read_ram_used() {
    local total avail
    total=$(grep '^MemTotal:' /proc/meminfo 2>/dev/null | awk '{print $2}')
    avail=$(grep '^MemAvailable:' /proc/meminfo 2>/dev/null | awk '{print $2}')
    [ -z "$total" ] || [ "$total" -eq 0 ] 2>/dev/null && return 1
    if [ -z "$avail" ]; then
        # Старые ядра без MemAvailable: MemFree+Buffers+Cached.
        avail=$(grep -E '^MemFree:|^Buffers:|^Cached:' /proc/meminfo 2>/dev/null | awk '{s+=$2} END{print s}')
    fi
    [ -z "$avail" ] && return 1
    echo $(( (total - avail) * 100 / total ))
}

# --- Нагрузка CPU в % (двухточечный замер /proc/stat, как watchdog.sh). ---
read_cpu_load() {
    local t1 t2 idle1 idle2
    t1=$(awk '/^cpu / {s=0; for(i=2;i<=9;i++) s+=$i; print s}' /proc/stat 2>/dev/null)
    idle1=$(awk '/^cpu / {print $5}' /proc/stat 2>/dev/null)
    [ -z "$t1" ] && return 1
    sleep 1
    t2=$(awk '/^cpu / {s=0; for(i=2;i<=9;i++) s+=$i; print s}' /proc/stat 2>/dev/null)
    idle2=$(awk '/^cpu / {print $5}' /proc/stat 2>/dev/null)
    [ -z "$t2" ] && return 1
    local dt=$((t2 - t1)) didle=$((idle2 - idle1))
    [ "$dt" -le 0 ] && return 1
    echo $(( (dt - didle) * 100 / dt ))
}

# --- Температура WiFi0/WiFi1 из RCI Keenetic (curl). Пусто при недоступности. ---
read_wifi_temp() {
    local iface="$1"
    "$CURL" -s -m 3 "http://127.0.0.1:79/rci/show/interface/$iface" 2>/dev/null | \
        grep -o '"temperature":[0-9]*' | head -1 | sed 's/[^0-9]//g'
}

# --- Температура дисков (макс по всем sd*) через кеш Go или smartctl. ---
read_disk_temp() {
    local dev t maxd=0
    # Сначала свежий кеш Go-модуля (/tmp/entware/cache/disk/*) — дёшево и без
    # пробуждения спящих дисков. Если кеша нет — ищем по /proc/partitions.
    for c in /tmp/entware/cache/disk/sd[a-z]; do
        [ -f "$c" ] || continue
        t=$(grep 'Temperature_Celsius' "$c" 2>/dev/null | awk '{print $10}')
        [ -z "$t" ] && continue
        [ "$t" -gt "$maxd" ] 2>/dev/null && maxd=$t
    done
    [ "$maxd" -gt 0 ] 2>/dev/null && { echo "$maxd"; return 0; }
    # Если кеша нет — читаем smartctl напрямую (с защитой busy).
    for dev in /dev/sd[a-z]; do
        [ -e "$dev" ] || continue
        # Пропускаем, если на диске уже висит smartctl (D-состояние).
        find_pids "smartctl.*$dev" >/dev/null 2>&1 && continue
        t=$(smartctl -A -d auto "$dev" 2>/dev/null | grep 'Temperature_Celsius' | awk '{print $10}')
        [ -z "$t" ] && continue
        [ "$t" -gt "$maxd" ] 2>/dev/null && maxd=$t
    done
    [ "$maxd" -gt 0 ] 2>/dev/null && { echo "$maxd"; return 0; }
    return 1
}

# --- Чтение состояния порога из state-файла. ---
read_state() {
    local key="$1"
    [ -f "$THRESHOLD_STATE" ] || { echo normal; return; }
    grep "^$key=" "$THRESHOLD_STATE" 2>/dev/null | sed "s/^$key=//" | head -1
}

# --- Запись состояния порога (temp+mv, атомарно, RULES 10). ---
write_state() {
    local key="$1" val="$2" tmp
    mkdir -p "$STATE_DIR" 2>/dev/null
    tmp="$THRESHOLD_STATE.tmp"
    grep -v "^$key=" "$THRESHOLD_STATE" 2>/dev/null > "$tmp"
    echo "$key=$val" >> "$tmp"
    mv -f "$tmp" "$THRESHOLD_STATE"
}

# --- Проверка одного порога с анти-спамом (переход alarm/normal). ---
check_one_threshold() {
    local metric="$1" en="$2" val="$3" cur="$4" name="$5" unit="$6"
    [ "$en" = "true" ] || return 0
    [ -z "$cur" ] && return 0   # недоступная метрика — не менять состояние, молчать
    local state
    state=$(read_state "$metric")
    if [ "$cur" -gt "$val" ] 2>/dev/null; then
        if [ "$state" != "alarm" ]; then
            send_raw "$(printf '🔴 📊 <b>%s</b>\nПревышен порог: %s%s &gt; %s%s' "$(html_escape_tg "$name")" "$cur" "$unit" "$val" "$unit")"
            log_message "WARN" "[telegram] threshold $metric alarm (${cur}${unit} > ${val}${unit})"
            write_state "$metric" "alarm"
        fi
    else
        if [ "$state" = "alarm" ]; then
            send_raw "$(printf '✅ 📊 <b>%s</b>\nВернулась в норму: %s%s &lt; %s%s' "$(html_escape_tg "$name")" "$cur" "$unit" "$val" "$unit")"
            log_message "INFO" "[telegram] threshold $metric normal (${cur}${unit} < ${val}${unit})"
            write_state "$metric" "normal"
        fi
    fi
}

# --- Проверка всех порогов (дешёвые → дорогие). ---
check_thresholds() {
    local cpu_t wifi0 wifi1 cpu_l ram disk
    cpu_t=$(read_cpu_temp)
    check_one_threshold "cpu_temp" "$T_CPU_TEMP_EN" "$T_CPU_TEMP_VAL" "$cpu_t" "CPU температура" "°C"
    ram=$(read_ram_used)
    check_one_threshold "ram_used" "$T_RAM_EN" "$T_RAM_VAL" "$ram" "Память" "%"
    cpu_l=$(read_cpu_load)
    check_one_threshold "cpu_load" "$T_CPU_LOAD_EN" "$T_CPU_LOAD_VAL" "$cpu_l" "Нагрузка CPU" "%"
    wifi0=$(read_wifi_temp "WifiMaster0")
    check_one_threshold "wifi0_temp" "$T_WIFI0_EN" "$T_WIFI0_VAL" "$wifi0" "WiFi0 температура" "°C"
    wifi1=$(read_wifi_temp "WifiMaster1")
    check_one_threshold "wifi1_temp" "$T_WIFI1_EN" "$T_WIFI1_VAL" "$wifi1" "WiFi1 температура" "°C"
    disk=$(read_disk_temp)
    check_one_threshold "disk_temp" "$T_DISK_EN" "$T_DISK_VAL" "$disk" "Температура дисков" "°C"
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
        # Критические пороги — раз в ~6 итераций (~60с), чтобы не блокировать
        # обработку логов на smartctl/RCI (дешёвые метрики идут первыми).
        if [ $((it % 6)) -eq 0 ]; then
            check_thresholds
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
