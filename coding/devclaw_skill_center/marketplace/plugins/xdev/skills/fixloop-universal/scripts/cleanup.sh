#!/bin/bash
# cleanup.sh - Kill fixloop-started servers and clean up PID files
# Usage: bash cleanup.sh [port]

if [ -n "$1" ]; then
  # Kill specific port
  PID_FILE="/tmp/fixloop-dev-server-$1.pid"
  MOCK_PID_FILE="/tmp/fixloop-mock-server-$1.pid"

  for f in "$PID_FILE" "$MOCK_PID_FILE"; do
    if [ -f "$f" ]; then
      PID=$(cat "$f")
      if kill -0 "$PID" 2>/dev/null; then
        kill "$PID" 2>/dev/null
        echo "Killed PID $PID (from $f)"
      fi
      rm -f "$f"
    fi
  done
else
  # Kill all fixloop processes
  for f in /tmp/fixloop-*.pid; do
    if [ -f "$f" ]; then
      PID=$(cat "$f")
      if kill -0 "$PID" 2>/dev/null; then
        kill "$PID" 2>/dev/null
        echo "Killed PID $PID (from $f)"
      fi
      rm -f "$f"
    fi
  done
fi

echo "Cleanup complete"
