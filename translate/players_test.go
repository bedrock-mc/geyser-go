package translate

import (
	"testing"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
)

func TestDecodePlayerInfoUpdate(t *testing.T) {
	actions := javaPlayerInfoAddPlayer |
		javaPlayerInfoInitializeChat |
		javaPlayerInfoUpdateGameMode |
		javaPlayerInfoUpdateListed |
		javaPlayerInfoUpdateLatency |
		javaPlayerInfoUpdateDisplayName |
		javaPlayerInfoUpdateHat |
		javaPlayerInfoUpdateListOrder
	w := javaprotocol.NewWriter()
	_ = w.Byte(actions)
	_ = w.VarInt(1)
	for i := 0; i < 16; i++ {
		_ = w.Byte(byte(i))
	}
	_ = w.String("Notch")
	_ = w.VarInt(0) // profile properties
	_ = w.Bool(false)
	_ = w.VarInt(1)
	_ = w.Bool(true)
	_ = w.VarInt(42)
	_ = w.Bool(false) // no display name
	_ = w.VarInt(3)
	_ = w.Bool(true)

	update, err := DecodePlayerInfoUpdate(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if update.Actions != actions || len(update.Entries) != 1 {
		t.Fatalf("unexpected update: %#v", update)
	}
	entry := update.Entries[0]
	if entry.UUID[0] != 0 || entry.UUID[15] != 15 || entry.Name != "Notch" {
		t.Fatalf("unexpected profile: %#v", entry)
	}
	if !entry.HasGameMode || entry.GameMode != 1 || !entry.HasListed || !entry.Listed || !entry.HasLatency || entry.Latency != 42 {
		t.Fatalf("unexpected player state: %#v", entry)
	}
	if !entry.HasListOrder || entry.ListOrder != 3 || !entry.HasShowHat || !entry.ShowHat {
		t.Fatalf("unexpected player presentation state: %#v", entry)
	}
}

func TestDecodePlayerInfoRemove(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.VarInt(2)
	for i := 0; i < 32; i++ {
		_ = w.Byte(byte(i))
	}
	removed, err := DecodePlayerInfoRemove(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(removed.UUIDs) != 2 || removed.UUIDs[0][0] != 0 || removed.UUIDs[1][15] != 31 {
		t.Fatalf("unexpected removed UUIDs: %#v", removed.UUIDs)
	}
}
