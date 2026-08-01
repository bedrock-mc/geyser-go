package translate

import (
	"encoding/binary"
	"math"
)

const javaVaultDefaultParticleRange float32 = 4.5

// projectJavaVault converts the observable shared vault state. Java stores the
// state below shared_data, while Bedrock reads these fields directly from the
// block-entity compound.
func projectJavaVault(tag map[string]any) {
	projectJavaVaultWithResolver(tag, nil)
}

func projectJavaVaultWithResolver(tag map[string]any, actorRuntimeIDs map[[16]byte]int64) {
	sharedData, _ := javaNBTCompound(tag["shared_data"])

	displayItem := javaVaultEmptyItem()
	if javaItem, ok := javaNBTCompound(sharedData["display_item"]); ok {
		if projected, ok := projectJavaBlockEntityItem(javaItem); ok {
			displayItem = projected
		}
	}
	tag["display_item"] = displayItem

	// Geyser resolves Java UUIDs against the session's entity cache. The pure
	// helper keeps its existing direct-ID behavior, while the session-aware path
	// resolves only the exact UUID int-array representation emitted by Java NBT.
	if actorRuntimeIDs == nil {
		tag["connected_players"] = projectJavaVaultConnectedPlayers(sharedData["connected_players"])
	} else {
		tag["connected_players"] = projectJavaVaultConnectedPlayersWithResolver(sharedData["connected_players"], actorRuntimeIDs)
	}
	tag["connected_particle_range"] = projectJavaVaultParticleRange(sharedData["connected_particles_range"])
}

func javaVaultEmptyItem() map[string]any {
	return map[string]any{
		"Count":       byte(0),
		"Damage":      int16(0),
		"Name":        "",
		"WasPickedUp": byte(0),
	}
}

func projectJavaVaultConnectedPlayers(value any) []int64 {
	players := make([]int64, 0)
	appendPlayer := func(value any) {
		if player, ok := javaNBTInt64Value(value); ok {
			players = append(players, player)
		}
	}

	switch value := value.(type) {
	case []int64:
		players = append(players, value...)
	case []any:
		for _, entry := range value {
			appendPlayer(entry)
		}
	}
	if len(players) > maxJavaCollectionSize {
		players = players[:maxJavaCollectionSize]
	}
	return players
}

func projectJavaVaultConnectedPlayersWithResolver(value any, actorRuntimeIDs map[[16]byte]int64) []int64 {
	players := make([]int64, 0)
	appendPlayer := func(value any) {
		playerUUID, ok := javaVaultUUID(value)
		if !ok {
			return
		}
		if playerUUID == [16]byte{} {
			return
		}
		if runtimeID, ok := actorRuntimeIDs[playerUUID]; ok {
			players = append(players, runtimeID)
		}
	}

	switch value := value.(type) {
	case []any:
		for _, entry := range value {
			appendPlayer(entry)
		}
	case [][4]int32:
		for _, entry := range value {
			appendPlayer(entry)
		}
	}
	if len(players) > maxJavaCollectionSize {
		players = players[:maxJavaCollectionSize]
	}
	return players
}

// javaVaultUUID decodes the int-array UUID representation used by Java NBT.
// The pinned NBT decoder exposes TAG_IntArray as [4]int32 for a UUID; the
// words are the UUID's big-endian 32-bit words, not actor IDs.
func javaVaultUUID(value any) ([16]byte, bool) {
	words, ok := value.([4]int32)
	if !ok {
		return [16]byte{}, false
	}
	var result [16]byte
	for index, word := range words {
		binary.BigEndian.PutUint32(result[index*4:], uint32(word))
	}
	return result, true
}

func projectJavaVaultParticleRange(value any) float32 {
	particleRange, ok := javaNBTFloat32Value(value)
	if !ok || math.IsNaN(float64(particleRange)) || math.IsInf(float64(particleRange), 0) {
		return javaVaultDefaultParticleRange
	}
	if particleRange < 0 {
		return 0
	}
	return particleRange
}
