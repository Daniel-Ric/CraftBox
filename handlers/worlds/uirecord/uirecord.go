package uirecord

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

const triggerWindow = 3 * time.Second

type EntityInfo func(rid uint64) (typeID, nameTag string, ok bool)

type Form struct {
	Key  string         `json:"key"`
	Data map[string]any `json:"data"`
}

type objective struct {
	Name        string           `json:"name"`
	DisplayName string           `json:"displayName"`
	Slots       map[string]int32 `json:"slots"`
	Scores      map[string]int32 `json:"scores"`
}

type pendingResponse struct {
	formKey   string
	selection string
	at        time.Time
}

type trigger struct {
	kind string
	id   string
	at   time.Time
}

type Recorder struct {
	mu sync.Mutex

	entityInfo EntityInfo
	itemNames  map[int32]string

	forms     map[string]*Form
	formOrder []string
	formIDs   map[uint32]string
	next      map[string]map[string]string
	items     map[string]string
	entities  map[string]string

	lastTrigger  *trigger
	lastResponse *pendingResponse

	objectives map[string]*objective
	entryNames map[int64]string
}

func New(entityInfo EntityInfo) *Recorder {
	return &Recorder{
		entityInfo: entityInfo,
		itemNames:  make(map[int32]string),
		forms:      make(map[string]*Form),
		formIDs:    make(map[uint32]string),
		next:       make(map[string]map[string]string),
		items:      make(map[string]string),
		entities:   make(map[string]string),
		objectives: make(map[string]*objective),
		entryNames: make(map[int64]string),
	}
}

func (r *Recorder) HasContent() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.forms) > 0 || len(r.objectives) > 0
}

func (r *Recorder) ResetCapture() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.forms = make(map[string]*Form)
	r.formOrder = nil
	r.formIDs = make(map[uint32]string)
	r.next = make(map[string]map[string]string)
	r.items = make(map[string]string)
	r.entities = make(map[string]string)
	r.lastTrigger = nil
	r.lastResponse = nil
}

