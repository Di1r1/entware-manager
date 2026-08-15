// Entware Manager — разработчик Di1r1
// Версия: 0.82 (исправления XSS, безопасность)
// Дата: 2026-04-06

const BASE_URL = window.location.protocol + '//' + window.location.hostname;
const CACHE_KEY = 'entware_available_packages';
const CACHE_TIME_KEY = 'entware_available_timestamp';
const CACHE_MAX_AGE = 3600 * 1000;

let settingsInterval = null;
let servicesInterval = null;
let upgradableData = [];

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
            if (collapseToggle) collapseToggle.innerHTML = '<svg class="icon" width="20" height="20"><use href="/entware-manager/icons.svg?v=2#icon-chevron-left"/></svg>';
        }
        localStorage.removeItem('sidebar_collapsed');
    } else {
        const savedCollapsed = localStorage.getItem('sidebar_collapsed');
        if (savedCollapsed === 'true' && !sidebar.classList.contains('collapsed')) {
            sidebar.classList.add('collapsed');
            if (collapseToggle) collapseToggle.innerHTML = '<svg class="icon" width="20" height="20"><use href="/entware-manager/icons.svg?v=2#icon-chevron-right"/></svg>';
        } else if (savedCollapsed === 'false' && sidebar.classList.contains('collapsed')) {
            sidebar.classList.remove('collapsed');
            if (collapseToggle) collapseToggle.innerHTML = '<svg class="icon" width="20" height="20"><use href="/entware-manager/icons.svg?v=2#icon-chevron-left"/></svg>';
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
            collapseToggle.innerHTML = '<svg class="icon" width="20" height="20"><use href="/entware-manager/icons.svg?v=2#icon-chevron-right"/></svg>';
        } else {
            collapseToggle.innerHTML = '<svg class="icon" width="20" height="20"><use href="/entware-manager/icons.svg?v=2#icon-chevron-left"/></svg>';
        }
        collapseToggle.addEventListener('click', () => {
            sidebar.classList.toggle('collapsed');
            const isCollapsed = sidebar.classList.contains('collapsed');
            localStorage.setItem('sidebar_collapsed', isCollapsed);
            collapseToggle.innerHTML = isCollapsed
                ? '<svg class="icon" width="20" height="20"><use href="/entware-manager/icons.svg?v=2#icon-chevron-right"/></svg>'
                : '<svg class="icon" width="20" height="20"><use href="/entware-manager/icons.svg?v=2#icon-chevron-left"/></svg>';
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
        const savedTab = localStorage.getItem('entware_active_tab');
        Menu.setActiveTab(savedTab || 'stats');
    });
    handleResponsive();
    window.addEventListener('resize', debounce(handleResponsive, 200));
    const savedTab = localStorage.getItem('entware_active_tab');
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
    themeToggle.querySelector('use')?.setAttribute('href', '/entware-manager/icons.svg?v=2#' + (isNight ? 'icon-moon' : 'icon-sun'));
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

    if (themeToggle) {
        themeToggle.addEventListener('mouseenter', () => {
            buildThemePopup();
            if (popup) popup.classList.add('show');
        });
        themeToggle.addEventListener('click', () => {
            if (window.Theme) {
                window.Theme.set(window.Theme.current(), !window.Theme.isNight());
            }
            updateThemeIcon();
            if (popup) popup.classList.remove('show');
        });
    }
    if (popup) {
        popup.addEventListener('mouseleave', () => popup.classList.remove('show'));
        popup.addEventListener('click', (e) => {
            if (e.target.closest('.theme-swatch')) popup.classList.remove('show');
        });
    }
}

async function loadTab(tabName) {
    const ver = window.APP_VERSION || 'loading...';
    console.log(`[v${ver}] Загрузка вкладки:`, tabName);
    if (settingsInterval) clearInterval(settingsInterval);
    settingsInterval = null;
    if (servicesInterval) clearInterval(servicesInterval);
    servicesInterval = null;
    if (typeof MONITOR !== 'undefined' && MONITOR.stopUpdates) MONITOR.stopUpdates();
    if (typeof SMART !== 'undefined' && SMART.stopUpdates) SMART.stopUpdates();
    if (typeof RDP !== 'undefined' && RDP.stopUpdates) RDP.stopUpdates();

    if (tabName === 'available') { loadAvailableTab(); Menu.setActiveTab(tabName); return; }
    if (tabName === 'updates') { renderUpdatesTab(); Menu.setActiveTab(tabName); return; }
    if (tabName === 'processes') { renderProcessesTab(); Menu.setActiveTab(tabName); return; }
    if (tabName === 'terminal') { renderTerminalTab(); Menu.setActiveTab(tabName); return; }
    if (tabName === 'settings') { renderSettingsTab(); Menu.setActiveTab(tabName); return; }
    if (tabName === 'system-services') { loadSystemServicesTab(); Menu.setActiveTab(tabName); return; }
    if (tabName === 'monitor') { loadMonitorTab(); Menu.setActiveTab(tabName); return; }
    if (tabName === 'logs') { loadLogsTab(); Menu.setActiveTab(tabName); return; }
    if (tabName === 'network') { loadNetworkTab(); Menu.setActiveTab(tabName); return; }
    if (tabName === 'help') {
        contentDiv.innerHTML = '<p>Загрузка...</p>';
        try {
            const response = await apiFetch('/help.cgi');
            const html = await response.text();
            contentDiv.innerHTML = html;
            initHelpSearch();
            Menu.setActiveTab(tabName);
        } catch (err) {
            contentDiv.innerHTML = `<p class="error">Ошибка загрузки: ${err.message}</p>`;
            Menu.setActiveTab(tabName);
        }
        return;
    }
    if (tabName === 'smart') {
        if (!window.SMART_LOADED) {
            await loadScript('/entware-manager/smart.js?v=1');
            window.SMART_LOADED = true;
        }
        SMART.init(); Menu.setActiveTab(tabName); return;
    }
    if (tabName === 'rdp') {
        if (!window.RDP_LOADED) {
            await loadScript('/entware-manager/rdp.js?v=10');
            window.RDP_LOADED = true;
        }
        RDP.init(); Menu.setActiveTab(tabName); return;
    }

    contentDiv.innerHTML = '<p>Загрузка...</p>';
    try {
        const response = await apiFetch('/' + tabName + '.cgi');
        const html = await response.text();
        contentDiv.innerHTML = html;
        if (tabName === 'packages') initPackagesSearch();
        else if (tabName === 'stats') {
            initStatsTabs();
            loadNetworkStatus();
            setTimeout(() => { renderLinksOnStats(); enableTableSorting(); }, 100);
        }
        Menu.setActiveTab(tabName);
    } catch (err) {
        contentDiv.innerHTML = `<p class="error">Ошибка загрузки: ${err.message}</p>`;
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
        else Modal.info(`<pre>${data.info}</pre>`, `Пакет: ${pkg}`);
    } catch (err) {
        Modal.error('Ошибка запроса: ' + err.message);
    }
}

