package translate

import (
	"strings"
	"testing"

	"github.com/bedrock-mc/geyser-go/data"
	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestJavaBlockEntityRegistryAndBedrockIDRules(t *testing.T) {
	tests := []struct {
		typeID int32
		name   string
		id     string
	}{
		{0, "furnace", "Furnace"},
		{2, "trapped_chest", "Chest"},
		{11, "piston", "PistonArm"},
		{13, "enchanting_table", "EnchantTable"},
		{32, "jigsaw", "JigsawBlock"},
		{44, "vault", "Vault"},
	}
	for _, test := range tests {
		name, ok := JavaBlockEntityTypeName(test.typeID)
		if !ok || name != test.name {
			t.Fatalf("type %d = (%q, %v), want %q", test.typeID, name, ok, test.name)
		}
		tag, ok := BedrockBlockEntityTag(test.typeID, 12, 64, -3, map[string]any{"Custom": int32(7)})
		if !ok {
			t.Fatalf("type %d did not produce a tag", test.typeID)
		}
		if tag["id"] != test.id || tag["x"] != int32(12) || tag["y"] != int32(64) || tag["z"] != int32(-3) {
			t.Fatalf("type %d tag identity = %#v", test.typeID, tag)
		}
		if tag["Custom"] != int32(7) {
			t.Fatalf("type %d lost Java payload: %#v", test.typeID, tag)
		}
	}
	if _, ok := JavaBlockEntityTypeName(-1); ok {
		t.Fatal("negative block entity type unexpectedly resolved")
	}
	if _, ok := JavaBlockEntityTypeName(Java1214BlockEntityTypeCount); ok {
		t.Fatal("out-of-range block entity type unexpectedly resolved")
	}
}

func TestBedrockBlockEntityForChunkUsesWorldPosition(t *testing.T) {
	position, tag, ok := BedrockBlockEntityForChunk(2, -3, JavaBlockEntity{
		X: 5, Y: 70, Z: 14, Type: 1,
	})
	if !ok {
		t.Fatal("chunk block entity did not translate")
	}
	want := gtprotocol.BlockPos{37, 70, -34}
	if position != want {
		t.Fatalf("world position = %v, want %v", position, want)
	}
	if tag["x"] != int32(37) || tag["z"] != int32(-34) {
		t.Fatalf("tag world coordinates = %#v", tag)
	}
}

func TestBedrockSignBlockEntityProjectsJavaText(t *testing.T) {
	tag, ok := BedrockBlockEntityTag(7, 12, 64, -3, map[string]any{
		"front_text": map[string]any{
			"messages":         []string{`{"text":"Hello ","extra":[{"text":"world"}]}`, `{"translate":"chat.type.text","with":["A","B"]}`},
			"color":            "red",
			"has_glowing_text": int8(1),
		},
		"back_text": map[string]any{
			"messages":         []map[string]any{{"text": "Back"}},
			"color":            "blue",
			"has_glowing_text": false,
		},
		"is_waxed": int8(1),
	})
	if !ok {
		t.Fatal("sign block entity did not translate")
	}
	front, ok := tag["FrontText"].(map[string]any)
	if !ok {
		t.Fatalf("front text = %#v", tag["FrontText"])
	}
	if front["Text"] != "Hello world\n<A> B" || front["SignTextColor"] != int32(-5231066) || front["IgnoreLighting"] != true {
		t.Fatalf("front text projection = %#v", front)
	}
	back, ok := tag["BackText"].(map[string]any)
	if !ok || back["Text"] != "Back" || back["SignTextColor"] != int32(-12827478) || back["IgnoreLighting"] != false {
		t.Fatalf("back text projection = %#v", tag["BackText"])
	}
	if tag["IsWaxed"] != true || tag["front_text"] != nil || tag["back_text"] != nil || tag["is_waxed"] != nil {
		t.Fatalf("Java sign fields were not replaced: %#v", tag)
	}
}

func TestBedrockHangingSignUsesSameProjection(t *testing.T) {
	tag, ok := BedrockBlockEntityTag(8, 1, 2, 3, map[string]any{
		"front_text": map[string]any{"messages": []string{`"plain"`}},
	})
	if !ok {
		t.Fatal("hanging sign block entity did not translate")
	}
	front, ok := tag["FrontText"].(map[string]any)
	if !ok || front["Text"] != "plain" {
		t.Fatalf("hanging sign front text = %#v", tag["FrontText"])
	}
	if tag["id"] != "HangingSign" {
		t.Fatalf("hanging sign id = %#v", tag["id"])
	}
	empty, ok := BedrockBlockEntityTag(7, 1, 2, 3, map[string]any{
		"front_text": map[string]any{"messages": []string{`""`, `""`, `""`, `""`}},
	})
	if !ok || empty["FrontText"].(map[string]any)["Text"] != "\n\n\n" {
		t.Fatalf("empty sign lines = %#v", empty["FrontText"])
	}
}

func TestBedrockCampfireProjectsJavaItems(t *testing.T) {
	tag, ok := BedrockBlockEntityTag(33, 12, 64, -3, map[string]any{
		"Items": []map[string]any{
			{
				"Slot":  int8(0),
				"id":    "minecraft:cod",
				"count": int32(2),
				"components": map[string]any{
					"minecraft:custom_name": `{"text":"Fresh cod"}`,
					"minecraft:custom_data": map[string]any{"geyser_test": int32(1)},
				},
			},
			{
				"Slot":  int8(3),
				"id":    "minecraft:chain",
				"count": int32(4),
			},
			// These are semantically odd item records and must not create
			// out-of-range Bedrock slot fields or disconnect the session.
			{"Slot": int8(4), "id": "minecraft:stone", "count": int32(1)},
			{"Slot": int8(1), "id": "minecraft:not_an_item", "count": int32(1)},
		},
	})
	if !ok {
		t.Fatal("campfire block entity did not translate")
	}
	if _, exists := tag["Items"]; exists {
		t.Fatalf("Java campfire item list was not replaced: %#v", tag["Items"])
	}

	item, ok := tag["Item1"].(map[string]any)
	if !ok || item["Name"] != "minecraft:cod" || item["Count"] != byte(2) || item["Damage"] != int16(0) {
		t.Fatalf("campfire item 1 = %#v", tag["Item1"])
	}
	itemTag, ok := item["tag"].(map[string]any)
	if !ok || itemTag["geyser_test"] != int32(1) {
		t.Fatalf("campfire custom data = %#v", item["tag"])
	}
	display, ok := itemTag["display"].(map[string]any)
	if !ok || display["Name"] != "Fresh cod" {
		t.Fatalf("campfire custom name = %#v", itemTag["display"])
	}

	item, ok = tag["Item4"].(map[string]any)
	if !ok || item["Name"] != "minecraft:iron_chain" || item["Count"] != byte(4) {
		t.Fatalf("campfire aliased item = %#v", tag["Item4"])
	}
	if _, exists := tag["Item2"]; exists {
		t.Fatalf("unknown item unexpectedly projected: %#v", tag["Item2"])
	}
}

func TestBedrockCampfireClampsMalformedCount(t *testing.T) {
	tag, ok := BedrockBlockEntityTag(33, 0, 0, 0, map[string]any{
		"Items": []any{
			map[string]any{"Slot": int8(0), "id": "minecraft:stone", "count": int64(9000)},
		},
	})
	if !ok {
		t.Fatal("campfire block entity did not translate")
	}
	item, ok := tag["Item1"].(map[string]any)
	if !ok || item["Count"] != byte(127) {
		t.Fatalf("clamped campfire item = %#v", tag["Item1"])
	}
}

func TestBedrockBeaconNormalizesEffectIDs(t *testing.T) {
	tag, ok := BedrockBlockEntityTag(15, 0, 64, 0, map[string]any{
		"primary_effect":   "minecraft:speed",
		"secondary_effect": "minecraft:jump_boost",
	})
	if !ok {
		t.Fatal("beacon block entity did not translate")
	}
	if tag["primary"] != int32(1) || tag["secondary"] != int32(8) || tag["primary_effect"] != nil || tag["secondary_effect"] != nil {
		t.Fatalf("beacon effects = %#v", tag)
	}
	legacy, ok := BedrockBlockEntityTag(15, 0, 64, 0, map[string]any{
		"primary":   int32(-1),
		"secondary": int64(10),
	})
	if !ok || legacy["primary"] != int32(0) || legacy["secondary"] != int32(10) {
		t.Fatalf("legacy beacon effects = %#v", legacy)
	}
}

func TestBedrockEndGatewayProjectsSafeExitPortal(t *testing.T) {
	tag, ok := BedrockBlockEntityTag(22, 0, 64, 0, map[string]any{
		"Age":         int64(1 << 40),
		"exit_portal": [3]int32{12, 64, -8},
	})
	if !ok {
		t.Fatal("end gateway block entity did not translate")
	}
	if tag["Age"] != int32(1<<31-1) {
		t.Fatalf("end gateway age = %#v", tag["Age"])
	}
	exitPortal, ok := tag["ExitPortal"].([]int32)
	if !ok || len(exitPortal) != 3 || exitPortal[0] != 12 || exitPortal[1] != 64 || exitPortal[2] != -8 {
		t.Fatalf("end gateway exit portal = %#v", tag["ExitPortal"])
	}
	if _, exists := tag["exit_portal"]; exists {
		t.Fatalf("Java exit portal field was not removed: %#v", tag)
	}

	missing, ok := BedrockBlockEntityTag(22, 0, 64, 0, nil)
	if !ok {
		t.Fatal("missing exit portal block entity did not translate")
	}
	if exitPortal, ok := missing["ExitPortal"].([]int32); !ok || len(exitPortal) != 3 || exitPortal[0] != 0 || exitPortal[1] != 0 || exitPortal[2] != 0 {
		t.Fatalf("missing exit portal = %#v", missing["ExitPortal"])
	}
}

func TestBedrockDecoratedPotNormalizesSherds(t *testing.T) {
	tag, ok := BedrockBlockEntityTag(41, 0, 64, 0, map[string]any{
		"sherds": []any{"minecraft:brick", "minecraft:arms_up", int32(7)},
	})
	if !ok {
		t.Fatal("decorated pot block entity did not translate")
	}
	sherds, ok := tag["sherds"].([]string)
	if !ok || len(sherds) != 2 || sherds[0] != "minecraft:brick" || sherds[1] != "minecraft:arms_up" {
		t.Fatalf("decorated pot sherds = %#v", tag["sherds"])
	}
}

func TestBedrockMobSpawnerProjectsEntityIdentifier(t *testing.T) {
	tag, ok := BedrockBlockEntityTag(9, 0, 64, 0, map[string]any{
		"Delay":               int16(20),
		"MaxSpawnDelay":       int16(240),
		"SpawnData":           map[string]any{"entity": map[string]any{"id": "minecraft:zombie_villager"}},
		"SpawnCount":          int8(2),
		"RequiredPlayerRange": int16(16),
	})
	if !ok {
		t.Fatal("mob spawner block entity did not translate")
	}
	if tag["EntityIdentifier"] != "minecraft:zombie_villager_v2" {
		t.Fatalf("spawner entity identifier = %#v", tag["EntityIdentifier"])
	}
	if _, exists := tag["SpawnData"]; exists {
		t.Fatalf("Java spawn data was not removed: %#v", tag)
	}
	if tag["Delay"] != int16(20) || tag["SpawnCount"] != int8(2) {
		t.Fatalf("spawner timing fields changed: %#v", tag)
	}
}

func TestBedrockTrialSpawnerProjectsSpawnData(t *testing.T) {
	tag, ok := BedrockBlockEntityTag(43, 0, 64, 0, map[string]any{
		"spawn_data": map[string]any{
			"entity": map[string]any{
				"id":   "minecraft:zombie_villager",
				"Size": int32(3),
			},
		},
		"normal_cooldown": int32(40),
	})
	if !ok {
		t.Fatal("trial spawner block entity did not translate")
	}
	spawnData, ok := tag["spawn_data"].(map[string]any)
	if !ok || spawnData["TypeId"] != "minecraft:zombie_villager_v2" || spawnData["Weight"] != int32(3) {
		t.Fatalf("trial spawner spawn data = %#v", tag["spawn_data"])
	}
	if _, exists := tag["SpawnData"]; exists {
		t.Fatalf("Java trial spawner payload was not removed: %#v", tag)
	}
	if tag["normal_cooldown"] != int32(40) {
		t.Fatalf("trial spawner timing field changed: %#v", tag)
	}

	unknown, ok := BedrockBlockEntityTag(43, 0, 64, 0, map[string]any{
		"spawn_data": map[string]any{
			"entity": map[string]any{"id": "minecraft:not_a_vanilla_entity", "Size": int64(-4)},
		},
	})
	if !ok {
		t.Fatal("unknown trial spawner entity did not translate")
	}
	spawnData, ok = unknown["spawn_data"].(map[string]any)
	if !ok || spawnData["TypeId"] != nil || spawnData["Weight"] != int32(0) {
		t.Fatalf("unknown trial spawner payload = %#v", unknown["spawn_data"])
	}
}

func TestBedrockBrushableBlockProjectsItemAndState(t *testing.T) {
	state := findJavaState(t, func(name string) bool {
		return strings.HasPrefix(name, "minecraft:suspicious_sand[") && strings.Contains(name, "dusted=2")
	})
	tag, ok := BedrockBlockEntityTagWithState(40, 0, 64, 0, map[string]any{
		"item": map[string]any{
			"id":    "minecraft:diamond",
			"count": int32(2),
		},
		"hit_direction": int8(4),
	}, state)
	if !ok {
		t.Fatal("brushable block entity did not translate")
	}
	item, ok := tag["item"].(map[string]any)
	if !ok || item["Name"] != "minecraft:diamond" || item["Count"] != byte(2) {
		t.Fatalf("brushable item = %#v", tag["item"])
	}
	if tag["brush_direction"] != int8(4) || tag["brush_count"] != int32(2) || tag["type"] != "minecraft:suspicious_sand" {
		t.Fatalf("brushable state = %#v", tag)
	}
	if _, exists := tag["hit_direction"]; exists {
		t.Fatalf("Java hit direction was not replaced: %#v", tag)
	}

	retracted, ok := BedrockBlockEntityTagWithState(40, 0, 64, 0, map[string]any{
		"item":          map[string]any{"id": "minecraft:air", "count": int32(0)},
		"hit_direction": int8(-1),
	}, state)
	if !ok {
		t.Fatal("retracted brushable block entity did not translate")
	}
	if _, exists := retracted["item"]; exists {
		t.Fatalf("air brushable item unexpectedly projected: %#v", retracted)
	}
	if _, exists := retracted["brush_direction"]; exists {
		t.Fatalf("retracted brush direction unexpectedly projected: %#v", retracted)
	}
}

func TestStateAwareBlockEntityProjection(t *testing.T) {
	bannerState := findJavaState(t, func(name string) bool {
		return strings.HasPrefix(name, "minecraft:red_banner[")
	})
	banner, ok := BedrockBlockEntityTagWithState(20, 0, 64, 0, map[string]any{
		"patterns": []map[string]any{{"pattern": "minecraft:stripe_bottom", "color": "minecraft:blue"}},
	}, bannerState)
	if !ok || banner["Base"] != int32(1) {
		t.Fatalf("banner base = %#v", banner)
	}
	patterns, ok := banner["Patterns"].([]map[string]any)
	if !ok || len(patterns) != 1 || patterns[0]["Pattern"] != "bs" || patterns[0]["Color"] != int32(4) {
		t.Fatalf("banner patterns = %#v", banner["Patterns"])
	}

	skullState := findJavaState(t, func(name string) bool {
		return strings.HasPrefix(name, "minecraft:player_head[") && strings.Contains(name, "rotation=7")
	})
	skull, ok := BedrockBlockEntityTagWithState(16, 0, 64, 0, nil, skullState)
	if !ok || skull["Rotation"] != float32(157.5) {
		t.Fatalf("skull state = %#v", skull)
	}

	commandState := findJavaState(t, func(name string) bool {
		return strings.HasPrefix(name, "minecraft:command_block[") && strings.Contains(name, "conditional=true")
	})
	command, ok := BedrockBlockEntityTagWithState(23, 0, 64, 0, nil, commandState)
	if !ok || command["conditionalMode"] != true {
		t.Fatalf("command state = %#v", command)
	}

	jigsawState := findJavaState(t, func(name string) bool {
		return strings.HasPrefix(name, "minecraft:jigsaw[") && strings.Contains(name, "orientation=west_up")
	})
	jigsaw, ok := BedrockBlockEntityTagWithState(32, 0, 64, 0, map[string]any{
		"name":        "minecraft:test",
		"target_pool": "minecraft:test_pool",
		"final_state": "minecraft:air",
		"target":      "minecraft:test_target",
	}, jigsawState)
	if !ok || jigsaw["joint"] != "aligned" || jigsaw["name"] != "minecraft:test" {
		t.Fatalf("jigsaw state = %#v", jigsaw)
	}

	structure, ok := BedrockBlockEntityTag(21, 0, 64, 0, map[string]any{
		"name":            "test_structure",
		"mode":            "LOAD",
		"mirror":          "FRONT_BACK",
		"rotation":        "CLOCKWISE_90",
		"ignoreEntities":  int8(1),
		"powered":         int8(1),
		"showboundingbox": int8(1),
		"sizeX":           int32(3),
		"sizeY":           int32(4),
		"sizeZ":           int32(5),
		"posX":            int32(-1),
		"posY":            int32(2),
		"posZ":            int32(-3),
		"integrity":       float32(0.5),
	})
	if !ok || structure["structureName"] != "test_structure" || structure["data"] != int32(2) ||
		structure["mirror"] != int8(1) || structure["rotation"] != int8(1) ||
		structure["xStructureOffset"] != int32(-1) || structure["yStructureOffset"] != int32(2) ||
		structure["zStructureOffset"] != int32(-3) || structure["integrity"] != float32(0.5) {
		t.Fatalf("structure state = %#v", structure)
	}
}

func TestBedrockDoubleChestProjectsPairPosition(t *testing.T) {
	tests := []struct {
		name       string
		state      string
		pair       gtprotocol.BlockPos
		pairLeader bool
	}{
		{name: "north left", state: "minecraft:chest[facing=north,type=left,waterlogged=false]", pair: gtprotocol.BlockPos{11, 64, 20}},
		{name: "south right", state: "minecraft:chest[facing=south,type=right,waterlogged=false]", pair: gtprotocol.BlockPos{11, 64, 20}, pairLeader: true},
		{name: "east left", state: "minecraft:chest[facing=east,type=left,waterlogged=false]", pair: gtprotocol.BlockPos{10, 64, 21}},
		{name: "west right", state: "minecraft:chest[facing=west,type=right,waterlogged=false]", pair: gtprotocol.BlockPos{10, 64, 21}, pairLeader: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tag, ok := BedrockBlockEntityTagWithState(1, 10, 64, 20, nil, findJavaState(t, func(name string) bool { return name == test.state }))
			if !ok || tag["pairx"] != test.pair[0] || tag["pairz"] != test.pair[2] {
				t.Fatalf("double chest pair = %#v, want (%d,%d)", tag, test.pair[0], test.pair[2])
			}
			if test.pairLeader {
				if tag["pairlead"] != byte(1) {
					t.Fatalf("right chest pairlead = %#v", tag["pairlead"])
				}
			} else if _, exists := tag["pairlead"]; exists {
				t.Fatalf("left chest unexpectedly has pairlead: %#v", tag["pairlead"])
			}
		})
	}

	single, ok := BedrockBlockEntityTagWithState(2, 10, 64, 20, nil, findJavaState(t, func(name string) bool {
		return name == "minecraft:trapped_chest[facing=north,type=single,waterlogged=false]"
	}))
	if !ok {
		t.Fatal("single trapped chest did not translate")
	}
	if _, exists := single["pairx"]; exists {
		t.Fatalf("single trapped chest unexpectedly paired: %#v", single)
	}
}

