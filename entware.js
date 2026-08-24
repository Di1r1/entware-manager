// Entware Manager — разработчик Di1r1
// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
// Версия: 0.82 (исправления XSS, безопасность)
// Дата: 2026-04-06

const BASE_URL = window.location.protocol + '//' + window.location.hostname;
const CACHE_KEY = 'entware_available_packages';
const CACHE_TIME_KEY = 'entware_available_timestamp';
const CACHE_MAX_AGE = 3600 * 1000;
const CACHE_INSTALLED_KEY = 'entware_installed_packages';
const CACHE_INSTALLED_TIME = 'entware_installed_timestamp';
const CACHE_UPGRADABLE_KEY = 'entware_upgradable_packages';
const CACHE_UPGRADABLE_TIME = 'entware_upgradable_timestamp';
const CACHE_PKG_MAX_AGE = 60 * 1000;

let settingsInterval = null;
let servicesInterval = null;

let contentDiv, sidebar, menuToggle, collapseToggle;

function mapIconToId(icon) {
    const map = {
        '🌐': 'router', '📦': 'package', '🛡️': 'shield', '⬇️': 'download',
        '📊': 'chart', '⚙️': 'settings', '💻': 'terminal', '⛨': 'shield-alt',
        '🔗': 'link', '🗄️': 'disk', '📘': 'help', '🌙': 'moon', '☀️': 'sun',
        '▶': 'arrow-right', '📋': 'list', '🔄': 'refresh', '🧠': 'memory',
        '💾': 'disk', '🖥️': 'stats',
        '📧': 'email', '📷': 'camera', '🔍': 'search', '☁️': 'cloud',
        '👤': 'user', '📅': 'calendar', '📝': 'notes', '📡': 'rss',
        '🔒': 'lock', '🏠': 'home', '🎵': 'music', '🎬': 'video',
        '🌦️': 'weather', '📁': 'file', '🔐': 'vpn'
    };
    const id = map[icon];
    return id ? `icon-${id}` : 'icon-default';
}

const iconList = [
    { id: 'router', name: 'Роутер' },
    { id: 'package', name: 'Пакет' },
    { id: 'shield', name: 'Щит' },
    { id: 'shield-alt', name: 'Щит (альт.)' },
    { id: 'download', name: 'Загрузка' },
    { id: 'chart', name: 'График' },
    { id: 'settings', name: 'Настройки' },
    { id: 'terminal', name: 'Терминал' },
    { id: 'link', name: 'Ссылка' },
    { id: 'disk', name: 'Диск' },
    { id: 'help', name: 'Справка' },
    { id: 'moon', name: 'Луна' },
    { id: 'sun', name: 'Солнце' },
    { id: 'arrow-right', name: 'Стрелка' },
    { id: 'arrow-left', name: 'Стрелка влево' },
    { id: 'list', name: 'Список' },
    { id: 'refresh', name: 'Обновить' },
    { id: 'memory', name: 'Память' },
    { id: 'stats', name: 'Статистика' },
    { id: 'folder', name: 'Папка' },
    { id: 'services', name: 'Службы' },
    { id: 'process', name: 'Процесс' },
    { id: 'update', name: 'Обновление' },
    { id: 'email', name: 'Почта' },
    { id: 'camera', name: 'Камера' },
    { id: 'search', name: 'Поиск' },
    { id: 'cloud', name: 'Облако' },
    { id: 'user', name: 'Пользователь' },
    { id: 'calendar', name: 'Календарь' },
    { id: 'notes', name: 'Заметки' },
    { id: 'rss', name: 'RSS' },
    { id: 'vpn', name: 'VPN' },
    { id: 'file', name: 'Файл' },
    { id: 'music', name: 'Музыка' },
    { id: 'video', name: 'Видео' },
    { id: 'weather', name: 'Погода' },
    { id: 'lock', name: 'Замок' },
    { id: 'home', name: 'Домой' }
];

function handleResponsive() {
    const isMobile = window.innerWidth <= 800;
    if (isMobile) {
        if (sidebar.classList.contains('collapsed')) {
            sidebar.classList.remove('collapsed');
            if (collapseToggle) collapseToggle.innerHTML = '<svg class="icon" width="20" height="20"><use href="/entware-manager/icons.svg?v=6#icon-chevron-left"/></svg>';
        }
        localStorage.removeItem('sidebar_collapsed');
    } else {
        const savedCollapsed = localStorage.getItem('sidebar_collapsed');
        if (savedCollapsed === 'true' && !sidebar.classList.contains('collapsed')) {
            sidebar.classList.add('collapsed');
            if (collapseToggle) collapseToggle.innerHTML = '<svg class="icon" width="20" height="20"><use href="/entware-manager/icons.svg?v=6#icon-chevron-right"/></svg>';
        } else if (savedCollapsed === 'false' && sidebar.classList.contains('collapsed')) {
            sidebar.classList.remove('collapsed');
            if (collapseToggle) collapseToggle.innerHTML = '<svg class="icon" width="20" height="20"><use href="/entware-manager/icons.svg?v=6#icon-chevron-left"/></svg>';
        }
    }
}

function debounce(func, delay) {
    let timeout;
    return function() {
        clearTimeout(timeout);
        timeout = setTimeout(() => func.apply(this, arguments), delay);
    };
}

function init() {
    contentDiv = document.getElementById('content');
    sidebar = document.getElementById('sidebar');
    menuToggle = document.getElementById('menuToggle');
    collapseToggle = document.getElementById('collapseToggle');

    initTheme();

    // Проверка сессии: если пароль панели настроен, а сессии нет — показать вход.
    checkSessionAndStart();
}

function checkSessionAndStart() {
    fetch('/entware-cgi/session.cgi?_=' + Date.now())
        .then(r => r.json())
        .then(data => {
            if (data && data.authenticated) {
                document.body.classList.add('auth-ready');
                initPanel();
            } else {
                document.body.classList.add('login-shown');
                showLogin();
            }
        })
        .catch(() => {
            document.body.classList.add('auth-ready');
            initPanel();
        });
}

// Запуск виджетов панели после авторизации (температура, версия, проверка
// обновлений) — чтобы не делать гейченных запросов до входа в панель.
function startPanelWidgets() {
    if (typeof updateTemp === 'function') {
        updateTemp();
        setInterval(updateTemp, 30000);
    }
    if (typeof updateWifiTemp === 'function') {
        updateWifiTemp();
        setInterval(updateWifiTemp, 30000);
    }
    fetch('/entware-manager/version.json?_=' + Date.now())
        .then(r => r.json())
        .then(data => {
            window.APP_VERSION = data.version;
            document.title = `Entware Manager v${data.version}`;
            const footer = document.getElementById('mainFooter');
            if (footer) footer.innerHTML = `Entware Manager v${escapeHtml(data.version)} — интерфейс на базе CGI и вкладок. Разработчик: Di1r1`;
            const sidebarVersion = document.getElementById('sidebarVersion');
            if (sidebarVersion) {
                // Версия всегда кликабельна → ведёт на текущий релиз на GitHub.
                const vEsc = escapeHtml(data.version);
                sidebarVersion.innerHTML =
                    `<a href="https://github.com/Di1r1/entware-manager/releases/tag/v${vEsc}" target="_blank" rel="noopener" style="color:var(--text-muted);text-decoration:none;">v${vEsc}</a>`;
            }
            fetch('/entware-cgi/update_check.cgi').then(r => r.json()).then(upd => {
                if (upd.has_update && sidebarVersion) {
                    const curEsc = escapeHtml(upd.current);
                    const latEsc = escapeHtml(upd.latest);
                    sidebarVersion.innerHTML =
                        `<a href="https://github.com/Di1r1/entware-manager/releases/tag/v${curEsc}" target="_blank" rel="noopener" style="color:var(--text-muted);text-decoration:none;">v${curEsc}</a>` +
                        ` → <a href="https://github.com/Di1r1/entware-manager/releases/tag/v${latEsc}" target="_blank" rel="noopener" style="color:#2ecc71;text-decoration:none;">v${latEsc}</a>`;
                }
            }).catch(function(){});
        })
        .catch(e => { window.APP_VERSION = 'error'; console.error('Ошибка загрузки версии', e) });
}

function initPanel() {
    // Кнопка «Выйти» видна только когда пароль панели настроен (есть логин).
    const logoutBtn = document.getElementById('logoutBtn');
    if (logoutBtn) {
        fetch('/entware-cgi/session.cgi?_=' + Date.now())
            .then(r => r.json())
            .then(d => { if (d && d.authenticated) logoutBtn.style.display = 'block'; })
            .catch(() => {});
    }
    if (collapseToggle) {
        const collapsedState = localStorage.getItem('sidebar_collapsed');
        if (collapsedState === 'true') {
            sidebar.classList.add('collapsed');
            collapseToggle.innerHTML = '<svg class="icon" width="20" height="20"><use href="/entware-manager/icons.svg?v=6#icon-chevron-right"/></svg>';
        } else {
            collapseToggle.innerHTML = '<svg class="icon" width="20" height="20"><use href="/entware-manager/icons.svg?v=6#icon-chevron-left"/></svg>';
        }
        collapseToggle.addEventListener('click', () => {
            sidebar.classList.toggle('collapsed');
            const isCollapsed = sidebar.classList.contains('collapsed');
            localStorage.setItem('sidebar_collapsed', isCollapsed);
            collapseToggle.innerHTML = isCollapsed
                ? '<svg class="icon" width="20" height="20"><use href="/entware-manager/icons.svg?v=6#icon-chevron-right"/></svg>'
                : '<svg class="icon" width="20" height="20"><use href="/entware-manager/icons.svg?v=6#icon-chevron-left"/></svg>';
            if (!isCollapsed) {
                sidebar.classList.add('menu-animate');
                setTimeout(() => sidebar.classList.remove('menu-animate'), 1400);
            }
        });
    }

    if (menuToggle) {
        menuToggle.addEventListener('click', () => sidebar.classList.toggle('menu-open'));
    }

    Menu.init('#dynamic-menu').then(() => {
        let savedTab = localStorage.getItem('entware_active_tab');
        if (savedTab === 'available' || savedTab === 'updates') {
            savedTab = 'packages';
            localStorage.setItem('entware_active_tab', 'packages');
        }
        Menu.setActiveTab(savedTab || 'stats');
    });
    handleResponsive();
    window.addEventListener('resize', debounce(handleResponsive, 200));
    let savedTab = localStorage.getItem('entware_active_tab');
    if (savedTab === 'available' || savedTab === 'updates') {
        savedTab = 'packages';
        localStorage.setItem('entware_active_tab', 'packages');
    }
    loadTab(savedTab || 'stats');
    startPanelWidgets();
}

function showLogin() {
    const overlay = document.getElementById('loginOverlay');
    const pass = document.getElementById('loginPassword');
    const btn = document.getElementById('loginBtn');
    const err = document.getElementById('loginError');
    if (!overlay) return;
    // Показываем экран входа и гасим панель: снимаем auth-ready (скрывает
    // app-container через body:not(.auth-ready)) и ставим login-shown (видна
    // карточка логина через body:not(.login-shown) .login-card {visibility:hidden}).
    // Иначе при 401 после инвалидации сессии экран остаётся пустым.
    document.body.classList.remove('auth-ready');
    document.body.classList.add('login-shown');
    overlay.style.display = 'flex';
    if (pass) pass.focus();

    const doLogin = function() {
        if (!pass || !pass.value) return;
        if (btn) btn.disabled = true;
        apiPost('/login.cgi', 'password=' + encodeURIComponent(pass.value))
            .then(function(data) {
                if (data && data.status === 'ok') {
                    overlay.style.display = 'none';
                    location.reload();
                } else {
                    if (err) { err.style.display = 'block'; err.textContent = (data && data.message) || 'Неверный пароль'; }
                    if (pass) { pass.value = ''; pass.focus(); }
                    if (btn) btn.disabled = false;
                }
            })
            .catch(function(e) {
                if (err) { err.style.display = 'block'; err.textContent = 'Ошибка: ' + e.message; }
                if (btn) btn.disabled = false;
            });
    };

    if (btn) {
        btn.onclick = doLogin;
        btn.disabled = false;
    }
    if (pass) {
        pass.onkeydown = function(e) { if (e.key === 'Enter') doLogin(); };
    }
}

function doLogout() {
    apiPost('/logout.cgi', '')
        .then(function() { location.reload(); })
        .catch(function() { location.reload(); });
}

function updateThemeIcon() {
    const themeToggle = document.getElementById('themeToggle');
    if (!themeToggle) return;
    const isNight = document.documentElement.classList.contains('night');
    themeToggle.querySelector('use')?.setAttribute('href', '/entware-manager/icons.svg?v=6#' + (isNight ? 'icon-moon' : 'icon-sun'));
}

function buildThemePopup() {
    const popup = document.getElementById('themePopup');
    if (!popup || !window.Theme) return;
    const current = window.Theme.current();
    popup.innerHTML = '';
    window.Theme.THEMES.forEach(t => {
        const btn = document.createElement('button');
        btn.className = 'theme-swatch' + (t.id === current ? ' active' : '');
        btn.type = 'button';
        btn.dataset.theme = t.id;
        btn.innerHTML = '<span class="swatch-dot" style="background:' + t.color + ';" title="' + t.label + '"></span>';
        btn.addEventListener('click', () => {
            window.Theme.set(t.id, window.Theme.isNight());
            popup.classList.remove('show');
            updateThemeIcon();
            buildThemePopup();
        });
        popup.appendChild(btn);
    });
}

function initTheme() {
    const themeToggle = document.getElementById('themeToggle');
    const popup = document.getElementById('themePopup');
    if (window.Theme) window.Theme.init();
    updateThemeIcon();
    buildThemePopup();

    // Показ выбора цвета при удержании мыши на иконке >1 сек
    // (переключение день/ночь кликом — мгновенное, без задержки).
    let themeHoverTimer = null;
    let themeHideTimer = null;

    function showThemePopup() {
        buildThemePopup();
        if (popup) popup.classList.add('show');
    }
    function clearThemeHoverTimer() {
        if (themeHoverTimer) { clearTimeout(themeHoverTimer); themeHoverTimer = null; }
    }
    function clearThemeHideTimer() {
        if (themeHideTimer) { clearTimeout(themeHideTimer); themeHideTimer = null; }
    }
    // Скрытие с задержкой: даёт мыши перейти с иконки на попап, не пряча его.
    function scheduleThemeHide() {
        clearThemeHoverTimer();
        clearThemeHideTimer();
        themeHideTimer = setTimeout(() => {
            if (popup) popup.classList.remove('show');
        }, 300);
    }

    if (themeToggle) {
        themeToggle.addEventListener('mouseenter', () => {
            clearThemeHideTimer();
            clearThemeHoverTimer();
            themeHoverTimer = setTimeout(showThemePopup, 1000);
        });
        themeToggle.addEventListener('mouseleave', () => {
            // Если попап ещё не показан (убрали до 1 сек) — просто отменить таймер.
            if (!popup || !popup.classList.contains('show')) {
                clearThemeHoverTimer();
                return;
            }
            // Попап показан — скрываем с задержкой, чтобы успеть перейти на него.
            scheduleThemeHide();
        });
        themeToggle.addEventListener('click', () => {
            clearThemeHoverTimer();
            clearThemeHideTimer();
            if (window.Theme) {
                window.Theme.set(window.Theme.current(), !window.Theme.isNight());
            }
            updateThemeIcon();
            if (popup) popup.classList.remove('show');
        });
    }
    if (popup) {
        // При переходе мыши с иконки на попап — не скрывать.
        popup.addEventListener('mouseenter', () => {
            clearThemeHoverTimer();
            clearThemeHideTimer();
        });
        popup.addEventListener('mouseleave', () => {
            clearThemeHideTimer();
            popup.classList.remove('show');
        });
        popup.addEventListener('click', (e) => {
            if (e.target.closest('.theme-swatch')) popup.classList.remove('show');
        });
    }
}

// Открыть вкладку «Справка» и проскроллить к разделу Telegram-бота.
// Используется ссылкой из блока настроек Telegram.
async function openHelpTG() {
    try { await loadTab('help'); } catch (e) {}
    setTimeout(function () {
        var el = document.getElementById('tg-help');
        if (el && el.scrollIntoView) el.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }, 250);
    return false;
}

async function loadTab(tabName) {    const ver = window.APP_VERSION || 'loading...';
    console.log(`[v${ver}] Загрузка вкладки:`, tabName);
    if (settingsInterval) clearInterval(settingsInterval);
    settingsInterval = null;
    if (servicesInterval) clearInterval(servicesInterval);
    servicesInterval = null;
    if (typeof MONITOR !== 'undefined' && MONITOR.stopUpdates) MONITOR.stopUpdates();
    if (typeof SMART !== 'undefined' && SMART.stopUpdates) SMART.stopUpdates();
    if (typeof RDP !== 'undefined' && RDP.stopUpdates) RDP.stopUpdates();

    if (tabName === 'packages' || tabName === 'available' || tabName === 'updates') {
        renderPackagesTab(tabName);
        Menu.setActiveTab('packages');
        return;
    }
    if (tabName === 'processes') { renderProcessesTab(); Menu.setActiveTab(tabName); return; }
    if (tabName === 'terminal') { renderTerminalTab(); Menu.setActiveTab(tabName); return; }
    if (tabName === 'settings') { renderSettingsTab(); Menu.setActiveTab(tabName); return; }
    if (tabName === 'system-services') { loadSystemServicesTab(); Menu.setActiveTab(tabName); return; }
    if (tabName === 'monitor') { loadMonitorTab(); Menu.setActiveTab(tabName); return; }
    if (tabName === 'logs') { loadLogsTab(); Menu.setActiveTab(tabName); return; }
    if (tabName === 'network') { loadNetworkTab(); Menu.setActiveTab(tabName); return; }
    if (tabName === 'bridge') { renderBridgeTab(); Menu.setActiveTab(tabName); return; }
    if (tabName === 'help') {
        contentDiv.innerHTML = '<p>Загрузка...</p>';
        try {
            const response = await apiFetch('/help.cgi');
            const html = await response.text();
            contentDiv.innerHTML = html;
            initHelpSearch();
            Menu.setActiveTab(tabName);
        } catch (err) {
            contentDiv.innerHTML = `<p class="error">Ошибка загрузки: ${escapeHtml(err.message)}</p>`;
            Menu.setActiveTab(tabName);
        }
        return;
    }
    if (tabName === 'smart') {
        if (!window.SMART_LOADED) {
            await loadScript('/entware-manager/smart.js?v=11');
            window.SMART_LOADED = true;
        }
        SMART.init(); Menu.setActiveTab(tabName); return;
    }
    if (tabName === 'rdp') {
        if (!window.RDP_LOADED) {
            await loadScript('/entware-manager/rdp.js?v=26');
            window.RDP_LOADED = true;
        }
        RDP.init(); Menu.setActiveTab(tabName); return;
    }

    contentDiv.innerHTML = '<p>Загрузка...</p>';
    try {
        const response = await apiFetch('/' + tabName + '.cgi');
        const html = await response.text();
        contentDiv.innerHTML = html;
        if (tabName === 'stats') {
            initStatsTabs();
            loadNetworkStatus();
            setTimeout(() => { renderLinksOnStats(); renderBridgeCardsOnStats(); enableTableSorting(); }, 100);
        }
        Menu.setActiveTab(tabName);
    } catch (err) {
        contentDiv.innerHTML = `<p class="error">Ошибка загрузки: ${escapeHtml(err.message)}</p>`;
        Menu.setActiveTab(tabName);
    }
}

function initHelpSearch() {
    const input = document.getElementById('helpSearch');
    if (!input) return;
    const content = document.getElementById('helpContent');
    if (!content) return;
    const sections = content.querySelectorAll('h3');

    input.addEventListener('input', () => {
        const q = input.value.toLowerCase().trim();
        sections.forEach(h3 => {
            let match = !q || h3.textContent.toLowerCase().includes(q);
            let el = h3.nextElementSibling;
            const items = [h3];
            while (el && el.tagName !== 'H3') { items.push(el); el = el.nextElementSibling; }
            items.forEach(el2 => {
                if (!match && q && el2.textContent.toLowerCase().includes(q)) match = true;
            });
            items.forEach(el2 => el2.style.display = !q || match ? '' : 'none');
        });
    });
}

function initStatsTabs() {
    const tabButtons = document.querySelectorAll('.tab-button');
    if (!tabButtons.length) return;
    const switchTab = (event) => {
        const targetTab = event.currentTarget.dataset.tab;
        document.querySelectorAll('.tab-content').forEach(c => c.classList.remove('active'));
        tabButtons.forEach(b => b.classList.remove('active'));
        document.getElementById(`tab-${targetTab}`).classList.add('active');
        event.currentTarget.classList.add('active');
    };
    tabButtons.forEach(btn => {
        btn.removeEventListener('click', switchTab);
        btn.addEventListener('click', switchTab);
    });
}

async function showPackageInfo(pkg) {
    Modal.loading(`Загрузка информации о ${pkg}...`);
    try {
        const data = await apiGet('/api.cgi?action=info&package=' + encodeURIComponent(pkg));
        if (data.error) Modal.error(data.error);
        else Modal.info(`<pre>${escapeHtml(data.info)}</pre>`, `Пакет: ${pkg}`);
    } catch (err) {
        Modal.error('Ошибка запроса: ' + err.message);
    }
}

let pkgInstalledData = [];
let pkgUpgradableData = [];
let pkgAvailableData = null;
let pkgCurrentFilter = 'installed';

const PKG_FILTERS = [
    { id: 'all', text: 'Все' },
    { id: 'installed', text: 'Установленные' },
    { id: 'updates', text: 'Обновления' },
    { id: 'available', text: 'Доступные' }
];

