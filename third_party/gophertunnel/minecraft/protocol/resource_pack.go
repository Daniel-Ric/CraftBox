package protocol

import "github.com/google/uuid"

const (
	PackSettingTypeFloat = iota
	PackSettingTypeBool
	PackSettingTypeString
	PackSettingTypeStringList
)

type TexturePackInfo struct {
	UUID uuid.UUID

	Version string

	Size uint64

	ContentKey string

	SubPackName string

	ContentIdentity string

	HasScripts bool

	AddonPack bool

	RTXEnabled bool

	DownloadURL string
}

func (x *TexturePackInfo) Marshal(r IO) {
	r.UUID(&x.UUID)
	r.String(&x.Version)
	r.Uint64(&x.Size)
	r.String(&x.ContentKey)
	r.String(&x.SubPackName)
	r.String(&x.ContentIdentity)
	r.Bool(&x.HasScripts)
	r.Bool(&x.AddonPack)
	r.Bool(&x.RTXEnabled)
	r.String(&x.DownloadURL)
}

type StackResourcePack struct {
	UUID string

	Version string

	SubPackName string
}

func (x *StackResourcePack) Marshal(r IO) {
	r.String(&x.UUID)
	r.String(&x.Version)
	r.String(&x.SubPackName)
}

type PackURL struct {
	UUIDVersion string

	URL string
}

func (x *PackURL) Marshal(r IO) {
	r.String(&x.UUIDVersion)
	r.String(&x.URL)
}

type PackSetting struct {
	Name string

	Value any
}
