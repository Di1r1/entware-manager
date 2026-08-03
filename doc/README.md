# Entware Manager — полная документация

Актуально для версии `1.03.10` от `2026-06-02`.

Этот каталог содержит документацию по архитектуре, установке, API, эксплуатации и диагностике проекта `web_entware`.

## Содержание

1. `doc/INSTALL.md` — требования, установка, обновление и удаление.
2. `doc/ARCHITECTURE.md` — структура проекта, модули и потоки данных.
3. `doc/API.md` — описание CGI API (эндпоинты, параметры, ответы).
4. `doc/OPERATIONS.md` — эксплуатация, cron, backup/restore, регламенты.
5. `doc/TROUBLESHOOTING.md` — типовые проблемы и способы решения.
6. `doc/SECURITY.md` — ограничения доступа, безопасность и права.
7. `doc/NETWORK.md` — модуль "Сеть": мониторинг, интерфейсы, маршруты, ARP.
8. `../go/SPEC.md` — спецификация для миграции на Go (черновик).

## Быстрый старт

1. Установите зависимости:
   ```sh
   opkg update
   opkg install lighttpd ttyd htop jq coreutils-base procps-ng bridge-utils ip-full
   ```
2. Скопируйте проект в `/opt/web_entware`.
3. Настройте `lighttpd` для алиасов `/entware-manager/` и `/entware-cgi/`.
4. Перезапустите веб-сервер: `/opt/etc/init.d/S80lighttpd restart`.
5. Откройте: `http://<router-ip>:8087/entware-manager/`.

> **Если на роутере уже есть другой lighttpd** (например, веб-панель zapret на 8088):
> стандартный `S80lighttpd` не поднимет порт 8087 (rc.func управляет по имени
> процесса и считает lighttpd уже запущенным). Просто перезапустите `install.sh` —
> установщик обнаружит чужой lighttpd, создаст `/opt/etc/init.d/S80entware-lighttpd`
> (управление по pid-файлу) и поднимет менеджер отдельным экземпляром, не трогая чужой.

Подробности — в `doc/INSTALL.md`.
