package app

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// A Host is the loopback HTTP server Run puts behind the webview.
//
// Listen is the testable half of the host: it does not need a display or
// `-tags desktop`. The caller must Shutdown.
type Host struct {
	srv      *http.Server
	ln       net.Listener
	url      string
	hostport string
}

// Listen serves opts.Handler on a loopback address and returns the bound
// host. Run calls this before opening the window.
func Listen(opts Options) (*Host, error) {
	if err := opts.fill(); err != nil {
		return nil, err
	}
	bind, err := loopbackBind(opts.Addr)
	if err != nil {
		return nil, err
	}
	ln, err := net.Listen("tcp", bind)
	if err != nil {
		return nil, fmt.Errorf("app: listen: %w", err)
	}
	tcp, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		_ = ln.Close()
		return nil, fmt.Errorf("app: listen: not a TCP address: %s", ln.Addr())
	}
	hostport := net.JoinHostPort(tcp.IP.String(), strconv.Itoa(tcp.Port))
	h := &Host{
		ln:       ln,
		url:      "http://" + hostport,
		hostport: hostport,
		srv: &http.Server{
			Handler:           loopbackHandler(opts.Handler, hostport),
			ReadHeaderTimeout: 10 * time.Second,
			BaseContext: func(net.Listener) context.Context {
				return context.Background()
			},
		},
	}
	go func() {
		_ = h.srv.Serve(ln)
	}()
	return h, nil
}

// URL is the origin the webview should navigate to, with no path.
func (h *Host) URL() string { return h.url }

// Hostport is the Host header value that will be accepted, host:port.
func (h *Host) Hostport() string { return h.hostport }

// Shutdown closes the listener and waits for in-flight requests, the same
// contract as http.Server.Shutdown. An SSE stream will be cut.
func (h *Host) Shutdown(ctx context.Context) error {
	if h == nil || h.srv == nil {
		return nil
	}
	err := h.srv.Shutdown(ctx)
	_ = h.ln.Close()
	return err
}

// loopbackBind accepts only loopback addresses. Empty means 127.0.0.1:0.
// A host of "" / 0.0.0.0 / :: is the thing we are here to prevent.
func loopbackBind(addr string) (string, error) {
	if addr == "" {
		return "127.0.0.1:0", nil
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", fmt.Errorf("app: bind address %q: %w", addr, err)
	}
	if host == "" || host == "0.0.0.0" || host == "::" || host == "[::]" {
		return "", fmt.Errorf("app: bind address must be loopback, not %q", addr)
	}
	if host == "localhost" {
		return net.JoinHostPort("127.0.0.1", port), nil
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return "", fmt.Errorf("app: bind address %q is not an IP", addr)
	}
	if !ip.IsLoopback() {
		return "", fmt.Errorf("app: bind address must be loopback, not %q", addr)
	}
	if v4 := ip.To4(); v4 != nil {
		return net.JoinHostPort(v4.String(), port), nil
	}
	return net.JoinHostPort(ip.String(), port), nil
}

// loopbackHandler refuses requests that did not come from the bound
// loopback origin. The Host check is DNS-rebinding; the RemoteAddr check
// is a second belt for anything that spoofed Host but not the TCP peer.
func loopbackHandler(next http.Handler, hostport string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !hostAllowed(r.Host, hostport) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if !remoteLoopback(r.RemoteAddr) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func hostAllowed(got, want string) bool {
	return canonicalHost(got) == canonicalHost(want)
}

func canonicalHost(h string) string {
	h = strings.TrimSpace(h)
	h = strings.TrimPrefix(h, "[")
	if i := strings.LastIndex(h, "]"); i >= 0 {
		port := ""
		if i+1 < len(h) && h[i+1] == ':' {
			port = h[i+1:]
		}
		h = h[1:i] + port
	}
	host, port, err := net.SplitHostPort(h)
	if err != nil {
		return strings.ToLower(h)
	}
	return strings.ToLower(host) + ":" + port
}

func remoteLoopback(remote string) bool {
	host, _, err := net.SplitHostPort(remote)
	if err != nil {
		host = remote
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
