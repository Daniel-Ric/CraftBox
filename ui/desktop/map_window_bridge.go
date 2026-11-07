package desktop

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/Daniel-Ric/CraftBox/internal/worldexport"
	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

type MapWindowSnapshot struct {
	State    State  `json:"state"`
	Tiles    []Tile `json:"tiles"`
	Sequence uint64 `json:"sequence"`
	Reset    bool   `json:"reset"`
	Focus    uint64 `json:"focus"`
	Theme    string `json:"theme"`
}

type mapWindowHost struct {
	mu             sync.Mutex
	server         *http.Server
	address, token string
	process        *exec.Cmd
	closed         bool
}

func tileVersionKey(tile Tile) string { return fmt.Sprintf("%s:%d,%d", tile.WorldID, tile.X, tile.Z) }

func (a *App) mapWindowSnapshot(after uint64) MapWindowSnapshot {
	a.mu.Lock()
	defer a.mu.Unlock()
	state := a.snapshotLocked()
	state.AuthCode = ""
	state.AuthURI = ""
	state.Account = ""
	state.Authenticating = false
	state.Packs = []Pack{}
	state.Skins = []Skin{}
	reset := after == 0 || after > a.tileSequence || after < a.tileResetSequence
	result := MapWindowSnapshot{State: state, Tiles: []Tile{}, Sequence: a.tileSequence, Reset: reset, Focus: a.mapFocus, Theme: a.mapTheme}
	if result.Theme == "" {
		result.Theme = "dark"
	}
	add := func(tile Tile) {
		if reset || a.tileVersions[tileVersionKey(tile)] > after {
			result.Tiles = append(result.Tiles, tile)
		}
	}
	for _, tile := range a.tiles {
		add(tile)
	}
	for _, worldMap := range a.worldTiles {
		for _, tile := range worldMap {
			add(tile)
		}
	}
	return result
}

func (a *App) mapWindowHandler(token string) http.Handler {
	return http.HandlerFunc(func(out http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Origin") != "" || subtle.ConstantTimeCompare([]byte(req.Header.Get("Authorization")), []byte("Bearer "+token)) != 1 {
			http.Error(out, "Forbidden", http.StatusForbidden)
			return
		}
		if req.URL.Path == "/scene" || req.URL.Path == "/scene-export" {
			a.serveScene(out, req)
			return
		}
		if req.URL.Path == "/scene-page" {
			a.serveScenePage(out, req)
			return
		}
		if req.URL.Path == "/map-page" {
			a.serveMapPage(out, req)
			return
		}
		if strings.HasPrefix(req.URL.Path, "/composition-") {
			a.serveComposition(out, req)
			return
		}
		if req.URL.Path == "/world-metadata" {
			if req.Method != http.MethodPost {
				http.Error(out, "POST required", 405)
				return
			}
			var payload struct {
				WorldID string               `json:"worldId"`
				Name    *string              `json:"name,omitempty"`
				Spawn   *[]int32             `json:"spawn,omitempty"`
				Export  *worldexport.Options `json:"export,omitempty"`
			}
			decoder := json.NewDecoder(http.MaxBytesReader(out, req.Body, 4096))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&payload); err != nil {
				http.Error(out, "Invalid world data", 400)
				return
			}
			if err := decoder.Decode(new(any)); err != io.EOF {
				http.Error(out, "Invalid world data", 400)
				return
			}
			request := messages.EditWorld{WorldID: payload.WorldID, Name: payload.Name, Export: payload.Export}
			if payload.Spawn != nil {
				if len(*payload.Spawn) != 3 {
					http.Error(out, "Spawn point requires X, Y and Z", 400)
					return
				}
				request.Spawn = &[3]int32{(*payload.Spawn)[0], (*payload.Spawn)[1], (*payload.Spawn)[2]}
			}
			if err := a.EditWorld(request); err != nil {
				http.Error(out, err.Error(), 409)
				return
			}
			out.WriteHeader(http.StatusNoContent)
			return
		}
		if req.URL.Path == "/world-delete" {
			if req.Method != http.MethodPost {
				http.Error(out, "POST required", 405)
				return
			}
			var payload struct {
				WorldID string `json:"worldId"`
			}
			dec := json.NewDecoder(http.MaxBytesReader(out, req.Body, 4096))
			dec.DisallowUnknownFields()
			if err := dec.Decode(&payload); err != nil || payload.WorldID == "" {
				http.Error(out, "worldId required", 400)
				return
			}
			if err := dec.Decode(new(any)); err != io.EOF {
				http.Error(out, "Invalid request", 400)
				return
			}
			if err := a.DeleteCapturedWorld(payload.WorldID); err != nil {
				http.Error(out, err.Error(), 409)
				return
			}
			out.WriteHeader(http.StatusNoContent)
			return
		}
		if req.URL.Path == "/world-region" {
			if req.Method != http.MethodPost {
				out.Header().Set("Allow", "POST")
				http.Error(out, "POST required", 405)
				return
			}
			var payload struct {
				WorldID string `json:"worldId"`
				Name    string `json:"name"`
				Bounds  *struct {
					MinX *int32 `json:"minX"`
					MinZ *int32 `json:"minZ"`
					MaxX *int32 `json:"maxX"`
					MaxZ *int32 `json:"maxZ"`
				} `json:"bounds"`
				Move *bool `json:"move"`
			}
			decoder := json.NewDecoder(http.MaxBytesReader(out, req.Body, 16*1024))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&payload); err != nil {
				http.Error(out, "Invalid region data", 400)
				return
			}
			if err := decoder.Decode(new(any)); err != io.EOF {
				http.Error(out, "Invalid region data", 400)
				return
			}
			if payload.Bounds == nil || payload.Move == nil || payload.Bounds.MinX == nil || payload.Bounds.MinZ == nil || payload.Bounds.MaxX == nil || payload.Bounds.MaxZ == nil {
				http.Error(out, "Region bounds and save mode are required", 400)
				return
			}
			request := messages.CreateWorldRegion{WorldID: payload.WorldID, Name: payload.Name, Bounds: messages.ChunkBounds{MinX: *payload.Bounds.MinX, MinZ: *payload.Bounds.MinZ, MaxX: *payload.Bounds.MaxX, MaxZ: *payload.Bounds.MaxZ}, Move: *payload.Move}
			if err := request.Validate(); err != nil {
				http.Error(out, err.Error(), 400)
				return
			}
			region, err := a.CreateWorldRegion(request)
			if err != nil {
				http.Error(out, err.Error(), 409)
				return
			}
			out.Header().Set("Content-Type", "application/json")
			out.Header().Set("Cache-Control", "no-store")
			out.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(out).Encode(region)
			return
		}
		if req.Method != http.MethodGet {
			out.Header().Set("Allow", "GET")
			http.Error(out, "Read only", http.StatusMethodNotAllowed)
			return
		}
		if req.URL.Path != "/map-window" {
			http.NotFound(out, req)
			return
		}
		after, err := strconv.ParseUint(req.URL.Query().Get("after"), 10, 64)
		if req.URL.Query().Get("after") != "" && err != nil {
			http.Error(out, "Invalid tile sequence", http.StatusBadRequest)
			return
		}
		out.Header().Set("Content-Type", "application/json")
		out.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(out).Encode(a.mapWindowSnapshot(after))
	})
}

