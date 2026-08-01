package data

import "testing"

func TestJava1214BiomeMappings(t *testing.T) {
	for name, want := range map[string]int32{
		"minecraft:ocean":        0,
		"minecraft:plains":       1,
		"minecraft:the_end":      9,
		"minecraft:deep_dark":    190,
		"minecraft:cherry_grove": 192,
	} {
		if got, ok := Java1214BiomeRuntimeID(name); !ok || got != want {
			t.Fatalf("biome %q = %d, ok=%v, want %d", name, got, ok, want)
		}
	}
	if _, ok := Java1214BiomeRuntimeID("minecraft:custom_biome"); ok {
		t.Fatal("custom biome unexpectedly reported as a vanilla mapping")
	}
}
