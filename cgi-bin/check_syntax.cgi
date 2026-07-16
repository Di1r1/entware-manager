#!/bin/sh
# ==============================================
# Entware Manager - проверка синтаксиса всех скриптов
# Версия: 1.1 (исправлена логика подсчета ошибок)
# Дата: 2026-05-02
# ==============================================

. /opt/web_entware/lib/common.sh

BASE_DIR="/opt/web_entware"
TOTAL_ERRORS=0
first=true

# Функция проверки одного файла
# Возвращает: "ok" или "error: <сообщение>"
check_file() {
    _file="$1"
    _output=$(sh -n "$_file" 2>&1)
    if [ $? -eq 0 ]; then
        echo "ok"
    else
        echo "error: $_output"
    fi
}

# Сбор файлов для проверки (sh/cgi скрипты)
FILES_LIST=""
# CGI скрипты основной папки
for f in "$BASE_DIR"/cgi-bin/*.cgi; do
    [ -f "$f" ] && FILES_LIST="$FILES_LIST $f"
done
# CGI скрипты в подпапках
for f in "$BASE_DIR"/cgi-bin/*/*.cgi; do
    [ -f "$f" ] && FILES_LIST="$FILES_LIST $f"
done
# Библиотеки
for f in "$BASE_DIR"/lib/*.sh; do
    [ -f "$f" ] && FILES_LIST="$FILES_LIST $f"
done

# Сбор JSON
content=$( 
    printf '{"results":['
    for f in $FILES_LIST; do
        [ ! -f "$f" ] && continue
        
        rel_path=$(echo "$f" | sed "s|$BASE_DIR/||")
        result=$(check_file "$f")
        
        if [ "$first" = "true" ]; then
            first=false
        else
            printf ','
        fi
        
        status="ok"
        msg=""
        if echo "$result" | grep -q "^error:"; then
            status="error"
            msg=$(echo "$result" | sed 's/^error: //' | sed 's/"/\\"/g')
            TOTAL_ERRORS=$((TOTAL_ERRORS + 1))
        fi
        
        printf '{"file":"%s","status":"%s","message":"%s"}' "$rel_path" "$status" "$msg"
    done
    
    printf '],"total_errors":%d,"timestamp":"%s"}\n' "$TOTAL_ERRORS" "$(date '+%Y-%m-%d %H:%M:%S')"
)

json_out "$content"
