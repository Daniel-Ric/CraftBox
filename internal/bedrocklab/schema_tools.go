package bedrocklab

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

type SchemaToolRequest struct {
	Version string `json:"version"`
	Channel string `json:"channel"`
}
type SchemaBundle struct {
	Path    string `json:"path"`
	Version string `json:"version"`
	Channel string `json:"channel"`
	Count   int    `json:"count"`
	Source  string `json:"source"`
	Hash    string `json:"hash"`
}

var serverVersion = regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`)

type ServerRelease struct {
	Channel string `json:"channel"`
	Version string `json:"version"`
	URL     string `json:"url"`
}

var officialServerURL = regexp.MustCompile(`^https://www\.minecraft\.net/bedrockdedicatedserver/(bin-win|bin-win-preview)/bedrock-server-(\d+\.\d+\.\d+\.\d+)\.zip$`)

func parseServerReleases(b []byte) ([]ServerRelease, error) {
	var data struct {
		Result struct {
			Links []struct {
				DownloadType string `json:"downloadType"`
				DownloadURL  string `json:"downloadUrl"`
			} `json:"links"`
		} `json:"result"`
	}
	if e := json.Unmarshal(b, &data); e != nil {
		return nil, e
	}
	result := []ServerRelease{}
	for _, link := range data.Result.Links {
		channel := ""
		switch link.DownloadType {
		case "serverBedrockWindows":
			channel = "stable"
		case "serverBedrockPreviewWindows":
			channel = "preview"
		default:
			continue
		}
		match := officialServerURL.FindStringSubmatch(link.DownloadURL)
		if match == nil {
			continue
		}
		result = append(result, ServerRelease{channel, match[2], link.DownloadURL})
	}
	return result, nil
}
func LatestServerReleases(ctx context.Context) ([]ServerRelease, error) {
	request, e := http.NewRequestWithContext(ctx, http.MethodGet, "https://net-secondary.web.minecraft-services.net/api/v1.0/download/links", nil)
	if e != nil {
		return nil, e
	}
	request.Header.Set("User-Agent", "Mozilla/5.0 CraftBox/BedrockLab")
	client := &http.Client{Timeout: 20 * time.Second}
	response, e := client.Do(request)
	if e != nil {
		return nil, e
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return nil, fmt.Errorf("official download metadata returned HTTP %d", response.StatusCode)
	}
	b, e := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if e != nil {
		return nil, e
	}
	return parseServerReleases(b)
}

