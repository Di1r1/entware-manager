#!/bin/sh
# ==============================================
# Ротация и архивирование логов
# Версия: 1.3 (исправлена обработка .old файлов)
# Дата: 2026-04-05
# ==============================================

SOURCE_DIR="/tmp/entware/logs"
TARGET_DIR="/opt/var/log/entware"
mkdir -p "$TARGET_DIR"

# 1. Ежедневные логи
yesterday=$(date -D "%s" -d "$(($(date +%s) - 86400))" +%Y-%m-%d 2>/dev/null || date +%Y-%m-%d -d "yesterday" 2>/dev/null || date -v-1d +%Y-%m-%d 2>/dev/null)
src="$SOURCE_DIR/$yesterday.log"
if [ -f "$src" ]; then
    cp "$src" "$TARGET_DIR/" 2>/dev/null && rm -f "$src"
fi

# 2. Логи watchdog (service_watchdog)
if [ -f "$SOURCE_DIR/service_events.log" ]; then
    cp "$SOURCE_DIR/service_events.log" "$TARGET_DIR/service_events.log" 2>/dev/null
    find "$SOURCE_DIR" -maxdepth 1 -name "service_events.log.*.old" -exec cp {} "$TARGET_DIR/" \; 2>/dev/null
fi

# 3. Логи watchdog (network_watchdog)
if [ -f "$SOURCE_DIR/network_events.log" ]; then
    cp "$SOURCE_DIR/network_events.log" "$TARGET_DIR/network_events.log" 2>/dev/null
    find "$SOURCE_DIR" -maxdepth 1 -name "network_events.log.*.old" -exec cp {} "$TARGET_DIR/" \; 2>/dev/null
fi

# 4. Очистка старых файлов в архиве (старше 30 дней)
find "$TARGET_DIR" -name "*.log" -mtime +30 -delete 2>/dev/null
find "$TARGET_DIR" -name "*.old" -mtime +30 -delete 2>/dev/null

# 5. Очистка .old файлов в /tmp (старше 7 дней)
find "$SOURCE_DIR" -name "*.old" -mtime +7 -delete 2>/dev/null

exit 0
