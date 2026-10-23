package query

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	"strings"
)

var version = [2]byte{0xfe, 0xfd}

var padding = [4]byte{0xff, 0xff, 0xff, 0x01}

var splitNum = [9]byte{'S', 'P', 'L', 'I', 'T', 'N', 'U', 'M', 0x00}

var playerKey = [...]byte{0x00, 0x01, 'p', 'l', 'a', 'y', 'e', 'r', '_', 0x00, 0x00}

const (
	queryTypeHandshake   = 0x09
	queryTypeInformation = 0x00
)

type request struct {
	RequestType byte

	SequenceNumber int32

	ResponseNumber int32
}

type response struct {
	ResponseType byte

	SequenceNumber int32

	ResponseNumber int32

	Information map[string]string
}

func (pk *request) Marshal(w io.Writer) {
	_, _ = w.Write(version[:])
	_ = binary.Write(w, binary.BigEndian, pk.RequestType)
	_ = binary.Write(w, binary.BigEndian, pk.SequenceNumber)
	if pk.RequestType == queryTypeInformation {
		_ = binary.Write(w, binary.BigEndian, pk.ResponseNumber)
		_, _ = w.Write(padding[:])
	}
}

func (pk *request) Unmarshal(r io.Reader) error {
	v := make([]byte, 2)
	if _, err := r.Read(v); err != nil {
		return err
	}
	if !bytes.Equal(v, version[:]) {
		return fmt.Errorf("invalid query request version: expected %X, got %X", version, v)
	}
	if err := binary.Read(r, binary.BigEndian, &pk.RequestType); err != nil {
		return err
	}
	if err := binary.Read(r, binary.BigEndian, &pk.RequestType); err != nil {
		return err
	}
	if pk.RequestType == queryTypeInformation {
		if err := binary.Read(r, binary.BigEndian, &pk.ResponseNumber); err != nil {
			return err
		}
		p := make([]byte, 4)
		_, err := r.Read(p)
		return err
	} else if pk.RequestType != queryTypeHandshake {
		return fmt.Errorf("unknown request type %X", pk.RequestType)
	}
	return nil
}

func (pk *response) Marshal(w io.Writer) {
	_ = binary.Write(w, binary.BigEndian, pk.ResponseType)
	_ = binary.Write(w, binary.BigEndian, pk.SequenceNumber)
	if pk.ResponseType == queryTypeHandshake {
		v := []byte(fmt.Sprint(pk.ResponseNumber))
		if len(v) != 12 {

			v = append(v, make([]byte, 12-len(v))...)
		}
		_, _ = w.Write(v)
	} else {
		_, _ = w.Write(splitNum[:])
		_ = binary.Write(w, binary.BigEndian, byte(0x80))
		_ = binary.Write(w, binary.BigEndian, byte(0))
		values := make([][]byte, 0, len(pk.Information)*2)
		for key, value := range pk.Information {
			values = append(values, []byte(key))
			values = append(values, []byte(value))
		}

		_, _ = w.Write(bytes.Join(values, []byte{0x00}))
	}
}

func (pk *response) Unmarshal(r io.Reader) error {
	if err := binary.Read(r, binary.BigEndian, &pk.ResponseType); err != nil {
		return err
	}
	if err := binary.Read(r, binary.BigEndian, &pk.SequenceNumber); err != nil {
		return err
	}
	switch pk.ResponseType {
	case queryTypeHandshake:
		numBytes := make([]byte, 12)
		if _, err := r.Read(numBytes); err != nil {
			return err
		}
		index := bytes.Index(numBytes, []byte{0x00})
		if index != -1 {
			numBytes = numBytes[:index]
		}
		num, err := strconv.ParseInt(string(numBytes), 10, 32)
		if err != nil {
			return fmt.Errorf("invalid response number in handshake query response: %w", err)
		}
		pk.ResponseNumber = int32(num)
	case queryTypeInformation:

		v := make([]byte, 11)
		if _, err := r.Read(v); err != nil {
			return err
		}
		information, err := io.ReadAll(r)
		if err != nil {
			return err
		}
		playerIndex := bytes.Index(information, playerKey[:])
		data := information
		if playerIndex != -1 {
			information = information[:playerIndex]
		}
		values := bytes.Split(information, []byte{0x00})
		pk.Information = make(map[string]string, len(values)/2)
		if len(values)%2 != 0 {

			values = values[:len(values)-1]
		}
		for i := 0; i < len(values); i += 2 {
			pk.Information[string(values[i])] = string(values[i+1])
		}
		if playerIndex != -1 {
			playerData := data[playerIndex+len(playerKey):]
			values = bytes.Split(playerData, []byte{0x00})
			players := make([]string, 0, len(values))
			for i := 0; i < len(values); i++ {
				if len(values[i]) == 0 {

					break
				}
				players = append(players, string(values[i]))
			}
			pk.Information["players"] = strings.Join(players, ", ")
		}

	default:
		return fmt.Errorf("unknown response type %X", pk.ResponseType)
	}
	return nil
}
