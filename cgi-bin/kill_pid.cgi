#!/bin/sh
# ==============================================
# Entware Manager - убийство процесса по PID
# Версия: 1.0
# ==============================================

. /opt/web_entware/lib/common.sh

if [ "$REQUEST_METHOD" = "POST" ]; then
    _POST_BODY=$(cat); export _POST_BODY
    pid=$(post_param "pid" "")
    pid=$(echo "$pid" | grep -o '^[0-9]*$')
elif [ "$REQUEST_METHOD" = "GET" ]; then
    pid=$(get_param "pid" "")
    pid=$(echo "$pid" | grep -o '^[0-9]*$')
fi

if [ -z "$pid" ]; then
    json_out '{"status":"error","error":"PID не указан или неверный"}'
fi

if [ ! -d "/proc/$pid" ]; then
    json_out '{"status":"error","error":"Процесс с PID '"$pid"' не найден"}'
fi

kill -9 "$pid" 2>/dev/null
if [ $? -eq 0 ]; then
    log_action "INFO" "Принудительно завершён процесс PID=$pid"
    json_out '{"status":"ok"}'
else
    json_out '{"status":"error","error":"Не удалось завершить процесс PID='"$pid"'"}'
fi
