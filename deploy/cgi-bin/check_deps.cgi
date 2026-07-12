#!/bin/sh
# ==============================================
# Entware Manager - проверка системных зависимостей
# Версия: 1.0 (модульный, использует common.sh)
# Дата: 2026-05-02
# ==============================================

. /opt/web_entware/lib/common.sh

echo "Content-type: application/json; charset=utf-8"
echo ""

# Вызов функции проверки из библиотеки
check_deps_logic
