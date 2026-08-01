package translate

import (
	"testing"

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
