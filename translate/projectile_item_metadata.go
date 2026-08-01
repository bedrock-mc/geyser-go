package translate

import (
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// javaPotionBedrockIDs is the Java 1.21.4 Potion registry ordinal to Bedrock
// aux-value table used by Geyser's Potion enum. The Java registry ID is the
// enum ordinal; it is not the Bedrock damage value for several potions.
var javaPotionBedrockIDs = [...]int16{
	0, 1, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
	13, 14, 15, 16, 17, 18, 42, 37, 38, 39, 19, 20,
	21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32,
	33, 34, 35, 2, 40, 41, 43, 44, 45, 46,
}

func javaPotionBedrockID(javaID int32) (int16, bool) {
	if javaID < 0 || int64(javaID) >= int64(len(javaPotionBedrockIDs)) {
		return 0, false
	}
	return javaPotionBedrockIDs[javaID], true
}

func javaPotionIsEnchanted(javaID int32) bool {
	// Water, mundane, thick, and awkward are Geyser's four non-enchanted
	// potion variants. These values are Java registry ordinals.
	switch javaID {
	case 0, 1, 2, 3:
		return false
	default:
		return true
	}
}

func javaThrownPotionEntity(entityType string) bool {
	switch entityType {
	case "minecraft:potion", "minecraft:splash_potion", "minecraft:lingering_potion":
		return true
	default:
		return false
	}
}

func translateJavaPotionEntityMetadata(entityType string, item JavaEntityItemMetadata, hasItemUpdate bool, metadata gtprotocol.EntityMetadata) {
	if !javaThrownPotionEntity(entityType) || !hasItemUpdate {
		return
	}
	auxValue := int16(0)
	enchanted := false
	if item.HasPotionID {
		if bedrockID, ok := javaPotionBedrockID(item.PotionID); ok {
			auxValue = bedrockID
			enchanted = javaPotionIsEnchanted(item.PotionID)
		}
	}
	metadata[gtprotocol.EntityDataKeyAuxValueData] = auxValue
	setProjectedFlag(metadata, gtprotocol.EntityDataFlagEnchanted, enchanted)
	setProjectedFlag(metadata, gtprotocol.EntityDataFlagLingering, entityType == "minecraft:lingering_potion")
}

// updateJavaFireworkAttachmentLocked projects Java's optional gliding-owner
// metadata into Bedrock's movement prediction effect. The caller holds b.mu.
func (b *Basic) updateJavaFireworkAttachmentLocked(entityID int32, entity *javaEntityState, entries []JavaEntityMetadataEntry) (duration int32, send bool) {
	if entity == nil || entity.entityType != "minecraft:fireworks_rocket" {
		return 0, false
	}
	for _, entry := range entries {
		if entry.Index != 9 {
			continue
		}
		owner, hasOwner := javaIntegerValue(entry.Value)
		attachedToPlayer := hasOwner && owner == int64(int32(uint32(b.gameData.EntityRuntimeID)))
		if attachedToPlayer == entity.fireworkAttachedToPlayer {
			continue
		}
		entity.fireworkAttachedToPlayer = attachedToPlayer
		if b.fireworkAttachments == nil {
			b.fireworkAttachments = make(map[int32]struct{})
		}
		if attachedToPlayer {
			b.fireworkAttachments[entityID] = struct{}{}
			return 1000000, true
		}
		delete(b.fireworkAttachments, entityID)
		if len(b.fireworkAttachments) == 0 {
			return 0, true
		}
	}
	return 0, false
}

func (b *Basic) writeFireworkMovementEffect(bedrock interface {
	WritePacket(packet.Packet) error
}, duration int32) error {
	b.mu.Lock()
	runtimeID := b.gameData.EntityRuntimeID
	tick := b.clientTick
	b.mu.Unlock()
	return bedrock.WritePacket(&packet.MovementEffect{
		EntityRuntimeID: runtimeID,
		Type:            packet.MovementEffectTypeGlideBoost,
		Duration:        duration,
		Tick:            tick,
	})
}
