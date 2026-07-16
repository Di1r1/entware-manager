#!/bin/sh
# ==============================================
# Entware Manager - просмотр содержимого tmpfs
# Версия: 2.0 (снято ограничение на пути)
# Дата: 2026-03-31
# ==============================================

. /opt/web_entware/lib/common.sh

html_header

QUERY_STRING="${QUERY_STRING:-}"
path=$(echo "$QUERY_STRING" | sed -n 's/.*path=\([^&]*\).*/\1/p' | tr -d '\r')
path=$(url_decode "$path")
[ -z "$path" ] && path="/tmp"

case "$path" in
    /*) real_path="$path" ;;
    *)  real_path="$(pwd)/$path" ;;
esac
if [ ! -d "$real_path" ]; then
    echo "<p class='error'>Директория не существует: $real_path</p>"; exit 0
fi

FILEMGR_AUTH="false"
if [ -f "/opt/web_entware/auth_config.json" ]; then
    enabled=$(jq -r '.enabled // false' "/opt/web_entware/auth_config.json" 2>/dev/null)
    [ "$enabled" = "true" ] && FILEMGR_AUTH="true"
fi

human_size() {
    size=$1
    if [ $size -lt 1024 ]; then echo "${size}B"
    elif [ $size -lt 1048576 ]; then echo "$((size / 1024))K"
    elif [ $size -lt 1073741824 ]; then echo "$((size / 1048576))M"
    else echo "$((size / 1073741824))G"; fi
}

parent_path=$(dirname "$real_path")
[ "$parent_path" = "/" ] && parent_path=""

cat <<HTMLHEAD
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>tmpfs: $(html_escape "$real_path")</title>
    <link rel="stylesheet" href="/entware-manager/style.css">
    <script src="/entware-manager/modal.js?v=1"></script>
</head>
<body class="packages-body">
<div class="packages-container">
    <div class="file-manager">
        <h2 style="display: flex; align-items: center; gap: 8px;">
            <span class="stat-icon" style="width: 28px; height: 28px;">
                <svg class="icon" width="28" height="28">
                    <use href="/entware-manager/icons.svg?v=2#icon-folder"/>
                </svg>
            </span>
            tmpfs: $(html_escape "$real_path")
        </h2>
        <div class="path-nav">
            <svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-folder"/></svg>
            <a href="?path=/">root</a>
HTMLHEAD

temp_path=""
for seg in $(echo "$real_path" | tr '/' ' '); do
    [ -z "$seg" ] && continue
    temp_path="$temp_path/$seg"
    echo '<svg class="icon" width="14" height="14"><use href="/entware-manager/icons.svg?v=2#icon-chevron-right"/></svg>'
    if [ "$temp_path" != "$real_path" ]; then
        echo "<a href=\"?path=$temp_path\">$seg</a>"
    else
        echo "<span>$seg</span>"
    fi
done

cat <<HTMLMID
        </div>
        <table class="file-table">
            <thead>
                <th>Имя</th><th>Размер</th><th>Изменён</th><th>Права</th><th>Владелец</th><th>Действие</th>
            </thead>
            <tbody>
HTMLMID

if [ "$real_path" != "/" ] && [ "$real_path" != "/tmp" ] && [ "$real_path" != "/dev" ] && [ "$real_path" != "/dev/shm" ]; then
    echo "    <tr><td colspan='6'><a href=\"?path=$parent_path\"><svg class=\"icon\" width=\"14\" height=\"14\"><use href=\"/entware-manager/icons.svg?v=2#icon-arrow-left\"/></svg> .. (наверх)</a></td></tr>"
fi

ls -lA "$real_path" 2>/dev/null | awk -v realpath="$real_path" -v base="/entware-manager/icons.svg?v=2" '
    $1 != "total" {
        perms = $1
        links = $2
        user = $3
        group = $4
        size = $5
        month = $6
        day = $7
        time = $8
        name = substr($0, index($0, $9))
        gsub(/^[[:space:]]+/, "", name)
        gsub(/&/, "\\&amp;", name)
        gsub(/</, "\\&lt;", name)
        gsub(/>/, "\\&gt;", name)
        gsub(/"/, "\\&quot;", name)
        if (substr(perms,1,1) == "d") {
            icon = "folder"
            icon_class = "folder"
            name_link = "<a href=\"?path=" realpath "/" name "\">" name "</a>"
            action = "<button class=\"delete-file-btn\" data-path=\"" realpath "/" name "\" data-name=\"" name "\" data-type=\"dir\"><svg class=\"icon\" width=\"14\" height=\"14\"><use href=\"/entware-manager/icons.svg?v=2#icon-trash\"/></svg></button>"
        } else {
            icon = "file"
            icon_class = "file"
            name_link = "<a href=\"#\" class=\"file-link\" data-path=\"" realpath "/" name "\">" name "</a>"
            action = "<button class=\"delete-file-btn\" data-path=\"" realpath "/" name "\" data-name=\"" name "\" data-type=\"file\"><svg class=\"icon\" width=\"14\" height=\"14\"><use href=\"/entware-manager/icons.svg?v=2#icon-trash\"/></svg></button>"
        }
        if (size ~ /^[0-9]+$/) {
            if (size < 1024) hsize = size "B"
            else if (size < 1048576) hsize = int(size/1024) "K"
            else if (size < 1073741824) hsize = int(size/1048576) "M"
            else hsize = int(size/1073741824) "G"
        } else hsize = "-"
        date_str = month " " day " " time
        printf "    <tr><td><span class=\"file-icon %s\"><svg class=\"icon\" width=\"16\" height=\"16\"><use href=\"%s#icon-%s\"/></svg></span> %s</td><td>%s</td><td>%s</td><td>%s</td><td>%s:%s</td><td>%s</td></tr>\n",
            icon_class, base, icon, name_link, hsize, date_str, perms, user, group, action
    }' || {
        file_list=$(ls -1A "$real_path" 2>&1)
        if [ $? -ne 0 ]; then
            echo "<tr><td colspan='6' style='text-align:center; color:red;'>Ошибка доступа: $file_list</td></tr>"
        elif [ -z "$file_list" ]; then
            echo '<tr><td colspan="6" style="text-align:center;"><svg class="icon" width="16" height="16"><use href="/entware-manager/icons.svg?v=2#icon-default"/></svg> Директория пуста</td></tr>'
        else
            echo "$file_list" | while read name; do
                [ -z "$name" ] && continue
                full_path="$real_path/$name"
                ls_line=$(ls -ld "$full_path" 2>/dev/null)
                if [ -z "$ls_line" ]; then
                    perms="?"; size="?"; user="?"; group="?"; date_str="?"
                    if [ -d "$full_path" ]; then
                        icon="folder"; icon_class="folder"
                        action="<button class='delete-file-btn' data-path='$full_path' data-name='$name' data-type='dir'><svg class=\"icon\" width=\"14\" height=\"14\"><use href=\"/entware-manager/icons.svg?v=2#icon-trash\"/></svg></button>"
                    else
                        icon="file"; icon_class="file"
                        action="<button class='delete-file-btn' data-path='$full_path' data-name='$name' data-type='file'><svg class=\"icon\" width=\"14\" height=\"14\"><use href=\"/entware-manager/icons.svg?v=2#icon-trash\"/></svg></button>"
                    fi
                else
                    perms=$(echo "$ls_line" | awk '{print $1}')
                    user=$(echo "$ls_line" | awk '{print $3}')
                    group=$(echo "$ls_line" | awk '{print $4}')
                    size=$(echo "$ls_line" | awk '{print $5}')
                    month=$(echo "$ls_line" | awk '{print $6}')
                    day=$(echo "$ls_line" | awk '{print $7}')
                    time=$(echo "$ls_line" | awk '{print $8}')
                    date_str="$month $day $time"
                    if [ "${perms:0:1}" = "d" ]; then
                        icon="folder"; icon_class="folder"
                        action="<button class='delete-file-btn' data-path='$full_path' data-name='$name' data-type='dir'><svg class=\"icon\" width=\"14\" height=\"14\"><use href=\"/entware-manager/icons.svg?v=2#icon-trash\"/></svg></button>"
                    else
                        icon="file"; icon_class="file"
                        action="<button class='delete-file-btn' data-path='$full_path' data-name='$name' data-type='file'><svg class=\"icon\" width=\"14\" height=\"14\"><use href=\"/entware-manager/icons.svg?v=2#icon-trash\"/></svg></button>"
                    fi
                    if [ "$size" -eq "$size" ] 2>/dev/null; then
                        size=$(human_size $size)
                    fi
                fi
                name_esc=$(echo "$name" | sed 's/&/\&amp;/g; s/</\&lt;/g; s/>/\&gt;/g; s/"/\&quot;/g')
                if [ "$icon_class" = "folder" ]; then
                    name_link="<a href=\"?path=$full_path\">$name_esc</a>"
                else
                    name_link="<a href=\"#\" class=\"file-link\" data-path=\"$full_path\">$name_esc</a>"
                fi
                echo "<tr><td><span class='file-icon $icon_class'><svg class='icon' width='16' height='16'><use href='/entware-manager/icons.svg?v=2#icon-$icon'/></svg></span> $name_link</td><td>$size</td><td>$date_str</td><td>$perms</td><td>$user:$group</td><td>$action</td></tr>"
            done
        fi
    }

cat <<'HTMLFOOT'
            </tbody>
         </table>
        <p style="margin-top:1rem;"><a href="javascript:history.back()" class="packages-delete-btn" style="background:#4a5568;"><svg class="icon" width="14" height="14"><use href="/entware-manager/icons.svg?v=2#icon-arrow-left"/></svg> Назад</a></p>
    </div>
</div>
HTMLFOOT

echo '<script>
(function() {'
echo "var AUTH_ENABLED = ${FILEMGR_AUTH:-false};"
echo "var FILEMGR_PASS = (function(){ try { return sessionStorage.getItem('filemgr_pass') || ''; } catch(e) { return ''; } })();"

cat <<'JSEOF'
    function parseSize(str) {
        if (!str) return 0;
        str = str.trim();
        var units = { 'B': 1, 'K': 1024, 'M': 1048576, 'G': 1073741824 };
        var match = str.match(/^([\d.,]+)\s*([KMGT]?B?)$/i);
        if (!match) return 0;
        var val = parseFloat(match[1].replace(',', '.'));
        var unit = match[2].toUpperCase();
        if (unit === 'B') unit = 'B';
        if (!units[unit]) return val;
        return val * units[unit];
    }

    function sortTable(table, colIndex, dataType) {
        var tbody = table.tBodies[0];
        var rows = Array.from(tbody.rows);
        var sortOrder = table.dataset.sortOrder === 'asc' ? 'desc' : 'asc';
        table.dataset.sortOrder = sortOrder;

        rows.sort(function(a, b) {
            var aVal = a.cells[colIndex] ? a.cells[colIndex].innerText.trim() : '';
            var bVal = b.cells[colIndex] ? b.cells[colIndex].innerText.trim() : '';

            if (dataType === 'size') {
                aVal = parseSize(aVal);
                bVal = parseSize(bVal);
            } else if (dataType === 'percent') {
                aVal = parseFloat(aVal);
                bVal = parseFloat(bVal);
            }

            if (sortOrder === 'asc') {
                return aVal > bVal ? 1 : aVal < bVal ? -1 : 0;
            } else {
                return aVal < bVal ? 1 : aVal > bVal ? -1 : 0;
            }
        });

        rows.forEach(function(row) { tbody.appendChild(row); });
        updateSortIndicators(table, colIndex, sortOrder);
    }

    function updateSortIndicators(table, activeCol, sortOrder) {
        var headers = table.querySelectorAll('thead th');
        headers.forEach(function(th, idx) {
            th.classList.remove('sort-asc', 'sort-desc');
            if (idx === activeCol) {
                th.classList.add(sortOrder === 'asc' ? 'sort-asc' : 'sort-desc');
            }
        });
    }

    function enableSorting() {
        var tables = document.querySelectorAll('.file-table');
        tables.forEach(function(table) {
            if (table.dataset.sortable) return;
            table.dataset.sortable = 'true';
            var headers = table.querySelectorAll('thead th');
            headers.forEach(function(th, idx) {
                th.style.cursor = 'pointer';
                th.addEventListener('click', function() {
                    var dataType = 'string';
                    var colText = th.innerText.toLowerCase();
                    if (colText === 'размер') {
                        dataType = 'size';
                    }
                    sortTable(table, idx, dataType);
                });
            });
        });
    }

    function deleteFile(path, type, name) {
        if (!confirm('Удалить ' + (type === 'dir' ? 'папку' : 'файл') + ' "' + name + '"?')) return;

        var body = 'path=' + encodeURIComponent(path);

        if (AUTH_ENABLED) {
            if (!FILEMGR_PASS) {
                FILEMGR_PASS = prompt('Введите пароль для доступа к файловому менеджеру:');
                if (!FILEMGR_PASS) return;
                sessionStorage.setItem('filemgr_pass', FILEMGR_PASS);
            }
            body += '&password=' + encodeURIComponent(FILEMGR_PASS);
        }

        fetch('/entware-cgi/delete_file.cgi', {
            method: 'POST',
            headers: {'Content-Type': 'application/x-www-form-urlencoded'},
            body: body
        })
        .then(function(response) { return response.json(); })
        .then(function(data) {
            if (data.status === 'ok') {
                Toast.show('Удаление выполнено', false);
                setTimeout(function() { location.reload(); }, 1500);
            } else if (data.status === 'error' && data.message === 'Неверный пароль') {
                FILEMGR_PASS = '';
                sessionStorage.removeItem('filemgr_pass');
                Toast.show('Неверный пароль', true);
            } else {
                Toast.show('Ошибка: ' + data.message, true);
            }
        })
        .catch(function(err) {
            Toast.show('Ошибка запроса: ' + err.message, true);
        });
    }

    function escapeHtml(str) {
        return str.replace(/[&<>"]/g, function(m) {
            return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[m];
        });
    }

    function viewFile(path) {
        fetch('/entware-cgi/view_file.cgi?path=' + encodeURIComponent(path))
            .then(function(response) { return response.json(); })
            .then(function(data) {
                if (data.status === 'ok') {
                    var html = '<pre style="max-height:70vh;overflow:auto;background:#1a1a2e;padding:16px;border-radius:8px;font-size:13px;line-height:1.4;color:#e0e0e0;white-space:pre-wrap;word-wrap:break-word;">' + escapeHtml(data.content) + '</pre>';
                    Modal.info(html, 'Файл: ' + data.name);
                } else {
                    Toast.show('Ошибка: ' + data.message, true);
                }
            })
            .catch(function(err) {
                Toast.show('Ошибка запроса: ' + err.message, true);
            });
    }

    document.addEventListener('DOMContentLoaded', function() {
        enableSorting();
        document.querySelectorAll('.delete-file-btn').forEach(function(btn) {
            btn.addEventListener('click', function(e) {
                e.preventDefault();
                deleteFile(this.dataset.path, this.dataset.type, this.dataset.name);
            });
        });
        document.querySelectorAll('.file-link').forEach(function(link) {
            link.addEventListener('click', function(e) {
                e.preventDefault();
                viewFile(this.dataset.path);
            });
        });
    });
})();
</script>
</body>
</html>
JSEOF
