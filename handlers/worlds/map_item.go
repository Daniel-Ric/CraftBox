package worlds

import (
	"context"
	"errors"
	"image"
	"image/draw"
	"math"
	"net"
	"sync"
	"time"

	"github.com/Daniel-Ric/CraftBox/ui/messages"
	"github.com/Daniel-Ric/CraftBox/utils"
	"github.com/go-gl/mathgl/mgl32"

	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sirupsen/logrus"
)

const ViewMapID = 0x424242

var mapItem = protocol.ItemInstance{
	StackNetworkID: 0x424242,
	Stack: protocol.ItemStack{
		ItemType: protocol.ItemType{
			NetworkID:     420,
			MetadataValue: 0,
		},
		BlockRuntimeID: 0,
		Count:          1,
		NBTData: map[string]any{
			"map_name_index": int64(1),
			"map_uuid":       int64(ViewMapID),
		},
	},
}

func (m *MapUI) GetBounds() (Min, Max protocol.ChunkPos) {
	if len(m.renderedChunks) == 0 {
		return
	}
	Min = protocol.ChunkPos{math.MaxInt32, math.MaxInt32}
	for chunk := range m.renderedChunks {
		Min[0] = min(Min[0], chunk[0])
		Min[1] = min(Min[1], chunk[1])
		Max[0] = max(Max[0], chunk[0])
		Max[1] = max(Max[1], chunk[1])
	}
	return
}

type renderElem struct {
	ch  *chunk.Chunk
	pos protocol.ChunkPos
}

type MapUI struct {
	worldID        string
	log            *logrus.Entry
	mapImage       *image.RGBA
	renderQueue    []*renderElem
	renderedChunks map[protocol.ChunkPos]*image.RGBA
	oldRendered    map[protocol.ChunkPos]*image.RGBA
	ticker         *time.Ticker
	w              *worldsHandler

	ChunkRenderer *utils.ChunkRenderer

	mu         sync.Mutex
	haveColors chan struct{}

	zoomLevel  int
	needRedraw bool
	isDisabled bool

	offHandItem protocol.ItemInstance
}

func NewMapUI(w *worldsHandler) *MapUI {
	m := &MapUI{
		log:            logrus.WithField("part", "MapUI"),
		mapImage:       image.NewRGBA(image.Rect(0, 0, 128, 128)),
		zoomLevel:      16,
		renderedChunks: make(map[protocol.ChunkPos]*image.RGBA),
		oldRendered:    make(map[protocol.ChunkPos]*image.RGBA),
		needRedraw:     true,
		w:              w,
		haveColors:     make(chan struct{}),
		ChunkRenderer:  nil,
	}
	return m
}

func (m *MapUI) SetEnabled(enabled bool) {
	change := enabled == m.isDisabled
	if !change {
		return
	}
	m.isDisabled = !enabled

	var newItem protocol.ItemInstance
	if m.isDisabled {
		newItem = m.offHandItem
	} else {
		newItem = mapItem
	}
	err := m.w.session.ClientWritePacket(&packet.InventoryContent{
		WindowID: 119,
		Content:  []protocol.ItemInstance{newItem},
	})
	if err != nil {
		m.log.Error(err)
		return
	}
}

func (m *MapUI) mapUpdater(ctx context.Context) {
	var oldPos mgl32.Vec3
	for range m.ticker.C {
		if ctx.Err() != nil {
			return
		}
		newPos := m.w.session.Player.Position
		if int(oldPos.X()) != int(newPos.X()) || int(oldPos.Z()) != int(newPos.Z()) {
			m.needRedraw = true
			oldPos = newPos
		}

		if m.needRedraw {
			m.needRedraw = false
			m.redraw()

			if err := m.w.session.ClientWritePacket(&packet.ClientBoundMapItemData{
				MapID:  ViewMapID,
				Scale:  protocol.Option(uint8(4)),
				Width:  protocol.Option(int32(128)),
				Height: protocol.Option(int32(128)),
				Pixels: protocol.Option(utils.Img2rgba(m.mapImage)),
			}); err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}
				if errors.Is(err, context.Canceled) {
					return
				}
				m.log.Error(err)
				return
			}
		}
	}
}

func (m *MapUI) itemSender() {
	t := time.NewTicker(1 * time.Second)
	for range t.C {
		if m.w.session.Client == nil {
			return
		}
		if m.isDisabled {
			continue
		}
		err := m.w.session.ClientWritePacket(&packet.InventoryContent{
			WindowID: 119,
			Content:  []protocol.ItemInstance{mapItem},
		})
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			if errors.Is(err, context.Canceled) {
				return
			}
			m.log.Error(err)
			return
		}
	}
}

func (m *MapUI) Start(ctx context.Context) {

	err := m.w.session.ClientWritePacket(&packet.ClientBoundMapItemData{
		MapID:          ViewMapID,
		Scale:          protocol.Option(uint8(4)),
		MapsIncludedIn: protocol.Option([]int64{ViewMapID}),
	})
	if err != nil {
		m.log.Error(err)
		return
	}

	m.ticker = time.NewTicker(33 * time.Millisecond)
	m.ChunkRenderer = utils.NewChunkRenderer(m.w.serverState.blocks)
	go func() {
		m.ChunkRenderer.ResolveColors(
			m.w.serverState.customBlocks,
			m.w.session.ActiveResourcePacks(),
		)
		close(m.haveColors)
	}()
	go m.mapUpdater(ctx)
	go m.itemSender()
}

