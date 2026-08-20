#!/bin/sh
# ==============================================
# Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
# Ротация и архивирование логов
# Версия: 1.4 (вывод ротированных файлов: путь + размер)
# Дата: 2026-08-16
# ==============================================

SOURCE_DIR="/tmp/entware/logs"
TARGET_DIR="/opt/var/log/entware"
mkdir -p "$TARGET_DIR"

# Выводит "ROTATED|путь|размер" для переданного файла.
report() {
    [ -f "$1" ] || return 0
    size=$(wc -c < "$1" 2>/dev/null || echo 0)
    echo "ROTATED|$1|$size"
}

# 1. Ежедневные логи
yesterday=$(date -d "@$(($(date +%s) - 86400))" +%Y-%m-%d 2>/dev/null)
src="$SOURCE_DIR/$yesterday.log"
if [ -f "$src" ]; then
    cp "$src" "$TARGET_DIR/" 2>/dev/null && rm -f "$src"
    report "$TARGET_DIR/$yesterday.log"
fi

# 2. Логи watchdog (service_watchdog)
if [ -f "$SOURCE_DIR/service_events.log" ]; then
    cp "$SOURCE_DIR/service_events.log" "$TARGET_DIR/service_events.log" 2>/dev/null
    report "$TARGET_DIR/service_events.log"
    for f in "$SOURCE_DIR"/service_events.log.*.old; do
        [ -f "$f" ] || continue
        cp "$f" "$TARGET_DIR/" 2>/dev/null
        report "$TARGET_DIR/$(basename "$f")"
    done
fi

# 3. Логи watchdog (network_watchdog)
if [ -f "$SOURCE_DIR/network_events.log" ]; then
    cp "$SOURCE_DIR/network_events.log" "$TARGET_DIR/network_events.log" 2>/dev/null
    report "$TARGET_DIR/network_events.log"
    for f in "$SOURCE_DIR"/network_events.log.*.old; do
        [ -f "$f" ] || continue
        cp "$f" "$TARGET_DIR/" 2>/dev/null
        report "$TARGET_DIR/$(basename "$f")"
    done
fi

# 4. Очистка старых файлов в архиве (старше 30 дней)
find "$TARGET_DIR" -name "*.log" -mtime +30 -delete 2>/dev/null
find "$TARGET_DIR" -name "*.old" -mtime +30 -delete 2>/dev/null

# 5. Очистка .old файлов в /tmp (старше 7 дней)
find "$SOURCE_DIR" -name "*.old" -mtime +7 -delete 2>/dev/null

exit 0
