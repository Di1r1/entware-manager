# Entware Manager — Go Migration Specification

> Версия: 1.0
> Статус: черновик
> Цель: единый reference для всех Go-компонентов, чтобы JSON-форматы, пути, время и Content-Type были согласованы.

---

## 1. Системные пути (config.go)

### Константы

```go
WebRoot         = "/opt/web_entware"
ServicesDir     = "/opt/etc/init.d"
TempDir         = "/tmp"
LogDir          = "/tmp/entware/logs"
TempHistoryDir  = "/tmp/temp_history"
LoggerConfig    = "/opt/web_entware/logger/config.json"
VersionFile     = "/opt/web_entware/version.json"
LinksFile       = "/opt/web_entware/links.json"
MonitorConfig   = "/opt/web_entware/monitor_config.json"
NetworkConfig   = "/opt/web_entware/network_config.json"
ServiceConfig   = "/opt/web_entware/service_config.json"
BackupDir       = "/opt/web_entware/tmp"
LoggerLib       = "/opt/web_entware/logger/lib/logging.sh"
WatchdogPID     = "/tmp/watchdog.pid"
CronPID         = "/opt/var/run/cron.pid"
LighttpdPID     = "/opt/var/run/lighttpd.pid"
```

### Принципы

- Все пути — константы в `shared/config.go`.
- Никаких хардкодов в CGI-коде.
- При смене префикса `/opt` → `/opt/entware` — править одну строку.

---

## 2. Форматы времени

### 2.1. Для логов

```
Формат: YYYY-MM-DD HH:MM:SS
Пример: 2026-06-02 15:04:05
```

```go
const LogTimestampFmt = "2006-01-02 15:04:05"
```

### 2.2. Для имён файлов логов

```
Формат: YYYY-MM-DD
Пример: 2026-06-02
```

```go
const LogFileDateFmt = "2006-01-02"
```

Файл лога: `/tmp/entware/logs/2026-06-02.log`

### 2.3. Для истории температур

```
Формат: HH:MM:SS
Пример: 15:04:05
```

```go
const TimeOnlyFmt = "15:04:05"
```

Файл истории: `/tmp/temp_history/cpu.2026-06-02`
Формат строки: `15:04:05|65`

### 2.4. ISO 8601 (для check_deps и совместимости)

```
Формат: 2006-01-02T15:04:05Z07:00
Пример: 2026-06-02T15:04:05Z
```

```go
const ISO8601Fmt = "2006-01-02T15:04:05Z07:00"
```

---

## 3. Content-Type заголовки

### JSON

```
Content-type: application/json; charset=utf-8
```

Обязательно: `charset=utf-8`

### HTML

```
Content-type: text/html; charset=utf-8
```

Обязательно: `charset=utf-8`

### Разделитель

После заголовков — пустая строка (`\r\n` или `\n` — как CGI standard).

```go
func JSONHeader() {
    fmt.Print("Content-type: application/json; charset=utf-8\r\n\r\n")
}
```

---

## 4. Форматы логов

### 4.1. Основной лог менеджера

Файл: `/tmp/entware/logs/YYYY-MM-DD.log`

```
[2026-06-02 15:04:05] [INFO] [192.168.1.1] [1234] [stats] Системная статистика загружена
```

Поля через пробел:
```
[timestamp] [level] [IP] [PID] [script_name] message
```

- `level`: INFO, WARN, ERROR
- `script_name`: имя CGI без расширения (например, `stats`, `services`, `kill_pid`)

### 4.2. Системные события

Файл: `/opt/var/log/entware/system.log`

```
[2026-06-02 15:04:05] [SYSTEM] Включение логирования
[2026-06-02 15:04:05] [SYSTEM] Отключение логирования (было: enabled)
```

### 4.3. Монитор защиты

Файл: `/opt/temp/logs/monitor.log`

Формат — любой, не стандартизирован под Go.

---

## 5. JSON-форматы CGI (полные схемы)

### 5.1. temperature.cgi

**URL:** `GET /entware-cgi/temperature.cgi`
**Ответ:**
```json
{
    "temperature": 65
}
```
- `temperature`: `int | null`. Целое число градусов Цельсия. Если датчик недоступен — `null`.

### 5.2. wifi_temp.cgi

