# Правила для LLM-ассистента v3.0

## 1. Поиск существующих реализаций перед созданием новых
Перед написанием новой функции/скрипта выполнить `grep`/`ls` по проекту.

## 2. Единый стиль shell (разделение CGI и демонов)

**Демоны и утилиты** (install.sh, backup.sh, watchdog.sh и т.п.):
```sh
#!/bin/sh
set -eu
export PATH=/opt/sbin:/opt/bin:/sbin:/bin:/usr/sbin:/usr/bin
```

**CGI-скрипты:**
```sh
#!/bin/sh
export PATH=/opt/sbin:/opt/bin:/sbin:/bin:/usr/sbin:/usr/bin
```
- `set -eu` **запрещён** – ошибки обрабатываются через `die()` из `common.sh`.

## 3. Общие функции из lib/common.sh (таблица статусов)

| Функция | Статус | Описание |
|---------|--------|----------|
| `pid_is_alive(pid)` | ✅ Существует | Замена `kill -0` |
| `find_pids(pattern)` | ✅ Существует | Замена `pgrep` |
| `url_decode(str)` | ✅ Существует | Декодирование URL |
| `json_out(data)` | ✅ Существует | JSON + Content-Type |
| `html_header()` | ✅ Существует | HTML-заголовок |
| `post_param(key, default)` | ✅ Существует | Парсинг POST |
| `log_message(level, msg)` | ✅ Существует | Единое логирование |
| `load_config(config_file)` | ✅ Существует | Загрузка JSON-конфига |
| `html_escape(string)` | ✅ Существует | Экранирование HTML |
| `human_size(bytes)` | ✅ Существует | Форматирование размера |
| `die(message, code)` | ❌ Не добавлена | Обработка ошибок CGI |

## 4. Все пути – переменные в начале, никаких хардкодов
Запрещены жёсткие IP-адреса, пути, имена файлов.

## 5. Совместимость с BusyBox
Запрещены GNU-флаги: `--no-headers`, `--sort`, `-printf` и т.п.

## 6. Журнал правок
Правки фиксируются в `doc/CHANGELOG.md` (с версией и датой). Короткое описание — в `version.json`.

## 7. Версионирование (SemVer + version.json + CHANGELOG.md)
Обновлять при каждом релизе/значимом изменении.

## 8. Линтинг shell
`shellcheck` + `checkbashisms` перед коммитом (по возможности). Если нет доступа — проверить вручную на BusyBox-совместимость.

## 9. Запрет kill -0 и pgrep
Только `pid_is_alive()` и `find_pids()` из `common.sh`.

## 10. Атомарность и идемпотентность
PID-файлы через temp + mv; start/stop/restart идемпотентны.

## 11. Deploy
**Go-бинарники:** компиляция через `GOOS=linux GOARCH=arm64 go build`, сжатие `upx -9`, загрузка через `ssh cat >`.
**Shell-скрипты:** загрузка через `ssh cat >` напрямую в `/opt/web_entware/`.
**Запрещено** использовать `install.sh` для деплоя — он стирает конфиги.

Перед заменой shell CGI на Go:
- Сохранить оригинал в `tmp/` проекта и на роутере в `/tmp/entware_backup_cgi/`
- Создать wrapper (`ENDPOINT=xxx exec /opt/web_entware/cgi-bin/go/бинарник`)

## 12. Единая обработка ошибок CGI (после добавления die())
Все CGI через `die(message, code)`.

## 13. Единый вывод Content-Type
JSON → `json_out()`, HTML → `html_header()`. Никаких ручных `echo`.

---

# Правила для Go

## G1. Архитектура
- Один бинарник на логическую группу эндпоинтов (сеть, smart, система, пакеты)
- Диспетчер через `os.Getenv("ENDPOINT")` + `switch`
- Общий код — в `go/internal/` подпакетах
- Content-Type выводится в начале `main()`, не в каждом handler'е

## G2. Сборка
```sh
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o /tmp/бинарник ./cmd/бинарник/
upx -9 /tmp/бинарник -o /tmp/бинарник.upx
```

## G3. JSON-форматы
- Поля именовать как в shell-оригинале (`snake_case`)
- `omitempty` для опциональных полей
- Все эндпоинты READ-ONLY (GET), если не требуется запись

## G4. Импорт пути
Модуль `entware-manager`, внутренние пакеты: `entware-manager/internal/группа/`

## G5. Wrapper для CGI (shell)
```sh
#!/bin/sh
export PATH=/opt/sbin:/opt/bin:/sbin:/bin:/usr/sbin:/usr/bin
ENDPOINT=network_interfaces exec /opt/web_entware/cgi-bin/go/entware-net
```

---

## Возможные риски
- shellcheck не настроен в CI
- часть скриптов использует устаревшие конструкции
- BusyBox-совместимость проверяется только на реальном роутере
- Go-бинарники статически слинкованы (~500KB-2MB против ~1KB shell-скрипта)
