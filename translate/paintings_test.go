package translate

import (
	"testing"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/go-gl/mathgl/mgl32"
)

func TestJavaPaintingCatalogFollowsNegotiatedRegistry(t *testing.T) {
	configuration := javaprotocol.ConfigurationData{
		Registries: []javaprotocol.RegistryData{{
			ID: "minecraft:painting_variant",
			Entries: []javaprotocol.RegistryEntry{
				{Key: "minecraft:custom", HasValue: true, Value: map[string]any{
					"width": int32(3), "height": int32(2), "asset_id": "minecraft:aztec",
				}},
			},
		}},
	}
	catalog := javaPaintingCatalog(configuration)
	if len(catalog) != 1 || catalog[0].Width != 3 || catalog[0].Height != 2 || catalog[0].Motive != "Aztec" {
		t.Fatalf("painting catalog = %#v", catalog)
	}
}

func TestJavaPaintingPositionAndDirection(t *testing.T) {
	position := javaPaintingPosition(mgl32.Vec3{10, 20, 30}, 3, 2, 2)
	if position != (mgl32.Vec3{10.5, 20.5, 29.53125}) {
		t.Fatalf("painting position = %v", position)
	}
	for direction, want := range map[int32]int32{2: 2, 3: 0, 4: 1, 5: 3} {
		if got := javaPaintingDirection(direction); got != want {
			t.Fatalf("painting direction %d = %d, want %d", direction, got, want)
		}
	}
}