func extractServer(reader io.ReaderAt, size int64, destination string) error {
	z, e := zip.NewReader(reader, size)
	if e != nil {
		return e
	}
	if len(z.File) > 200000 {
		return errors.New("server ZIP has too many entries")
	}
	var total uint64
	for _, f := range z.File {
		name := strings.TrimSuffix(f.Name, "/")
		if !safePath(name) || f.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("unsafe server ZIP entry %q", f.Name)
		}
		total += f.UncompressedSize64
		if f.UncompressedSize64 > 1<<30 || total > 4<<30 {
			return errors.New("server ZIP exceeds extraction limits")
		}
		p := filepath.Join(destination, filepath.FromSlash(name))
		if f.FileInfo().IsDir() {
			if e = os.MkdirAll(p, 0700); e != nil {
				return e
			}
			continue
		}
		if e = os.MkdirAll(filepath.Dir(p), 0700); e != nil {
			return e
		}
		src, e := f.Open()
		if e != nil {
			return e
		}
		dst, e := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if e != nil {
			src.Close()
			return e
		}
		_, e = io.Copy(dst, io.LimitReader(src, int64(f.UncompressedSize64)+1))
		src.Close()
		closeErr := dst.Close()
		if e != nil {
			return e
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}
func (s *Store) SchemaBundles() ([]SchemaBundle, error) {
	result := []SchemaBundle{}
	entries, e := os.ReadDir(filepath.Join(s.Root, "schemas"))
	if os.IsNotExist(e) {
		return result, nil
	}
	if e != nil {
		return nil, e
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		b, e := os.ReadFile(filepath.Join(s.Root, "schemas", entry.Name(), "craftbox-schema.json"))
		if e != nil {
			continue
		}
		var m SchemaBundle
		if json.Unmarshal(b, &m) == nil {
			m.Path = filepath.Join(s.Root, "schemas", entry.Name())
			result = append(result, m)
		}
	}
	return result, nil
}
func (s *Store) PrepareSchemas(ctx context.Context, request SchemaToolRequest, progress func(Progress)) (SchemaBundle, error) {
	var result SchemaBundle
	if runtime.GOOS != "windows" {
		return result, errors.New("automatic BDS documentation generation requires Windows; import an existing schema folder instead")
	}
	if !serverVersion.MatchString(request.Version) {
		return result, errors.New("enter the exact four-part BDS version, for example 1.26.52.3")
	}
	channel := "preview"
	folder := "bin-win-preview"
	if request.Channel == "stable" {
		channel = "stable"
		folder = "bin-win"
	}
	result.Version, result.Channel = request.Version, channel
	result.Path = filepath.Join(s.Root, "schemas", channel+"-"+request.Version)
	if _, err := os.Stat(result.Path); err == nil {
		return result, errors.New("schema bundle already exists; select it from the library")
	}
	result.Source = "https://www.minecraft.net/bedrockdedicatedserver/" + folder + "/bedrock-server-" + request.Version + ".zip"
	if e := os.MkdirAll(filepath.Join(s.Root, "schema-jobs"), 0700); e != nil {
		return result, e
	}
	work, e := os.MkdirTemp(filepath.Join(s.Root, "schema-jobs"), "bds-*")
	if e != nil {
		return result, e
	}

	work, e = filepath.Abs(work)
	if e != nil {
		return result, e
	}
	emit := func(phase string) {
		if progress != nil {
			progress(Progress{Running: true, Phase: phase, Path: request.Version})
		}
	}
	emit("Downloading official BDS")
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, result.Source, nil)
	if e != nil {
		return result, e
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 CraftBox/BedrockLab")
	client := &http.Client{Timeout: 8 * time.Minute, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 5 || req.URL.Scheme != "https" {
			return errors.New("unsafe BDS redirect")
		}
		return nil
	}}
	response, e := client.Do(req)
	if e != nil {
		return result, e
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return result, fmt.Errorf("official BDS download returned HTTP %d; this build may not have a matching server release", response.StatusCode)
	}
	archive, e := os.Create(filepath.Join(work, "bedrock-server.zip"))
	if e != nil {
		return result, e
	}
	hash := sha256.New()
	n, e := io.Copy(io.MultiWriter(archive, hash), io.LimitReader(response.Body, 1<<30+1))
	archive.Close()
	if e != nil {
		return result, e
	}
	if n > 1<<30 {
		return result, errors.New("BDS download exceeds 1 GiB")
	}
	result.Hash = hex.EncodeToString(hash.Sum(nil))
	emit("Unpacking BDS")
	server := filepath.Join(work, "server")
	if e = os.MkdirAll(server, 0700); e != nil {
		return result, e
	}
	archive, e = os.Open(filepath.Join(work, "bedrock-server.zip"))
	if e != nil {
		return result, e
	}
	e = extractServer(archive, n, server)
	archive.Close()
	if e != nil {
		return result, e
	}
	if e = ctx.Err(); e != nil {
		return result, e
	}
	if e = os.WriteFile(filepath.Join(server, "test_config.json"), []byte(`{"generate_documentation":true,"generate_api_metadata":true}`), 0600); e != nil {
		return result, e
	}

	ports := make([]net.PacketConn, 0, 2)
	defer func() {
		for _, port := range ports {
			_ = port.Close()
		}
	}()
	for i := 0; i < 2; i++ {
		port, err := net.ListenPacket("udp4", "127.0.0.1:0")
		if err != nil {
			return result, err
		}
		ports = append(ports, port)
	}
	properties := fmt.Sprintf("server-name=CraftBox schema generation\nonline-mode=true\nallow-list=true\nmax-players=1\nserver-port=%d\nserver-portv6=%d\nenable-lan-visibility=false\n", ports[0].LocalAddr().(*net.UDPAddr).Port, ports[1].LocalAddr().(*net.UDPAddr).Port)
	if e = os.WriteFile(filepath.Join(server, "server.properties"), []byte(properties), 0600); e != nil {
		return result, e
	}
	emit("Generating schemas")
	runCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(runCtx, filepath.Join(server, "bedrock_server.exe"))
	cmd.Dir = server
	hideCommand(cmd)
	log, e := os.Create(filepath.Join(work, "bds.log"))
	if e != nil {
		return result, e
	}
	defer log.Close()
	cmd.Stdout, cmd.Stderr = log, log
	stdin, e := cmd.StdinPipe()
	if e != nil {
		return result, e
	}
	for _, port := range ports {
		_ = port.Close()
	}
	if e = cmd.Start(); e != nil {
		return result, e
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	defer func() { _ = cmd.Process.Kill() }()
	last, steady := 0, 0
	timer := time.NewTicker(time.Second)
	defer timer.Stop()
	ready, exited := false, false
	for !ready {
		select {
		case <-runCtx.Done():
			return result, fmt.Errorf("BDS schema generation stopped: %w; log: %s", runCtx.Err(), log.Name())
		case e := <-done:
			if e != nil {
				return result, fmt.Errorf("BDS schema generation failed (%v); log: %s", e, log.Name())
			}
			if _, err := LoadSchemas(filepath.Join(server, "docs", "json_schemas")); err != nil {
				return result, fmt.Errorf("BDS produced no usable schemas: %w; log: %s", err, log.Name())
			}
			ready, exited = true, true
		case <-timer.C:
			count := 0
			_ = filepath.WalkDir(filepath.Join(server, "docs", "json_schemas"), func(p string, d os.DirEntry, e error) error {
				if e == nil && !d.IsDir() && strings.HasSuffix(strings.ToLower(p), ".json") {
					count++
				}
				return nil
			})
			if count > 0 && count == last {
				steady++
			} else {
				steady = 0
			}
			last = count
			ready = steady >= 4
		}
	}
	if !exited {
		_, _ = io.WriteString(stdin, "stop\n")
		stdin.Close()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			_ = cmd.Process.Kill()
			<-done
		}
	}
	registry, e := LoadSchemas(filepath.Join(server, "docs", "json_schemas"))
	if e != nil {
		return result, e
	}
	result.Count = len(registry.documents)
	result.Path = filepath.Join(s.Root, "schemas", channel+"-"+request.Version)
	if e = os.MkdirAll(filepath.Dir(result.Path), 0700); e != nil {
		return result, e
	}
	if _, e = os.Stat(result.Path); e == nil {
		return result, errors.New("schema bundle already exists; select it from the library")
	}
	target := filepath.Join(work, "bundle")
	if e = os.MkdirAll(filepath.Join(target, "metadata"), 0700); e != nil {
		return result, e
	}
	if e = os.Rename(filepath.Join(server, "docs", "json_schemas"), filepath.Join(target, "metadata", "json_schemas")); e != nil {
		return result, e
	}
	b, _ := json.MarshalIndent(result, "", "  ")
	if e = os.WriteFile(filepath.Join(target, "craftbox-schema.json"), b, 0600); e != nil {
		return result, e
	}
	return result, os.Rename(target, result.Path)
}
