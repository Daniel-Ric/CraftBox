package query

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"net"
	"time"
)

func Do(address string) (information map[string]string, err error) {
	conn, err := net.Dial("udp", address)
	if err != nil {
		return nil, fmt.Errorf("dial udp: %w", err)
	}

	if err := conn.SetDeadline(time.Now().Add(time.Second * 5)); err != nil {
		return nil, err
	}
	r := rand.New(rand.NewSource(time.Now().Unix()))

	b := new(bytes.Buffer)
	(&request{
		RequestType:    queryTypeHandshake,
		SequenceNumber: r.Int31(),
	}).Marshal(b)
	if _, err := conn.Write(b.Bytes()); err != nil {
		return nil, err
	}
	b.Reset()

	data := make([]byte, math.MaxUint16)
	n, err := conn.Read(data)
	if err != nil {
		return nil, err
	}
	resp := &response{}
	if err := resp.Unmarshal(bytes.NewBuffer(data[:n])); err != nil {
		return nil, err
	}

	(&request{
		RequestType:    queryTypeInformation,
		SequenceNumber: r.Int31(),
		ResponseNumber: resp.ResponseNumber,
	}).Marshal(b)
	if _, err := conn.Write(b.Bytes()); err != nil {
		return nil, err
	}

	n, err = conn.Read(data)
	if err != nil {
		return nil, err
	}
	resp = &response{}
	return resp.Information, resp.Unmarshal(bytes.NewBuffer(data[:n]))
}
