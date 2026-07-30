#!/bin/sh
# shellcheck disable=SC3043,SC3037,SC3057,SC1090,SC1091,SC2155,SC2034
# ==============================================
# Entware Manager - общие функции для CGI
# Версия: 2.6 (добавлены die, log_message, html_escape, human_size, load_config)
# Дата: 2026-07-16
# ==============================================

export PATH=/opt/sbin:/opt/bin:/sbin:/bin:/usr/sbin:/usr/bin
export HOME=/tmp

# --- Декодирование URL ---
url_decode() {
    printf '%s' "$1" | sed \
        -e 's/%21/_EXCL_/g' \
        -e 's/%23/_HASH_/g' \
        -e 's/%24/_DOLLAR_/g' \
        -e 's/%26/_AMP_/g' \
        -e 's/%27/_QUOTE_/g' \
        -e 's/%28/_LPAR_/g' \
        -e 's/%29/_RPAR_/g' \
        -e 's/%2C/_COMMA_/g' \
        -e 's/%2F/\//g' \
        -e 's/%3A/:/g' \
        -e 's/%3B/;/g' \
        -e 's/%3D/=/g' \
        -e 's/%3F/?/g' \
        -e 's/%40/@/g' \
        -e 's/%5B/[/g' \
        -e 's/%5C/\\/g' \
        -e 's/%5D/]/g' \
        -e 's/%7E/~/g' \
        -e 's/%60/_BACKQ_/g' \
        -e 's/%22/_DQUOT_/g' \
        -e 's/%3C/</g' \
        -e 's/%3E/>/g' \
        -e 's/%25/%/g' \
        -e 's/%20/ /g' \
        -e 's/+/ /g' \
        -e 's/_EXCL_/!/g' \
        -e 's/_HASH_/#/g' \
        -e 's/_DOLLAR_/$/g' \
        -e 's/_AMP_/\&/g' \
        -e "s/_QUOTE_/'/g" \
        -e 's/_LPAR_/(/g' \
        -e 's/_RPAR_/)/g' \
        -e 's/_COMMA_/,/g' \
        -e 's/_BACKQ_/`/g' \
        -e 's/_DQUOT_/"/g'
}

