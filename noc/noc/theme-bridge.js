/* Entware Manager — theme-bridge (СКЕЛЕТ, tmp/noc-migration).
 * Мост между классикой (theme.js: data-theme + класс night) и NOC-моком
 * (console.js: data-console / data-accent / data-density).
 *
 * Статус: черновик для ревью, прод не тронут. Подключение (будущее):
 *   <script src="/entware-commander/theme-bridge.js?v=1"></script>
 * поверх theme.js (classic) или вместо блока тем в console.js (NOC).
 *
 * Источник правды — 4 ключа localStorage (не переименовываются):
 *   entware_theme        — 7 акцентов, общие у обеих систем
 *   entware_night        — '1'/'0', ночь классики
 *   entware_console_base — 'noc-dark'/'ops-light'/'hc', база NOC
 *   entware_density      — 'comfortable'/'compact', плотность NOC
 */
(function () {
    'use strict';

    var THEME_KEY = 'entware_theme';
    var NIGHT_KEY = 'entware_night';
    var BASE_KEY = 'entware_console_base';
    var DENSITY_KEY = 'entware_density';

    var ACCENTS = ['violet', 'ocean', 'forest', 'teal', 'amber', 'ruby', 'rose'];
    var BASES = ['noc-dark', 'ops-light', 'hc'];

    function read(key) {
        try { return localStorage.getItem(key); } catch (e) { return null; }
    }
    function write(key, val) {
        try { localStorage.setItem(key, val); } catch (e) {}
    }

    function validAccent(a) {
        return ACCENTS.indexOf(a) >= 0 ? a : 'violet';
    }
    function validBase(b) {
        return BASES.indexOf(b) >= 0 ? b : null;
    }

    /* Ночь классики -> база NOC (только если base явно не задан).
     * night=1 -> noc-dark, night=0 -> ops-light. 'hc' — только вручную. */
    function baseFromNight(night, explicitBase) {
        var vb = validBase(explicitBase);
        if (vb) return vb;
        return night ? 'noc-dark' : 'ops-light';
    }

    function isNight() {
        var raw = read(NIGHT_KEY);
        if (raw === '1') return true;
        if (raw === '0') return false;
        /* авто-режим как в theme.js: ночь 20:00–06:00 */
        var h = new Date().getHours();
        return h >= 20 || h < 6;
    }

    /* Применить оба набора атрибутов сразу — классика и NOC довольны. */
    function applyAll() {
        var root = document.documentElement;
        var accent = validAccent(read(THEME_KEY) || 'violet');
        var night = isNight();
        var base = baseFromNight(night, read(BASE_KEY));
        var density = read(DENSITY_KEY) === 'compact' ? 'compact' : 'comfortable';

        /* классика (theme.js) */
        root.setAttribute('data-theme', accent);
        if (night) root.classList.add('night');
        else root.classList.remove('night');

        /* NOC (console.js) */
        root.setAttribute('data-accent', accent);
        root.setAttribute('data-console', base);
        root.setAttribute('data-density', density);
    }

    /* Смена из классики (кружки theme-popup / кнопка день-ночь). */
    function setClassic(themeId, night) {
        if (themeId) write(THEME_KEY, validAccent(themeId));
        if (night !== undefined) write(NIGHT_KEY, night ? '1' : '0');
        applyAll();
    }

    /* Смена из NOC (accentDots / baseDots / densityBtn). */
    function setNoc(accent, base, density) {
        if (accent) write(THEME_KEY, validAccent(accent));
        if (base && validBase(base)) write(BASE_KEY, base);
        if (density) write(DENSITY_KEY, density === 'compact' ? 'compact' : 'comfortable');
        applyAll();
    }

    function init() {
        applyAll();
        window.addEventListener('storage', function (e) {
            if (e.key === THEME_KEY || e.key === NIGHT_KEY ||
                e.key === BASE_KEY || e.key === DENSITY_KEY) {
                applyAll();
            }
        });
    }

    window.ThemeBridge = {
        init: init,
        applyAll: applyAll,
        setClassic: setClassic,
        setNoc: setNoc,
        isNight: isNight
    };

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
})();
