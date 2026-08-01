package translate

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/bedrock-mc/geyser-go/data"
	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

const (
	javaChunkSectionSize = 16 * 16 * 16
	javaBiomeSectionSize = 4 * 4 * 4
	maxChunkSections     = 32
	maxLightArrayBytes   = 4096
	maxChunkDataBytes    = 8 << 20
)

// JavaChunk is the typed subset of ClientboundLevelChunkWithLightPacket that
// is needed to produce a Bedrock LevelChunk. Block values retain Java's YZX
// packed index until the Bedrock encoder converts them to XZY.
type JavaChunk struct {
	X, Z          int32
	Sections      []JavaChunkSection
	BlockEntities []JavaBlockEntity
}

type JavaChunkSection struct {
	BlockCount int16
	Blocks     []int32
	Biomes     []int32
}

type JavaBlockEntity struct {
	X, Y, Z int
	Type    int32
	Data    map[string]any
}

// DecodeMapChunk decodes the Java 1.21.4 map-chunk-with-light packet. Light
// arrays are consumed and validated here; a later lighting translator can use
// them without changing the packet framing contract.
func DecodeMapChunk(payload []byte) (JavaChunk, error) {
	r := javaprotocol.NewReader(payload)
	x, err := r.Int32()
	if err != nil {
		return JavaChunk{}, fmt.Errorf("translate: chunk x: %w", err)
	}
	z, err := r.Int32()
	if err != nil {
		return JavaChunk{}, fmt.Errorf("translate: chunk z: %w", err)
	}
	var heightmaps map[string]any
	if err := nbt.NewDecoderWithEncoding(r, nbt.NetworkBigEndian).Decode(&heightmaps); err != nil {
		return JavaChunk{}, fmt.Errorf("translate: chunk heightmaps: %w", err)
	}
	chunkLength, err := r.VarInt()
	if err != nil {
		return JavaChunk{}, fmt.Errorf("translate: chunk data length: %w", err)
	}
	if chunkLength < 0 || int64(chunkLength) > int64(maxChunkDataBytes) || int64(chunkLength) > int64(r.Remaining()) {
		return JavaChunk{}, fmt.Errorf("translate: invalid chunk data length %d", chunkLength)
	}
	chunkData, err := r.Bytes(int(chunkLength))
	if err != nil {
		return JavaChunk{}, fmt.Errorf("translate: chunk data: %w", err)
	}
	sections, err := decodeJavaChunkSections(chunkData)
	if err != nil {
		return JavaChunk{}, err
	}

	entityCount, err := readCollectionCount(r, "block entity")
	if err != nil {
		return JavaChunk{}, err
	}
	entities := make([]JavaBlockEntity, 0, entityCount)
	for i := 0; i < entityCount; i++ {
		packed, err := r.Byte()
		if err != nil {
			return JavaChunk{}, fmt.Errorf("translate: block entity %d position: %w", i, err)
		}
		y, err := r.Int16()
		if err != nil {
			return JavaChunk{}, fmt.Errorf("translate: block entity %d y: %w", i, err)
		}
		entityType, err := r.VarInt()
		if err != nil {
			return JavaChunk{}, fmt.Errorf("translate: block entity %d type: %w", i, err)
		}
		entityData, err := decodeOptionalNBT(r)
		if err != nil {
			return JavaChunk{}, fmt.Errorf("translate: block entity %d NBT: %w", i, err)
		}
		entities = append(entities, JavaBlockEntity{
			X: int(packed & 0x0f), Z: int((packed >> 4) & 0x0f), Y: int(y), Type: entityType, Data: entityData,
		})
	}

	for _, name := range []string{"sky light mask", "block light mask", "empty sky light mask", "empty block light mask"} {
		if err := skipLongArray(r, name); err != nil {
			return JavaChunk{}, err
		}
	}
	for _, name := range []string{"sky light", "block light"} {
		if err := skipByteArrays(r, name); err != nil {
			return JavaChunk{}, err
		}
	}
	if r.Remaining() != 0 {
		return JavaChunk{}, fmt.Errorf("translate: map chunk has %d trailing bytes", r.Remaining())
	}
	return JavaChunk{X: x, Z: z, Sections: sections, BlockEntities: entities}, nil
}