function renderPackagesTab(initialFilter) {
    const filter = (initialFilter === 'available' || initialFilter === 'updates')
        ? (initialFilter === 'available' ? 'available' : 'updates')
        : 'installed';
    pkgCurrentFilter = filter;
    const html = `
        <h2 style="display: flex; align-items: center; gap: 8px;">
            <span class="stat-icon" style="width: 28px; height: 28px;">
                <svg class="icon" width="28" height="28"><use href="/entware-manager/icons.svg?v=6#icon-package"/></svg>
            </span>
            <span id="pkg-title">Пакеты</span>
        </h2>
        <div class="pkg-actions">
            <button id="runUpdateBtn" class="packages-delete-btn pkg-action-btn" style="background:#4a5568;"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=6#icon-refresh"/></svg> Обновить списки пакетов</button>
            <button id="upgradeAllBtn" class="packages-delete-btn pkg-action-btn" style="background:#e67e22;"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=6#icon-update"/></svg> Обновить все пакеты</button>
        </div>
        <div id="update-result" style="margin-bottom: 20px;"></div>
        <div class="pkg-tabs" id="pkg-tabs">
            ${PKG_FILTERS.map(f => `<button class="packages-delete-btn pkg-filter-btn" data-filter="${f.id}">${f.text}${f.id === 'updates' ? ' <span id="pkg-upd-count" style="display:none;"></span>' : ''}</button>`).join('')}
        </div>
        <div style="display: flex; gap: 10px; align-items: center; margin-bottom: 24px;">
            <div class="search-container" style="display: flex; gap: 8px; align-items: center; flex: 1; background: var(--input-bg); border: 2px solid var(--input-border); border-radius: 40px; padding: 0 12px; transition: border-color 0.3s ease, box-shadow 0.3s ease;">
                <svg class="icon" width="18" height="18" style="color: var(--text-muted);"><use href="/entware-manager/icons.svg?v=6#icon-search"/></svg>
                <input type="text" id="searchPkg" placeholder="Поиск по названию..." style="flex: 1; background: transparent; border: none; outline: none; padding: 14px 0; font-size: 16px; color: var(--text-primary);">
            </div>
        </div>
        <div id="pkg-table-container" class="packages-table-wrapper"><div class="loading-spinner"></div></div>
    `;
    contentDiv.innerHTML = html;
    document.getElementById('runUpdateBtn').addEventListener('click', runPkgUpdate);
    document.getElementById('upgradeAllBtn').addEventListener('click', upgradeAll);
    document.querySelectorAll('.pkg-filter-btn').forEach(btn => {
        btn.addEventListener('click', () => setPkgFilter(btn.dataset.filter));
    });
    setPkgFilter(filter);
}

function setPkgFilter(filter) {
    pkgCurrentFilter = filter;
    document.querySelectorAll('.pkg-filter-btn').forEach(btn => {
        btn.classList.toggle('active', btn.dataset.filter === filter);
    });
    loadPkgTable();
}

let pkgLoadSeq = 0;

function pkgCacheGet(key, timeKey, ttl) {
    try {
        const cached = localStorage.getItem(key);
        const timestamp = localStorage.getItem(timeKey);
        const now = Date.now();
        if (cached && timestamp && (now - parseInt(timestamp) < ttl)) {
            return JSON.parse(cached);
        }
    } catch (e) { /* битый кэш — игнорируем */ }
    return null;
}

function pkgCacheSet(key, timeKey, data) {
    try {
        localStorage.setItem(key, JSON.stringify(data));
        localStorage.setItem(timeKey, Date.now().toString());
    } catch (e) { /* квота localStorage — не критично */ }
}

function pkgCacheClear(key, timeKey) {
    localStorage.removeItem(key);
    localStorage.removeItem(timeKey);
}

async function fetchInstalledWithRetry(seq) {
    for (let attempt = 0; attempt < 3; attempt++) {
        try {
            const data = await apiGet('/installed.cgi');
            // Успешный ответ (даже пустой []) — валиден: не ретраим.
            return Array.isArray(data) ? data : [];
        } catch (e) { /* временная ошибка — повторим */ }
        if (seq !== pkgLoadSeq) return null;
        if (attempt < 2) await new Promise(r => setTimeout(r, 800));
    }
    return null;
}

async function loadPkgData(forceRefresh, seq) {
    let installed = forceRefresh ? null : pkgCacheGet(CACHE_INSTALLED_KEY, CACHE_INSTALLED_TIME, CACHE_PKG_MAX_AGE);
    let upgradable = forceRefresh ? null : pkgCacheGet(CACHE_UPGRADABLE_KEY, CACHE_UPGRADABLE_TIME, CACHE_PKG_MAX_AGE);

    if (installed && !installed.length) installed = null;
    if (upgradable && !upgradable.length) upgradable = null;

    if (!installed) {
        installed = await fetchInstalledWithRetry(seq);
        if (seq !== pkgLoadSeq) return;
        if (installed && installed.length) pkgCacheSet(CACHE_INSTALLED_KEY, CACHE_INSTALLED_TIME, installed);
    }
    if (!upgradable) {
        try {
            upgradable = await apiGet('/upgradable.cgi');
        } catch (e) {
            upgradable = null;
        }
        if (seq !== pkgLoadSeq) return;
        if (upgradable && upgradable.length) pkgCacheSet(CACHE_UPGRADABLE_KEY, CACHE_UPGRADABLE_TIME, upgradable);
    }

    if (seq !== pkgLoadSeq) return;
    pkgInstalledData = installed || [];
    pkgUpgradableData = (upgradable || []).filter(p => p.package && p.current && p.new && p.package !== 'undefined');

    if (pkgCurrentFilter === 'all' || pkgCurrentFilter === 'available') {
        const cached = localStorage.getItem(CACHE_KEY);
        const timestamp = localStorage.getItem(CACHE_TIME_KEY);
        const now = Date.now();
        if (!forceRefresh && cached && timestamp && (now - parseInt(timestamp) < CACHE_MAX_AGE)) {
            pkgAvailableData = JSON.parse(cached);
        } else {
            pkgAvailableData = await apiGet('/available.cgi');
            localStorage.setItem(CACHE_KEY, JSON.stringify(pkgAvailableData));
            localStorage.setItem(CACHE_TIME_KEY, now.toString());
        }
        if (seq !== pkgLoadSeq) return;
    }
}

function buildPkgRows() {
    const upgradableMap = {};
    pkgUpgradableData.forEach(p => { if (p.package) upgradableMap[p.package] = p; });

    let rows = [];
    if (pkgCurrentFilter === 'updates') {
        const installedMap = {};
        pkgInstalledData.forEach(p => { if (p.package) installedMap[p.package] = p; });
        rows = pkgUpgradableData.map(p => {
            const inst = installedMap[p.package];
            return {
                name: p.package,
                current: p.current,
                new: p.new,
                installed_date: inst ? (inst.installed_date || '') : '',
                status: 'upgrade'
            };
        });
    } else if (pkgCurrentFilter === 'installed') {
        rows = pkgInstalledData.map(p => {
            const up = upgradableMap[p.package];
            return {
                name: p.package,
                version: p.version,
                installed_date: p.installed_date || '',
                status: up ? 'upgrade' : 'installed'
            };
        });
    } else {
        const installedMap = {};
        pkgInstalledData.forEach(p => { if (p.package) installedMap[p.package] = p; });
        if (!pkgAvailableData) {
            rows = pkgInstalledData.map(p => {
                const up = upgradableMap[p.package];
                return { name: p.package, version: p.version, installed_date: p.installed_date || '', status: up ? 'upgrade' : 'installed' };
            });
        } else {
            const source = pkgCurrentFilter === 'available'
                ? pkgAvailableData.filter(p => p.package && !installedMap[p.package])
                : pkgAvailableData;
            rows = source.map(p => {
                const inst = installedMap[p.package];
                const up = upgradableMap[p.package];
                return {
                    name: p.package,
                    version: inst ? inst.version : p.version,
                    installed_date: inst ? (inst.installed_date || '') : '',
                    status: up ? 'upgrade' : (inst ? 'installed' : 'available')
                };
            });
        }
    }
    return rows;
}

function statusBadge(status) {
    if (status === 'upgrade') return '<span style="display:inline-block; padding:2px 10px; border-radius:20px; font-size:0.8rem; background:rgba(230,126,34,0.15); color:#e67e22; white-space:nowrap;">есть обновление</span>';
    if (status === 'installed') return '<span style="display:inline-block; padding:2px 10px; border-radius:20px; font-size:0.8rem; background:rgba(39,174,96,0.15); color:#27ae60; white-space:nowrap;">установлен</span>';
    return '<span style="display:inline-block; padding:2px 10px; border-radius:20px; font-size:0.8rem; background:rgba(120,120,120,0.15); color:var(--text-secondary); white-space:nowrap;">доступен</span>';
}

function actionCell(row) {
    if (row.status === 'upgrade') {
        return `<form method="post" style="display:inline;" onsubmit="opkgAction(event, 'upgrade', this.package.value); return false;">
            <input type="hidden" name="package" value="${escapeHtml(row.name)}">
            <input type="submit" value="Обновить" class="packages-delete-btn" style="background:#27ae60;">
        </form>`;
    }
    if (row.status === 'installed') {
        return `<form method="post" style="display:inline;" onsubmit="opkgAction(event, 'remove', this.package.value); return false;">
            <input type="hidden" name="package" value="${escapeHtml(row.name)}">
            <input type="submit" value="Удалить" class="packages-delete-btn">
        </form>`;
    }
    return `<form method="post" style="display:inline;" onsubmit="opkgAction(event, 'install', this.package.value); return false;">
        <input type="hidden" name="package" value="${escapeHtml(row.name)}">
        <input type="submit" value="Установить" class="packages-delete-btn">
    </form>`;
}

let pkgSortCol = -1;
let pkgSortAsc = true;

function enablePkgTableSorting(table) {
    const dataTypes = ['string', 'version', 'date', 'status'];
    initTableSorting(table, {
        excludeCol: [4], // «Действие» — не сортируем
        onSort: function(idx) {
            if (pkgSortCol === idx) {
                pkgSortAsc = !pkgSortAsc;
            } else {
                pkgSortCol = idx;
                pkgSortAsc = true;
            }
            sortTableRows(table, idx, dataTypes[idx], pkgSortAsc ? 'asc' : 'desc');
        }
    });
    if (pkgSortCol >= 0) sortTableRows(table, pkgSortCol, dataTypes[pkgSortCol], pkgSortAsc ? 'asc' : 'desc');
}

function renderPkgTable(rows) {
    const container = document.getElementById('pkg-table-container');
    if (!container) return;
    const count = document.getElementById('pkg-upd-count');
    if (count) {
        if (pkgUpgradableData.length > 0) {
            count.style.display = 'inline';
            count.textContent = '(' + pkgUpgradableData.length + ')';
        } else {
            count.style.display = 'none';
        }
    }
    if (!rows.length) {
        container.innerHTML = '<p style="color: var(--text-secondary);">Ничего не найдено.</p>';
        return;
    }
    let html = '<table class="packages-table" id="pkgTable"><thead><th>Пакет</th><th>Версия</th><th>Установлен</th><th>Статус</th><th>Действие</th></thead><tbody>';
    rows.forEach(row => {
        let ver = row.version ? escapeHtml(row.version) : (row.current ? escapeHtml(row.current) + ' → ' + escapeHtml(row.new) : '?');
        if (row.status === 'upgrade' && row.current && row.new) {
            ver = '<span style="color:var(--text-muted);">' + escapeHtml(row.current) + '</span> → <b style="color:#27ae60;">' + escapeHtml(row.new) + '</b>';
        }
        const instDate = row.installed_date ? escapeHtml(row.installed_date) : '<span style="color:var(--text-muted);">—</span>';
        html += `<tr>
            <td>${escapeHtml(row.name)}</td>
            <td>${ver}</td>
            <td style="white-space:nowrap;">${instDate}</td>
            <td>${statusBadge(row.status)}</td>
            <td>${actionCell(row)}</td>
        </tr>`;
    });
    html += '</tbody></table>';
    container.innerHTML = html;
    initTableSearch('searchPkg', 'pkgTable', -1);
    const table = document.getElementById('pkgTable');
    const rowsEl = table.getElementsByTagName('tr');
    for (let i = 1; i < rowsEl.length; i++) {
        rowsEl[i].style.cursor = 'pointer';
        rowsEl[i].addEventListener('click', function(e) {
            if (e.target.tagName === 'INPUT' || e.target.tagName === 'FORM') return;
            const pkgName = this.cells[0].textContent.trim();
            showPackageInfo(pkgName);
        });
    }
    enablePkgTableSorting(table);
}

async function loadPkgTable() {
    const container = document.getElementById('pkg-table-container');
    if (!container) return;
    container.innerHTML = '<div class="loading-spinner"></div>';
    const seq = ++pkgLoadSeq;
    try {
        await loadPkgData(false, seq);
        if (seq !== pkgLoadSeq) return;
        renderPkgTable(buildPkgRows());
    } catch (err) {
        if (seq !== pkgLoadSeq) return;
        container.innerHTML = `<p class="error">Ошибка загрузки: ${escapeHtml(err.message)}</p>`;
    }
}

function pkgCacheClearAll() {
    pkgCacheClear(CACHE_KEY, CACHE_TIME_KEY);
    pkgCacheClear(CACHE_INSTALLED_KEY, CACHE_INSTALLED_TIME);
    pkgCacheClear(CACHE_UPGRADABLE_KEY, CACHE_UPGRADABLE_TIME);
}

async function runPkgUpdate() {
    const updateBtn = document.getElementById('runUpdateBtn');
    const resultDiv = document.getElementById('update-result');
    updateBtn.disabled = true;
    updateBtn.innerHTML = '<svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=6#icon-refresh"/></svg> Обновление...';
    resultDiv.innerHTML = '<div class="loading-spinner"></div>';

    try {
        const response = await apiFetch('/update.cgi', {
            method: 'POST',
            headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
            body: 'run=1'
        });
        // Сервер сам экранирует opkg-вывод (htmlEscape на сервере) и отдаёт готовый HTML
        const text = await response.text();
        resultDiv.innerHTML = text;
        pkgCacheClearAll();
        await loadPkgTable();
    } catch (err) {
        resultDiv.innerHTML = `<p class="error">Ошибка: ${escapeHtml(err.message)}</p>`;
    } finally {
        updateBtn.disabled = false;
        updateBtn.innerHTML = '<svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=6#icon-refresh"/></svg> Обновить списки пакетов';
    }
}

async function upgradeAll() {
    const upgradeAllBtn = document.getElementById('upgradeAllBtn');
    const resultDiv = document.getElementById('update-result');
    
    if (!confirm('Обновить все пакеты? Это может занять продолжительное время.')) {
        return;
    }
    
    upgradeAllBtn.disabled = true;
    upgradeAllBtn.innerHTML = '<svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=6#icon-refresh"/></svg> Обновление...';
    resultDiv.innerHTML = '<div class="loading-spinner"></div>';
    
    try {
        const response = await apiFetch('/upgrade.cgi', {
            method: 'POST',
            headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
            body: 'upgrade_all=1'
        });
        const text = await response.text();
        resultDiv.innerHTML = text;
        pkgCacheClearAll();
        await loadPkgTable();
    } catch (err) {
        resultDiv.innerHTML = `<p class="error">Ошибка: ${escapeHtml(err.message)}</p>`;
    } finally {
        upgradeAllBtn.disabled = false;
        upgradeAllBtn.innerHTML = '<svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=6#icon-update"/></svg> Обновить все пакеты';
    }
}

function renderProcessesTab() {
    const html = `
        <h2 style="display: flex; align-items: center; gap: 8px;">
            <span class="stat-icon" style="width: 28px; height: 28px;">
                <svg class="icon" width="28" height="28"><use href="/entware-manager/icons.svg?v=6#icon-process"/></svg>
            </span>
            <span id="htop-title">Процессы (htop)</span>
        </h2>
        <div id="htop-content"><div class="loading-spinner"></div></div>
    `;
    contentDiv.innerHTML = html;
    loadHtopContent();
}

async function loadHtopContent() {
    try {
        const data = await apiGet('/ttyd_control.cgi');
        const htop = data.htop;
        const container = document.getElementById('htop-content');
        if (htop.state === 'running') {
            container.innerHTML = `
                <div style="display: flex; gap: 10px; margin-bottom: 15px;">
                    <a href="/htop/" target="_blank" rel="noopener" class="packages-delete-btn" style="background:#4a5568;"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=6#icon-link"/></svg> Открыть в новой вкладке</a>
                </div>
                <iframe id="htopFrame" src="/htop/" width="100%" height="600" style="border: none; border-radius: 8px;" allow="fullscreen; autoplay"></iframe>
            `;
            focusTtydFrame('htopFrame');
        } else {
            container.innerHTML = '<p style="color: var(--text-secondary); font-size: 0.9rem;">htop не запущен. Откройте <b>Настройки → Терминал</b>, задайте пароль и нажмите <b>Запустить</b>.</p>';
        }
    } catch (err) {
        const container = document.getElementById('htop-content');
        if (container) container.innerHTML = '<p class="error">Ошибка: ' + escapeHtml(err.message) + '</p>';
    }
}

function renderTerminalTab() {
    const html = `
        <h2 style="display: flex; align-items: center; gap: 8px;">
            <span class="stat-icon" style="width: 28px; height: 28px;">
                <svg class="icon" width="28" height="28"><use href="/entware-manager/icons.svg?v=6#icon-terminal"/></svg>
            </span>
            <span id="terminal-title">Терминал</span>
        </h2>
        <div id="terminal-content"><div class="loading-spinner"></div></div>
    `;
    contentDiv.innerHTML = html;
    loadTerminalContent();
}

async function loadTerminalContent() {
    try {
        const data = await apiGet('/ttyd_control.cgi');
        const term = data.terminal;
        const container = document.getElementById('terminal-content');
        const title = document.getElementById('terminal-title');
        if (term.state === 'running') {
            const modeLabel = term.mode === 'telnet' ? 'Telnet' : 'Entware';
            title.textContent = 'Терминал (' + modeLabel + ')';
            container.innerHTML = `
                <div style="display: flex; gap: 10px; margin-bottom: 15px;">
                    <a href="/terminal/" target="_blank" rel="noopener" class="packages-delete-btn" style="background:#4a5568;"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=6#icon-link"/></svg> Открыть в новой вкладке</a>
                </div>
                <iframe id="terminalFrame" src="/terminal/" width="100%" height="600" style="border: none; border-radius: 8px;" allow="fullscreen; autoplay; clipboard-read; clipboard-write"></iframe>
            `;
            focusTtydFrame('terminalFrame');
        } else {
            title.textContent = 'Терминал';
            container.innerHTML = '<p style="color: var(--text-secondary); font-size: 0.9rem;">Терминал не запущен. Откройте <b>Настройки → Терминал</b>, задайте пароль и нажмите <b>Запустить</b>.</p>';
        }
    } catch (err) {
        const container = document.getElementById('terminal-content');
        if (container) container.innerHTML = '<p class="error">Ошибка: ' + escapeHtml(err.message) + '</p>';
    }
}

function focusTtydFrame(id) {
    const frame = document.getElementById(id);
    if (!frame) return;
    const focusInner = () => {
        try {
            const w = frame.contentWindow;
            if (!w) return;
            // ttyd выставляет window.term (xterm.js) — фокусируем textarea,
            // иначе Ctrl+V уходит в PTY как литеральный ^V.
            if (w.term && typeof w.term.focus === 'function') {
                w.term.focus();
            } else {
                w.focus();
            }
        } catch (e) { /* cross-origin или ещё не готово — игнорируем */ }
    };
    frame.addEventListener('load', () => {
        setTimeout(focusInner, 400);
        setTimeout(focusInner, 1200);
    });
}

