/* NOC-мок: только фейковые данные и переключение тем. Никаких fetch к CGI. */
(function () {
  'use strict';
  var root = document.documentElement;

  /* --- фейковый лог --- */
  var lines = [
    '[08:12:01] watchdog: check ok (cpu 12%, mem 61%)',
    '[08:12:31] monitor: top snapshot (5 procs)',
    '[08:13:02] network: br0 carrier 1, ppp0 up 94 Mbit/s',
    '[08:14:44] opkg: 3 upgrades available (htop, jq, ttyd)',
    '[08:15:00] ttyd: session open :9089 (entware)',
    '[08:15:20] backup: skipped (no changes since 02:00)'
  ];
  var logEl = document.getElementById('fakeLog');
  if (logEl) logEl.textContent = lines.join('\n');

  /* --- живая имитация метров + sparklines --- */
  var cpu = 23, ram = 61, disk = 38, wan = 94, temp = 58;
  var SPARK_N = 24;
  function seedSeries(base, spread) {
    var a = [], i, v = base;
    for (i = 0; i < SPARK_N; i++) {
      v = Math.max(0, Math.min(100, v + (Math.random() * spread * 2 - spread)));
      a.push(Math.round(v * 10) / 10);
    }
    a[a.length - 1] = base;
    return a;
  }
  var series = {
    cpu: seedSeries(cpu, 8),
    ram: seedSeries(ram, 3),
    disk: seedSeries(disk, 0.4),
    wan: seedSeries(wan, 6),
    temp: seedSeries(temp, 1.5)
  };
  function barColor(v) {
    if (v >= 90) return 'var(--crit)';
    if (v >= 75) return 'var(--warn)';
    return 'var(--accent)';
  }
  function pushSeries(key, val) {
    var a = series[key];
    a.push(val);
    if (a.length > SPARK_N) a.shift();
  }
  function drawSpark(svgIdOrEl, data, maxHint, height) {
    var el = typeof svgIdOrEl === 'string' ? document.getElementById(svgIdOrEl) : svgIdOrEl;
    if (!el || !data || !data.length) return;
    var w = 120, h = height || 36, pad = 2;
    var max = maxHint || 100;
    var min = 0;
    var i, n = data.length;
    for (i = 0; i < n; i++) {
      if (data[i] > max) max = data[i];
    }
    if (max <= 0) max = 1;
    function x(i) { return pad + (i / Math.max(1, n - 1)) * (w - pad * 2); }
    function y(v) { return h - pad - ((v - min) / (max - min)) * (h - pad * 2); }
    var line = '', fill = '';
    for (i = 0; i < n; i++) {
      var xi = x(i), yi = y(data[i]);
      line += (i ? ' L' : 'M') + xi.toFixed(1) + ' ' + yi.toFixed(1);
    }
    fill = line + ' L' + x(n - 1).toFixed(1) + ' ' + (h - pad) +
      ' L' + x(0).toFixed(1) + ' ' + (h - pad) + ' Z';
    var lastX = x(n - 1), lastY = y(data[n - 1]);
    el.setAttribute('viewBox', '0 0 ' + w + ' ' + h);
    el.innerHTML =
      '<path class="fill" d="' + fill + '"/>' +
      '<path class="line" d="' + line + '"/>' +
      '<circle class="dot" cx="' + lastX.toFixed(1) + '" cy="' + lastY.toFixed(1) + '" r="2"/>';
  }
  function paintSparks() {
    drawSpark('sparkCpu', series.cpu, 100);
    drawSpark('sparkRam', series.ram, 100);
    drawSpark('sparkDisk', series.disk, 100);
    drawSpark('sparkWan', series.wan, 120);
    drawSpark('sparkTemp', series.temp, 90);
    paintModSparks();
  }
  /* service micro-sparks — filled after MODS is defined */
  var modSeries = {};
  function parseCpuPct(s) {
    var n = parseFloat(String(s).replace('%', ''));
    return isNaN(n) ? 0 : n;
  }
  function ensureModSeries() {
    if (!window.__MODS_REF) return;
    window.__MODS_REF.forEach(function (m) {
      if (!modSeries[m.id]) {
        var base = m.on ? parseCpuPct(m.cpu) : 0;
        modSeries[m.id] = seedSeries(base || 0.5, base > 0 ? Math.max(0.4, base * 0.35) : 0.2);
        if (!m.on) {
          for (var i = 0; i < modSeries[m.id].length; i++) modSeries[m.id][i] = 0;
        }
      }
    });
  }
  function paintModSparks() {
    ensureModSeries();
    var nodes = document.querySelectorAll('.svc-spark[data-mid]');
    Array.prototype.forEach.call(nodes, function (svg) {
      var id = svg.getAttribute('data-mid');
      var data = modSeries[id];
      if (!data) return;
      var maxH = Math.max.apply(null, data.concat([1]));
      drawSpark(svg, data, Math.max(maxH * 1.2, 2), 42);
    });
  }
  function jitterModSeries() {
    ensureModSeries();
    if (!window.__MODS_REF) return;
    window.__MODS_REF.forEach(function (m) {
      var a = modSeries[m.id];
      if (!a) return;
      if (!m.on) {
        a.push(0);
      } else {
        var cur = a[a.length - 1];
        var base = parseCpuPct(m.cpu) || cur;
        var next = Math.max(0, Math.min(30, cur + (Math.random() * 1.4 - 0.7)));
        // gently pull toward nominal
        next = next * 0.7 + base * 0.3;
        a.push(Math.round(next * 10) / 10);
      }
      if (a.length > SPARK_N) a.shift();
    });
  }
  function paintMeters() {
    var cb = document.getElementById('cpuBar'), rb = document.getElementById('ramBar');
    if (cb) { cb.style.width = cpu + '%'; cb.style.background = barColor(cpu); }
    if (rb) { rb.style.width = ram + '%'; rb.style.background = barColor(ram); }
    var cv = document.getElementById('cpuVal'), rv = document.getElementById('ramVal');
    if (cv) cv.textContent = cpu + '%';
    if (rv) rv.textContent = ram + '%';
    var kc = document.getElementById('kpiCpu');
    if (kc) kc.textContent = String(cpu);
    var kr = document.getElementById('kpiRam');
    if (kr) kr.textContent = String(ram);
    var kd = document.getElementById('kpiDisk');
    if (kd) kd.textContent = String(Math.round(disk));
    var kw = document.getElementById('kpiWan');
    /* WAN-скорость: источника нет — в LIVE цифру не рисуем */
    if (kw && !window.__NOC_LIVE_STATS) kw.textContent = String(Math.round(wan));
    var kt = document.getElementById('kpiTemp');
    if (kt) kt.textContent = String(Math.round(temp));
    paintSparks();
  }
  function jitterMeters() {
    if (window.__NOC_LIVE_STATS) return; /* LIVE-режим: цифры с роутера, мок заморожен */
    window.__NOC_MOCK_N = (window.__NOC_MOCK_N || 0) + 1; /* диагностика: сколько раз сработал мок */
    var jb = document.getElementById('dashRefresh');
    if (jb && window.__NOC_LIVE_STATS !== true && window.__NOC_LIVE_ONCE) {
      jb.textContent = '↻ MOCK (live слетел) #' + window.__NOC_MOCK_N;
    }
    cpu = Math.max(5, Math.min(95, cpu + Math.round(Math.random() * 6 - 3)));
    ram = Math.max(40, Math.min(90, ram + Math.round(Math.random() * 2 - 1)));
    disk = Math.max(30, Math.min(50, disk + (Math.random() * 0.3 - 0.1)));
    wan = Math.max(60, Math.min(110, wan + Math.round(Math.random() * 8 - 4)));
    temp = Math.max(48, Math.min(72, temp + (Math.random() * 1.2 - 0.6)));
    pushSeries('cpu', cpu);
    pushSeries('ram', ram);
    pushSeries('disk', disk);
    jitterModSeries();
    pushSeries('wan', wan);
    pushSeries('temp', temp);
    paintMeters();
  }
  paintMeters();
  setInterval(jitterMeters, 2000);

  /* --- темы: базы + акценты (ключи как в theme.js + entware_console_base) --- */
  var ACCENTS = ['violet', 'ocean', 'forest', 'teal', 'amber', 'ruby', 'rose'];
  var ACCENT_HEX = { violet: '#8b5cf6', ocean: '#0ea5e9', forest: '#10b981', teal: '#14b8a6', amber: '#f59e0b', ruby: '#ef4444', rose: '#ec4899' };
  var BASES = ['noc-dark', 'ops-light', 'hc'];
  var BASE_HEX = { 'noc-dark': '#0f172a', 'ops-light': '#ffffff', 'hc': '#facc15' };

  function lsGet(k) { try { return localStorage.getItem(k); } catch (e) { return null; } }
  function lsSet(k, v) { try { localStorage.setItem(k, v); } catch (e) {} }

  function applyAll() {
    root.setAttribute('data-console', lsGet('entware_console_base') || 'noc-dark');
    root.setAttribute('data-accent', lsGet('entware_theme') || 'violet');
    root.setAttribute('data-density', lsGet('entware_density') || 'comfortable');
    paintDots();
  }
  function paintDots() {
    var cur = lsGet('entware_theme') || 'violet';
    var ad = document.getElementById('accentDots');
    ad.innerHTML = '';
    ACCENTS.forEach(function (a) {
      var i = document.createElement('i');
      i.title = a;
      i.style.background = ACCENT_HEX[a];
      if (a === cur) i.className = 'on';
      i.onclick = function () { lsSet('entware_theme', a); applyAll(); };
      ad.appendChild(i);
    });
    var cb = lsGet('entware_console_base') || 'noc-dark';
    var bd = document.getElementById('baseDots');
    bd.innerHTML = '';
    BASES.forEach(function (b) {
      var i = document.createElement('i');
      i.title = b;
      i.style.background = BASE_HEX[b];
      if (b === cb) i.className = 'on';
      i.onclick = function () { lsSet('entware_console_base', b); applyAll(); };
      bd.appendChild(i);
    });
  }

  document.getElementById('densityBtn').onclick = function () {
    var d = lsGet('entware_density') || 'comfortable';
    lsSet('entware_density', d === 'comfortable' ? 'compact' : 'comfortable');
    applyAll();
    toast('Плотность: ' + (lsGet('entware_density') === 'compact' ? 'компакт' : 'комфорт'));
  };

  /* --- rail: переключение экранов --- */
  var railItems = Array.prototype.slice.call(document.querySelectorAll('.ri'));
  var crumbs = document.getElementById('crumbs');
  var viewStats = document.getElementById('view-stats');
  var viewMods = document.getElementById('view-modules');
  var CRUMBS = { stats: 'обзор / статистика', packages: 'система / пакеты', processes: 'система / процессы', terminal: 'система / терминал', 'system-services': 'система / службы и cron', network: 'сеть / сеть', bridge: 'сеть / модули', smart: 'сеть / smart', rdp: 'сеть / rdp', settings: 'администрирование / настройки', monitor: 'администрирование / защита', help: 'администрирование / справка', logs: 'администрирование / логи' };
  var VIEWS = { stats: 'view-stats', bridge: 'view-modules', packages: 'view-packages', processes: 'view-processes', terminal: 'view-terminal', 'system-services': 'view-services', network: 'view-network', smart: 'view-smart', rdp: 'view-rdp', settings: 'view-settings', monitor: 'view-monitor', help: 'view-help', logs: 'view-logs' };
  railItems.forEach(function (b) {
    b.onclick = function () {
      railItems.forEach(function (x) { x.classList.remove('active'); });
      b.classList.add('active');
      var tab = b.getAttribute('data-tab');
      lsSet('entware_noc_tab', tab);
      crumbs.textContent = CRUMBS[tab] || ('мок / ' + b.getAttribute('data-name').toLowerCase());
      Object.keys(VIEWS).forEach(function (k) {
        var el = document.getElementById(VIEWS[k]);
        if (el) el.hidden = (k !== tab);
      });
    };
  });
  document.getElementById('railPin').onclick = function () {
    document.querySelector('.noc').classList.toggle('pinned');
  };
  document.addEventListener('keydown', function (e) {
    if (e.altKey && e.key.toLowerCase() === 'm') {
      e.preventDefault();
      document.querySelector('.noc').classList.toggle('pinned');
    }
  });

  /* --- палитра Ctrl+K --- */
  var TABS = [
    ['stats', 'Статистика', 'обзор'], ['packages', 'Пакеты', 'система'],
    ['processes', 'Процессы', 'система'], ['terminal', 'Терминал', 'система'],
    ['system-services', 'Службы и Cron', 'система'], ['network', 'Сеть', 'сеть'],
    ['bridge', 'Модули', 'сеть'], ['smart', 'SMART', 'сеть'], ['rdp', 'RDP', 'сеть'],
    ['settings', 'Настройки', 'администрирование'], ['monitor', 'Защита', 'администрирование'],
    ['help', 'Справка', 'администрирование'], ['logs', 'Логи', 'администрирование']
  ];
  var ACTS = [
    ['Проверка системы', 'действие'], ['Бэкап', 'действие'],
    ['Обновление', 'действие'], ['Перезагрузка демонов', 'действие']
  ];
  var wrap = document.getElementById('palWrap');
  var input = document.getElementById('palInput');
  var list = document.getElementById('palList');
  var sel = 0, shown = [];

  function openPal() { wrap.hidden = false; input.value = ''; renderPal(''); setTimeout(function () { input.focus(); }, 0); }
  function closePal() { wrap.hidden = true; }
  function renderPal(q) {
    q = q.toLowerCase();
    shown = [];
    TABS.forEach(function (t) {
      if (!q || t[1].toLowerCase().indexOf(q) >= 0 || t[2].indexOf(q) >= 0) shown.push({ label: t[1], hint: 'вкладка · ' + t[2], tab: t[0] });
    });
    ACTS.forEach(function (a) {
      if (!q || a[0].toLowerCase().indexOf(q) >= 0) shown.push({ label: a[0], hint: a[1] + ' · заглушка', act: a[0] });
    });
    sel = 0;
    list.innerHTML = '';
    if (!shown.length) { list.innerHTML = '<div class="pal-item"><span>Ничего не найдено</span></div>'; return; }
    shown.forEach(function (s, i) {
      var d = document.createElement('div');
      d.className = 'pal-item' + (i === 0 ? ' sel' : '');
      d.innerHTML = '<span></span><small></small>';
      d.firstChild.textContent = s.label;
      d.lastChild.textContent = s.hint;
      d.onclick = function () { runItem(s); };
      list.appendChild(d);
    });
  }
  function runItem(s) {
    closePal();
    if (s.tab) {
      var b = document.querySelector('.ri[data-tab="' + s.tab + '"]');
      if (b) b.click();
    } else {
      toast('«' + s.label + '» — заглушка мока, в проде будет реальное действие');
    }
  }
  input.oninput = function () { renderPal(input.value); };
  input.onkeydown = function (e) {
    if (e.key === 'Escape') { closePal(); return; }
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      e.preventDefault();
      sel = e.key === 'ArrowDown' ? Math.min(shown.length - 1, sel + 1) : Math.max(0, sel - 1);
      var items = list.querySelectorAll('.pal-item');
      for (var i = 0; i < items.length; i++) items[i].classList.toggle('sel', i === sel);
      return;
    }
    if (e.key === 'Enter' && shown[sel]) runItem(shown[sel]);
  };
  wrap.onclick = function (e) { if (e.target === wrap) closePal(); };
  document.getElementById('paletteBtn').onclick = openPal;
  document.addEventListener('keydown', function (e) {
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') { e.preventDefault(); wrap.hidden ? openPal() : closePal(); }
    if (e.key === 'Escape' && !wrap.hidden) closePal();
  });

  var toastT;
  function toast(msg, type) {
    var t = document.getElementById('toast');
    t.textContent = msg;
    t.hidden = false;
    t.style.borderColor = type === 'warn' ? 'var(--warn)' : type === 'err' ? 'var(--crit)' : 'var(--accent)';
    clearTimeout(toastT);
    toastT = setTimeout(function () { t.hidden = true; }, 2400);
  }

  /* --- модули: фейковые данные в структуре renderBridgeCard --- */
  /* --- модули: манифесты /opt/web_entware/bridge/ + prefs --- */
  var MODS = [
    {
      id: 'adguard', name: 'AdGuard Home', state: 'running', color: '#22c55e', label: 'работает',
      detail: 'base :8080 · probe /control/status · OK',
      port: ':8080', base: 'http://127.0.0.1:8080',
      mem: '66M', cpu: '3.7%', up: '12д 4ч',
      on: true, notify: true, builtin: true, ctl: false,
      actions: [
        { id: 'open', label: 'Открыть' },
        { id: 'protection_toggle', label: 'Переключить защиту', confirm: true }
      ],
      fields: [
        { path: 'protection_enabled', label: 'Защита', type: 'bool', tile: true, on: 'вкл', off: 'выкл', value: true },
        { path: 'num_dns_queries', label: 'Запросов за сутки', from: 'stats', type: 'num', tile: true, value: 12408 },
        { path: 'num_blocked_filtering', label: 'Заблокировано', from: 'stats', type: 'num', tile: true, value: 2234 },
        { path: 'avg_processing_time', label: 'Ответ DNS (мс)', from: 'stats', type: 'ms', tile: true, value: 0.004 },
        { path: 'version', label: 'Версия', value: '0.107.52' },
        { path: 'top_clients', label: 'Топ клиенты', from: 'stats', type: 'top', value: 'phone 4 102 · laptop 3 377' },
        { path: 'top_blocked_domains', label: 'Топ блокировок', from: 'stats', type: 'top', value: 'metrics.× 412' }
      ]
    },
    {
      id: 'koffe', name: 'Koffe VPN', state: 'running', color: '#22c55e', label: 'работает',
      detail: 'base :9097 · probe ?action=version · OK',
      port: ':9097', base: 'http://127.0.0.1:9097/',
      mem: '12M', cpu: '1.9%', up: '12д 4ч',
      on: true, notify: true, builtin: true, ctl: false,
      actions: [
        { id: 'open', label: 'Открыть' },
        { id: 'restart', label: 'Перезапустить', confirm: true }
      ],
      fields: [
        { path: 'running', label: 'Туннель', type: 'bool', tile: true, on: 'вкл', off: 'выкл', value: true },
        { path: 'mode', label: 'Режим', tile: true, value: 'split' },
        { path: 'server', label: 'Сервер', tile: true, value: 'nl-ams-1' },
        { path: 'last_latency_ms', label: 'Пинг VPS (мс)', from: 'health', tile: true, value: 28 },
        { path: 'current_name', label: 'Failover: активный сервер', from: 'failover', value: 'nl-ams-1' },
        { path: 'device_bypass', label: 'Байпас', type: 'count', value: 2 },
        { path: 'device_tunnel', label: 'Туннель устройств', type: 'count', value: 5 },
        { path: 'uptime', label: 'Аптайм', value: '12д 4ч' },
        { path: 'split_count', label: 'Списки IP: сплит', tile: false, value: 3 }
      ]
    },
    {
      id: 'syncthing', name: 'Syncthing', state: 'running', color: '#22c55e', label: 'работает',
      detail: 'process + init · probe / · OK',
      port: ':8384', base: 'http://127.0.0.1:8384',
      mem: '52M', cpu: '2.4%', up: '12д 4ч',
      on: true, notify: false, builtin: true, ctl: true, init: 'syncthing', process: ['syncthing'],
      actions: [{ id: 'open', label: 'Открыть' }],
      fields: [
        { path: 'uptime', label: 'Аптайм', type: 'dur', tile: true, value: 1058400 },
        { path: 'total.inBytesTotal', label: 'Принято всего', from: 'conns', type: 'bytes', tile: true, value: 8589934592 },
        { path: 'total.outBytesTotal', label: 'Отправлено всего', from: 'conns', type: 'bytes', tile: true, value: 3221225472 },
        { path: 'myID', label: 'ID устройства', value: 'ABCD123 ··· XYZ9' }
      ]
    },
    {
      id: 'xray', name: 'xray', state: 'running', color: '#22c55e', label: 'работает',
      detail: 'process: xray · без веб-UI',
      port: '', base: '',
      mem: '41M', cpu: '8.2%', up: '6д 2ч',
      on: true, notify: false, builtin: false, ctl: false, process: ['xray'],
      actions: [],
      fields: [
        { path: 'pid', label: 'PID', tile: true, value: '8162' },
        { path: 'rss', label: 'RSS', tile: true, value: '41M' }
      ]
    },
    {
      id: 'hysteria', name: 'hysteria', state: 'running', color: '#22c55e', label: 'работает',
      detail: 'process: hysteria · без веб-UI',
      port: '', base: '',
      mem: '9M', cpu: '0.6%', up: '2д 8ч',
      on: true, notify: false, builtin: false, ctl: false, process: ['hysteria'],
      actions: [],
      fields: [
        { path: 'pid', label: 'PID', tile: true, value: '7204' },
        { path: 'rss', label: 'RSS', tile: true, value: '9M' }
      ]
    },
    {
      id: 'sync', name: 'syncFOR', state: 'running', color: '#22c55e', label: 'работает',
      detail: 'base :7077 · process syncfor · probe / · OK',
      port: ':7077', base: 'http://127.0.0.1:7077',
      mem: '28M', cpu: '1.2%', up: '1д 3ч',
      on: true, notify: false, builtin: false, ctl: false, process: ['syncfor'],
      actions: [{ id: 'open', label: 'Открыть' }],
      fields: [
        { path: 'overall_stage', label: 'overall stage', tile: true, value: 'idle' },
        { path: 'version', label: 'version', tile: true, value: '1.4.2' },
        { path: 'scheduler.BAKKK.enabled', label: 'enabled', type: 'bool', tile: true, on: 'да', off: 'нет', value: true },
        { path: 'scheduler.backup_mirror.running', label: 'running', type: 'bool', tile: true, on: 'да', off: 'нет', value: false },
        { path: 'scheduler.backup_mirror.stage', label: 'stage', tile: true, value: 'done' },
        { path: 'scheduler.backup_mirror.last_duration', label: 'last duration', type: 'ms', tile: true, value: 8848 },
        { path: 'time', label: 'time', value: '2026-09-06 23:10' },
        { path: 'scheduler.backup_mirror.current_path', label: 'current path', value: '—' }
      ]
    },
    {
      id: 'netdata', name: 'Netdata', state: 'running', color: '#22c55e', label: 'работает',
      detail: 'base :19999 · prefs enabled',
      port: ':19999', base: 'http://127.0.0.1:19999',
      mem: '48M', cpu: '2.1%', up: '12д 4ч',
      on: true, notify: false, builtin: false, ctl: false,
      actions: [{ id: 'open', label: 'Открыть' }],
      fields: [
        { path: 'alarms', label: 'Алармы', tile: true, value: '0' },
        { path: 'charts', label: 'Charts', tile: true, value: '214' }
      ]
    },
    {
      id: 'transmission', name: 'Transmission', state: 'disabled', color: '#7d8fb0', label: 'выключен',
      detail: 'base :8090 · ports 8090/9091 · prefs off',
      port: ':8090', base: 'http://127.0.0.1:8090',
      mem: '—', cpu: '—', up: '—',
      on: false, notify: false, builtin: true, ctl: true,
      actions: [
        { id: 'open', label: 'Открыть' },
        { id: 'torrents_start', label: 'Запустить все торренты', confirm: true },
        { id: 'torrents_stop', label: 'Остановить все торренты', confirm: true }
      ],
      fields: [
        { path: 'arguments.torrentCount', label: 'Всего торрентов', tile: true, value: 12 },
        { path: 'arguments.activeTorrentCount', label: 'Активные', tile: true, value: 4 },
        { path: 'arguments.downloadSpeed', label: 'Загрузка', type: 'kbs', tile: true, color: '#8b5cf6', value: 0 },
        { path: 'arguments.uploadSpeed', label: 'Отдача', type: 'kbs', tile: true, color: '#38a169', value: 0 },
        { path: 'arguments.pausedTorrentCount', label: 'На паузе', value: 3 },
        { path: 'arguments.cumulative-stats.downloadedBytes', label: 'Скачано за всё время', type: 'bytes', value: 48318382080 },
        { path: 'arguments.cumulative-stats.uploadedBytes', label: 'Отдано за всё время', type: 'bytes', value: 21474836480 }
      ]
    },
    {
      id: 'koffe1', name: 'koffe-servis', state: 'disabled', color: '#7d8fb0', label: 'выключен',
      detail: 'process: koffe-api · failover · health · stats · telegram',
      port: '', base: '',
      mem: '—', cpu: '—', up: '—',
      on: false, notify: false, builtin: false, ctl: false,
      process: ['koffe-api', 'koffe-failover', 'koffe-server-health', 'koffe-stats', 'koffe-telegram'],
      actions: [],
      fields: []
    },
    {
      id: 'grdp-proxy', name: 'grdp-proxy', state: 'disabled', color: '#7d8fb0', label: 'выключен',
      detail: 'process + init: grdp-proxy · prefs off',
      port: '', base: '',
      mem: '—', cpu: '—', up: '—',
      on: false, notify: false, builtin: false, ctl: false, init: 'grdp-proxy', process: ['grdp-proxy'],
      actions: [],
      fields: []
    },
    {
      id: 'entware-bridge', name: 'entware-bridge', state: 'disabled', color: '#7d8fb0', label: 'выключен',
      detail: 'process: bridge · server · telegram · prefs off',
      port: '', base: '',
      mem: '—', cpu: '—', up: '—',
      on: false, notify: false, builtin: false, ctl: false,
      process: ['entware-bridge', 'entware-server', 'entware-telegram'],
      actions: [],
      fields: []
    },
    {
      id: 'ttyd', name: 'ttyd', state: 'disabled', color: '#7d8fb0', label: 'выключен',
      detail: 'терминал :9089 · prefs off · notify on',
      port: ':9089', base: 'http://127.0.0.1:9089',
      mem: '—', cpu: '—', up: '—',
      on: false, notify: true, builtin: true, ctl: false,
      actions: [{ id: 'open', label: 'Открыть' }],
      fields: []
    }
  ];

  window.__MODS_REF = MODS;

  /* порядок карточек модулей (localStorage) */
  function applyModOrder() {
    try {
      var raw = lsGet('entware_mod_order');
      if (!raw) return;
      var order = JSON.parse(raw);
      if (!Array.isArray(order) || !order.length) return;
      var map = {};
      MODS.forEach(function (m) { map[m.id] = m; });
      var next = [];
      order.forEach(function (id) {
        if (map[id]) { next.push(map[id]); delete map[id]; }
      });
      Object.keys(map).forEach(function (id) { next.push(map[id]); });
      MODS.length = 0;
      next.forEach(function (m) { MODS.push(m); });
      window.__MODS_REF = MODS;
    } catch (e) {}
  }
  function saveModOrder() {
    try {
      lsSet('entware_mod_order', JSON.stringify(MODS.map(function (m) { return m.id; })));
    } catch (e) {}
  }
  applyModOrder();

  function esc(s) { return String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;'); }
  function modCard(m) {
    var live = m.on ? (m.state === 'degraded' ? 'warn' : 'ok') : '';
    return '<div class="card mod mod-mini' + (m.on ? '' : ' off') + '" data-mid="' + esc(m.id) + '">' +
      '<div class="mod-mini-row">' +
        '<span class="dot ' + live + '" title="' + esc(m.label) + '"></span>' +
        '<b class="mod-mini-name">' + esc(m.name) + '</b>' +
        '<span class="tag mono">' + esc(m.id) + '</span>' +
        '<label class="tgl" title="Включить/выключить модуль"><input type="checkbox" data-mtgl="' + esc(m.id) + '"' + (m.on ? ' checked' : '') + '><i></i></label>' +
      '</div>' +
      '<div class="mod-mini-flags">' +
        '<label class="chk"><input type="checkbox" data-mnotify="' + esc(m.id) + '"' + (m.notify ? ' checked' : '') + '> уведомления</label>' +
        '<label class="chk"><input type="checkbox" data-mctl="' + esc(m.id) + '"' + (m.ctl ? ' checked' : '') + '> управление</label>' +
      '</div>' +
    '</div>';
  }
  var modGrid = document.getElementById('modGrid');
  function renderMods(filter) {
    var q = (filter || '').toLowerCase(), n = 0, on = 0;
    var html = '';
    MODS.forEach(function (m) {
      if (q && (m.name.toLowerCase().indexOf(q) < 0 && m.id.indexOf(q) < 0)) return;
      n++; if (m.on) on++;
      html += modCard(m);
    });
    modGrid.innerHTML = html || '<div class="card"><div class="mod-detail">Ничего не найдено.</div></div>';
    document.getElementById('modCount').textContent = n + ' модулей · ' + on + ' вкл';
  }
  renderMods('');
  document.getElementById('modSearch').oninput = function (e) { renderMods(e.target.value); };
  modGrid.onclick = function (e) {
    var t = e.target;
    if (t.hasAttribute('data-mtgl')) {
      var card = t.closest('.mod');
      var on = t.checked;
      card.classList.toggle('off', !on);
      var mid = card.getAttribute('data-mid');
      for (var i = 0; i < MODS.length; i++) {
        if (MODS[i].id === mid) {
          MODS[i].on = on;
          MODS[i].state = on ? 'running' : 'disabled';
          MODS[i].label = on ? 'работает' : 'выключен';
          MODS[i].color = on ? '#22c55e' : '#7d8fb0';
        }
      }
      var dot = card.querySelector('.dot');
      if (dot) {
        dot.className = 'dot ' + (on ? 'ok' : '');
        dot.title = on ? 'работает' : 'выключен';
      }
      renderStatMods();
      toast('Модуль «' + mid + '»: ' + (on ? 'включён (мок)' : 'выключен (мок)'));
      return;
    }
    if (t.hasAttribute('data-mnotify')) {
      var idn = t.getAttribute('data-mnotify');
      for (var j = 0; j < MODS.length; j++) {
        if (MODS[j].id === idn) MODS[j].notify = t.checked;
      }
      toast('Уведомления «' + idn + '»: ' + (t.checked ? 'вкл' : 'выкл'));
      return;
    }
    if (t.hasAttribute('data-mctl')) {
      var idc = t.getAttribute('data-mctl');
      for (var k = 0; k < MODS.length; k++) {
        if (MODS[k].id === idc) MODS[k].ctl = t.checked;
      }
      renderStatMods();
      toast('Управление «' + idc + '»: ' + (t.checked ? 'вкл' : 'выкл'));
      return;
    }
    if (t.hasAttribute('data-mact')) { toast('«' + t.textContent + '» — заглушка мока'); }
  };
  document.querySelectorAll('.scan-tabs .st').forEach(function (b) {
    b.onclick = function () {
      document.querySelectorAll('.scan-tabs .st').forEach(function (x) { x.classList.remove('active'); });
      b.classList.add('active');
      toast('Вкладка сканера — заглушка мока');
    };
  });
  document.getElementById('scanBtn').onclick = function () {
    var p = document.getElementById('scanPanel');
    p.hidden = !p.hidden;
  };
  document.getElementById('newModBtn').onclick = function () { toast('Редактор модуля — будет в проде, здесь заглушка'); };

  /* --- дашборд: колонка модулей из тех же MODS, что и вкладка Модули --- */
  function dashMet(m) {
    if (m.id === 'adguard') return '12 408 запр. · блок 18% · 4 мс';
    if (m.id === 'transmission') return '↓ 1.2 МБ/с · 4 раздачи';
    if (m.id === 'koffe') return 'socks5 :1080 · failover';
    if (m.id === 'syncthing') return m.on ? '1 узел offline' : 'выключен';
    if (m.id === 'xray-proxy') return '41M · 8.2%';
    return m.detail;
  }
  function applyDashModsVisible() {
    var show = lsGet('entware_noc_show_mods') !== '0';
    var col = document.getElementById('statModCol');
    var tgl = document.getElementById('dashModsToggle');
    if (tgl) tgl.checked = show;
    if (!col) return;
    col.style.display = show ? '' : 'none';
  }

  function formatField(f) {
    if (!f || f.value === undefined || f.value === null) return '—';
    var v = f.value;
    var type = f.type || '';
    if (type === 'bool') {
      return v ? (f.on || 'да') : (f.off || 'нет');
    }
    if (type === 'num') {
      return String(v).replace(/\B(?=(\d{3})+(?!\d))/g, ' ');
    }
    if (type === 'bytes') {
      var n = Number(v);
      if (n >= 1073741824) return (n / 1073741824).toFixed(1) + ' ГБ';
      if (n >= 1048576) return (n / 1048576).toFixed(1) + ' МБ';
      if (n >= 1024) return (n / 1024).toFixed(1) + ' КБ';
      return n + ' Б';
    }
    if (type === 'kbs') {
      var kb = Number(v);
      if (kb >= 1024) return (kb / 1024).toFixed(1) + ' МБ/с';
      return Math.round(kb) + ' КБ/с';
    }
    if (type === 'ms') {
      var ms = Number(v);
      if (ms < 1) return Math.round(ms * 1000) + ' мс';
      return (Math.round(ms * 10) / 10) + ' мс';
    }
    if (type === 'dur') {
      var s = Number(v);
      if (s >= 86400) return Math.floor(s / 86400) + 'д ' + Math.floor((s % 86400) / 3600) + 'ч';
      if (s >= 3600) return Math.floor(s / 3600) + ' ч ' + Math.floor((s % 3600) / 60) + ' мин';
      if (s >= 60) return Math.floor(s / 60) + ' мин';
      return s + ' с';
    }
    if (type === 'count') return String(Array.isArray(v) ? v.length : v);
    if (type === 'top') return String(v);
    return String(v);
  }

  function renderStatMods() {
    var box = document.getElementById('statMods');
    if (!box) return;
    var html = '', on = 0;
    MODS.forEach(function (m) {
      if (m.on) on++;
      if (!m.on) return; /* выключенные только на вкладке Модули */
      var btns = '';
      if (m.actions && m.actions.length) {
        m.actions.forEach(function (a) {
          var lab = typeof a === 'string' ? a : a.label;
          var aid = typeof a === 'string' ? a : (a.id || a.label);
          var conf = (typeof a === 'object' && a.confirm) ? ' data-confirm="1"' : '';
          btns += '<button class="btn sm ghost" data-mact="' + esc(aid) + '"' + conf + '>' + esc(lab) + '</button>';
        });
      }
      if (m.on && m.ctl) {
        btns += '<button class="btn sm ghost" data-mact="stop" data-confirm="1">Стоп</button>';
        btns += '<button class="btn sm ghost" data-mact="restart" data-confirm="1">Рестарт</button>';
      } else if (!m.on && m.ctl) {
        btns += '<button class="btn sm ghost" data-mact="start">Старт</button>';
      }
      var tiles = '';
      /* process metrics always visible when module is process-based or running */
      tiles += '<div class="tile mono"><label>MEM</label><b>' + esc(m.mem) + '</b></div>';
      tiles += '<div class="tile mono"><label>CPU</label><b>' + esc(m.cpu) + '</b></div>';
      tiles += '<div class="tile mono"><label>UPTIME</label><b>' + esc(m.up) + '</b></div>';
      if (m.on && m.fields && m.fields.length) {
        m.fields.forEach(function (f) {
          if (f.tile === false) return;
          /* show tile fields + default-tile (tile true or undefined for key metrics already listed) */
          if (f.tile !== true && f.tile !== undefined) return;
          if (f.tile !== true) return; /* only explicit tile:true from manifest */
          var style = f.color ? ' style="color:' + esc(f.color) + '"' : '';
          tiles += '<div class="tile mono"><label>' + esc(f.label) + '</label><b' + style + '>' + esc(formatField(f)) + '</b></div>';
        });
      }
      var probe = m.on
        ? (m.process ? 'process · ' + (m.process[0] || m.id) : (m.port ? 'probe ' + m.port + ' · OK' : 'probe OK'))
        : 'не отвечает / выключен';
      html +=
        '<div class="card mod' + (m.on ? '' : ' off') + '" data-mid="' + esc(m.id) + '" draggable="true">' +
          '<div class="mod-top">' +
            '<span class="mod-drag" title="Перетащить" draggable="false">⠿</span>' +
            '<span class="dot ok" title="работает"></span>' +
            '<b>' + esc(m.name) + '</b>' +
            '<span class="tag mono">' + esc(m.id) + '</span>' +
          '</div>' +
          '<div class="mod-state" style="color:' + m.color + '">● ' + esc(m.label) + '</div>' +
          '<div class="mod-detail">' + esc(m.on ? m.detail : 'модуль выключен · манифест: /opt/web_entware/bridge/' + m.id + '.json') +
            (m.on && m.fields ? (function () {
              var extra = [];
              m.fields.forEach(function (f) {
                if (f.tile === true) return;
                extra.push(esc(f.label) + ': ' + esc(formatField(f)));
              });
              return extra.length ? '<div class="mod-extra mono mut">' + extra.join(' · ') + '</div>' : '';
            })() : '') +
          '</div>' +
          '<div class="tiles">' + tiles + '</div>' +
          '<svg class="svc-spark spark" data-mid="' + esc(m.id) + '" viewBox="0 0 120 42" preserveAspectRatio="none" aria-hidden="true"></svg>' +
          '<div class="mod-probe mono mut">' + esc(probe) + '</div>' +
          '<div class="mod-btns">' +
            '<label class="chk"><input type="checkbox"' + (m.notify ? ' checked' : '') + ' disabled> уведомления</label>' +
            (m.ctl ? '<label class="chk"><input type="checkbox" checked disabled> управление</label>' : '') +
            btns +
          '</div>' +
        '</div>';
    });
    box.innerHTML = '<div class="svc-list mods-on-stats">' + html + '</div>';
    var cnt = document.getElementById('statModCnt');
    if (cnt) cnt.textContent = on + ' активных · ' + MODS.length + ' всего';
    applyDashModsVisible();
    paintModSparks();
    bindModDrag(box);
  }

  /* DnD reorder module cards */
  var modDragId = null;
  function bindModDrag(box) {
    if (!box || box._modDragBound) return;
    box._modDragBound = true;
    box.addEventListener('dragstart', function (e) {
      var card = e.target.closest && e.target.closest('.mod[data-mid]');
      if (!card || !box.contains(card)) return;
      /* don't start drag from interactive controls */
      if (e.target.closest('button, a, input, label.tgl, label.chk, .mod-btns')) {
        e.preventDefault();
        return;
      }
      modDragId = card.getAttribute('data-mid');
      card.classList.add('dragging');
      e.dataTransfer.effectAllowed = 'move';
      try { e.dataTransfer.setData('text/plain', modDragId); } catch (err) {}
    });
    box.addEventListener('dragend', function (e) {
      var card = e.target.closest && e.target.closest('.mod');
      if (card) card.classList.remove('dragging');
      Array.prototype.forEach.call(box.querySelectorAll('.mod'), function (c) {
        c.classList.remove('drag-over');
      });
      modDragId = null;
    });
    box.addEventListener('dragover', function (e) {
      e.preventDefault();
      e.dataTransfer.dropEffect = 'move';
      var over = e.target.closest && e.target.closest('.mod[data-mid]');
      Array.prototype.forEach.call(box.querySelectorAll('.mod'), function (c) {
        c.classList.toggle('drag-over', !!(over && c === over && c.getAttribute('data-mid') !== modDragId));
      });
    });
    box.addEventListener('dragleave', function (e) {
      var card = e.target.closest && e.target.closest('.mod');
      if (card && !card.contains(e.relatedTarget)) card.classList.remove('drag-over');
    });
    box.addEventListener('drop', function (e) {
      e.preventDefault();
      var over = e.target.closest && e.target.closest('.mod[data-mid]');
      Array.prototype.forEach.call(box.querySelectorAll('.mod'), function (c) {
        c.classList.remove('drag-over', 'dragging');
      });
      if (!over || !modDragId) return;
      var toId = over.getAttribute('data-mid');
      if (!toId || toId === modDragId) return;
      var from = -1, to = -1, i;
      for (i = 0; i < MODS.length; i++) {
        if (MODS[i].id === modDragId) from = i;
        if (MODS[i].id === toId) to = i;
      }
      if (from < 0 || to < 0 || from === to) return;
      var item = MODS.splice(from, 1)[0];
      MODS.splice(to, 0, item);
      saveModOrder();
      renderStatMods();
      renderMods(document.getElementById('modSearch') ? document.getElementById('modSearch').value : '');
      toast('Порядок модулей сохранён');
    });
  }

  function setModOn(id, on) {
    for (var i = 0; i < MODS.length; i++) {
      if (MODS[i].id === id) {
        MODS[i].on = on;
        MODS[i].state = on ? 'running' : 'disabled';
        MODS[i].label = on ? 'работает' : 'выключен';
        MODS[i].color = on ? '#22c55e' : '#7d8fb0';
      }
    }
    renderMods(document.getElementById('modSearch').value);
    renderStatMods();
  }
  renderStatMods();
  var dashTgl = document.getElementById('dashModsToggle');
  if (dashTgl) {
    dashTgl.onclick = function () {
      lsSet('entware_noc_show_mods', dashTgl.checked ? '1' : '0');
      applyDashModsVisible();
      toast(dashTgl.checked ? 'Блок модулей показан' : 'Блок модулей скрыт');
    };
  }
  document.getElementById('statMods').onclick = function (e) {
    var t = e.target;
    if (t.hasAttribute && t.hasAttribute('data-dmod')) {
      setModOn(t.getAttribute('data-dmod'), t.checked);
      toast('Модуль «' + t.getAttribute('data-dmod') + '»: ' + (t.checked ? 'включён (мок)' : 'выключен (мок)'));
      return;
    }
    if (t.hasAttribute && t.hasAttribute('data-mact')) {
      var conf = t.getAttribute('data-confirm') === '1';
      toast((conf ? '⚠ confirm · ' : '') + '«' + t.textContent.trim() + '» — заглушка мока');
    }
  };
  function goTab(tab) {
    var b = document.querySelector('.ri[data-tab="' + tab + '"]');
    if (b) b.click();
  }
  document.getElementById('statModsManage').onclick = function () { goTab('bridge'); };
  document.getElementById('kpiPkgGo').onclick = function () { goTab('packages'); };
  document.getElementById('pkgUpdateLists').onclick = function () {
    if (!window.__NOC_LIVE_STATS) { toast('Нужна сессия'); return; }
    pkgAct('update.cgi', 'run=1', 'Обновление списков');
  };
  document.getElementById('pkgUpgradeAll').onclick = function () {
    if (!window.__NOC_LIVE_STATS) { toast('Нужна сессия'); return; }
    if (confirm('Обновить все пакеты? Это может занять продолжительное время.')) {
      pkgAct('upgrade.cgi', 'upgrade_all=1', 'Обновление всех пакетов');
    }
  };
  document.getElementById('dashRefresh').onclick = function () {
    jitterMeters();
    renderStatMods();
    toast('Дашборд обновлён (мок)');
  };

  /* --- секции статистики: сворачивание по клику на заголовок --- */
  Array.prototype.forEach.call(document.querySelectorAll('[data-sect]'), function (h) {
    h.onclick = function () { h.parentNode.classList.toggle('closed'); };
  });

  /* --- подвкладки: data-sub группа, кнопки data-sp, панели data-pp --- */
  Array.prototype.forEach.call(document.querySelectorAll('[data-sub]'), function (g) {
    var btns = g.querySelectorAll('[data-sp]');
    Array.prototype.forEach.call(btns, function (b) {
      b.onclick = function () {
        Array.prototype.forEach.call(btns, function (x) { x.classList.remove('active'); });
        b.classList.add('active');
        var key = b.getAttribute('data-sp');
        Array.prototype.forEach.call(g.querySelectorAll('[data-pp]'), function (p) {
          p.hidden = (p.getAttribute('data-pp') !== key);
        });
      };
    });
  });

  /* --- справка: аккордеон + живой поиск + переходы на вкладки --- */
  Array.prototype.forEach.call(document.querySelectorAll('.acc-h'), function (h) {
    h.onclick = function () { h.parentNode.classList.toggle('open'); };
  });
  Array.prototype.forEach.call(document.querySelectorAll('[data-gotab]'), function (a) {
    a.onclick = function (e) { e.preventDefault(); goTab(a.getAttribute('data-gotab')); };
  });
  var helpSearch = document.getElementById('helpSearch');
  if (helpSearch) {
    helpSearch.oninput = function () {
      var q = helpSearch.value.toLowerCase(), n = 0;
      Array.prototype.forEach.call(document.querySelectorAll('.acc-item'), function (it) {
        var hit = it.textContent.toLowerCase().indexOf(q) >= 0;
        it.style.display = hit ? '' : 'none';
        if (hit) n++;
      });
      if (q) {
        Array.prototype.forEach.call(document.querySelectorAll('.acc-item'), function (it) {
          if (it.style.display !== 'none') it.classList.add('open');
        });
      }
    };
  }

  /* --- закладки: выезжающие полоски справа, long-press reorder --- */
  var BM_DEFAULT = [
    { id: 'adguard', ico: '⧉', label: 'AdGuard', meta: ':8080', url: 'http://192.168.3.1:8080' },
    { id: 'trans', ico: '⧉', label: 'Transmission', meta: ':9091', url: 'http://192.168.3.1:9091' },
    { id: 'sync', ico: '⧉', label: 'Syncthing', meta: ':8384', url: 'http://192.168.3.1:8384' },
    { id: 'ttyd', ico: '⧉', label: 'ttyd', meta: ':9089', url: 'http://192.168.3.1:9089' }
  ];
  /* Порядок перетаскивания храним отдельно (массив id) — сами пункты едут
   * с сервера (links_load.cgi, правятся в Настройках), иначе локальная копия
   * замораживала бы серверные правки. */
  function bmOrderLoad() {
    try {
      var p = JSON.parse(lsGet('entware_bookmark_order'));
      if (Array.isArray(p)) return p;
    } catch (e) {}
    return null;
  }
  function bmOrderSave() {
    try { lsSet('entware_bookmark_order', JSON.stringify(bookmarks.map(function (b) { return b.id; }))); } catch (e) {}
  }
  function bmApplyOrder() {
    var ord = bmOrderLoad();
    if (!ord || !ord.length) return;
    var pos = {};
    for (var i = 0; i < ord.length; i++) pos[ord[i]] = i;
    bookmarks.sort(function (a, b) {
      var pa = (a.id in pos) ? pos[a.id] : 1e9;
      var pb = (b.id in pos) ? pos[b.id] : 1e9;
      return pa - pb;
    });
  }
  function bmSlug(s) {
    s = String(s || '').toLowerCase().replace(/[^a-z0-9а-яё]+/gi, '-').replace(/^-+|-+$/g, '');
    return s || 'link';
  }
  /* Серверная ссылка {name,url,icon} → пункт rail. meta — порт или путь. */
  function bmFromServer(list) {
    var out = [];
    list.forEach(function (l) {
      if (!l || !l.url) return;
      var url = String(l.url), meta = '';
      var m = url.match(/:(\d+)(\/|$)/);
      if (m) meta = ':' + m[1];
      else {
        var p = url.replace(/^https?:\/\/[^\/]+/i, '');
        if (p && p !== '/') meta = p.length > 18 ? p.slice(0, 18) + '…' : p;
      }
      var ico = '⧉';
      if (l.icon && String(l.icon).length <= 4) ico = String(l.icon);
      out.push({ id: 'srv-' + bmSlug(url), ico: ico, label: l.name || url, meta: meta, url: url });
    });
    return out;
  }
  function bmRefreshFromServer() {
    if (!window.__NOC_LIVE_STATS || typeof liveFetch !== 'function') return;
    liveFetch('links_load.cgi').then(function (d) {
      if (!Array.isArray(d) || !d.length) return;
      bookmarks = bmFromServer(d);
      bmApplyOrder();
      renderBookmarks();
    });
  }
  function loadBookmarks() {
    var legacy = null;
    try {
      var raw = lsGet('entware_bookmarks');
      if (raw) {
        var parsed = JSON.parse(raw);
        if (Array.isArray(parsed) && parsed.length) legacy = parsed;
      }
    } catch (e) {}
    if (legacy) {
      /* разовая миграция: полный массив → только порядок, ключ удаляем,
       * дальше пункты едут с сервера */
      try {
        lsSet('entware_bookmark_order', JSON.stringify(legacy.map(function (b) { return b.id; })));
        localStorage.removeItem('entware_bookmarks');
      } catch (e) {}
      return legacy;
    }
    return BM_DEFAULT.slice();
  }
  function saveBookmarks() { bmOrderSave(); }
  var bookmarks = loadBookmarks();
  bmApplyOrder();
  var bmRail = document.getElementById('bmRail');
  var bmDrag = { active: false, id: null, timer: null, startY: 0, suppressClick: false };

  function renderBookmarks() {
    if (!bmRail) return;
    var html = '';
    bookmarks.forEach(function (b, idx) {
      html +=
        '<a class="bm" href="' + esc(b.url || '#') + '" target="_blank" rel="noopener" data-id="' + esc(b.id) + '" data-idx="' + idx + '"' +
        ' title="' + esc(b.label + (b.meta ? ' ' + b.meta : '') + ' · долгое нажатие — переставить') + '">' +
        '<span class="bm-ico">' + esc(b.ico || '⧉') + '</span>' +
        '<span class="bm-label">' + esc(b.label) + '</span>' +
        (b.meta ? '<span class="bm-meta">' + esc(b.meta) + '</span>' : '') +
        '<span class="bm-handle" aria-hidden="true"></span></a>';
    });
    bmRail.innerHTML = html;
    if (bmDrag.active && bmDrag.id) {
      var still = bmRail.querySelector('.bm[data-id="' + bmDrag.id + '"]');
      if (still) still.classList.add('dragging', 'open');
    }
  }

  function bmPoint(e) {
    if (e.touches && e.touches[0]) return { x: e.touches[0].clientX, y: e.touches[0].clientY };
    return { x: e.clientX, y: e.clientY };
  }

  function bmClearTimer() {
    if (bmDrag.timer) { clearTimeout(bmDrag.timer); bmDrag.timer = null; }
  }

  function bmEndDrag() {
    bmClearTimer();
    if (!bmDrag.active) return;
    bmDrag.active = false;
    bmDrag.id = null;
    Array.prototype.forEach.call(bmRail.querySelectorAll('.bm'), function (x) {
      x.classList.remove('dragging', 'drag-over', 'open');
    });
    toast('Порядок закладок сохранён');
  }

  if (bmRail) {
    bmRail.addEventListener('click', function (e) {
      var a = e.target.closest('.bm');
      if (!a) return;
      if (bmDrag.suppressClick) {
        bmDrag.suppressClick = false;
        e.preventDefault();
        return;
      }
      var id = a.getAttribute('data-id');
      var b = null;
      for (var i = 0; i < bookmarks.length; i++) if (bookmarks[i].id === id) b = bookmarks[i];
      var url = b ? b.url : a.getAttribute('href');
      if (!url || url === '#') { e.preventDefault(); toast('Нет адреса', 'warn'); return; }
      e.preventDefault();
      window.open(url, '_blank', 'noopener');
    });

    bmRail.addEventListener('mousedown', function (e) {
      if (e.button !== 0) return;
      var a = e.target.closest('.bm');
      if (!a) return;
      bmDrag.suppressClick = false;
      bmDrag.startY = e.clientY;
      bmClearTimer();
      bmDrag.timer = setTimeout(function () {
        bmDrag.active = true;
        bmDrag.id = a.getAttribute('data-id');
        bmDrag.suppressClick = true;
        a.classList.add('dragging', 'open');
        toast('Перетащите закладку');
      }, 380);
    });

    bmRail.addEventListener('touchstart', function (e) {
      var a = e.target.closest('.bm');
      if (!a) return;
      var p = bmPoint(e);
      bmDrag.suppressClick = false;
      bmDrag.startY = p.y;
      bmClearTimer();
      bmDrag.timer = setTimeout(function () {
        bmDrag.active = true;
        bmDrag.id = a.getAttribute('data-id');
        bmDrag.suppressClick = true;
        a.classList.add('dragging', 'open');
        toast('Перетащите закладку');
      }, 380);
    }, { passive: true });

    window.addEventListener('mousemove', function (e) {
      if (!bmDrag.timer && !bmDrag.active) return;
      if (!bmDrag.active) {
        if (Math.abs(e.clientY - bmDrag.startY) > 8) bmClearTimer();
        return;
      }
      e.preventDefault();
      var overEl = document.elementFromPoint(e.clientX, e.clientY);
      var over = overEl && overEl.closest ? overEl.closest('.bm') : null;
      Array.prototype.forEach.call(bmRail.querySelectorAll('.bm'), function (x) {
        x.classList.toggle('drag-over', !!(over && x.getAttribute('data-id') !== bmDrag.id && x === over));
      });
      if (!over) return;
      var fromId = bmDrag.id;
      var toId = over.getAttribute('data-id');
      if (!fromId || fromId === toId) return;
      var from = -1, to = -1, i;
      for (i = 0; i < bookmarks.length; i++) {
        if (bookmarks[i].id === fromId) from = i;
        if (bookmarks[i].id === toId) to = i;
      }
      if (from < 0 || to < 0 || from === to) return;
      var item = bookmarks.splice(from, 1)[0];
      bookmarks.splice(to, 0, item);
      saveBookmarks(bookmarks);
      renderBookmarks();
    });

    window.addEventListener('touchmove', function (e) {
      if (!bmDrag.timer && !bmDrag.active) return;
      var p = bmPoint(e);
      if (!bmDrag.active) {
        if (Math.abs(p.y - bmDrag.startY) > 8) bmClearTimer();
        return;
      }
      e.preventDefault();
      var overEl = document.elementFromPoint(p.x, p.y);
      var over = overEl && overEl.closest ? overEl.closest('.bm') : null;
      Array.prototype.forEach.call(bmRail.querySelectorAll('.bm'), function (x) {
        x.classList.toggle('drag-over', !!(over && x.getAttribute('data-id') !== bmDrag.id && x === over));
      });
      if (!over) return;
      var fromId = bmDrag.id;
      var toId = over.getAttribute('data-id');
      if (!fromId || fromId === toId) return;
      var from = -1, to = -1, i;
      for (i = 0; i < bookmarks.length; i++) {
        if (bookmarks[i].id === fromId) from = i;
        if (bookmarks[i].id === toId) to = i;
      }
      if (from < 0 || to < 0 || from === to) return;
      var item = bookmarks.splice(from, 1)[0];
      bookmarks.splice(to, 0, item);
      saveBookmarks(bookmarks);
      renderBookmarks();
    }, { passive: false });

    window.addEventListener('mouseup', function () {
      bmClearTimer();
      if (bmDrag.active) bmEndDrag();
    });
    window.addEventListener('touchend', function () {
      bmClearTimer();
      if (bmDrag.active) bmEndDrag();
    });
  }

  renderBookmarks();

  window.toast = toast; // инлайн-обработчики onclick в HTML зовут toast()

  /* ===== LIVE (experiment branch): живые CGI через same-origin прокси /entware-cgi/ → :8087.
   * Мок (jitter) — фолбэк без сессии. Живое: топы, CPU%, температуры, сеть, обновления,
   * версия, health. Мок-only (нет JSON-источника на панели): RAM/диск/WAN-скорость/аптайм/тома. */
  var LIVE_BASE = '/entware-cgi/';
  var LIVE_EXP_BASE = '/entware-cgi/monitor/';
  var LIVE_TICK_MS = 7000;
  var LIVE_TIMEOUT_MS = 12000;
  var livePending = 0;
  var liveTickN = 0;
  var liveTickErr = [];
  function liveGo(name, promise, apply) {
    livePending++;
    promise.then(function (d) {
      try {
        if (d === null || d === undefined) { liveTickErr.push(name); return; }
        apply(d);
      } catch (e) {
        liveTickErr.push(name + ':js');
      } finally {
        livePending--;
      }
    });
  }
  function liveFetch(name, opts, timeoutMs) { return liveFetchRaw(LIVE_BASE + name, opts, timeoutMs); }
  function liveFetchExp(name, opts) { return liveFetchRaw(LIVE_EXP_BASE + name, opts); }
  function liveFetchRaw(url, opts, timeoutMs) {
    if (typeof AbortController !== 'undefined') {
      var ctl = new AbortController();
      var timer = setTimeout(function () { ctl.abort(); }, timeoutMs || LIVE_TIMEOUT_MS);
      return fetch(url, opts).then(function (r) {
        clearTimeout(timer);
        var ct = r.headers.get('content-type') || '';
        if (ct.indexOf('application/json') < 0) return r.text();
        return r.json();
      }).catch(function () { clearTimeout(timer); return null; });
    }
    return fetch(url, opts).then(function (r) {
      var ct = r.headers.get('content-type') || '';
      if (ct.indexOf('application/json') < 0) return r.text();
      return r.json();
    }).catch(function () { return null; });
  }
  function liveNum(v) {
    if (v === null || v === undefined) return NaN;
    var n = parseFloat(String(v).replace(',', '.'));
    return isNaN(n) ? NaN : n;
  }
  function liveShort(cmd, n) {
    cmd = String(cmd || '—');
    return cmd.length > (n || 40) ? cmd.slice(0, n || 40) + '…' : cmd;
  }
  function liveRow(cells, cls) {
    return '<tr>' + cells.map(function (c, i) {
      return '<td class="' + (i < 2 ? 'l' : '') + (i === 0 ? ' mut' : '') + '">' + c + '</td>';
    }).join('') + '</tr>';
  }
  var liveMonProcs = [];
  var liveMemProcs = [];
  var liveFullProcs = [];
  function rssFmt(kb) {
    kb = Math.round(liveNum(kb));
    if (isNaN(kb) || kb <= 0) return '—';
    if (kb >= 1024) return Math.round(kb / 1024) + 'M';
    return kb + 'K';
  }
  function applyFullProcs(d) {
    /* полный слепок из своего procs.cgi: нулевые пропуски */
    if (!d || !Array.isArray(d.processes) || !d.processes.length) return;
    liveFullProcs = d.processes;
    renderTops();
    procRender();
    /* CPU%: то же окно, что и топы (busy = 100 − idle − iowait, как панель) */
    if (d.cpu && typeof d.cpu === 'object') {
      var u = liveNum(d.cpu.user), s = liveNum(d.cpu.system),
          w = liveNum(d.cpu.iowait), id = liveNum(d.cpu.idle);
      if (!isNaN(u) && !isNaN(s) && !isNaN(w) && !isNaN(id)) {
        cpu = Math.max(0, Math.min(100, Math.round(100 - id - w)));
        pushSeries('cpu', cpu);
        var cf = document.querySelector('[data-metric="cpu"] .res-f');
        if (cf) {
          cf.textContent = 'usr ' + u + ' · sys ' + s + ' · io ' + w +
            ' · irq ' + Math.max(0, Math.round(100 - id - w - u - s));
        }
        paintMeters();
      }
    }
    /* системные данные из того же слепка: ядра, loadavg, память */
    var cc = parseInt(d.cpu_count, 10);
    if (cc > 0) {
      var ch = document.querySelector('[data-metric="cpu"] .res-h .mut');
      if (ch) ch.textContent = cc + ' cores';
    }
    if (d.loadavg) {
      var lf = document.querySelector('[data-metric="cpu"] .res-f');
      if (lf) lf.textContent = 'load ' + d.loadavg;
    }
    var mt = parseInt(d.mem_total_kb, 10), ma = parseInt(d.mem_avail_kb, 10);
    if (mt > 0 && ma >= 0) {
      var usedMb = Math.round((mt - ma) / 1024), totMb = Math.round(mt / 1024);
      var swMb = Math.round(liveNum(d.mem_swap_kb) / 1024);
      if (!isNaN(swMb)) window.__NOC_SWAP = swMb;
      ram = Math.max(0, Math.min(100, Math.round((mt - ma) * 100 / mt)));
      pushSeries('ram', ram);
      var rh = document.querySelector('[data-metric="ram"] .res-h .mut');
      if (rh) rh.textContent = totMb + ' МБ';
      var rf = document.querySelector('[data-metric="ram"] .res-f');
      if (rf) rf.textContent = 'used ' + usedMb + ' · free ' + (totMb - usedMb) + ' · swap ' + (isNaN(swMb) ? '—' : swMb);
      paintMeters();
    }
  }
  function procKey(s) {
    /* join-ключ: basename argv[0] нижним регистром */
    s = String(s || '').trim();
    if (!s) return '';
    return s.split(' ')[0].split('/').pop().toLowerCase().replace(/^\(|\)$/g, '');
  }
  function renderTops() {
    /* CPU-топ: порядок по %CPU (monitor) + RSS из мем-топа; мем-топ: порядок по RSS + PID/%CPU/время из monitor */
    var cb = document.getElementById('topCpuBody');
    var mb = document.getElementById('topMemBody');
    if (liveFullProcs.length) {
      /* полные данные из procs.cgi — пропусков нет (кроме RSS=0 у ядерных потоков) */
      var byCpu = liveFullProcs.slice().sort(function (a, b) { return (liveNum(b.pcpu) || 0) - (liveNum(a.pcpu) || 0); });
      var byMem = liveFullProcs.slice().sort(function (a, b) { return (liveNum(b.rss_kb) || 0) - (liveNum(a.rss_kb) || 0); });
      if (cb) {
        var html = '';
        byCpu.slice(0, 7).forEach(function (p) {
          html += liveRow([esc(p.pid), esc(liveShort(p.command)), esc(p.pcpu) + '%', rssFmt(p.rss_kb), esc(p.time || '—')]);
        });
        cb.innerHTML = html;
      }
      if (mb) {
        var html2 = '';
        byMem.slice(0, 7).forEach(function (m) {
          html2 += liveRow([esc(m.pid), esc(liveShort(m.command)), rssFmt(m.rss_kb), esc(m.pcpu) + '%', esc(m.time || '—')]);
        });
        mb.innerHTML = html2;
      }
      return;
    }
    /* фолбэк без полного слепка: только если procs.cgi молчит несколько тиков */
    if (liveTickN < 3) return;
    var memByKey = {};
    liveMemProcs.forEach(function (m) { if (!memByKey[procKey(m.cmd)]) memByKey[procKey(m.cmd)] = m; });
    var monByKey = {};
    liveMonProcs.forEach(function (p) { if (!monByKey[procKey(p.command)]) monByKey[procKey(p.command)] = p; });
    if (cb && liveMonProcs.length) {
      var html = '';
      liveMonProcs.slice(0, 7).forEach(function (p) {
        var mm = memByKey[procKey(p.command)];
        var pcpu = p.pcpu !== undefined ? p.pcpu + '%' : '—';
        html += liveRow([esc(p.pid), esc(liveShort(p.command)), pcpu, esc(mm ? mm.rss : '—'), esc(p.time || '—')]);
      });
      cb.innerHTML = html;
    }
    if (mb && liveMemProcs.length) {
      var html2 = '';
      liveMemProcs.slice(0, 7).forEach(function (m) {
        var mp = monByKey[procKey(m.cmd)];
        var pcpu2 = mp && mp.pcpu !== undefined ? mp.pcpu + '%' : '—';
        html2 += liveRow([esc(mp ? mp.pid : '—'), esc(liveShort(m.cmd)), esc(m.rss), pcpu2, esc((mp && mp.time) || '—')]);
      });
      mb.innerHTML = html2;
    }
  }
  function applyProcs(d) {
    if (!d || !d.processes || !d.processes.length) return;
    liveMonProcs = d.processes.slice(0, 7);
    renderTops();
    monBadge(d);
    monTop(d);
    var pill = document.querySelector('.health-pill');
    if (pill) {
      var ok = d.daemon_status === 'running';
      pill.className = 'health-pill ' + (ok ? 'ok' : 'warn');
      pill.innerHTML = '<i class="live-dot"></i>' + (ok ? 'live' : 'degraded');
    }
  }
  function applyTemp(d) {
    var v = d ? liveNum(d.temperature) : NaN;
    if (isNaN(v)) return;
    temp = v;
    pushSeries('temp', temp);
    var tv = document.getElementById('tempVal');
    if (tv) tv.textContent = Math.round(v) + '°';
    paintMeters();
  }
  function applyWifiTemp(d) {
    if (!d || !d.combined || d.combined === 'null') return;
    var f = document.querySelector('[data-metric="temp"] .res-f');
    if (f) f.textContent = String(d.combined);
  }
  function applyNet(d) {
    if (!d || !d.interfaces) return;
    var html = '', seen = {}, n = 0, k;
    if (!window.__NOC_LANIP && d.interfaces) {
      for (var j = 0; j < d.interfaces.length; j++) {
        var ip = d.interfaces[j].ip || '';
        if (d.interfaces[j].iface === 'br0' && ip && ip !== '--') { window.__NOC_LANIP = ip; break; }
        if (!window.__NOC_LANIP && ip && ip !== '--' &&
            (/^(192\.168\.|10\.|172\.(1[6-9]|2\d|3[01])\.)/.test(ip))) window.__NOC_LANIP = ip;
      }
    }
    for (var k = 0; k < d.interfaces.length && n < 8; k++) {
      var i = d.interfaces[k];
      if (seen[i.iface]) continue; /* один адрес на интерфейс — дубли скрываем */
      seen[i.iface] = 1;
      n++;
      html += '<tr><td class="l">' + esc(i.iface) + '</td><td class="ok-t">UP</td>' +
        '<td class="l mut">' + esc(i.ip) + '</td><td>—</td><td>—</td></tr>';
    }
    var ib = document.getElementById('ifBody');
    if (ib && html) ib.innerHTML = html;
    var ic = document.getElementById('ifCnt');
    if (ic) ic.textContent = n + ' up';
    if (d.wan) {
      var w = document.querySelector('[data-metric="wan"] .res-h span');
      if (w) { w.textContent = String(d.wan); w.className = String(d.wan).indexOf('up') >= 0 ? 'ok-t' : 'warn-t'; }
      var wf = document.querySelector('[data-metric="wan"] .res-f');
      if (wf) wf.textContent = String(d.wan);
      var kw2 = document.getElementById('kpiWan');
      if (kw2) kw2.textContent = '—';
      var sbp = document.getElementById('sbPpp');
      if (sbp) sbp.innerHTML = '<i class="dot ' + (String(d.wan).indexOf('up') >= 0 ? 'ok' : 'warn') + '"></i> ' + esc(String(d.wan));
    }
  }
  function applyUpdateCheck(d) {
    if (!d) return;
    var vb = document.getElementById('verBadge');
    if (vb && d.current) {
      var sp = vb.querySelectorAll('span');
      if (sp.length) sp[0].textContent = 'v' + d.current;
      var up = vb.querySelector('i.upd');
      if (up) {
        if (d.has_update && d.latest) { up.textContent = '→ v' + d.latest; up.style.display = ''; }
        else { up.style.display = 'none'; }
      }
    }
  }
  function applyUpgradable(a) {
    var badge = document.getElementById('updBadge');
    var box = document.getElementById('updList');
    if (!box) return;
    if (!Array.isArray(a)) {
      if (badge) { badge.textContent = '?'; badge.className = 'badge'; }
      liveTickErr.push('upgradable:' + ((a && a.error) || 'bad-shape'));
      return; /* объект ошибки — не «всё обновлено», оставляем мок */
    }
    if (!a.length) {
      if (badge) { badge.textContent = '0'; badge.className = 'badge ok'; }
      var sb0 = document.getElementById('sbPkg');
      if (sb0) sb0.innerHTML = '<i class="dot ok"></i> пакеты в порядке';
      box.innerHTML = '<div class="upd"><b>всё обновлено</b></div>';
      return;
    }
    if (badge) { badge.textContent = String(a.length); badge.className = 'badge warn'; }
    var sb = document.getElementById('sbPkg');
    if (sb) sb.innerHTML = '<i class="dot warn"></i> ' + a.length + ' ' + plural(a.length, 'пакет', 'пакета', 'пакетов') + ' к обновлению';
    var html = '';
    a.slice(0, 5).forEach(function (p) {
      html += '<div class="upd"><b>' + esc(p.package) + '</b><span class="mut">' + esc(p.current) +
        '</span><span class="arrow">→</span><span class="warn-t">' + esc(p.new) + '</span></div>';
    });
    box.innerHTML = html;
  }
  function applyVersion(d) {
    if (!d || !d.version) return;
    var vb = document.getElementById('verBadge');
    if (vb) {
      var sp = vb.querySelectorAll('span');
      if (sp.length) sp[0].textContent = 'v' + d.version;
    }
  }
  var evtFail = 0, evtData = [], evtFilter = 'all';
  /* Кэш уже загружаемых источников для фильтров без новых запросов:
   * monLogCache — текст monitor/log.cgi (обновляется каждый тик),
   * pkgLogCache — массив packages_log.cgi (лениво, один раз). */
  var monLogCache = '', pkgLogCache = [], pkgLogLoading = false;
  try { evtFilter = localStorage.getItem('noc_evtf') || 'all'; } catch (e) {}
  /* Кнопки карточки Событий → источники журнала (Service из tag=all) */
  var EVT_GROUPS = {
    net: ['network'], svc: ['service', 'crontab_update'], mon: ['monitor'],
    auth: ['login'], pkg: ['update', 'packages']
  };
  function evtTs(ts) {
    ts = String(ts || '');
    /* свежие — ЧЧ:ММ, старые — ДД.ММ ЧЧ:ММ */
    if (/^\d{4}-\d{2}-\d{2} /.test(ts)) {
      var today = new Date();
      var todayS = today.getFullYear() + '-' + ('0' + (today.getMonth() + 1)).slice(-2) + '-' + ('0' + today.getDate()).slice(-2);
      ts = ts.indexOf(todayS) === 0 ? ts.slice(11, 16) : ts.slice(8, 10) + '.' + ts.slice(5, 7) + ' ' + ts.slice(11, 16);
    } else if (ts.length > 16) ts = ts.slice(11, 16);
    return ts;
  }
  function renderEvents() {
    var box = document.getElementById('evtList');
    if (!box) return;
    /* Пакеты живут в отдельном файле (package_changes.log) и в network_events
     * не попадают — рисуем из уже загруженного кэша packages_log.cgi. */
    if (evtFilter === 'pkg') {
      if (!pkgLogCache.length) {
        if (window.__NOC_LIVE_STATS && !pkgLogLoading) pkgJournalLoad();
        box.innerHTML = '<div class="evt mut">' + (window.__NOC_LIVE_STATS ? 'Загрузка…' : 'событий нет') + '</div>';
        return;
      }
      var html = '';
      pkgLogCache.slice(0, 6).forEach(function (e) {
        var ok = String(e.status || '') === 'success';
        var actRu = PKG_ACT_RU[String(e.action || '')] || String(e.action || '?');
        html += '<div class="evt"><span class="ts">' + esc(evtTs(e.timestamp)) + '</span>' +
          '<span class="badge ' + (ok ? 'ok' : 'crit') + '">' + (ok ? '✓' : '✗') + '</span><span>' +
          esc(e.package || '?') + ' — ' + esc(actRu) + '</span></div>';
      });
      box.innerHTML = html;
      return;
    }
    /* Защита: newest-30 окна events не хватает (сплошной [network]) —
     * рисуем из текста monitor/log.cgi (до 200 строк [monitor]). */
    if (evtFilter === 'mon') {
      var lines = monLogCache.split('\n').filter(function (s) { return s.replace(/^\s+|\s+$/g, ''); });
      if (!lines.length) { box.innerHTML = '<div class="evt mut">' + (evtData.length ? 'нет записей' : 'событий нет') + '</div>'; return; }
      var html2 = '';
      lines.slice(0, 6).forEach(function (ln) {
        var m = ln.match(/^\[(\d{4}-\d{2}-\d{2} \d{2}:\d{2})/);
        var lv = /\[ERROR\]|\[CRIT\]/i.test(ln) ? ['CRIT', 'crit'] : (/\[WARN\]/i.test(ln) ? ['WARN', 'warn'] : ['INFO', 'ok']);
        var txt = ln.replace(/^\s+|\s+$/g, '');
        if (txt.length > 110) txt = txt.slice(0, 110) + '…';
        html2 += '<div class="evt"><span class="ts">' + esc(m ? evtTs(m[1]) : '') + '</span>' +
          '<span class="badge ' + lv[1] + '">' + lv[0] + '</span><span>' + esc(txt) + '</span></div>';
      });
      box.innerHTML = html2;
      return;
    }
    var grp = EVT_GROUPS[evtFilter];
    var list = grp ? evtData.filter(function (e) { return grp.indexOf(String(e.service || '')) >= 0; }) : evtData.slice();
    if (!list.length) { box.innerHTML = '<div class="evt mut">' + (evtData.length ? 'нет записей' : 'событий нет') + '</div>'; return; }
    var html = '';
    list.slice(0, evtFilter === 'all' ? 10 : 6).forEach(function (e) {
      var lv = String(e.level || 'INFO').toUpperCase();
      var cls = lv === 'WARN' ? 'warn' : (lv === 'ERROR' || lv === 'CRIT' ? 'crit' : 'ok');
      html += '<div class="evt"><span class="ts">' + esc(evtTs(e.timestamp)) + '</span>' +
        '<span class="badge ' + cls + '">' + esc(lv) + '</span><span>' +
        esc(e.event || e.details || '') + '</span></div>';
    });
    if (html) box.innerHTML = html;
  }
  function applyEvents(d) {
    var box = document.getElementById('evtList');
    if (!box) return;
    if (!d || !Array.isArray(d.events)) {
      evtFail++;
      if (evtFail >= 3) box.innerHTML = '<div class="evt mut">нет данных</div>';
      return;
    }
    evtFail = 0;
    evtData = d.events;
    renderEvents();
  }
  var monFail = 0;
  function setText(id, t) { var el = document.getElementById(id); if (el) el.textContent = t; }
  function applyMonLog(t) {
    /* monitor/log.cgi — plain text, [monitor] сегодняшнего журнала.
     * Рендерим в таблицу (как «Последние события» на вкладке Службы). */
    if (typeof t !== 'string' || !t.replace(/^\s+|\s+$/g, '')) {
      monFail++;
      if (monFail >= 3) renderMonLogTable([]);
      return;
    }
    monFail = 0;
    monLogCache = t.replace(/^\s+|\s+$/g, '');
    renderMonLogTable(monLogCache.split('\n').filter(function (s) { return s.replace(/^\s+|\s+$/g, ''); }));
    if (evtFilter === 'mon') renderEvents();
  }
  function renderMonLogTable(lines) {
    var tb = document.getElementById('monLogFull');
    if (!tb) return;
    if (!lines.length) { tb.innerHTML = '<tr><td class="l mut">событий нет</td></tr>'; return; }
    var html = '';
    lines.slice(0, 200).forEach(function (ln) {
      var m = ln.match(/^\[(\d{4}-\d{2}-\d{2} \d{2}:\d{2})/);
      var lv = /\[ERROR\]|\[CRIT\]/i.test(ln) ? ['CRIT', 'crit'] : (/\[WARN\]/i.test(ln) ? ['WARN', 'warn'] : ['INFO', 'ok']);
      var txt = ln.replace(/^\s+|\s+$/g, '');
      if (txt.length > 160) txt = txt.slice(0, 160) + '…';
      html += '<tr><td class="l"><span class="ts">' + esc(m ? m[1] : '') + '</span>' +
        ' <span class="badge ' + lv[1] + '">' + lv[0] + '</span>' +
        ' <span>' + esc(txt) + '</span></td></tr>';
    });
    tb.innerHTML = html;
  }
  function applyMemTop(t) {
    /* настоящий топ по RSS из stats.cgi (в JSON-источниках RSS нет) */
    if (typeof t !== 'string') return;
    var sec = t.match(/Топ по памяти<\/th><\/tr>([\s\S]*?)<\/table>/);
    if (!sec) return;
    var re = /<tr><td>([^<]*)<\/td><td>([^<]*)<\/td><\/tr>/g, m, list = [];
    while ((m = re.exec(sec[1])) && list.length < 7) {
      list.push({ cmd: m[1], rss: m[2] });
    }
    if (list.length) { liveMemProcs = list; renderTops(); }
  }
  function sysCell(t, label) {
    var m = t.match(new RegExp(label + ':<\\/td><td>([^<]*)<'));
    return m ? m[1].replace(/^\s+|\s+$/g, '') : '';
  }
  function prettyUp(s) {
    return String(s || '').replace(' day ', 'д ');
  }
  function applySysInfo(t) {
    if (typeof t !== 'string' || !t) return;
    var model = sysCell(t, 'Модель'), host = sysCell(t, 'Имя хоста'),
        arch = sysCell(t, 'Архитектура'), kern = sysCell(t, 'Версия ядра'),
        up = prettyUp(sysCell(t, 'Время работы'));
    if (host) {
      var th = document.getElementById('tbHost');
      if (th) th.textContent = host + (window.__NOC_LANIP ? ' · ' + window.__NOC_LANIP : '');
      var tu = document.getElementById('tbUptime');
      if (tu && up) tu.textContent = 'up ' + up;
    }
    var hs = document.getElementById('heroSub');
    if (hs) {
      var parts = [];
      if (model && model !== 'н/д') parts.push(model);
      else if (host) parts.push(host);
      if (arch && arch !== 'н/д') parts.push(arch);
      if (kern && kern !== 'н/д') parts.push(kern);
      if (up) parts.push('up ' + up);
      if (parts.length) hs.textContent = parts.join(' · ');
    }
  }
  function applyDfVols(t) {
    /* тома из таблиц stats.cgi (tmpfs + блочные); sda1 и т.п. — настоящие */
    if (typeof t !== 'string' || !t) return;
    var re = /<tr><td>([^<]*)<\/td><td>([^<]*)<\/td><td>([^<]*)<\/td><td>([^<]*)<\/td><td><span[^>]*>([^<]*)<\/span><\/td><td><a[^>]*>([^<]*)<\/a>/g;
    var m, vols = [];
    while ((m = re.exec(t))) {
      vols.push({ fs: m[1], size: m[2], used: m[3], avail: m[4], pct: m[5], mount: m[6] });
    }
    if (!vols.length) return;
    var box = document.getElementById('volList');
    if (box) {
      var html = '';
      vols.slice(0, 4).forEach(function (v) {
        var pctN = parseInt(v.pct, 10);
        if (isNaN(pctN)) pctN = 0;
        var kind = v.fs.indexOf('tmpfs') === 0 ? 'tmpfs · RAM' : esc(v.fs) + ' · local';
        html += '<div class="vol"><div class="vol-h"><b>' + esc(v.mount) + '</b><span>' +
          esc(v.used) + ' / ' + esc(v.size) + '</span><span class="mut">' + esc(v.pct) +
          '</span></div><div class="prog"><i style="width:' + pctN + '%"></i></div>' +
          '<div class="vol-f mut">' + kind + '</div></div>';
      });
      box.innerHTML = html;
    }
    for (var i = 0; i < vols.length; i++) {
      if (vols[i].mount === '/opt') {
        var df = document.querySelector('[data-metric="disk"] .res-f');
        if (df) df.textContent = vols[i].used + ' / ' + vols[i].size + ' · ' + vols[i].fs.split('/').pop();
        break;
      }
    }
  }
  function applyStatsHtml(t) {
    if (typeof t !== 'string' || !t) return;
    applyMemTop(t);
    applyDfVols(t);
    applySysInfo(t);
    var m = t.match(/Память \(RAM\)[\s\S]{0,800}?Загрузка:<\/td><td><span[^>]*>(\d+)%/);
    if (m) {
      ram = Math.max(0, Math.min(100, parseInt(m[1], 10)));
      pushSeries('ram', ram);
      var mi = t.match(/Использовано \/ Всего:<\/td><td>([^<]*)</);
      if (mi) {
        var parts = mi[1].split('/');
        if (parts.length === 2) {
          var u = parseInt(parts[0], 10), tot = parseInt(parts[1], 10);
          var rf = document.querySelector('[data-metric="ram"] .res-f');
          if (rf && !isNaN(u) && !isNaN(tot)) rf.textContent = 'used ' + u + ' · free ' + (tot - u) + ' · swap ' + (window.__NOC_SWAP === undefined ? '—' : window.__NOC_SWAP);
        }
      }
      paintMeters();
    }
    var dd = t.match(/Диск \(\/opt\)[\s\S]{0,1200}?Загрузка:<\/td><td><span[^>]*>([^<]*)</);
    if (dd) {
      var dv = liveNum(dd[1]);
      if (!isNaN(dv)) {
        disk = Math.max(0, Math.min(100, dv));
        pushSeries('disk', disk);
        var sz = t.match(/Размер:<\/td><td>([^<]*)</);
        var us = t.match(/<td>Использовано:<\/td><td>([^<]*)</);
        var df = document.querySelector('[data-metric="disk"] .res-f');
        if (df && sz && us) df.textContent = us[1] + ' / ' + sz[1];
        paintMeters();
      }
    }
  }
  function liveStamp() {
    var b = document.getElementById('dashRefresh');
    if (!b) return;
    var d = new Date();
    var p = function (n) { return (n < 10 ? '0' : '') + n; };
    var err = liveTickErr.length ? ' err:' + liveTickErr.slice(0, 3).join(',') : '';
    b.textContent = '↻ LIVE ' + p(d.getHours()) + ':' + p(d.getMinutes()) + ':' + p(d.getSeconds()) + ' #' + liveTickN + err;
  }
  function applySession(d) {
    if (!d || d.authenticated !== true) liveStop();
  }
  function liveStop() {
    if (!window.__NOC_LIVE_STATS) return;
    window.__NOC_LIVE_STATS = false;
    mockRestore();
    toast('Сессия потеряна — мок', 'warn');
    liveLogin();
  }
  function liveTick() {
    if (livePending > 4) return; /* прошлый тик ещё висит — не копим запросы */
    liveTickN++;
    liveTickErr = [];
    liveGo('session', liveFetch('session.cgi'), applySession);
    liveGo('status', liveFetch('monitor/status.cgi'), applyProcs);
    liveGo('temp', liveFetch('temperature.cgi'), applyTemp);
    liveGo('wifitemp', liveFetch('wifi_temp.cgi'), applyWifiTemp);
    liveGo('net', liveFetch('network_stats.cgi'), applyNet);
    liveGo('updcheck', liveFetch('update_check.cgi'), applyUpdateCheck);
    liveGo('upgradable', liveFetch('upgradable.cgi'), applyUpgradable);
    liveGo('version', liveFetch('version.cgi'), applyVersion);
    liveGo('events', liveFetch('network_events.cgi?limit=200&tag=all'), applyEvents);
    liveGo('monlog', liveFetch('monitor/log.cgi'), applyMonLog);
    liveGo('stats', liveFetch('stats.cgi'), applyStatsHtml);
    liveGo('procs', liveFetchExp('procs.cgi'), applyFullProcs);
    /* Packages summary for stats card */
    livePkgLoad();
    /* WAN-скорость: источника нет — мок не продвигаем, значение «—» */
    paintMeters();
    liveStamp();
  }
  function liveStart() {
    if (!window.__NOC_TICK) window.__NOC_TICK = setInterval(liveTick, LIVE_TICK_MS);
    if (window.__NOC_LIVE_STATS) return;
    window.__NOC_LIVE_STATS = true;
    window.__NOC_LIVE_ONCE = true;
    toast('LIVE: данные с роутера');
    /* честная загрузка вместо мок-строк: дальше только живые данные */
    [['topCpuBody', 5], ['topMemBody', 5], ['ifBody', 5]].forEach(function (x) {
      var el = document.getElementById(x[0]);
      if (el) el.innerHTML = '<tr><td colspan="' + x[1] + '" class="mut">Загрузка…</td></tr>';
    });
    var ul = document.getElementById('updList');
    if (ul) ul.innerHTML = '<div class="upd mut">Загрузка…</div>';
    var ev = document.getElementById('evtList');
    if (ev) ev.innerHTML = '<div class="evt mut">Загрузка…</div>';
    var rb = document.getElementById('dashRefresh');
    if (rb) rb.onclick = function () { liveTick(); renderStatMods(); toast('Дашборд обновлён (LIVE)'); };
    liveTick();
    bootLiveTab();
    bmRefreshFromServer(); /* rail закладок — из Настроек (links_load.cgi) */
  }
  function liveLogin() {
    var old = document.getElementById('liveLogin');
    if (old) { old.hidden = false; return; }
    var w = document.createElement('div');
    w.id = 'liveLogin';
    w.className = 'pal-wrap';
    w.innerHTML = '<div class="pal"><div style="padding:14px 16px">' +
      '<div class="c-head">Вход · LIVE-данные</div>' +
      '<div class="mut mono" style="margin-bottom:8px">Панель требует сессию (тот же пароль, что в оригинале). Без входа — мок.</div>' +
      '<div class="frm mono"><label>Пароль панели<input type="password" id="livePass"></label>' +
      '<span><button class="btn sm" id="liveGo">Войти</button> ' +
      '<button class="btn sm ghost" id="liveMock">Остаться в моке</button></span>' +
      '<span class="mut" id="liveMsg"></span></div></div></div>';
    document.body.appendChild(w);
    document.getElementById('liveGo').onclick = function () {
      var p = document.getElementById('livePass').value || '';
      fetch(LIVE_BASE + 'login.cgi', {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body: 'password=' + encodeURIComponent(p) + '&action=login'
      }).then(function (r) { return r.json(); }).then(function (d) {
        if (d && d.status === 'ok') { w.hidden = true; liveStart(); }
        else {
          var m = document.getElementById('liveMsg');
          if (m) m.textContent = 'Ошибка: ' + ((d && d.message) || 'вход отклонён');
        }
      }).catch(function () {
        var m = document.getElementById('liveMsg');
        if (m) m.textContent = 'Нет связи (нужен прокси /entware-cgi/ → :8087)';
      });
    };
    document.getElementById('liveMock').onclick = function () {
      w.hidden = true;
      mockRestore();
      toast('Режим мок (без сессии)');
    };
    var inp = document.getElementById('livePass');
    if (inp) {
      inp.onkeydown = function (e) {
        if (e.key === 'Enter') document.getElementById('liveGo').click();
      };
      setTimeout(function () { inp.focus(); }, 0);
    }
  }
  /* честный старт: пустота вместо мок-цифр, пока нет живых данных.
   * Мок возвращается только явным выбором (кнопка «Остаться в моке»). */
  var bootHTML = {};
  (function bootBlank() {
    ['kpiCpu', 'kpiRam', 'kpiDisk', 'kpiWan', 'kpiTemp', 'cpuVal', 'ramVal', 'tempVal'].forEach(function (id) {
      var el = document.getElementById(id);
      if (el) el.textContent = '—';
    });
    [['cpuBar'], ['ramBar']].forEach(function (x) {
      var el = document.getElementById(x[0]);
      if (el) el.style.width = '0%';
    });
    ['topCpuBody', 'topMemBody', 'ifBody', 'updList', 'evtList', 'volList'].forEach(function (id) {
      var el = document.getElementById(id);
      if (el) {
        bootHTML[id] = el.innerHTML;
        el.innerHTML = '';
      }
    });
    Array.prototype.forEach.call(document.querySelectorAll('#kpiRow .res-f'), function (el) {
      el.textContent = '—';
    });
    ['cpu', 'ram'].forEach(function (m) {
      var el = document.querySelector('[data-metric="' + m + '"] .res-h .mut');
      if (el) el.textContent = '—';
    });
  })();
  function mockRestore() {
    Object.keys(bootHTML).forEach(function (id) {
      var el = document.getElementById(id);
      if (el) el.innerHTML = bootHTML[id];
    });
  }
  function liveBoot() {
    if (typeof fetch === 'undefined') return;
    liveFetch('session.cgi').then(function (d) {
      if (d && d.authenticated === true) liveStart();
      else liveLogin();
    });
  }
  /* ===== LIVE packages (read-only): installed/upgradable/available.
   * Без сессии — исходные мок-строки. Действия (установка/удаление) — только чтение. */
  var PKG_MOCK = [
    { name: 'htop', ver: '3.3.0-1 → 3.4.1-1', date: '2026-05-11', st: 'upgrade' },
    { name: 'jq', ver: '1.7.1-1 → 1.8.0-1', date: '2026-04-02', st: 'upgrade' },
    { name: 'ttyd', ver: '1.7.7-1 → 1.8.2-1', date: '2026-06-19', st: 'upgrade' },
    { name: 'bash', ver: '5.3-1', date: '2026-03-26', st: 'installed' },
    { name: 'lighttpd', ver: '1.4.79-1', date: '2026-03-26', st: 'installed' },
    { name: 'wireguard-tools', ver: '1.0.20210914-1', date: '—', st: 'available' }
  ];
  var pkgState = { installed: [], upgradable: [], available: [], filter: 'all', q: '', sortKey: 'package', sortDir: 1, loaded: false, loading: false };
  function plural(n, a, b, c) {
    var m = Math.abs(n) % 100, d = m % 10;
    if (m > 10 && m < 20) return c;
    if (d > 1 && d < 5) return b;
    if (d === 1) return a;
    return c;
  }
  function pkgShortDate(s) { return String(s || '—').slice(0, 10); }
  function pkgInfo(nm) {
    var u = null, ins = null, av = null, i;
    for (i = 0; i < pkgState.upgradable.length; i++) {
      if (pkgState.upgradable[i].package === nm) u = pkgState.upgradable[i];
    }
    for (i = 0; i < pkgState.installed.length; i++) {
      if (pkgState.installed[i].package === nm) ins = pkgState.installed[i];
    }
    for (i = 0; i < pkgState.available.length; i++) {
      if (pkgState.available[i].package === nm) { av = pkgState.available[i]; break; }
    }
    var t = nm;
    if (u) t += ': ' + (u.current || '?') + ' → ' + (u.new || '?');
    else if (ins) t += ' ' + (ins.version || '');
    else if (av) t += ' ' + (av.version || '');
    if (ins && ins.installed_date) t += ' · уст. ' + pkgShortDate(ins.installed_date);
    if (av && av.description) t += ' — ' + String(av.description).slice(0, 120);
    return t;
  }
  function livePost(name, body, timeoutMs) {
    /* POST-мутации пакетов: тело обязательно (иначе lighttpd 411), таймаут под opkg */
    return liveFetchRaw(LIVE_BASE + name, {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: body
    }, timeoutMs || 180000).then(function (t) { return t; });
  }
  function htmlToText(html) {
    var div = document.createElement('div');
    div.innerHTML = String(html || '');
    var t = (div.textContent || '').replace(/[ \t]+\n/g, '\n').replace(/\n{3,}/g, '\n\n').trim();
    var lines = t.split('\n');
    if (lines.length > 20) lines = lines.slice(-20);
    t = lines.join('\n');
    return t.length > 2000 ? t.slice(-2000) : t;
  }
  var PKG_ACT_RU = { install: 'установлен', remove: 'удалён', upgrade: 'обновлён', 'upgrade-all': 'всё обновлено' };
  function pkgJournalLoad() {
    /* Журнал: карточка «Пакеты» на Статистике удалена — пишем только во вкладку
     * и в кэш для фильтра Событий. Новых тиков не добавляем — вызов
     * ленивый: кнопка Журнала, вход на вкладку Пакеты, фильтр Пакеты. */
    if (!window.__NOC_LIVE_STATS) return;
    pkgLogLoading = true;
    var fb = document.getElementById('pkgJournalFullBody');
    if (fb) fb.textContent = 'Загрузка…';
    liveFetch('packages_log.cgi').then(function (d) {
      pkgLogLoading = false;
      var jf = document.getElementById('pkgJournalFullBody');
      if (!Array.isArray(d) || !d.length) {
        if (jf) jf.innerHTML = '<span class="mut">журнал пуст</span>';
        return;
      }
      pkgLogCache = d;
      var html = '';
      var capped = d.length > 100;
      d.slice(0, 100).forEach(function (e) {
        var ok = String(e.status || '') === 'success';
        var actRu = PKG_ACT_RU[String(e.action || '')] || esc(e.action || '?');
        html += '<div class="evt"><span class="ts">' + esc(String(e.timestamp || '').slice(0, 16)) + '</span>' +
          '<span class="badge ' + (ok ? 'ok' : 'crit') + '">' + (ok ? '✓' : '✗') + '</span><span>' +
          esc(e.package || '?') + ' — ' + actRu + '</span></div>';
      });
      if (capped) html += '<div class="evt"><span class="mut">показаны последние 100 из ' + d.length + '</span></div>';
      if (jf) jf.innerHTML = html;
      if (evtFilter === 'pkg') renderEvents();
    });
  }
  /* ===== LIVE monitor (вкладка Защита): status/top/config/action.
   * Топ и бейдж обновляются из тикового monitor/status.cgi (applyProcs) —
   * новых периодических запросов нет; точечные fetch — по входу на вкладку
   * и по кнопкам. Все мутации — POST с телом (иначе lighttpd 411). */
  function monNeedSession() { if (!window.__NOC_LIVE_STATS) { toast('Нужна сессия', 'warn'); return false; } return true; }
  function monBtn(id, dis) { var el = document.getElementById(id); if (el) el.disabled = !!dis; }
  function monBadge(d) {
    var b = document.getElementById('monBadge');
    if (!b) return;
    var running = d && d.daemon_status === 'running';
    var pid = d && (d.daemon_pid || d.pid || (d.processes && d.processes[0] && d.processes[0].pid));
    b.textContent = running ? 'активен · PID ' + (pid || '—') : 'остановлен';
    b.className = 'badge ' + (running ? 'ok' : 'warn');
    /* Визуальные состояния: при работе «Запустить» неактивна, при останове — наоборот. */
    monBtn('monStartBtn', running);
    monBtn('monStopBtn', !running);
    monBtn('monRestartBtn', !running);
  }
  function monTop(d) {
    var tb = document.getElementById('monTopBody');
    if (!tb) return;
    if (!d || !Array.isArray(d.processes) || !d.processes.length) { tb.innerHTML = '<tr><td colspan="5" class="mut">Нет данных</td></tr>'; return; }
    var html = '';
    d.processes.slice(0, 7).forEach(function (p) {
      html += '<tr><td>' + esc(p.pid) + '</td><td>' + esc(p.pcpu) + '</td><td>' + esc(p.time || '—') + '</td>' +
        '<td class="l cmd" title="' + esc(p.command) + '">' + esc(liveShort(p.command, 50)) + '</td>' +
        '<td><button class="btn sm ghost" data-mpid="' + esc(p.pid) + '">Убить</button></td></tr>';
    });
    tb.innerHTML = html;
  }
  function monRefresh() {
    if (!window.__NOC_LIVE_STATS) return;
    liveFetch('monitor/status.cgi').then(function (d) { if (d) { monBadge(d); monTop(d); } });
  }
  function monActionResp(t, fallback) {
    var msg = fallback, ok = true;
    if (t && typeof t === 'object') { msg = t.message || fallback; ok = t.status === 'ok'; }
    else if (typeof t === 'string' && t) msg = t.slice(0, 200);
    toast(msg, ok ? undefined : 'warn');
    monRefresh();
  }
  function monAct(action, extra) {
    if (!monNeedSession()) return;
    livePost('monitor/action.cgi', 'action=' + action + (extra || '')).then(function (t) {
      monActionResp(t, action + ': готово');
    });
  }
  function monCfgLoad() {
    if (!window.__NOC_LIVE_STATS) return;
    liveFetch('monitor/config.cgi').then(function (c) {
      if (!c || typeof c !== 'object') return;
      function set(id, v) { var el = document.getElementById(id); if (el) el.checked = !!v; }
      function setVal(id, v) { var el = document.getElementById(id); if (el && v !== undefined && v !== null) el.value = v; }
      set('monEn', c.enabled);
      setVal('monInt', c.interval);
      set('monIndEn', c.individual && c.individual.enabled);
      setVal('monCpu', c.individual && c.individual.threshold_cpu);
      if (c.individual && c.individual.threshold_time !== undefined) {
        setVal('monTime', Math.round(c.individual.threshold_time / 60));
      }
      if (Array.isArray(c.ignore)) setVal('monIgnoreFull', c.ignore.join(', '));
      set('monPsExc', c.ignore_ps === true);
      setVal('monMax', c.max_processes || 200);
      set('monAutoFull', c.autostart === true);
    });
  }
  function monCfgSave() {
    if (!monNeedSession()) return;
    function checked(id) { var el = document.getElementById(id); return el ? el.checked : false; }
    function val(id) { var el = document.getElementById(id); return el ? el.value : ''; }
    var cfg = {
      enabled: checked('monEn'),
      interval: parseInt(val('monInt'), 10) || 60,
      individual: {
        enabled: checked('monIndEn'),
        threshold_cpu: parseInt(val('monCpu'), 10) || 80,
        threshold_time: (parseInt(val('monTime'), 10) || 5) * 60
      },
      ignore: String(val('monIgnoreFull')).split(',').map(function (s) { return s.replace(/^\s+|\s+$/g, ''); }).filter(function (s) { return s; }),
      ignore_ps: checked('monPsExc'),
      max_processes: parseInt(val('monMax'), 10) || 200,
      autostart: checked('monAutoFull')
    };
    var m = document.getElementById('monMsgFull');
    if (m) m.textContent = 'Сохранение…';
    liveFetchRaw(LIVE_BASE + 'monitor/config.cgi', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(cfg)
    }).then(function (t) {
      var ok = t && t.status === 'ok';
      if (m) m.textContent = ok ? 'сохранено' : 'ошибка: ' + ((t && t.message) || '—');
      toast(ok ? 'Настройки сохранены' : 'Ошибка: ' + ((t && t.message) || '—'), ok ? undefined : 'warn');
      if (ok) livePost('monitor/action.cgi', 'action=restart');
    });
  }
  function monBind() {
    function on(id, fn) { var el = document.getElementById(id); if (el) el.onclick = fn; }
    on('monStartBtn', function () { monAct('start'); });
    on('monStopBtn', function () { monAct('stop'); });
    on('monRestartBtn', function () { monAct('restart'); });
    on('monClearLogBtn', function () { if (confirm('Очистить лог защиты?')) monAct('clearlog'); });
    on('monSaveBtn', monCfgSave);
  }
  monBind();
  monBind();
  /* ===== LIVE logs (вкладка Логи): паритет классики (loadLogsTab в entware.js).
   * Панели действий всегда видимы (вне скрываемых data-pp) — табы logSeg
   * переключают только iframe. Динамические источники — общий ключ
   * localStorage 'entware_dynamic_sources' с классикой. */
  var logTab = 'mgr';
  function logNeedSession() { if (!window.__NOC_LIVE_STATS) { toast('Нужна сессия', 'warn'); return false; } return true; }
  function logFrame() { return document.getElementById('logsFrame'); }
  function logMsg(t) { var m = document.getElementById('logMsg'); if (m) m.textContent = t || ''; }
  function logDynLoad() {
    try {
      var raw = lsGet('entware_dynamic_sources');
      var a = raw ? JSON.parse(raw) : [];
      return Array.isArray(a) ? a : [];
    } catch (e) { return []; }
  }
  function logDynSave(a) { lsSet('entware_dynamic_sources', JSON.stringify(a)); }
  function logSources() {
    var sel = document.getElementById('logSource');
    if (!sel) return;
    fetch('/entware-manager/logger/system_sources.json?_=' + Date.now()).then(function (r) { return r.json(); }).then(function (d) {
      var all = (d && Array.isArray(d.sources) ? d.sources : []).concat(logDynLoad());
      var html = '<option value="">Выберите источник</option>';
      all.forEach(function (src) {
        var file = src.path || src.file || '';
        html += '<option value="' + esc(src.name) + '"' + (file ? ' data-file="' + esc(file) + '"' : '') + '>' + esc(src.name) + '</option>';
      });
      sel.innerHTML = html;
    }).catch(function () {});
  }
  function logShowSource() {
    var sel = document.getElementById('logSource');
    if (!sel || !sel.value) return;
    var opt = sel.options[sel.selectedIndex];
    var file = opt ? opt.getAttribute('data-file') : '';
    var f = logFrame();
    if (sel.value && file && f) f.src = LIVE_BASE + 'logger/system_logs.cgi?embed=noc&file=' + encodeURIComponent(file) + '&_=' + Date.now();
    else if (f) f.src = 'about:blank';
  }
  function logStatus() {
    if (!window.__NOC_LIVE_STATS) return;
    liveFetch('logger/config.cgi').then(function (c) {
      if (!c || typeof c !== 'object') return;
      var on = !!c.enabled;
      var v = document.getElementById('logVerdict');
      if (v) {
        v.innerHTML = '<span class="live-dot"></span>' + (on ? 'логирование вкл' : 'логирование выкл');
        v.className = 'verdict ' + (on ? 'ok-t' : 'mut');
      }
      var dot = document.getElementById('logStateDot');
      if (dot) dot.className = 'dot ' + (on ? 'ok' : 'warn');
      var t = document.getElementById('logStateTxt');
      if (t) t.textContent = on ? 'включено' : 'выключено';
      var b = document.getElementById('logToggleBtn');
      if (b) b.textContent = 'Логирование: ' + (on ? 'вкл' : 'выкл');
    });
  }
  function logSetTab(tab) {
    logTab = tab;
    lsSet('entware_noc_logtab', tab);
    var btns = document.querySelectorAll('#logSeg [data-ltab]');
    Array.prototype.forEach.call(btns, function (x) { x.classList.toggle('active', x.getAttribute('data-ltab') === tab); });
    var mc = document.getElementById('logMgrCard'), sc = document.getElementById('logSysCard');
    if (mc) mc.hidden = (tab !== 'mgr');
    if (sc) sc.hidden = (tab !== 'sys');
    var f = logFrame();
    if (!f) return;
    if (tab === 'mgr') f.src = LIVE_BASE + 'logger/view.cgi?embed=noc&_=' + Date.now();
    else { logSources(); f.src = 'about:blank'; }
  }
  function logBoot() {
    if (!window.__NOC_LIVE_STATS) return;
    logSources();
    logStatus();
  }
  (function logBind() {
    function on(id, fn) { var el = document.getElementById(id); if (el) el.onclick = fn; }
    on('logRefreshBtn', function () { var f = logFrame(); if (f) f.src = f.src; logMsg(''); toast('Логи обновлены (LIVE)'); });
    Array.prototype.forEach.call(document.querySelectorAll('#logSeg [data-ltab]'), function (b) {
      b.onclick = function () { logSetTab(b.getAttribute('data-ltab')); };
    });
    var sel = document.getElementById('logSource');
    if (sel) sel.onchange = logShowSource;
    on('logSysRefresh', function () { logShowSource(); });
    on('logSearchBtn', function () {
      if (!logNeedSession()) return;
      var q = prompt('Введите текст для поиска в имени файла (в /tmp):');
      if (!q || !q.replace(/^\s+|\s+$/g, '')) return;
      liveFetch('logger/find_by_name.cgi?q=' + encodeURIComponent(q)).then(function (files) {
        if (!Array.isArray(files) || !files.length) { toast('Файлы не найдены.', 'warn'); return; }
        var dyn = logDynLoad(), added = 0;
        files.forEach(function (fl) {
          var dup = false;
          dyn.forEach(function (s) { if (s.path === fl.path) dup = true; });
          if (!dup) { dyn.push({ name: fl.name, path: fl.path }); added++; }
        });
        if (!added) { toast('Все файлы уже в списке.'); return; }
        logDynSave(dyn);
        logSources();
        toast('Добавлено ' + added + '.');
      });
    });
    on('logClearSrcBtn', function () {
      if (!confirm('Удалить все сохранённые динамические источники логов?')) return;
      try { localStorage.removeItem('entware_dynamic_sources'); } catch (e) {}
      logSources();
      toast('Сохранённые источники очищены');
    });
    on('logClearBtn', function () {
      if (!logNeedSession()) return;
      if (!confirm('Удалить все логи старше 30 дней?')) return;
      livePost('logger/clear.cgi', '').then(function (t) {
        toast((t && t.message) || 'Готово', t && t.status === 'ok' ? undefined : 'warn');
        var f = logFrame();
        if (f && logTab === 'mgr') f.src = LIVE_BASE + 'logger/view.cgi?embed=noc&_=' + Date.now();
      });
    });
    on('logRotateBtn', function () {
      if (!logNeedSession()) return;
      if (!confirm('Запустить ротацию логов сейчас?')) return;
      livePost('logger/rotate.cgi', '').then(function (t) {
        var msg = (t && t.message) || 'Готово';
        if (t && Array.isArray(t.rotated) && t.rotated.length) {
          msg += ' | ' + t.rotated.map(function (fl) { return fl.path + ' (' + fl.size + ')'; }).join(', ');
        }
        toast(msg.length > 200 ? msg.slice(0, 200) + '…' : msg, t && t.status === 'ok' ? undefined : 'warn');
        var f = logFrame();
        if (f && logTab === 'mgr') setTimeout(function () { f.src = LIVE_BASE + 'logger/view.cgi?embed=noc&_=' + Date.now(); }, 1000);
      });
    });
    on('logToggleBtn', function () {
      if (!logNeedSession()) return;
      liveFetch('logger/config.cgi').then(function (c) {
        if (!c || typeof c !== 'object') { toast('Нет данных конфигурации', 'warn'); return; }
        var on = !!c.enabled;
        if (!confirm('Логирование сейчас ' + (on ? 'включено' : 'отключено') + '. Хотите изменить?')) return;
        c.enabled = !on;
        liveFetchRaw(LIVE_BASE + 'logger/config.cgi', {
          method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(c)
        }).then(function (t) {
          var ok = t && t.status === 'ok';
          toast(ok ? 'Настройки сохранены' : 'Ошибка сохранения', ok ? undefined : 'warn');
          if (ok) logStatus();
        });
      });
    });
    on('logEventsBtn', function () {
      var w = document.createElement('div');
      w.className = 'pal-wrap';
      w.innerHTML = '<div class="pal"><div style="padding:14px 16px">' +
        '<div class="c-head">Системные события <button class="btn sm ghost" id="logEvClose">Закрыть</button></div>' +
        '<div class="log mono" id="logEvBody" style="height:300px">Загрузка…</div></div></div>';
      document.body.appendChild(w);
      w.onclick = function (e) { if (e.target === w) w.remove(); };
      document.getElementById('logEvClose').onclick = function () { w.remove(); };
      liveFetch('logger/system_log.cgi').then(function (t) {
        var b = document.getElementById('logEvBody');
        if (b) b.innerHTML = (typeof t === 'string' && t) ? t : '<span class="mut">событий нет</span>';
      });
    });
  })();
  /* ===== LIVE services (вкладка Службы и Cron): watchdog/services/crontab.
   * Обновление — по входу на вкладку, по кнопке, по действиям + интервал 10 с.
   * Все мутации — POST с телом (иначе lighttpd 411). */
  var svcTimer = null;
  function svcNeedSession() { if (!window.__NOC_LIVE_STATS) { toast('Нужна сессия', 'warn'); return false; } return true; }
  function svcSplitList(s) { return String(s || '').split(',').map(function (x) { return x.replace(/^\s+|\s+$/g, ''); }).filter(function (x) { return x; }); }
  function svcWdBadge(running, pid) {
    var b = document.getElementById('svcWdBadge');
    if (!b) return;
    b.textContent = running ? 'активен · PID ' + (pid || '—') : 'остановлен';
    b.className = 'badge ' + (running ? 'ok' : 'warn');
    function dis(id, v) { var el = document.getElementById(id); if (el) el.disabled = !!v; }
    dis('svcWdStart', running);
    dis('svcWdStop', !running);
    dis('svcWdRestart', !running);
  }
  function svcWdStatus(applyCfg) {
    if (!window.__NOC_LIVE_STATS) return;
    liveFetch('service_watchdog/status.cgi').then(function (d) {
      if (!d) return;
      svcWdBadge(!!d.running, d.pid);
      if (applyCfg && d.config && typeof d.config === 'object') {
        var c = d.config;
        function set(id, v) { var el = document.getElementById(id); if (el) el.checked = !!v; }
        function setVal(id, v) { var el = document.getElementById(id); if (el && v !== undefined && v !== null) el.value = Array.isArray(v) ? v.join(', ') : v; }
        set('svcAuto', c.autostart === true);
        set('svcCustom', c.mode === 'custom');
        setVal('svcWatchList', c.watch_list);
        var wb = document.getElementById('svcWatchBox'); if (wb) wb.hidden = c.mode !== 'custom';
        var ex = c.exclude_list, hasEx = false;
        if (Array.isArray(ex)) { hasEx = ex.length > 0; setVal('svcExcList', ex); }
        else if (typeof ex === 'object' && ex) { var ks = Object.keys(ex).filter(function (k) { return ex[k] !== ''; }); hasEx = ks.length > 0; setVal('svcExcList', ks); }
        set('svcExcMode', hasEx);
        var eb = document.getElementById('svcExcBox'); if (eb) eb.hidden = !hasEx;
        set('svcAutoRestart', c.auto_restart == true || c.auto_restart === 'true');
      }
    });
  }
  function svcWdSave() {
    if (!svcNeedSession()) return;
    function checked(id) { var el = document.getElementById(id); return el ? el.checked : false; }
    var em = document.getElementById('svcExcMode'), exEl = document.getElementById('svcExcList');
    var exList = null;
    if (em && em.checked) exList = svcSplitList(exEl ? exEl.value : '');
    else if (em && !em.checked) exList = 'emptylist';
    var wlEl = document.getElementById('svcWatchList');
    var cfg = {
      mode: checked('svcCustom') ? 'custom' : 'initd',
      watch_list: svcSplitList(wlEl ? wlEl.value : ''),
      auto_restart: checked('svcAutoRestart'),
      autostart: checked('svcAuto'),
      exclude_list: exList
    };
    var m = document.getElementById('svcWdMsg');
    if (m) m.textContent = 'Сохранение…';
    liveFetchRaw(LIVE_BASE + 'service_watchdog/config.cgi', {
      method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(cfg)
    }).then(function (t) {
      var ok = t && t.status === 'ok';
      if (m) m.textContent = ok ? 'сохранено' : 'ошибка: ' + ((t && (t.message || t.error)) || '—');
      toast(ok ? ((t && t.message) || 'Конфигурация сохранена') : 'Ошибка: ' + ((t && (t.message || t.error)) || '—'), ok ? undefined : 'warn');
      if (ok) { svcWdStatus(false); svcWdEvents(); }
    });
  }
  function svcWdAct(action) {
    if (!svcNeedSession()) return;
    livePost('service_watchdog/action.cgi', 'action=' + action).then(function (t) {
      var ok = t && t.status === 'ok';
      toast((t && t.message) || (action + ': готово'), ok ? undefined : 'warn');
      svcWdStatus(false); svcWdEvents(); svcServices();
    });
  }
  function svcWdEvents() {
    var tb = document.getElementById('svcEventsBody');
    if (!tb || !window.__NOC_LIVE_STATS) return;
    liveFetch('service_watchdog/events.cgi?limit=15').then(function (d) {
      if (!d || !Array.isArray(d.events) || !d.events.length) { tb.innerHTML = '<tr><td class="mut">Нет событий</td></tr>'; return; }
      var html = '';
      d.events.forEach(function (e) {
        var lv = String(e.level || 'INFO').toUpperCase();
        var cls = lv === 'WARN' ? 'warn' : (lv === 'ERROR' || lv === 'CRIT' ? 'crit' : 'ok');
        var ts = String(e.timestamp || '');
        if (/^\d{4}-\d{2}-\d{2} /.test(ts)) ts = ts.slice(11, 19);
        html += '<tr><td>' + esc(ts) + '</td><td><span class="badge ' + cls + '">' + esc(lv) + '</span></td>' +
          '<td class="l">' + esc(e.service || '') + ': ' + esc(e.event || '') + ' ' + esc(e.details || '') + '</td></tr>';
      });
      tb.innerHTML = html;
    });
  }
  function svcServices() {
    var tb = document.getElementById('svcBody');
    if (!tb || !window.__NOC_LIVE_STATS) return;
    liveFetch('services.cgi').then(function (a) {
      if (!Array.isArray(a) || !a.length) { tb.innerHTML = '<tr><td colspan="5" class="mut">Нет данных</td></tr>'; return; }
      var html = '', run = 0;
      a.forEach(function (s) {
        var isRun = s.status === 'running';
        if (isRun) run++;
        var pids = Array.isArray(s.pids) ? s.pids : [];
        var pidCell = '—';
        if (isRun && pids.length) {
          pidCell = '<a href="#" data-spid="' + esc(pids[0]) + '">' + esc(pids[0]) +
            (pids.length > 1 ? ' <span class="badge">+' + (pids.length - 1) + '</span>' : '') + '</a>';
        }
        html += '<tr><td class="l">' + esc(s.name) + '</td>' +
          '<td class="' + (isRun ? 'ok-t' : 'mut') + '">' + (isRun ? '● active' : '○ stopped') + '</td>' +
          '<td>' + pidCell + '</td>' +
          '<td class="' + (s.enabled ? 'ok-t' : 'mut') + '">' + (s.enabled ? 'да' : 'нет') + '</td><td>' +
          '<button class="btn sm ghost" data-svc="' + esc(s.name) + '" data-sact="start"' + (isRun ? ' disabled' : '') + '>Старт</button> ' +
          '<button class="btn sm ghost" data-svc="' + esc(s.name) + '" data-sact="stop"' + (!isRun ? ' disabled' : '') + '>Стоп</button> ' +
          '<button class="btn sm ghost" data-svc="' + esc(s.name) + '" data-sact="restart"' + (!isRun ? ' disabled' : '') + '>Рестарт</button> ' +
          '<button class="btn sm ghost" data-svc="' + esc(s.name) + '" data-sact="' + (s.enabled ? 'disable' : 'enable') + '">' + (s.enabled ? 'Авто выкл' : 'Авто вкл') + '</button></td></tr>';
      });
      tb.innerHTML = html;
      var v = document.getElementById('svcVerdict');
      if (v) v.innerHTML = '<span class="live-dot"></span>watchdog · ' + run + '/' + a.length + ' службы';
    });
  }
  function svcAct(name, action) {
    if (!svcNeedSession()) return;
    if ((action === 'stop' || action === 'restart') && !confirm('Служба ' + name + ': ' + action + '?')) return;
    livePost('service_action.cgi', 'name=' + encodeURIComponent(name) + '&action=' + encodeURIComponent(action)).then(function (t) {
      var ok = t && t.status === 'ok';
      toast(ok ? ('Служба ' + name + ': ' + action + ' выполнено') : ('Ошибка: ' + ((t && (t.error || t.message)) || '—')), ok ? undefined : 'warn');
      svcServices();
    });
  }
  function svcCronLoad() {
    if (!window.__NOC_LIVE_STATS) return;
    [['system', 'cronSysTa'], ['opt', 'cronOptTa']].forEach(function (x) {
      liveFetch('crontab.cgi?type=' + x[0]).then(function (d) {
        var el = document.getElementById(x[1]);
        if (el) el.value = (d && d.crontab) || '';
      });
    });
  }
  function svcCronSave(type, taId, msgId) {
    if (!svcNeedSession()) return;
    var ta = document.getElementById(taId), m = document.getElementById(msgId);
    livePost('crontab_update.cgi', 'type=' + encodeURIComponent(type) + '&crontab=' + encodeURIComponent(ta ? ta.value : '')).then(function (t) {
      var ok = t && t.status === 'ok';
      if (m) m.textContent = ok ? '✓ Сохранено' : 'ошибка: ' + ((t && (t.message || t.error)) || '—');
      toast(ok ? ('Crontab ' + type + ' сохранён') : ('Ошибка: ' + ((t && (t.message || t.error)) || '—')), ok ? undefined : 'warn');
    });
  }
  function svcTick() { svcWdStatus(false); svcWdEvents(); svcServices(); }
  function svcBoot() {
    if (!window.__NOC_LIVE_STATS) return;
    svcWdStatus(true); svcWdEvents(); svcServices(); svcCronLoad();
    if (!svcTimer) svcTimer = setInterval(function () { if (window.__NOC_LIVE_STATS) svcTick(); }, 10000);
  }
  function svcBind() {
    function on(id, fn) { var el = document.getElementById(id); if (el) el.onclick = fn; }
    on('svcWdStart', function () { svcWdAct('start'); });
    on('svcWdStop', function () { svcWdAct('stop'); });
    on('svcWdRestart', function () { svcWdAct('restart'); });
    on('svcRefreshBtn', function () { svcWdStatus(true); svcWdEvents(); svcServices(); svcCronLoad(); toast('Службы обновлены (LIVE)'); });
    on('cronSysSave', function () { svcCronSave('system', 'cronSysTa', 'cronSysMsg'); });
    on('cronOptSave', function () { svcCronSave('opt', 'cronOptTa', 'cronOptMsg'); });
    [['svcAuto', true], ['svcCustom', true], ['svcExcMode', false], ['svcAutoRestart', true]].forEach(function (x) {
      var el = document.getElementById(x[0]);
      if (el) el.onchange = function () {
        if (x[0] === 'svcCustom') { var wb = document.getElementById('svcWatchBox'); if (wb) wb.hidden = !el.checked; }
        if (x[0] === 'svcExcMode') { var eb = document.getElementById('svcExcBox'); if (eb) eb.hidden = !el.checked; }
        if (x[1]) svcWdSave();
      };
    });
    function onchg(id) { var el = document.getElementById(id); if (el) el.onchange = svcWdSave; }
    onchg('svcWatchList'); onchg('svcExcList');
  }
  svcBind();
  /* ===== LIVE network (вкладка Сеть): daemon/if/routes/arp/wifi/events.
   * Обновление — по входу на вкладку, по кнопке + интервал 10 с.
   * Мутации — POST с телом (иначе lighttpd 411). */
  var netTimer = null, netCfg = null;
  function netNeedSession() { if (!window.__NOC_LIVE_STATS) { toast('Нужна сессия', 'warn'); return false; } return true; }
  function netBadge(running, pid) {
    var b = document.getElementById('netBadge');
    if (!b) return;
    b.textContent = running ? 'работает · PID ' + (pid || '—') : 'остановлен';
    b.className = 'badge ' + (running ? 'ok' : 'warn');
    function dis(id, v) { var el = document.getElementById(id); if (el) el.disabled = !!v; }
    dis('netStartBtn', running);
    dis('netStopBtn', !running);
    dis('netRestartBtn', !running);
  }
  function netStatus() {
    if (!window.__NOC_LIVE_STATS) return;
    liveFetch('network/status.cgi').then(function (d) { if (d) netBadge(!!d.running, d.pid); });
  }
  function netIfaces() {
    var tb = document.getElementById('netIfBody');
    if (!tb || !window.__NOC_LIVE_STATS) return;
    var hideUn = document.getElementById('netHideUnknown');
    hideUn = hideUn ? hideUn.checked : false;
    liveFetch('network/interfaces.cgi').then(function (d) {
      var list = d && Array.isArray(d.interfaces) ? d.interfaces : [];
      if (hideUn) list = list.filter(function (i) { return i.state !== 'UNKNOWN'; });
      if (!list.length) { tb.innerHTML = '<tr><td colspan="6" class="mut">Нет данных</td></tr>'; return; }
      var html = '';
      list.forEach(function (i) {
        html += '<tr><td class="l"><b>' + esc(i.name) + '</b></td>' +
          '<td class="' + (i.state === 'UP' ? 'ok-t' : 'crit-t') + '">' + esc(i.state) + '</td>' +
          '<td class="l">' + esc(i.ip) + '</td><td class="l">' + esc(i.mac || '—') + '</td>' +
          '<td class="l">' + esc(i.type || '—') + (i.ssid ? ' (' + esc(i.ssid) + ')' : '') + '</td>' +
          '<td>' + esc(i.speed || '—') + '</td></tr>';
      });
      tb.innerHTML = html;
    });
  }
  function netRoutes() {
    var tb = document.getElementById('netRtBody');
    if (!tb || !window.__NOC_LIVE_STATS) return;
    liveFetch('network/routes.cgi').then(function (d) {
      var list = d && Array.isArray(d.routes) ? d.routes : [];
      if (!list.length) { tb.innerHTML = '<tr><td colspan="4" class="mut">Нет данных</td></tr>'; return; }
      var html = '';
      list.forEach(function (r) {
        html += '<tr><td class="l">' + esc(r.destination) + '</td><td class="l">' + esc(r.gateway) + '</td>' +
          '<td class="l">' + esc(r.interface) + '</td><td>' + esc(r.metric || '—') + '</td></tr>';
      });
      tb.innerHTML = html;
    });
  }
  function netArp() {
    var tb = document.getElementById('netArpBody');
    if (!tb || !window.__NOC_LIVE_STATS) return;
    liveFetch('network/arp.cgi').then(function (d) {
      var list = d && Array.isArray(d.entries) ? d.entries : [];
      if (!list.length) { tb.innerHTML = '<tr><td colspan="5" class="mut">Таблица пуста</td></tr>'; return; }
      var html = '';
      list.forEach(function (e) {
        html += '<tr><td class="l">' + esc(e.ip) + '</td><td class="l">' + esc(e.name || '—') + '</td>' +
          '<td class="l">' + esc(e.mac) + '</td><td class="l">' + esc(e.interface) + '</td>' +
          '<td class="' + (String(e.state).indexOf('REACHABLE') === 0 || e.state === 'PERMANENT' ? 'ok-t' : 'mut') + '">' + esc(e.state) + '</td></tr>';
      });
      tb.innerHTML = html;
    });
  }
  function netWifi() {
    var tb = document.getElementById('netWfBody');
    if (!tb || !window.__NOC_LIVE_STATS) return;
    liveFetch('network/wifi.cgi').then(function (d) {
      var list = d && Array.isArray(d.clients) ? d.clients : [];
      if (!list.length) { tb.innerHTML = '<tr><td colspan="7" class="mut">Клиентов Wi-Fi не найдено</td></tr>'; return; }
      var html = '';
      list.forEach(function (c) {
        var r = parseInt(c.rssi, 10);
        var cls = isNaN(r) ? 'mut' : (r >= -50 ? 'ok-t' : (r >= -70 ? 'warn-t' : 'crit-t'));
        html += '<tr><td class="l">' + esc(c.name || c.hostname || '—') + '</td><td class="l">' + esc(c.ip) + '</td>' +
          '<td class="l">' + esc(c.mac) + '</td><td class="' + cls + '">' + esc(c.rssi) + '</td>' +
          '<td class="l">' + esc(c.mode || '—') + '</td><td>' + esc(c.txrate > 0 ? c.txrate : '—') + '</td>' +
          '<td class="l">' + esc(c.segment || '—') + '</td></tr>';
      });
      tb.innerHTML = html;
    });
  }
  function netEvents() {
    var tb = document.getElementById('netEvBody');
    if (!tb || !window.__NOC_LIVE_STATS) return;
    liveFetch('network/events.cgi?limit=20').then(function (d) {
      if (!d || !Array.isArray(d.events) || !d.events.length) { tb.innerHTML = '<tr><td class="mut">Нет событий</td></tr>'; return; }
      var html = '';
      d.events.forEach(function (e) {
        var lv = String(e.level || 'INFO').toUpperCase();
        var cls = lv === 'WARN' ? 'warn' : (lv === 'ERROR' || lv === 'CRIT' ? 'crit' : 'ok');
        var ts = String(e.timestamp || '');
        if (/^\d{4}-\d{2}-\d{2} /.test(ts)) ts = ts.slice(5, 16);
        html += '<tr><td>' + esc(ts) + '</td><td><span class="badge ' + cls + '">' + esc(lv) + '</span></td>' +
          '<td class="l">' + esc(e.service || '') + ': ' + esc(e.event || '') + ' ' + esc(e.details || '') + '</td></tr>';
      });
      tb.innerHTML = html;
    });
  }
  function netTick() { netStatus(); netIfaces(); netRoutes(); netArp(); netEvents(); }
  function netBoot() {
    if (!window.__NOC_LIVE_STATS) return;
    netStatus(); netIfaces(); netRoutes(); netArp(); netWifi(); netEvents();
    liveFetch('network/config.cgi').then(function (c) {
      netCfg = (c && typeof c === 'object') ? c : {};
      var a = document.getElementById('netAuto');
      if (a) a.checked = netCfg.autostart === true;
    });
    if (!netTimer) netTimer = setInterval(function () { if (window.__NOC_LIVE_STATS) netTick(); }, 10000);
  }
  function netAct(action) {
    if (!netNeedSession()) return;
    livePost('network/action.cgi', 'action=' + action).then(function (t) {
      var ok = t && t.status === 'ok';
      toast((t && t.message) || (action + ': готово'), ok ? undefined : 'warn');
      netStatus(); netEvents();
    });
  }
  function netBind() {
    function on(id, fn) { var el = document.getElementById(id); if (el) el.onclick = fn; }
    on('netStartBtn', function () { netAct('start'); });
    on('netStopBtn', function () { netAct('stop'); });
    on('netRestartBtn', function () { netAct('restart'); });
    on('netRefreshBtn', function () { netTick(); netWifi(); toast('Сеть обновлена (LIVE)'); });
    var a = document.getElementById('netAuto');
    if (a) a.onchange = function () {
      if (!netNeedSession()) { a.checked = !a.checked; return; }
      if (!netCfg) netCfg = {};
      netCfg.autostart = !!a.checked;
      var m = document.getElementById('netMsg');
      liveFetchRaw(LIVE_BASE + 'network/config.cgi', {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(netCfg)
      }).then(function (t) {
        var ok = t && t.status === 'ok';
        if (m) m.textContent = ok ? 'сохранено' : 'ошибка';
        toast(ok ? (a.checked ? 'Автозапуск включён' : 'Автозапуск выключен') : 'Ошибка сохранения', ok ? undefined : 'warn');
      });
    };
    var h = document.getElementById('netHideUnknown');
    if (h) h.onchange = netIfaces;
  }
  netBind();
  /* ===== LIVE settings-1 (Терминал и ссылки): ttyd_control + links.
   * Загрузка — по входу на вкладку Настройки. Пароли чистятся после действий. */
  var SET_LINK_ICONS = [['link', '⧉ link'], ['box', '▦ box'], ['router', 'router'], ['package', 'package'], ['shield', 'shield'], ['download', 'download'], ['process', 'process'], ['terminal', 'terminal']];
  function setNeedSession() { if (!window.__NOC_LIVE_STATS) { toast('Нужна сессия', 'warn'); return false; } return true; }
  function setTtydRow(label, path, inst, modeLabel) {
    var run = inst && inst.state === 'running';
    return '<tr><td class="l">' + label + '</td><td class="l">' + path + '</td>' +
      '<td class="' + (run ? 'ok-t' : 'mut') + '">' + (run ? '● запущен' : '○ остановлен') + '</td>' +
      '<td>' + esc((inst && inst.pid) || '—') + '</td><td class="l">' + esc(modeLabel || '—') + '</td></tr>';
  }
  function setTtydStatus() {
    if (!window.__NOC_LIVE_STATS) return;
    liveFetch('ttyd_control.cgi').then(function (d) {
      if (!d) return;
      var tb = document.getElementById('setTtydBody');
      var t = d.terminal || {}, h = d.htop || {};
      if (tb) tb.innerHTML = setTtydRow('htop', '/htop/', h, '—') + setTtydRow('Терминал', '/terminal/', t, t.mode === 'telnet' ? 'telnet' : 'entware');
      function dis(id, v) { var el = document.getElementById(id); if (el) el.disabled = !!v; }
      dis('setHtopStart', h.state === 'running');
      dis('setHtopStop', h.state !== 'running');
      dis('setHtopRestart', h.state !== 'running');
      dis('setTermStart', t.state === 'running');
      dis('setTermStop', t.state !== 'running');
      dis('setTermRestart', t.state !== 'running');
      var ms = document.getElementById('setTermMode');
      if (ms && t.state === 'running') ms.value = t.mode === 'telnet' ? 'telnet' : 'entware';
    });
  }
  function setTtydAct(action, port, passId, modeId) {
    if (!setNeedSession()) return;
    var body = 'action=' + encodeURIComponent(action) + '&port=' + encodeURIComponent(port);
    var passEl = passId ? document.getElementById(passId) : null;
    if (passEl && passEl.value) body += '&pass=' + encodeURIComponent(passEl.value);
    if (modeId) { var me = document.getElementById(modeId); if (me && me.value) body += '&mode=' + encodeURIComponent(me.value); }
    livePost('ttyd_control.cgi', body || 'action=' + encodeURIComponent(action)).then(function (t) {
      toast((t && t.message) || (action + ': готово'), (t && t.status === 'ok') || typeof t === 'string' ? undefined : 'warn');
      if (passEl && (action === 'start' || action === 'stop' || action === 'restart')) passEl.value = '';
      setTtydStatus();
    });
  }
  function setIconSelect(cur) {
    var html = '<select class="link-icon-select">';
    SET_LINK_ICONS.forEach(function (o) {
      html += '<option value="' + o[0] + '"' + (o[0] === cur ? ' selected' : '') + '>' + o[1] + '</option>';
    });
    if (cur && !SET_LINK_ICONS.some(function (o) { return o[0] === cur; })) {
      html += '<option value="' + esc(cur) + '" selected>' + esc(cur) + '</option>';
    }
    return html + '</select>';
  }
  function setLinksRender(list) {
    var tb = document.getElementById('setLinksBody');
    if (!tb) return;
    if (!list.length) { tb.innerHTML = '<tr><td colspan="5" class="mut">Ссылок нет</td></tr>'; return; }
    var html = '';
    list.forEach(function (l, i) {
      html += '<tr data-idx="' + i + '"><td class="mut">⋮⋮</td><td>' + setIconSelect(l.icon || 'link') + '</td>' +
        '<td><input class="link-name" value="' + esc(l.name || '') + '"></td>' +
        '<td><input class="link-url" value="' + esc(l.url || '') + '" style="width:220px"></td>' +
        '<td><button class="btn sm" data-linksave="' + i + '">Сохранить</button> ' +
        '<button class="btn sm ghost" data-linkdel="' + i + '">Удалить</button></td></tr>';
    });
    tb.innerHTML = html;
  }
  function setLinksCollect() {
    var tb = document.getElementById('setLinksBody');
    var out = [];
    if (!tb) return out;
    Array.prototype.forEach.call(tb.querySelectorAll('tr[data-idx],tr:not([data-idx])'), function (r) {
      var n = r.querySelector('.link-name'), u = r.querySelector('.link-url'), s = r.querySelector('.link-icon-select');
      if (n && u && n.value && u.value) out.push({ name: n.value, url: u.value, icon: s ? s.value : 'link' });
    });
    return out;
  }
  function setLinksLoad() {
    if (!window.__NOC_LIVE_STATS) return;
    liveFetch('links_load.cgi').then(function (d) {
      setLinksRender(Array.isArray(d) ? d : []);
    });
  }
  function setLinksSave(list, okMsg) {
    if (!setNeedSession()) return;
    var m = document.getElementById('setLinksMsg');
    liveFetchRaw(LIVE_BASE + 'links_save.cgi', {
      method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(list)
    }).then(function (t) {
      var ok = t && t.status === 'ok';
      if (m) m.textContent = ok ? 'сохранено' : 'ошибка: ' + ((t && (t.message || t.error)) || '—');
      toast(ok ? okMsg : 'Ошибка сохранения ссылок', ok ? undefined : 'warn');
      if (ok) { setLinksLoad(); bmRefreshFromServer(); }
    });
  }
  /* ===== LIVE уведомления (подвкладка notif): Telegram-конфиг + пороги.
   * Паритет классики (entware.js): GET/POST telegram_config.cgi (form-urlencoded),
   * POST telegram_test.cgi с пустым телом, пороги блоком thresholds=JSON. */
  var NTG_SRCS = ['system', 'monitor', 'network', 'service', 'packages', 'login', 'bridge'];
  var NTH_KEYS = ['cpu_temp', 'wifi0_temp', 'wifi1_temp', 'cpu_load', 'ram_used', 'disk_temp'];
  function ntgStatus(msg, ok) {
    var el = document.getElementById('ntgStatus');
    if (el) { el.textContent = msg; el.style.color = ok === false ? '#e53e3e' : ok === true ? '#2ecc71' : ''; }
  }
  function nthStatus(msg, ok) {
    var el = document.getElementById('nthStatus');
    if (el) { el.textContent = msg; el.style.color = ok === false ? '#e53e3e' : ok === true ? '#2ecc71' : ''; }
  }
  function ntgFillThresholds(th) {
    th = th || {};
    NTH_KEYS.forEach(function (k) {
      var cb = document.getElementById('nth-' + k), val = document.getElementById('nth-' + k + '-val');
      if (cb) cb.checked = !!(th[k] && th[k].enabled);
      if (val) val.value = (th[k] && th[k].value) || 0;
    });
  }
  function ntgLoad() {
    if (!window.__NOC_LIVE_STATS) return;
    liveFetch('telegram_config.cgi').then(function (d) {
      if (!d || typeof d !== 'object') { ntgStatus('нет связи', false); return; }
      if (d.status === 'error') { ntgStatus('Ошибка: ' + (d.message || '—'), false); return; }
      function setC(id, v) { var el = document.getElementById(id); if (el) el.checked = !!v; }
      function setV(id, v) { var el = document.getElementById(id); if (el && v !== undefined && v !== null) el.value = v; }
      setC('ntgEnabled', d.enabled);
      setC('ntgBot', d.bot_enabled);
      setC('ntgAutostart', d.autostart);
      setV('ntgLevel', d.level || 'ERROR');
      setV('ntgChat', d.chat_id || '');
      setV('ntgChatExtra', d.chat_ids_extra || '');
      setV('ntgProxy', d.proxy_url || '');
      setC('ntgQuiet', d.quiet_enabled);
      setV('ntgQuietFrom', d.quiet_from || '');
      setV('ntgQuietTo', d.quiet_to || '');
      var srcs = d.sources || [];
      NTG_SRCS.forEach(function (s) { setC('ntg-src-' + s, srcs.indexOf(s) !== -1); });
      ntgFillThresholds(d.thresholds);
      var m = document.getElementById('ntgMsg');
      if (m) m.textContent = d.configured ? 'настроен (токен скрыт)' : 'не настроен';
      var tb = document.getElementById('ntgTest');
      if (tb) tb.disabled = !d.configured;
      ntgStatus(d.configured ? '' : 'Укажите токен и chat_id.', undefined);
    });
  }
  function ntgSourcesCsv() {
    var out = [];
    NTG_SRCS.forEach(function (s) {
      var el = document.getElementById('ntg-src-' + s);
      if (el && el.checked) out.push(s);
    });
    return out.join(',');
  }
  function ntgVal(id) { var el = document.getElementById(id); return el ? el.value : ''; }
  function ntgChecked(id) { var el = document.getElementById(id); return !!(el && el.checked); }
  function ntgSave() {
    if (!setNeedSession()) return;
    var body = 'enabled=' + (ntgChecked('ntgEnabled') ? 'true' : 'false');
    body += '&bot_enabled=' + (ntgChecked('ntgBot') ? 'true' : 'false');
    body += '&autostart=' + (ntgChecked('ntgAutostart') ? 'true' : 'false');
    body += '&level=' + encodeURIComponent(ntgVal('ntgLevel'));
    body += '&chat_id=' + encodeURIComponent(ntgVal('ntgChat'));
    body += '&sources=' + encodeURIComponent(ntgSourcesCsv());
    body += '&quiet_enabled=' + (ntgChecked('ntgQuiet') ? 'true' : 'false');
    if (ntgVal('ntgQuietFrom') !== '') body += '&quiet_from=' + encodeURIComponent(ntgVal('ntgQuietFrom'));
    if (ntgVal('ntgQuietTo') !== '') body += '&quiet_to=' + encodeURIComponent(ntgVal('ntgQuietTo'));
    if (ntgVal('ntgChatExtra')) body += '&chat_ids_extra=' + encodeURIComponent(ntgVal('ntgChatExtra'));
    if (ntgVal('ntgProxy')) body += '&proxy_url=' + encodeURIComponent(ntgVal('ntgProxy'));
    if (ntgVal('ntgToken')) body += '&bot_token=' + encodeURIComponent(ntgVal('ntgToken'));
    ntgStatus('Сохранение…', undefined);
    livePost('telegram_config.cgi', body).then(function (t) {
      if (t && t.status === 'ok') {
        ntgStatus('✓ Настройки сохранены', true);
        var tk = document.getElementById('ntgToken');
        if (tk) tk.value = '';
        ntgLoad();
      } else {
        ntgStatus('Ошибка: ' + ((t && (t.message || t.error)) || 'нет связи'), false);
      }
    });
  }
  function ntgTest() {
    if (!setNeedSession()) return;
    ntgStatus('Отправка теста…', undefined);
    livePost('telegram_test.cgi', '').then(function (t) {
      var ok = t && t.status === 'ok';
      ntgStatus(ok ? '✓ Тестовое сообщение отправлено' : 'Ошибка: ' + ((t && (t.message || t.error)) || 'нет связи'), ok);
    });
  }
  function nthCollect() {
    var th = {};
    NTH_KEYS.forEach(function (k) {
      var cb = document.getElementById('nth-' + k), val = document.getElementById('nth-' + k + '-val');
      th[k] = { enabled: !!(cb && cb.checked), value: parseInt(val ? val.value : 0, 10) || 0 };
    });
    return th;
  }
  function nthSave() {
    if (!setNeedSession()) return;
    var body = 'thresholds=' + encodeURIComponent(JSON.stringify(nthCollect()));
    if (ntgVal('ntgChat')) body += '&chat_id=' + encodeURIComponent(ntgVal('ntgChat'));
    nthStatus('Сохранение…', undefined);
    livePost('telegram_config.cgi', body).then(function (t) {
      var ok = t && t.status === 'ok';
      nthStatus(ok ? '✓ Пороги сохранены' : 'Ошибка: ' + ((t && (t.message || t.error)) || 'нет связи'), ok);
    });
  }
  /* ===== LIVE безопасность (подвкладка sec): пароль панели, TLS, журнал входов.
   * Паритет классики (entware.js): auth_config.cgi (enabled/password/current_password),
   * tls_config.cgi (password+enabled+port, после успеха — reload), auth_log.cgi. */
  var NSEC_AUTH_ON = false;
  function nsecStatus(msg, ok) {
    var el = document.getElementById('nsecStatus');
    if (el) { el.textContent = msg; el.style.color = ok === false ? '#e53e3e' : ok === true ? '#2ecc71' : ''; }
  }
  function ntlsStatus(msg, ok) {
    var el = document.getElementById('ntlsStatus');
    if (el) { el.textContent = msg; el.style.color = ok === false ? '#e53e3e' : ok === true ? '#2ecc71' : ''; }
  }
  function nsecLoad() {
    if (!window.__NOC_LIVE_STATS) return;
    liveFetch('auth_config.cgi').then(function (d) {
      if (!d || typeof d !== 'object') return;
      var en = !!d.enabled;
      NSEC_AUTH_ON = !!(d.configured || en);
      var cb = document.getElementById('nsecEnabled');
      if (cb) cb.checked = en;
      var m = document.getElementById('nsecMsg');
      if (m) m.textContent = NSEC_AUTH_ON ? 'защита включена' : 'защита выключена';
    });
    ntlsLoad();
    nalLoad();
  }
  function nsecSave() {
    if (!setNeedSession()) return;
    var en = document.getElementById('nsecEnabled');
    var enabled = !!(en && en.checked);
    var pass = document.getElementById('nsecPass');
    var conf = document.getElementById('nsecPassConfirm');
    var cur = document.getElementById('nsecCurrent');
    var pv = pass ? pass.value : '', cv = conf ? conf.value : '';
    if (enabled && pv && pv !== cv) { nsecStatus('Пароли не совпадают', false); return; }
    if (enabled && pv && pv.length < 8) { nsecStatus('Пароль должен быть минимум 8 символов', false); return; }
    var body = 'enabled=' + (enabled ? 'true' : 'false') + '&password=' + encodeURIComponent(pv);
    if (cur && cur.value) body += '&current_password=' + encodeURIComponent(cur.value);
    nsecStatus('Сохранение…', undefined);
    livePost('auth_config.cgi', body).then(function (t) {
      if (t && t.status === 'ok') {
        nsecStatus('✓ Настройки сохранены', true);
        if (pass) pass.value = '';
        if (conf) conf.value = '';
        if (cur) cur.value = '';
        NSEC_AUTH_ON = enabled;
        nsecLoad();
      } else {
        nsecStatus('Ошибка: ' + ((t && (t.message || t.error)) || 'нет связи'), false);
      }
    });
  }
  function ntlsLoad() {
    if (!window.__NOC_LIVE_STATS) return;
    liveFetch('tls_config.cgi').then(function (d) {
      if (!d || typeof d !== 'object') return;
      var cb = document.getElementById('ntlsEnabled');
      if (cb) cb.checked = !!d.tls;
      if (d.tls_port) {
        var p = document.getElementById('ntlsPort');
        if (p) p.value = d.tls_port;
      }
      var hint = [];
      if (!d.server_running) hint.push('режим lighttpd — недоступно');
      if (d.has_cert) hint.push('сертификат создан');
      var m = document.getElementById('ntlsMsg');
      if (m) m.textContent = hint.join(' · ');
      if (cb) cb.disabled = !d.server_running;
      var sb = document.getElementById('ntlsSave');
      if (sb) sb.disabled = !d.server_running;
    });
  }
  function ntlsSave() {
    if (!setNeedSession()) return;
    var pw = document.getElementById('ntlsPass');
    if (!pw || !pw.value) { ntlsStatus('Введите пароль панели для подтверждения', false); if (pw) pw.focus(); return; }
    var en = document.getElementById('ntlsEnabled');
    var port = document.getElementById('ntlsPort');
    var body = 'password=' + encodeURIComponent(pw.value) +
      '&enabled=' + ((en && en.checked) ? 'true' : 'false') +
      '&port=' + encodeURIComponent(port && port.value ? port.value : '8443');
    ntlsStatus('Сохранение и перезапуск сервера…', undefined);
    livePost('tls_config.cgi', body).then(function (t) {
      if (t && t.status === 'ok') {
        ntlsStatus(t.message || 'Готово — страница перезагрузится', true);
        setTimeout(function () { location.reload(); }, 2500);
      } else {
        ntlsStatus('Ошибка: ' + ((t && (t.message || t.error)) || 'нет связи'), false);
      }
    });
  }
  function nalLoad() {
    if (!window.__NOC_LIVE_STATS) return;
    var tb = document.getElementById('nalBody');
    liveFetch('auth_log.cgi').then(function (d) {
      if (!tb) return;
      if (!d || typeof d !== 'object') { tb.innerHTML = '<tr><td colspan="3" class="mut">нет связи</td></tr>'; return; }
      var entries = d.entries || [];
      var badge = document.getElementById('nalBadge');
      if (badge) {
        if (d.failed_24h > 0) {
          badge.textContent = d.failed_24h + ' неудачных попыток за 24ч';
          badge.style.cssText = 'display:inline-block;background:#e53e3e;color:#fff;border-radius:12px;padding:2px 10px;font-size:0.8rem;';
        } else {
          badge.textContent = '';
          badge.style.cssText = '';
        }
      }
      if (!entries.length) { tb.innerHTML = '<tr><td colspan="3" class="mut">Попыток входа не зафиксировано</td></tr>'; return; }
      var html = '';
      entries.forEach(function (e) {
        var color = '';
        if (/Неверный пароль|CSRF|отклон/.test(e.message)) color = '#e53e3e';
        else if (/заблокирован|Слишком много/.test(e.message)) color = '#f59e0b';
        else if (/Успешный/.test(e.message)) color = '#2ecc71';
        html += '<tr><td>' + esc(e.time) + '</td><td><code>' + esc(e.ip) + '</code></td>' +
          '<td' + (color ? ' style="color:' + color + ';font-weight:600"' : '') + '>' + esc(e.message) + '</td></tr>';
      });
      tb.innerHTML = html;
    });
  }
  /* ===== LIVE обслуживание (подвкладка mnt): бэкап/восстановление, офлайн-пакет, обновление.
   * Паритет классики (entware.js): ссылка backup.cgi, POST backup_restore.cgi (gzip-тело),
   * GET prepare_offline.cgi (blob), update_check/update_run/update_status.cgi с поллингом 2с.
   * Бинарные ответы качаем голым fetch: liveFetchRaw испортил бы tar.gz через .text(). */
  var NMNT_POLL = null;
  function nmntBackupStatus(msg, ok) {
    var el = document.getElementById('nmntBackupStatus');
    if (el) { el.textContent = msg; el.style.color = ok === false ? '#e53e3e' : ok === true ? '#2ecc71' : ''; }
  }
  function nmntUpdStatus(html) {
    var el = document.getElementById('nmntStatus');
    if (el) el.innerHTML = html;
  }
  function nmntRestore(input) {
    var file = input.files && input.files[0];
    if (!file) return;
    if (!setNeedSession()) { input.value = ''; return; }
    nmntBackupStatus('Восстановление…', undefined);
    file.arrayBuffer().then(function (buf) {
      return fetch(LIVE_BASE + 'backup_restore.cgi', {
        method: 'POST', headers: { 'Content-Type': 'application/gzip' }, body: buf
      }).then(function (r) { return r.json(); });
    }).then(function (t) {
      var ok = t && t.status === 'ok';
      nmntBackupStatus(ok ? '✓ ' + (t.message || 'Восстановлено') : 'Ошибка: ' + ((t && t.message) || '—'), ok);
    }).catch(function (e) {
      nmntBackupStatus('Ошибка: ' + (e && e.message ? e.message : e), false);
    });
    input.value = '';
  }
  function nmntOffline() {
    if (!setNeedSession()) return;
    nmntBackupStatus('⏳ Подготовка офлайн-пакета… (скачивание зависимостей, ~30-60 сек)', undefined);
    fetch(LIVE_BASE + 'prepare_offline.cgi?_=' + Date.now()).then(function (r) {
      if (!r.ok) throw new Error('Сервер ответил ' + r.status);
      var disp = r.headers.get('Content-Disposition') || '';
      var m = disp.split('filename=')[1];
      var name = m ? m.replace(/"/g, '').trim() : 'entware-manager-offline.tar.gz';
      return r.blob().then(function (b) { return { blob: b, name: name }; });
    }).then(function (o) {
      var url = URL.createObjectURL(o.blob);
      var a = document.createElement('a');
      a.href = url;
      a.download = o.name;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      setTimeout(function () { URL.revokeObjectURL(url); }, 5000);
      nmntBackupStatus('✓ Офлайн-пакет готов: ' + o.name + ' (сохранён в Загрузки)', true);
    }).catch(function (e) {
      nmntBackupStatus('Ошибка: ' + (e && e.message ? e.message : e), false);
    });
  }
  function nmntApplyCheck(d, btn) {
    var cur = document.getElementById('nmntCurrent');
    var run = document.getElementById('nmntRun');
    if (!d || typeof d !== 'object') {
      if (cur) cur.textContent = 'нет связи';
      if (btn) btn.disabled = false;
      return;
    }
    if (cur) cur.textContent = d.current || '?';
    if (d.has_update) {
      var v = document.getElementById('nmntVersion');
      if (v) v.textContent = d.latest;
      if (run) run.hidden = false;
      nmntUpdStatus('<span style="color:#2ecc71">Доступна версия ' + esc(d.latest) + '</span>');
    } else if (d.error) {
      nmntUpdStatus('<span style="color:#e53e3e">' + esc(d.error) + '</span>');
    } else {
      if (run) run.hidden = true;
      nmntUpdStatus('<span class="mut">Установлена последняя версия</span>');
    }
    if (btn) btn.disabled = false;
  }
  function nmntInfo() {
    if (!window.__NOC_LIVE_STATS) return;
    liveFetch('update_check.cgi').then(function (d) { nmntApplyCheck(d, null); });
  }
  function nmntCheck() {
    if (!setNeedSession()) return;
    var btn = document.getElementById('nmntCheck');
    if (btn) btn.disabled = true;
    nmntUpdStatus('<span class="mut">Проверка…</span>');
    liveFetch('update_check.cgi').then(function (d) { nmntApplyCheck(d, btn); });
  }
  function nmntPoll() {
    if (NMNT_POLL) clearInterval(NMNT_POLL);
    var startedAt = Date.now();
    NMNT_POLL = setInterval(function () {
      liveFetch('update_status.cgi').then(function (d) {
        if (!d || typeof d !== 'object') return;
        var log = document.getElementById('nmntLog');
        if (log && d.lines && d.lines.length) {
          log.textContent = d.lines.join('\n');
          log.scrollTop = log.scrollHeight;
        }
        if (d.status === 'done') {
          clearInterval(NMNT_POLL);
          if (typeof pkgCacheClearAll === 'function') { try { pkgCacheClearAll(); } catch (e) {} }
          nmntUpdStatus('<span style="color:#2ecc71">✓ Обновление завершено</span>');
          var run = document.getElementById('nmntRun');
          if (run) run.hidden = true;
          var rb = document.getElementById('nmntReinstall');
          if (rb) rb.disabled = false;
        } else if (d.status === 'error') {
          clearInterval(NMNT_POLL);
          nmntUpdStatus('<span style="color:#e53e3e">Ошибка: ' + esc(d.error || '—') + '</span>');
          var r2 = document.getElementById('nmntRun');
          if (r2) r2.disabled = false;
          var rb2 = document.getElementById('nmntReinstall');
          if (rb2) rb2.disabled = false;
        } else if (d.status === 'running') {
          var el = Math.round((Date.now() - startedAt) / 1000);
          var ts = el >= 60 ? Math.floor(el / 60) + ' мин ' + (el % 60) + ' с' : el + ' с';
          var prog = d.progress || (d.lines && d.lines.length ? d.lines[d.lines.length - 1] : '');
          nmntUpdStatus('<span style="color:#f59e0b"><b>' + esc(prog) + '</b> ' +
            '<span class="mut">(' + ts + ' — обычно до 1 минуты; не закрывайте страницу)</span></span>');
        }
      });
    }, 2000);
  }
  function nmntRun(mode) {
    if (!setNeedSession()) return;
    if (mode === 'reinstall' && !confirm('Переустановить текущую версию Entware Manager?\nБудет повторно скачана и установлена установленная версия. Конфигурация сохранится.')) return;
    var log = document.getElementById('nmntLog');
    if (log) { log.hidden = false; log.textContent = mode === 'reinstall' ? 'Запуск переустановки…' : 'Запуск обновления…'; }
    nmntUpdStatus('<span style="color:#f59e0b">Запущено…</span>');
    var run = document.getElementById('nmntRun');
    if (run) run.disabled = true;
    var rb = document.getElementById('nmntReinstall');
    if (rb) rb.disabled = true;
    livePost('update_run.cgi', mode === 'reinstall' ? 'mode=reinstall' : '').then(function (t) {
      if (t && t.status === 'error') {
        nmntUpdStatus('<span style="color:#e53e3e">' + esc(t.message || '—') + '</span>');
        if (run) run.disabled = false;
        if (rb) rb.disabled = false;
        return;
      }
      nmntPoll();
    });
  }
  function setBoot() {
    if (!window.__NOC_LIVE_STATS) return;
    setTtydStatus();
    setLinksLoad();
    ntgLoad();
    nsecLoad();
    nmntInfo();
  }
  (function setBind() {
    function on(id, fn) { var el = document.getElementById(id); if (el) el.onclick = fn; }
    on('setHtopStart', function () { setTtydAct('start', 8089, 'setTermPass', null); });
    on('setHtopStop', function () { setTtydAct('stop', 8089, 'setTermPass', null); });
    on('setHtopRestart', function () { setTtydAct('restart', 8089, 'setTermPass', null); });
    on('setTermStart', function () { setTtydAct('start', 9089, 'setTermPass', null); });
    on('setTermStop', function () { setTtydAct('stop', 9089, 'setTermPass', null); });
    on('setTermRestart', function () { setTtydAct('restart', 9089, 'setTermPass', null); });
    on('setLinkAdd', function () {
      if (!setNeedSession()) return;
      var tb = document.getElementById('setLinksBody');
      if (!tb) return;
      if (tb.querySelector('.mut')) tb.innerHTML = '';
      var r = document.createElement('tr');
      r.innerHTML = '<td class="mut">⋮⋮</td><td>' + setIconSelect('link') + '</td>' +
        '<td><input class="link-name" value="Новая ссылка"></td>' +
        '<td><input class="link-url" value="http://" style="width:220px"></td>' +
        '<td><button class="btn sm" data-linksavenew="1">Сохранить</button> ' +
        '<button class="btn sm ghost" data-linkdelnew="1">Удалить</button></td>';
      tb.appendChild(r);
    });
    on('setLinksSaveAll', function () {
      var list = setLinksCollect();
      setLinksSave(list, 'Ссылки сохранены на сервер');
    });
    on('setLinksReset', function () {
      if (!confirm('Восстановить ссылки по умолчанию? Текущие будут потеряны.')) return;
      var h = window.location.hostname || '192.168.3.1';
      setLinksSave([
        { name: 'Роутер', url: 'http://' + h, icon: 'router' },
        { name: 'Entware Manager', url: '/entware-manager/', icon: 'package' },
        { name: 'htop (ttyd)', url: '/htop/', icon: 'process' },
        { name: 'Терминал (ttyd)', url: '/terminal/', icon: 'terminal' }
      ], 'Ссылки сброшены к настройкам по умолчанию');
    });
    on('ntgSave', ntgSave);
    on('ntgTest', ntgTest);
    on('nthSave', nthSave);
    on('nsecSave', nsecSave);
    on('ntlsSave', ntlsSave);
    on('nalRefresh', nalLoad);
    on('nmntOffline', nmntOffline);
    on('nmntCheck', nmntCheck);
    on('nmntRun', function () { nmntRun('update'); });
    on('nmntReinstall', function () { nmntRun('reinstall'); });
    var nrf = document.getElementById('nmntRestoreFile');
    if (nrf && !nrf.dataset.bound) {
      nrf.dataset.bound = '1';
      nrf.addEventListener('change', function () { nmntRestore(nrf); });
    }
  })();
  /* ===== LIVE rdp (вкладка RDP): config/status/start/stop/port.
   * Клиент /rdp/ (WASM-форк) не трогаем — только встраиваем iframe.
   * Пароль панели — поле в тулбаре (в URL/cookie не кладём). */
  var rdpTimer = null, rdpCfg = null, rdpFrameUrl = '', rdpFrameKey = '';
  function rdpNeedSession() { if (!window.__NOC_LIVE_STATS) { toast('Нужна сессия', 'warn'); return false; } return true; }
  function rdpPass() { var el = document.getElementById('rdpPass'); return el ? el.value : ''; }
  function rdpPassClear() { var el = document.getElementById('rdpPass'); if (el) el.value = ''; }
  function rdpRender(running, port, pid) {
    var v = document.getElementById('rdpVerdict');
    if (v) {
      v.innerHTML = '<span class="live-dot"></span>' + (running ? ('прокси :' + esc(port) + (pid ? ' · PID ' + esc(pid) : '')) : 'прокси остановлен');
      v.className = 'verdict ' + (running ? 'ok-t' : 'mut');
    }
    function dis(id, val) { var el = document.getElementById(id); if (el) el.disabled = !!val; }
    dis('rdpStartBtn', running);
    dis('rdpStopBtn', !running);
    var link = document.getElementById('rdpOpenLink');
    if (link) {
      link.hidden = !running;
      if (running && rdpFrameUrl) link.href = rdpFrameUrl;
    }
    var line = document.getElementById('rdpStatusLine');
    if (line) {
      line.innerHTML = running
        ? 'Клиент: <a href="#" id="rdpUrlToggle">показать адрес</a>' + (pid ? ' · PID ' + esc(pid) : '')
        : 'Запустите прокси, чтобы подключиться к ПК по RDP.';
      var tgl = document.getElementById('rdpUrlToggle');
      if (tgl) tgl.onclick = function (e) {
        if (e && e.preventDefault) e.preventDefault();
        tgl.textContent = tgl.getAttribute('data-full') === '1' ? 'показать адрес' : rdpFrameUrl;
        tgl.setAttribute('data-full', tgl.getAttribute('data-full') === '1' ? '0' : '1');
        return false;
      };
    }
    var wrap = document.getElementById('rdpFrameWrap');
    var key = (running ? '1' : '0') + '|' + rdpFrameUrl;
    if (wrap && key !== rdpFrameKey) {
      rdpFrameKey = key;
      wrap.hidden = !running;
      if (running) {
        var f = document.getElementById('rdpFrame');
        if (f && f.getAttribute('src') !== rdpFrameUrl) f.setAttribute('src', rdpFrameUrl);
      }
    }
  }
  function rdpStatus() {
    if (!window.__NOC_LIVE_STATS) return;
    liveFetch('rdp_status.cgi').then(function (d) {
      if (!d) return;
      var running = d.state === 'running';
      rdpRender(running, d.port || (rdpCfg && rdpCfg.proxy_port) || 9199, d.pid);
    });
  }
  function rdpBoot() {
    if (!window.__NOC_LIVE_STATS) return;
    rdpFrameUrl = window.location.protocol + '//' + window.location.host + '/rdp/?v=18';
    liveFetch('rdp_config.cgi').then(function (d) {
      if (d && d.status === 'ok') {
        rdpCfg = d;
        var p = document.getElementById('rdpPort');
        if (p) p.value = d.proxy_port || 9199;
      }
      rdpStatus();
    });
    if (!rdpTimer) rdpTimer = setInterval(function () { if (window.__NOC_LIVE_STATS) rdpStatus(); }, 10000);
  }
  (function rdpBind() {
    function on(id, fn) { var el = document.getElementById(id); if (el) el.onclick = fn; }
    on('rdpStartBtn', function () {
      if (!rdpNeedSession()) return;
      var pw = rdpPass();
      if (!pw) { toast('Введите пароль панели', 'warn'); return; }
      livePost('rdp_start.cgi', 'port=' + encodeURIComponent((rdpCfg && rdpCfg.proxy_port) || 9199) + '&password=' + encodeURIComponent(pw)).then(function (t) {
        var ok = t && t.status === 'ok';
        toast((t && t.message) || (ok ? 'Прокси запущен' : 'Ошибка'), ok ? undefined : 'warn');
        rdpPassClear();
        rdpStatus();
      });
    });
    on('rdpStopBtn', function () {
      if (!rdpNeedSession()) return;
      var pw = rdpPass();
      if (!pw) { toast('Введите пароль панели', 'warn'); return; }
      livePost('rdp_stop.cgi', 'password=' + encodeURIComponent(pw)).then(function (t) {
        var ok = t && t.status === 'ok';
        toast((t && t.message) || (ok ? 'Прокси остановлен' : 'Ошибка'), ok ? undefined : 'warn');
        rdpPassClear();
        rdpStatus();
      });
    });
    on('rdpPortSave', function () {
      if (!rdpNeedSession()) return;
      var p = document.getElementById('rdpPort'), m = document.getElementById('rdpPortMsg');
      var raw = p ? String(p.value || '').trim() : '';
      if (!/^\d+$/.test(raw)) { toast('Введите номер порта цифрами (1-65535)', 'warn'); return; }
      var port = parseInt(raw, 10);
      if (port < 1 || port > 65535) { toast('Порт должен быть в диапазоне 1-65535', 'warn'); return; }
      if (rdpCfg && rdpCfg.proxy_port === port) { toast('Этот порт уже используется'); return; }
      var pw = rdpPass();
      if (!pw) { toast('Введите пароль панели', 'warn'); return; }
      livePost('rdp_config.cgi', 'proxy_port=' + encodeURIComponent(port) + '&password=' + encodeURIComponent(pw)).then(function (t) {
        var ok = t && t.status === 'ok';
        if (m) m.textContent = ok ? 'сохранено' : 'ошибка: ' + ((t && t.message) || '—');
        toast(ok ? ('Порт прокси изменён на ' + port) : ((t && t.message) || 'Ошибка'), ok ? undefined : 'warn');
        rdpPassClear();
        if (ok) { if (rdpCfg) rdpCfg.proxy_port = port; rdpStatus(); }
      });
    });
  })();
  /* ===== LIVE terminal (вкладка Терминал): статус ttyd + iframe.
   * Запуск/остановка — в Настройках → Терминал (отдельная вкладка).
   * Iframe пересоздаём только при смене состояния, иначе фокус/ввод рвался бы. */
  var termTimer = null, termStateKey = '';
  function termRender(d) {
    if (!d) return;
    var t = d.terminal || {};
    var tMode = t.mode === 'telnet' ? 'telnet' : 'entware';
    var v = document.getElementById('termVerdict');
    if (v) {
      if (t.state === 'running') {
        v.innerHTML = '<span class="live-dot"></span>ttyd :' + esc(t.port || '—') +
          ' · ' + tMode + ' · PID ' + esc(t.pid || '—');
        v.className = 'verdict ok-t';
      } else {
        v.textContent = 'остановлен';
        v.className = 'verdict mut';
      }
    }
    var seg = document.getElementById('termModeSeg');
    if (seg) Array.prototype.forEach.call(seg.querySelectorAll('[data-tm]'), function (b) {
      if (b.getAttribute('data-tm') === tMode) b.classList.add('active');
      else b.classList.remove('active');
    });
    var link = document.getElementById('termOpenLink');
    if (link) link.hidden = t.state !== 'running';
    var wrap = document.getElementById('termFrameWrap');
    var key = (t.state || '-') + '|' + (t.port || '-') + '|' + tMode;
    if (wrap && key !== termStateKey) {
      termStateKey = key;
      if (t.state === 'running') {
        wrap.innerHTML = '<iframe id="termFrame" src="/terminal/" width="100%" height="480" style="border:none;border-radius:8px" allow="fullscreen; autoplay; clipboard-read; clipboard-write"></iframe>';
      } else {
        wrap.innerHTML = '<div class="mut mono">Терминал не запущен. Откройте <b>Настройки → Терминал</b>, задайте пароль и нажмите <b>Запустить</b>.</div>';
      }
    }
  }
  function termFetch() {
    if (!window.__NOC_LIVE_STATS) { toast('Нужна сессия', 'warn'); return; }
    liveFetch('ttyd_control.cgi').then(termRender);
  }
  /* Переключение режима прямо из вкладки Терминал (seg termModeSeg):
   * один общий пароль на оба режима — поле termPass здесь же во вкладке
   * (как в RDP), запасной вариант — поле Настроек. Отдельного пароля
   * для telnet нет. Пароль для старта обязателен (бэкенд без него
   * отказывает): если везде пусто — просим ввести здесь, никуда не уводим. */
  function termPassVal() {
    var a = document.getElementById('termPass');
    if (a && a.value) return a.value;
    var b = document.getElementById('setTermPass');
    return b ? b.value : '';
  }
  function termPassClear() {
    var a = document.getElementById('termPass');
    if (a) a.value = '';
    var b = document.getElementById('setTermPass');
    if (b) b.value = '';
  }
  function termSwitchMode(mode) {
    if (!window.__NOC_LIVE_STATS) { toast('Нужна сессия', 'warn'); return; }
    if (mode !== 'telnet' && mode !== 'entware') return;
    var pw = termPassVal();
    if (!pw) {
      toast('Введите пароль терминала', 'warn');
      var inp = document.getElementById('termPass');
      if (inp) inp.focus();
      return;
    }
    liveFetch('ttyd_control.cgi').then(function (d) {
      var t = d && d.terminal ? d.terminal : {};
      var cur = t.mode === 'telnet' ? 'telnet' : 'entware';
      if (cur === mode && t.state === 'running') {
        toast('Уже режим ' + (mode === 'telnet' ? 'Telnet' : 'Entware'));
        return;
      }
      var label = mode === 'telnet' ? 'Telnet' : 'Entware';
      var body = 'action=restart&port=9089&mode=' + encodeURIComponent(mode) +
        '&pass=' + encodeURIComponent(pw);
      toast('Переключение в режим ' + label + '…');
      livePost('ttyd_control.cgi', body).then(function (r) {
        var ok = r && r.status === 'ok';
        toast((r && r.message) || (ok ? 'Режим ' + label + ' включён' : 'Ошибка переключения'), ok ? undefined : 'warn');
        termPassClear();
        var ms = document.getElementById('setTermMode');
        if (ms) ms.value = mode;
        liveFetch('ttyd_control.cgi').then(termRender);
      });
    });
  }
  function termBoot() {
    if (!window.__NOC_LIVE_STATS) return;
    termFetch();
    if (!termTimer) termTimer = setInterval(function () {
      if (window.__NOC_LIVE_STATS) liveFetch('ttyd_control.cgi').then(termRender);
    }, 10000);
  }
  (function termBind() {
    var r = document.getElementById('termRefreshBtn');
    if (r) r.onclick = function () { termFetch(); toast('Терминал обновлён (LIVE)'); };
    var seg = document.getElementById('termModeSeg');
    if (seg) Array.prototype.forEach.call(seg.querySelectorAll('[data-tm]'), function (b) {
      b.onclick = function () { termSwitchMode(b.getAttribute('data-tm')); };
    });
  })();
  /* ===== LIVE processes (вкладка Процессы): полный слепок procs.cgi.
   * Данные уже тянет тик (applyFullProcs) — новых периодических запросов нет;
   * точечный fetch — по входу на вкладку и по кнопке. Убить — POST kill_pid.cgi. */
  function procRender() {
    var tb = document.getElementById('procBody');
    if (!tb || !window.__NOC_LIVE_STATS || !liveFullProcs.length) return;
    var byCpu = liveFullProcs.slice().sort(function (a, b) { return (liveNum(b.pcpu) || 0) - (liveNum(a.pcpu) || 0); });
    var html = '';
    byCpu.slice(0, 25).forEach(function (p) {
      html += '<tr><td>' + esc(p.pid) + '</td>' +
        '<td class="l cmd" title="' + esc(p.command) + '">' + esc(p.command) + '</td>' +
        '<td>' + esc(p.pcpu) + '</td><td>' + rssFmt(p.rss_kb) + '</td><td>' + esc(p.time || '—') + '</td>' +
        '<td><button class="btn sm ghost" data-ppid="' + esc(p.pid) + '">Завершить</button></td></tr>';
    });
    tb.innerHTML = html;
    var v = document.getElementById('procVerdict');
    if (v) {
      var d = new Date(), pad = function (n) { return (n < 10 ? '0' : '') + n; };
      v.innerHTML = '<span class="live-dot"></span>' + liveFullProcs.length + ' ' +
        plural(liveFullProcs.length, 'процесс', 'процесса', 'процессов') +
        ' · ' + pad(d.getHours()) + ':' + pad(d.getMinutes()) + ':' + pad(d.getSeconds());
    }
  }
  function procFetch() {
    if (!window.__NOC_LIVE_STATS) { toast('Нужна сессия', 'warn'); return; }
    liveFetchExp('procs.cgi').then(function (d) { applyFullProcs(d); procRender(); });
  }
  function pkgReload() {
    pkgEpoch++; /* гасим цепочку в полёте, если есть */
    pkgLoadTries = 0;
    pkgState.loaded = false;
    pkgState.installed = []; pkgState.upgradable = []; pkgState.available = [];
    livePkgLoad();
  }
  function pkgAct(endpoint, body, label) {
    pkgStatus(label + '…');
    livePost(endpoint, body, 300000).then(function (html) {
      pkgStatus(label + ': готово.\n' + htmlToText(html));
      pkgReload();
    });
  }
  function pkgStatus(t) {
    var box = document.getElementById('updateResult');
    if (box) box.textContent = t;
  }
  function pkgSummary() {
    var more = (pkgState.rowsShown || 0) > 100 ? ' · показаны первые 100 из ' + pkgState.rowsShown : '';
    return 'Установлено: ' + pkgState.installed.length + ' · к обновлению: ' + pkgState.upgradable.length +
      ' · доступно: ' + pkgState.available.length + more + '.';
  }
  function livePkgLoad() {
    /* opkg на роутере медленный: тянем цепочкой с длинными таймаутами, статус по шагам.
     * Не-массив в ответе = сбой запроса (таймаут/opkg занят): ретраим цепочку
     * до 2 раз, иначе список молча остаётся пустым навсегда (loaded уже true).
     * Эпоха my/pkgEpoch гасит устаревшие цепочки: ручное «Обновить» или ретрай
     * стартуют новую, а опоздавшие ответы старой игнорируются, а не затирают данные.
     * Рендер один — в самом конце: промежуточные перерисовки давали «прыжки» таблицы. */
    if (!window.__NOC_LIVE_STATS || pkgState.loaded) return;
    pkgState.loaded = true;
    pkgState.loading = true;
    renderPkg();
    var my = ++pkgEpoch;
    pkgStatus('Загрузка: установленные…');
    liveFetch('installed.cgi', null, 45000).then(function (d) {
      if (my !== pkgEpoch) return;
      if (!Array.isArray(d)) { pkgRetry('установленные', my); return; }
      pkgState.installed = d;
      pkgStatus('Установленные: ' + pkgState.installed.length + ' · загрузка: обновления…');
      liveFetch('upgradable.cgi', null, 45000).then(function (u) {
        if (my !== pkgEpoch) return;
        if (!Array.isArray(u)) { pkgRetry('обновления', my); return; }
        pkgState.upgradable = u;
        pkgStatus('Установленные: ' + pkgState.installed.length + ' · загрузка: доступные (долго)…');
        liveFetch('available.cgi', null, 90000).then(function (a) {
          if (my !== pkgEpoch) return;
          if (!Array.isArray(a)) { pkgRetry('доступные', my); return; }
          pkgState.available = a;
          pkgLoadTries = 0;
          pkgState.loading = false;
          renderPkg();
          pkgStatus('Готово. ' + pkgSummary());
        });
      });
    });
  }
  var pkgLoadTries = 0, pkgEpoch = 0;
  function pkgRetry(step, my) {
    /* повтор всей цепочки с нуля: частичные данные не показываем */
    if (my !== pkgEpoch) return;
    if (pkgLoadTries < 2 && window.__NOC_LIVE_STATS) {
      pkgLoadTries++;
      pkgStatus('Ошибка загрузки (' + step + '). Повтор ' + pkgLoadTries + '/2 через 5 с…');
      setTimeout(function () {
        if (my !== pkgEpoch) return;
        pkgState.loaded = false;
        pkgState.installed = []; pkgState.upgradable = []; pkgState.available = [];
        livePkgLoad();
      }, 5000);
      return;
    }
    pkgLoadTries = 0;
    pkgState.loading = false;
    renderPkg();
    pkgStatus('Ошибка загрузки (' + step + '). Нажмите «Обновить».');
  }
  function pkgRows() {
    if (!pkgState.loaded) return PKG_MOCK.slice();
    var up = {};
    pkgState.upgradable.forEach(function (u) { up[u.package] = u; });
    var inst = {};
    pkgState.installed.forEach(function (p) { inst[p.package] = p; });
    var rows = [], f = pkgState.filter, q = pkgState.q.toLowerCase();
    function pass(name) { return !q || name.toLowerCase().indexOf(q) >= 0; }
    if (f === 'all' || f === 'installed' || f === 'upgrades') {
      pkgState.installed.forEach(function (p) {
        var u = up[p.package];
        var st = u ? 'upgrade' : 'installed';
        if (f === 'installed' && st !== 'installed') return;
        if (f === 'upgrades' && st !== 'upgrade') return;
        if (!pass(p.package)) return;
        rows.push({ name: p.package, ver: u ? (p.version + ' → ' + u.new) : p.version, date: pkgShortDate(p.installed_date), st: st });
      });
    }
    if (f === 'all' || f === 'available') {
      pkgState.available.forEach(function (p) {
        if (inst[p.package]) return;
        if (!pass(p.package)) return;
        rows.push({ name: p.package, ver: p.version, date: '—', st: 'available' });
      });
    }
    var k = pkgState.sortKey, dir = pkgState.sortDir;
    rows.sort(function (a, b) {
      var va = k === 'date' ? a.date : (k === 'version' ? a.ver : a.name);
      var vb = k === 'date' ? b.date : (k === 'version' ? b.ver : b.name);
      if (va < vb) return -dir;
      if (va > vb) return dir;
      return 0;
    });
    return rows;
  }
  function renderPkg() {
    var body = document.getElementById('pkgBody');
    if (!body) return;
    if (pkgState.loading) {
      /* цепочка в полёте: скелетон вместо мок-строк и вместо пустой
       * таблицы — никаких «неправильных данных» и промежуточных прыжков */
      body.innerHTML = '<tr><td colspan="5" class="mut">Загрузка…</td></tr>';
      var vl = document.getElementById('pkgVerdictFull');
      if (vl) { vl.textContent = 'загрузка…'; vl.className = 'verdict mut'; }
      var sl = document.getElementById('pkgSubFull');
      if (sl) sl.textContent = 'LIVE · загрузка…';
      return;
    }
    var rows = pkgRows();
    var badge = { upgrade: '<span class="badge warn">обновление</span>', installed: '<span class="badge ok">установлен</span>', available: '<span class="badge">доступен</span>' };
    var act = { upgrade: '<button class="btn sm" data-pact="upgrade">Обновить</button>', installed: '<button class="btn sm ghost" data-pact="remove">Удалить</button>', available: '<button class="btn sm" data-pact="install">Установить</button>' };
    var CAP = 100, html = '';
    pkgState.rowsShown = rows.length;
    rows.slice(0, CAP).forEach(function (r) {
      html += '<tr data-pkg="' + esc(r.name) + '"><td class="l">' + esc(r.name) + '</td><td>' + esc(r.ver || '—') + '</td><td>' + esc(r.date) + '</td><td>' + badge[r.st] + '</td><td>' + act[r.st].replace('data-pact="', 'data-pkg="' + esc(r.name) + '" data-pact="') + '</td></tr>';
    });
    body.innerHTML = html || '<tr><td colspan="5" class="mut">Нет данных (нужна сессия) или пусто.</td></tr>';
    var verdictTxt = '', verdictCls = '', subTxt = '';
    if (!pkgState.loaded) { verdictTxt = '3 обновления доступны'; verdictCls = 'verdict warn-t'; }
    else if (!pkgState.installed.length && !pkgState.available.length) { verdictTxt = 'нет данных'; verdictCls = 'verdict mut'; }
    else {
      var n = pkgState.upgradable.length;
      if (n) { verdictTxt = n + ' ' + plural(n, 'обновление', 'обновления', 'обновлений') + ' доступно'; verdictCls = 'verdict warn-t'; }
      else { verdictTxt = 'всё обновлено'; verdictCls = 'verdict ok-t'; }
    }
    if (pkgState.loaded && (pkgState.installed.length || pkgState.available.length)) {
      subTxt = pkgState.installed.length + ' уст. · ' + pkgState.available.length + ' доступно · LIVE';
    }
    /* Вердикт: карточка «Пакеты» на Статистике удалена — только вкладка. */
    var vf = document.getElementById('pkgVerdictFull');
    if (vf && verdictTxt) { vf.textContent = verdictTxt; vf.className = verdictCls; }
    if (subTxt) {
      var sf = document.getElementById('pkgSubFull');
      if (sf) sf.textContent = subTxt;
    }
  }
  document.addEventListener('click', function (e) {
    var t = e.target && e.target.closest ? e.target.closest('.ri[data-tab="packages"]') : null;
    if (t) { livePkgLoad(); pkgJournalLoad(); }
    var mt = e.target && e.target.closest ? e.target.closest('.ri[data-tab="monitor"]') : null;
    if (mt) { monRefresh(); monCfgLoad(); }
    var sv = e.target && e.target.closest ? e.target.closest('.ri[data-tab="system-services"]') : null;
    if (sv) { svcBoot(); }
    var nv = e.target && e.target.closest ? e.target.closest('.ri[data-tab="network"]') : null;
    if (nv) { netBoot(); }
    var nsp = e.target && e.target.closest ? e.target.closest('[data-sub="net"] [data-sp]') : null;
    if (nsp && nsp.getAttribute('data-sp') === 'wf') { netWifi(); }
    var pv = e.target && e.target.closest ? e.target.closest('.ri[data-tab="processes"]') : null;
    if (pv) { procFetch(); }
    var tv = e.target && e.target.closest ? e.target.closest('.ri[data-tab="terminal"]') : null;
    if (tv) { termBoot(); }
    var rdpv = e.target && e.target.closest ? e.target.closest('.ri[data-tab="rdp"]') : null;
    if (rdpv) { rdpBoot(); }
    var stv = e.target && e.target.closest ? e.target.closest('.ri[data-tab="settings"]') : null;
    if (stv) { setBoot(); }
    var lgv = e.target && e.target.closest ? e.target.closest('.ri[data-tab="logs"]') : null;
    if (lgv) { logBoot(); }
    var lsave = e.target && e.target.closest ? e.target.closest('[data-linksave]') : null;
    if (lsave) {
      var li = parseInt(lsave.getAttribute('data-linksave'), 10);
      var lr = lsave.closest('tr');
      (function (row, idx) {
        if (!setNeedSession()) return;
        var n = row.querySelector('.link-name'), u = row.querySelector('.link-url'), s = row.querySelector('.link-icon-select');
        if (!n || !n.value || !u || !u.value) { toast('Заполните название и URL', 'warn'); return; }
        liveFetch('links_load.cgi').then(function (d) {
          var list = Array.isArray(d) ? d : [];
          list[idx] = { name: n.value, url: u.value, icon: s ? s.value : 'link' };
          setLinksSave(list, 'Ссылка сохранена');
        });
      })(lr, li);
      return;
    }
    var ldel = e.target && e.target.closest ? e.target.closest('[data-linkdel]') : null;
    if (ldel) {
      if (!confirm('Удалить ссылку?')) return;
      (function (idx) {
        if (!setNeedSession()) return;
        liveFetch('links_load.cgi').then(function (d) {
          var list = Array.isArray(d) ? d : [];
          list.splice(idx, 1);
          setLinksSave(list, 'Ссылка удалена');
        });
      })(parseInt(ldel.getAttribute('data-linkdel'), 10));
      return;
    }
    var lsn = e.target && e.target.closest ? e.target.closest('[data-linksavenew]') : null;
    if (lsn) {
      (function (row) {
        if (!setNeedSession()) return;
        var n = row.querySelector('.link-name'), u = row.querySelector('.link-url'), s = row.querySelector('.link-icon-select');
        if (!n || !n.value || !u || !u.value) { toast('Заполните название и URL', 'warn'); return; }
        liveFetch('links_load.cgi').then(function (d) {
          var list = Array.isArray(d) ? d : [];
          list.push({ name: n.value, url: u.value, icon: s ? s.value : 'link' });
          setLinksSave(list, 'Ссылка сохранена');
        });
      })(lsn.closest('tr'));
      return;
    }
    var ldn = e.target && e.target.closest ? e.target.closest('[data-linkdelnew]') : null;
    if (ldn) { var rmn = ldn.closest('tr'); if (rmn) rmn.remove(); return; }
    var prf = e.target && e.target.closest ? e.target.closest('#procRefreshBtn') : null;
    if (prf) { procFetch(); toast('Процессы обновлены (LIVE)'); return; }
    var ppid = e.target && e.target.closest ? e.target.closest('[data-ppid]') : null;
    if (ppid) {
      var pp = ppid.getAttribute('data-ppid');
      if (!window.__NOC_LIVE_STATS) { toast('Нужна сессия', 'warn'); return; }
      if (pp && confirm('Завершить процесс PID ' + pp + '?')) {
        livePost('kill_pid.cgi', 'pid=' + encodeURIComponent(pp)).then(function (t) {
          var ok = t && t.status === 'ok';
          toast(ok ? ('Процесс ' + pp + ' завершён') : ('Ошибка: ' + ((t && (t.error || t.message)) || '—')), ok ? undefined : 'warn');
          if (ok) procFetch();
        });
      }
      return;
    }
    var sact = e.target && e.target.closest ? e.target.closest('[data-svc]') : null;
    if (sact) { svcAct(sact.getAttribute('data-svc'), sact.getAttribute('data-sact')); return; }
    var spid = e.target && e.target.closest ? e.target.closest('[data-spid]') : null;
    if (spid) {
      if (e.preventDefault) e.preventDefault();
      var pid = spid.getAttribute('data-spid');
      if (!window.__NOC_LIVE_STATS) { toast('Нужна сессия', 'warn'); return; }
      if (pid && confirm('Убить процесс PID ' + pid + '?')) {
        livePost('kill_pid.cgi', 'pid=' + encodeURIComponent(pid)).then(function (t) {
          var ok = t && t.status === 'ok';
          toast(ok ? ('Процесс ' + pid + ' завершён') : ('Ошибка: ' + ((t && (t.error || t.message)) || '—')), ok ? undefined : 'warn');
          if (ok) svcServices();
        });
      }
      return;
    }
    var mkill = e.target && e.target.closest ? e.target.closest('[data-mpid]') : null;
    if (mkill) {
      var mp = mkill.getAttribute('data-mpid');
      if (!window.__NOC_LIVE_STATS) { toast('Нужна сессия', 'warn'); return; }
      if (mp && confirm('Убить процесс ' + mp + '?')) monAct('kill', '&pid=' + encodeURIComponent(mp));
      return;
    }
    var s = e.target && e.target.closest ? e.target.closest('#pkgSeg [data-pf]') : null;
    if (s) {
      var btns = document.querySelectorAll('#pkgSeg [data-pf]');
      Array.prototype.forEach.call(btns, function (x) { x.classList.remove('active'); });
      s.classList.add('active');
      pkgState.filter = s.getAttribute('data-pf');
      renderPkg();
      return;
    }
    var th = e.target && e.target.closest ? e.target.closest('th[data-ps]') : null;
    if (th) {
      var k = th.getAttribute('data-ps');
      if (pkgState.sortKey === k) pkgState.sortDir = -pkgState.sortDir;
      else { pkgState.sortKey = k; pkgState.sortDir = 1; }
      renderPkg();
      return;
    }
    var pact = e.target && e.target.closest ? e.target.closest('[data-pact]') : null;
    if (pact) {
      var pn = pact.getAttribute('data-pkg'), pa = pact.getAttribute('data-pact');
      if (!pn || !window.__NOC_LIVE_STATS) { toast('Нужна сессия'); return; }
      if (pa === 'upgrade') {
        if (confirm('Обновить пакет ' + pn + '?')) pkgAct('upgrade.cgi', 'package=' + encodeURIComponent(pn), 'Обновление ' + pn);
      } else if (pa === 'remove') {
        if (confirm('Удалить пакет ' + pn + '?')) pkgAct('remove.cgi', 'package=' + encodeURIComponent(pn), 'Удаление ' + pn);
      } else if (pa === 'install') {
        if (confirm('Установить пакет ' + pn + '?')) pkgAct('install.cgi', 'package=' + encodeURIComponent(pn), 'Установка ' + pn);
      }
      return;
    }
    var ef = e.target && e.target.closest ? e.target.closest('#evtBar [data-f]') : null;
    if (ef) {
      var ebtns = document.querySelectorAll('#evtBar [data-f]');
      Array.prototype.forEach.call(ebtns, function (x) { x.classList.remove('active'); });
      ef.classList.add('active');
      evtFilter = ef.getAttribute('data-f');
      try { localStorage.setItem('noc_evtf', evtFilter); } catch (err) {}
      renderEvents();
      return;
    }
    var row = e.target && e.target.closest ? e.target.closest('tr[data-pkg]') : null;
    if (row) toast(pkgInfo(row.getAttribute('data-pkg')));
  });
  /* ===== Card resize: ручка справа (ширина) + ручка снизу (высота), запоминаются ===== */
  var CARDW_KEY = 'noc_cardw2';
  var CARDH_KEY = 'noc_cardh2';
  function cardWLoad() {
    try { return JSON.parse(localStorage.getItem(CARDW_KEY) || '{}'); }
    catch (err) { return {}; }
  }
  function cardWSave(map) {
    try { localStorage.setItem(CARDW_KEY, JSON.stringify(map)); } catch (err) {}
  }
  function cardHLoad() {
    try { return JSON.parse(localStorage.getItem(CARDH_KEY) || '{}'); }
    catch (err) { return {}; }
  }
  function cardHSave(map) {
    try { localStorage.setItem(CARDH_KEY, JSON.stringify(map)); } catch (err) {}
  }
  function cardId(el) {
    return (el.dataset && el.dataset.nocid) || '';
  }
  /* Стабильные id: раздаются при каждой загрузке в порядке статического DOM
   * (до применения порядка), поэтому переживают перестановки и перезагрузки. */
  function cardBootIds() {
    var n = 0;
    Array.prototype.forEach.call(document.querySelectorAll('.card'), function (el) {
      if (el.id) el.dataset.nocid = 'id:' + el.id;
      else el.dataset.nocid = 'ix:' + (n++);
    });
  }
  function cardClamp(w, maxw) {
    w = Math.round(w);
    if (w < 240) w = 240;
    if (maxw > 0 && w > maxw) w = maxw;
    return w;
  }
  function cardClampH(h) {
    h = Math.round(h);
    if (h < 120) h = 120;
    return h;
  }
  function cardResizeBoot() {
    var saved = cardWLoad();
    var savedH = cardHLoad();
    var cards = document.querySelectorAll('.card');
    Array.prototype.forEach.call(cards, function (el) {
      var id = cardId(el);
      var st = el.style;
      if (saved[id]) { st.width = saved[id] + 'px'; st.maxWidth = '100%'; el.classList.add('card-fixedw'); }
      if (savedH[id]) { st.height = savedH[id] + 'px'; el.classList.add('card-fixedh'); }
      if (el.querySelector(':scope > .card-resizer')) return;
      var h = document.createElement('div');
      h.className = 'card-resizer';
      h.title = 'Тяните, чтобы изменить ширину (двойной клик — сброс)';
      el.appendChild(h);
      h.addEventListener('pointerdown', function (e) {
        e.preventDefault();
        var startX = e.clientX;
        var startW = el.getBoundingClientRect().width;
        var maxw = el.parentElement ? el.parentElement.clientWidth : 0;
        h.classList.add('on');
        try { h.setPointerCapture(e.pointerId); } catch (err) {}
        var move = function (ev) {
          st.width = cardClamp(startW + ev.clientX - startX, maxw) + 'px';
          st.maxWidth = '100%';
          el.classList.add('card-fixedw');
        };
        var up = function () {
          h.classList.remove('on');
          h.removeEventListener('pointermove', move);
          h.removeEventListener('pointerup', up);
          h.removeEventListener('pointercancel', up);
          var m = cardWLoad();
          m[id] = parseInt(st.width, 10) || 0;
          if (!m[id]) delete m[id];
          cardWSave(m);
        };
        h.addEventListener('pointermove', move);
        h.addEventListener('pointerup', up);
        h.addEventListener('pointercancel', up);
      });
      h.title = 'Тяните, чтобы изменить ширину (двойной клик — размер по сетке: вся / половина / треть)';
      h.addEventListener('dblclick', function (e) {
        e.preventDefault();
        /* сетка из трёх: вся ширина, половина, треть контейнера (минус зазоры).
         * Первый клик — подстроить под ближайший, повторный — следующий по кругу,
         * после трети — возврат к авто (как было).
         * Ограничитель: если в ряду уже 2+ карточки — полную ширину не
         * предлагаем (иначе ряд развалится); только половина/треть. */
        var rowN = 1;
        try {
          var sibs = contCards(el.parentElement);
          var top = el.offsetTop, n = 0;
          sibs.forEach(function (x) { if (Math.abs(x.offsetTop - top) < 4) n++; });
          if (n > 0) rowN = n;
        } catch (err) {}
        var contW = el.parentElement ? el.parentElement.clientWidth : 0;
        if (contW <= 0) contW = el.getBoundingClientRect().width;
        /* запас вниз: round округлял вверх и 2×половины / 3×трети давали
         * на 1–2px больше контейнера — карточка уходила на другую строку,
         * хотя плейсхолдер показывал место. Считаем floor с запасом 8px
         * (пробелы inline-block + отступы) и ещё −1px для надёжности. */
        var gap = 14;
        var half = Math.floor((contW - gap - 8) / 2) - 1;
        var third = Math.floor((contW - gap * 2 - 8) / 3) - 1;
        if (half < 120) half = 120;
        if (third < 120) third = 120;
        var targets = rowN >= 2 ? [half, third] : [Math.round(contW), half, third];
        var cur = parseInt(st.width, 10) || el.getBoundingClientRect().width;
        var best = 0, bestD = Math.abs(cur - targets[0]);
        targets.forEach(function (t, i) {
          var d = Math.abs(cur - t);
          if (d < bestD) { bestD = d; best = i; }
        });
        var m = cardWLoad();
        var next;
        if (bestD > 12) next = best;
        else next = (best + 1) % targets.length;
        /* «вся ширина» — только индекс 0 полного списка; в ограниченном
         * списке (ряд из 2+) нулевой элемент — половина, её ставим, не сброс. */
        if (next === 0 && rowN < 2) {
          st.width = '';
          st.maxWidth = '';
          el.classList.remove('card-fixedw');
          delete m[id];
        } else {
          st.width = targets[next] + 'px';
          st.maxWidth = '100%';
          el.classList.add('card-fixedw');
          m[id] = targets[next];
        }
        cardWSave(m);
      });
      /* ручка высоты снизу — тот же механизм по вертикали */
      if (el.querySelector(':scope > .card-resizer-b')) return;
      var hb = document.createElement('div');
      hb.className = 'card-resizer-b';
      hb.title = 'Тяните, чтобы изменить высоту (двойной клик — сброс)';
      el.appendChild(hb);
      hb.addEventListener('pointerdown', function (e) {
        e.preventDefault();
        var startY = e.clientY;
        var startH = el.getBoundingClientRect().height;
        hb.classList.add('on');
        try { hb.setPointerCapture(e.pointerId); } catch (err) {}
        var moveB = function (ev) {
          st.height = cardClampH(startH + ev.clientY - startY) + 'px';
          el.classList.add('card-fixedh');
        };
        var upB = function () {
          hb.classList.remove('on');
          hb.removeEventListener('pointermove', moveB);
          hb.removeEventListener('pointerup', upB);
          hb.removeEventListener('pointercancel', upB);
          var m = cardHLoad();
          m[id] = parseInt(st.height, 10) || 0;
          if (!m[id]) delete m[id];
          cardHSave(m);
        };
        hb.addEventListener('pointermove', moveB);
        hb.addEventListener('pointerup', upB);
        hb.addEventListener('pointercancel', upB);
      });
      hb.addEventListener('dblclick', function (e) {
        e.preventDefault();
        st.height = '';
        el.classList.remove('card-fixedh');
        var m = cardHLoad();
        delete m[id];
        cardHSave(m);
      });
    });
  }
  /* ===== Card order: перенос за шапку, порядок запоминается ===== */
  /* Только статические карточки (динамические сетки перерисовываются сами). */
  var CORD_KEY = 'noc_cardorder2';
  function cardOrderLoad() {
    try { return JSON.parse(localStorage.getItem(CORD_KEY) || '{}'); }
    catch (err) { return {}; }
  }
  function cardOrderSave(map) {
    try { localStorage.setItem(CORD_KEY, JSON.stringify(map)); } catch (err) {}
  }
  function contKey(cont) {
    if (cont.dataset && cont.dataset.nocont) return 'cont:' + cont.dataset.nocont;
    if (cont.id) return 'id:' + cont.id;
    var sibs = cont.parentElement ? cont.parentElement.children : [];
    var ix = Array.prototype.indexOf.call(sibs, cont);
    return 'ix:' + cont.tagName + ':' + ix;
  }
  function contCards(cont) {
    var out = [];
    Array.prototype.forEach.call(cont.children, function (x) {
      if (x.classList && x.classList.contains('card')) out.push(x);
    });
    return out;
  }
  function cardApplyOrder() {
    var saved = cardOrderLoad();
    var keys = Object.keys(saved);
    if (!keys.length) return;
    var byId = {};
    var contByKey = {};
    Array.prototype.forEach.call(document.querySelectorAll('.card'), function (el) {
      byId[cardId(el)] = el;
      if (el.parentElement) contByKey[contKey(el.parentElement)] = el.parentElement;
    });
    keys.forEach(function (ck) {
      var ids = saved[ck];
      var cont = contByKey[ck];
      if (!cont || !ids || !ids.length) return;
      ids.forEach(function (id) {
        var el = byId[id];
        /* appendChild перемещает и между контейнерами — так порядок
         * переживает перезагрузку и при переносе через строки.
         * Чужой view не трогаем: после изменения DOM старые ключи
         * контейнеров могут указывать не туда (карточки сыпались
         * в чужую вкладку, напр. Мониторинг в Терминал). */
        if (el && sameView(el.parentElement, cont)) cont.appendChild(el);
      });
    });
  }
  /* Контейнеры одного view: перенос между строками разрешён. */
  function sameView(a, b) {
    if (!a || !b) return false;
    var va = a.closest ? a.closest('[id^="view-"]') : null;
    var vb = b.closest ? b.closest('[id^="view-"]') : null;
    if (!va && !vb) return true;
    return !!va && va === vb;
  }
  function cardOrderBoot() {
    cardBootIds();
    /* маркировка раньше apply: ключи cont:N должны существовать до чтения */
    var markCards = document.querySelectorAll('.card');
    var ncont = 0;
    Array.prototype.forEach.call(markCards, function (el) {
      if (el.parentElement) {
        el.parentElement.dataset.dndcont = '1';
        /* свободная раскладка — ТОЛЬКО двухколоночным секциям
         * (section.grid.two): служебные контейнеры главной
         * (#view-stats, .stats-main, .panel-row) трогать нельзя —
         * flex ломает их вёрстку. В блочных контейнерах суженые
         * карточки встают в ряд через inline-block (CSS .card-fixedw). */
        if (el.parentElement.classList.contains('two')) el.parentElement.classList.add('dnd-flow');
        if (!el.parentElement.dataset.nocont) el.parentElement.dataset.nocont = '' + (ncont++);
      }
    });
    cardApplyOrder();
    var cards = document.querySelectorAll('.card');
    Array.prototype.forEach.call(cards, function (el) {
      var head = el.querySelector(':scope > .panel-h, :scope > .c-head');
      if (!head || head.dataset.dnd) return;
      head.dataset.dnd = '1';
      head.addEventListener('pointerdown', function (e) {
        if (e.button !== undefined && e.button !== 0) return;
        if (e.target.closest && e.target.closest('button,a,input,select,textarea,label')) return;
        /* мышь: давим нативное выделение — иначе оно перехватывает drag;
         * тач не трогаем (там скролл регулирует touch-action в CSS). */
        if (!e.pointerType || e.pointerType === 'mouse') {
          try { e.preventDefault(); } catch (err) {}
        }
        var startX = e.clientX, startY = e.clientY;
        var dragging = false;
        var ph = null;
        var rect = null;
        var srcCont = el.parentElement;
        var move = function (ev) {
          if (!dragging) {
            if (Math.abs(ev.clientX - startX) + Math.abs(ev.clientY - startY) < 6) return;
            dragging = true;
            rect = el.getBoundingClientRect();
            /* захват указателя шапкой: move/up доходят даже за пределами
             * окна — иначе отпуск кнопки теряется и карточка «зависает». */
            try { head.setPointerCapture(e.pointerId); } catch (err) {}
            ph = document.createElement('div');
            ph.className = 'card-placeholder';
            ph.style.height = rect.height + 'px';
            /* суженая карточка — плейсхолдер того же размера, иначе в ряд не встать */
            if (el.classList.contains('card-fixedw') && el.style.width) {
              ph.style.width = el.style.width;
              ph.style.flex = 'none';
            }
            /* grid-контейнер: для высокой карточки (card-fixedh) плейсхолдер на 2 ряда */
            el.parentElement.insertBefore(ph, el);
            el.classList.add('card-dragging');
            document.body.classList.add('card-dragging');
            el.style.position = 'fixed';
            el.style.left = rect.left + 'px';
            el.style.top = rect.top + 'px';
            el.style.width = rect.width + 'px';
            el.style.margin = '0';
            el.style.zIndex = '50';
          }
          el.style.left = (rect.left + ev.clientX - startX) + 'px';
          el.style.top = (rect.top + ev.clientY - startY) + 'px';
          /* Попадание считаем сами по rect'ам — без elementFromPoint:
           * хит-тест окружения ненадёжен (плейсхолдер не двигался при
           * живых событиях). Скрытые view (нулевые rect) пропускаем. */
          var placed = false;
          Array.prototype.forEach.call(document.querySelectorAll('.card'), function (other) {
            if (placed || other === el || !other.parentElement) return;
            var r = other.getBoundingClientRect();
            if (!r || r.width <= 0 || r.height <= 0) return;
            if (ev.clientX < r.left - 8 || ev.clientX > r.right + 8 ||
                ev.clientY < r.top - 8 || ev.clientY > r.bottom + 8) return;
            var oc = other.parentElement, pc = ph.parentElement;
            if (!oc || !pc) return;
            /* чужой контейнер — только в том же view и только помеченный
             * (статические карточки); динамические сетки не трогаем. */
            if (oc !== pc && (!sameView(oc, pc) || !oc.dataset.dndcont || !pc.dataset.dndcont)) return;
            /* куда вставить: у верхнего/нижнего края — по вертикали (в ряд
             * выше/ниже), в средней полосе карточки — по горизонтали
             * (слева/справа в том же ряду). Иначе 3 в ряд не собрать:
             * сброс правее карточки уходил бы строкой выше. */
            var yFrac = r.height > 0 ? (ev.clientY - r.top) / r.height : 0.5;
            var before;
            if (yFrac > 0.25 && yFrac < 0.75) before = (ev.clientX - r.left) < r.width / 2;
            else before = yFrac < 0.5;
            oc.insertBefore(ph, before ? other : other.nextSibling);
            placed = true;
          });
          if (!placed) {
            /* пустое место контейнера — placeholder в конец (сброс туда же);
             * точку под курсором не трогаем, если это сама тащимая карточка. */
            var under2 = null;
            try { under2 = document.elementFromPoint(ev.clientX, ev.clientY); } catch (err2) {}
            if (under2 && under2.closest && !(el.contains && el.contains(under2))) {
              var ec = under2.closest('[data-dndcont]');
              if (ec && sameView(ec, srcCont) && ec !== ph.parentElement) ec.appendChild(ph);
            }
          }
        };
        var up = function () {
          document.removeEventListener('pointermove', move);
          document.removeEventListener('pointerup', up);
          document.removeEventListener('pointercancel', up);
          if (!dragging) return;
          el.classList.remove('card-dragging');
          document.body.classList.remove('card-dragging');
          el.style.position = '';
          el.style.left = '';
          el.style.top = '';
          el.style.margin = '';
          el.style.zIndex = '';
          /* fixed-режим затирал ширину: вернуть запомненную, иначе сужение слетает */
          var wmap = cardWLoad();
          var wid = cardId(el);
          if (wmap[wid]) {
            el.style.width = wmap[wid] + 'px';
            el.style.maxWidth = '100%';
            el.classList.add('card-fixedw');
          } else {
            el.style.width = '';
            el.classList.remove('card-fixedw');
          }
          var cont = ph.parentElement;
          cont.insertBefore(el, ph);
          cont.removeChild(ph);
          /* сохраняем оба затронутых контейнера (источник мог отдать карточку) */
          var map = cardOrderLoad();
          [srcCont, cont].forEach(function (c) {
            if (!c || !c.dataset.dndcont) return;
            var ids = contCards(c).map(function (x) { return cardId(x); });
            map[contKey(c)] = ids;
          });
          cardOrderSave(map);
        };
        /* move/up — на document: без захвата указателя шапка перестаёт
         * получать события, как только карточка уходит из-под курсора
         * (fixed + pointer-events:none) — иначе перенос зависает. */
        document.addEventListener('pointermove', move);
        document.addEventListener('pointerup', up);
        document.addEventListener('pointercancel', up);
      });
    });
  }
  var pkgSearch = document.getElementById('pkgSearch');
  if (pkgSearch) pkgSearch.oninput = function (e) { pkgState.q = e.target.value || ''; renderPkg(); };
  var pkgRefreshBtn = document.getElementById('pkgRefreshBtn');
  if (pkgRefreshBtn) pkgRefreshBtn.onclick = function () {
    pkgState.q = '';
    if (pkgSearch) pkgSearch.value = '';
    pkgReload();
    toast('Список пакетов обновляется (LIVE)');
  };
  renderPkg();
  cardOrderBoot();
  cardResizeBoot();
  (function evtBarBoot() {
    var cur = document.querySelector('#evtBar [data-f="' + evtFilter + '"]');
    if (cur) {
      var all = document.querySelectorAll('#evtBar [data-f]');
      Array.prototype.forEach.call(all, function (x) { x.classList.remove('active'); });
      cur.classList.add('active');
    }
  })();
  /* ===== Запоминание вкладки: активный таб rail + подтаб логов.
   * Клик по rail сохраняет таб; при загрузке восстанавливаем его вместо
   * стартового. bootLiveTab дожимает LIVE-загрузчики активной вкладки
   * после входа (к моменту liveStart таб уже восстановлен). */
  function bootLiveTab() {
    var act = document.querySelector('.ri.active');
    var t = act ? act.getAttribute('data-tab') : 'stats';
    if (t === 'logs') logBoot();
    else if (t === 'system-services') svcBoot();
    else if (t === 'network') netBoot();
    else if (t === 'monitor') { monRefresh(); monCfgLoad(); }
    else if (t === 'packages') { livePkgLoad(); pkgJournalLoad(); }
    else if (t === 'processes') procFetch();
    else if (t === 'terminal') termBoot();
    else if (t === 'rdp') rdpBoot();
    else if (t === 'settings') setBoot();
  }
  (function restoreTab() {
    var t = lsGet('entware_noc_tab');
    if (!t || !VIEWS[t]) return;
    var b = document.querySelector('.ri[data-tab="' + t + '"]');
    if (!b) return;
    b.click();
    if (t === 'logs') {
      var lt = lsGet('entware_noc_logtab');
      if (lt === 'sys' || lt === 'mgr') logSetTab(lt);
    }
  })();
  liveBoot();
  applyAll();
  /* ResizeObserver: при изменении ширины контейнера (rail hover) пропорционально
   * подстраиваем зафиксированные ширины карточек, чтобы они не вылезали за края
   * и сохраняли относительные пропорции. */
  (function cardResizeObserver() {
    var main = document.querySelector('.main');
    if (!main || !window.ResizeObserver) return;
    var lastW = main.clientWidth;
    var ro = new ResizeObserver(function () {
      var w = main.clientWidth;
      if (w === lastW || w === 0) return;
      var ratio = w / lastW;
      lastW = w;
      var wmap = cardWLoad();
      var changed = false;
      document.querySelectorAll('.card-fixedw').forEach(function (el) {
        var id = cardId(el);
        var cur = wmap[id];
        if (!cur) return;
        var nw = Math.round(cur * ratio);
        if (nw < 240) nw = 240;
        if (nw !== cur) {
          el.style.width = nw + 'px';
          wmap[id] = nw;
          changed = true;
        }
      });
      if (changed) cardWSave(wmap);
    });
    ro.observe(main);
  })();
})();
