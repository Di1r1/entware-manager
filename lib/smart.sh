#!/bin/sh
# ==============================================
# Entware Manager - SMART detection and parsing
# Версия: 1.0
# ==============================================

# Обнаружение физических дисков
# Читает /proc/partitions (BusyBox-совместимо)
# Возвращает: имя устройства (sdX) построчно
smart_discover_disks() {
    awk '{print $4}' /proc/partitions 2>/dev/null | grep -E '^(sd[a-z]$|nvme[0-9]+n[0-9]+$)' | sort -u
}

# Размер диска в байтах из /proc/partitions
smart_disk_size() {
    local dev="$1"
    local blocks
    blocks=$(awk -v dev="$dev" '$4 == dev {print $3}' /proc/partitions 2>/dev/null)
    [ -n "$blocks" ] && echo $((blocks * 1024)) || echo ""
}

# Определение типа для smartctl -d
smart_detect_type() {
    local device="$1"
    case "$device" in
        nvme*) echo "nvme" ;;
        sd*)   echo "sat" ;;
        *)     echo "sat" ;;
    esac
}

# Выполнение smartctl с sudo и таймаутом
smartctl_run() {
    local device="$1"
    local args="$2"
    local smartctl="/opt/sbin/smartctl"
    local output

    [ -x "$smartctl" ] || { echo "ERROR: smartctl not found"; return 1; }

    _timeout=""
    command -v timeout >/dev/null 2>&1 && _timeout="timeout 10"

    if command -v sudo >/dev/null 2>&1; then
        output=$($_timeout sudo "$smartctl" $args "$device" 2>&1)
    else
        output=$($_timeout "$smartctl" $args "$device" 2>&1)
    fi

    echo "$output"
}

# Извлечение значения из smartctl вывода
smart_extract() {
    local key="$1"
    local output="$2"
    echo "$output" | grep -i "$key" | head -1 | awk '{print $NF}'
}

# Экранирование строки для JSON
smart_escape() {
    echo "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

# Экранирование многострочной строки для JSON (newlines → \n)
smart_escape_multiline() {
    echo "$1" | sed 's/\\/\\\\/g; s/"/\\"/g' | awk '{printf "%s\\n", $0}' | sed 's/\\n$//'
}

# Получение JSON-объекта диска
smart_disk_json() {
    local device="$1"
    local devpath="/dev/$device"
    local disk_type disk_size model serial health temperature power_on
    local output

    disk_type=$(smart_detect_type "$device")
    disk_size=$(smart_disk_size "$device")
    output=$(smartctl_run "$devpath" "-a -d $disk_type")

    model=$(echo "$output" | grep "Device Model" | head -1 | cut -d: -f2- | sed 's/^ *//;s/ *$//')
    [ -z "$model" ] && model=$(echo "$output" | grep "Model Number" | head -1 | cut -d: -f2- | sed 's/^ *//;s/ *$//')
    [ -z "$model" ] && model=$(echo "$output" | grep "Product" | head -1 | cut -d: -f2- | sed 's/^ *//;s/ *$//')
    [ -z "$model" ] && model="Unknown"

    serial=$(echo "$output" | grep "Serial" | grep -v "Number" | head -1 | cut -d: -f2- | sed 's/^ *//;s/ *$//')
    [ -z "$serial" ] && serial=$(echo "$output" | grep "Serial Number" | head -1 | cut -d: -f2- | sed 's/^ *//;s/ *$//')
    [ -z "$serial" ] && serial="\u2014"

    # Health
    health=$(echo "$output" | grep "SMART overall-health" | head -1 | awk '{print $NF}')
    [ -z "$health" ] && health=$(echo "$output" | grep "SMART Health Status" | head -1 | awk '{print $NF}')
    [ -z "$health" ] && health="UNKNOWN"

    # Temperature — берём 10-е поле (первое слово RAW_VALUE)
    temperature=$(echo "$output" | grep -i "Temperature_Celsius" | head -1 | awk '{print $10}')
    [ -z "$temperature" ] && temperature=$(echo "$output" | grep -i "Current Temperature" | head -1 | awk '{print $10}')
    # Если всё ещё пусто — пробуем grep "194 " (ID Temperature_Celsius)
    [ -z "$temperature" ] && temperature=$(echo "$output" | grep "194 " | head -1 | awk '{print $10}')
    # Проверяем, что это число
    case "$temperature" in
        ''|*[!0-9]*) temperature="null" ;;
    esac

    # Power on hours — тоже 10-е поле
    power_on=$(echo "$output" | grep -i "Power_On_Hours" | head -1 | awk '{print $10}' | tr -d '+')
    [ -z "$power_on" ] && power_on=$(echo "$output" | grep -i "Power On Hours" | head -1 | awk '{print $10}')
    [ -z "$power_on" ] && power_on=$(echo "$output" | grep "^ *9 " | head -1 | awk '{print $10}')
    case "$power_on" in
        ''|*[!0-9]*) power_on="null" ;;
    esac
    [ -z "$power_on" ] && power_on="null"

    # Экранируем строки
    model=$(smart_escape "$model")
    serial=$(smart_escape "$serial")

    printf '{"device":"/dev/%s","model":"%s","serial":"%s","size":"%s","type":"%s","health":"%s","temperature":%s,"power_on_hours":%s}' \
        "$device" "$model" "$serial" "$disk_size" "$disk_type" "$health" "$temperature" "$power_on"
}