func (a *App) OpenMapWindow(theme string) error {
	if theme != "light" && theme != "system" {
		theme = "dark"
	}
	a.mu.Lock()
	a.mapTheme = theme
	a.mapFocus++
	if a.mapHost == nil {
		a.mapHost = &mapWindowHost{}
	}
	host := a.mapHost
	a.mu.Unlock()
	host.mu.Lock()
	defer host.mu.Unlock()
	if host.closed {
		return fmt.Errorf("The main window is closing")
	}
	if host.server == nil {
		listener, err := net.Listen("tcp4", "127.0.0.1:0")
		if err != nil {
			return err
		}
		secret := make([]byte, 32)
		if _, err := rand.Read(secret); err != nil {
			listener.Close()
			return err
		}
		host.token = base64.RawURLEncoding.EncodeToString(secret)
		host.address = "http://" + listener.Addr().String() + "/map-window"
		host.server = &http.Server{Handler: a.mapWindowHandler(host.token), ReadHeaderTimeout: 2 * time.Second, IdleTimeout: 15 * time.Second}
		go func() { _ = host.server.Serve(listener) }()
	}
	if host.process != nil {
		return nil
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	command := exec.Command(executable)
	command.Env = append(os.Environ(), "CRAFT_BOX_MAP_BRIDGE="+host.address, "CRAFT_BOX_MAP_TOKEN="+host.token)
	command.Stderr = os.Stderr
	if err := command.Start(); err != nil {
		return err
	}
	host.process = command
	go func() {
		_ = command.Wait()
		host.mu.Lock()
		if host.process == command {
			host.process = nil
		}
		host.mu.Unlock()
	}()
	return nil
}

func (a *App) closeMapWindowHost() {
	a.mu.Lock()
	host := a.mapHost
	a.mu.Unlock()
	if host == nil {
		return
	}
	host.mu.Lock()
	defer host.mu.Unlock()
	host.closed = true
	if host.server != nil {
		_ = host.server.Close()
	}
}
