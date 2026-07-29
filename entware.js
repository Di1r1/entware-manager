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
    initAutoTheme();

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
}

function initTheme() {
    const themeToggle = document.getElementById('themeToggle');
    const savedTheme = localStorage.getItem('entware_theme');
    if (savedTheme === 'night') {
        document.body.classList.add('night');
        themeToggle.querySelector('use')?.setAttribute('href', '/entware-manager/icons.svg?v=2#icon-moon');
    } else {
        document.body.classList.remove('night');
        themeToggle.querySelector('use')?.setAttribute('href', '/entware-manager/icons.svg?v=2#icon-sun');
    }
    themeToggle.addEventListener('click', () => {
        document.body.classList.toggle('night');
        const isNight = document.body.classList.contains('night');
        localStorage.setItem('entware_theme', isNight ? 'night' : 'day');
        const useEl = themeToggle.querySelector('use');
        if (useEl) useEl.setAttribute('href', isNight ? '/entware-manager/icons.svg?v=2#icon-moon' : '/entware-manager/icons.svg?v=2#icon-sun');
    });
}

function initAutoTheme() {
    const hour = new Date().getHours();
    const isNightTime = hour >= 20 || hour < 6;
    const savedTheme = localStorage.getItem('entware_theme');
    if (!savedTheme) {
        if (isNightTime) {
            document.body.classList.add('night');
            document.getElementById('themeToggle')?.querySelector('use')?.setAttribute('href', '/entware-manager/icons.svg?v=2#icon-moon');
        } else {
            document.body.classList.remove('night');
            document.getElementById('themeToggle')?.querySelector('use')?.setAttribute('href', '/entware-manager/icons.svg?v=2#icon-sun');
        }
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

async function runUpdate() {
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
    document.getElementById('runUpdateBtn').addEventListener('click', runUpdate);
    document.getElementById('upgradeAllBtn').addEventListener('click', upgradeAll);
    fetchUpgradable();
}

function renderProcessesTab() {
    const html = `
        <h2 style="display: flex; align-items: center; gap: 8px;">
            <span class="stat-icon" style="width: 28px; height: 28px;">
                <svg class="icon" width="28" height="28"><use href="/entware-manager/icons.svg?v=2#icon-process"/></svg>
            </span>
            Процессы (htop)
        </h2>
        <div style="display: flex; gap: 10px; margin-bottom: 15px;">
            <a href="${BASE_URL}:8089" target="_blank" class="packages-delete-btn" style="background:#4a5568;"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-link"/></svg> Открыть в новой вкладке</a>
        </div>
        <p style="color: var(--text-secondary); margin-bottom: 15px; font-size: 0.9rem;">
            Если страница не открывается — запустите ttyd в <b>Настройки → Терминал</b>
        </p>
        <iframe src="${BASE_URL}:8089" width="100%" height="600" style="border: none; border-radius: 8px;"></iframe>
    `;
    contentDiv.innerHTML = html;
}

function renderTerminalTab() {
    const html = `
        <h2 style="display: flex; align-items: center; gap: 8px;">
            <span class="stat-icon" style="width: 28px; height: 28px;">
                <svg class="icon" width="28" height="28"><use href="/entware-manager/icons.svg?v=2#icon-terminal"/></svg>
            </span>
            Терминал (bash)
        </h2>
        <div style="display: flex; gap: 10px; margin-bottom: 15px;">
            <a href="${BASE_URL}:9089" target="_blank" class="packages-delete-btn" style="background:#4a5568;"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-link"/></svg> Открыть в новой вкладке</a>
        </div>
        <p style="color: var(--text-secondary); margin-bottom: 15px; font-size: 0.9rem;">
            Если страница не открывается — запустите ttyd в <b>Настройки → Терминал</b>
        </p>
        <iframe src="${BASE_URL}:9089" width="100%" height="600" style="border: none; border-radius: 8px;"></iframe>
    `;
    contentDiv.innerHTML = html;
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

    let html = '<h3>Текущее состояние ttyd</h3>';
    html += '<table class="stat-table">';
    html += `  <tr><td>htop (порт 8089):</td><td><span class="${htop.state === 'running' ? 'stat-value-normal' : 'stat-value-critical'}">${htop.state}</span> ${htop.pid ? '(PID ' + htop.pid + ')' : ''}</td></tr>`;
    html += `  <tr><td>Терминал (порт 9089):</td><td><span class="${term.state === 'running' ? 'stat-value-normal' : 'stat-value-critical'}">${term.state}</span> ${term.pid ? '(PID ' + term.pid + ')' : ''}</td></tr>`;
    html += '</table>';

    html += '<div style="display: flex; gap: 20px; margin-top: 20px;">';
    html += '<div style="flex:1;"><h4>htop (порт 8089)</h4>';
    html += `<button class="packages-delete-btn" style="background:#4a5568;" onclick="controlTtyd('start', 8089, '')" ${htop.state === 'running' ? 'disabled' : ''}>Запустить</button> `;
    html += `<button class="packages-delete-btn" style="background:#e53e3e;" onclick="controlTtyd('stop', 8089, '')" ${htop.state !== 'running' ? 'disabled' : ''}>Остановить</button> `;
    html += `<button class="packages-delete-btn" style="background:#f59e0b;" onclick="controlTtyd('restart', 8089, '')" ${htop.state !== 'running' ? 'disabled' : ''}>Перезапустить</button>`;
    html += '</div><div style="flex:1;"><h4>Терминал (порт 9089)</h4>';
    html += '<div style="margin-bottom: 10px;"><input type="password" id="termPass" placeholder="Пароль (admin)" style="width: 100%; padding: 8px; border: 1px solid #ddd; border-radius: 4px;"></div>';
    html += `<button class="packages-delete-btn" style="background:#4a5568;" onclick="controlTtyd('start', 9089, document.getElementById('termPass').value)" ${term.state === 'running' ? 'disabled' : ''}>Запустить</button> `;
    html += `<button class="packages-delete-btn" style="background:#e53e3e;" onclick="controlTtyd('stop', 9089, '')" ${term.state !== 'running' ? 'disabled' : ''}>Остановить</button> `;
    html += `<button class="packages-delete-btn" style="background:#f59e0b;" onclick="controlTtyd('restart', 9089, document.getElementById('termPass').value)" ${term.state !== 'running' ? 'disabled' : ''}>Перезапустить</button>`;
    html += '</div></div>';
    statusDiv.innerHTML = html;
}

window.controlTtyd = async function(action, port, pass) {
    const formData = new URLSearchParams();
    formData.append('action', action);
    formData.append('port', port);
    if (pass) formData.append('pass', pass);
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
        { name: 'Entware Manager', url: h + ':8087/entware-manager/', icon: 'package' },
        { name: 'AdGuard Home', url: h + ':3000', icon: 'shield' },
        { name: 'Transmission', url: h + ':9091', icon: 'download' },
        { name: 'Netdata', url: h + ':19999', icon: 'chart' },
        { name: 'htop (ttyd)', url: h + ':8089', icon: 'process' },
        { name: 'Терминал (ttyd)', url: h + ':9089', icon: 'terminal' }
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
        if (result.status !== 'ok') Toast.show('Ошибка сохранения ссылок: ' + result.message, true);
    } catch (err) {
        Toast.show('Ошибка соединения с сервером', true);
    }
}

function renderIconSelect(selectedId) {
    let html = `<select class="link-icon-select" style="width: 100%; padding: 6px; border-radius: 6px;">`;
    iconList.forEach(icon => {
        const selected = (icon.id === selectedId) ? 'selected' : '';
        html += `<option value="${icon.id}" ${selected}>${icon.name}</option>`;
    });
    html += `</select>`;
    return html;
}

async function loadNetworkStatus() {
    const table = document.getElementById('networkTable');
    if (!table) return;
    
    try {
        const data = await apiGet('/network_stats.cgi');
        
        let html = '<div class="stat-table">';
        
        // Таблица 1: Интерфейсы с IP
        html += '<div><table>';
        html += '<tr><th colspan="2">Интерфейсы</th></tr>';
        if (data.interfaces && data.interfaces.length > 0) {
            data.interfaces.forEach(iface => {
                if (iface.ip && iface.ip !== '--') {
                    html += `<tr><td>${escapeHtml(iface.iface)}:</td><td><code>${escapeHtml(iface.ip)}</code></td></tr>`;
                }
            });
        }
        html += '</table></div>';
        
        // Таблица 2: Физические порты
        html += '<div><table>';
        html += '<tr><th colspan="2">Физические порты</th></tr>';
        if (data.ports && data.ports.length > 0) {
            data.ports.forEach(port => {
                const carrier = port.carrier === '✓';
                const speed = port.speed && port.speed !== '—' ? port.speed : '';
                const label = carrier ? (speed ? `${port.iface} — ${speed}` : `${port.iface} — ✓`) : `${port.iface} —`;
                html += `<tr><td>${escapeHtml(port.iface)}</td><td class="${carrier ? 'stat-value-normal' : 'stat-value-critical'}">${escapeHtml(label)}</td></tr>`;
            });
        }
        html += '</table></div>';
        
        // Таблица 3: Сети
        html += '<div><table>';
        html += '<tr><th colspan="2">Сети</th></tr>';
        if (data.networks && data.networks.length > 0) {
            data.networks.forEach(net => {
                const members = net.members ? escapeHtml(net.members) : '—';
                const memberCount = net.members ? net.members.split(/\s+/).filter(Boolean).length : 0;
                const displayed = memberCount > 0 ? members : '';
                html += `<tr><td>${escapeHtml(net.name)}</td><td>${displayed ? `<code>${displayed}</code>` : `<span class="stat-value-critical">${escapeHtml(net.bridge)}</span>`}</td></tr>`;
            });
        }
        html += '</table></div>';
        
        html += '</div>';
        
        // Таблица 4: WiFi сети (на отдельной строке)
        if (data.wifi_info && data.wifi_info.length > 0 && data.wifi_info[0].name !== '--') {
            html += '<div class="stat-table" style="margin-top: 10px;">';
            html += '<div><table>';
            html += '<tr><th colspan="4">WiFi сети</th></tr>';
            html += '<tr><th>Сеть</th><th>2.4GHz</th><th>5GHz</th><th>TX/RX</th></tr>';
            data.wifi_info.forEach(wifi => {
                html += `<tr><td>${escapeHtml(wifi.name)}</td><td>${escapeHtml(wifi['2g'])}</td><td>${escapeHtml(wifi['5g'])}</td><td>${escapeHtml(wifi.tx)} / ${escapeHtml(wifi.rx)}</td></tr>`;
            });
            html += '</table></div>';
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
    
    document.getElementById('network-refresh')?.addEventListener('click', loadNetworkStatus);
}

async function renderLinksOnStats() {
    const statsContent = document.querySelector('.stats-grid');
    if (!statsContent) return;
    const links = await loadLinks();
    if (links.length === 0) return;
    if (document.querySelector('.links-grid')) return;

    let html = '<h3 style="margin-top: 30px;"><svg class="icon" width="20" height="20"><use href="/entware-manager/icons.svg?v=2#icon-link"/></svg> Полезные ссылки</h3><div class="links-grid">';
    links.forEach(link => {
        const iconId = `icon-${link.icon || 'link'}`;
        html += `
            <a href="${escapeHtml(link.url)}" target="_blank" class="link-card">
                <span class="link-icon"><svg class="icon" width="32" height="32"><use href="/entware-manager/icons.svg?v=2#${iconId}"/></svg></span>
                <span class="link-name">${escapeHtml(link.name)}</span>
            </a>
        `;
    });
    html += '</div>';
    statsContent.insertAdjacentHTML('afterend', html);
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
                 <td><input type="text" class="link-name" value="${escapeHtml(link.name)}" style="width:100%;"></td>
                 <td><input type="url" class="link-url" value="${escapeHtml(link.url)}" style="width:100%;"></td>
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
        <p class="settings-note" style="margin-top: 20px; font-size: 0.9rem;">
            Управление веб-терминалами ttyd. Для терминала требуется пароль.<br>
            После изменения состояния обновите вкладки "Процессы" и "Терминал".
        </p>
        <h3 style="margin-top: 30px;"><svg class="icon" width="20" height="20"><use href="/entware-manager/icons.svg?v=2#icon-lock"/></svg> Защита файлового менеджера</h3>
        <p>Включите пароль для доступа к изменению и удалению файлов через встроенный менеджер (tmpfs).</p>
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
        statusEl.innerHTML = '<span style="color:#f59e0b;">Обновление... <span id="update-progress"></span></span>';

        // Poll status every 2 seconds
        pollUpdateStatus();
    } catch (err) {
        statusEl.innerHTML = '<span style="color:#e53e3e;">Ошибка: ' + err.message + '</span>';
        btn.disabled = false;
    }
}

let updatePollInterval;

function pollUpdateStatus() {
    if (updatePollInterval) clearInterval(updatePollInterval);
    updatePollInterval = setInterval(async () => {
        try {
            const data = await apiGet('/update_status.cgi');
            const logPre = document.getElementById('update-log');
            if (data.lines && data.lines.length) {
                logPre.textContent = data.lines.join('\n');
                logPre.scrollTop = logPre.scrollHeight;
            }

            if (data.status === 'done') {
                clearInterval(updatePollInterval);
                document.getElementById('update-status').innerHTML = '<span style="color:#2ecc71;">✓ Обновление завершено</span>';
                document.getElementById('update-run-btn').style.display = 'none';
                document.getElementById('update-current').textContent = data.lines.length > 0
                    ? (data.lines[data.lines.length-1].replace(/.*v/, '').replace(/ .*/, '') || '?')
                    : '?';
            } else if (data.status === 'error') {
                clearInterval(updatePollInterval);
                document.getElementById('update-status').innerHTML = '<span style="color:#e53e3e;">Ошибка: ' + (data.error || 'Неизвестная ошибка') + '</span>';
                document.getElementById('update-run-btn').disabled = false;
            } else if (data.status === 'running') {
                const progress = data.lines.length > 0 ? data.lines[data.lines.length-1] : '';
                document.getElementById('update-progress').textContent = progress;
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
        let html = `
            <label style="display: flex; align-items: center; gap: 8px; margin: 10px 0;">
                <input type="checkbox" id="filemgrAuthEnabled" ${enabled ? 'checked' : ''} onchange="toggleFilemgrPassFields()">
                Включить защиту паролем
            </label>
            <div id="filemgrPassFields" style="display: ${enabled ? 'block' : 'none'}; margin: 10px 0;">
                <div style="margin-bottom: 8px;">
                    <label>Новый пароль (мин. 4 символа):</label>
                    <input type="password" id="filemgrPass" placeholder="Оставьте пустым чтобы не менять" style="width: 100%; max-width: 300px; padding: 8px; border: 1px solid #ddd; border-radius: 4px;">
                </div>
                <div style="margin-bottom: 8px;">
                    <label>Подтверждение:</label>
                    <input type="password" id="filemgrPassConfirm" placeholder="Повторите пароль" style="width: 100%; max-width: 300px; padding: 8px; border: 1px solid #ddd; border-radius: 4px;">
                </div>
                <button class="packages-delete-btn" style="background:#4a5568;" onclick="saveAuthConfig()">Сохранить</button>
                <span id="filemgrAuthStatus" style="margin-left: 10px;"></span>
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
    if (enabled && password && password.length < 4) {
        statusEl.innerHTML = '<span style="color:#e53e3e;">Пароль должен быть минимум 4 символа</span>';
        return;
    }

    try {
        const formData = new URLSearchParams();
        formData.append('enabled', enabled ? 'true' : 'false');
        formData.append('password', password);
        const data = await apiPost('/auth_config.cgi', formData.toString());
        statusEl.innerHTML = '<span style="color:#2ecc71;">✓ Настройки сохранены</span>';
        document.getElementById('filemgrPass').value = '';
        document.getElementById('filemgrPassConfirm').value = '';
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
          <td><input type="text" class="link-name" value="Новая ссылка" style="width:100%;"></td>
          <td><input type="url" class="link-url" value="http://" style="width:100%;"></td>
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
    await saveLinks(links);
    Toast.show('Ссылка сохранена');
    renderSettingsTab();
};

window.deleteLink = async function(btn) {
    if (!confirm('Удалить ссылку?')) return;
    const row = btn.closest('tr');
    const index = row.dataset.index;
    if (index !== undefined) {
        const links = await loadLinks();
        links.splice(index, 1);
        await saveLinks(links);
        Toast.show('Ссылка удалена');
        renderSettingsTab();
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
    await saveLinks(links);
    Toast.show('Ссылки сохранены на сервер');
    renderSettingsTab();
}

async function resetDefaultLinks() {
    if (confirm('Восстановить ссылки по умолчанию? Текущие будут потеряны.')) {
        await saveLinks(getDefaultLinks());
        Toast.show('Ссылки сброшены к настройкам по умолчанию');
        renderSettingsTab();
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
        } else if (dataType === 'percent') {
            aVal = parseFloat(aVal);
            bVal = parseFloat(bVal);
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
        html += `<li>lighttpd (запущен): <b style="color:${data.base.lighttpd_running ? '#38a169' : '#e53e3e'}">${data.base.lighttpd_running ? 'ДА' : 'НЕТ'}</b></li>`;
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

        // Рекомендации
        if (data.overall_status !== 'ok') {
            html += '<hr style="border-color:#4a5568;"><h4>Рекомендации:</h4>';
            if (!deps.cron_installed) html += '<p style="color:#e53e3e;">• Установите cron: <code>opkg install cron</code></p>';
            if (!deps.jq) html += '<p style="color:#e53e3e;">• Установите jq: <code>opkg install jq</code></p>';
            if (!deps.ip) html += '<p style="color:#e53e3e;">• Установите ip-full: <code>opkg install ip-full</code> (или проверьте наличие ip в системе)</p>';
        }

        html += '</div>';
        Modal.info(html, 'Проверка системы');

    } catch (e) {
        Modal.error('Ошибка проверки: ' + e.message);
    }
}