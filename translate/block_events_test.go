package translate

import (
	"testing"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestDecodeJavaBlockEvent(t *testing.T) {
	w := javaprotocol.NewWriter()
	if err := w.Int64(packJavaPosition(gtprotocol.BlockPos{12, 64, -3})); err != nil {
		t.Fatal(err)
	}
	_ = w.Byte(1)
	_ = w.Byte(2)
	_ = w.VarInt(javaBlockEventChest)
	event, err := DecodeJavaBlockEvent(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if event.Position != (gtprotocol.BlockPos{12, 64, -3}) || event.Type != 1 || event.Value != 2 || event.BlockID != javaBlockEventChest {
		t.Fatalf("unexpected block event: %#v", event)
	}
}

func TestDecodeJavaBlockEventRejectsTrailingBytes(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.Int64(packJavaPosition(gtprotocol.BlockPos{}))
	_ = w.Byte(0)
	_ = w.Byte(0)
	_ = w.VarInt(javaBlockEventNoteBlock)
	_ = w.Byte(1)
	if _, err := DecodeJavaBlockEvent(w.Bytes()); err == nil {
		t.Fatal("expected trailing-byte error")
	}
}

func TestJavaBlockEventChestClassification(t *testing.T) {
	for _, id := range []int32{
		javaBlockEventChest,
		javaBlockEventEnderChest,
		javaBlockEventTrappedChest,
		javaBlockEventShulkerBoxLower,
		javaBlockEventShulkerBoxUpper,
		javaBlockEventCopperChestLower,
		javaBlockEventCopperChestUpper,
	} {
		if !javaBlockEventIsChest(id) {
			t.Fatalf("block ID %d was not classified as chest-like", id)
		}
	}
	if javaBlockEventIsChest(javaBlockEventNoteBlock) {
		t.Fatal("note block classified as chest-like")
	}
}

func packJavaPosition(pos gtprotocol.BlockPos) int64 {
	return int64((uint64(uint32(pos[0])&0x3ffffff) << 38) |
		(uint64(uint32(pos[2])&0x3ffffff) << 12) |
		uint64(uint32(pos[1])&0xfff))
}
