// hostname-prototype exercises logical host preservation without changing OS DNS.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	nn "github.com/df-mc/go-nethernet"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type redirectTrace struct{ Redirects atomic.Int64 }

func aliasClient(alias, physical string) (*http.Client, *redirectTrace, error) {
	u, err := url.Parse(alias)
	if err != nil || u.Scheme != "http" || u.Hostname() == "" || u.Port() == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return nil, nil, fmt.Errorf("expected a plain HTTP hostname and explicit port")
	}
	host, _, err := net.SplitHostPort(physical)
	if err != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() {
		return nil, nil, fmt.Errorf("physical target must be a literal loopback address")
	}
	trace := &redirectTrace{}
	transport := &http.Transport{Proxy: nil, DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
		if !strings.EqualFold(address, u.Host) {
			return nil, fmt.Errorf("prototype refuses another endpoint: %s", address)
		}
		trace.Redirects.Add(1)
		return (&net.Dialer{}).DialContext(ctx, network, physical)
	}}
	return &http.Client{Transport: transport, Timeout: 10 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
		return fmt.Errorf("HTTP redirects disabled in prototype")
	}}, trace, nil
}

type result struct {
	LogicalAddress     string `json:"logicalAddress"`
	PhysicalProxy      string `json:"physicalProxy"`
	LocalDestination   string `json:"localDestination"`
	LoginServerAddress string `json:"loginServerAddress"`
	HostnamePreserved  bool   `json:"hostnamePreservedInLogin"`
	TextRoundTrip      bool   `json:"textPacketDelivered"`
	Redirects          int64  `json:"httpConnectionsRedirected"`
	RelayedMessages    int64  `json:"relayedMessages"`
	FlarialHUDVerified bool   `json:"flarialHUDVerified"`
	OSDNSChanged       bool   `json:"osDNSChanged"`
	Scope              string `json:"scope"`
}

func probe(ctx context.Context, hostname string) (result, error) {
	r := result{Scope: "Native Go Minecraft client -> hostname transport redirect -> local NetherNet relay -> local Minecraft server. No Xbox login, external destination, OS DNS or Flarial changes. Actual Minecraft/Flarial HUD still requires a manual join."}
	if hostname == "" || strings.ContainsAny(hostname, "/:@?# \\ \t\r\n") || net.ParseIP(hostname) != nil {
		return r, fmt.Errorf("expected a DNS hostname")
	}
	network := minecraft.DirectNetherNet{DisableLANDiscovery: true}
	data := make(chan string, 1)
	cfg := minecraft.ListenConfig{AuthenticationDisabled: true, StatusProvider: minecraft.NewStatusProvider("CraftBox hostname probe", "Local test"), OnClientData: func(c *minecraft.Conn) {
		select {
		case data <- c.ClientData().ServerAddress:
		default:
		}
	}}
	destination, err := cfg.ListenNetwork(network, "127.0.0.1:0")
	if err != nil {
		return r, err
	}
	defer destination.Close()
	r.LocalDestination = destination.Addr().String()
	relay, err := network.Listen("127.0.0.1:0")
	if err != nil {
		return r, err
	}
	defer relay.Close()
	r.PhysicalProxy = relay.Addr().String()
	r.LogicalAddress = "http://" + net.JoinHostPort(hostname, portOf(r.PhysicalProxy))
	clientHTTP, trace, err := aliasClient(r.LogicalAddress, r.PhysicalProxy)
	if err != nil {
		return r, err
	}
	defer clientHTTP.CloseIdleConnections()
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		select {
		case <-ctx.Done():
			destination.Close()
			relay.Close()
		case <-stop:
		}
	}()
	var count atomic.Int64
	relayErrors := make(chan error, 1)
	go func() {
		incoming, e := relay.Accept()
		if e != nil {
			relayErrors <- e
			return
		}
		defer incoming.Close()
		upstream, e := network.DialContext(ctx, "http://"+r.LocalDestination)
		if e != nil {
			relayErrors <- e
			return
		}
		defer upstream.Close()
		a, aOK := incoming.(*nn.Conn)
		b, bOK := upstream.(*nn.Conn)
		if !aOK || !bOK {
			relayErrors <- fmt.Errorf("unexpected transport connection types")
			return
		}
		done := make(chan error, 4)
		for _, reliability := range []nn.MessageReliability{nn.MessageReliabilityReliable, nn.MessageReliabilityUnreliable} {
			go relayMessages(a, b, reliability, &count, done)
			go relayMessages(b, a, reliability, &count, done)
		}
		select {
		case e = <-done:
			relayErrors <- e
		case <-ctx.Done():
		case <-stop:
		}
	}()
	serverDone := make(chan error, 1)
	const message = "CraftBox hostname prototype OK"
	go func() {
		c, e := destination.Accept()
		if e != nil {
			serverDone <- e
			return
		}
		defer c.Close()
		server := c.(*minecraft.Conn)
		e = server.StartGameContext(ctx, minecraft.GameData{WorldName: "Hostname probe", EntityUniqueID: 1, EntityRuntimeID: 1})
		if e == nil {
			e = server.WritePacket(&packet.Text{TextType: packet.TextTypeRaw, Message: message})
		}
		if e == nil {
			e = server.Flush()
		}
		serverDone <- e
		select {
		case <-stop:
		case <-ctx.Done():
		}
	}()
	client, err := (minecraft.Dialer{}).DialContextNetwork(ctx, minecraft.DirectNetherNet{HTTPClient: clientHTTP, DisableLANDiscovery: true}, r.LogicalAddress)
	if err != nil {
		select {
		case e := <-relayErrors:
			return r, fmt.Errorf("login: %w; relay: %v", err, e)
		default:
			return r, fmt.Errorf("login: %w", err)
		}
	}
	defer client.Close()
	select {
	case r.LoginServerAddress = <-data:
	case <-ctx.Done():
		return r, ctx.Err()
	}
	r.HostnamePreserved = strings.Contains(r.LoginServerAddress, hostname+":")
	select {
	case err = <-serverDone:
		if err != nil {
			return r, err
		}
	case <-ctx.Done():
		return r, ctx.Err()
	}
	if deadline, ok := ctx.Deadline(); ok {
		client.SetReadDeadline(deadline)
	}
	for {
		pk, _, e := client.ReadPacketWithTime()
		if e != nil {
			return r, e
		}
		if text, ok := pk.(*packet.Text); ok && text.Message == message {
			r.TextRoundTrip = true
			break
		}
	}
	r.Redirects = trace.Redirects.Load()
	r.RelayedMessages = count.Load()
	return r, nil
}

func relayMessages(source, target *nn.Conn, reliability nn.MessageReliability, count *atomic.Int64, done chan<- error) {
	for {
		data, err := source.Receive(reliability)
		if err != nil {
			done <- err
			return
		}
		if _, err = target.Send(data, reliability); err != nil {
			done <- err
			return
		}
		count.Add(1)
	}
}
func portOf(address string) string { _, port, _ := net.SplitHostPort(address); return port }

func main() {
	hostname := flag.String("hostname", "craftbox-probe.test", "Logical hostname (transport mapping is process-local)")
	report := flag.String("report", "tmp/hostname-prototype/report.json", "Report output")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	r, err := probe(ctx, *hostname)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	encoded, err := json.MarshalIndent(r, "", "  ")
	if err == nil {
		err = os.MkdirAll(filepath.Dir(*report), 0755)
	}
	if err == nil {
		err = os.WriteFile(*report, append(encoded, '\n'), 0644)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(string(encoded))
}