# Парсинг атрибутов SMART (-A) в JSON-массив
smart_attributes_json() {
    local device="$1"
    local devpath="/dev/$device"
    local disk_type output lines
    local first result id name raw value worst thresh line r

    disk_type=$(smart_detect_type "$device")
    output=$(smartctl_run "$devpath" "-A -d $disk_type")
    lines=$(echo "$output" | awk 'NR>1 && /^[[:space:]]*[0-9]+/ {print $0}')
    result=""
    first=1

    while read -r line; do
        [ -z "$line" ] && continue

        id=$(echo "$line" | awk '{print $1}')
        name=$(echo "$line" | awk '{print $2}')
        value=$(echo "$line" | awk '{print $4}')
        worst=$(echo "$line" | awk '{print $5}')
        thresh=$(echo "$line" | awk '{print $6}')

        # RAW_VALUE — с 10-го поля (после WHEN_FAILED)
        raw=$(echo "$line" | awk '{
            for(i=10;i<=NF;i++) {
                if(i==10) r=$i; else r=r" "$i
            }
            print r
        }')

        value=$(echo "$value" | sed 's/^0*//'); [ -z "$value" ] && value=0
        worst=$(echo "$worst" | sed 's/^0*//'); [ -z "$worst" ] && worst=0
        thresh=$(echo "$thresh" | sed 's/^0*//'); [ -z "$thresh" ] && thresh=0
        [ "$thresh" = "-" ] && thresh="0"

        raw=$(echo "$raw" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
        [ -z "$raw" ] || [ "$raw" = "-" ] && raw="0"

        name=$(smart_escape "$name")
        raw=$(smart_escape "$raw")

        if [ "$first" -eq 1 ]; then
            first=0
        else
            result="${result},"
        fi
        result="${result}{\"id\":$id,\"name\":\"$name\",\"value\":$value,\"worst\":$worst,\"threshold\":$thresh,\"raw\":\"$raw\"}"
    done <<SMART_ATTR
$lines
SMART_ATTR

    echo "{\"attributes\":[${result}]}"
}

# Проверка здоровья
smart_health_json() {
    local device="$1"
    local devpath="/dev/$device"
    local disk_type output health msg

    disk_type=$(smart_detect_type "$device")
    output=$(smartctl_run "$devpath" "-H -d $disk_type")

    health=$(echo "$output" | grep "SMART overall-health" | head -1)
    [ -z "$health" ] && health=$(echo "$output" | grep "SMART Health Status" | head -1)
    [ -z "$health" ] && health="SMART: Unable to determine health status"

    msg=$(smart_escape "$(echo "$health" | sed 's/^ *//;s/ *$//')")
    result=$(echo "$health" | awk '{print $NF}')

    printf '{"health":"%s","message":"%s"}' "$result" "$msg"
}

# Информация о диске (smartctl -i)
smart_info_json() {
    local device="$1"
    local devpath="/dev/$device"
    local disk_type output info

    disk_type=$(smart_detect_type "$device")
    output=$(smartctl_run "$devpath" "-i -d $disk_type")
    info=$(smart_escape_multiline "$output")

    printf '{"info":"%s"}' "$info"
}

# Запуск самотеста
smart_test_start() {
    local device="$1"
    local type="$2"
    local devpath="/dev/$device"
    local disk_type output

    disk_type=$(smart_detect_type "$device")
    output=$(smartctl_run "$devpath" "-t $type -d $disk_type")

    case "$output" in
        *"START"*)
            printf '{"status":"ok","message":"Тест %s запущен"}' "$type"
            ;;
        *"already"*)
            printf '{"status":"error","message":"Тест уже выполняется"}'
            ;;
        *)
            msg=$(smart_escape "$(echo "$output" | head -5 | tr '\n' ' ' | sed 's/^ *//;s/ *$//')")
            printf '{"status":"error","message":"%s"}' "$msg"
            ;;
    esac
}

# Статус самотеста
smart_test_status() {
    local device="$1"
    local devpath="/dev/$device"
    local disk_type output line status progress

    disk_type=$(smart_detect_type "$device")
    output=$(smartctl_run "$devpath" "-l selftest -d $disk_type")

    # Первая строка после заголовка — последний тест
    line=$(echo "$output" | grep "^#" | head -1)
    if [ -n "$line" ]; then
        status=$(echo "$line" | awk '{print $5}')
        progress=$(echo "$line" | awk '{print $NF}')
        [ -z "$progress" ] || [ "$progress" = "-" ] && progress="100"
        printf '{"status":"%s","progress":%s}' "$status" "$progress"
    else
        # Может быть "No self-tests have been logged"
        printf '{"status":"No tests logged","progress":100}'
    fi
}

# Информация о разделах диска (через df)
smart_disk_usage() {
    local dev="$1"
    command -v df >/dev/null 2>&1 || { echo '{"partitions":[]}'; return; }
    local first=1
    echo -n '{"partitions":['
    df -h 2>/dev/null | while IFS= read -r line; do
        case "$line" in
            "/dev/${dev}"[0-9]*)
                set -- $line
                local pct=$(echo "$5" | sed 's/%//')
                [ "$first" -eq 1 ] && first=0 || echo -n ','
                echo -n "{\"part\":\"$(basename $1)\",\"size\":\"$2\",\"used\":\"$3\",\"avail\":\"$4\",\"pct\":$pct,\"mnt\":\"$6\"}"
                ;;
        esac
    done
    echo ']}'
}
