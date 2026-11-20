# The landing page

A single static page, published to GitHub Pages by `.github/workflows/pages.yml`.
No framework, no external fonts, scripts or requests.

```bash
python3 site/build.py                                  # copy the figure in, then check
python3 -m http.server 8000 --directory site           # look at it on http://localhost:8000/
```

## What each file is

| File | What it is |
|---|---|
| `index.html` | the page: markup, CSS, and the script that replays the captures and copies the command |
| `assets/icon.svg` | the tab icon |
| `assets/goat-client-*.svg` | copied in by `build.py` from `docs/figures/`; not kept twice |
| `captures/capture-cli.sh` | records `captures/cli-session.txt` |
| `captures/cli-session.txt` | the real daemon and CLI, run unprivileged on a laptop with a throwaway signing key: a tampered bundle refused, a signed one imported, then `getmode` and `status` |
| `captures/capture-handshake.sh` | records `captures/handshake.txt` |
| `captures/handshake.txt` | the hermetic real-protocol test (`tests/integration`, tag `mobile_realprotocol`): two WireGuard devices in one process on user-space network stacks, a bundle signed and imported through the real import path, the tunnel up, a ping across it |
| `build.py` | the check the pages workflow runs |

## The first screen

The terminal replays `captures/cli-session.txt`, then `captures/handshake.txt`,
line by line as recorded; the status light and the mode row follow what the lines
say. With reduced motion it shows the whole session at once. Unprivileged, the
daemon cannot create a TUN device, so the first recording stops before `connect`
(its import says so in its last line); the handshake comes from the test, which
needs no TUN device and no root.

Both recordings use made-up names and documentation addresses: the bundle is minted
by `cmd/smoke-mint` (device `smoke-mobile-01`, site `smoke-lab`, endpoint
`192.0.2.10:51820`), and the test uses its own loopback and test addresses. Run the
two scripts again to record new ones; `build.py` then says which numbers to change.

## Where each number comes from

`build.py` measures each one and fails when the page says something else. The page
marks each with `data-n`.

| Page says | `data-n` | Measured from |
|---|---|---|
| 556 B signed bundle | `bundle-bytes` | the `# bundle: N bytes` line of `captures/cli-session.txt` (`wc -c` of the minted bundle) |
| 0.50 s import, tunnel up, handshake | `handshake-s` | `--- PASS: TestMobile_RealProtocolHandshake (0.50s)` in `captures/handshake.txt` |
| 6 desktop builds | `desktop-targets` | rows of the `gui` job's matrix in `.github/workflows/release.yml` |
| 3 modes | `modes` | the `Modes:` line of the usage text in `cmd/goat-client/main.go` |

`build.py` also fails if the code shown faintly behind the first screen is no
longer in `internal/bundle/trust.go`.
