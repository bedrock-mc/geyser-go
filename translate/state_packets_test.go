package translate

import (
	"errors"
	"testing"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
)

func TestDecodeHeldItemSlot(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.VarInt(4)
	update, err := DecodeHeldItemSlot(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if update.Slot != 4 {
		t.Fatalf("slot=%d, want 4", update.Slot)
	}
}

func TestDecodeEntityVelocity(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.VarInt(-12)
	_ = w.Int16(8000)
	_ = w.Int16(-4000)
	_ = w.Int16(0)
	velocity, err := DecodeEntityVelocity(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if velocity.EntityID != -12 || velocity.Velocity[0] != 1 || velocity.Velocity[1] != -0.5 {
		t.Fatalf("unexpected velocity: %#v", velocity)
	}
}

func TestDecodeEntityEquipmentTerminatedArray(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.VarInt(19)
	_ = w.Byte(0x80) // main hand, followed by another entry.
	writeJavaStoneSlot(t, w, 2)
	_ = w.Byte(4) // helmet, final entry.
	writeJavaStoneSlot(t, w, 1)
	equipment, err := DecodeEntityEquipment(w.Bytes(), func() int32 { return 7 })
	if err != nil {
		t.Fatal(err)
	}
	if equipment.EntityID != 19 || len(equipment.Items) != 2 {
		t.Fatalf("unexpected equipment: %#v", equipment)
	}
	if equipment.Items[0].Stack.Count != 2 || equipment.Items[4].Stack.Count != 1 {
		t.Fatalf("unexpected item counts: main=%d helmet=%d", equipment.Items[0].Stack.Count, equipment.Items[4].Stack.Count)
	}
}

func TestDecodeSetPlayerInventory(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.VarInt(45)
	writeJavaStoneSlot(t, w, 3)
	update, err := DecodeSetPlayerInventory(w.Bytes(), func() int32 { return 9 })
	if err != nil {
		t.Fatal(err)
	}
	if update.Slot != 45 || update.Item.Item.Stack.Count != 3 || !update.Item.Known {
		t.Fatalf("unexpected player inventory update: %#v", update)
	}
}

func TestDecodeEntityEquipmentRejectsUnsupportedComponentAsSemanticSkip(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.VarInt(19)
	_ = w.Byte(0)
	_ = w.VarInt(1) // count
	_ = w.VarInt(1) // stone
	_ = w.VarInt(1) // one added component
	_ = w.VarInt(0) // removed components
	_, err := DecodeEntityEquipment(w.Bytes(), nil)
	if !errors.Is(err, ErrUnsupportedJavaItemComponent) {
		t.Fatalf("error=%v, want unsupported component", err)
	}
}

func writeJavaStoneSlot(t *testing.T, w *javaprotocol.Writer, count int32) {
	t.Helper()
	_ = w.VarInt(count)
	_ = w.VarInt(1)
	_ = w.VarInt(0)
	_ = w.VarInt(0)
}
