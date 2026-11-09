package desktop

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/png"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"
	"weak"

	"github.com/Daniel-Ric/CraftBox/internal/assetlibrary"
	"github.com/Daniel-Ric/CraftBox/internal/bedrocklab"
	"github.com/Daniel-Ric/CraftBox/internal/commandexplorer"
	"github.com/Daniel-Ric/CraftBox/internal/diagnostics"
	"github.com/Daniel-Ric/CraftBox/internal/localworlds"
	"github.com/Daniel-Ric/CraftBox/internal/packetinspector"
	"github.com/Daniel-Ric/CraftBox/internal/scene"
	"github.com/Daniel-Ric/CraftBox/internal/structure"
	"github.com/Daniel-Ric/CraftBox/internal/terminal"

	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/Daniel-Ric/CraftBox/utils/auth"
	"github.com/Daniel-Ric/CraftBox/utils/commands"
	"github.com/Daniel-Ric/CraftBox/utils/connectinfo"
	"github.com/Daniel-Ric/CraftBox/utils/proxy"
	"github.com/Daniel-Ric/CraftBox/utils/updater"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/resource"
)

type Option struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Flag        string `json:"flag"`
	Default     string `json:"default"`
	Type        string `json:"type"`
	FileType    string `json:"fileType"`
}
type CommandInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Options     []Option `json:"options"`
}
type StartRequest struct {
	Command string            `json:"command"`
	Target  string            `json:"target"`
	Values  map[string]string `json:"values"`
}
type SavedWorld struct {
	CapturePoints []messages.CapturePoint `json:"capturePoints,omitempty"`
	CaptureFolder string                  `json:"-"`
	HeightRange   [2]int                  `json:"heightRange"`
	Spawn         *[3]int32               `json:"spawn,omitempty"`
	ParentID      string                  `json:"parentId,omitempty"`
	ID            string                  `json:"id"`
	ImagePath     string                  `json:"imagePath"`
	Player        [3]float32              `json:"player"`
	HasPlayer     bool                    `json:"hasPlayer"`
	Name          string                  `json:"name"`
	Path          string                  `json:"path"`
	Chunks        int                     `json:"chunks"`
	Entities      int                     `json:"entities"`
	State         string                  `json:"state"`
}
type Pack struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Version  string `json:"version"`
	Path     string `json:"path"`
	Size     uint64 `json:"size"`
	Received uint64 `json:"received"`
	Icon     string `json:"icon"`
	Error    string `json:"error"`
	Done     bool   `json:"done"`
}
type Skin struct {
	Name  string `json:"name"`
	Image string `json:"image"`
}
type Tile struct {
	WorldID string `json:"worldId"`
	X       int32  `json:"x"`
	Z       int32  `json:"z"`
	Image   string `json:"image"`
}
type State struct {
	CaptureID            uint64                   `json:"captureID,omitempty"`
	MinecraftJoinReady   bool                     `json:"minecraftJoinReady,omitempty"`
	ExpectedJoinAddress  string                   `json:"expectedJoinAddress,omitempty"`
	MinecraftJoinAddress string                   `json:"minecraftJoinAddress,omitempty"`
	JoinHostnameMatches  bool                     `json:"joinHostnameMatches,omitempty"`
	ActiveWorldID        string                   `json:"activeWorldId"`
	Revision             uint64                   `json:"revision"`
	Command              string                   `json:"command"`
	Running              bool                     `json:"running"`
	Stopping             bool                     `json:"stopping"`
	Connection           int                      `json:"connection"`
	ListenAddr           string                   `json:"listenAddr"`
	LocalIP              string                   `json:"localIP"`
	Worlds               []SavedWorld             `json:"worlds"`
	Packs                []Pack                   `json:"packs"`
	Skins                []Skin                   `json:"skins"`
	Player               [3]float32               `json:"player"`
	HasPlayer            bool                     `json:"hasPlayer"`
	Account              string                   `json:"account"`
	LoggedIn             bool                     `json:"loggedIn"`
	AuthCode             string                   `json:"authCode"`
	AuthURI              string                   `json:"authURI"`
	Authenticating       bool                     `json:"authenticating"`
	UpdateVersion        string                   `json:"updateVersion"`
	UpdateProgress       int                      `json:"updateProgress"`
	Error                string                   `json:"error"`
	ErrorConnection      *proxy.ConnectionFailure `json:"errorConnection,omitempty"`
	LogSessionID         string                   `json:"logSessionID,omitempty"`
}
type Bootstrap struct {
	InstallationID   string        `json:"installationId"`
	SceneBudgetBytes uint64        `json:"sceneBudgetBytes"`
	MapWindow        bool          `json:"mapWindow"`
	Theme            string        `json:"theme"`
	Version          string        `json:"version"`
	Minecraft        string        `json:"minecraft"`
	Commands         []CommandInfo `json:"commands"`
	State            State         `json:"state"`
}

