package protocol

import (
	"image/color"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"golang.org/x/exp/constraints"
)

type IO interface {
	Uint16(x *uint16)
	Int16(x *int16)
	Uint32(x *uint32)
	Int32(x *int32)
	BEInt32(x *int32)
	Uint64(x *uint64)
	Int64(x *int64)
	Float32(x *float32)
	Float64(x *float64)
	Uint8(x *uint8)
	Int8(x *int8)
	Bool(x *bool)
	Varint64(x *int64)
	Varuint64(x *uint64)
	Varint32(x *int32)
	Varuint32(x *uint32)

	ActorRuntimeID(x *uint64)
	ActorRuntimeIDVarint64(x *int64)
	ActorRuntimeIDVaruint32(x *uint32)
	ActorUniqueID(x *int64)
	ActorUniqueIDInt64(x *int64)
	ActorUniqueIDUint64(x *uint64)
	ActorUniqueIDVaruint64(x *uint64)
	String(x *string)
	StringUTF(x *string)
	ByteSlice(x *[]byte)
	Vec3(x *mgl32.Vec3)
	Vec2(x *mgl32.Vec2)
	BlockPos(x *BlockPos)
	ChunkPos(x *ChunkPos)
	SubChunkPos(x *SubChunkPos)
	SoundPos(x *mgl32.Vec3)
	ByteFloat(x *float32)
	Bytes(p *[]byte)
	NBT(m *map[string]any, encoding nbt.Encoding)
	NBTList(m *[]any, encoding nbt.Encoding)
	UUID(x *uuid.UUID)
	RGB(x *color.RGBA)
	RGBA(x *color.RGBA)
	BEARGB(x *color.RGBA)
	EntityMetadata(x *EntityMetadata)
	Item(x *ItemStack)
	ItemInstance(i *ItemInstance)
	StackRequestItem(x *StackRequestItem)
	ItemDescriptorCount(i *ItemDescriptorCount)
	StackRequestAction(x *StackRequestAction)
	MaterialReducer(x *MaterialReducer)
	EventType(x *Event)
	EventOrdinal(x *Event)
	TransactionDataType(x *InventoryTransactionData)
	PlayerInventoryAction(x *UseItemTransactionData)
	GameRule(x *GameRule)
	AbilityValue(x *any)
	Bitset(x *Bitset, size int)
	PackSetting(x *PackSetting)
	ShapeData(x *ShapeData)

	ShieldID() int32
	UnknownEnumOption(value any, enum string)
	InvalidValue(value any, forField, reason string)
}

type Marshaler interface {
	Marshal(r IO)
}

func Slice[T any, S ~*[]T, A PtrMarshaler[T]](r IO, x S) {
	count := uint32(len(*x))
	r.Varuint32(&count)
	SliceOfLen[T, S, A](r, count, x)
}

func SliceUint8Length[T any, S *[]T, A PtrMarshaler[T]](r IO, x S) {
	count := uint8(len(*x))
	r.Uint8(&count)
	SliceOfLen[T, S, A](r, uint32(count), x)
}

func SliceUint32Length[T any, S ~*[]T, A PtrMarshaler[T]](r IO, x S) {
	count := uint32(len(*x))
	r.Uint32(&count)
	SliceOfLen[T, S, A](r, count, x)
}

func FuncSliceUint32Length[T any, S ~*[]T](r IO, x S, f func(*T)) {
	count := uint32(len(*x))
	r.Uint32(&count)
	FuncSliceOfLen(r, count, x, f)
}

func FuncSlice[T any, S ~*[]T](r IO, x S, f func(*T)) {
	count := uint32(len(*x))
	r.Varuint32(&count)
	FuncSliceOfLen(r, count, x, f)
}

func FuncIOSlice[T any, S ~*[]T](r IO, x S, f func(IO, *T)) {
	FuncSlice(r, x, func(v *T) {
		f(r, v)
	})
}

const maxSliceLength = 1024

func SliceOfLen[T any, S ~*[]T, A PtrMarshaler[T]](r IO, l uint32, x S) {
	limit, ok := r.(sliceReader)
	if ok {
		limit.SliceLength(l, maxSliceLength)
		*x = make([]T, l)
	}

	for i := uint32(0); i < l; i++ {
		A(&(*x)[i]).Marshal(r)
	}
}

func FuncSliceOfLen[T any, S ~*[]T](r IO, l uint32, x S, f func(*T)) {
	limit, ok := r.(sliceReader)
	if ok {
		limit.SliceLength(l, maxSliceLength)
		*x = make([]T, l)
	}

	for i := uint32(0); i < l; i++ {
		f(&(*x)[i])
	}
}

type sliceReader interface {
	SliceLength(value uint32, max uint32)
}

func FuncIOSliceOfLen[T any, S ~*[]T](r IO, l uint32, x S, f func(IO, *T)) {
	FuncSliceOfLen(r, l, x, func(v *T) {
		f(r, v)
	})
}

type PtrMarshaler[T any] interface {
	Marshaler
	*T
}

func Single[T any, S PtrMarshaler[T]](r IO, x S) {
	x.Marshal(r)
}

func IntegerFunc[S, W constraints.Integer](x *S, f func(*W)) {
	w := W(*x)
	f(&w)
	*x = S(w)
}

type Optional[T any] struct {
	set bool
	val T
}

func Option[T any](val T) Optional[T] {
	return Optional[T]{set: true, val: val}
}

func (o Optional[T]) Value() (T, bool) {
	return o.val, o.set
}

func OptionalFunc[T any](r IO, x *Optional[T], f func(*T)) any {
	r.Bool(&x.set)
	if x.set {
		f(&x.val)
	}
	return x
}

func OptionalMarshaler[T any, A PtrMarshaler[T]](r IO, x *Optional[T]) {
	r.Bool(&x.set)
	if x.set {
		A(&x.val).Marshal(r)
	}
}