func decodeJavaChunkSections(payload []byte) ([]JavaChunkSection, error) {
	if len(payload) == 0 {
		return nil, fmt.Errorf("translate: map chunk has no sections")
	}
	r := javaprotocol.NewReader(payload)
	sections := make([]JavaChunkSection, 0, maxChunkSections)
	for r.Remaining() > 0 {
		if len(sections) == maxChunkSections {
			return nil, fmt.Errorf("translate: map chunk has more than %d sections", maxChunkSections)
		}
		blockCount, err := r.Int16()
		if err != nil {
			return nil, fmt.Errorf("translate: chunk section %d block count: %w", len(sections), err)
		}
		blocks, err := decodeJavaPalettedContainer(r, javaChunkSectionSize)
		if err != nil {
			return nil, fmt.Errorf("translate: chunk section %d blocks: %w", len(sections), err)
		}
		biomes, err := decodeJavaPalettedContainer(r, javaBiomeSectionSize)
		if err != nil {
			return nil, fmt.Errorf("translate: chunk section %d biomes: %w", len(sections), err)
		}
		sections = append(sections, JavaChunkSection{BlockCount: blockCount, Blocks: blocks, Biomes: biomes})
	}
	return sections, nil
}

func decodeJavaPalettedContainer(r *javaprotocol.Reader, size int) ([]int32, error) {
	bitsByte, err := r.Uint8()
	if err != nil {
		return nil, err
	}
	bits := int(bitsByte)
	if bits > 32 {
		return nil, fmt.Errorf("invalid palette bits %d", bits)
	}
	var palette []int32
	if bits == 0 {
		value, err := r.VarInt()
		if err != nil {
			return nil, fmt.Errorf("singleton palette: %w", err)
		}
		palette = []int32{value}
	} else if bits <= 8 {
		paletteLength, err := readCollectionCount(r, "palette")
		if err != nil {
			return nil, err
		}
		if paletteLength == 0 {
			return nil, fmt.Errorf("palette has zero entries")
		}
		palette = make([]int32, paletteLength)
		for i := range palette {
			palette[i], err = r.VarInt()
			if err != nil {
				return nil, fmt.Errorf("palette entry %d: %w", i, err)
			}
		}
	}

	wordCount, err := readCollectionCount(r, "palette data")
	if err != nil {
		return nil, err
	}
	if bits == 0 {
		if wordCount != 0 {
			return nil, fmt.Errorf("singleton palette has %d data words", wordCount)
		}
		values := make([]int32, size)
		for i := range values {
			values[i] = palette[0]
		}
		return values, nil
	}
	valuesPerWord := 64 / bits
	if valuesPerWord == 0 {
		return nil, fmt.Errorf("palette bits %d cannot fit in a long", bits)
	}
	expectedWords := (size + valuesPerWord - 1) / valuesPerWord
	if wordCount != expectedWords {
		return nil, fmt.Errorf("palette data has %d words, expected %d", wordCount, expectedWords)
	}
	words := make([]uint64, wordCount)
	for i := range words {
		value, err := r.Int64()
		if err != nil {
			return nil, fmt.Errorf("palette data word %d: %w", i, err)
		}
		words[i] = uint64(value)
	}
	mask := uint64(1<<bits) - 1
	values := make([]int32, size)
	for i := range values {
		word := words[i/valuesPerWord]
		paletteIndex := int32((word >> (uint(i%valuesPerWord) * uint(bits))) & mask)
		if palette != nil {
			if paletteIndex < 0 || int(paletteIndex) >= len(palette) {
				return nil, fmt.Errorf("palette index %d exceeds palette length %d", paletteIndex, len(palette))
			}
			values[i] = palette[paletteIndex]
		} else {
			values[i] = paletteIndex
		}
	}
	return values, nil
}

