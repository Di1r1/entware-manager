# Изменения проекта

Правила проекта: [`RULES.md`](../RULES.md)

## 1.07.4 (2026-08-01)

### Оптимизация для слабых роутеров

- **Кэширование тяжёлых CGI-ответов** (новый пакет `go/internal/cache`, атомарная запись temp+rename, TTL):
  - `stats.cgi`: кэш списков opkg (`opkg list` + `opkg list-installed`) на 60 с — **180→98 мс (-46%)**;
  - `network_stats.cgi`: кэш JSON-ответа на 5 с, принудительное обновление через `?fresh=1` (кнопка обновления в карточке сети) — **50→38 мс (-31%)**;
  - `update_check.cgi`: кэш проверки обновлений на 60 с — **470→79 мс (-83%)**.
- **Инвалидация кэша opkg**: при успешной установке/удалении/обновлении пакета кэшированные списки удаляются — счётчики и списки пакетов всегда актуальны.
- **B1 — один jq вместо нескольких**: `watchdog.sh` (8→1 вызов), `service_watchdog.sh` (8→1), `network_watchdog.sh` (3→1) — конфиг читается одним `jq -r` (значения построчно через `read`, с сохранением дефолтов и обработки массивов). Демоны проверены на роутере (старт/работа/стоп).
- Тесты: `go test ./...`, `go vet`, `sh -n` всех скриптов, ShellCheck — чисто.

## 1.07.3 (2026-08-01)

### Новое

- **Проверка системы — новые зависимости**: `check_deps.cgi` теперь проверяет `curl`, `bash` и `brctl` (bridge-utils). Секция `network` требует и `ip`, и `brctl` (brctl нужен для `brctl show` в карточке сети). В рекомендациях — подсказки `opkg install curl/bash/bridge-utils`.
- **Проверка синтаксиса в UI**: новый блок «Проверка синтаксиса скриптов» прямо в окне «Проверка системы» — вывод `check_syntax.cgi` (каждый `.cgi`/`.sh` через `sh -n`, статус по каждому файлу + счётчик ошибок).
- **Тесты**: `TestCheckDeps_NewFields`, `TestCheckDeps_SyntaxFields` (go test/vet/build — чисто).

### Исправлено

- **check_syntax.cgi отдавал 404 из UI** — симлинк не создавался в `build-deploy.sh` (в списке эндпоинтов отсутствовал). Добавлен, эндпоинт снова доступен.

## 1.07.2 (2026-08-01)

### Новое

- **Пресеты тем** (violet / ocean / forest) + единый `theme.js` для всех страниц; выбор через попап-кружки у кнопки темы; миграция старых `day`/`night` → violet; синхронизация между вкладками.
- **Карточка сети**: блоки Интерфейсы/Порты/Сети/виртуальные порты переделаны в карточки-счётчики; WiFi — раздельные счётчики TX/RX с детализацией по интерфейсам.
- **WAN/порты**: статус по carrier или членству в мосте; виртуальные интерфейсы без `device` скрыты (ethoip0); WAN ppp0 — зелёный бейдж.
- Кнопки и заголовки таблиц затемнены (тёмный градиент).

## 1.07.1 (2026-07-31)

### Исправлено

- **install.sh**: очистка пустых `alias.url` блоков в lighttpd.conf (фикс invalid-conf на старых установках).
- **Файловый менеджер**: строка «.. (наверх)» зафиксирована вне сортировки; стрелки направления сортировки видны; цвет ссылок-папок читаем на тёмной теме; кнопка «На главную»; наследование тёмной темы.

## 1.07.0 (2026-07-31)

### Новое

- **Офлайн-пакет** (`prepare_offline`): скачивание Entware Manager + всех зависимостей через UI.
- **Makefile**: таргеты `test`/`lint`/`ci`, per-arch `tar.gz`, `all.tar.gz` без таймстемпа; авто-версия из git-тега.
- **ShellCheck clean** + GitHub Actions (shellcheck.yml + ci.yml).
- **ttyd**: `/bin/sh` → `/opt/bin/bash`, добавлен режим telnet; bash в Depends ipk и проверку install.sh.

## 1.06.6 (2026-07-29)

### Исправлено

- **ipk для aarch64**: маппинг arch `arm64`→`aarch64-3.10`, per-arch зависимости (`coreutils, bridge` вместо `coreutils-base, bridge-utils`). Симлинки `.cgi` теперь включаются в ipk (не удаляются при сборке).
- **install.sh**: при ipk-установке файлы уже на месте — проверка `SELF_DIR==TARGET_DIR` пропускает `cp -a` (чинит `go.cgi не найден` и `HTTP 404`).
- **entware.js**: `backup.cgi` и `backup_restore.cgi` — абсолютные пути `/entware-cgi/backup.cgi`, подставлено имя файла для скачивания.

## 1.06.5 (2026-07-29)

### Новое

- **arm64 (aarch64)**: добавлена поддержка роутеров на ARM64 — Keenetic Ultra 1812 и аналоги. Go-бинарники компилируются под `GOARCH=arm64`, создаются `ipk` и `tar.gz` для arm64. `build-deploy.sh` и `build-ipk.sh` обновлены.

### Новое

- **Встроенное обновление** — 3 новых CGI-эндпоинта в `entware-stats`:
  - `update_check.cgi` — GET: GitHub API → semver сравнение → JSON `{current, latest, has_update}`
  - `update_run.cgi` — POST: фоновая goroutine, download tar.gz → `archive/tar` + `compress/gzip` extract → exec `install.sh`
  - `update_status.cgi` — GET: tail лога `/tmp/entware/update.log` → JSON `{status, pid, lines}`
  - UI: секция «Обновление» в Настройках (текущая версия, кнопка Проверить, кнопка Обновить, лог с авто-polling)
- **install.sh**: сохраняет `.arch` при успешной установке (`echo "$ROUTER_ARCH" > "$TARGET_DIR/.arch"`) — для автоопределения архитектуры при обновлении

### Исправления

- **build-ipk.sh**: формат ipk изменён с `ar` на `tar.gz`. Entware на Keenetic использует gzip-архив (tar.gz) вместо ar — эталонный `geo-split-data_0.6.0_all.ipk` подтвердил это. Убрана проверка `command -v ar`, `control.tar.gz` с `./` префиксом, сборка через `tar -czf` вместо `ar qc`. Ошибка `Malformed package file` устранена.
- **tar.gz имена папок**: per-arch архивы (`entware-manager-arm64.tar.gz`) содержали `deploy-arm64/` внутри вместо `deploy/`, из-за чего `cd deploy && sh Install/install.sh` не работал. Исправлено — во всех архивах корневая папка `deploy/`.
- **Service watchdog не стартовал** — в `service_config.json` отсутствовало поле `"enabled"`. `jq -r '.enabled'` возвращал `"null"` (строка), проверка `[ "$ENABLED" = "true" ]` проваливалась. Исправлено: `service_watchdog.sh` — обработка `"null"` → `true`; Go POST-хендлер мержит с существующим конфигом (не теряет поля); GET-хендлер добавляет `"enabled": true` если отсутствует.
- **Монитор возвращал «Не удалось start демон»** — при повторном нажатии Пуск `watchdog.sh` выходил с exit 1 ("Already running"), Go-хендлер трактовал как ошибку. Исправлено: `action.go` — если вывод содержит "Already running", возвращаем `status: ok`.
- **network_stats.cgi → 404** — symlink отсутствовал в `build-deploy.sh` (был `network_status`, но не `network_stats`). Статистика сети на вкладке «Статистика системы» показывала «Ошибка загрузки сети». Исправлено: добавлен в список симлинков.

## 1.06.4 (2026-07-29)

### Исправления

- **Вкладки Процессы/Терминал**: упрощение — вместо проверки статуса ttyd через API и кнопки запуска теперь прямой iframe + ссылка «Открыть в новой вкладке» + подсказка «Если не открывается — запустите ttyd в Настройки → Терминал». Удалены `renderTtydTab()`, `checkTtydAndRender()`, `startTtydAndReload()`.
- **build-ipk.sh**: добавлена проверка `command -v ar` в начале сборки; убран битый fallback `tar -czf` (создавал tar.gz с именем `.ipk`, opkg отвергал как Malformed package file).
- **install.sh**: проверка успешности `mkdir -p "$LOG_DIR"` и создания `$LOG_FILE` после первого `log()`. В `log()` добавлен `|| true` чтобы скрипт не обрывался при проблемах с записью.
- **getRouterIP()** (`go/internal/stats/links.go`): определение LAN-адреса через `net.Dial("udp", REMOTE_ADDR+":80")` — ядро возвращает source-IP, которым роутер отвечает клиенту. Fallback: перебор интерфейсов с фильтром `isPrivateIP()` (только приватные диапазоны 10.x, 172.16-31.x, 192.168.x). Раньше возвращал IP первого попавшегося интерфейса (мог быть WAN).

