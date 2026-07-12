#!/bin/sh
echo "Content-type: text/plain"
echo ""
echo "PATH=$PATH"
echo "---"
which cat 2>&1 || echo "cat not found"
which sed 2>&1 || echo "sed not found"
cat /opt/var/log/entware/system.log 2>&1 || echo "cat failed"
