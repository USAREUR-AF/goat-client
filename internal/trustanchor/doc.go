// Package trustanchor pins the ECDSA P-256 public keys that goat-client
// accepts as enrollment-bundle signers.
//
// The trust anchor is not user-configurable. The active set is compiled into the binary at
// build time from anchors.yaml so a tampered config file cannot persuade
// the daemon to accept a malicious offline-CA root.
//
// Multiple anchors may be active simultaneously during CA-rotation
// windows. Verify tries each anchor whose
// [valid_from, valid_until] window contains the current wall-clock and
// returns the first match. Stale binaries whose anchors have all expired
// fail closed with [ErrNoActiveAnchors] — the daemon surfaces a "please
// update goat-client" prompt rather than accepting an unverified bundle.
//
// The public tree pins no key. A deployment lists its CA public keys in
// anchors.yaml and regenerates embedded.go with cmd/anchorgen before
// building.
package trustanchor

//go:generate go run ../../cmd/anchorgen -in anchors.yaml -out embedded.go
