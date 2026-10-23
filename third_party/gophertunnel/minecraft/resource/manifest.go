package resource

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

type Manifest struct {
	FormatVersion int `json:"format_version"`

	Header Header `json:"header"`

	Modules []Module `json:"modules"`

	Dependencies []Dependency `json:"dependencies,omitempty"`

	Capabilities []Capability `json:"capabilities,omitempty"`

	Metadata *Metadata `json:"metadata,omitempty"`

	worldTemplate bool
}

type Capability string

type Header struct {
	Name string `json:"name"`

	Description string `json:"description"`

	UUID uuid.UUID `json:"uuid"`

	Version Version `json:"version"`

	MinimumGameVersion Version `json:"min_engine_version"`
}

type Module struct {
	UUID string `json:"uuid"`

	Description string `json:"description"`

	Type string `json:"type"`

	Version Version `json:"version"`
}

type Dependency struct {
	UUID string `json:"uuid"`

	Version Version `json:"version"`
}

type Metadata struct {
	Authors []string `json:"authors,omitempty"`

	License string `json:"license,omitempty"`

	URL string `json:"url,omitempty"`
}

type Version [3]int

func (v *Version) UnmarshalJSON(b []byte) error {

	var arr [3]int
	if err := json.Unmarshal(b, &arr); err == nil {
		*v = arr
		return nil
	}

	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		s = strings.TrimSpace(s)
		parts := strings.Split(s, ".")
		if len(parts) != 3 {
			return fmt.Errorf("invalid version %q (need x.y.z)", s)
		}
		for i := range 3 {
			n, err := strconv.Atoi(parts[i])
			if err != nil {
				return fmt.Errorf("invalid version component %q in %q", parts[i], s)
			}
			v[i] = n
		}
		return nil
	}

	return fmt.Errorf("invalid version: %s", string(b))
}
