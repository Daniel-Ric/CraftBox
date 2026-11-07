package desktop

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/klauspost/compress/s2"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sandertv/gophertunnel/minecraft/resource"
)

type AssetRecovery struct {
	Recovered int      `json:"recovered"`
	Remaining int      `json:"remaining"`
	Warnings  []string `json:"warnings"`
}

func capturePackKeys(path string) (keys map[string]string, err error) {
	keys = map[string]string{}
	defer func() {
		if recover() != nil {
			err = fmt.Errorf("Capture handshake could not be read")
		}
	}()
	f, e := os.Open(path)
	if e != nil {
		return keys, e
	}
	defer f.Close()
	i, e := f.Stat()
	if e != nil {
		return keys, e
	}
	var head [16]byte
	if _, e = io.ReadFull(f, head[:]); e != nil {
		return keys, e
	}
	size := binary.LittleEndian.Uint64(head[8:])
	if string(head[:4]) != "BTCP" || binary.LittleEndian.Uint32(head[4:]) != 5 || size > uint64(i.Size()-16) {
		return keys, fmt.Errorf("No complete version 5 capture header")
	}
	f.Seek(int64(size)+16, 0)
	var total uint32
	for count := 0; count < 128; count++ {
		var frame [17]byte
		if _, e = io.ReadFull(f, frame[:]); e != nil {
			return keys, e
		}
		n := binary.LittleEndian.Uint32(frame[4:])
		if binary.LittleEndian.Uint32(frame[:]) != 0xaaaaaaaa || n > 4<<20 || total+n > 32<<20 {
			return keys, fmt.Errorf("Capture handshake exceeds the recovery limit")
		}
		total += n
		raw := make([]byte, int(n)+4)
		if _, e = io.ReadFull(f, raw); e != nil {
			return keys, e
		}
		if binary.LittleEndian.Uint32(raw[n:]) != 0xbbbbbbbb {
			return keys, fmt.Errorf("Capture frame is incomplete")
		}
		decodedSize, e := s2.DecodedLen(raw[:n])
		if e != nil || decodedSize > 8<<20 {
			return keys, fmt.Errorf("Invalid compressed capture frame")
		}
		decoded, e := s2.Decode(nil, raw[:n])
		if e != nil {
			return keys, e
		}
		if frame[8] != 0 {
			continue
		}
		data, e := minecraft.ParseData(decoded)
		if e != nil {
			return keys, e
		}
		if data.Header.PacketID != packet.IDResourcePacksInfo {
			continue
		}
		info := &packet.ResourcePacksInfo{}
		info.Marshal(protocol.NewReader(bytes.NewBuffer(data.Payload.Bytes()), 0, false))
		for _, p := range info.TexturePacks {
			if len(p.ContentKey) == 32 {
				keys[p.UUID.String()+"_"+p.Version] = p.ContentKey
			}
		}
		return keys, nil
	}
	return keys, fmt.Errorf("Resource-pack handshake was not found")
}

func (a *App) RecoverCapturedAssets() (AssetRecovery, error) {
	result := AssetRecovery{Warnings: []string{}}
	items, e := a.assets().List()
	if e != nil {
		return result, e
	}
	wanted := map[string]string{}
	for _, v := range items {
		if v.Encrypted {
			wanted[v.UUID+"_"+v.Version] = v.ID
		}
	}
	if len(wanted) == 0 {
		return result, nil
	}
	entries, e := os.ReadDir(utils.PathData("captures"))
	if e != nil {
		return result, fmt.Errorf("No readable capture folder. Capture this server again to restore its model data")
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() > entries[j].Name() })
	keys := map[string]string{}
	read := 0
	for _, entry := range entries {
		if read >= 20 || len(keys) == len(wanted) {
			break
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".pcap2" {
			continue
		}
		read++
		found, e := capturePackKeys(utils.PathData("captures", entry.Name()))
		if e != nil {
			continue
		}
		for id, key := range found {
			if _, ok := wanted[id]; ok {
				keys[id] = key
			}
		}
	}
	for id, key := range keys {
		if e = a.ctx.Err(); e != nil {
			return result, e
		}
		root, e := a.assets().RootFor(wanted[id])
		if e != nil {
			return result, e
		}
		pack, e := resource.ReadPath(root)
		if e != nil {
			result.Warnings = append(result.Warnings, "A saved pack could not be opened")
			continue
		}
		readable, e := utils.ReadablePack(pack.WithContentKey(key))
		if e != nil {
			result.Warnings = append(result.Warnings, "A capture key did not match the saved pack")
			continue
		}
		if e = a.importReadableAssetPack(readable); e != nil {
			result.Warnings = append(result.Warnings, "A readable pack could not be saved: "+e.Error())
			continue
		}
		result.Recovered++
	}
	result.Remaining = len(wanted) - result.Recovered
	return result, nil
}
func (a *App) importReadableAssetPack(pack *resource.Pack) error {
	f, e := os.CreateTemp("", "craftbox-readable-pack-*.mcpack")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	_, e = pack.WriteTo(f)
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return e
	}
	_, e = a.assets().Import(a.ctx, "packs", []string{f.Name()}, "Capture")
	return e
}
