package minecraft

import (
	"context"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/df-mc/go-nethernet"
	"github.com/df-mc/go-nethernet/endpoint"
)

type answerObserver struct {
	*endpoint.Client
	mu         sync.Mutex
	candidates [][2]string
}
type observedNotifier struct {
	nethernet.Notifier
	observer *answerObserver
}

func (n observedNotifier) NotifySignal(signal *nethernet.Signal) bool {
	if signal.Type == nethernet.SignalTypeAnswer {
		n.observer.mu.Lock()
		for _, line := range strings.Split(signal.Data, "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 6 && strings.HasPrefix(fields[0], "a=candidate:") {
				n.observer.candidates = append(n.observer.candidates, [2]string{fields[4], fields[5]})
			}
		}
		n.observer.mu.Unlock()
	}
	return n.Notifier.NotifySignal(signal)
}
func (o *answerObserver) Notify(n nethernet.Notifier) func() {
	return o.Client.Notify(observedNotifier{Notifier: n, observer: o})
}

func TestDirectNetherNetAdvertisesLocalhostAndFixedUDPPort(t *testing.T) {
	l, err := (DirectNetherNet{}).Listen("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	go func() {
		c, err := l.Accept()
		if err == nil {
			defer c.Close()
			<-ctx.Done()
		}
	}()
	signaling := &answerObserver{Client: endpoint.NewClient()}
	c, err := (nethernet.Dialer{}).DialContext(ctx, "http://"+l.Addr().String(), signaling)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_, port, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	signaling.mu.Lock()
	defer signaling.mu.Unlock()
	found := false
	for _, candidate := range signaling.candidates {
		if ip := net.ParseIP(candidate[0]); ip != nil && ip.IsLoopback() && candidate[1] == port {
			found = true
		}
	}
	if !found {
		t.Fatalf("no localhost candidate on UDP port %s; observed %d candidates", port, len(signaling.candidates))
	}
	portNumber, _ := strconv.Atoi(port)
	udp, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: portNumber})
	if err == nil {
		udp.Close()
		t.Fatal("NetherNet did not reserve its configured UDP port")
	}
}
