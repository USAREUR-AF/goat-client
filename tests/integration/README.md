# Integration tests — goat-client

End-to-end tests that build `goat-clientd` and drive it through its IPC
surface (the `Client` interface in [`internal/ipc/types.go`](../../internal/ipc/types.go)).
Two tiers, selected by Go build tags.

## Tier A — `integration` build tag (hermetic)

The test process mints a throwaway ECDSA P-256 trust root, builds and
signs a CBOR enrollment bundle with it, builds `cmd/goat-clientd`, spawns
the daemon on a temporary socket, and exercises over IPC:

- `GetStatus` before import reports no bundle
- `ImportBundle` accepts a valid bundle and rejects an empty or
  foreign-signed one
- `Connect` before import fails with a bundle error
- `GetDiagnostics` returns a non-empty log tail
- imported state persists across a daemon restart

`Connect` after import is not exercised here: bringing up the wg-cp0
tunnel needs `CAP_NET_ADMIN` and a TUN device. That is Tier B.

```bash
go test -tags integration -count=1 -v ./tests/integration/...
```

Runs in under 30 seconds and needs no environment variables.

## Tier B — `realprotocol` build tag (live endpoint)

Imports a real signed bundle, brings the wg-cp0 tunnel up against the
first endpoint in the bundle's `KnownEndpoints`, waits for a WireGuard
handshake, then TCP-connects to a target address through the tunnel to
confirm the data plane. The test skips unless the three required
variables are set.

| Variable | Required | Default | Purpose |
|---|---|---|---|
| `GOAT_LAB_BUNDLE_PATH` | yes | | CBOR enrollment bundle for a wg-cp0 endpoint you control |
| `GOAT_LAB_TRUST_ROOTS_PATH` | yes | | PEM file with the ECDSA P-256 public key that signed the bundle |
| `GOAT_LAB_TARGET_IP` | yes | | mesh address of a peer reachable through the tunnel |
| `GOAT_LAB_PROBE_PORT` | no | `80` | TCP port probed on the target |
| `GOAT_LAB_TIMEOUT_SEC` | no | `30` | handshake deadline per attempt (one retry after 60s) |
| `GOAT_LAB_PROBE_TIMEOUT_SEC` | no | `5` | TCP probe timeout |

```bash
export GOAT_LAB_BUNDLE_PATH=/path/to/test.bundle.cbor
export GOAT_LAB_TRUST_ROOTS_PATH=/path/to/trust-roots.pem
export GOAT_LAB_TARGET_IP=100.127.0.1
go test -tags realprotocol -count=1 -v ./tests/integration/...
```

The test needs `CAP_NET_ADMIN` and a TUN device. The endpoint must already list the bundle's `CPDevicePubkey` as an
allowed peer and route the mesh subnet to the target. Each failure is
tagged `[phase=import|connect|handshake|probe]`:

| Phase | Usual cause |
|---|---|
| `import` | bundle expired, or signed by a key other than the trust root |
| `connect` | daemon or IPC setup failure on the local machine |
| `handshake` | endpoint does not know the client key, or UDP egress is blocked |
| `probe` | tunnel is up but the target is not listening or not routed |

## `mobile_realprotocol` build tag (hermetic)

Stands up two in-process wireguard-go devices on netstack TUNs, imports a
signed bundle through the real pipeline, drives the mobile engine and
asserts a handshake via an ICMP echo round-trip. No environment needed.

```bash
go test -tags mobile_realprotocol -count=1 -v -run=TestMobile ./tests/integration/...
```

The bundle wire format is defined in
[`internal/bundle/bundle.go`](../../internal/bundle/bundle.go).
