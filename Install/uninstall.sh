#!/bin/sh
# ==============================================
# Полное удаление Entware Manager с роутера
# ==============================================

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

# ========== 2. ОСТАНОВКА LIGHTTPD ==========
echo ""
echo "${BOLD}[2/7] Остановка lighttpd${NC}"
echo "────────────────────────────────────────"

if pgrep -f lighttpd >/dev/null; then
	PID=$(pgrep -f lighttpd | head -1)
	if /opt/etc/init.d/S80lighttpd stop 2>/dev/null; then
		sleep 1
		if pgrep -f lighttpd >/dev/null; then
			warn "lighttpd (PID $PID) не остановился, убиваю..."
			kill "$PID" 2>/dev/null
		fi
		ok "lighttpd остановлен"
	else
		warn "Не удалось остановить lighttpd"
	fi
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

# ========== 4. ВОССТАНОВЛЕНИЕ КОНФИГОВ ИЗ БЭКАПА ==========
echo ""
echo "${BOLD}[4/8] Восстановление конфигов из бэкапа${NC}"
echo "────────────────────────────────────────"

if [ -d "$BACKUP_DIR" ]; then
	# Восстанавливаем lighttpd.conf
	if [ -f "$BACKUP_DIR/etc/lighttpd/lighttpd.conf" ]; then
		cp -a "$BACKUP_DIR/etc/lighttpd/lighttpd.conf" "$LIGHTTPD_CONF"
		ok "lighttpd.conf восстановлен из бэкапа"
	else
		# Если бэкапа нет — чистим вручную
		[ -f "$LIGHTTPD_CONF" ] && sed -i '\|"/entware-manager/"|d' "$LIGHTTPD_CONF" || true
		[ -f "$LIGHTTPD_CONF" ] && sed -i '\|"/entware-cgi/"|d' "$LIGHTTPD_CONF" || true
		ok "Строки entware удалены из lighttpd.conf (бэкап не найден)"
	fi

	# Восстанавливаем 30-cgi.conf
	if [ -f "$BACKUP_DIR/etc/lighttpd/conf.d/30-cgi.conf" ]; then
		cp -a "$BACKUP_DIR/etc/lighttpd/conf.d/30-cgi.conf" "$CGI_CONF"
		ok "30-cgi.conf восстановлен из бэкапа"
	else
		rm -f "$CGI_CONF" 2>/dev/null
		ok "30-cgi.conf удалён (бэкап не найден)"
	fi
else
	# Нет бэкапа — чистим вручную
	[ -f "$LIGHTTPD_CONF" ] && sed -i '\|"/entware-manager/"|d' "$LIGHTTPD_CONF" || true
	[ -f "$LIGHTTPD_CONF" ] && sed -i '\|"/entware-cgi/"|d' "$LIGHTTPD_CONF" || true
	rm -f "$CGI_CONF" 2>/dev/null
	ok "Строки entware удалены из lighttpd.conf, 30-cgi.conf удалён"
fi

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

if [ -f /opt/etc/init.d/S80lighttpd ]; then
	if /opt/etc/init.d/S80lighttpd start 2>/dev/null; then
		sleep 1
		if pgrep -f lighttpd >/dev/null; then
			ok "lighttpd запущен"
		else
			warn "lighttpd не запустился"
		fi
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
