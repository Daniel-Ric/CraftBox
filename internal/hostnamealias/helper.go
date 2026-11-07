package hostnamealias

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

const helperFlag = "--craftbox-hosts-helper"

type helperMessage struct {
	Token     string   `json:"token,omitempty"`
	Hostname  string   `json:"hostname,omitempty"`
	Addresses []net.IP `json:"addresses,omitempty"`
	Ready     bool     `json:"ready,omitempty"`
	Stop      bool     `json:"stop,omitempty"`
	Closed    bool     `json:"closed,omitempty"`
	Error     string   `json:"error,omitempty"`
}

func readMessage(reader *bufio.Reader) (helperMessage, error) {
	var m helperMessage
	line, err := reader.ReadSlice('\n')
	if err != nil {
		return m, err
	}
	err = json.Unmarshal(line, &m)
	return m, err
}
func writeMessage(conn net.Conn, m helperMessage) error { return json.NewEncoder(conn).Encode(m) }

// HandleHelperArgs is called before application/account/UI initialization.
func HandleHelperArgs() (bool, error) {
	if len(os.Args) < 2 || os.Args[1] != helperFlag {
		return false, nil
	}
	if len(os.Args) != 4 {
		return true, fmt.Errorf("invalid hostname helper arguments")
	}
	address, token := os.Args[2], os.Args[3]
	host, _, err := net.SplitHostPort(address)
	if err != nil || host != "127.0.0.1" || len(token) != 64 {
		return true, fmt.Errorf("invalid local helper endpoint")
	}
	if _, err = hex.DecodeString(token); err != nil {
		return true, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c, err := (&net.Dialer{}).DialContext(ctx, "tcp4", address)
	if err != nil {
		return true, err
	}
	defer c.Close()
	return true, serveHelper(c, token, func(name string) (*Lease, error) { return acquireLocal(ctx, name) })
}
func serveHelper(conn net.Conn, token string, acquire func(string) (*Lease, error)) error {
	conn.SetDeadline(time.Now().Add(30 * time.Second))
	reader := bufio.NewReaderSize(conn, 4096)
	if err := writeMessage(conn, helperMessage{Token: token}); err != nil {
		return err
	}
	request, err := readMessage(reader)
	if err != nil {
		return err
	}
	name, err := Name(request.Hostname)
	if err != nil || name != request.Hostname {
		return fmt.Errorf("invalid helper destination")
	}
	lease, err := acquire(name)
	if err != nil {
		writeMessage(conn, helperMessage{Error: err.Error()})
		return err
	}
	defer lease.Close()
	if err = writeMessage(conn, helperMessage{Ready: true, Hostname: name, Addresses: lease.Addresses}); err != nil {
		return err
	}
	conn.SetDeadline(time.Time{})
	request, err = readMessage(reader)
	// EOF also closes the lease: parent exit/crash cannot retain a live override.
	closeErr := lease.Close()
	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	response := helperMessage{Closed: true}
	if closeErr != nil {
		response.Error = closeErr.Error()
	}
	writeMessage(conn, response)
	if closeErr != nil {
		return closeErr
	}
	if err != nil {
		return err
	}
	if !request.Stop {
		return fmt.Errorf("invalid helper stop request")
	}
	return nil
}
func acquireRemote(ctx context.Context, name string, launch func(string, string) error) (*Lease, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	l, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		return nil, err
	}
	defer l.Close()
	nonce := make([]byte, 32)
	if _, err = rand.Read(nonce); err != nil {
		return nil, err
	}
	token := hex.EncodeToString(nonce)
	if err = launch(l.Addr().String(), token); err != nil {
		return nil, fmt.Errorf("Windows permission for destination hostname was not granted. Approve the UAC prompt or switch off Destination hostname in Preferences > Connection: %w", err)
	}
	deadline := time.Now().Add(30 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	l.SetDeadline(deadline)
	stopCancel := context.AfterFunc(ctx, func() { l.Close() })
	defer stopCancel()
	for {
		conn, e := l.Accept()
		if e != nil {
			return nil, fmt.Errorf("hostname helper could not connect: %w", e)
		}
		conn.SetDeadline(deadline)
		reader := bufio.NewReaderSize(conn, 4096)
		hello, e := readMessage(reader)
		if e != nil || subtle.ConstantTimeCompare([]byte(hello.Token), []byte(token)) != 1 {
			conn.Close()
			continue
		}
		cancelConn := context.AfterFunc(ctx, func() { conn.Close() })
		if e = writeMessage(conn, helperMessage{Hostname: name}); e != nil {
			cancelConn()
			conn.Close()
			return nil, e
		}
		ready, e := readMessage(reader)
		cancelConn()
		if e != nil {
			conn.Close()
			return nil, e
		}
		if ready.Error != "" {
			conn.Close()
			return nil, fmt.Errorf("destination hostname could not be enabled: %s", ready.Error)
		}
		if !ready.Ready || ready.Hostname != name || len(ready.Addresses) == 0 {
			conn.Close()
			return nil, fmt.Errorf("invalid hostname helper response")
		}
		for _, ip := range ready.Addresses {
			if ip == nil || ip.IsLoopback() || ip.IsUnspecified() {
				conn.Close()
				return nil, fmt.Errorf("invalid upstream IP from hostname helper")
			}
		}
		conn.SetDeadline(time.Time{})
		lease := &Lease{Hostname: name, Addresses: ready.Addresses, Client: PinnedClient(name, ready.Addresses)}
		lease.closeRemote = func() error {
			defer conn.Close()
			conn.SetDeadline(time.Now().Add(10 * time.Second))
			if e := writeMessage(conn, helperMessage{Stop: true}); e != nil {
				return e
			}
			closed, e := readMessage(reader)
			if e != nil {
				return e
			}
			if !closed.Closed || closed.Error != "" {
				return fmt.Errorf("hostname cleanup failed: %s", strings.TrimSpace(closed.Error))
			}
			return nil
		}
		return lease, nil
	}
}
