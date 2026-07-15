#!/bin/sh
# ==============================================
# Полная установка Entware Manager на роутер
# Версия: 2.0
# - устанавливает пакеты (opkg)
# - копирует файлы
# - настраивает lighttpd
# ==============================================

SELF_DIR="$(cd "$(dirname "$0")/.." && pwd)"
TARGET_DIR="/opt/web_entware"
LIGHTTPD_CONF="/opt/etc/lighttpd/lighttpd.conf"

echo "========================================"
echo " Установка Entware Manager"
echo "========================================"
echo ""

# ========== 1. ПРОВЕРКА ИСТОЧНИКА ==========
if [ ! -d "$SELF_DIR/lib" ]; then
    echo "Ошибка: исходный каталог $SELF_DIR не найден."
    echo "Скопируй папку deploy на роутер и запусти install.sh из неё."
    exit 1
fi

# ========== 2. ПРОВЕРКА ПАКЕТОВ ==========
echo ">>> Шаг 1: проверка установленных пакетов..."

# Список пакетов: имя_пакета|бинарник_или_файл_для_проверки
PACKAGES="\
lighttpd|/opt/sbin/lighttpd
ttyd|/opt/bin/ttyd
htop|/opt/bin/htop
jq|/opt/bin/jq
coreutils-base|/opt/bin/dirname
procps-ng|/opt/bin/ps
bridge-utils|/opt/sbin/brctl
ip-full|/opt/sbin/ip"

# Проверка busybox/встроенных утилит (не opkg)
MISSING_UTILS=""
for util in sed awk grep ps head cut tr sort; do
    command -v "$util" >/dev/null 2>&1 || MISSING_UTILS="$MISSING_UTILS $util"
done
if [ -n "$MISSING_UTILS" ]; then
    echo "  ⚠ встроенные утилиты отсутствуют:$MISSING_UTILS"
    echo "    (должны быть в busybox или procps-ng)"
fi

# Собираем список отсутствующих пакетов
MISSING_PKGS=$(echo "$PACKAGES" | while IFS='|' read -r pkg check_path; do
    [ -z "$pkg" ] && continue
    [ -f "$check_path" ] || [ -x "$check_path" ] || printf "%s " "$pkg"
done)
MISSING_PKGS=$(echo "$MISSING_PKGS" | sed 's/ $//')

if [ -z "$MISSING_PKGS" ]; then
    echo ""
    echo "  Все пакеты уже установлены ✓"
else
    echo ""
    echo "  Отсутствуют:$MISSING_PKGS"
    echo ""

    # ========== 3. УСТАНОВКА ПАКЕТОВ ==========
    echo ">>> Шаг 2: обновление списков и установка..."
    opkg update || echo "  ⚠ opkg update не удался, но пробуем продолжить"

    for pkg in $MISSING_PKGS; do
        echo "  → устанавливаю $pkg..."
        if opkg install "$pkg"; then
            echo "    ✓ $pkg установлен"
        else
            echo "    ✗ $pkg не удалось установить"
            FAILED_PKGS="$FAILED_PKGS $pkg"
        fi
    done

    # Проверка после установки
    POST_FAIL=""
    for pkg in $MISSING_PKGS; do
        check_path=$(echo "$PACKAGES" | grep "^${pkg}|" | cut -d'|' -f2)
        if [ -n "$check_path" ] && [ ! -f "$check_path" ] && [ ! -x "$check_path" ]; then
            POST_FAIL="$POST_FAIL $pkg"
        fi
    done

    if [ -n "$POST_FAIL" ]; then
        echo ""
        echo "  ⚠ После установки не найдены:$POST_FAIL"
        echo "    Возможно, нужен другой источник пакетов (opkg update)"
    fi
fi

# ========== 4. НАСТРОЙКА LIGHTTPD ==========
echo ""
echo ">>> Шаг 3: настройка lighttpd..."

# Если нет .so для mod_cgi — доустанавливаем пакет
if [ ! -f "/opt/lib/lighttpd/mod_cgi.so" ]; then
    echo "  → mod_cgi.so нет, устанавливаю lighttpd-mod-cgi..."
    opkg install lighttpd-mod-cgi 2>/dev/null || true
fi

# === alias.url: добавляем Entware Manager (через +=, если уже есть) ===
# Сначала удаляем старые entware-строки (чтобы не дублировать)
sed -i '\|"/entware-manager/"|d' "$LIGHTTPD_CONF" 2>/dev/null
sed -i '\|"/entware-cgi/"|d' "$LIGHTTPD_CONF" 2>/dev/null

