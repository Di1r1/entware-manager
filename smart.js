// Entware Manager - SMART мониторинг дисков
// Версия: 0.1 (UI-таб)

let smartInterval = null;

async function loadSmartTab() {
    const ver = window.APP_VERSION || 'dev';
    console.log(`[v${ver}] Загрузка вкладки SMART`);
    
    if (smartInterval) clearInterval(smartInterval);
    smartInterval = null;

    contentDiv.innerHTML = '<p>Загрузка SMART...</p>';
    
    try {
        const response = await apiFetch('/smart.cgi?action=list');
        const data = await response.json();
        
        if (data.error) {
            contentDiv.innerHTML = `<p class="error">Ошибка: ${escapeHtml(data.error)}</p>`;
            return;
        }
        
        renderSmartTable(data.disks || []);
        Menu.setActiveTab('smart');
    } catch (err) {
        contentDiv.innerHTML = `<p class="error">Ошибка загрузки: ${err.message}</p>`;
    }
}

function renderSmartTable(disks) {
    let html = `
        <h2 style="display: flex; align-items: center; gap: 8px;">
            <span class="stat-icon" style="width: 28px; height: 28px;">
                <svg class="icon" width="28" height="28">
                    <use href="/entware-manager/icons.svg?v=2#icon-hdd"/>
                </svg>
            </span>
            SMART дисков (${disks.length})
        </h2>
    `;
    
    if (disks.length === 0) {
        html += `
            <div class="packages-no-data">
                Диски не найдены или smartmontools не установлен.
                <br><small>Установите: opkg install smartmontools</small>
            </div>
        `;
        contentDiv.innerHTML = html;
        return;
    }

    html += `
        <div style="display: flex; gap: 8px; align-items: center; margin-bottom: 20px;">
            <div class="search-container" style="display: flex; gap: 8px; align-items: center; flex: 1; background: var(--input-bg); border: 2px solid var(--input-border); border-radius: 40px; padding: 0 12px; transition: border-color 0.3s ease, box-shadow 0.3s ease;">
                <svg class="icon" width="18" height="18" style="color: var(--text-muted);"><use href="/entware-manager/icons.svg?v=2#icon-search"/></svg>
                <input type="text" id="searchSmart" placeholder="Поиск по модели/серийнику..." style="flex: 1; background: transparent; border: none; outline: none; padding: 14px 0; font-size: 16px; color: var(--text-primary);">
            </div>
            <button id="refreshSmart" class="packages-delete-btn" style="background: #4a5568;"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-refresh"/></svg> Обновить</button>
        </div>
        <div class="packages-table-wrapper">
            <table class="packages-table" id="smartTable">
                <thead>
                    <tr>
                        <th>Устройство</th>
                        <th>Модель</th>
                        <th>Серийный №</th>
                        <th>Размер</th>
                        <th>Тип</th>
                        <th>Health</th>
                        <th>Temp</th>
                        <th>Power-On</th>
                        <th>Действия</th>
                    </tr>
                </thead>
                <tbody id="smartTableBody">
    `;

    disks.forEach(disk => {
        const healthClass = disk.health === 'PASSED' ? 'status-running' : 'status-stopped';
        const healthIcon = disk.health === 'PASSED' ? 'icon-check' : 'icon-cross';
        const tempClass = disk.temperature > 55 ? 'text-red' : (disk.temperature > 45 ? 'text-yellow' : '');
        const tempText = disk.temperature ? `${disk.temperature}°C` : '—';
        const powerOn = disk.power_on_hours ? `${disk.power_on_hours} ч` : '—';

        html += `
            <tr data-device="${escapeHtml(disk.device)}">
                <td>${escapeHtml(disk.device)}</td>
                <td>${escapeHtml(disk.model || '—')}</td>
                <td>${escapeHtml(disk.serial || '—')}</td>
                <td>${escapeHtml(disk.size || '—')}</td>
                <td>${escapeHtml(disk.type || '—')}</td>
                <td><span class="status-badge ${healthClass}"><svg class="icon" width="12" height="12"><use href="/entware-manager/icons.svg?v=2#${healthIcon}"/></svg> ${escapeHtml(disk.health)}</span></td>
                <td class="${tempClass}">${tempText}</td>
                <td>${powerOn}</td>
                <td>
                    <div style="display: flex; gap: 4px;">
                        <button class="packages-delete-btn smart-attr-btn" data-device="${escapeHtml(disk.device)}" style="padding: 4px 8px; font-size: 12px; background: #3182ce;">Атрибуты</button>
                        <button class="packages-delete-btn smart-health-btn" data-device="${escapeHtml(disk.device)}" style="padding: 4px 8px; font-size: 12px; background: #2c7a7b;">Health</button>
                        <button class="packages-delete-btn smart-test-btn" data-device="${escapeHtml(disk.device)}" style="padding: 4px 8px; font-size: 12px; background: #c05621;">Тест</button>
                    </div>
                </td>
            </tr>
        `;
    });

    html += `
                </tbody>
            </table>
        </div>
    `;

    contentDiv.innerHTML = html;

    // Search
    initTableSearch('searchSmart', 'smartTable', 0);

    // Buttons
    document.getElementById('refreshSmart').addEventListener('click', loadSmartTab);

    document.querySelectorAll('.smart-attr-btn').forEach(btn => {
        btn.addEventListener('click', (e) => {
            e.stopPropagation();
            showSmartAttributes(btn.dataset.device);
        });
    });

    document.querySelectorAll('.smart-health-btn').forEach(btn => {
        btn.addEventListener('click', (e) => {
            e.stopPropagation();
            showSmartHealth(btn.dataset.device);
        });
    });

    document.querySelectorAll('.smart-test-btn').forEach(btn => {
        btn.addEventListener('click', (e) => {
            e.stopPropagation();
            showSmartTest(btn.dataset.device);
        });
    });
}