function initPackagesSearch() {
    initTableSearch('searchInput', 'packagesTable', 0);
    const table = document.getElementById('packagesTable');
    if (!table) return;
    const rows = table.getElementsByTagName('tr');
    for (let i = 1; i < rows.length; i++) {
        rows[i].style.cursor = 'pointer';
        rows[i].addEventListener('click', function(e) {
            if (e.target.tagName === 'INPUT' || e.target.tagName === 'FORM') return;
            const packageName = this.cells[0].textContent.trim();
            showPackageInfo(packageName);
        });
    }
}

function renderAvailableTable(packages) {
    let html = `<h2 style="display: flex; align-items: center; gap: 8px;">
        <span class="stat-icon" style="width: 28px; height: 28px;">
            <svg class="icon" width="28" height="28"><use href="/entware-manager/icons.svg?v=2#icon-package"/></svg>
        </span>
        Доступные пакеты (${packages.length})
    </h2>`;
    html += '<div style="display: flex; gap: 10px; align-items: center; margin-bottom: 24px;">';
    html += '<div class="search-container" style="display: flex; gap: 8px; align-items: center; flex: 1; background: var(--input-bg); border: 2px solid var(--input-border); border-radius: 40px; padding: 0 12px; transition: border-color 0.3s ease, box-shadow 0.3s ease;">';
    html += '<svg class="icon" width="18" height="18" style="color: var(--text-muted);"><use href="/entware-manager/icons.svg?v=2#icon-search"/></svg>';
    html += '<input type="text" id="searchAvailable" placeholder="Поиск по названию..." style="flex: 1; background: transparent; border: none; outline: none; padding: 14px 0; font-size: 16px; color: var(--text-primary);">';
    html += '</div>';
    html += '<button id="refreshAvailable" class="packages-delete-btn" style="background: #4a5568;"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-refresh"/></svg> Обновить</button>';
    html += '</div>';
    html += '<div class="packages-table-wrapper">';
    html += '<table class="packages-table" id="availableTable">';
    html += '<thead> <th>Пакет</th><th>Версия</th><th>Описание</th><th>Действие</th> </thead>';
    html += '<tbody id="availableTableBody">';
    packages.forEach(pkg => {
        html += `  <tr>
              <td>${escapeHtml(pkg.package)}</td>
              <td>${escapeHtml(pkg.version)}</td>
              <td>${escapeHtml(pkg.description)}</td>
              <td>
                <form method="post" style="display:inline;" onsubmit="opkgAction(event, 'install', '${escapeHtml(pkg.package)}'); return false;">
                    <input type="hidden" name="package" value="${escapeHtml(pkg.package)}">
                    <input type="submit" value="Установить" class="packages-delete-btn">
                </form>
              </td>
          </tr>`;
    });
    html += '</tbody></table></div>';
    contentDiv.innerHTML = html;

    initTableSearch('searchAvailable', 'availableTable', 0);
    document.getElementById('refreshAvailable').addEventListener('click', () => loadAvailableTab(true));
}

async function loadAvailableTab(forceRefresh = false) {
    const cached = localStorage.getItem(CACHE_KEY);
    const timestamp = localStorage.getItem(CACHE_TIME_KEY);
    const now = Date.now();
    if (!forceRefresh && cached && timestamp && (now - parseInt(timestamp) < CACHE_MAX_AGE)) {
        const packages = JSON.parse(cached);
        renderAvailableTable(packages);
    } else {
        contentDiv.innerHTML = '<p>Загрузка...</p>';
        try {
            const packages = await apiGet('/available.cgi');
            localStorage.setItem(CACHE_KEY, JSON.stringify(packages));
            localStorage.setItem(CACHE_TIME_KEY, now.toString());
            renderAvailableTable(packages);
        } catch (err) {
            contentDiv.innerHTML = `<p class="error">Ошибка загрузки: ${err.message}</p>`;
        }
    }
}

async function fetchUpgradable() {
    try {
        let data = await apiGet('/upgradable.cgi');
        data = data.filter(pkg => pkg.package && pkg.current && pkg.new && pkg.package !== 'undefined');
        upgradableData = data;
        renderUpdatesTabContent(upgradableData);
    } catch (err) {
        document.getElementById('upgradable-table-container').innerHTML = `<p class="error">Ошибка загрузки обновлений: ${err.message}</p>`;
    }
}

