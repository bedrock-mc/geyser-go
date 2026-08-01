package translate

import (
	"errors"
	"reflect"
	"testing"

	"github.com/bedrock-mc/geyser-go/data"
	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
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
	_ = w.VarInt(javaItemComponentItemModel)
	_ = w.String("minecraft:test")
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

func TestDecodeJavaCursorItem(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.VarInt(2)
	_ = w.VarInt(1)
	_ = w.VarInt(0)
	_ = w.VarInt(0)
	item, err := DecodeJavaCursorItem(w.Bytes(), func() int32 { return 12 })
	if err != nil {
		t.Fatal(err)
	}
	runtimeID, ok := data.JavaItemRuntimeID(1)
	if !ok || !item.Known || item.Item.Stack.NetworkID != runtimeID || item.Item.Stack.Count != 2 || item.Item.StackNetworkID != 12 {
		t.Fatalf("decoded cursor item = %+v, known=%t", item.Item, item.Known)
	}
}

func TestDecodeJavaItemComponentsProjectsCommonData(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.VarInt(0)                           // window ID
	_ = w.VarInt(12)                          // state ID
	_ = w.VarInt(1)                           // slot count
	_ = w.VarInt(2)                           // item count
	_ = w.VarInt(1)                           // stone
	_ = w.VarInt(8)                           // added components
	_ = w.VarInt(1)                           // removed component count
	_ = w.VarInt(javaItemComponentCustomData) // custom_data
	_ = w.BytesValue(mustAnonymousItemNBT(t, map[string]any{"Custom": int32(4)}))
	_ = w.VarInt(3) // damage
	_ = w.VarInt(5)
	_ = w.VarInt(5) // custom_name
	_ = w.BytesValue(mustAnonymousItemNBT(t, map[string]any{"text": "Renamed"}))
	_ = w.VarInt(8) // lore
	_ = w.VarInt(1)
	_ = w.BytesValue(mustAnonymousItemNBT(t, map[string]any{"text": "Line one"}))
	_ = w.VarInt(10) // enchantments
	_ = w.VarInt(1)
	_ = w.VarInt(32) // Java sharpness
	_ = w.VarInt(3)
	_ = w.Bool(true)
	_ = w.VarInt(17) // repair_cost
	_ = w.VarInt(2)
	_ = w.VarInt(19) // enchantment_glint_override
	_ = w.Bool(true)
	_ = w.VarInt(34) // dyed_color
	_ = w.Int32(0x123456)
	_ = w.Bool(true)
	_ = w.VarInt(7) // removed item_model
	_ = w.VarInt(0) // carried item

	update, err := DecodeJavaWindowItems(w.Bytes(), func() int32 { return 91 })
	if err != nil {
		t.Fatal(err)
	}
	item := update.Items[0]
	if item.Stack.MetadataValue != 5 || item.StackNetworkID != 91 {
		t.Fatalf("item metadata/network ID = %d/%d", item.Stack.MetadataValue, item.StackNetworkID)
	}
	if item.Stack.NBTData["Custom"] != int32(4) || item.Stack.NBTData["RepairCost"] != int32(2) || item.Stack.NBTData["customColor"] != int32(0x123456) {
		t.Fatalf("item custom NBT = %#v", item.Stack.NBTData)
	}
	display, ok := item.Stack.NBTData["display"].(map[string]any)
	if !ok || display["Name"] != "Renamed" {
		t.Fatalf("item display = %#v", item.Stack.NBTData["display"])
	}
	if lore, ok := display["Lore"].([]string); !ok || len(lore) != 1 || lore[0] != "Line one" {
		t.Fatalf("item lore = %#v", display["Lore"])
	}
	ench, ok := item.Stack.NBTData["ench"].([]map[string]any)
	if !ok || len(ench) != 1 || ench[0]["id"] != int16(9) || ench[0]["lvl"] != int16(3) {
		t.Fatalf("item enchantments = %#v", item.Stack.NBTData["ench"])
	}
}

func TestDecodeJavaPotionContentsProjectsBedrockAuxValue(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.VarInt(1) // item count
	_ = w.VarInt(1) // stone registry ID; the component shape is what matters here
	_ = w.VarInt(1) // added component count
	_ = w.VarInt(0) // removed component count
	_ = w.VarInt(javaItemComponentPotionContents)
	_ = w.Bool(true) // potion registry holder is present
	_ = w.VarInt(18) // Java LONG_SLOWNESS ordinal -> Bedrock aux 42
	_ = w.Bool(false)
	_ = w.VarInt(0) // custom effect count
	_ = w.Bool(false)

	item, err := DecodeJavaCursorItem(w.Bytes(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !item.HasPotionID || item.PotionID != 18 || item.Item.Stack.MetadataValue != 42 {
		t.Fatalf("decoded potion item = %+v", item)
	}
}

func TestDecodeJavaFireworksComponentRetainsExplosions(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.VarInt(1)
	_ = w.VarInt(1)
	_ = w.VarInt(1)
	_ = w.VarInt(0)
	_ = w.VarInt(javaItemComponentFireworks)
	_ = w.VarInt(2) // flight duration
	_ = w.VarInt(1) // explosion count
	_ = w.VarInt(3) // shape
	_ = w.VarInt(1)
	_ = w.Int32(0xff0000)
	_ = w.VarInt(1)
	_ = w.Int32(0x00ff00)
	_ = w.Bool(true)
	_ = w.Bool(false)

	item, err := DecodeJavaCursorItem(w.Bytes(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if item.Fireworks == nil || item.Fireworks.FlightDuration != 2 || len(item.Fireworks.Explosions) != 1 {
		t.Fatalf("decoded fireworks = %+v", item.Fireworks)
	}
	explosion := item.Fireworks.Explosions[0]
	if explosion.Shape != 3 || len(explosion.Colors) != 1 || explosion.Colors[0] != 0xff0000 || len(explosion.FadeColors) != 1 || !explosion.Flicker || explosion.Trail {
		t.Fatalf("decoded firework explosion = %+v", explosion)
	}
	fireworksTag, ok := item.Item.Stack.NBTData["Fireworks"].(map[string]any)
	if !ok || fireworksTag["Flight"] != byte(2) {
		t.Fatalf("projected fireworks tag = %#v", item.Item.Stack.NBTData["Fireworks"])
	}
	explosions, ok := fireworksTag["Explosions"].([]map[string]any)
	if !ok || len(explosions) != 1 || explosions[0]["FireworkType"] != byte(3) || explosions[0]["FireworkTrail"] != false || explosions[0]["FireworkFlicker"] != true {
		t.Fatalf("projected firework explosions = %#v", fireworksTag["Explosions"])
	}
	colors := reflect.ValueOf(explosions[0]["FireworkColor"])
	if colors.Kind() != reflect.Array || colors.Len() != 1 || colors.Index(0).Uint() != 14 {
		t.Fatalf("projected firework colors = %#v", explosions[0]["FireworkColor"])
	}
	if _, err := nbt.Marshal(item.Item.Stack.NBTData); err != nil {
		t.Fatalf("marshal projected firework NBT: %v", err)
	}
}

func TestDecodeJavaItemComponentTruncationIsWireFatal(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.VarInt(1)
	_ = w.VarInt(1)
	_ = w.VarInt(1) // added component count
	_ = w.VarInt(0) // removed component count
	_ = w.VarInt(javaItemComponentDamage)
	_, err := DecodeJavaWindowItems(appendWindowItemPayload(w.Bytes()), nil)
	if err == nil || errors.Is(err, ErrUnsupportedJavaItemComponent) {
		t.Fatalf("error = %v, want a fatal truncated component decode", err)
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

func mustAnonymousItemNBT(t *testing.T, value map[string]any) []byte {
	t.Helper()
	data, err := nbt.MarshalEncoding(value, nbt.NetworkBigEndian)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
