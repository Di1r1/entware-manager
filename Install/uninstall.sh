#!/bin/sh
# ==============================================
# Полное удаление Entware Manager с роутера
# ==============================================

# shellcheck disable=SC2034
RED='\033[1;31m'
GREEN='\033[1;32m'
YELLOW='\033[1;33m'
BOLD='\033[1m'
NC='\033[0m'

TARGET_DIR="/opt/web_entware"
BACKUP_DIR="$TARGET_DIR/backup"
LIGHTTPD_CONF="/opt/etc/lighttpd/lighttpd.conf"
CGI_CONF="/opt/etc/lighttpd/conf.d/30-cgi.conf"
LOG_DIR="/tmp/entware"
SUDOERS_FILE="/opt/etc/sudoers.d/entware-smartctl"

echo "${BOLD}========================================"
echo " Удаление Entware Manager"
echo "========================================${NC}"
echo ""

OK=0
WARN=0

ok()   { echo "${GREEN}  ✓ $1${NC}"; OK=$((OK+1)); }
warn() { echo "${YELLOW}  ⚠ $1${NC}"; WARN=$((WARN+1)); }

# ========== 1. ПРОВЕРКА ==========
echo "${BOLD}[1/7] Проверка установки${NC}"
echo "────────────────────────────────────────"

if [ -d "$TARGET_DIR" ]; then
	SIZE=$(du -sh "$TARGET_DIR" 2>/dev/null | cut -f1)
	FILES=$(find "$TARGET_DIR" -type f | wc -l)
	ok "Найдена установка: $TARGET_DIR ($SIZE, $FILES файлов)"
else
	warn "Установка не найдена ($TARGET_DIR отсутствует)"
fi

# ========== 2. ОСТАНОВКА СЕРВИСОВ ==========
echo ""
echo "${BOLD}[2/8] Остановка сервисов${NC}"
echo "────────────────────────────────────────"

# Останавливаем наш entware-server (если установлен)
EWM_SERVER_INIT="/opt/etc/init.d/S80entware-server"
EWM_SERVER_RAN=0
if [ -x "$EWM_SERVER_INIT" ]; then
	if $EWM_SERVER_INIT stop 2>/dev/null; then
		EWM_SERVER_RAN=1
		ok "entware-server остановлен (S80entware-server)"
	else
		warn "Не удалось остановить entware-server"
	fi
else
	warn "entware-server не установлен (S80entware-server нет)"
fi

LIGHTTPD_PIDF=/opt/var/run/lighttpd.pid
LIGHTTPD_PID=""
if [ -f "$LIGHTTPD_PIDF" ]; then
	LIGHTTPD_PID=$(cat "$LIGHTTPD_PIDF" 2>/dev/null | tr -d ' ')
	[ -d "/proc/$LIGHTTPD_PID" ] || LIGHTTPD_PID=""
fi

# Если есть наш init-скрипт — останавливаем ТОЛЬКО свой экземпляр (по pid-файлу),
# не трогая чужой lighttpd (например zapret). Иначе — стандартный S80lighttpd.
EWM_INIT="/opt/etc/init.d/S80entware-lighttpd"
if [ "$EWM_SERVER_RAN" = "1" ]; then
	# Режим entware-server: lighttpd не наш, чужой (nfqws/zapret) не трогаем
	warn "lighttpd не трогаю (режим entware-server)"
elif [ -n "$LIGHTTPD_PID" ] || [ -x "$EWM_INIT" ]; then
	if [ -x "$EWM_INIT" ]; then
		$EWM_INIT stop 2>/dev/null
		sleep 1
		ok "lighttpd остановлен (S80entware-lighttpd)"
	elif [ -n "$LIGHTTPD_PID" ]; then
		kill "$LIGHTTPD_PID" 2>/dev/null
		sleep 1
		rm -f "$LIGHTTPD_PIDF" 2>/dev/null
		ok "lighttpd остановлен (pid $LIGHTTPD_PID)"
	fi
elif /opt/etc/init.d/S80lighttpd stop 2>/dev/null; then
	sleep 1
	ok "lighttpd остановлен"
else
	warn "lighttpd не запущен"
fi

# ========== 3. УДАЛЕНИЕ ФАЙЛОВ ==========
echo ""
echo "${BOLD}[3/7] Удаление файлов${NC}"
echo "────────────────────────────────────────"

if [ -d "$TARGET_DIR" ]; then
	rm -rf "$TARGET_DIR"
	if [ ! -d "$TARGET_DIR" ]; then
		ok "$TARGET_DIR удалён"
	else
		warn "Не удалось полностью удалить $TARGET_DIR"
	fi
