#!/bin/sh
# shellcheck disable=SC3043,SC3037,SC3057,SC1090,SC1091,SC2155,SC2034
# ==============================================
# Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
# Entware Manager - общие функции для демонов
# Версия: 3.0 (удалены мёртвые функции shell-CGI эпохи)
# Дата: 2026-08-16
# ==============================================

export PATH=/opt/sbin:/opt/bin:/sbin:/bin:/usr/sbin:/usr/bin
export HOME=/tmp

# --- Проверка PID (жив и не зомби) ---
pid_is_alive() {
    [ -z "$1" ] && return 1
    [ ! -d "/proc/$1" ] && return 1
    local s
    s=$(grep "State:" "/proc/$1/status" 2>/dev/null | awk '{print $2}')
    [ "$s" = "Z" ] && return 1
    return 0
}

# --- Поиск PID по паттерну cmdline (ps — основной путь, pgrep — ускорение) ---
# ps — всегда есть (BusyBox/procps), pgrep может отсутствовать или отличаться
# поведением между сборками, поэтому он только fallback.
find_pids() {
    local pattern="$1"
    if command -v ps >/dev/null 2>&1; then
        ps w 2>/dev/null | grep -v grep | grep -E "$pattern" | awk '{print $1}'
    elif command -v pgrep >/dev/null 2>&1; then
        pgrep -f "$pattern" 2>/dev/null
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
    local ts ip
    ts=$(date '+%Y-%m-%d %H:%M:%S')
    ip="${REMOTE_ADDR:-localhost}"
    echo "[$ts] [$level] [$ip] [$$] $msg" >> "$log_dir/$(date '+%Y-%m-%d').log"
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
    # Атомарная запись pid-файла (temp + mv, RULES п.10): чтение никогда не
    # увидит частично записанный файл.
    echo $! > "$pidfile.tmp.$$" 2>/dev/null && mv -f "$pidfile.tmp.$$" "$pidfile" 2>/dev/null
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

# --- Дата N дней назад (YYYY-MM-DD), чистый POSIX ---
# date_days_ago [N] — без GNU date -d (его может не быть в BusyBox).
# Арифметика через юлианский день: корректно переходит через границы
# месяцев и лет, включая високосные. Нужны только date +%Y/%m/%d.
date_days_ago() {
    local n="${1:-1}"
    local y m d
    y=$(date +%Y 2>/dev/null) || return 1
    m=$(date +%m 2>/dev/null) || return 1
    d=$(date +%d 2>/dev/null) || return 1
    # Убрать ведущие нули: POSIX $(( )) считает 08/09 ошибкой восьмеричной системы
    m=${m#0}
    d=${d#0}
    local a yy mm jdn b c dd e mon day
    a=$(( (14 - m) / 12 ))
    yy=$(( y + 4800 - a ))
    mm=$(( m + 12 * a - 3 ))
    jdn=$(( d + (153 * mm + 2) / 5 + 365 * yy + yy / 4 - yy / 100 + yy / 400 - 32045 - n ))
    a=$(( jdn + 32044 ))
    b=$(( (4 * a + 3) / 146097 ))
    c=$(( a - (146097 * b) / 4 ))
    dd=$(( (4 * c + 3) / 1461 ))
    e=$(( c - (1461 * dd) / 4 ))
    mon=$(( (5 * e + 2) / 153 ))
    day=$(( e - (153 * mon + 2) / 5 + 1 ))
    y=$(( 100 * b + dd - 4800 + (mon / 10) ))
    m=$(( mon + 3 - 12 * (mon / 10) ))
    [ "$m" -lt 10 ] && m="0$m"
    [ "$day" -lt 10 ] && day="0$day"
    printf '%s-%s-%s\n' "$y" "$m" "$day"
}