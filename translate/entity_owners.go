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

// snapshotJavaVaultActorRuntimeIDs returns the UUID-to-Bedrock-runtime mapping
// needed by the session-aware vault translator. The copy is made while the
// session mutex is held so packet encoding can run without holding b.mu.
func (b *Basic) snapshotJavaVaultActorRuntimeIDs() map[[16]byte]int64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.snapshotJavaVaultActorRuntimeIDsLocked()
}

func (b *Basic) snapshotJavaVaultActorRuntimeIDsLocked() map[[16]byte]int64 {
	actorRuntimeIDs := make(map[[16]byte]int64, len(b.entities)+1)
	add := func(playerUUID [16]byte, runtimeID uint64) {
		if playerUUID == [16]byte{} {
			return
		}
		actorRuntimeIDs[playerUUID] = int64(runtimeID)
	}
	for _, entity := range b.entities {
		if entity == nil {
			continue
		}
		add(entity.entityUUID, entity.runtimeID)
		if entity.player {
			add(entity.playerUUID, entity.runtimeID)
		}
	}
	// The local player is not necessarily present in b.entities because Java
	// does not send a normal spawn for the logged-in player.
	add(b.playerUUID, b.gameData.EntityRuntimeID)
	return actorRuntimeIDs
}