func (m *MapUI) Stop() {
	if m.ticker != nil {
		m.ticker.Stop()
	}
}

func (m *MapUI) SwitchWorld(id string) {
	m.mu.Lock()
	if m.worldID == id {
		m.mu.Unlock()
		return
	}
	m.worldID = id
	m.renderQueue = nil
	m.renderedChunks = make(map[protocol.ChunkPos]*image.RGBA)
	m.oldRendered = make(map[protocol.ChunkPos]*image.RGBA)
	m.mu.Unlock()
	m.SchedRedraw()
}
func (m *MapUI) Reset() {
	m.mu.Lock()
	m.renderQueue = nil
	m.renderedChunks = make(map[protocol.ChunkPos]*image.RGBA)
	m.oldRendered = make(map[protocol.ChunkPos]*image.RGBA)
	messages.SendEvent(&messages.EventResetMap{WorldID: m.worldID})
	m.mu.Unlock()
	m.SchedRedraw()
}

func (m *MapUI) ChangeZoom() {
	m.zoomLevel /= 2
	if m.zoomLevel == 0 {
		m.zoomLevel = 16
	}
	m.SchedRedraw()
}

func (m *MapUI) SchedRedraw() {
	m.needRedraw = true
}

func (m *MapUI) processQueue() []protocol.ChunkPos {
	<-m.haveColors

	updatedChunks := make([]protocol.ChunkPos, 0, len(m.renderQueue))
	for _, r := range m.renderQueue {
		if r.ch != nil {
			img := m.ChunkRenderer.Chunk2Img(r.ch)
			m.renderedChunks[r.pos] = img
			updatedChunks = append(updatedChunks, r.pos)
		} else {
			if img, ok := m.oldRendered[r.pos]; ok {
				m.renderedChunks[r.pos] = img
			} else {
				delete(m.renderedChunks, r.pos)
			}
		}
	}
	m.renderQueue = m.renderQueue[:0]
	return updatedChunks
}

func (m *MapUI) redraw() {
	m.redrawAt(m.w.session.Player.Position)
}

func (m *MapUI) redrawAt(position mgl32.Vec3) {
	m.mu.Lock()
	defer m.mu.Unlock()
	updatedChunks := m.processQueue()

	middle := protocol.ChunkPos{
		int32(position.X()),
		int32(position.Z()),
	}
	chunksPerLine := float64(128 / m.zoomLevel)
	pxPerBlock := 128 / chunksPerLine / 16
	pxSizeChunk := int(math.Floor(pxPerBlock * 16))

	for i := 0; i < len(m.mapImage.Pix); i++ {
		m.mapImage.Pix[i] = 0
	}
	for _ch := range m.renderedChunks {
		relativeMiddleX := float64(_ch.X()*16 - middle.X())
		relativeMiddleZ := float64(_ch.Z()*16 - middle.Z())
		px := image.Point{
			X: int(math.Floor(relativeMiddleX*pxPerBlock)) + 64,
			Y: int(math.Floor(relativeMiddleZ*pxPerBlock)) + 64,
		}

		if !m.mapImage.Rect.Intersect(image.Rect(px.X, px.Y, px.X+pxSizeChunk, px.Y+pxSizeChunk)).Empty() {
			utils.DrawImgScaledPos(m.mapImage, m.renderedChunks[_ch], px, pxSizeChunk)
		}
	}

	messages.SendEvent(&messages.EventPlayerPosition{WorldID: m.worldID, Position: position})
	var tiles []messages.MapTile
	for _, coord := range updatedChunks {
		tiles = append(tiles, messages.MapTile{
			Pos: coord,
			Img: *m.renderedChunks[coord],
		})
	}
	messages.SendEvent(&messages.EventMapTiles{
		WorldID: m.worldID, Tiles: tiles,
	})
}

func (m *MapUI) ToImage() *image.RGBA {
	return m.ToImageWhere(nil)
}

func (m *MapUI) ToImageWhere(keep func(x, z int32) bool) *image.RGBA {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.processQueue()

	var minimum, maximum protocol.ChunkPos
	found := false
	for pos := range m.renderedChunks {
		if keep != nil && !keep(pos.X(), pos.Z()) {
			continue
		}
		if !found {
			minimum, maximum, found = pos, pos, true
		} else {
			minimum[0], minimum[1] = min(minimum[0], pos[0]), min(minimum[1], pos[1])
			maximum[0], maximum[1] = max(maximum[0], pos[0]), max(maximum[1], pos[1])
		}
	}
	chunksX := int(maximum[0] - minimum[0] + 1)
	chunksY := int(maximum[1] - minimum[1] + 1)

	img := image.NewRGBA(image.Rect(0, 0, chunksX*16, chunksY*16))

	for pos, tile := range m.renderedChunks {
		if keep != nil && !keep(pos.X(), pos.Z()) {
			continue
		}
		px := image.Pt(
			int((pos.X()-minimum.X())*16),
			int((pos.Z()-minimum.Z())*16),
		)
		draw.Draw(img, image.Rect(
			px.X, px.Y,
			px.X+16, px.Y+16,
		), tile, image.Point{}, draw.Src)
	}
	return img
}

func (m *MapUI) SetChunk(pos world.ChunkPos, ch *chunk.Chunk) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.renderQueue = append(m.renderQueue, &renderElem{
		ch:  ch,
		pos: (protocol.ChunkPos)(pos),
	})
	m.SchedRedraw()
}