else
	warn "$TARGET_DIR уже удалён"
fi

# ========== 4. УДАЛЕНИЕ КОНФИГОВ LIGHTTPD ==========
echo ""
echo "${BOLD}[4/8] Удаление конфигов lighttpd${NC}"
echo "────────────────────────────────────────"

# Удаляем наш отдельный конфиг
rm -f "/opt/etc/lighttpd/conf.d/90-entware-manager.conf" 2>/dev/null
ok "90-entware-manager.conf удалён"

# Наш init-скрипт entware-server
rm -f "/opt/etc/init.d/S80entware-server" 2>/dev/null
ok "S80entware-server удалён"

# Старый init-скрипт entware-lighttpd
rm -f "/opt/etc/init.d/S80entware-lighttpd" 2>/dev/null
ok "S80entware-lighttpd удалён"

# Автозапуск демонов мониторинга
rm -f "/opt/etc/init.d/S85entware-watchdogs" 2>/dev/null
ok "S85entware-watchdogs удалён"

# 30-cgi.conf — удаляем (восстанавливать нечего, это наш файл)
rm -f "$CGI_CONF" 2>/dev/null
ok "30-cgi.conf удалён"

# Чистим старые строки из lighttpd.conf (на случай обновления с предыдущей версии)
[ -f "$LIGHTTPD_CONF" ] && sed -i '\|"/entware-manager/"|d' "$LIGHTTPD_CONF" || true
[ -f "$LIGHTTPD_CONF" ] && sed -i '\|"/entware-cgi/"|d' "$LIGHTTPD_CONF" || true
sed -i '/^[[:space:]]*server\.port[[:space:]]*=.*8087.*/d' "$LIGHTTPD_CONF" 2>/dev/null
sed -i '/^alias\.url = (\s*)$/d' "$LIGHTTPD_CONF" 2>/dev/null

# ========== 5. УДАЛЕНИЕ SUDOERS ==========
echo ""
echo "${BOLD}[5/8] Удаление sudoers${NC}"
echo "────────────────────────────────────────"

if [ -f "$SUDOERS_FILE" ]; then
	rm -f "$SUDOERS_FILE"
	ok "$SUDOERS_FILE удалён"
else
	warn "sudoers уже удалён"
fi

# ========== 6. УДАЛЕНИЕ ЛОГОВ ==========
echo ""
echo "${BOLD}[6/8] Удаление логов и временных файлов${NC}"
echo "────────────────────────────────────────"

if [ -d "$LOG_DIR" ]; then
	rm -rf "$LOG_DIR"
	ok "$LOG_DIR удалён"
else
	warn "Логи уже удалены"
fi

# ========== 7. ЗАПУСК LIGHTTPD ==========
echo ""
echo "${BOLD}[7/8] Запуск lighttpd${NC}"
echo "────────────────────────────────────────"

if [ "$EWM_SERVER_RAN" = "1" ]; then
	# Режим entware-server: светтитхтd чужой, ничего не перезапускаем
	warn "lighttpd не перезапускаю (режим entware-server, он не наш)"
elif [ -x "/opt/etc/init.d/S80entware-lighttpd" ]; then
	if /opt/etc/init.d/S80entware-lighttpd start 2>/dev/null; then
		sleep 1
		ok "lighttpd запущен (S80entware-lighttpd)"
	else
		warn "Не удалось запустить lighttpd (S80entware-lighttpd)"
	fi
elif [ -f /opt/etc/init.d/S80lighttpd ]; then
	if /opt/etc/init.d/S80lighttpd start 2>/dev/null; then
		sleep 1
		ok "lighttpd запущен"
	else
		warn "Не удалось запустить lighttpd"
	fi
else
	warn "Скрипт запуска lighttpd не найден"
fi

echo ""
echo "${BOLD}[8/8] Итог${NC}"
echo ""
echo "${GREEN}  ✓ Выполнено: $OK${NC}"
if [ $WARN -gt 0 ]; then
	echo "${YELLOW}  ⚠ Предупреждений: $WARN${NC}"
fi
echo ""
echo "  Пакеты Entware (lighttpd, jq, curl и др.) не удалены —"
echo "  они могут использоваться другими программами."
echo "  Если хочешь удалить вручную:"
echo "    opkg remove lighttpd lighttpd-mod-cgi jq curl sudo smartmontools ..."
echo ""
if [ -d "$BACKUP_DIR" ]; then
	echo "  Бэкап конфигов: $BACKUP_DIR/ (не удалён)"
	echo "  При переустановке install.sh восстановит их."
fi
echo ""
