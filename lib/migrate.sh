#!/bin/sh
# shellcheck disable=SC3043,SC2155,SC2034,SC2015
# ==============================================
# Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
# Entware Manager - миграция lighttpd → entware-server (Variant 1)
# Порт-хранитель общего lighttpd + бесшовный переход на go-режим.
# Source из install.sh / uninstall.sh / build-ipk.sh (prerm). Сам не исполняется.
# ==============================================

# EWM_PATH_PREFIX — только для тестов (подменяет netstat/ss). В проде пусто.
export PATH="${EWM_PATH_PREFIX:+$EWM_PATH_PREFIX:}/opt/sbin:/opt/bin:/sbin:/bin:/usr/sbin:/usr/bin"

# --- Пути (переопределяются в тестах через env ДО source) ---
EWM_LIGHTTPD_DIR="${EWM_LIGHTTPD_DIR:-/opt/etc/lighttpd}"
EWM_LIGHTTPD_CONF="${EWM_LIGHTTPD_CONF:-$EWM_LIGHTTPD_DIR/lighttpd.conf}"
EWM_PORT_KEEPER="${EWM_PORT_KEEPER:-$EWM_LIGHTTPD_DIR/conf.d/90-entware-manager.conf}"
EWM_FALLBACK_PORT="${EWM_FALLBACK_PORT:-8086}"
EWM_LIGHTTPD_INIT="${EWM_LIGHTTPD_INIT:-/opt/etc/init.d/S80lighttpd}"

PORT_KEEPER_MARKER="EM port-keeper: DO NOT DELETE"

# --- Эффективный server.port общего lighttpd (last-wins, main+conf.d, glob-порядок).
# Наш 90-conf пропускается (он будет удалён/переписан). Scoped-блоки
# ($HTTP[...], $SERVER["socket"]) пропускаются — это не глобальный порт.
# Принимаются "=" и ":=" (у nfqws встречается ":="). Выводит порт или пусто (=80).
migrate_effective_lighttpd_port() {
	local file depth line stripped p
	port=""
	for file in "$EWM_LIGHTTPD_CONF" "$EWM_LIGHTTPD_DIR"/conf.d/*.conf; do
		[ -f "$file" ] || continue
		[ "$file" = "$EWM_PORT_KEEPER" ] && continue
		depth=0
		while IFS= read -r line; do
			case "$line" in
				*\}) depth=$((depth - 1)); [ "$depth" -lt 0 ] && depth=0; continue ;;
			esac
			case "$line" in
				*'$HTTP['*\{*|*'$SERVER['*\{*) depth=$((depth + 1)); continue ;;
			esac
			[ "$depth" -gt 0 ] && continue
			stripped=$(printf '%s' "$line" | sed 's/[[:space:]]*#.*//')
			case "$stripped" in
				server.port*)
					p=$(printf '%s' "$stripped" | sed -n 's/^[[:space:]]*server\.port[[:space:]:=]*[[:space:]]*\([0-9][0-9]*\).*/\1/p')
					[ -n "$p" ] && port="$p" ;;
			esac
		done < "$file"
	done
	printf '%s' "$port"
}

# --- Свободен ли порт? (netstat-first, ss fallback; без обоих — считаем свободным,
# как portIsBusy в go/internal/rdp/port.go). Возвращает 0 если свободен.
migrate_port_free() {
	local port="$1" out
	if command -v netstat >/dev/null 2>&1; then
		out=$(netstat -ltn 2>/dev/null)
	elif command -v ss >/dev/null 2>&1; then
		out=$(ss -ltn 2>/dev/null)
	else
		return 0
	fi
	printf '%s\n' "$out" | grep -q ":${port}[[:space:]]" && return 1
	return 0
}

# --- Есть ли ЧУЖИЕ конфиги в conf.d (koffe 98/99, web4static/nfqws2 30-cgi и т.п.)?
# Системные/наши исключаются. 30-cgi.conf — наш, если ровно наш шаблон (≤3 строк,
# cgi.assign .cgi /bin/sh, без perl/ruby/python/php).
migrate_has_third_party_confd() {
	local f base
	for f in "$EWM_LIGHTTPD_DIR"/conf.d/*.conf; do
		[ -f "$f" ] || continue
		base=$(basename "$f")
		case "$base" in
			*.bak|*.orig|*~) continue ;;
			30-mime.conf|30-dir-listing.conf|30-access.conf|30-deflate.conf|30-proxy.conf|90-entware-manager.conf) continue ;;
			30-cgi.conf)
				if [ "$(wc -l < "$f" 2>/dev/null || echo 0)" -le 3 ] \
					&& grep -q 'cgi\.assign.*\.cgi.*/bin/sh' "$f" 2>/dev/null \
					&& ! grep -Eq 'perl|ruby|python|php' "$f" 2>/dev/null; then
					continue
				fi
				;;
		esac
		return 0
	done
	return 1
}

