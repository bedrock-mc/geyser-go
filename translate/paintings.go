package translate

import (
	"strings"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/go-gl/mathgl/mgl32"
)

type javaPaintingDefinition struct {
	Name   string
	Motive string
	Width  int32
	Height int32
}

// The motive names and fallback dimensions are the Bedrock names used by
// Geyser's painting projection. Negotiated Java registry values override this
// table, including custom variants and future registry ordering.
var javaPaintingDefaults = []javaPaintingDefinition{
	{Name: "minecraft:kebab", Motive: "Kebab", Width: 1, Height: 1},
	{Name: "minecraft:aztec", Motive: "Aztec", Width: 1, Height: 1},
	{Name: "minecraft:alban", Motive: "Alban", Width: 1, Height: 1},
	{Name: "minecraft:aztec2", Motive: "Aztec2", Width: 1, Height: 1},
	{Name: "minecraft:bomb", Motive: "Bomb", Width: 1, Height: 1},
	{Name: "minecraft:plant", Motive: "Plant", Width: 1, Height: 1},
	{Name: "minecraft:wasteland", Motive: "Wasteland", Width: 1, Height: 1},
	{Name: "minecraft:wanderer", Motive: "Wanderer", Width: 1, Height: 2},
	{Name: "minecraft:graham", Motive: "Graham", Width: 1, Height: 2},
	{Name: "minecraft:pool", Motive: "Pool", Width: 2, Height: 1},
	{Name: "minecraft:courbet", Motive: "Courbet", Width: 2, Height: 1},
	{Name: "minecraft:sunset", Motive: "Sunset", Width: 2, Height: 1},
	{Name: "minecraft:sea", Motive: "Sea", Width: 2, Height: 1},
	{Name: "minecraft:creebet", Motive: "Creebet", Width: 2, Height: 1},
	{Name: "minecraft:match", Motive: "Match", Width: 2, Height: 2},
	{Name: "minecraft:bust", Motive: "Bust", Width: 2, Height: 2},
	{Name: "minecraft:stage", Motive: "Stage", Width: 2, Height: 2},
	{Name: "minecraft:void", Motive: "Void", Width: 2, Height: 2},
	{Name: "minecraft:skull_and_roses", Motive: "SkullAndRoses", Width: 2, Height: 2},
	{Name: "minecraft:wither", Motive: "Wither", Width: 2, Height: 2},
	{Name: "minecraft:fighters", Motive: "Fighters", Width: 4, Height: 2},
	{Name: "minecraft:skeleton", Motive: "Skeleton", Width: 4, Height: 3},
	{Name: "minecraft:donkey_kong", Motive: "DonkeyKong", Width: 4, Height: 3},
	{Name: "minecraft:pointer", Motive: "Pointer", Width: 4, Height: 4},
	{Name: "minecraft:pigscene", Motive: "Pigscene", Width: 4, Height: 4},
	{Name: "minecraft:burning_skull", Motive: "BurningSkull", Width: 4, Height: 4},
	{Name: "minecraft:earth", Motive: "Earth", Width: 2, Height: 2},
	{Name: "minecraft:wind", Motive: "Wind", Width: 2, Height: 2},
	{Name: "minecraft:water", Motive: "Water", Width: 2, Height: 2},
	{Name: "minecraft:fire", Motive: "Fire", Width: 2, Height: 2},
	{Name: "minecraft:meditative", Motive: "meditative", Width: 1, Height: 1},
	{Name: "minecraft:prairie_ride", Motive: "prairie_ride", Width: 1, Height: 2},
	{Name: "minecraft:baroque", Motive: "baroque", Width: 2, Height: 2},
	{Name: "minecraft:humble", Motive: "humble", Width: 2, Height: 2},
	{Name: "minecraft:unpacked", Motive: "unpacked", Width: 4, Height: 4},
	{Name: "minecraft:backyard", Motive: "backyard", Width: 3, Height: 4},
	{Name: "minecraft:bouquet", Motive: "bouquet", Width: 3, Height: 3},
	{Name: "minecraft:cavebird", Motive: "cavebird", Width: 3, Height: 3},
	{Name: "minecraft:changing", Motive: "changing", Width: 4, Height: 2},
	{Name: "minecraft:cotan", Motive: "cotan", Width: 3, Height: 3},
	{Name: "minecraft:endboss", Motive: "endboss", Width: 3, Height: 3},
	{Name: "minecraft:fern", Motive: "fern", Width: 3, Height: 3},
	{Name: "minecraft:finding", Motive: "finding", Width: 4, Height: 2},
	{Name: "minecraft:lowmist", Motive: "lowmist", Width: 4, Height: 2},
	{Name: "minecraft:orb", Motive: "orb", Width: 4, Height: 4},
	{Name: "minecraft:owlemons", Motive: "owlemons", Width: 3, Height: 3},
	{Name: "minecraft:passage", Motive: "passage", Width: 4, Height: 2},
	{Name: "minecraft:pond", Motive: "pond", Width: 3, Height: 4},
	{Name: "minecraft:sunflowers", Motive: "sunflowers", Width: 3, Height: 3},
	{Name: "minecraft:tides", Motive: "tides", Width: 3, Height: 3},
	{Name: "minecraft:dennis", Motive: "dennis", Width: 3, Height: 3},
}

