package innermesh

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/USAREUR-AF/goat-client/internal/innermesh/fakemgmt"
	"github.com/USAREUR-AF/goat-client/internal/innermesh/fakesignal"
)

const identityTestSetupKey = "11111111-1111-1111-1111-111111111111"

func startFakes(t *testing.T) (*fakesignal.Server, *fakemgmt.Server) {
	t.Helper()
	sig, err := fakesignal.Listen(t)
	if err != nil {
		t.Fatalf("fakesignal.Listen: %v", err)
	}
	mgmt, err := fakemgmt.Listen(t, fakemgmt.WithSignalURI(sig.Addr()))
	if err != nil {
		t.Fatalf("fakemgmt.Listen: %v", err)
	}
	return sig, mgmt
}

// connectOnce configures and connects a Netbird on stateDir, then disconnects it.
func connectOnce(t *testing.T, stateDir, managementURL string, budget time.Duration) *Netbird {
	t.Helper()
	nb := NewNetbirdWithStateDir("goat-identity-peer", stateDir)
	t.Cleanup(func() { _ = nb.Close() })
	if err := nb.Configure(Config{ManagementURL: managementURL, SetupKey: identityTestSetupKey}); err != nil {
		t.Fatalf("Configure: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), budget)
	defer cancel()
	if err := nb.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v\nlogs:\n%s", err, strings.Join(nb.Logs(80), "\n"))
	}
	if got := nb.State(); got != StateUp {
		t.Fatalf("State after Connect: got %v, want %v", got, StateUp)
	}
	return nb
}

// TestNetbird_RestartKeepsPeerKey: a second start on the same state directory
// logs in with the same WireGuard key, so the device stays one peer; another
// management URL gets its own key.
func TestNetbird_RestartKeepsPeerKey(t *testing.T) {
	if raceDetectorEnabled {
		t.Skip("upstream netbird embed.Client connect/stop race; tracked separately")
	}
	_, mgmt := startFakes(t)
	stateDir := t.TempDir()
	url := "http://" + mgmt.Addr()

	first := connectOnce(t, stateDir, url, 30*time.Second)
	key1 := mgmt.LastPeerKey()
	if err := first.Disconnect(context.Background()); err != nil {
		t.Fatalf("Disconnect: %v", err)
	}
	if key1 == "" {
		t.Fatal("management saw no login")
	}

	second := connectOnce(t, stateDir, url, 30*time.Second)
	key2 := mgmt.LastPeerKey()
	_ = second.Disconnect(context.Background())
	if key2 != key1 {
		t.Fatalf("restart changed the peer key: %s then %s", key1, key2)
	}

	cfgPath := filepath.Join(profileDir(stateDir, url), "config.json")
	fi, err := os.Stat(cfgPath)
	if err != nil {
		t.Fatalf("netbird config not persisted at %s: %v", cfgPath, err)
	}
	if mode := fi.Mode().Perm(); mode&0o077 != 0 && os.PathSeparator == '/' {
		t.Errorf("netbird config %s is readable by others: %v", cfgPath, mode)
	}

	_, other := startFakes(t)
	third := connectOnce(t, stateDir, "http://"+other.Addr(), 30*time.Second)
	_ = third.Disconnect(context.Background())
	if other.LastPeerKey() == key1 {
		t.Fatal("a second management URL reused the first one's WireGuard key")
	}
}

// TestNetbird_RestartWithManagementDownStartsFromCache: after one connection
// the client has cached its network map beside its state file. With
// management and signal stopped, a restart still comes up, from the cache.
func TestNetbird_RestartWithManagementDownStartsFromCache(t *testing.T) {
	if raceDetectorEnabled {
		t.Skip("upstream netbird embed.Client connect/stop race; tracked separately")
	}
	sig, mgmt := startFakes(t)
	stateDir := t.TempDir()
	url := "http://" + mgmt.Addr()

	first := connectOnce(t, stateDir, url, 30*time.Second)
	cachePath := filepath.Join(profileDir(stateDir, url), "state.goat-map.json")
	deadline := time.Now().Add(20 * time.Second)
	for {
		if _, err := os.Stat(cachePath); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("no cached network map at %s\nlogs:\n%s", cachePath, strings.Join(first.Logs(80), "\n"))
		}
		time.Sleep(200 * time.Millisecond)
	}
	if err := first.Disconnect(context.Background()); err != nil {
		t.Fatalf("Disconnect: %v", err)
	}

	mgmt.Stop()
	sig.Stop()

	// The embed client gives up on management after its dial timeouts
	// (about 30 s each for the login and for the engine's connect), then
	// starts from the cache.
	started := time.Now()
	second := connectOnce(t, stateDir, url, 150*time.Second)
	logs := strings.Join(second.Logs(400), "\n")
	_ = second.Disconnect(context.Background())
	if !strings.Contains(logs, "Started from a cached network map") {
		t.Fatalf("the restart did not start from the cached network map\nlogs:\n%s", logs)
	}
	t.Logf("restart with management and signal stopped came up from the cache in %s", time.Since(started).Round(time.Second))
}
