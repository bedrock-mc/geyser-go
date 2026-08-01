package main

import "testing"

func TestNormalizeBlockMappingUsesPaletteAliases(t *testing.T) {
	palette := map[string][]uint32{
		"minecraft:iron_chain":             {1, 2, 3},
		"minecraft:pale_oak_standing_sign": {4},
		"minecraft:skeleton_skull":         {5, 6},
		"minecraft:wither_skeleton_skull":  {7, 8},
	}

	tests := []struct {
		name       string
		javaKey    string
		target     blockMappingTarget
		wantName   string
		wantStates map[string]any
		wantChange bool
	}{
		{
			name:       "iron chain alias",
			javaKey:    "minecraft:chain[axis=z,waterlogged=false]",
			target:     blockMappingTarget{name: "minecraft:chain", states: map[string]any{"pillar_axis": "z"}},
			wantName:   "minecraft:iron_chain",
			wantStates: map[string]any{"pillar_axis": "z"},
			wantChange: true,
		},
		{
			name:       "standing pale oak sign",
			javaKey:    "minecraft:pale_oak_sign[rotation=13,waterlogged=true]",
			target:     blockMappingTarget{name: "minecraft:pale_oak_sign"},
			wantName:   "minecraft:pale_oak_standing_sign",
			wantStates: map[string]any{"ground_sign_direction": int32(13)},
			wantChange: true,
		},
		{
			name:       "skeleton wall skull",
			javaKey:    "minecraft:skeleton_wall_skull[facing=east,powered=false]",
			target:     blockMappingTarget{name: "minecraft:skull", states: map[string]any{"facing_direction": int32(5)}},
			wantName:   "minecraft:skeleton_skull",
			wantStates: map[string]any{"facing_direction": int32(5)},
			wantChange: true,
		},
		{
			name:       "unsupported custom head stays explicit",
			javaKey:    "minecraft:player_head[powered=false,rotation=0]",
			target:     blockMappingTarget{name: "minecraft:skull"},
			wantName:   "minecraft:skull",
			wantStates: map[string]any{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			target := test.target
			if changed := normalizeBlockMapping(test.javaKey, &target, palette); changed != test.wantChange {
				t.Fatalf("changed=%v, want %v; target=%+v", changed, test.wantChange, target)
			}
			if target.name != test.wantName {
				t.Fatalf("name=%q, want %q", target.name, test.wantName)
			}
			if len(target.states) != len(test.wantStates) {
				t.Fatalf("states=%v, want %v", target.states, test.wantStates)
			}
			for key, want := range test.wantStates {
				if got := target.states[key]; got != want {
					t.Fatalf("state %q=%v, want %v", key, got, want)
				}
			}
		})
	}
}

func TestJavaStatePropertyIsBounded(t *testing.T) {
	if value, ok := javaStateProperty("minecraft:chain[axis=x,waterlogged=false]", "rotation"); ok || value != 0 {
		t.Fatalf("missing numeric property = (%d, %v)", value, ok)
	}
	if value, ok := javaStateProperty("minecraft:pale_oak_sign[rotation=15,waterlogged=false]", "rotation"); !ok || value != 15 {
		t.Fatalf("rotation = (%d, %v), want (15, true)", value, ok)
	}
}
