package translate

import (
	"github.com/bedrock-mc/geyser-go/data"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

const (
	javaItemFrameDefaultDirection = 3 // Java Direction.SOUTH.
	itemFrameUpdateFlags          = packet.BlockUpdateNeighbours | packet.BlockUpdateNetwork | packet.BlockUpdatePriority
	maxItemFrameCount             = uint16(255)
)

func javaItemFramePosition(position [3]float32) gtprotocol.BlockPos {
	// Geyser's Vector3i.toInt() is a Java narrowing conversion, not a floor.
	// This matters for hanging entities on negative coordinates.
	return gtprotocol.BlockPos{int32(position[0]), int32(position[1]), int32(position[2])}
}

func javaItemFrameRuntimeID(entityType string, direction int32) (uint32, bool) {
	if direction < 0 || direction > 5 {
		direction = javaItemFrameDefaultDirection
	}
	name := "minecraft:frame"
	if entityType == "minecraft:glow_item_frame" {
		name = "minecraft:glow_frame"
	}
	return data.BedrockBlockRuntimeID(name, map[string]any{
		"facing_direction":     direction,
		"item_frame_map_bit":   uint8(0),
		"item_frame_photo_bit": uint8(0),
	})
}

func javaItemFrameBlockUpdate(position gtprotocol.BlockPos, runtimeID uint32) *packet.UpdateBlock {
	return &packet.UpdateBlock{
		Position:          position,
		NewBlockRuntimeID: runtimeID,
		Flags:             itemFrameUpdateFlags,
		Layer:             0,
	}
}

func javaItemFrameTag(position gtprotocol.BlockPos, entityType string, item gtprotocol.ItemInstance, rotation int32) map[string]any {
	tag := map[string]any{
		"x":              position[0],
		"y":              position[1],
		"z":              position[2],
		"isMovable":      byte(1),
		"id":             "ItemFrame",
		"ItemDropChance": float32(1),
		"ItemRotation":   float32(rotation * 45),
	}
	if entityType == "minecraft:glow_item_frame" {
		tag["id"] = "GlowItemFrame"
	}
	if itemEmpty(item) || item.Stack.Count == 0 {
		return tag
	}
	name, ok := data.BedrockItemName(item.Stack.ItemType.NetworkID)
	if !ok {
		return tag
	}
	count := item.Stack.Count
	if count > maxItemFrameCount {
		count = maxItemFrameCount
	}
	itemTag := map[string]any{
		"Count":  byte(count),
		"Damage": int16(bedrockItemDamage(item)),
		"Name":   name,
	}
	if customData := itemNBTWithoutDamage(item.Stack.NBTData); customData != nil {
		itemTag["tag"] = customData
	}
	tag["Item"] = itemTag
	return tag
}

func (b *Basic) writeJavaItemFrame(bedrock interface{ WritePacket(packet.Packet) error }, entityType string, position gtprotocol.BlockPos, direction int32, item gtprotocol.ItemInstance, rotation int32) error {
	runtimeID, ok := javaItemFrameRuntimeID(entityType, direction)
	if !ok {
		b.logSemanticAnomaly("skipping item frame with unknown Bedrock block state", "entity", entityType, "direction", direction)
		return nil
	}
	if err := bedrock.WritePacket(javaItemFrameBlockUpdate(position, runtimeID)); err != nil {
		return err
	}
	return bedrock.WritePacket(&packet.BlockActorData{
		Position: position,
		NBTData:  javaItemFrameTag(position, entityType, item, rotation),
	})
}

func (b *Basic) clearJavaItemFrame(bedrock interface{ WritePacket(packet.Packet) error }, position gtprotocol.BlockPos) error {
	airRuntimeID, found := data.BedrockBlockRuntimeID("minecraft:air", nil)
	if !found {
		b.logSemanticAnomaly("could not resolve Bedrock air while clearing item frame", "position", position)
		return nil
	}
	return bedrock.WritePacket(javaItemFrameBlockUpdate(position, airRuntimeID))
}
