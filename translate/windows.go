package translate

import (
	"fmt"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/sandertv/gophertunnel/minecraft"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// JavaOpenWindow is the 1.21.4 ClientboundOpenScreenPacket wire shape.
// InventoryType is the MenuType registry ordinal, not a Bedrock container ID.
type JavaOpenWindow struct {
	WindowID      int32
	InventoryType int32
	Title         any
}

func DecodeJavaOpenWindow(payload []byte) (JavaOpenWindow, error) {
	r := javaprotocol.NewReader(payload)
	windowID, err := r.VarInt()
	if err != nil {
		return JavaOpenWindow{}, fmt.Errorf("translate: open window ID: %w", err)
	}
	inventoryType, err := r.VarInt()
	if err != nil {
		return JavaOpenWindow{}, fmt.Errorf("translate: open window inventory type: %w", err)
	}
	title, err := decodeJavaNBTValue(r)
	if err != nil {
		return JavaOpenWindow{}, fmt.Errorf("translate: open window title: %w", err)
	}
	if r.Remaining() != 0 {
		return JavaOpenWindow{}, fmt.Errorf("translate: open window has %d trailing bytes", r.Remaining())
	}
	return JavaOpenWindow{WindowID: windowID, InventoryType: inventoryType, Title: title}, nil
}

type JavaContainerClose struct {
	WindowID int32
}

func DecodeJavaContainerClose(payload []byte) (JavaContainerClose, error) {
	r := javaprotocol.NewReader(payload)
	windowID, err := r.VarInt()
	if err != nil {
		return JavaContainerClose{}, fmt.Errorf("translate: close window ID: %w", err)
	}
	if r.Remaining() != 0 {
		return JavaContainerClose{}, fmt.Errorf("translate: close window has %d trailing bytes", r.Remaining())
	}
	return JavaContainerClose{WindowID: windowID}, nil
}

type javaWindowState struct {
	javaID          int32
	bedrockID       byte
	bedrockType     byte
	contentType     byte
	containerSize   int
	contentWireSize int
	javaBlockState  int32
	blockEntityID   string
	title           string
	position        gtprotocol.BlockPos
	stateID         int32
	items           []gtprotocol.ItemInstance
}

// Java 1.21.4 MenuType ordinals are stable within the profile and are kept
// here rather than inferred from a Java name that is not present on the wire.
const (
	javaWindowGeneric9x1 int32 = iota
	javaWindowGeneric9x2
	javaWindowGeneric9x3
	javaWindowGeneric9x4
	javaWindowGeneric9x5
	javaWindowGeneric9x6
	javaWindowGeneric3x3
	javaWindowAnvil
	javaWindowBeacon
	javaWindowBlastFurnace
	javaWindowBrewingStand
	javaWindowCrafting
	javaWindowEnchantment
	javaWindowFurnace
	javaWindowGrindstone
	javaWindowHopper
	javaWindowLectern
	javaWindowLoom
	javaWindowMerchant
	javaWindowShulkerBox
	javaWindowSmithing
	javaWindowSmoker
	javaWindowCartography
	javaWindowStonecutter
	javaWindowCrafter
)

func javaWindowMapping(inventoryType int32) (javaWindowState, bool) {
	window := javaWindowState{contentType: gtprotocol.ContainerLevelEntity}
	switch inventoryType {
	case javaWindowGeneric9x1, javaWindowGeneric9x2, javaWindowGeneric9x3,
		javaWindowGeneric9x4, javaWindowGeneric9x5, javaWindowGeneric9x6:
		rows := int(inventoryType - javaWindowGeneric9x1 + 1)
		window.bedrockType = gtprotocol.ContainerTypeContainer
		window.containerSize = rows * 9
		if rows <= 3 {
			window.contentWireSize = 27
		} else {
			window.contentWireSize = 54
		}
		window.javaBlockState = 3010 // minecraft:chest default state
		window.blockEntityID = "Chest"
	case javaWindowGeneric3x3:
		window.bedrockType = gtprotocol.ContainerTypeDispenser
		window.containerSize, window.contentWireSize = 9, 9
		window.javaBlockState = 567 // minecraft:dispenser default state
		window.blockEntityID = "Dispenser"
	case javaWindowAnvil:
		window.bedrockType = gtprotocol.ContainerTypeAnvil
		window.containerSize, window.contentWireSize = 3, 3
		window.javaBlockState = 9906
		window.blockEntityID = "" // Anvils are not block entities in Java.
	case javaWindowBeacon:
		window.bedrockType = gtprotocol.ContainerTypeBeacon
		window.containerSize, window.contentWireSize = 1, 1
		window.javaBlockState = 8692
		window.blockEntityID = "Beacon"
	case javaWindowBlastFurnace:
		window.bedrockType = gtprotocol.ContainerTypeBlastFurnace
		window.containerSize, window.contentWireSize = 3, 3
		window.javaBlockState = 19442
		window.blockEntityID = "BlastFurnace"
	case javaWindowBrewingStand:
		window.bedrockType = gtprotocol.ContainerTypeBrewingStand
		window.containerSize, window.contentWireSize = 5, 5
		window.javaBlockState = 8171
		window.blockEntityID = "BrewingStand"
	case javaWindowCrafting:
		window.bedrockType = gtprotocol.ContainerTypeWorkbench
		window.containerSize, window.contentWireSize = 10, 10
		window.javaBlockState = 4332
		window.blockEntityID = ""
	case javaWindowEnchantment:
		window.bedrockType = gtprotocol.ContainerTypeEnchantment
		window.containerSize, window.contentWireSize = 2, 2
		window.javaBlockState = 8163
		window.blockEntityID = "EnchantTable"
	case javaWindowFurnace:
		window.bedrockType = gtprotocol.ContainerTypeFurnace
		window.containerSize, window.contentWireSize = 3, 3
		window.javaBlockState = 4350
		window.blockEntityID = "Furnace"
	case javaWindowGrindstone:
		window.bedrockType = gtprotocol.ContainerTypeGrindstone
		window.containerSize, window.contentWireSize = 3, 3
		window.javaBlockState = 19455
		window.blockEntityID = ""
	case javaWindowHopper:
		window.bedrockType = gtprotocol.ContainerTypeHopper
		window.containerSize, window.contentWireSize = 5, 5
		window.javaBlockState = 10024
		window.blockEntityID = "Hopper"
	case javaWindowLectern:
		window.bedrockType = gtprotocol.ContainerTypeLectern
		window.containerSize, window.contentWireSize = 1, 1
		window.javaBlockState = 19466
		window.blockEntityID = "Lectern"
	case javaWindowLoom:
		window.bedrockType = gtprotocol.ContainerTypeLoom
		window.containerSize, window.contentWireSize = 4, 4
		window.javaBlockState = 19417
		window.blockEntityID = ""
	case javaWindowShulkerBox:
		window.bedrockType = gtprotocol.ContainerTypeContainer
		window.containerSize, window.contentWireSize = 27, 27
		window.javaBlockState = 13579
		window.blockEntityID = "ShulkerBox"
	case javaWindowSmithing:
		window.bedrockType = gtprotocol.ContainerTypeSmithingTable
		window.containerSize, window.contentWireSize = 4, 4
		window.javaBlockState = 19479
		window.blockEntityID = ""
	case javaWindowSmoker:
		window.bedrockType = gtprotocol.ContainerTypeSmoker
		window.containerSize, window.contentWireSize = 3, 3
		window.javaBlockState = 19434
		window.blockEntityID = "Smoker"
	case javaWindowCartography:
		window.bedrockType = gtprotocol.ContainerTypeCartography
		window.containerSize, window.contentWireSize = 3, 3
		window.javaBlockState = 19449
		window.blockEntityID = ""
	case javaWindowStonecutter:
		window.bedrockType = gtprotocol.ContainerTypeStonecutter
		window.containerSize, window.contentWireSize = 2, 2
		window.javaBlockState = 19480
		window.blockEntityID = ""
	case javaWindowCrafter:
		window.bedrockType = gtprotocol.ContainerTypeCrafter
		window.containerSize, window.contentWireSize = 10, 10
		window.javaBlockState = 27648
		window.blockEntityID = "Crafter"
	default:
		return javaWindowState{}, false
	}
	return window, true
}

func (w javaWindowState) javaSlotToBedrock(slot int) (uint32, bool) {
	if slot < 0 || slot >= w.containerSize {
		return 0, false
	}
	switch w.bedrockType {
	case gtprotocol.ContainerTypeBrewingStand:
		switch slot {
		case 0:
			return 1, true
		case 1:
			return 2, true
		case 2:
			return 3, true
		case 3:
			return 0, true
		case 4:
			return 4, true
		}
	case gtprotocol.ContainerTypeWorkbench:
		if slot == 0 {
			return 50, true
		}
		return uint32(slot + 31), true
	}
	return uint32(slot), true
}

func (w javaWindowState) slotContainerID(slot int) byte {
	switch w.bedrockType {
	case gtprotocol.ContainerTypeFurnace, gtprotocol.ContainerTypeBlastFurnace, gtprotocol.ContainerTypeSmoker:
		switch slot {
		case 1:
			return gtprotocol.ContainerFurnaceFuel
		case 2:
			return gtprotocol.ContainerFurnaceResult
		default:
			return gtprotocol.ContainerFurnaceIngredient
		}
	case gtprotocol.ContainerTypeBrewingStand:
		switch slot {
		case 0, 1, 2:
			return gtprotocol.ContainerBrewingStandResult
		case 3:
			return gtprotocol.ContainerBrewingStandInput
		default:
			return gtprotocol.ContainerBrewingStandFuel
		}
	case gtprotocol.ContainerTypeWorkbench:
		if slot == 0 {
			return gtprotocol.ContainerCraftingOutputPreview
		}
		return gtprotocol.ContainerCraftingInput
	case gtprotocol.ContainerTypeEnchantment:
		if slot == 0 {
			return gtprotocol.ContainerEnchantingInput
		}
		return gtprotocol.ContainerEnchantingMaterial
	case gtprotocol.ContainerTypeAnvil:
		switch slot {
		case 0:
			return gtprotocol.ContainerAnvilInput
		case 1:
			return gtprotocol.ContainerAnvilMaterial
		default:
			return gtprotocol.ContainerAnvilResultPreview
		}
	case gtprotocol.ContainerTypeGrindstone:
		switch slot {
		case 0:
			return gtprotocol.ContainerGrindstoneInput
		case 1:
			return gtprotocol.ContainerGrindstoneAdditional
		default:
			return gtprotocol.ContainerGrindstoneResultPreview
		}
	case gtprotocol.ContainerTypeSmithingTable:
		switch slot {
		case 0:
			return gtprotocol.ContainerSmithingTableTemplate
		case 1:
			return gtprotocol.ContainerSmithingTableInput
		case 2:
			return gtprotocol.ContainerSmithingTableMaterial
		default:
			return gtprotocol.ContainerSmithingTableResultPreview
		}
	case gtprotocol.ContainerTypeStonecutter:
		if slot == 0 {
			return gtprotocol.ContainerStonecutterInput
		}
		return gtprotocol.ContainerStonecutterResultPreview
	case gtprotocol.ContainerTypeCartography:
		switch slot {
		case 0:
			return gtprotocol.ContainerCartographyInput
		case 1:
			return gtprotocol.ContainerCartographyAdditional
		default:
			return gtprotocol.ContainerCartographyResultPreview
		}
	}
	return w.contentType
}

func (b *Basic) translateOpenWindow(bedrock *minecraft.Conn, java *javaprotocol.Client, open JavaOpenWindow) error {
	if open.WindowID <= 0 {
		b.logSemanticAnomaly("skipping Java open window with an invalid semantic ID", "window", open.WindowID)
		return nil
	}
	mapping, ok := javaWindowMapping(open.InventoryType)
	if !ok {
		b.logSemanticAnomaly("skipping unsupported Java window type", "window", open.WindowID, "type", open.InventoryType)
		data, err := encodeJavaContainerClose(open.WindowID)
		if err != nil {
			return err
		}
		return java.Conn.WritePacket(b.Profile.PlayServerboundContainerCloseID, data)
	}

	// Bedrock has one active block-backed UI per session. Close a stale holder
	// before replacing it, while preserving the Java window ID in the close
	// packet if the IDs need to be remapped in a later protocol revision.
	b.mu.Lock()
	var oldWindow *javaWindowState
	if old := b.windows[open.WindowID]; old != nil {
		oldCopy := *old
		oldWindow = &oldCopy
		delete(b.windows, old.javaID)
		delete(b.bedrockWindows, old.bedrockID)
	}
	bedrockID := b.allocateWindowIDLocked()
	mapping.javaID = open.WindowID
	mapping.bedrockID = bedrockID
	mapping.title = JavaTextComponentText(open.Title)
	if mapping.title == "" {
		mapping.title = "Container"
	}
	mapping.position = virtualWindowPositionLocked(b.position)
	b.windows[open.WindowID] = &mapping
	b.bedrockWindows[bedrockID] = &mapping
	b.activeWindowID = bedrockID
	b.mu.Unlock()
	if oldWindow != nil {
		if err := b.closeVirtualWindow(bedrock, oldWindow); err != nil {
			return err
		}
	}

	runtimeID, known := JavaBlockRuntimeID(mapping.javaBlockState)
	if !known {
		b.logSemanticAnomaly("virtual Java window block state is outside the generated registry", "state", mapping.javaBlockState)
		return nil
	}
	if err := bedrock.WritePacket(&packet.UpdateBlock{
		Position:          mapping.position,
		NewBlockRuntimeID: runtimeID,
		Flags:             packet.BlockUpdateNeighbours | packet.BlockUpdateNetwork | packet.BlockUpdatePriority,
		Layer:             0,
	}); err != nil {
		return err
	}
	if mapping.blockEntityID != "" {
		tag := map[string]any{
			"x":          mapping.position[0],
			"y":          mapping.position[1],
			"z":          mapping.position[2],
			"id":         mapping.blockEntityID,
			"CustomName": mapping.title,
		}
		if err := bedrock.WritePacket(&packet.BlockActorData{Position: mapping.position, NBTData: tag}); err != nil {
			return err
		}
	}
	return bedrock.WritePacket(&packet.ContainerOpen{
		WindowID:                mapping.bedrockID,
		ContainerType:           mapping.bedrockType,
		ContainerPosition:       mapping.position,
		ContainerEntityUniqueID: 0,
	})
}

func (b *Basic) allocateWindowIDLocked() byte {
	for i := 0; i < 255; i++ {
		candidate := b.nextWindowID
		if candidate == 0 {
			candidate = 1
		}
		b.nextWindowID = candidate + 1
		if b.nextWindowID == 0 {
			b.nextWindowID = 1
		}
		if _, exists := b.bedrockWindows[candidate]; !exists {
			return candidate
		}
	}
	return 1
}

func virtualWindowPositionLocked(position javaPosition) gtprotocol.BlockPos {
	block := floorBlockPosition(position)
	block[0] += 2
	block[1] += 1
	return block
}

func (b *Basic) translateJavaWindowClose(bedrock *minecraft.Conn, javaID int32) error {
	b.mu.Lock()
	window := b.windows[javaID]
	if window != nil {
		delete(b.windows, window.javaID)
		delete(b.bedrockWindows, window.bedrockID)
		if b.activeWindowID == window.bedrockID {
			b.activeWindowID = 0
		}
	}
	b.mu.Unlock()
	if window == nil {
		return nil
	}
	return b.closeVirtualWindow(bedrock, window)
}

func (b *Basic) translateBedrockContainerClose(bedrock *minecraft.Conn, java *javaprotocol.Client, closePacket *packet.ContainerClose) error {
	if closePacket.WindowID == 0 {
		return nil
	}
	b.mu.Lock()
	window := b.bedrockWindows[closePacket.WindowID]
	if window != nil {
		delete(b.windows, window.javaID)
		delete(b.bedrockWindows, window.bedrockID)
		if b.activeWindowID == window.bedrockID {
			b.activeWindowID = 0
		}
	}
	b.mu.Unlock()
	javaID := int32(closePacket.WindowID)
	if window != nil {
		javaID = window.javaID
	}
	data, err := encodeJavaContainerClose(javaID)
	if err != nil {
		return err
	}
	if err := java.Conn.WritePacket(b.Profile.PlayServerboundContainerCloseID, data); err != nil {
		return err
	}
	if window == nil {
		return nil
	}
	return b.restoreVirtualWindow(bedrock, window)
}

func (b *Basic) closeVirtualWindow(bedrock *minecraft.Conn, window *javaWindowState) error {
	if err := bedrock.WritePacket(&packet.ContainerClose{
		WindowID:      window.bedrockID,
		ContainerType: window.bedrockType,
		ServerSide:    true,
	}); err != nil {
		return err
	}
	return b.restoreVirtualWindow(bedrock, window)
}

func (b *Basic) restoreVirtualWindow(bedrock *minecraft.Conn, window *javaWindowState) error {
	air, known := JavaBlockRuntimeID(0)
	if !known {
		return fmt.Errorf("translate: generated air block state is unavailable")
	}
	return bedrock.WritePacket(&packet.UpdateBlock{
		Position:          window.position,
		NewBlockRuntimeID: air,
		Flags:             packet.BlockUpdateNeighbours | packet.BlockUpdateNetwork | packet.BlockUpdatePriority,
		Layer:             0,
	})
}

func (b *Basic) translateWindowContent(bedrock *minecraft.Conn, update JavaWindowItems) error {
	b.mu.Lock()
	window := b.windows[update.WindowID]
	if window == nil {
		b.mu.Unlock()
		b.logSemanticAnomaly("skipping content for unknown Java window", "window", update.WindowID)
		return nil
	}
	window.stateID = update.StateID
	window.items = make([]gtprotocol.ItemInstance, window.containerSize+36)
	copy(window.items, update.Items)
	b.cursorItem = update.CarriedItem
	windowSnapshot := *window
	b.updatePlayerInventoryFromWindowLocked(update.Items, window.containerSize)
	b.mu.Unlock()

	content := make([]gtprotocol.ItemInstance, windowSnapshot.contentWireSize)
	for slot, item := range update.Items {
		if slot >= windowSnapshot.containerSize {
			break
		}
		bedrockSlot, ok := windowSnapshot.javaSlotToBedrock(slot)
		if ok && int(bedrockSlot) < len(content) {
			content[bedrockSlot] = item
		}
	}
	if err := bedrock.WritePacket(&packet.InventoryContent{
		WindowID:  uint32(windowSnapshot.bedrockID),
		Content:   content,
		Container: gtprotocol.FullContainerName{ContainerID: windowSnapshot.contentType},
	}); err != nil {
		return err
	}
	// UI menus such as crafting use Bedrock slot indices outside the compact
	// content array. Send typed slot updates as well, matching Geyser's UI
	// updater behavior while the dedicated stack-request translators are built.
	for slot, item := range update.Items {
		if slot >= windowSnapshot.containerSize {
			break
		}
		bedrockSlot, ok := windowSnapshot.javaSlotToBedrock(slot)
		if !ok {
			continue
		}
		if err := bedrock.WritePacket(&packet.InventorySlot{
			WindowID:  uint32(windowSnapshot.bedrockID),
			Slot:      bedrockSlot,
			Container: gtprotocol.Option(gtprotocol.FullContainerName{ContainerID: windowSnapshot.slotContainerID(slot)}),
			NewItem:   item,
		}); err != nil {
			return err
		}
	}
	return bedrock.WritePacket(&packet.InventorySlot{
		WindowID:  0,
		Slot:      0,
		Container: gtprotocol.Option(gtprotocol.FullContainerName{ContainerID: gtprotocol.ContainerCursor}),
		NewItem:   update.CarriedItem,
	})
}

func (b *Basic) updatePlayerInventoryFromWindowLocked(items []gtprotocol.ItemInstance, containerSize int) {
	for index, item := range items[minimumInt(containerSize, len(items)):] {
		var slot int
		switch {
		case index < 27:
			slot = 9 + index
		case index < 36:
			slot = 36 + index - 27
		default:
			continue
		}
		b.playerItems[slot] = item
	}
}

func minimumInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (b *Basic) translateWindowSlot(bedrock *minecraft.Conn, update JavaSetSlot) (bool, error) {
	if update.WindowID <= 0 {
		return false, nil
	}
	b.mu.Lock()
	window := b.windows[update.WindowID]
	if window == nil {
		b.mu.Unlock()
		b.logSemanticAnomaly("skipping slot update for unknown Java window", "window", update.WindowID)
		return true, nil
	}
	snapshot := *window
	if update.Slot >= 0 && int(update.Slot) < len(window.items) {
		window.items[update.Slot] = update.Item
	}
	window.stateID = update.StateID
	if playerSlot, ok := javaWindowPlayerSlot(update.Slot, snapshot.containerSize); ok && int(playerSlot) < len(b.playerItems) {
		b.playerItems[playerSlot] = update.Item
	}
	b.mu.Unlock()
	if update.Slot < 0 || int(update.Slot) >= snapshot.containerSize {
		if playerSlot, ok := javaWindowPlayerSlot(update.Slot, snapshot.containerSize); ok {
			return true, b.translateSetSlot(bedrock, JavaSetSlot{WindowID: 0, StateID: update.StateID, Slot: playerSlot, Item: update.Item, Known: update.Known})
		}
		return true, nil
	}
	bedrockSlot, ok := snapshot.javaSlotToBedrock(int(update.Slot))
	if !ok {
		return true, nil
	}
	return true, bedrock.WritePacket(&packet.InventorySlot{
		WindowID:  uint32(snapshot.bedrockID),
		Slot:      bedrockSlot,
		Container: gtprotocol.Option(gtprotocol.FullContainerName{ContainerID: snapshot.slotContainerID(int(update.Slot))}),
		NewItem:   update.Item,
	})
}

func javaWindowPlayerSlot(slot int16, containerSize int) (int16, bool) {
	index := int(slot) - containerSize
	if index < 0 || index >= 36 {
		return 0, false
	}
	if index < 27 {
		return int16(9 + index), true
	}
	return int16(36 + index - 27), true
}

func javaWindowMenuSlot(playerSlot, containerSize int) (int, bool) {
	switch {
	case playerSlot >= 9 && playerSlot <= 35:
		return containerSize + playerSlot - 9, true
	case playerSlot >= 36 && playerSlot <= 44:
		return containerSize + 27 + playerSlot - 36, true
	default:
		return 0, false
	}
}
