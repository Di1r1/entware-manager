// Entware Manager — единый механизм тем (пресеты + день/ночь)
// Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
// Ставит data-theme="violet|ocean|forest|teal|amber|ruby|rose" и класс night на <html>.

(function() {
    'use strict';

    var THEMES = [
        { id: 'violet', label: 'Фиолетовый', color: '#8b5cf6' },
        { id: 'ocean',   label: 'Океан',      color: '#0ea5e9' },
        { id: 'forest',  label: 'Изумруд',    color: '#10b981' },
        { id: 'teal',    label: 'Бирюза',     color: '#14b8a6' },
        { id: 'amber',   label: 'Янтарь',     color: '#f59e0b' },
        { id: 'ruby',    label: 'Рубин',      color: '#ef4444' },
        { id: 'rose',    label: 'Роза',       color: '#ec4899' }
    ];

    var STORAGE_KEY = 'entware_theme';
    var NIGHT_KEY = 'entware_night';
    var POLISH_STYLE_ID = 'entware-ui-polish';

    function readStorage(key) {
        try { return localStorage.getItem(key); } catch (e) { return null; }
    }
    function writeStorage(key, val) {
        try { localStorage.setItem(key, val); } catch (e) {}
    }
    function removeStorage(key) {
        try { localStorage.removeItem(key); } catch (e) {}
    }

    // Миграция старых значений: 'day'/'night' → пресет violet + night флаг
    function migrate() {
        var v = readStorage(STORAGE_KEY);
        if (v === 'day' || v === 'night') {
            writeStorage(STORAGE_KEY, 'violet');
            if (v === 'night') writeStorage(NIGHT_KEY, '1');
            else removeStorage(NIGHT_KEY);
        }
    }

    function currentTheme() {
        var id = readStorage(STORAGE_KEY);
        if (!id) return 'violet';
        return THEMES.some(function(t) { return t.id === id; }) ? id : 'violet';
    }

    function isNight() {
        return readStorage(NIGHT_KEY) === '1';
    }

    function applyTheme(themeId, night) {
        var el = document.documentElement;
        el.setAttribute('data-theme', themeId || 'violet');
        if (night) el.classList.add('night');
        else el.classList.remove('night');
    }

    function applyFromStorage() {
        migrate();
        var night = isNight();
        if (!readStorage(NIGHT_KEY)) {
            var h = new Date().getHours();
            night = h >= 20 || h < 6;
        }
        applyTheme(currentTheme(), night);
    }

    // Small visual/accessibility layer kept here so every panel surface that
    // already loads theme.js (main UI, logger and embedded help pages) gets the
    // same polish without another request or a new static-file whitelist entry.
    function injectPolishStyles() {
        if (document.getElementById(POLISH_STYLE_ID)) return;
        var style = document.createElement('style');
        style.id = POLISH_STYLE_ID;
        style.textContent = [
            ':root{--ui-radius:16px;--ui-radius-sm:12px;--ui-focus:0 0 0 3px rgba(var(--accent-rgb),.22);--ui-soft-shadow:0 10px 30px -18px rgba(15,23,42,.28)}',
            'html.night{--ui-soft-shadow:0 14px 34px -20px rgba(0,0,0,.72)}',
            '.sidebar{box-shadow:10px 0 35px -32px rgba(15,23,42,.45)}',
            'html.night .sidebar{box-shadow:10px 0 35px -30px rgba(0,0,0,.9)}',
            '.content{background-image:radial-gradient(circle at 92% 4%,rgba(var(--accent-rgb),.055),transparent 26rem)}',
            '.menu li{margin:5px 10px;padding:8px 13px;border-radius:14px;border:1px solid transparent;box-shadow:none;transition:background-color .18s ease,border-color .18s ease,transform .18s ease,box-shadow .18s ease}',
            '.menu li:hover{transform:translateX(2px);border-color:rgba(var(--accent-rgb),.18);box-shadow:0 8px 22px -18px rgba(var(--accent-rgb),.9)}',
            '.menu li.active{border-left:3px solid var(--accent);border-top-color:rgba(var(--accent-rgb),.18);border-right-color:rgba(var(--accent-rgb),.18);border-bottom-color:rgba(var(--accent-rgb),.18);box-shadow:inset 0 0 0 1px rgba(var(--accent-rgb),.04),0 8px 22px -20px rgba(var(--accent-rgb),.85)}',
            '.stats-hero,.stat-card,.link-card,.packages-table-wrapper,.modal-content{border-radius:var(--ui-radius);box-shadow:var(--ui-soft-shadow)}',
            '.stat-card,.link-card{transition:transform .18s ease,box-shadow .18s ease,border-color .18s ease}',
            '.stat-card:hover,.link-card:hover{transform:translateY(-2px);box-shadow:0 16px 34px -22px rgba(var(--accent-rgb),.75)}',
            'table{border-radius:var(--ui-radius-sm);box-shadow:var(--ui-soft-shadow)}',
            'th,td{padding-top:13px;padding-bottom:13px}',
            'button,input[type="submit"],.packages-delete-btn{transition:transform .16s ease,box-shadow .16s ease,filter .16s ease}',
            'button:hover:not(:disabled),input[type="submit"]:hover:not(:disabled),.packages-delete-btn:hover:not(:disabled){transform:translateY(-1px);box-shadow:0 9px 20px -12px rgba(var(--accent-rgb),.8);filter:saturate(1.04)}',
            'button:focus-visible,a:focus-visible,input:focus-visible,select:focus-visible,textarea:focus-visible,[tabindex]:focus-visible{outline:2px solid var(--accent);outline-offset:2px;box-shadow:var(--ui-focus)}',
            '.settings-input:focus-visible{outline:none;border-color:var(--accent);box-shadow:var(--ui-focus)}',
            '.theme-toggle-edge:focus-visible,.collapse-btn:focus-visible{border-radius:50%}',
            '::selection{background:rgba(var(--accent-rgb),.24)}',
            '@media (max-width:800px){.content{padding:14px}.menu li{margin:4px 8px;border-radius:12px}.stat-card{padding:1.25rem}.stats-hero{padding:1.1rem}}',
            '@media (prefers-reduced-motion:reduce){*,*::before,*::after{scroll-behavior:auto!important;animation-duration:.01ms!important;animation-iteration-count:1!important;transition-duration:.01ms!important}.menu li:hover,.stat-card:hover,.link-card:hover,button:hover:not(:disabled){transform:none!important}}'
        ].join('');
        (document.head || document.documentElement).appendChild(style);
    }

    function init() {
        injectPolishStyles();
        applyFromStorage();
        window.addEventListener('storage', function(e) {
            if (e.key === STORAGE_KEY || e.key === NIGHT_KEY) applyFromStorage();
        });
    }

    function set(themeId, night) {
        writeStorage(STORAGE_KEY, themeId);
        if (night === undefined) {
            // сохраняем текущее/авто состояние дня
        } else {
            writeStorage(NIGHT_KEY, night ? '1' : '0');
        }
        applyTheme(themeId, night === undefined ? isNight() : night);
    }

    window.Theme = {
        THEMES: THEMES,
        init: init,
        set: set,
        current: currentTheme,
        isNight: isNight,
        applyFromStorage: applyFromStorage
    };
})();