if grep -q 'alias\.url' "$LIGHTTPD_CONF" 2>/dev/null; then
    echo 'alias.url += (' >> "$LIGHTTPD_CONF"
    echo '    "/entware-manager/" => "/opt/web_entware/",' >> "$LIGHTTPD_CONF"
    echo '    "/entware-cgi/" => "/opt/web_entware/cgi-bin/"' >> "$LIGHTTPD_CONF"
    echo ')' >> "$LIGHTTPD_CONF"
else
    echo 'alias.url = (' >> "$LIGHTTPD_CONF"
    echo '    "/entware-manager/" => "/opt/web_entware/",' >> "$LIGHTTPD_CONF"
    echo '    "/entware-cgi/" => "/opt/web_entware/cgi-bin/"' >> "$LIGHTTPD_CONF"
    echo ')' >> "$LIGHTTPD_CONF"
fi

# === server.port (8087 если не задан) ===
grep -q 'server\.port' "$LIGHTTPD_CONF" 2>/dev/null || \
    echo 'server.port = 8087' >> "$LIGHTTPD_CONF"

# === server.modules: mod_alias, mod_cgi если отсутствуют ===
grep -q 'mod_alias' "$LIGHTTPD_CONF" 2>/dev/null || \
    echo 'server.modules += ( "mod_alias" )' >> "$LIGHTTPD_CONF"
grep -q 'mod_cgi' "$LIGHTTPD_CONF" 2>/dev/null || \
    echo 'server.modules += ( "mod_cgi" )' >> "$LIGHTTPD_CONF"

# Удаляем cgi.execute-x-only из main.conf (чтобы не было дубля с 30-cgi.conf)
sed -i '/cgi\.execute-x-only/d' "$LIGHTTPD_CONF" 2>/dev/null

# === 30-cgi.conf: правим маппинг .cgi на /bin/sh и включаем execute-x-only ===
CGI_CONF="/opt/etc/lighttpd/conf.d/30-cgi.conf"
if [ -f "$CGI_CONF" ]; then
    if grep -q '\.cgi.*=>.*/opt/bin/perl' "$CGI_CONF" 2>/dev/null; then
        sed -i 's|".cgi" => "/opt/bin/perl"|".cgi" => "/bin/sh"|' "$CGI_CONF"
        echo "  → 30-cgi.conf: .cgi ➔ /bin/sh"
    elif ! grep -q '\.cgi.*=>.*/bin/sh' "$CGI_CONF" 2>/dev/null; then
        sed -i '/^cgi\.assign/,/)/{ /)/i\                               ".cgi" => "/bin/sh",' "$CGI_CONF"
        echo "  → 30-cgi.conf: добавлен .cgi ➔ /bin/sh"
    fi
    if grep -q '^#cgi\.execute-x-only = "enable"' "$CGI_CONF" 2>/dev/null; then
        sed -i 's|^#cgi.execute-x-only = "enable"|cgi.execute-x-only = "enable"|' "$CGI_CONF"
        echo "  → 30-cgi.conf: включён cgi.execute-x-only"
    elif ! grep -q '^cgi\.execute-x-only' "$CGI_CONF" 2>/dev/null; then
        echo 'cgi.execute-x-only = "enable"' >> "$CGI_CONF"
        echo "  → 30-cgi.conf: добавлен cgi.execute-x-only"
    fi
fi

# === static-file.exclude-extensions: .cgi если нет ===
if grep -q 'static-file\.exclude-extensions' "$LIGHTTPD_CONF" 2>/dev/null; then
    if ! grep -q 'static-file\.exclude-extensions.*\.cgi' "$LIGHTTPD_CONF" 2>/dev/null; then
        sed -i '/static-file\.exclude-extensions = (/s/)$/, ".cgi")/' "$LIGHTTPD_CONF"
    fi
fi

# Валидация
if lighttpd -t -f "$LIGHTTPD_CONF" 2>/dev/null; then
    echo "  ✓ конфиг OK"
else
    echo "  ⚠ конфиг не прошёл проверку. Проверь вручную:"
    echo "    lighttpd -t -f $LIGHTTPD_CONF"
fi

