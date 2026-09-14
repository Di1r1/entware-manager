#!/bin/sh
set -eu

ROOT="/opt/web_entware"
INDEX="$ROOT/index.html"
BACKUP="$ROOT/index.html.before-local-i18n-v5"
TMP_CLEAN="/tmp/ewm-i18n-v5-clean.$$.html"
TMP_BLOCK="/tmp/ewm-i18n-v5-block.$$.html"
TMP_INDEX="/tmp/ewm-i18n-v5-index.$$.html"

cleanup() {
    rm -f "$TMP_CLEAN" "$TMP_BLOCK" "$TMP_INDEX"
}
trap cleanup EXIT INT TERM

[ -f "$INDEX" ] || { echo "ERROR: $INDEX bulunamadi"; exit 1; }

if [ ! -f "$BACKUP" ]; then
    cp -p "$INDEX" "$BACKUP"
    echo "Backup: $BACKUP"
fi

# V5 authoritative runtime: remove every earlier local i18n block to avoid
# multiple MutationObservers and partial/unsafe replacements.
sed \
  -e '/<!-- EWM_LOCAL_I18N_BEGIN -->/,/<!-- EWM_LOCAL_I18N_END -->/d' \
  -e '/<!-- EWM_LOCAL_I18N_V2_BEGIN -->/,/<!-- EWM_LOCAL_I18N_V2_END -->/d' \
  -e '/<!-- EWM_LOCAL_I18N_V3_BEGIN -->/,/<!-- EWM_LOCAL_I18N_V3_END -->/d' \
  -e '/<!-- EWM_LOCAL_I18N_V4_BEGIN -->/,/<!-- EWM_LOCAL_I18N_V4_END -->/d' \
  -e '/<!-- EWM_LOCAL_I18N_V5_BEGIN -->/,/<!-- EWM_LOCAL_I18N_V5_END -->/d' \
  "$INDEX" > "$TMP_CLEAN"

