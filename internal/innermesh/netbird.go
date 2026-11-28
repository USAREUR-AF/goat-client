package innermesh

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	netbird "github.com/netbirdio/netbird/client/embed"
	nbgrpc "github.com/netbirdio/netbird/client/grpc"

	ipcgrpc "github.com/USAREUR-AF/goat-client/internal/ipc/grpc"
)

// The NetBird client trusts the goat offline-CA root when it dials the
// management and signal servers (macOS will not import the Ed25519 root).
func init() { nbgrpc.ExtraRootCAsPEM = ipcgrpc.ExtraRootsPEM() }

// Netbird is a Mesh implementation backed by netbird's
// `github.com/netbirdio/netbird/client/embed` public package — the
// upstream-maintained library surface designed for embedding netbird
// directly into Go programs without a separate daemon install.
//
// Mapping to the Mesh interface:
//
//	Configure(cfg) → build embed.Options + lazily defer client construction
//	                  to first Connect (embed.New does the netbird-config
//	                  resolution which we want to happen at Connect-time)
//	Connect(ctx)   → embed.New(opts) + client.Start(ctx)
//	Disconnect(ctx)→ client.Stop(ctx)
//	State()        → derived from internal lifecycle flags + last-error
//	Stats()        → reads peer.FullStatus via client.Status(), maps to Stats
//	Logs(tail)     → ring buffer fed from embed.Options.LogOutput
//	Close()        → release client + nil out (Mesh unusable after Close)
//
// Userspace-WG-only enforcement: embed.Options.NoUserspace stays
// false. The brief mandates this for the library posture; the
// netbird embed default already matches.
type Netbird struct {
	mu       sync.Mutex
	cfg      Config
	client   *netbird.Client
	state    State
	closed   bool
	upAt     time.Time
	logBuf   *ringWriter
	deviceID string
	// stateDir holds the netbird config (with this device's WireGuard key),
	// state file and cached network map, one subdirectory per management
	// URL. Empty keeps everything in memory, so every start is a new peer.
	stateDir string
}

// netbirdMaxLogs is the Netbird impl's log ring-buffer capacity in
// bytes. The buffer holds log lines fed from netbird's logrus
// output; we split by newline at Logs() read time. 64 KiB is large
// enough for ~1000 typical log lines.
const netbirdMaxLogs = 64 << 10

// NewNetbird returns a fresh Netbird mesh in StateClosed. DeviceID
// (often the machine's hostname) is the peer-name netbird uses to
// register with the management plane; pass a stable value across
// daemon restarts so reconnections preserve the same peer identity.
//
// Construction is side-effect-free — the netbird client is built
// lazily on first Connect from the Configure-stored profile, so
// Configure errors surface at Configure time rather than at New
// time.
func NewNetbird(deviceID string) *Netbird {
	return NewNetbirdWithStateDir(deviceID, DefaultStateDir())
}

// NewNetbirdWithStateDir is NewNetbird with an explicit state directory.
// An empty stateDir keeps the netbird config in memory.
func NewNetbirdWithStateDir(deviceID, stateDir string) *Netbird {
	return &Netbird{
		state:    StateClosed,
		logBuf:   newRingWriter(netbirdMaxLogs),
		deviceID: deviceID,
		stateDir: stateDir,
	}
}

// DefaultStateDir is the per-user directory for the inner mesh's netbird
// state: <user config dir>/goat-client/netbird. Empty when the platform
// reports no user config directory.
func DefaultStateDir() string {
	root, err := os.UserConfigDir()
	if err != nil || root == "" {
		return ""
	}
	return filepath.Join(root, "goat-client", "netbird")
}

// profileDir returns the state directory for one management URL. Keeping
// one directory per management means a device that joins a second goatnet
// gets a second WireGuard key instead of reusing the first one.
func profileDir(stateDir, managementURL string) string {
	sum := sha256.Sum256([]byte(managementURL))
	return filepath.Join(stateDir, hex.EncodeToString(sum[:8]))
}