async function showSmartAttributes(device) {
    Modal.loading('Загрузка атрибутов...');
    try {
        const response = await apiFetch(`/smart.cgi?action=attributes&device=${encodeURIComponent(device)}`);
        const data = await response.json();
        
        if (data.error) {
            Modal.error(data.error);
            return;
        }

        let html = `
            <h3 style="margin-bottom: 12px;">Атрибуты SMART: ${escapeHtml(device)}</h3>
            <div style="overflow-x: auto;">
                <table class="packages-table" style="width: 100%;">
                    <thead>
                        <tr>
                            <th>ID</th>
                            <th>Имя атрибута</th>
                            <th>Value</th>
                            <th>Worst</th>
                            <th>Thresh</th>
                            <th>Raw</th>
                            <th>Статус</th>
                        </tr>
                    </thead>
                    <tbody>
        `;

        (data.attributes || []).forEach(attr => {
            const isCritical = attr.value <= attr.threshold && attr.threshold > 0;
            const statusClass = isCritical ? 'status-stopped' : 'status-running';
            const statusIcon = isCritical ? 'icon-cross' : 'icon-check';
            const statusText = isCritical ? 'КРИТИЧНО' : 'OK';

            html += `
                <tr class="${isCritical ? 'text-red' : ''}">
                    <td>${attr.id}</td>
                    <td>${escapeHtml(attr.name)}</td>
                    <td>${attr.value}</td>
                    <td>${attr.worst}</td>
                    <td>${attr.threshold}</td>
                    <td>${attr.raw}</td>
                    <td><span class="status-badge ${statusClass}"><svg class="icon" width="12" height="12"><use href="/entware-manager/icons.svg?v=2#${statusIcon}"/></svg> ${statusText}</span></td>
                </tr>
            `;
        });

        html += `
                    </tbody>
                </table>
            </div>
        `;

        Modal.show(html, false, `SMART атрибуты — ${escapeHtml(device)}`);
    } catch (err) {
        Modal.error('Ошибка: ' + err.message);
    }
}