cat > "$TMP_BLOCK" <<'EOB'
<!-- EWM_LOCAL_I18N_V5_BEGIN -->
<style id="ewm-local-i18n-v5-style">
.ewm-language-wrap {
  display:flex;
  justify-content:center;
  width:100%;
  margin:2px 0 5px;
}
.ewm-language-select {
  width:100%;
  max-width:150px;
  padding:4px 7px;
  border-radius:6px;
  border:1px solid var(--border-color,#cbd5e1);
  background:var(--input-bg,#fff);
  color:var(--text-primary,#111827);
  font-size:11px;
  color-scheme:light;
}
html.night .ewm-language-select,
body.night .ewm-language-select,
[data-theme="dark"] .ewm-language-select {
  background:var(--input-bg,#1f2937);
  color:var(--text-primary,#f8fafc);
  border-color:var(--input-border,#475569);
  color-scheme:dark;
}
.ewm-language-select option {
  background:inherit;
  color:inherit;
}
</style>
<script id="ewm-local-i18n-v5">
(function () {
  'use strict';

  var STORAGE_KEY = 'entware_language';
  var SUPPORTED = {ru:'Русский',en:'English',tr:'Türkçe'};
  var MAPS = {
    tr: {"Управление пакетами и системой":"Paket ve sistem yönetimi","Проверка системы":"Sistem kontrolü","Выйти":"Çıkış yap","Статистика":"İstatistikler","Пакеты":"Paketler","Процессы":"İşlemler","Процессы (htop)":"İşlemler (htop)","Терминал":"Terminal","Службы и Cron":"Servisler ve Cron","Сеть":"Ağ","Модули":"Modüller","Настройки":"Ayarlar","Защита":"Koruma","Справка":"Yardım","Логи":"Günlükler","Обновить":"Yenile","Запустить":"Başlat","Остановить":"Durdur","Перезапустить":"Yeniden başlat","Сохранить":"Kaydet","Отмена":"İptal","Закрыть":"Kapat","Добавить":"Ekle","Изменить":"Düzenle","Удалить":"Kaldır","Установить":"Kur","Проверить":"Kontrol et","Очистить":"Temizle","Скачать":"İndir","Восстановить":"Geri yükle","Создать":"Oluştur","Применить":"Uygula","Сбросить":"Sıfırla","Поиск":"Ara","Включить":"Etkinleştir","Отключить":"Devre dışı bırak","Включено":"Etkin","Выключено":"Devre dışı","Работает":"Çalışıyor","Остановлен":"Durduruldu","Остановлено":"Durduruldu","остановлен":"durduruldu","запущен":"çalışıyor","running":"çalışıyor","stopped":"durduruldu","Да":"Evet","Нет":"Hayır","Имя":"Ad","Статус":"Durum","Действие":"İşlem","Действия":"İşlemler","Описание":"Açıklama","Версия":"Sürüm","Размер":"Boyut","Свободно":"Boş","Использовано":"Kullanılan","Всего":"Toplam","Интерфейс":"Arayüz","Маршруты":"Rotalar","Сетевые интерфейсы":"Ağ arayüzleri","Подключено":"Bağlı","Отключено":"Bağlı değil","Скорость":"Hız","Сигнал":"Sinyal","Канал":"Kanal","Службы":"Servisler","Автозапуск":"Otomatik başlatma","Запущен":"Başlatıldı","Не запущен":"Çalışmıyor","Обновления":"Güncellemeler","Доступные":"Mevcut","Установленные":"Kurulu","Все":"Tümü","Назад":"Geri","Далее":"İleri","Подтвердить":"Onayla","Внимание":"Dikkat","Ошибка":"Hata","Успешно":"Başarılı","Информация":"Bilgi","Загрузка...":"Yükleniyor...","загрузка...":"yükleniyor...","Пароль":"Parola","Войти":"Giriş yap","Язык":"Dil","Язык интерфейса":"Arayüz dili","Статистика системы":"Sistem istatistikleri","Обзор ресурсов, пакетов и дисков роутера":"Yönlendiricinin kaynak, paket ve disk özeti","Система":"Sistem","Модель:":"Model:","Имя хоста:":"Ana bilgisayar adı:","Архитектура:":"Mimari:","Версия ядра:":"Çekirdek sürümü:","Время работы:":"Çalışma süresi:","Память (RAM)":"Bellek (RAM)","Использовано / Всего:":"Kullanılan / Toplam:","Загрузка:":"Kullanım:","Средняя нагрузка:":"Ortalama yük:","Топ по памяти":"Bellek kullanımında ilkler","Процессор (CPU)":"İşlemci (CPU)","Топ по CPU":"CPU kullanımında ilkler","Пакеты Entware":"Entware paketleri","Установлено:":"Kurulu:","Доступно:":"Mevcut:","Диск (/opt)":"Disk (/opt)","Размер:":"Boyut:","Использовано:":"Kullanılan:","Обновить списки пакетов":"Paket listelerini yenile","Обновить все пакеты":"Tüm paketleri güncelle","Поиск по названию...":"Ada göre ara...","ПАКЕТ":"PAKET","ВЕРСИЯ":"SÜRÜM","УСТАНОВЛЕН":"KURULUM","СТАТУС":"DURUM","ДЕЙСТВИЕ":"İŞLEM","установлен":"kurulu","доступен":"mevcut","есть обновление":"güncelleme var","Обновить пакет":"Paketi güncelle","htop не запущен. Откройте Настройки → Терминал, задайте пароль и нажмите Запустить.":"htop çalışmıyor. Ayarlar → Terminal bölümünü açın, parola belirleyin ve Başlat düğmesine basın.","Терминал не запущен. Откройте Настройки → Терминал, задайте пароль и нажмите Запустить.":"Terminal çalışmıyor. Ayarlar → Terminal bölümünü açın, parola belirleyin ve Başlat düğmesine basın.","Системные службы и планировщик":"Sistem servisleri ve zamanlayıcı","Мониторинг:":"İzleme:","Автозапуск при загрузке":"Açılışta otomatik başlat","Кастомный список":"Özel liste","Исключения":"İstisnalar","Автоперезапуск служб при падении":"Servis çökerse otomatik yeniden başlat","Последние события мониторинга":"Son izleme olayları","Нет событий":"Olay yok","Службы (init.d)":"Servisler (init.d)","СЛУЖБА":"SERVİS","СОСТОЯНИЕ":"DURUM","АВТОЗАПУСК":"OTOMATİK BAŞLATMA","ДЕЙСТВИЯ":"İŞLEMLER","Авто":"Oto","Системный crontab (crontab -l)":"Sistem crontab'ı (crontab -l)","Сохранить системный crontab":"Sistem crontab'ını kaydet","Entware crontab (/opt/etc/crontab)":"Entware crontab'ı (/opt/etc/crontab)","Сохранить Entware crontab":"Entware crontab'ını kaydet","Демон:":"Daemon:","Интерфейсы":"Arayüzler","События":"Olaylar","IP адрес":"IP adresi","IP АДРЕС":"IP ADRESİ","ТИП":"TÜR","СКОРОСТЬ":"HIZ","IPIP-туннель":"IPIP tüneli","IPv6-туннель":"IPv6 tüneli","SIT-туннель":"SIT tüneli","Мбит/с":"Mbit/sn","Кбит/с":"Kbit/sn","Гбит/с":"Gbit/sn","Локальные сервисы Entware, обнаруженные на этом роутере. Ползунок включает/выключает модуль в панели, галочка управляет уведомлениями о падении/восстановлении сервиса.":"Bu yönlendiricide bulunan yerel Entware servisleri. Anahtar modülü panelde açıp kapatır; onay kutusu servis durma/geri gelme bildirimlerini yönetir.","не установлен":"kurulu değil","уведомления":"bildirimler","процесс не найден":"işlem bulunamadı","управление":"yönetim","Пересканировать":"Yeniden tara","Манифесты":"Manifestolar","Терминал ttyd":"ttyd terminali","сервис не отвечает":"servis yanıt vermiyor","SMART дисков":"Disk SMART","Поиск по модели/серийному...":"Model/seri numarasına göre ara...","УСТРОЙСТВО":"CİHAZ","МОДЕЛЬ":"MODEL","СЕРИЙНЫЙ №":"SERİ NO","РАЗМЕР":"BOYUT","HEALTH":"SAĞLIK","TEMP":"SICAKLIK","POWER-ON":"ÇALIŞMA SÜRESİ","РАЗДЕЛ":"BÖLÜM","ТОЧКА":"BAĞLAMA NOKTASI","ИСП.":"KULL.","СВОБ.":"BOŞ","ЗАНЯТО":"DOLULUK","SMART информация":"SMART bilgisi","SMART атрибуты":"SMART öznitelikleri","Атрибуты SMART":"SMART öznitelikleri","Кликните на имя атрибута для справки":"Açıklama için öznitelik adına tıklayın","ИМЯ АТРИБУТА":"ÖZNİTELİK ADI","Предупреждение":"Uyarı","Критично":"Kritik","Критичный атрибут":"Kritik öznitelik","Важный атрибут":"Önemli öznitelik","SMART Health":"SMART Sağlığı","Health":"Sağlık","Прокси не запущен":"Proxy çalışmıyor","Порт:":"Port:","Запустите прокси, чтобы подключиться к ПК по RDP.":"RDP ile bilgisayara bağlanmak için proxy'yi başlatın.","Запустить прокси":"Proxy'yi başlat","Остановить прокси":"Proxy'yi durdur","Этот порт уже используется":"Bu port zaten kullanımda","Терминал и ссылки":"Terminal ve bağlantılar","Уведомления":"Bildirimler","Безопасность":"Güvenlik","Обслуживание":"Bakım","Управление ttyd":"ttyd yönetimi","Текущее состояние ttyd":"Mevcut ttyd durumu","htop (8089, доступ /htop/):":"htop (8089, erişim /htop/):","Терминал (9089, доступ /terminal/):":"Terminal (9089, erişim /terminal/):","htop (порт 8089)":"htop (port 8089)","Терминал (порт 9089)":"Terminal (port 9089)","Консоль Entware":"Entware konsolu","Управление веб-терминалами ttyd. Пароль обязателен для обоих сервисов — терминал доступен извне через панель, без пароля запуск запрещён.":"ttyd web terminallerini yönetir. Her iki servis için parola zorunludur; terminal panele üzerinden erişilebilir ve parola olmadan başlatılamaz.","Доступ: панель → /terminal/ и /htop/ (тот же origin, порты 9089/8089 слушают только loopback).":"Erişim: panel → /terminal/ ve /htop/ (aynı origin; 9089/8089 portları yalnızca loopback üzerinde dinler).","Управление ссылками на главной (общие для всех устройств)":"Ana sayfa bağlantılarını yönet (tüm cihazlar için ortak)","Здесь можно добавлять, редактировать и удалять ссылки. Изменения сразу видны на всех устройствах.":"Buradan bağlantı ekleyebilir, düzenleyebilir ve silebilirsiniz. Değişiklikler tüm cihazlarda hemen görünür.","Добавить ссылку":"Bağlantı ekle","ИКОНКА":"SİMGE","НАЗВАНИЕ":"AD","Иконка":"Simge","Название":"Ad","Роутер":"Yönlendirici","Пакет":"Paket","Щит":"Kalkan","Загрузка":"İndirme","График":"Grafik","Процесс":"İşlem","Сохранить все на сервер":"Tümünü sunucuya kaydet","Сбросить по умолчанию":"Varsayılana sıfırla","Настройки терминала":"Terminal ayarları","Настройки безопасности":"Güvenlik ayarları","Настройки уведомлений":"Bildirim ayarları","Системное обслуживание":"Sistem bakımı","Пароль панели":"Panel parolası","Защита панели":"Panel koruması","Включить защиту":"Korumayı etkinleştir","Текущий пароль":"Mevcut parola","Новый пароль":"Yeni parola","Повторите пароль":"Parolayı tekrar girin","Резервная копия":"Yedek","Создать резервную копию":"Yedek oluştur","Восстановление":"Geri yükleme","Выберите файл":"Dosya seçin","Обновление":"Güncelleme","Проверить обновления":"Güncellemeleri kontrol et","Последняя версия":"Son sürüm","Текущая версия":"Mevcut sürüm","Доступно обновление":"Güncelleme mevcut","Обновлений нет":"Güncelleme yok","Защита от зависших процессов":"Takılı kalan işlemlere karşı koruma","Статус демона:":"Daemon durumu:","Топ процессов по нагрузке CPU":"CPU yüküne göre en yoğun işlemler","CPU время":"CPU süresi","Команда":"Komut","Убить":"Sonlandır","Настройки защиты":"Koruma ayarları","Включить защиту (глобально)":"Korumayı etkinleştir (genel)","Интервал сканирования (сек):":"Tarama aralığı (sn):","Индивидуальный режим":"Bireysel mod","Порог CPU (%):":"CPU eşiği (%):","Время непрерывной нагрузки (мин):":"Kesintisiz yüksek yük süresi (dk):","Игнорируемые процессы (имена, через запятую):":"Yok sayılacak işlemler (adlar, virgülle):","Исключать ps из мониторинга (убирает ложные предупреждения)":"ps işlemini izlemeden çıkar (yanlış uyarıları azaltır)","Дополнительные настройки":"Ek ayarlar","Максимум процессов для сканирования:":"Taranacak azami işlem sayısı:","Сохранить настройки":"Ayarları kaydet","Лог событий":"Olay günlüğü","Лог-файл не найден":"Günlük dosyası bulunamadı","Очистить лог":"Günlüğü temizle","Действия менеджера":"Yönetici işlemleri","Системные логи":"Sistem günlükleri","Очистить логи старше 30 дней":"30 günden eski günlükleri temizle","Ротация сейчас":"Şimdi döndür","Настройки логирования":"Günlükleme ayarları","Системные события":"Sistem olayları","Показать заголовок":"Başlığı göster","Показать футер":"Alt bilgiyi göster","Пароль панели:":"Panel parolası:","Неверный пароль":"Hatalı parola","Загрузка графиков...":"Grafikler yükleniyor...","Мин":"Min","Средняя":"Ortalama","Макс":"Maks","Сейчас":"Şimdi","Нет данных. Графики появятся через несколько минут.":"Henüz veri yok. Grafikler birkaç dakika içinde görünecek.","Температура":"Sıcaklık","Ошибка загрузки данных":"Veri yükleme hatası","Справка по Entware Manager":"Entware Manager Yardımı","Ниже описаны все доступные вкладки, примеры команд и инструкции по настройке.":"Aşağıda kullanılabilir bölümler, komut örnekleri ve yapılandırma bilgileri yer alır.","Поиск по справке...":"Yardımda ara...","Версия интерфейса":"Arayüz sürümü","дата":"tarih","Разработчик":"Geliştirici","интерфейс на базе CGI и вкладок":"CGI ve sekme tabanlı arayüz","Мониторинг":"İzleme","Состояние":"Durum","Доступ":"Erişim","Запущено":"Çalışıyor","Сохранено":"Kaydedildi","Настройки сохранены":"Ayarlar kaydedildi","Токен":"Token","Чат ID":"Sohbet ID","Отправить тест":"Test gönder","Ротация":"Döndürme","Управление":"Yönetim","Системный":"Sistem","АВТОМАТИЧЕСКИЙ ЗАПУСК":"OTOMATİK BAŞLATMA"},
    en: {"Управление пакетами и системой":"Package and system management","Проверка системы":"System check","Выйти":"Log out","Статистика":"Statistics","Пакеты":"Packages","Процессы":"Processes","Процессы (htop)":"Processes (htop)","Терминал":"Terminal","Службы и Cron":"Services & Cron","Сеть":"Network","Модули":"Modules","Настройки":"Settings","Защита":"Protection","Справка":"Help","Логи":"Logs","Обновить":"Refresh","Запустить":"Start","Остановить":"Stop","Перезапустить":"Restart","Сохранить":"Save","Отмена":"Cancel","Закрыть":"Close","Добавить":"Add","Изменить":"Edit","Удалить":"Remove","Установить":"Install","Проверить":"Check","Очистить":"Clear","Скачать":"Download","Восстановить":"Restore","Создать":"Create","Применить":"Apply","Сбросить":"Reset","Поиск":"Search","Включить":"Enable","Отключить":"Disable","Включено":"Enabled","Выключено":"Disabled","Работает":"Running","Остановлен":"Stopped","Остановлено":"Stopped","остановлен":"stopped","запущен":"running","running":"running","stopped":"stopped","Да":"Yes","Нет":"No","Имя":"Name","Статус":"Status","Действие":"Action","Действия":"Actions","Описание":"Description","Версия":"Version","Размер":"Size","Свободно":"Free","Использовано":"Used","Всего":"Total","Интерфейс":"Interface","Маршруты":"Routes","Сетевые интерфейсы":"Network interfaces","Подключено":"Connected","Отключено":"Disconnected","Скорость":"Speed","Сигнал":"Signal","Канал":"Channel","Службы":"Services","Автозапуск":"Autostart","Запущен":"Started","Не запущен":"Not running","Обновления":"Updates","Доступные":"Available","Установленные":"Installed","Все":"All","Назад":"Back","Далее":"Next","Подтвердить":"Confirm","Внимание":"Warning","Ошибка":"Error","Успешно":"Success","Информация":"Information","Загрузка...":"Loading...","загрузка...":"loading...","Пароль":"Password","Войти":"Sign in","Язык":"Language","Язык интерфейса":"Interface language","Статистика системы":"System statistics","Обзор ресурсов, пакетов и дисков роутера":"Router resources, packages and disks overview","Система":"System","Модель:":"Model:","Имя хоста:":"Host name:","Архитектура:":"Architecture:","Версия ядра:":"Kernel version:","Время работы:":"Uptime:","Память (RAM)":"Memory (RAM)","Использовано / Всего:":"Used / Total:","Загрузка:":"Usage:","Средняя нагрузка:":"Average load:","Топ по памяти":"Top by memory","Процессор (CPU)":"Processor (CPU)","Топ по CPU":"Top by CPU","Пакеты Entware":"Entware packages","Установлено:":"Installed:","Доступно:":"Available:","Диск (/opt)":"Disk (/opt)","Размер:":"Size:","Использовано:":"Used:","Обновить списки пакетов":"Refresh package lists","Обновить все пакеты":"Update all packages","Поиск по названию...":"Search by name...","ПАКЕТ":"PACKAGE","ВЕРСИЯ":"VERSION","УСТАНОВЛЕН":"INSTALLED","СТАТУС":"STATUS","ДЕЙСТВИЕ":"ACTION","установлен":"installed","доступен":"available","есть обновление":"update available","Обновить пакет":"Update package","htop не запущен. Откройте Настройки → Терминал, задайте пароль и нажмите Запустить.":"htop is not running. Open Settings → Terminal, set a password and press Start.","Терминал не запущен. Откройте Настройки → Терминал, задайте пароль и нажмите Запустить.":"Terminal is not running. Open Settings → Terminal, set a password and press Start.","Системные службы и планировщик":"System services and scheduler","Мониторинг:":"Monitoring:","Автозапуск при загрузке":"Start automatically at boot","Кастомный список":"Custom list","Исключения":"Exclusions","Автоперезапуск служб при падении":"Automatically restart failed services","Последние события мониторинга":"Latest monitoring events","Нет событий":"No events","Службы (init.d)":"Services (init.d)","СЛУЖБА":"SERVICE","СОСТОЯНИЕ":"STATE","АВТОЗАПУСК":"AUTOSTART","ДЕЙСТВИЯ":"ACTIONS","Авто":"Auto","Системный crontab (crontab -l)":"System crontab (crontab -l)","Сохранить системный crontab":"Save system crontab","Entware crontab (/opt/etc/crontab)":"Entware crontab (/opt/etc/crontab)","Сохранить Entware crontab":"Save Entware crontab","Демон:":"Daemon:","Интерфейсы":"Interfaces","События":"Events","IP адрес":"IP address","IP АДРЕС":"IP ADDRESS","ТИП":"TYPE","СКОРОСТЬ":"SPEED","IPIP-туннель":"IPIP tunnel","IPv6-туннель":"IPv6 tunnel","SIT-туннель":"SIT tunnel","Мбит/с":"Mbit/s","Кбит/с":"Kbit/s","Гбит/с":"Gbit/s","Локальные сервисы Entware, обнаруженные на этом роутере. Ползунок включает/выключает модуль в панели, галочка управляет уведомлениями о падении/восстановлении сервиса.":"Local Entware services detected on this router. The switch enables/disables the module in the panel; the checkbox controls service down/recovery notifications.","не установлен":"not installed","уведомления":"notifications","процесс не найден":"process not found","управление":"management","Пересканировать":"Rescan","Манифесты":"Manifests","Терминал ttyd":"ttyd terminal","сервис не отвечает":"service is not responding","SMART дисков":"Disk SMART","Поиск по модели/серийному...":"Search by model/serial...","УСТРОЙСТВО":"DEVICE","МОДЕЛЬ":"MODEL","СЕРИЙНЫЙ №":"SERIAL NO.","РАЗМЕР":"SIZE","HEALTH":"HEALTH","TEMP":"TEMP","POWER-ON":"POWER-ON","РАЗДЕЛ":"PARTITION","ТОЧКА":"MOUNT POINT","ИСП.":"USED","СВОБ.":"FREE","ЗАНЯТО":"USAGE","SMART информация":"SMART information","SMART атрибуты":"SMART attributes","Атрибуты SMART":"SMART attributes","Кликните на имя атрибута для справки":"Click an attribute name for help","ИМЯ АТРИБУТА":"ATTRIBUTE NAME","Предупреждение":"Warning","Критично":"Critical","Критичный атрибут":"Critical attribute","Важный атрибут":"Important attribute","SMART Health":"SMART Health","Health":"Health","Прокси не запущен":"Proxy is not running","Порт:":"Port:","Запустите прокси, чтобы подключиться к ПК по RDP.":"Start the proxy to connect to a PC via RDP.","Запустить прокси":"Start proxy","Остановить прокси":"Stop proxy","Этот порт уже используется":"This port is already in use","Терминал и ссылки":"Terminal & links","Уведомления":"Notifications","Безопасность":"Security","Обслуживание":"Maintenance","Управление ttyd":"ttyd management","Текущее состояние ttyd":"Current ttyd status","htop (8089, доступ /htop/):":"htop (8089, access /htop/):","Терминал (9089, доступ /terminal/):":"Terminal (9089, access /terminal/):","htop (порт 8089)":"htop (port 8089)","Терминал (порт 9089)":"Terminal (port 9089)","Консоль Entware":"Entware console","Управление веб-терминалами ttyd. Пароль обязателен для обоих сервисов — терминал доступен извне через панель, без пароля запуск запрещён.":"Manage ttyd web terminals. A password is required for both services; the terminal is accessible through the panel and cannot be started without a password.","Доступ: панель → /terminal/ и /htop/ (тот же origin, порты 9089/8089 слушают только loopback).":"Access: panel → /terminal/ and /htop/ (same origin; ports 9089/8089 listen on loopback only).","Управление ссылками на главной (общие для всех устройств)":"Manage home-page links (shared by all devices)","Здесь можно добавлять, редактировать и удалять ссылки. Изменения сразу видны на всех устройствах.":"Add, edit and remove links here. Changes are immediately visible on all devices.","Добавить ссылку":"Add link","ИКОНКА":"ICON","НАЗВАНИЕ":"NAME","Иконка":"Icon","Название":"Name","Роутер":"Router","Пакет":"Package","Щит":"Shield","Загрузка":"Download","График":"Graph","Процесс":"Process","Сохранить все на сервер":"Save all to server","Сбросить по умолчанию":"Reset to defaults","Настройки терминала":"Terminal settings","Настройки безопасности":"Security settings","Настройки уведомлений":"Notification settings","Системное обслуживание":"System maintenance","Пароль панели":"Panel password","Защита панели":"Panel protection","Включить защиту":"Enable protection","Текущий пароль":"Current password","Новый пароль":"New password","Повторите пароль":"Repeat password","Резервная копия":"Backup","Создать резервную копию":"Create backup","Восстановление":"Restore","Выберите файл":"Choose file","Обновление":"Update","Проверить обновления":"Check for updates","Последняя версия":"Latest version","Текущая версия":"Current version","Доступно обновление":"Update available","Обновлений нет":"No updates","Защита от зависших процессов":"Protection against stuck processes","Статус демона:":"Daemon status:","Топ процессов по нагрузке CPU":"Top processes by CPU load","CPU время":"CPU time","Команда":"Command","Убить":"Kill","Настройки защиты":"Protection settings","Включить защиту (глобально)":"Enable protection (global)","Интервал сканирования (сек):":"Scan interval (sec):","Индивидуальный режим":"Per-process mode","Порог CPU (%):":"CPU threshold (%):","Время непрерывной нагрузки (мин):":"Continuous high-load time (min):","Игнорируемые процессы (имена, через запятую):":"Ignored processes (names, comma-separated):","Исключать ps из мониторинга (убирает ложные предупреждения)":"Exclude ps from monitoring (reduces false warnings)","Дополнительные настройки":"Additional settings","Максимум процессов для сканирования:":"Maximum processes to scan:","Сохранить настройки":"Save settings","Лог событий":"Event log","Лог-файл не найден":"Log file not found","Очистить лог":"Clear log","Действия менеджера":"Manager actions","Системные логи":"System logs","Очистить логи старше 30 дней":"Delete logs older than 30 days","Ротация сейчас":"Rotate now","Настройки логирования":"Logging settings","Системные события":"System events","Показать заголовок":"Show header","Показать футер":"Show footer","Пароль панели:":"Panel password:","Неверный пароль":"Incorrect password","Загрузка графиков...":"Loading charts...","Мин":"Min","Средняя":"Average","Макс":"Max","Сейчас":"Now","Нет данных. Графики появятся через несколько минут.":"No data yet. Charts will appear in a few minutes.","Температура":"Temperature","Ошибка загрузки данных":"Failed to load data","Справка по Entware Manager":"Entware Manager Help","Ниже описаны все доступные вкладки, примеры команд и инструкции по настройке.":"Available sections, command examples, and configuration guidance are described below.","Поиск по справке...":"Search help...","Версия интерфейса":"Interface version","дата":"date","Разработчик":"Developer","интерфейс на базе CGI и вкладок":"CGI and tab-based interface","Мониторинг":"Monitoring","Состояние":"State","Доступ":"Access","Запущено":"Running","Сохранено":"Saved","Настройки сохранены":"Settings saved","Токен":"Token","Чат ID":"Chat ID","Отправить тест":"Send test","Ротация":"Rotation","Управление":"Management","Системный":"System","АВТОМАТИЧЕСКИЙ ЗАПУСК":"AUTOSTART"}
  };

  var HELP = {
    tr: "<h3>İstatistikler</h3>\n<p>Yönlendiricinin modelini, ana bilgisayar adını, mimarisini, çekirdek sürümünü, çalışma süresini, RAM/CPU kullanımını, Entware paketlerini, bağlı diskleri ve <code>/opt</code> kullanımını gösterir. Tablo başlıkları sıralanabilir ve Yenile düğmesi verileri tekrar alır.</p>\n\n<h3>Paketler</h3>\n<p>Kurulu ve mevcut Entware paketlerini tek tabloda gösterir. Paket listelerini yenileyebilir, güncellemeleri görebilir, ada göre arayabilir, paket kurabilir/kaldırabilir ve tüm paketleri güncelleyebilirsiniz.</p>\n<div class=\"command-block\">opkg list-installed<br>opkg list<br>opkg update<br>opkg list-upgradable<br>opkg install &lt;paket&gt;<br>opkg remove &lt;paket&gt;<br>opkg upgrade &lt;paket&gt;</div>\n\n<h3>İşlemler (htop)</h3>\n<p><strong>htop</strong>, ttyd üzerinden çalışan etkileşimli işlem görüntüleyicisidir. Çalışmıyorsa Ayarlar → Terminal bölümünden parola belirleyip htop hizmetini başlatın. F3 arama, F5 ağaç görünümü, F6 sıralama, F9 sinyal gönderme, q çıkış.</p>\n\n<h3>Terminal</h3>\n<p>Web terminali <code>/terminal/</code>, htop ise <code>/htop/</code> yolundan açılır. İki servis de yalnızca loopback üzerinde dinler ve başlatılırken parola gerektirir. Terminal modu Entware konsolu veya yönlendirici telnet konsolu olabilir.</p>\n\n<h3>Syncthing</h3>\n<p>Telefon, bilgisayar ve yönlendirici arasında doğrudan, bulutsuz dosya eşitlemesi sağlar. Kurulumdan sonra servisi bir kez başlatın ve Syncthing web arayüzünde kullanıcı/parola belirleyin.</p>\n<div class=\"command-block\">opkg update<br>opkg install syncthing<br>/opt/etc/init.d/S92syncthing start</div>\n\n<h3>Modüller</h3>\n<p>Yerel servisler <code>/opt/web_entware/bridge/</code> altındaki manifestolarla panele eklenir. Bir modül HTTP probe ile veya çalışan işlem adına göre izlenebilir. Yönetim düğmeleri için uygun <code>init.d</code> betiği ve yönetim izni gerekir.</p>\n<div class=\"command-block\">{ \"id\": \"mydaemon\", \"name\": \"My Daemon\", \"process\": [\"mydaemon\"], \"init\": \"mydaemon\" }</div>\n\n<h3>Ağ</h3>\n<p>Arayüzleri, rotaları, ARP tablosunu, Wi‑Fi durumunu ve ağ izleme olaylarını gösterir. Ağ izleme daemon'u arayüz durum değişikliklerini ve internet erişimini takip eder.</p>\n<div class=\"command-block\">ip addr<br>ip route<br>ip neigh</div>\n\n<h3>Disk SMART</h3>\n<p>HDD, SSD, NVMe ve USB disklerini S.M.A.R.T. üzerinden izler. Model, seri numarası, boyut, sıcaklık, sağlık ve çalışma süresi gösterilir. Disk alanlarına tıklayarak bilgi, bölüm kullanımı, SMART öznitelikleri ve öz test ayrıntıları açılabilir.</p>\n<div class=\"command-block\">smartctl -a /dev/sda | head -5</div>\n\n<h3>RDP</h3>\n<p>Yerel ağdaki bilgisayarlara tarayıcıdan RDP bağlantısı için <code>grdp-proxy</code> kullanılır. Proxy yalnızca <code>127.0.0.1</code> üzerinde dinler ve panel üzerinden erişilir. Windows kullanıcı adı ve parolası yönlendiricide saklanmaz.</p>\n\n<h3>Servisler ve Cron</h3>\n<p><code>/opt/etc/init.d/</code> altındaki Entware servisleri başlatılabilir, durdurulabilir ve yeniden başlatılabilir. Otomatik başlatma S/K önekleriyle yönetilir. Service watchdog servislerin düşmesini izleyebilir ve istenirse otomatik yeniden başlatabilir.</p>\n<p>Sistem crontab'ı ile Entware crontab'ı ayrı düzenleyicilerden yönetilir.</p>\n<div class=\"command-block\">crontab -l<br>crontab -e<br>cat /opt/etc/crontab</div>\n\n<h3>Ayarlar</h3>\n<p>ttyd/htop yönetimi, ana sayfa bağlantıları, bildirimler, panel güvenliği, bakım, yedekleme ve güncelleme seçenekleri burada bulunur. Panel parolası etkinleştirildiğinde oturum çereziyle korunan giriş ekranı kullanılır.</p>\n\n<h3>Koruma</h3>\n<p>Uzun süre yüksek CPU kullanan işlemleri izleyen koruma daemon'unu yönetir. Tarama aralığı, CPU eşiği, yüksek yük süresi, yok sayılacak işlemler ve taranacak azami işlem sayısı yapılandırılabilir.</p>\n\n<h3>Günlükler</h3>\n<p>Yönetici işlemleri ve sistem günlükleri görüntülenebilir. Eski günlükler temizlenebilir, günlük döndürme elle çalıştırılabilir ve günlükleme açılıp kapatılabilir.</p>\n\n<h3>Telegram bildirimleri ve bot</h3>\n<p>Bot token'ı ve <code>chat_id</code> ile sistem, ağ, servis, paket, sıcaklık ve kaynak uyarıları gönderilebilir. Bot yalnızca yapılandırılmış kullanıcıya yanıt verir. Temel komutlar: <code>/help</code>, <code>/status</code>, <code>/temp</code>, <code>/ip</code>, <code>/services</code>, <code>/smart</code>, <code>/log</code>, <code>/top</code>, <code>/ports</code>, <code>/devices</code>, <code>/wifi</code>, <code>/updates</code>, <code>/cron</code>.</p>\n\n<h3>Sistem kontrolü</h3>\n<p><code>opkg</code>, lighttpd, BusyBox araçları, cron, jq, ip, curl, bash ve bridge-utils gibi bağımlılıkları; ayrıca CGI/shell betiklerinin sözdizimini kontrol eder ve eksikler için kurulum önerileri gösterir.</p>\n\n<h3>Çevrimdışı kurulum</h3>\n<p>İnternet erişimi olmayan yönlendiriciler için çevrimdışı paket hazırlanabilir. Hedef cihazda arşivi açıp <code>install-offline.sh</code> çalıştırılır.</p>\n\n<h3>Yedekleme</h3>\n<p>Yedekleme; panel dosyalarını, bağlantıları, yapılandırmaları ve ilgili günlükleri saklar. Geri yükleme işleminden önce mevcut yapılandırmanın ayrıca yedeğini tutmanız önerilir.</p>\n\n<h3>Temel komutlar</h3>\n<div class=\"command-block\">\nopkg update<br>\nopkg list-installed<br>\nopkg list-upgradable<br>\n/opt/etc/init.d/S&lt;servis&gt; start<br>\n/opt/etc/init.d/S&lt;servis&gt; stop<br>\n/opt/etc/init.d/S&lt;servis&gt; restart<br>\ncrontab -l<br>\nps<br>\nfree -h<br>\ndf -h<br>\nip addr<br>\nip route<br>\nls -la /opt\n</div>\n<p><strong>Dikkat:</strong> root yetkisiyle çalışan komutlarda özellikle silme ve paket güncelleme işlemlerinde dikkatli olun. Web terminali için parola kullanın.</p>",
    en: "<h3>Statistics</h3>\n<p>Shows the router model, host name, architecture, kernel version, uptime, RAM/CPU usage, Entware packages, attached disks, and <code>/opt</code> usage. Table headers can be sorted and Refresh reloads the data.</p>\n\n<h3>Packages</h3>\n<p>Shows installed and available Entware packages in one table. You can refresh package lists, inspect updates, search by name, install/remove packages, and update all packages.</p>\n<div class=\"command-block\">opkg list-installed<br>opkg list<br>opkg update<br>opkg list-upgradable<br>opkg install &lt;package&gt;<br>opkg remove &lt;package&gt;<br>opkg upgrade &lt;package&gt;</div>\n\n<h3>Processes (htop)</h3>\n<p><strong>htop</strong> is the interactive process viewer served through ttyd. If it is not running, open Settings → Terminal, set a password, and start the htop service. F3 searches, F5 toggles tree view, F6 sorts, F9 sends a signal, and q exits.</p>\n\n<h3>Terminal</h3>\n<p>The web terminal is available at <code>/terminal/</code> and htop at <code>/htop/</code>. Both services listen on loopback only and require a password when started. The terminal can use the Entware console or the router telnet console.</p>\n\n<h3>Syncthing</h3>\n<p>Provides direct cloud-free file synchronization between phones, computers, and the router. After installation, start the service once and configure a user/password in the Syncthing web interface.</p>\n<div class=\"command-block\">opkg update<br>opkg install syncthing<br>/opt/etc/init.d/S92syncthing start</div>\n\n<h3>Modules</h3>\n<p>Local services are added with manifests under <code>/opt/web_entware/bridge/</code>. A module can be monitored through an HTTP probe or by process name. Management buttons require a suitable <code>init.d</code> script and management permission.</p>\n<div class=\"command-block\">{ \"id\": \"mydaemon\", \"name\": \"My Daemon\", \"process\": [\"mydaemon\"], \"init\": \"mydaemon\" }</div>\n\n<h3>Network</h3>\n<p>Shows interfaces, routes, the ARP table, Wi‑Fi state, and network monitoring events. The network watchdog tracks interface changes and internet reachability.</p>\n<div class=\"command-block\">ip addr<br>ip route<br>ip neigh</div>\n\n<h3>Disk SMART</h3>\n<p>Monitors HDD, SSD, NVMe, and USB devices through S.M.A.R.T. It shows model, serial number, size, temperature, health, and power-on time. Click disk fields for information, partition usage, SMART attributes, and self-test details.</p>\n<div class=\"command-block\">smartctl -a /dev/sda | head -5</div>\n\n<h3>RDP</h3>\n<p><code>grdp-proxy</code> provides browser-based RDP access to computers on the local network. The proxy listens only on <code>127.0.0.1</code> and is reached through the panel. Windows credentials are not stored on the router.</p>\n\n<h3>Services & Cron</h3>\n<p>Entware services under <code>/opt/etc/init.d/</code> can be started, stopped, and restarted. Autostart is controlled by S/K prefixes. The service watchdog can detect failed services and optionally restart them.</p>\n<p>The system crontab and Entware crontab are managed by separate editors.</p>\n<div class=\"command-block\">crontab -l<br>crontab -e<br>cat /opt/etc/crontab</div>\n\n<h3>Settings</h3>\n<p>Contains ttyd/htop management, home-page links, notifications, panel security, maintenance, backup, and update options. When a panel password is enabled, access is protected by a session-cookie login screen.</p>\n\n<h3>Protection</h3>\n<p>Manages the protection daemon that watches processes with sustained high CPU usage. Configure scan interval, CPU threshold, high-load duration, ignored processes, and the maximum number of processes to scan.</p>\n\n<h3>Logs</h3>\n<p>Displays manager actions and system logs. Old logs can be deleted, log rotation can be run manually, and logging can be enabled or disabled.</p>\n\n<h3>Telegram notifications and bot</h3>\n<p>With a bot token and <code>chat_id</code>, the manager can send system, network, service, package, temperature, and resource alerts. The bot responds only to the configured owner. Common commands include <code>/help</code>, <code>/status</code>, <code>/temp</code>, <code>/ip</code>, <code>/services</code>, <code>/smart</code>, <code>/log</code>, <code>/top</code>, <code>/ports</code>, <code>/devices</code>, <code>/wifi</code>, <code>/updates</code>, and <code>/cron</code>.</p>\n\n<h3>System check</h3>\n<p>Checks dependencies such as <code>opkg</code>, lighttpd, BusyBox tools, cron, jq, ip, curl, bash, and bridge-utils, plus CGI/shell syntax, and suggests installation commands for missing components.</p>\n\n<h3>Offline installation</h3>\n<p>An offline package can be prepared for routers without internet access. Extract it on the target router and run <code>install-offline.sh</code>.</p>\n\n<h3>Backup</h3>\n<p>Backup stores panel files, links, configuration, and related logs. Keep an additional copy of the current configuration before restoring a backup.</p>\n\n<h3>Common commands</h3>\n<div class=\"command-block\">\nopkg update<br>\nopkg list-installed<br>\nopkg list-upgradable<br>\n/opt/etc/init.d/S&lt;service&gt; start<br>\n/opt/etc/init.d/S&lt;service&gt; stop<br>\n/opt/etc/init.d/S&lt;service&gt; restart<br>\ncrontab -l<br>\nps<br>\nfree -h<br>\ndf -h<br>\nip addr<br>\nip route<br>\nls -la /opt\n</div>\n<p><strong>Warning:</strong> be careful with delete operations and package updates when commands run as root. Use a password for the web terminal.</p>"
  };

  var SAFE_FRAGMENTS = {
    tr: {
      'Версия интерфейса:':'Arayüz sürümü:',
      'версия загружается из':'sürüm şu dosyadan yüklenir:',
      'интерфейс на базе CGI и вкладок':'CGI ve sekme tabanlı arayüz',
      'Разработчик:':'Geliştirici:',
      'Прокси не запущен':'Proxy çalışmıyor',
      'Запустите прокси, чтобы подключиться к ПК по RDP.':"RDP ile bilgisayara bağlanmak için proxy'yi başlatın.",
      'Автозапуск при загрузке':'Açılışta otomatik başlat',
      'не установлен':'kurulu değil',
      'процесс не найден':'işlem bulunamadı',
      'сервис не отвечает':'servis yanıt vermiyor',
      'не запущен':'çalışmıyor',
      'не запущена':'çalışmıyor',
      'Мбит/с':'Mbit/sn',
      'Кбит/с':'Kbit/sn',
      'Гбит/с':'Gbit/sn',
      'Управление веб-терминалами ttyd.':'ttyd web terminallerini yönetir.',
      'Пароль обязателен':'Parola zorunludur',
      'для обоих сервисов — терминал доступен извне через панель, без пароля запуск запрещён.':'Her iki servis için de terminale panel üzerinden erişilir; parola olmadan başlatma engellenir.',
      'Доступ: панель →':'Erişim: panel →',
      '(тот же origin, порты 9089/8089 слушают только loopback).':'(aynı origin; 9089/8089 portları yalnızca loopback üzerinde dinler).'
    },
    en: {
      'Версия интерфейса:':'Interface version:',
      'версия загружается из':'version is loaded from',
      'интерфейс на базе CGI и вкладок':'CGI and tab-based interface',
      'Разработчик:':'Developer:',
      'Прокси не запущен':'Proxy is not running',
      'Запустите прокси, чтобы подключиться к ПК по RDP.':'Start the proxy to connect to a PC via RDP.',
      'Автозапуск при загрузке':'Start automatically at boot',
      'не установлен':'not installed',
      'процесс не найден':'process not found',
      'сервис не отвечает':'service is not responding',
      'не запущен':'not running',
      'не запущена':'not running',
      'Мбит/с':'Mbit/s',
      'Кбит/с':'Kbit/s',
      'Гбит/с':'Gbit/s',
      'Управление веб-терминалами ttyd.':'Manage ttyd web terminals.',
      'Пароль обязателен':'A password is required',
      'для обоих сервисов — терминал доступен извне через панель, без пароля запуск запрещён.':'for both services; the terminal is accessible through the panel and cannot be started without a password.',
      'Доступ: панель →':'Access: panel →',
      '(тот же origin, порты 9089/8089 слушают только loopback).':'(same origin; ports 9089/8089 listen on loopback only).'
    }
  };

  var fallbackSeen = [];
  var busy = false;
  var observer = null;

  function currentLanguage() {
    try {
      var saved = localStorage.getItem(STORAGE_KEY);
      if (SUPPORTED[saved]) return saved;
    } catch (e) {}
    var browser = String((navigator && navigator.language) || '').toLowerCase().split('-')[0];
    return SUPPORTED[browser] ? browser : 'ru';
  }

  function setLanguage(lang) {
    if (!SUPPORTED[lang]) return;
    try { localStorage.setItem(STORAGE_KEY, lang); } catch (e) {}
    location.reload();
  }

  function hasCyrillic(value) {
    return /[А-Яа-яЁё]/.test(String(value || ''));
  }

  function mapFor(lang) {
    return MAPS[lang] || {};
  }

  function normalizeSpaces(value) {
    return String(value || '').replace(/\u00a0/g, ' ');
  }

  function translatePattern(value, lang) {
    var out = value;

    // Dynamic SMART modal titles.
    out = out.replace(/SMART информация\s*—\s*(\/dev\/\S+)/g,
      lang === 'tr' ? 'SMART bilgisi — $1' : 'SMART information — $1');
    out = out.replace(/Информация:\s*(\/dev\/\S+)/g,
      lang === 'tr' ? 'Bilgi: $1' : 'Information: $1');
    out = out.replace(/SMART атрибуты\s*—\s*(\/dev\/\S+)/g,
      lang === 'tr' ? 'SMART öznitelikleri — $1' : 'SMART attributes — $1');
    out = out.replace(/Атрибуты SMART:\s*(\/dev\/\S+)/g,
      lang === 'tr' ? 'SMART öznitelikleri: $1' : 'SMART attributes: $1');
    out = out.replace(/SMART Health\s*—\s*(\/dev\/\S+)/g,
      lang === 'tr' ? 'SMART Sağlığı — $1' : 'SMART Health — $1');

    // Help header with dynamic version/date.
    out = out.replace(/Версия интерфейса:\s*([^\s]+)\s*\(дата:\s*([^,\)]+),\s*версия загружается из\s*([^\)]+)\)/g,
      lang === 'tr'
        ? 'Arayüz sürümü: $1 (tarih: $2, sürüm $3 dosyasından yüklenir)'
        : 'Interface version: $1 (date: $2, version is loaded from $3)');

    // Footer.
    out = out.replace(/Entware Manager v([^\s]+)\s*—\s*интерфейс на базе CGI и вкладок\.\s*Разработчик:\s*([^\s]+)/g,
      lang === 'tr'
        ? 'Entware Manager v$1 — CGI ve sekme tabanlı arayüz. Geliştirici: $2'
        : 'Entware Manager v$1 — CGI and tab-based interface. Developer: $2');

    // Common dynamic process/service labels.
    out = out.replace(/Процессы:\s*(.+)/g,
      lang === 'tr' ? 'İşlemler: $1' : 'Processes: $1');
    out = out.replace(/Порт:\s*(\d+)/g,
      lang === 'tr' ? 'Port: $1' : 'Port: $1');

    return out;
  }

  function translateValue(value, lang, allowFallback) {
    if (!value || typeof value !== 'string' || lang === 'ru') return value;
    var normalized = normalizeSpaces(value);
    var trimmed = normalized.trim();
    if (!trimmed) return value;

    var dict = mapFor(lang);
    if (Object.prototype.hasOwnProperty.call(dict, trimmed)) {
      var pos = normalized.indexOf(trimmed);
      return normalized.slice(0,pos) + dict[trimmed] + normalized.slice(pos + trimmed.length);
    }

    var out = translatePattern(normalized, lang);

    var fragments = SAFE_FRAGMENTS[lang] || {};
    Object.keys(fragments).sort(function(a,b){return b.length-a.length;}).forEach(function(key){
      if (out.indexOf(key) !== -1) out = out.split(key).join(fragments[key]);
    });

    // Try exact mapping again after dynamic substitutions.
    var outTrim = out.trim();
    if (Object.prototype.hasOwnProperty.call(dict, outTrim)) {
      var p = out.indexOf(outTrim);
      out = out.slice(0,p) + dict[outTrim] + out.slice(p + outTrim.length);
    }

    if (allowFallback !== false && hasCyrillic(out)) {
      if (fallbackSeen.indexOf(trimmed) === -1) fallbackSeen.push(trimmed);
      // Fail closed for localization: never expose untranslated Russian in TR/EN UI.
      // Keep obvious Latin/numeric technical tokens when possible.
      var tokens = out.match(/(?:\/[^\s,;:()]+)|(?:[A-Za-z0-9_.:@-]+)|(?:\d+(?:\.\d+)?%?)/g) || [];
      var suffix = tokens.length ? ' (' + tokens.slice(0,6).join(' ') + ')' : '';
      return lang === 'tr' ? 'Arayüz metni' + suffix : 'Interface text' + suffix;
    }
    return out;
  }

  function ignoredElement(el) {
    return !!(el && el.closest && el.closest('script,style'));
  }

  function translateTextNode(node, lang) {
    if (!node || node.nodeType !== 3 || ignoredElement(node.parentElement)) return;
    var next = translateValue(node.nodeValue, lang, true);
    if (next !== node.nodeValue) node.nodeValue = next;
  }

  function translateElement(el, lang) {
    if (!el || el.nodeType !== 1 || ignoredElement(el)) return;

    ['title','placeholder','aria-label'].forEach(function(name){
      if (!el.hasAttribute(name)) return;
      var cur = el.getAttribute(name);
      var next = translateValue(cur, lang, true);
      if (next !== cur) el.setAttribute(name, next);
    });

    // Buttons implemented as input[value].
    if (el.tagName === 'INPUT') {
      var type = String(el.type || '').toLowerCase();
      if (type === 'button' || type === 'submit' || type === 'reset') {
        var v = el.value;
        var nv = translateValue(v, lang, true);
        if (nv !== v) el.value = nv;
      } else if ((type === 'text' || type === 'search') && hasCyrillic(el.value)) {
        // Default link names are editable inputs. Translate for display without
        // dispatching input/change events; app state is untouched unless user saves.
        if (!el.dataset.ewmI18nSource) el.dataset.ewmI18nSource = el.value;
        var tv = translateValue(el.value, lang, true);
        if (tv !== el.value) el.value = tv;
      }
    }
  }

  function translateTree(root, lang) {
    if (!root || lang === 'ru') return;
    if (root.nodeType === 3) {
      translateTextNode(root, lang);
      return;
    }
    if (root.nodeType !== 1 && root.nodeType !== 9) return;

    if (root.nodeType === 1) translateElement(root, lang);
    var walker = document.createTreeWalker(root, NodeFilter.SHOW_ELEMENT | NodeFilter.SHOW_TEXT);
    var n;
    while ((n = walker.nextNode())) {
      if (n.nodeType === 3) translateTextNode(n, lang);
      else translateElement(n, lang);
    }
  }

  function localizeHelp(lang) {
    if (lang === 'ru') return;
    var content = document.getElementById('helpContent');
    if (!content) return;
    if (content.getAttribute('data-ewm-v5-lang') === lang) return;
    content.innerHTML = HELP[lang] || '';
    content.setAttribute('data-ewm-v5-lang', lang);
  }

  function makeSelect(lang) {
    var select = document.createElement('select');
    select.className = 'ewm-language-select';
    select.title = lang === 'tr' ? 'Arayüz dili' : lang === 'en' ? 'Interface language' : 'Язык интерфейса';
    select.setAttribute('aria-label', select.title);
    Object.keys(SUPPORTED).forEach(function(code){
      var option = document.createElement('option');
      option.value = code;
      if (code === 'ru') option.textContent = lang === 'tr' ? 'Rusça' : lang === 'en' ? 'Russian' : 'Русский';
      else if (code === 'en') option.textContent = lang === 'ru' ? 'English' : 'English';
      else option.textContent = lang === 'ru' ? 'Türkçe' : 'Türkçe';
      select.appendChild(option);
    });
    select.value = lang;
    select.addEventListener('change', function(){ setLanguage(select.value); });
    return select;
  }

  function addSelector(lang) {
    var footer = document.querySelector('.sidebar-footer');
    if (footer && !footer.querySelector('.ewm-language-select')) {
      var wrap = document.createElement('div');
      wrap.className = 'ewm-language-wrap';
      wrap.appendChild(makeSelect(lang));
      footer.insertBefore(wrap, footer.firstChild);
    }

    var card = document.querySelector('.login-card');
    if (card && !card.querySelector('.ewm-language-select')) {
      var loginWrap = document.createElement('div');
      loginWrap.className = 'ewm-language-wrap';
      var loginSelect = makeSelect(lang);
      loginSelect.style.maxWidth = '100%';
      loginWrap.appendChild(loginSelect);
      card.appendChild(loginWrap);
    }
  }

  function audit(lang) {
    if (lang === 'ru') return [];
    var found = [];
    var walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
    var n;
    while ((n = walker.nextNode())) {
      if (ignoredElement(n.parentElement)) continue;
      var s = String(n.nodeValue || '').trim();
      if (s && hasCyrillic(s) && found.indexOf(s) === -1) found.push(s);
    }
    document.querySelectorAll('[title],[placeholder],[aria-label]').forEach(function(el){
      ['title','placeholder','aria-label'].forEach(function(a){
        var s = el.getAttribute(a);
        if (s && hasCyrillic(s) && found.indexOf(s) === -1) found.push(s);
      });
    });
    return found;
  }

  function runPass(lang) {
    if (lang === 'ru') {
      addSelector(lang);
      document.documentElement.lang = 'ru';
      return;
    }
    busy = true;
    try {
      localizeHelp(lang);
      translateTree(document.body, lang);
      addSelector(lang);
      document.documentElement.lang = lang === 'tr' ? 'tr' : 'en';
    } finally {
      busy = false;
    }
  }

  function start() {
    var lang = currentLanguage();
    runPass(lang);

    observer = new MutationObserver(function(mutations){
      if (busy) return;
      var current = currentLanguage();
      if (current === 'ru') {
        addSelector(current);
        return;
      }
      busy = true;
      try {
        localizeHelp(current);
        mutations.forEach(function(m){
          if (m.type === 'characterData') translateTextNode(m.target, current);
          if (m.type === 'attributes') translateElement(m.target, current);
          if (m.addedNodes) Array.prototype.forEach.call(m.addedNodes, function(node){
            translateTree(node, current);
          });
        });
        addSelector(current);
      } finally {
        busy = false;
      }
    });

    observer.observe(document.body, {
      subtree:true,
      childList:true,
      characterData:true,
      attributes:true,
      attributeFilter:['title','placeholder','aria-label']
    });

    // Lazy tabs can update shortly after insertion.
    [150,500,1200,2500].forEach(function(ms){
      setTimeout(function(){ runPass(currentLanguage()); }, ms);
    });

    window.EwmI18nV5 = {
      language: currentLanguage,
      setLanguage: setLanguage,
      audit: function(){ return audit(currentLanguage()); },
      fallbacks: function(){ return fallbackSeen.slice(); },
      rerun: function(){ runPass(currentLanguage()); return audit(currentLanguage()); }
    };

    setTimeout(function(){
      var left = audit(currentLanguage());
      console.info('[EWM i18n V5] language=' + currentLanguage() +
        ' cyrillic-visible=' + left.length +
        ' fallbacks=' + fallbackSeen.length);
      if (left.length) console.warn('[EWM i18n V5] remaining Cyrillic:', left);
      if (fallbackSeen.length) console.warn('[EWM i18n V5] fallback sources:', fallbackSeen);
    }, 3000);
  }

  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', start, {once:true});
  else start();
})();
</script>
<!-- EWM_LOCAL_I18N_V5_END -->
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

echo "V5 patch uygulandi."
echo "Eski local i18n V1-V4 bloklari kaldirildi; tek V5 runtime etkin."
echo "Diller: Русский / English / Türkçe"
echo "TR/EN modunda gorunen Kiril metinler fail-closed temizlenir ve audit edilebilir."
echo "Tarayicida Ctrl+F5 yap."
