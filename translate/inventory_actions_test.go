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

func TestResolveActiveWindowStackSlots(t *testing.T) {
	b := NewBasic(javaprotocol.Java1214, nil)
	window, ok := javaWindowMapping(javaWindowGeneric9x3)
	if !ok {
		t.Fatal("generic chest mapping missing")
	}
	window.javaID = 7
	window.bedrockID = 3
	window.items = make([]gtprotocol.ItemInstance, window.containerSize+36)
	b.windows[window.javaID] = &window
	b.bedrockWindows[window.bedrockID] = &window
	b.activeWindowID = window.bedrockID

	javaSlot, cursor, ok := b.resolveBedrockStackSlot(gtprotocol.StackRequestSlotInfo{
		Container: gtprotocol.FullContainerName{ContainerID: gtprotocol.ContainerLevelEntity},
		Slot:      0,
	})
	if !ok || cursor || javaSlot != 0 {
		t.Fatalf("active container slot = (%d, %t, %t)", javaSlot, cursor, ok)
	}
	javaSlot, cursor, ok = b.resolveBedrockStackSlot(gtprotocol.StackRequestSlotInfo{
		Container: gtprotocol.FullContainerName{ContainerID: gtprotocol.ContainerHotBar},
		Slot:      0,
	})
	if !ok || cursor || javaSlot != window.containerSize+27 {
		t.Fatalf("active hotbar slot = (%d, %t, %t)", javaSlot, cursor, ok)
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
	actionType, err := r.VarInt()
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
	itemCount, err := r.VarInt()
	if err != nil {
		t.Fatal(err)
	}
	itemID, err := r.VarInt()
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
	cursorCount, err := r.VarInt()
	if err != nil {
		t.Fatal(err)
	}
	if windowID != 0 || stateID != 17 || slot != 36 || param != javaClickRight || actionType != int32(javaContainerActionClickItem) || changedCount != 1 || changedSlot != 36 || itemCount != 3 || itemID != 1 || addedComponents != 0 || removedComponents != 0 || cursorCount != 0 || r.Remaining() != 0 {
		t.Fatalf("encoded Java click = window=%d state=%d slot=%d param=%d action=%d changed=%d changedSlot=%d itemCount=%d item=%d added=%d removed=%d cursorCount=%d remaining=%d", windowID, stateID, slot, param, actionType, changedCount, changedSlot, itemCount, itemID, addedComponents, removedComponents, cursorCount, r.Remaining())
	}
}

func TestEncodeJavaSlotPreservesCustomDataAndDamageComponent(t *testing.T) {
	runtimeID, ok := data.JavaItemRuntimeID(1)
	if !ok {
		t.Fatal("generated stone item mapping is missing")
	}
	item := gtprotocol.ItemInstance{Stack: gtprotocol.ItemStack{
		ItemType: gtprotocol.ItemType{NetworkID: runtimeID},
		Count:    2,
		NBTData: map[string]any{
			"Custom": int32(4),
			"Damage": int32(5),
		},
	}}

	w := javaprotocol.NewWriter()
	if err := writeJavaSlot(w, item); err != nil {
		t.Fatal(err)
	}
	r := javaprotocol.NewReader(w.Bytes())
	if count, err := r.VarInt(); err != nil || count != 2 {
		t.Fatalf("Java item count = %d, err=%v", count, err)
	}
	if itemID, err := r.VarInt(); err != nil || itemID != 1 {
		t.Fatalf("Java item ID = %d, err=%v", itemID, err)
	}
	componentCount, err := r.VarInt()
	if err != nil || componentCount != 2 {
		t.Fatalf("Java added component count = %d, err=%v", componentCount, err)
	}
	componentType, err := r.VarInt()
	if err != nil || componentType != javaItemComponentCustomData {
		t.Fatalf("first Java component = %d, err=%v", componentType, err)
	}
	customData, err := javaprotocol.DecodeAnonymousNBT(r)
	if err != nil {
		t.Fatal(err)
	}
	custom, ok := customData.(map[string]any)
	if !ok || custom["Custom"] != int32(4) {
		t.Fatalf("Java custom_data = %#v", customData)
	}
	componentType, err = r.VarInt()
	if err != nil || componentType != javaItemComponentDamage {
		t.Fatalf("second Java component = %d, err=%v", componentType, err)
	}
	damage, err := r.VarInt()
	if err != nil || damage != 5 {
		t.Fatalf("Java damage component = %d, err=%v", damage, err)
	}
	removed, err := r.VarInt()
	if err != nil || removed != 0 || r.Remaining() != 0 {
		t.Fatalf("Java removed components = %d, err=%v, remaining=%d", removed, err, r.Remaining())
	}
}

func TestItemIdentityIncludesProjectedDamageAndClonesNBT(t *testing.T) {
	runtimeID, ok := data.JavaItemRuntimeID(1)
	if !ok {
		t.Fatal("generated stone item mapping is missing")
	}
	item := gtprotocol.ItemInstance{StackNetworkID: 7, Stack: gtprotocol.ItemStack{
		ItemType: gtprotocol.ItemType{NetworkID: runtimeID},
		Count:    2,
		NBTData: map[string]any{
			"Custom": int32(4),
			"Damage": int32(5),
		},
	}}
	clone := cloneItem(item)
	if !sameItem(item, clone) || !itemSame(item, clone) {
		t.Fatalf("cloned item identity changed: original=%#v clone=%#v", item, clone)
	}
	clone.Stack.NBTData["Damage"] = int32(6)
	if sameItem(item, clone) || itemSame(item, clone) || item.Stack.NBTData["Damage"] != int32(5) {
		t.Fatalf("damage change was lost in item identity/clone: original=%#v clone=%#v", item, clone)
	}
	clone = cloneItem(item)
	clone.Stack.NBTData["Custom"] = int32(9)
	if sameItem(item, clone) || itemSame(item, clone) || item.Stack.NBTData["Custom"] != int32(4) {
		t.Fatalf("custom NBT change was lost in item identity/clone: original=%#v clone=%#v", item, clone)
	}
}

func TestEncodeJavaContainerClickWindowID(t *testing.T) {
	payload, err := encodeJavaContainerClick(4, javaInventoryClick{windowID: 7, slot: 0, actionType: javaContainerActionClickItem, param: javaClickLeft}, nil, gtprotocol.ItemInstance{})
	if err != nil {
		t.Fatal(err)
	}
	r := javaprotocol.NewReader(payload)
	windowID, err := r.VarInt()
	if err != nil {
		t.Fatal(err)
	}
	if windowID != 7 {
		t.Fatalf("encoded Java menu window ID = %d, want 7", windowID)
	}
}

func TestEncodeJavaContainerClose(t *testing.T) {
	payload, err := encodeJavaContainerClose(7)
	if err != nil {
		t.Fatal(err)
	}
	r := javaprotocol.NewReader(payload)
	windowID, err := r.VarInt()
	if err != nil {
		t.Fatal(err)
	}
	if windowID != 7 || r.Remaining() != 0 {
		t.Fatalf("encoded close = window=%d remaining=%d", windowID, r.Remaining())
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
	sim := inventorySimulation{items: append([]gtprotocol.ItemInstance(nil), b.playerItems[:]...), cursor: b.cursorItem}
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
