package translate

import (
	"testing"

	"github.com/bedrock-mc/geyser-go/data"
	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestBedrockStackSlotMapping(t *testing.T) {
	tests := []struct {
		name      string
		container byte
		slot      byte
		javaSlot  int
		cursor    bool
	}{
		{name: "cursor", container: gtprotocol.ContainerCursor, slot: 0, javaSlot: -1, cursor: true},
		{name: "hotbar", container: gtprotocol.ContainerHotBar, slot: 3, javaSlot: 39},
		{name: "inventory", container: gtprotocol.ContainerInventory, slot: 12, javaSlot: 12},
		{name: "combined hotbar", container: gtprotocol.ContainerCombinedHotBarAndInventory, slot: 5, javaSlot: 41},
		{name: "combined inventory", container: gtprotocol.ContainerCombinedHotBarAndInventory, slot: 20, javaSlot: 20},
		{name: "armor", container: gtprotocol.ContainerArmor, slot: 2, javaSlot: 7},
		{name: "offhand", container: gtprotocol.ContainerOffhand, slot: 0, javaSlot: 45},
		{name: "crafting", container: gtprotocol.ContainerCraftingInput, slot: 29, javaSlot: 2},
		{name: "created output", container: gtprotocol.ContainerCreatedOutput, slot: 0, javaSlot: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			javaSlot, cursor, ok := bedrockStackSlot(gtprotocol.StackRequestSlotInfo{
				Container: gtprotocol.FullContainerName{ContainerID: test.container},
				Slot:      test.slot,
			})
			if !ok || cursor != test.cursor || javaSlot != test.javaSlot {
				t.Fatalf("slot mapping = (%d, %t, %t), want (%d, %t, true)", javaSlot, cursor, ok, test.javaSlot, test.cursor)
			}
		})
	}

	dynamic, _, ok := bedrockStackSlot(gtprotocol.StackRequestSlotInfo{
		Container: gtprotocol.FullContainerName{
			ContainerID:        gtprotocol.ContainerInventory,
			DynamicContainerID: gtprotocol.Option(uint32(7)),
		},
		Slot: 9,
	})
	if ok || dynamic != 0 {
		t.Fatalf("dynamic container mapped to Java slot %d, want rejection", dynamic)
	}
}

func TestEncodeJavaContainerClick(t *testing.T) {
	runtimeID, ok := data.JavaItemRuntimeID(1)
	if !ok {
		t.Fatal("generated stone item mapping is missing")
	}
	item := gtprotocol.ItemInstance{
		StackNetworkID: 12,
		Stack: gtprotocol.ItemStack{
			ItemType: gtprotocol.ItemType{NetworkID: runtimeID},
			Count:    3,
		},
	}
	payload, err := encodeJavaContainerClick(17, javaInventoryClick{
		slot:       36,
		actionType: javaContainerActionClickItem,
		param:      javaClickRight,
	}, map[int]gtprotocol.ItemInstance{36: item}, gtprotocol.ItemInstance{})
	if err != nil {
		t.Fatal(err)
	}
	r := javaprotocol.NewReader(payload)
	windowID, err := r.VarInt()
	if err != nil {
		t.Fatal(err)
	}
	stateID, err := r.VarInt()
	if err != nil {
		t.Fatal(err)
	}
	slot, err := r.Int16()
	if err != nil {
		t.Fatal(err)
	}
	param, err := r.Byte()
	if err != nil {
		t.Fatal(err)
	}
	actionType, err := r.Byte()
	if err != nil {
		t.Fatal(err)
	}
	changedCount, err := r.VarInt()
	if err != nil {
		t.Fatal(err)
	}
	changedSlot, err := r.Int16()
	if err != nil {
		t.Fatal(err)
	}
	present, err := r.Bool()
	if err != nil {
		t.Fatal(err)
	}
	itemID, err := r.VarInt()
	if err != nil {
		t.Fatal(err)
	}
	count, err := r.VarInt()
	if err != nil {
		t.Fatal(err)
	}
	addedComponents, err := r.VarInt()
	if err != nil {
		t.Fatal(err)
	}
	removedComponents, err := r.VarInt()
	if err != nil {
		t.Fatal(err)
	}
	cursorPresent, err := r.Bool()
	if err != nil {
		t.Fatal(err)
	}
	if windowID != 0 || stateID != 17 || slot != 36 || param != javaClickRight || actionType != javaContainerActionClickItem || changedCount != 1 || changedSlot != 36 || !present || itemID != 1 || count != 3 || addedComponents != 0 || removedComponents != 0 || cursorPresent || r.Remaining() != 0 {
		t.Fatalf("encoded Java click = window=%d state=%d slot=%d param=%d action=%d changed=%d changedSlot=%d present=%t item=%d count=%d added=%d removed=%d cursor=%t remaining=%d", windowID, stateID, slot, param, actionType, changedCount, changedSlot, present, itemID, count, addedComponents, removedComponents, cursorPresent, r.Remaining())
	}
}

func TestApplyInventoryClicks(t *testing.T) {
	runtimeID, ok := data.JavaItemRuntimeID(1)
	if !ok {
		t.Fatal("generated stone item mapping is missing")
	}
	item := gtprotocol.ItemInstance{
		StackNetworkID: 7,
		Stack: gtprotocol.ItemStack{
			ItemType: gtprotocol.ItemType{NetworkID: runtimeID},
			Count:    4,
		},
	}
	b := NewBasic(javaprotocol.Java1214, nil)
	sim := inventorySimulation{items: b.playerItems, cursor: b.cursorItem}
	sim.items[36] = item
	changed, err := b.applyInventoryClickLocked(&sim, javaInventoryClick{slot: 36, actionType: javaContainerActionClickItem, param: javaClickLeft})
	if err != nil {
		t.Fatal(err)
	}
	if !itemEmpty(sim.items[36]) || !itemSame(sim.cursor, item) || len(changed) != 1 || !itemSame(changed[36], gtprotocol.ItemInstance{}) {
		t.Fatalf("left click simulation = slot=%+v cursor=%+v changed=%+v", sim.items[36], sim.cursor, changed)
	}

	changed, err = b.applyInventoryClickLocked(&sim, javaInventoryClick{slot: 36, actionType: javaContainerActionClickItem, param: javaClickRight})
	if err != nil {
		t.Fatal(err)
	}
	if sim.items[36].Stack.Count != 1 || sim.cursor.Stack.Count != 3 || len(changed) != 1 || changed[36].Stack.Count != 1 {
		t.Fatalf("right click simulation = slot=%+v cursor=%+v changed=%+v", sim.items[36], sim.cursor, changed)
	}

	changed, err = b.applyInventoryClickLocked(&sim, javaInventoryClick{slot: 36, actionType: javaContainerActionDropItem, param: javaClickRight})
	if err != nil {
		t.Fatal(err)
	}
	if !itemEmpty(sim.items[36]) || len(changed) != 1 || !itemEmpty(changed[36]) {
		t.Fatalf("drop simulation = slot=%+v changed=%+v", sim.items[36], changed)
	}
}
