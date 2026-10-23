package protocol

import "slices"

type InputFlags struct {
	set  bool
	size int
	ids  []int32
}

func NewInputFlags(size int) InputFlags {
	return InputFlags{set: true, size: size}
}

func NewInputFlagsFromIDs(size int, ids []int32) InputFlags {
	flags := NewInputFlags(size)
	for _, id := range ids {
		flags.Set(int(id))
	}
	return flags
}

func (f InputFlags) Present() bool {
	return f.set
}

func (f InputFlags) Load(i int) bool {
	if !f.set {
		return false
	}
	f.check(i)
	return slices.Contains(f.ids, int32(i))
}

func (f *InputFlags) Set(i int) {
	f.check(i)
	f.set = true
	if !slices.Contains(f.ids, int32(i)) {
		f.ids = append(f.ids, int32(i))
	}
}

func (f *InputFlags) Unset(i int) {
	if !f.set {
		return
	}
	f.check(i)
	if at := slices.Index(f.ids, int32(i)); at != -1 {
		f.ids = slices.Delete(f.ids, at, at+1)
	}
}

func (f InputFlags) Len() int {
	return f.size
}

func (f InputFlags) check(i int) {
	if i < 0 || i >= f.size {
		panic("index out of bounds")
	}
}

func InputFlagList(r IO, x *InputFlags, size int) {

	count := uint32(len(x.ids))
	r.Varuint32(&count)
	if count > uint32(size) {
		r.InvalidValue(count, "player auth input data", "too many flags")
		return
	}
	ids := x.ids
	FuncSliceOfLen(r, count, &ids, r.Varint32)

	for n, id := range ids {
		if id < 0 || int(id) >= size {
			r.UnknownEnumOption(id, "player auth input data")
		} else if slices.Contains(ids[:n], id) {
			r.InvalidValue(id, "player auth input data", "flags must be unique")
		}
	}
	*x = InputFlags{set: true, size: size, ids: ids}
}