**URL:** `GET /entware-cgi/wifi_temp.cgi`
**Ответ:**
```json
{
    "temp0": 45,
    "temp1": 42,
    "combined": "WiFi0: 45C / WiFi1: 42C"
}
```
- `temp0`: `int | null`. Температура WifiMaster0.
- `temp1`: `int | null`. Температура WifiMaster1.
- `combined`: `string | null`. Склеенная строка для виджета. Если обе null — `null`.

Варианты `combined`:
```
"WiFi0: 45C / WiFi1: 42C"  — обе температуры
"WiFi0: 45C"                — только WifiMaster0
"WiFi1: 42C"                — только WifiMaster1
null                         — ни одной
```

### 5.3. version.cgi

**URL:** `GET /entware-cgi/version.cgi`
**Ответ:**
```json
{
    "version": "1.03.10"
}
```
- `version`: `string`. Из поля `version` файла `version.json`. Если файл недоступен — `"unknown"`.

### 5.4. services.cgi

**URL:** `GET /entware-cgi/services.cgi`
**Ответ:** JSON-массив объектов.

```json
[
    {
        "name": "S99kvas-ws",
        "status": "running",
        "enabled": true,
        "pid": "1234",
        "pids": ["1234", "5678"]
    },
    {
        "name": "S80cron",
        "status": "stopped",
        "enabled": true,
        "pid": "",
        "pids": []
    }
]
```

- `name`: `string`. Имя скрипта (например, `S99kvas-ws`).
- `status`: `"running" | "stopped"`.
- `enabled`: `bool`. `true` если скрипт начинается с `S`, `false` если с `K`.
- `pid`: `string`. Первый PID процесса, строка (может быть пустой).
- `pids`: `[]string`. Все PID процесса. Если процессов нет — пустой массив `[]`.

**Порядок обнаружения PID:**
1. PIDFILE из скрипта
2. Стандартные pid-файлы (`/tmp/name.pid`, `/var/run/name.pid`, etc.)
3. PROCS/NAME/DAEMON из скрипта → поиск в ps
4. По базовому имени (без цифрового префикса) → поиск в ps
5. По полному имени (S99kvas-ws) → поиск в ps
6. По .py файлу из SCRIPT

### 5.5. network_status.cgi

**URL:** `GET /entware-cgi/network_status.cgi`
**Ответ:**
```json
{
    "interfaces": [
        {"iface": "br0", "ip": "192.168.1.1"},
        {"iface": "eth2", "ip": "10.0.0.1"}
    ],
    "lan": "192.168.1.1, 10.0.0.1",
    "wifi": "подключено",
    "wifi_info": [
        {
            "name": "LAN",
            "2g": "ra0",
            "5g": "rai0",
            "rx": "150 MB",
            "tx": "30 MB"
        }
    ],
    "wan": "ppp0 (up)",
    "ports": [
        {"iface": "eth0", "speed": "1000Mbps", "carrier": "✓"},
        {"iface": "eth1", "speed": "—", "carrier": "—"}
    ],
    "networks": [
        {
            "name": "LAN",
            "bridge": "br0",
            "members": "eth0 eth1 ra0 rai0"
        },
        {
            "name": "WAN",
            "bridge": "ppp0",
            "members": ""
        }
    ]
}
```

- `interfaces`: `[]struct`. Интерфейсы с IP-адресами (из `ip -4 addr show`). Исключён `127.0.0.1`. Если нет — `[{"iface":"-","ip":"--"}]`.
- `lan`: `string`. IP через запятую для виджета. Если нет — `"--"`.
- `wifi`: `string`. `"подключено"` если br0 или ra/rai интерфейсы подняты; `"—"` если нет.
- `wifi_info`: `[]struct`. Информация по WiFi мостам. Если нет — `[{"name":"--","2g":"--","5g":"--","rx":"--","tx":"--"}]`.
  - `name`: `"LAN" | "Guest" | "Guest2" | <bridge_name>`.
  - `2g`: интерфейс 2.4GHz (ra*). `"--"` если нет.
  - `5g`: интерфейс 5GHz (rai*). `"--"` если нет.
  - `rx`: трафик RX в MB/GB.
  - `tx`: трафик TX в MB/GB.
- `wan`: `string`. WAN-интерфейс: `<iface> (up)`, `"down"`, `"--"`.
  - Порядок проверки: ppp0, ppoe0, wwan0 → default route.
- `ports`: `[]struct`. Физические порты (eth*, исключая VLAN ethX.Y).
  - `iface`: имя интерфейса.
  - `speed`: `"1000Mbps" | "—"`.
  - `carrier`: `"✓"` если carrier=1, `"—"` если нет. Если нет ни одного — `[{"iface":"—","speed":"—","carrier":"—"}]`.
