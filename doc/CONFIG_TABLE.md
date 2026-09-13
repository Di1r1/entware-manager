# Конфиги Entware Manager — таблица (анализ, без правок исходников проекта в этой сессии)

Анализ выполнен по запросу «покажи таблицу, код не менялся». Правки в исходниках (`go/`) выполнены в текущей сессии (NOC-совместимость: `.cgi` whitelist в `static.go`, `packages_log` endpoint, `events.go` `tag=all`), но таблица описывает управление конфигами — это не код-правки.

## Таблица (8 файлов)

`auth_config.json` — `/opt/web_entware/auth_config.json` — `install.sh` (1044) — `stats/authconfig` — `cp -a` при upgrade — в deploy ❌
`server_config.json` — `/opt/web_entware/server_config.json` — `install.sh` (687, порт 8087) — `server/LoadConfig` — `cp -a` — ❌
`rdp_config.json` — `/opt/web_entware/rdp_config.json` — `install.sh` (906, порт 9099) — `rdp/proxy` (`.port`) — `cp -a` — ❌
`telegram_config.json` — `/opt/web_entware/telegram_config.json` — `install.sh` (924) — `telegram/config` — `cp -a` — ❌
`service_config.json` — `/opt/web_entware/service_config.json` — ❌ в deploy — `services/watchdog_config` — `cp -a` — ✅
`monitor_config.json` — `/opt/web_entware/monitor_config.json` — ❌ в deploy — `monitor/config` — `cp -a` — ✅
`network_config.json` — `/opt/web_entware/network_config.json` — `build-deploy.sh` (57, `cp -a`) — `network/config` — `cp -a`; в deploy ✅
`links.json` — `/opt/web_entware/links.json` — `.gitignore` (❌ в deploy) — `stats/links` — `cp -a` — ✅

## Правило (без правок)
`*_config.json` исключены из deploy через `.gitignore` + `build-deploy.sh` (строка 39: `*_config.json` в case-exclusion). `install.sh` шаг 5 (`STAGE_DIR` атомарный swap) переносит `*_config.json` + `links.json` + `.arch` + `backup/` из `$TARGET_DIR` в `$STAGE_DIR` через `cp -a`. `prerm` (строка 100-103 `build-ipk.sh`): при `upgrade` (`$1=upgrade`) `exit 0` — конфиги НЕ удаляются; удаляются только при `remove`/`purge`. Защита от потери настроек при `opkg upgrade`.
