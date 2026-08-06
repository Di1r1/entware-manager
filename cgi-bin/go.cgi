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

case "$dir" in
cgi-bin)
	case "$name" in
	available|packages|install|remove|upgrade|update|upgradable|api)
		ENDPOINT="$name" exec "$(go_bin pkg)" ;;
	stats|version|help|links_load|links_save|tmpfs|tmpfs_clean|view_file|delete_file|auth_config|crontab|crontab_update|backup|backup_restore|update_check|update_run|update_status|prepare_offline)
		ENDPOINT="$name" exec "$(go_bin stats)" ;;
	network_interfaces|network_routes|network_arp|network_status|network_stats|network_events|network_config|network_action)
		ENDPOINT="$name" exec "$(go_bin net)" ;;
	check_syntax|check_deps|services|service_action|ttyd_control|debug)
		ENDPOINT="$name" exec "$(go_bin services)" ;;
	temperature|wifi_temp|temp_history|wifi_temp_history|kill_pid|monitor_status|monitor_action|monitor_config|monitor_log)
		ENDPOINT="$name" exec "$(go_bin monitor)" ;;
	smart)
		exec "$(go_bin smart)" ;;
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
