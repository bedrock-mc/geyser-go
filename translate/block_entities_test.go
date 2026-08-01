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
