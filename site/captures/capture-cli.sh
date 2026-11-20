#!/usr/bin/env bash
# Records the terminal session shown on the landing page: the real daemon and
# CLI, run unprivileged on a laptop, with a throwaway signing key made here.
#
#   site/captures/capture-cli.sh          # writes site/captures/cli-session.txt
#
# Needs Go and openssl. Nothing leaves the machine; the keys and the bundle are
# made in a temporary directory and deleted afterwards. Unprivileged, the
# daemon cannot create a TUN device, so the session stops before `connect`
# (the import says so in its last line);
# the handshake itself is shown by the mobile real-protocol test
# (capture-handshake.sh).
set -euo pipefail
root="$(cd "$(dirname "$0")/../.." && pwd)"
out="$root/site/captures/cli-session.txt"
work="$(mktemp -d /tmp/gc-capture.XXXXXX)"
trap 'pkill -f "$work/bin/goat-clientd" 2>/dev/null || true; rm -rf "$work"' EXIT

mkdir -p "$work/bin"
(cd "$root" && CGO_ENABLED=0 go build -o "$work/bin/goat-clientd" ./cmd/goat-clientd)
(cd "$root" && go build -o "$work/bin/goat-client" ./cmd/goat-client)
(cd "$root" && go build -o "$work/bin/smoke-mint" ./cmd/smoke-mint)
export PATH="$work/bin:$PATH"
cd "$work"

# A throwaway offline CA (ECDSA P-256) and two WireGuard keys.
openssl ecparam -name prime256v1 -genkey -noout -out ca.key 2>/dev/null
openssl ec -in ca.key -pubout -out roots.pem 2>/dev/null
for n in client endpoint; do openssl genpkey -algorithm x25519 -out "$n.pem" 2>/dev/null; done
priv() { openssl pkey -in "$1" -outform DER | tail -c 32 | base64; }
pub() { openssl pkey -in "$1" -pubout -outform DER | tail -c 32 | base64; }
smoke-mint --ca-key ca.key --client-priv "$(priv client.pem)" --client-pub "$(pub client.pem)" \
  --endpoint-pub "$(pub endpoint.pem)" --endpoint-addr 192.0.2.10:51820 --out alice.cbor >/dev/null
python3 - <<'PY'
b = bytearray(open("alice.cbor", "rb").read())
i = b.find(b"smoke-lab"); b[i] = ord("S")
open("tampered.cbor", "wb").write(b)
PY

flags="--socket d.sock --trust-roots roots.pem --bundle bundle.cbor --config config.toml"
run() { echo "\$ $*"; "$@" 2>&1 || echo "[exit $?]"; }
{
  run goat-clientd $flags --import-bundle tampered.cbor
  run goat-clientd $flags --import-bundle alice.cbor
} > import.txt

goat-clientd $flags --mode combined --auto-connect=false > daemon.log 2>&1 &
for _ in $(seq 50); do [ -S d.sock ] && break; sleep 0.1; done
{
  run goat-client --daemon-addr d.sock getmode
  run goat-client --daemon-addr d.sock status
} > session.txt

{ echo "# bundle: $(wc -c < alice.cbor | tr -d ' ') bytes"; cat import.txt session.txt; } > "$out"
echo "wrote $out"
