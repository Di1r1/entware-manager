#!/bin/sh
# Di1r1
export PATH=/opt/sbin:/opt/bin:/sbin:/bin:/usr/sbin:/usr/bin

detect_arch() {
	local arch
	arch=$(opkg print-architecture 2>/dev/null | awk '/^arch /{print $2}' | grep -v '^all$\|^noarch$' | head -1)
	[ -n "$arch" ] && echo "$arch" | sed 's/-[^-]*$//; s/aarch64/arm64/; s/x86_64/amd64/; s/i[3-6]86/386/' && return

	case "$(uname -m)" in
		aarch64)  echo "arm64" ;;
		armv7l|armv6l|armv5tejl|armv5tel) echo "arm" ;;
		mips)
			ELF="/opt/bin/opkg"
			[ ! -f "$ELF" ] && ELF="/proc/self/exe"
			byte=$(dd if="$ELF" bs=1 count=6 2>/dev/null | od -b | head -1 | awk '{print $7}')
			[ "$byte" = "001" ] && echo "mipsel" || echo "mips"
			;;
		mipsel)   echo "mipsel" ;;
		x86_64|amd64) echo "amd64" ;;
		i[3-6]86) echo "386" ;;
		*)        return 1 ;;
	esac
}

ARCH=$(detect_arch) || {
	echo "Content-type: text/plain"
	echo ""
	echo "Unsupported architecture: $(uname -m)"
	exit 1
}

GO_BASE="/opt/web_entware/cgi-bin/go"

self=$0
name=$(basename "$self" .cgi)
dir=$(basename "$(dirname "$self")")

go_bin() {
	local bin="$GO_BASE/$ARCH/entware-$1"
	if [ -x "$bin" ]; then
		echo "$bin"
	else
		# fallback: flat layout (до версии 1.06)
		echo "$GO_BASE/entware-$1"
	fi
}

# --- Гейт авторизации (lighttpd-режим) ---
# Если в auth_config.json enabled=true и задан hash — все CGI, кроме
# login/logout/session, требуют валидную файл-сессию /opt/var/run/panel_session.
# Токен сравнивается с HTTP_COOKIE (constant-time через cmp).
SESSION_FILE="/opt/var/run/panel_session"
SESSION_TTL_SECONDS=86400

auth_gate() {
	case "$name" in
	login|logout|session) return 0 ;;
	esac
	[ -f /opt/web_entware/auth_config.json ] || return 0
	ENABLED=$(jq -r '.enabled // false' /opt/web_entware/auth_config.json 2>/dev/null)
	[ "$ENABLED" = "true" ] || return 0
	HASH=$(jq -r '.password_hash // .password // ""' /opt/web_entware/auth_config.json 2>/dev/null)
	[ -n "$HASH" ] || return 0

	[ -f "$SESSION_FILE" ] || { echo_401; exit 1; }
	TOKEN=$(echo "$HTTP_COOKIE" | tr ';' '\n' | sed -n 's/^[[:space:]]*panel_session=//p' | head -1)
	[ -n "$TOKEN" ] || { echo_401; exit 1; }
	printf '%s\n' "$TOKEN" > /tmp/panel_cookie.$$ 2>/dev/null || { echo_401; exit 1; }
	chmod 600 /tmp/panel_cookie.$$ 2>/dev/null
	if ! cmp -s /tmp/panel_cookie.$$ "$SESSION_FILE"; then
		rm -f /tmp/panel_cookie.$$ 2>/dev/null
		echo_401
		exit 1
	fi
	rm -f /tmp/panel_cookie.$$ 2>/dev/null
	# TTL по mtime (BusyBox: stat -c %Y недоступен, используем date -r)
	MTIME=$(date -r "$SESSION_FILE" +%s 2>/dev/null || echo 0)
	NOW=$(date +%s)
	[ $((NOW - MTIME)) -le "$SESSION_TTL_SECONDS" ] || { rm -f "$SESSION_FILE"; echo_401; exit 1; }
	# Sliding TTL: продлеваем сессию не чаще раза в 10 минут (анти-износ флеша).
	if [ $((NOW - MTIME)) -gt 600 ]; then
		touch "$SESSION_FILE" 2>/dev/null
	fi
	return 0
}

echo_401() {
	echo "Status: 401 Unauthorized"
	echo "Content-Type: application/json; charset=utf-8"
	echo ""
	echo '{"error":"unauthorized"}'
}

auth_gate

case "$dir" in
cgi-bin)
	case "$name" in
	available|packages|installed|install|remove|upgrade|update|upgradable|api)
		ENDPOINT="$name" exec "$(go_bin pkg)" ;;
	stats|version|help|links_load|links_save|tmpfs|tmpfs_clean|view_file|delete_file|auth_config|crontab|crontab_update|backup|backup_restore|update_check|update_run|update_status|prepare_offline|login|logout|session)
		ENDPOINT="$name" exec "$(go_bin stats)" ;;
	network_interfaces|network_routes|network_arp|network_status|network_stats|network_events|network_action|network_wifi)
		ENDPOINT="$name" exec "$(go_bin net)" ;;
	check_syntax|check_deps|services|service_action|ttyd_control)
		ENDPOINT="$name" exec "$(go_bin services)" ;;
	temperature|wifi_temp|temp_history|wifi_temp_history|kill_pid)
		ENDPOINT="$name" exec "$(go_bin monitor)" ;;
	smart)
		exec "$(go_bin smart)" ;;
	rdp_status|rdp_start|rdp_stop|rdp_config)
		ENDPOINT="$name" exec "$(go_bin rdp)" ;;
	telegram_config|telegram_test)
		ENDPOINT="$name" exec "$(go_bin telegram)" ;;
	*)
		echo "Content-type: text/plain"
		echo ""
		echo "Unknown endpoint: $name"
		exit 1 ;;
	esac
	;;
network)
	ENDPOINT="network_$name" exec "$(go_bin net)" ;;
logger)
	ENDPOINT="logger_$name" exec "$(go_bin logger)" ;;
monitor)
	ENDPOINT="monitor_$name" exec "$(go_bin monitor)" ;;
service_watchdog)
	ENDPOINT="service_watchdog_$name" exec "$(go_bin services)" ;;
*)
	echo "Content-type: text/plain"
	echo ""
	echo "Unknown directory: $dir"
	exit 1 ;;
esac
