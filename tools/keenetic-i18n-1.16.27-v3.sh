#!/bin/sh
set -eu

ROOT="/opt/web_entware"
INDEX="$ROOT/index.html"
BACKUP="$ROOT/index.html.before-local-i18n-v3"
TMP_CLEAN="/tmp/ewm-i18n-v3-clean.$$.html"
TMP_BLOCK="/tmp/ewm-i18n-v3-block.$$.html"
TMP_INDEX="/tmp/ewm-i18n-v3-index.$$.html"

cleanup() {
    rm -f "$TMP_CLEAN" "$TMP_BLOCK" "$TMP_INDEX"
}
trap cleanup EXIT INT TERM

[ -f "$INDEX" ] || { echo "ERROR: $INDEX bulunamadi"; exit 1; }
grep -q 'EWM_LOCAL_I18N_BEGIN' "$INDEX" 2>/dev/null || {
    echo "ERROR: temel i18n patch bulunamadi."
    exit 1
}

if [ ! -f "$BACKUP" ]; then
    cp -p "$INDEX" "$BACKUP"
    echo "Backup: $BACKUP"
fi

# V2'de tek harf/alt-dize cevirisi Rusca kelimeleri bozabiliyordu.
# V2 ve varsa eski V3 blogunu kaynaktan tamamen kaldiriyoruz.
sed \
  -e '/<!-- EWM_LOCAL_I18N_V2_BEGIN -->/,/<!-- EWM_LOCAL_I18N_V2_END -->/d' \
  -e '/<!-- EWM_LOCAL_I18N_V3_BEGIN -->/,/<!-- EWM_LOCAL_I18N_V3_END -->/d' \
  "$INDEX" > "$TMP_CLEAN"

