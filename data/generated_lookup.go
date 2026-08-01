package data

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

// JavaEntityTypeName returns the Bedrock identifier for a Java entity type
// registry ID in the active protocol profile.
func JavaEntityTypeName(entityTypeID int32) (string, bool) {
	if entityTypeID < 0 || int64(entityTypeID) >= int64(len(Java1214EntityTypeNames)) {
		return "", false
	}
	name := Java1214EntityTypeNames[entityTypeID]
	return name, name != ""
}