# --- Санитайз ---
sanitize_alnum() {
    str="$1"
    clean=""
    i=0
    while [ $i -lt ${#str} ]; do
        char="${str:$i:1}"
        case "$char" in
            [a-zA-Z0-9._-]) clean="${clean}${char}" ;;
        esac
        i=$((i+1))
    done
    echo "$clean"
}

# --- Извлечение GET-параметра ---
get_param() {
    key="$1"
    default="$2"
    value=$(echo "$QUERY_STRING" | sed -n "s/.*${key}=\([^&]*\).*/\1/p")
    if [ -z "$value" ]; then
        echo "$default"
    else
        url_decode "$value"
    fi
}

# --- Извлечение параметра из POST-данных ($_POST_BODY) ---
post_param() {
    key="$1"
    default="$2"
    value=$(echo "$_POST_BODY" | sed -n "s/.*${key}=\([^&]*\).*/\1/p" | tr -d '\r')
    if [ -z "$value" ]; then
        echo "$default"
    else
        url_decode "$value"
    fi
}

# --- Вывод JSON-ответа ---
json_out() {
    echo "Content-type: application/json; charset=utf-8"
    echo ""
    echo "$1"
    exit 0
}

# --- Вывод HTML-заголовков ---
html_header() {
    echo "Content-type: text/html; charset=utf-8"
    echo ""
}

# --- Проверка PID (жив и не зомби) ---
pid_is_alive() {
    [ -z "$1" ] && return 1
    [ ! -d "/proc/$1" ] && return 1
    local s
    s=$(grep "State:" "/proc/$1/status" 2>/dev/null | awk '{print $2}')
    [ "$s" = "Z" ] && return 1
    return 0
}

# --- Поиск PID по паттерну cmdline (pgrep или fallback на ps) ---
find_pids() {
    local pattern="$1"
    if command -v pgrep >/dev/null 2>&1; then
        pgrep -f "$pattern" 2>/dev/null
    else
        ps w 2>/dev/null | grep -v grep | grep -E "$pattern" | awk '{print $1}'
    fi
}

# --- Фатальная ошибка (для демонов с set -eu) ---
die() {
    echo "[FATAL] $1" >&2
    exit "${2:-1}"
}

# --- Единое логирование в /tmp/entware/logs/ ---
log_message() {
    local level="$1" msg="$2"
    local log_dir="/tmp/entware/logs"
    mkdir -p "$log_dir" 2>/dev/null
    local ts
    ts=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[$ts] [$level] [$$] $msg" >> "$log_dir/$(date '+%Y-%m-%d').log"
}

# --- Экранирование HTML (sed) ---
html_escape() {
    echo "$1" | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g; s/"/\&quot;/g'
}

# --- Байты в человекочитаемый размер (BusyBox-совместимо) ---
human_size() {
    local b="$1"
    if [ "$b" -lt 1024 ]; then
        echo "${b} B"
    elif [ "$b" -lt 1048576 ]; then
        echo "$((b / 1024)).$(((b % 1024) * 10 / 1024)) KiB"
    elif [ "$b" -lt 1073741824 ]; then
        echo "$((b / 1048576)).$(((b % 1048576) * 10 / 1048576)) MiB"
    else
        echo "$((b / 1073741824)).$(((b % 1073741824) * 10 / 1073741824)) GiB"
    fi
}

# --- Загрузка JSON-конфига (проверка существования + валидация) ---
load_config() {
    local cfg="$1"
    if [ -f "$cfg" ] && command -v jq >/dev/null 2>&1; then
        jq -e '.' "$cfg" >/dev/null 2>&1 || return 1
        return 0
    fi
    return 1
}

# --- Парсинг событий из дневного лога в JSON ---
parse_log_events() {
    local tag="$1" limit="${2:-20}"
    local log_file="/tmp/entware/logs/$(date '+%Y-%m-%d').log"

    if [ ! -f "$log_file" ]; then
        echo '{"events":[]}'
        return
    fi

    local events
    events=$(tail -n 1000 "$log_file" 2>/dev/null | grep -i "\[$tag\]" | tail -n "$limit" | tr '[:upper:]' '[:lower:]')

    if [ -z "$events" ]; then
        echo '{"events":[]}'
        return
    fi

    local first=1 result=""
    while IFS= read -r line; do
        local ts lvl rest
        ts=$(echo "$line" | cut -c2-20)
        lvl=$(echo "$line" | sed -n 's/.*\[\(info\|warn\|error\)\].*/\1/p' | tr '[:lower:]' '[:upper:]')
        [ -z "$lvl" ] && lvl="INFO"

        rest=$(echo "$line" | sed "s/.*\[$tag\] //")

        local svc evt dtl
        svc=$(echo "$rest" | awk '{print $1}' | tr -d ':')
        [ -z "$svc" ] && svc="unknown"

        local rest_after_svc
        rest_after_svc=$(echo "$rest" | sed "s/^$svc //")

        evt=$(echo "$rest_after_svc" | awk '{print $1}')
        [ -z "$evt" ] && evt="unknown"

        dtl=$(echo "$rest_after_svc" | sed "s/^$evt //" | tr -d '()')
        [ -z "$dtl" ] && dtl="-"

        # JSON-экранирование
        svc=$(echo "$svc" | sed 's/\\/\\\\/g; s/"/\\"/g')
        evt=$(echo "$evt" | sed 's/\\/\\\\/g; s/"/\\"/g')
        dtl=$(echo "$dtl" | sed 's/\\/\\\\/g; s/"/\\"/g')

        [ "$first" -eq 1 ] && first=0 || result="${result},"
        result="${result}{\"timestamp\":\"${ts}\",\"level\":\"${lvl}\",\"service\":\"${svc}\",\"event\":\"${evt}\",\"details\":\"${dtl}\"}"
    done <<EOF
$events
EOF

    echo "{\"events\":[${result}]}"
}

# --- CGI-логирование (обёртка над log_message с IP/script_name) ---
log_action() {
    level="$1"
    message="$2"
    LOGGING_SH="/opt/web_entware/logger/lib/logging.sh"
    
    if [ -f "$LOGGING_SH" ]; then
        . "$LOGGING_SH" 2>/dev/null
        if [ "$LOG_ENABLED" = "true" ]; then
            log_action_original "$level" "$message" 2>/dev/null
        fi
        return 0
    fi
    # Fallback — через log_message
    ip="${REMOTE_ADDR:-localhost}"
    script_base=$(basename "${SCRIPT_NAME:-unknown}" .cgi)
    log_message "$level" "[$ip] [$script_base] $message"
}

# --- Проверка, запущен ли демон ---
# daemon_is_running pidfile pattern
# Возвращает 0 + PID если запущен, 1 если нет
daemon_is_running() {
    local pidfile="$1" pattern="$2" pid=""
    if [ -f "$pidfile" ]; then
        pid=$(cat "$pidfile" 2>/dev/null)
        if [ -n "$pid" ] && pid_is_alive "$pid"; then
            echo "$pid"
            return 0
        fi
    fi
    pid=$(find_pids "$pattern" | head -1)
    if [ -n "$pid" ] && pid_is_alive "$pid"; then
        echo "$pid"
        return 0
    fi
    return 1
}

# --- Запуск демона (форкает $0 daemon) ---
# daemon_start name pidfile logfile pattern
daemon_start() {
    local name="$1" pidfile="$2" logfile="$3" pattern="$4"
    local pid
    pid=$(daemon_is_running "$pidfile" "$pattern") && {
        echo "Already running with PID $pid"
        exit 1
    }
    mkdir -p "$(dirname "$pidfile")" "$(dirname "$logfile")" 2>/dev/null
    "$0" daemon >> "$logfile" 2>&1 &
    echo $! > "$pidfile"
    log_message "INFO" "[$name] Демон запущен (PID: $(cat "$pidfile"))"
    echo "Started with PID $(cat "$pidfile")"
}

# --- Остановка демона ---
# daemon_stop pidfile pattern [extra_cleanup]
daemon_stop() {
    local pidfile="$1" pattern="$2" extra_cleanup="$3"
    for p in $(find_pids "$pattern" 2>/dev/null); do
        kill -9 "$p" 2>/dev/null
    done
    rm -f "$pidfile"
    [ -n "$extra_cleanup" ] && eval "$extra_cleanup"
}

# --- Статус демона ---
# daemon_status pidfile pattern
daemon_status() {
    local pidfile="$1" pattern="$2"
    local pid
    pid=$(daemon_is_running "$pidfile" "$pattern") && {
        echo "Running with PID $pid"
        exit 0
    }
    echo "Not running"
    exit 1
}

# --- Проверка пароля файлового менеджера ---
# check_filemgr_auth password_field
# Возвращает 0 если пароль верный или защита выключена
check_filemgr_auth() {
    local pass_field="${1:-password}"
    local cfg="/opt/web_entware/auth_config.json"
    [ -f "$cfg" ] || return 0
    local enabled; enabled=$(jq -r '.enabled // false' "$cfg" 2>/dev/null)
    [ "$enabled" = "true" ] || return 0
    local stored_hash; stored_hash=$(jq -r '.password_hash // ""' "$cfg" 2>/dev/null)
    [ -n "$stored_hash" ] || return 0
    local user_pass; user_pass=$(post_param "$pass_field" "")
    local user_hash
    if command -v sha256sum >/dev/null 2>&1; then
        user_hash=$(echo -n "$user_pass" | sha256sum 2>/dev/null | cut -d' ' -f1)
    elif command -v openssl >/dev/null 2>&1; then
        user_hash=$(echo -n "$user_pass" | openssl dgst -sha256 2>/dev/null | cut -d' ' -f2)
    else
        # Fallback — нет sha256sum, проверка невозможна → запрещаем
        return 1
    fi
    [ "$user_hash" = "$stored_hash" ] && return 0
    return 1
}

get_version() {
    version_file="/opt/web_entware/version.json"
    if [ -f "$version_file" ] && /opt/bin/jq --version >/dev/null 2>&1; then
        /opt/bin/jq -r '.version' "$version_file" 2>/dev/null
    else
        echo "unknown"
    fi
}

# --- Проверка системных зависимостей ---
check_deps_logic() {
    # Базовые утилиты BusyBox
    sed_ok="false"; command -v sed >/dev/null 2>&1 && sed_ok="true"
    awk_ok="false"; command -v awk >/dev/null 2>&1 && awk_ok="true"
    grep_ok="false"; command -v grep >/dev/null 2>&1 && grep_ok="true"
    ps_ok="false"; command -v ps >/dev/null 2>&1 && ps_ok="true"

    # Opkg
    opkg_ok="false"
    if command -v /opt/bin/opkg >/dev/null 2>&1 && /opt/bin/opkg --version >/dev/null 2>&1; then
        opkg_ok="true"
    fi

    # Lighttpd
    lighttpd_installed="false"; /opt/bin/opkg list-installed 2>/dev/null | grep -q "^lighttpd " && lighttpd_installed="true"
    lighttpd_running="false"
    if [ -f /opt/var/run/lighttpd.pid ]; then
        pid=$(cat /opt/var/run/lighttpd.pid 2>/dev/null)
        if [ -n "$pid" ] && pid_is_alive "$pid"; then
            lighttpd_running="true"
        fi
    fi

    # Cron
    cron_installed="false"; /opt/bin/opkg list-installed 2>/dev/null | grep -q "^cron " && cron_installed="true"
    cron_running="false"
    if [ -f /opt/var/run/cron.pid ]; then
        pid=$(cat /opt/var/run/cron.pid 2>/dev/null)
        if [ -n "$pid" ] && pid_is_alive "$pid"; then
            cron_running="true"
        fi
    fi

    # JQ
    jq_ok="false"; command -v /opt/bin/jq >/dev/null 2>&1 && jq_ok="true"

    # IP (ip-full в Entware, может быть встроен в прошивку)
    ip_ok="false"
    ip_path=""
    if command -v ip >/dev/null 2>&1; then
        ip_ok="true"
        ip_path=$(command -v ip)
    fi

    # Пакет ip-full (для рекомендаций)
    ip_pkg_installed="false"
    /opt/bin/opkg list-installed 2>/dev/null | grep -q "^ip-full " && ip_pkg_installed="true"

    # Статус разделов
    # Packages: нужен только opkg
    pkg_status="ok"; [ "$opkg_ok" = "false" ] && pkg_status="missing"

    # Services: нужны cron и jq
    srv_status="ok"
    if [ "$cron_installed" = "false" ] || [ "$jq_ok" = "false" ]; then
        srv_status="missing"
    fi

    # Monitoring: нужны cron (для сбора данных) и jq
    mon_status="ok"
    if [ "$cron_installed" = "false" ] || [ "$jq_ok" = "false" ]; then
        mon_status="partial"
    fi

    # Network: нужен ip (утилита)
    net_status="ok"; [ "$ip_ok" = "false" ] && net_status="missing"

    # Logger: нужен jq
    log_status="ok"; [ "$jq_ok" = "false" ] && log_status="missing"

    # SMART: нужен smartctl
    smart_status="ok"
    if ! command -v /opt/sbin/smartctl >/dev/null 2>&1 && ! command -v smartctl >/dev/null 2>&1; then
        smart_status="missing"
    fi

    # Итоговый статус
    overall="ok"
    if [ "$srv_status" = "missing" ] || [ "$net_status" = "missing" ] || [ "$log_status" = "missing" ] || [ "$smart_status" = "missing" ]; then
        overall="partial"
    fi
    if [ "$opkg_ok" = "false" ] || [ "$lighttpd_running" = "false" ]; then
        overall="critical"
    fi

    timestamp=$(date -u '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || date '+%Y-%m-%d %H:%M:%S')

    # Вывод JSON (ip - утилита, ip_pkg - пакет Entware)
    printf '{"base":{"opkg":%s,"lighttpd_running":%s,"sed":%s,"awk":%s,"grep":%s,"ps":%s},"deps":{"cron_installed":%s,"cron_running":%s,"jq":%s,"ip":%s,"ip_path":"%s","ip_pkg_installed":%s},"sections":{"packages":"%s","services":"%s","monitoring":"%s","network":"%s","logger":"%s","smart":"%s"},"overall_status":"%s","timestamp":"%s"}' \
        "$opkg_ok" "$lighttpd_running" "$sed_ok" "$awk_ok" "$grep_ok" "$ps_ok" \
        "$cron_installed" "$cron_running" "$jq_ok" "$ip_ok" "$ip_path" "$ip_pkg_installed" \
        "$pkg_status" "$srv_status" "$mon_status" "$net_status" "$log_status" "$smart_status" \
        "$overall" "$timestamp"
}