async function runPkgUpdate() {
    const updateBtn = document.getElementById('runUpdateBtn');
    const resultDiv = document.getElementById('update-result');
    updateBtn.disabled = true;
    updateBtn.innerHTML = '<svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-refresh"/></svg> Обновление...';
    resultDiv.innerHTML = '<div class="loading-spinner"></div>';

    try {
        const response = await apiFetch('/update.cgi?run=1');
        const text = await response.text();
        resultDiv.innerHTML = `<pre>${text}</pre>`;
        await fetchUpgradable();
    } catch (err) {
        resultDiv.innerHTML = `<p class="error">Ошибка: ${err.message}</p>`;
    } finally {
        updateBtn.disabled = false;
        updateBtn.innerHTML = '<svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-refresh"/></svg> Обновить списки пакетов';
    }
}

async function upgradeAll() {
    const upgradeAllBtn = document.getElementById('upgradeAllBtn');
    const resultDiv = document.getElementById('update-result');
    
    if (!confirm('Обновить все пакеты? Это может занять продолжительное время.')) {
        return;
    }
    
    upgradeAllBtn.disabled = true;
    upgradeAllBtn.innerHTML = '<svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-refresh"/></svg> Обновление...';
    resultDiv.innerHTML = '<div class="loading-spinner"></div>';
    
    try {
        const response = await apiFetch('/upgrade.cgi', {
            method: 'POST',
            headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
            body: 'upgrade_all=1'
        });
        const text = await response.text();
        resultDiv.innerHTML = `<pre>${text}</pre>`;
        await fetchUpgradable();
    } catch (err) {
        resultDiv.innerHTML = `<p class="error">Ошибка: ${err.message}</p>`;
    } finally {
        upgradeAllBtn.disabled = false;
        upgradeAllBtn.innerHTML = '<svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-update"/></svg> Обновить все пакеты';
    }
}

function renderUpdatesTabContent(packages) {
    const container = document.getElementById('upgradable-table-container');
    if (!container) return;
    if (packages.length === 0) {
        container.innerHTML = '<p>Все пакеты актуальны.</p>';
        return;
    }
    let html = '<div class="packages-table-wrapper"><table class="packages-table"><thead> <th>Пакет</th><th>Текущая версия</th><th>Новая версия</th><th>Действие</th> </thead><tbody>';
    packages.forEach(pkg => {
        html += `  <tr>
              <td>${escapeHtml(pkg.package)}</td>
              <td>${escapeHtml(pkg.current)}</td>
              <td>${escapeHtml(pkg.new)}</td>
              <td>
                <form method="post" style="display:inline;" onsubmit="opkgAction(event, 'upgrade', '${escapeHtml(pkg.package)}'); return false;">
                    <input type="hidden" name="package" value="${escapeHtml(pkg.package)}">
                    <input type="submit" value="Обновить" class="packages-delete-btn" style="background:#27ae60;">
                </form>
              </td>
          </tr>`;
    });
    html += '</tbody></table></div>';
    container.innerHTML = html;
}

function renderUpdatesTab() {
    const html = `
        <h2 style="display: flex; align-items: center; gap: 8px;">
            <span class="stat-icon" style="width: 28px; height: 28px;">
                <svg class="icon" width="28" height="28"><use href="/entware-manager/icons.svg?v=2#icon-update"/></svg>
            </span>
            Обновления и списки
        </h2>
        <div style="margin-bottom: 20px;">
            <button id="runUpdateBtn" class="packages-delete-btn" style="background:#4a5568;"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-refresh"/></svg> Обновить списки пакетов</button>
            <button id="upgradeAllBtn" class="packages-delete-btn" style="background:#e67e22;"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-update"/></svg> Обновить все пакеты</button>
        </div>
        <div id="update-result" style="margin-bottom: 20px;"></div>
        <h3>Доступные обновления</h3>
        <div id="upgradable-table-container"><div class="loading-spinner"></div></div>
    `;
    contentDiv.innerHTML = html;
    document.getElementById('runUpdateBtn').addEventListener('click', runPkgUpdate);
    document.getElementById('upgradeAllBtn').addEventListener('click', upgradeAll);
    fetchUpgradable();
}

