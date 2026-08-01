package data

import (
	"strings"
	"sync"
)

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

var (
	javaItemNamesOnce   sync.Once
	javaItemNames       map[string]int32
	javaEntityNamesOnce sync.Once
	javaEntityNames     map[string]string
)

var javaEntityIdentifierAliases = map[string]string{
	"minecraft:end_crystal":        "minecraft:ender_crystal",
	"minecraft:evoker_fangs":       "minecraft:evocation_fang",
	"minecraft:experience_bottle":  "minecraft:xp_bottle",
	"minecraft:experience_orb":     "minecraft:xp_orb",
	"minecraft:eye_of_ender":       "minecraft:eye_of_ender_signal",
	"minecraft:firework_rocket":    "minecraft:fireworks_rocket",
	"minecraft:fishing_bobber":     "minecraft:fishing_hook",
	"minecraft:tropical_fish":      "minecraft:tropicalfish",
	"minecraft:villager":           "minecraft:villager_v2",
	"minecraft:wind_charge":        "minecraft:wind_charge_projectile",
	"minecraft:breeze_wind_charge": "minecraft:breeze_wind_charge_projectile",
	"minecraft:zombie_villager":    "minecraft:zombie_villager_v2",
	"minecraft:zombified_piglin":   "minecraft:zombie_pigman",
}

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

// JavaItemID resolves a namespaced Java 1.21.4 item identifier to its
// negotiated registry ID. The generated name table is used by NBT-backed
// block entities whose item compounds carry names rather than registry IDs.
func JavaItemID(name string) (int32, bool) {
	javaItemNamesOnce.Do(func() {
		javaItemNames = make(map[string]int32, len(Java1214ItemNames))
		for itemID, itemName := range Java1214ItemNames {
			if itemName != "" {
				javaItemNames[itemName] = int32(itemID)
			}
		}
	})
	itemID, ok := javaItemNames[name]
	return itemID, ok
}

// BedrockEntityIdentifier resolves the Java entity identifier used by saved
// spawner data to the negotiated Bedrock actor identifier. The generated
// entity table already contains Geyser's canonical names; the alias table
// accepts the Java spellings that were normalized during generation.
func BedrockEntityIdentifier(name string) (string, bool) {
	if !strings.Contains(name, ":") {
		name = "minecraft:" + name
	}
	javaEntityNamesOnce.Do(func() {
		javaEntityNames = make(map[string]string, len(Java1214EntityTypeNames)+len(javaEntityIdentifierAliases))
		for _, identifier := range Java1214EntityTypeNames {
			if identifier != "" {
				javaEntityNames[identifier] = identifier
			}
		}
		for javaIdentifier, bedrockIdentifier := range javaEntityIdentifierAliases {
			javaEntityNames[javaIdentifier] = bedrockIdentifier
		}
	})
	identifier, ok := javaEntityNames[name]
	return identifier, ok
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