function loadLogsTab() {
    const html = `
        <h2 style="display: flex; align-items: center; gap: 8px;">
            <span class="stat-icon" style="width: 28px; height: 28px;">
                <svg class="icon" width="28" height="28"><use href="/entware-manager/icons.svg?v=6#icon-list"/></svg>
            </span>
            Логи
        </h2>
        <div style="margin-bottom: 20px;">
            <div style="display: flex; gap: 8px; border-bottom: 1px solid var(--border-color); padding-bottom: 8px;">
                <button id="tab-manager" class="tab-button active">Действия менеджера</button>
                <button id="tab-system" class="tab-button">Системные логи</button>
            </div>
            <div id="manager-controls" style="margin-top: 16px; display: flex; gap: 12px; align-items: center;">
                <button id="clearOldLogsBtn" class="packages-delete-btn" style="background:#e53e3e;">
                    <svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=6#icon-trash"/></svg> Очистить логи старше 30 дней
                </button>
                <button id="rotateNowBtn" class="packages-delete-btn" style="background:#f59e0b;">
                    <svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=6#icon-refresh"/></svg> Ротация сейчас
                </button>
                <button id="toggleLoggingBtn" class="packages-delete-btn" style="background:#4a5568; display: flex; align-items: center; gap: 8px;">
                    <span id="loggingStatusIndicator" style="display: inline-block; width: 12px; height: 12px; border-radius: 50%; background: gray;"></span>
                    Настройки логирования
                </button>
                <button id="systemEventsBtn" class="packages-delete-btn" style="background:#2c5282;">
                    <svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=6#icon-info"/></svg> Системные события
                </button>
            </div>
            <div id="system-controls" style="margin-top: 16px; display: none; gap: 12px; flex-wrap: wrap; align-items: center;">
                <select id="system-source" class="packages-delete-btn" style="background: var(--input-bg); color: var(--text-primary);">
                    <option value="">Выберите источник</option>
                </select>
                <button id="refreshSystemLogs" class="packages-delete-btn" style="background:#4a5568;">
                    <svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=6#icon-refresh"/></svg> Обновить
                </button>
                <button id="searchByNameBtn" class="packages-delete-btn" style="background:#4a5568;">
                    <svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=6#icon-search"/></svg> Поиск по имени
                </button>
                <button id="clearDynamicSourcesBtn" class="packages-delete-btn" style="background:#e53e3e;">
                    <svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=6#icon-trash"/></svg> Очистить источники
                </button>
            </div>
        </div>
        <iframe id="logsFrame" src="/entware-cgi/logger/view.cgi" width="100%" height="600" style="border: none; border-radius: 8px;"></iframe>
    `;
    contentDiv.innerHTML = html;

    let dynamicSources = [];
    let storedSources = localStorage.getItem('entware_dynamic_sources');
    if (storedSources) {
        try { dynamicSources = JSON.parse(storedSources); } catch(e) {}
    }

    async function updateSystemSourcesList() {
        const select = document.getElementById('system-source');
        if (!select) return;
        let staticSources = [];
        try {
            const resp = await fetch('/entware-manager/logger/system_sources.json?_=' + Date.now());
            const data = await resp.json();
            staticSources = data.sources || [];
        } catch(e) {}
        const allSources = [...staticSources, ...dynamicSources];
        select.innerHTML = '<option value="">Выберите источник</option>';
        for (const src of allSources) {
            const option = document.createElement('option');
            option.value = src.name;
            option.textContent = src.name;
            if (src.path) option.dataset.file = src.path;
            else if (src.file) option.dataset.file = src.file;
            select.appendChild(option);
        }
    }

    const searchBtn = document.getElementById('searchByNameBtn');
    if (searchBtn) {
        searchBtn.addEventListener('click', async () => {
            const query = prompt('Введите текст для поиска в имени файла (в /tmp):');
            if (!query || !query.trim()) return;
            try {
                const files = await apiGet('/logger/find_by_name.cgi?q=' + encodeURIComponent(query));
                if (files.length === 0) { Toast.show('Файлы не найдены.', true); return; }
                let added = 0;
                for (const f of files) {
                    if (!dynamicSources.some(src => src.path === f.path)) {
                        dynamicSources.push({ name: f.name, path: f.path });
                        added++;
                    }
                }
                if (added === 0) { Toast.show('Все файлы уже в списке.'); return; }
                localStorage.setItem('entware_dynamic_sources', JSON.stringify(dynamicSources));
                await updateSystemSourcesList();
                Toast.show(`Добавлено ${added} файлов.`);
            } catch(err) { Toast.show('Ошибка поиска.', true); }
        });
    }

    const clearDynamicBtn = document.getElementById('clearDynamicSourcesBtn');
    if (clearDynamicBtn) {
        clearDynamicBtn.addEventListener('click', () => {
            if (!confirm('Удалить все сохранённые динамические источники логов?')) return;
            localStorage.removeItem('entware_dynamic_sources');
            dynamicSources = [];
            updateSystemSourcesList();
            Toast.show('Сохранённые источники очищены');
        });
    }

    const refreshSystemLogs = document.getElementById('refreshSystemLogs');
    if (refreshSystemLogs) {
        refreshSystemLogs.addEventListener('click', () => {
            const select = document.getElementById('system-source');
            const selectedOption = select.options[select.selectedIndex];
            const filePath = selectedOption?.dataset.file;
            const sourceName = select.value;
            if (sourceName && filePath) {
                const iframe = document.getElementById('logsFrame');
                iframe.src = API_BASE + '/logger/system_logs.cgi?file=' + encodeURIComponent(filePath) + '&_=' + Date.now();
            }
        });
    }

    const sourceSelect = document.getElementById('system-source');
    if (sourceSelect) {
        sourceSelect.addEventListener('change', () => {
            const selectedOption = sourceSelect.options[sourceSelect.selectedIndex];
            const filePath = selectedOption?.dataset.file;
            const sourceName = sourceSelect.value;
            const iframe = document.getElementById('logsFrame');
            if (sourceName && filePath) {
                iframe.src = API_BASE + '/logger/system_logs.cgi?file=' + encodeURIComponent(filePath) + '&_=' + Date.now();
            } else {
                iframe.src = 'about:blank';
            }
        });
    }

    const tabManager = document.getElementById('tab-manager');
    const tabSystem = document.getElementById('tab-system');
    const managerControls = document.getElementById('manager-controls');
    const systemControls = document.getElementById('system-controls');
    const logsFrame = document.getElementById('logsFrame');

    tabManager.addEventListener('click', () => {
        tabManager.classList.add('active');
        tabSystem.classList.remove('active');
        managerControls.style.display = 'flex';
        systemControls.style.display = 'none';
        logsFrame.src = API_BASE + '/logger/view.cgi?_=' + Date.now();
    });

    tabSystem.addEventListener('click', async () => {
        tabSystem.classList.add('active');
        tabManager.classList.remove('active');
        managerControls.style.display = 'none';
        systemControls.style.display = 'flex';
        await updateSystemSourcesList();
        logsFrame.src = 'about:blank';
    });

    document.getElementById('clearOldLogsBtn').addEventListener('click', async () => {
        if (!confirm('Удалить все логи старше 30 дней?')) return;
        const data = await apiPost('/logger/clear.cgi', '');
        Toast.show(data.message);
        logsFrame.src = API_BASE + '/logger/view.cgi?_=' + Date.now();
    });

    document.getElementById('rotateNowBtn').addEventListener('click', async () => {
        if (!confirm('Запустить ротацию логов сейчас?')) return;
        const data = await apiPost('/logger/rotate.cgi', '');
        if (data.rotated && data.rotated.length > 0) {
            const lines = data.rotated.map(f => f.path + ' (' + fmtBytesJS(f.size) + ')');
            Toast.show(data.message + '\n' + lines.join('\n'), false, 6000);
        } else {
            Toast.show(data.message);
        }
        setTimeout(() => logsFrame.src = API_BASE + '/logger/view.cgi?_=' + Date.now(), 1000);
    });

    document.getElementById('toggleLoggingBtn').addEventListener('click', async () => {
        const cfg = await apiGet('/logger/config.cgi');
        const enabled = cfg.enabled;
        const newState = confirm(`Логирование сейчас ${enabled ? 'включено' : 'отключено'}. Хотите изменить?`);
        if (newState) {
            const newCfg = { ...cfg, enabled: !enabled };
            const result = await apiPostJSON('/logger/config.cgi', newCfg);
            if (result.status === 'ok') { Toast.show('Настройки сохранены. Страница будет перезагружена.'); location.reload(); }
            else Toast.show('Ошибка сохранения', true);
        }
    });

    async function updateLoggingStatus() {
        try {
            const cfg = await apiGet('/logger/config.cgi');
            const indicator = document.getElementById('loggingStatusIndicator');
            if (indicator) {
                indicator.style.backgroundColor = cfg.enabled ? '#2ecc71' : '#e74c3c';
                indicator.title = cfg.enabled ? 'Логирование включено' : 'Логирование отключено';
            }
        } catch(e) {}
    }
    updateLoggingStatus();
    setInterval(updateLoggingStatus, 10000);

    document.getElementById('systemEventsBtn').addEventListener('click', async () => {
        const modal = document.createElement('div');
        modal.id = 'systemEventsModal';
        modal.style.cssText = 'position:fixed;top:0;left:0;right:0;bottom:0;background:rgba(0,0,0,0.8);z-index:9999;display:flex;align-items:center;justify-content:center;';
        modal.innerHTML = `
            <div style="background:var(--modal-bg);border-radius:12px;padding:0;max-width:900px;width:95%;max-height:90vh;display:flex;flex-direction:column;overflow:hidden;">
                <div style="display:flex;justify-content:space-between;align-items:center;padding:16px 20px;border-bottom:1px solid var(--border-color,#333);">
                    <span style="font-size:16px;font-weight:500;color:var(--text-primary);">Системные события</span>
                    <button id="closeSystemModal" style="background:none;border:none;color:var(--text-primary);font-size:24px;cursor:pointer;">&times;</button>
                </div>
                <div id="systemLogContent" style="flex:1;overflow:auto;padding:16px;color:var(--text-primary);"></div>
            </div>
        `;
        document.body.appendChild(modal);
        document.getElementById('closeSystemModal').addEventListener('click', () => modal.remove());
        modal.addEventListener('click', (e) => { if (e.target === modal) modal.remove(); });
        
        // Fetch and display system log
        try {
            const resp = await apiFetch('/logger/system_log.cgi');
            const html = await resp.text();
            const parser = new DOMParser();
            const doc = parser.parseFromString(html, 'text/html');
            const body = doc.body.innerHTML;
            document.getElementById('systemLogContent').innerHTML = body;
        } catch(e) {
            document.getElementById('systemLogContent').innerHTML = '<p style="color:#e74c3c;">Ошибка загрузки</p>';
        }
    });
}

async function fetchTtydStatus() {
    try {
        const data = await apiGet('/ttyd_control.cgi');
        updateTtydStatus(data);
    } catch (err) {
        document.getElementById('ttyd-status').innerHTML = `<p class="error">Ошибка: ${escapeHtml(err.message)}</p>`;
    }
}

function updateTtydStatus(data) {
    const statusDiv = document.getElementById('ttyd-status');
    const htop = data.htop;
    const term = data.terminal;
    const modeLabel = term.mode === 'telnet' ? 'Telnet' : 'Entware';

    let html = '<h3>Текущее состояние ttyd</h3>';
    html += '<table class="stat-table">';
    html += `  <tr><td>htop (8089, доступ /htop/):</td><td><span class="${htop.state === 'running' ? 'stat-value-normal' : 'stat-value-critical'}">${escapeHtml(htop.state)}</span> ${htop.pid ? '(PID ' + escapeHtml(htop.pid) + ')' : ''}</td></tr>`;
    html += `  <tr><td>Терминал (9089, доступ /terminal/):</td><td><span class="${term.state === 'running' ? 'stat-value-normal' : 'stat-value-critical'}">${escapeHtml(term.state)}</span> ${term.pid ? '(PID ' + escapeHtml(term.pid) + ')' : ''} ${term.state === 'running' ? '(' + modeLabel + ')' : ''}</td></tr>`;
    html += '</table>';
    statusDiv.innerHTML = html;

    const s = function(id) { return document.getElementById(id); };
    s('htop-start').disabled = (htop.state === 'running');
    s('htop-stop').disabled = (htop.state !== 'running');
    s('htop-restart').disabled = (htop.state !== 'running');
    s('term-start').disabled = (term.state === 'running');
    s('term-stop').disabled = (term.state !== 'running');
    s('term-restart').disabled = (term.state !== 'running');

    const modeSelect = s('termMode');
    if (modeSelect && term.state === 'running') {
        modeSelect.value = term.mode;
    }
}

window.controlTtyd = async function(action, port, pass, mode) {
    const formData = new URLSearchParams();
    formData.append('action', action);
    formData.append('port', port);
    if (pass) formData.append('pass', pass);
    if (mode) formData.append('mode', mode);
    try {
        const data = await apiPost('/ttyd_control.cgi', formData.toString());
        Toast.show(data.message);
        // Очищаем поле пароля после запуска/остановки, чтобы не оставался в DOM.
        if (action === 'start' || action === 'stop' || action === 'restart') {
            const passId = port === 8089 ? 'htopPass' : 'termPass';
            const passEl = document.getElementById(passId);
            if (passEl) passEl.value = '';
        }
        fetchTtydStatus();
    } catch (err) {
        Toast.show('Ошибка: ' + err.message, true);
    }
};

function getDefaultLinks() {
    const h = BASE_URL;
    return [
        { name: 'Роутер', url: h, icon: 'router' },
        { name: 'Entware Manager', url: '/entware-manager/', icon: 'package' },
        { name: 'AdGuard Home', url: h + ':3000', icon: 'shield' },
        { name: 'Transmission', url: h + ':9091', icon: 'download' },
        { name: 'Netdata', url: h + ':19999', icon: 'chart' },
        { name: 'htop (ttyd)', url: '/htop/', icon: 'process' },
        { name: 'Терминал (ttyd)', url: '/terminal/', icon: 'terminal' }
    ];
}

async function loadLinks() {
    try {
        const links = await apiGet('/links_load.cgi');
        return links.map(link => {
            if (link.icon && link.icon.startsWith('icon-')) link.icon = link.icon.replace('icon-', '');
            else if (link.icon && (link.icon.length === 1 || link.icon.match(/[\u{1F300}-\u{1F6FF}]/u))) {
                const id = mapIconToId(link.icon).replace('icon-', '');
                link.icon = id === 'default' ? 'link' : id;
            } else if (!link.icon) link.icon = 'link';
            return link;
        });
    } catch (err) {
        const stored = localStorage.getItem('entware_links');
        if (stored) {
            try {
                const parsed = JSON.parse(stored);
                return parsed.map(link => {
                    if (link.icon && link.icon.startsWith('icon-')) link.icon = link.icon.replace('icon-', '');
                    else if (link.icon && !link.icon.match(/^[a-z-]+$/)) link.icon = 'link';
                    return link;
                });
            } catch(e) {}
        }
        return getDefaultLinks();
    }
}

async function saveLinks(links) {
    try {
        const result = await apiPostJSON('/links_save.cgi', links);
        if (result.status !== 'ok') {
            Toast.show('Ошибка сохранения ссылок: ' + result.message, true);
            return false;
        }
        return true;
    } catch (err) {
        Toast.show('Ошибка соединения с сервером', true);
        return false;
    }
}

function renderIconSelect(selectedId) {
    let html = `<select class="link-icon-select settings-input" style="max-width: 150px; padding: 6px;">`;
    iconList.forEach(icon => {
        const selected = (icon.id === selectedId) ? 'selected' : '';
        html += `<option value="${icon.id}" ${selected}>${icon.name}</option>`;
    });
    html += `</select>`;
    return html;
}

async function loadNetworkStatus(fresh) {
    const table = document.getElementById('networkTable');
    if (!table) return;
    
    try {
        const data = await apiGet(fresh ? '/network_stats.cgi?fresh=1&_=' + Date.now() : '/network_stats.cgi');
        
        let html = '<div class="stat-table">';
        
        // Таблица 1: Интерфейсы с IP (карточки)
        html += '<div class="net-counters">';
        html += '<div class="net-counters-title">Интерфейсы</div>';
        if (data.interfaces && data.interfaces.length > 0) {
            data.interfaces.forEach(iface => {
                if (iface.ip && iface.ip !== '--') {
                    html += `<div class="net-counter-card">
                        <div class="net-counter-head">
                            <span class="net-counter-name">${escapeHtml(iface.iface)}</span>
                            <code class="net-counter-value">${escapeHtml(iface.ip)}</code>
                        </div>
                    </div>`;
                }
            });
        }
        html += '</div>';
        
        // Таблица 2: Физические порты (карточки)
        html += '<div class="net-counters">';
        html += '<div class="net-counters-title">Физические порты</div>';
        if (data.ports && data.ports.length > 0) {
            data.ports.forEach(port => {
                const carrier = port.carrier === '✓';
                const speed = port.speed && port.speed !== '—' ? port.speed : '';
                const label = carrier ? (speed ? `${speed}` : '✓') : '—';
                const cls = carrier ? 'net-counter-status ok' : 'net-counter-status bad';
                html += `<div class="net-counter-card">
                    <div class="net-counter-head">
                        <span class="net-counter-name">${escapeHtml(port.iface)}</span>
                        <span class="${cls}">${escapeHtml(label)}</span>
                    </div>
                </div>`;
            });
        }
        html += '</div>';
        
        // Таблица 3: Сети (карточки)
        html += '<div class="net-counters">';
        html += '<div class="net-counters-title">Сети</div>';
        if (data.networks && data.networks.length > 0) {
            data.networks.forEach(net => {
                const members = net.members ? net.members.trim().split(/\s+/).filter(Boolean) : [];
                let valueHtml;
                if (members.length > 0) {
                    valueHtml = '<div class="net-counter-chips">' + members.map(m =>
                        `<span class="net-counter-chip">${escapeHtml(m)}</span>`
                    ).join('') + '</div>';
                } else if (net.bridge && net.bridge !== '--' && net.bridge !== '—') {
                    const wanUp = data.wan && data.wan.indexOf(net.bridge) !== -1;
                    valueHtml = wanUp
                        ? `<span class="net-counter-status ok">${escapeHtml(data.wan)}</span>`
                        : `<span class="net-counter-chip">${escapeHtml(net.bridge)}</span>`;
                } else {
                    valueHtml = '<span class="net-counter-status bad">—</span>';
                }
                html += `<div class="net-counter-card">
                    <div class="net-counter-head">
                        <span class="net-counter-name">${escapeHtml(net.name)}</span>
                        ${valueHtml}
                    </div>
                </div>`;
            });
        }
        html += '</div>';
        
        html += '</div>';
        
        // Таблица 4: WiFi сети (на отдельной строке)
        if (data.wifi_info && data.wifi_info.length > 0 && data.wifi_info[0].name !== '--') {
            html += '<div class="stat-table wifi-counters" style="margin-top: 10px;">';
            html += '<div class="wifi-counters-title">WiFi сети</div>';
            data.wifi_info.forEach(wifi => {
                const ifaces = wifi.interfaces && wifi.interfaces.length > 0
                    ? wifi.interfaces
                    : [];
                const summary = `<div class="wifi-counter-tx">TX <b>${escapeHtml(wifi.tx)}</b></div>
                                 <div class="wifi-counter-rx">RX <b>${escapeHtml(wifi.rx)}</b></div>`;
                let ifaceHtml = '';
                if (ifaces.length > 0) {
                    ifaceHtml = '<div class="wifi-counter-ifaces">' + ifaces.map(ifc =>
                        `<div class="wifi-counter-iface"><span class="wifi-iface-name">${escapeHtml(ifc.iface)}</span><span class="wifi-iface-tx">TX ${escapeHtml(ifc.tx)}</span><span class="wifi-iface-rx">RX ${escapeHtml(ifc.rx)}</span></div>`
                    ).join('') + '</div>';
                }
                html += `<div class="wifi-counter-card">
                            <div class="wifi-counter-head">
                                <span class="wifi-counter-name">${escapeHtml(wifi.name)}</span>
                                ${summary}
                            </div>
                            ${ifaceHtml}
                        </div>`;
            });
            html += '</div>';
        }
        
        if (html) {
            table.innerHTML = html;
        } else {
            table.innerHTML = '<div style="padding: 0.5rem 1rem;">Нет данных</div>';
        }
    } catch (e) {
        table.innerHTML = '<div style="padding: 0.5rem 1rem;">Ошибка загрузки сети</div>';
    }
    
    document.getElementById('network-refresh')?.addEventListener('click', () => loadNetworkStatus(true));
}

// ===== Мост сервисов (вкладка «Модули» + карточки на Статистике) =====

const BRIDGE_STATE_LABELS = {
    running: ['работает', '#38a169'],
    auth_required: ['нужна авторизация', '#d69e2e'],
    absent: ['не установлен', '#718096'],
    disabled: ['выключен', '#718096']
};

function bridgeStateBadge(st) {
    const [label, color] = BRIDGE_STATE_LABELS[st] || [st || '?', '#718096'];
    return '<span style="color:' + color + ';font-weight:600;">' + escapeHtml(label) + '</span>';
}

async function bridgeDiscover() {
    const data = await apiGet('/bridge_discover.cgi');
    return data.services || [];
}

// Читаем настройки уведомлений из localStorage (Серверная часть — Этап 4).
function bridgeNotifKey(id) { return 'bridge_notif_' + id; }

function renderBridgeCard(svc, prefs) {
    const p = prefs && prefs[svc.id] || {};
    const enabled = p.enabled !== false && svc.state !== 'disabled';
    const notif = p.notifications !== undefined ? !!p.notifications
        : localStorage.getItem(bridgeNotifKey(svc.id)) !== 'off';
    const dimStyle = enabled ? '' : 'opacity:0.55;';
    const actionsHtml = enabled ? (svc.actions || []).map(a =>
        '<button class="packages-delete-btn" style="background:#4a5568;padding:4px 10px;font-size:0.8rem;" data-bridge-id="' +
        escapeHtml(svc.id) + '" data-action="' + escapeHtml(a.id) + '"' +
        (a.confirm ? ' data-confirm="1"' : '') + '>' + escapeHtml(a.label) + '</button>'
    ).join(' ') : '';
    return '<div class="stat-card" style="min-width:230px;' + dimStyle + '">' +
        '<h4 style="margin:0 0 6px 0;display:flex;align-items:center;justify-content:space-between;gap:8px;">' +
        '<span>' + escapeHtml(svc.name) + '</span>' +
        '<label class="ewm-toggle" title="Включить/выключить модуль">' +
        '<input type="checkbox" class="bridge-enabled" data-id="' + escapeHtml(svc.id) + '"' + (enabled ? ' checked' : '') + '>' +
        '<span class="ewm-slider"></span></label></h4>' +
        '<div style="margin-bottom:6px;">' + bridgeStateBadge(enabled ? svc.state : 'disabled') + '</div>' +
        (enabled && svc.detail ? '<div style="font-size:0.75rem;color:var(--text-muted);margin-bottom:6px;">' + escapeHtml(svc.detail) + '</div>' : '') +
        '<div style="display:flex;gap:8px;align-items:center;flex-wrap:wrap;">' +
        '<label style="display:flex;align-items:center;gap:4px;font-size:0.8rem;color:var(--text-muted);">' +
        '<input type="checkbox" class="bridge-notif" data-id="' + escapeHtml(svc.id) + '"' + (notif ? ' checked' : '') + '> уведомления</label>' +
        actionsHtml + '</div></div>';
}

