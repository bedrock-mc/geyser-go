package translate

import (
	"testing"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
)

func TestDecodeEntityAttributes(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.VarInt(42)
	_ = w.VarInt(1)
	_ = w.VarInt(16)
	_ = w.Float64(24)
	_ = w.VarInt(1)
	_ = w.String("test-modifier")
	_ = w.Float64(2.5)
	_ = w.Byte(1)
	update, err := DecodeEntityAttributes(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if update.EntityID != 42 || len(update.Attributes) != 1 || update.Attributes[0].Key != 16 || update.Attributes[0].Value != 24 || len(update.Attributes[0].Modifiers) != 1 {
		t.Fatalf("decoded attributes = %+v", update)
	}
	modifier := update.Attributes[0].Modifiers[0]
	if modifier.UUID != "test-modifier" || modifier.Amount != 2.5 || modifier.Operation != 1 {
		t.Fatalf("decoded modifier = %+v", modifier)
	}
}

func TestDecodeEntityAttributesRejectsTrailingData(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.VarInt(1)
	_ = w.VarInt(0)
	_ = w.Byte(0xff)
	if _, err := DecodeEntityAttributes(w.Bytes()); err == nil {
		t.Fatal("trailing attribute data unexpectedly accepted")
	}
}
