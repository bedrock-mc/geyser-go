package translate

import (
	"testing"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestDecodeJavaOpenWindow(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.VarInt(7)
	_ = w.VarInt(javaWindowGeneric9x3)
	_ = w.Byte(8) // anonymous NBT TAG_String
	_ = w.Int16(5)
	_ = w.BytesValue([]byte("Chest"))

	open, err := DecodeJavaOpenWindow(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if open.WindowID != 7 || open.InventoryType != javaWindowGeneric9x3 || open.Title != "Chest" {
		t.Fatalf("decoded open window = %+v", open)
	}
}

func TestDecodeJavaContainerClose(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.VarInt(9)
	closeWindow, err := DecodeJavaContainerClose(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if closeWindow.WindowID != 9 {
		t.Fatalf("decoded close window = %+v", closeWindow)
	}
}

func TestJavaWindowMapping(t *testing.T) {
	window, ok := javaWindowMapping(javaWindowGeneric9x2)
	if !ok || window.containerSize != 18 || window.contentWireSize != 27 || window.bedrockType != gtprotocol.ContainerTypeContainer {
		t.Fatalf("chest mapping = %+v, ok=%t", window, ok)
	}
	if slot, ok := window.javaSlotToBedrock(17); !ok || slot != 17 {
		t.Fatalf("chest slot = %d, ok=%t", slot, ok)
	}

	window, ok = javaWindowMapping(javaWindowBrewingStand)
	if !ok {
		t.Fatal("brewing mapping missing")
	}
	if slot, ok := window.javaSlotToBedrock(0); !ok || slot != 1 || window.slotContainerID(3) != gtprotocol.ContainerBrewingStandInput {
		t.Fatalf("brewing slot mapping = slot=%d ok=%t container=%d", slot, ok, window.slotContainerID(3))
	}

	if _, ok := javaWindowMapping(999); ok {
		t.Fatal("unknown Java window type unexpectedly mapped")
	}
}

func TestJavaWindowPlayerSlot(t *testing.T) {
	if slot, ok := javaWindowPlayerSlot(27, 27); !ok || slot != 9 {
		t.Fatalf("first player inventory slot = %d, ok=%t", slot, ok)
	}
	if slot, ok := javaWindowPlayerSlot(62, 27); !ok || slot != 44 {
		t.Fatalf("last player hotbar slot = %d, ok=%t", slot, ok)
	}
	if _, ok := javaWindowPlayerSlot(63, 27); ok {
		t.Fatal("slot past player inventory unexpectedly mapped")
	}
}

func TestJavaWindowMappingUsesVersionedBlockStates(t *testing.T) {
	for _, inventoryType := range []int32{javaWindowGeneric9x3, javaWindowFurnace, javaWindowCrafter} {
		window, ok := javaWindowMapping(inventoryType)
		if !ok {
			t.Fatalf("window type %d missing", inventoryType)
		}
		if _, known := JavaBlockRuntimeID(window.javaBlockState); !known {
			t.Fatalf("window type %d block state %d is not in generated mapping", inventoryType, window.javaBlockState)
		}
	}
}
