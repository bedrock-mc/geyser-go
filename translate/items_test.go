package translate

import (
	"errors"
	"testing"

	"github.com/bedrock-mc/geyser-go/data"
	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestDecodeJavaWindowItems(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.VarInt(0)
	_ = w.VarInt(7)
	_ = w.VarInt(2)
	// Stone x3, with no added or removed components.
	_ = w.VarInt(3)
	_ = w.VarInt(1)
	_ = w.VarInt(0)
	_ = w.VarInt(0)
	// Empty second slot and empty carried item.
	_ = w.VarInt(0)
	_ = w.VarInt(0)

	next := int32(40)
	update, err := DecodeJavaWindowItems(w.Bytes(), func() int32 {
		id := next
		next++
		return id
	})
	if err != nil {
		t.Fatal(err)
	}
	if update.WindowID != 0 || update.StateID != 7 || len(update.Items) != 2 {
		t.Fatalf("decoded window metadata = %+v", update)
	}
	stoneRuntimeID, _ := data.JavaItemRuntimeID(1)
	if update.Items[0].Stack.NetworkID != stoneRuntimeID || update.Items[0].Stack.Count != 3 || update.Items[0].StackNetworkID != 40 {
		t.Fatalf("decoded stone stack = %+v", update.Items[0])
	}
	if update.Items[1].Stack.NetworkID != 0 || update.CarriedItem.Stack.NetworkID != 0 {
		t.Fatalf("expected empty slots: item=%+v carried=%+v", update.Items[1], update.CarriedItem)
	}
}

func TestDecodeJavaItemSlotRejectsUnsupportedComponents(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.VarInt(1)
	_ = w.VarInt(1)
	_ = w.VarInt(1)
	_ = w.VarInt(0)
	_, err := DecodeJavaWindowItems(appendWindowItemPayload(w.Bytes()), nil)
	if !errors.Is(err, ErrUnsupportedJavaItemComponent) {
		t.Fatalf("error = %v, want unsupported component", err)
	}
}

func TestDecodeJavaSetSlot(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.VarInt(-2)
	_ = w.VarInt(11)
	_ = w.Int16(36)
	_ = w.VarInt(1)
	_ = w.VarInt(1)
	_ = w.VarInt(0)
	_ = w.VarInt(0)
	update, err := DecodeJavaSetSlot(w.Bytes(), func() int32 { return 9 })
	if err != nil {
		t.Fatal(err)
	}
	if update.WindowID != -2 || update.StateID != 11 || update.Slot != 36 || update.Item.StackNetworkID != 9 {
		t.Fatalf("decoded set slot = %+v", update)
	}
}

func TestJavaPlayerSlotMapping(t *testing.T) {
	tests := []struct {
		javaSlot  int16
		container byte
		slot      uint32
	}{
		{9, gtprotocol.ContainerInventory, 9},
		{36, gtprotocol.ContainerHotBar, 0},
		{5, gtprotocol.ContainerArmor, 0},
		{45, gtprotocol.ContainerOffhand, 0},
		{1, gtprotocol.ContainerCraftingInput, 28},
		{0, gtprotocol.ContainerCreatedOutput, 0},
	}
	for _, test := range tests {
		container, slot, ok := javaPlayerSlot(test.javaSlot)
		if !ok || container != test.container || slot != test.slot {
			t.Fatalf("java slot %d = (%d, %d, %t), want (%d, %d, true)", test.javaSlot, container, slot, ok, test.container, test.slot)
		}
	}
}

func appendWindowItemPayload(slot []byte) []byte {
	w := javaprotocol.NewWriter()
	_ = w.VarInt(0)
	_ = w.VarInt(0)
	_ = w.VarInt(1)
	_ = w.BytesValue(slot)
	_ = w.VarInt(0)
	return w.Bytes()
}
