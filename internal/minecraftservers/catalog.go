package minecraftservers

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/Daniel-Ric/CraftBox/internal/assetlibrary"
	"github.com/Daniel-Ric/CraftBox/utils/connectinfo"
)

type Server struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

type Profile struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Channel string   `json:"channel"`
	Path    string   `json:"path"`
	Servers []Server `json:"-"`
	Count   int      `json:"count"`
	Skipped int      `json:"skipped"`
	Warning string   `json:"warning,omitempty"`
}

func ReadProfiles(installations []assetlibrary.Installation) []Profile {
	profiles := []Profile{}
	for _, installation := range installations {
		path := filepath.Join(installation.Storage, "minecraftpe", "external_servers.txt")
		file, err := os.Open(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		name := "Stable"
		if installation.Channel == "preview" {
			name = "Preview"
		}
		if installation.Layout == "GDK" {
			name += " · " + filepath.Base(filepath.Dir(filepath.Dir(installation.Storage)))
		} else {
			name += " · " + installation.Layout
		}
		profile := Profile{ID: installation.ID, Name: name, Channel: installation.Channel, Path: path, Servers: []Server{}}
		if err == nil {
			info, statErr := file.Stat()
			if statErr != nil {
				err = statErr
			} else if !info.Mode().IsRegular() || info.Size() > 8<<20 {
				err = errors.New("Server list must be a regular file up to 8 MiB")
			} else {
				err = readServers(file, &profile)
			}
			file.Close()
		}
		profile.Count = len(profile.Servers)
		if err != nil {
			profile.Warning = "Could not read the complete Minecraft server list: " + err.Error()
		} else if profile.Skipped > 0 {
			profile.Warning = fmt.Sprintf("Skipped %d invalid server entries", profile.Skipped)
		}
		profiles = append(profiles, profile)
	}
	return profiles
}

func readServers(file *os.File, profile *Profile) error {
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4096), 64<<10)
	seen := map[string]bool{}
	lines := 0
	for scanner.Scan() {
		lines++
		if lines > 60000 {
			return errors.New("Server list exceeds 60000 lines")
		}
		line := strings.TrimSpace(strings.TrimPrefix(scanner.Text(), "\ufeff"))
		if line == "" {
			continue
		}
		server, err := parseServer(line)
		if err != nil {
			profile.Skipped++
			continue
		}
		key := strings.ToLower(server.Address)
		if seen[key] {
			continue
		}
		seen[key] = true
		profile.Servers = append(profile.Servers, server)
	}
	return scanner.Err()
}

func parseServer(line string) (Server, error) {
	fields := strings.SplitN(line, ":", 3)
	if len(fields) != 3 {
		return Server{}, errors.New("Invalid server entry")
	}
	if _, err := strconv.ParseUint(fields[0], 10, 32); err != nil {
		return Server{}, err
	}
	endpoint := fields[2]
	var host, port string
	if strings.HasPrefix(endpoint, "[") {
		close := strings.Index(endpoint, "]:")
		if close < 0 {
			return Server{}, errors.New("Invalid IPv6 address")
		}
		host = endpoint[1:close]
		port, _, _ = strings.Cut(endpoint[close+2:], ":")
		if net.ParseIP(host) == nil {
			return Server{}, errors.New("Invalid IPv6 address")
		}
	} else {
		parts := strings.Split(endpoint, ":")
		if len(parts) < 2 || len(parts) > 3 {
			return Server{}, errors.New("Invalid server address")
		}
		host, port = parts[0], parts[1]
	}
	host, port = strings.TrimSpace(host), strings.TrimSpace(port)
	n, err := strconv.ParseUint(port, 10, 16)
	if err != nil || n == 0 || host == "" || len(host) > 512 || strings.ContainsAny(host, "/\\@?#[]") || strings.IndexFunc(host, unicode.IsSpace) >= 0 || strings.IndexFunc(host, unicode.IsControl) >= 0 {
		return Server{}, errors.New("Invalid server endpoint")
	}
	address := net.JoinHostPort(host, strconv.FormatUint(n, 10))
	target := connectinfo.ConnectInfo{Value: address}
	if _, err := target.Parsed(); err != nil {
		return Server{}, err
	}
	name := strings.TrimSpace(fields[1])
	if len(name) > 1024 || strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return Server{}, errors.New("Invalid server name")
	}
	if name == "" {
		name = host
	}
	return Server{Name: name, Address: address}, nil
}
