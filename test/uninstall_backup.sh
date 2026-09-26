#!/bin/sh
# shellcheck disable=SC3043,SC2086,SC2046
# ==============================================
# Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
# Тесты Install/uninstall.sh: вынос бэкапа конфигов за пределы $TARGET_DIR
# перед rm -rf + честный итог (аудит P0-01).
# Запуск: sh test/uninstall_backup.sh   (подхватывается в make test)
#
# Скрипт гоняется в песочнице: все пути переопределяются через EWM_*,
# поэтому реальный /opt/web_entware не затрагивается.
# ==============================================
set -u

cd "$(dirname "$0")/.." || exit 1

PASS=0
FAIL=0
ok()   { PASS=$((PASS + 1)); echo "ok: $1"; }
fail() { FAIL=$((FAIL + 1)); echo "FAIL: $1${2:+ ($2)}"; }
assert_contains() { # desc haystack needle
	case "$2" in
		*"$3"*) ok "$1" ;;
		*) fail "$1" "не найдено '$3' в выводе" ;;
	esac
}
assert_rc() { # desc expected actual
	if [ "$2" = "$3" ]; then ok "$1"; else fail "$1" "rc expected '$2' got '$3'"; fi
}

T=$(mktemp -d) || exit 1
trap 'chmod -R u+w "$T" 2>/dev/null; rm -rf "$T"' EXIT

UNINSTALL="sh ./Install/uninstall.sh"

# Песочница: разворачиваем «установку» с бэкапом конфигов.
setup_install() { # root
	root=$1
	rm -rf "$root"
	mkdir -p "$root/web_entware/backup/opt/etc/lighttpd/conf.d" "$root/web_entware/bridge" "$root/conf.d" "$root/keep"
	echo "port 80" > "$root/web_entware/backup/opt/etc/lighttpd/lighttpd.conf"
	echo "alias" > "$root/web_entware/backup/opt/etc/lighttpd/conf.d/90-entware-manager.conf"
	echo '{"enabled":true}' > "$root/web_entware/auth_config.json"
	echo '{"id":"app"}' > "$root/web_entware/bridge/myapp.json"
	: > "$root/lighttpd.conf"
}

# Прогон uninstall в песочнице. $1 — корень песочницы.
run_uninstall() { # root
	root=$1
	env \
		EWM_TARGET_DIR="$root/web_entware" \
		EWM_KEEP_DIR="${EWM_KEEP_DIR:-$root/keep}" \
		EWM_LIGHTTPD_CONF="$root/lighttpd.conf" \
		EWM_CGI_CONF="$root/conf.d/30-cgi.conf" \
		EWM_CONFD_OUR="$root/conf.d/90-entware-manager.conf" \
		EWM_LOG_DIR="$root/entware-tmp" \
		EWM_SUDOERS_FILE="$root/sudoers.d/entware-smartctl" \
		EWM_SERVER_INIT="$root/init.d/S80entware-server" \
		EWM_INIT="$root/init.d/S80entware-lighttpd" \
		EWM_S85_INIT="$root/init.d/S85entware-watchdogs" \
		EWM_LIGHTTPD_INIT="$root/init.d/S80lighttpd" \
		EWM_LIGHTTPD_PIDF="$root/run/lighttpd.pid" \
		$UNINSTALL 2>&1
}

echo "=== uninstall: сохранение бэкапа вне удаляемого каталога ==="

# --- 1. Бэкап вынесен ДО удаления, содержимое на месте ---
root="$T/case1"
setup_install "$root"
out=$(run_uninstall "$root"); rc=$?
assert_rc "A) uninstall завершился без ошибки" 0 "$rc"
if [ -f "$root/keep/opt/etc/lighttpd/lighttpd.conf" ] && \
   [ -f "$root/keep/opt/etc/lighttpd/conf.d/90-entware-manager.conf" ]; then
	ok "A) бэкап вынесен в \$KEEP_DIR (оба файла)"
else
	fail "A) бэкап не вынесен" "keep=$(ls -R "$root/keep" 2>/dev/null | tr '\n' ' ')"
fi
if [ ! -d "$root/web_entware" ]; then
	ok "A) панель удалена"
else
	fail "A) панель не удалена"
