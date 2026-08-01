package main

import (
	"strings"
	"testing"
)

func TestNormalizeBlockMappingUsesPaletteAliases(t *testing.T) {
	palette := map[string][]uint32{
		"minecraft:iron_chain":             {1, 2, 3},
		"minecraft:pale_oak_standing_sign": {4},
		"minecraft:skeleton_skull":         {5, 6},
		"minecraft:wither_skeleton_skull":  {7, 8},
		"minecraft:zombie_head":            {9, 10},
		"minecraft:player_head":            {11, 12},
		"minecraft:creeper_head":           {13, 14},
		"minecraft:dragon_head":            {15, 16},
		"minecraft:piglin_head":            {17, 18},
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
			name:       "zombie wall head",
			javaKey:    "minecraft:zombie_wall_head[facing=east,powered=false]",
			target:     blockMappingTarget{name: "minecraft:skull", states: map[string]any{"facing_direction": int32(5)}},
			wantName:   "minecraft:zombie_head",
			wantStates: map[string]any{"facing_direction": int32(5)},
			wantChange: true,
		},
		{
			name:       "player head",
			javaKey:    "minecraft:player_head[powered=false,rotation=0]",
			target:     blockMappingTarget{name: "minecraft:skull", states: map[string]any{"facing_direction": int32(1)}},
			wantName:   "minecraft:player_head",
			wantStates: map[string]any{"facing_direction": int32(1)},
			wantChange: true,
		},
		{
			name:       "creeper head",
			javaKey:    "minecraft:creeper_head[powered=true,rotation=0]",
			target:     blockMappingTarget{name: "minecraft:skull", states: map[string]any{"facing_direction": int32(1)}},
			wantName:   "minecraft:creeper_head",
			wantStates: map[string]any{"facing_direction": int32(1)},
			wantChange: true,
		},
		{
			name:       "dragon wall head",
			javaKey:    "minecraft:dragon_wall_head[facing=north,powered=true]",
			target:     blockMappingTarget{name: "minecraft:skull", states: map[string]any{"facing_direction": int32(2)}},
			wantName:   "minecraft:dragon_head",
			wantStates: map[string]any{"facing_direction": int32(2)},
			wantChange: true,
		},
		{
			name:       "piglin head",
			javaKey:    "minecraft:piglin_head[powered=false,rotation=15]",
			target:     blockMappingTarget{name: "minecraft:skull", states: map[string]any{"facing_direction": int32(1)}},
			wantName:   "minecraft:piglin_head",
			wantStates: map[string]any{"facing_direction": int32(1)},
			wantChange: true,
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

func TestNormalizeBedrockItemNameUsesCompletePaletteAliases(t *testing.T) {
	bedrock := map[string]int32{
		"minecraft:iron_chain": -286,
		"minecraft:diamond":    -100,
	}
	if got := normalizeBedrockItemName("minecraft:chain", bedrock); got != "minecraft:iron_chain" {
		t.Fatalf("chain item alias = %q, want minecraft:iron_chain", got)
	}
	if got := normalizeBedrockItemName("minecraft:diamond", bedrock); got != "minecraft:diamond" {
		t.Fatalf("known item name changed to %q", got)
	}
	if got := normalizeBedrockItemName("minecraft:missing", bedrock); got != "minecraft:missing" {
		t.Fatalf("unknown item name changed to %q", got)
	}
}

func TestRenderStateNamesPreservesRegistryOrder(t *testing.T) {
	output := renderStateNames([]javaState{
		{id: 0, key: "minecraft:air"},
		{id: 1, key: "minecraft:stone"},
	}, "state-hash")
	if !strings.Contains(output, "Java1214BlockStateNameCount = 2") ||
		!strings.Contains(output, `"minecraft:air", "minecraft:stone"`) {
		t.Fatalf("state-name output lost registry order: %s", output)
	}
}

func TestBuildEntityDisplayDimensionsResolvesInheritedBuilders(t *testing.T) {
	source := `
        static {
            EntityTypeBase<Entity> root = EntityTypeDefinition.baseBuilder(Entity.class).build();
            EntityTypeBase<Entity> parent = EntityTypeBase.baseInherited(Entity.class, root)
                .height(1.25f).width(0.75f).build();
            CHILD = VanillaEntityType.inherited(Entity::new, parent)
                .type(EntityType.CHILD).build();
            GRANDCHILD = VanillaEntityType.inherited(Entity::new, CHILD)
                .type(EntityType.GRANDCHILD).heightAndWidth(0.5f).build();
            OMITTED = VanillaEntityType.inherited(Entity::new, root)
                .type(EntityType.OMITTED).build(false);
        }
    `
	entries, omissions, err := buildEntityDisplayDimensions([]byte(source), []namedID{
		{id: 0, name: "minecraft:child"},
		{id: 1, name: "minecraft:grandchild"},
		{id: 2, name: "minecraft:omitted"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || len(omissions) != 1 || omissions[0].JavaIdentifier != "minecraft:omitted" {
		t.Fatalf("entries=%#v omissions=%#v", entries, omissions)
	}
	if entries[0].JavaIdentifier != "minecraft:child" || entries[0].Width != 0.75 || entries[0].Height != 1.25 {
		t.Fatalf("inherited dimensions=%#v", entries[0])
	}
	if entries[1].JavaIdentifier != "minecraft:grandchild" || entries[1].Width != 0.5 || entries[1].Height != 0.5 {
		t.Fatalf("overridden dimensions=%#v", entries[1])
	}
}