func decodeOptionalNBT(r *javaprotocol.Reader) (map[string]any, error) {
	value, err := decodeJavaNBTValue(r)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, nil
	}
	compound, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("translate: expected optional NBT compound, got %T", value)
	}
	return compound, nil
}

func readCollectionCount(r *javaprotocol.Reader, what string) (int, error) {
	value, err := r.VarInt()
	if err != nil {
		return 0, fmt.Errorf("translate: %s count: %w", what, err)
	}
	if value < 0 || value > maxJavaCollectionSize {
		return 0, fmt.Errorf("translate: invalid %s count %d", what, value)
	}
	return int(value), nil
}

func skipLongArray(r *javaprotocol.Reader, what string) error {
	count, err := readCollectionCount(r, what)
	if err != nil {
		return err
	}
	if count > 256 {
		return fmt.Errorf("translate: %s count %d is too large", what, count)
	}
	for i := 0; i < count; i++ {
		if _, err := r.Int64(); err != nil {
			return fmt.Errorf("translate: %s entry %d: %w", what, i, err)
		}
	}
	return nil
}

func skipByteArrays(r *javaprotocol.Reader, what string) error {
	count, err := readCollectionCount(r, what)
	if err != nil {
		return err
	}
	for i := 0; i < count; i++ {
		length, err := r.VarInt()
		if err != nil {
			return fmt.Errorf("translate: %s %d length: %w", what, i, err)
		}
		if length < 0 || length > maxLightArrayBytes {
			return fmt.Errorf("translate: invalid %s %d length %d", what, i, length)
		}
		if _, err := r.Bytes(int(length)); err != nil {
			return fmt.Errorf("translate: %s %d data: %w", what, i, err)
		}
	}
	return nil
}

// EncodeBedrockChunk converts a Java chunk into the raw payload carried by a
// Gophertunnel LevelChunk packet. It emits the standard Bedrock dimension
// height for the dimension selected during Join Game and pads missing Java
// sections with empty sections.
func EncodeBedrockChunk(chunk JavaChunk, dimension int32) ([]byte, uint32, error) {
	sectionCount, minSection := bedrockDimensionSections(dimension)
	if len(chunk.Sections) > maxChunkSections {
		return nil, 0, fmt.Errorf("translate: Java chunk has %d sections", len(chunk.Sections))
	}
	airRuntimeID := data.Java1214ToBedrock[0]
	var payload bytes.Buffer
	for section := 0; section < sectionCount; section++ {
		var source JavaChunkSection
		if section < len(chunk.Sections) {
			source = chunk.Sections[section]
		}
		values := make([]uint32, javaChunkSectionSize)
		for i := range values {
			values[i] = airRuntimeID
		}
		if len(source.Blocks) == javaChunkSectionSize {
			for yzx, javaState := range source.Blocks {
				runtimeID := airRuntimeID
				if javaState >= 0 && int(javaState) < len(data.Java1214ToBedrock) {
					runtimeID = data.Java1214ToBedrock[javaState]
				}
				x := yzx & 15
				y := (yzx >> 8) & 15
				z := (yzx >> 4) & 15
				values[(x<<8)|(z<<4)|y] = runtimeID
			}
		} else if len(source.Blocks) != 0 {
			return nil, 0, fmt.Errorf("translate: Java chunk section %d has %d blocks", section, len(source.Blocks))
		}
		writeBedrockSection(&payload, values, byte(int8(minSection+section)), airRuntimeID)
	}
	// Bedrock expects one biome storage per sub-chunk. Zero is the diagnostic
	// ocean biome until Java biome registry translation is added.
	for i := 0; i < sectionCount; i++ {
		payload.WriteByte(1) // singleton palette, runtime palette
		writeSignedVarInt(&payload, 0)
	}
	payload.WriteByte(0) // Education Edition border blocks marker.
	for _, entity := range chunk.BlockEntities {
		_, tag, ok := BedrockBlockEntityForChunk(chunk.X, chunk.Z, entity)
		if !ok {
			// Unknown registry entries are semantically odd but valid Java data.
			// The caller logs the anomaly at packet level; the chunk remains usable.
			continue
		}
		encoded, err := nbt.MarshalEncoding(tag, nbt.NetworkLittleEndian)
		if err != nil {
			return nil, 0, fmt.Errorf("translate: block entity %d NBT: %w", entity.Type, err)
		}
		if _, err := payload.Write(encoded); err != nil {
			return nil, 0, fmt.Errorf("translate: block entity %d NBT: %w", entity.Type, err)
		}
	}
	return payload.Bytes(), uint32(sectionCount), nil
}

