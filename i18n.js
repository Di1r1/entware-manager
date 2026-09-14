// Entware Manager — lightweight frontend localization layer.
// Russian strings remain the canonical source/fallback so existing behavior is
// unchanged when a locale file is missing or incomplete.
(function () {
    'use strict';

    const STORAGE_KEY = 'entware_language';
    const DEFAULT_LANGUAGE = 'ru';
    const SUPPORTED = {
        ru: 'Русский',
        en: 'English',
        tr: 'Türkçe'
    };
    const TRANSLATABLE_ATTRIBUTES = ['title', 'placeholder', 'aria-label'];

    let language = DEFAULT_LANGUAGE;
    let dictionary = {};
    let observer = null;
    let translating = false;

    function uiBase() {
        if (typeof window.UI_BASE === 'string' && window.UI_BASE) return window.UI_BASE;
        return '/entware-manager';
    }

    function readLanguage() {
        try {
            const saved = localStorage.getItem(STORAGE_KEY);
            if (saved && Object.prototype.hasOwnProperty.call(SUPPORTED, saved)) return saved;
        } catch (_) {}

        const browser = String(navigator.language || '').toLowerCase().split('-')[0];
        if (Object.prototype.hasOwnProperty.call(SUPPORTED, browser)) return browser;
        return DEFAULT_LANGUAGE;
    }

    async function loadDictionary(lang) {
        try {
            const response = await fetch(uiBase() + '/locales/' + encodeURIComponent(lang) + '.json?_=' + Date.now(), {
                cache: 'no-store'
            });
            if (!response.ok) throw new Error('HTTP ' + response.status);
            const data = await response.json();
            return data && typeof data === 'object' ? data : {};
        } catch (err) {
            console.warn('i18n: locale could not be loaded, using Russian fallback:', lang, err);
            return {};
        }
    }

    function translated(value) {
        if (typeof value !== 'string' || !value) return value;
        const key = value.trim();
        if (!key) return value;
        const replacement = dictionary[key];
        if (typeof replacement !== 'string' || replacement === key) return value;
        const start = value.indexOf(key);
        if (start < 0) return value;
        return value.slice(0, start) + replacement + value.slice(start + key.length);
    }

    function translateTextNode(node) {
        if (!node || node.nodeType !== Node.TEXT_NODE) return;
        const next = translated(node.nodeValue);
        if (next !== node.nodeValue) node.nodeValue = next;
    }

    function translateAttributes(element) {
        if (!element || element.nodeType !== Node.ELEMENT_NODE) return;
        TRANSLATABLE_ATTRIBUTES.forEach(function (name) {
            if (!element.hasAttribute(name)) return;
            const current = element.getAttribute(name);
            const next = translated(current);
            if (next !== current) element.setAttribute(name, next);
        });
    }

    function translateTree(root) {
        if (!root || language === DEFAULT_LANGUAGE || !dictionary) return;
        translating = true;
        try {
            if (root.nodeType === Node.TEXT_NODE) {
                translateTextNode(root);
                return;
            }
            if (root.nodeType !== Node.ELEMENT_NODE && root.nodeType !== Node.DOCUMENT_NODE) return;

            if (root.nodeType === Node.ELEMENT_NODE) translateAttributes(root);
            const walker = document.createTreeWalker(root, NodeFilter.SHOW_ELEMENT | NodeFilter.SHOW_TEXT);
            let node;
            while ((node = walker.nextNode())) {
                if (node.nodeType === Node.TEXT_NODE) translateTextNode(node);
                else translateAttributes(node);
            }
        } finally {
            translating = false;
        }
    }

    function observe() {
        if (!document.body || observer) return;
        observer = new MutationObserver(function (mutations) {
            if (translating || language === DEFAULT_LANGUAGE) return;
            mutations.forEach(function (mutation) {
                if (mutation.type === 'characterData') {
                    translateTextNode(mutation.target);
                    return;
                }
                if (mutation.type === 'attributes') {
                    translateAttributes(mutation.target);
                    return;
                }
                mutation.addedNodes.forEach(translateTree);
            });
        });
        observer.observe(document.body, {
            subtree: true,
            childList: true,
            characterData: true,
            attributes: true,
            attributeFilter: TRANSLATABLE_ATTRIBUTES
        });
    }

    function writeLanguage(lang) {
        if (!Object.prototype.hasOwnProperty.call(SUPPORTED, lang)) return;
        try { localStorage.setItem(STORAGE_KEY, lang); } catch (_) {}
        window.location.reload();
    }

    function makeSelect(context) {
        const select = document.createElement('select');
        select.className = 'i18n-language-select';
        select.setAttribute('aria-label', language === 'tr' ? 'Dil' : language === 'en' ? 'Language' : 'Язык');
        select.title = language === 'tr' ? 'Arayüz dili' : language === 'en' ? 'Interface language' : 'Язык интерфейса';
        Object.keys(SUPPORTED).forEach(function (code) {
            const option = document.createElement('option');
            option.value = code;
            option.textContent = SUPPORTED[code];
            select.appendChild(option);
        });
        select.value = language;
        select.addEventListener('change', function () { writeLanguage(select.value); });

        if (context === 'login') {
            select.style.cssText = 'margin-top:12px;width:100%;padding:7px 10px;border-radius:8px;border:1px solid var(--border-color,#cbd5e1);background:var(--card-bg,#fff);color:var(--text-primary,#1f2937);';
        } else {
            select.style.cssText = 'width:100%;max-width:150px;margin:4px auto 6px;padding:4px 6px;border-radius:6px;border:1px solid var(--border-color,#cbd5e1);background:var(--card-bg,#fff);color:var(--text-primary,#1f2937);font-size:11px;';
        }
        return select;
    }

    function addLanguageSwitchers() {
        const footer = document.querySelector('.sidebar-footer');
        if (footer && !footer.querySelector('.i18n-language-select')) {
            const wrapper = document.createElement('div');
            wrapper.className = 'i18n-language-switcher';
            wrapper.style.cssText = 'display:flex;justify-content:center;width:100%;';
            wrapper.appendChild(makeSelect('sidebar'));
            footer.insertBefore(wrapper, footer.firstChild);
        }

        const loginCard = document.querySelector('.login-card');
        if (loginCard && !loginCard.querySelector('.i18n-language-select')) {
            loginCard.appendChild(makeSelect('login'));
        }
    }

    async function initialize() {
        language = readLanguage();
        document.documentElement.lang = language;
        dictionary = await loadDictionary(language);

        if (document.readyState === 'loading') {
            await new Promise(function (resolve) {
                document.addEventListener('DOMContentLoaded', resolve, { once: true });
            });
        }

        translateTree(document.body);
        addLanguageSwitchers();
        observe();
    }

    const ready = initialize();

    window.I18n = {
        ready: ready,
        getLanguage: function () { return language; },
        setLanguage: writeLanguage,
        t: function (source) {
            if (typeof source !== 'string') return source;
            return dictionary[source] || source;
        },
        translate: translateTree,
        supported: Object.assign({}, SUPPORTED)
    };
})();
