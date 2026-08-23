// Entware Manager - общие утилиты для JavaScript
// Версия: 1.2 (добавлены apiFetch, apiGet, apiPost, apiPostJSON)
// Дата: 2026-07-16

const API_BASE = '/entware-cgi';
const UI_BASE = '/entware-manager';
const ICONS = UI_BASE + '/icons.svg?v=5';

function escapeHtml(text) {
    return String(text).replace(/[&<>"']/g, function(m) {
        if (m === '&') return '&amp;';
        if (m === '<') return '&lt;';
        if (m === '>') return '&gt;';
        if (m === '"') return '&quot;';
        if (m === "'") return '&#39;';
        return m;
    });
}

function apiFetch(path, options) {
    var url = API_BASE + path;
    if (!options || !options.method || options.method === 'GET') {
        var sep = path.indexOf('?') > -1 ? '&' : '?';
        url += sep + '_=' + Date.now();
    }
    return fetch(url, options).then(function(r) {
        if (r.status === 401 && path.indexOf('/login.cgi') === -1 && path.indexOf('/session.cgi') === -1) {
            // сессия истекла — вернуть к экрану входа
            if (typeof showLogin === 'function') showLogin();
            throw new Error('unauthorized');
        }
        if (!r.ok) throw new Error(r.statusText);
        return r;
    });
}

function apiGet(path) {
    return apiFetch(path).then(function(r) { return r.json(); });
}

function apiPost(path, body) {
    return apiFetch(path, {
        method: 'POST',
        headers: {'Content-Type': 'application/x-www-form-urlencoded'},
        body: body
    }).then(function(r) { return r.json(); });
}

function apiPostJSON(path, data) {
    return apiFetch(path, {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify(data)
    }).then(function(r) { return r.json(); });
}

function initTableSearch(inputId, tableId, cellIndex) {
    var input = document.getElementById(inputId);
    var table = document.getElementById(tableId);
    if (!input || !table) return;
    // защита от повторного бинда: если поиск уже инициализирован на этом
    // input — не вешаем второй обработчик keyup (иначе дубли фильтрации).
    if (input.dataset.tableSearchInit === '1') return;
    input.dataset.tableSearchInit = '1';
    var rows = table.getElementsByTagName('tr');
    input.addEventListener('keyup', function() {
        var filter = input.value.toLowerCase();
        var i;
        // Первый проход: обычные строки. Вложенные (.smart-usage-row) не участвуют в поиске.
        for (i = 1; i < rows.length; i++) {
            if (rows[i].className.indexOf('smart-usage-row') > -1) continue;
            var show = false;
            var cells = rows[i].getElementsByTagName('td');
            if (cellIndex < 0) {
                // Поиск по всем колонкам
                for (var j = 0; j < cells.length; j++) {
                    if (cells[j] && cells[j].textContent.toLowerCase().indexOf(filter) > -1) {
                        show = true;
                        break;
                    }
                }
            } else {
                // Поиск по конкретной колонке
                var cell = cells[cellIndex || 0];
                if (cell) show = cell.textContent.toLowerCase().indexOf(filter) > -1;
            }
            if (filter === '') {
                rows[i].style.removeProperty('display');
            } else {
                rows[i].style.display = show ? '' : 'none';
            }
        }
        // Второй проход: вложенные строки видимы только если родитель видим И открыт (.usage-open).
        for (i = 1; i < rows.length; i++) {
            if (rows[i].className.indexOf('smart-usage-row') === -1) continue;
            var parent = rows[i].previousElementSibling;
            if (parent && parent.style.display !== 'none' && parent.className.indexOf('usage-open') > -1) {
                rows[i].style.removeProperty('display');
            } else {
                rows[i].style.display = 'none';
            }
        }
    });
}

function loadScript(src) {
    return new Promise(function(resolve, reject) {
        if (document.querySelector('script[src="' + src + '"]')) {
            resolve();
            return;
        }
        var script = document.createElement('script');
        script.src = src;
        script.onload = resolve;
        script.onerror = reject;
        document.head.appendChild(script);
    });
}

// Размер из строки в байты: «2 TB», «1.5 GB», «500 MB», кириллица «2 ТБ»/«500 МБ».
// Единая реализация (панель + tmpfs.html) вместо дублей.
function parseSize(str) {
    if (!str) return 0;
    str = String(str).trim();
    if (!str || str === '—' || str === '-') return 0;
    var units = {
        'B': 1, 'K': 1024, 'M': 1048576, 'G': 1073741824, 'T': 1099511627776,
        'KB': 1024, 'MB': 1048576, 'GB': 1073741824, 'TB': 1099511627776,
        'КБ': 1024, 'МБ': 1048576, 'ГБ': 1073741824, 'ТБ': 1099511627776
    };
    var m = str.match(/^([\d.,]+)\s*([A-Za-zА-Яа-я]+)?$/);
    if (!m) return 0;
    var val = parseFloat(m[1].replace(',', '.'));
    if (isNaN(val)) return 0;
    if (!m[2]) return val;
    var unit = m[2].toUpperCase();
    return units[unit] !== undefined ? val * units[unit] : val;
}

// Размер в байтах → человекочитаемая строка (SMART: «1.5 TB», «500 GB»).
function formatSize(bytes) {
    var n = parseInt(bytes);
    if (!n || isNaN(n)) return '—';
    var KB = 1024, MB = 1048576, GB = 1073741824;
    if (n >= GB * 1024) return (n / (GB * 1024)).toFixed(1) + ' TB';
    if (n >= GB) return Math.round(n / GB) + ' GB';
    if (n >= MB) return Math.round(n / MB) + ' MB';
    if (n >= KB) return Math.round(n / KB) + ' KB';
    return n + ' B';
}
