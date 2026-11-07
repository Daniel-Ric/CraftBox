package connectinfo

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var (
	realmRegex      = regexp.MustCompile(`^realm:(?P<Name>\S.*)$`)
	pcapRegex       = regexp.MustCompile(`^(?P<Filename>(?P<Name>.*)\.pcap2)(?:\?(?P<Args>.*))?$`)
	gatheringRegex  = regexp.MustCompile(`^gathering:(?P<Title>\S.*)$`)
	experienceRegex = regexp.MustCompile(`^experience:(?P<ID>\S.*)$`)
)

func regexGetParams(r *regexp.Regexp, s string) (params map[string]string) {
	match := r.FindStringSubmatch(s)
	params = make(map[string]string)
	for i, name := range r.SubexpNames() {
		if i > 0 && i < len(match) {
			params[name] = match[i]
		}
	}
	return params
}

func ValidateServerInput(server string) bool {
	parsed, err := parseConnectInfo(server)
	if err != nil {
		return false
	}
	if parsed.serverAddress == "" {
		return true
	}
	host, _, _ := net.SplitHostPort(parsed.serverAddress)
	if u, err := url.Parse(parsed.serverAddress); err == nil && u.Hostname() != "" {
		host = u.Hostname()
	}

	ip := net.ParseIP(host)
	if ip != nil {
		return true
	}

	ips, _ := net.LookupIP(host)
	return len(ips) > 0
}

func serverAddress(value string) (string, error) {
	explicit := strings.Contains(value, "://")
	address := value
	if !explicit {
		if ip := net.ParseIP(strings.Trim(value, "[]")); ip != nil {
			address = net.JoinHostPort(strings.Trim(value, "[]"), "19132")
		}
		address = "http://" + address
	}
	u, err := url.Parse(address)
	if err != nil {
		return "", fmt.Errorf("invalid server address: %w", err)
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" || strings.ContainsAny(u.Hostname(), " \t\r\n") {
		return "", fmt.Errorf("invalid server address %q", value)
	}
	port := u.Port()
	if port == "" {
		port = "19132"
		if explicit {
			port = "80"
			if u.Scheme == "https" {
				port = "443"
			}
		}
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return "", fmt.Errorf("invalid port %q", port)
	}
	u.Host = net.JoinHostPort(u.Hostname(), port)
	u.Path = ""
	if explicit {
		return u.String(), nil
	}
	return u.Host, nil
}

func NormalizeAddress(value string) (string, error) {
	address, err := serverAddress(strings.TrimSpace(value))
	if err != nil {
		return "", err
	}
	if !strings.Contains(address, "://") {
		address = "https://" + address
	}
	return address, nil
}
