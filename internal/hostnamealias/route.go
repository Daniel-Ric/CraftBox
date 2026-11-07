package hostnamealias

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// PinnedClient retains the URL hostname, Host header and TLS verification/SNI,
// but connects to the destination IPs captured before the hosts override.
func PinnedClient(host string, ips []net.IP) *http.Client {
	snapshot := make([]net.IP, len(ips))
	for i, ip := range ips {
		snapshot[i] = append(net.IP(nil), ip...)
	}
	transport := &http.Transport{ForceAttemptHTTP2: true, TLSHandshakeTimeout: 10 * time.Second, IdleConnTimeout: 90 * time.Second, DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
		target, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		d := net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}
		if !strings.EqualFold(strings.TrimSuffix(target, "."), host) {
			return d.DialContext(ctx, network, address)
		}
		var failures []error
		for _, ip := range snapshot {
			c, e := d.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
			if e == nil {
				return c, nil
			}
			failures = append(failures, e)
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
		}
		return nil, fmt.Errorf("original destination %s unavailable: %w", host, errors.Join(failures...))
	}}
	return &http.Client{Transport: transport}
}
