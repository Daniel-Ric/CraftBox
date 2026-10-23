package identity

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)



func LoadOrCreate(filename string) (*ecdsa.PrivateKey, error) {
	key, err := load(filename)
	if err == nil {
		return key, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(filename), 0700); err != nil {
		return nil, err
	}
	key, err = ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return nil, err
	}
	encoded, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}
	temporary, err := os.CreateTemp(filepath.Dir(filename), ".listener-key-*")
	if err != nil {
		return nil, err
	}
	defer os.Remove(temporary.Name())
	if _, err = temporary.Write(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: encoded})); err != nil {
		temporary.Close()
		return nil, err
	}
	if err = temporary.Sync(); err != nil {
		temporary.Close()
		return nil, err
	}
	if err = temporary.Close(); err != nil {
		return nil, err
	}
	if err = os.Link(temporary.Name(), filename); err != nil {
		if errors.Is(err, os.ErrExist) {
			return load(filename)
		}
		return nil, fmt.Errorf("publish NetherNet listener identity: %w", err)
	}
	return key, nil
}

func load(filename string) (*ecdsa.PrivateKey, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	block, rest := pem.Decode(data)
	if block == nil || block.Type != "EC PRIVATE KEY" || len(rest) != 0 {
		return nil, fmt.Errorf("invalid NetherNet listener key file %s", filename)
	}
	key, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse NetherNet listener key: %w", err)
	}
	if key.Curve.Params().BitSize != 384 {
		return nil, errors.New("NetherNet listener identity requires an ECDSA P-384 key")
	}
	return key, nil
}
