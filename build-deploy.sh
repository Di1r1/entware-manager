#!/bin/bash
# ==============================================
# Сборка deploy-папки для Entware Manager
# Версия: 1.0
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

# Удаляем мусор (бэкапы, .bak, лишнее)
find "$DEPLOY_DIR" -name '*.bak' -delete 2>/dev/null

# Выставляем права
chmod 755 "$DEPLOY_DIR"/cgi-bin/*.cgi 2>/dev/null || true
[ -d "$DEPLOY_DIR/cgi-bin/monitor" ] && chmod 755 "$DEPLOY_DIR"/cgi-bin/monitor/*.cgi
[ -d "$DEPLOY_DIR/cgi-bin/logger" ] && chmod 755 "$DEPLOY_DIR"/cgi-bin/logger/*.cgi
[ -d "$DEPLOY_DIR/cgi-bin/network" ] && chmod 755 "$DEPLOY_DIR"/cgi-bin/network/*.cgi
[ -d "$DEPLOY_DIR/cgi-bin/service_watchdog" ] && chmod 755 "$DEPLOY_DIR"/cgi-bin/service_watchdog/*.cgi
chmod 755 "$DEPLOY_DIR"/watchdog.sh 2>/dev/null || true
chmod 755 "$DEPLOY_DIR"/network_watchdog.sh 2>/dev/null || true
chmod 755 "$DEPLOY_DIR"/service_watchdog.sh 2>/dev/null || true
chmod 755 "$DEPLOY_DIR"/backup.sh 2>/dev/null || true
[ -f "$DEPLOY_DIR/logger/lib/logging.sh" ] && chmod 755 "$DEPLOY_DIR"/logger/lib/*.sh
[ -d "$DEPLOY_DIR/logger/scripts" ] && chmod 755 "$DEPLOY_DIR"/logger/scripts/*.sh
find "$DEPLOY_DIR" -type f -name "*.sh" -exec chmod 755 {} \;
find "$DEPLOY_DIR" -type f \( -name "*.js" -o -name "*.css" -o -name "*.html" -o -name "*.json" -o -name "*.svg" \) -exec chmod 644 {} \;
find "$DEPLOY_DIR/cgi-bin" -type d -exec chmod 755 {} \;

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
echo "  1. Скопируйте deploy/ в /opt/temp/web_entware/"
echo "  2. chmod +x /opt/temp/web_entware/Install/install.sh"
echo "  3. /opt/temp/web_entware/Install/install.sh"
