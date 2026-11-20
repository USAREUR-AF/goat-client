#!/usr/bin/env bash
# Records the handshake shown on the landing page: the hermetic real-protocol
# test that stands up two WireGuard devices in one process (user-space network
# stacks, no TUN device, no root), signs and imports a bundle through the real
# import path, brings the tunnel up and pings across it.
#
#   site/captures/capture-handshake.sh    # writes site/captures/handshake.txt
#
# Only Go's own "go: downloading" lines are left out of the file.
set -euo pipefail
root="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$root"
go test -tags mobile_realprotocol -count=1 -v -run TestMobile_RealProtocolHandshake ./tests/integration/... 2>&1 \
  | grep -v '^go: downloading' > site/captures/handshake.txt
echo "wrote site/captures/handshake.txt"