function renderProcessesTab() {
    const html = `
        <h2 style="display: flex; align-items: center; gap: 8px;">
            <span class="stat-icon" style="width: 28px; height: 28px;">
                <svg class="icon" width="28" height="28"><use href="/entware-manager/icons.svg?v=2#icon-process"/></svg>
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
                    <a href="/htop/" target="_blank" rel="noopener" class="packages-delete-btn" style="background:#4a5568;"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-link"/></svg> Открыть в новой вкладке</a>
                </div>
                <iframe src="/htop/" width="100%" height="600" style="border: none; border-radius: 8px;"></iframe>
            `;
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
                <svg class="icon" width="28" height="28"><use href="/entware-manager/icons.svg?v=2#icon-terminal"/></svg>
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
                    <a href="/terminal/" target="_blank" rel="noopener" class="packages-delete-btn" style="background:#4a5568;"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-link"/></svg> Открыть в новой вкладке</a>
                </div>
                <iframe src="/terminal/" width="100%" height="600" style="border: none; border-radius: 8px;"></iframe>
            `;
        } else {
            title.textContent = 'Терминал';
            container.innerHTML = '<p style="color: var(--text-secondary); font-size: 0.9rem;">Терминал не запущен. Откройте <b>Настройки → Терминал</b>, задайте пароль и нажмите <b>Запустить</b>.</p>';
        }
    } catch (err) {
        const container = document.getElementById('terminal-content');
        if (container) container.innerHTML = '<p class="error">Ошибка: ' + escapeHtml(err.message) + '</p>';
    }
}

function loadLogsTab() {
    const html = `
        <h2 style="display: flex; align-items: center; gap: 8px;">
            <span class="stat-icon" style="width: 28px; height: 28px;">
                <svg class="icon" width="28" height="28"><use href="/entware-manager/icons.svg?v=2#icon-list"/></svg>
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
                    <svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-trash"/></svg> Очистить логи старше 30 дней
                </button>
                <button id="rotateNowBtn" class="packages-delete-btn" style="background:#f59e0b;">
                    <svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-refresh"/></svg> Ротация сейчас
                </button>
                <button id="toggleLoggingBtn" class="packages-delete-btn" style="background:#4a5568; display: flex; align-items: center; gap: 8px;">
                    <span id="loggingStatusIndicator" style="display: inline-block; width: 12px; height: 12px; border-radius: 50%; background: gray;"></span>
                    Настройки логирования
                </button>
                <button id="systemEventsBtn" class="packages-delete-btn" style="background:#2c5282;">
                    <svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-info"/></svg> Системные события
                </button>
            </div>
            <div id="system-controls" style="margin-top: 16px; display: none; gap: 12px; flex-wrap: wrap; align-items: center;">
                <select id="system-source" class="packages-delete-btn" style="background: var(--input-bg); color: var(--text-primary);">
                    <option value="">Выберите источник</option>
                </select>
                <button id="refreshSystemLogs" class="packages-delete-btn" style="background:#4a5568;">
                    <svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-refresh"/></svg> Обновить
                </button>
                <button id="searchByNameBtn" class="packages-delete-btn" style="background:#4a5568;">
                    <svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-search"/></svg> Поиск по имени
                </button>
                <button id="clearDynamicSourcesBtn" class="packages-delete-btn" style="background:#e53e3e;">
                    <svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-trash"/></svg> Очистить источники
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
        Toast.show(data.message);
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
            <div style="background:var(--card-bg,#16213e);border-radius:12px;padding:0;max-width:900px;width:95%;max-height:90vh;display:flex;flex-direction:column;overflow:hidden;">
                <div style="display:flex;justify-content:space-between;align-items:center;padding:16px 20px;border-bottom:1px solid var(--border-color,#333);">
                    <span style="font-size:16px;font-weight:500;">Системные события</span>
                    <button id="closeSystemModal" style="background:none;border:none;color:#fff;font-size:24px;cursor:pointer;">&times;</button>
                </div>
                <div id="systemLogContent" style="flex:1;overflow:auto;padding:16px;"></div>
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
        document.getElementById('ttyd-status').innerHTML = `<p class="error">Ошибка: ${err.message}</p>`;
    }
}

