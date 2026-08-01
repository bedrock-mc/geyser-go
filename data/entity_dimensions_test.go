package data

import "testing"

func TestJava1214EntityDisplayDimensionsResolveInheritedDefinitions(t *testing.T) {
	tests := []struct {
		name              string
		javaIdentifier    string
		bedrockIdentifier string
		width             float32
		height            float32
	}{
		{name: "mob base", javaIdentifier: "minecraft:allay", bedrockIdentifier: "minecraft:allay", width: 0.35, height: 0.6},
		{name: "minecart base", javaIdentifier: "minecraft:chest_minecart", bedrockIdentifier: "minecraft:chest_minecart", width: 0.98, height: 0.7},
		{name: "zombie inherited", javaIdentifier: "minecraft:zombie", bedrockIdentifier: "minecraft:zombie", width: 0.6, height: 1.8},
		{name: "zombie villager alias", javaIdentifier: "minecraft:zombie_villager", bedrockIdentifier: "minecraft:zombie_villager_v2", width: 0.6, height: 1.8},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := BedrockEntityDisplayDimensions(test.javaIdentifier, test.bedrockIdentifier)
			if !ok || got.Width != test.width || got.Height != test.height {
				t.Fatalf("dimensions=%#v ok=%t, want width=%v height=%v", got, ok, test.width, test.height)
			}
		})
	}

	if got, ok := BedrockEntityDisplayDimensions("", "zombie_villager_v2"); !ok || got.Width != 0.6 || got.Height != 1.8 {
		t.Fatalf("normalized alias lookup=%#v ok=%t", got, ok)
	}
	if _, ok := BedrockEntityDisplayDimensions("minecraft:marker", "minecraft:marker"); ok {
		t.Fatal("marker unexpectedly has a network display definition")
	}
	if _, ok := BedrockEntityDisplayDimensions("minecraft:not_a_vanilla_entity", "minecraft:not_a_vanilla_entity"); ok {
		t.Fatal("unknown entity unexpectedly has display dimensions")
	}
}

func TestJava1214EntityDisplayDimensionsHaveExplicitOmissions(t *testing.T) {
	if len(Java1214EntityDisplayDimensions) != 146 {
		t.Fatalf("dimension entries=%d, want 146", len(Java1214EntityDisplayDimensions))
	}
	if len(Java1214EntityDisplayDimensionOmissions) != 3 {
		t.Fatalf("dimension omissions=%d, want 3", len(Java1214EntityDisplayDimensionOmissions))
	}
	for _, identifier := range []string{"minecraft:block_display", "minecraft:item_display", "minecraft:marker"} {
		found := false
		for _, omission := range Java1214EntityDisplayDimensionOmissions {
			if omission == identifier {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing explicit omission for %s", identifier)
		}
	}
}