// EmptyBedrockChunkPayload is the Bedrock biome-and-border payload used with
// a LevelChunk whose SubChunkCount is zero. Bedrock still expects one biome
// storage followed by carry-forward markers for the dimension's remaining
// vertical sections.
func EmptyBedrockChunkPayload(dimension int32) []byte {
	sectionCount, _ := bedrockDimensionSections(dimension)
	payload := make([]byte, 0, sectionCount+2)
	payload = append(payload, 1, 0) // singleton biome palette, runtime ID 0
	for i := 1; i < sectionCount; i++ {
		payload = append(payload, 0xff) // (127 << 1) | 1: carry prior biome
	}
	payload = append(payload, 0) // Education Edition border blocks marker.
	return payload
}

func bedrockDimensionSections(dimension int32) (count, minSection int) {
	switch dimension {
	case 1:
		return 8, 0
	case 2:
		return 16, 0
	default:
		return 24, -4
	}
}

func writeBedrockSection(out *bytes.Buffer, values []uint32, subChunkIndex byte, airRuntimeID uint32) {
	out.WriteByte(9)
	if allAir(values, airRuntimeID) {
		out.WriteByte(0)
		out.WriteByte(subChunkIndex)
		return
	}
	out.WriteByte(1)
	out.WriteByte(subChunkIndex)
	writeBedrockStorage(out, values)
}

func allAir(values []uint32, airRuntimeID uint32) bool {
	for _, value := range values {
		if value != airRuntimeID {
			return false
		}
	}
	return true
}

func writeBedrockStorage(out *bytes.Buffer, values []uint32) {
	palette := make([]uint32, 0, 16)
	indices := make([]uint32, len(values))
	lookup := make(map[uint32]uint32, 16)
	for i, value := range values {
		index, ok := lookup[value]
		if !ok {
			index = uint32(len(palette))
			lookup[value] = index
			palette = append(palette, value)
		}
		indices[i] = index
	}
	bits := bedrockBitsForPalette(len(palette))
	out.WriteByte(byte(bits<<1) | 1)
	if bits != 0 {
		entriesPerWord := 32 / bits
		wordCount := (len(indices) + entriesPerWord - 1) / entriesPerWord
		for wordIndex := 0; wordIndex < wordCount; wordIndex++ {
			var word uint32
			for slot := 0; slot < entriesPerWord; slot++ {
				index := wordIndex*entriesPerWord + slot
				if index >= len(indices) {
					break
				}
				word |= indices[index] << (slot * bits)
			}
			var encoded [4]byte
			binary.LittleEndian.PutUint32(encoded[:], word)
			_, _ = out.Write(encoded[:])
		}
		writeSignedVarInt(out, int32(len(palette)))
	}
	for _, value := range palette {
		writeSignedVarInt(out, int32(value))
	}
}

func bedrockBitsForPalette(size int) int {
	for _, bits := range []int{1, 2, 3, 4, 5, 6, 8, 16} {
		if size <= 1<<bits {
			return bits
		}
	}
	return 16
}

func writeSignedVarInt(out io.ByteWriter, value int32) {
	encoded := uint32(value<<1) ^ uint32(value>>31)
	for encoded >= 0x80 {
		_ = out.WriteByte(byte(encoded) | 0x80)
		encoded >>= 7
	}
	_ = out.WriteByte(byte(encoded))
}