## 1.06.3 (2026-07-28)

### Исправления

- **Вкладки Процессы/Терминал**: вместо прямого iframe (ошибка «Попытка соединения не удалась» при неработающем ttyd) теперь проверяется статус через API. Если ttyd не запущен — показывается сообщение и кнопка запуска. После запуска подгружается iframe.

## 1.06.2 (2026-07-28)

### Исправления

- **ttyd terminal**: замена `/opt/bin/bash` → `/bin/sh`. На Keenetic Viva 1910/KN-2311 (mipsel) OpenSSL 3.x несовместим со старым glibc — `ttyd -p 9089 /opt/bin/bash` выдавал `Error relocating libssl.so.3: pthread_cond_timedwait: symbol not found`. htop работал, т.к. не использует bash. `/bin/sh` (BusyBox ash) не тянет openssl.
- **build-deploy.sh**: добавлена фильтрация `*.ipk` — ipk-файлы больше не попадают в deploy/ при сборке.

## 1.06.1 (2026-07-28)

### Исправления

- **wifi-temp**: замена `localhost` → `127.0.0.1` в запросе RCI API. После обновления прошивки Keenetic (CVE-2026-42533 / NDM-4566) `localhost` перестал резолвиться, WiFi температура всегда показывала `null`.
- **null history guard**: saveWifiTempPoint и температура — проверка на null/-; index.html скрывает °C при отсутствии сенсора.
- **wget → curl**: install.sh и README.
- **arch detection**: armv5tel; mipsel endianness через opkg + ELF byte 5; обрезка суффикса `-*` (mipsel-3.4 → mipsel) в install.sh и go.cgi.
- **server.port**: grep игнорирует закомментированные строки (`#server.port`).
- **backup check**: `$BACKUP_DIR/etc` → `$BACKUP_DIR/opt/etc` (всегда писало «чистая установка» при повторной установке).
- **install.sh**: BusyBox-совместимость (ANSI через printf, od -b, hostname -I → ip, process substitution → mkfifo/pipe).
- **install.sh**: literal `\n` → реальные переносы строк в ERRORS/CHECK_ERRS (читаемый вывод ошибок).
- **install.sh**: подсказка `tail -f` для просмотра лога установки.
- **30-cgi.conf**: полная перезапись вместо sed.
- **entware.js:842**: `network_status.cgi` → `network_stats.cgi` (пустые интерфейсы/порты на dashboard).
- **build-deploy.sh**: сборка только для arm (GOARM=5), mips, mipsel; arm64/amd64/386 удалены.
- **footer**: подпись разработчика.

### Тестирование

- **mipsel (Keenetic Giga)**: установка успешно протестирована 2026-07-28 — все 9 шагов, 60 симлинков, 7 Go-бинарников, HTTP 200.
- **mipsel (Keenetic KN-2311)**: установка протестирована 2026-07-28 — подтверждена после правок arch suffix + server.port.

## 1.06.0 (2026-07-26)

### Мультиархитектурная поддержка

- **build-deploy.sh v3.0**: компиляция Go-бинарников для arm (GOARM=5), mips, mipsel; флаг `--arch=ARCH` для сборки под одну arch
- **go.cgi**: автоопределение архитектуры роутера (`uname -m`) и выбор соответствующих бинарников; fallback на старый плоский layout
- **install.sh**: определяет arch роутера, удаляет бинарники для чужих архитектур, копирует только нужные
- **Исправление**: root-level `.cgi` symlinks создавались с пробелами в имени (heredoc read)

## 1.05.2 (2026-07-25)

### Исправления

- **monitor endpoints**: JS вызывал `/monitor/monitor_status.cgi`, но symlink — `monitor/status.cgi` — все 4 эндпоинта (status, config, action, log) возвращали 404; исправлено на `/monitor/status.cgi` и т.д.
- **build-deploy.sh**: удалены корневые симлинки `monitor_*` (дублировали поддиректорию `monitor/`)
- **help page**: добавлены разделы про архитектуру (go.cgi + 7 Go-бинарников), attr_health (цветовая индикация Health), объединённые графики температуры

## 1.05.1 (2026-07-24)

### Улучшения графиков температуры

- **Масштаб**: холсты 900px с `width:100%`, padding 20px — без пустых пространств
- **Ось X**: отображается дата вместо времени; данные сохраняются с датой
- **Тултип**: подсветка точки на графике при наведении, дата + значение
- **Клик по статам**: Мин/Средняя/Макс/Сейчас — подсвечивается точка на графике
- **Линии**: 1.5px, высота 190px
- **Заголовки**: CPU/WiFi крупнее, с тенью
- **Отступы**: Wi-Fi строка в сайдбаре compact

## 1.05.0 (2026-07-24)

### Полная миграция shell → Go

Все shell CGI скрипты Entware Manager переписаны на Go. Проект больше не использует shell-зависимые CGI.

| Бинарь | Эндпоинты |
|---|---|
| `entware-pkg` | 8 — available, packages, install, remove, upgrade, update, upgradable, api |
| `entware-stats` | 11 — stats, version, help, links_load, links_save, tmpfs, view_file, delete_file, auth_config, crontab, crontab_update |
| `entware-net` | 8 — interfaces, routes, arp, status, stats, events, config, action |
| `entware-logger` | 9 — config, view, system_logs, system_log, find_by_name, rotate, clear, debug, debug_path |
| `entware-services` | 11 — services, service_action, ttyd_control, watchdog_status, watchdog_action, watchdog_config, watchdog_events, check_syntax, check_deps, debug |
| `entware-monitor` | 9 — status, action, config, log, temperature, wifi_temp, temp_history, wifi_temp_history, kill_pid |
| **`entware-smart`** | 6 — list, info, attributes, health, usage, selftest |

**Итого: 62 эндпоинта, 7 Go-бинарников, 0 shell CGI.**

### Улучшения

- **SMART**: `attr_health` — анализ критических атрибутов (5,10,187,196,197,198) с подсветкой колонки Health (зелёный/оранжевый/красный/серый)
- **SMART**: USB-флешки без SMART показывают `—` вместо `UNKNOWN`
- **SMART**: исправлено двойное экранирование `\n` в info, потеря вывода smartctl при exit code != 0
- **check_deps**: мигрирован на Go с полной совместимостью JSON-формата
- **check_syntax**: мигрирован на Go — обход .cgi/.sh файлов, `sh -n` через exec.Command
- **smartctl**: timeout увеличен до 30s, добавлен CombinedOutput (stderr), sudo fallback

### Сборка и деплой

- Все 7 бинарников собраны `GOARCH=arm64 CGO_ENABLED=0 -ldflags="-s -w"` и сжаты UPX -9
- Все shell-оригиналы сохранены: `web_entware/tmp/` на роутере, `tmp/` локально
- `common.sh`, `smart.sh` — shell-библиотеки не удалены (используются build-deploy.sh, backup.sh)

## 1.04.14 (2026-07-24)

### Новые Go-эндпоинты

- **entware-stats**: добавлен `tmpfs` — файловый менеджер tmpfs (`os.ReadDir` + `os.Stat` вместо `ls -lA | awk`/`ls -1A` + per-file `ls -ld`)
  - Все shell-зависимости удалены: `ls`, `awk`, `sed`, `jq`, `dirname`, `tr`, `while read`, `mkdir`, `date`
  - Владелец: группа: числовые UID/GID из `syscall.Stat_t`
  - Ширина вывода: 210 строк HTML (против 355 shell)
  - Защита от directory traversal через `filepath.Clean`
- **entware-stats**: добавлен `view_file` — просмотр файлов (JSON для XHR, HTML для браузера)
  - `os.ReadFile` + проверка на null-байты (вместо `od | tr | grep`)
  - Ограничение 1 MB, последние 1000 строк
  - Путь только `/tmp/*` и `/dev/shm/*`
- **entware-stats**: добавлен `delete_file` — POST-удаление файлов/папок (`os.Remove` вместо `rm / rmdir`)
  - Проверка пароля через `crypto/sha256` (вместо `sha256sum`/`openssl`)
  - Логирование в `/tmp/entware/logs/`
- **entware-stats**: добавлен `links_save` — POST-сохранение ссылок (`json.Valid` вместо `jq empty`)
- **entware-stats**: добавлен `auth_config` — GET/POST управление паролем файлового менеджера
- **entware-stats**: добавлены `crontab` + `crontab_update` — чтение/сохранение crontab (system/opt)
  - `exec.Command("crontab")` с stdin вместо temp-файла
  - `syscall.SIGHUP` для перезагрузки cron
- **entware-pkg**: добавлен `api` — информация о пакете через `opkg info`
- **entware-monitor**: добавлен `kill_pid` — принудительное завершение процесса (`os.FindProcess` + `os.Kill`)

### Новые Go-эндпоинты (шаг 2 — service_watchdog)

