#!/bin/sh
# ==============================================
# Entware Manager - управление службами (actions)
# Версия: 0.04 (полные пути /opt/bin для CGI контекста)
# Дата: 2026-03-28
# ==============================================

. /opt/web_entware/lib/common.sh

SERVICES_DIR="/opt/etc/init.d"

# Парсинг параметров
if [ "$REQUEST_METHOD" = "POST" ]; then
    name=$(post_param "name" "")
    action=$(post_param "action" "")
else
    name=$(get_param "name" "")
    action=$(get_param "action" "")
fi

# Валидация действия
case "$action" in
    start|stop|restart|enable|disable) ;;
    *) json_out "{\"error\":\"Недопустимое действие: $action\"}" ;;
esac

# Поиск скрипта службы
script=""
for prefix in S K ""; do
    candidate="$SERVICES_DIR/$prefix$name"
    if [ -x "$candidate" ]; then
        script="$candidate"
        break
    fi
done

if [ -z "$script" ]; then
    json_out "{\"error\":\"Служба $name не найдена или не исполняема\"}"
fi

# Выполнение действия
case "$action" in
    start)
        "$script" "$action" >/dev/null 2>&1
        if [ $? -eq 0 ]; then
            log_action "INFO" "Служба $name: $action"
            json_out '{"status":"ok"}'
        else
            log_action "ERROR" "Ошибка при $action службы $name"
            json_out "{\"error\":\"Ошибка при выполнении $action\"}"
        fi
        ;;
    stop)
        "$script" "$action" >/dev/null 2>&1
        if [ $? -eq 0 ]; then
            log_action "INFO" "Служба $name: $action"
            json_out '{"status":"ok"}'
        else
            log_action "ERROR" "Ошибка при $action службы $name"
            json_out "{\"error\":\"Ошибка при выполнении $action\"}"
        fi
        ;;
    restart)
        "$script" "$action" >/dev/null 2>&1
        if [ $? -eq 0 ]; then
            log_action "INFO" "Служба $name: $action"
            json_out '{"status":"ok"}'
        else
            log_action "ERROR" "Ошибка при $action службы $name"
            json_out "{\"error\":\"Ошибка при выполнении $action\"}"
        fi
        ;;
    enable)
        base=$(basename "$script")
        if [ "${base#S}" = "$base" ]; then
            # Нет префикса S или есть K
            if [ "${base#K}" != "$base" ]; then
                newname="$SERVICES_DIR/S${base#K}"
            else
                newname="$SERVICES_DIR/S$base"
            fi
            mv "$script" "$newname" 2>/dev/null
            if [ $? -eq 0 ]; then
                log_action "INFO" "Служба $name: включен автозапуск"
                json_out '{"status":"ok"}'
            else
                json_out "{\"error\":\"Не удалось включить автозапуск\"}"
            fi
        else
            # Уже есть S
            json_out '{"status":"ok"}'
        fi
        ;;
    disable)
        base=$(basename "$script")
        if [ "${base#S}" != "$base" ]; then
            # Есть S, меняем на K
            newname="$SERVICES_DIR/K${base#S}"
            mv "$script" "$newname" 2>/dev/null
            if [ $? -eq 0 ]; then
                log_action "INFO" "Служба $name: отключен автозапуск"
                json_out '{"status":"ok"}'
            else
                json_out "{\"error\":\"Не удалось отключить автозапуск\"}"
            fi
        else
            # Уже без S
            json_out '{"status":"ok"}'
        fi
        ;;
esac
