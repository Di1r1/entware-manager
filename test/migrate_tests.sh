#!/bin/sh
# shellcheck disable=SC3043,SC2086,SC2046
# ==============================================
# Copyright (c) 2026 Di1r1 — https://github.com/Di1r1/entware-manager
# Юнит-тесты lib/migrate.sh (миграция lighttpd → entware-server).
# Запуск: sh test/migrate_tests.sh  (подхватывается в make test)
# ==============================================
set -u

cd "$(dirname "$0")/.." || exit 1

T=$(mktemp -d) || exit 1
trap 'rm -rf "$T"' EXIT

mkdir -p "$T/conf.d" "$T/fakebin"

# Фейковый netstat (BusyBox на роутере не имеет ss; здесь — только netstat).
cat > "$T/fakebin/netstat" <<'EOF'
#!/bin/sh
while IFS= read -r p; do
	[ -z "$p" ] && continue
	echo "tcp        0      0 0.0.0.0:$p 0.0.0.0:*               LISTEN"
done < "${FAKE_LISTEN:-/dev/null}"
EOF
chmod +x "$T/fakebin/netstat"

export EWM_PATH_PREFIX="$T/fakebin"
export FAKE_LISTEN="$T/listens"
export EWM_LIGHTTPD_DIR="$T"
export EWM_LIGHTTPD_CONF="$T/lighttpd.conf"
export EWM_PORT_KEEPER="$T/conf.d/90-entware-manager.conf"

. ./lib/migrate.sh

PASS=0
FAIL=0
assert_eq() { # desc expected actual
	if [ "$2" = "$3" ]; then
		PASS=$((PASS + 1)); echo "ok: $1"
	else
		FAIL=$((FAIL + 1)); echo "FAIL: $1 (expected='$2' actual='$3')"
	fi
}
rc_ok() { # desc expected_rc actual_rc
	if [ "$2" = "$3" ]; then
		PASS=$((PASS + 1)); echo "ok: $1"
	else
		FAIL=$((FAIL + 1)); echo "FAIL: $1 (rc expected '$2' got '$3')"
	fi
}

# --- setup: общий пустой конфиг ---
: > "$T/lighttpd.conf"
: > "$T/listens"

