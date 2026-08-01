package translate

import (
	"testing"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
)

func TestDecodeEntityEffect(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.VarInt(42)
	_ = w.VarInt(1)
	_ = w.VarInt(2)
	_ = w.VarInt(100)
	_ = w.Byte(3)

	effect, err := DecodeEntityEffect(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if effect.EntityID != 42 || effect.EffectID != 1 || effect.Amplifier != 2 || effect.Duration != 100 || effect.Flags != 3 {
		t.Fatalf("decoded effect = %+v", effect)
	}
}

func TestDecodeRemoveEntityEffect(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.VarInt(42)
	_ = w.VarInt(1)

	effect, err := DecodeRemoveEntityEffect(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if effect.EntityID != 42 || effect.EffectID != 1 {
		t.Fatalf("decoded remove effect = %+v", effect)
	}
}

func TestDecodeEntityEffectRejectsTrailingData(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.VarInt(42)
	_ = w.VarInt(1)
	_ = w.VarInt(2)
	_ = w.VarInt(100)
	_ = w.Byte(0)
	_ = w.Byte(0xff)

	if _, err := DecodeEntityEffect(w.Bytes()); err == nil {
		t.Fatal("trailing entity-effect data unexpectedly accepted")
	}
}
