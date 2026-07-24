#!/bin/sh
export PATH=/opt/sbin:/opt/bin:/sbin:/bin:/usr/sbin:/usr/bin
ENDPOINT=service_watchdog_events exec /opt/web_entware/cgi-bin/go/entware-services
