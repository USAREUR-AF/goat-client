# Changelog

All notable changes to goat-client are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html)
with the `goat-client-` tag prefix described in [`CONTRIBUTING.md`](CONTRIBUTING.md#releases).

## [Unreleased]

First public release of goat-client.

- Cross-platform daemon (`goat-clientd`) and Fyne GUI / CLI (`goat-client`)
  for Linux, macOS and Windows, plus iOS and Android shells.
- Three operating modes on every platform: `wg-cp0-only`, `netbird-only`
  and `combined`, chosen at install and switchable at runtime.
- The inner mesh embeds the NetBird client from
  [USAREUR-AF/netbird](https://github.com/USAREUR-AF/netbird).
- A public build trusts the system roots only and pins no enrollment-bundle
  signing key. Deployments add their private CA root in
  `internal/ipc/grpc/trust/` and their signing keys in
  `internal/trustanchor/anchors.yaml` before building.
