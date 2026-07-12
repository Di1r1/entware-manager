#!/bin/sh
SYSTEM_LOG="/opt/var/log/entware/system.log"
echo "Content-type: text/plain"
echo ""
echo "--- DEBUG ---"
echo "SYSTEM_LOG=$SYSTEM_LOG"
test -f "$SYSTEM_LOG" && echo "FILE EXISTS" || echo "FILE NOT FOUND"
test -f "$SYSTEM_LOG" && test -s "$SYSTEM_LOG" && echo "FILE NOT EMPTY" || echo "FILE EMPTY OR NOT EXIST"
ls -la "$SYSTEM_LOG" 2>&1
echo "--- END ---"