function bindBridgeCards(container) {
    container.querySelectorAll('.bridge-enabled').forEach(cb => {
        cb.addEventListener('change', async () => {
            cb.disabled = true;
            try {
                await saveBridgePref(cb.dataset.id, 'enabled', cb.checked);
                Toast.show('Модуль «' + cb.dataset.id + '»: ' + (cb.checked ? 'включён' : 'выключен'));
                renderBridgeTab(); // перерисовать с учётом состояния
            } catch(e) {
                Toast.show('Ошибка: ' + e.message);
                cb.disabled = false;
            }
        });
    });
    container.querySelectorAll('.bridge-notif').forEach(cb => {
        cb.addEventListener('change', async () => {
            try {
                await saveBridgePref(cb.dataset.id, 'notifications', cb.checked);
                Toast.show('Уведомления для «' + cb.dataset.id + '»: ' + (cb.checked ? 'вкл' : 'выкл'));
            } catch(e) {
                Toast.show('Ошибка: ' + e.message);
                cb.checked = !cb.checked;
            }
        });
    });
    container.querySelectorAll('[data-action]').forEach(btn => {
        btn.addEventListener('click', async () => {
            const id = btn.dataset.bridgeId, action = btn.dataset.action;
            let password;
            if (btn.dataset.confirm) {
                password = prompt('Повторите пароль панели для подтверждения:');
            } else {
                password = prompt('Пароль панели:');
            }
            if (!password) return;
            btn.disabled = true;
            try {
                const res = await apiPost('/bridge_action.cgi',
                    'id=' + encodeURIComponent(id) + '&action=' + encodeURIComponent(action) +
                    '&password=' + encodeURIComponent(password));
                if (res.status === 'ok') Toast.show('Выполнено (' + (res.result && res.result.raw || 'ok') + ')');
                else Toast.show(res.message || res.status);
            } catch(e) { Toast.show('Ошибка: ' + e.message); }
            btn.disabled = false;
        });
    });
}

let bridgePrefsCache = null;

async function loadBridgePrefs() {
    try {
        const data = await apiGet('/bridge_prefs.cgi');
        bridgePrefsCache = data.modules || {};
    } catch(e) { bridgePrefsCache = {}; }
    return bridgePrefsCache;
}

async function saveBridgePref(id, field, value) {
    // prefs хранятся парами enabled/notifications — читаем текущие и обновляем поле
    const p = (bridgePrefsCache && bridgePrefsCache[id]) || {};
    const body = 'id=' + encodeURIComponent(id) +
        '&enabled=' + (field === 'enabled' ? String(value) : String(p.enabled !== false)) +
        '&notifications=' + (field === 'notifications' ? String(value) : String(!!p.notifications));
    const res = await apiPost('/bridge_prefs.cgi', body);
    if (res.status !== 'ok') throw new Error(res.message || res.status);
    bridgePrefsCache[id] = { enabled: res.enabled, notifications: res.notifications };
}

async function renderBridgeTab() {
    const contentDiv = document.getElementById('content');
    contentDiv.innerHTML = '<p>Загрузка...</p>';
    let services = [];
    try {
        services = await bridgeDiscover();
        await loadBridgePrefs();
    } catch (e) {
        contentDiv.innerHTML = '<p class="error">Мост недоступен: ' + escapeHtml(e.message) + '</p>';
        return;
    }

    let html = '<h2><svg class="icon" width="24" height="24"><use href="/entware-manager/icons.svg?v=6#icon-modules"/></svg> Модули</h2>';
    html += '<p style="color:var(--text-muted);">Локальные сервисы Entware, обнаруженные на этом роутере. Ползунок включает/выключает модуль в панели, галочка управляет уведомлениями о падении/восстановлении сервиса.</p>';
    if (!services.length) {
        html += '<p>Ничего не найдено.</p>';
    } else {
        const prefs = bridgePrefsCache;
        html += '<div class="stats-grid" id="bridge-grid">' + services.map(s => renderBridgeCard(s, prefs)).join('') + '</div>';
    }
    html += '<button class="packages-delete-btn" style="background:#4a5568;margin-top:16px;" onclick="renderBridgeTab()">' +
        '<svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=6#icon-refresh"/></svg> Пересканировать</button>';
    contentDiv.innerHTML = html;

    bindBridgeCards(contentDiv);

    // Спец-карточка AdGuard Home: статистика + управление защитой
    const agh = services.find(s => s.id === 'adguard');
    if (agh) {
        const zone = document.createElement('div');
        zone.innerHTML = '<h3 style="margin-top:24px;"><svg class="icon" width="20" height="20"><use href="/entware-manager/icons.svg?v=6#icon-shield"/></svg> AdGuard Home</h3><div id="agh-zone"><p>Загрузка статистики…</p></div>';
        contentDiv.appendChild(zone);
        loadAdGuardZone();
    }
}

async function loadAdGuardZone() {
    const zone = document.getElementById('agh-zone');
    if (!zone) return;
    try {
        const res = await apiGet('/bridge_stats.cgi?id=adguard&block=status');
        if (res.status !== 'ok') { zone.innerHTML = '<p class="error">' + escapeHtml(res.message || 'нет данных') + '</p>'; return; }
        const r = res.result || {};
        if (r.http_code === 401 || r.http_code === 403) {
            zone.innerHTML = aghAuthFormHTML();
            bindAghAuthForm();
            return;
        }
        if (r.error) { zone.innerHTML = '<p class="error">' + escapeHtml(r.error) + '</p>'; return; }

        const st = r.body || {};
        let html = '<div class="stats-grid">';
        html += '<div class="stat-card"><h4>Защита</h4><div id="agh-prot" style="font-weight:700;font-size:1.1em;">' +
            (st.protection_enabled ? '<span style="color:#38a169;">ВКЛЮЧЕНА</span>' : '<span style="color:#e53e3e;">ОТКЛЮЧЕНА</span>') + '</div>' +
            '<button class="packages-delete-btn" style="margin-top:8px;background:#4a5568;" data-agh="toggle">' +
            (st.protection_enabled ? 'Выключить' : 'Включить') + '</button></div>';
        html += '<div class="stat-card"><h4>Версия</h4><div style="font-weight:600;">' + escapeHtml(st.version || '—') + '</div></div>';
        html += '</div><div id="agh-stats-extra" style="margin-top:12px;"></div>' +
            '<button class="packages-delete-btn" style="background:#4a5568;margin-top:8px;" onclick="loadAdGuardZone()">' +
            '<svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=6#icon-refresh"/></svg> Обновить</button>';
        zone.innerHTML = html;
        bindAghToggle();

        apiGet('/bridge_stats.cgi?id=adguard&block=stats').then(extra => {
            const box = document.getElementById('agh-stats-extra');
            if (!box || extra.status !== 'ok' || !extra.result || !extra.result.body) return;
            const b = extra.result.body;
            const fmtN = n => Number(n).toLocaleString('ru-RU');
            let h = '<div class="stats-grid">' +
                '<div class="stat-card"><div style="color:var(--text-muted);">Запросов за сутки</div><div style="font-size:1.3em;font-weight:700;">' + fmtN(b.num_dns_queries || 0) + '</div></div>' +
                '<div class="stat-card"><div style="color:var(--text-muted);">Заблокировано</div><div style="font-size:1.3em;font-weight:700;color:#d69e2e;">' + fmtN(b.num_blocked_filtering || 0) + '</div></div>' +
                '</div>';
            if ((b.top_blocked_domains || []).length) {
                h += '<div style="margin-top:8px;font-size:0.85rem;color:var(--text-muted);">Топ блокировок: ' +
                    b.top_blocked_domains.slice(0, 5).map(d => {
                        const name = (typeof d === 'string') ? d : (d.domain || '?');
                        return escapeHtml(name);
                    }).join(', ') + '</div>';
            }
            box.innerHTML = h;
        }).catch(() => {});
    } catch(e) {
        zone.innerHTML = '<p class="error">' + escapeHtml(e.message) + '</p>';
    }
}

function aghAuthFormHTML() {
    return '<div class="stat-card" style="max-width:420px;">' +
        '<p style="margin-top:0;">AdGuard Home требует авторизацию. Введите логин и пароль от его веб-интерфейса — они сохранятся на роутере (файл 0600, не покидают устройство).</p>' +
        '<input type="text" id="agh-user" class="settings-input" placeholder="Логин AGH" style="width:100%;margin-bottom:6px;">' +
        '<input type="password" id="agh-pass" class="settings-input" placeholder="Пароль AGH" style="width:100%;margin-bottom:6px;">' +
        '<input type="password" id="agh-panel-pass" class="settings-input" placeholder="Пароль панели EM" style="width:100%;margin-bottom:8px;">' +
        '<button class="packages-delete-btn" style="background:#2ecc71;" onclick="saveAghAuth()">Сохранить</button>' +
        '<span id="agh-auth-status"></span></div>';
}

function bindAghAuthForm() {
    const inp = document.getElementById('agh-pass');
    if (inp) inp.addEventListener('keydown', e => { if (e.key === 'Enter') saveAghAuth(); });
}

async function saveAghAuth() {
    const user = document.getElementById('agh-user').value.trim();
    const appPass = document.getElementById('agh-pass').value;
    const panelPass = document.getElementById('agh-panel-pass').value;
    const st = document.getElementById('agh-auth-status');
    if (!user || !appPass || !panelPass) { st.innerHTML = '<span style="color:#e53e3e;">Заполните все поля</span>'; return; }
    st.innerHTML = 'Сохранение…';
    try {
        const res = await apiPost('/bridge_auth.cgi',
            'id=adguard&cred_type=basic&username=' + encodeURIComponent(user) +
            '&app_password=' + encodeURIComponent(appPass) +
            '&password=' + encodeURIComponent(panelPass));
        st.innerHTML = res.status === 'ok'
            ? '<span style="color:#38a169;">Сохранено</span>'
            : '<span style="color:#e53e3e;">' + escapeHtml(res.message || res.status) + '</span>';
        if (res.status === 'ok') setTimeout(loadAdGuardZone, 800);
    } catch(e) { st.innerHTML = '<span style="color:#e53e3e;">' + escapeHtml(e.message) + '</span>'; }
}

function bindAghToggle() {
    document.querySelector('[data-agh="toggle"]')?.addEventListener('click', async function() {
        this.disabled = true;
        const password = prompt('Повторите пароль панели для подтверждения:');
        if (!password) { this.disabled = false; return; }
        try {
            const res = await apiPost('/bridge_action.cgi',
                'id=adguard&action=protection_toggle&password=' + encodeURIComponent(password));
            Toast.show(res.status === 'ok' ? 'Применено (' + ((res.result||{}).raw||'ok') + ')' : (res.message || res.status));
            setTimeout(loadAdGuardZone, 1200);
        } catch(e) { Toast.show('Ошибка: ' + e.message); this.disabled = false; }
    });
}

async function renderBridgeCardsOnStats() {
    const statsContent = document.querySelector('.stats-grid');
    if (!statsContent || document.querySelector('#bridge-stats-zone')) return;
    await loadBridgePrefs();
    let services = [];
    try {
        services = (await bridgeDiscover())
            .filter(s => (s.state === 'running' || s.state === 'auth_required'))
            .filter(s => { const p = bridgePrefsCache && bridgePrefsCache[s.id]; return !p || p.enabled !== false; });
    } catch (e) { return; }
    if (!services.length) return;
    let html = '<div id="bridge-stats-zone"><h3 style="margin-top:30px;display:flex;align-items:center;gap:8px;">' +
        '<svg class="icon" width="20" height="20"><use href="/entware-manager/icons.svg?v=6#icon-modules"/></svg> Модули</h3>' +
        '<div class="stats-grid">' + services.map(s => {
            const [label, color] = BRIDGE_STATE_LABELS[s.state] || [s.state, '#718096'];
            return '<div class="stat-card" style="min-width:180px;">' +
                '<div style="font-weight:700;">' + escapeHtml(s.name) + '</div>' +
                '<div style="color:' + color + ';font-weight:600;margin-top:4px;">● ' + label + '</div>' +
                '</div>';
        }).join('') + '</div></div>';
    statsContent.insertAdjacentHTML('afterend', html);
}

async function renderLinksOnStats() {
    const statsContent = document.querySelector('.stats-grid');
    if (!statsContent) return;
    const links = await loadLinks();
    if (links.length === 0) return;
    if (document.querySelector('.links-grid')) return;

    let html = '<h3 style="margin-top: 30px;"><svg class="icon" width="20" height="20"><use href="/entware-manager/icons.svg?v=6#icon-link"/></svg> Полезные ссылки</h3><div class="links-grid">';
    links.forEach(link => {
        if (!isSafeLinkUrl(link.url)) return;
        const iconId = 'icon-' + (link.icon && isSafeLinkIcon(link.icon) ? link.icon : 'link');
        html += `
            <a href="${escapeHtml(link.url)}" target="_blank" rel="noopener noreferrer" class="link-card">
                <span class="link-icon"><svg class="icon" width="32" height="32"><use href="/entware-manager/icons.svg?v=6#${escapeHtml(iconId)}"/></svg></span>
                <span class="link-name">${escapeHtml(link.name)}</span>
            </a>
        `;
    });
    html += '</div>';
    statsContent.insertAdjacentHTML('afterend', html);
}

// isSafeLinkUrl — только http/https и относительные пути (/...).
function isSafeLinkUrl(u) {
    if (!u || typeof u !== 'string') return false;
    u = u.trim();
    if (u.length === 0 || u.length > 2048) return false;
    if (u.charAt(0) === '/') return u.charAt(1) !== '/';
    return /^https?:\/\//i.test(u);
}

// isSafeLinkIcon — только латиница, цифры, "-" и "_".
function isSafeLinkIcon(s) {
    if (!s) return true;
    if (s.length > 32) return false;
    return /^[a-zA-Z0-9_-]+$/.test(s);
}

function fmtBytesJS(n) {
    n = Number(n) || 0;
    if (n < 1024) return n + ' B';
    if (n < 1048576) return (n / 1024).toFixed(1) + ' КБ';
    if (n < 1073741824) return (n / 1048576).toFixed(1) + ' МБ';
    return (n / 1073741824).toFixed(2) + ' ГБ';
}

// Очистка tmpfs: сканирование корня + удаление выбранных папок (tmpfs_clean.cgi).
function tmpfsClean(mount) {
    mount = decodeURIComponent(mount);
    var state = { mount: mount, data: null };

    function renderResults(data) {
        var res = document.getElementById('tmpfs-clean-results');
        if (!res) return;
        var dirs = data.dirs || [];
        if (!dirs.length) {
            var th = document.getElementById('tmpfs-clean-threshold');
            var mb = th ? parseInt(th.value, 10) : (1 << 20);
            res.innerHTML = '<p style="padding:0.5rem 0;">Объектов размером от ' + fmtBytesJS(mb) + ' нет.</p>';
            return;
        }
        var rows = dirs.map(function(d) {
            var icon = d.type === 'file' ? 'file' : 'folder';
            return '<tr>' +
                '<td style="padding:4px 6px;width:30px;"><input type="checkbox" class="tmpfs-clean-cb" data-path="' + escapeHtml(d.path) + '"></td>' +
                '<td><span class="file-icon ' + icon + '"><svg class="icon" width="16" height="16"><use href="'+ICONS+'#icon-' + icon + '"/></svg></span> ' + escapeHtml(d.name) + '</td>' +
                '<td style="text-align:right;white-space:nowrap;">' + fmtBytesJS(d.bytes) + '</td>' +
                '<td style="text-align:right;color:var(--muted, #999);white-space:nowrap;">' + (d.type === 'file' ? 'файл' : d.files + ' файлов') + '</td>' +
                '</tr>';
        }).join('');
        res.innerHTML =
            '<p style="margin:0 0 6px;color:var(--muted,#999);font-size:13px;">Выбрано <span id="clean-count">0</span> из ' + dirs.length + ' · показываются файлы и папки</p>' +
            '<table class="file-table" style="width:100%;"><tbody>' + rows + '</tbody></table>' +
            '<label style="display:inline-flex;align-items:center;gap:6px;margin-top:8px;cursor:pointer;">' +
            '<input type="checkbox" id="clean-select-all"> Выбрать все</label>' +
            ' <button class="packages-delete-btn" style="background:#e53e3e;padding:4px 12px;margin-top:8px;" id="clean-do">' +
            '<svg class="icon" width="14" height="14"><use href="'+ICONS+'#icon-trash"/></svg> Удалить выбранные</button>';
        wireResults();
    }

    function wireResults() {
        var res = document.getElementById('tmpfs-clean-results');
        if (!res) return;
        var boxes = res.querySelectorAll('.tmpfs-clean-cb');
        for (var i = 0; i < boxes.length; i++) {
            boxes[i].addEventListener('change', updateCount);
        }
        var sa = document.getElementById('clean-select-all');
        if (sa) sa.addEventListener('change', function() {
            for (var j = 0; j < boxes.length; j++) { boxes[j].checked = sa.checked; }
            updateCount();
        });
        var del = document.getElementById('clean-do');
        if (del) del.addEventListener('click', doDelete);
    }

    function updateCount() {
        var res = document.getElementById('tmpfs-clean-results');
        var boxes = res ? res.querySelectorAll('.tmpfs-clean-cb:checked') : [];
        var el = document.getElementById('clean-count');
        if (el) el.textContent = boxes.length;
    }

    function doDelete() {
        if (!state.data) return;
        var res = document.getElementById('tmpfs-clean-results');
        var paths = [];
        var checked = res ? res.querySelectorAll('.tmpfs-clean-cb:checked') : [];
        for (var i = 0; i < checked.length; i++) { paths.push(checked[i].dataset.path); }
        if (!paths.length) { Toast.show('Ничего не выбрано', true); return; }

        function proceed(password) {
            // Пустой пароль допустим только когда авторизация выключена
            // (тогда POST уходит без password). Если требуется пароль и он
            // пуст/отмена — молча выходим.
            if (state.data.auth_required && !password) return;
            if (!confirm('Удалить выбранные папки (' + paths.length + ')?')) return;
            apiPost('/tmpfs_clean.cgi', 'paths=' + encodeURIComponent(paths.join('\n')) + '&password=' + encodeURIComponent(password || ''))
                .then(function(data) {
                    if (data.status === 'ok') {
                        Toast.show('Удалено: ' + data.deleted);
                        scan(); // повторное сканирование
                    } else if (data.message === 'Доступ запрещен') {
                        Toast.show('Доступ запрещен', true);
                    } else if (data.message === 'Неверный пароль') {
                        Toast.show('Неверный пароль', true);
                    } else {
                        Toast.show('Ошибка: ' + data.message, true);
                    }
                })
                .catch(function(e) { Toast.show('Ошибка: ' + e.message, true); });
        }

        if (state.data.auth_required) {
            // Пароль запрашивается при каждом действии (политика C8 — без кэша).
            Modal.promptPassword('Введите пароль для доступа к файлам', proceed);
        } else {
            proceed('');
        }
    }

    function scan() {
        var th = document.getElementById('tmpfs-clean-threshold');
        var mb = th ? parseInt(th.value, 10) : (1 << 20);
        var res = document.getElementById('tmpfs-clean-results');
        if (res) res.innerHTML = '<div class="loading-spinner" style="margin:12px 0;"></div>';
        apiGet('/tmpfs_clean.cgi?path=' + encodeURIComponent(state.mount) + '&min_bytes=' + mb)
            .then(function(data) {
                state.data = data;
                renderResults(data);
            })
            .catch(function(e) {
                if (res) res.innerHTML = '<p class="error">Ошибка: ' + escapeHtml(e.message) + '</p>';
            });
    }

    var bodies = [1048576, 5242880, 10485760, 52428800];
    var labels = ['1 МБ', '5 МБ', '10 МБ', '50 МБ'];
    var th = bodies.map(function(v, i) {
        return '<option value="' + v + '"' + (i === 0 ? ' selected' : '') + '>' + labels[i] + '</option>';
    }).join('');

    var html =
        '<div style="min-width:460px;">' +
        '<div style="margin-bottom:10px;display:flex;align-items:center;gap:10px;">' +
        '<label style="color:var(--muted,#999);">Минимум:</label>' +
        '<select id="tmpfs-clean-threshold" class="settings-input" style="max-width:140px;">' + th + '</select>' +
        '<button class="packages-delete-btn" style="background:#4a5568;padding:4px 12px;" id="clean-rescan">Сканировать</button>' +
        '</div>' +
        '<div style="color:var(--muted,#999);font-size:13px;margin-bottom:8px;">Корень: <b style="color:inherit;">' + escapeHtml(state.mount) + '</b></div>' +
        '<div id="tmpfs-clean-results"><div class="loading-spinner" style="margin:12px 0;"></div></div>' +
        '</div>';

    Modal.show(html, false, 'Очистка tmpfs');

    document.getElementById('tmpfs-clean-threshold').addEventListener('change', scan);
    document.getElementById('clean-rescan').addEventListener('click', scan);
    scan();
}

// Под-вкладки Настроек: чистое переключение классов, не гасит интервалы.
function switchEmTab(name) {
    const valid = ['terminal', 'notifications', 'security', 'maintenance'];
    if (valid.indexOf(name) === -1) name = 'terminal';
    document.querySelectorAll('#em-settings-tabs .em-tab').forEach(function(t) {
        t.classList.toggle('active', t.dataset.emtab === name);
    });
    document.querySelectorAll('.em-tab-panel').forEach(function(p) {
        p.classList.toggle('active', p.id === 'em-panel-' + name);
    });
    try { localStorage.setItem('settings_sub_tab', name); } catch(e) {}
}