function updateTtydStatus(data) {
    const statusDiv = document.getElementById('ttyd-status');
    const htop = data.htop;
    const term = data.terminal;
    const modeLabel = term.mode === 'telnet' ? 'Telnet' : 'Entware';

    let html = '<h3>Текущее состояние ttyd</h3>';
    html += '<table class="stat-table">';
    html += `  <tr><td>htop (8089, доступ /htop/):</td><td><span class="${htop.state === 'running' ? 'stat-value-normal' : 'stat-value-critical'}">${htop.state}</span> ${htop.pid ? '(PID ' + htop.pid + ')' : ''}</td></tr>`;
    html += `  <tr><td>Терминал (9089, доступ /terminal/):</td><td><span class="${term.state === 'running' ? 'stat-value-normal' : 'stat-value-critical'}">${term.state}</span> ${term.pid ? '(PID ' + term.pid + ')' : ''} ${term.state === 'running' ? '(' + modeLabel + ')' : ''}</td></tr>`;
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

async function renderLinksOnStats() {
    const statsContent = document.querySelector('.stats-grid');
    if (!statsContent) return;
    const links = await loadLinks();
    if (links.length === 0) return;
    if (document.querySelector('.links-grid')) return;

    let html = '<h3 style="margin-top: 30px;"><svg class="icon" width="20" height="20"><use href="/entware-manager/icons.svg?v=2#icon-link"/></svg> Полезные ссылки</h3><div class="links-grid">';
    links.forEach(link => {
        if (!isSafeLinkUrl(link.url)) return;
        const iconId = 'icon-' + (link.icon && isSafeLinkIcon(link.icon) ? link.icon : 'link');
        html += `
            <a href="${escapeHtml(link.url)}" target="_blank" rel="noopener noreferrer" class="link-card">
                <span class="link-icon"><svg class="icon" width="32" height="32"><use href="/entware-manager/icons.svg?v=2#${escapeHtml(iconId)}"/></svg></span>
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

    function getPass() {
        try { return sessionStorage.getItem('filemgr_pass') || ''; }
        catch (e) { return ''; }
    }
    function setPass(p) {
        try { sessionStorage.setItem('filemgr_pass', p); } catch (e) {}
    }

    function renderResults(data) {
        var res = document.getElementById('tmpfs-clean-results');
        if (!res) return;
        var dirs = data.dirs || [];
        if (!dirs.length) {
            var th = document.getElementById('tmpfs-clean-threshold');
            var mb = th ? parseInt(th.value, 10) : (1 << 20);
            res.innerHTML = '<p style="padding:0.5rem 0;">Папок размером от ' + fmtBytesJS(mb) + ' нет.</p>';
            return;
        }
        var rows = dirs.map(function(d) {
            return '<tr>' +
                '<td style="padding:4px 6px;width:30px;"><input type="checkbox" class="tmpfs-clean-cb" data-path="' + escapeHtml(d.path) + '"></td>' +
                '<td><span class="file-icon folder"><svg class="icon" width="16" height="16"><use href="'+ICONS+'#icon-folder"/></svg></span> ' + escapeHtml(d.name) + '</td>' +
                '<td style="text-align:right;white-space:nowrap;">' + fmtBytesJS(d.bytes) + '</td>' +
                '<td style="text-align:right;color:var(--muted, #999);white-space:nowrap;">' + d.files + ' файлов</td>' +
                '</tr>';
        }).join('');
        res.innerHTML =
            '<p style="margin:0 0 6px;color:var(--muted,#999);font-size:13px;">Выбрано <span id="clean-count">0</span> из ' + dirs.length + ' · показываются закрытые/свободные папки</p>' +
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
        var password = '';
        if (state.data.auth_required) {
            password = getPass();
            if (!password) {
                password = prompt('Введите пароль для доступа к файлам:');
                if (!password) return;
                setPass(password);
            }
        }
        if (!confirm('Удалить выбранные папки (' + paths.length + ')?')) return;
        apiPost('/tmpfs_clean.cgi', 'paths=' + encodeURIComponent(paths.join('\n')) + '&password=' + encodeURIComponent(password))
            .then(function(data) {
                if (data.status === 'ok') {
                    Toast.show('Удалено: ' + data.deleted);
                    scan(); // повторное сканирование
                } else if (data.message === 'Доступ запрещен') {
                    Toast.show('Доступ запрещен', true);
                } else if (data.message === 'Неверный пароль') {
                    setPass('');
                    Toast.show('Неверный пароль', true);
                } else {
                    Toast.show('Ошибка: ' + data.message, true);
                }
            })
            .catch(function(e) { Toast.show('Ошибка: ' + e.message, true); });
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

async function renderSettingsTab() {
    const links = await loadLinks();
    let html = `
        <h2 style="display: flex; align-items: center; gap: 8px;">
            <span class="stat-icon" style="width: 28px; height: 28px;">
                <svg class="icon" width="28" height="28"><use href="/entware-manager/icons.svg?v=2#icon-settings"/></svg>
            </span>
            Настройки
        </h2>
        <h3><svg class="icon" width="20" height="20"><use href="/entware-manager/icons.svg?v=2#icon-terminal"/></svg> Управление ttyd</h3>
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
        <h3 style="margin-top: 30px;"><svg class="icon" width="20" height="20"><use href="/entware-manager/icons.svg?v=2#icon-link"/></svg> Управление ссылками на главной (общие для всех устройств)</h3>
        <p>Здесь можно добавлять, редактировать и удалять ссылки. Изменения сразу видны на всех устройствах.</p>
        <div style="margin-bottom: 15px;">
            <button id="addLinkBtn" class="packages-delete-btn" style="background:#4a5568;"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-list"/></svg> Добавить ссылку</button>
        </div>
        <div class="packages-table-wrapper">
            <table class="packages-table" id="linksTable">
                <thead> <th>Иконка</th><th>Название</th><th>URL</th><th>Действия</th> </thead>
                <tbody id="linksTableBody">
    `;
    links.forEach((link, index) => {
        const iconId = link.icon || 'link';
        html += `
            <tr data-index="${index}">
                <td style="min-width: 150px;">
                    <div style="display:flex; align-items:center; gap:8px;">
                        <svg class="icon" width="24" height="24"><use href="/entware-manager/icons.svg?v=2#icon-${iconId}"/></svg>
                        ${renderIconSelect(iconId)}
                    </div>
                 </td>
                 <td><input type="text" class="link-name settings-input" value="${escapeHtml(link.name)}"></td>
                 <td><input type="url" class="link-url settings-input" value="${escapeHtml(link.url)}"></td>
                 <td>
                    <button class="packages-delete-btn" style="background:#27ae60;" onclick="saveLink(this)"><svg class="icon" width="14" height="14"><use href="/entware-manager/icons.svg?v=2#icon-disk"/></svg></button>
                    <button class="packages-delete-btn" style="background:#e53e3e;" onclick="deleteLink(this)"><svg class="icon" width="14" height="14"><use href="/entware-manager/icons.svg?v=2#icon-default"/></svg></button>
                 </td>
              </tr>
        `;
    });
    html += `
                </tbody>
            </table>
        </div>
        <div style="margin-top: 15px;">
            <button id="saveAllLinksBtn" class="packages-delete-btn" style="background:#4a5568;"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-disk"/></svg> Сохранить все на сервер</button>
            <button id="resetDefaultLinksBtn" class="packages-delete-btn" style="background:#f59e0b;"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-refresh"/></svg> Сбросить по умолчанию</button>
        </div>
        <h3 style="margin-top: 30px;"><svg class="icon" width="20" height="20"><use href="/entware-manager/icons.svg?v=2#icon-lock"/></svg> Защита панели</h3>
        <p>Пароль используется для входа в панель и для доступа к изменению и удалению файлов через встроенный менеджер (tmpfs). Если пароль задан — при открытии панели будет показан экран входа.</p>
        <div id="filemgr-auth-settings">
            <div class="loading-spinner"></div>
        </div>
    `;
    html += `
        <h3 style="margin-top: 30px;"><svg class="icon" width="20" height="20"><use href="/entware-manager/icons.svg?v=2#icon-disk"/></svg> Бэкап и восстановление</h3>
        <p>Скачайте бэкап настроек перед сбросом роутера или для переноса на новое устройство.</p>
        <p style="font-size: 0.85rem; color: var(--text-muted);">Сохраняется: ссылки, настройки монитора, сети, watchdog и лога.</p>
        <div style="display: flex; gap: 12px; flex-wrap: wrap; align-items: center; margin-top: 10px;">
            <a href="/entware-cgi/backup.cgi" class="packages-delete-btn" style="background:#4a5568;" download="entware-manager-backup.tar.gz">
                <svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-download"/></svg> Скачать бэкап
            </a>
            <label class="packages-delete-btn" style="background:#4a5568; cursor: pointer;">
                <svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-refresh"/></svg> Восстановить
                <input type="file" id="restoreBackupFile" accept=".tar.gz" style="display: none;" onchange="restoreBackup(this)">
            </label>
            <button class="packages-delete-btn" style="background:#e67e22;" onclick="prepareOffline()">
                <svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-download"/></svg> Подготовить офлайн-пакет
            </button>
            <span id="backupStatus"></span>
        </div>

        <h3 style="margin-top: 30px;"><svg class="icon" width="20" height="20"><use href="/entware-manager/icons.svg?v=2#icon-refresh"/></svg> Обновление</h3>
        <p>Проверьте и установите новую версию Entware Manager.</p>
        <div id="update-section" style="margin-top: 10px;">
            <div style="display: flex; gap: 12px; flex-wrap: wrap; align-items: center;">
                <span><strong>Текущая версия:</strong> <span id="update-current">загрузка...</span></span>
                <button id="update-check-btn" class="packages-delete-btn" style="background:#4a5568;" onclick="checkUpdate()">
                    <svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-search"/></svg> Проверить
                </button>
                <button id="update-run-btn" class="packages-delete-btn" style="background:#2ecc71; display:none;" onclick="runUpdate()">
                    <svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-refresh"/></svg> Обновить до <span id="update-version"></span>
                </button>
                <button id="update-reinstall-btn" class="packages-delete-btn" style="background:#e67e22;" onclick="reinstallUpdate()">
                    <svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-update"/></svg> Переустановить
                </button>
            </div>
            <div id="update-status" style="margin-top: 8px;"></div>
            <pre id="update-log" style="background: var(--pre-bg); padding: 0.5rem; height: 150px; overflow-y: auto; margin-top: 8px; display:none; font-size: 0.85rem;"></pre>
        </div>
    `;
    contentDiv.innerHTML = html;
    fetchTtydStatus();
    loadUpdateInfo();
    if (settingsInterval) clearInterval(settingsInterval);
    settingsInterval = setInterval(fetchTtydStatus, 5000);
    document.getElementById('addLinkBtn').addEventListener('click', addLinkRow);
    document.getElementById('saveAllLinksBtn').addEventListener('click', saveAllLinks);
    document.getElementById('resetDefaultLinksBtn').addEventListener('click', resetDefaultLinks);
    loadAuthConfig();
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
            statusEl.innerHTML = '<span style="color:#2ecc71;">✓ ' + (result.message || 'Восстановлено') + '</span>';
        } else {
            statusEl.innerHTML = '<span style="color:#e53e3e;">Ошибка: ' + (result.message || 'Неизвестная ошибка') + '</span>';
        }
    } catch (err) {
        statusEl.innerHTML = '<span style="color:#e53e3e;">Ошибка: ' + err.message + '</span>';
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

        statusEl.innerHTML = '<span style="color:#2ecc71;">✓ Офлайн-пакет готов: ' + filename + '</span>' +
            '<p style="margin-top:6px;font-size:0.85rem;color:var(--text-secondary);">Файл сохранён в папку Загрузки браузера.</p>' +
            '<p style="margin-top:6px;font-size:0.85rem;color:var(--text-secondary);">Перенесите его на целевой роутер в <b>/opt/tmp/</b> (через SMB или USB) и выполните:</p>' +
            '<pre style="margin-top:8px;background:var(--pre-bg);padding:0.5rem;font-size:0.85rem;white-space:pre-wrap;">tar xzf ' + filename + '\ncd ' + filename.replace('.tar.gz', '') + '\nsh install-offline.sh</pre>';
    } catch (err) {
        statusEl.innerHTML = '<span style="color:#e53e3e;">Ошибка: ' + err.message + '</span>';
    }
};

async function loadUpdateInfo() {
    try {
        const data = await apiGet('/update_check.cgi');
        document.getElementById('update-current').textContent = data.current;
        if (data.has_update) {
            document.getElementById('update-version').textContent = data.latest;
            document.getElementById('update-run-btn').style.display = '';
            document.getElementById('update-status').innerHTML = '<span style="color:#2ecc71;">Доступна версия ' + data.latest + '</span>';
        } else if (data.error) {
            document.getElementById('update-status').innerHTML = '<span style="color:#e53e3e;">' + data.error + '</span>';
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
            statusEl.innerHTML = '<span style="color:#2ecc71;">Доступна версия ' + data.latest + '</span>';
        } else if (data.error) {
            statusEl.innerHTML = '<span style="color:#e53e3e;">' + data.error + '</span>';
        } else {
            statusEl.innerHTML = '<span style="color:var(--text-muted);">Установлена последняя версия</span>';
        }
    } catch (err) {
        statusEl.innerHTML = '<span style="color:#e53e3e;">Ошибка: ' + err.message + '</span>';
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
            statusEl.innerHTML = '<span style="color:#e53e3e;">' + data.message + '</span>';
            btn.disabled = false;
            return;
        }
        statusEl.innerHTML = '<span style="color:#f59e0b;">Обновление... <span id="update-progress"></span> <span style="color:var(--text-muted); font-size:0.85em;">(обычно до 1 минуты при хорошей сети; не закрывайте страницу)</span></span>';

        // Poll status every 2 seconds
        pollUpdateStatus();
    } catch (err) {
        statusEl.innerHTML = '<span style="color:#e53e3e;">Ошибка: ' + err.message + '</span>';
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
            statusEl.innerHTML = '<span style="color:#e53e3e;">' + data.message + '</span>';
            btn.disabled = false;
            return;
        }
        statusEl.innerHTML = '<span style="color:#f59e0b;">Переустановка... <span id="update-progress"></span> <span style="color:var(--text-muted); font-size:0.85em;">(обычно до 1 минуты при хорошей сети; не закрывайте страницу)</span></span>';

        pollUpdateStatus();
    } catch (err) {
        statusEl.innerHTML = '<span style="color:#e53e3e;">Ошибка: ' + err.message + '</span>';
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
    let currentPassword = '';

    if (enabled && password && password !== confirm) {
        statusEl.innerHTML = '<span style="color:#e53e3e;">Пароли не совпадают</span>';
        return;
    }
    if (enabled && password && password.length < 4) {
        statusEl.innerHTML = '<span style="color:#e53e3e;">Пароль должен быть минимум 4 символа</span>';
        return;
    }

    try {
        const formData = new URLSearchParams();
        formData.append('enabled', enabled ? 'true' : 'false');
        formData.append('password', password);

        // Если авторизация уже включена (или был задан пароль) — запросить текущий.
        if (window.AUTH_CURRENTLY_ENABLED) {
            currentPassword = prompt('Введите текущий пароль:');
            if (currentPassword === null) {
                statusEl.innerHTML = '<span style="color:#e53e3e;">Отменено</span>';
                return;
            }
            formData.append('current_password', currentPassword);
        }

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
        statusEl.innerHTML = '<span style="color:#e53e3e;">Ошибка: ' + err.message + '</span>';
    }
};

function addLinkRow() {
    const tbody = document.getElementById('linksTableBody');
    const newRow = document.createElement('tr');
    newRow.innerHTML = `
          <td>
            <div style="display:flex; align-items:center; gap:8px;">
                <svg class="icon" width="24" height="24"><use href="/entware-manager/icons.svg?v=2#icon-link"/></svg>
                ${renderIconSelect('link')}
            </div>
          </td>
          <td><input type="text" class="link-name settings-input" value="Новая ссылка"></td>
          <td><input type="url" class="link-url settings-input" value="http://"></td>
          <td>
            <button class="packages-delete-btn" style="background:#27ae60;" onclick="saveLink(this)"><svg class="icon" width="14" height="14"><use href="/entware-manager/icons.svg?v=2#icon-disk"/></svg></button>
            <button class="packages-delete-btn" style="background:#e53e3e;" onclick="deleteLink(this)"><svg class="icon" width="14" height="14"><use href="/entware-manager/icons.svg?v=2#icon-default"/></svg></button>
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

        // Если текущая активная вкладка – "Установленные", перезагружаем её (оригинальная логика)
        const activeTab = document.querySelector('.menu li.active')?.dataset.tab;
        if (activeTab === 'packages') {
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
                <svg class="icon" width="28" height="28"><use href="/entware-manager/icons.svg?v=2#icon-services"/></svg>
            </span>
            Системные службы и планировщик
        </h2>
        <div id="service-monitor-panel" style="background: var(--command-block-bg); padding: 1rem; border-radius: 12px; margin-bottom: 1rem;">
            <div style="display: flex; gap: 1rem; flex-wrap: wrap; align-items: center;">
                <span><strong>Мониторинг:</strong> <span id="service-watchdog-status" class="stat-value-normal">загрузка...</span></span>
                <button id="service-watchdog-start" class="packages-delete-btn"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-play"/></svg> Запустить</button>
                <button id="service-watchdog-stop" class="packages-delete-btn"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-stop"/></svg> Остановить</button>
                <button id="service-watchdog-restart" class="packages-delete-btn"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-refresh"/></svg> Перезапустить</button>
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
        <h3><svg class="icon" width="20" height="20"><use href="/entware-manager/icons.svg?v=2#icon-services"/></svg> Службы (init.d)</h3>
        <div id="services-list" class="packages-table-wrapper"><div class="loading-spinner"></div></div>
        <h3><svg class="icon" width="20" height="20"><use href="/entware-manager/icons.svg?v=2#icon-list"/></svg> Системный crontab (crontab -l)</h3>
        <div class="cron-editor">
            <textarea id="cron-system" rows="6" style="width:100%; font-family:monospace; padding:8px; border-radius:8px; border:1px solid var(--border-color); background:var(--input-bg); color:var(--text-primary);"></textarea>
            <div style="margin-top:10px; display:flex; gap:10px; flex-wrap:wrap;">
                <button id="save-cron-system" class="packages-delete-btn" style="background:#4a5568;"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-disk"/></svg> Сохранить системный crontab</button>
                <span id="cron-system-message" style="margin-left:10px; align-self:center;"></span>
            </div>
        </div>
        <h3><svg class="icon" width="20" height="20"><use href="/entware-manager/icons.svg?v=2#icon-list"/></svg> Entware crontab (/opt/etc/crontab)</h3>
        <div class="cron-editor">
            <textarea id="cron-opt" rows="6" style="width:100%; font-family:monospace; padding:8px; border-radius:8px; border:1px solid var(--border-color); background:var(--input-bg); color:var(--text-primary);"></textarea>
            <div style="margin-top:10px; display:flex; gap:10px; flex-wrap:wrap;">
                <button id="save-cron-opt" class="packages-delete-btn" style="background:#4a5568;"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-disk"/></svg> Сохранить Entware crontab</button>
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
        container.innerHTML = `<p class="error">Ошибка загрузки служб: ${err.message}</p>`;
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
                pidHtml = `<span class="pid-link" onclick="showProcessList('${escapeHtml(s.name)}')">${displayPid} <span class="pid-badge">+${extra}</span></span>`;
            } else {
                pidHtml = `<span class="pid-link" onclick="showProcessList('${escapeHtml(s.name)}')">${displayPid}</span>`;
            }
        }
        html += `  <tr>
              <td>${escapeHtml(s.name)}</td>
              <td><span class="stat-value-${s.status === 'running' ? 'normal' : 'critical'}">${s.status}</span></td>
              <td>${pidHtml}</td>
              <td style="text-align: center;">
                <svg class="icon" width="20" height="20" style="display: inline-block; vertical-align: middle;">
                    <use href="/entware-manager/icons.svg?v=2#icon-${s.enabled ? 'check' : 'cross'}"/>
                </svg>
              </td>
              <td>
                <button class="packages-delete-btn" style="background:#4a5568;" onclick="serviceAction('${s.name}', 'start')" ${s.status === 'running' ? 'disabled' : ''}>Запустить</button>
                <button class="packages-delete-btn" style="background:#e53e3e;" onclick="serviceAction('${s.name}', 'stop')" ${s.status !== 'running' ? 'disabled' : ''}>Остановить</button>
                <button class="packages-delete-btn" style="background:#f59e0b;" onclick="serviceAction('${s.name}', 'restart')" ${s.status !== 'running' ? 'disabled' : ''}>Перезапустить</button>
                <button class="packages-delete-btn" style="background:${s.enabled ? '#e53e3e' : '#27ae60'};" onclick="serviceAction('${s.name}', '${s.enabled ? 'disable' : 'enable'}')">Авто</button>
              </td>
          </tr>`;
    });
    html += '</tbody></table>';
    document.getElementById('services-list').innerHTML = html;
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
                    <button class="packages-delete-btn process-kill-btn" onclick="killProcess('${escapeHtml(pid)}', '${escapeHtml(serviceName)}')">
                        <svg class="icon" width="14" height="14"><use href="/entware-manager/icons.svg?v=2#icon-stop"/></svg> Убить
                    </button>
                </div>`;
            });
            html += '</div>';
            Modal.show(html, false, 'Процессы: ' + escapeHtml(serviceName));
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

function parseSize(str) {
    if (!str) return 0;
    str = str.trim();
    const units = { 'B': 1, 'K': 1024, 'M': 1048576, 'G': 1073741824 };
    const match = str.match(/^([\d.,]+)\s*([KMGT]?B?)$/i);
    if (!match) return 0;
    let val = parseFloat(match[1].replace(',', '.'));
    let unit = match[2].toUpperCase();
    if (unit === 'B') unit = 'B';
    if (!units[unit]) return val;
    return val * units[unit];
}

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
        let aVal = a.cells[colIndex]?.innerText.trim() || '';
        let bVal = b.cells[colIndex]?.innerText.trim() || '';

        if (dataType === 'size') {
            aVal = parseSize(aVal);
            bVal = parseSize(bVal);
        } else if (dataType === 'percent' || dataType === 'number') {
            aVal = parseFloat(aVal) || 0;
            bVal = parseFloat(bVal) || 0;
        } else if (dataType === 'speed') {
            aVal = parseSpeed(aVal);
            bVal = parseSpeed(bVal);
        } else if (dataType === 'ip') {
            const aIP = parseIP(aVal);
            const bIP = parseIP(bVal);
            if (sortOrder === 'asc') {
                for (let i = 0; i < 4; i++) {
                    if (aIP[i] !== bIP[i]) return (aIP[i] || 0) - (bIP[i] || 0);
                }
                return 0;
            } else {
                for (let i = 0; i < 4; i++) {
                    if (aIP[i] !== bIP[i]) return (bIP[i] || 0) - (aIP[i] || 0);
                }
                return 0;
            }
        }

        if (sortOrder === 'asc') {
            return aVal > bVal ? 1 : aVal < bVal ? -1 : 0;
        } else {
            return aVal < bVal ? 1 : aVal > bVal ? -1 : 0;
        }
    });

    rows.forEach(row => tbody.appendChild(row));
    updateSortIndicators(table, colIndex, sortOrder);
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

function enableTableSorting() {
    const statTables = document.querySelectorAll('.stat-card.tmpfs table, .stat-card.storage table');
    const fileTables = document.querySelectorAll('.file-manager .file-table');
    const allTables = [...statTables, ...fileTables];

    allTables.forEach(table => {
        if (table.dataset.sortable) return;
        table.dataset.sortable = 'true';
        const headers = table.querySelectorAll('thead th');
        headers.forEach((th, idx) => {
            th.style.cursor = 'pointer';
            th.addEventListener('click', () => {
                let dataType = 'string';
                const columnText = th.innerText.toLowerCase();
                if (columnText.includes('размер')) {
                    dataType = 'size';
                } else if (columnText.includes('загрузка')) {
                    dataType = 'percent';
                }
                sortTable(table, idx, dataType);
            });
        });
    });
}

document.addEventListener('DOMContentLoaded', init);

function loadNetworkTab() {
    if (typeof initNetworkTab === 'function') {
        initNetworkTab();
        return;
    }
    const script = document.createElement('script');
    script.src = '/entware-manager/network.js?v=2';
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
    script.src = '/entware-manager/monitor.js?v=2';
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
        
        try {
            const result = await apiPostJSON('/service_watchdog/config.cgi', { mode: mode, watch_list: watchList, auto_restart: auto_restart, exclude_list: exclude_list });
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
            const res = await apiFetch('/service_watchdog/action.cgi?action=' + action);
            const data = await res.json();
            
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