# ========== 5. КОПИРОВАНИЕ ФАЙЛОВ ==========
echo ""
echo ">>> Шаг 4: копирование файлов..."

mkdir -p "$TARGET_DIR"
cp -a "$SELF_DIR"/* "$TARGET_DIR/"

echo "  ✓ файлы скопированы в $TARGET_DIR"

# ========== 6. ПРАВА ДОСТУПА ==========
echo ""
echo ">>> Шаг 5: установка прав..."

chmod 755 "$TARGET_DIR"/cgi-bin/*.cgi 2>/dev/null
[ -d "$TARGET_DIR/cgi-bin/monitor" ] && chmod 755 "$TARGET_DIR"/cgi-bin/monitor/*.cgi
[ -d "$TARGET_DIR/cgi-bin/logger" ] && chmod 755 "$TARGET_DIR"/cgi-bin/logger/*.cgi
[ -d "$TARGET_DIR/cgi-bin/network" ] && chmod 755 "$TARGET_DIR"/cgi-bin/network/*.cgi
[ -d "$TARGET_DIR/cgi-bin/service_watchdog" ] && chmod 755 "$TARGET_DIR"/cgi-bin/service_watchdog/*.cgi
chmod 755 "$TARGET_DIR"/watchdog.sh 2>/dev/null
chmod 755 "$TARGET_DIR"/network_watchdog.sh 2>/dev/null
chmod 755 "$TARGET_DIR"/service_watchdog.sh 2>/dev/null
chmod 755 "$TARGET_DIR"/backup.sh 2>/dev/null
[ -f "$TARGET_DIR/logger/lib/logging.sh" ] && chmod 755 "$TARGET_DIR"/logger/lib/*.sh
[ -d "$TARGET_DIR/logger/scripts" ] && chmod 755 "$TARGET_DIR"/logger/scripts/*.sh
find "$TARGET_DIR" -type f -name "*.sh" -exec chmod 755 {} \;
find "$TARGET_DIR" -type f \( -name "*.js" -o -name "*.css" -o -name "*.html" -o -name "*.json" -o -name "*.svg" \) -exec chmod 644 {} \;
find "$TARGET_DIR/cgi-bin" -type d -exec chmod 755 {} \;

echo "  ✓ права установлены"

# ========== 7. ЗАПУСК LIGHTTPD ==========
echo ""
echo ">>> Шаг 6: запуск lighttpd..."

# Проверяем, не запущен ли уже
if pgrep -f "lighttpd.*$LIGHTTPD_CONF" >/dev/null 2>&1; then
    echo "  ✓ lighttpd уже запущен"
    echo "  → перезапускаю..."
    /opt/etc/init.d/S80lighttpd restart 2>&1
    sleep 1
    if pgrep -f lighttpd >/dev/null; then
        echo "  ✓ lighttpd перезапущен"
    else
        echo "  ⚠ lighttpd упал после перезапуска"
    fi
else
    echo "  → запускаю lighttpd..."
    /opt/etc/init.d/S80lighttpd start 2>&1
    sleep 1
    if pgrep -f lighttpd >/dev/null; then
        echo "  ✓ lighttpd запущен"
    else
        echo "  ⚠ lighttpd не запустился. Проверь:"
        echo "     lighttpd -D -f $LIGHTTPD_CONF"
    fi
fi

# ========== 8. ИТОГ ==========
echo ""
echo "========================================"
echo " РЕЗУЛЬТАТ УСТАНОВКИ"
echo "========================================"

if [ -n "$FAILED_PKGS" ]; then
    echo "  ⚠ Не удалось установить:$FAILED_PKGS"
    echo "    Проверь интернет и репозитории: opkg update"
fi

if [ -n "$POST_FAIL" ]; then
    echo "  ⚠ После установки не найдены:$POST_FAIL"
fi

echo ""
echo "  ✓ Файлы:    /opt/web_entware/"
echo "  ✓ Статика:  http://$(hostname):8087/entware-manager/"
echo "  ✓ CGI:      http://$(hostname):8087/entware-cgi/"
echo ""
echo "  Версия: $(jq -r .version /opt/web_entware/version.json 2>/dev/null || echo 'неизвестна')"
echo ""
  echo "  Открой в браузере: http://$(hostname -I | awk '{print $1}'):8087/entware-manager/"
echo ""
echo "  Если что-то пошло не так:"
echo "    /opt/var/log/lighttpd/error.log"
echo "    /tmp/entware/logs/"