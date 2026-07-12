#!/bin/sh
# ==============================================
# Entware Manager - удаление файла/папки в tmpfs
# Версия: 0.06 (финальная без отладки)
# Дата: 2026-03-30
# ==============================================

. /opt/web_entware/lib/common.sh

if [ "$REQUEST_METHOD" != "POST" ]; then
    json_out '{"status":"error","message":"Метод не поддерживается"}'
    exit 0
fi

path_raw=$(post_param "path" "")
path=$(url_decode "$path_raw")

case "$path" in
    /tmp/*|/tmp|/dev/shm/*|/dev/shm) ;;
    *)
        log_action "WARN" "Попытка удаления с недопустимым путём: $path"
        json_out '{"status":"error","message":"Доступ запрещен"}'
        exit 0
        ;;
esac

real_path="$path"

if [ ! -e "$real_path" ]; then
    log_action "WARN" "Попытка удаления несуществующего объекта: $real_path"
    json_out '{"status":"error","message":"Файл/папка не существует"}'
    exit 0
fi

if [ -d "$real_path" ]; then
    if rmdir "$real_path" 2>/dev/null; then
        log_action "INFO" "Удалена пустая папка: $real_path"
        json_out '{"status":"ok"}'
    else
        log_action "WARN" "Не удалось удалить папку (не пуста): $real_path"
        json_out '{"status":"error","message":"Папка не пуста, удаление отменено"}'
    fi
    exit 0
fi

if rm -f "$real_path" 2>/dev/null; then
    log_action "INFO" "Удалён файл: $real_path"
    json_out '{"status":"ok"}'
else
    log_action "WARN" "Не удалось удалить файл: $real_path"
    json_out '{"status":"error","message":"Не удалось удалить файл"}'
fi
