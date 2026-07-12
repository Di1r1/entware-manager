#!/bin/sh
# ==============================================
# Entware Manager - список служб (init.d) с PID
# Версия: 3.8 (все PID + дубли)
# ==============================================

. /opt/web_entware/lib/common.sh

echo "Content-type: application/json"
echo ""

SERVICES_DIR="/opt/etc/init.d"

# ========== ОДНОКРАТНЫЙ СБОР ДАННЫХ О ПРОЦЕССАХ ==========
PS_DATA=$(ps 2>/dev/null | grep -v "init.d" | grep -v "grep")

# Все PID по паттерну (без head -1)
find_all_pids_by_ps() {
    pattern="$1"
    [ -z "$pattern" ] && return 1
    echo "$PS_DATA" | grep -i "$pattern" | awk '{print $1}'
}

# Первый PID по паттерну
find_first_pid_by_ps() {
    pattern="$1"
    [ -z "$pattern" ] && return 1
    find_all_pids_by_ps "$pattern" | head -1
}

# ========== ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ ==========
get_var() {
    script="$1"
    var_name="$2"
    if [ -L "$script" ]; then
        target=$(readlink -f "$script")
        [ -f "$target" ] && script="$target"
    fi
    grep -E "^${var_name}=" "$script" 2>/dev/null | head -1 | cut -d'=' -f2 | tr -d '"' | tr -d "'"
}

build_pids_json() {
    pids="$1"
    first_pid=""
    json=""
    for p in $pids; do
        [ -z "$first_pid" ] && first_pid="$p"
        json="${json}\"${p}\","
    done
    json="[${json%,}]"
    echo "$json|$first_pid"
}

# Собрать все PID для службы
find_all_pids_for_service() {
    script="$1"
    service_name="$2"
    base_name="$3"

    # 1. PIDFILE из скрипта
    pidfile=$(get_var "$script" "PIDFILE")
    if [ -n "$pidfile" ] && [ -f "$pidfile" ]; then
        pid=$(cat "$pidfile" 2>/dev/null | tr -d '\n\r')
        if [ -n "$pid" ] && [ -d "/proc/$pid" ]; then
            echo "$pid"
            return
        fi
    fi

    # 2. Стандартные pid-файлы
    for pf in "/tmp/${base_name}.pid" "/var/run/${base_name}.pid" "/opt/var/run/${base_name}.pid" \
              "/tmp/${service_name}.pid" "/var/run/${service_name}.pid"; do
        if [ -f "$pf" ]; then
            pid=$(cat "$pf" 2>/dev/null | tr -d '\n\r')
            if [ -n "$pid" ] && [ -d "/proc/$pid" ]; then
                echo "$pid"
                return
            fi
        fi
    done

    # 3. По PROCS/NAME/DAEMON — все PID
    proc_name=$(get_var "$script" "PROCS")
    [ -z "$proc_name" ] && proc_name=$(get_var "$script" "NAME")
    [ -z "$proc_name" ] && proc_name=$(get_var "$script" "DAEMON")
    if [ -n "$proc_name" ]; then
        all=$(find_all_pids_by_ps "$proc_name")
        [ -n "$all" ] && echo "$all" && return
    fi

    # 4. По базовому имени
    all=$(find_all_pids_by_ps "$base_name")
    [ -n "$all" ] && echo "$all" && return

    # 5. По полному имени (S99kvas-ws)
    all=$(find_all_pids_by_ps "$service_name")
    [ -n "$all" ] && echo "$all" && return

    # 6. По .py файлу
    script_path=$(get_var "$script" "SCRIPT")
    if [ -n "$script_path" ]; then
        script_base=$(basename "$script_path")
        all=$(find_all_pids_by_ps "$script_base")
        [ -n "$all" ] && echo "$all" && return
    fi
}

# ========== ОСНОВНОЙ ЦИКЛ ПО СЛУЖБАМ ==========
first=1
echo "["
for script in "$SERVICES_DIR"/S* "$SERVICES_DIR"/K*; do
    [ -f "$script" ] || continue
    fullname=$(basename "$script")
    case "$fullname" in
        S*) name="${fullname#S}" ;;
        K*) name="${fullname#K}" ;;
        *)  continue ;;
    esac

    base_name=$(echo "$name" | sed 's/^[0-9]*//')
    pids_raw=$(find_all_pids_for_service "$script" "$fullname" "$base_name")

    if [ -n "$pids_raw" ]; then
        status="running"
        pids_json=$(build_pids_json "$pids_raw")
        pid="${pids_json#*|}"
        pids_arr="${pids_json%|*}"
    else
        status="stopped"
        pid=""
        pids_arr="[]"
    fi

    if [ "${fullname#S}" != "$fullname" ]; then
        enabled="true"
    else
        enabled="false"
    fi

    [ $first -eq 0 ] && echo ","
    first=0
    printf '{"name":"%s","status":"%s","enabled":%s,"pid":"%s","pids":%s}' "$name" "$status" "$enabled" "$pid" "$pids_arr"
done
echo "]"