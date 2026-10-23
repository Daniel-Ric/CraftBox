package protocol

import (
	"bytes"
	"fmt"
	"image/color"
	"io"
	"math/big"
	"reflect"
	"sort"
	"unsafe"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/internal"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

type Writer struct {
	w interface {
		io.Writer
		io.ByteWriter
	}
	shieldID int32
	buf      [8]byte
}

func NewWriter(w interface {
	io.Writer
	io.ByteWriter
}, shieldID int32) *Writer {
	writer := new(Writer)
	writer.Reset(w, shieldID)
	return writer
}

func (w *Writer) Reset(dst interface {
	io.Writer
	io.ByteWriter
}, shieldID int32) {
	w.w = dst
	w.shieldID = shieldID
}

func (w *Writer) Uint8(x *uint8) {
	_ = w.w.WriteByte(*x)
}

func (w *Writer) Int8(x *int8) {
	_ = w.w.WriteByte(byte(*x) & 0xff)
}

func (w *Writer) Bool(x *bool) {
	_ = w.w.WriteByte(*(*byte)(unsafe.Pointer(x)))
}

func (w *Writer) StringUTF(x *string) {
	l := int16(len(*x))
	w.Int16(&l)
	w.writeString(*x)
}

func (w *Writer) String(x *string) {
	l := uint32(len(*x))
	w.Varuint32(&l)
	w.writeString(*x)
}

type stringWriter interface {
	WriteString(string) (int, error)
}

func (w *Writer) writeString(s string) {
	if sw, ok := w.w.(stringWriter); ok {
		_, _ = sw.WriteString(s)
		return
	}
	_, _ = w.w.Write([]byte(s))
}

func (w *Writer) ByteSlice(x *[]byte) {
	l := uint32(len(*x))
	w.Varuint32(&l)
	_, _ = w.w.Write(*x)
}

func (w *Writer) Bytes(x *[]byte) {
	_, _ = w.w.Write(*x)
}

func (w *Writer) ByteFloat(x *float32) {
	_ = w.w.WriteByte(byte(*x / (360.0 / 256.0)))
}

func (w *Writer) Vec3(x *mgl32.Vec3) {
	w.Float32(&x[0])
	w.Float32(&x[1])
	w.Float32(&x[2])
}

func (w *Writer) Vec2(x *mgl32.Vec2) {
	w.Float32(&x[0])
	w.Float32(&x[1])
}

func (w *Writer) BlockPos(x *BlockPos) {
	w.Varint32(&x[0])
	w.Varint32(&x[1])
	w.Varint32(&x[2])
}

func (w *Writer) ChunkPos(x *ChunkPos) {
	w.Varint32(&x[0])
	w.Varint32(&x[1])
}

func (w *Writer) SubChunkPos(x *SubChunkPos) {
	w.Int32(&x[0])
	w.Int32(&x[1])
	w.Int32(&x[2])
}

func (w *Writer) SoundPos(x *mgl32.Vec3) {
	b := BlockPos{int32((*x)[0] * 8), int32((*x)[1] * 8), int32((*x)[2] * 8)}
	w.BlockPos(&b)
}

func (w *Writer) RGB(x *color.RGBA) {
	red := float32(x.R) / 255
	green := float32(x.G) / 255
	blue := float32(x.B) / 255
	w.Float32(&red)
	w.Float32(&green)
	w.Float32(&blue)
}

func (w *Writer) RGBA(x *color.RGBA) {
	val := uint32(x.R) | uint32(x.G)<<8 | uint32(x.B)<<16 | uint32(x.A)<<24
	w.Uint32(&val)
}

func (w *Writer) BEARGB(x *color.RGBA) {
	val := int32(x.A) | int32(x.R)<<8 | int32(x.G)<<16 | int32(x.B)<<24
	w.BEInt32(&val)
}

func (w *Writer) UUID(x *uuid.UUID) {
	b := append((*x)[8:], (*x)[:8]...)
	for i, j := 0, 15; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	_, _ = w.w.Write(b)
}

func (w *Writer) PlayerInventoryAction(x *UseItemTransactionData) {
	w.Varint32(&x.LegacyRequestID)
	OptionalFunc(w, &x.LegacySetItemSlots, func(slots *[]LegacySetItemSlot) {
		Slice(w, slots)
	})
	Slice(w, &x.Actions)
	IntegerFunc(&x.ActionType, w.Varint32)
	IntegerFunc(&x.TriggerType, w.Uint8)
	w.BlockPos(&x.BlockPosition)
	IntegerFunc(&x.BlockFace, w.Uint8)
	w.Varint32(&x.HotBarSlot)
	w.Uint8(&x.Hand)
	w.ItemInstance(&x.HeldItem)
	w.Vec3(&x.Position)
	w.Vec3(&x.ClickedPosition)
	w.Varuint32(&x.BlockRuntimeID)
	w.Uint8(&x.ClientPrediction)
	w.Uint8(&x.ClientCooldownState)
}

func (w *Writer) GameRule(x *GameRule) {
	w.String(&x.Name)
	w.Bool(&x.CanBeModifiedByPlayer)

	switch v := x.Value.(type) {
	case nil:
		id := uint32(0)
		w.Varuint32(&id)
	case bool:
		id := uint32(1)
		w.Varuint32(&id)
		w.Bool(&v)
	case uint32:
		id := uint32(2)
		w.Varuint32(&id)
		w.Uint32(&v)
	case float32:
		id := uint32(3)
		w.Varuint32(&id)
		w.Float32(&v)
	default:
		w.UnknownEnumOption(fmt.Sprintf("%T", v), "game rule type")
	}
}

func (w *Writer) EntityMetadata(x *EntityMetadata) {
	l := uint32(len(*x))
	w.Varuint32(&l)

	keys := make([]int, 0, l)
	for k := range *x {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	for _, k := range keys {
		key := uint32(k)
		value := (*x)[uint32(k)]
		w.Varuint32(&key)
		writeType := func(dataType uint32) {
			w.Varuint32(&dataType)
			legacyDataType := byte(dataType)
			w.Uint8(&legacyDataType)
		}
		switch v := value.(type) {
		case byte:
			writeType(EntityDataTypeByte)
			w.Uint8(&v)
		case int16:
			writeType(EntityDataTypeInt16)
			w.Int16(&v)
		case int32:
			writeType(EntityDataTypeInt32)
			w.Varint32(&v)
		case float32:
			writeType(EntityDataTypeFloat32)
			w.Float32(&v)
		case string:
			writeType(EntityDataTypeString)
			w.String(&v)
		case map[string]any:
			writeType(EntityDataTypeCompoundTag)
			w.NBT(&v, nbt.NetworkLittleEndian)
		case BlockPos:
			writeType(EntityDataTypeBlockPos)
			w.BlockPos(&v)
		case int64:
			writeType(EntityDataTypeInt64)
			w.Varint64(&v)
		case mgl32.Vec3:
			writeType(EntityDataTypeVec3)
			w.Vec3(&v)
		default:
			w.UnknownEnumOption(reflect.TypeOf(value), "entity metadata")
		}
	}
}

func (w *Writer) ItemDescriptorCount(i *ItemDescriptorCount) {
	id, name, ok := itemDescriptorType(i.Descriptor)
	if !ok {
		w.UnknownEnumOption(fmt.Sprintf("%T", i.Descriptor), "item descriptor type")
		return
	}
	variant := uint32(id)
	if variant > ItemDescriptorDefault {
		variant = ItemDescriptorDefault
	}
	w.Varuint32(&variant)
	if id == ItemDescriptorInvalid {
		aux := int32(32767)
		w.Varint32(&aux)
	} else {
		w.String(&name)
		i.Descriptor.Marshal(w)
		if id == ItemDescriptorItemTag {
			aux := int32(32767)
			w.Varint32(&aux)
		}
	}
	w.Varint32(&i.Count)
}

func (w *Writer) ItemInstance(i *ItemInstance) {
	x := &i.Stack
	id := int16(x.NetworkID)
	w.Int16(&id)

	w.Uint16(&x.Count)
	w.Varuint32(&x.MetadataValue)

	hasNetID := i.StackNetworkID != 0
	w.Bool(&hasNetID)

	if hasNetID {
		w.Varint32(&i.StackNetworkID)
	}

	IntegerFunc(&x.BlockRuntimeID, w.Varuint32)
	w.itemUserData(itemStackUserData(x), x.NetworkID != 0, x.NetworkID == w.shieldID)
}

func (w *Writer) Item(x *ItemStack) {
	w.Varint32(&x.NetworkID)

	w.Uint16(&x.Count)
	w.Varuint32(&x.MetadataValue)
	w.Varint32(&x.BlockRuntimeID)
	w.itemUserData(itemStackUserData(x), x.NetworkID != 0, x.NetworkID == w.shieldID)
}

func (w *Writer) StackRequestItem(x *StackRequestItem) {
	hasItem := x.Identifier != ""
	variant := uint32(0)
	if hasItem {
		variant = ItemDescriptorDefault
	}
	w.Varuint32(&variant)
	legacyVariant := uint8(variant)
	w.Uint8(&legacyVariant)
	if hasItem {
		w.String(&x.Identifier)
		IntegerFunc(&x.MetadataValue, w.Varint32)
	}
	IntegerFunc(&x.Count, w.Int16)
	IntegerFunc(&x.BlockRuntimeID, w.Varuint32)
	w.itemUserData(stackRequestItemUserData(x), hasItem, x.Identifier == "minecraft:shield")
}

func (w *Writer) itemUserData(x itemUserData, present, shield bool) {
	if !present {
		var zero uint32
		w.Varuint32(&zero)
		return
	}

	buf := internal.BufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer func() {
		buf.Reset()
		internal.BufferPool.Put(buf)
	}()
	bufWriter := NewWriter(buf, w.shieldID)
	var length int16
	if len(x.nbtData) != 0 {
		length = -1
		version := uint8(1)
		bufWriter.Int16(&length)
		bufWriter.Uint8(&version)
		bufWriter.NBT(&x.nbtData, nbt.LittleEndian)
	} else {
		bufWriter.Int16(&length)
	}
	FuncSliceUint32Length(bufWriter, &x.canBePlacedOn, bufWriter.StringUTF)
	FuncSliceUint32Length(bufWriter, &x.canBreak, bufWriter.StringUTF)
	if shield {
		bufWriter.Int64(&x.blockingTick)
	}
	extraData := buf.Bytes()
	w.ByteSlice(&extraData)
}

func (w *Writer) StackRequestAction(x *StackRequestAction) {
	var id byte
	if !lookupStackRequestActionType(*x, &id) {
		w.UnknownEnumOption(fmt.Sprintf("%T", *x), "stack request action type")
	}
	variant := stackRequestActionVariant(id)
	w.Varuint32(&variant)
	w.Uint8(&id)
	(*x).Marshal(w)
}

func (w *Writer) MaterialReducer(m *MaterialReducer) {
	mix := (m.InputItem.NetworkID << 16) | int32(m.InputItem.MetadataValue)
	w.Varint32(&mix)
	Slice(w, &m.Outputs)
}

func (w *Writer) EventType(x *Event) {
	var t int32
	if !lookupEventType(*x, &t) {
		w.UnknownEnumOption(*x, "event packet event type")
	}
	w.Varint32(&t)
}

func (w *Writer) EventOrdinal(x *Event) {
	var ordinal uint32
	if !lookupEventOrdinal(*x, &ordinal) {
		w.UnknownEnumOption(*x, "event packet event ordinal")
		return
	}
	w.Varuint32(&ordinal)
}

func (w *Writer) TransactionDataType(x *InventoryTransactionData) {
	var id uint32
	if !lookupTransactionDataType(*x, &id) {
		w.UnknownEnumOption(fmt.Sprintf("%T", x), "inventory transaction data type")
	}
	w.Varuint32(&id)
}

func (w *Writer) AbilityValue(x *any) {
	switch val := (*x).(type) {
	case bool:
		valType, defaultVal := uint8(1), float32(0)
		w.Uint8(&valType)
		w.Bool(&val)
		w.Float32(&defaultVal)
	case float32:
		valType, defaultVal := uint8(2), false
		w.Uint8(&valType)
		w.Bool(&defaultVal)
		w.Float32(&val)
	default:
		w.InvalidValue(*x, "ability value type", "must be bool or float32")
	}
}

var varintMaxByteValue = big.NewInt(0x80)

func (w *Writer) Bitset(x *Bitset, size int) {
	if x.size != size {
		w.panicf("bitset size mismatch: expected %v, got %v", size, x.size)
	}
	u := new(big.Int)
	u.Set(x.int)

	if len(u.Bits()) == 0 {
		_ = w.w.WriteByte(0)
		return
	}

	for u.Cmp(varintMaxByteValue) >= 0 {
		_ = w.w.WriteByte(byte(u.Bits()[0]) | 0x80)
		u.Rsh(u, 7)
	}
	_ = w.w.WriteByte(byte(u.Bits()[0]))
}

func (w *Writer) PackSetting(x *PackSetting) {
	w.String(&x.Name)
	var id uint32
	switch val := x.Value.(type) {
	case float32:
		id = PackSettingTypeFloat
		w.Varuint32(&id)
		w.Float32(&val)
	case bool:
		id = PackSettingTypeBool
		w.Varuint32(&id)
		w.Bool(&val)
	case string:
		id = PackSettingTypeString
		w.Varuint32(&id)
		w.String(&val)
	case []string:
		id = PackSettingTypeStringList
		w.Varuint32(&id)
		FuncSlice(w, &val, w.String)
	default:
		w.UnknownEnumOption(x.Value, "pack setting")
	}
}

func (w *Writer) ShapeData(x *ShapeData) {
	var shapeDataType uint32
	if !lookupShapeDataType(*x, &shapeDataType) {
		w.UnknownEnumOption(fmt.Sprintf("%T", *x), "debug shape data type")
	}
	w.Varuint32(&shapeDataType)
	(*x).Marshal(w)
}

func (w *Writer) Varint64(x *int64) {
	u := *x
	ux := uint64(u) << 1
	if u < 0 {
		ux = ^ux
	}
	for ux >= 0x80 {
		_ = w.w.WriteByte(byte(ux) | 0x80)
		ux >>= 7
	}
	_ = w.w.WriteByte(byte(ux))
}

func (w *Writer) ActorRuntimeID(x *uint64) {
	w.Varuint64(x)
}

func (w *Writer) ActorRuntimeIDVarint64(x *int64) {
	w.Varint64(x)
}

func (w *Writer) ActorRuntimeIDVaruint32(x *uint32) {
	w.Varuint32(x)
}

func (w *Writer) ActorUniqueID(x *int64) {
	w.Varint64(x)
}

func (w *Writer) ActorUniqueIDInt64(x *int64) {
	w.Int64(x)
}

func (w *Writer) ActorUniqueIDUint64(x *uint64) {
	w.Uint64(x)
}

func (w *Writer) ActorUniqueIDVaruint64(x *uint64) {
	w.Varuint64(x)
}

func (w *Writer) Varuint64(x *uint64) {
	u := *x
	for u >= 0x80 {
		_ = w.w.WriteByte(byte(u) | 0x80)
		u >>= 7
	}
	_ = w.w.WriteByte(byte(u))
}

func (w *Writer) Varint32(x *int32) {
	u := *x
	ux := uint32(u) << 1
	if u < 0 {
		ux = ^ux
	}
	for ux >= 0x80 {
		_ = w.w.WriteByte(byte(ux) | 0x80)
		ux >>= 7
	}
	_ = w.w.WriteByte(byte(ux))
}

func (w *Writer) Varuint32(x *uint32) {
	u := *x
	for u >= 0x80 {
		_ = w.w.WriteByte(byte(u) | 0x80)
		u >>= 7
	}
	_ = w.w.WriteByte(byte(u))
}

func (w *Writer) NBT(x *map[string]any, encoding nbt.Encoding) {
	if err := nbt.NewEncoderWithEncoding(w.w, encoding).Encode(*x); err != nil {
		panic(err)
	}
}

func (w *Writer) NBTList(x *[]any, encoding nbt.Encoding) {
	if err := nbt.NewEncoderWithEncoding(w.w, encoding).Encode(*x); err != nil {
		panic(err)
	}
}

func (w *Writer) ShieldID() int32 {
	return w.shieldID
}

func (w *Writer) UnknownEnumOption(value any, enum string) {
	w.panicf("unknown value '%#v' for enum type '%v'", value, enum)
}

func (w *Writer) InvalidValue(value any, forField, reason string) {
	w.panicf("invalid value '%v' for %v: %v", value, forField, reason)
}

func (w *Writer) panicf(format string, a ...any) {
	panic(fmt.Errorf(format, a...))
}
