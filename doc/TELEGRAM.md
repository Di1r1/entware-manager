# Модуль Telegram-уведомлений и чат-бота

Правила проекта: [`RULES.md`](../RULES.md)

## Назначение

Отправка событий (уведомлений) в Telegram и опционально ответы чат-бота на команды пользователя.
Модуль реализован как **независимый шлюз**: отдельный демон-процесс, который читает уже
существующие файлы-источники событий и шлёт их в Telegram через Bot API. Он НЕ встраивается
в существующие обработчики панели — отказ/недоступность Telegram никак не влияет на основные
функции Entware Manager.

## Архитектура

```
[Источники событий]  →  [ШЛЮЗ-демон]  →  [Telegram Bot API]
   /tmp/entware/logs/*.log     telegram_gateway.sh        sendMessage
   /opt/var/log/entware/system.log
```

- Шлюз — отдельный процесс (`telegram_gateway.sh`), запускается через `S85entware-watchdogs`
  по `autostart:true` в `telegram_config.json`.
- Читает источники **напрямую** по offset-файлу — существующий код логирования не меняется.
- Фильтрует по уровню (ERROR/WARN/INFO) и источнику, отправляет через `curl`.

## Конфигурация

Файл: `/opt/web_entware/telegram_config.json` (права **0600**).

```json
{
  "enabled": false,
  "bot_token": "",
  "chat_id": "",
  "level": "ERROR",
  "sources": ["system", "monitor"],
  "bot_enabled": false,
  "autostart": false
}
```

| Поле | Тип | Описание |
|---|---|---|
| `enabled` | bool | Включить уведомления |
| `bot_token` | string | Токен бота (никогда не отдаётся в GET) |
| `chat_id` | string | ID чата получателя (только цифры) |
| `level` | string | Минимальный уровень: `ERROR`, `WARN`, `INFO`, `OFF` |
| `sources` | array | Источники: `system`, `monitor`, `network`, `service` |
| `bot_enabled` | bool | Включить чат-бота (этап 4) |
| `autostart` | bool | Автозапуск демона при загрузке |

### Безопасность секрета

- `bot_token` **не возвращается** через GET-эндпоинт — только флаг `configured`.
- POST принимает токен, сохраняет атомарно (temp+mv), права 0600.
- `telegram_config.json` подпадает под 403-список (`*_config.json`).
- Токен **redact** в логах/ошибках демона и CGI.
- `chat_id` валидируется (только цифры).
- ⚠️ Токен попадает в бэкап-архив (`backup.sh` копирует `/opt/web_entware`). При
  компрометации/ротации токена — сменить его через панель (BotFather).

## Эндпоинты

| Эндпоинт | Метод | Описание |
|---|---|---|
| `telegram_config.cgi` | GET | Конфиг без токена: `{enabled, configured, level, sources, bot_enabled}` |
| `telegram_config.cgi` | POST | Сохранение конфига (Origin-чек) |
| `telegram_test.cgi` | POST | Отправка тестового сообщения (Origin-чек, анти-спам) |

Маппинг — в 6 местах: `cgi-bin/go.cgi`, `go/internal/server/cgi.go`, `build-deploy.sh`
(цикл сборки + симлинки), `Install/install.sh` (GO_BINS + счётчик бинарников).

## Чат-бот (этап 4)

Отдельный режим демона — polling `getUpdates`, команды `/status`, `/disk`, `/services`,
`/uptime`, `/help`. Отвечает только в разрешённый `chat_id`. Команды — только чтение,
без мутаций панели.
