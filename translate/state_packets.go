package translate

import (
	"fmt"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/go-gl/mathgl/mgl32"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
)

const javaEntityEquipmentContinuation = 0x80

type JavaHeldItemSlot struct {
	Slot int32
}

func DecodeHeldItemSlot(payload []byte) (JavaHeldItemSlot, error) {
	r := javaprotocol.NewReader(payload)
	slot, err := r.VarInt()
	if err != nil {
		return JavaHeldItemSlot{}, fmt.Errorf("translate: held item slot: %w", err)
	}
	if slot < 0 || slot > 8 {
		return JavaHeldItemSlot{}, fmt.Errorf("translate: held item slot %d is outside 0..8", slot)
	}
	if r.Remaining() != 0 {
		return JavaHeldItemSlot{}, fmt.Errorf("translate: held item slot has %d trailing bytes", r.Remaining())
	}
	return JavaHeldItemSlot{Slot: slot}, nil
}

type JavaEntityVelocity struct {
	EntityID int32
	Velocity mgl32.Vec3
}

func DecodeEntityVelocity(payload []byte) (JavaEntityVelocity, error) {
	r := javaprotocol.NewReader(payload)
	entityID, err := r.VarInt()
	if err != nil {
		return JavaEntityVelocity{}, fmt.Errorf("translate: entity velocity ID: %w", err)
	}
	x, err := r.Int16()
	if err != nil {
		return JavaEntityVelocity{}, fmt.Errorf("translate: entity velocity x: %w", err)
	}
	y, err := r.Int16()
	if err != nil {
		return JavaEntityVelocity{}, fmt.Errorf("translate: entity velocity y: %w", err)
	}
	z, err := r.Int16()
	if err != nil {
		return JavaEntityVelocity{}, fmt.Errorf("translate: entity velocity z: %w", err)
	}
	if r.Remaining() != 0 {
		return JavaEntityVelocity{}, fmt.Errorf("translate: entity velocity has %d trailing bytes", r.Remaining())
	}
	return JavaEntityVelocity{
		EntityID: entityID,
		Velocity: mgl32.Vec3{
			float32(float64(x) / javaEntityVelocityScale),
			float32(float64(y) / javaEntityVelocityScale),
			float32(float64(z) / javaEntityVelocityScale),
		},
	}, nil
}

// JavaEntityEquipment carries the terminated array used by Java's equipment
// packet. The low seven bits are the equipment slot and the high bit says that
// another pair follows.
type JavaEntityEquipment struct {
	EntityID int32
	Items    map[byte]gtprotocol.ItemInstance
}

func DecodeEntityEquipment(payload []byte, nextStackID func() int32) (JavaEntityEquipment, error) {
	r := javaprotocol.NewReader(payload)
	entityID, err := r.VarInt()
	if err != nil {
		return JavaEntityEquipment{}, fmt.Errorf("translate: entity equipment ID: %w", err)
	}
	items := make(map[byte]gtprotocol.ItemInstance, 6)
	for i := 0; i < 16; i++ {
		slot, err := r.Uint8()
		if err != nil {
			return JavaEntityEquipment{}, fmt.Errorf("translate: entity equipment slot %d: %w", i, err)
		}
		item, err := decodeJavaItemSlot(r, nextStackID)
		if err != nil {
			return JavaEntityEquipment{}, fmt.Errorf("translate: entity equipment slot %d: %w", i, err)
		}
		items[slot&^javaEntityEquipmentContinuation] = item.Item
		if slot&javaEntityEquipmentContinuation == 0 {
			if r.Remaining() != 0 {
				return JavaEntityEquipment{}, fmt.Errorf("translate: entity equipment has %d trailing bytes", r.Remaining())
			}
			return JavaEntityEquipment{EntityID: entityID, Items: items}, nil
		}
	}
	return JavaEntityEquipment{}, fmt.Errorf("translate: entity equipment has more than 16 entries")
}

type JavaSetPlayerInventory struct {
	Slot int32
	Item JavaItemSlot
}

func DecodeSetPlayerInventory(payload []byte, nextStackID func() int32) (JavaSetPlayerInventory, error) {
	r := javaprotocol.NewReader(payload)
	slot, err := r.VarInt()
	if err != nil {
		return JavaSetPlayerInventory{}, fmt.Errorf("translate: player inventory slot: %w", err)
	}
	if slot < 0 || slot > 45 {
		return JavaSetPlayerInventory{}, fmt.Errorf("translate: player inventory slot %d is outside 0..45", slot)
	}
	item, err := decodeJavaItemSlot(r, nextStackID)
	if err != nil {
		return JavaSetPlayerInventory{}, fmt.Errorf("translate: player inventory item: %w", err)
	}
	if r.Remaining() != 0 {
		return JavaSetPlayerInventory{}, fmt.Errorf("translate: player inventory has %d trailing bytes", r.Remaining())
	}
	return JavaSetPlayerInventory{Slot: slot, Item: item}, nil
}