async function showSmartHealth(device) {
    Modal.loading('Загрузка health...');
    try {
        const response = await apiFetch(`/smart.cgi?action=health&device=${encodeURIComponent(device)}`);
        const data = await response.json();

        if (data.error) {
            Modal.error(data.error);
            return;
        }

        let html = `
            <h3 style="margin-bottom: 12px;">Health: ${escapeHtml(device)}</h3>
            <div style="font-family: monospace; white-space: pre-wrap; background: var(--input-bg); padding: 12px; border-radius: 6px;">
${escapeHtml(data.message || data.health || 'Нет данных')}
            </div>
        `;

        Modal.show(html, false, `SMART Health — ${escapeHtml(device)}`);
    } catch (err) {
        Modal.error('Ошибка: ' + err.message);
    }
}

async function showSmartTest(device) {
    let html = `
        <h3 style="margin-bottom: 12px;">Самодиагностика: ${escapeHtml(device)}</h3>
        <p>Выберите тип теста:</p>
        <div style="display: flex; flex-direction: column; gap: 8px; margin-bottom: 16px;">
            <button class="packages-delete-btn" onclick="runSmartTest('${escapeHtml(device)}', 'short')" style="background: #3182ce;">
                <svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-play"/></svg> Короткий тест (~2 мин)
            </button>
            <button class="packages-delete-btn" onclick="runSmartTest('${escapeHtml(device)}', 'long')" style="background: #2c7a7b;">
                <svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-play"/></svg> Полный тест (~60-120 мин)
            </button>
            <button class="packages-delete-btn" onclick="runSmartTest('${escapeHtml(device)}', 'conveyance')" style="background: #c05621;">
                <svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-play"/></svg> Conveyance тест (~5 мин)
            </button>
        </div>
        <div id="smartTestStatus" style="font-family: monospace; background: var(--input-bg); padding: 12px; border-radius: 6px; min-height: 60px;">
            Нажмите кнопку для запуска теста...
        </div>
    `;

    Modal.show(html, false, `SMART тест — ${escapeHtml(device)}`);
}

window.runSmartTest = async function(device, type) {
    const statusDiv = document.getElementById('smartTestStatus');
    statusDiv.innerHTML = `Запуск ${type} теста...`;
    
    try {
        const response = await apiFetch('/smart.cgi', {
            method: 'POST',
            headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
            body: `action=selftest&device=${encodeURIComponent(device)}&type=${type}`
        });
        const data = await response.json();
        
        if (data.error) {
            statusDiv.innerHTML = `Ошибка: ${escapeHtml(data.error)}`;
            return;
        }
        
        statusDiv.innerHTML = `Тест запущен: ${escapeHtml(data.message || 'OK')}`;
        
        // Поллинг статуса
        let attempts = 0;
        const maxAttempts = 240; // 20 минут для long теста
        const pollInterval = setInterval(async () => {
            try {
                const r = await apiFetch(`/smart.cgi?action=selftest&device=${encodeURIComponent(device)}`);
                const d = await r.json();
                
                if (d.status) {
                    statusDiv.innerHTML = `Статус: ${escapeHtml(d.status)}<br>Прогресс: ${d.progress || '?'}%`;
                    if (d.status.includes('Completed') || d.status.includes('Aborted') || d.status.includes('Interrupted')) {
                        clearInterval(pollInterval);
                        statusDiv.innerHTML += '<br><span class="status-badge status-running">Тест завершён</span>';
                    }
                }
            } catch (e) {
                // ignore
            }
            
            if (++attempts >= maxAttempts) clearInterval(pollInterval);
        }, 5000);
        
    } catch (err) {
        statusDiv.innerHTML = `Ошибка: ${err.message}`;
    }
};