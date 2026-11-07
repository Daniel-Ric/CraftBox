package worldexport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
)

func (s *Settings) UnmarshalJSON(data []byte) error {
	type plain Settings
	var shape struct {
		Spawn json.RawMessage `json:"spawn"`
	}
	if err := json.Unmarshal(data, &shape); err != nil {
		return err
	}
	if len(shape.Spawn) > 0 && !bytes.Equal(bytes.TrimSpace(shape.Spawn), []byte("null")) {
		var coordinates []int32
		if err := json.Unmarshal(shape.Spawn, &coordinates); err != nil {
			return err
		}
		if len(coordinates) != 3 {
			return fmt.Errorf("spawn requires exactly three integer coordinates")
		}
	}
	var decoded plain
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil {
		return err
	}
	*s = Settings(decoded)
	return nil
}

type Options struct {
	Profile         string          `json:"profile"`
	Settings        Settings        `json:"settings"`
	Compact         bool            `json:"compact"`
	Dimensions      []DimensionRule `json:"dimensions,omitempty"`
	Biomes          []BiomeRule     `json:"biomes,omitempty"`
	PlayerDimension *int32          `json:"playerDimension,omitempty"`
}
type Settings struct {
	Name       *string   `json:"name,omitempty"`
	Spawn      *[3]int32 `json:"spawn,omitempty"`
	GameMode   *int32    `json:"gameMode,omitempty"`
	Difficulty *int32    `json:"difficulty,omitempty"`
	Time       *int64    `json:"time,omitempty"`
	Weather    *string   `json:"weather,omitempty"`
	Rules      GameRules `json:"rules"`
}
type GameRules struct {
	DaylightCycle   *bool  `json:"daylightCycle,omitempty"`
	WeatherCycle    *bool  `json:"weatherCycle,omitempty"`
	MobSpawning     *bool  `json:"mobSpawning,omitempty"`
	MobGriefing     *bool  `json:"mobGriefing,omitempty"`
	FireTick        *bool  `json:"fireTick,omitempty"`
	KeepInventory   *bool  `json:"keepInventory,omitempty"`
	TileDrops       *bool  `json:"tileDrops,omitempty"`
	EntityDrops     *bool  `json:"entityDrops,omitempty"`
	ShowCoordinates *bool  `json:"showCoordinates,omitempty"`
	CommandBlocks   *bool  `json:"commandBlocks,omitempty"`
	Commands        *bool  `json:"commands,omitempty"`
	RandomTickSpeed *int32 `json:"randomTickSpeed,omitempty"`
}
type Profile struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func Profiles() []Profile {
	return []Profile{
		{"original", "Original capture", "Preserve recorded world settings, weather and game rules."},
		{"creative", "Creative visit", "Creative mode, peaceful difficulty, clear weather and no new mobs."},
		{"screenshot", "Screenshot world", "Creative mode with noon lighting, frozen weather and no environmental changes."},
	}
}
func pointer[T any](v T) *T { return &v }
func profileSettings(id string) (Settings, error) {
	switch id {
	case "", "original":
		return Settings{}, nil
	case "creative":
		return Settings{GameMode: pointer(int32(1)), Difficulty: pointer(int32(0)), Weather: pointer("clear"), Rules: GameRules{MobSpawning: pointer(false), KeepInventory: pointer(true), Commands: pointer(true)}}, nil
	case "screenshot":
		return Settings{GameMode: pointer(int32(1)), Difficulty: pointer(int32(0)), Time: pointer(int64(6000)), Weather: pointer("clear"), Rules: GameRules{DaylightCycle: pointer(false), WeatherCycle: pointer(false), MobSpawning: pointer(false), MobGriefing: pointer(false), FireTick: pointer(false), KeepInventory: pointer(true), Commands: pointer(true), RandomTickSpeed: pointer(int32(0))}}, nil
	default:
		return Settings{}, fmt.Errorf("unknown world export profile %q", id)
	}
}
func (o Options) Validate() error {
	if err := validateTransforms(o); err != nil {
		return err
	}
	if _, err := profileSettings(o.Profile); err != nil {
		return err
	}
	s := o.Settings
	if s.Name != nil {
		n := strings.TrimSpace(*s.Name)
		if n == "" || len([]rune(n)) > 80 || strings.IndexFunc(n, unicode.IsControl) >= 0 {
			return fmt.Errorf("world name must contain 1 to 80 characters without control characters")
		}
	}

	if s.GameMode != nil && *s.GameMode != 0 && *s.GameMode != 1 && *s.GameMode != 2 {
		return fmt.Errorf("world game mode must be survival, creative or adventure")
	}
	if s.Difficulty != nil && (*s.Difficulty < 0 || *s.Difficulty > 3) {
		return fmt.Errorf("difficulty must be between 0 and 3")
	}
	if s.Time != nil && *s.Time < 0 {
		return fmt.Errorf("world time must not be negative")
	}
	if s.Weather != nil && *s.Weather != "clear" && *s.Weather != "rain" && *s.Weather != "thunder" {
		return fmt.Errorf("weather must be clear, rain or thunder")
	}
	if s.Rules.RandomTickSpeed != nil && (*s.Rules.RandomTickSpeed < 0 || *s.Rules.RandomTickSpeed > 4096) {
		return fmt.Errorf("random tick speed must be between 0 and 4096")
	}
	return nil
}
func ApplySettings(source map[string]any, options Options) (map[string]any, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}
	out := cloneNBT(source).(map[string]any)
	preset, _ := profileSettings(options.Profile)
	apply(out, preset)
	apply(out, options.Settings)
	return out, nil
}
func apply(out map[string]any, s Settings) {
	if s.Name != nil {
		out["LevelName"] = strings.TrimSpace(*s.Name)
	}
	if s.Spawn != nil {
		out["SpawnX"], out["SpawnY"], out["SpawnZ"] = s.Spawn[0], s.Spawn[1], s.Spawn[2]
	}
	if s.GameMode != nil {
		out["GameType"] = *s.GameMode
		out["ForceGameType"] = uint8(1)
	}
	if s.Difficulty != nil {
		out["Difficulty"] = *s.Difficulty
	}
	if s.Time != nil {
		out["Time"] = *s.Time
	}
	if s.Weather != nil {
		rain, lightning := float32(0), float32(0)
		if *s.Weather == "rain" || *s.Weather == "thunder" {
			rain = 1
		}
		if *s.Weather == "thunder" {
			lightning = 1
		}
		out["rainLevel"], out["lightningLevel"] = rain, lightning
		out["rainTime"], out["lightningTime"] = int32(12000), int32(12000)
	}
	rules := map[string]*bool{"dodaylightcycle": s.Rules.DaylightCycle, "doweathercycle": s.Rules.WeatherCycle, "domobspawning": s.Rules.MobSpawning, "mobgriefing": s.Rules.MobGriefing, "dofiretick": s.Rules.FireTick, "keepinventory": s.Rules.KeepInventory, "dotiledrops": s.Rules.TileDrops, "doentitydrops": s.Rules.EntityDrops, "showcoordinates": s.Rules.ShowCoordinates, "commandblocksenabled": s.Rules.CommandBlocks, "commandsEnabled": s.Rules.Commands}
	for name, value := range rules {
		if value != nil {
			out[name] = uint8(0)
			if *value {
				out[name] = uint8(1)
			}
		}
	}
	if s.Rules.Commands != nil {
		out["cheatsEnabled"] = uint8(0)
		if *s.Rules.Commands {
			out["cheatsEnabled"] = uint8(1)
		}
	}
	if s.Rules.RandomTickSpeed != nil {
		out["randomtickspeed"] = *s.Rules.RandomTickSpeed
	}
}
func cloneNBT(value any) any {
	switch v := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, x := range v {
			out[k] = cloneNBT(x)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, x := range v {
			out[i] = cloneNBT(x)
		}
		return out
	case []byte:
		return append([]byte(nil), v...)
	case []int32:
		return append([]int32(nil), v...)
	case []int64:
		return append([]int64(nil), v...)
	default:
		return value
	}
}
