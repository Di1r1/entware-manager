#!/bin/bash
# ==============================================
# Сборка deploy-папки для Entware Manager
# Версия: 2.0 — Go compilation + symlinks
# Использование: ./build-deploy.sh [--tar]
#   --tar  — дополнительно создать tar.gz архив
# ==============================================

set -e

PROJECT_DIR="$(dirname "$0")"
DEPLOY_DIR="$PROJECT_DIR/deploy"
TIMESTAMP=$(date '+%Y%m%d_%H%M%S')

# Очищаем deploy
rm -rf "$DEPLOY_DIR"
mkdir -p "$DEPLOY_DIR"

echo "=== Сборка deploy ==="

# Копируем корневые файлы (исключая dev-файлы)
for f in "$PROJECT_DIR"/*; do
    name=$(basename "$f")
    case "$name" in
        deploy|go|tmp|build-deploy.sh|TECH_SPEC.md|*.tar.gz)
            continue ;;
    esac
    if [ -d "$f" ]; then
        cp -a "$f" "$DEPLOY_DIR/"
        echo "  DIR:  $name"
    elif [ -f "$f" ]; then
        cp -a "$f" "$DEPLOY_DIR/"
        echo "  FILE: $name"
    fi
done

# Компиляция Go бинарников
echo ""
echo "=== Компиляция Go ==="
mkdir -p "$DEPLOY_DIR/cgi-bin/go"
cd "$PROJECT_DIR/go"
for cmd in entware-pkg entware-stats entware-net entware-logger entware-services entware-monitor entware-smart; do
    echo -n "  $cmd... "
    GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o "$DEPLOY_DIR/cgi-bin/go/$cmd" "./cmd/$cmd/" 2>&1
    echo "OK ($(du -h "$DEPLOY_DIR/cgi-bin/go/$cmd" | cut -f1))"
done

# ZIP-упаковка бинарников (опционально, если upx доступен)
if command -v upx &>/dev/null || [ -x /tmp/upx-4.2.4-amd64_linux/upx ]; then
    UPX=$(command -v upx || echo "/tmp/upx-4.2.4-amd64_linux/upx")
    echo ""
    echo "=== UPX сжатие ==="
    for f in "$DEPLOY_DIR"/cgi-bin/go/entware-*; do
        echo -n "  $(basename $f)... "
        tmpf=$(mktemp)
        "$UPX" -9 "$f" -o "$tmpf" 2>/dev/null && mv "$tmpf" "$f" && echo "OK ($(du -h "$f" | cut -f1))" || { rm -f "$tmpf"; echo "SKIP"; }
    done
fi

# Создаём go.cgi диспетчер
cp "$PROJECT_DIR/cgi-bin/go.cgi" "$DEPLOY_DIR/cgi-bin/go.cgi"

# Создаём симлинки для всех эндпоинтов
echo ""
echo "=== Симлинки cgi → go.cgi ==="

# Корневые эндпоинты
cd "$DEPLOY_DIR/cgi-bin"
cat <<'LIST' | while read ep; do ln -sf go.cgi "$ep.cgi"; echo "  $ep.cgi -> go.cgi"; done
api auth_config available check_deps check_syntax crontab crontab_update debug
delete_file help install kill_pid links_load links_save network_action network_arp
network_config network_events network_interfaces network_routes network_stats
network_status packages remove service_action services smart stats temp_history
temperature tmpfs ttyd_control update upgradable upgrade version view_file
wifi_temp wifi_temp_history
LIST

# Поддиректории
for d in network logger monitor service_watchdog; do
    mkdir -p "$d"
done

cd network
for ep in action arp config events interfaces routes stats status; do
    ln -sf ../go.cgi "$ep.cgi"
done
echo "  network/*.cgi -> ../go.cgi"

cd ../logger
for ep in clear config debug debug_path find_by_name rotate system_log system_logs view; do
    ln -sf ../go.cgi "$ep.cgi"
done
echo "  logger/*.cgi -> ../go.cgi"

cd ../monitor
for ep in action config log status; do
    ln -sf ../go.cgi "$ep.cgi"
done
echo "  monitor/*.cgi -> ../go.cgi"

cd ../service_watchdog
for ep in action config events status; do
    ln -sf ../go.cgi "$ep.cgi"
done
echo "  service_watchdog/*.cgi -> ../go.cgi"

cd "$DEPLOY_DIR"

# Выставляем права
find "$DEPLOY_DIR/cgi-bin" -type l -exec chmod 755 {} \;
chmod 755 "$DEPLOY_DIR/cgi-bin/go.cgi"
chmod 755 "$DEPLOY_DIR/cgi-bin/go/"*
find "$DEPLOY_DIR" -type d -exec chmod 755 {} \;
find "$DEPLOY_DIR" -type f -name "*.js" -o -name "*.css" -o -name "*.html" -o -name "*.json" -o -name "*.svg" | xargs chmod 644 2>/dev/null || true
find "$DEPLOY_DIR" -type f -name "*.sh" -exec chmod 755 {} \;

# Удаляем мусор
find "$DEPLOY_DIR" -name '*.bak' -delete 2>/dev/null

echo ""
echo "=== Deploy собран: $DEPLOY_DIR ==="
echo "Размер: $(du -sh "$DEPLOY_DIR" | cut -f1)"
echo "Файлов: $(find "$DEPLOY_DIR" -type f | wc -l)"

# Опционально: tar.gz
if [ "$1" = "--tar" ]; then
    ARCHIVE="$PROJECT_DIR/entware-manager_$TIMESTAMP.tar.gz"
    tar -czf "$ARCHIVE" -C "$PROJECT_DIR" deploy/
    echo "Архив: $ARCHIVE ($(du -h "$ARCHIVE" | cut -f1))"
fi

echo ""
echo "Для установки на роутер:"
echo "  1. Скопируйте deploy/ в /opt/tmp/ на роутере"
echo "  2. chmod +x /opt/tmp/deploy/Install/install.sh"
echo "  3. /opt/tmp/deploy/Install/install.sh"
