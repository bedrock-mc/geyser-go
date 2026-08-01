package translate

import (
	"fmt"
	"math"
	"sort"

	"github.com/bedrock-mc/geyser-go/data"
	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/sandertv/gophertunnel/minecraft"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

const (
	javaContainerActionClickItem    = 0
	javaContainerActionMoveToHotbar = 2
	javaContainerActionDropItem     = 4
	javaClickLeft                   = 0
	javaClickRight                  = 1
	javaClickOutside                = -999
	javaMaxStackSize                = uint16(64)
)

type javaInventoryClick struct {
	slot       int
	actionType byte
	param      byte
}

type inventorySimulation struct {
	items  [46]gtprotocol.ItemInstance
	cursor gtprotocol.ItemInstance
}

// translateItemStackRequests handles the player inventory subset of
// Bedrock's server-authoritative stack protocol. Requests are translated into
// the Java 1.21.4 hashed-stack container-click packet and receive an explicit
// Bedrock response; unsupported windows/actions are rejected without tearing
// down a well-formed session.
func (b *Basic) translateItemStackRequests(bedrock *minecraft.Conn, java *javaprotocol.Client, requests []gtprotocol.ItemStackRequest) error {
	responses := make([]gtprotocol.ItemStackResponse, 0, len(requests))
	for _, request := range requests {
		response, err := b.translateItemStackRequest(java, request)
		if err != nil {
			return err
		}
		responses = append(responses, response)
	}
	if len(responses) == 0 {
		return nil
	}
	return bedrock.WritePacket(&packet.ItemStackResponse{Responses: responses})
}

func (b *Basic) translateItemStackRequest(java *javaprotocol.Client, request gtprotocol.ItemStackRequest) (gtprotocol.ItemStackResponse, error) {
	if len(request.Actions) == 0 {
		b.logSemanticAnomaly("rejecting empty Bedrock item stack request", "request_id", request.RequestID)
		return rejectedItemStackRequest(request), nil
	}
	affected := make(map[int]struct{})
	for _, action := range request.Actions {
		var err error
		switch action := action.(type) {
		case *gtprotocol.TakeStackRequestAction:
			err = b.translateStackTransfer(java, action.Source, action.Destination, action.Count, affected)
		case *gtprotocol.PlaceStackRequestAction:
			err = b.translateStackTransfer(java, action.Source, action.Destination, action.Count, affected)
		case *gtprotocol.PlaceInContainerStackRequestAction:
			err = b.translateStackTransfer(java, action.Source, action.Destination, action.Count, affected)
		case *gtprotocol.TakeOutContainerStackRequestAction:
			err = b.translateStackTransfer(java, action.Source, action.Destination, action.Count, affected)
		case *gtprotocol.SwapStackRequestAction:
			err = b.translateStackSwap(java, action.Source, action.Destination, affected)
		case *gtprotocol.DropStackRequestAction:
			err = b.translateStackDrop(java, action.Source, action.Count, affected)
		case *gtprotocol.MineBlockStackRequestAction:
			// Block destruction itself is represented by PlayerBlockAction or
			// InventoryTransaction. This action only confirms the client's
			// predicted held-stack identity.
			if action.HotbarSlot < 0 || action.HotbarSlot > 8 {
				err = fmt.Errorf("hotbar slot %d is outside 0..8", action.HotbarSlot)
				break
			}
			err = b.validateStackReferenceForSlot(gtprotocol.StackRequestSlotInfo{
				Container:      gtprotocol.FullContainerName{ContainerID: gtprotocol.ContainerHotBar},
				Slot:           byte(action.HotbarSlot),
				StackNetworkID: action.StackNetworkID,
			}, 36+int(action.HotbarSlot), false)
		default:
			err = fmt.Errorf("unsupported Bedrock item stack action %T", action)
		}
		if err != nil {
			b.logSemanticAnomaly("rejecting unsupported or invalid Bedrock item stack request", "request_id", request.RequestID, "error", err)
			return rejectedItemStackRequest(request), nil
		}
	}
	return b.acceptedItemStackRequest(request, affected), nil
}

func rejectedItemStackRequest(request gtprotocol.ItemStackRequest) gtprotocol.ItemStackResponse {
	return gtprotocol.ItemStackResponse{
		Status:    gtprotocol.ItemStackResponseStatusError,
		RequestID: request.RequestID,
	}
}

func (b *Basic) acceptedItemStackRequest(request gtprotocol.ItemStackRequest, affected map[int]struct{}) gtprotocol.ItemStackResponse {
	b.mu.Lock()
	items := b.playerItems
	cursor := b.cursorItem
	b.mu.Unlock()

	byContainer := make(map[byte][]gtprotocol.StackResponseSlotInfo)
	for javaSlot := range affected {
		containerID, bedrockSlot, ok := javaPlayerSlot(int16(javaSlot))
		if !ok {
			continue
		}
		byContainer[containerID] = append(byContainer[containerID], makeStackResponseSlot(bedrockSlot, items[javaSlot]))
	}
	for containerID := range byContainer {
		sort.Slice(byContainer[containerID], func(i, j int) bool {
			return byContainer[containerID][i].Slot < byContainer[containerID][j].Slot
		})
	}
	containerIDs := make([]int, 0, len(byContainer))
	for containerID := range byContainer {
		containerIDs = append(containerIDs, int(containerID))
	}
	sort.Ints(containerIDs)
	containers := make([]gtprotocol.StackResponseContainerInfo, 0, len(containerIDs)+1)
	for _, containerID := range containerIDs {
		id := byte(containerID)
		containers = append(containers, gtprotocol.StackResponseContainerInfo{
			Container: gtprotocol.FullContainerName{ContainerID: id},
			SlotInfo:  byContainer[id],
		})
	}
	containers = append(containers, gtprotocol.StackResponseContainerInfo{
		Container: gtprotocol.FullContainerName{ContainerID: gtprotocol.ContainerCursor},
		SlotInfo:  []gtprotocol.StackResponseSlotInfo{makeStackResponseSlot(0, cursor)},
	})
	return gtprotocol.ItemStackResponse{
		Status:        gtprotocol.ItemStackResponseStatusOK,
		RequestID:     request.RequestID,
		ContainerInfo: containers,
	}
}

func makeStackResponseSlot(slot uint32, item gtprotocol.ItemInstance) gtprotocol.StackResponseSlotInfo {
	if itemEmpty(item) {
		return gtprotocol.StackResponseSlotInfo{Slot: uint8(slot), HotbarSlot: uint8(slot)}
	}
	count := item.Stack.Count
	if count > math.MaxUint8 {
		count = math.MaxUint8
	}
	return gtprotocol.StackResponseSlotInfo{
		Slot:           uint8(slot),
		HotbarSlot:     uint8(slot),
		Count:          uint8(count),
		StackNetworkID: item.StackNetworkID,
	}
}

func (b *Basic) translateStackTransfer(java *javaprotocol.Client, source, destination gtprotocol.StackRequestSlotInfo, count byte, affected map[int]struct{}) error {
	if count == 0 {
		return fmt.Errorf("transfer count is zero")
	}
	sourceSlot, sourceCursor, sourceOK := bedrockStackSlot(source)
	destinationSlot, destinationCursor, destinationOK := bedrockStackSlot(destination)
	if !sourceOK || !destinationOK || (sourceCursor && destinationCursor) {
		return fmt.Errorf("transfer uses an unsupported container")
	}
	if err := b.validateStackReferenceForSlot(source, sourceSlot, sourceCursor); err != nil {
		return err
	}
	if err := b.validateStackReferenceForSlot(destination, destinationSlot, destinationCursor); err != nil {
		return err
	}
	b.mu.Lock()
	sim := inventorySimulation{items: b.playerItems, cursor: b.cursorItem}
	sourceItem := sim.cursor
	if !sourceCursor {
		sourceItem = sim.items[sourceSlot]
	}
	if itemEmpty(sourceItem) || int(sourceItem.Stack.Count) < int(count) {
		b.mu.Unlock()
		return fmt.Errorf("transfer exceeds source stack")
	}
	cursor := sim.cursor
	destinationItem := sim.cursor
	if !destinationCursor {
		destinationItem = sim.items[destinationSlot]
	}
	if !itemEmpty(cursor) && !sourceCursor && !destinationCursor {
		b.mu.Unlock()
		return fmt.Errorf("non-empty cursor cannot be preserved across a slot transfer")
	}
	b.mu.Unlock()

	if destinationCursor {
		if itemEmpty(cursor) {
			sourceCount := sourceItem.Stack.Count
			if int(count) == int(sourceCount) {
				return b.sendInventoryClicks(java, []javaInventoryClick{{slot: sourceSlot, actionType: javaContainerActionClickItem, param: javaClickLeft}}, affected)
			}
			if int(count) == int(sourceCount)-int(sourceCount)/2 {
				return b.sendInventoryClicks(java, []javaInventoryClick{{slot: sourceSlot, actionType: javaContainerActionClickItem, param: javaClickRight}}, affected)
			}
			clicks := []javaInventoryClick{{slot: sourceSlot, actionType: javaContainerActionClickItem, param: javaClickLeft}}
			for i := 0; i < int(sourceCount)-int(count); i++ {
				clicks = append(clicks, javaInventoryClick{slot: sourceSlot, actionType: javaContainerActionClickItem, param: javaClickRight})
			}
			return b.sendInventoryClicks(java, clicks, affected)
		}
		if !sameItem(cursor, sourceItem) || int(count) != int(sourceItem.Stack.Count) {
			return fmt.Errorf("partial transfer into a non-empty cursor is unsupported")
		}
		return b.sendInventoryClicks(java, []javaInventoryClick{{slot: sourceSlot, actionType: javaContainerActionClickItem, param: javaClickLeft}}, affected)
	}

	if sourceCursor {
		if int(count) == int(cursor.Stack.Count) {
			return b.sendInventoryClicks(java, []javaInventoryClick{{slot: destinationSlot, actionType: javaContainerActionClickItem, param: javaClickLeft}}, affected)
		}
		clicks := make([]javaInventoryClick, 0, int(count))
		for i := 0; i < int(count); i++ {
			clicks = append(clicks, javaInventoryClick{slot: destinationSlot, actionType: javaContainerActionClickItem, param: javaClickRight})
		}
		return b.sendInventoryClicks(java, clicks, affected)
	}

	if sourceSlot == destinationSlot || (!itemEmpty(destinationItem) && !sameItem(sourceItem, destinationItem)) {
		return fmt.Errorf("slot transfer has incompatible destination")
	}
	if int(count) == int(sourceItem.Stack.Count) {
		return b.sendInventoryClicks(java, []javaInventoryClick{
			{slot: sourceSlot, actionType: javaContainerActionClickItem, param: javaClickLeft},
			{slot: destinationSlot, actionType: javaContainerActionClickItem, param: javaClickLeft},
		}, affected)
	}
	sourceCount := sourceItem.Stack.Count
	if int(count) == int(sourceCount)-int(sourceCount)/2 {
		return b.sendInventoryClicks(java, []javaInventoryClick{
			{slot: sourceSlot, actionType: javaContainerActionClickItem, param: javaClickRight},
			{slot: destinationSlot, actionType: javaContainerActionClickItem, param: javaClickLeft},
		}, affected)
	}
	clicks := []javaInventoryClick{{slot: sourceSlot, actionType: javaContainerActionClickItem, param: javaClickLeft}}
	for i := 0; i < int(count); i++ {
		clicks = append(clicks, javaInventoryClick{slot: destinationSlot, actionType: javaContainerActionClickItem, param: javaClickRight})
	}
	clicks = append(clicks, javaInventoryClick{slot: sourceSlot, actionType: javaContainerActionClickItem, param: javaClickLeft})
	return b.sendInventoryClicks(java, clicks, affected)
}

func (b *Basic) translateStackSwap(java *javaprotocol.Client, source, destination gtprotocol.StackRequestSlotInfo, affected map[int]struct{}) error {
	sourceSlot, sourceCursor, sourceOK := bedrockStackSlot(source)
	destinationSlot, destinationCursor, destinationOK := bedrockStackSlot(destination)
	if !sourceOK || !destinationOK || (sourceCursor && destinationCursor) {
		return fmt.Errorf("swap uses an unsupported container")
	}
	if err := b.validateStackReferenceForSlot(source, sourceSlot, sourceCursor); err != nil {
		return err
	}
	if err := b.validateStackReferenceForSlot(destination, destinationSlot, destinationCursor); err != nil {
		return err
	}
	if !sourceCursor && !destinationCursor && destination.Container.ContainerID == gtprotocol.ContainerHotBar && destination.Slot <= 8 {
		if sourceSlot == destinationSlot {
			return fmt.Errorf("swap source and destination are equal")
		}
		return b.sendInventoryClicks(java, []javaInventoryClick{{
			slot:       sourceSlot,
			actionType: javaContainerActionMoveToHotbar,
			param:      destination.Slot,
		}}, affected)
	}
	if sourceCursor {
		return b.sendInventoryClicks(java, []javaInventoryClick{{slot: destinationSlot, actionType: javaContainerActionClickItem, param: javaClickLeft}}, affected)
	}
	if destinationCursor {
		return b.sendInventoryClicks(java, []javaInventoryClick{{slot: sourceSlot, actionType: javaContainerActionClickItem, param: javaClickLeft}}, affected)
	}
	if sourceSlot == destinationSlot {
		return fmt.Errorf("swap source and destination are equal")
	}
	return b.sendInventoryClicks(java, []javaInventoryClick{
		{slot: sourceSlot, actionType: javaContainerActionClickItem, param: javaClickLeft},
		{slot: destinationSlot, actionType: javaContainerActionClickItem, param: javaClickLeft},
		{slot: sourceSlot, actionType: javaContainerActionClickItem, param: javaClickLeft},
	}, affected)
}

func (b *Basic) translateStackDrop(java *javaprotocol.Client, source gtprotocol.StackRequestSlotInfo, count byte, affected map[int]struct{}) error {
	if count == 0 {
		return fmt.Errorf("drop count is zero")
	}
	slot, cursor, ok := bedrockStackSlot(source)
	if !ok {
		return fmt.Errorf("drop uses an unsupported container")
	}
	if err := b.validateStackReferenceForSlot(source, slot, cursor); err != nil {
		return err
	}
	b.mu.Lock()
	item := b.cursorItem
	if !cursor {
		item = b.playerItems[slot]
	}
	b.mu.Unlock()
	if itemEmpty(item) || int(item.Stack.Count) < int(count) {
		return fmt.Errorf("drop exceeds source stack")
	}
	clicks := make([]javaInventoryClick, 0, int(count))
	if cursor {
		if int(count) == int(item.Stack.Count) {
			clicks = append(clicks, javaInventoryClick{slot: javaClickOutside, actionType: javaContainerActionDropItem, param: javaClickLeft})
		} else {
			for i := 0; i < int(count); i++ {
				clicks = append(clicks, javaInventoryClick{slot: javaClickOutside, actionType: javaContainerActionDropItem, param: javaClickRight})
			}
		}
	} else if int(count) == int(item.Stack.Count) && item.Stack.Count > 1 {
		clicks = append(clicks, javaInventoryClick{slot: slot, actionType: javaContainerActionDropItem, param: javaClickRight})
	} else {
		for i := 0; i < int(count); i++ {
			clicks = append(clicks, javaInventoryClick{slot: slot, actionType: javaContainerActionDropItem, param: javaClickLeft})
		}
	}
	return b.sendInventoryClicks(java, clicks, affected)
}

func (b *Basic) sendInventoryClicks(java *javaprotocol.Client, clicks []javaInventoryClick, affected map[int]struct{}) error {
	for _, click := range clicks {
		changed, err := b.sendInventoryClick(java, click)
		if err != nil {
			return err
		}
		for slot := range changed {
			affected[slot] = struct{}{}
		}
	}
	return nil
}

func (b *Basic) sendInventoryClick(java *javaprotocol.Client, click javaInventoryClick) (map[int]gtprotocol.ItemInstance, error) {
	b.mu.Lock()
	sim := inventorySimulation{items: b.playerItems, cursor: b.cursorItem}
	changed, err := b.applyInventoryClickLocked(&sim, click)
	if err != nil {
		b.mu.Unlock()
		return nil, err
	}
	payload, err := encodeJavaContainerClick(b.inventoryStateID, click, changed, sim.cursor)
	if err != nil {
		b.mu.Unlock()
		return nil, err
	}
	b.playerItems = sim.items
	b.cursorItem = sim.cursor
	if b.inventoryStateID < math.MaxInt32 {
		b.inventoryStateID++
	}
	b.mu.Unlock()
	if err := java.Conn.WritePacket(b.Profile.PlayServerboundContainerClickID, payload); err != nil {
		return nil, err
	}
	return changed, nil
}

func (b *Basic) applyInventoryClickLocked(sim *inventorySimulation, click javaInventoryClick) (map[int]gtprotocol.ItemInstance, error) {
	changed := make(map[int]gtprotocol.ItemInstance, 1)
	if click.slot == javaClickOutside {
		if click.actionType != javaContainerActionClickItem && click.actionType != javaContainerActionDropItem {
			return nil, fmt.Errorf("outside click has unsupported action type %d", click.actionType)
		}
		if itemEmpty(sim.cursor) {
			return changed, nil
		}
		if click.actionType == javaContainerActionDropItem || click.param == javaClickRight {
			sim.cursor = subtractItem(sim.cursor, 1)
		} else {
			sim.cursor = gtprotocol.ItemInstance{}
		}
		return changed, nil
	}
	if click.slot < 0 || int(click.slot) >= len(sim.items) {
		return nil, fmt.Errorf("slot %d is outside player inventory", click.slot)
	}
	slot := int(click.slot)
	before := sim.items[slot]
	switch click.actionType {
	case javaContainerActionClickItem:
		if click.param == javaClickRight {
			b.applyRightClickLocked(sim, slot)
		} else {
			b.applyLeftClickLocked(sim, slot)
		}
	case javaContainerActionMoveToHotbar:
		if click.param > 8 {
			return nil, fmt.Errorf("hotbar target %d is outside 0..8", click.param)
		}
		destination := 36 + int(click.param)
		sim.items[slot], sim.items[destination] = sim.items[destination], sim.items[slot]
		changed[destination] = sim.items[destination]
	case javaContainerActionDropItem:
		if click.param == javaClickRight {
			sim.items[slot] = gtprotocol.ItemInstance{}
		} else {
			sim.items[slot] = subtractItem(sim.items[slot], 1)
		}
	default:
		return nil, fmt.Errorf("Java inventory action type %d is unsupported", click.actionType)
	}
	if !itemSame(before, sim.items[slot]) {
		changed[slot] = sim.items[slot]
	}
	return changed, nil
}

func (b *Basic) applyLeftClickLocked(sim *inventorySimulation, slot int) {
	target := sim.items[slot]
	if itemEmpty(sim.cursor) {
		sim.cursor = target
		sim.items[slot] = gtprotocol.ItemInstance{}
		return
	}
	if itemEmpty(target) {
		sim.items[slot] = sim.cursor
		sim.cursor = gtprotocol.ItemInstance{}
		return
	}
	if !sameItem(sim.cursor, target) {
		sim.items[slot], sim.cursor = sim.cursor, target
		return
	}
	space := int(javaMaxStackSize - minUint16(target.Stack.Count, javaMaxStackSize))
	move := minInt(space, int(sim.cursor.Stack.Count))
	if move > 0 {
		target.Stack.Count += uint16(move)
		sim.cursor = subtractItem(sim.cursor, move)
		sim.items[slot] = target
	}
}

func (b *Basic) applyRightClickLocked(sim *inventorySimulation, slot int) {
	target := sim.items[slot]
	if itemEmpty(sim.cursor) {
		if itemEmpty(target) {
			return
		}
		take := int(target.Stack.Count) - int(target.Stack.Count)/2
		sim.cursor = target
		sim.cursor.Stack.Count = uint16(take)
		sim.cursor.StackNetworkID = b.nextStackNetworkIDLocked()
		sim.items[slot] = subtractItem(target, take)
		return
	}
	if itemEmpty(target) {
		taken := cloneItem(sim.cursor)
		taken.Stack.Count = 1
		taken.StackNetworkID = b.nextStackNetworkIDLocked()
		sim.items[slot] = taken
		sim.cursor = subtractItem(sim.cursor, 1)
		return
	}
	if !sameItem(sim.cursor, target) {
		sim.items[slot], sim.cursor = sim.cursor, target
		return
	}
	if target.Stack.Count < javaMaxStackSize {
		target.Stack.Count++
		sim.items[slot] = target
		sim.cursor = subtractItem(sim.cursor, 1)
	}
}

func (b *Basic) nextStackNetworkIDLocked() int32 {
	id := b.nextStackID
	if id <= 1 {
		id = 2
	}
	b.nextStackID = id + 1
	if b.nextStackID <= 0 {
		b.nextStackID = 2
	}
	return id
}

func bedrockStackSlot(slot gtprotocol.StackRequestSlotInfo) (int, bool, bool) {
	if _, dynamic := slot.Container.DynamicContainerID.Value(); dynamic {
		return 0, false, false
	}
	switch slot.Container.ContainerID {
	case gtprotocol.ContainerCursor:
		return -1, true, slot.Slot == 0
	case gtprotocol.ContainerHotBar:
		if slot.Slot <= 8 {
			return 36 + int(slot.Slot), false, true
		}
	case gtprotocol.ContainerInventory:
		if slot.Slot >= 9 && slot.Slot <= 35 {
			return int(slot.Slot), false, true
		}
	case gtprotocol.ContainerCombinedHotBarAndInventory:
		if slot.Slot <= 8 {
			return 36 + int(slot.Slot), false, true
		}
		if slot.Slot <= 35 {
			return int(slot.Slot), false, true
		}
	case gtprotocol.ContainerArmor:
		if slot.Slot <= 3 {
			return 5 + int(slot.Slot), false, true
		}
	case gtprotocol.ContainerOffhand:
		if slot.Slot <= 1 {
			return 45, false, true
		}
	case gtprotocol.ContainerCraftingInput:
		if slot.Slot >= 28 && slot.Slot <= 31 {
			return int(slot.Slot) - 27, false, true
		}
	case gtprotocol.ContainerCreatedOutput:
		if slot.Slot == 0 || slot.Slot == 50 {
			return 0, false, true
		}
	}
	return 0, false, false
}

func (b *Basic) validateStackReferenceForSlot(reference gtprotocol.StackRequestSlotInfo, javaSlot int, cursor bool) error {
	current := gtprotocol.ItemInstance{}
	b.mu.Lock()
	if cursor {
		current = b.cursorItem
	} else {
		current = b.playerItems[javaSlot]
	}
	b.mu.Unlock()
	requested := reference.StackNetworkID
	if requested < 0 || requested == 1 {
		return nil
	}
	if requested != current.StackNetworkID {
		return fmt.Errorf("stack network ID %d does not match current ID %d", requested, current.StackNetworkID)
	}
	return nil
}

func itemEmpty(item gtprotocol.ItemInstance) bool {
	return item.Stack.Count == 0 || item.Stack.NetworkID == 0
}

func sameItem(a, b gtprotocol.ItemInstance) bool {
	return !itemEmpty(a) && !itemEmpty(b) && a.Stack.NetworkID == b.Stack.NetworkID && a.Stack.MetadataValue == b.Stack.MetadataValue
}

func itemSame(a, b gtprotocol.ItemInstance) bool {
	if a.StackNetworkID != b.StackNetworkID || a.Stack.NetworkID != b.Stack.NetworkID || a.Stack.MetadataValue != b.Stack.MetadataValue || a.Stack.Count != b.Stack.Count {
		return false
	}
	return true
}

func cloneItem(item gtprotocol.ItemInstance) gtprotocol.ItemInstance {
	return item
}

func subtractItem(item gtprotocol.ItemInstance, count int) gtprotocol.ItemInstance {
	if itemEmpty(item) || count <= 0 {
		return item
	}
	if count >= int(item.Stack.Count) {
		return gtprotocol.ItemInstance{}
	}
	item.Stack.Count -= uint16(count)
	return item
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func minUint16(a, b uint16) uint16 {
	if a < b {
		return a
	}
	return b
}

func encodeJavaContainerClick(stateID int32, click javaInventoryClick, changed map[int]gtprotocol.ItemInstance, cursor gtprotocol.ItemInstance) ([]byte, error) {
	w := javaprotocol.NewWriter()
	if err := w.VarInt(0); err != nil {
		return nil, err
	}
	if err := w.VarInt(stateID); err != nil {
		return nil, err
	}
	if err := w.Int16(int16(click.slot)); err != nil {
		return nil, err
	}
	param := click.param
	if click.actionType == javaContainerActionDropItem {
		param %= 2
	}
	if err := w.Byte(param); err != nil {
		return nil, err
	}
	if err := w.Byte(click.actionType); err != nil {
		return nil, err
	}
	keys := make([]int, 0, len(changed))
	for slot := range changed {
		keys = append(keys, slot)
	}
	sort.Ints(keys)
	if err := w.VarInt(int32(len(keys))); err != nil {
		return nil, err
	}
	for _, slot := range keys {
		if slot < math.MinInt16 || slot > math.MaxInt16 {
			return nil, fmt.Errorf("changed slot %d is outside Java short range", slot)
		}
		if err := w.Int16(int16(slot)); err != nil {
			return nil, err
		}
		if err := writeJavaHashedStack(w, changed[slot]); err != nil {
			return nil, err
		}
	}
	if err := writeJavaHashedStack(w, cursor); err != nil {
		return nil, err
	}
	return append([]byte(nil), w.Bytes()...), nil
}

func encodeJavaContainerClose(windowID byte) ([]byte, error) {
	w := javaprotocol.NewWriter()
	if err := w.VarInt(int32(windowID)); err != nil {
		return nil, err
	}
	return append([]byte(nil), w.Bytes()...), nil
}

func writeJavaHashedStack(w *javaprotocol.Writer, item gtprotocol.ItemInstance) error {
	if itemEmpty(item) {
		return w.Bool(false)
	}
	itemID, ok := data.BedrockItemRuntimeID(item.Stack.NetworkID)
	if !ok {
		return fmt.Errorf("Bedrock runtime item %d has no Java registry ID", item.Stack.NetworkID)
	}
	if err := w.Bool(true); err != nil {
		return err
	}
	if err := w.VarInt(itemID); err != nil {
		return err
	}
	if err := w.VarInt(int32(item.Stack.Count)); err != nil {
		return err
	}
	if err := w.VarInt(0); err != nil { // added component hashes
		return err
	}
	return w.VarInt(0) // removed component IDs
}
