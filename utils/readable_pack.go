package utils

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/sandertv/gophertunnel/minecraft/resource"
	"io"
	"io/fs"
	"path"
	"strings"
)



func ReadablePack(pack *resource.Pack) (*resource.Pack, error) {
	if pack.CanRead() {
		return pack, nil
	}
	if len(pack.ContentKey()) != 32 {
		return nil, fmt.Errorf("resource pack has an invalid content-key length")
	}
	contents, err := fs.ReadFile(pack, "contents.json")
	if err != nil {
		return nil, fmt.Errorf("resource pack contents: %w", err)
	}
	if len(contents) < 256 || !bytes.Equal(contents[4:8], []byte{0xfc, 0xb9, 0xcf, 0x9b}) {
		return nil, fmt.Errorf("invalid encrypted resource-pack header")
	}
	plaintext := bytes.TrimRight(CfbDecrypt(append([]byte{}, contents[256:]...), []byte(pack.ContentKey())), "\x00 \r\n\t")
	var index struct {
		Content []struct {
			Path string `json:"path"`
			Key  string `json:"key,omitempty"`
		} `json:"content"`
	}
	if err = json.Unmarshal(plaintext, &index); err != nil {
		return nil, fmt.Errorf("resource-pack contents could not be decoded")
	}
	keys := map[string]string{}
	for i, entry := range index.Content {
		if !fs.ValidPath(entry.Path) || strings.Contains(entry.Path, "\\") || strings.Contains(entry.Path, ":") {
			return nil, fmt.Errorf("invalid path in resource-pack contents")
		}
		if entry.Key != "" && len(entry.Key) != 32 {
			return nil, fmt.Errorf("invalid file-key length in resource pack")
		}
		keys[entry.Path] = entry.Key
		index.Content[i].Key = ""
	}
	cleanIndex, err := json.Marshal(index)
	if err != nil {
		return nil, err
	}
	var original bytes.Buffer
	if _, err = pack.WriteTo(&original); err != nil {
		return nil, err
	}
	reader, err := zip.NewReader(bytes.NewReader(original.Bytes()), int64(original.Len()))
	if err != nil {
		return nil, err
	}
	base := pack.BaseDir()
	if base != "" {
		base += "/"
	}
	var output bytes.Buffer
	seen := map[string]bool{}
	writer := zip.NewWriter(&output)
	for _, file := range reader.File {
		if !strings.HasPrefix(file.Name, base) {
			continue
		}
		name := strings.TrimPrefix(file.Name, base)
		if file.FileInfo().IsDir() {
			continue
		}
		seen[name] = true
		if !fs.ValidPath(name) || strings.Contains(name, "\\") || strings.Contains(name, ":") {
			return nil, fmt.Errorf("invalid resource-pack archive path")
		}
		if path.Base(name) == "signatures.json" {
			continue
		}
		input, err := file.Open()
		if err != nil {
			return nil, err
		}
		data, readErr := io.ReadAll(input)
		closeErr := input.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		if name == "contents.json" {
			data = cleanIndex
		} else if key := keys[name]; key != "" {
			data = CfbDecrypt(data, []byte(key))
		}
		target, err := writer.Create(name)
		if err != nil {
			return nil, err
		}
		if _, err = target.Write(data); err != nil {
			return nil, err
		}
	}
	if err = writer.Close(); err != nil {
		return nil, err
	}
	for name, key := range keys {
		if key != "" && !seen[name] {
			return nil, fmt.Errorf("encrypted resource-pack file is missing: %s", name)
		}
	}
	return resource.Read(bytes.NewReader(output.Bytes()))
}
