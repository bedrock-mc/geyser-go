package translate

import (
	"testing"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
)

func TestJavaEntityVariantMappingsFollowNegotiatedRegistry(t *testing.T) {
	configuration := javaprotocol.ConfigurationData{
		Registries: []javaprotocol.RegistryData{
			{ID: "minecraft:wolf_variant", Entries: []javaprotocol.RegistryEntry{
				{Key: "minecraft:pale"},
				{Key: "minecraft:custom_wolf"},
				{Key: "minecraft:ashen"},
			}},
		},
	}
	mappings := newJavaEntityVariantMappings(configuration)
	if got, ok := javaEntityVariantID("minecraft:cat", 0, mappings); !ok || got != 8 {
		t.Fatalf("cat fallback mapping = %d, ok=%v", got, ok)
	}
	if got, ok := javaEntityVariantID("minecraft:wolf", 0, mappings); !ok || got != 0 {
		t.Fatalf("wolf pale mapping = %d, ok=%v", got, ok)
	}
	if got, ok := javaEntityVariantID("minecraft:wolf", 1, mappings); !ok || got != 0 {
		t.Fatalf("unknown wolf mapping = %d, ok=%v", got, ok)
	}
	if got, ok := javaEntityVariantID("minecraft:wolf", 2, mappings); !ok || got != 1 {
		t.Fatalf("wolf ashen mapping = %d, ok=%v", got, ok)
	}
}