- **entware-services**: добавлены `watchdog_status`, `watchdog_action`, `watchdog_config`, `watchdog_events`
  - `status.cgi` — статус watchdog: running/PID, конфиг, PIDS-карта из `/proc`
  - `action.cgi` — start/stop/restart/update через `service_watchdog.sh` (exec.Command)
  - `config.cgi` — GET/POST конфиг; POST валидация JSON, ключи `enabled`, `interval`, `mode`, `watch_list`, `auto_restart`, `exclude_list`, `log_to_monitor`, `pid_history_days`
  - `events.cgi` — парсинг лога через `parseWatchdogLog` (аналог `tail -r | sed | awk | jq`)
  - Код: ~180 строк Go + 16 тестов (все PASS)
  - Пути вынесены в `var` для тестируемости
  - Вспомогательные: `cmdPrivate()` для exec.Command, `apiFetch.GET` для совместимости

### Исправления

- **entware.js** (service_watchdog action): `apiFetch` с POST без тела (lighttpd 411) → заменён на GET (как network.js)
- Врапперы CGI обновлены под новый код (4 файла, ~150 байт каждый)

### Сборка и деплой

- `entware-services`: пересобран, UPX -9 (1.9M → 702K), задеплоен на роутер
- Новые CGI-врапперы загружены: `service_watchdog/{status,action,config,events}.cgi`
- Оригиналы shell CGI сохранены: `web_entware/tmp/service_watchdog/` (SMB)

### Новые Go-эндпоинты (шаг 3 — check_syntax + check_deps)

- **entware-services**: добавлены `check_syntax`, `check_deps`
  - `check_syntax.cgi` — проверка синтаксиса sh всех .cgi/.sh файлов через `sh -n` (exec.Command)
  - `check_deps.cgi` — проверка системных зависимостей: sed/awk/grep/ps (LookPath), opkg (--version), lighttpd/cron PID, jq, ip, smartctl
  - Формат JSON `check_deps` полностью совместим с shell-версией (ожидается `entware.js`)
  - Код: ~170 строк Go + 8 тестов (все PASS)
  - Типы: `DepsResult`, `DepsBase`, `DepsDeps`, `DepsSections`, `SyntaxResult`, `SyntaxFile`

### Исправления

- **cgi-bin/check_deps.cgi**, **cgi-bin/check_syntax.cgi**: переписаны как 3-строчные Go-врапперы

### Сборка и деплой

- `entware-services`: пересобран, UPX -9 (2.0M → 739K), задеплоен на роутер
- Оригиналы shell CGI сохранены: `tmp/check_deps.cgi`, `tmp/check_syntax.cgi` + `web_entware/tmp/` (SMB)

### Новые Go-эндпоинты (шаг 4 — SMART)

- **entware-smart**: новый Go-бинарник (6 эндпоинтов, 14 тестов)
  - `list` — обнаружение дисков через `/proc/partitions`, SMART-данные через smartctl
  - `info` — `smartctl -i`, вывод информации о диске
  - `attributes` — `smartctl -A`, парсинг таблицы атрибутов (22 аттрибута)
  - `health` — `smartctl -H`, статус здоровья
  - `usage` — `df -h`, разбивка по разделам
  - `selftest` (GET) — статус самотеста; (POST) — запуск теста (short/long/conveyance)
  - Типы: `DiskInfo`, `AttrInfo`, `PartitionInfo`
  - Утилиты: `smartctlRun` (sudo fallback), `discoverDisks`, `detectType`, `parseIntOrNull`
  - Пути вынесены в `var` для тестируемости

### Исправления SMART

- **info**: убрано двойное экранирование `\n` (escapeJSON → json.Marshal) — переводы строк корректны
- **list**: `smartctl -a` с exit code != 0 не отбрасывал вывод — модель/серийный/health извлекаются из info-секции

### Улучшения SMART

- **attr_health**: новый флаг в JSON диска — парсинг критических атрибутов (ID 5, 10, 187, 196, 197, 198)
  - `ok` — всё хорошо
  - `warning` — значение близко к порогу (разница < 10)
  - `critical` — порог превышен или health != PASSED
  - `inactive` — USB-флешка без SMART, отображается как `—` (серый, без бейджа)
- **UI**: колонка Health теперь зелёная (ok), оранжевая (warning) или красная (critical)
  - Иконка `icon-alert` добавлена в `icons.svg`

### Сборка и деплой

- **entware-smart**: новый бинарник, UPX -9 (1.9M → 695K), задеплоен на роутер
- `cgi-bin/smart.cgi`: переписан как 2-строчный Go-враппер
- Оригинал сохранён: `tmp/smart.cgi`

### Debug CGI (3 файла, text/plain)

- **entware-logger**: `logger_debug`, `logger_debug_path`
  - `debug.cgi` — проверка `/opt/var/log/entware/system.log` (существует/размер/ls)
  - `debug_path.cgi` — PATH, LookPath(cat/sed), последние 50 строк лога
- **entware-services**: `debug`
  - `debug.cgi` — REQUEST_METHOD/CONTENT_LENGTH, POST body, sanitize_alnum
  - Оригиналы сохранены: `tmp/{debug,logger_debug,logger_debug_path}.cgi`

### Новые Go-эндпоинты (шаг 1 — network)

- **entware-net**: добавлены `network_events`, `network_config`, `network_action`
  - `events.cgi` — парсинг лога `/tmp/entware/logs/YYYY-MM-DD.log`, фильтр по `[network]`
  - `config.cgi` — GET (чтение/дефолт `network_config.json`) + POST (валидация JSON, запись)
  - `action.cgi` — start/stop/restart через `network_watchdog.sh` (exec.Command)
  - Шелл-зависимости удалены: `tail`, `grep`, `sed`, `cut`, `tr`, `awk`, `jq`, `cat`, `date`
  - Код: ~220 строк Go + 3 тестовых файла (17 тестов)
  - Пути вынесены в `var` (пакетные переменные) для тестируемости
  - `shared.go`: +`IsPOST()`, +`GetParam(key)`, +`parseFormBody` + `urlDecode`

### Исправления

- **network/action.cgi**: убрана проверка метода (`!IsGET()`) — фронтенд шлёт POST с `action` в query string; `GetParam()` читает QUERY_STRING корректно
- **network.js**: `apiFetch` с `method: 'POST'` без тела не работал — lighttpd 1.4.82 требует `Content-Length` для POST. Изменён вызов на GET (action уже в URL)
- **service_watchdog/action.cgi** (shell): аналогичная проблема — `apiFetch` с POST без тела — будет исправлен при миграции

### Сборка и деплой

- Все 6 Go-бинарников пересобраны и сжаты UPX -9:
  - `entware-pkg`: 754K
  - `entware-stats`: 808K
  - `entware-net`: 747K (был 2.1M без UPX)
  - `entware-logger`: 736K
  - `entware-services`: 692K
  - `entware-monitor`: 1.7M
- Оригиналы shell CGI сохранены: `tmp/network/` (локально), `web_entware/tmp/network/` (SMB), `/tmp/entware_backup_cgi/network/` (роутер)

## 1.04.12 (2026-07-21)

### Новые Go-эндпоинты

- **entware-stats** (main.go): добавлены `version`, `help`, `links_load`
  - `version.cgi` — читает `/opt/web_entware/version.json` напрямую
  - `help.cgi` — HTML-страница справки через `//go:embed help.html`
  - `links_load.cgi` — читает `links.json`, fallback на дефолтные ссылки, определение IP через `net.InterfaceAddrs()`
- **entware-monitor** (main.go): добавлены `temperature`, `wifi_temp`, `temp_history`, `wifi_temp_history`
  - `temperature.cgi` — читает `/sys/class/thermal/` напрямую, без `cat`/`sed`
  - `wifi_temp.cgi` — HTTP-запрос к Keenetic API через `net/http` вместо `wget`+`jq`
  - `temp_history.cgi` / `wifi_temp_history.cgi` — чтение/запись истории через Go (glob, os.ReadFile, json.Marshal)
- `shared.go`: добавлен `GetParam(key)` — читает `QUERY_STRING` + POST body

### Снятые shell-зависимости

- Удалены вызовы: `cat`, `sed`, `wget`, `jq`, `grep`, `cut`, `tr`, `head`, `hostname`, `awk`, `date`, `find`, `ls`, `rm`, `mkdir` из этих 7 CGI
- Общее сокращение: ~7 shell-подпроцессов на каждый вызов температуры (×5760 раз/день = ~40 000 подпроцессов/день)

## 1.04.11 (2026-07-19)

### Исправления

- **Auth fail-open** (`lib/common.sh`): `check_filemgr_auth()` — `return 0` → `return 1` при отсутствии sha256sum/openssl
- **monitor_action** (`go/internal/monitor/action.go`): добавлена проверка `err` от `cmd.Run()` при start/restart — если `watchdog.sh` завершился с ошибкой, возвращается `"Не удалось start демон"` вместо `"Демон не запустился"`
- **monitor_status** (`go/internal/monitor/status.go`): исправлен json tag `demon_status` → `daemon_status` (JS ждёт `data.daemon_status`)
- **SPEC.md** (go/SPEC.md): обновлены секции 9-11 (текущая архитектура, 6 бинарников, сборка)
- **CHANGELOG.md**: обновлена таблица итогов в 1.04.06

