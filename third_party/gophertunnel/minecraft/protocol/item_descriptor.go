package protocol

import "fmt"

type ItemDescriptorCount struct {
	Descriptor ItemDescriptor

	Count int32
}

type ItemDescriptor interface {
	Marshaler
}

const (
	ItemDescriptorInvalid = iota
	ItemDescriptorDefault
	ItemDescriptorMoLang
	ItemDescriptorItemTag
)

type InvalidItemDescriptor struct{}

func (*InvalidItemDescriptor) Marshal(IO) {}

type DefaultItemDescriptor struct {
	Name string

	MetadataValue int32
}

func (x *DefaultItemDescriptor) Marshal(r IO) {
	r.String(&x.Name)
	r.Varint32(&x.MetadataValue)
}

type MoLangItemDescriptor struct {
	Expression string
	Version    int16
}

func (x *MoLangItemDescriptor) Marshal(r IO) {
	r.String(&x.Expression)
	r.Int16(&x.Version)
}

func itemDescriptorType(x ItemDescriptor) (uint8, string, bool) {
	switch x.(type) {
	case nil, *InvalidItemDescriptor:
		return ItemDescriptorInvalid, "empty", true
	case *DefaultItemDescriptor:
		return ItemDescriptorDefault, "name", true
	case *MoLangItemDescriptor:
		return ItemDescriptorMoLang, "tags", true
	case *ItemTagItemDescriptor:
		return ItemDescriptorItemTag, "item_tag", true
	default:
		return 0, "", false
	}
}

func itemDescriptorFromType(id uint8) (ItemDescriptor, bool) {
	switch id {
	case ItemDescriptorInvalid:
		return &InvalidItemDescriptor{}, true
	case ItemDescriptorDefault:
		return &DefaultItemDescriptor{}, true
	case ItemDescriptorMoLang:
		return &MoLangItemDescriptor{}, true
	case ItemDescriptorItemTag:
		return &ItemTagItemDescriptor{}, true
	default:
		return nil, false
	}
}

func StackRequestItemDescriptorCount(r IO, x *ItemDescriptorCount) {
	id, _, ok := itemDescriptorType(x.Descriptor)
	if !ok {
		r.UnknownEnumOption(fmt.Sprintf("%T", x.Descriptor), "stack request item descriptor type")
		return
	}
	variant := uint32(id)
	r.Varuint32(&variant)
	legacyID := id
	r.Uint8(&legacyID)
	if variant > ItemDescriptorItemTag {
		r.UnknownEnumOption(variant, "stack request item descriptor type")
		return
	}
	descriptor := x.Descriptor
	if descriptor == nil {
		descriptor = &InvalidItemDescriptor{}
	}
	if uint32(id) != variant {
		var found bool
		descriptor, found = itemDescriptorFromType(uint8(variant))
		if !found {
			r.UnknownEnumOption(variant, "stack request item descriptor type")
			return
		}
		x.Descriptor = descriptor
	}
	descriptor.Marshal(r)
	IntegerFunc(&x.Count, r.Uint16)
}

type ItemTagItemDescriptor struct {
	Tag string
}

func (x *ItemTagItemDescriptor) Marshal(r IO) {
	r.String(&x.Tag)
}
