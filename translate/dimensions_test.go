package translate

import (
	"testing"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestJavaDimensionCatalogProjectsCustomDimensions(t *testing.T) {
	configuration := javaprotocol.ConfigurationData{
		Registries: []javaprotocol.RegistryData{{
			ID: "minecraft:dimension_type",
			Entries: []javaprotocol.RegistryEntry{
				{Key: "minecraft:overworld", HasValue: true, Value: map[string]any{
					"min_y": int32(-64), "height": int32(384),
				}},
				{Key: "minecraft:skylands", HasValue: true, Value: map[string]any{
					"min_y": int32(-32), "height": int32(128),
					"effects": "minecraft:the_end",
				}},
			},
		}},
	}

	catalog := newJavaDimensionCatalog(configuration)
	if got := catalog.IDs["minecraft:overworld"]; got != 0 {
		t.Fatalf("overworld dimension ID = %d, want 0", got)
	}
	if got := catalog.IDs["minecraft:skylands"]; got != firstCustomBedrockDimension {
		t.Fatalf("custom dimension ID = %d, want %d", got, firstCustomBedrockDimension)
	}
	layout, ok := catalog.Layouts[firstCustomBedrockDimension]
	if !ok || layout != (javaDimensionLayout{SectionCount: 8, MinSection: -2}) {
		t.Fatalf("custom dimension layout = %#v, present=%t", layout, ok)
	}
	if len(catalog.Definitions) != 1 {
		t.Fatalf("custom dimension definitions = %#v", catalog.Definitions)
	}
	definition := catalog.Definitions[0]
	if definition.Name != "minecraft:skylands" || definition.Range != [2]int32{96, -32} || definition.Generator != gtprotocol.GeneratorEnd || definition.DimensionType != firstCustomBedrockDimension {
		t.Fatalf("custom dimension definition = %#v", definition)
	}
}

func TestJoinGameUsesCustomDimensionCatalog(t *testing.T) {
	configuration := javaprotocol.ConfigurationData{
		Registries: []javaprotocol.RegistryData{{
			ID: "minecraft:dimension_type",
			Entries: []javaprotocol.RegistryEntry{{
				Key: "minecraft:skylands", HasValue: true, Value: map[string]any{
					"min_y": int32(0), "height": int32(256),
				},
			}},
		}},
	}
	gameData := (JoinGame{World: SpawnInfo{Name: "minecraft:skylands"}}).GameData(nil, nil, configuration)
	if gameData.Dimension != firstCustomBedrockDimension {
		t.Fatalf("GameData dimension = %d, want %d", gameData.Dimension, firstCustomBedrockDimension)
	}
	if len(gameData.Dimensions) != 1 || gameData.Dimensions[0].Name != "minecraft:skylands" {
		t.Fatalf("GameData dimensions = %#v", gameData.Dimensions)
	}
}

func TestEncodeBedrockChunkWithCustomLayout(t *testing.T) {
	layout := javaDimensionLayout{SectionCount: 8, MinSection: -2}
	payload, sections, err := EncodeBedrockChunkWithLayout(JavaChunk{}, firstCustomBedrockDimension, layout)
	if err != nil {
		t.Fatal(err)
	}
	if sections != uint32(layout.SectionCount) {
		t.Fatalf("custom chunk sections = %d, want %d", sections, layout.SectionCount)
	}
	if len(payload) < 3 || payload[0] != 9 || payload[1] != 0 || payload[2] != 0xfe {
		t.Fatalf("custom chunk first section header = %v", payload[:minTest(len(payload), 3)])
	}
	if got := len(EmptyBedrockChunkPayloadWithLayout(layout)); got != layout.SectionCount+2 {
		t.Fatalf("custom empty chunk payload length = %d, want %d", got, layout.SectionCount+2)
	}
}
