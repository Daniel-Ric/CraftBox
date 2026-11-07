package structure

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strings"
)

type Material struct {
	Name      string `json:"name"`
	Blocks    int64  `json:"blocks"`
	Items     int64  `json:"items"`
	Stacks    int64  `json:"stacks"`
	Remainder int64  `json:"remainder"`
	Estimated bool   `json:"estimated"`
}
type Materials struct {
	Rows    []Material `json:"rows"`
	Blocks  int64      `json:"blocks"`
	Unknown int64      `json:"unknown"`
	Void    int64      `json:"void"`
}

func CountMaterials(ctx context.Context, d *Document, boxes []Bounds) (Materials, error) {
	result := Materials{Rows: []Material{}}
	boxes, err := checkedBounds(d, boxes)
	if err != nil {
		return result, err
	}
	counts := map[string]*Material{}
	for i, b := range boxes {
		err = each(ctx, b, func(p Vec3) error {
			if selected(boxes[:i], p) {
				return nil
			}
			c := d.Cell(p)
			if !c.Known {
				result.Unknown++
				return nil
			}
			if c.Layers[0] < 0 {
				result.Void++
				return nil
			}
			for _, index := range c.Layers {
				if index < 0 {
					continue
				}
				s := d.Palette[index]
				if Air(s) {
					continue
				}
				m := counts[s.Name]
				if m == nil {
					m = &Material{Name: s.Name}
					counts[s.Name] = m
				}
				m.Blocks++
				result.Blocks++
				// Upper halves are still blocks, but require no additional item.
				upper, _ := integer(s.States["upper_block_bit"])
				if upper != 0 && (strings.HasSuffix(s.Name, "door") || strings.HasSuffix(s.Name, "tall_grass") || strings.HasSuffix(s.Name, "double_plant")) {
					continue
				}
				n := int64(1)
				if strings.Contains(s.Name, "double_") && strings.Contains(s.Name, "slab") {
					n = 2
				}
				m.Items += n
				m.Estimated = m.Estimated || !strings.HasPrefix(s.Name, "minecraft:") || strings.Contains(s.Name, "water") || strings.Contains(s.Name, "lava")
			}
			return nil
		})
		if err != nil {
			return result, err
		}
	}
	for _, m := range counts {
		m.Stacks = m.Items / 64
		m.Remainder = m.Items % 64
		result.Rows = append(result.Rows, *m)
	}
	sort.Slice(result.Rows, func(i, j int) bool { return result.Rows[i].Name < result.Rows[j].Name })
	return result, nil
}
func WriteMaterialsCSV(w io.Writer, m Materials) error {
	c := csv.NewWriter(w)
	if err := c.Write([]string{"Block", "Block count", "Item estimate", "Stacks of 64", "Remainder", "Estimated"}); err != nil {
		return err
	}
	for _, r := range m.Rows {
		if err := c.Write([]string{r.Name, fmt.Sprint(r.Blocks), fmt.Sprint(r.Items), fmt.Sprint(r.Stacks), fmt.Sprint(r.Remainder), fmt.Sprint(r.Estimated)}); err != nil {
			return err
		}
	}
	c.Flush()
	return c.Error()
}
