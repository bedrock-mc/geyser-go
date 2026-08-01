package translate

import (
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
)

// javaSpawnEntityProjection returns the Bedrock metadata that must accompany
// Java's special object entities. Java puts the payload in Spawn Entity's
// object-data field, while Bedrock expects most of these values in actor
// metadata.
//
// The bool reports whether the entity can be projected safely. A fishing hook
// without an owner has no lossless Bedrock representation: Geyser drops that
// spawn rather than emitting a hook with a broken fishing line.
func javaSpawnEntityProjection(entityType string, objectData int32) (gtprotocol.EntityMetadata, bool) {
	metadata := gtprotocol.NewEntityMetadataWithCapacity(4)
	switch entityType {
	case "minecraft:xp_orb":
		// Geyser uses this marker to select the Bedrock XP-orb texture. The
		// Java object-data value is the orb's XP amount, not this Bedrock
		// metadata field, and the vanilla projection keeps the marker at 1.
		metadata[gtprotocol.EntityDataKeyTradeExperience] = int32(1)
	case "minecraft:falling_block":
		runtimeID, known := JavaBlockRuntimeID(objectData)
		metadata[gtprotocol.EntityDataKeyDisplayTileRuntimeID] = int32(runtimeID)
		return metadata, known
	case "minecraft:fishing_hook":
		if objectData < 0 {
			return metadata, false
		}
		// This bridge deliberately uses the Java entity ID as the Bedrock
		// runtime ID, so the owner can be projected without a second lookup.
		metadata[gtprotocol.EntityDataKeyOwner] = int64(uint32(objectData))
	}
	return metadata, true
}
