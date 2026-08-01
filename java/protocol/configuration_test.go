package protocol

import (
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

func TestDecodeRegistryData(t *testing.T) {
	value, err := nbt.MarshalEncoding(map[string]any{
		"height": int32(384),
		"min_y":  int32(-64),
	}, nbt.NetworkBigEndian)
	if err != nil {
		t.Fatal(err)
	}
	w := NewWriter()
	_ = w.String("minecraft:dimension_type")
	_ = w.VarInt(2)
	_ = w.String("minecraft:overworld")
	_ = w.Bool(true)
	_ = w.BytesValue(value)
	_ = w.String("minecraft:custom")
	_ = w.Bool(false)

	registry, err := DecodeRegistryData(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if registry.ID != "minecraft:dimension_type" || len(registry.Entries) != 2 {
		t.Fatalf("registry = %#v", registry)
	}
	if !registry.Entries[0].HasValue {
		t.Fatal("first registry entry lost its value marker")
	}
	compound, ok := registry.Entries[0].Value.(map[string]any)
	if !ok || compound["height"] != int32(384) || compound["min_y"] != int32(-64) {
		t.Fatalf("registry NBT = %#v", registry.Entries[0].Value)
	}
	if registry.Entries[1].HasValue || registry.Entries[1].Value != nil {
		t.Fatalf("empty registry entry = %#v", registry.Entries[1])
	}
}

func TestDecodeConfigurationLists(t *testing.T) {
	flagsWriter := NewWriter()
	_ = flagsWriter.VarInt(2)
	_ = flagsWriter.String("minecraft:vanilla")
	_ = flagsWriter.String("minecraft:trade_rebalance")
	flags, err := DecodeFeatureFlags(flagsWriter.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(flags) != 2 || flags[1] != "minecraft:trade_rebalance" {
		t.Fatalf("flags = %#v", flags)
	}

	tagsWriter := NewWriter()
	_ = tagsWriter.VarInt(1)
	_ = tagsWriter.String("minecraft:block")
	_ = tagsWriter.VarInt(2)
	_ = tagsWriter.String("minecraft:buttons")
	_ = tagsWriter.VarInt(3)
	_ = tagsWriter.VarInt(4)
	_ = tagsWriter.VarInt(7)
	_ = tagsWriter.VarInt(9)
	_ = tagsWriter.String("minecraft:non_flammable_wood")
	_ = tagsWriter.VarInt(0)
	tags, err := DecodeTags(tagsWriter.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if got := tags["minecraft:block"]["minecraft:buttons"]; len(got) != 3 || got[0] != 4 || got[2] != 9 {
		t.Fatalf("tags = %#v", tags)
	}
	if len(tags["minecraft:block"]["minecraft:non_flammable_wood"]) != 0 {
		t.Fatal("empty tag should have no entries")
	}
}

func TestDecodeAnonymousNBTRejectsUnsupportedRoot(t *testing.T) {
	w := NewWriter()
	_ = w.Byte(1) // TAG_Byte; Java registry values are compound NBT here.
	if _, err := DecodeAnonymousNBT(NewReader(w.Bytes())); err == nil {
		t.Fatal("unsupported root was accepted")
	}
}

func TestConfigurationDecodersRejectTrailingWire(t *testing.T) {
	w := NewWriter()
	_ = w.VarInt(0)
	_ = w.Byte(1)
	if _, err := DecodeFeatureFlags(w.Bytes()); err == nil {
		t.Fatal("trailing feature flag bytes were accepted")
	}
}
