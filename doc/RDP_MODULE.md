# RDP-модуль: анализ, реализация, настройки

- Статус: **РЕАЛИЗОВАНО (бэкенд v1.08.7 + клиентские доработки), проверено на dev-роутере**
- Дата анализа: 2026-08-10 · реализация: 2026-08-10
- Агенты-аналитики: ewm-reviewer (архитектура), ewm-sec (безопасность), ewm-designer (фронтенд), ewm-busybox (сборка/деплой)
- Базовый модуль: `github.com/nakagami/grdpwasm` (Go+WASM веб-RDP-клиент, GPL-3.0), форк в `/tmp/opencode/grdpwasm`

---

## 0. Фактическое состояние и настройки (актуально)

Ниже — как всё реализовано фактически и какие настройки доступны пользователю.

### Состав (что задеплоено)

| Компонент | Расположение на роутере | Назначение |
|---|---|---|
| Прокси `grdp-proxy` | `/opt/web_entware/cgi-bin/go/grdp-proxy` | статика клиента + WebSocket-релей (`/ws?target=host:port`) |
| Статика клиента | `/opt/web_entware/static/rdp/` (`index.html`, `main.wasm` ~10 МБ, `wasm_exec.js`) | веб-RDP-клиент grdpwasm (Go+WASM) |
| Init-скрипт | `/opt/etc/init.d/S90grdp-proxy` → симлинк на `/opt/web_entware/Install/S90grdp-proxy` | порт/пути из `rdp_config.json`, pid `/opt/var/run/grdp-proxy.pid`, идемпотентные start/stop/restart/check |
| Config | `/opt/web_entware/rdp_config.json` | `proxy_port` (9099), `proxy_host`, `bin_path`, `static_dir`, `enabled` |
| Бэкенд | `go/cmd/entware-rdp` + `go/internal/rdp` | `rdp_status.cgi` (GET), `rdp_start/stop.cgi` (POST+пароль+Origin), `rdp_config.cgi` (GET/POST) |
| Аутентификация | `go/internal/auth` (fail-closed) | пароль по `auth_config.json`, Origin-чек на мутациях |
|Frontend | `/opt/web_entware/rdp.js` (+ `entware.js` ветка `tabName === 'rdp'`) | статус, кнопки start/stop, еслиrame встраивается в панель |

### Клиентские доработки (исправление артефактов картинки)

Форк grdpwasm доработан, файлы `static/index.html` и `main.go` пересобраны:

1. **`DisableAVC444()`** — WASM-клиент рекламирует только AVC420. AVC444/LC2 (chroma-upgrade) без встроенного декодера терялись и рвали картинку.
2. **Codec из SPS, а не хардкод.** Раньше был фиксированный `avc1.42E01E` (Baseline), RDP-сервер (Windows) чаще шлёт Main/High — расхождение вызывало артефакты в Chrome/Opera. Теперь профиль/уровень берутся из реального SPS (`h264CodecFromSps`).
3. **Ожидание первого IDR** — delta-кадры до первого ключевого отбрасываются (WebCodecs не умеет декодировать delta без keyframe).
4. **Сброс декодера при ошибке** — при `decoder.error` сбрасываются `h264SeenKey` и очередь кадров, клиент ждёт свежий IDR.
5. **Fallback на битмапы (RemoteFX/NSCodec).** Если WebCodecs недоступен или `VideoDecoder.isConfigSupported()` не подтверждает H.264 (например, Android Chrome без аппаратного кодера), выставляется `window.noWebCodecs`, Go-сторона НЕ регистрирует `OnH264Raw` → CAPS_ADVERTISE v8.0 + `capFlagAVCDisabled` → сервер переключается на RemoteFX/NSCodec, картинка идёт через `OnBitmap → renderBitmaps`.
6. **FPS-счётчик** — в статус-баре клиента `Connected (WebCodecs H.264) — N fps`, обновление каждые 500 мс; считаются кадры обоих путей (H.264 `output` и `renderBitmaps`).

### Пресеты потока (настройка качества/скорости)

В тулбаре клиента добавлены селект **Preset** и поле **Quality** (входное значение `queueDepth`), см. `static/index.html`:

