package protocol

const (
	DataStoreChangeTypeUpdate = iota
	DataStoreChangeTypeChange
	DataStoreChangeTypeRemoval
)

const (
	DataStoreControlDouble = iota
	DataStoreControlBoolean
	DataStoreControlString
)

const (
	DataStorePropertyTypeNone   = 0
	DataStorePropertyTypeBool   = 1
	DataStorePropertyTypeInt64  = 2
	DataStorePropertyTypeDouble = 3
	DataStorePropertyTypeString = 4
	DataStorePropertyTypeList   = 5
	DataStorePropertyTypeMap    = 6
)

type DataStoreChangeEntry struct {
	ChangeType uint32

	Update DataStoreUpdate

	Change DataStoreChange

	Removal DataStoreRemoval
}

func (x *DataStoreChangeEntry) Marshal(r IO) {
	r.Varuint32(&x.ChangeType)
	switch x.ChangeType {
	case DataStoreChangeTypeUpdate:
		Single(r, &x.Update)
	case DataStoreChangeTypeChange:
		r.String(&x.Change.DataStoreName)
		r.String(&x.Change.Property)
		r.Uint32(&x.Change.UpdateCount)
		MarshalDataStorePropertyValue(r, &x.Change.NewValue)
	case DataStoreChangeTypeRemoval:
		r.String(&x.Removal.DataStoreName)
	default:
		r.UnknownEnumOption(x.ChangeType, "data store change type")
	}
}

type DataStoreUpdate struct {
	DataStoreName string

	Property string

	Path string

	ControlType uint32

	DoubleValue float64

	BoolValue bool

	StringValue string

	PropertyUpdateCount uint32

	PathUpdateCount uint32
}

func (x *DataStoreUpdate) Marshal(r IO) {
	r.String(&x.DataStoreName)
	r.String(&x.Property)
	r.String(&x.Path)
	r.Varuint32(&x.ControlType)
	switch x.ControlType {
	case DataStoreControlDouble:
		r.Float64(&x.DoubleValue)
	case DataStoreControlBoolean:
		r.Bool(&x.BoolValue)
	case DataStoreControlString:
		r.String(&x.StringValue)
	default:
		r.UnknownEnumOption(x.ControlType, "data store control type")
	}
	r.Uint32(&x.PropertyUpdateCount)
	r.Uint32(&x.PathUpdateCount)
}

type DataStorePropertyValue struct {
	Type int32

	BoolValue bool

	Int64Value int64

	DoubleValue float64

	StringValue string

	ListValue []DataStorePropertyValue

	MapValue []DataStoreMapEntry
}

type DataStoreMapEntry struct {
	Key string

	Value DataStorePropertyValue
}

func MarshalDataStorePropertyValue(r IO, x *DataStorePropertyValue) {
	r.Int32(&x.Type)
	switch x.Type {
	case DataStorePropertyTypeNone:

	case DataStorePropertyTypeBool:
		r.Bool(&x.BoolValue)
	case DataStorePropertyTypeInt64:
		r.Int64(&x.Int64Value)
	case DataStorePropertyTypeDouble:
		r.Float64(&x.DoubleValue)
	case DataStorePropertyTypeString:
		r.String(&x.StringValue)
	case DataStorePropertyTypeList:
		FuncSlice(r, &x.ListValue, func(v *DataStorePropertyValue) {
			MarshalDataStorePropertyValue(r, v)
		})
	case DataStorePropertyTypeMap:
		FuncSlice(r, &x.MapValue, func(entry *DataStoreMapEntry) {
			r.String(&entry.Key)
			MarshalDataStorePropertyValue(r, &entry.Value)
		})
	default:
		r.UnknownEnumOption(x.Type, "data store property type")
	}
}

type DataStoreChange struct {
	DataStoreName string

	Property string

	UpdateCount uint32

	NewValue DataStorePropertyValue
}

type DataStoreRemoval struct {
	DataStoreName string
}