# --- Выбрать, что делать с 90-conf. Выводит:
#   ""        — удалить (эффективный порт свободен и чужих сервисов нет);
#   <число>   — писать порт-хранитель на этот порт.
# EWM_LIGHTTPD_PORT переопределяет порт. 8087 (порт EM) не используется.
migrate_choose_portkeeper() {
	local eff override cand
	# Уже порт-хранитель → возвращаем его текущий порт (идемпотентно, порт может
	# быть «занят» нашим же lighttpd после предыдущей миграции).
	if migrate_is_portkeeper "$EWM_PORT_KEEPER"; then
		grep -E '^[[:space:]]*server\.port' "$EWM_PORT_KEEPER" 2>/dev/null \
			| sed -n 's/^[[:space:]]*server\.port[[:space:]:=]*[[:space:]]*\([0-9][0-9]*\).*/\1/p' \
			| tail -1
		return 0
	fi
	override="${EWM_LIGHTTPD_PORT:-}"
	if [ -n "$override" ]; then
		case "$override" in
			''|*[!0-9]*) echo ""; return 0 ;;
		esac
		echo "$override"
		return 0
	fi
	eff=$(migrate_effective_lighttpd_port)
	if [ -n "$eff" ] && migrate_port_free "$eff"; then
		# Чужие сервисы есть → держим их на текущем свободном порту; нет → удаляем.
		if migrate_has_third_party_confd; then echo "$eff"; else echo ""; fi
		return 0
	fi
	# eff пуст (дефолт 80) и 80 свободен + чужих нет → можно удалить.
	if [ -z "$eff" ] && migrate_port_free 80 && ! migrate_has_third_party_confd; then
		echo ""
		return 0
	fi
	# Нужен порт-хранитель: fallback, иначе ближайший свободный (кроме 8087).
	for cand in "$EWM_FALLBACK_PORT" 8085 8084 8083 8082 8081 8090 8091; do
		if [ "$cand" != "8087" ] && migrate_port_free "$cand"; then
			echo "$cand"
			return 0
		fi
	done
	echo ""
	return 0
}

# --- Записать порт-хранитель (атомарно temp+mv). Переносим server.modules из
# прежнего 90-conf (mod_alias/mod_cgi/mod_proxy/mod_access) + модули, которых ждут
# чужие conf.d (koffe alias/cgi, web4static) — иначе lighttpd начнёт игнорировать
# alias.url/cgi.assign (WARNING "unknown config-key") и сервисы сломаются.
migrate_write_portkeeper() {
	local port="$1" f mods
	# Идемпотентность: уже такой порт-хранитель → не перезаписываем.
	if migrate_is_portkeeper "$EWM_PORT_KEEPER" \
		&& grep -q "^[[:space:]]*server\.port[[:space:]:=]*[[:space:]]*$port\$" "$EWM_PORT_KEEPER" 2>/dev/null; then
		return 0
	fi
	mods=""
	# Переносим server.modules из прежнего 90-conf/порт-хранителя (идемпотентно:
	# повторная запись не теряет ранее перенесённые модули, напр. mod_access).
	if [ -f "$EWM_PORT_KEEPER" ]; then
		mods=$(printf '%s\n' "$mods"; grep -E 'server\.modules \+=' "$EWM_PORT_KEEPER" 2>/dev/null)
	fi
	for f in "$EWM_LIGHTTPD_DIR"/conf.d/*.conf; do
		[ -f "$f" ] || continue
		[ "$f" = "$EWM_PORT_KEEPER" ] && continue
		grep -q 'cgi\.assign' "$f" 2>/dev/null && mods=$(printf '%s\n' "$mods"; echo "server.modules += ( \"mod_cgi\" )")
		grep -q 'alias\.url' "$f" 2>/dev/null && mods=$(printf '%s\n' "$mods"; echo "server.modules += ( \"mod_alias\" )")
		grep -q 'proxy\.server' "$f" 2>/dev/null && mods=$(printf '%s\n' "$mods"; echo "server.modules += ( \"mod_proxy\" )")
	done
	mkdir -p "$(dirname "$EWM_PORT_KEEPER")" 2>/dev/null
	{
		echo "# Entware Manager $PORT_KEEPER_MARKER — shared lighttpd stays on $port."
		echo "# Панель EM переехала на entware-server:8087. Директив EM здесь нет."
		printf '%s\n' "$mods" | sed '/^$/d' | sort -u
		echo "server.port = $port"
	} > "$EWM_PORT_KEEPER.tmp.$$" 2>/dev/null && mv -f "$EWM_PORT_KEEPER.tmp.$$" "$EWM_PORT_KEEPER" 2>/dev/null
}

# --- Это порт-хранитель? Строго: маркер + ровно одна строка server.port.
migrate_is_portkeeper() {
	local file="$1" n
	[ -f "$file" ] || return 1
	grep -q "$PORT_KEEPER_MARKER" "$file" 2>/dev/null || return 1
	n=$(grep -c '^[[:space:]]*server\.port' "$file" 2>/dev/null || echo 0)
	[ "$n" = "1" ] || return 1
	return 0
}

# --- Перезагрузить lighttpd так, чтобы он освободил $1 (порт EM) и применил
# новый набор модулей. Полный restart надёжнее SIGHUP: SIGHUP НЕ перечитывает
# server.modules, из-за чего alias.url/cgi.assign игнорируются (WARNING
# "unknown config-key"), а это ломает koffe/web4static. Проверено на роутере.
migrate_reload_lighttpd() {
	local port="$1" pid i
	pid=$(pgrep lighttpd 2>/dev/null | head -1)
	[ -z "$pid" ] && pid=$(ps w 2>/dev/null | grep lighttpd | grep -v grep | awk 'NR==1{print $1}')
	[ -n "$pid" ] || return 0
	if command -v lighttpd >/dev/null 2>&1; then
		lighttpd -t -f "$EWM_LIGHTTPD_CONF" >/dev/null 2>&1 || return 1
	fi
	if [ -x "$EWM_LIGHTTPD_INIT" ]; then
		"$EWM_LIGHTTPD_INIT" restart >/dev/null 2>&1 || kill -HUP "$pid" 2>/dev/null
	else
		kill -HUP "$pid" 2>/dev/null
	fi
	i=0
	while [ "$i" -lt 15 ]; do
		if migrate_port_free "$port"; then return 0; fi
		sleep 1
		i=$((i + 1))
	done
	return 1
}