| Пресет | Разрешение | queueDepth (Quality) | Эффект |
|---|---|---|---|
| Compact | 1024×600 | 60 | сильное ограничение → плавнее, меньше трафик |
| Balanced | 1280×800 | 20 | умеренное, по умолчанию |
| Crisp | 1920×1080 | 5 | близко к max качеству |
| Custom | как введено | 0 | без троттлинга (максимум качества) |

- `queueDepth` передаётся в Go: `rdpConnect(..., swapAltMeta, queueDepth)` → `g.SetQueueDepthHint()` после логина (MS-RDPEGFX FRAME_ACKNOWLEDGE; 0 = off, 10–50 умеренный, 100+ сильный, `0xFFFFFFFF` = пауза потока).
- Выбор пресета/глубины сохраняется в cookie `rdp_preset` / `rdp_queueDepth` (`SameSite=Strict`, год) и восстанавливается `loadCookies()`.

### Настройки в зависимости от конфига

- `rdp_config.json` (единая точка) — см. раздел 7; порт меняется только из него (`proxy_port`), URL iframe строится из `proxy_host || location.hostname` + числовой порт.
- Кэш-версии фронтенда: `entware.js` грузит `rdp.js?v=9`; при изменении `rdp.js`/`entware.js` версию поднимать.

### Известные ограничения клиента

- Firefox (десктоп) — H.264 WebCodecs известен багами; при проблемах клиент автоматически переходит на битмап-путь (`noWebCodecs`).
- Если iframe-страница периодически перезагружалась и не давала ввести данные — исправлено: поллинг статуса в `rdp.js` больше не переприсваивает `frame.src` (guard через атрибут `data-src`), см. раздел 6.
- Высота iframe подстраивается под окно (`fitFrame()` на resize, минимум 300px) — скролл панели не нужен.

---

## 1. Идея модуля

Вкладка **RDP** в веб-панели Entware Manager, которая позволяет:
- Подключаться к ПК по RDP прямо из браузера (через веб-интерфейс менеджера).
- Управлять подключениями (запуск, остановка, статус).
- Использовать **grdpwasm** (веб-RDP-клиент на Go+WASM) как клиентскую часть.
- Запускать лёгкий прокси-сервер на роутере (`grdp-proxy`) для перенаправления WebSocket-трафика к RDP-порту (3389).

### Архитектура связки (как спроектирован grdpwasm)
```
Browser (WASM) ←WebSocket→ proxy (Go, на роутере) ←TCP→ RDP server (ПК в LAN, порт 3389)
```
- Прокси `./proxy` — одновременно статик-сервер клиента и WebSocket-релей, БЕЗ встроенной авторизации.
- Клиент открывает WS: `ws://host:port/ws?target=<rdp_host>:<port>` — **цель подключения задаётся клиентом в URL**.
- Креды (`user`, `password`, `domain`) — в JS-памяти браузера, **не в URL**.
- Рендер RDP-сессии — в браузере клиента (Go-WASM runtime, ~100–300 МБ на вкладку); на роутер ложится только пересылка байтов.

---

## 2. Вердикт аналитиков (сводка)

**Реализуемо, но с тремя принципиальными правками плана:**

1. **Порт 8080 ЗАНЯТ** на dev-роутере — там AdGuard Home (`links.json:1`). Дефолт прокси `:8080` → конфликт. Нужен конфигурируемый порт с дефолтом **9099** (8089/9089 — ttyd, 9097 — koffe-api).
2. **Безопасность — главный риск.** Прокси без авторизации, креды RDP по plaintext WebSocket, `/ws?target=<любой host:port>` = открытый TCP-релей/SSRF/брутфорс RDP с LAN. Плюс пали-орана `checkFilemgrAuth` (fail-open) и незащищённый `auth_config.cgi` (можно отключить пароль без ввода текущего).
3. **Сборка WASM**: grdpwasm требует **Go 1.24+** (проект на 1.22; `GOTOOLCHAIN=auto` доскачает), `grdp-proxy` нужен для всех 3 arch (arm64/mips/mipsel), WASM (GOOS=js) — один раз, архитектурно независим. +8–20 МБ в релизные архивы. `grdp-proxy` сжимается **UPX** (5.9МБ → ~2.1МБ); **WASM UPX не поддерживает** (UnknownExecutableFormatException). WASM-клиент сжимается **gzip на лету** самим grdp-proxy (gzipHandler для статики, ~10МБ → ~3МБ) — работает и в lighttpd-режиме (через `/rdp/`-прокси), и в go-режиме; в lighttpd-режиме также `mod_deflate` на `application/wasm`.

