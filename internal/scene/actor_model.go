package scene

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"io/fs"
	"math"
	"path"
	"sort"
	"strings"
)

type Actor struct {
	RawNBT               []byte `json:"-"`
	ID                   int64
	Type                 string
	Position             [3]float32
	Yaw, Scale           float32
	Variant, MarkVariant int
	OwnerID              int64
	Invisible            bool
	Excluded             bool
	Name                 string
	NameVisible          bool
	Height               float32
	Properties           map[string]any
}

type actorDefinition struct {
	Identifier       string              `json:"identifier"`
	Geometry         map[string]string   `json:"geometry"`
	Textures         map[string]string   `json:"textures"`
	Materials        map[string]string   `json:"materials"`
	Animations       map[string]string   `json:"animations"`
	LegacyAnimations []map[string]string `json:"animation_controllers"`
	Controllers      []any               `json:"render_controllers"`
	Scripts          struct {
		Animate []any `json:"animate"`
		Scale   any   `json:"scale"`
	} `json:"scripts"`
}

func actorAllowed(a Actor) bool {
	n := strings.ToLower(a.Type)
	return !a.Excluded && n != "minecraft:player" && !strings.HasPrefix(n, "craftbox_player:") && !strings.HasPrefix(n, "cubecraft:buddy.") && !strings.HasPrefix(n, "cubecraft:buddy_") && (a.OwnerID == 0 || a.OwnerID == -1)
}

