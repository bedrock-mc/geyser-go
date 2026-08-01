package translate

import (
	"strings"
	"testing"
)

func TestJavaShulkerBoxProjectsFacingForEveryJavaDirection(t *testing.T) {
	tests := []struct {
		facing string
		want   int8
	}{
		{facing: "down", want: 0},
		{facing: "up", want: 1},
		{facing: "north", want: 2},
		{facing: "south", want: 3},
		{facing: "west", want: 4},
		{facing: "east", want: 5},
	}

	for _, test := range tests {
		t.Run(test.facing, func(t *testing.T) {
			stateID := findJavaState(t, func(name string) bool {
				return strings.HasPrefix(name, "minecraft:shulker_box[") && strings.Contains(name, "facing="+test.facing+"]")
			})
			tag, ok := BedrockBlockEntityTagWithState(24, 3, 64, -5, nil, stateID)
			if !ok {
				t.Fatal("shulker block entity did not translate")
			}
			if got, ok := tag["facing"].(int8); !ok || got != test.want {
				t.Fatalf("facing = %#v, want byte %d", tag["facing"], test.want)
			}
		})
	}

	coloredState := findJavaState(t, func(name string) bool {
		return strings.HasPrefix(name, "minecraft:purple_shulker_box[") && strings.Contains(name, "facing=west]")
	})
	colored, ok := BedrockBlockEntityTagWithState(24, 3, 64, -5, nil, coloredState)
	if !ok || colored["facing"] != int8(4) {
		t.Fatalf("colored shulker facing = %#v, want byte 4", colored["facing"])
	}
}

func TestJavaShulkerBoxProjectsFacingThroughChunkPath(t *testing.T) {
	stateID := findJavaState(t, func(name string) bool {
		return name == "minecraft:shulker_box[facing=up]"
	})
	position, tag, ok := BedrockBlockEntityForChunkWithState(2, -3, JavaBlockEntity{
		X: 5, Y: 70, Z: 14, Type: 24,
	}, stateID)
	if !ok {
		t.Fatal("chunk shulker block entity did not translate")
	}
	if position[0] != 37 || position[1] != 70 || position[2] != -34 {
		t.Fatalf("chunk position = %v", position)
	}
	if tag["facing"] != int8(1) {
		t.Fatalf("chunk facing = %#v, want byte 1", tag["facing"])
	}
}

func TestJavaShulkerBoxIgnoresMissingOrOddState(t *testing.T) {
	tests := []string{
		"",
		"minecraft:shulker_box",
		"minecraft:shulker_box[facing=sideways]",
		"minecraft:stone[facing=north]",
	}
	for _, stateName := range tests {
		t.Run(stateName, func(t *testing.T) {
			tag := map[string]any{}
			projectJavaShulkerBox(tag, stateName)
			if _, ok := tag["facing"]; ok {
				t.Fatalf("odd state produced facing: %#v", tag)
			}
		})
	}
}

func TestUnsupportedItemStateTilesRemainGeneric(t *testing.T) {
	tests := []struct {
		name      string
		typeID    int32
		stateName string
		forbidden []string
	}{
		{
			name:      "jukebox",
			typeID:    4,
			stateName: "minecraft:jukebox[has_record=true]",
			forbidden: []string{"facing", "RecordItem", "record"},
		},
		{
			name:      "lectern",
			typeID:    30,
			stateName: "minecraft:lectern[facing=north,has_book=true,powered=false]",
			forbidden: []string{"facing", "book", "page", "PageCount"},
		},
		{
			name:      "chiseled_bookshelf",
			typeID:    39,
			stateName: "minecraft:chiseled_bookshelf[facing=east,slot_0_occupied=true,slot_1_occupied=false,slot_2_occupied=false,slot_3_occupied=false,slot_4_occupied=false,slot_5_occupied=false]",
			forbidden: []string{"facing", "Items"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stateID := findJavaState(t, func(name string) bool { return name == test.stateName })
			tag, ok := BedrockBlockEntityTagWithState(test.typeID, 0, 64, 0, nil, stateID)
			if !ok {
				t.Fatal("well-formed block entity did not translate")
			}
			for _, key := range test.forbidden {
				if _, exists := tag[key]; exists {
					t.Fatalf("unsupported tile guessed %q: %#v", key, tag)
				}
			}
		})
	}
}
