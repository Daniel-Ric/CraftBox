package protocol

type EntityProperties struct {
	IntegerProperties []IntegerEntityProperty

	FloatProperties []FloatEntityProperty
}

func (e *EntityProperties) Marshal(r IO) {
	Slice(r, &e.IntegerProperties)
	Slice(r, &e.FloatProperties)
}

type IntegerEntityProperty struct {
	Index uint32

	Value int32
}

func (i *IntegerEntityProperty) Marshal(r IO) {
	r.Varuint32(&i.Index)
	r.Varint32(&i.Value)
}

type FloatEntityProperty struct {
	Index uint32

	Value float32
}

func (f *FloatEntityProperty) Marshal(r IO) {
	r.Varuint32(&f.Index)
	r.Float32(&f.Value)
}
