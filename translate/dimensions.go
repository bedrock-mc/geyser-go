package translate

import (
	"sort"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

const firstCustomBedrockDimension = int32(1000)

// javaDimensionLayout is the vertical section layout used by the Bedrock
// chunk encoder. Java dimension_type values are retained as NBT, so this
// small projection is deliberately independent of Dragonfly's world model.
type javaDimensionLayout struct {
	SectionCount int
	MinSection   int
}

type javaDimensionCatalog struct {
	IDs             map[string]int32
	Layouts         map[int32]javaDimensionLayout
	Definitions     []gtprotocol.DimensionDefinition
	BiomeRuntimeIDs []uint32
}

func newJavaDimensionCatalog(configuration javaprotocol.ConfigurationData) javaDimensionCatalog {
	catalog := javaDimensionCatalog{
		IDs: map[string]int32{
			"minecraft:overworld":  packet.DimensionOverworld,
			"minecraft:the_nether": packet.DimensionNether,
			"minecraft:the_end":    packet.DimensionEnd,
		},
		Layouts:         map[int32]javaDimensionLayout{},
		BiomeRuntimeIDs: javaBiomeRuntimeIDs(configuration),
	}
	registry, ok := configuration.Registry("minecraft:dimension_type")
	if !ok {
		return catalog
	}

	type entry struct {
		name   string
		minY   int32
		height int32
		kind   int32
	}
	entries := make([]entry, 0, len(registry.Entries))
	for _, registryEntry := range registry.Entries {
		if !registryEntry.HasValue {
			continue
		}
		value, ok := registryEntry.Value.(map[string]any)
		if !ok {
			continue
		}
		minY, okMin := javaNBTInt32(value, "min_y")
		height, okHeight := javaNBTInt32(value, "height")
		if !okMin || !okHeight || height <= 0 || height%16 != 0 || minY%16 != 0 || height/16 > maxChunkSections {
			continue
		}
		kind := dimensionGenerator(value["effects"])
		entries = append(entries, entry{name: registryEntry.Key, minY: minY, height: height, kind: kind})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })

	nextID := firstCustomBedrockDimension
	for _, dimension := range entries {
		switch dimension.name {
		case "minecraft:overworld", "minecraft:the_nether", "minecraft:the_end":
			// Bedrock reserves 0-2 for these dimensions. The data-driven
			// packet cannot override their built-in definitions, so retain
			// the Java values for future validation but use the standard
			// layouts in the current vanilla path.
			continue
		}
		id := nextID
		nextID++
		catalog.IDs[dimension.name] = id
		catalog.Layouts[id] = javaDimensionLayout{
			SectionCount: int(dimension.height / 16),
			MinSection:   int(dimension.minY / 16),
		}
		catalog.Definitions = append(catalog.Definitions, gtprotocol.DimensionDefinition{
			Name:          dimension.name,
			Range:         [2]int32{dimension.minY + dimension.height, dimension.minY},
			Generator:     dimension.kind,
			DimensionType: id,
		})
	}
	return catalog
}

func dimensionGenerator(value any) int32 {
	effects, _ := value.(string)
	switch effects {
	case "minecraft:the_nether":
		return gtprotocol.GeneratorNether
	case "minecraft:the_end":
		return gtprotocol.GeneratorEnd
	default:
		return gtprotocol.GeneratorOverworld
	}
}

func javaNBTInt32(value map[string]any, key string) (int32, bool) {
	raw, ok := value[key]
	if !ok {
		return 0, false
	}
	switch value := raw.(type) {
	case int32:
		return value, true
	case int16:
		return int32(value), true
	case int64:
		if value < -1<<31 || value > 1<<31-1 {
			return 0, false
		}
		return int32(value), true
	case int:
		if value < -1<<31 || value > 1<<31-1 {
			return 0, false
		}
		return int32(value), true
	default:
		return 0, false
	}
}

func (b *Basic) javaDimensionID(name string) int32 {
	b.mu.Lock()
	dimension, ok := b.dimensionIDs[name]
	b.mu.Unlock()
	if ok {
		return dimension
	}
	return javaDimensionID(name)
}
