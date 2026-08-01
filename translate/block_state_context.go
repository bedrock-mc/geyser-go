package translate

import (
	"container/list"

	"github.com/bedrock-mc/geyser-go/data"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
)

// JavaChunkBlockStateAt returns the Java state stored at a world position in a
// decoded chunk. Java chunk palettes use YZX order, while the world position
// uses X/Y/Z coordinates; keeping this conversion here prevents block-entity
// translators from depending on the Bedrock runtime layout.
func JavaChunkBlockStateAt(chunk JavaChunk, position gtprotocol.BlockPos, minSection int) (int32, bool) {
	chunkOriginX := int64(chunk.X) * 16
	chunkOriginZ := int64(chunk.Z) * 16
	localX := int64(position[0]) - chunkOriginX
	localZ := int64(position[2]) - chunkOriginZ
	if localX < 0 || localX >= 16 || localZ < 0 || localZ >= 16 {
		return 0, false
	}

	sectionY := floorDiv16(position[1])
	sectionIndex := int(sectionY) - minSection
	if sectionIndex < 0 || sectionIndex >= len(chunk.Sections) {
		return 0, false
	}
	blocks := chunk.Sections[sectionIndex].Blocks
	if len(blocks) != javaChunkSectionSize {
		return 0, false
	}
	localY := floorMod16(position[1])
	index := (localY << 8) | (int(localZ) << 4) | int(localX)
	stateID := blocks[index]
	if stateID < 0 || int64(stateID) >= int64(data.Java1214BlockStateCount) {
		return 0, false
	}
	return stateID, true
}

func floorDiv16(value int32) int {
	quotient := int(value) / 16
	if value < 0 && value%16 != 0 {
		quotient--
	}
	return quotient
}

func floorMod16(value int32) int {
	modulo := int(value) % 16
	if modulo < 0 {
		modulo += 16
	}
	return modulo
}

func chunkForBlockPosition(position gtprotocol.BlockPos) gtprotocol.ChunkPos {
	return gtprotocol.ChunkPos{int32(floorDiv16(position[0])), int32(floorDiv16(position[2]))}
}

// rememberChunkBlockEntityStates replaces the block-entity state context for
// one chunk. Only positions that have block entities are retained, avoiding a
// full world-sized block-state map while still supporting standalone updates
// and later block-state changes for those positions.
func (b *Basic) rememberChunkBlockEntityStates(chunk JavaChunk, minSection int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.ensureBlockEntityStateMapsLocked()
	chunkPosition := gtprotocol.ChunkPos{chunk.X, chunk.Z}
	b.clearBlockEntityChunkLocked(chunkPosition)
	b.blockEntityChunks[chunkPosition] = make(map[gtprotocol.BlockPos]struct{})
	for _, entity := range chunk.BlockEntities {
		position, ok := chunkBlockEntityPosition(chunk.X, chunk.Z, entity)
		if !ok {
			continue
		}
		stateID, hasState := JavaChunkBlockStateAt(chunk, position, minSection)
		if !hasState {
			stateID = -1
		}
		b.blockEntityStates[position] = stateID
		if hasState {
			b.blockStates.put(position, stateID)
		}
		b.blockEntityChunks[chunkPosition][position] = struct{}{}
	}
}

func (b *Basic) rememberBlockEntityPosition(position gtprotocol.BlockPos) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.ensureBlockEntityStateMapsLocked()
	stateID, hasState := b.blockStates.get(position)
	if !hasState {
		stateID = -1
	}
	if _, exists := b.blockEntityStates[position]; !exists || hasState {
		b.blockEntityStates[position] = stateID
	}
	chunkPosition := chunkForBlockPosition(position)
	if b.blockEntityChunks[chunkPosition] == nil {
		b.blockEntityChunks[chunkPosition] = make(map[gtprotocol.BlockPos]struct{})
	}
	b.blockEntityChunks[chunkPosition][position] = struct{}{}
}

func (b *Basic) rememberBlockStateChange(position gtprotocol.BlockPos, stateID int32) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.blockStates.put(position, stateID)
	if _, tracked := b.blockEntityStates[position]; tracked {
		if stateID >= 0 && int64(stateID) < int64(data.Java1214BlockStateCount) {
			b.blockEntityStates[position] = stateID
		} else {
			b.blockEntityStates[position] = -1
		}
	}
}

func (b *Basic) cachedBlockEntityState(position gtprotocol.BlockPos) (int32, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	stateID, exists := b.blockEntityStates[position]
	if !exists || stateID < 0 {
		stateID, exists = b.blockStates.get(position)
	}
	return stateID, exists && stateID >= 0
}

func (b *Basic) forgetBlockEntityChunk(chunkX, chunkZ int32) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.blockEntityChunks == nil {
		return
	}
	b.clearBlockEntityChunkLocked(gtprotocol.ChunkPos{chunkX, chunkZ})
	b.blockStates.removeChunk(chunkX, chunkZ)
}

func (b *Basic) ensureBlockEntityStateMapsLocked() {
	if b.blockEntityStates == nil {
		b.blockEntityStates = make(map[gtprotocol.BlockPos]int32)
	}
	if b.blockEntityChunks == nil {
		b.blockEntityChunks = make(map[gtprotocol.ChunkPos]map[gtprotocol.BlockPos]struct{})
	}
}

func (b *Basic) clearBlockEntityChunkLocked(chunkPosition gtprotocol.ChunkPos) {
	positions := b.blockEntityChunks[chunkPosition]
	for position := range positions {
		delete(b.blockEntityStates, position)
	}
	delete(b.blockEntityChunks, chunkPosition)
	if positions != nil {
		b.blockEntityChunks[chunkPosition] = make(map[gtprotocol.BlockPos]struct{})
	}
}

const maxCachedJavaBlockStates = 32768

type javaBlockStateCache struct {
	values map[gtprotocol.BlockPos]*list.Element
	order  *list.List
}

type javaBlockStateCacheEntry struct {
	position gtprotocol.BlockPos
	stateID  int32
}

func (c *javaBlockStateCache) ensure() {
	if c.values == nil {
		c.values = make(map[gtprotocol.BlockPos]*list.Element)
	}
	if c.order == nil {
		c.order = list.New()
	}
}

func (c *javaBlockStateCache) put(position gtprotocol.BlockPos, stateID int32) {
	c.ensure()
	if element, exists := c.values[position]; exists {
		element.Value.(*javaBlockStateCacheEntry).stateID = stateID
		c.order.MoveToBack(element)
		return
	}
	element := c.order.PushBack(&javaBlockStateCacheEntry{position: position, stateID: stateID})
	c.values[position] = element
	for c.order.Len() > maxCachedJavaBlockStates {
		oldest := c.order.Front()
		if oldest == nil {
			break
		}
		delete(c.values, oldest.Value.(*javaBlockStateCacheEntry).position)
		c.order.Remove(oldest)
	}
}

func (c *javaBlockStateCache) get(position gtprotocol.BlockPos) (int32, bool) {
	if c.values == nil {
		return 0, false
	}
	element, exists := c.values[position]
	if !exists {
		return 0, false
	}
	c.order.MoveToBack(element)
	return element.Value.(*javaBlockStateCacheEntry).stateID, true
}

func (c *javaBlockStateCache) removeChunk(chunkX, chunkZ int32) {
	if c.values == nil {
		return
	}
	for position, element := range c.values {
		if int32(floorDiv16(position[0])) != chunkX || int32(floorDiv16(position[2])) != chunkZ {
			continue
		}
		delete(c.values, position)
		c.order.Remove(element)
	}
}
