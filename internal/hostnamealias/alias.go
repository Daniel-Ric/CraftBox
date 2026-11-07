// Package hostnamealias manages temporary destination hostname redirection.
package hostnamealias

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const legacyMarker = " # CraftBox hostname-alias v1"
const marker = " # CraftBox hostname-redirect v2"

type Lease struct {
	Hostname            string
	Addresses           []net.IP
	Client              *http.Client
	closeRemote         func() error
	path                string
	original, installed []byte
	release             func()
	once                sync.Once
	err                 error
}

// Name preserves the destination hostname without adding an application suffix.
func Name(address string) (string, error) {
	if !strings.Contains(address, "://") {
		address = "http://" + address
	}
	u, e := url.Parse(address)
	if e != nil || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") || u.RawQuery != "" || u.Fragment != "" || u.Path != "" {
		return "", fmt.Errorf("hostname alias requires a direct server hostname")
	}
	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	if net.ParseIP(host) != nil || host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".craftbox.test") || len(host) > 253 || !validHost(host) {
		return "", fmt.Errorf("Destination hostname needs a valid server domain, such as play.example.com. IP addresses and local aliases are not supported. Enter a server domain or turn off Destination hostname in Preferences > Connection, then try again")
	}
	return host, nil
}
func validHost(host string) bool {
	if host == "" {
		return false
	}
	for _, label := range strings.Split(host, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, c := range label {
			if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
				return false
			}
		}
	}
	return true
}

// Acquire asks Windows for a separate elevated helper when required. The main
// application stays unelevated. Original IPs are captured before redirection.
func Acquire(ctx context.Context, address string) (*Lease, error) {
	name, err := Name(address)
	if err != nil {
		return nil, err
	}
	return acquirePlatform(ctx, name)
}

func acquireLocal(ctx context.Context, name string) (*Lease, error) {
	path, release, err := systemHosts()
	if err != nil {
		return nil, err
	}
	lease, err := prepare(ctx, path, name, net.DefaultResolver.LookupIPAddr)
	if err != nil {
		release()
		return nil, err
	}
	lease.release = release
	flushDNS()
	return lease, nil
}
func prepare(ctx context.Context, path, name string, lookup func(context.Context, string) ([]net.IPAddr, error)) (*Lease, error) {
	raw, err := readHosts(path)
	if err != nil {
		return nil, err
	}
	clean := stripManaged(raw)
	if err = replaceHosts(path, raw, clean); err != nil {
		return nil, err
	}
	flushDNS()
	resolved, err := lookup(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("could not resolve original destination %s: %w", name, err)
	}
	var ips []net.IP
	for _, address := range resolved {
		if address.IP != nil && !address.IP.IsLoopback() && !address.IP.IsUnspecified() {
			ips = append(ips, address.IP)
		}
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("original destination %s resolves only to local addresses; refusing a proxy loop", name)
	}
	lease, err := install(path, name)
	if err != nil {
		return nil, err
	}
	lease.Addresses = ips
	lease.Client = PinnedClient(name, ips)
	return lease, nil
}

func readHosts(path string) ([]byte, error) {
	f, e := os.Open(path)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	raw, e := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if e != nil {
		return nil, e
	}
	if len(raw) > 1<<20 || bytes.Contains(raw, []byte{0}) {
		return nil, fmt.Errorf("unsupported hosts file encoding or size")
	}
	return raw, nil
}
func managedLine(line string) bool {
	line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
	if strings.HasSuffix(line, legacyMarker) {
		fields := strings.Fields(strings.TrimSuffix(line, legacyMarker))
		return len(fields) == 2 && fields[0] == "127.0.0.1" && strings.HasSuffix(fields[1], ".craftbox.test") && validHost(fields[1])
	}
	if !strings.HasSuffix(line, marker) {
		return false
	}
	fields := strings.Fields(strings.TrimSuffix(line, marker))
	return len(fields) == 2 && fields[0] == "127.0.0.1" && validHost(fields[1])
}
func stripManaged(raw []byte) []byte {
	var out []byte
	for _, line := range strings.SplitAfter(string(raw), "\n") {
		if !managedLine(line) {
			out = append(out, line...)
		}
	}
	return out
}
func install(path, name string) (*Lease, error) {
	if actual, err := Name(name); err != nil || actual != name {
		return nil, fmt.Errorf("invalid local hostname")
	}
	raw, err := readHosts(path)
	if err != nil {
		return nil, err
	}
	clean := stripManaged(raw)
	for _, line := range strings.Split(string(clean), "\n") {
		fields := strings.Fields(strings.SplitN(strings.TrimPrefix(line, "\xef\xbb\xbf"), "#", 2)[0])
		if len(fields) < 2 {
			continue
		}
		for _, field := range fields[1:] {
			if strings.EqualFold(field, name) {
				return nil, fmt.Errorf("hosts already contains an unmanaged mapping for %s", name)
			}
		}
	}
	newline := "\n"
	if bytes.Contains(raw, []byte("\r\n")) {
		newline = "\r\n"
	}
	next := append([]byte{}, clean...)
	if len(next) > 0 && next[len(next)-1] != '\n' {
		next = append(next, newline...)
	}
	next = append(next, "127.0.0.1 "+name+marker+newline...)
	if err = replaceHosts(path, raw, next); err != nil {
		return nil, err
	}
	// Stale entries from an interrupted session are not restored on normal close.
	return &Lease{Hostname: name, path: path, original: clean, installed: next}, nil
}
func (l *Lease) Close() error {
	l.once.Do(func() {
		if l.Client != nil {
			l.Client.CloseIdleConnections()
		}
		if l.closeRemote != nil {
			l.err = l.closeRemote()
			return
		}
		if l.release != nil {
			defer l.release()
		}
		raw, err := readHosts(l.path)
		if err != nil {
			l.err = err
			return
		}
		next := stripManaged(raw)
		if bytes.Equal(raw, l.installed) {
			next = l.original
		}
		l.err = replaceHosts(l.path, raw, next)
		flushDNS()
	})
	return l.err
}
func replaceHosts(path string, expected, next []byte) error {
	if bytes.Equal(expected, next) {
		return nil
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".craftbox-hosts-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	keepRecovery := false
	defer func() {
		if !keepRecovery {
			os.Remove(tmp)
		}
	}()
	if _, err = f.Write(next); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	current, err := readHosts(path)
	if err != nil {
		return err
	}
	if !bytes.Equal(current, expected) {
		return fmt.Errorf("hosts file changed during update; please retry")
	}
	if err := atomicReplace(path, tmp); err != nil {
		keepRecovery = true
		return fmt.Errorf("%w; replacement recovery path: %s", err, tmp)
	}
	return nil
}
