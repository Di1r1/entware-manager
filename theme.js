// Entware Manager — единый механизм тем (пресеты + день/ночь)
// Ставит data-theme="violet|ocean|forest" и класс night на <html>.

(function() {
    'use strict';

    var THEMES = [
        { id: 'violet', label: 'Фиолетовый', color: '#8b5cf6' },
        { id: 'ocean',   label: 'Океан',      color: '#0ea5e9' },
        { id: 'forest',  label: 'Изумруд',    color: '#10b981' }
    ];

    var STORAGE_KEY = 'entware_theme';
    var NIGHT_KEY = 'entware_night';

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

    function init() {
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
