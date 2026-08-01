package data

import "testing"

func TestBedrockItemFrameStatesAreComplete(t *testing.T) {
	for _, name := range []string{"minecraft:frame", "minecraft:glow_frame"} {
		for direction := int32(0); direction <= 5; direction++ {
			runtimeID, ok := BedrockBlockRuntimeID(name, map[string]any{
				"facing_direction":     direction,
				"item_frame_map_bit":   uint8(0),
				"item_frame_photo_bit": uint8(0),
			})
			if !ok || runtimeID == 0 {
				t.Fatalf("state %s direction %d did not resolve: runtime=%d ok=%v", name, direction, runtimeID, ok)
			}
		}
	}
}

func TestBedrockItemNamesResolveGeneratedMappings(t *testing.T) {
	for _, runtimeID := range []int32{1, 2, 3} {
		if name, ok := BedrockItemName(runtimeID); !ok || name == "" {
			t.Fatalf("runtime item %d did not resolve: name=%q ok=%v", runtimeID, name, ok)
		}
	}
}