async function renderSettingsTab() {
    const links = await loadLinks();
    let html = `
        <h2 style="display: flex; align-items: center; gap: 8px;">
            <span class="stat-icon" style="width: 28px; height: 28px;">
                <svg class="icon" width="28" height="28"><use href="/entware-manager/icons.svg?v=6#icon-settings"/></svg>
            </span>
            Настройки
        </h2>
        <div class="em-tabs" id="em-settings-tabs">
            <span class="em-tab" data-emtab="terminal">🖥 Терминал и ссылки</span>
            <span class="em-tab" data-emtab="notifications">🌐 Уведомления</span>
            <span class="em-tab" data-emtab="security">🔒 Безопасность</span>
            <span class="em-tab" data-emtab="maintenance">📦 Обслуживание</span>
        </div>
<div class="em-tab-panel active" id="em-panel-terminal">
        <h3><svg class="icon" width="20" height="20"><use href="/entware-manager/icons.svg?v=6#icon-terminal"/></svg> Управление ttyd</h3>
        <div id="ttyd-status"><div class="loading-spinner"></div></div>
        <div id="ttyd-controls" style="display: flex; gap: 20px; margin-top: 20px;">
          <div style="flex:1;"><h4>htop (порт 8089)</h4>
            <input type="password" id="htopPass" class="settings-input" placeholder="Пароль" style="margin-bottom: 8px;">
            <button class="packages-delete-btn" style="background:#4a5568;" onclick="controlTtyd('start', 8089, document.getElementById('htopPass').value, 'htop')" id="htop-start">Запустить</button>
            <button class="packages-delete-btn" style="background:#e53e3e;" onclick="controlTtyd('stop', 8089, '', '')" id="htop-stop">Остановить</button>
            <button class="packages-delete-btn" style="background:#f59e0b;" onclick="controlTtyd('restart', 8089, document.getElementById('htopPass').value, 'htop')" id="htop-restart">Перезапустить</button>
          </div>
          <div style="flex:1;"><h4>Терминал (порт 9089)</h4>
            <select id="termMode" class="settings-input" style="margin-bottom: 8px;">
              <option value="entware">Консоль Entware</option>
              <option value="telnet">Консоль роутера (telnet)</option>
            </select>
            <input type="password" id="termPass" class="settings-input" placeholder="Пароль" style="margin-bottom: 8px;">
            <button class="packages-delete-btn" style="background:#4a5568;" onclick="controlTtyd('start', 9089, document.getElementById('termPass').value, document.getElementById('termMode').value)" id="term-start">Запустить</button>
            <button class="packages-delete-btn" style="background:#e53e3e;" onclick="controlTtyd('stop', 9089, '', '')" id="term-stop">Остановить</button>
            <button class="packages-delete-btn" style="background:#f59e0b;" onclick="controlTtyd('restart', 9089, document.getElementById('termPass').value, document.getElementById('termMode').value)" id="term-restart">Перезапустить</button>
          </div>
        </div>
        <p class="settings-note" style="margin-top: 20px; font-size: 0.9rem;">
            Управление веб-терминалами ttyd. <strong>Пароль обязателен</strong> для обоих сервисов — терминал доступен извне через панель, без пароля запуск запрещён.<br>
            Доступ: панель → <code>/terminal/</code> и <code>/htop/</code> (тот же origin, порты 9089/8089 слушают только loopback).
        </p>
        <h3 style="margin-top: 30px;"><svg class="icon" width="20" height="20"><use href="/entware-manager/icons.svg?v=6#icon-link"/></svg> Управление ссылками на главной (общие для всех устройств)</h3>
        <p>Здесь можно добавлять, редактировать и удалять ссылки. Изменения сразу видны на всех устройствах.</p>
        <div style="margin-bottom: 15px;">
            <button id="addLinkBtn" class="packages-delete-btn" style="background:#4a5568;"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=6#icon-list"/></svg> Добавить ссылку</button>
        </div>
        <div class="packages-table-wrapper">
            <table class="packages-table" id="linksTable">
                <thead> <th>Иконка</th><th>Название</th><th>URL</th><th>Действия</th> </thead>
                <tbody id="linksTableBody">
    `;
    links.forEach((link, index) => {
        const iconId = link.icon && isSafeLinkIcon(link.icon) ? link.icon : 'link';
        html += `
            <tr data-index="${index}">
                <td style="min-width: 150px;">
                    <div style="display:flex; align-items:center; gap:8px;">
                        <span class="link-drag" title="Перетащите, чтобы изменить порядок"><svg class="icon" width="14" height="14"><use href="/entware-manager/icons.svg?v=6#icon-grip-dots"/></svg></span>
                        <svg class="icon" width="24" height="24"><use href="/entware-manager/icons.svg?v=6#icon-${iconId}"/></svg>
                        ${renderIconSelect(iconId)}
                    </div>
                 </td>
                 <td><input type="text" class="link-name settings-input" value="${escapeHtml(link.name)}"></td>
                 <td><input type="url" class="link-url settings-input" value="${escapeHtml(link.url)}"></td>
                 <td>
                    <div style="display:flex; gap:4px; align-items:center;">
                    <button class="packages-delete-btn" style="background:#27ae60;" title="Сохранить ссылку" onclick="saveLink(this)"><svg class="icon" width="14" height="14"><use href="/entware-manager/icons.svg?v=6#icon-disk"/></svg></button>
                    <button class="packages-delete-btn" style="background:#e53e3e;" title="Удалить ссылку" onclick="deleteLink(this)"><svg class="icon" width="14" height="14"><use href="/entware-manager/icons.svg?v=6#icon-default"/></svg></button>
                    </div>
                 </td>
              </tr>
        `;
    });
    html += `
                </tbody>
            </table>
        </div>
        <div style="margin-top: 15px;">
            <button id="saveAllLinksBtn" class="packages-delete-btn" style="background:#4a5568;"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=6#icon-disk"/></svg> Сохранить все на сервер</button>
            <button id="resetDefaultLinksBtn" class="packages-delete-btn" style="background:#f59e0b;"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=6#icon-refresh"/></svg> Сбросить по умолчанию</button>
        </div>
        </div>
    <div class="em-tab-panel" id="em-panel-security">
    <h3 style=""><svg class="icon" width="20" height="20"><use href="/entware-manager/icons.svg?v=6#icon-lock"/></svg> Защита панели</h3>
        <p>Пароль используется для входа в панель и для доступа к изменению и удалению файлов через встроенный менеджер (tmpfs). Если пароль задан — при открытии панели будет показан экран входа.</p>
        <div id="filemgr-auth-settings">
            <div class="loading-spinner"></div>
        </div>
        <h3 style="margin-top: 30px;"><svg class="icon" width="20" height="20"><use href="/entware-manager/icons.svg?v=6#icon-lock"/></svg> HTTPS (свой сертификат)</h3>
        <p style="font-size:0.85rem; color:var(--text-muted);">Дополнительный защищённый доступ по адресу <code>https://&lt;адрес-роутера&gt;:8443</code>. Сертификат создаётся автоматически (самоподписанный, на 10 лет) — браузер один раз попросит подтвердить доверие. Обычный HTTP при этом продолжает работать.</p>
        <div style="display:flex; align-items:center; gap:12px; margin:10px 0; flex-wrap:wrap;">
            <label><input type="checkbox" id="tls-enabled"> Включить HTTPS</label>
            <label>Порт: <input type="number" id="tls-port" value="8443" min="1" max="65535" class="settings-input" style="width:90px;"></label>
            <button class="packages-delete-btn" style="background:#2ecc71;" onclick="saveTLSConfig()">Применить</button>
            <span id="tls-status"></span>
        </div>

        <h3 style="margin-top: 30px;"><svg class="icon" width="20" height="20"><use href="/entware-manager/icons.svg?v=6#icon-shield"/></svg> Попытки входа</h3>
        <p style="font-size: 0.85rem; color: var(--text-muted);">Кто и когда пытался войти в панель (сегодня и вчера). Из внешней сети через KeenDNS/проброс порта виден настоящий IP посетителя.</p>
        <div style="margin: 10px 0;">
            <span id="authlog-failed-badge" style="display:none;"></span>
            <button class="packages-delete-btn" style="background:#4a5568;" onclick="loadAuthLog()"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=6#icon-refresh"/></svg> Обновить</button>
        </div>
        <div class="packages-table-wrapper">
            <table class="packages-table">
                <thead>
                    <tr><th>Время</th><th>IP адрес</th><th>Событие</th></tr>
                </thead>
                <tbody id="authlog-tbody">
                    <tr><td colspan="3">Загрузка...</td></tr>
                </tbody>
            </table>
        </div>
    `;
    html += `
        </div>
    <div class="em-tab-panel" id="em-panel-maintenance">
    <h3 style=""><svg class="icon" width="20" height="20"><use href="/entware-manager/icons.svg?v=6#icon-disk"/></svg> Бэкап и восстановление</h3>
        <p>Скачайте бэкап настроек перед сбросом роутера или для переноса на новое устройство.</p>
        <p style="font-size: 0.85rem; color: var(--text-muted);">Сохраняется: ссылки, настройки монитора, сети, watchdog и лога.</p>
        <div style="display: flex; gap: 12px; flex-wrap: wrap; align-items: center; margin-top: 10px;">
            <a href="/entware-cgi/backup.cgi" class="packages-delete-btn" style="background:#4a5568;" download="entware-manager-backup.tar.gz">
                <svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=6#icon-download"/></svg> Скачать бэкап
            </a>
            <label class="packages-delete-btn" style="background:#4a5568; cursor: pointer;">
                <svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=6#icon-refresh"/></svg> Восстановить
                <input type="file" id="restoreBackupFile" accept=".tar.gz" style="display: none;" onchange="restoreBackup(this)">
            </label>
            <button class="packages-delete-btn" style="background:#e67e22;" onclick="prepareOffline()">
                <svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=6#icon-download"/></svg> Подготовить офлайн-пакет
            </button>
            <span id="backupStatus"></span>
        </div>

        <h3 style="margin-top: 30px;"><svg class="icon" width="20" height="20"><use href="/entware-manager/icons.svg?v=6#icon-refresh"/></svg> Обновление</h3>
        <p>Проверьте и установите новую версию Entware Manager.</p>
        <div id="update-section" style="margin-top: 10px;">
            <div style="display: flex; gap: 12px; flex-wrap: wrap; align-items: center;">
                <span><strong>Текущая версия:</strong> <span id="update-current">загрузка...</span></span>
                <button id="update-check-btn" class="packages-delete-btn" style="background:#4a5568;" onclick="checkUpdate()">
                    <svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=6#icon-search"/></svg> Проверить
                </button>
                <button id="update-run-btn" class="packages-delete-btn" style="background:#2ecc71; display:none;" onclick="runUpdate()">
                    <svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=6#icon-refresh"/></svg> Обновить до <span id="update-version"></span>
                </button>
                <button id="update-reinstall-btn" class="packages-delete-btn" style="background:#e67e22;" onclick="reinstallUpdate()">
                    <svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=6#icon-update"/></svg> Переустановить
                </button>
            </div>
            <div id="update-status" style="margin-top: 8px;"></div>
            <pre id="update-log" style="background: var(--pre-bg); padding: 0.5rem; height: 150px; overflow-y: auto; margin-top: 8px; display:none; font-size: 0.85rem;"></pre>
        </div>
        </div>
    <div class="em-tab-panel active" id="em-panel-notifications">
    <h3 style=""><svg class="icon" width="20" height="20"><use href="/entware-manager/icons.svg?v=6#icon-email"/></svg> Telegram-уведомления</h3>
        <p class="text-secondary" style="font-size:0.75rem;">Отправка событий в Telegram через независимый шлюз. Токен бота хранится скрыто и не отображается. <a href="#" onclick="return openHelpTG()" style="color:var(--accent-color);">Инструкция и все команды бота — во вкладке «Справка».</a></p>
        <div id="telegram-form" style="margin-top: 10px; max-width: 520px;">
            <div style="display: flex; align-items: center; gap: 10px; margin-bottom: 10px;">
                <label style="flex: 0 0 160px;">Включить</label>
                <input type="checkbox" id="tg-enabled" style="transform: scale(1.3);">
            </div>
            <div style="display: flex; align-items: center; gap: 10px; margin-bottom: 10px;">
                <label style="flex: 0 0 160px;">Bot Token</label>
                <input type="password" id="tg-token" class="settings-input" placeholder="123456:ABC-DEF..." style="flex:1;">
            </div>
            <div style="display: flex; align-items: center; gap: 10px; margin-bottom: 10px;">
                <label style="flex: 0 0 160px;">Chat ID</label>
                <input type="text" id="tg-chat" class="settings-input" placeholder="123456789" style="flex:1;">
            </div>
            <div style="display: flex; align-items: center; gap: 10px; margin-bottom: 10px;">
                <label style="flex: 0 0 160px;">Прокси</label>
                <input type="text" id="tg-proxy" class="settings-input" placeholder="http://127.0.0.1:10871 или socks5://127.0.0.1:1080" style="flex:1;">
            </div>
            <div style="display: flex; align-items: center; gap: 10px; margin-bottom: 10px;">
                <label style="flex: 0 0 160px;">Уровень</label>
                <select id="tg-level" class="settings-input" style="flex:1;">
                    <option value="ERROR">ERROR (только ошибки)</option>
                    <option value="WARN">WARN (ошибки и предупреждения)</option>
                    <option value="INFO">INFO (все)</option>
                    <option value="OFF">OFF (выключено)</option>
                </select>
            </div>
            <div style="display: flex; align-items: center; gap: 10px; margin-bottom: 10px;">
                <label style="flex: 0 0 160px;">Источники</label>
                <div style="display:flex; gap:14px; flex-wrap:wrap;">
                    <label><input type="checkbox" id="tg-src-system" value="system"> Система</label>
                    <label><input type="checkbox" id="tg-src-monitor" value="monitor"> Монитор</label>
                    <label><input type="checkbox" id="tg-src-network" value="network"> Сеть</label>
                    <label><input type="checkbox" id="tg-src-service" value="service"> Службы</label>
                    <label><input type="checkbox" id="tg-src-packages" value="packages"> Пакеты</label>
                    <label><input type="checkbox" id="tg-src-login" value="login"> Входы в панель</label>
                </div>
            </div>
            <div style="display: flex; align-items: center; gap: 10px; margin-bottom: 10px;">
                <label style="flex: 0 0 160px;">Чат-бот</label>
                <input type="checkbox" id="tg-bot" style="transform: scale(1.3);">
            </div>
            <div style="display: flex; align-items: center; gap: 10px; margin-bottom: 10px;">
                <label style="flex: 0 0 160px;">Автозапуск</label>
                <input type="checkbox" id="tg-autostart" style="transform: scale(1.3);">
            </div>
            <div style="margin-bottom: 10px;">
                <div style="display: flex; align-items: center; gap: 10px;">
                    <label style="flex: 0 0 160px;">Тихий режим</label>
                    <input type="checkbox" id="tg-quiet" style="transform: scale(1.3);">
                    <span class="text-secondary" style="font-size:0.75rem;">с</span>
                    <input type="number" id="tg-quiet-from" class="settings-input" min="0" max="23" placeholder="23" style="width:60px;text-align:center;">
                    <span class="text-secondary" style="font-size:0.75rem;">до</span>
                    <input type="number" id="tg-quiet-to" class="settings-input" min="0" max="23" placeholder="7" style="width:60px;text-align:center;">
                </div>
                <p class="text-secondary" style="font-size:0.7rem;line-height:1.4;margin:4px 0 0 170px;">Ночью алерты не отправляются, а копятся — утром придёт сводка. Часы 0–23.</p>
            </div>
            <div style="display: flex; align-items: center; gap: 10px; margin-bottom: 10px;">
                <label style="flex: 0 0 160px;">Доп. chat ID</label>
                <input type="text" id="tg-chat-extra" class="settings-input" placeholder="111222333, 444555666 (через запятую)" style="flex:1;">
            </div>
            <div style="display: flex; gap: 10px; margin-top: 12px;">
                <button class="packages-delete-btn" style="background:#4a5568;" id="tg-save">Сохранить</button>
                <button class="packages-delete-btn" style="background:#27ae60;" id="tg-test">Отправить тест</button>
            </div>
            <div id="tg-status" style="margin-top: 10px; font-size: 0.9rem;"></div>
        </div>
        <h3 style="margin-top: 30px;"><svg class="icon" width="20" height="20"><use href="/entware-manager/icons.svg?v=6#icon-alert"/></svg> Критические пороги</h3>
        <p>При превышении порога бот пришлёт уведомление, при возврате в норму — сообщение о восстановлении.</p>
        <div id="tg-thresholds" style="margin-top: 10px; max-width: 520px;">
            <div style="display:flex;align-items:center;gap:10px;margin-bottom:8px;">
                <label style="flex:0 0 180px;">🔥 Температура CPU</label>
                <input type="checkbox" id="th-cpu_temp" style="transform:scale(1.2);">
                <input type="number" id="th-cpu_temp-val" class="settings-input" style="flex:1;max-width:90px;" min="0" max="150">
                <span style="color:var(--text-muted);">°C</span>
            </div>
            <div style="display:flex;align-items:center;gap:10px;margin-bottom:8px;">
                <label style="flex:0 0 180px;">📶 Температура WiFi0</label>
                <input type="checkbox" id="th-wifi0_temp" style="transform:scale(1.2);">
                <input type="number" id="th-wifi0_temp-val" class="settings-input" style="flex:1;max-width:90px;" min="0" max="150">
                <span style="color:var(--text-muted);">°C</span>
            </div>
            <div style="display:flex;align-items:center;gap:10px;margin-bottom:8px;">
                <label style="flex:0 0 180px;">📶 Температура WiFi1</label>
                <input type="checkbox" id="th-wifi1_temp" style="transform:scale(1.2);">
                <input type="number" id="th-wifi1_temp-val" class="settings-input" style="flex:1;max-width:90px;" min="0" max="150">
                <span style="color:var(--text-muted);">°C</span>
            </div>
            <div style="display:flex;align-items:center;gap:10px;margin-bottom:8px;">
                <label style="flex:0 0 180px;">⚡ Нагрузка CPU</label>
                <input type="checkbox" id="th-cpu_load" style="transform:scale(1.2);">
                <input type="number" id="th-cpu_load-val" class="settings-input" style="flex:1;max-width:90px;" min="0" max="100">
                <span style="color:var(--text-muted);">%</span>
            </div>
            <div style="display:flex;align-items:center;gap:10px;margin-bottom:8px;">
                <label style="flex:0 0 180px;">🧠 Занятость памяти</label>
                <input type="checkbox" id="th-ram_used" style="transform:scale(1.2);">
                <input type="number" id="th-ram_used-val" class="settings-input" style="flex:1;max-width:90px;" min="0" max="100">
                <span style="color:var(--text-muted);">%</span>
            </div>
            <div style="display:flex;align-items:center;gap:10px;margin-bottom:8px;">
                <label style="flex:0 0 180px;">💾 Температура дисков</label>
                <input type="checkbox" id="th-disk_temp" style="transform:scale(1.2);">
                <input type="number" id="th-disk_temp-val" class="settings-input" style="flex:1;max-width:90px;" min="0" max="150">
                <span style="color:var(--text-muted);">°C</span>
            </div>
            <div style="display:flex;gap:10px;margin-top:12px;">
                <button class="packages-delete-btn" style="background:#4a5568;" id="th-save">Сохранить пороги</button>
            </div>
            <div id="th-status" style="margin-top:10px;font-size:0.9rem;"></div>
        </div>
        </div><!-- /notifications -->
    `;
    contentDiv.innerHTML = html;

    // Восстановить сохранённую под-вкладку
    let savedSub = 'terminal';
    try {
        const v = localStorage.getItem('settings_sub_tab');
        if (['terminal','notifications','security','maintenance'].indexOf(v) !== -1) savedSub = v;
    } catch(e) {}
    switchEmTab(savedSub);

    // Обработчик кликов по табам
    const tabsBar = document.getElementById('em-settings-tabs');
    if (tabsBar) tabsBar.addEventListener('click', function(e){
        const t = e.target.closest('.em-tab');
        if (t) switchEmTab(t.dataset.emtab);
    });

    fetchTtydStatus();
    loadUpdateInfo();
    loadTelegramConfig();
    if (settingsInterval) clearInterval(settingsInterval);
    settingsInterval = setInterval(fetchTtydStatus, 5000);
    document.getElementById('addLinkBtn').addEventListener('click', addLinkRow);
    document.getElementById('saveAllLinksBtn').addEventListener('click', saveAllLinks);
    document.getElementById('resetDefaultLinksBtn').addEventListener('click', resetDefaultLinks);
    initLinksDrag();
    loadAuthConfig();
    loadAuthLog();
    loadTLSConfig();
}

