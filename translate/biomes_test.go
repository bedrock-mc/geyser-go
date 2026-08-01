package translate

import (
	"testing"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
)

func TestJavaBiomeRuntimeIDsFollowRegistryOrder(t *testing.T) {
	configuration := javaprotocol.ConfigurationData{
		Registries: []javaprotocol.RegistryData{{
			ID: "minecraft:worldgen/biome",
			Entries: []javaprotocol.RegistryEntry{
				{Key: "minecraft:plains", HasValue: true},
				{Key: "minecraft:custom_biome", HasValue: true},
				{Key: "minecraft:deep_dark", HasValue: true},
			},
		}},
	}
	got := javaBiomeRuntimeIDs(configuration)
	if len(got) != 3 || got[0] != 1 || got[1] != 0 || got[2] != 190 {
		t.Fatalf("Java biome runtime IDs = %#v", got)
	}
}

func TestEncodeBedrockChunkProjectsJavaBiomes(t *testing.T) {
	biomes := make([]int32, javaBiomeSectionSize)
	biomes[0] = 0
	biomes[1] = 1
	payload, sections, err := EncodeBedrockChunkWithLayoutAndBiomes(
		JavaChunk{Sections: []JavaChunkSection{{Biomes: biomes}}},
		1000,
		javaDimensionLayout{SectionCount: 1},
		[]uint32{0, 42},
	)
	if err != nil {
		t.Fatal(err)
	}
	if sections != 1 || len(payload) < 520 {
		t.Fatalf("custom biome chunk sections=%d payload=%d", sections, len(payload))
	}
	// Empty block section: version, storage count, subchunk index.
	if payload[0] != 9 || payload[1] != 0 || payload[2] != 0 {
		t.Fatalf("empty section prefix = %v", payload[:3])
	}
	// Two biome IDs require a one-bit, runtime-ID palette over 4096 entries.
	if payload[3] != 3 || payload[516] != 4 || payload[517] != 0 || payload[518] != 84 {
		t.Fatalf("biome storage header/palette = %v", payload[3:519])
	}
}
