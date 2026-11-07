package packetinspector

import (
	"compress/flate"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"github.com/klauspost/compress/s2"
	"github.com/sandertv/gophertunnel/minecraft"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func (s *Store) Import(path string) (result Capture, err error) {
	info, e := os.Lstat(path)
	if e != nil {
		return result, e
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return result, errors.New("Choose a regular capture file or session folder")
	}
	if info.IsDir() {
		path = filepath.Join(path, "packets.log")
		info, e = os.Stat(path)
		if e != nil {
			return result, errors.New("ProxyPass session folder must contain packets.log")
		}
	}
	if !info.Mode().IsRegular() || info.Size() > 16<<30 {
		return result, errors.New("Import a capture of at most 16 GiB")
	}
	f, e := os.Open(path)
	if e != nil {
		return result, e
	}
	defer f.Close()
	result, e = s.Create(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)), filepath.Base(path), 0)
	if e != nil {
		return result, e
	}
	success := false
	defer func() {
		if !success {
			_ = s.Delete(result.ID)
		}
	}()
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".pcap2":
		err = s.importPcap2(result.ID, f, info.Size())
	case ".json":
		err = s.importJSON(result.ID, f)
	case ".log", ".txt":
		err = s.importProxyPass(result.ID, f, info.ModTime())
	default:
		err = errors.New("Choose ProxyWeb/native JSON, ProxyPass packets.log or CraftBox pcap2")
	}
	if err != nil {
		return result, err
	}
	_ = s.Finish(result.ID)
	result, err = s.Capture(result.ID)
	success = err == nil
	return result, err
}
func importedPacket(m map[string]json.RawMessage) (Packet, error) {
	var p Packet
	p.OriginSession = scalar(m["session"])
	p.Name = scalar(m["name"])
	p.Direction = strings.ToLower(scalar(m["direction"]))
	if p.Name == "" || len(p.Name) > 256 {
		return p, errors.New("Missing or invalid packet name")
	}
	if p.Direction != "clientbound" && p.Direction != "serverbound" {
		return p, errors.New("Invalid direction")
	}
	_ = json.Unmarshal(m["time"], &p.Time)
	if p.Time <= 0 {
		return p, errors.New("Invalid timestamp")
	}
	num, _ := strconv.ParseInt(scalar(m["packetId"]), 10, 64)
	if num < 0 || num > 1<<32-1 {
		p.Partial = true
	} else {
		p.PacketID = uint32(num)
	}
	_ = json.Unmarshal(m["bytes"], &p.Bytes)
	if p.Bytes < 0 {
		return p, errors.New("Invalid packet size")
	}
	_ = json.Unmarshal(m["header"], &p.Header)
	p.Payload = scalar(m["payload"])
	if p.Payload == "" {
		p.Payload = "{}"
	}
	if len(p.Payload) > maxRecord {
		return p, errors.New("Payload too large")
	}
	if raw := scalar(m["raw"]); raw != "" {
		var e error
		p.Raw, e = base64.StdEncoding.DecodeString(raw)
		if e != nil {
			return p, errors.New("Invalid base64 bytes")
		}
	}
	var truncated bool
	_ = json.Unmarshal(m["truncated"], &truncated)
	p.Partial = p.Partial || truncated
	var partial bool
	_ = json.Unmarshal(m["partial"], &partial)
	p.Partial = p.Partial || partial
	p.Stage = scalar(m["stage"])
	if p.Stage == "" {
		p.Stage = "imported"
	}
	p.Action = scalar(m["action"])
	p.Before = scalar(m["before"])
	p.Reason = scalar(m["reason"])
	p.Note = scalar(m["note"])
	if len(p.OriginSession) > 256 || len(p.Stage) > 64 || len(p.Note) > 8192 || len(p.Action) > 64 {
		return p, errors.New("Packet metadata exceeds supported field limits")
	}
	_ = json.Unmarshal(m["bookmarked"], &p.Bookmarked)
	return p, nil
}

var proxyLine = regexp.MustCompile(`^\[(\d\d:\d\d:\d\d[:.]\d{3})\] \[(SERVER BOUND|CLIENT BOUND)\] - ([A-Za-z0-9_]+)(.*)$`)

func (s *Store) importPcap2(id string, f *os.File, size int64) error {
	var head [16]byte
	if _, e := io.ReadFull(f, head[:]); e != nil {
		return e
	}
	if string(head[:4]) != "BTCP" {
		return errors.New("Invalid pcap2 header")
	}
	version := binary.LittleEndian.Uint32(head[4:])
	zipSize := binary.LittleEndian.Uint64(head[8:])
	if version < 4 || version > 5 || zipSize > uint64(size-16) {
		return errors.New("Unsupported or damaged pcap2 file")
	}
	if _, e := f.Seek(int64(zipSize)+16, io.SeekStart); e != nil {
		return e
	}
	var reader io.Reader = f
	if version == 4 {
		compressed := flate.NewReader(f)
		defer compressed.Close()
		reader = compressed
	}
	count := 0
	for {
		var row [17]byte
		if _, e := io.ReadFull(reader, row[:]); e == io.EOF {
			break
		} else if e != nil {
			return e
		}
		length := binary.LittleEndian.Uint32(row[4:])
		if binary.LittleEndian.Uint32(row[:]) != 0xAAAAAAAA || length > maxRecord || row[8] > 1 {
			return errors.New("Invalid pcap2 frame")
		}
		body := make([]byte, int(length)+4)
		if _, e := io.ReadFull(reader, body); e != nil {
			return e
		}
		if binary.LittleEndian.Uint32(body[len(body)-4:]) != 0xBBBBBBBB {
			return errors.New("Invalid pcap2 frame trailer")
		}
		raw := body[:len(body)-4]
		if version == 5 {
			n, e := s2.DecodedLen(raw)
			if e != nil || n > maxRecord {
				return errors.New("pcap2 frame exceeds decode limit")
			}
			raw, e = s2.Decode(nil, raw)
			if e != nil {
				return e
			}
		}
		data, e := minecraft.ParseData(raw)
		if e != nil {
			return e
		}
		direction := "clientbound"
		if row[8] == 1 {
			direction = "serverbound"
		}
		at := time.UnixMilli(int64(binary.LittleEndian.Uint64(row[9:])))
		p := FromRaw(data.Header.PacketID, data.Payload.Bytes(), direction, "imported", at, 0)
		p.Header = append([]byte{}, raw[:len(raw)-len(data.Payload.Bytes())]...)
		if _, e = s.Append(id, p); e != nil {
			return e
		}
		count++
	}
	if count == 0 {
		return errors.New("Capture contains no packets")
	}
	return nil
}
