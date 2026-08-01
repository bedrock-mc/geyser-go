package translate

import (
	"github.com/bedrock-mc/geyser-go/data"
	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
)

// javaBiomeRuntimeIDs resolves Java's configuration registry order to the
// Bedrock biome runtime IDs used by the chunk codec. Unknown or custom Java
// biomes deliberately fall back to Bedrock ocean (ID 0), matching Geyser's
// lenient registry behavior while keeping the chunk wire format valid.
func javaBiomeRuntimeIDs(configuration javaprotocol.ConfigurationData) []uint32 {
	registry, ok := configuration.Registry("minecraft:worldgen/biome")
	if !ok {
		return nil
	}
	ids := make([]uint32, len(registry.Entries))
	for i, entry := range registry.Entries {
		if !entry.HasValue {
			continue
		}
		id, ok := data.Java1214BiomeRuntimeID(entry.Key)
		if ok && id >= 0 {
			ids[i] = uint32(id)
		}
	}
	return ids
}