// ===== Telegram-уведомления =====
function loadTelegramConfig() {
    apiGet('/telegram_config.cgi').then(function(data) {
        if (data.status === 'error') { showTgStatus('Ошибка: ' + escapeHtml(data.message), true); return; }
        var enabled = document.getElementById('tg-enabled');
        if (!enabled) return;
        enabled.checked = !!data.enabled;
        var bot = document.getElementById('tg-bot');
        if (bot) bot.checked = !!data.bot_enabled;
        var auto = document.getElementById('tg-autostart');
        if (auto) auto.checked = !!data.autostart;
        var level = document.getElementById('tg-level');
        if (level) level.value = data.level || 'ERROR';
        var chat = document.getElementById('tg-chat');
        if (chat) chat.value = data.chat_id || '';
        var proxy = document.getElementById('tg-proxy');
        if (proxy) proxy.value = data.proxy_url || '';
        var quiet = document.getElementById('tg-quiet');
        if (quiet) quiet.checked = !!data.quiet_enabled;
        var qFrom = document.getElementById('tg-quiet-from');
        if (qFrom) qFrom.value = data.quiet_from || '';
        var qTo = document.getElementById('tg-quiet-to');
        if (qTo) qTo.value = data.quiet_to || '';
        var chatExtra = document.getElementById('tg-chat-extra');
        if (chatExtra) chatExtra.value = data.chat_ids_extra || '';
        var srcs = data.sources || [];
        ['system','monitor','network','service','packages','login'].forEach(function(s) {
            var el = document.getElementById('tg-src-' + s);
            if (el) el.checked = srcs.indexOf(s) !== -1;
        });
        showTgStatus(data.configured ? 'Telegram настроен (токен скрыт).' : 'Telegram не настроен — укажите токен и chat_id.', false);
        var testBtn = document.getElementById('tg-test');
        if (testBtn) testBtn.disabled = !data.configured;
        fillThresholds(data.thresholds);
    }).catch(function(err) {
        showTgStatus('Ошибка загрузки: ' + escapeHtml(err.message), true);
    });
    var saveBtn = document.getElementById('tg-save');
    if (saveBtn && !saveBtn.dataset.bound) {
        saveBtn.dataset.bound = '1';
        saveBtn.addEventListener('click', saveTelegramConfig);
    }
    var testBtn2 = document.getElementById('tg-test');
    if (testBtn2 && !testBtn2.dataset.bound) {
        testBtn2.dataset.bound = '1';
        testBtn2.addEventListener('click', testTelegram);
    }
    var thSave = document.getElementById('th-save');
    if (thSave && !thSave.dataset.bound) {
        thSave.dataset.bound = '1';
        thSave.addEventListener('click', saveThresholds);
    }
}

// Заполнение полей критических порогов из конфига.
function fillThresholds(th) {
    th = th || {};
    var keys = ['cpu_temp','wifi0_temp','wifi1_temp','cpu_load','ram_used','disk_temp'];
    keys.forEach(function(k) {
        var cb = document.getElementById('th-' + k);
        var val = document.getElementById('th-' + k + '-val');
        if (cb) cb.checked = !!(th[k] && th[k].enabled);
        if (val) val.value = (th[k] && th[k].value) || 0;
    });
}

// Сборка JSON-объекта порогов из полей формы.
function collectThresholds() {
    var th = {};
    var keys = ['cpu_temp','wifi0_temp','wifi1_temp','cpu_load','ram_used','disk_temp'];
    keys.forEach(function(k) {
        var cb = document.getElementById('th-' + k);
        var val = document.getElementById('th-' + k + '-val');
        var value = parseInt(val ? val.value : 0) || 0;
        th[k] = { enabled: !!(cb && cb.checked), value: value };
    });
    return th;
}

// Сохранение критических порогов (POST с JSON-блоком thresholds).
function saveThresholds() {
    var th = collectThresholds();
    var statusEl = document.getElementById('th-status');
    statusEl.innerHTML = 'Сохранение...';
    // Включаем chat_id из формы, чтобы сохранение порогов не теряло введённый
    // (но ещё не сохранённый основной кнопкой) chat_id.
    var body = 'thresholds=' + encodeURIComponent(JSON.stringify(th));
    var chat = document.getElementById('tg-chat');
    if (chat && chat.value) body += '&chat_id=' + encodeURIComponent(chat.value);
    apiPost('/telegram_config.cgi', body).then(function(res) {
        if (res.status === 'ok') {
            statusEl.innerHTML = '<span style="color:#2ecc71;">✓ Пороги сохранены</span>';
        } else {
            statusEl.innerHTML = '<span style="color:#e53e3e;">Ошибка: ' + escapeHtml(res.message || res.status) + '</span>';
        }
    }).catch(function(err) {
        statusEl.innerHTML = '<span style="color:#e53e3e;">Ошибка: ' + escapeHtml(err.message) + '</span>';
    });
}

function tgSources() {
    var out = [];
    ['system','monitor','network','service','packages','login'].forEach(function(s) {
        var el = document.getElementById('tg-src-' + s);
        if (el && el.checked) out.push(s);
    });
    return out.join(',');
}

function saveTelegramConfig() {
    var data = 'enabled=' + (document.getElementById('tg-enabled').checked ? 'true' : 'false');
    data += '&bot_enabled=' + (document.getElementById('tg-bot').checked ? 'true' : 'false');
    data += '&autostart=' + (document.getElementById('tg-autostart').checked ? 'true' : 'false');
    data += '&level=' + encodeURIComponent(document.getElementById('tg-level').value);
    data += '&chat_id=' + encodeURIComponent(document.getElementById('tg-chat').value);
    data += '&sources=' + encodeURIComponent(tgSources());
    data += '&quiet_enabled=' + (document.getElementById('tg-quiet') && document.getElementById('tg-quiet').checked ? 'true' : 'false');
    var qf = document.getElementById('tg-quiet-from'), qt = document.getElementById('tg-quiet-to');
    if (qf && qf.value !== '') data += '&quiet_from=' + encodeURIComponent(qf.value);
    if (qt && qt.value !== '') data += '&quiet_to=' + encodeURIComponent(qt.value);
    var chatExtra = document.getElementById('tg-chat-extra');
    if (chatExtra) data += '&chat_ids_extra=' + encodeURIComponent(chatExtra.value);
    var proxy = document.getElementById('tg-proxy').value;
    if (proxy) data += '&proxy_url=' + encodeURIComponent(proxy);
    var token = document.getElementById('tg-token').value;
    if (token) data += '&bot_token=' + encodeURIComponent(token);
    showTgStatus('Сохранение...', false);
    apiPost('/telegram_config.cgi', data).then(function(res) {
        if (res.status === 'ok') {
            showTgStatus('✓ Настройки сохранены', false);
            document.getElementById('tg-token').value = '';
            loadTelegramConfig();
        } else {
            showTgStatus('Ошибка: ' + escapeHtml(res.message || res.status), true);
        }
    }).catch(function(err) {
        showTgStatus('Ошибка: ' + escapeHtml(err.message), true);
    });
}

function testTelegram() {
    showTgStatus('Отправка теста...', false);
    apiPost('/telegram_test.cgi', '').then(function(res) {
        showTgStatus(res.status === 'ok' ? '✓ Тестовое сообщение отправлено' : 'Ошибка: ' + escapeHtml(res.message || res.status), res.status !== 'ok');
    }).catch(function(err) {
        showTgStatus('Ошибка: ' + escapeHtml(err.message), true);
    });
}

function showTgStatus(msg, isError) {
    var el = document.getElementById('tg-status');
    if (!el) return;
    el.innerHTML = '<span style="color:' + (isError ? '#e53e3e' : 'var(--text-primary)') + ';">' + msg + '</span>';
}

window.restoreBackup = async function(input) {
    const file = input.files[0];
    if (!file) return;
    const statusEl = document.getElementById('backupStatus');
    statusEl.innerHTML = '<span style="color: var(--text-muted);">Восстановление...</span>';
    try {
        const response = await fetch('/entware-cgi/backup_restore.cgi', {
            method: 'POST',
            headers: { 'Content-Type': 'application/gzip' },
            body: await file.arrayBuffer()
        });
        const result = await response.json();
        if (result.status === 'ok') {
            statusEl.innerHTML = '<span style="color:#2ecc71;">✓ ' + escapeHtml(result.message || 'Восстановлено') + '</span>';
        } else {
            statusEl.innerHTML = '<span style="color:#e53e3e;">Ошибка: ' + escapeHtml(result.message || 'Неизвестная ошибка') + '</span>';
        }
    } catch (err) {
        statusEl.innerHTML = '<span style="color:#e53e3e;">Ошибка: ' + escapeHtml(err.message) + '</span>';
    }
    input.value = '';
};

window.prepareOffline = async function() {
    const statusEl = document.getElementById('backupStatus');
    statusEl.innerHTML = '<span style="color:#f59e0b;">⏳ Подготовка офлайн-пакета... (скачивание зависимостей, ~30-60 сек)</span>';

    try {
        const response = await fetch('/entware-cgi/prepare_offline.cgi?_=' + Date.now());
        if (!response.ok) {
            throw new Error('Сервер ответил ' + response.status);
        }

        const disposition = response.headers.get('Content-Disposition') || '';
        const filename = disposition.split('filename=')[1] ? disposition.split('filename=')[1].replace(/"/g, '').trim() : 'entware-manager-offline.tar.gz';

        const blob = await response.blob();
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);

        const filenameEsc = escapeHtml(filename);
        statusEl.innerHTML = '<span style="color:#2ecc71;">✓ Офлайн-пакет готов: ' + filenameEsc + '</span>' +
            '<p style="margin-top:6px;font-size:0.85rem;color:var(--text-secondary);">Файл сохранён в папку Загрузки браузера.</p>' +
            '<p style="margin-top:6px;font-size:0.85rem;color:var(--text-secondary);">Перенесите его на целевой роутер в <b>/opt/tmp/</b> (через SMB или USB) и выполните:</p>' +
            '<pre style="margin-top:8px;background:var(--pre-bg);padding:0.5rem;font-size:0.85rem;white-space:pre-wrap;">tar xzf ' + filenameEsc + '\ncd ' + filenameEsc.replace('.tar.gz', '') + '\nsh install-offline.sh</pre>';
    } catch (err) {
        statusEl.innerHTML = '<span style="color:#e53e3e;">Ошибка: ' + escapeHtml(err.message) + '</span>';
    }
};

async function loadUpdateInfo() {
    try {
        const data = await apiGet('/update_check.cgi');
        document.getElementById('update-current').textContent = data.current;
        if (data.has_update) {
            document.getElementById('update-version').textContent = data.latest;
            document.getElementById('update-run-btn').style.display = '';
            document.getElementById('update-status').innerHTML = '<span style="color:#2ecc71;">Доступна версия ' + escapeHtml(data.latest) + '</span>';
        } else if (data.error) {
            document.getElementById('update-status').innerHTML = '<span style="color:#e53e3e;">' + escapeHtml(data.error) + '</span>';
        } else {
            document.getElementById('update-status').innerHTML = '<span style="color:var(--text-muted);">Установлена последняя версия</span>';
        }
    } catch (err) {
        document.getElementById('update-current').textContent = 'ошибка';
    }
}

async function checkUpdate() {
    const btn = document.getElementById('update-check-btn');
    const statusEl = document.getElementById('update-status');
    btn.disabled = true;
    statusEl.innerHTML = '<span style="color:var(--text-muted);">Проверка...</span>';
    try {
        const data = await apiGet('/update_check.cgi');
        document.getElementById('update-current').textContent = data.current;
        if (data.has_update) {
            document.getElementById('update-version').textContent = data.latest;
            document.getElementById('update-run-btn').style.display = '';
            statusEl.innerHTML = '<span style="color:#2ecc71;">Доступна версия ' + escapeHtml(data.latest) + '</span>';
        } else if (data.error) {
            statusEl.innerHTML = '<span style="color:#e53e3e;">' + escapeHtml(data.error) + '</span>';
        } else {
            statusEl.innerHTML = '<span style="color:var(--text-muted);">Установлена последняя версия</span>';
        }
    } catch (err) {
        statusEl.innerHTML = '<span style="color:#e53e3e;">Ошибка: ' + escapeHtml(err.message) + '</span>';
    }
    btn.disabled = false;
}

async function runUpdate() {
    const btn = document.getElementById('update-run-btn');
    const logPre = document.getElementById('update-log');
    const statusEl = document.getElementById('update-status');
    btn.disabled = true;
    logPre.style.display = '';
    logPre.textContent = 'Запуск обновления...';
    statusEl.innerHTML = '<span style="color:#f59e0b;">Обновление запущено...</span>';

    try {
        const data = await apiPost('/update_run.cgi', '');
        if (data.status === 'error') {
            statusEl.innerHTML = '<span style="color:#e53e3e;">' + escapeHtml(data.message) + '</span>';
            btn.disabled = false;
            return;
        }
        statusEl.innerHTML = '<span style="color:#f59e0b;">Обновление... <span id="update-progress"></span> <span style="color:var(--text-muted); font-size:0.85em;">(обычно до 1 минуты при хорошей сети; не закрывайте страницу)</span></span>';

        // Poll status every 2 seconds
        pollUpdateStatus();
    } catch (err) {
        statusEl.innerHTML = '<span style="color:#e53e3e;">Ошибка: ' + escapeHtml(err.message) + '</span>';
        btn.disabled = false;
    }
}

let updatePollInterval;

async function reinstallUpdate() {
    if (!confirm('Переустановить текущую версию Entware Manager?\nБудет повторно скачана и установлена установленная версия. Это полезно, если что-то повредилось при прошлой установке (пропали вкладки, 404, битые файлы). Конфигурация сохранится.')) {
        return;
    }
    const btn = document.getElementById('update-reinstall-btn');
    const logPre = document.getElementById('update-log');
    const statusEl = document.getElementById('update-status');
    btn.disabled = true;
    logPre.style.display = '';
    logPre.textContent = 'Запуск переустановки...';
    statusEl.innerHTML = '<span style="color:#f59e0b;">Переустановка запущена...</span>';

    try {
        const data = await apiPost('/update_run.cgi', 'mode=reinstall');
        if (data.status === 'error') {
            statusEl.innerHTML = '<span style="color:#e53e3e;">' + escapeHtml(data.message) + '</span>';
            btn.disabled = false;
            return;
        }
        statusEl.innerHTML = '<span style="color:#f59e0b;">Переустановка... <span id="update-progress"></span> <span style="color:var(--text-muted); font-size:0.85em;">(обычно до 1 минуты при хорошей сети; не закрывайте страницу)</span></span>';

        pollUpdateStatus();
    } catch (err) {
        statusEl.innerHTML = '<span style="color:#e53e3e;">Ошибка: ' + escapeHtml(err.message) + '</span>';
        btn.disabled = false;
    }
}

function pollUpdateStatus() {
    if (updatePollInterval) clearInterval(updatePollInterval);
    const startedAt = Date.now();
    updatePollInterval = setInterval(async () => {
        try {
            const data = await apiGet('/update_status.cgi');
            const logPre = document.getElementById('update-log');
            if (data.lines && data.lines.length) {
                logPre.textContent = data.lines.join('\n');
                logPre.scrollTop = logPre.scrollHeight;
            }

            const statusEl = document.getElementById('update-status');

            if (data.status === 'done') {
                clearInterval(updatePollInterval);
                pkgCacheClearAll();
                statusEl.innerHTML = '<span style="color:#2ecc71;">✓ Обновление завершено</span>';
                document.getElementById('update-run-btn').style.display = 'none';
                const rbtn = document.getElementById('update-reinstall-btn');
                if (rbtn) rbtn.disabled = false;
                document.getElementById('update-current').textContent = data.lines.length > 0
                    ? (data.lines[data.lines.length-1].replace(/.*v/, '').replace(/ .*/, '') || '?')
                    : '?';
            } else if (data.status === 'error') {
                clearInterval(updatePollInterval);
                statusEl.innerHTML = '<span style="color:#e53e3e;">Ошибка: ' + (data.error || 'Неизвестная ошибка') + '</span>';
                document.getElementById('update-run-btn').disabled = false;
                const rbtn = document.getElementById('update-reinstall-btn');
                if (rbtn) rbtn.disabled = false;
            } else if (data.status === 'running') {
                const elapsed = Math.round((Date.now() - startedAt) / 1000);
                const mins = Math.floor(elapsed / 60);
                const secs = elapsed % 60;
                const timeStr = mins > 0 ? mins + ' мин ' + secs + ' с' : secs + ' с';
                const progress = data.progress || (data.lines.length > 0 ? data.lines[data.lines.length-1] : '');
                statusEl.innerHTML = '<span style="color:#f59e0b;">' +
                    '<b>' + escapeHtml(progress) + '</b>' +
                    ' <span style="color:var(--text-muted); font-size:0.85em;">(' + timeStr + ' — обычно до 1 минуты при хорошей сети; не закрывайте страницу)</span>' +
                    '</span>';
            }
        } catch (err) {
            // ignore polling errors
        }
    }, 2000);
}

function tlsStatus(msg, isErr) {
    const el = document.getElementById('tls-status');
    if (el) el.innerHTML = '<span style="color:' + (isErr ? '#e53e3e' : '#2ecc71') + ';">' + escapeHtml(msg) + '</span>';
}

async function loadTLSConfig() {
    const cb = document.getElementById('tls-enabled');
    if (!cb) return;
    try {
        const data = await apiGet('/tls_config.cgi');
        cb.checked = !!data.tls;
        if (data.tls_port) document.getElementById('tls-port').value = data.tls_port;
        const hint = [];
        if (!data.server_running) hint.push('режим lighttpd — недоступно');
        if (data.has_cert) hint.push('сертификат создан');
        if (hint.length) tlsStatus(hint.join(' · '), !data.server_running);
    } catch(e) { /* тихо: блок необязательный */ }
}

async function saveTLSConfig() {
    const password = prompt('Пароль панели для подтверждения:');
    if (!password) return;
    const enabled = document.getElementById('tls-enabled').checked ? 'true' : 'false';
    const port = document.getElementById('tls-port').value || '8443';
    tlsStatus('Сохранение и перезапуск сервера…', false);
    try {
        const res = await apiPost('/tls_config.cgi', 'password=' + encodeURIComponent(password) + '&enabled=' + enabled + '&port=' + encodeURIComponent(port));
        if (res.status === 'ok') {
            tlsStatus(res.message || 'Готово', false);
            setTimeout(() => location.reload(), 2500);
        } else {
            tlsStatus(res.message || res.status, true);
        }
    } catch(err) {
        tlsStatus('Ошибка: ' + err.message, true);
    }
}

async function loadAuthLog() {
    const tbody = document.getElementById('authlog-tbody');
    if (!tbody) return;
    try {
        const data = await apiGet('/auth_log.cgi');
        const entries = data.entries || [];
        const badge = document.getElementById('authlog-failed-badge');
        if (badge) {
            if (data.failed_24h > 0) {
                badge.style.display = 'inline-block';
                badge.style.cssText = 'display:inline-block;background:#e53e3e;color:#fff;border-radius:12px;padding:2px 10px;font-size:0.8rem;margin-right:8px;';
                badge.textContent = data.failed_24h + ' неудачных попыток за 24ч';
            } else {
                badge.style.display = 'none';
            }
        }
        if (!entries.length) {
            tbody.innerHTML = '<tr><td colspan="3">Попыток входа не зафиксировано</td></tr>';
            return;
        }
        tbody.innerHTML = entries.map(e => {
            let color = '';
            if (/Неверный пароль|CSRF|отклон/.test(e.message)) color = '#e53e3e';
            else if (/заблокирован|Слишком много/.test(e.message)) color = '#f59e0b';
            else if (/Успешный/.test(e.message)) color = '#2ecc71';
            return '<tr>' +
                '<td>' + escapeHtml(e.time) + '</td>' +
                '<td><code>' + escapeHtml(e.ip) + '</code></td>' +
                '<td style="' + (color ? 'color:' + color + ';font-weight:600;' : '') + '">' + escapeHtml(e.message) + '</td>' +
                '</tr>';
        }).join('');
    } catch(err) {
        tbody.innerHTML = '<tr><td colspan="3" style="color:var(--danger-color,#e53e3e);">Ошибка загрузки: ' + escapeHtml(err.message) + '</td></tr>';
    }
}

async function loadAuthConfig() {
    try {
        const data = await apiGet('/auth_config.cgi');
        const enabled = data.enabled;
        window.AUTH_CURRENTLY_ENABLED = !!(data.configured || enabled);
        let html = `
            <label style="display: flex; align-items: center; gap: 8px; margin: 10px 0;">
                <input type="checkbox" id="filemgrAuthEnabled" ${enabled ? 'checked' : ''} onchange="toggleFilemgrPassFields()">
                Включить защиту паролем
            </label>
            <div id="filemgrPassFields" style="display: ${enabled ? 'block' : 'none'}; margin: 10px 0;">
                <div style="margin-bottom: 8px;">
                    <label>Новый пароль (мин. 4 символа):</label>
                    <input type="password" id="filemgrPass" class="settings-input" style="max-width: 300px;" placeholder="Оставьте пустым чтобы не менять">
                </div>
                <div style="margin-bottom: 8px;">
                    <label>Подтверждение:</label>
                    <input type="password" id="filemgrPassConfirm" class="settings-input" style="max-width: 300px;" placeholder="Повторите пароль">
                </div>
            </div>
            <div style="margin: 10px 0; display: flex; align-items: center; gap: 10px;">
                <button class="packages-delete-btn" style="background:#4a5568;" onclick="saveAuthConfig()">Сохранить</button>
                <span id="filemgrAuthStatus"></span>
            </div>
        `;
        document.getElementById('filemgr-auth-settings').innerHTML = html;
    } catch (err) {
        document.getElementById('filemgr-auth-settings').innerHTML = '<p class="error">Ошибка загрузки настроек</p>';
    }
}

window.toggleFilemgrPassFields = function() {
    const enabled = document.getElementById('filemgrAuthEnabled').checked;
    document.getElementById('filemgrPassFields').style.display = enabled ? 'block' : 'none';
};