func javaPaintingCatalog(configuration javaprotocol.ConfigurationData) []javaPaintingDefinition {
	registry, ok := configuration.Registry("minecraft:painting_variant")
	if !ok || len(registry.Entries) == 0 {
		return append([]javaPaintingDefinition(nil), javaPaintingDefaults...)
	}
	definitions := make([]javaPaintingDefinition, len(registry.Entries))
	for index, entry := range registry.Entries {
		definition := javaPaintingDefinition{
			Name:   entry.Key,
			Motive: "Kebab",
			Width:  1,
			Height: 1,
		}
		if fallback, found := paintingDefault(entry.Key); found {
			definition = fallback
			definition.Name = entry.Key
		}
		if value, ok := entry.Value.(map[string]any); ok {
			if width, found := javaNBTInt32(value, "width"); found && width > 0 {
				definition.Width = width
			}
			if height, found := javaNBTInt32(value, "height"); found && height > 0 {
				definition.Height = height
			}
			if asset, found := javaNBTString(value, "asset_id"); found {
				if motive, known := paintingMotive(asset); known {
					definition.Motive = motive
				}
			}
		}
		definitions[index] = definition
	}
	return definitions
}

func (b *Basic) paintingDefinition(variant JavaPaintingVariant) (javaPaintingDefinition, bool) {
	if variant.Custom {
		motive, known := paintingMotive(variant.AssetID)
		if !known {
			motive = "Kebab"
		}
		if variant.Width <= 0 || variant.Height <= 0 || variant.AssetID == "" {
			return javaPaintingDefinition{}, false
		}
		return javaPaintingDefinition{Name: variant.AssetID, Motive: motive, Width: variant.Width, Height: variant.Height}, true
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if variant.RegistryID >= 0 && int64(variant.RegistryID) < int64(len(b.paintings)) {
		definition := b.paintings[variant.RegistryID]
		return definition, definition.Motive != ""
	}
	if variant.RegistryID >= 0 && int64(variant.RegistryID) < int64(len(javaPaintingDefaults)) {
		return javaPaintingDefaults[variant.RegistryID], true
	}
	return javaPaintingDefinition{}, false
}

func paintingDefault(name string) (javaPaintingDefinition, bool) {
	for _, definition := range javaPaintingDefaults {
		if definition.Name == name {
			return definition, true
		}
	}
	return javaPaintingDefinition{}, false
}

func paintingMotive(name string) (string, bool) {
	name = strings.TrimPrefix(name, "minecraft:")
	for _, definition := range javaPaintingDefaults {
		if strings.TrimPrefix(definition.Name, "minecraft:") == name {
			return definition.Motive, true
		}
	}
	return "", false
}

func javaNBTString(value map[string]any, key string) (string, bool) {
	raw, ok := value[key]
	if !ok {
		return "", false
	}
	result, ok := raw.(string)
	return result, ok && result != ""
}

func javaPaintingDirection(direction int32) int32 {
	switch direction {
	case 3: // south
		return 0
	case 4: // west
		return 1
	case 2: // north
		return 2
	case 5: // east
		return 3
	default:
		return 0
	}
}

func javaPaintingPosition(position mgl32.Vec3, direction, width, height int32) mgl32.Vec3 {
	const offset = float32(-0.46875)
	var widthOffset, heightOffset float32
	if width > 1 && width != 3 {
		widthOffset = 0.5
	}
	if height > 1 && height != 3 {
		heightOffset = 0.5
	}
	switch direction {
	case 3: // south
		return position.Add(mgl32.Vec3{widthOffset, heightOffset, offset})
	case 4: // west
		return position.Add(mgl32.Vec3{-offset, heightOffset, widthOffset})
	case 2: // north
		return position.Add(mgl32.Vec3{-widthOffset, heightOffset, -offset})
	case 5: // east
		return position.Add(mgl32.Vec3{offset, heightOffset, -widthOffset})
	default:
		return position
	}
}
