package data

var bedrockToJavaItem = func() map[int32]int32 {
	lookup := make(map[int32]int32, len(Java1214ToBedrockItem))
	for itemID, runtimeID := range Java1214ToBedrockItem {
		if itemID == 0 || runtimeID == 0 {
			continue
		}
		if _, exists := lookup[runtimeID]; !exists {
			lookup[runtimeID] = int32(itemID)
		}
	}
	return lookup
}()

// JavaItemRuntimeID maps a Java item registry ID to the Bedrock item network
// ID used by Gophertunnel's ItemInstance format. Unknown IDs are mapped to
// air and reported as false so semantically odd server data does not tear down
// a valid session.
func JavaItemRuntimeID(itemID int32) (int32, bool) {
	// Bedrock's empty ItemInstance is encoded as NetworkID 0. The complete
	// Cloudburst item-state table retains the legacy registry value for the air
	// entry, but that value must never be emitted as a non-empty stack.
	if itemID == 0 {
		return 0, true
	}
	if itemID < 0 || int64(itemID) >= int64(len(Java1214ToBedrockItem)) {
		return 0, false
	}
	return Java1214ToBedrockItem[itemID], true
}

// BedrockItemRuntimeID returns the first Java registry ID represented by a
// Bedrock item network ID. The generated mapping can contain aliases, so the
// result is intentionally one valid Java spelling rather than a claim that
// the mapping is one-to-one.
func BedrockItemRuntimeID(runtimeID int32) (int32, bool) {
	if runtimeID == 0 {
		return 0, true
	}
	itemID, ok := bedrockToJavaItem[runtimeID]
	if !ok {
		return 0, false
	}
	return itemID, true
}

// JavaEntityTypeName returns the Bedrock identifier for a Java entity type
// registry ID in the active protocol profile.
func JavaEntityTypeName(entityTypeID int32) (string, bool) {
	if entityTypeID < 0 || int64(entityTypeID) >= int64(len(Java1214EntityTypeNames)) {
		return "", false
	}
	name := Java1214EntityTypeNames[entityTypeID]
	return name, name != ""
}
