Привет!

Сделал веб-морду для управления Entware на Keenetic / Netcraze.
Всё прямо в браузере, SSH не нужен.

Что умеет:
• ставить/удалять/обновлять пакеты
• смотреть температуру CPU и WiFi (с графиками за неделю)
• управлять сервисами (вкл/выкл/перезапуск)
• следить за сетью (интерфейсы, трафик, ARP)
• смотреть SMART дисков (HDD/SSD/NVMe)
• читать логи, смотреть файлы
• веб-терминал прямо в браузере
• автоперезапуск упавших процессов (watchdog)

Всё на одной странице, тёмная тема, работает с телефона.

Как установить:

  1. Заходим на роутер по SSH
  2. Смотрим архитектуру: uname -m

     aarch64 → entware-manager-v1.06.1-arm64.tar.gz
     armv7l / armv5tel → entware-manager-v1.06.1-arm.tar.gz
     mips → entware-manager-v1.06.1-mips.tar.gz
     mipsel → entware-manager-v1.06.1-mipsel.tar.gz
     x86_64 → entware-manager-v1.06.1-amd64.tar.gz
     i686 → entware-manager-v1.06.1-386.tar.gz

  3. Качаем архив под свою архитектуру:

     # arm64 (aarch64)
     curl -LO https://github.com/Di1r1/entware-manager/releases/download/v1.06.1/entware-manager-v1.06.1-arm64.tar.gz

     # arm (armv7l / armv5tel)
     curl -LO https://github.com/Di1r1/entware-manager/releases/download/v1.06.1/entware-manager-v1.06.1-arm.tar.gz

     # mips
     curl -LO https://github.com/Di1r1/entware-manager/releases/download/v1.06.1/entware-manager-v1.06.1-mips.tar.gz

     # mipsel
     curl -LO https://github.com/Di1r1/entware-manager/releases/download/v1.06.1/entware-manager-v1.06.1-mipsel.tar.gz

     # amd64 (x86_64)
     curl -LO https://github.com/Di1r1/entware-manager/releases/download/v1.06.1/entware-manager-v1.06.1-amd64.tar.gz

     # 386 (i686)
     curl -LO https://github.com/Di1r1/entware-manager/releases/download/v1.06.1/entware-manager-v1.06.1-386.tar.gz

  4. Распаковываем и ставим:

     tar -xzf entware-manager-v1.06.1-*.tar.gz
     cd deploy && sh Install/install.sh

  5. Открываем http://192.168.1.1:8087/entware-manager/

Все архивы: https://github.com/Di1r1/entware-manager/releases

Обновление — теми же командами.
Удаление — sh /opt/web_entware/Install/uninstall.sh

Важно: я только разработчик, протестировать установку на всех архитектурах
не могу — есть только arm64. Если кто-то поставит на mips/mipsel/arm/x86 —
напишите, работает или нет.

Исходники: https://github.com/Di1r1/entware-manager

Пользуйтесь, спрашивайте)
