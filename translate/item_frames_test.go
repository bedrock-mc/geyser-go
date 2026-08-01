package translate

import (
	"testing"

	"github.com/bedrock-mc/geyser-go/data"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestJavaItemFrameProjectionUsesBedrockStateAndActorNBT(t *testing.T) {
	position := gtprotocol.BlockPos{15, 79, -262}
	itemRuntimeID, ok := data.JavaItemRuntimeID(1)
	if !ok {
		t.Fatal("stone mapping missing")
	}
	item := gtprotocol.ItemInstance{Stack: gtprotocol.ItemStack{
		ItemType: gtprotocol.ItemType{NetworkID: itemRuntimeID, MetadataValue: 2},
		Count:    1,
		NBTData: map[string]any{
			"Damage":  int32(7),
			"display": map[string]any{"Name": "Frame item"},
		},
	}}

	runtimeID, ok := javaItemFrameRuntimeID("minecraft:item_frame", 3)
	if !ok || runtimeID == 0 {
		t.Fatalf("item frame state did not resolve: runtime=%d ok=%v", runtimeID, ok)
	}
	tag := javaItemFrameTag(position, "minecraft:item_frame", item, 3)
	if tag["id"] != "ItemFrame" || tag["ItemRotation"] != float32(135) || tag["x"] != int32(15) || tag["z"] != int32(-262) {
		t.Fatalf("frame tag = %#v", tag)
	}
	itemTag, ok := tag["Item"].(map[string]any)
	if !ok || itemTag["Name"] != "minecraft:stone" || itemTag["Damage"] != int16(7) || itemTag["Count"] != byte(1) {
		t.Fatalf("frame item tag = %#v", tag["Item"])
	}
	if itemTag["tag"] == nil {
		t.Fatalf("frame item lost its Bedrock item tag: %#v", itemTag)
	}
}

func TestJavaItemFramePositionMatchesGeyserNarrowing(t *testing.T) {
	got := javaItemFramePosition([3]float32{-262.46875, 79.75, 15.5})
	if got != (gtprotocol.BlockPos{-262, 79, 15}) {
		t.Fatalf("frame position = %v", got)
	}
}