### Детальные выводы по направлениям

#### Архитектура (ewm-reviewer)
- Паттерн диспетчера `os.Getenv("ENDPOINT")` + switch — совпадает с `go/cmd/entware-services/main.go:12-41`. Нужен `_ "entware-manager/internal/localtime"` (как у всех cmd/*).
- JSON-конвенция `snake_case` + `omitempty` — как `ttydInstance` (`services/ttyd.go:12-17`).
- Импорт пути `entware-manager/internal/rdp/` соответствует `go.mod` (module entware-manager).
- Все места маппинга нового flat-бинарника `entware-rdp` — см. раздел 5 (таблица).
- **`HTTP_ORIGIN` / `HTTP_SEC_FETCH_SITE` не передаются** в CGI-окружение (`cgi.go:180-207 buildCGIEnv`) — Origin-чек из скила `ewm-auth-fixes` будет мёртвым, нужен проброс в общем коде.
- Запуск долгоживущего процесса из CGI: прецедент `services/ttyd.go:93-131` (exec.Command + Start + опрос). Риски: `cmd.Run()` убьёт по таймауту (300с, `cgi.go:150`) — нужен `cmd.Start()`; в lighttpd-режиме `mod_cgi` шлёт SIGTERM группе — нужен `SysProcAttr{Setpgid:true}`, иначе прокси умрёт при закрытии соединения.
- PID-файл должен быть `/opt/var/run/entware-rdp.pid` (конвенция проекта), не `/var/run/...`; атомарная запись temp+mv (правило 10 RULES).

#### Безопасность (ewm-sec) — чек-лист обязательных фиксов ДО реализации
1. **Открытый релей `:port` (SSRF/пивотинг/брутфорс, любой на сети)** — Высокий. Минимум: bind на LAN-интерфейс (не `0.0.0.0`), валидация `target` (только сохранённые цели), одноразовый токен `?token=` от `rdp_start`, CSP `frame-ancestors http://<router>:8087`.
2. **`rdp_config` с хранением паролей / настраиваемым staticDir/port** — Высокий. Креды НЕ хранить (нужен plaintext для подключения): только host/port/domain/размер экрана; пароль вводится при каждом подключении и затирается. `staticDir` жёстко `/opt/web_entware/static/rdp/` (иначе инъекция заставит прокси раздавать конфиги/ключи). Валидация host/port.
3. **Пали-орана `checkFilemgrAuth` (fail-open) + незащищённый `auth_config.cgi`** — Высокий. Нужен общий `go/internal/auth` (fail-closed: файл отсутствует/битый/нет hash → deny + «Настройте пароль в разделе Защита»; `enabled=false` в валидном конфиге → allow). Защитить `auth_config.cgi` текущим паролем.
4. **CSRF на rdp_*** — Средний/Высокий. Только POST + пароль на каждое действие + Origin-чек (пустой Origin → allow; иначе сравнение scheme/host/port с `HTTP_HOST`; `Sec-Fetch-Site: cross-site` → deny). Идемпотентный start/stop по точному совпадению cmdline.
5. **Пароли в `/tmp` / логирование кредов / sessionStorage-кэш** — Средний. env/stdin, права 0600, немедленное удаление; логировать только «RDP-сессия к host:port начата/завершена», без user/password; запрет кэша пароля.
6. **Clipboard-синк (экфильтрация в обе стороны)** — Средний. Отключить `rdpClipboardChanged` или явное предупреждение.
7. **iframing / mixed content** — Низкий. Cross-origin iframe по умолчанию разрешён (у grdpwasm нет X-Frame-Options). Проблемы только при HTTPS-панели (тогда http-iframe заблокируется) и фокус клавиатуры/pointer-lock.
8. **XSS при выводе конфига** — Низкий/Средний. `escapeHtml`, JSON-ответы, `encodeURIComponent`; URL iframe строить только из `window.location.hostname` + числового порта.

#### Фронтенд (ewm-designer)
- Вкладка проходит 4 точки регистрации: `menu/menu.json`, ветка `loadTab()` в `entware.js:201-258`, `static.go` whitelist, `Install/install.sh:646` (список веб-файлов).
- Ленивая загрузка — паттерн SMART (`entware.js:234-239`): `await loadScript('/entware-manager/rdp.js?v=9')` + флаг `window.RDP_LOADED`; в блок очистки таймеров (`entware.js:204-209`) добавить `RDP.stopUpdates()`.
- Прецедент iframe: ttyd-вкладки уже встраивают cross-origin iframe на другой порт (`entware.js:511-513`, `549-551`); управление демоном start/stop — `controlTtyd` (`entware.js:829-842`). RDP копирует оба паттерна.
- iframe: `allow="fullscreen; pointer-lock; autoplay"` + `allowfullscreen` + подсказка «кликните внутри окна»; кнопка «Открыть в новой вкладке» с `rel="noopener noreferrer"` (в ttyd-коде её нет — новый код сразу делаем правильно).
- URL iframe собирается из `BASE_URL` (`entware.js:5`, `window.location.protocol + '//' + window.location.hostname`) + настроенного порта прокси — без хардкода IP/порта.
- Стили — только CSS-переменные (см. таблицу в разделе 6); классы `.rdp-*`. `style.css?v=30`, `entware.js?v=12`.
- Иконки RDP в `icons.svg` нет — на первый релиз reuse `icon-terminal`/`icon-vpn` (без правки спрайта; иначе глобальная замена `icons.svg?v=2 → 3`).
- Кэш-версии: `?v=N` поднимать при любом изменении файла (правило скила `ewm-frontend-edit`).

#### Сборка/деплой (ewm-busybox)
- WASM: реалистичный размер `main.wasm` **8–20 МБ**, `wasm_exec.js` ~1.5 МБ. lighttpd 1.4.82 таймауты не мешают; MIME `.wasm => application/wasm` уже в `mime.conf` (проверено). entware-server (Go) отдаёт `.wasm` корректно (Go ≥1.17).
- Прокси: статический бинарник (`CGO_ENABLED=0`), RSS ~10–25 МБ, CPU ~0. Совместим с ядром 4.9 aarch64.
- Управление: **init.d-скрипт `S90grdp-proxy`** (по образцу `Install/S80entware-server`) вместо голого exec.Command — появится во вкладке «Сервисы» и под watchdog'ом. Go-хендлер запускает по паттерну `handleWrapperStart()`.
- BusyBox: `ss` НЕТ — проверки портов через `netstat -tln`; `curl` есть; скрипты ash/POSIX без bash-измов, только `find_pids()`/`pid_is_alive()` из `lib/common.sh`.
- `grdp-proxy` — НЕ класть в `cgi-bin/go/<arch>/` (попадёт под upx и install.sh раскидает как CGI). Отдельная сборка в `deploy/bin/<arch>/grdp-proxy`.
- Go 1.24+ для WASM: локально 1.22.2 + `GOTOOLCHAIN=auto` (авто-скачивание) — проверено, сеть до github есть.
- Размеры релизов: +8–20 МБ WASM на архив. На USB (1.5 ГБ свободно на dev) — ок; на JFFS2 (128–256 МБ) — существенно.

---

## 3. Решения пользователя (2026-08-10)

| Вопрос | Решение |
|---|---|
| Как проксировать WebSocket | **Свой веб-сервер** (вариант B): `grdp-proxy` сам раздаёт статику + WS на своём порту, без reverse-proxy через 8087. Пилот сначала. |
| Объём работ | **Пилот → архитектура → реализация**: шаг 1 вручную проверить grdpwasm на роутере, шаг 2 зафиксировать архитектуру, шаг 3 полная реализация. |
| Статус в релизах | **Не смешивать с основным функционалом**: RDP изолирован (свой каталог/бинарник/веб-сервер), существующий функционал не трогается; отдельный модуль поставки не требуется. |

---

## 4. Этапность работ (полный план)

### Этап 0 — Подготовка (факты проверены)
- Сеть: `git ls-remote https://github.com/nakagami/grdpwasm` отвечает (master: `a9e7e92`), GitHub API доступен.
- Локальный Go 1.22.2, `GOTOOLCHAIN=auto` — доскачает 1.24 для WASM-сборки.
- Роутер: заняты 8080 (AdGuard Home), 8081, 8087 (панель); **9099 свободен (проверено)**.
- Роутерные утилиты: `jq` есть, `python3`/`curl` нет, `ss` нет, `netstat` есть.
- Вендор grdpwasm + grdp в `vendor/` (вне основного `go.mod`), чтобы не тащить зависимости в entware-manager.

### Этап 1 — Пилот (проверка связки вручную на роутере)
1. Клонировать grdpwasm в `/tmp/opencode/grdpwasm`.
2. Собрать **grdp-proxy** для arm64:
   ```
   cd grdpwasm/proxy && GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o /tmp/grdp-proxy .
   ```
   (upx -9 опционально)
3. Собрать **WASM-клиент**:
   ```
   cd grdpwasm && GOOS=js GOARCH=wasm go build -o /tmp/rdp-static/main.wasm .
   cp $(go env GOROOT)/lib/wasm/wasm_exec.js /tmp/rdp-static/
   ```
   + index.html из репо.
4. Задеплоить на роутер через `ssh cat >` (НЕ install.sh):
   - `grdp-proxy` → `/opt/web_entware/cgi-bin/go/`
   - статика → `/opt/web_entware/static/rdp/` (каталоги создать)
5. Запустить вручную:
   ```
   /opt/web_entware/cgi-bin/go/grdp-proxy -listen 127.0.0.1:9099 -static /opt/web_entware/static/rdp/
   ```
6. Проверить на роутере:
   - `curl -sI http://127.0.0.1:9099/` → 200, Content-Type статики.
   - WS-хэндшейк `/ws` (соединение открывается).
   - Если есть ПК в LAN — реальное RDP-подключение; иначе — открытие клиента и подключение к любому host:port.
7. Проверить `X-Frame-Options`/CSP у grdp-proxy (должно не блокировать iframe) и MIME `.wasm`.
8. **Критерий выхода из пилота:** клиент загружается в браузере с 8087 (iframe на 9099), прокси стабилен. Записать фактические версии/размеры/поведение в DEVLOG.

### Этап 2 — Архитектура (после успешного пилота)
- Зафиксировать решения по разделу «Безопасность» (см. п.2 ewm-sec) — что вошло, что отложено.
- Определить интерфейс `go/internal/rdp` (функции, PID-файл `/opt/var/run/entware-rdp.pid`, атомарность).
- Определить эндпоинты и их методы (rdp_status/rdp_start/rdp_stop/rdp_config — POST для мутаций).
- Схема защиты: общий `go/internal/auth` (fail-closed) + Origin-чек + проброс `HTTP_ORIGIN` в `cgi.go`.
- Формат конфига `rdp_config.json`: host/port/domain/размер экрана (без паролей), порт прокси (дефолт 9099, сменный через rdp_config.json).

### Этап 3 — Реализация (полный список мест интеграции)
См. таблицу в разделе 5 (все места маппинга) + фронтенд (раздел 6) + сборка/установка (раздел 7).

### Этап 4 — Проверка
- `make ci` (check → lint → test), `gofmt -l .` пусто, shellcheck clean.
- Сборка arm64, деплой на роутер, проверка всех эндпоинтов + iframe 200 + существующий функционал не тронут.
- Идемпотентность start/stop, PID-файл, поведение при reboot (протухший PID → статус stopped).

---

## 5. Полная таблица мест интеграции (архитектура, бинарник `entware-rdp`, flat)

| # | Файл | Что добавить |
|---|------|--------------|
| 1 | `cgi-bin/go.cgi` (case-ветки) | `rdp_status\|rdp_start\|rdp_stop\|rdp_config) ENDPOINT="$name" exec "$(go_bin rdp)" ;;` |
| 2 | `go/internal/server/cgi.go` (`flatDispatch`) | `"rdp_status":"rdp", "rdp_start":"rdp", "rdp_stop":"rdp", "rdp_config":"rdp"` |
| 3 | `build-deploy.sh` (цикл компиляции, ~стр.74) | `entware-rdp` в `for cmd in ...` |
| 4 | `build-deploy.sh` (flat-симлинки, ~стр.104) | `rdp_config rdp_start rdp_status rdp_stop` |
| 5 | `Install/install.sh` (`GO_BINS`) | `entware-rdp`; **и счётчик `[ $GO_OK -eq 8 ]` → `-eq 9`** (~стр.626/638) |
| 6 | `Install/install.sh` (список веб-файлов, ~стр.646) | `rdp.js` |
| 7 | `go/internal/server/static.go` (`staticWhitelist`) | `"/rdp.js": true` |
| 8 | `go/internal/server/cgi.go` (`buildCGIEnv`) | проброс `HTTP_ORIGIN` (+ `HTTP_SEC_FETCH_SITE`) — общий код |
| 9 | `deploy/menu/menu.json` | `{ "tab": "rdp", "icon": "rdp", "text": "RDP" }` |
| 10 | `entware.js` (`loadTab`) | ветка rdp + `RDP.stopUpdates()` в блок очистки таймеров |
| 11 | `Install/install.sh` (блок копирования) | mkdir bin/static; grdp-proxy, main.wasm, wasm_exec.js; `S90grdp-proxy` в `/opt/etc/init.d/` |
| 12 | `index.html` | кэш-версии `style.css?v=30`, `entware.js?v=12` |
| 13 | `doc/CHANGELOG.md` + `version.json` | версия, описание (RULES п.6-7) |
| 14 | Новые файлы | `go/cmd/entware-rdp/main.go`, `go/internal/rdp/*.go`, `rdp.js`, `style.css` (.rdp-*), `Install/S90grdp-proxy`, `rdp_config.json` |

---

## 6. Фронтенд: вкладка RDP (детали)

### Регистрация вкладки
1. `menu/menu.json`: `{ "tab": "rdp", "icon": "rdp", "text": "RDP" }` (fallback-меню в `menu/menu.js:23-34`).
2. `entware.js loadTab()` — после SMART-ветки (стр.234-239):
   ```js
   if (tabName === 'rdp') {
       if (!window.RDP_LOADED) { await loadScript('/entware-manager/rdp.js?v=9'); window.RDP_LOADED = true; }
       RDP.init(); Menu.setActiveTab(tabName); return;
   }
   ```
3. В блок очистки таймеров (стр.204-209): `if (typeof RDP !== 'undefined' && RDP.stopUpdates) RDP.stopUpdates();`

### Модуль rdp.js (по образцу smart.js: `const RDP = { intervalId, init, stopUpdates, renderHTML, loadStatus }`)
- Форма: host/port/domain/размер экрана (пароль НЕ сохраняется, вводится при подключении).
- Статус-панель: запущен/остановлен + PID + порт (поллинг, останавливается в stopUpdates).
- Кнопки start/stop — `apiPost('/rdp_control.cgi', formData)` по паттерну `controlTtyd` (`entware.js:829-842`).
- iframe: `src = BASE_URL + ':' + proxyPort`, атрибуты `allow="fullscreen; pointer-lock; autoplay"` `allowfullscreen`; кнопка «Открыть в новой вкладке» с `rel="noopener noreferrer"`.
- XSS: все выводы через `escapeHtml()`; URL только из `window.location.hostname` + числовой порт (`parseInt` + диапазон 1-65535); пароль не выводить обратно.

### Стили (.rdp-*), только CSS-переменные
| Назначение | Светлая `:root` | Ночная `html.night` |
|---|---|---|
| Акцент/кнопки | `--accent`, `--btn-gradient`, `--btn-text`, `--btn-shadow` | 156-163, 186-189 |
| Текст | `--text-primary`, `--text-secondary`, `--text-muted` | 153-155 |
| Фоны | `--content-bg`, `--modal-bg`, `--pre-bg`, `--command-block-bg` | 150-172, 184 |
| Поля | `--input-bg`, `--input-border`, `--input-focus` | 168-170 |
| Бордеры | `--border-color`, `--link-card-border` | 164, 182-183 |
| Статусы | `--stat-normal-*`, `--stat-warning-*`, `--stat-critical-*` | 176-181 |

---

## 7. Сборка и установка (детали)

### Makefile / build-deploy.sh
- `entware-rdp` — в цикл компиляции `build-deploy.sh` (для всех 3 arch) + в symlink-список flat-эндпоинтов.
- `grdp-proxy` — отдельная сборка в `deploy/bin/<arch>/grdp-proxy` (все 3 arch), НЕ в cgi-bin/go.
- WASM (`GOOS=js GOARCH=wasm`) — один раз, в `deploy/static/rdp/`.
- Go 1.24+ для WASM: `GOTOOLCHAIN=go1.24.x go build ...` (авто-скачивание).

### Install/install.sh (официальная установка)
```
mkdir -p /opt/web_entware/cgi-bin/go /opt/web_entware/static/rdp
cp grdp-proxy  → /opt/web_entware/cgi-bin/go/
cp main.wasm wasm_exec.js index.html → /opt/web_entware/static/rdp/
ln -sf /opt/web_entware/Install/S90grdp-proxy /opt/etc/init.d/S90grdp-proxy (+chmod 755 Install/S90)
```
- В проверки: `entware-rdp` в `GO_BINS`, отдельно `grdp-proxy`.
- Dev-деплой — через `ssh cat >` (скил `ewm-deploy-router`), НЕ install.sh (стирает конфиги).

### S90grdp-proxy (init.d, ash/POSIX)
- По образцу `Install/S80entware-server`: pid-файл `/opt/var/run/grdp-proxy.pid`, start/stop/restart/status идемпотентны, аргументы `-listen :<порт> -static <dir>`.
- Без `pkill`/`pgrep`/`kill -0` — `find_pids()`/`pid_is_alive()` из `lib/common.sh`.
- Проверка порта при старте — `netstat -tln` (ss нет).

---

## 8. Безопасность: обязательный минимум (до релиза)

1. Общий `go/internal/auth` — fail-closed проверка пароля + `AuthRequired()` + Origin-helper.
2. Защита `auth_config.cgi` — смена/отключение пароля только после ввода текущего.
3. `rdp_*` — только POST + пароль на каждое действие + Origin-чек; идемпотентный start/stop по точному совпадению cmdline.
4. `rdp_config.json` — без паролей, без настраиваемого staticDir/port; валидация цели; белый список сохранённых целей.
5. `exec.Command` фиксированными аргументами (никакого `/bin/sh -c`); никаких паролей в `/tmp`-файлах и логах.
6. Прокси: bind на LAN-интерфейс, одноразовый токен, валидация `target`, CSP `frame-ancestors`; отключить или предупредить про clipboard-синк.
7. Проброс `HTTP_ORIGIN`/`HTTP_SEC_FETCH_SITE` в `buildCGIEnv` (`cgi.go:180-207`).

---

## 9. Критерии успешной реализации (ТЗ + аудит)

- [ ] `entware-rdp` собран и работает на роутере aarch64; grdp-proxy под управлением S90.
- [ ] Вкладка RDP появилась в меню, ленивая загрузка rdp.js.
- [ ] Через вкладку можно запустить/остановить прокси (POST + пароль + Origin).
- [ ] Веб-RDP-клиент (grdpwasm) подключается к ПК (iframe на :9099 / отдельная вкладка).
- [ ] Существующий функционал менеджера не сломан (единая структура: всё в `/opt/web_entware/`).
- [ ] `make ci` чисто, gofmt пусто, деплой проверен на роутере.
- [ ] Сборка `make all` включает entware-rdp и grdp-proxy для всех архитектур; WASM — один раз.

---

## 10. Риски и ограничения

- RDP-функционал grdp — базовый (RDP 5.x–8.x, без RemoteFX/multimon/smartcard/MS-RDPEDYC). Ожидания в UI ограничить.
- WASM +8–20 МБ в релизах — приемлемо на USB, существенно на JFFS2.
- Без TLS в WebSocket креды RDP по сети LAN в открытом виде (в RDP-протоколе с TLS-безопасностью — защищены; при legacy-RC4 — нет). Рекомендация: только доверенная LAN, SSH-туннель извне. (Реализовано: grdp-proxy слушает только 127.0.0.1, доступ через панель `/rdp/`; извне — Keenetic Remote/KeenDNS даёт HTTPS.)
- Открытый релей закрыт: `-allow-target <host:port>` из `rdp_config.json`, прочие `target=` в `/ws` → 403.
- iframe перестанет работать при HTTPS-панели (mixed content) — решено reverse-proxy подпутей через порт 8087 (одноязыковой origin).
- `GOTOOLCHAIN` требует сети при первой сборке WASM.
