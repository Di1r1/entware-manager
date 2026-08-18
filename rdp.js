// Entware Manager - RDP-модуль (веб-RDP-клиент grdpwasm)
// Версия: 1.0 (интеграция UI, единый конфиг rdp_config.json)
// Дата: 2026-08-10
//
// Изолированный модуль: отдельный файл, свой конфиг, не зависит от других вкладок.
// Интеграция через общие механизмы панели (lib/utils.js, CSS-переменные, меню).
// Порт прокси и пути берутся ТОЛЬКО из rdp_config.json (единая точка).

const RDP = {
    intervalId: null,
    cfg: null,
    frameUrl: '',

    init() {
        this.stopUpdates();
        this.renderHTML();
        this.loadConfig();
        window.addEventListener('resize', () => this.fitFrame());
    },

    stopUpdates() {
        if (this.intervalId) {
            clearInterval(this.intervalId);
            this.intervalId = null;
        }
    },

    renderHTML() {
        contentDiv.innerHTML = `
            <h2 style="display: flex; align-items: center; gap: 8px;">
                <span class="stat-icon" style="width: 28px; height: 28px;">
                    <svg class="icon" width="28" height="28">
                        <use href="/entware-manager/icons.svg?v=4#icon-vpn"/>
                    </svg>
                </span>
                RDP
            </h2>
            <div class="rdp-panel" id="rdpPanel">
                <div class="rdp-toolbar">
                    <div class="rdp-status" id="rdpStatus">
                        <p class="rdp-meta">Загрузка конфигурации...</p>
                    </div>
                    <div class="rdp-actions">
                        <button id="rdpStartBtn" class="packages-delete-btn rdp-btn-start" disabled>
                            <svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=4#icon-play"/></svg> Запустить прокси
                        </button>
                        <button id="rdpStopBtn" class="packages-delete-btn rdp-btn-stop" disabled>
                            <svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=4#icon-stop"/></svg> Остановить
                        </button>
                        <a id="rdpOpenLink" href="#" target="_blank" rel="noopener noreferrer" class="packages-delete-btn rdp-btn-open" style="display:none;">
                            <svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=4#icon-link"/></svg> Открыть в новой вкладке
                        </a>
                    </div>
                </div>
                <div id="rdpFrameWrap" style="display:none;">
                    <iframe id="rdpFrame" width="100%"
                        style="border: 1px solid var(--border-color); border-radius: 8px; background: var(--pre-bg);"
                        allow="fullscreen; autoplay; clipboard-read; clipboard-write"></iframe>
                    <p class="rdp-hint">
                        Нажмите внутри окна, чтобы захватить клавиатуру и мышь. Раскладка: Windows + Space.
                    </p>
                </div>
            </div>
        `;

        document.getElementById('rdpStartBtn').addEventListener('click', () => this.startProxy());
        document.getElementById('rdpStopBtn').addEventListener('click', () => this.stopProxy());
        document.getElementById('rdpOpenLink').addEventListener('click', function() {
            this.href = RDP.frameUrl;
            this.setAttribute('target', '_blank');
        });
    },

    async loadConfig() {
        try {
            const resp = await fetch('/entware-manager/rdp_config.json?_=' + Date.now());
            if (!resp.ok) throw new Error(resp.statusText);
            this.cfg = await resp.json();
            this.buildFrameUrl();
            this.loadStatus();
        } catch (err) {
            const st = document.getElementById('rdpStatus');
            if (st) st.innerHTML = '<p class="error">Не удалось прочитать rdp_config.json: ' + escapeHtml(err.message) + '</p>';
        }
    },

    buildFrameUrl() {
        // Клиент grdpwasm загружается с того же origin панели (reverse-proxy /rdp/):
        // WS он строит как location.host + /ws?target=… сам, статику тянет относительно.
        // Прямой порт прокси (9099) наружу не публикуем — только через панель.
        this.frameUrl = window.location.protocol + '//' + window.location.host + '/rdp/?v=18';
    },

    // Статус от бэкенда rdp_status.cgi (PID, порт, enabled).
    async loadStatus() {
        try {
            const data = await apiGet('/rdp_status.cgi');
            const running = data.state === 'running';
            this.renderStatus(running, data.port || (this.cfg && this.cfg.proxy_port), data);
            this.bindProxyControl(running);
            return;
        } catch (e) {
            // бэкенд недоступен — состояние неизвестно
            const st = document.getElementById('rdpStatus');
            if (st) st.innerHTML = '<p class="rdp-meta">Статус: <code style="background: var(--pre-bg); padding: 2px 6px; border-radius: 4px;">rdp_status.cgi</code> недоступен</p>';
            this.bindProxyControl(false);
        }
    },

    renderStatus(running, port, data) {
        const st = document.getElementById('rdpStatus');
        if (!st) return;
        const cfg = this.cfg || {};

        if (running) {
            const proxyPort = (data && data.port) || port || cfg.proxy_port;
            const pid = data && data.pid;
            st.innerHTML = `
                <div style="display: flex; align-items: center; gap: 12px; flex-wrap: wrap;">
                    <span class="status-badge status-running">
                        <svg class="icon" width="12" height="12"><use href="/entware-manager/icons.svg?v=4#icon-check"/></svg>
                        Прокси запущен
                    </span>
                    <span class="rdp-meta" style="white-space: nowrap;">Порт: <b>${escapeHtml(String(proxyPort))}</b></span>
                    ${pid ? `<span class="rdp-meta" style="white-space: nowrap;">PID: ${escapeHtml(String(pid))}</span>` : ''}
                    <span class="rdp-meta rdp-url" id="rdpClientUrl" style="cursor:pointer; white-space:nowrap;" title="Кликните, чтобы показать полный адрес клиента">Клиент: показать</span>
                </div>
            `;
            var urlEl = document.getElementById('rdpClientUrl');
            if (urlEl) {
                var full = RDP.frameUrl;
                urlEl.addEventListener('click', function() {
                    if (this.getAttribute('data-full') === '1') {
                        this.textContent = 'Клиент: показать';
                        this.setAttribute('data-full', '0');
                    } else {
                        this.textContent = 'Клиент: ' + full;
                        this.setAttribute('data-full', '1');
                    }
                });
            }
            this.showFrame(true);
        } else {
            st.innerHTML = `
                <div style="display: flex; align-items: center; gap: 12px; flex-wrap: nowrap;">
                    <span class="status-badge status-stopped">
                        <svg class="icon" width="12" height="12"><use href="/entware-manager/icons.svg?v=4#icon-cross"/></svg>
                        Прокси не запущен
                    </span>
                    <span class="rdp-meta" style="white-space: nowrap;">Порт: <b>${escapeHtml(String(port))}</b></span>
                    <span class="rdp-meta rdp-url">Запустите прокси, чтобы подключиться к ПК по RDP.</span>
                </div>
            `;
            this.showFrame(false);
        }
    },

    showFrame(show) {
        const wrap = document.getElementById('rdpFrameWrap');
        const link = document.getElementById('rdpOpenLink');
        const frame = document.getElementById('rdpFrame');
        if (!wrap) return;
        wrap.style.display = show ? 'block' : 'none';
        if (link) { link.style.display = show ? 'inline-block' : 'none'; }
        if (frame && show && frame.getAttribute('data-src') !== this.frameUrl) {
            frame.setAttribute('data-src', this.frameUrl);
            frame.src = this.frameUrl;
        }
        if (show) this.fitFrame();
    },

    // Подгоняет высоту iframe под окно без вертикального скролла панели.
    fitFrame() {
        const wrap = document.getElementById('rdpFrameWrap');
        const frame = document.getElementById('rdpFrame');
        if (!wrap || !frame || wrap.style.display === 'none') return;
        const top = wrap.getBoundingClientRect().top;
        const available = Math.max(300, window.innerHeight - top - 12);
        frame.style.height = available + 'px';
    },

    // bindProxyControl(running) — управление кнопками start/stop.
    bindProxyControl(running) {
        const startBtn = document.getElementById('rdpStartBtn');
        const stopBtn = document.getElementById('rdpStopBtn');
        if (!startBtn || !stopBtn) return;

        startBtn.disabled = !!running;
        stopBtn.disabled = !running;
        startBtn.title = running ? 'Прокси уже запущен' : 'Запустить RDP-прокси';
        stopBtn.title = running ? 'Остановить RDP-прокси' : 'Прокси не запущен';

        // поллинг статуса (останавливается в stopUpdates)
        if (!this.intervalId) {
            this.intervalId = setInterval(() => {
                this.loadStatus();
            }, 10000);
        }
    },

    async startProxy() {
        const startBtn = document.getElementById('rdpStartBtn');
        if (startBtn) startBtn.disabled = true;
        Modal.promptPassword('Введите пароль (раздел «Защита»)', async (password) => {
            if (!password) {
                if (startBtn) startBtn.disabled = false;
                return;
            }
            try {
                const data = await apiPost('/rdp_start.cgi', 'port=' + encodeURIComponent((this.cfg && this.cfg.proxy_port) || 9099) + '&password=' + encodeURIComponent(password));
                Toast.show(data.message || (data.status === 'ok' ? 'Прокси запущен' : 'Ошибка'), data.status !== 'ok');
                this.loadStatus();
            } catch (err) {
                Toast.show('Ошибка: ' + err.message, true);
                if (startBtn) startBtn.disabled = false;
            }
        });
    },

    async stopProxy() {
        Modal.promptPassword('Введите пароль (раздел «Защита»)', async (password) => {
            if (!password) return;
            try {
                const data = await apiPost('/rdp_stop.cgi', 'password=' + encodeURIComponent(password));
                Toast.show(data.message || (data.status === 'ok' ? 'Прокси остановлен' : 'Ошибка'), data.status !== 'ok');
                this.loadStatus();
            } catch (err) {
                Toast.show('Ошибка: ' + err.message, true);
            }
        });
    }
};

window.RDP = RDP;