func (c *Catalog) loadActorAssets(source fs.FS) error {
	if c.actors == nil {
		c.actors = map[string]actorDefinition{}
		c.renderControllers = map[string]map[string]any{}
		c.animations = map[string]map[string]any{}
		c.animationControllers = map[string]map[string]any{}
	}
	for _, root := range []string{"entity", "render_controllers", "animations", "animation_controllers"} {
		count := 0
		err := fs.WalkDir(source, root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				if p == root && errors.Is(err, fs.ErrNotExist) {
					return nil
				}
				return err
			}
			if d.IsDir() || !strings.HasSuffix(strings.ToLower(p), ".json") {
				return nil
			}
			count++
			if count > 4096 {
				return fs.SkipAll
			}
			info, e := d.Info()
			if e != nil {
				return e
			}
			if info.Size() > 8<<20 {
				return nil
			}
			raw, e := fs.ReadFile(source, p)
			if e != nil {
				return e
			}
			raw, e = standardizePackJSON(raw)
			if e != nil {
				return fmt.Errorf("%s: %w", p, e)
			}
			var wrapper struct {
				Entity struct {
					Description actorDefinition `json:"description"`
				} `json:"minecraft:client_entity"`
				Controllers          map[string]map[string]any `json:"render_controllers"`
				Animations           map[string]map[string]any `json:"animations"`
				AnimationControllers map[string]map[string]any `json:"animation_controllers"`
			}
			if e = json.Unmarshal(raw, &wrapper); e != nil {
				return e
			}
			if wrapper.Entity.Description.Identifier != "" {
				c.actors[wrapper.Entity.Description.Identifier] = wrapper.Entity.Description
			}
			for k, v := range wrapper.Controllers {
				c.renderControllers[strings.ToLower(k)] = v
			}
			for k, v := range wrapper.Animations {
				c.animations[k] = v
			}
			for k, v := range wrapper.AnimationControllers {
				c.animationControllers[k] = v
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func actorStates(a Actor) map[string]any {
	s := map[string]any{"query.variant": a.Variant, "q.variant": a.Variant, "query.mark_variant": a.MarkVariant, "q.mark_variant": a.MarkVariant, "query.anim_time": 0, "query.life_time": 0, "query.modified_distance_moved": 0, "query.is_baby": false, "query.is_sitting": false, "query.is_moving": false}
	for k, v := range a.Properties {
		s[k] = v
	}
	return s
}
func namedEntries(v any, states map[string]any) []string {
	out := []string{}
	add := func(v any) {
		switch x := v.(type) {
		case string:
			out = append(out, x)
		case map[string]any:
			keys := make([]string, 0, len(x))
			for k := range x {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				ok := truth(x[k])
				if expr, yes := x[k].(string); yes {
					ok, _ = blockCondition(expr, states)
				}
				if ok {
					out = append(out, k)
				}
			}
		}
	}
	if list, ok := v.([]any); ok {
		for _, v := range list {
			add(v)
		}
	} else {
		add(v)
	}
	return out
}

func aliasValue(aliases map[string]string, expression, prefix string) string {
	expression = strings.TrimSpace(expression)
	if !strings.HasPrefix(strings.ToLower(expression), prefix+".") {
		return ""
	}
	name := expression[len(prefix)+1:]
	for k, v := range aliases {
		if strings.EqualFold(k, name) {
			return v
		}
	}
	return ""
}
func actorReference(expression string, rc map[string]any, kind string, aliases map[string]string, states map[string]any, depth int) string {
	if depth > 8 {
		return ""
	}
	expression = strings.TrimSpace(expression)
	prefix := map[string]string{"textures": "texture", "geometries": "geometry", "materials": "material"}[kind]
	if v := aliasValue(aliases, expression, prefix); v != "" {
		return v
	}
	if i := strings.IndexByte(expression, '['); i > 0 && strings.HasSuffix(expression, "]") {
		index, ok := staticExpression(expression[i+1:len(expression)-1], states)
		if !ok {
			return ""
		}
		arrays, _ := rc["arrays"].(map[string]any)
		group, _ := arrays[kind].(map[string]any)
		for k, v := range group {
			if strings.EqualFold(k, expression[:i]) {
				list, _ := v.([]any)
				if len(list) == 0 {
					return ""
				}
				n := int(math.Max(0, numeric(index))) % len(list)
				s, _ := list[n].(string)
				return actorReference(s, rc, kind, aliases, states, depth+1)
			}
		}
	}
	return ""
}

func poseChannel(value any, states map[string]any, fallback [3]float32) ([3]float32, bool) {
	if keys, ok := value.(map[string]any); ok {
		if v, ok := keys["post"]; ok {
			return poseChannel(v, states, fallback)
		}
		if v, ok := keys["pre"]; ok {
			return poseChannel(v, states, fallback)
		}
		earliest := math.Inf(1)
		var selected any
		for k, v := range keys {
			var time float64
			if _, err := fmt.Sscan(k, &time); err == nil && time < earliest {
				earliest = time
				selected = v
			}
		}
		if selected == nil {
			return fallback, false
		}
		return poseChannel(selected, states, fallback)
	}
	if list, ok := value.([]any); ok && len(list) == 3 {
		for i, v := range list {
			if s, ok := v.(string); ok {
				n, supported := staticExpression(s, states)
				if !supported {
					return fallback, false
				}
				fallback[i] = float32(numeric(n))
			} else {
				fallback[i] = float32(numeric(v))
			}
		}
		return fallback, true
	}
	if str, ok := value.(string); ok {
		v, supported := staticExpression(str, states)
		if !supported {
			return fallback, false
		}
		n := float32(numeric(v))
		return [3]float32{n, n, n}, true
	}
	if value != nil {
		n := float32(numeric(value))
		return [3]float32{n, n, n}, true
	}
	return fallback, false
}

func (c *Catalog) actorGeometry(g blockGeometry, d actorDefinition, states map[string]any) blockGeometry {
	g.Bones = append([]geometryBone(nil), g.Bones...)
	applied := map[string]bool{}
	var apply func(string, int)
	apply = func(alias string, depth int) {
		if depth > 8 {
			return
		}
		id := d.Animations[alias]
		if id == "" {
			id = alias
		}
		if applied[id] {
			return
		}
		applied[id] = true
		if controller, ok := c.animationControllers[id]; ok {
			initial, _ := controller["initial_state"].(string)
			if initial == "" {
				initial = "default"
			}
			all, _ := controller["states"].(map[string]any)
			state, _ := all[initial].(map[string]any)
			for _, a := range namedEntries(state["animations"], states) {
				apply(a, depth+1)
			}
			return
		}
		animation := c.animations[id]
		bones, _ := animation["bones"].(map[string]any)
		for i := range g.Bones {
			b := &g.Bones[i]
			channels, _ := bones[b.Name].(map[string]any)
			if v, ok := poseChannel(channels["rotation"], states, [3]float32{}); ok {
				for j := range v {
					b.Rotation[j] += v[j]
				}
			}
			if v, ok := poseChannel(channels["position"], states, [3]float32{}); ok {
				for j := range v {
					b.Position[j] += v[j]
				}
			}
			if v, ok := poseChannel(channels["scale"], states, [3]float32{1, 1, 1}); ok {
				b.Scale = &v
			}
		}
	}
	for _, alias := range namedEntries(d.Scripts.Animate, states) {
		apply(alias, 0)
	}
	for _, entry := range d.LegacyAnimations {
		keys := make([]string, 0, len(entry))
		for k := range entry {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, key := range keys {
			apply(entry[key], 0)
		}
	}
	return g
}

func (c *Catalog) actorModel(a Actor) blockModel {
	out := blockModel{name: a.Type}
	if !actorAllowed(a) || a.Scale == 0 || a.Invisible {
		return out
	}
	d, ok := c.actors[a.Type]
	if !ok {
		return out
	}
	states := actorStates(a)
	for _, id := range namedEntries(d.Controllers, states) {
		rc := c.renderControllers[strings.ToLower(id)]
		if rc == nil && strings.HasPrefix(a.Type, "minecraft:") && d.Geometry["default"] != "" && d.Textures["default"] != "" {
			rc = map[string]any{"geometry": "Geometry.default", "textures": []any{"Texture.default"}}
			out.approx = true
		}
		geo, _ := rc["geometry"].(string)
		gid := actorReference(geo, rc, "geometries", d.Geometry, states, 0)
		g, ok := c.geometries[gid]
		if !ok {
			continue
		}
		texture := ""
		if list, ok := rc["textures"].([]any); ok {
			for _, v := range list {
				expr, _ := v.(string)
				texture = actorReference(expr, rc, "textures", d.Textures, states, 0)
				if texture != "" {
					break
				}
			}
		}
		if texture == "" {
			continue
		}
		visibility := map[string]any{}
		for _, v := range namedVisibility(rc["part_visibility"]) {
			patterns := make([]string, 0, len(v))
			for pattern := range v {
				patterns = append(patterns, pattern)
			}
			sort.Slice(patterns, func(i, j int) bool {
				a, b := strings.Count(patterns[i], "*"), strings.Count(patterns[j], "*")
				if a != b {
					return a > b
				}
				return patterns[i] < patterns[j]
			})
			for _, pattern := range patterns {
				value := v[pattern]
				for _, bone := range g.Bones {
					match, _ := pathMatch(pattern, bone.Name)
					if match {
						visibility[bone.Name] = value
					}
				}
			}
		}
		posed := c.actorGeometry(g, d, states)
		method := "alpha_test"
		materialName := ""
		if entries, ok := rc["materials"].([]any); ok {
			for _, entry := range entries {
				if values, ok := entry.(map[string]any); ok {
					if expr, ok := values["*"].(string); ok {
						materialName = actorReference(expr, rc, "materials", d.Materials, states, 0)
					}
				}
			}
		}
		if materialName == "" {
			materialName = d.Materials["default"]
		}
		lowerMaterial := strings.ToLower(materialName)
		blended := strings.Contains(lowerMaterial, "blend") || strings.Contains(lowerMaterial, "experience_orb") || strings.Contains(lowerMaterial, "entity_translucent")
		if blended {
			method = "blend"
		}
		unlit := truth(rc["ignore_lighting"]) || strings.Contains(lowerMaterial, "emissive") || strings.Contains(lowerMaterial, "experience_orb")

		const key = "craftbox:actor_pose"
		old, existed := c.geometries[key]
		c.geometries[key] = posed
		m, _ := c.definitionModel(a.Type, states, blockDefinition{Components: map[string]any{"minecraft:geometry": map[string]any{"identifier": key, "bone_visibility": visibility}, "minecraft:material_instances": map[string]any{"*": map[string]any{"texture": texture, "render_method": method}}}})
		out.approx = out.approx || m.approx
		if existed {
			c.geometries[key] = old
		} else {
			delete(c.geometries, key)
		}
		scale := a.Scale

		if v, ok := d.Scripts.Scale.(float64); ok && v > 0 {
			scale *= float32(v)
		}
		if expr, ok := d.Scripts.Scale.(string); ok {
			if value, supported := staticExpression(expr, states); supported {
				scale *= float32(numeric(value))
			}
		}
		if scale <= 0 {
			continue
		}

		transform := mgl32.HomogRotate3DY((180 - a.Yaw) * math.Pi / 180).Mul4(mgl32.Scale3D(scale, scale, scale))
		for _, q := range m.quads {
			for i, p := range q.points {
				v := transform.Mul4x1(mgl32.Vec4{p[0] - .5, p[1], p[2] - .5, 1})
				q.points[i] = [3]float32{v[0], v[1], v[2]}
			}
			n := transform.Mul4x1(mgl32.Vec4{q.normal[0], q.normal[1], q.normal[2], 0}).Vec3()
			if n.Len() > 0 {
				q.normal = [3]float32(n.Normalize())
			}
			if unlit {
				q.mode = "unlit"
				if blended {
					q.mode = "unlit-blend"
				}
			}
			out.quads = append(out.quads, q)
		}
	}
	return out
}

func namedVisibility(v any) []map[string]any {
	out := []map[string]any{}
	list, _ := v.([]any)
	for _, v := range list {
		if m, ok := v.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}
func pathMatch(pattern, name string) (bool, error) { return path.Match(pattern, name) }
