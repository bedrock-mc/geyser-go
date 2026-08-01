package translate

import javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"

// javaEntityVariantMappings converts Java's negotiated registry ordinals to
// the Bedrock variant IDs expected by the vanilla actor definitions. The two
// registries are not ordered the same way, so forwarding the Java ordinal is
// not a safe projection.
type javaEntityVariantMappings struct {
	cat  map[int32]int32
	wolf map[int32]int32
}

var javaCatVariantBedrockIDs = map[string]int32{
	"minecraft:white":             0,
	"minecraft:black":             1,
	"minecraft:red":               2,
	"minecraft:siamese":           3,
	"minecraft:british_shorthair": 4,
	"minecraft:calico":            5,
	"minecraft:persian":           6,
	"minecraft:ragdoll":           7,
	"minecraft:tabby":             8,
	"minecraft:all_black":         9,
	"minecraft:jellie":            10,
}

var javaWolfVariantBedrockIDs = map[string]int32{
	"minecraft:pale":     0,
	"minecraft:ashen":    1,
	"minecraft:black":    2,
	"minecraft:chestnut": 3,
	"minecraft:rusty":    4,
	"minecraft:snowy":    5,
	"minecraft:spotted":  6,
	"minecraft:striped":  7,
	"minecraft:woods":    8,
}

// The 1.21.4 Java cat registry is not sent by Paper's configuration stream,
// but its built-in order is stable and version-pinned here. Wolf uses the
// negotiated registry when present and this order as a bounded fallback.
var java1214CatVariantOrder = []string{
	"minecraft:tabby", "minecraft:black", "minecraft:red", "minecraft:siamese",
	"minecraft:british_shorthair", "minecraft:calico", "minecraft:persian",
	"minecraft:ragdoll", "minecraft:white", "minecraft:jellie", "minecraft:all_black",
}

var java1214WolfVariantOrder = []string{
	"minecraft:ashen", "minecraft:black", "minecraft:chestnut", "minecraft:pale",
	"minecraft:rusty", "minecraft:snowy", "minecraft:spotted", "minecraft:striped",
	"minecraft:woods",
}

func newJavaEntityVariantMappings(configuration javaprotocol.ConfigurationData) javaEntityVariantMappings {
	mappings := javaEntityVariantMappings{
		cat:  make(map[int32]int32),
		wolf: make(map[int32]int32),
	}
	for index, key := range java1214CatVariantOrder {
		mappings.cat[int32(index)] = javaCatVariantBedrockIDs[key]
	}
	for index, key := range java1214WolfVariantOrder {
		mappings.wolf[int32(index)] = javaWolfVariantBedrockIDs[key]
	}
	if registry, ok := configuration.Registry("minecraft:cat_variant"); ok {
		mappings.cat = make(map[int32]int32, len(registry.Entries))
		for index, entry := range registry.Entries {
			bedrockID, known := javaCatVariantBedrockIDs[entry.Key]
			if !known {
				bedrockID = 1 // minecraft:black, Geyser's cat fallback.
			}
			mappings.cat[int32(index)] = bedrockID
		}
	}
	if registry, ok := configuration.Registry("minecraft:wolf_variant"); ok {
		mappings.wolf = make(map[int32]int32, len(registry.Entries))
		for index, entry := range registry.Entries {
			bedrockID, known := javaWolfVariantBedrockIDs[entry.Key]
			if !known {
				bedrockID = 0 // minecraft:pale, Geyser's wolf fallback.
			}
			mappings.wolf[int32(index)] = bedrockID
		}
	}
	return mappings
}

func defaultJavaEntityVariantMappings() javaEntityVariantMappings {
	return newJavaEntityVariantMappings(javaprotocol.ConfigurationData{})
}

func javaEntityVariantID(entityType string, javaID int32, mappings javaEntityVariantMappings) (int32, bool) {
	if javaID < 0 {
		return 0, false
	}
	var variants map[int32]int32
	switch entityType {
	case "minecraft:cat":
		variants = mappings.cat
	case "minecraft:wolf":
		variants = mappings.wolf
	default:
		return 0, false
	}
	bedrockID, ok := variants[javaID]
	if !ok {
		// Unknown well-formed registry ordinals use the Geyser built-in
		// fallback rather than exposing a Java ordinal as a Bedrock ID.
		if entityType == "minecraft:cat" {
			return 1, true // minecraft:black
		}
		return 0, true // minecraft:pale
	}
	return bedrockID, true
}