cat > "$TMP_BLOCK" <<'EOB'
<!-- EWM_LOCAL_I18N_V3_BEGIN -->
<style id="ewm-local-i18n-v3-style">
.ewm-language-select {
  background:#fff !important;
  color:#111827 !important;
  border-color:#cbd5e1 !important;
  color-scheme:light !important;
}
.ewm-language-select option { background:#fff !important; color:#111827 !important; }
html.night .ewm-language-select,
body.night .ewm-language-select,
[data-theme="dark"] .ewm-language-select {
  background:#1f2937 !important;
  color:#f8fafc !important;
  border-color:#475569 !important;
  color-scheme:dark !important;
}
html.night .ewm-language-select option,
body.night .ewm-language-select option,
[data-theme="dark"] .ewm-language-select option { background:#1f2937 !important; color:#f8fafc !important; }
</style>
<script id="ewm-local-i18n-v3">
(function () {
  'use strict';

  var MAPS = {
    tr: {
      'Статистика системы':'Sistem istatistikleri',
      'Обзор ресурсов, пакетов и дисков роутера':'Yönlendiricinin kaynak, paket ve disk özeti',
      'Система':'Sistem',
      'Модель:':'Model:',
      'Имя хоста:':'Ana bilgisayar adı:',
      'Архитектура:':'Mimari:',
      'Версия ядра:':'Çekirdek sürümü:',
      'Время работы:':'Çalışma süresi:',
      'Память (RAM)':'Bellek (RAM)',
      'Использовано / Всего:':'Kullanılan / Toplam:',
      'Загрузка:':'Kullanım:',
      'Средняя нагрузка:':'Ortalama yük:',
      'Топ по памяти':'Bellek kullanımında ilkler',
      'Процессор (CPU)':'İşlemci (CPU)',
      'Топ по CPU':'CPU kullanımında ilkler',
      'Пакеты Entware':'Entware paketleri',
      'Установлено:':'Kurulu:',
      'Доступно:':'Mevcut:',
      'Диск (/opt)':'Disk (/opt)',
      'Размер:':'Boyut:',
      'Использовано:':'Kullanılan:',
      'Обновить списки пакетов':'Paket listelerini yenile',
      'Обновить все пакеты':'Tüm paketleri güncelle',
      'Поиск по названию...':'Ada göre ara...',
      'Процессы (htop)':'İşlemler (htop)',
      'htop не запущен. Откройте Настройки → Терминал, задайте пароль и нажмите Запустить.':'htop çalışmıyor. Ayarlar → Terminal bölümünü açın, parola belirleyin ve Başlat düğmesine basın.',
      'Терминал не запущен. Откройте Настройки → Терминал, задайте пароль и нажмите Запустить.':'Terminal çalışmıyor. Ayarlar → Terminal bölümünü açın, parola belirleyin ve Başlat düğmesine basın.',
      'Системные службы и планировщик':'Sistem servisleri ve zamanlayıcı',
      'Мониторинг:':'İzleme:',
      'Автозапуск при загрузке':'Açılışta otomatik başlat',
      'Кастомный список':'Özel liste',
      'Исключения':'İstisnalar',
      'Автоперезапуск служб при падении':'Servis çökerse otomatik yeniden başlat',
      'Последние события мониторинга':'Son izleme olayları',
      'Нет событий':'Olay yok',
      'Службы (init.d)':'Servisler (init.d)',
      'СЛУЖБА':'SERVİS',
      'СТАТУС':'DURUM',
      'АВТОЗАПУСК':'OTOMATİK BAŞLATMA',
      'ДЕЙСТВИЯ':'İŞLEMLER',
      'Авто':'Oto',
      'Демон:':'Daemon:',
      'Интерфейсы':'Arayüzler',
      'События':'Olaylar',
      'АРП':'ARP',
      'IP адрес':'IP adresi',
      'IP АДРЕС':'IP ADRESİ',
      'ТИП':'TÜR',
      'СКОРОСТЬ':'HIZ',
      'IPIP-туннель':'IPIP tüneli',
      'IPv6-туннель':'IPv6 tüneli',
      'SIT-туннель':'SIT tüneli',
      'Локальные сервисы Entware, обнаруженные на этом роутере. Ползунок включает/выключает модуль в панели, галочка управляет уведомлениями о падении/восстановлении сервиса.':'Bu yönlendiricide bulunan yerel Entware servisleri. Anahtar modülü panelde açıp kapatır; onay kutusu servis durma/geri gelme bildirimlerini yönetir.',
      'не установлен':'kurulu değil',
      'уведомления':'bildirimler',
      'процесс не найден':'işlem bulunamadı',
      'управление':'yönetim',
      'Пересканировать':'Yeniden tara',
      'Манифесты':'Manifestolar',
      'Терминал ttyd':'ttyd terminali',
      'сервис не отвечает':'servis yanıt vermiyor',
      'SMART дисков':'Disk SMART',
      'Поиск по модели/серийному...':'Model/seri numarasına göre ara...',
      'УСТРОЙСТВО':'CİHAZ',
      'МОДЕЛЬ':'MODEL',
      'СЕРИЙНЫЙ №':'SERİ NO',
      'РАЗМЕР':'BOYUT',
      'Прокси не запущен':'Proxy çalışmıyor',
      'Порт:':'Port:',
      'Запустите прокси, чтобы подключиться к ПК по RDP.':'RDP ile bilgisayara bağlanmak için proxy\'yi başlatın.',
      'Запустить прокси':'Proxy\'yi başlat',
      'Терминал и ссылки':'Terminal ve bağlantılar',
      'Уведомления':'Bildirimler',
      'Безопасность':'Güvenlik',
      'Обслуживание':'Bakım',
      'Управление ttyd':'ttyd yönetimi',
      'Текущее состояние ttyd':'Mevcut ttyd durumu',
      'htop (8089, доступ /htop/):':'htop (8089, erişim /htop/):',
      'Терминал (9089, доступ /terminal/):':'Terminal (9089, erişim /terminal/):',
      'htop (порт 8089)':'htop (port 8089)',
      'Терминал (порт 9089)':'Terminal (port 9089)',
      'Консоль Entware':'Entware konsolu',
      'Управление веб-терминалами ttyd. Пароль обязателен для обоих сервисов — терминал доступен извне через панель, без пароля запуск запрещён.':'ttyd web terminallerini yönetir. Her iki servis için parola zorunludur; terminal panele üzerinden erişilebilir ve parola olmadan başlatılamaz.',
      'Доступ: панель → /terminal/ и /htop/ (тот же origin, порты 9089/8089 слушают только loopback).':'Erişim: panel → /terminal/ ve /htop/ (aynı origin; 9089/8089 portları yalnızca loopback üzerinde dinler).',
      'Управление ссылками на главной (общие для всех устройств)':'Ana sayfa bağlantılarını yönet (tüm cihazlar için ortak)',
      'Здесь можно добавлять, редактировать и удалять ссылки. Изменения сразу видны на всех устройствах.':'Buradan bağlantı ekleyebilir, düzenleyebilir ve silebilirsiniz. Değişiklikler tüm cihazlarda hemen görünür.',
      'Добавить ссылку':'Bağlantı ekle',
      'ИКОНКА':'SİMGE',
      'НАЗВАНИЕ':'AD',
      'Иконка':'Simge',
      'Название':'Ad',
      'Действия':'İşlemler',
      'Роутер':'Yönlendirici',
      'Щит':'Kalkan',
      'Загрузка':'İndirme',
      'График':'Grafik',
      'Процесс':'İşlem',
      'Сохранить все на сервер':'Tümünü sunucuya kaydet',
      'Сбросить по умолчанию':'Varsayılana sıfırla',
      'Защита от зависших процессов':'Takılı kalan işlemlere karşı koruma',
      'Статус демона:':'Daemon durumu:',
      'остановлен':'durduruldu',
      'запущен':'çalışıyor',
      'Топ процессов по нагрузке CPU':'CPU yüküne göre en yoğun işlemler',
      'CPU время':'CPU süresi',
      'Команда':'Komut',
      'Убить':'Sonlandır',
      'Настройки защиты':'Koruma ayarları',
      'Включить защиту (глобально)':'Korumayı etkinleştir (genel)',
      'Интервал сканирования (сек):':'Tarama aralığı (sn):',
      'Индивидуальный режим':'Bireysel mod',
      'Порог CPU (%):':'CPU eşiği (%):',
      'Время непрерывной нагрузки (мин):':'Kesintisiz yüksek yük süresi (dk):',
      'Игнорируемые процессы (имена, через запятую):':'Yok sayılacak işlemler (adlar, virgülle):',
      'Исключать ps из мониторинга (убирает ложные предупреждения)':'ps işlemini izlemeden çıkar (yanlış uyarıları azaltır)',
      'Дополнительные настройки':'Ek ayarlar',
      'Максимум процессов для сканирования:':'Taranacak azami işlem sayısı:',
      'Лог событий':'Olay günlüğü',
      'Лог-файл не найден':'Günlük dosyası bulunamadı',
      'Очистить лог':'Günlüğü temizle',
      'Действия менеджера':'Yönetici işlemleri',
      'Системные логи':'Sistem günlükleri',
      'Очистить логи старше 30 дней':'30 günden eski günlükleri temizle',
      'Ротация сейчас':'Şimdi döndür',
      'Настройки логирования':'Günlükleme ayarları',
      'Системные события':'Sistem olayları'
    },
    en: {
      'Статистика системы':'System statistics',
      'Обзор ресурсов, пакетов и дисков роутера':'Router resources, packages and disks overview',
      'Система':'System',
      'Модель:':'Model:',
      'Имя хоста:':'Host name:',
      'Архитектура:':'Architecture:',
      'Версия ядра:':'Kernel version:',
      'Время работы:':'Uptime:',
      'Память (RAM)':'Memory (RAM)',
      'Использовано / Всего:':'Used / Total:',
      'Загрузка:':'Usage:',
      'Средняя нагрузка:':'Average load:',
      'Топ по памяти':'Top by memory',
      'Процессор (CPU)':'Processor (CPU)',
      'Топ по CPU':'Top by CPU',
      'Пакеты Entware':'Entware packages',
      'Установлено:':'Installed:',
      'Доступно:':'Available:',
      'Диск (/opt)':'Disk (/opt)',
      'Размер:':'Size:',
      'Использовано:':'Used:',
      'Обновить списки пакетов':'Refresh package lists',
      'Обновить все пакеты':'Update all packages',
      'Поиск по названию...':'Search by name...',
      'Процессы (htop)':'Processes (htop)',
      'htop не запущен. Откройте Настройки → Терминал, задайте пароль и нажмите Запустить.':'htop is not running. Open Settings → Terminal, set a password and press Start.',
      'Терминал не запущен. Откройте Настройки → Терминал, задайте пароль и нажмите Запустить.':'Terminal is not running. Open Settings → Terminal, set a password and press Start.',
      'Системные службы и планировщик':'System services and scheduler',
      'Мониторинг:':'Monitoring:',
      'Автозапуск при загрузке':'Autostart on boot',
      'Кастомный список':'Custom list',
      'Исключения':'Exclusions',
      'Автоперезапуск служб при падении':'Auto-restart services on failure',
      'Последние события мониторинга':'Latest monitoring events',
      'Нет событий':'No events',
      'Службы (init.d)':'Services (init.d)',
      'СЛУЖБА':'SERVICE',
      'СТАТУС':'STATUS',
      'АВТОЗАПУСК':'AUTOSTART',
      'ДЕЙСТВИЯ':'ACTIONS',
      'Авто':'Auto',
      'Демон:':'Daemon:',
      'Интерфейсы':'Interfaces',
      'События':'Events',
      'IP адрес':'IP address',
      'IP АДРЕС':'IP ADDRESS',
      'ТИП':'TYPE',
      'СКОРОСТЬ':'SPEED',
      'IPIP-туннель':'IPIP tunnel',
      'IPv6-туннель':'IPv6 tunnel',
      'SIT-туннель':'SIT tunnel',
      'не установлен':'not installed',
      'уведомления':'notifications',
      'процесс не найден':'process not found',
      'управление':'management',
      'Пересканировать':'Rescan',
      'Манифесты':'Manifests',
      'Терминал ttyd':'ttyd terminal',
      'сервис не отвечает':'service is not responding',
      'SMART дисков':'Disk SMART',
      'Поиск по модели/серийному...':'Search by model/serial...',
      'УСТРОЙСТВО':'DEVICE',
      'МОДЕЛЬ':'MODEL',
      'СЕРИЙНЫЙ №':'SERIAL NO.',
      'РАЗМЕР':'SIZE',
      'Прокси не запущен':'Proxy is not running',
      'Порт:':'Port:',
      'Запустите прокси, чтобы подключиться к ПК по RDP.':'Start the proxy to connect to a PC over RDP.',
      'Запустить прокси':'Start proxy',
      'Терминал и ссылки':'Terminal & links',
      'Уведомления':'Notifications',
      'Безопасность':'Security',
      'Обслуживание':'Maintenance',
      'Управление ttyd':'ttyd management',
      'Текущее состояние ttyd':'Current ttyd status',
      'htop (8089, доступ /htop/):':'htop (8089, access /htop/):',
      'Терминал (9089, доступ /terminal/):':'Terminal (9089, access /terminal/):',
      'htop (порт 8089)':'htop (port 8089)',
      'Терминал (порт 9089)':'Terminal (port 9089)',
      'Консоль Entware':'Entware console',
      'Управление ссылками на главной (общие для всех устройств)':'Manage home-page links (shared by all devices)',
      'Здесь можно добавлять, редактировать и удалять ссылки. Изменения сразу видны на всех устройствах.':'Add, edit and remove links here. Changes are immediately visible on all devices.',
      'Добавить ссылку':'Add link',
      'ИКОНКА':'ICON',
      'НАЗВАНИЕ':'NAME',
      'Иконка':'Icon',
      'Название':'Name',
      'Действия':'Actions',
      'Роутер':'Router',
      'Щит':'Shield',
      'Загрузка':'Download',
      'График':'Chart',
      'Процесс':'Process',
      'Сохранить все на сервер':'Save all to server',
      'Сбросить по умолчанию':'Reset to default',
      'Защита от зависших процессов':'Hung process protection',
      'Статус демона:':'Daemon status:',
      'остановлен':'stopped',
      'запущен':'running',
      'Топ процессов по нагрузке CPU':'Top processes by CPU load',
      'CPU время':'CPU time',
      'Команда':'Command',
      'Убить':'Kill',
      'Настройки защиты':'Protection settings',
      'Включить защиту (глобально)':'Enable protection (global)',
      'Интервал сканирования (сек):':'Scan interval (sec):',
      'Индивидуальный режим':'Individual mode',
      'Порог CPU (%):':'CPU threshold (%):',
      'Время непрерывной нагрузки (мин):':'Continuous load time (min):',
      'Игнорируемые процессы (имена, через запятую):':'Ignored processes (names, comma-separated):',
      'Исключать ps из мониторинга (убирает ложные предупреждения)':'Exclude ps from monitoring (reduces false warnings)',
      'Дополнительные настройки':'Additional settings',
      'Максимум процессов для сканирования:':'Maximum processes to scan:',
      'Лог событий':'Event log',
      'Лог-файл не найден':'Log file not found',
      'Очистить лог':'Clear log',
      'Действия менеджера':'Manager actions',
      'Системные логи':'System logs',
      'Очистить логи старше 30 дней':'Clear logs older than 30 days',
      'Ротация сейчас':'Rotate now',
      'Настройки логирования':'Logging settings',
      'Системные события':'System events'
    }
  };

  function currentLanguage() {
    try {
      var v = localStorage.getItem('entware_language');
      return (v === 'tr' || v === 'en') ? v : 'ru';
    } catch (e) { return 'ru'; }
  }

  function ignored(node) {
    var p = node && (node.parentElement || node);
    return !!(p && p.closest && p.closest('script,style,code,pre,textarea'));
  }

  function preserveWhitespace(value, replacement) {
    var lead = (value.match(/^\s*/) || [''])[0];
    var trail = (value.match(/\s*$/) || [''])[0];
    return lead + replacement + trail;
  }

  function translateValue(value) {
    var lang = currentLanguage();
    if (!value || typeof value !== 'string' || lang === 'ru') return value;
    var core = value.trim();
    if (!core) return value;
    var map = MAPS[lang] || {};
    if (Object.prototype.hasOwnProperty.call(map, core)) {
      return preserveWhitespace(value, map[core]);
    }

    var m;
    if ((m = core.match(/^(\d+(?:[.,]\d+)?) Мбит\/с$/))) {
      return preserveWhitespace(value, lang === 'tr' ? (m[1] + ' Mbit/sn') : (m[1] + ' Mbit/s'));
    }
    if ((m = core.match(/^Entware Manager v([0-9.]+) — интерфейс на базе CGI и вкладок\. Разработчик: Di1r1$/))) {
      return preserveWhitespace(value, lang === 'tr'
        ? ('Entware Manager v' + m[1] + ' — CGI ve sekme tabanlı arayüz. Geliştirici: Di1r1')
        : ('Entware Manager v' + m[1] + ' — CGI and tab-based interface. Developer: Di1r1'));
    }
    return value;
  }

  function translateText(node) {
    if (!node || node.nodeType !== 3 || ignored(node)) return;
    var next = translateValue(node.nodeValue);
    if (next !== node.nodeValue) node.nodeValue = next;
  }

  function translateAttrs(el) {
    if (!el || el.nodeType !== 1 || ignored(el)) return;
    ['title','placeholder','aria-label'].forEach(function (name) {
      if (!el.hasAttribute(name)) return;
      var cur = el.getAttribute(name);
      var next = translateValue(cur);
      if (next !== cur) el.setAttribute(name, next);
    });
  }

  function translateTree(root) {
    if (!root || currentLanguage() === 'ru') return;
    if (root.nodeType === 3) { translateText(root); return; }
    if (root.nodeType !== 1 && root.nodeType !== 9) return;
    if (root.nodeType === 1) translateAttrs(root);
    var walker = document.createTreeWalker(root, NodeFilter.SHOW_ELEMENT | NodeFilter.SHOW_TEXT);
    var n;
    while ((n = walker.nextNode())) {
      if (n.nodeType === 3) translateText(n); else translateAttrs(n);
    }
  }

  function run() {
    translateTree(document.body);
    var obs = new MutationObserver(function (ms) {
      ms.forEach(function (m) {
        if (m.type === 'characterData') translateText(m.target);
        if (m.type === 'attributes') translateAttrs(m.target);
        if (m.addedNodes) Array.prototype.forEach.call(m.addedNodes, translateTree);
      });
    });
    obs.observe(document.body, {
      subtree:true,
      childList:true,
      characterData:true,
      attributes:true,
      attributeFilter:['title','placeholder','aria-label']
    });

    window.EwmLocalI18nV3 = {
      rerun:function(){ translateTree(document.body); },
      remaining:function(){
        var found=[];
        var walker=document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
        var n;
        while ((n=walker.nextNode())) {
          if (!ignored(n) && /[А-Яа-яЁё]/.test(n.nodeValue || '')) {
            var s=(n.nodeValue || '').trim();
            if (s && found.indexOf(s)===-1) found.push(s);
          }
        }
        return found;
      }
    };
  }

  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', run, {once:true});
  else run();
})();
</script>
<!-- EWM_LOCAL_I18N_V3_END -->
EOB

awk '
FNR==NR { block = block $0 ORS; next }
/<\/body>/ && !done { printf "%s", block; done=1 }
{ print }
END { if (!done) exit 42 }
' "$TMP_BLOCK" "$TMP_CLEAN" > "$TMP_INDEX" || {
    echo "ERROR: index.html icinde </body> bulunamadi; dosya degistirilmedi."
    exit 1
}

mv "$TMP_INDEX" "$INDEX"
chmod 0644 "$INDEX"

echo "V3 patch uygulandi."
echo "V2'nin kelime bozan alt-dize cevirisi kaldirildi."
echo "Ceviri artik yalnizca tam metin eslesmesi ve guvenli kaliplarla yapiliyor."
echo "Tarayicida Ctrl+F5 yap."
