package translate

import "math"

const javaVaultDefaultParticleRange float32 = 4.5

// projectJavaVault converts the observable shared vault state. Java stores the
// state below shared_data, while Bedrock reads these fields directly from the
// block-entity compound.
func projectJavaVault(tag map[string]any) {
	sharedData, _ := javaNBTCompound(tag["shared_data"])

	displayItem := javaVaultEmptyItem()
	if javaItem, ok := javaNBTCompound(sharedData["display_item"]); ok {
		if projected, ok := projectJavaBlockEntityItem(javaItem); ok {
			displayItem = projected
		}
	}
	tag["display_item"] = displayItem

	// Geyser resolves Java UUIDs against the session's entity cache. This
	// stateless shared helper cannot manufacture that mapping, so it only
	// forwards values that are already Bedrock entity IDs and emits a valid
	// empty list for the Java UUID-array form. The original Java payload remains
	// in shared_data for callers that have a resolver.
	tag["connected_players"] = projectJavaVaultConnectedPlayers(sharedData["connected_players"])
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
