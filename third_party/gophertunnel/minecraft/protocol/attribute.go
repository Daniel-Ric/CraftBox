package protocol

const (
	AttributeModifierOperationAddition = iota
	AttributeModifierOperationMultiplyBase
	AttributeModifierOperationMultiplyTotal
	AttributeModifierOperationCap
)

const (
	AttributeModifierOperandMin = iota
	AttributeModifierOperandMax
	AttributeModifierOperandCurrent
)

type AttributeValue struct {
	Name string

	Value float32

	Max, Min float32
}

func (x *AttributeValue) Marshal(r IO) {
	r.String(&x.Name)
	r.Float32(&x.Min)
	r.Float32(&x.Value)
	r.Float32(&x.Max)
}

type Attribute struct {
	AttributeValue

	DefaultMin float32

	DefaultMax float32

	Default float32

	Modifiers []AttributeModifier
}

func (x *Attribute) Marshal(r IO) {
	r.Float32(&x.Min)
	r.Float32(&x.Max)
	r.Float32(&x.Value)
	r.Float32(&x.DefaultMin)
	r.Float32(&x.DefaultMax)
	r.Float32(&x.Default)
	r.String(&x.Name)
	Slice(r, &x.Modifiers)
}

type AttributeModifier struct {
	ID string

	Name string

	Amount float32

	Operation int32

	Operand int32

	Serializable bool
}

func (x *AttributeModifier) Marshal(r IO) {
	r.String(&x.ID)
	r.String(&x.Name)
	r.Float32(&x.Amount)
	r.Int32(&x.Operation)
	r.Int32(&x.Operand)
	r.Bool(&x.Serializable)
}