// Configure stores the profile. Validates that ManagementURL +
// SetupKey are present (embed.New would reject otherwise). The
// netbird client is constructed lazily at Connect-time.
func (n *Netbird) Configure(cfg Config) error {
	if cfg.ManagementURL == "" {
		return errors.New("innermesh/netbird: Config.ManagementURL required")
	}
	if cfg.SetupKey == "" {
		return errors.New("innermesh/netbird: Config.SetupKey required")
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.closed {
		return errors.New("innermesh/netbird: closed")
	}
	n.cfg = cfg
	n.state = StateConfiguring
	return nil
}

// Connect builds the embed client from the stored profile and
// starts it. Blocks until Start returns (success or error).
func (n *Netbird) Connect(ctx context.Context) error {
	n.mu.Lock()
	if n.closed {
		n.mu.Unlock()
		return errors.New("innermesh/netbird: closed")
	}
	if n.cfg.ManagementURL == "" {
		n.mu.Unlock()
		return errors.New("innermesh/netbird: not configured")
	}
	cfg := n.cfg
	n.state = StateConfiguring
	n.mu.Unlock()

	psk := ""
	if len(cfg.PreSharedKey) > 0 {
		psk = string(cfg.PreSharedKey)
	}
	// Compose the wire-side DeviceName from the operator-assigned bundle
	// DeviceID (cfg.BundleDeviceID, populated by FromBundle from
	// b.DeviceID) and the device-reported deviceID (n.deviceID). The
	// composed string arrives server-side as peer.Meta.Hostname and
	// feeds the {hostname} substitution in the SetupKey's
	// AutoPeerNameTemplate (USAREUR-AF/netbird 970d9506a).
	deviceName := composeIdentity(cfg.BundleDeviceID, n.deviceID)
	opts := netbird.Options{
		DeviceName:    deviceName,
		SetupKey:      cfg.SetupKey,
		ManagementURL: cfg.ManagementURL,
		PreSharedKey:  psk,
		LogOutput:     n.logBuf,
		// NoUserspace stays false: the client requires userspace
		// WireGuard. netbird's embed default already matches.
	}
	// With a state directory the netbird config, and with it this device's
	// WireGuard key, survives a restart: the device stays the same peer, and
	// the client can start from its cached network map when management is
	// unreachable (the cache is written beside the state file).
	if n.stateDir != "" {
		dir := profileDir(n.stateDir, cfg.ManagementURL)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			n.mu.Lock()
			n.state = StateError
			n.mu.Unlock()
			return fmt.Errorf("innermesh/netbird: state directory: %w", err)
		}
		opts.ConfigPath = filepath.Join(dir, "config.json")
		opts.StatePath = filepath.Join(dir, "state.json")
	}
	client, err := netbird.New(opts)
	if err != nil {
		n.mu.Lock()
		n.state = StateError
		n.mu.Unlock()
		return fmt.Errorf("innermesh/netbird: build client: %w", err)
	}
	if err := client.Start(ctx); err != nil {
		n.mu.Lock()
		n.state = StateError
		n.mu.Unlock()
		return fmt.Errorf("innermesh/netbird: start: %w", err)
	}
	n.mu.Lock()
	n.client = client
	n.state = StateUp
	n.upAt = time.Now()
	n.mu.Unlock()
	return nil
}

// Disconnect stops the embed client. Idempotent — safe to call when
// the mesh is already down or never came up.
func (n *Netbird) Disconnect(ctx context.Context) error {
	n.mu.Lock()
	client := n.client
	n.client = nil
	n.state = StateClosed
	n.upAt = time.Time{}
	n.mu.Unlock()
	if client == nil {
		return nil
	}
	return client.Stop(ctx)
}

// State returns the current lifecycle state. Safe from any goroutine.
func (n *Netbird) State() State {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.state
}

// Stats reads netbird's FullStatus and maps to our Stats. Aggregates
// per-peer counters across all connected peers; per-peer detail
// surfaces via GetInnerMeshDiagnostics's PeerStats once that wiring
// lands (M3).
func (n *Netbird) Stats() (Stats, error) {
	n.mu.Lock()
	client := n.client
	n.mu.Unlock()
	if client == nil {
		return Stats{}, nil
	}
	fs, err := client.Status()
	if err != nil {
		return Stats{}, fmt.Errorf("innermesh/netbird: client.Status: %w", err)
	}
	out := Stats{PeerCount: len(fs.Peers)}
	for _, p := range fs.Peers {
		out.BytesIn += uint64(p.BytesRx)
		out.BytesOut += uint64(p.BytesTx)
		if p.LastWireguardHandshake.After(out.LastHandshake) {
			out.LastHandshake = p.LastWireguardHandshake
		}
	}
	return out, nil
}

// Logs returns the most recent tail log lines from the ring buffer
// fed by netbird's logrus output.
func (n *Netbird) Logs(tail int) []string {
	return n.logBuf.Lines(tail)
}

// Close releases the client + ring buffer. Idempotent.
func (n *Netbird) Close() error {
	n.mu.Lock()
	client := n.client
	n.client = nil
	n.closed = true
	n.state = StateClosed
	n.mu.Unlock()
	if client != nil {
		_ = client.Stop(context.Background())
	}
	return nil
}

// ringWriter is a bounded byte-buffer io.Writer. netbird's embed
// package writes log lines through it (LogOutput field of Options);
// Logs(tail) reads the buffer split by newline.
type ringWriter struct {
	mu  sync.Mutex
	buf bytes.Buffer
	cap int
}

func newRingWriter(capBytes int) *ringWriter {
	if capBytes <= 0 {
		capBytes = 64 << 10
	}
	return &ringWriter{cap: capBytes}
}

// Write implements io.Writer. Trims the head when capacity is
// exceeded. Always returns len(p), nil — log writes must not fail.
func (r *ringWriter) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.buf.Write(p)
	if r.buf.Len() > r.cap {
		// Trim to capacity by skipping the head; align to the next
		// newline so we don't return a half-line on the next Lines call.
		excess := r.buf.Len() - r.cap
		bs := r.buf.Bytes()
		nl := bytes.IndexByte(bs[excess:], '\n')
		if nl >= 0 {
			drop := excess + nl + 1
			r.buf.Next(drop)
		} else {
			r.buf.Next(excess)
		}
	}
	return len(p), nil
}

// Lines returns the most recent tail lines. tail <= 0 returns all.
func (r *ringWriter) Lines(tail int) []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.buf.Len() == 0 {
		return nil
	}
	all := splitLines(r.buf.String())
	if tail <= 0 || tail >= len(all) {
		return all
	}
	return all[len(all)-tail:]
}

func splitLines(s string) []string {
	out := []string{}
	for len(s) > 0 {
		i := indexNewline(s)
		if i < 0 {
			if s != "" {
				out = append(out, s)
			}
			return out
		}
		if i > 0 {
			out = append(out, s[:i])
		}
		s = s[i+1:]
	}
	return out
}

func indexNewline(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			return i
		}
	}
	return -1
}

// compile-time assertions.
var (
	_ Mesh      = (*Netbird)(nil)
	_ io.Writer = (*ringWriter)(nil)
)
