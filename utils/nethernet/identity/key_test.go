package identity

import (
	"crypto/ecdsa"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestListenerIdentitySurvivesReload(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "nethernet", "listener-key.pem")
	first, err := LoadOrCreate(filename)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan *ecdsa.PrivateKey, 8)
	failures := make(chan error, 8)
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			key, err := LoadOrCreate(filename)
			if err != nil {
				failures <- err
			} else {
				results <- key
			}
		}()
	}
	wg.Wait()
	close(results)
	close(failures)
	for err := range failures {
		t.Error(err)
	}
	for key := range results {
		if !first.PublicKey.Equal(&key.PublicKey) || key.Curve.Params().BitSize != 384 {
			t.Fatal("listener identity changed")
		}
	}
}
func TestInvalidExistingListenerKeyIsPreserved(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "listener-key.pem")
	original := []byte("invalid existing key")
	if err := os.WriteFile(filename, original, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadOrCreate(filename); err == nil {
		t.Fatal("invalid existing key was accepted")
	}
	data, err := os.ReadFile(filename)
	if err != nil || string(data) != string(original) {
		t.Fatal("existing file was changed")
	}
}

func TestConcurrentListenerIdentityCreation(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "listener-key.pem")
	start := make(chan struct{})
	keys := make(chan *ecdsa.PrivateKey, 8)
	failures := make(chan error, 8)
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			key, err := LoadOrCreate(filename)
			if err != nil {
				failures <- err
			} else {
				keys <- key
			}
		}()
	}
	close(start)
	wg.Wait()
	close(keys)
	close(failures)
	for err := range failures {
		t.Error(err)
	}
	var first *ecdsa.PrivateKey
	count := 0
	for key := range keys {
		count++
		if first == nil {
			first = key
		} else if !first.PublicKey.Equal(&key.PublicKey) {
			t.Fatal("concurrent starts used different identities")
		}
	}
	if count != 8 {
		t.Fatalf("loaded %d identities, want 8", count)
	}
}
