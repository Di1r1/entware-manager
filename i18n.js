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
    const LANGUAGE_NAMES = {
        ru: { ru: 'Русский', en: 'English', tr: 'Türkçe' },
        en: { ru: 'Russian', en: 'English', tr: 'Turkish' },
        tr: { ru: 'Rusça', en: 'İngilizce', tr: 'Türkçe' }
    };
    const TRANSLATABLE_ATTRIBUTES = ['title', 'placeholder', 'aria-label'];
    const CYRILLIC_RE = /[\u0400-\u04FF]/;
    const SKIP_TEXT_TAGS = /^(SCRIPT|STYLE|NOSCRIPT|CODE|PRE)$/i;

    let language = DEFAULT_LANGUAGE;
    let dictionary = {};
    const observers = new WeakMap();
    const observedFrames = new WeakSet();
    const pendingByDocument = new WeakMap();

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
        if (!node || node.nodeType !== 3) return;
        const parent = node.parentElement;
        if (parent && SKIP_TEXT_TAGS.test(parent.tagName)) return;
        const next = translated(node.nodeValue);
        if (next !== node.nodeValue) node.nodeValue = next;
    }

    function translateAttributes(element) {
        if (!element || element.nodeType !== 1) return;
        TRANSLATABLE_ATTRIBUTES.forEach(function (name) {
            if (!element.hasAttribute(name)) return;
            const current = element.getAttribute(name);
            const next = translated(current);
            if (next !== current) element.setAttribute(name, next);
        });
    }

    function translateIframe(frame) {
        if (!frame || frame.nodeType !== 1 || frame.tagName !== 'IFRAME') return;
        const apply = function () {
            try {
                const doc = frame.contentDocument;
                if (!doc || !doc.body) return;
                translateTree(doc);
                observeDocument(doc);
            } catch (_) {
                // Cross-origin frames are intentionally ignored.
            }
        };
        if (!observedFrames.has(frame)) {
            observedFrames.add(frame);
            frame.addEventListener('load', apply);
        }
        apply();
    }

    function translateTree(root) {
        if (!root || language === DEFAULT_LANGUAGE || !dictionary) return;
        if (root.nodeType === 3) {
            translateTextNode(root);
            return;
        }
        if (root.nodeType !== 1 && root.nodeType !== 9) return;

        const doc = root.nodeType === 9 ? root : (root.ownerDocument || document);
        const view = doc.defaultView || window;
        if (root.nodeType === 1) {
            translateAttributes(root);
            if (root.tagName === 'IFRAME') translateIframe(root);
        }
        const walker = doc.createTreeWalker(root, view.NodeFilter.SHOW_ELEMENT | view.NodeFilter.SHOW_TEXT);
        let node;
        while ((node = walker.nextNode())) {
            if (node.nodeType === 3) {
                translateTextNode(node);
            } else {
                translateAttributes(node);
                if (node.tagName === 'IFRAME') translateIframe(node);
            }
        }
    }

    function pendingState(doc) {
        let state = pendingByDocument.get(doc);
        if (!state) {
            state = { nodes: new Set(), scheduled: false };
            pendingByDocument.set(doc, state);
        }
        return state;
    }

    function isCoveredByAnotherRoot(node, nodes) {
        if (!node || !nodes) return false;
        for (const other of nodes) {
            if (other === node || !other || other.nodeType !== 1) continue;
            try {
                if (other.contains(node)) return true;
            } catch (_) {}
        }
        return false;
    }

    function flushPending(doc) {
        const state = pendingByDocument.get(doc);
        if (!state) return;
        state.scheduled = false;
        if (language === DEFAULT_LANGUAGE || !state.nodes.size) {
            state.nodes.clear();
            return;
        }

        const nodes = Array.from(state.nodes);
        state.nodes.clear();
        nodes.forEach(function (node) {
            if (!node || !node.isConnected || isCoveredByAnotherRoot(node, nodes)) return;
            translateTree(node);
        });
    }

    function queueTranslation(doc, node) {
        if (!doc || !node || language === DEFAULT_LANGUAGE) return;
        const state = pendingState(doc);
        state.nodes.add(node);
        if (state.scheduled) return;
        state.scheduled = true;
        const view = doc.defaultView || window;
        const schedule = typeof view.requestAnimationFrame === 'function'
            ? view.requestAnimationFrame.bind(view)
            : function (cb) { return view.setTimeout(cb, 0); };
        schedule(function () { flushPending(doc); });
    }

    function observeDocument(doc) {
        if (!doc || !doc.body || observers.has(doc)) return;
        const ViewMutationObserver = (doc.defaultView && doc.defaultView.MutationObserver) || MutationObserver;
        const observer = new ViewMutationObserver(function (mutations) {
            if (language === DEFAULT_LANGUAGE) return;
            mutations.forEach(function (mutation) {
                if (mutation.type === 'characterData') {
                    queueTranslation(doc, mutation.target);
                    return;
                }
                mutation.addedNodes.forEach(function (node) {
                    queueTranslation(doc, node);
                });
            });
        });
        // Attribute changes are intentionally not observed. Attributes are
        // translated during the initial/added-subtree pass. Observing our own
        // title/placeholder writes creates avoidable feedback and heavy work on
        // large dynamic tables.
        observer.observe(doc.body, {
            subtree: true,
            childList: true,
            characterData: true
        });
        observers.set(doc, observer);
    }

    function writeLanguage(lang) {
        if (!Object.prototype.hasOwnProperty.call(SUPPORTED, lang)) return;
        try { localStorage.setItem(STORAGE_KEY, lang); } catch (_) {}

        // A cache-busting navigation avoids reusing an old HTML/JS shell after
        // an Entware Manager update. Keep the rest of the current query intact.
        try {
            const url = new URL(window.location.href);
            url.searchParams.set('_lang', String(Date.now()));
            window.location.replace(url.toString());
        } catch (_) {
            window.location.reload();
        }
    }

    function languageLabel(code) {
        const labels = LANGUAGE_NAMES[language] || LANGUAGE_NAMES[DEFAULT_LANGUAGE];
        return labels[code] || SUPPORTED[code] || code;
    }

    function makeSelect(context) {
        const select = document.createElement('select');
        select.className = 'i18n-language-select';
        select.setAttribute('aria-label', language === 'tr' ? 'Dil' : language === 'en' ? 'Language' : 'Язык');
        select.title = language === 'tr' ? 'Arayüz dili' : language === 'en' ? 'Interface language' : 'Язык интерфейса';
        Object.keys(SUPPORTED).forEach(function (code) {
            const option = document.createElement('option');
            option.value = code;
            option.textContent = languageLabel(code);
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

    function collectCyrillic(doc, out) {
        if (!doc || !doc.body) return;
        const view = doc.defaultView || window;
        const walker = doc.createTreeWalker(doc.body, view.NodeFilter.SHOW_TEXT);
        let node;
        while ((node = walker.nextNode())) {
            const parent = node.parentElement;
            if (!parent || /^(SCRIPT|STYLE|NOSCRIPT)$/i.test(parent.tagName)) continue;
            const text = String(node.nodeValue || '').trim();
            if (text && CYRILLIC_RE.test(text)) out.add(text);
        }
        doc.querySelectorAll('iframe').forEach(function (frame) {
            try { collectCyrillic(frame.contentDocument, out); } catch (_) {}
        });
    }

    function audit() {
        const out = new Set();
        collectCyrillic(document, out);
        return Array.from(out).sort();
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
        observeDocument(document);
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
        audit: audit,
        supported: Object.assign({}, SUPPORTED)
    };
})();
