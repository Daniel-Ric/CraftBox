package entity

func MetadataNonZero(value any) bool {
	switch value := value.(type) {
	case byte:
		return value != 0
	case int8:
		return value != 0
	case int16:
		return value != 0
	case int32:
		return value != 0
	case int64:
		return value != 0
	case int:
		return value != 0
	case float32:
		return value != 0
	case float64:
		return value != 0
	case bool:
		return value
	}
	return false
}
