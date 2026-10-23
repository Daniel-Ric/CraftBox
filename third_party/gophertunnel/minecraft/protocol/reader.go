package protocol

import (
	"bytes"
	"errors"
	"fmt"
	"image/color"
	"io"
	"math"
	"math/big"
	"math/bits"
	"unsafe"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

type Reader struct {
	r interface {
		io.Reader
		io.ByteReader
	}
	shieldID      int32
	limitsEnabled bool
	buf           [8]byte
}

func NewReader(r interface {
	io.Reader
	io.ByteReader
}, shieldID int32, enableLimits bool) *Reader {
	return &Reader{r: r, shieldID: shieldID, limitsEnabled: enableLimits}
}

func (r *Reader) Uint8(x *uint8) {
	var err error
	*x, err = r.r.ReadByte()
	if err != nil {
		r.panic(err)
	}
}

func (r *Reader) Int8(x *int8) {
	var b uint8
	r.Uint8(&b)
	*x = int8(b)
}

func (r *Reader) Bool(x *bool) {
	u, err := r.r.ReadByte()
	if err != nil {
		r.panic(err)
	}
	*x = *(*bool)(unsafe.Pointer(&u))
}

var errStringTooLong = errors.New("string length overflows a 32-bit integer")

func (r *Reader) StringUTF(x *string) {
	var length int16
	r.Int16(&length)
	l := int(length)
	if l > math.MaxInt16 {
		r.panic(errStringTooLong)
	}
	r.checkRemaining(l, "string")
	data := make([]byte, l)
	if _, err := io.ReadFull(r.r, data); err != nil {
		r.panic(err)
	}
	*x = *(*string)(unsafe.Pointer(&data))
}

func (r *Reader) String(x *string) {
	var length uint32
	r.Varuint32(&length)
	l := int(length)
	if l > math.MaxInt32 {
		r.panic(errStringTooLong)
	}
	r.checkRemaining(l, "string")
	data := make([]byte, l)
	if _, err := io.ReadFull(r.r, data); err != nil {
		r.panic(err)
	}
	*x = *(*string)(unsafe.Pointer(&data))
}

func (r *Reader) ByteSlice(x *[]byte) {
	var length uint32
	r.Varuint32(&length)
	l := int(length)
	if l > math.MaxInt32 {
		r.panic(errStringTooLong)
	}
	r.checkRemaining(l, "byte slice")
	data := make([]byte, l)
	if _, err := io.ReadFull(r.r, data); err != nil {
		r.panic(err)
	}
	*x = data
}

func (r *Reader) Vec3(x *mgl32.Vec3) {
	r.Float32(&x[0])
	r.Float32(&x[1])
	r.Float32(&x[2])
}

func (r *Reader) Vec2(x *mgl32.Vec2) {
	r.Float32(&x[0])
	r.Float32(&x[1])
}

func (r *Reader) BlockPos(x *BlockPos) {
	r.Varint32(&x[0])
	r.Varint32(&x[1])
	r.Varint32(&x[2])
}

func (r *Reader) ChunkPos(x *ChunkPos) {
	r.Varint32(&x[0])
	r.Varint32(&x[1])
}

func (r *Reader) SubChunkPos(x *SubChunkPos) {
	r.Int32(&x[0])
	r.Int32(&x[1])
	r.Int32(&x[2])
}

func (r *Reader) SoundPos(x *mgl32.Vec3) {
	var b BlockPos
	r.BlockPos(&b)
	*x = mgl32.Vec3{float32(b[0]) / 8, float32(b[1]) / 8, float32(b[2]) / 8}
}

func (r *Reader) ByteFloat(x *float32) {
	var v uint8
	r.Uint8(&v)
	*x = float32(v) * (360.0 / 256.0)
}

func (r *Reader) RGB(x *color.RGBA) {
	var red, green, blue float32
	r.Float32(&red)
	r.Float32(&green)
	r.Float32(&blue)
	*x = color.RGBA{
		R: uint8(red * 255),
		G: uint8(green * 255),
		B: uint8(blue * 255),
	}
}

func (r *Reader) RGBA(x *color.RGBA) {
	var v uint32
	r.Uint32(&v)
	*x = color.RGBA{
		R: byte(v),
		G: byte(v >> 8),
		B: byte(v >> 16),
		A: byte(v >> 24),
	}
}

func (r *Reader) BEARGB(x *color.RGBA) {
	var v int32
	r.BEInt32(&v)
	*x = color.RGBA{
		A: byte(v),
		R: byte(v >> 8),
		G: byte(v >> 16),
		B: byte(v >> 24),
	}
}

func (r *Reader) Bytes(p *[]byte) {
	var err error
	*p, err = io.ReadAll(r.r)
	if err != nil {
		r.panic(err)
	}
}

func (r *Reader) NBT(m *map[string]any, encoding nbt.Encoding) {
	dec := nbt.NewDecoderWithEncoding(r.r, encoding)
	dec.AllowZero = true

	if err := dec.Decode(m); err != nil {
		r.panic(err)
	}
}

func (r *Reader) NBTList(m *[]any, encoding nbt.Encoding) {
	if err := nbt.NewDecoderWithEncoding(r.r, encoding).Decode(m); err != nil {
		r.panic(err)
	}
}

func (r *Reader) UUID(x *uuid.UUID) {
	var b [16]byte
	if _, err := io.ReadFull(r.r, b[:]); err != nil {
		r.panic(err)
	}

	for i, j := 0, 7; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	for i, j := 8, 15; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	*x = b
}

func (r *Reader) PlayerInventoryAction(x *UseItemTransactionData) {
	r.Varint32(&x.LegacyRequestID)
	OptionalFunc(r, &x.LegacySetItemSlots, func(slots *[]LegacySetItemSlot) {
		Slice(r, slots)
	})
	Slice(r, &x.Actions)
	IntegerFunc(&x.ActionType, r.Varint32)
	IntegerFunc(&x.TriggerType, r.Uint8)
	r.BlockPos(&x.BlockPosition)
	IntegerFunc(&x.BlockFace, r.Uint8)
	r.Varint32(&x.HotBarSlot)
	r.Uint8(&x.Hand)
	r.ItemInstance(&x.HeldItem)
	r.Vec3(&x.Position)
	r.Vec3(&x.ClickedPosition)
	r.Varuint32(&x.BlockRuntimeID)
	r.Uint8(&x.ClientPrediction)
	r.Uint8(&x.ClientCooldownState)
}

func (r *Reader) GameRule(x *GameRule) {
	r.String(&x.Name)
	r.Bool(&x.CanBeModifiedByPlayer)
	var t uint32
	r.Varuint32(&t)

	switch t {
	case 0:
		x.Value = nil
	case 1:
		var v bool
		r.Bool(&v)
		x.Value = v
	case 2:
		var v uint32
		r.Uint32(&v)
		x.Value = v
	case 3:
		var v float32
		r.Float32(&v)
		x.Value = v
	default:
		r.UnknownEnumOption(t, "game rule type")
	}
}

func (r *Reader) EntityMetadata(x *EntityMetadata) {
	*x = EntityMetadata{}

	var count uint32
	r.Varuint32(&count)
	for i := uint32(0); i < count; i++ {
		var key, dataType uint32
		r.Varuint32(&key)
		r.Varuint32(&dataType)
		var legacyDataType byte
		r.Uint8(&legacyDataType)
		if dataType != uint32(legacyDataType) {
			r.InvalidValue(legacyDataType, "entity metadata type", fmt.Sprintf("does not match cereal selector %d", dataType))
			return
		}
		switch dataType {
		case EntityDataTypeByte:
			var v byte
			r.Uint8(&v)
			(*x)[key] = v
		case EntityDataTypeInt16:
			var v int16
			r.Int16(&v)
			(*x)[key] = v
		case EntityDataTypeInt32:
			var v int32
			r.Varint32(&v)
			(*x)[key] = v
		case EntityDataTypeFloat32:
			var v float32
			r.Float32(&v)
			(*x)[key] = v
		case EntityDataTypeString:
			var v string
			r.String(&v)
			(*x)[key] = v
		case EntityDataTypeCompoundTag:
			var v map[string]any
			r.NBT(&v, nbt.NetworkLittleEndian)
			(*x)[key] = v
		case EntityDataTypeBlockPos:
			var v BlockPos
			r.BlockPos(&v)
			(*x)[key] = v
		case EntityDataTypeInt64:
			var v int64
			r.Varint64(&v)
			(*x)[key] = v
		case EntityDataTypeVec3:
			var v mgl32.Vec3
			r.Vec3(&v)
			(*x)[key] = v
		default:
			r.UnknownEnumOption(dataType, "entity metadata")
		}
	}
}

func (r *Reader) ItemDescriptorCount(i *ItemDescriptorCount) {
	var variant uint32
	r.Varuint32(&variant)
	if variant == ItemDescriptorInvalid {
		i.Descriptor = &InvalidItemDescriptor{}
		var aux int32
		r.Varint32(&aux)
	} else {
		if variant != ItemDescriptorDefault {
			r.UnknownEnumOption(variant, "item descriptor variant")
			return
		}
		var name string
		r.String(&name)
		var id uint8
		switch name {
		case "name":
			id = ItemDescriptorDefault
		case "molang":
			id = ItemDescriptorMoLang
		case "item_tag":
			id = ItemDescriptorItemTag
		default:
			r.UnknownEnumOption(name, "item descriptor type")
			return
		}
		i.Descriptor, _ = itemDescriptorFromType(id)
		i.Descriptor.Marshal(r)
		if id == ItemDescriptorItemTag {
			var aux int32
			r.Varint32(&aux)
		}
	}
	r.Varint32(&i.Count)
}

func (r *Reader) ItemInstance(i *ItemInstance) {
	x := &i.Stack
	var id int16
	r.Int16(&id)
	x.NetworkID = int32(id)

	r.Uint16(&x.Count)
	r.Varuint32(&x.MetadataValue)

	var hasNetID bool
	r.Bool(&hasNetID)

	if hasNetID {
		r.Varint32(&i.StackNetworkID)
	} else {
		i.StackNetworkID = 0
	}

	IntegerFunc(&x.BlockRuntimeID, r.Varuint32)
	data := r.itemUserData(x.NetworkID == r.shieldID)
	x.NBTData, x.CanBePlacedOn, x.CanBreak, x.BlockingTick = data.nbtData, data.canBePlacedOn, data.canBreak, data.blockingTick
}

func (r *Reader) Item(x *ItemStack) {
	r.Varint32(&x.NetworkID)

	r.Uint16(&x.Count)
	r.Varuint32(&x.MetadataValue)
	r.Varint32(&x.BlockRuntimeID)
	data := r.itemUserData(x.NetworkID == r.shieldID)
	x.NBTData, x.CanBePlacedOn, x.CanBreak, x.BlockingTick = data.nbtData, data.canBePlacedOn, data.canBreak, data.blockingTick
}

func (r *Reader) StackRequestItem(x *StackRequestItem) {
	var variant uint32
	r.Varuint32(&variant)
	var legacyVariant uint8
	r.Uint8(&legacyVariant)
	hasItem := variant == ItemDescriptorDefault
	if variant != ItemDescriptorInvalid && !hasItem {
		r.UnknownEnumOption(variant, "stack request item descriptor")
		return
	}
	if hasItem {
		r.String(&x.Identifier)
		IntegerFunc(&x.MetadataValue, r.Varint32)
	} else {
		x.Identifier = ""
		x.MetadataValue = 0
	}
	IntegerFunc(&x.Count, r.Int16)
	IntegerFunc(&x.BlockRuntimeID, r.Varuint32)
	data := r.itemUserData(x.Identifier == "minecraft:shield")
	x.NBTData, x.CanBePlacedOn, x.CanBreak, x.BlockingTick = data.nbtData, data.canBePlacedOn, data.canBreak, data.blockingTick
}

func (r *Reader) itemUserData(shield bool) itemUserData {
	var x itemUserData
	var extraData []byte
	r.ByteSlice(&extraData)
	if len(extraData) == 0 {
		return x
	}
	buf := bytes.NewBuffer(extraData)
	bufReader := NewReader(buf, r.shieldID, r.limitsEnabled)
	var length int16
	bufReader.Int16(&length)
	switch length {
	case 0:
		x.nbtData = nil
	case -1:
		var version uint8
		bufReader.Uint8(&version)
		if version != 1 {
			bufReader.UnknownEnumOption(version, "item user data version")
			return x
		}
		bufReader.NBT(&x.nbtData, nbt.LittleEndian)
	default:
		bufReader.NBT(&x.nbtData, nbt.LittleEndian)
	}
	FuncSliceUint32Length(bufReader, &x.canBePlacedOn, bufReader.StringUTF)
	FuncSliceUint32Length(bufReader, &x.canBreak, bufReader.StringUTF)
	if shield {
		bufReader.Int64(&x.blockingTick)
	}
	return x
}

func (r *Reader) StackRequestAction(x *StackRequestAction) {
	var variant uint32
	r.Varuint32(&variant)
	var legacyID uint8
	r.Uint8(&legacyID)
	if variant > stackRequestActionVariant(StackRequestActionCraftResultsDeprecated) {
		r.UnknownEnumOption(variant, "stack request action variant")
		return
	}
	id := uint8(variant)
	if variant >= uint32(StackRequestActionPlaceInContainer) {
		id += 2
	}
	if !lookupStackRequestAction(id, x) {
		r.UnknownEnumOption(id, "stack request action type")
		return
	}
	(*x).Marshal(r)
}

func (r *Reader) MaterialReducer(m *MaterialReducer) {
	var mix int32
	r.Varint32(&mix)
	m.InputItem = ItemType{NetworkID: mix << 16, MetadataValue: uint32(mix & 0x7fff)}
	Slice(r, &m.Outputs)
}

func (r *Reader) EventType(x *Event) {
	var t int32
	r.Varint32(&t)
	if !lookupEvent(t, x) {
		r.UnknownEnumOption(t, "event packet event type")
	}
}

func (r *Reader) EventOrdinal(x *Event) {
	var ordinal uint32
	if !lookupEventOrdinal(*x, &ordinal) {
		r.UnknownEnumOption(*x, "event packet event ordinal")
		return
	}
	r.Varuint32(&ordinal)
}

func (r *Reader) TransactionDataType(x *InventoryTransactionData) {
	var transactionType uint32
	r.Varuint32(&transactionType)
	if !lookupTransactionData(transactionType, x) {
		r.UnknownEnumOption(transactionType, "inventory transaction data type")
	}
}

func (r *Reader) AbilityValue(x *any) {
	valType, boolVal, floatVal := uint8(0), false, float32(0)
	r.Uint8(&valType)
	r.Bool(&boolVal)
	r.Float32(&floatVal)
	switch valType {
	case 1:
		*x = boolVal
	case 2:
		*x = floatVal
	default:
		r.InvalidValue(valType, "ability value type", "must be bool or float32")
	}
}

func (r *Reader) Bitset(x *Bitset, size int) {
	*x = NewBitset(size)
	for i := 0; i < size; i += 7 {
		b, err := r.r.ReadByte()
		if err != nil {
			r.panic(err)
		} else if i+bits.Len8(b) > size {
			r.panic(errBitsetOverflow)
		}

		bi := big.NewInt(int64(b & 0x7f))
		x.int.Or(x.int, bi.Lsh(bi, uint(i)))
		if b&0x80 == 0 {
			return
		}
	}

	r.panic(errBitsetOverflow)
}

func (r *Reader) PackSetting(x *PackSetting) {
	r.String(&x.Name)
	var t uint32
	r.Varuint32(&t)
	switch t {
	case PackSettingTypeFloat:
		var v float32
		r.Float32(&v)
		x.Value = v
	case PackSettingTypeBool:
		var v bool
		r.Bool(&v)
		x.Value = v
	case PackSettingTypeString:
		var v string
		r.String(&v)
		x.Value = v
	case PackSettingTypeStringList:
		var v []string
		FuncSlice(r, &v, r.String)
		x.Value = v
	default:
		r.UnknownEnumOption(t, "pack setting")
	}
}

func (r *Reader) ShapeData(x *ShapeData) {
	var shapeDataType uint32
	r.Varuint32(&shapeDataType)
	if !lookupShapeData(shapeDataType, x) {
		r.UnknownEnumOption(shapeDataType, "debug shape data type")
		return
	}
	(*x).Marshal(r)
}

func (r *Reader) SliceLength(value uint32, max uint32) {
	if value > max && r.limitsEnabled {
		r.panicf("slice length was too long: length of %v (max %v)", value, max)
	}
	if remaining, ok := r.r.(interface{ Len() int }); ok && uint64(value) > uint64(remaining.Len()) {
		r.panicf("slice length %v exceeds remaining packet payload %v", value, remaining.Len())
	}
}

func (r *Reader) checkRemaining(length int, field string) {
	if length < 0 {
		r.panicf("%s length was negative: %v", field, length)
	}
	if remaining, ok := r.r.(interface{ Len() int }); ok && length > remaining.Len() {
		r.panicf("%s length %v exceeds remaining packet payload %v", field, length, remaining.Len())
	}
}

func (r *Reader) ShieldID() int32 {
	return r.shieldID
}

func (r *Reader) UnknownEnumOption(value any, enum string) {
	r.panicf("unknown value '%#v' for enum type '%v'", value, enum)
}

func (r *Reader) InvalidValue(value any, forField, reason string) {
	r.panicf("invalid value '%v' for %v: %v", value, forField, reason)
}

var errVarIntOverflow = errors.New("varint overflows integer")
var errBitsetOverflow = errors.New("bitset overflows size")

func (r *Reader) Varint64(x *int64) {
	var ux uint64
	for i := 0; i < 70; i += 7 {
		b, err := r.r.ReadByte()
		if err != nil {
			r.panic(err)
		}

		ux |= uint64(b&0x7f) << i
		if b&0x80 == 0 {
			*x = int64(ux >> 1)
			if ux&1 != 0 {
				*x = ^*x
			}
			return
		}
	}
	r.panic(errVarIntOverflow)
}

func (r *Reader) ActorRuntimeID(x *uint64) {
	r.Varuint64(x)
}

func (r *Reader) ActorRuntimeIDVarint64(x *int64) {
	r.Varint64(x)
}

func (r *Reader) ActorRuntimeIDVaruint32(x *uint32) {
	r.Varuint32(x)
}

func (r *Reader) ActorUniqueID(x *int64) {
	r.Varint64(x)
}

func (r *Reader) ActorUniqueIDInt64(x *int64) {
	r.Int64(x)
}

func (r *Reader) ActorUniqueIDUint64(x *uint64) {
	r.Uint64(x)
}

func (r *Reader) ActorUniqueIDVaruint64(x *uint64) {
	r.Varuint64(x)
}

func (r *Reader) Varuint64(x *uint64) {
	var v uint64
	for i := 0; i < 70; i += 7 {
		b, err := r.r.ReadByte()
		if err != nil {
			r.panic(err)
		}

		v |= uint64(b&0x7f) << i
		if b&0x80 == 0 {
			*x = v
			return
		}
	}
	r.panic(errVarIntOverflow)
}

func (r *Reader) Varint32(x *int32) {
	var ux uint32
	for i := 0; i < 35; i += 7 {
		b, err := r.r.ReadByte()
		if err != nil {
			r.panic(err)
		}

		ux |= uint32(b&0x7f) << i
		if b&0x80 == 0 {
			*x = int32(ux >> 1)
			if ux&1 != 0 {
				*x = ^*x
			}
			return
		}
	}
	r.panic(errVarIntOverflow)
}

func (r *Reader) Varuint32(x *uint32) {
	var v uint32
	for i := 0; i < 35; i += 7 {
		b, err := r.r.ReadByte()
		if err != nil {
			r.panic(err)
		}

		v |= uint32(b&0x7f) << i
		if b&0x80 == 0 {
			*x = v
			return
		}
	}
	r.panic(errVarIntOverflow)
}

func (r *Reader) panicf(format string, a ...any) {
	panic(fmt.Errorf(format, a...))
}

func (r *Reader) panic(err error) {
	panic(err)
}