func installationID(executable string) string {
	directory := filepath.Clean(filepath.Dir(executable))
	if runtime.GOOS == "windows" {
		directory = strings.ToLower(directory)
	}
	return fmt.Sprintf("%x", sha256.Sum256([]byte(directory)))
}

type App struct {
	labMu                 sync.Mutex
	labCancel             context.CancelFunc
	labProgress           bedrocklab.Progress
	marketMu              sync.Mutex
	marketSessions        map[string]*marketplaceSession
	packetMu              sync.Mutex
	packetRecorder        *packetinspector.Controller
	commandArchive        *commandexplorer.Store
	modelPickRead         func() (ModelPickResult, error)
	modelPickSession      string
	modelPickBusy         bool
	modelIndexMu          sync.Mutex
	modelIndex            map[string][]assetlibrary.Object
	modelIndexSources     []assetlibrary.PackPreviewSource
	modelIndexEpoch       uint64
	modelIndexWarming     bool
	clearHotbarFeedback   func()
	packetModules         packetinspector.Settings
	packetModulesSet      bool
	packetShieldReader    func() int32
	assetMu               sync.Mutex
	assetLibrary          *assetlibrary.Library
	assetTasks            chan func() error
	assetWorkerDone       chan struct{}
	assetQueuedBytes      int
	assetCapturedPacks    map[weak.Pointer[resource.Pack]]string
	installationScopeID   string
	pendingUpdate         *updater.Update
	updateCheckSequence   uint64
	terminals             *terminal.Manager
	modelExportCancel     context.CancelFunc
	modelExportStatus     ModelExportStatus
	worldExportCancel     context.CancelFunc
	worldExportSource     string
	worldExportStatus     WorldExportStatus
	localWorldSources     map[string]localworlds.World
	localWorldRoots       []localworlds.Root
	localWorldCancel      context.CancelFunc
	localWorldProgress    localworlds.Progress
	studioProjectPath     string
	pendingStudioDocument *renderDocument
	pendingStudioPath     string
	studioProjectDirty    bool
	documentCloseManaged  bool
	studioCloseApproved   bool
	sessionCloseApproved  bool
	renderDocument        *renderDocument
	renderDocuments       map[string]*renderDocument
	structureDocuments    map[string]*structure.Session
	structureScenes       map[string]*structureScene
	structureOperations   map[string]*structure.Session
	structurePreviews     map[string]*structurePreview
	renderTabImports      map[string]*App
	renderTabCancels      map[string]context.CancelFunc
	studioTabPaths        map[string]string
	renderCancel          context.CancelFunc
	renderGeneration      uint64
	scenes                map[string]*scene.Store
	sceneSources          map[string]*messages.EventSceneAvailable
	sceneResets           map[string]uint64
	worldRegions          map[string][]messages.WorldRegion
	mapHost               *mapWindowHost
	mapTheme              string
	mapFocus              uint64
	tileSequence          uint64
	tileResetSequence     uint64
	tileVersions          map[string]uint64
	mu                    sync.Mutex
	editMu                sync.Mutex
	ctx                   context.Context
	cancel                context.CancelFunc
	done                  chan struct{}
	loginDone             chan struct{}
	loginCancel           context.CancelFunc
	accountBusy           bool
	state                 State
	worldTiles            map[string]map[[2]int32]Tile
	tiles                 map[[2]int32]Tile
	sources               map[string]*connectinfo.ConnectInfo
	compositionStage      *compositionStage
	compositionCancel     context.CancelFunc
	compositionProgress   CompositionProgress
	logs                  []LogEntry
	logSequence           uint64
	emit                  func(string, any)
}