## 1.04.10 (2026-07-19)

### Go migration — monitor/*.cgi → entware-monitor

- **4 CGI** (215 строк) модуля защиты переписаны на Go: `entware-monitor` (735KB UPX)
- **go/cmd/entware-monitor/** + **go/internal/monitor/**
- **monitor\_status**: PID файл + топ-5 процессов через прямое чтение `/proc/[pid]/stat` (вместо `top -bn1 | sed | head | awk` — 4 fork → 0)
- **monitor\_action**: start/stop/restart (`watchdog.sh`), kill (`SIGKILL`), clearlog
- **monitor\_config**: GET/POST JSON, авто-миграция `max_processes`, SIGHUP демону
- **monitor\_log**: grep `[monitor]` из дневного лога + tail 200 → text/plain
- **Оригиналы**: `tmp/monitor_*.cgi.original` + SMB

### Исправления

- **monitor_action**: добавлена проверка `err` от `cmd.Run()` при start/restart — если `watchdog.sh` завершился с ошибкой, возвращается `"Не удалось start демон"` вместо `"Демон не запустился"`

## 1.04.09 (2026-07-19)

### Go migration — services.cgi + service_action.cgi → entware-services

- **services.cgi** (144 строк) + **service_action.cgi** (112 строк) + **ttyd_control.cgi** (134 строк) переписаны на Go: `entware-services` (708KB UPX)
- **go/cmd/entware-services/** + **go/internal/services/** — dispatch по ENDPOINT
- **services** (`ENDPOINT=services`): чтение `/opt/etc/init.d/S*`/`K*`, поиск PID:
  1. PIDFILE из скрипта
  2. Стандартные pid-файлы (`/tmp/name.pid`, `/var/run/name.pid`, `/opt/var/run/name.pid`)
  3. PROCS/NAME/DAEMON из скрипта → поиск в `/proc/[pid]/cmdline`
  4. По базовому имени (без цифр)
  5. По полному имени (S99name)
  6. По .py файлу из SCRIPT
- **Вместо `ps | grep`**: однократное сканирование `/proc` (все PID, cmdline, status)
- **service_action** (`ENDPOINT=service_action`): start/stop/restart через `exec.Command`, enable/disable через `os.Rename`
- **ttyd_control**: GET → статус ttyd (8089/9089) через `/proc`, POST → start/stop/restart через `exec.Command` + background
- **Оригиналы**: `tmp/services.cgi.original` + `tmp/service_action.cgi.original` + `tmp/ttyd_control.cgi.original` + SMB

- **7 CGI** (385 строк) модуля логирования переписаны на Go: `entware-logger` (737KB UPX)
- **Бинарь**: `go/cmd/entware-logger/`, пакет `go/internal/logger/`
- **ENDPOINTы**: `logger_config`, `logger_view`, `logger_system_logs`, `logger_system_log`, `logger_find_by_name`, `logger_rotate`, `logger_clear`
- **JSON**: `config.cgi` (GET/POST), `find_by_name.cgi`, `rotate.cgi` (POST), `clear.cgi` (POST)
- **HTML**: `view.cgi` (фильтр awk → bufio.Scanner), `system_logs.cgi`, `system_log.cgi`, `config.cgi?pretty`
- **Зависимости**: удалены jq, sed, awk — JSON парсится напрямую, файлы читаются через os.ReadFile/bufio
- **Оригиналы**: `tmp/*.cgi.original` + SMB `web_entware/tmp/`
- **Оставлены в shell**: `debug.cgi`, `debug_path.cgi` (отладочные)

### Итог по Go-миграции

| Бинарь | ENDPOINTы | Размер (UPX) |
|--------|-----------|-------------|
| `entware-net` | `network_interfaces`, `network_routes`, `network_arp`, `network_status`, `network_stats` | 739KB |
| `entware-pkg` | `available`, `packages`, `install`, `remove`, `upgrade`, `update`, `upgradable` | 765KB |
| `entware-stats` | `stats` | 630KB |
| `entware-logger` | `logger_config`, `logger_view`, `logger_system_logs`, `logger_system_log`, `logger_find_by_name`, `logger_rotate`, `logger_clear` | 737KB |

### Улучшение модалок температуры

- **Графики**: hover tooltip с точным значением + время, оси `#a0aec0` (вместо невидимого `#4a5568`)
- **WiFi-график**: добавлены подписи времени по оси X

## 1.04.07 (2026-07-18)

### Go migration — network_status.cgi → entware-net

- **network_status.cgi** (291 строк shell) — карточка статистики сети для sidebar — переписан на Go
- **go/internal/network/network_stats.go** — новый хендлер `HandleNetworkStats()`
- **ENDPOINT** `network_stats` — добавлен в dispatch `entware-net`
- **Прямое чтение**: `/proc/net/dev` (трафик), `/sys/class/net/*/carrier` + `speed` (порты)
- **exec**: `ip -4 addr show`, `brctl show`, `ip link show`, `ip route show default`
- **Fork/exec**: 10+ → 5 вызовов
- **Оригинал**: `tmp/network_status.cgi.original` + SMB `web_entware/tmp/`
- **Бинарь**: UPX 2.1MB → 739KB (36%)
- **Итого entware-net**: 5 эндпоинтов (interfaces, routes, arp, status, stats)

### Улучшение модалок температуры

- **Текст крупнее и ярче**: новый CSS-класс `.temp-stat .value` → `1.6rem` (было `1.15rem`), цвет `var(--accent)`
- **Hover-эффект**: `scale(1.15)` + `text-shadow(glow)` + смена на белый
- **WiFi-модалка**: добавлены текущие значения температуры WiFi0°C / WiFi1°C (последние точки из истории)

## 1.04.06 (2026-07-18)

### Go migration — stats.cgi → entware-stats

- **stats.cgi** (273 строк shell) переписан на Go: `entware-stats` (630KB UPX)
- **go/internal/stats/stats.go** — сбор данных из /proc (meminfo, uptime, model, /proc/*/status) + вызовы df/opkg
- **Время**: 400ms → **132ms** (3× быстрее)
- **Fork/exec**: 10+ вызовов → 3 (только opkg×2 + df×1)
- **Секции**: Система, Память (RAM) + топ-процессы, Пакеты Entware + изменения, Диск (/opt), tmpfs, Блочные устройства, Сеть (lazy JS)
- **Оригинал**: `tmp/stats.cgi.original` + SMB `web_entware/tmp/`

### Go migration — network/status.cgi → entware-net

- **network/status.cgi** (56 строк shell) переписан на Go: добавлена `HandleStatus()` в `go/internal/network/status.go`
- **ENDPOINT** `network_status` — dispatch в существующий `entware-net`
- **Зависимости**: удалены jq, cut, awk — JSON и `/proc/[pid]/stat` парсятся напрямую
- **Оригинал**: `tmp/status.cgi.original` + SMB `web_entware/tmp/`

### Итог по Go-миграции

| Бинарь | ENDPOINTы | Размер (UPX) |
|--------|-----------|:------------:|
| `entware-pkg` | `available`, `packages`, `install`, `remove`, `upgrade`, `update`, `upgradable` | 765KB |
| `entware-stats` | `stats` | 630KB |
| `entware-net` | `network_interfaces`, `network_routes`, `network_arp`, `network_status`, `network_stats` | 739KB |

Все 12 эндпоинтов проверены и работают.

## 1.04.05 (2026-07-18)

### Go migration — все пакетные CGIs → entware-pkg

- **7 CGIs** заменены на единый Go-бинарник `entware-pkg` (765KB UPX), dispatch по `ENDPOINT=имя_файла.cgi`:
  - `available.cgi` — `opkg list` → JSON (2.7× быстрее: 500ms → 184ms)
  - `packages.cgi` — `opkg list-installed` → HTML (12.5× быстрее: 600ms → 48ms)
  - `install.cgi`, `remove.cgi`, `upgrade.cgi` — POST → HTML
  - `update.cgi` — `opkg update` → HTML
  - `upgradable.cgi` — `opkg list-upgradable` → JSON (уже был)
- **Новый пакет** `go/internal/packages/` — shared.go + 7 handler'ов
- **Удалён** старый `go/internal/upgradable/` (код перенесён в packages)
- **Оригиналы** сохранены в `tmp/` проекта и в SMB `web_entware/tmp/`

### Пакетный лог — перенос в постоянное хранилище

- **PKG_LOG** перенесён из `/tmp/entware/logs/` (tmpfs) в `/opt/var/log/package_changes.log`
- **stats.cgi** сам создаёт `mkdir -p + touch` при загрузке — секция «Последние изменения» всегда видна
- **upgrade_all** теперь логируется в `package_changes.log` (раньше нет)

### Инфраструктура

- **build-deploy.sh**: chmod +x для `cgi-bin/go/*`
- **backup.sh**: `cgi-bin/go/entware-*` в проверке ключевых файлов и chmod в restore
- **README.md**, **go/SPEC.md**: SMB-учётка → плейсхолдер `USER%PASS`

## 1.04.04 (2026-07-17)

### SMART — кликабельные зоны вместо кнопок

- **кнопки удалены**: строка → Атрибуты, Health-бейдж → Health modal, температура → Тест-диалог (делегированный click на tbody)
- **цветные типы дисков**: HDD — синий, SSD — зелёный, NVMe — фиолетовый
- **подсветка строки**: левая граница (border-left) — зелёная при PASSED, красная при FAILED
- **подсказки атрибутов**: клик на имя атрибута → Toast с описанием (19 атрибутов, 4000ms)
- **Escape → Close**: глобальный keydown listener в Modal.init()

## 1.04.03 (2026-07-17)

### Интерфейс

- **modal-header sticky**: заголовок модального окна зафиксирован (`position: sticky; top: 0; background: var(--modal-bg)`) — не скроллится вместе с содержимым
- **зазор между заголовком и верхом модалки**: `padding-top` перенесён с `.modal-content` на `.modal-header` — заголовок начинается от верхнего края модального окна
- **modal-header прозрачный зазор**: `margin-bottom` заменён на `padding-bottom` — зазор под бордером теперь внутри блока, скроллящийся контент (таблица атрибутов) не виден сквозь него
- **th внутри модалок**: отключены `position: sticky` и `backdrop-filter` у `<th>` в модальных окнах — убран конфликт с sticky-заголовком и артефакты композитинга
- **меню десктопа**: `.menu-container` получил `overflow-y: auto` + кастомный тонкий скроллбар (8px) — нижние пункты (Сеть, SMART, Настройки, Защита, Справка, Логи) теперь доступны без схлопывания сайдбара

## 1.04.01 (2026-07-17)

### Унификация модального окна

- **modal scrollbar**: приведён к единому стилю с `.content` — `scrollbar-width: thin`, WebKit 8px, `border-radius: 4px`, track прозрачный (не выпирает за `border-radius: 32px`)
- **modal padding**: `24px` единый (убрано `24px 0 24px 24px` + workaround `#modalBody padding-right`)
- **.close**: убран `float: right` (родитель использует `display: flex; justify-content: space-between`)
- **кастомный scrollbar 6px**: удалён (был с `transparent` треком — выбивался из дизайн-системы)

### Инфраструктура

- **RULES.md**: правила для LLM-ассистента выделены из devlog.md, таблица функций актуализирована
- **devlog.md → CHANGELOG.md**: devlog.md удалён, содержимое перенесено в CHANGELOG (добавлены версии 1.03.21-1.03.25)

## 1.04.00 (2026-07-17)

### SMART-модуль мониторинга дисков

- **lib/smart.sh** — новая библиотека для работы со SMART:
  - `smart_discover_disks()` — обнаружение дисков через `/proc/partitions` (BusyBox)
  - `smart_disk_json()` — парсинг `smartctl -a` в JSON (модель, серийник, размер, health, температура, power-on)
  - `smart_attributes_json()` — парсинг `smartctl -A` в JSON-массив атрибутов
  - `smart_health_json()`, `smart_info_json()` — health и базовая информация
  - `smart_test_start()`, `smart_test_status()` — запуск и мониторинг самотестов
  - `smartctl_run()` — вызов `smartctl` через `sudo` с таймаутом

- **cgi-bin/smart.cgi** — REST API по `action=list|info|attributes|health|selftest`

- **smart.js** — UI-таб SMART:
  - Таблица дисков (устройство, модель, серийник, размер, тип, health, температура, power-on, действия)
  - Модалка атрибутов (цветовая индикация: value ≤ threshold = красный)
  - Модалка health и запуск самотестов (short/long/conveyance) через POST + поллинг
  - Поиск по таблице
  - Унифицирован как `const SMART = { init(), stopUpdates(), ... }`

- **icons.svg** — добавлена иконка `#icon-hdd` (диск)

- **menu/menu.json** — пункт `{ "tab": "smart", "icon": "hdd", "text": "SMART" }` после "Сеть"

- **lib/utils.js** — добавлена `loadScript(src)` для динамической загрузки JS-модулей

### Инфраструктура

- **build-deploy.sh** — копирует `lib/*.sh` (нужно для `lib/smart.sh` на роутере)
- **Install/install.sh** — в `PACKAGES` добавлены `sudo`, `smartmontools`, `smartmontools-drivedb`; создаётся `/opt/etc/sudoers.d/entware-smartctl` (nobody → smartctl без пароля)

## 1.03.12 (2026-07-07)

### Стандартизация путей и исправление lighttpd

- **install.sh полностью переписан**:
  - Безопасное дополнение конфига lighttpd — `server.modules +=`, `alias.url +=`, `static-file.exclude-extensions += .cgi` (с проверкой через `grep -q`, дубли не создаются)
  - Патч `/opt/etc/lighttpd/conf.d/30-cgi.conf` — `cgi.assign` и `cgi.execute-x-only = "enable"` устанавливаются только здесь
  - Удалён код, удалявший чужие `alias.url`, `mod_alias`, `mod_cgi`
  - Не удаляет чужие настройки

- **Исправление lighttpd**:
  - `cgi.assign` удалён из `main.conf` (дубль валил lighttpd)
  - `static-file.exclude-extensions` содержит `.cgi`
  - `mod_alias`/`mod_cgi` добавляются через `+=` без дублирования

- **Стандартизированы пути директорий**:
  - `logs/` — `/tmp/entware/logs/`
  - `pid/` — `/tmp/entware/pid/`
  - `counters/` — `/tmp/entware/counters/`
  - `counters_ignore/` — `/tmp/entware/counters_ignore/`
  - `temp_history/` — `/tmp/entware/temp_history/`

- **Все watchdog-скрипты**:
  - `mkdir -p` для всех стандартизированных директорий
  - `nohup` заменён на `sh ... &` (BusyBox на Keenetic не содержит `nohup`)

- **Исправление stats.cgi**:
  - `ps -e -o rss,comm` заменён на чтение `/proc/[pid]/status` (VmRSS) — BusyBox `ps` не поддерживает `-e -o`
  - stats.cgi теперь работает на Keenetic

### Обновлённые файлы

| Файл | Описание |
|------|----------|
| `install.sh` | Полностью переписан: безопасное дополнение конфига, патч 30-cgi.conf |
| `network_watchdog.sh` | `nohup` → `sh &`, `mkdir -p` для стандартных путей |
| `service_watchdog.sh` | `nohup` → `sh &`, `mkdir -p` для стандартных путей |
| `watchdog.sh` | `mkdir -p` для стандартных путей |
| `cgi-bin/stats.cgi` | Чтение VmRSS из /proc/[pid]/status вместо ps |
| `version.json` | 1.03.12 |

---

## 1.03.14 (2026-07-16)

### Исправление Content-Type в CGI

- **links_load.cgi**: `cat` → `json_out()` — добавлен HTTP-заголовок `Content-Type: application/json`, браузер корректно распознаёт JSON
- **api.cgi:43**: `echo ... | jq` → `json_out()` — исправлен fallthrough без заголовка при успешном ответе

### Правила для LLM-ассистента

- **devlog.md**: обновлены правила до v2.1:
  - Разделение `set -eu` для CGI (запрещён) и демонов/утилит (обязателен)
  - Таблица статусов всех функций из `common.sh` (✅/❌)
  - Добавлены пункты 12 (единая обработка ошибок CGI) и 13 (единый Content-Type)

### Обновлённые файлы

| Файл | Описание |
|------|----------|
| `cgi-bin/links_load.cgi` | cat → json_out() |
| `cgi-bin/api.cgi` | echo...jq → json_out() |
| `devlog.md` | Правила v2.1 |
| `version.json` | 1.03.14 |

---

## 1.03.18 (2026-07-16)

### BUGFIX: post_param() терял POST-данные в subshell

- **Критический баг**: `post_param()` кэшировал POST-данные через `_POST_CACHED`, но при вызове через `$(post_param ...)` каждый вызов создаёт новый subshell, где кэш теряется. После первого `post_param` stdin пустел, все последующие возвращали пустоту.
- **Исправление**: `post_param()` переведён на внешнюю переменную `$_POST_BODY`. CGIs с POST должны читать stdin один раз: `_POST_BODY=$(cat); export _POST_BODY`
- **Затронуты**: `ttyd_control.cgi` (починена работа ttyd), `crontab_update.cgi`, `monitor_action.cgi`, `service_action.cgi`, `kill_pid.cgi`
- **`ttyd_control.cgi`**: убран двойной `url_decode` (post_param уже декодирует)
- **`kill_pid.cgi`**: исправлен мёртвый код (GET-парсинг внутри POST-блока)
- **grep**: `\+` → `*` (BusyBox-совместимость)

### Обновлённые файлы

| Файл | Описание |
|------|----------|
| `lib/common.sh` | post_param() → $_POST_BODY |
| `cgi-bin/ttyd_control.cgi` | _POST_BODY, убран url_decode, grep \+ → * |
| `cgi-bin/crontab_update.cgi` | _POST_BODY, убран url_decode |
| `cgi-bin/monitor/monitor_action.cgi` | _POST_BODY, убран url_decode |
| `cgi-bin/service_action.cgi` | _POST_BODY |
| `cgi-bin/kill_pid.cgi` | _POST_BODY + исправлен GET/POST |
| `version.json` | 1.03.18 |

---

## 1.03.25 (2026-07-16)

### #19–23 — JS-рефакторинг

- **lib/utils.js**: добавлены константы `API_BASE` (`/entware-cgi`), `UI_BASE` (`/entware-manager`), `ICONS`. Функция `initTableSearch(inputId, tableId, cellIndex)`.
- **cgi-bin/monitor/monitor_status.cgi**: `demon_*` → `daemon_*` (опечатка)
- **monitor.js**: `demon` → `daemon`, все fetch через `API_BASE`, удалён хардкод `log_file/log_max_size` из saveConfig
- **network.js**: удалён `escapeHtml()` wrapper (делегировал глобальной), все fetch через `API_BASE`
- **entware.js**: все fetch через `API_BASE`, дубликаты `initPackagesSearch`/`renderAvailableTable` заменены на `initTableSearch()` (+40 строк → +10)
- **menu/menu.js**: fetch через `UI_BASE`

### Обновлённые файлы

| Файл | Описание |
|------|----------|
| `lib/utils.js` | API_BASE, UI_BASE, ICONS, initTableSearch |
| `monitor.js` | fetch через API_BASE, demon → daemon |
| `network.js` | fetch через API_BASE, escapeHtml wrapper удалён |
| `entware.js` | fetch через API_BASE, initTableSearch |
| `menu/menu.js` | fetch через UI_BASE |
| `cgi-bin/monitor/monitor_status.cgi` | demon → daemon |
| `version.json` | 1.03.25 |

---

## 1.03.23 (2026-07-16)

### parse_log_events() — унифицированный парсинг событий

- **lib/common.sh**: добавлена функция `parse_log_events(tag, limit)` — читает дневной лог, фильтрует по тегу, парсит строки в JSON-массив событий (timestamp, level, service, event, details). JSON-экранирование через sed.
- **network/events.cgi** и **service_watchdog/events.cgi**: сокращены с ~60 строк до 10 строк (один вызов `parse_log_events`)

### Обновлённые файлы

| Файл | Описание |
|------|----------|
| `lib/common.sh` | parse_log_events() |
| `cgi-bin/network/events.cgi` | Упрощён до 3 строк |
| `cgi-bin/service_watchdog/events.cgi` | Упрощён до 3 строк |
| `version.json` | 1.03.23 |

---

## 1.03.22 (2026-07-16)

### #9 fix — undefined log(), log-viewer CGIs

- **watchdog.sh**: `log "INFO" "Лог ротирован"` → `log_message "INFO" "[monitor] Лог ротирован"` (вызывал "not found" при ротации)
- **monitor_log.cgi**: теперь читает из `/tmp/entware/logs/YYYY-MM-DD.log`, фильтр `[monitor]`
- **network/events.cgi**, **service_watchdog/events.cgi**: переведены на дневной лог, grep -i, регистронезависимый парсинг

### Обновлённые файлы

| Файл | Описание |
|------|----------|
| `watchdog.sh` | log → log_message |
| `cgi-bin/monitor/monitor_log.cgi` | Чтение из дневного лога |
| `cgi-bin/network/events.cgi` | grep -i, дневной лог |
| `cgi-bin/service_watchdog/events.cgi` | grep -i, дневной лог |
| `version.json` | 1.03.22 |

---

## 1.03.21 (2026-07-16)

### Унификация логирования — log_service/log_event → log_message()

- **watchdog.sh**: `log()` → `log_message()` (локальная функция удалена)
- **network_watchdog.sh**: `log_event()` → `log_message()` (локальная функция удалена)
- **service_watchdog.sh**: `log_service()` → `log_message()` (локальная функция удалена)
- **lib/common.sh**: `log_action()` fallback теперь делегирует `log_message()` вместо дублирования mkdir/echo
- Все `log_action()` в start/stop/restart всех 3 демонов → `log_message()`
- Формат сообщений: `[модуль] подсистема: событие (детали)`
- Все пишут в `/tmp/entware/logs/YYYY-MM-DD.log` через единый `log_message()`

### Обновлённые файлы

| Файл | Описание |
|------|----------|
| `lib/common.sh` | log_action fallback → log_message |
| `watchdog.sh` | log → log_message |
| `network_watchdog.sh` | log_event → log_message |
| `service_watchdog.sh` | log_service → log_message |
| `version.json` | 1.03.21 |

---

## 1.03.17 (2026-07-16)

- **#3**: Удалён дубликат `get_wifi_status()` в `network_status.cgi` (строки 282–288)
- **#12**: `printf '%b'` → `url_decode()` из common.sh в 13 местах (tmpfs.cgi, ttyd_control.cgi, monitor_action.cgi, crontab.cgi/update.cgi, logger/*.cgi)
- **#13**: Ручной POST-парсинг (`cat | sed`) → `post_param()` в 5 CGI (crontab_update.cgi, ttyd_control.cgi, monitor_action.cgi, service_action.cgi, kill_pid.cgi)
- **#14**: `crontab.cgi` подключён к common.sh (добавлен `. /opt/web_entware/lib/common.sh`, использует `get_param`/`json_out`)
- **#16**: Ручные `echo "Content-type:..."` заменены на `json_out()` / `html_header()` в 17 CGI

### Обновлённые файлы

| Файл | Описание |
|------|----------|
| `cgi-bin/network_status.cgi` | #3: удалён дубликат |
| 8 файлов | #12: printf '%b' → url_decode |
| 5 файлов | #13: POST_DATA → post_param |
| `cgi-bin/crontab.cgi` | #14: подключен common.sh |
| 17 файлов | #16: ручной Content-Type → json_out/html_header |
| `version.json` | 1.03.17 |

---

## 1.03.16 (2026-07-16)

- **entware.js**: удалена `escapeHTML()` — не экранировала `'`, создавала XSS-уязвимость. Все вызовы переведены на `escapeHtml()` из `lib/utils.js`
- **network.js**: `this.escapeHtml()` теперь делегирует глобальной `escapeHtml()` из `lib/utils.js`

### Обновлённые файлы

| Файл | Описание |
|------|----------|
| `entware.js` | Удалена escapeHTML(), escapeHTML(...) → escapeHtml(...) |
| `network.js` | this.escapeHtml() → делегат на глобальную escapeHtml() |
| `version.json` | 1.03.16 |

---

## 1.03.15 (2026-07-16)

### Удаление хардкода IP 192.168.3.1

- **entware.js**: добавлена константа `BASE_URL = window.location.protocol + '//' + window.location.hostname`; все жёсткие IP заменены на `BASE_URL`; `DEFAULT_LINKS` → функция `getDefaultLinks()`
- **links_load.cgi**: IP определяется через `hostname -I`, fallback 192.168.3.1
- **install.sh, backup.sh**: сообщения с IP генерируются динамически
- **TECH_SPEC.md, Install/Install.txt**: IP заменён на плейсхолдер `<IP_роутера>`

### Обновлённые файлы

| Файл | Описание |
|------|----------|
| `entware.js` | BASE_URL, getDefaultLinks() вместо DEFAULT_LINKS |
| `cgi-bin/links_load.cgi` | ROUTER_IP динамический |
| `Install/install.sh` | IP через hostname -I |
| `backup.sh` | IP через hostname -I |
| `TECH_SPEC.md` | IP → `<IP_роутера>` |
| `Install/Install.txt` | IP → `<IP_роутера>` |
| `version.json` | 1.03.15 |

---

## 1.03.14 (2026-07-16)

### Исправление Content-Type в CGI

- **links_load.cgi**: `cat` → `json_out()` — добавлен HTTP-заголовок `Content-Type: application/json`, браузер корректно распознаёт JSON
- **api.cgi:43**: `echo ... | jq` → `json_out()` — исправлен fallthrough без заголовка при успешном ответе

### Правила для LLM-ассистента

- **devlog.md**: обновлены правила до v2.1:
  - Разделение `set -eu` для CGI (запрещён) и демонов/утилит (обязателен)
  - Таблица статусов всех функций из `common.sh` (✅/❌)
  - Добавлены пункты 12 (единая обработка ошибок CGI) и 13 (единый Content-Type)

### Обновлённые файлы

| Файл | Описание |
|------|----------|
| `cgi-bin/links_load.cgi` | cat → json_out() |
| `cgi-bin/api.cgi` | echo...jq → json_out() |
| `devlog.md` | Правила v2.1 |
| `version.json` | 1.03.14 |

---

## 1.03.13 (2026-07-07)

### Исправление демона защиты от зависших процессов

- **watchdog.sh полностью переписан**:
  - Добавлен единый интерфейс `start|stop|restart|status|daemon` (как у network_watchdog/service_watchdog)
  - `start` читает конфиг ДО запуска — если `enabled: false`, отказывает с ошибкой, не создаёт PID-файл
  - `daemon_loop` убрана проверка `ENABLED` — демон больше не выходит сам после старта
  - PID-файл создаётся в `start`, а не внутри `daemon_loop`
  - Использованы `pid_is_alive()` и `find_pids()` из common.sh

- **monitor_action.cgi**:
  - Вместо `$DEMON_SCRIPT >> log &` вызывает `watchdog.sh start`/`stop`
  - После `start` делает `sleep 1` и проверяет `watchdog.sh status`
  - `kill -0` → `pid_is_alive()`

- **monitor_status.cgi**:
  - `-d "/proc/$pid"` → `pid_is_alive()` (зомби-процессы больше не показывают running)

### Обновлённые файлы

| Файл | Описание |
|------|----------|
| `watchdog.sh` | Полностью переписан: интерфейс start/stop/restart/status |
| `cgi-bin/monitor/monitor_action.cgi` | Использует watchdog.sh start/stop |
| `cgi-bin/monitor/monitor_status.cgi` | pid_is_alive вместо -d /proc/$pid |
| `lib/common.sh` | Добавлены pid_is_alive(), find_pids() |
| `version.json` | 1.03.13 |

---

## 1.03.10 (2026-06-02)

### Новые функции

- **Убийство процессов и отображение дублей PID в службах**:
  - `cgi-bin/services.cgi v3.8` — полностью переписан поиск PID, теперь возвращает все PID процесса (поле `pids`). При нескольких процессах у службы показывается первый PID + кликабельный бейдж `+N`
  - `cgi-bin/kill_pid.cgi v1.0` — новый CGI для принудительного завершения процесса по PID (`kill -9`) с валидацией и логированием
  - `entware.js` — `renderServices()` обновлена: отображение дублей, кликабельный PID открывает модалку `showProcessList()` со списком всех процессов и кнопками "Убить"
  - `style.css` — `.pid-link`, `.pid-badge`, `.process-list`, `.process-item`, `.process-kill-btn`

- **Физические порты + сети на главной странице**:
  - `cgi-bin/network_status.cgi` — старый блок "LAN порты" (жёстко зашитые Порт 0/1/2/3/4/5) заменён на "Физические порты" (детектирует `eth*` интерфейсы, читает `carrier` и `speed` из sysfs) + "Сети" (парсит `brctl show` для мостов и определяет WAN)
  - `entware.js` — `loadNetworkStatus()` обновлена под новый формат данных

- **Блочные устройства — кликабельные, открываются в файловом менеджере**:
  - `cgi-bin/stats.cgi` — точки монтирования блочных устройств теперь `<a href>` на `tmpfs.cgi`
  - `cgi-bin/tmpfs.cgi v2.0` — снято ограничение на пути (работало только `/tmp`, `/dev`, `/dev/shm`). Теперь открывается любая существующая директория

- **Топ процессов по памяти в карточке RAM**:
  - `cgi-bin/stats.cgi` — добавлен сбор топ-5 процессов по RSS (через `ps -e -o rss,comm`), вывод компактной таблицы в карточку памяти
  - `style.css` — `.top-mem-wrapper` с hover-эффектом: подъём, тень и подсветка строк цветом, зависящим от загрузки RAM (normal→фиолетовый, warning→янтарный, critical→красный). Уменьшены отступы карточки памяти для компенсации

### Исправления

- **Плавность меню**:
  - `style.css` — `.sidebar .menu li` transition теперь включает `transform` и `box-shadow` (было только `padding`/`margin` из-за более высокой специфичности `.sidebar .menu li`). Ховер на пунктах меню стал плавным
  - `entware.js` — таймаут `menu-animate` увеличен с 500ms до 1400ms, чтобы pop-in анимация успевала завершиться для всех 15 пунктов меню

### Обновлённые файлы

| Файл | Версия | Описание |
|------|--------|----------|
| `cgi-bin/services.cgi` | 3.8 | Возвращает все PID (поле pids). Поддержка дублей |
| `cgi-bin/kill_pid.cgi` | 1.0 | Новый CGI: убийство процесса по PID |
| `cgi-bin/tmpfs.cgi` | 2.0 | Снято ограничение на пути |
| `cgi-bin/network_status.cgi` | — | Физические порты + сети вместо LAN портов |
| `cgi-bin/stats.cgi` | 0.26 | Кликабельные блочные устройства + топ по памяти |
| `entware.js` | — | PID бейджи, showProcessList, killProcess, обновлён network |
| `style.css` | — | PID badge, процесс-лист, top-mem, исправлен menu hover |
| `version.json` | 1.03.10 | Обновлено |

---

## 1.03 (2026-05-03)

### Новые функции

- **Модуль проверки системных зависимостей**:
  - `lib/common.sh v2.6` — добавлена функция `check_deps_logic()` для проверки всех зависимостей
  - `cgi-bin/check_deps.cgi v1.0` — новый CGI-скрипт проверки системы (cron, jq, iproute2, lighttpd)
  - `cgi-bin/check_syntax.cgi v1.1` — скрипт проверки синтаксиса всех CGI и библиотек
  - `entware.js v0.83` — добавлена функция `checkSystemDeps()` с красивым выводом статуса в модальном окне
  - `index.html` — добавлена кнопка "Проверка системы" в сайдбар

### Что проверяется

- **Базовые компоненты**: opkg, lighttpd (работает ли)
- **Утилиты BusyBox**: sed, awk, grep, ps
- **Пакеты Entware**: cron (установлен/запущен), jq, iproute2 (ip)
- **Статус разделов**: packages, services, monitoring, network, logger

### Интеграция

- Функция `check_deps_logic()` в `common.sh` может быть вызвана любым CGI-скриптом
- Кнопка в сайдбаре открывает модальное окно с цветовой индикацией статуса
- Рекомендации по установке отсутствующих пакетов

---

## 1.02 (2026-04-08)

### Исправления истории температур

- **temp_history.cgi v1.8** — исправлен парсинг:
  - Проблема: "temp":62},{"time":"23:59:33","temp":6200 - лишние данные из-за переноса строк между файлами
  - Решение: объединение файлов через $'\n' + надёжный awk с split()

- **wifi_temp_history.cgi v1.5** — аналогичное исправление:
  - Проблема: "temp0":71,"temp1":5700:43:38 - склейка данных из разных дней
  - Решение: такой же подход с объединением файлов

### Исправления интерфейса

- **index.html** — убран gradient с графика температуры CPU:
  - Удалена полупрозрачная заливка под графиком
  - Оставлена только чистая линия без "запыления"

---

## 1.01 (2026-04-08)

### Оптимизация производительности

- **services.cgi v3.7-fast** — полностью переписан для оптимизации:
  - Однократный сбор данных процессов (`ps` один раз в переменную)
  - Поддержка PIDFILE из скриптов init.d
  - Функция `get_var()` для извлечения PROCS/NAME/DAEMON/SCRIPT из скриптов
  - Проверка cmdline для избежания ложных срабатываний (zombie/мертвые процессы)
  - **Выигрыш: 94%** (с 3.2 сек до 0.37 сек)

- **network/status.cgi v1.7** — исправлен парсинг uptime и state file:
  - Заменён heavy `ps -o pid,etime | grep | awk` на `/proc/uptime` + `/proc/PID/stat`
  - Исправлен парсинг `last_check` из state file (`/tmp/network_watchdog_state.json`)
  - Защита от переводов строк в JSON (`tr -d '\n\r'`)
  - **Выигрыш: 60%**

- **upgradable.cgi v0.13** — исправлен парсинг:
  - Использует `$1`, `$3`, `$5` для правильного парсинга вывода `opkg list-upgradable`
  - Фильтрация строк с пустыми версиями и "-"
  - Клиентская фильтрация в `fetchUpgradable()` для отсеивания undefined

### Новые функции

- **Кнопка "Обновить все пакеты"**:
  - Новая кнопка в интерфейсе обновлений (оранжевый цвет)
  - Функция `upgradeAll()` в entware.js с подтверждением
  - Поддержка `upgrade_all=1` в upgrade.cgi
  - Запускает `opkg upgrade` без аргументов

### Исправления Service Watchdog

- **service_watchdog.sh v1.13** — улучшенная проверка PID:
  - Проверка cmdline для избежания ложных PID (kernel boot params)
  - Проверка zombie процессов через /proc/PID/status
  - Pattern для shadowsocks изменён на `ss-redir ss-local ss-server`

- **service_watchdog/events.cgi v2.4** — исправлен парсинг:
  - `tr -d '()'` удаляет все скобки из details
  - Раньше показывало `skipped (cooldown 60s))`, теперь `skipped cooldown 60s`

### Изменения интерфейса

- **entware.js** — убрано автообновление таблицы служб:
  - servicesInterval setInterval удалён
  - Таблица обновляется только при открытии вкладки и после действий пользователя
  - Значительно снижена нагрузка на роутер

### Обновлённые файлы

| Файл | Версия | Описание |
|------|--------|----------|
| `services.cgi` | 3.7-fast | Однократный ps, поддержка PIDFILE |
| `network/status.cgi` | 1.7 | Исправлен uptime + state file |
| `upgradable.cgi` | 0.13 | Исправленный парсинг $1,$3,$5 |
| `upgrade.cgi` | 0.03 | Поддержка upgrade all |
| `service_watchdog.sh` | 1.13 | Проверка cmdline + zombie |
| `service_watchdog/events.cgi` | 2.4 | tr -d '()' |
| `entware.js` | — | Кнопка "Обновить все", фильтрация undefined |
| `version.json` | 1.01 | Обновлено |

### Итог производительности

| Метрика | До | После | Улучшение |
|---------|-----|-------|-----------|
| services.cgi | 3.2 сек | 0.37 сек | **8.6x** |
| Общая нагрузка | ~42 сек/мин | ~4 сек/мин | **10x** |

---

## 0.68 (2026-04-06)

### Новые функции
- **Мониторинг служб (service watchdog)** — обновлённый функционал
  - **Автоперезапуск**: при падении службы демон автоматически перезапускает её через init.d скрипт
  - **Список исключений**: позволяет исключить определённые службы из мониторинга (по умолчанию: dropbear, kvas-ws, service_watchdog)
  - **Cooldown**: защита от повторного автоперезапуска (60 секунд), чтобы не мешать ручному управлению
  - **Режим custom**: переименован из "all", отслеживает только процессы из списка watch_list
  - **Логирование**: все изменения конфигурации и действия (start/stop/restart) записываются в основной лог

### Изменения интерфейса
- **Переключатели мониторинга**:
  - "Кастомный список" — включает режим custom с полем ввода списка процессов
  - "Исключения" — включает отображение списка исключений
  - "Автоперезапуск" — отдельный переключатель для включения автоперезапуска

### Исправления
- Исправлен парсинг JSON массивов в config.cgi (извлечение watch_list и exclude_list)
- Исправлена проблема с переключателем "Исключения" (сбрасывался при загрузке)
- Исправлена проблема с автоперезапуском (找到 неправильный init.d скрипт)
- Демон перезагружает конфигурацию при получении сигнала HUP
- service_watchdog исключает сам себя из мониторинга

### Обновлённые файлы
| Файл | Версия | Описание |
|------|--------|----------|
| `service_watchdog.sh` | 1.4 | Автоперезапуск, exclude_list, cooldown, HUP перезагрузка |
| `cgi-bin/service_watchdog/config.cgi` | 1.4 | Исправлен парсинг JSON, поддержка exclude_list |
| `cgi-bin/service_watchdog/status.cgi` | 1.2 | Возвращает exclude_list |
| `cgi-bin/service_watchdog/action.cgi` | 1.3 | Логирование всех действий |
| `cgi-bin/service_watchdog/events.cgi` | 2.0 | Исправлен парсинг details (убраны скобки) |
| `entware.js` | — | Новый UI: переключатели, поля ввода |

## 0.65 (2026-04-04)

### Новые функции
- **Модуль "Сеть" (Network)** — новая вкладка для мониторинга сети
  - Sidebar виджет статуса сети
  - Вкладка "Интерфейсы" — список сетевых интерфейсов
  - Вкладка "Маршруты" — таблица маршрутизации
  - Вкладка "ARP" — ARP таблица с именами хостов
  - Вкладка "События" — лог событий watchdog демона
  - Демон мониторинга network_watchdog.sh

### Улучшения интерфейса
- **Фильтр интерфейсов**: добавлен toggle "Скрыть неизвестные"
  - Скрывает интерфейсы со состоянием UNKNOWN
  - Полезно на устройствах с множеством виртуальных интерфейсов

### Исправления
- **Вкладки Network**: исправлена работа переключения вкладок (JavaScript)
- **action.cgi**: исправлен парсинг POST данных для управления демоном
- **ARP таблица**: добавлено определение имени хоста через getent hosts

## 0.64 (2026-04-01)

### Исправления безопасности
- **Исправлена критическая проблема**: логирование НЕ отключалось при `enabled=false`
  - Причина: `jq '// true'` трактует `false` как "falsy", возвращая `true`
  - Решение: используется явная проверка `if [ "$ENABLED_VALUE" = "true" ]`
  - Затронутые файлы: `logger/lib/logging.sh`, `lib/common.sh`
- **Исправлена проблема с PATH в CGI контексте lighttpd**:
  - CGI получал PATH: `/sbin:/usr/sbin:/bin:/usr/bin` (без `/opt/bin`)
  - Утилиты `jq`, `cat` не находились
  - Решение: использован полный путь `/opt/bin/jq`, `/opt/bin/cat`
  - Затронутые файлы: `common.sh`, `config.cgi`, `crontab_update.cgi`, `links_save.cgi`, `monitor/*.cgi`, и др.

### Версии после исправлений
| Файл | Версия | Описание |
|------|--------|----------|
| `lib/common.sh` | 2.5 | Чистая версия, использует logging.sh |
| `logger/lib/logging.sh` | 1.7 | Исправлен jq для false, чистая версия |
| `logger/config.cgi` | 1.5 | Исправлены пути |
| `cgi-bin/logger/system_log.cgi` | 1.8 | Использованы /opt/bin/cat, /opt/bin/sed |
| `cgi-bin/crontab_update.cgi` | 2.6 | Использует log_action из common.sh |
| `cgi-bin/links_save.cgi` | 0.04 | Исправлены пути |
| `monitor/*.cgi` | разные | Исправлены пути |
| `api.cgi` | 0.05 | Исправлены пути |
| `wifi_temp.cgi` | 0.04 | Исправлены пути |
| `ttyd_control.cgi` | 0.07 | Исправлены пути |
| `service_action.cgi` | 0.04 | Исправлены пути |

## 0.63 (2026-04-01)

### Новая функциональность
- Добавлен системный лог событий: `/opt/var/log/entware/system.log`
- Создан `logger/system_log.cgi` (v1.7) для просмотра системных событий
- Обновлён `logger/config.cgi` (v1.2) — теперь пишет в системный лог при включении/выключении логирования, показывает красивый вывод конфига
- Добавлена кнопка "Системные события" в UI (вкладка Логи)
- Обновлена справка `help.cgi` (v0.55) — добавлено описание системных событий

### Исправления
- **lib/common.sh v2.1**: исправлен `return` в `log_action()` — теперь не падает в fallback при выключенном логировании.
- `tmpfs.cgi v1.9`: убран `realpath`, заменён на встроенную нормализацию пути.
- `links_save.cgi v0.03`: добавлена валидация JSON через `jq empty`.
- `logger/view.cgi v1.4`: используется `index()` для поиска.

### Рефакторинг
- Создан общий модуль `lib/utils.js` с функцией `escapeHtml`.
- Удалён мёртвый код: `version.cgi`, `logger/status.cgi`.
- Удалено неиспользуемое поле `group` из конфига монитора.

### Улучшения UI
- Добавлен `?v=` к `monitor.js`.
- Удалена внешняя загрузка Google Fonts.

## 0.62 (2026-04-01)

### Исправления
- Исправлено URL-декодирование в `lib/common.sh` (`url_decode`) для корректной обработки `%2F` и других кодов.
- В ряде CGI добавлены явные завершения после отдачи JSON/ответа, чтобы исключить лишнее выполнение.
- `delete_file.cgi`: улучшены проверки и оптимизирована логика под слабые устройства.
- `wifi_temp.cgi`: исправлено формирование ответа.

### Оптимизация
- История температур CPU/WiFi: очистка старых данных запускается не на каждый запрос, а периодически.
- Улучшены отдельные участки обработки в `services.cgi`, `stats.cgi`, `upgradable.cgi`.

### Новые возможности
- Добавлены `temp_history.cgi` и `wifi_temp_history.cgi` для хранения истории температур (до 7 дней).
- В UI добавлены графики температуры CPU/WiFi.

## 0.57 (2026-03-29)

- Выделена общая библиотека CGI: `lib/common.sh`.
- Улучшена структура frontend-кода (`modal.js`, улучшение обработки ошибок).
- Доработаны `crontab_update.cgi`, `ttyd_control.cgi`.
- Обновлен `backup.sh`.

## Примечание

Каноническая текущая версия хранится в `version.json`.