fi
assert_contains "A) итог сообщает, что бэкап сохранён" "$out" "Бэкап конфигов сохранён"
case "$out" in
	*"не сохранялся"*) fail "A) итог противоречив: бэкап сохранён, но сказано «не сохранялся»" ;;
	*) ok "A) итог не противоречив (бэкап сохранён)" ;;
esac
case "$out" in
	*"Бэкап конфигов: $root/web_entware/backup/ (не удалён)"*)
		fail "A) нет старой врущей строки" ;;
	*)
		ok "A) старой врущей строки нет" ;;
esac

echo "--- предупреждение о конфигурации панели ---"
assert_contains "A) предупреждает, что конфиг панели удаляется" "$out" "Конфигурация панели"
assert_contains "A) предупреждает, что восстановить будет нечем" "$out" "восстановить будет нечем"

# --- 2. Без auth_config.json предупреждения нет ---
root="$T/case2"
setup_install "$root"
rm -f "$root/web_entware/auth_config.json"
out=$(run_uninstall "$root")
case "$out" in
	*"Конфигурация панели"*) fail "B) предупреждение без auth_config.json" ;;
	*) ok "B) без auth_config.json предупреждения нет" ;;
esac
assert_contains "B) бэкап всё равно сохранён" "$out" "Бэкап конфигов сохранён"

# --- 3. Нет каталога бэкапа — uninstall не падает, итог честный ---
root="$T/case3"
setup_install "$root"
rm -rf "$root/web_entware/backup"
out=$(run_uninstall "$root"); rc=$?
assert_rc "C) uninstall без каталога бэкапа — rc 0" 0 "$rc"
assert_contains "C) итог: бэкап не сохранялся" "$out" "не сохранялся"
if [ ! -d "$root/web_entware" ]; then
	ok "C) панель удалена"
else
	fail "C) панель не удалена"
fi

# --- 4. KEEP_DIR недоступен — uninstall всё равно завершается ---
# Цель структурно невозможна (ENOTDIR), а не запрещена правами: chmod 500
# не мешает root'у, и кейс был бы недетерминированным.
root="$T/case4"
setup_install "$root"
echo "not a dir" > "$root/notadir"
out=$(EWM_KEEP_DIR="$root/notadir/keep" run_uninstall "$root"); rc=$?
assert_rc "D1) удаление не сломалось, но код возврата 1 (бэкап пропадёт)" 1 "$rc"
assert_contains "D) предупреждает, что копии будут удалены" "$out" "Не удалось сохранить бэкап"
assert_contains "D) итог: бэкап не сохранялся" "$out" "не сохранялся"
if [ ! -d "$root/web_entware" ]; then
	ok "D) панель удалена несмотря на ошибку копирования"
else
	fail "D) панель не удалена"
fi

# --- 4b. Тот же сценарий, но цепочка копирования срабатывает: uninstall
# сообщает ненулевым кодом, что бэкап пропадёт (критерий аудита P0-01).
root="$T/case4b"
setup_install "$root"
out=$(run_uninstall "$root"); rc=$?
assert_rc "D2) успешный сценарий — rc 0" 0 "$rc"

# --- 5. Повторный запуск (бэкап уже вынесен) не ломает итог ---
root="$T/case5"
setup_install "$root"
mkdir -p "$root/web_entware/backup" && echo "v1" > "$root/web_entware/backup/keepme.txt"
run_uninstall "$root" > /dev/null 2>&1
# кладём бэкап заново (как если бы install.sh его восстановил) и гоняем ещё раз
mkdir -p "$root/web_entware/backup" && echo "v2" > "$root/web_entware/backup/keepme2.txt"
out=$(run_uninstall "$root"); rc=$?
assert_rc "E) повторный uninstall — rc 0" 0 "$rc"
assert_contains "E) бэкап перезаписан и сохранён" "$out" "Бэкап конфигов сохранён"
if [ -f "$root/keep/keepme2.txt" ]; then
	ok "E) содержимое второго прогона на месте"
else
	fail "E) содержимое второго прогона потеряно"
fi

echo "========================================"
echo "PASS: $PASS  FAIL: $FAIL"
[ "$FAIL" -eq 0 ] || exit 1
echo "ВСЕ ТЕСТЫ ПРОЙДЕНЫ"