- `networks`: `[]struct`. Мосты (brctl show) + WAN.
  - `name`: `"LAN" | "Guest" | "Guest2" | "WAN" | <bridge>`.
  - `bridge`: имя bridge или WAN-интерфейс.
  - `members`: участники bridge, через пробел, отсортированы. Для WAN — пустая строка.

### 5.6. monitor_status.cgi (monitor/monitor_status.cgi)

**URL:** `GET /entware-cgi/monitor/monitor_status.cgi`
**Ответ:**
```json
{
    "demon_status": "running",
    "demon_pid": "1234",
    "processes": [
        {"pid": 1234, "pcpu": "1.5", "time": "01:23:45", "command": "kvas-ws"},
        {"pid": 5678, "pcpu": "0.8", "time": "12:34:56", "command": "cron"}
    ]
}
```

- `demon_status`: `"running" | "stopped"`.
- `demon_pid`: `string`. PID демона или `""`.
- `processes`: `[]struct`. Топ-5 процессов по CPU из `ps -e -o pid,pcpu,etime,comm --no-headers --sort=-pcpu`.
  - `pid`: `int`.
  - `pcpu`: `string`. Процент CPU.
  - `time`: `string`. Время жизни процесса (etime).
  - `command`: `string`. Имя команды, экранировано.

**Примечание:** поле называется `demon_status` (с ошибкой — `demon`, а не `daemon`). Сохранить для обратной совместимости.

### 5.7. stats.cgi

**URL:** `GET /entware-cgi/stats.cgi`
**Ответ:** HTML (text/html). Не JSON.

Структура HTML-вывода (карточки):

```
Статистика системы
├── Система (таблица: Модель, Имя хоста, Архитектура, Версия ядра, Время работы)
├── Память (RAM)
│   ├── Использовано / Всего: "123 MB / 256 MB"
│   ├── Загрузка: "48%"
│   ├── progress-bar
│   └── Топ по памяти (таблица: command | RSS)
├── Пакеты Entware
│   ├── Установлено: "42"
│   └── Доступно: "3500"
├── Диск (/opt)
│   ├── Размер / Использовано / Доступно / Загрузка
│   └── progress-bar
├── Сеть (загружается через network_status.cgi)
├── tmpfs (таблица: ФС | Размер | Использовано | Доступно | Загрузка | Точка монтирования)
└── Блочные устройства (таблица: ФС | Размер | ... | Точка монтирования)
```

**Топ по памяти (top-mem-wrapper):**
```html
<div class="top-mem-wrapper top-mem-normal">
  <table class="top-mem">
    <tr><th colspan="2">Топ по памяти</th></tr>
    <tr><td>kvas-ws</td><td>45 MB</td></tr>
    <tr><td>php-fpm</td><td>32 MB</td></tr>
    ...
  </table>
</div>
```

Классы `top-mem-normal`, `top-mem-warning`, `top-mem-critical` — в зависимости от `MEM_CLASS`.

**Точки монтирования** — кликабельны: `<a href="/entware-cgi/tmpfs.cgi?path=/tmp">/tmp</a>`.

### 5.8. tmpfs.cgi

**URL:** `GET /entware-cgi/tmpfs.cgi?path=/tmp/some/dir`
**Ответ:** HTML (text/html). Файловый менеджер.

Параметры:
- `path` — путь к директории (URL-encoded).

Вывод содержит навигацию (`../`), таблицу файлов (имя, размер, тип, дата), кнопки удаления.

### 5.9. temp_history.cgi

**URL (save):** `POST /entware-cgi/temp_history.cgi?action=save&temp=65`
**Ответ:**
```json
{"status": "ok"}
```

**URL (history):** `GET /entware-cgi/temp_history.cgi?action=history`
**Ответ:**
```json
[
    {"time": "15:04:05", "temp": 65},
    {"time": "15:04:35", "temp": 66}
]
```

- `time`: `string`. Формат `HH:MM:SS`.
- `temp`: `int`. Целое число градусов.

Если данных нет — `[]`.

**Очистка:** файлы старше 7 дней удаляются при `action=save`.

### 5.10. wifi_temp_history.cgi

**URL (save):** `POST /entware-cgi/wifi_temp_history.cgi?action=save&temp0=45&temp1=42`
**Ответ:**
```json
{"status": "ok"}
```

