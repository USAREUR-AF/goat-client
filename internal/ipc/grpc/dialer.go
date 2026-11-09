// Package grpc carries the management-API gRPC dialer: TLS validation
// against the system roots plus any extra roots compiled in from the trust/
// directory, and the ServerName port strip needed because service
// certificates may carry IP SANs rather than DNS SANs.
//
// Adapted from the NetBird client's client/grpc/dialer.go. It drops that
// file's embedded Mozilla bundle fallback and custom dialer hook: the system
// pool is reliable on the desktop targets and the OS default dialer is used.
package grpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"embed"
	"fmt"
	"io/fs"
	"net"
	"path"
	"runtime"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// trustFS holds every file in trust/. A build that must reach endpoints signed
// by a private CA places that CA's root certificate there as a .pem file
// before compiling; the public tree ships the directory with a README only,
// so a public build trusts the system roots alone.
//
//go:embed trust
var trustFS embed.FS

// extraRootsPEM is the concatenation of every trust/*.pem file, in name order.
var extraRootsPEM = loadExtraRoots()

func loadExtraRoots() []byte {
	var out []byte
	_ = fs.WalkDir(trustFS, "trust", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || path.Ext(p) != ".pem" {
			return nil
		}
		b, err := trustFS.ReadFile(p)
		if err != nil {
			return err
		}
		out = append(out, b...)
		out = append(out, '\n')
		return nil
	})
	return out
}

// CreateConnection creates a gRPC client connection that trusts the system
// roots plus any compiled-in extra roots for TLS. Passing tlsEnabled=false drops to insecure
// credentials (test-only).
func CreateConnection(ctx context.Context, addr string, tlsEnabled bool, extraOpts ...grpc.DialOption) (*grpc.ClientConn, error) {
	transportOption := grpc.WithTransportCredentials(insecure.NewCredentials())
	if tlsEnabled && runtime.GOOS != "js" {
		certPool, err := x509.SystemCertPool()
		if err != nil || certPool == nil {
			certPool = x509.NewCertPool()
		}
		if len(extraRootsPEM) > 0 {
			if ok := certPool.AppendCertsFromPEM(extraRootsPEM); !ok {
				return nil, fmt.Errorf("trust/*.pem holds no parseable certificate")
			}
		}

		// gRPC passes "host:port" as the server name, which fails an IP SAN
		// match because "203.0.113.1:443" does not parse as an IP. Strip the
		// port.
		serverName := addr
		if h, _, splitErr := net.SplitHostPort(addr); splitErr == nil {
			serverName = h
		}
		transportOption = grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			RootCAs:    certPool,
			ServerName: serverName,
		}))
	}

	connCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	opts := []grpc.DialOption{
		transportOption,
		grpc.WithBlock(),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:    30 * time.Second,
			Timeout: 10 * time.Second,
		}),
	}
	opts = append(opts, extraOpts...)

	conn, err := grpc.DialContext(connCtx, addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("dial context: %w", err)
	}
	return conn, nil
}

// ExtraRootsPEM returns the compiled-in extra root certificates (empty in a
// public build). The inner mesh hands them to the NetBird client's dialer
// (grpc.ExtraRootCAsPEM in USAREUR-AF/netbird), which embeds none of its own.
func ExtraRootsPEM() []byte { return extraRootsPEM }
