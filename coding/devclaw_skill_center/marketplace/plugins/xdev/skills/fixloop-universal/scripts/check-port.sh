#!/bin/bash
# check-port.sh - Check if a port is available, find next free port if not
# Usage: bash check-port.sh [start_port] [end_port]
# Output: prints a single available port number

START=${1:-8005}
END=${2:-8999}

for port in $(seq $START $END); do
  if ! lsof -i :$port -sTCP:LISTEN >/dev/null 2>&1; then
    echo $port
    exit 0
  fi
done

echo "ERROR: No available port in range $START-$END" >&2
exit 1