window.saveAuthConfig = async function() {
    const enabled = document.getElementById('filemgrAuthEnabled').checked;
    const password = document.getElementById('filemgrPass').value;
    const confirm = document.getElementById('filemgrPassConfirm').value;
    const statusEl = document.getElementById('filemgrAuthStatus');

    if (enabled && password && password !== confirm) {
        statusEl.innerHTML = '<span style="color:#e53e3e;">Пароли не совпадают</span>';
        return;
    }
    if (enabled && password && password.length < 8) {
        statusEl.innerHTML = '<span style="color:#e53e3e;">Пароль должен быть минимум 8 символов</span>';
        return;
    }

    async function doSave(currentPassword) {
        const formData = new URLSearchParams();
        formData.append('enabled', enabled ? 'true' : 'false');
        formData.append('password', password);
        if (currentPassword) formData.append('current_password', currentPassword);

        try {
            const data = await apiPost('/auth_config.cgi', formData.toString());
            statusEl.innerHTML = '<span style="color:#2ecc71;">✓ Настройки сохранены</span>';
            document.getElementById('filemgrPass').value = '';
            document.getElementById('filemgrPassConfirm').value = '';
            // после успешного сохранения обновить флаг
            window.AUTH_CURRENTLY_ENABLED = enabled;
            if (statusEl.previousElementSibling) {
                statusEl.previousElementSibling.disabled = false;
            }
        } catch (err) {
            statusEl.innerHTML = '<span style="color:#e53e3e;">Ошибка: ' + escapeHtml(err.message) + '</span>';
        }
    }

    // Если авторизация уже включена (или был задан пароль) — запросить текущий.
    if (window.AUTH_CURRENTLY_ENABLED) {
        Modal.promptPassword('Введите текущий пароль', function(val) {
            if (!val) {
                statusEl.innerHTML = '<span style="color:#e53e3e;">Отменено</span>';
                return;
            }
            doSave(val);
        });
    } else {
        doSave('');
    }
};

function addLinkRow() {
    const tbody = document.getElementById('linksTableBody');
    const newRow = document.createElement('tr');
    newRow.innerHTML = `
          <td>
            <div style="display:flex; align-items:center; gap:8px;">
                <span class="link-drag" title="Перетащите, чтобы изменить порядок"><svg class="icon" width="14" height="14"><use href="/entware-manager/icons.svg?v=6#icon-grip-dots"/></svg></span>
                <svg class="icon" width="24" height="24"><use href="/entware-manager/icons.svg?v=6#icon-link"/></svg>
                ${renderIconSelect('link')}
            </div>
          </td>
          <td><input type="text" class="link-name settings-input" value="Новая ссылка"></td>
          <td><input type="url" class="link-url settings-input" value="http://"></td>
          <td>
            <div style="display:flex; gap:4px; align-items:center;">
            <button class="packages-delete-btn" style="background:#27ae60;" title="Сохранить ссылку" onclick="saveLink(this)"><svg class="icon" width="14" height="14"><use href="/entware-manager/icons.svg?v=6#icon-disk"/></svg></button>
            <button class="packages-delete-btn" style="background:#e53e3e;" title="Удалить ссылку" onclick="deleteLink(this)"><svg class="icon" width="14" height="14"><use href="/entware-manager/icons.svg?v=6#icon-default"/></svg></button>
            </div>
          </td>
    `;
    tbody.appendChild(newRow);
}

window.saveLink = async function(btn) {
    const row = btn.closest('tr');
    const index = row.dataset.index;
    const iconSelect = row.querySelector('.link-icon-select');
    const icon = iconSelect ? iconSelect.value : 'link';
    const name = row.querySelector('.link-name').value;
    const url = row.querySelector('.link-url').value;
    if (!name || !url) { Toast.show('Заполните название и URL'); return; }
    const links = await loadLinks();
    const newLink = { name, url, icon };
    if (index !== undefined) links[index] = newLink;
    else links.push(newLink);
    if (await saveLinks(links)) {
        Toast.show('Ссылка сохранена');
        renderSettingsTab();
    }
};

window.deleteLink = async function(btn) {
    if (!confirm('Удалить ссылку?')) return;
    const row = btn.closest('tr');
    const index = row.dataset.index;
    if (index !== undefined) {
        const links = await loadLinks();
        links.splice(index, 1);
        if (await saveLinks(links)) {
            Toast.show('Ссылка удалена');
            renderSettingsTab();
        }
    } else {
        row.remove();
    }
};

// Перемещение ссылки вверх/вниз: меняет строки местами в таблице (DOM-порядок
// = порядок сохранения на сервере), затем сохраняет. Порядок на главной
// (статистика) следует порядку массива links.json.
// Перетаскивание строк таблицы ссылок мышкой (drag & drop). Порядок DOM-строк
// = порядок сохранения на сервере (links.json), поэтому после перестановки
// сохраняем и рендерим заново.
function saveCurrentLinkOrder() {
    const tbody = document.getElementById('linksTableBody');
    if (!tbody) return;
    const rows = tbody.querySelectorAll('tr');
    const links = [];
    rows.forEach(r => {
        const iconSelect = r.querySelector('.link-icon-select');
        const icon = iconSelect ? iconSelect.value : 'link';
        const name = r.querySelector('.link-name').value;
        const url = r.querySelector('.link-url').value;
        if (name && url) links.push({ name, url, icon });
    });
    return saveLinks(links).then(ok => {
        if (ok) { Toast.show('Порядок обновлён'); renderSettingsTab(); }
    });
}

// Привязывает drag & drop к строкам таблицы ссылок (вызывается после рендера).
function initLinksDrag() {
    const tbody = document.getElementById('linksTableBody');
    if (!tbody) return;
    let dragging = null;

    tbody.addEventListener('mousedown', function(e) {
        const grip = e.target.closest('.link-drag');
        if (!grip) return;
        const row = grip.closest('tr');
        if (!row) return;
        e.preventDefault();
        dragging = row;
        row.classList.add('link-dragging');
    });

    document.addEventListener('mousemove', function(e) {
        if (!dragging) return;
        // строка под курсором
        const rows = Array.prototype.slice.call(tbody.querySelectorAll('tr'));
        const target = rows.filter(function(r) {
            const rect = r.getBoundingClientRect();
            return e.clientY >= rect.top && e.clientY <= rect.bottom;
        })[0];
        if (target && target !== dragging) {
            const pos = e.clientY < target.getBoundingClientRect().top + target.offsetHeight / 2;
            if (pos) tbody.insertBefore(dragging, target);
            else tbody.insertBefore(dragging, target.nextSibling);
        }
    });

    document.addEventListener('mouseup', function() {
        if (!dragging) return;
        const row = dragging;
        dragging = null;
        row.classList.remove('link-dragging');
        // переиндексировать data-index всех строк
        const rows = tbody.querySelectorAll('tr');
        for (let i = 0; i < rows.length; i++) rows[i].dataset.index = i;
        saveCurrentLinkOrder();
    });
}

window.moveLink = null;

async function saveAllLinks() {
    const rows = document.querySelectorAll('#linksTable tbody tr');
    const links = [];
    rows.forEach(row => {
        const iconSelect = row.querySelector('.link-icon-select');
        const icon = iconSelect ? iconSelect.value : 'link';
        const name = row.querySelector('.link-name').value;
        const url = row.querySelector('.link-url').value;
        if (name && url) links.push({ name, url, icon });
    });
    if (await saveLinks(links)) {
        Toast.show('Ссылки сохранены на сервер');
        renderSettingsTab();
    }
}

async function resetDefaultLinks() {
    if (confirm('Восстановить ссылки по умолчанию? Текущие будут потеряны.')) {
        if (await saveLinks(getDefaultLinks())) {
            Toast.show('Ссылки сброшены к настройкам по умолчанию');
            renderSettingsTab();
        }
    }
}

async function opkgAction(event, action, pkg) {
    event.preventDefault();
    Modal.loading('Выполнение...');
    let url;
    if (action === 'install') url = '/install.cgi';
    else if (action === 'remove') url = '/remove.cgi';
    else if (action === 'upgrade') url = '/upgrade.cgi';
    else return;
    const formData = new URLSearchParams();
    formData.append('package', pkg);
    try {
        const response = await apiFetch(url, {
            method: 'POST',
            headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
            body: formData.toString()
        });
        const text = await response.text();
        Modal.show(text, false, `${action === 'install' ? 'Установка' : action === 'remove' ? 'Удаление' : 'Обновление'} пакета`);

        pkgCacheClearAll();

        // Если активна вкладка пакетов — перезагружаем её (единая вкладка «Пакеты»)
        const activeTab = document.querySelector('.menu li.active')?.dataset.tab;
        if (activeTab === 'packages' || activeTab === 'available' || activeTab === 'updates') {
            loadTab('packages');
        }
    } catch (err) {
        Modal.error('Ошибка соединения: ' + err.message);
    }
}

async function loadSystemServicesTab() {
    const html = `
        <h2 style="display: flex; align-items: center; gap: 8px;">
            <span class="stat-icon" style="width: 28px; height: 28px;">
                <svg class="icon" width="28" height="28"><use href="/entware-manager/icons.svg?v=6#icon-services"/></svg>
            </span>
            Системные службы и планировщик
        </h2>
        <div id="service-monitor-panel" style="background: var(--command-block-bg); padding: 1rem; border-radius: 12px; margin-bottom: 1rem;">
            <div style="display: flex; gap: 1rem; flex-wrap: wrap; align-items: center;">
                <span><strong>Мониторинг:</strong> <span id="service-watchdog-status" class="stat-value-normal">загрузка...</span></span>
                <button id="service-watchdog-start" class="packages-delete-btn"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=6#icon-play"/></svg> Запустить</button>
                <button id="service-watchdog-stop" class="packages-delete-btn"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=6#icon-stop"/></svg> Остановить</button>
                <button id="service-watchdog-restart" class="packages-delete-btn"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=6#icon-refresh"/></svg> Перезапустить</button>
                <label class="service-watch-toggle" title="Запускать демон мониторинга при загрузке роутера">
                    <input type="checkbox" id="service-autostart" style="display: none;">
                    <span class="toggle-slider"></span>
                    <span>Автозапуск при загрузке</span>
                </label>
            </div>
            <div style="margin-top: 10px; display: flex; gap: 20px; flex-wrap: wrap; align-items: center;">
                <label class="service-watch-toggle" title="Мониторить кастомный список процессов">
                    <input type="checkbox" id="service-watch-mode" style="display: none;">
                    <span class="toggle-slider"></span>
                    <span>Кастомный список</span>
                </label>
                <label class="service-watch-toggle" title="Показать список исключений">
                    <input type="checkbox" id="service-exclude-mode" style="display: none;">
                    <span class="toggle-slider"></span>
                    <span>Исключения</span>
                </label>
            </div>
            <div id="service-watch-list-container" style="margin-top: 10px; display: none;">
                <label>Список процессов (через запятую):</label>
                <input type="text" id="service-watch-list" style="width: 100%; padding: 8px; margin-top: 5px; border-radius: 6px; border: 1px solid var(--input-border); background: var(--input-bg); color: var(--text-primary);">
            </div>
            <div id="service-exclude-list-container" style="margin-top: 10px; display: none;">
                <label>Список исключений (через запятую):</label>
                <input type="text" id="service-exclude-list" style="width: 100%; padding: 8px; margin-top: 5px; border-radius: 6px; border: 1px solid var(--input-border); background: var(--input-bg); color: var(--text-primary);">
            </div>
            <div style="margin-top: 10px;">
                <label class="service-watch-toggle" title="Автоматически перезапускать службы при падении">
                    <input type="checkbox" id="service-auto-restart" style="display: none;">
                    <span class="toggle-slider"></span>
                    <span>Автоперезапуск служб при падении</span>
                </label>
            </div>
        </div>
        <div id="service-events-panel" style="background: var(--command-block-bg); padding: 1rem; border-radius: 12px; margin-bottom: 1rem; max-height: 200px; overflow-y: auto;">
            <h4 style="margin: 0 0 10px 0;">Последние события мониторинга</h4>
            <div id="service-events-list">
                <p style="color: var(--text-muted);">Загрузка...</p>
            </div>
        </div>
        <h3><svg class="icon" width="20" height="20"><use href="/entware-manager/icons.svg?v=6#icon-services"/></svg> Службы (init.d)</h3>
        <div id="services-list" class="packages-table-wrapper"><div class="loading-spinner"></div></div>
        <h3><svg class="icon" width="20" height="20"><use href="/entware-manager/icons.svg?v=6#icon-list"/></svg> Системный crontab (crontab -l)</h3>
        <div class="cron-editor">
            <textarea id="cron-system" rows="6" style="width:100%; font-family:monospace; padding:8px; border-radius:8px; border:1px solid var(--border-color); background:var(--input-bg); color:var(--text-primary);"></textarea>
            <div style="margin-top:10px; display:flex; gap:10px; flex-wrap:wrap;">
                <button id="save-cron-system" class="packages-delete-btn" style="background:#4a5568;"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=6#icon-disk"/></svg> Сохранить системный crontab</button>
                <span id="cron-system-message" style="margin-left:10px; align-self:center;"></span>
            </div>
        </div>
        <h3><svg class="icon" width="20" height="20"><use href="/entware-manager/icons.svg?v=6#icon-list"/></svg> Entware crontab (/opt/etc/crontab)</h3>
        <div class="cron-editor">
            <textarea id="cron-opt" rows="6" style="width:100%; font-family:monospace; padding:8px; border-radius:8px; border:1px solid var(--border-color); background:var(--input-bg); color:var(--text-primary);"></textarea>
            <div style="margin-top:10px; display:flex; gap:10px; flex-wrap:wrap;">
                <button id="save-cron-opt" class="packages-delete-btn" style="background:#4a5568;"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=6#icon-disk"/></svg> Сохранить Entware crontab</button>
                <span id="cron-opt-message" style="margin-left:10px; align-self:center;"></span>
            </div>
        </div>
    `;
    contentDiv.innerHTML = html;
    
    if (typeof servicesInterval !== 'undefined' && servicesInterval) {
        clearInterval(servicesInterval);
        servicesInterval = null;
    }
    
    fetchServices();
    fetchCrontabType('system', 'cron-system');
    fetchCrontabType('opt', 'cron-opt');
    document.getElementById('save-cron-system').addEventListener('click', () => saveCrontabType('system', 'cron-system', 'cron-system-message'));
    document.getElementById('save-cron-opt').addEventListener('click', () => saveCrontabType('opt', 'cron-opt', 'cron-opt-message'));
    
    initServiceWatchdog();
}

async function fetchServices() {
    const container = document.getElementById('services-list');
    if (!container) return;
    
    try {
        const services = await apiGet('/services.cgi');
        renderServices(services);
    } catch (err) {
        container.innerHTML = `<p class="error">Ошибка загрузки служб: ${escapeHtml(err.message)}</p>`;
    }
}

function renderServices(services) {
    let html = '<table class="packages-table"><thead> <th>Служба</th><th>Статус</th><th>PID</th><th>Автозапуск</th><th>Действия</th> </thead><tbody>';
    services.forEach(s => {
        let pidHtml = '-';
        if (s.status === 'running' && s.pids && s.pids.length > 0) {
            const pids = s.pids;
            const displayPid = escapeHtml(pids[0]);
            if (pids.length > 1) {
                const extra = pids.length - 1;
                pidHtml = `<span class="pid-link" data-service-name="${escapeHtml(s.name)}">${displayPid} <span class="pid-badge">+${extra}</span></span>`;
            } else {
                pidHtml = `<span class="pid-link" data-service-name="${escapeHtml(s.name)}">${displayPid}</span>`;
            }
        }
        html += `  <tr>
              <td>${escapeHtml(s.name)}</td>
              <td><span class="stat-value-${s.status === 'running' ? 'normal' : 'critical'}">${s.status}</span></td>
              <td>${pidHtml}</td>
              <td style="text-align: center;">
                <svg class="icon" width="20" height="20" style="display: inline-block; vertical-align: middle;">
                    <use href="/entware-manager/icons.svg?v=6#icon-${s.enabled ? 'check' : 'cross'}"/>
                </svg>
              </td>
              <td>
                <button class="packages-delete-btn" style="background:#4a5568;" data-service-name="${escapeHtml(s.name)}" data-service-action="start" ${s.status === 'running' ? 'disabled' : ''}>Запустить</button>
                <button class="packages-delete-btn" style="background:#e53e3e;" data-service-name="${escapeHtml(s.name)}" data-service-action="stop" ${s.status !== 'running' ? 'disabled' : ''}>Остановить</button>
                <button class="packages-delete-btn" style="background:#f59e0b;" data-service-name="${escapeHtml(s.name)}" data-service-action="restart" ${s.status !== 'running' ? 'disabled' : ''}>Перезапустить</button>
                <button class="packages-delete-btn" style="background:${s.enabled ? '#e53e3e' : '#27ae60'};" data-service-name="${escapeHtml(s.name)}" data-service-action="${s.enabled ? 'disable' : 'enable'}">Авто</button>
              </td>
          </tr>`;
    });
    html += '</tbody></table>';
    const servicesContainer = document.getElementById('services-list');
    servicesContainer.innerHTML = html;
    servicesContainer.querySelectorAll('[data-service-name]').forEach(el => {
        el.addEventListener('click', () => {
            const name = el.dataset.serviceName;
            if (el.dataset.serviceAction) {
                serviceAction(name, el.dataset.serviceAction);
            } else {
                showProcessList(name);
            }
        });
    });
}

window.showProcessList = function(serviceName) {
    // Найти службу по имени в уже загруженных данных
    apiGet('/services.cgi')
        .then(services => {
            const svc = services.find(s => s.name === serviceName);
            if (!svc || !svc.pids || svc.pids.length === 0) {
                Modal.info('Нет запущенных процессов для службы ' + escapeHtml(serviceName), 'Процессы: ' + escapeHtml(serviceName));
                return;
            }
            const pids = svc.pids;
            let html = `<div style="margin-bottom:10px; color:var(--text-secondary);">Найдено процессов: <b>${pids.length}</b></div>`;
            html += '<div class="process-list">';
            pids.forEach(pid => {
                html += `<div class="process-item">
                    <span class="process-pid">PID: <b>${escapeHtml(pid)}</b></span>
                    <button class="packages-delete-btn process-kill-btn" data-pid="${escapeHtml(pid)}" data-service-name="${escapeHtml(serviceName)}">
                        <svg class="icon" width="14" height="14"><use href="/entware-manager/icons.svg?v=6#icon-stop"/></svg> Убить
                    </button>
                </div>`;
            });
            html += '</div>';
            Modal.show(html, false, 'Процессы: ' + escapeHtml(serviceName));
            document.querySelectorAll('#modalBody .process-kill-btn').forEach(btn => {
                btn.addEventListener('click', () => killProcess(btn.dataset.pid, btn.dataset.serviceName));
            });
        })
        .catch(err => Modal.error('Ошибка загрузки процессов: ' + err.message));
};

window.killProcess = async function(pid, serviceName) {
    if (!confirm(`Завершить процесс PID ${pid} (${serviceName})?`)) return;
    const formData = new URLSearchParams();
    formData.append('pid', pid);
    try {
        const result = await apiPost('/kill_pid.cgi', formData.toString());
        if (result.status === 'ok') {
            Toast.show(`Процесс ${pid} завершён`, false, 3000);
            Modal.hide();
            fetchServices();
        } else {
            Toast.show('Ошибка: ' + (result.error || 'Не удалось завершить процесс'), true);
        }
    } catch (err) {
        Toast.show('Ошибка соединения: ' + err.message, true);
    }
};

window.serviceAction = async function(name, action) {
    const formData = new URLSearchParams();
    formData.append('name', name);
    formData.append('action', action);
    try {
        const result = await apiPost('/service_action.cgi', formData.toString());
        if (result.status === 'ok') {
            Toast.show(`Служба ${name}: ${action} выполнено`);
            fetchServices();
        } else {
            Toast.show('Ошибка: ' + (result.error || 'Неизвестная ошибка'), true);
        }
    } catch (err) {
        Toast.show('Ошибка соединения: ' + err.message, true);
    }
};

async function fetchCrontabType(type, textareaId) {
    try {
        const data = await apiGet('/crontab.cgi?type=' + type);
        document.getElementById(textareaId).value = data.crontab || '';
    } catch (err) {
        document.getElementById(textareaId).value = 'Ошибка загрузки crontab';
        Toast.show('Ошибка загрузки crontab: ' + err.message, true);
    }
}

async function saveCrontabType(type, textareaId, messageId) {
    const content = document.getElementById(textareaId).value;
    const formData = new URLSearchParams();
    formData.append('type', type);
    formData.append('crontab', content);
    try {
        const result = await apiPost('/crontab_update.cgi', formData.toString());
        if (result.status === 'ok') {
            const msgSpan = document.getElementById(messageId);
            if (msgSpan) msgSpan.innerHTML = '<span style="color:green;">✓ Сохранено</span>';
            setTimeout(() => { if (msgSpan) msgSpan.innerHTML = ''; }, 3000);
            Toast.show(`Crontab ${type} сохранён`);
        } else {
            Toast.show('Ошибка: ' + result.message, true);
        }
    } catch (err) {
        Toast.show('Ошибка: ' + err.message, true);
    }
}