**URL (history):** `GET /entware-cgi/wifi_temp_history.cgi?action=history`
**Ответ:**
```json
[
    {"time": "15:04:05", "temp0": 45, "temp1": 42},
    {"time": "15:04:35", "temp0": null, "temp1": 43}
]
```

- `time`: `string`. Формат `HH:MM:SS`.
- `temp0`: `int | null`.
- `temp1`: `int | null`.

Если данных нет — `[]`.

### 5.11. kill_pid.cgi

**URL:** `POST /entware-cgi/kill_pid.cgi`
**Body:** `pid=1234`
**Ответ:**
```json
{"status": "ok"}
```

При ошибке:
```json
{"status": "error", "message": "PID не указан"}
```

### 5.12. ttyd_control.cgi

**URL:** `GET /entware-cgi/ttyd_control.cgi?action=status`
**URL:** `POST /entware-cgi/ttyd_control.cgi?action=start|stop|restart`

**Ответ (status):**
```json
{
    "htop": {"pid": "1234", "port": 8089, "status": "running"},
    "terminal": {"pid": "", "port": 9089, "status": "stopped"}
}
```

**Ответ (action):**
```json
{"status": "ok"}
```

### 5.13. logger/view.cgi

**URL:** `GET /entware-cgi/logger/view.cgi?date=2026-06-02&level=INFO&search=...`
**Ответ:** HTML (text/html).

### 5.14. logger/config.cgi

**URL:** `GET /entware-cgi/logger/config.cgi`
**Ответ:**
```json
{"enabled": true}
```

**URL:** `POST /entware-cgi/logger/config.cgi`
**Body:** `enabled=true|false`
**Ответ:**
```json
{"status": "ok"}
```

### 5.15. check_deps.cgi

**URL:** `GET /entware-cgi/check_deps.cgi`
**Ответ:**
```json
{
    "base": {
        "opkg": "true",
        "lighttpd_running": "true",
        "sed": "true",
        "awk": "true",
        "grep": "true",
        "ps": "true"
    },
    "deps": {
        "cron_installed": "true",
        "cron_running": "true",
        "jq": "true",
        "ip": "true",
        "ip_path": "/sbin/ip",
        "ip_pkg_installed": "true"
    },
    "sections": {
        "packages": "ok",
        "services": "ok",
        "monitoring": "ok",
        "network": "ok",
        "logger": "ok"
    },
    "overall_status": "ok",
    "timestamp": "2026-06-02T15:04:05Z"
}
```

Все поля — строки `"true"` или `"false"`, намеренно не булевы (обратная совместимость).
- `overall_status`: `"ok" | "partial" | "critical"`.
- `timestamp`: ISO 8601.

---

## 6. Обработка ошибок

### 6.1. JSON-эндпоинты

При успехе:
```json
{"status": "ok"}  // или ожидаемые данные
```

При ошибке:
```json
{"status": "error", "message": "Описание ошибки"}
```

HTTP-статус: всегда 200 (CGI не может менять статус). Ошибка передаётся в JSON.

### 6.2. Валидация входных данных

- `SanitizeAlnum(s)` — `[^a-zA-Z0-9._-]` → удалить. Для имён пакетов, служб.
- `path.Clean(path)` + `strings.HasPrefix(resolved, baseDir)` — для защиты от path traversal (tmpfs.cgi).
- PID — только цифры.

---

## 7. Формат истории температур (на диске)

### 7.1. CPU

Файл: `/tmp/temp_history/cpu.YYYY-MM-DD`
Формат строки (одна запись в строку):
```
15:04:05|65
```

Разделитель: `|` (pipe).
Данные не сортируются, добавляются в конец файла.

### 7.2. WiFi

Файл: `/tmp/temp_history/wifi.YYYY-MM-DD`
Формат строки:
```
15:04:05|45|42
```

Поля: `time|temp0|temp1`. Если температура недоступна — `-` (дефис).

Очистка: файлы старше 7 дней.

---

## 8. Процессы (/proc парсинг)

### 8.1. `/proc/[pid]/stat`

Поля (по POSIX):
1. pid — int
2. comm — string (в скобках)
3. state — char
4. ppid — int
...
24. rss — int64 (RSS в страницах, × page_size = bytes)

**RSS в KB:** `rss * 4` (для страниц 4KB).

### 8.2. `/proc/[pid]/status`

Поля (ключевые):
```
Name:   bash
VmRSS:  1234 kB
```

