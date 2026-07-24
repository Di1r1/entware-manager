#!/bin/sh
export PATH=/opt/sbin:/opt/bin:/sbin:/bin:/usr/sbin:/usr/bin
self=$0
name=$(basename "$self" .cgi)
dir=$(basename "$(dirname "$self")")

case "$dir" in
cgi-bin)
	case "$name" in
	available|packages|install|remove|upgrade|update|upgradable|api)
		ENDPOINT="$name" exec /opt/web_entware/cgi-bin/go/entware-pkg
		;;
	stats|version|help|links_load|links_save|tmpfs|view_file|delete_file|auth_config|crontab|crontab_update)
		ENDPOINT="$name" exec /opt/web_entware/cgi-bin/go/entware-stats
		;;
	network_interfaces|network_routes|network_arp|network_status|network_stats|network_events|network_config|network_action)
		ENDPOINT="$name" exec /opt/web_entware/cgi-bin/go/entware-net
		;;
	check_syntax|check_deps|services|service_action|ttyd_control|debug)
		ENDPOINT="$name" exec /opt/web_entware/cgi-bin/go/entware-services
		;;
	temperature|wifi_temp|temp_history|wifi_temp_history|kill_pid|monitor_status|monitor_action|monitor_config|monitor_log)
		ENDPOINT="$name" exec /opt/web_entware/cgi-bin/go/entware-monitor
		;;
	smart)
		exec /opt/web_entware/cgi-bin/go/entware-smart
		;;
	*)
		echo "Content-type: text/plain"
		echo ""
		echo "Unknown endpoint: $name"
		exit 1
		;;
	esac
	;;
network)
	ENDPOINT="network_$name" exec /opt/web_entware/cgi-bin/go/entware-net
	;;
logger)
	ENDPOINT="logger_$name" exec /opt/web_entware/cgi-bin/go/entware-logger
	;;
monitor)
	ENDPOINT="monitor_$name" exec /opt/web_entware/cgi-bin/go/entware-monitor
	;;
service_watchdog)
	ENDPOINT="service_watchdog_$name" exec /opt/web_entware/cgi-bin/go/entware-services
	;;
*)
	echo "Content-type: text/plain"
	echo ""
	echo "Unknown directory: $dir"
	exit 1
	;;
esac