echo "=== migrate_effective_lighttpd_port ==="
: > "$T/lighttpd.conf"
rm -f "$T/conf.d"/*.conf
assert_eq "пусто (нет server.port → 80)" "" "$(migrate_effective_lighttpd_port)"

printf 'server.port = 8080\n' > "$T/lighttpd.conf"
assert_eq "server.port в main conf" "8080" "$(migrate_effective_lighttpd_port)"

printf 'server.port = 8080\n' > "$T/lighttpd.conf"
printf 'server.port = 8081\n' > "$T/conf.d/98-test.conf"
assert_eq "last-wins (conf.d после main)" "8081" "$(migrate_effective_lighttpd_port)"

printf 'server.port := 8082\n' > "$T/lighttpd.conf"
rm -f "$T/conf.d"/*.conf
assert_eq "форма ':=' (nfqws)" "8082" "$(migrate_effective_lighttpd_port)"

printf 'server.port = 8083 # комментарий\n' > "$T/lighttpd.conf"
assert_eq "хвостовой комментарий" "8083" "$(migrate_effective_lighttpd_port)"

cat > "$T/conf.d/98-test.conf" <<'EOF'
$SERVER["socket"] == ":8443" {
    server.port = 8443
}
server.port = 8090
EOF
printf 'server.port = 8080\n' > "$T/lighttpd.conf"
assert_eq "scoped-блок пропускается, берётся глобальный" "8090" "$(migrate_effective_lighttpd_port)"

# Наш 90-conf пропускается парсером
printf 'server.port = 8087\n' > "$T/conf.d/90-entware-manager.conf"
rm -f "$T/conf.d/98-test.conf"
printf '' > "$T/lighttpd.conf"
assert_eq "наш 90-conf пропускается" "" "$(migrate_effective_lighttpd_port)"

echo "=== migrate_port_free ==="
printf '80\n' > "$T/listens"
rc_ok "80 занят" 1 $(migrate_port_free 80; echo $?)
rc_ok "8086 свободен" 0 $(migrate_port_free 8086; echo $?)

echo "=== migrate_choose_portkeeper ==="
# A: 80 занят, чужих нет, eff пусто → 8086
printf '80\n' > "$T/listens"
rm -f "$T/conf.d"/*.conf
printf '' > "$T/lighttpd.conf"
assert_eq "80 занят, чужих нет → 8086" "8086" "$(migrate_choose_portkeeper)"

# B: 80 свободен, чужих нет → удалить ("")
: > "$T/listens"
assert_eq "80 свободен, чужих нет → удалить" "" "$(migrate_choose_portkeeper)"

# C: koffe в conf.d + eff свободен 8085 → держим 8085
printf 'server.port = 8085\n' > "$T/lighttpd.conf"
printf 'alias.url += ( "/koffe/" => "/opt/koffe/web/" )\n' > "$T/conf.d/98-koffe-web.conf"
assert_eq "чужие есть, eff свободен → держим 8085" "8085" "$(migrate_choose_portkeeper)"

# D: EWM_LIGHTTPD_PORT override
export EWM_LIGHTTPD_PORT=8092
assert_eq "override EWM_LIGHTTPD_PORT=8092" "8092" "$(migrate_choose_portkeeper)"
unset EWM_LIGHTTPD_PORT

# E: eff занят (main conf server.port=8087, 8087 слушается), чужих нет → 8086
rm -f "$T/conf.d"/*.conf
printf 'server.port = 8087\n' > "$T/lighttpd.conf"
printf '8087\n' > "$T/listens"
assert_eq "eff 8087 занят → 8086" "8086" "$(migrate_choose_portkeeper)"

echo "=== migrate_is_portkeeper / write_portkeeper ==="
rm -f "$T/conf.d"/*.conf
: > "$T/listens"
printf 'server.port = 8086\n' > "$T/lighttpd.conf"
migrate_write_portkeeper 8086
rc_ok "write_portkeeper → is_portkeeper true" 0 $(migrate_is_portkeeper "$EWM_PORT_KEEPER"; echo $?)
grep -q "server.port := 8086" "$EWM_PORT_KEEPER" && rc_ok "порт 8086 в файле (:=')" 0 0 || rc_ok "порт 8086 в файле (:=')" 1 0

# не порт-хранитель (без маркера)
printf 'server.port = 8087\n' > "$T/conf.d/90-entware-manager.conf"
rc_ok "без маркера → false" 1 $(migrate_is_portkeeper "$EWM_PORT_KEEPER"; echo $?)

# маркер есть, но 2 строки server.port → false
cat > "$T/conf.d/90-entware-manager.conf" <<EOF
# $PORT_KEEPER_MARKER
server.port = 8086
server.port = 8087
EOF
rc_ok "маркер + 2 server.port → false" 1 $(migrate_is_portkeeper "$EWM_PORT_KEEPER"; echo $?)

echo "=== write_portkeeper: условный mod_cgi ==="
# чужие conf.d с cgi.assign → mod_cgi добавляется
printf 'cgi.assign = ( ".php" => "/opt/bin/php-cgi" )\n' > "$T/conf.d/98-test.conf"
migrate_write_portkeeper 8086
if grep -q 'mod_cgi' "$EWM_PORT_KEEPER"; then PASS=$((PASS + 1)); echo "ok: cgi.assign есть → mod_cgi добавлен"; else FAIL=$((FAIL + 1)); echo "FAIL: cgi.assign есть → mod_cgi НЕ добавлен"; fi

# без cgi.assign → mod_cgi нет
rm -f "$T/conf.d"/*.conf
printf 'alias.url += ( "/koffe/" => "/opt/koffe/web/" )\n' > "$T/conf.d/98-koffe-web.conf"
migrate_write_portkeeper 8086
if grep -q 'mod_cgi' "$EWM_PORT_KEEPER"; then FAIL=$((FAIL + 1)); echo "FAIL: без cgi.assign → mod_cgi лишний"; else PASS=$((PASS + 1)); echo "ok: без cgi.assign → mod_cgi нет"; fi
if grep -q 'mod_alias' "$EWM_PORT_KEEPER"; then PASS=$((PASS + 1)); echo "ok: alias.url есть → mod_alias добавлен"; else FAIL=$((FAIL + 1)); echo "FAIL: alias.url есть → mod_alias НЕ добавлен"; fi

# перенос server.modules из прежнего (полного) 90-conf — баг koffe (mod_alias)
rm -f "$T/conf.d"/*.conf
cat > "$EWM_PORT_KEEPER" <<'EOF'
server.port = 8087
server.modules += ( "mod_alias" )
server.modules += ( "mod_cgi" )
server.modules += ( "mod_proxy" )
EOF
migrate_write_portkeeper 8086
for m in mod_alias mod_cgi mod_proxy; do
	if grep -q "$m" "$EWM_PORT_KEEPER"; then PASS=$((PASS + 1)); echo "ok: модуль $m перенесён из старого 90-conf"; else FAIL=$((FAIL + 1)); echo "FAIL: модуль $m НЕ перенесён"; fi
done
grep -q "server.port := 8086" "$EWM_PORT_KEEPER" && PASS=$((PASS + 1)) || FAIL=$((FAIL + 1)); echo "  port 8086 (:=')"

echo "=== идемпотентность ==="
# повторная запись того же порт-хранителя — no-op (md5 не меняется)
m1=$(md5sum "$EWM_PORT_KEEPER" | awk '{print $1}')
migrate_write_portkeeper 8086
m2=$(md5sum "$EWM_PORT_KEEPER" | awk '{print $1}')
if [ "$m1" = "$m2" ]; then PASS=$((PASS + 1)); echo "ok: повторная запись идемпотентна (md5 совпадает)"; else FAIL=$((FAIL + 1)); echo "FAIL: повторная запись изменила файл"; fi
# choose на существующем порт-хранителе возвращает его порт, даже если он «занят»
printf '8086\n' > "$T/listens"
assert_eq "choose на существующем порт-хранителе → 8086" "8086" "$(migrate_choose_portkeeper)"

echo "=== healing: старый 'server.port = N' → ':=' ==="
# порт-хранитель от установки ≤1.16.18 (форма "=") переписывается в ":="
printf '8086\n' > "$T/listens"
cat > "$EWM_PORT_KEEPER" <<EOF
# Entware Manager $PORT_KEEPER_MARKER — shared lighttpd stays on 8086.
server.port = 8086
EOF
migrate_write_portkeeper 8086
if grep -q "server.port := 8086" "$EWM_PORT_KEEPER" && ! grep -q "server.port = 8086" "$EWM_PORT_KEEPER"; then
	PASS=$((PASS + 1)); echo "ok: старый '=' переписан в ':='"
else
	FAIL=$((FAIL + 1)); echo "FAIL: старый '=' не переписан в ':='"
fi

echo "=== migrate_has_third_party_confd ==="
rm -f "$T/conf.d"/*.conf
printf 'server.port = 8086\n' > "$T/conf.d/90-entware-manager.conf"
rc_ok "только наш 90-conf → чужих нет" 1 $(migrate_has_third_party_confd; echo $?)
printf 'alias.url += ( "/koffe/" => "/opt/koffe/web/" )\n' > "$T/conf.d/98-koffe-web.conf"
rc_ok "koffe → чужие есть" 0 $(migrate_has_third_party_confd; echo $?)

echo ""
echo "========================================"
echo "PASS: $PASS  FAIL: $FAIL"
[ "$FAIL" = "0" ] || exit 1
echo "ВСЕ ТЕСТЫ ПРОЙДЕНЫ"
