package translate

import (
	"math"
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

func TestProjectJavaHealth(t *testing.T) {
	attribute, ok := projectJavaHealth(18.5)
	if !ok {
		t.Fatal("expected finite health projection")
	}
	if attribute.Name != "minecraft:health" || attribute.Value != 19 || attribute.Max != 20 {
		t.Fatalf("health attribute = %+v", attribute)
	}
	attribute, ok = projectJavaHealth(40)
	if !ok || attribute.Value != 40 || attribute.Max != 40 {
		t.Fatalf("high health attribute = %+v, ok=%v", attribute, ok)
	}
	if _, ok := projectJavaHealth(float32(math.NaN())); ok {
		t.Fatal("NaN health should be skipped")
	}
	if _, ok := projectJavaHealth(-1); ok {
		t.Fatal("negative health should be skipped")
	}
}
