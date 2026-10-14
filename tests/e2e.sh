#!/usr/bin/env bash
# e2e: snapshot of 1000 small files must finish <5s. Machine specs printed.
set -euo pipefail
BIN="${BIN:-./bin/kage}"
TH="$(mktemp -d)"
trap 'rm -rf "$TH"' EXIT
export HOME="$TH"
mkdir -p "$TH/.config/many"
"$BIN" init >/dev/null
for i in $(seq 1 1000); do printf 'key=%s\n' "$i" > "$TH/.config/many/f${i}.conf"; done
"$BIN" track add "$TH/.config/many" >/dev/null
echo "machine: $(uname -m) $(nproc 2>/dev/null || echo ?cpu) git $(git --version | awk '{print $3}')"
start=$SECONDS
"$BIN" snapshot -m perf
dur=$(( SECONDS - start ))
echo "snapshot 1000 files: ${dur}s (target <5s)"
[ "$dur" -lt 5 ]
echo "E2E OK"