func TestBlockEntityStateCacheUsesPriorBlockChange(t *testing.T) {
	b := NewBasic(javaprotocol.Java1214, nil)
	position := gtprotocol.BlockPos{14, 65, 0}
	stateID := findJavaState(t, func(name string) bool { return strings.HasPrefix(name, "minecraft:red_banner[") })
	b.rememberBlockStateChange(position, stateID)
	b.rememberBlockEntityPosition(position)
	cached, ok := b.cachedBlockEntityState(position)
	if !ok || cached != stateID {
		t.Fatalf("cached state = (%d, %v), want %d", cached, ok, stateID)
	}
	tag, ok := BedrockBlockEntityTagWithState(20, position[0], position[1], position[2], nil, cached)
	if !ok || tag["Base"] != int32(1) {
		t.Fatalf("cached banner tag = %#v", tag)
	}
}

func TestJavaChunkBlockStateAtUsesWorldCoordinates(t *testing.T) {
	stateID := findJavaState(t, func(name string) bool { return name == "minecraft:stone" })
	blocks := make([]int32, javaChunkSectionSize)
	blocks[(15<<8)|(3<<4)|3] = stateID
	chunk := JavaChunk{X: -2, Z: 4, Sections: []JavaChunkSection{{Blocks: blocks}}}
	position := gtprotocol.BlockPos{-29, -1, 67}
	got, ok := JavaChunkBlockStateAt(chunk, position, -1)
	if !ok || got != stateID {
		t.Fatalf("state at %v = (%d, %v), want %d", position, got, ok, stateID)
	}
}

func findJavaState(t *testing.T, predicate func(string) bool) int32 {
	t.Helper()
	for id, name := range data.Java1214BlockStateNames {
		if predicate(name) {
			return int32(id)
		}
	}
	t.Fatal("matching Java block state not found")
	return -1
}

func TestDecodeBlockEntityUpdate(t *testing.T) {
	data, err := nbt.MarshalEncoding(map[string]any{"Custom": int32(9)}, nbt.NetworkBigEndian)
	if err != nil {
		t.Fatal(err)
	}
	w := javaprotocol.NewWriter()
	if err := w.Int64(encodeJavaPosition(gtprotocol.BlockPos{-4, 63, 11})); err != nil {
		t.Fatal(err)
	}
	if err := w.VarInt(2); err != nil {
		t.Fatal(err)
	}
	if err := w.BytesValue(data); err != nil {
		t.Fatal(err)
	}
	update, err := DecodeBlockEntityUpdate(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if update.Position != (gtprotocol.BlockPos{-4, 63, 11}) || update.Type != 2 {
		t.Fatalf("decoded update = %#v", update)
	}
	if update.Data["Custom"] != int32(9) {
		t.Fatalf("decoded NBT = %#v", update.Data)
	}
}
