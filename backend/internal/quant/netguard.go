package quant

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"syscall"
	"time"
)

var ErrEndpointNotAllowed = errors.New("endpoint not allowed")

// blockedIP reports addresses a user-supplied endpoint must not reach
// from the server: loopback, private ranges, link-local (incl. cloud
// metadata at 169.254.169.254), unspecified and multicast.
func blockedIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast() ||
		ip.IsInterfaceLocalMulticast()
}

// validateBaseURL checks a self-hosted endpoint before any request is
// made: http(s) only, no embedded credentials, and (unless allowPrivate)
// a hostname that doesn't resolve to a blocked address.
func validateBaseURL(ctx context.Context, raw string, allowPrivate bool) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("%w: base URL must be an http(s) URL", ErrEndpointNotAllowed)
	}
	if u.User != nil {
		return nil, fmt.Errorf("%w: credentials in the URL are not allowed", ErrEndpointNotAllowed)
	}
	if allowPrivate {
		return u, nil
	}
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", u.Hostname())
	if err != nil {
		return nil, fmt.Errorf("%w: cannot resolve %s", ErrEndpointNotAllowed, u.Hostname())
	}
	for _, ip := range ips {
		if blockedIP(ip) {
			return nil, fmt.Errorf("%w: %s resolves to a private or local address", ErrEndpointNotAllowed, u.Hostname())
		}
	}
	return u, nil
}

// guardedClient re-checks the address actually dialed, so a hostname that
// resolved to a public IP during validation can't be re-pointed at a
// private one (DNS rebinding) for the real connection. When an outbound
// HTTP proxy handles target, the dial goes to the proxy itself (often a
// private address), so only the up-front validateBaseURL check applies.
func guardedClient(target *url.URL, timeout time.Duration, allowPrivate bool) *http.Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	proxied := false
	if p, err := http.ProxyFromEnvironment(&http.Request{URL: target}); err == nil && p != nil {
		proxied = true
	}
	if !allowPrivate && !proxied {
		dialer.Control = func(_, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			if ip := net.ParseIP(host); ip != nil && blockedIP(ip) {
				return fmt.Errorf("%w: %s", ErrEndpointNotAllowed, host)
			}
			return nil
		}
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = dialer.DialContext
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		// A redirect could point somewhere validateBaseURL never saw.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}
