package server

import "testing"

func TestTransportRejectsRemovedRakNet(t *testing.T) {
	c := UserConfig{}
	c.Network.Transport = []string{"raknet"}
	if _, err := c.transportListeners(); err == nil {
		t.Fatal("removed RakNet transport accepted")
	}
}

func TestNetherNetTransportDefaultsAndDeduplicates(t *testing.T) {
	c := UserConfig{}
	listeners, err := c.transportListeners()
	if err != nil || len(listeners) != 1 {
		t.Fatalf("empty config: %d listeners, %v", len(listeners), err)
	}
	c.Network.Transport = []string{"nethernet", " NETHERNET "}
	listeners, err = c.transportListeners()
	if err != nil || len(listeners) != 1 {
		t.Fatalf("duplicate config: %d listeners, %v", len(listeners), err)
	}
}