func (r *Recorder) OnPacket(pk packet.Packet, toServer bool, now time.Time) {

	var entityTrigger string
	if it, ok := pk.(*packet.InventoryTransaction); ok && toServer && r.entityInfo != nil {
		if td, ok := it.TransactionData.(*protocol.UseItemOnEntityTransactionData); ok && td.ActionType == protocol.UseItemOnEntityActionInteract {
			if typeID, nameTag, ok := r.entityInfo(td.TargetEntityRuntimeID); ok {
				entityTrigger = typeID + "|" + nameTag
			}
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	switch pk := pk.(type) {
	case *packet.ItemRegistry:
		for _, it := range pk.Items {
			r.itemNames[int32(it.RuntimeID)] = it.Name
		}

	case *packet.InventoryTransaction:
		if !toServer {
			return
		}
		switch td := pk.TransactionData.(type) {
		case *protocol.UseItemTransactionData:
			if td.ActionType != protocol.UseItemActionClickAir && td.ActionType != protocol.UseItemActionClickBlock {
				return
			}
			if name, ok := r.itemNames[td.HeldItem.Stack.NetworkID]; ok && td.HeldItem.Stack.NetworkID != 0 {
				r.lastTrigger = &trigger{kind: "item", id: name, at: now}
			}
		case *protocol.UseItemOnEntityTransactionData:
			if entityTrigger != "" {
				r.lastTrigger = &trigger{kind: "entity", id: entityTrigger, at: now}
			}
		}

	case *packet.ModalFormRequest:
		if toServer {
			return
		}
		r.onFormRequest(pk, now)

	case *packet.ModalFormResponse:
		if !toServer {
			return
		}
		key, ok := r.formIDs[pk.FormID]
		if !ok {
			return
		}
		data, ok := pk.ResponseData.Value()
		if !ok {
			r.lastResponse = nil
			return
		}
		sel := selectionOf(r.forms[key], data)
		if sel == "" {
			r.lastResponse = nil
			return
		}
		r.lastResponse = &pendingResponse{formKey: key, selection: sel, at: now}

	case *packet.SetDisplayObjective:
		if toServer {
			return
		}
		o := r.objective(pk.ObjectiveName)
		o.DisplayName = pk.DisplayName
		if pk.DisplaySlot != "" {

			for _, other := range r.objectives {
				delete(other.Slots, pk.DisplaySlot)
			}
			o.Slots[pk.DisplaySlot] = pk.SortOrder
		}

	case *packet.RemoveObjective:
		if !toServer {
			delete(r.objectives, pk.ObjectiveName)
		}

	case *packet.SetScore:
		if toServer {
			return
		}
		for _, e := range pk.Entries {
			switch e.IdentityType {
			case protocol.ScoreboardIdentityFakePlayer:
				o := r.objective(e.ObjectiveName)
				if old, ok := r.entryNames[e.EntryID]; ok && old != e.DisplayName {
					delete(o.Scores, old)
				}
				r.entryNames[e.EntryID] = e.DisplayName
				o.Scores[e.DisplayName] = e.Score
			case protocol.ScoreboardIdentityRemove:
				name, ok := r.entryNames[e.EntryID]
				if !ok {
					continue
				}
				for _, o := range r.objectives {
					if e.ObjectiveName == "" || o.Name == e.ObjectiveName {
						delete(o.Scores, name)
					}
				}
				delete(r.entryNames, e.EntryID)
			}

		}
	}
}

func (r *Recorder) objective(name string) *objective {
	o, ok := r.objectives[name]
	if !ok {
		o = &objective{Name: name, DisplayName: name, Slots: map[string]int32{}, Scores: map[string]int32{}}
		r.objectives[name] = o
	}
	return o
}

func (r *Recorder) onFormRequest(pk *packet.ModalFormRequest, now time.Time) {
	var data map[string]any
	if err := json.Unmarshal(pk.FormData, &data); err != nil {
		return
	}
	switch data["type"] {
	case "form", "modal", "custom_form":
	default:
		return
	}

	canon, _ := json.Marshal(data)
	sum := sha1.Sum(canon)
	key := "f" + hex.EncodeToString(sum[:6])
	if _, ok := r.forms[key]; !ok {
		r.forms[key] = &Form{Key: key, Data: data}
		r.formOrder = append(r.formOrder, key)
	}
	r.formIDs[pk.FormID] = key

	switch {
	case r.lastResponse != nil && now.Sub(r.lastResponse.at) < triggerWindow:
		m := r.next[r.lastResponse.formKey]
		if m == nil {
			m = make(map[string]string)
			r.next[r.lastResponse.formKey] = m
		}
		if _, seen := m[r.lastResponse.selection]; !seen {
			m[r.lastResponse.selection] = key
		}
	case r.lastTrigger != nil && now.Sub(r.lastTrigger.at) < triggerWindow:
		target := r.items
		if r.lastTrigger.kind == "entity" {
			target = r.entities
		}
		if _, seen := target[r.lastTrigger.id]; !seen {
			target[r.lastTrigger.id] = key
		}
	}
	r.lastResponse = nil
	r.lastTrigger = nil
}

func selectionOf(f *Form, data []byte) string {
	s := strings.TrimSpace(string(data))
	if s == "" || s == "null" {
		return ""
	}
	if f == nil {
		return ""
	}
	switch f.Data["type"] {
	case "form":
		if _, err := strconv.Atoi(s); err == nil {
			return s
		}
	case "modal":
		switch s {
		case "true":
			return "0"
		case "false":
			return "1"
		}
	case "custom_form":
		return "submit"
	}
	return ""
}

func (r *Recorder) Script(scope ...string) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	forms := make(map[string]any, len(r.forms))
	for _, k := range r.formOrder {
		forms[k] = r.forms[k].Data
	}
	objectives := make([]*objective, 0, len(r.objectives))
	for _, o := range r.objectives {
		out := *o
		if len(scope) > 1 {
			sum := sha1.Sum([]byte(scope[1] + ":" + o.Name))
			out.Name = "cb_" + hex.EncodeToString(sum[:6])
		}
		objectives = append(objectives, &out)
	}
	sort.Slice(objectives, func(i, j int) bool { return objectives[i].Name < objectives[j].Name })

	js := func(v any) string {
		b, _ := json.MarshalIndent(v, "", "  ")
		return string(b)
	}

	var sb strings.Builder
	sb.WriteString(scriptHeader)
	dimension := ""
	if len(scope) > 0 {
		dimension = scope[0]
	}
	fmt.Fprintf(&sb, "const CAPTURE_DIMENSION = %s;\n", js(dimension))
	fmt.Fprintf(&sb, "const FORMS = %s;\n\n", js(forms))
	fmt.Fprintf(&sb, "const NEXT = %s;\n\n", js(r.next))
	fmt.Fprintf(&sb, "const ITEM_TRIGGERS = %s;\n\n", js(r.items))
	fmt.Fprintf(&sb, "const ENTITY_TRIGGERS = %s;\n\n", js(r.entities))
	fmt.Fprintf(&sb, "const OBJECTIVES = %s;\n\n", js(objectives))
	sb.WriteString(scriptBody)
	return sb.String()
}

const scriptHeader = `import { world, system, DisplaySlotId, ObjectiveSortOrder } from "@minecraft/server";
import { ActionFormData, MessageFormData, ModalFormData } from "@minecraft/server-ui";

`

const scriptBody = `function inCaptureDimension(player) {
  return player && (!CAPTURE_DIMENSION || player.dimension.id === CAPTURE_DIMENSION);
}

function text(t) {
  if (t === undefined || t === null) return "";
  if (typeof t === "string") return t;
  return { rawtext: Array.isArray(t.rawtext) ? t.rawtext : [{ text: JSON.stringify(t) }] };
}

function build(f) {
  switch (f.type) {
    case "form": {
      const form = new ActionFormData().title(text(f.title)).body(text(f.content));
      for (const b of f.buttons ?? []) {
        const icon = b.image && b.image.type === "path" ? b.image.data : undefined;
        form.button(text(b.text), icon);
      }
      return form;
    }
    case "modal":
      return new MessageFormData()
        .title(text(f.title))
        .body(text(f.content))
        .button1(text(f.button1))
        .button2(text(f.button2));
    case "custom_form": {
      const form = new ModalFormData().title(text(f.title));
      let pending = "";
      for (const e of f.content ?? []) {
        const label = pending + (typeof e.text === "string" ? e.text : "");
        pending = "";
        switch (e.type) {
          case "label":
            pending = label + "\n";
            break;
          case "input":
            form.textField(label, e.placeholder ?? "", e.default ?? "");
            break;
          case "toggle":
            form.toggle(label, !!e.default);
            break;
          case "slider":
            form.slider(label, e.min ?? 0, e.max ?? 100, e.step ?? 1, e.default ?? e.min ?? 0);
            break;
          case "dropdown":
            form.dropdown(label, (e.options ?? []).map(text), e.default ?? 0);
            break;
          case "step_slider":
            form.dropdown(label, (e.steps ?? []).map(text), e.default ?? 0);
            break;
        }
      }
      return form;
    }
  }
  return undefined;
}

function showForm(player, key) {
  if (!inCaptureDimension(player)) return;
  const f = FORMS[key];
  if (!f) return;
  const form = build(f);
  if (!form) return;
  form.show(player).then((r) => {
    if (r.canceled) return;
    const sel = f.type === "custom_form" ? "submit" : String(r.selection);
    const next = NEXT[key] && NEXT[key][sel];
    if (next) system.run(() => showForm(player, next));
  }).catch(() => {});
}

world.afterEvents.itemUse.subscribe((ev) => {
  const key = ITEM_TRIGGERS[ev.itemStack.typeId];
  if (key) system.run(() => showForm(ev.source, key));
});

world.afterEvents.playerInteractWithEntity.subscribe((ev) => {
  const t = ev.target;
  const key = ENTITY_TRIGGERS[t.typeId + "|" + (t.nameTag ?? "")] ?? ENTITY_TRIGGERS[t.typeId + "|"];
  if (key) system.run(() => showForm(ev.player, key));
});

system.afterEvents.scriptEventReceive.subscribe((ev) => {
  if (ev.id !== "craftbox:form") return;
  const player = ev.sourceEntity ?? world.getAllPlayers().find(inCaptureDimension);
  if (player) system.run(() => showForm(player, ev.message.trim()));
});

const SLOTS = { sidebar: DisplaySlotId.Sidebar, list: DisplaySlotId.List, belowname: DisplaySlotId.BelowName };

function applyScoreboards() {
  if (!world.getAllPlayers().some(inCaptureDimension)) return;
  for (const o of OBJECTIVES) {
    let obj = world.scoreboard.getObjective(o.name);
    if (!obj) {
      try {
        obj = world.scoreboard.addObjective(o.name, o.displayName);
      } catch (e) {
        continue;
      }
    }
    for (const [name, score] of Object.entries(o.scores)) {
      try { obj.setScore(name, score); } catch (e) {}
    }
    for (const [slot, order] of Object.entries(o.slots)) {
      const id = SLOTS[slot];
      if (!id) continue;
      world.scoreboard.setObjectiveAtDisplaySlot(id, {
        objective: obj,
        sortOrder: order === 1 ? ObjectiveSortOrder.Descending : ObjectiveSortOrder.Ascending,
      });
    }
  }
}

system.run(applyScoreboards);
world.afterEvents.playerSpawn.subscribe((ev) => {
  if (inCaptureDimension(ev.player)) system.run(applyScoreboards);
});
world.afterEvents.playerDimensionChange.subscribe((ev) => {
  if (ev.toDimension.id === CAPTURE_DIMENSION) system.run(applyScoreboards);
});
`
