package translate

import (
	"errors"
	"testing"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestDecodeEntityMetadataAndMapGenericFields(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.VarInt(7)
	_ = w.Byte(0)
	_ = w.VarInt(0)
	_ = w.Byte(0x0b) // on fire, sneaking, sprinting
	_ = w.Byte(2)
	_ = w.VarInt(4)
	_ = w.String("Pig")
	_ = w.Byte(3)
	_ = w.VarInt(8)
	_ = w.Bool(true)
	_ = w.Byte(0xff)

	metadata, err := DecodeEntityMetadata(w.Bytes(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if metadata.EntityID != 7 || len(metadata.Entries) != 3 {
		t.Fatalf("unexpected metadata: %#v", metadata)
	}
	bedrock := translateGenericEntityMetadata(metadata.Entries)
	if !bedrock.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagOnFire) ||
		!bedrock.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagSneaking) ||
		!bedrock.Flag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagSprinting) {
		t.Fatalf("generic flags not mapped: %#v", bedrock)
	}
	if bedrock[gtprotocol.EntityDataKeyName] != "Pig" || bedrock[gtprotocol.EntityDataKeyAlwaysShowNameTag] != int64(1) {
		t.Fatalf("generic metadata not mapped: %#v", bedrock)
	}
}

func TestDecodeEntityMetadataSkipsUnsupportedRegistryPayload(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.VarInt(7)
	_ = w.Byte(0)
	_ = w.VarInt(17) // particle payload requires the active particle registry.
	_, err := DecodeEntityMetadata(w.Bytes(), nil)
	if !errors.Is(err, ErrUnsupportedJavaEntityMetadata) {
		t.Fatalf("error=%v, want unsupported metadata", err)
	}
}

func TestDecodeItemEntityMetadataProjectsItemStack(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.VarInt(11)
	_ = w.Byte(8) // ItemEntity's item stack metadata index.
	_ = w.VarInt(7)
	_ = w.VarInt(1) // count
	_ = w.VarInt(1) // Java item ID
	_ = w.VarInt(0) // added components
	_ = w.VarInt(0) // removed components
	_ = w.Byte(0xff)

	metadata, err := DecodeEntityMetadata(w.Bytes(), func() int32 { return 42 })
	if err != nil {
		t.Fatal(err)
	}
	if len(metadata.Entries) != 1 {
		t.Fatalf("item metadata entries = %#v", metadata.Entries)
	}
	item, ok := metadata.Entries[0].Value.(gtprotocol.ItemInstance)
	if !ok || item.Stack.ItemType.NetworkID == 0 || item.Stack.Count != 1 || item.StackNetworkID != 42 {
		t.Fatalf("item metadata value = %#v, ok=%v", metadata.Entries[0].Value, ok)
	}
	other := item
	other.StackNetworkID++
	if !sameProjectedItem(item, other) {
		t.Fatal("projected item comparison treated stack network IDs as content")
	}
	countOnly := item
	countOnly.Stack.Count++
	if !sameProjectedItemExceptCount(item, countOnly) {
		t.Fatal("projected item comparison treated a count-only update as a content change")
	}
}
