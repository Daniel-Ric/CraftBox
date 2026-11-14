package resource

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

type Manifest struct {
	FormatVersion FormatVersion `json:"format_version"`
	Header        Header        `json:"header"`

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

type FormatVersion float64

func (f *FormatVersion) UnmarshalJSON(b []byte) error {
	*f = 0
	var number float64
	if err := json.Unmarshal(b, &number); err == nil {
		*f = FormatVersion(number)
		return nil
	}
	var v Version
	if err := v.UnmarshalJSON(b); err == nil {
		if parsed, err := strconv.ParseFloat(strconv.Itoa(v[0])+"."+strconv.Itoa(v[1]), 64); err == nil {
			*f = FormatVersion(parsed)
		}
	}
	return nil
}

type Version [3]int

var semVersion = regexp.MustCompile(`^(0|[1-9]\d{0,4})(?:\.(0|[1-9]\d{0,4})(?:\.(0|[1-9]\d{0,4})(?:-(?:(?:(?:0[0-9]*[A-Za-z-])|[1-9A-Za-z-])[0-9A-Za-z-]*|0)(?:\.(?:(?:(?:0[0-9]*[A-Za-z-])|[1-9A-Za-z-])[0-9A-Za-z-]*|0))*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?)?)?$`)

func parseVersion(s string) (Version, bool) {
	s = strings.TrimSpace(s)
	var v Version
	if match := semVersion.FindStringSubmatch(s); match != nil {
		for i, part := range match[1:] {
			if part != "" {
				v[i], _ = strconv.Atoi(part)
			}
		}
		return v, true
	}
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return v, false
	}
	for i, part := range parts {
		n, err := strconv.ParseUint(part, 10, 31)
		if err != nil {
			return v, false
		}
		v[i] = int(n)
	}
	return v, true
}

func (v *Version) UnmarshalJSON(b []byte) error {

	var arr [3]int
	if err := json.Unmarshal(b, &arr); err == nil {
		*v = arr
		return nil
	}

	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		parsed, ok := parseVersion(s)
		if !ok {
			return fmt.Errorf("invalid version %q", s)
		}
		*v = parsed
		return nil
	}

	return fmt.Errorf("invalid version: %s", string(b))
}
