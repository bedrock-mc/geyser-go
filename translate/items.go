package translate

import (
	"errors"
	"fmt"

	"github.com/bedrock-mc/geyser-go/data"
	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
)

// ErrUnsupportedJavaItemComponent identifies a well-formed structured item
// whose component payload is not translated by this tranche. The packet is
// safe to skip as a unit because its complete payload is already bounded; the
// session must remain usable while the component translator grows.
var ErrUnsupportedJavaItemComponent = errors.New("translate: unsupported Java item component")

type JavaItemSlot struct {
	Item    gtprotocol.ItemInstance
	ItemID  int32
	Known   bool
	Present bool
}

type JavaWindowItems struct {
	WindowID     int32
	StateID      int32
	Items        []gtprotocol.ItemInstance
	CarriedItem  gtprotocol.ItemInstance
	UnknownItems int
}

type JavaSetSlot struct {
	WindowID int32
	StateID  int32
	Slot     int16
	Item     gtprotocol.ItemInstance
	Known    bool
}

// DecodeJavaWindowItems reads ClientboundContainerSetContentPacket for the
// Java 1.21.4 structured Slot format. Stack IDs are allocated by the caller
// so all translated Bedrock stacks remain unique within a session.
func DecodeJavaWindowItems(payload []byte, nextStackID func() int32) (JavaWindowItems, error) {
	r := javaprotocol.NewReader(payload)
	windowID, err := r.VarInt()
	if err != nil {
		return JavaWindowItems{}, fmt.Errorf("translate: window items window ID: %w", err)
	}
	stateID, err := r.VarInt()
	if err != nil {
		return JavaWindowItems{}, fmt.Errorf("translate: window items state ID: %w", err)
	}
	count, err := boundedJavaCount(r, "window items count")
	if err != nil {
		return JavaWindowItems{}, err
	}
	items := make([]gtprotocol.ItemInstance, count)
	unknownItems := 0
	for i := range items {
		decoded, decodeErr := decodeJavaItemSlot(r, nextStackID)
		if decodeErr != nil {
			return JavaWindowItems{}, fmt.Errorf("translate: window items slot %d: %w", i, decodeErr)
		}
		items[i] = decoded.Item
		if decoded.Present && !decoded.Known {
			unknownItems++
		}
	}
	carried, err := decodeJavaItemSlot(r, nextStackID)
	if err != nil {
		return JavaWindowItems{}, fmt.Errorf("translate: window items carried item: %w", err)
	}
	if r.Remaining() != 0 {
		return JavaWindowItems{}, fmt.Errorf("translate: window items has %d trailing bytes", r.Remaining())
	}
	if carried.Present && !carried.Known {
		unknownItems++
	}
	return JavaWindowItems{
		WindowID:     windowID,
		StateID:      stateID,
		Items:        items,
		CarriedItem:  carried.Item,
		UnknownItems: unknownItems,
	}, nil
}

func DecodeJavaSetSlot(payload []byte, nextStackID func() int32) (JavaSetSlot, error) {
	r := javaprotocol.NewReader(payload)
	windowID, err := r.VarInt()
	if err != nil {
		return JavaSetSlot{}, fmt.Errorf("translate: set slot window ID: %w", err)
	}
	stateID, err := r.VarInt()
	if err != nil {
		return JavaSetSlot{}, fmt.Errorf("translate: set slot state ID: %w", err)
	}
	slot, err := r.Int16()
	if err != nil {
		return JavaSetSlot{}, fmt.Errorf("translate: set slot index: %w", err)
	}
	item, err := decodeJavaItemSlot(r, nextStackID)
	if err != nil {
		return JavaSetSlot{}, fmt.Errorf("translate: set slot item: %w", err)
	}
	if r.Remaining() != 0 {
		return JavaSetSlot{}, fmt.Errorf("translate: set slot has %d trailing bytes", r.Remaining())
	}
	return JavaSetSlot{WindowID: windowID, StateID: stateID, Slot: slot, Item: item.Item, Known: !item.Present || item.Known}, nil
}

