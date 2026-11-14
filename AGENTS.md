# AGENTS.md

Instructions for AI coding agents, and a short version for anyone working here. [CONTRIBUTING.md](CONTRIBUTING.md) is the full process.

## What this repository is

goat-client is the cross-platform daemon and GUI that joins a device to a goatnet. It reads an enrollment bundle signed by an offline certificate authority and brings up and maintains the device's WireGuard tunnels, in one of three modes: `wg-cp0-only`, `netbird-only` (through the embedded NetBird client from [USAREUR-AF/netbird](https://github.com/USAREUR-AF/netbird)) and `combined`. It is written in Go, with a Fyne desktop GUI and gomobile builds for iOS and Android. Its design and decision records are in [goatcore](https://github.com/USAREUR-AF/goatcore).

## Layout

| Path | What |
|---|---|
| `cmd/goat-clientd/`, `cmd/goat-client/` | the daemon and the GUI |
| `cmd/` | also the bundle, anchor and smoke-test tools |
| `internal/` | `bundle`, `daemon`, `innermesh`, `ipc`, `mode`, `profile`, `reachability`, `trustanchor`, `tunnel`, `ui` |
| `mobile/` | the gomobile bindings and the iOS and Android shells |
| `packaging/` | per-platform packages |
| `tests/integration/` | integration tests, selected by build tag |
| `docs/` | documentation for this repository |

## Before you push

CI (`.github/workflows/ci.yml` and `integration.yml`) runs these, with the Go version from `go.mod`:

```bash
go vet ./...
go test -race -count=1 ./...
go build ./...
go test -tags integration -count=1 -v -timeout 2m ./tests/integration/...
```

`make smoke-modes` runs the three-mode smoke test in process. CI also runs `golangci-lint` with `.golangci.yaml`, and cross-compiles for every platform. The desktop GUI and the mobile builds need platform toolchains; `mobile.yml` shows the exact steps.

## Rules

1. **Commits.** Signed, and signed off (`git commit -s`, the DCO); the `Signed-off-by:` trailer matches the commit author. The organization's required check `verified-commits` refuses a pull request that holds an unverified commit.
2. **Pull request titles** follow Conventional Commits (`fix(ipc): …`), because the title becomes the squash commit on `main`. Allowed types: `feat`, `fix`, `chore`, `docs`, `refactor`, `test`, `perf`, `ci`, `build`, `revert`, `ui`.
3. **Link the goatcore document** by URL when a design decision drives a change.
4. **Vulnerabilities** go through [SECURITY.md](SECURITY.md), never a public issue.

The contribution guide for goat as a whole is [goatcore CONTRIBUTING.md](https://github.com/USAREUR-AF/goatcore/blob/main/CONTRIBUTING.md).
