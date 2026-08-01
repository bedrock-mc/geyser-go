package translate

const javaUnknownOwnerEntityID int64 = 1<<63 - 1

// javaOwnerEntityIDLocked resolves a Java owner UUID to the Bedrock actor ID
// used by Gophertunnel. The caller holds b.mu. Geyser uses a sentinel for a
// valid but not-yet-visible owner, while an absent optional UUID is projected
// as zero by the metadata translator.
func (b *Basic) javaOwnerEntityIDLocked(ownerUUID [16]byte) int64 {
	if ownerUUID == [16]byte{} {
		return 0
	}
	if ownerUUID == b.playerUUID && b.playerUUID != [16]byte{} {
		return int64(b.gameData.EntityRuntimeID)
	}
	for _, entity := range b.entities {
		if entity == nil {
			continue
		}
		if entity.entityUUID == ownerUUID || (entity.player && entity.playerUUID == ownerUUID) {
			return int64(entity.runtimeID)
		}
	}
	return javaUnknownOwnerEntityID
}