type LogEntry struct {
	ID      uint64 `json:"id"`
	Time    string `json:"time"`
	Level   string `json:"level"`
	Message string `json:"message"`
	Source  string `json:"source"`
}

func (a *App) GetLogs() []LogEntry {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]LogEntry{}, a.logs...)
}
func (a *App) appendLog(entry LogEntry) {
	a.mu.Lock()
	a.logSequence++
	entry.ID = a.logSequence
	a.logs = append(a.logs, entry)
	if len(a.logs) > 2000 {
		a.logs = a.logs[len(a.logs)-2000:]
	}
	emit := a.emit
	a.mu.Unlock()
	if emit != nil {
		emit("desktop:log", entry)
	}
}

func NewApp() *App {
	return &App{tileSequence: 1, tileVersions: make(map[string]uint64), worldTiles: make(map[string]map[[2]int32]Tile), ctx: context.Background(), tiles: make(map[[2]int32]Tile), sources: make(map[string]*connectinfo.ConnectInfo), state: State{Worlds: []SavedWorld{}, Packs: []Pack{}, Skins: []Skin{}}}
}
func (a *App) GetBootstrap() (Bootstrap, error) {
	result := Bootstrap{SceneBudgetBytes: availableSceneBudget(), Version: utils.Version, Minecraft: protocol.CurrentVersion, State: a.Snapshot(), Commands: []CommandInfo{}}
	result.InstallationID = a.installationScope()
	for _, name := range []string{"worlds", "packs", "skins", "packets"} {
		cmd, ok := commands.Registered[name]
		if !ok {
			continue
		}
		args, err := commands.ParseArgsType(reflect.ValueOf(cmd.Settings()), nil, nil)
		if err != nil {
			return result, err
		}
		info := CommandInfo{Name: name, Description: cmd.Description(), Options: []Option{}}
		for _, arg := range args {
			info.Options = append(info.Options, Option{Name: arg.Name, Description: arg.Desc, Flag: arg.Flag, Default: arg.Default, Type: arg.Type, FileType: arg.ExtType})
		}
		result.Commands = append(result.Commands, info)
	}
	return result, nil
}
func (a *App) snapshotLocked() State {
	s := a.state
	if m := diagnostics.Current(); m != nil {
		s.LogSessionID = m.ID()
	}
	s.Worlds = append([]SavedWorld{}, s.Worlds...)
	for i := range s.Worlds {
		s.Worlds[i].CapturePoints = append([]messages.CapturePoint{}, s.Worlds[i].CapturePoints...)
	}
	s.Packs = append([]Pack{}, s.Packs...)
	s.Skins = append([]Skin{}, s.Skins...)
	s.LoggedIn = auth.Auth.LoggedIn()
	return s
}
func (a *App) Snapshot() State { a.mu.Lock(); defer a.mu.Unlock(); return a.snapshotLocked() }
func (a *App) publish() {
	a.mu.Lock()
	a.state.Revision++
	s := a.snapshotLocked()
	emit := a.emit
	a.mu.Unlock()
	if emit != nil {
		emit("desktop:state", s)
	}
}
func (a *App) DismissError() {
	a.mu.Lock()
	a.state.Error = ""
	a.state.ErrorConnection = nil
	a.mu.Unlock()
	a.publish()
}
func (a *App) setErrorLocked(err error) {
	a.state.Error = err.Error()
	a.state.ErrorConnection = nil
	var connection *proxy.ConnectionFailure
	if errors.As(err, &connection) {
		copy := *connection
		a.state.ErrorConnection = &copy
	}
}
func (a *App) reportError(err error) {
	if err == nil {
		return
	}
	diagnostics.Record("error", "desktop", "desktop.error", err.Error(), nil)
	a.mu.Lock()
	a.setErrorLocked(err)
	a.mu.Unlock()
	a.publish()
}
func (a *App) StartCommand(request StartRequest) error {
	cmd, ok := commands.Registered[request.Command]
	if !ok {
		return fmt.Errorf("unknown command %q", request.Command)
	}
	settings := cmd.Settings()
	args, err := commands.ParseArgsType(reflect.ValueOf(settings), nil, nil)
	if err != nil {
		return err
	}
	known := make(map[string]bool)
	var selectedSource *connectinfo.ConnectInfo
	needsAccount := false
	for _, arg := range args {
		known[arg.Flag] = true
		if arg.Type == "connectInfo" {
			if strings.TrimSpace(request.Target) == "" {
				return errors.New("Select a server or capture")
			}
			a.mu.Lock()
			target := a.sources[request.Target]
			a.mu.Unlock()
			selectedSource = target
			if target == nil {
				if strings.HasPrefix(request.Target, "source:") {
					return errors.New("This source is no longer available. Select it again")
				}
				target = &connectinfo.ConnectInfo{Value: strings.TrimSpace(request.Target)}
			}
			needsAccount = !target.IsReplay()
			copyTarget := *target
			arg.SetConnectInfo(&copyTarget)
			continue
		}
		value, exists := request.Values[arg.Flag]
		if !exists {
			value = arg.Default
		}
		if value == "" && arg.Type == "int" {
			value = "0"
		}
		if value == "" && arg.Type == "bool" {
			value = "false"
		}
		if err := arg.Set(value); err != nil {
			return fmt.Errorf("%s: %w", arg.Name, err)
		}
	}
	for key := range request.Values {
		if !known[key] {
			return fmt.Errorf("unknown option %q", key)
		}
	}
	a.mu.Lock()
	if a.state.Running || a.state.Authenticating || a.accountBusy {
		a.mu.Unlock()
		return errors.New("A session or sign-in is already in progress")
	}
	if selectedSource != nil && a.sources[request.Target] != selectedSource {
		a.mu.Unlock()
		return errors.New("The account or source has changed. Select the source again")
	}
	if needsAccount && a.state.Account != "" {
		account := auth.Auth.Account()
		if account == nil || account.Name() != a.state.Account {
			a.mu.Unlock()
			return errors.New("Sign in to the selected account profile first")
		}
	}
	ctx, cancel := context.WithCancel(a.ctx)
	ctx = proxy.WithExtraHandlers(ctx, a.inspectorHandlers(ctx, request.Command)...)
	a.cancel = cancel
	done := make(chan struct{})
	a.done = done
	account := a.state.Account
	revision := a.state.Revision
	a.worldTiles = make(map[string]map[[2]int32]Tile)
	a.worldRegions = make(map[string][]messages.WorldRegion)
	for _, store := range a.scenes {
		store.Close()
	}
	a.scenes = make(map[string]*scene.Store)
	a.sceneSources = make(map[string]*messages.EventSceneAvailable)
	a.sceneResets = make(map[string]uint64)
	a.tileSequence++
	a.tileResetSequence = a.tileSequence
	a.tileVersions = make(map[string]uint64)
	a.state = State{Revision: revision, CaptureID: revision + 1, Command: request.Command, Running: true, Account: account, Worlds: []SavedWorld{}, Packs: []Pack{}, Skins: []Skin{}}
	a.tiles = make(map[[2]int32]Tile)
	emit := a.emit
	a.mu.Unlock()
	if emit != nil {
		emit("desktop:reset-map", nil)
	}
	a.publish()
	go func() {
		diagnostics.Record("info", "capture", "capture.started", "Capture command started", map[string]any{"command": request.Command, "target": request.Target})
		defer close(done)
		defer func() {
			if p := recover(); p != nil {
				if m := diagnostics.Current(); m != nil {
					m.Snapshot(fmt.Sprintf("Capture command panic: %v", p))
				}
				a.reportError(fmt.Errorf("command panic: %v", p))
			}
			cancel()
			diagnostics.Record("info", "capture", "capture.finished", "Capture command finished", map[string]any{"command": request.Command})
			a.mu.Lock()
			a.state.Running = false
			a.state.Stopping = false
			a.state.Authenticating = false
			a.state.AuthCode = ""
			a.state.AuthURI = ""
			a.cancel = nil
			a.mu.Unlock()
			a.publish()
		}()
		if err := cmd.Run(ctx, settings); err != nil && !errors.Is(err, context.Canceled) {
			a.reportError(err)
		}
	}()
	return nil
}
func (a *App) StopCommand() {
	a.mu.Lock()
	cancel := a.cancel
	if cancel != nil {
		a.state.Stopping = true
	}
	a.mu.Unlock()
	if cancel != nil {
		cancel()
		a.publish()
	}
}
func (a *App) GetMapTiles() []Tile {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]Tile, 0, len(a.tiles))
	for _, tile := range a.tiles {
		out = append(out, tile)
	}
	for _, worldMap := range a.worldTiles {
		for _, tile := range worldMap {
			out = append(out, tile)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].X == out[j].X {
			return out[i].Z < out[j].Z
		}
		return out[i].X < out[j].X
	})
	return out
}
func imageURL(img image.Image) string {
	if img == nil || (reflect.ValueOf(img).Kind() == reflect.Pointer && reflect.ValueOf(img).IsNil()) {
		return ""
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		return ""
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(b.Bytes())
}
func (a *App) HandleEvent(event any) error {
	if e, ok := event.(*messages.EventIngameFeedback); ok {
		a.packetMu.Lock()
		clear := a.clearHotbarFeedback
		a.packetMu.Unlock()
		if !e.Enabled && clear != nil {
			clear()
		}
		a.mu.Lock()
		emit := a.emit
		a.mu.Unlock()
		if emit != nil {
			emit("desktop:ingame-feedback", e.Enabled)
			emit("desktop:ingame-interface", proxy.IngameInterfaceSettings())
		}
		return nil
	}
	if e, ok := event.(*messages.EventCapturePoints); ok && len(e.Points) > 0 {
		a.packetMu.Lock()
		recorder := a.packetRecorder
		a.packetMu.Unlock()
		if recorder != nil {
			point := e.Points[len(e.Points)-1]
			recorder.MarkMoment(point.Created, fmt.Sprintf("CraftBox %s: %s · X %.3f Y %.3f Z %.3f", point.Kind, point.Name, point.Position[0], point.Position[1], point.Position[2]))
		}
	}
	if e, ok := event.(*messages.EventWorldAbandoned); ok {
		a.abandonEmptyWorld(e.WorldID)
		return nil
	}
	a.persistAssetEvent(event)
	if _, ok := event.(*messages.EventCapturedBlockDefinitions); ok {
		return nil
	}
	if deletedWorldEvent(event) {
		return nil
	}
	if e, ok := event.(*messages.EventSceneAvailable); ok {
		a.mu.Lock()
		if messages.WorldDeleted(e.WorldID) {
			a.mu.Unlock()
			return nil
		}
		if a.scenes == nil {
			a.scenes = make(map[string]*scene.Store)
		}
		a.scenes[e.WorldID] = e.Store
		if a.sceneSources == nil {
			a.sceneSources = map[string]*messages.EventSceneAvailable{}
		}
		a.sceneSources[e.WorldID] = e
		e.Retained = true
		a.mu.Unlock()
		return nil
	}
	if e, ok := event.(*messages.EventWorldRegionCreated); ok {
		return a.addWorldRegion(e.Region)
	}
	if e, ok := event.(*messages.EventStructureCreated); ok {
		a.mu.Lock()
		emit := a.emit
		a.mu.Unlock()
		if emit != nil {
			emit("structure-created", e)
		}
		return nil
	}
	if e, ok := event.(*messages.EventMapTiles); ok {
		tiles := make([]Tile, 0, len(e.Tiles))
		for _, t := range e.Tiles {
			tiles = append(tiles, Tile{WorldID: e.WorldID, X: t.Pos.X(), Z: t.Pos.Z(), Image: imageURL(&t.Img)})
		}
		a.mu.Lock()
		if messages.WorldDeleted(e.WorldID) {
			a.mu.Unlock()
			return nil
		}
		target := a.tiles
		if e.WorldID != "" {
			target = a.worldTiles[e.WorldID]
			if target == nil {
				target = make(map[[2]int32]Tile)
				a.worldTiles[e.WorldID] = target
			}
		}
		for _, t := range tiles {
			moved := false
			for _, region := range a.worldRegions[e.WorldID] {
				if !region.Bounds.Contains(t.X, t.Z) {
					continue
				}
				a.storeRegionTileLocked(t, region.ID)
				moved = moved || region.Move
			}
			if moved {
				continue
			}
			target[[2]int32{t.X, t.Z}] = t
			a.tileSequence++
			a.tileVersions[tileVersionKey(t)] = a.tileSequence
		}
		emit := a.emit
		hasRegions := len(a.worldRegions[e.WorldID]) > 0
		a.mu.Unlock()
		if emit != nil {

			if !hasRegions {
				emit("desktop:tiles", tiles)
			}
		}
		return nil
	}
	a.mu.Lock()
	if deletedWorldEvent(event) {
		a.mu.Unlock()
		return nil
	}
	switch e := event.(type) {
	case *messages.EventCapturePoints:
		for i := range a.state.Worlds {
			if a.state.Worlds[i].ID == e.WorldID {
				a.state.Worlds[i].CapturePoints = append([]messages.CapturePoint{}, e.Points...)
			}
		}
		for _, region := range a.worldRegions[e.WorldID] {
			for i := range a.state.Worlds {
				if a.state.Worlds[i].ID == region.ID {
					a.state.Worlds[i].CapturePoints = messages.CapturePointsInBounds(e.Points, region.Bounds)
				}
			}
		}
	case *messages.EventResetMap:
		a.tileSequence++
		a.tileResetSequence = a.tileSequence
		if e.WorldID != "" {
			delete(a.worldTiles, e.WorldID)
		} else {
			a.tiles = make(map[[2]int32]Tile)
			a.worldTiles = make(map[string]map[[2]int32]Tile)
		}
		if a.emit != nil {
			defer a.emit("desktop:reset-map", e.WorldID)
		}
	case *messages.EventWorldOpened:
		found := false
		for i := range a.state.Worlds {
			if a.state.Worlds[i].ID == e.WorldID {
				a.state.Worlds[i].Name = e.WorldName
				a.state.Worlds[i].CaptureFolder = e.Folder
				if e.HeightRange[0] < e.HeightRange[1] {
					a.state.Worlds[i].HeightRange = e.HeightRange
				}
				found = true
				break
			}
		}
		if !found {
			a.state.Worlds = append(a.state.Worlds, SavedWorld{ID: e.WorldID, Name: e.WorldName, CaptureFolder: e.Folder, State: "Capturing", HeightRange: e.HeightRange, CapturePoints: messages.ReadCapturePoints(e.Folder)})
		}
		a.state.ActiveWorldID = e.WorldID
		a.state.HasPlayer = false
	case *messages.EventPlayerPosition:
		a.updateRegionPlayersLocked(e)
		for i := range a.state.Worlds {
			if a.state.Worlds[i].ID == e.WorldID {
				copy(a.state.Worlds[i].Player[:], e.Position[:])
				a.state.Worlds[i].HasPlayer = true
			}
		}
		if e.WorldID != "" && e.WorldID != a.state.ActiveWorldID {
			break
		}
		copy(a.state.Player[:], e.Position[:])
		a.state.HasPlayer = true
	case *messages.EventMinecraftJoinAddress:
		a.state.ExpectedJoinAddress = e.Expected
		a.state.MinecraftJoinAddress = e.Observed
		a.state.JoinHostnameMatches = e.HostnameMatches
	case *messages.EventConnectStateUpdate:
		a.state.Connection = int(e.State)
		if e.State == messages.ConnectStateBegin {
			a.state.MinecraftJoinReady = false
		}
		if e.State == messages.ConnectStateListening {
			a.state.MinecraftJoinReady = true
		}
		if e.ListenAddr != "" {
			a.state.ListenAddr = e.ListenAddr
		}
		a.state.LocalIP, _ = utils.GetLocalIP()
	case *messages.EventProcessingWorldUpdate:
		found := false
		for i := range a.state.Worlds {
			if (e.WorldID != "" && a.state.Worlds[i].ID == e.WorldID) || (e.WorldID == "" && a.state.Worlds[i].Name == e.WorldName) {
				a.state.Worlds[i].State = e.State
				found = true
				break
			}
		}
		if !found {
			a.state.Worlds = append(a.state.Worlds, SavedWorld{ID: e.WorldID, Name: e.WorldName, State: e.State})
		}
	case *messages.EventFinishedSavingWorld:
		result := SavedWorld{ID: e.WorldID, ImagePath: e.ImagePath, Name: e.WorldName, Path: e.Filepath, Chunks: e.Chunks, Entities: e.Entities, State: "finished"}
		result.CaptureFolder = strings.TrimSuffix(e.Filepath, ".mcworld")
		found := false
		for i := range a.state.Worlds {
			if (e.WorldID != "" && a.state.Worlds[i].ID == e.WorldID) || (e.WorldID == "" && a.state.Worlds[i].Name == e.WorldName) {
				result.Player = a.state.Worlds[i].Player
				result.CapturePoints = a.state.Worlds[i].CapturePoints
				result.ParentID = a.state.Worlds[i].ParentID
				result.Spawn = a.state.Worlds[i].Spawn
				result.HeightRange = a.state.Worlds[i].HeightRange
				if edit := messages.WorldMetadata(e.WorldID); edit.Name != nil {
					result.Name = *edit.Name
				}
				result.HasPlayer = a.state.Worlds[i].HasPlayer
				a.state.Worlds[i] = result
				found = true
				break
			}
		}
		if !found {
			a.state.Worlds = append(a.state.Worlds, result)
		}
	case *messages.EventInitialPacksInfo:
		a.state.Packs = []Pack{}
		for _, p := range e.Packs {
			a.state.Packs = append(a.state.Packs, Pack{ID: p.UUID.String(), Size: p.Size})
		}
	case *messages.EventPackDownloadProgress:
		for i := range a.state.Packs {
			if a.state.Packs[i].ID == strings.SplitN(e.ID, "_", 2)[0] {
				a.state.Packs[i].Received += uint64(max(e.BytesAdded, 0))
			}
		}
	case *messages.EventFinishedPack:
		p := Pack{ID: e.ID, Name: e.Name, Version: e.Version, Path: e.Filepath, Icon: imageURL(e.Icon), Done: true}
		if e.Error != nil {
			p.Error = e.Error.Error()
		}
		found := false
		for i := range a.state.Packs {
			if a.state.Packs[i].ID == p.ID {
				p.Size = a.state.Packs[i].Size
				p.Received = p.Size
				a.state.Packs[i] = p
				found = true
				break
			}
		}
		if !found {
			a.state.Packs = append(a.state.Packs, p)
		}
	case *messages.EventPlayerSkin:
		width, height := int(e.Skin.SkinImageWidth), int(e.Skin.SkinImageHeight)
		src := ""
		if width > 0 && height > 0 && width <= 2048 && height <= 2048 && len(e.Skin.SkinData) == width*height*4 {
			img := image.NewNRGBA(image.Rect(0, 0, width, height))
			copy(img.Pix, e.Skin.SkinData)
			src = imageURL(img)
		}
		a.state.Skins = append(a.state.Skins, Skin{Name: e.PlayerName, Image: src})
	case *messages.EventDisplayAuthCode:
		a.state.AuthCode = e.AuthCode
		a.state.AuthURI = e.URI
		a.state.Authenticating = true
	case *messages.EventAuthFinished:
		a.state.AuthCode = ""
		a.state.AuthURI = ""
		if a.loginDone == nil {
			a.state.Authenticating = false
		}
		if e.Error != nil && !errors.Is(e.Error, context.Canceled) {
			a.setErrorLocked(e.Error)
		}
	case *messages.EventUpdateAvailable:
		a.state.UpdateVersion = e.Version
	case *messages.EventUpdateDownloadProgress:
		a.state.UpdateProgress = e.Progress
	case *messages.EventError:
		if e.Error != nil {
			a.setErrorLocked(e.Error)
		}
	}
	a.mu.Unlock()
	a.publish()
	return nil
}