function updateSidebarDateTime() {
    const el = document.getElementById('sidebarDatetime');
    if (el) {
        const now = new Date();
        const options = { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit' };
        el.textContent = now.toLocaleString(undefined, options);
    }
}
document.addEventListener('DOMContentLoaded', updateSidebarDateTime);
setInterval(updateSidebarDateTime, 1000);

// parseSize и formatSize — в lib/utils.js (единая реализация с TB и кириллицей).

function parseSpeed(str) {
    if (!str) return 0;
    const m = str.trim().match(/^([\d.]+)/);
    return m ? parseFloat(m[1]) : 0;
}

function parseIP(str) {
    if (!str) return [];
    const parts = str.trim().split('.');
    if (parts.length !== 4) return parts.map(p => parseInt(p) || 0);
    return parts.map(p => parseInt(p) || 0);
}

function sortTable(table, colIndex, dataType = 'string') {
    const tbody = table.querySelector('tbody');
    const rows = Array.from(tbody.querySelectorAll('tr'));
    let sortOrder = table.dataset.sortOrder === 'asc' ? 'desc' : 'asc';
    table.dataset.sortOrder = sortOrder;

    rows.sort((a, b) => {
        const aVal = sortableValue(a.cells[colIndex]?.innerText || '', dataType);
        const bVal = sortableValue(b.cells[colIndex]?.innerText || '', dataType);
        const cmp = compareSortValues(aVal, bVal);
        return sortOrder === 'asc' ? cmp : -cmp;
    });

    rows.forEach(row => tbody.appendChild(row));
    updateSortIndicators(table, colIndex, sortOrder);
}

// Сортирует строки таблицы с явным порядком (для повторного применения сохранённой сортировки).
function sortTableRows(table, colIndex, dataType, sortOrder) {
    const tbody = table.querySelector('tbody');
    const rows = Array.from(tbody.querySelectorAll('tr'));

    rows.sort((a, b) => {
        const aVal = sortableValue(a.cells[colIndex]?.innerText || '', dataType);
        const bVal = sortableValue(b.cells[colIndex]?.innerText || '', dataType);
        const cmp = compareSortValues(aVal, bVal);
        return sortOrder === 'asc' ? cmp : -cmp;
    });

    rows.forEach(row => tbody.appendChild(row));
    updateSortIndicators(table, colIndex, sortOrder);
}

// Конвертация значения ячейки для сравнения по типу колонки.
function sortableValue(text, dataType) {
    if (dataType === 'size') return parseSize(text);
    if (dataType === 'percent' || dataType === 'number') return parseFloat(text) || 0;
    if (dataType === 'speed') return parseSpeed(text);
    if (dataType === 'ip') return parseIP(text);
    if (dataType === 'version') { // "1.0.3 → 1.0.4" — сравниваем по текущей (до стрелки)
        const cur = text.split('→')[0].trim();
        return cur.split('.').map(s => parseFloat(s) || 0);
    }
    if (dataType === 'date') { // «—» всегда в конец
        if (text.trim() === '—') return '~~~~';
        return text.trim().toLowerCase();
    }
    if (dataType === 'status') { // сначала требующие действия
        const t = text.trim();
        if (t === 'есть обновление') return 0;
        if (t === 'установлен') return 1;
        return 2;
    }
    return text.trim().toLowerCase();
}

// Сравнение значений: числовые массивы (версии/IP) — посегментно, остальное — строкой.
function compareSortValues(aVal, bVal) {
    if (Array.isArray(aVal)) {
        const maxLen = Math.max(aVal.length, bVal.length);
        for (let i = 0; i < maxLen; i++) {
            const diff = (aVal[i] || 0) - (bVal[i] || 0);
            if (diff !== 0) return diff;
        }
        return 0;
    }
    return aVal > bVal ? 1 : aVal < bVal ? -1 : 0;
}

function updateSortIndicators(table, activeCol, sortOrder) {
    const headers = table.querySelectorAll('thead th');
    headers.forEach((th, idx) => {
        th.classList.remove('sort-asc', 'sort-desc');
        if (idx === activeCol) {
            th.classList.add(sortOrder === 'asc' ? 'sort-asc' : 'sort-desc');
        }
    });
}

// Универсальное включение сортировки: навешивает click на thead th.
// opts: { excludeCol: number[], dataTypes: string[], onSort: function(idx, dataType) }.
// onSort — кастомная логика (например, сохранение сортировки), по умолчанию sortTable.
function initTableSorting(table, opts) {
    opts = opts || {};
    if (table.dataset.sortable) return;
    table.dataset.sortable = 'true';
    const headers = table.querySelectorAll('thead th');
    headers.forEach((th, idx) => {
        if (opts.excludeCol && opts.excludeCol.indexOf(idx) !== -1) return;
        th.style.cursor = 'pointer';
        th.addEventListener('click', function() {
            let dataType = 'string';
            if (opts.dataTypes && opts.dataTypes[idx]) {
                dataType = opts.dataTypes[idx];
            } else {
                const columnText = th.innerText.toLowerCase();
                if (columnText.includes('размер')) {
                    dataType = 'size';
                } else if (columnText.includes('загрузка')) {
                    dataType = 'percent';
                }
            }
            if (opts.onSort) opts.onSort(idx, dataType);
            else sortTable(table, idx, dataType);
        });
    });
}

function enableTableSorting() {
    const statTables = document.querySelectorAll('.stat-card.tmpfs table, .stat-card.storage table');
    const fileTables = document.querySelectorAll('.file-manager .file-table');
    const allTables = [...statTables, ...fileTables];

    allTables.forEach(table => {
        initTableSorting(table);
    });
}

document.addEventListener('DOMContentLoaded', init);

function loadNetworkTab() {
    if (typeof initNetworkTab === 'function') {
        initNetworkTab();
        return;
    }
    const script = document.createElement('script');
    script.src = '/entware-manager/network.js?v=5';
    script.onload = () => {
        if (typeof initNetworkTab === 'function') initNetworkTab();
        else document.getElementById('content').innerHTML = '<p class="error">Ошибка загрузки модуля сети</p>';
    };
    script.onerror = () => document.getElementById('content').innerHTML = '<p class="error">Не удалось загрузить модуль сети</p>';
    document.head.appendChild(script);
}

function loadMonitorTab() {
    if (typeof initMonitorTab === 'function') {
        initMonitorTab();
        return;
    }
    const script = document.createElement('script');
    script.src = '/entware-manager/monitor.js?v=4';
    script.onload = () => {
        if (typeof initMonitorTab === 'function') initMonitorTab();
        else document.getElementById('content').innerHTML = '<p class="error">Ошибка загрузки модуля защиты</p>';
    };
    script.onerror = () => document.getElementById('content').innerHTML = '<p class="error">Не удалось загрузить модуль защиты</p>';
    document.head.appendChild(script);
}

const SERVICE_WATCHDOG = {
    intervalId: null,
    
    async init() {
        await this.loadStatus();
        await this.loadEvents();
        this.attachEvents();
    },
    
    async loadStatus() {
        const statusSpan = document.getElementById('service-watchdog-status');
        if (!statusSpan) return;
        
        try {
            const data = await apiGet('/service_watchdog/status.cgi');
            
            if (data.running) {
                statusSpan.textContent = `активен (PID: ${data.pid})`;
                statusSpan.className = 'stat-value-normal';
            } else {
                statusSpan.textContent = 'остановлен';
                statusSpan.className = 'stat-value-critical';
            }
            
            if (data.config) {
                const modeEl = document.getElementById('service-watch-mode');
                const listEl = document.getElementById('service-watch-list');
                const listContainer = document.getElementById('service-watch-list-container');
                const autoRestartEl = document.getElementById('service-auto-restart');
                const excludeModeEl = document.getElementById('service-exclude-mode');
                const excludeListEl = document.getElementById('service-exclude-list');
                const excludeListContainer = document.getElementById('service-exclude-list-container');
                if (modeEl) modeEl.checked = data.config.mode === 'custom';
                if (listEl) {
                    const wl = data.config.watch_list;
                    listEl.value = (Array.isArray(wl) ? wl.join(', ') : (wl ? wl : ''));
                }
                if (listContainer) listContainer.style.display = data.config.mode === 'custom' ? 'block' : 'none';
                if (autoRestartEl) autoRestartEl.checked = data.config.auto_restart == true || data.config.auto_restart === 'true';
                const autostartEl = document.getElementById('service-autostart');
                if (autostartEl) autostartEl.checked = data.config.autostart === true;
                
                let excludeArr = data.config.exclude_list;
                if (excludeArr && typeof excludeArr === 'object' && !Array.isArray(excludeArr)) {
                    excludeArr = Object.keys(excludeArr).filter(k => excludeArr[k] !== '');
                }
                excludeArr = Array.isArray(excludeArr) ? excludeArr : [];
                if (excludeListEl) excludeListEl.value = excludeArr.join(', ');
                
                if (excludeListContainer) {
                    const isChecked = excludeModeEl && excludeModeEl.checked;
                    excludeListContainer.style.display = isChecked ? 'block' : 'none';
                }
            }
        } catch (e) {
            statusSpan.textContent = 'ошибка';
            statusSpan.className = 'stat-value-warning';
        }
    },
    
    async loadEvents() {
        const container = document.getElementById('service-events-list');
        if (!container) return;
        
        try {
            const data = await apiGet('/service_watchdog/events.cgi?limit=15');
            
            if (data.events && data.events.length > 0) {
                container.innerHTML = data.events.map(e => {
                    const levelColor = e.level === 'ERROR' ? '#ef4444' : e.level === 'WARN' ? '#f59e0b' : '#10b981';
                    return `<div style="padding: 4px 0; border-bottom: 1px solid var(--border-color); font-size: 0.85rem;">
                        <span style="color: var(--text-muted);">${escapeHtml(e.timestamp.split(' ')[1])}</span>
                        <span style="color: ${levelColor}; margin-left: 8px;">[${escapeHtml(e.level)}]</span>
                        <strong style="margin-left: 8px;">${escapeHtml(e.service)}</strong>
                        <span style="margin-left: 4px;">${escapeHtml(e.event)}</span>
                        <span style="color: var(--text-muted);">${escapeHtml(e.details)}</span>
                    </div>`;
                }).join('');
            } else {
                container.innerHTML = '<p style="color: var(--text-muted);">Нет событий</p>';
            }
        } catch (e) {
            container.innerHTML = '<p style="color: var(--text-muted);">Ошибка загрузки событий</p>';
        }
    },
    
    attachEvents() {
        document.getElementById('service-watchdog-start')?.addEventListener('click', () => this.doAction('start'));
        document.getElementById('service-watchdog-stop')?.addEventListener('click', () => this.doAction('stop'));
        document.getElementById('service-watchdog-restart')?.addEventListener('click', () => this.doAction('restart'));
        document.getElementById('service-autostart')?.addEventListener('change', () => this.saveConfig());
        
        const modeCheckbox = document.getElementById('service-watch-mode');
        const listContainer = document.getElementById('service-watch-list-container');
        const watchList = document.getElementById('service-watch-list');
        const autoRestartCheckbox = document.getElementById('service-auto-restart');
        const excludeModeCheckbox = document.getElementById('service-exclude-mode');
        const excludeListContainer = document.getElementById('service-exclude-list-container');
        const excludeList = document.getElementById('service-exclude-list');
        
        modeCheckbox?.addEventListener('change', () => {
            if (listContainer) listContainer.style.display = modeCheckbox.checked ? 'block' : 'none';
            this.saveConfig();
        });
        
        excludeModeCheckbox?.addEventListener('change', () => {
            if (excludeListContainer) excludeListContainer.style.display = excludeModeCheckbox.checked ? 'block' : 'none';
            this.saveConfig();
        });
        
        watchList?.addEventListener('change', () => this.saveConfig());
        excludeList?.addEventListener('change', () => this.saveConfig());
        autoRestartCheckbox?.addEventListener('change', () => this.saveConfig());
    },
    
    async saveConfig() {
        const modeCheckbox = document.getElementById('service-watch-mode');
        const watchListEl = document.getElementById('service-watch-list');
        const autoRestartCheckbox = document.getElementById('service-auto-restart');
        const excludeModeCheckbox = document.getElementById('service-exclude-mode');
        const excludeListEl = document.getElementById('service-exclude-list');
        if (!modeCheckbox || !watchListEl) return;
        
        const mode = modeCheckbox.checked ? 'custom' : 'initd';
        const auto_restart = autoRestartCheckbox?.checked || false;
        const watchList = watchListEl.value.split(',').map(s => s.trim()).filter(s => s);
        
        let exclude_list;
        if (excludeModeCheckbox && excludeModeCheckbox.checked && excludeListEl && excludeListEl.value && excludeListEl.value.trim()) {
            exclude_list = excludeListEl.value.split(',').map(s => s.trim()).filter(s => s);
        } else if (excludeModeCheckbox && !excludeModeCheckbox.checked) {
            exclude_list = "emptylist";
        } else {
            exclude_list = null;
        }
        
        const autostartEl = document.getElementById('service-autostart');
        const autostart = autostartEl?.checked || false;

        try {
            const result = await apiPostJSON('/service_watchdog/config.cgi', { mode: mode, watch_list: watchList, auto_restart: auto_restart, autostart: autostart, exclude_list: exclude_list });
            if (result.status === 'ok') {
                Toast.show(result.message || 'Конфигурация сохранена');
                await this.loadStatus();
                await this.loadEvents();
                setTimeout(() => this.loadEvents(), 2000);
            }
        } catch (e) {
            Toast.show('Ошибка сохранения: ' + e.message, true);
        }
    },
    
    async doAction(action) {
        const btnId = `service-watchdog-${action}`;
        const btn = document.getElementById(btnId);
        if (btn) btn.disabled = true;
        
        try {
            const data = await apiPost('/service_watchdog/action.cgi', 'action=' + action);
            
            if (data.status === 'ok') {
                Toast.show(data.message);
                await this.loadStatus();
                await this.loadEvents();
            } else {
                Toast.show(data.message || 'Ошибка', true);
            }
        } catch (e) {
            Toast.show('Ошибка: ' + e.message, true);
        }
        
        if (btn) btn.disabled = false;
    },
    
    startUpdates() {
        if (this.intervalId) clearInterval(this.intervalId);
        this.intervalId = setInterval(() => {
            this.loadStatus();
            this.loadEvents();
        }, 10000);
    }
};

function initServiceWatchdog() {
    if (SERVICE_WATCHDOG.intervalId) {
        clearInterval(SERVICE_WATCHDOG.intervalId);
        SERVICE_WATCHDOG.intervalId = null;
    }
    SERVICE_WATCHDOG.init();
    SERVICE_WATCHDOG.startUpdates();
}

// Проверка системных зависимостей
async function checkSystemDeps() {
    Modal.loading('Проверка системных зависимостей...');
    try {
        const data = await apiGet('/check_deps.cgi');

        let html = '<div style="font-family: monospace; font-size: 13px;">';

        // Статус системы
        const statusColor = data.overall_status === 'ok' ? '#38a169' : 
                           data.overall_status === 'partial' ? '#d69e2e' : '#e53e3e';
        html += `<h3 style="color:${statusColor}; margin-top:0;">Статус системы: ${data.overall_status.toUpperCase()}</h3>`;
        html += `<p style="color:#718096; font-size:12px;">Обновлено: ${data.timestamp}</p><hr style="border-color:#4a5568;">`;

        // Базовые компоненты
        html += '<h4>Базовые компоненты:</h4><ul style="list-style:none; padding:0;">';
        html += `<li>opkg: <b style="color:${data.base.opkg ? '#38a169' : '#e53e3e'}">${data.base.opkg ? 'OK' : 'НЕ НАЙДЕН'}</b></li>`;
        const wsUp = data.base.lighttpd_running || data.base.entware_server_running;
        const wsLabel = data.base.entware_server_running ? 'entware-server' : (data.base.lighttpd_running ? 'lighttpd' : '—');
        html += `<li>Веб-сервер (${wsLabel}): <b style="color:${wsUp ? '#38a169' : '#e53e3e'}">${wsUp ? 'запущен' : 'НЕ ЗАПУЩЕН'}</b></li>`;
        html += '</ul>';

        // Утилиты
        html += '<h4>Утилиты BusyBox:</h4><ul style="list-style:none; padding:0;">';
        ['sed', 'awk', 'grep', 'ps'].forEach(u => {
            const ok = data.base[u];
            html += `<li>${u}: <b style="color:${ok ? '#38a169' : '#e53e3e'}">${ok ? 'OK' : 'НЕ НАЙДЕН'}</b></li>`;
        });
        html += '</ul>';

        // Пакеты
        html += '<h4>Пакеты Entware:</h4><ul style="list-style:none; padding:0;">';
        const deps = data.deps;
        html += `<li>cron (установлен): <b style="color:${deps.cron_installed ? '#38a169' : '#e53e3e'}">${deps.cron_installed ? 'ДА' : 'НЕТ'}</b></li>`;
        html += `<li>cron (запущен): <b style="color:${deps.cron_running ? '#38a169' : '#e53e3e'}">${deps.cron_running ? 'ДА' : 'НЕТ'}</b></li>`;
        html += `<li>jq: <b style="color:${deps.jq ? '#38a169' : '#e53e3e'}">${deps.jq ? 'OK' : 'НЕ НАЙДЕН'}</b></li>`;
        const ipStatus = deps.ip;
        const ipPath = deps.ip_path || '';
        const ipPkg = deps.ip_pkg_installed;
        html += `<li>ip (утилита): <b style="color:${ipStatus ? '#38a169' : '#e53e3e'}">${ipStatus ? 'OK (' + ipPath + ')' : 'НЕ НАЙДЕН'}</b></li>`;
        if (!ipPkg && ipStatus) {
            html += `<li style="color:#d69e2e;">• Пакет ip-full не установлен, но системная утилита работает</li>`;
        }
        html += `<li>curl: <b style="color:${deps.curl ? '#38a169' : '#e53e3e'}">${deps.curl ? 'OK' : 'НЕ НАЙДЕН'}</b></li>`;
        html += `<li>bash: <b style="color:${deps.bash ? '#38a169' : '#e53e3e'}">${deps.bash ? 'OK' : 'НЕ НАЙДЕН'}</b></li>`;
        const brctlStatus = deps.brctl;
        const brctlPath = deps.brctl_path || '';
        html += `<li>brctl (bridge-utils): <b style="color:${brctlStatus ? '#38a169' : '#e53e3e'}">${brctlStatus ? 'OK (' + brctlPath + ')' : 'НЕ НАЙДЕН'}</b></li>`;
        html += '</ul>';

        // Статус разделов
        html += '<h4>Статус разделов:</h4><ul style="list-style:none; padding:0;">';
        const sections = data.sections;
        Object.keys(sections).forEach(sec => {
            const st = sections[sec];
            const color = st === 'ok' ? '#38a169' : (st === 'partial' ? '#d69e2e' : '#e53e3e');
            html += `<li>${sec}: <b style="color:${color}">${st.toUpperCase()}</b></li>`;
        });
        html += '</ul>';

        // Модули панели (демоны и опциональные компоненты)
        if (data.modules) {
            const m = data.modules;
            const modRow = (label, on, hint) => {
                const color = on ? '#38a169' : '#718096';
                return `<li>${label}: <b style="color:${color};">${on ? 'работает' : 'остановлен'}</b>${!on && hint ? ' <small style="color:#718096;">(' + hint + ')</small>' : ''}</li>`;
            };
            html += '<h4>Модули панели:</h4><ul style="list-style:none; padding:0;">';
            html += modRow('Watchdog защиты', m.watchdog_monitor);
            html += modRow('Watchdog сети', m.watchdog_network);
            html += modRow('Watchdog служб', m.watchdog_services);
            html += modRow('Telegram-шлюз', m.telegram_gateway, 'включается в Настройках');
            html += modRow('Telegram-бот', m.telegram_bot, 'галочка «Чат-бот»');
            html += modRow('RDP-прокси', m.rdp_proxy_running, m.rdp_proxy_bin ? 'запускается на вкладке RDP' : 'не установлен');
            html += modRow('Терминал (ttyd)', m.ttyd_installed, 'opkg install ttyd');
            html += '</ul>';
        }

        // Проверка синтаксиса скриптов
        const syn = await apiGet('/check_syntax.cgi');
        html += '<h4>Проверка синтаксиса скриптов:</h4><ul style="list-style:none; padding:0;">';
        (syn.results || []).forEach(f => {
            const ok = f.status === 'ok';
            html += `<li>${escapeHtml(f.file)}: <b style="color:${ok ? '#38a169' : '#e53e3e'}">${ok ? 'ok' : 'ошибка'}</b></li>`;
            if (f.message) {
                html += `<li style="color:#e53e3e; margin-left:16px;"><small>${escapeHtml(f.message)}</small></li>`;
            }
        });
        html += `</ul><p style="color:${syn.total_errors > 0 ? '#e53e3e' : '#38a169'}; font-size:12px;">Ошибок: ${syn.total_errors}</p>`;

        // Рекомендации
        if (data.overall_status !== 'ok') {
            html += '<hr style="border-color:#4a5568;"><h4>Рекомендации:</h4>';
            if (!deps.cron_installed) html += '<p style="color:#e53e3e;">• Установите cron: <code>opkg install cron</code></p>';
            if (!deps.jq) html += '<p style="color:#e53e3e;">• Установите jq: <code>opkg install jq</code></p>';
            if (!deps.ip) html += '<p style="color:#e53e3e;">• Установите ip-full: <code>opkg install ip-full</code> (или проверьте наличие ip в системе)</p>';
            if (!deps.curl) html += '<p style="color:#e53e3e;">• Установите curl: <code>opkg install curl</code></p>';
            if (!deps.bash) html += '<p style="color:#e53e3e;">• Установите bash: <code>opkg install bash</code></p>';
            if (!deps.brctl) html += '<p style="color:#e53e3e;">• Установите bridge-utils: <code>opkg install bridge-utils</code></p>';
        }

        html += '</div>';
        Modal.info(html, 'Проверка системы');

    } catch (e) {
        Modal.error('Ошибка проверки: ' + e.message);
    }
}