// DecodeJavaCursorItem reads ClientboundSetCursorItemPacket. Unlike a player
// slot update, this packet has no window or state ID; the cursor is its whole
// payload and is therefore decoded through the same bounded Slot path.
func DecodeJavaCursorItem(payload []byte, nextStackID func() int32) (JavaItemSlot, error) {
	r := javaprotocol.NewReader(payload)
	item, err := decodeJavaItemSlot(r, nextStackID)
	if err != nil {
		return JavaItemSlot{}, fmt.Errorf("translate: cursor item: %w", err)
	}
	if r.Remaining() != 0 {
		return JavaItemSlot{}, fmt.Errorf("translate: cursor item has %d trailing bytes", r.Remaining())
	}
	return item, nil
}

func decodeJavaItemSlot(r *javaprotocol.Reader, nextStackID func() int32) (JavaItemSlot, error) {
	count, err := r.VarInt()
	if err != nil {
		return JavaItemSlot{}, fmt.Errorf("item count: %w", err)
	}
	if count == 0 {
		return JavaItemSlot{}, nil
	}
	if count < 0 || count > 65535 {
		return JavaItemSlot{}, fmt.Errorf("invalid item count %d", count)
	}
	itemID, err := r.VarInt()
	if err != nil {
		return JavaItemSlot{}, fmt.Errorf("item ID: %w", err)
	}
	added, err := boundedJavaCount(r, "added component count")
	if err != nil {
		return JavaItemSlot{}, err
	}
	removed, err := boundedJavaCount(r, "removed component count")
	if err != nil {
		return JavaItemSlot{}, err
	}
	components, err := decodeJavaItemComponents(r, added)
	if err != nil {
		return JavaItemSlot{}, err
	}
	if len(components.unsupported) != 0 {
		return JavaItemSlot{}, fmt.Errorf("%w: id=%d components=%v", ErrUnsupportedJavaItemComponent, itemID, components.unsupported)
	}
	for i := 0; i < removed; i++ {
		if _, err := r.VarInt(); err != nil {
			return JavaItemSlot{}, fmt.Errorf("removed component %d: %w", i, err)
		}
	}
	runtimeID, known := data.JavaItemRuntimeID(itemID)
	if itemID == 0 {
		known = false
	}
	if !known {
		return JavaItemSlot{ItemID: itemID, Known: false, Present: true}, nil
	}
	item := gtprotocol.ItemInstance{
		Stack: gtprotocol.ItemStack{
			ItemType: gtprotocol.ItemType{NetworkID: runtimeID},
			Count:    uint16(count),
		},
	}
	if components.hasMetadata {
		item.Stack.MetadataValue = components.metadata
	}
	item.Stack.NBTData = components.nbt
	if nextStackID != nil {
		item.StackNetworkID = nextStackID()
	}
	return JavaItemSlot{Item: item, ItemID: itemID, Known: true, Present: true}, nil
}

func boundedJavaCount(r *javaprotocol.Reader, field string) (int, error) {
	count, err := r.VarInt()
	if err != nil {
		return 0, fmt.Errorf("translate: %s: %w", field, err)
	}
	if count < 0 || count > maxJavaCollectionSize {
		return 0, fmt.Errorf("translate: %s %d exceeds limit %d", field, count, maxJavaCollectionSize)
	}
	return int(count), nil
}

// decodeJavaNBT consumes the root compound used by Java anonymousNbt.
func decodeJavaNBT(r *javaprotocol.Reader) (map[string]any, error) {
	value, err := decodeJavaNBTValue(r)
	if err != nil {
		return nil, err
	}
	compound, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("translate: expected NBT compound, got %T", value)
	}
	return compound, nil
}

func decodeJavaNBTValue(r *javaprotocol.Reader) (any, error) {
	return javaprotocol.DecodeAnonymousNBT(r)
}
