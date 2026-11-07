// SPDX-License-Identifier: MIT
// CraftBox regression: NetherNet uses DTLS/SCTP without RTP media.
//go:build !js

package webrtc

import (
	"context"
	"errors"
	"github.com/pion/dtls/v3"
	"github.com/pion/dtls/v3/pkg/crypto/selfsign"
	"net"
	"testing"
	"time"
)

func TestCraftBoxDataOnlyDTLSWithoutSRTP(t *testing.T) {
	clientSocket, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer clientSocket.Close()
	serverSocket, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer serverSocket.Close()
	cert, err := selfsign.GenerateSelfSigned()
	if err != nil {
		t.Fatal(err)
	}
	client, err := dtls.ClientWithOptions(clientSocket, serverSocket.LocalAddr(), dtls.WithCertificates(cert), dtls.WithInsecureSkipVerify(true))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	server, err := dtls.ServerWithOptions(serverSocket, clientSocket.LocalAddr(), dtls.WithCertificates(cert), dtls.WithClientAuth(dtls.RequireAnyClientCert))
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- server.HandshakeContext(ctx) }()
	if err = client.HandshakeContext(ctx); err != nil {
		t.Fatal(err)
	}
	if err = <-done; err != nil {
		t.Fatal(err)
	}
	if _, selected := client.SelectedSRTPProtectionProfile(); selected {
		t.Fatal("test unexpectedly negotiated SRTP")
	}
	transport := &DTLSTransport{api: NewAPI(), state: DTLSTransportStateConnecting}
	if err = transport.completeStart(client); err != nil {
		t.Fatalf("data-only handshake rejected: %v", err)
	}
	if transport.State() != DTLSTransportStateConnected {
		t.Fatal("DTLS not connected")
	}
	if transport.srtpCanStart {
		t.Fatal("SRTP enabled without a negotiated profile")
	}
	if err = transport.wantSRTP(); !errors.Is(err, ErrNoSRTPProtectionProfile) {
		t.Fatalf("later media request: %v", err)
	}
	// Strict media negotiation must still reject the same profile-less connection.
	media := &DTLSTransport{api: NewAPI(), state: DTLSTransportStateConnecting, srtpWanted: true}
	if err = media.completeStart(client); !errors.Is(err, ErrNoSRTPProtectionProfile) {
		t.Fatalf("media request accepted: %v", err)
	}
	// An actual encrypted application payload still crosses the DTLS connection.
	payload := []byte("NetherNet data without RTP")
	go func() { _, e := server.Write(payload); done <- e }()
	_ = client.SetReadDeadline(time.Now().Add(3 * time.Second))
	buffer := make([]byte, 128)
	n, err := client.Read(buffer)
	if err != nil || string(buffer[:n]) != string(payload) {
		t.Fatalf("DTLS data: %q %v", buffer[:n], err)
	}
	if err = <-done; err != nil {
		t.Fatal(err)
	}
	_ = client.SetReadDeadline(time.Time{})
	serverTransport := &DTLSTransport{api: NewAPI(), state: DTLSTransportStateConnecting}
	if err = serverTransport.completeStart(server); err != nil {
		t.Fatal(err)
	}
	clientSCTP := transport.api.NewSCTPTransport(transport)
	serverSCTP := serverTransport.api.NewSCTPTransport(serverTransport)
	defer clientSCTP.Stop()
	defer serverSCTP.Stop()
	received := make(chan string, 1)
	serverSCTP.OnDataChannel(func(channel *DataChannel) {
		channel.OnMessage(func(message DataChannelMessage) { _ = channel.Send(message.Data) })
	})
	go func() { done <- serverSCTP.StartContext(ctx, SCTPCapabilities{}) }()
	if err = clientSCTP.StartContext(ctx, SCTPCapabilities{}); err != nil {
		t.Fatal(err)
	}
	if err = <-done; err != nil {
		t.Fatal(err)
	}
	id := uint16(2)
	channel, err := transport.api.NewDataChannel(clientSCTP, &DataChannelParameters{Label: "ReliableDataChannel", ID: &id, Ordered: true})
	if err != nil {
		t.Fatal(err)
	}
	channel.OnMessage(func(message DataChannelMessage) { received <- string(message.Data) })
	channel.OnOpen(func() {
		if e := channel.Send(payload); e != nil {
			received <- e.Error()
		}
	})
	select {
	case got := <-received:
		if got != string(payload) {
			t.Fatalf("SCTP data round trip: %q", got)
		}
	case <-ctx.Done():
		t.Fatalf("SCTP data round trip: %v", ctx.Err())
	}
}