Приоритет: `/proc/[pid]/status` точнее для RSS (готовое значение в kB), `/proc/[pid]/stat` быстрее (один read).

### 8.3. `/proc/meminfo`

```
MemTotal:       256000 kB
MemFree:        128000 kB
MemAvailable:   200000 kB
```

### 8.4. `/proc/uptime`

```
12345.67 67890.12
```

Первое поле — секунды работы системы.

### 8.5. `/proc/net/dev`

Интерфейсы с трафиком:
```
eth0:  123456  789012  ...  0  0  0  0  0  0  0  0  234567  890123  ...
```

Поля (2-е = RX bytes, 10-е = TX bytes).

### 8.6. `/proc/loadavg`

```
0.15 0.20 0.10 1/234 12345
```

Первые 3 поля — load average 1/5/15 min.

---

## 9. Архитектура Go-бинарников

### 9.1. Принцип

Каждый бинарник — группа логически связанных эндпоинтов. Диспетчеризация через `ENDPOINT` (env var):

```
cgi-bin/services.cgi → ENDPOINT=services      exec cgi-bin/go/entware-services
cgi-bin/monitor/monitor_status.cgi → ENDPOINT=monitor_status exec cgi-bin/go/entware-monitor
```

### 9.2. Бинарники

| Бинарь | ENDPOINTы | Размер (UPX) |
|--------|-----------|:------------:|
| `entware-pkg` | `available`, `packages`, `install`, `remove`, `upgrade`, `update`, `upgradable`, `api` | 765KB |
| `entware-stats` | `stats`, `version`, `help`, `links_load`, `links_save`, `tmpfs`, `view_file`, `delete_file`, `auth_config`, `crontab`, `crontab_update` | 791KB |
| `entware-net` | `network_interfaces`, `network_routes`, `network_arp`, `network_status`, `network_stats` | 739KB |
| `entware-logger` | `logger_config`, `logger_view`, `logger_system_logs`, `logger_system_log`, `logger_find_by_name`, `logger_rotate`, `logger_clear` | 737KB |
| `entware-services` | `services`, `service_action`, `ttyd_control` | 708KB |
| `entware-monitor` | `monitor_status`, `monitor_action`, `monitor_config`, `monitor_log`, `temperature`, `wifi_temp`, `temp_history`, `wifi_temp_history`, `kill_pid` | 1.8MB |

### 9.3. Структура пакетов

```
go/
├── cmd/
│   ├── entware-pkg/main.go
│   ├── entware-stats/main.go
│   ├── entware-net/main.go
│   ├── entware-logger/main.go
│   ├── entware-services/main.go
│   └── entware-monitor/main.go
├── internal/
│   ├── packages/     — shared + handlers для entware-pkg
│   ├── stats/        — handler для entware-stats
│   ├── network/      — shared + handlers для entware-net
│   ├── logger/       — shared + handlers для entware-logger
│   ├── services/     — shared + handlers для entware-services
│   └── monitor/      — shared + handlers для entware-monitor
└── go.mod
```

### 9.4. Wrapper CGI (shell)

Каждый CGI-файл — трёхстрочный shell-скрипт:

```sh
#!/bin/sh
export PATH=/opt/sbin:/opt/bin:/sbin:/bin:/usr/sbin:/usr/bin
ENDPOINT=endpoint_name exec /opt/web_entware/cgi-bin/go/entware-binary
```

---

## 10. Обратная совместимость

### Критические поля (не менять имя)

| CGI | Поле | Почему |
|-----|------|--------|
| monitor_status.cgi | `daemon_status` | `demon_status` — сохранено как в shell-оригинале |
| services.cgi | `pid` | строка, не int |
| services.cgi | `enabled` | bool |
| temperature.cgi | `temperature` | int или null |
| network_status.cgi | `2g` и `5g` | ключи с цифрами, в Go `map[string]string` |
| wifi_temp.cgi | `combined` | null при отсутствии данных |

---

## 11. Сборка и деплой

```sh
# Сборка
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o /tmp/entware-xxx ./cmd/entware-xxx/

# Сжатие
upx -9 /tmp/entware-xxx -o /tmp/entware-xxx.upx

# Деплой
smbclient //192.168.3.1/Entware_USB -U 'USER%PASS' \\
  -c 'put /tmp/entware-xxx.upx web_entware/cgi-bin/go/entware-xxx'

# Оригиналы shell сохраняются в tmp/ + SMB web_entware/tmp/
```
