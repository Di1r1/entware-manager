#!/bin/sh
# ==============================================
# Entware Manager - управление ttyd (исправленная версия)
# Версия: 0.07 (полные пути /opt/bin для CGI контекста)
# Дата: 2026-03-28
# ==============================================

. /opt/web_entware/lib/common.sh

# Функция получения статуса
get_status() {
    status_8089="stopped"
    status_9089="stopped"
    pid_8089=""
    pid_9089=""

    for pid in $(pgrep ttyd 2>/dev/null); do
        if [ -r "/proc/$pid/cmdline" ]; then
            cmdline=$(tr '\0' ' ' < "/proc/$pid/cmdline" 2>/dev/null)
            case "$cmdline" in
                *ttyd*8089*)
                    status_8089="running"
                    pid_8089=$pid
                    ;;
                *ttyd*9089*)
                    status_9089="running"
                    pid_9089=$pid
                    ;;
            esac
        fi
    done

    cat <<EOF
{
    "status": "ok",
    "htop": {"port": 8089, "state": "$status_8089", "pid": "$pid_8089"},
    "terminal": {"port": 9089, "state": "$status_9089", "pid": "$pid_9089"}
}
EOF
}

# Функции управления
start_ttyd() {
    port="$1"
    pass="$2"
    if [ "$port" = "8089" ]; then
        cmd="ttyd -p 8089 -W htop &"
    elif [ "$port" = "9089" ]; then
        cmd="ttyd -p 9089 -W --permit-any-origin"
        if [ -n "$pass" ]; then
            cmd="$cmd -c admin:$pass"
        fi
        if [ -x /opt/bin/bash ]; then
            cmd="$cmd /opt/bin/bash &"
        else
            cmd="$cmd /bin/sh &"
        fi
    else
        echo "{\"status\":\"error\",\"message\":\"Неизвестный порт\"}"
        return
    fi

    eval "$cmd"
    sleep 1
    if pgrep -f "ttyd.*-p $port" >/dev/null; then
        log_action "INFO" "ttyd запущен на порту $port"
        echo "{\"status\":\"ok\",\"message\":\"ttyd запущен на порту $port\"}"
    else
        echo "{\"status\":\"error\",\"message\":\"Не удалось запустить ttyd на порту $port\"}"
    fi
}

stop_ttyd() {
    port="$1"
    pids=$(pgrep -f "ttyd.*-p $port")
    if [ -n "$pids" ]; then
        kill $pids 2>/dev/null
        sleep 1
        if pgrep -f "ttyd.*-p $port" >/dev/null; then
            echo "{\"status\":\"error\",\"message\":\"Не удалось остановить ttyd на порту $port\"}"
        else
            log_action "INFO" "ttyd остановлен на порту $port"
            echo "{\"status\":\"ok\",\"message\":\"ttyd на порту $port остановлен\"}"
        fi
    else
        echo "{\"status\":\"error\",\"message\":\"ttyd на порту $port не найден\"}"
    fi
}

restart_ttyd() {
    port="$1"
    pass="$2"
    pids=$(pgrep -f "ttyd.*-p $port")
    [ -n "$pids" ] && kill $pids 2>/dev/null
    sleep 1
    start_ttyd "$port" "$pass"
}

# Основная логика
if [ "$REQUEST_METHOD" = "GET" ]; then
    json_out "$(get_status)"
elif [ "$REQUEST_METHOD" = "POST" ]; then
    # Извлекаем параметры
    action=$(post_param "action" "")
    port=$(post_param "port" "")
    pass=$(post_param "pass" "")

    # Декодируем URL
    action=$(url_decode "$action")
    port=$(url_decode "$port")
    pass=$(url_decode "$pass")

    # Проверка порта
    if ! echo "$port" | grep -q '^[0-9]\+$'; then
        json_out '{"status":"error","message":"Некорректный порт"}'
        exit 0
    fi

    case "$action" in
        start)
            result=$(start_ttyd "$port" "$pass")
            json_out "$result"
            ;;
        stop)
            result=$(stop_ttyd "$port")
            json_out "$result"
            ;;
        restart)
            result=$(restart_ttyd "$port" "$pass")
            json_out "$result"
            ;;
        *)
            json_out '{"status":"error","message":"Неизвестное действие"}'
            ;;
    esac
else
    json_out '{"status":"error","message":"Метод не поддерживается"}'
fi