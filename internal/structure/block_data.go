package structure

// translateBlockData moves absolute Bedrock references with their owning block.
func translateBlockData(data map[string]any, delta Vec3) {
	for _, key := range []string{"x", "pairx"} {
		if n, ok := integer(data[key]); ok {
			data[key] = n + delta[0]
		}
	}
	for _, key := range []string{"y"} {
		if n, ok := integer(data[key]); ok {
			data[key] = n + delta[1]
		}
	}
	for _, key := range []string{"z", "pairz"} {
		if n, ok := integer(data[key]); ok {
			data[key] = n + delta[2]
		}
	}
}

func (d *Document) ExportCopy(o ExportOptions) *Document {
	out := d.Clone()
	out.BlockData = map[string]map[string]any{}
	if o.BlockData {
		for p, data := range d.BlockData {
			v := cloneMap(data)
			if !o.Inventories {
				removeInventories(v)
			}
			out.BlockData[p] = v
		}
	}
	out.Entities = nil
	if o.Entities {
		for _, e := range d.Entities {
			if !isPlayer(e) {
				v := cloneMap(e)
				if !o.Inventories {
					removeInventories(v)
				}
				out.Entities = append(out.Entities, v)
			}
		}
	}
	return out
}
