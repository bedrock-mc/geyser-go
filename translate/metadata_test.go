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
	if bedrock[gtprotocol.EntityDataKeyName] != "Pig" || bedrock[gtprotocol.EntityDataKeyAlwaysShowNameTag] != byte(1) {
		t.Fatalf("generic metadata not mapped: %#v", bedrock)
	}
}

func TestTranslateGenericEntityMetadataClearsOwnedFlags(t *testing.T) {
	metadata := translateGenericEntityMetadata([]JavaEntityMetadataEntry{
		{Index: 0, Type: 0, Value: int8(0)},
		{Index: 4, Type: 8, Value: false},
		{Index: 5, Type: 8, Value: true},
		{Index: 6, Type: 21, Value: javaPoseSleeping},
		{Index: 7, Type: 1, Value: int32(280)},
		{Index: 8, Type: 0, Value: int8(0x03)},
		{Index: 11, Type: 8, Value: true},
	})
	for _, flag := range []uint8{
		gtprotocol.EntityDataFlagOnFire,
		gtprotocol.EntityDataFlagSneaking,
		gtprotocol.EntityDataFlagSprinting,
		gtprotocol.EntityDataFlagSwimming,
		gtprotocol.EntityDataFlagInvisible,
		gtprotocol.EntityDataFlagGliding,
		gtprotocol.EntityDataFlagSilent,
		gtprotocol.EntityDataFlagHasGravity,
		gtprotocol.EntityDataFlagDamageNearbyMobs,
	} {
		if metadata.Flag(gtprotocol.EntityDataKeyFlags, flag) {
			t.Fatalf("generic flag %d unexpectedly set: %#v", flag, metadata)
		}
	}
	if !metadata.Flag(gtprotocol.EntityDataKeyFlagsTwo, gtprotocol.EntityDataFlagSleeping-64) || !metadata.Flag(gtprotocol.EntityDataKeyFlagsTwo, gtprotocol.EntityDataFlagEmerging-64) {
		t.Fatalf("generic high flags = %#v", metadata)
	}
	if got := metadata[gtprotocol.EntityDataKeyFreezingEffectStrength]; got != float32(1) {
		t.Fatalf("freezing strength = %#v", got)
	}
	if got := metadata[gtprotocol.EntityDataKeyEffectAmbience]; got != byte(1) {
		t.Fatalf("effect ambience = %#v", got)
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

func TestDecodeJava1214EntityMetadataTypes(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.VarInt(11)
	_ = w.Byte(7)
	_ = w.VarInt(19) // villager data
	_ = w.VarInt(1)
	_ = w.VarInt(2)
	_ = w.VarInt(3)
	_ = w.Byte(8)
	_ = w.VarInt(26) // custom painting variant holder
	_ = w.VarInt(0)
	_ = w.VarInt(2)
	_ = w.VarInt(1)
	_ = w.String("minecraft:kebab")
	_ = w.Bool(false)
	_ = w.Bool(false)
	_ = w.Byte(0xff)

	metadata, err := DecodeEntityMetadata(w.Bytes(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(metadata.Entries) != 2 {
		t.Fatalf("metadata entries = %#v", metadata.Entries)
	}
	if got, ok := metadata.Entries[0].Value.([]int32); !ok || len(got) != 3 || got[2] != 3 {
		t.Fatalf("villager metadata = %#v", metadata.Entries[0].Value)
	}
	painting, ok := metadata.Entries[1].Value.(JavaPaintingVariant)
	if !ok || !painting.Custom || painting.Width != 2 || painting.Height != 1 || painting.AssetID != "minecraft:kebab" {
		t.Fatalf("painting metadata = %#v, ok=%v", metadata.Entries[1].Value, ok)
	}
}

func TestDecodeJava1214OptionalBlockStateIsDirectVarInt(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.VarInt(3)
	_ = w.Byte(9)
	_ = w.VarInt(15)
	_ = w.VarInt(42)
	_ = w.Byte(0xff)

	metadata, err := DecodeEntityMetadata(w.Bytes(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if value, ok := metadata.Entries[0].Value.(int32); !ok || value != 42 {
		t.Fatalf("optional block state = %#v", metadata.Entries[0].Value)
	}
}

func TestDecodeJava1214WolfVariantRegistryHolder(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.VarInt(12)
	_ = w.Byte(22)
	_ = w.VarInt(23)
	_ = w.VarInt(4) // registry ordinal 3, encoded as holder ID + 1
	_ = w.Byte(0xff)

	metadata, err := DecodeEntityMetadata(w.Bytes(), nil)
	if err != nil {
		t.Fatal(err)
	}
	variant, ok := metadata.Entries[0].Value.(JavaWolfVariant)
	if !ok || variant.Custom || variant.RegistryID != 3 {
		t.Fatalf("wolf variant metadata = %#v, ok=%v", metadata.Entries[0].Value, ok)
	}
}

func TestDecodeJava1214WolfVariantCustomHolder(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.VarInt(12)
	_ = w.Byte(22)
	_ = w.VarInt(23)
	_ = w.VarInt(0) // direct custom holder
	_ = w.String("minecraft:custom_wolf")
	_ = w.String("minecraft:custom_wolf_tame")
	_ = w.String("minecraft:custom_wolf_angry")
	_ = w.VarInt(0) // named biome tag
	_ = w.String("minecraft:is_forest")
	_ = w.Byte(0xff)

	metadata, err := DecodeEntityMetadata(w.Bytes(), nil)
	if err != nil {
		t.Fatal(err)
	}
	variant, ok := metadata.Entries[0].Value.(JavaWolfVariant)
	if !ok || !variant.Custom || variant.RegistryID != -1 {
		t.Fatalf("custom wolf variant metadata = %#v, ok=%v", metadata.Entries[0].Value, ok)
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
