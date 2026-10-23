package nbtconv

func Integer(value any) (int64, bool) {
	switch v := value.(type) {
	case uint8:
		return int64(v), true
	case int8:
		return int64(v), true
	case int16:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case int:
		return int64(v), true
	case uint16:
		return int64(v), true
	case uint32:
		return int64(v), true
	case bool:
		if v {
			return 1, true
		}
		return 0, true
	default:
		return 0, false
	}
}

func Flag(value any) bool { n, ok := Integer(value); return ok && n != 0 }
