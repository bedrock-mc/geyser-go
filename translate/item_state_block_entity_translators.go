package translate

// projectJavaShulkerBox projects the one item/state-bearing block-entity
// field directly covered by the pinned Geyser reference. Java 1.21.4 stores
// the shulker's direction in the block state; Bedrock expects the direction's
// ordinal as the block-entity "facing" byte.
//
// Keep this projection state-only. The pinned Geyser reference handles shulker
// contents through its inventory flow rather than establishing an item-list
// block-entity projection. It also provides no source-backed block-entity
// projection for jukebox, lectern, or chiseled_bookshelf in this Java version.
func projectJavaShulkerBox(tag map[string]any, stateName string) {
	if !javaShulkerBoxState(stateName) {
		return
	}
	facing, ok := javaBlockStateProperty(stateName, "facing")
	if !ok {
		return
	}
	ordinal, ok := javaShulkerFacingOrdinals[facing]
	if !ok {
		return
	}
	tag["facing"] = int8(ordinal)
}

func javaShulkerBoxState(stateName string) bool {
	switch javaBlockStateBaseName(stateName) {
	case "shulker_box", "white_shulker_box", "orange_shulker_box", "magenta_shulker_box",
		"light_blue_shulker_box", "yellow_shulker_box", "lime_shulker_box", "pink_shulker_box",
		"gray_shulker_box", "light_gray_shulker_box", "cyan_shulker_box", "purple_shulker_box",
		"blue_shulker_box", "brown_shulker_box", "green_shulker_box", "red_shulker_box",
		"black_shulker_box":
		return true
	default:
		return false
	}
}

// This ordering follows the Direction enum used by Geyser's
// blockState.getValue(Properties.FACING).ordinal() projection.
var javaShulkerFacingOrdinals = map[string]int8{
	"down":  0,
	"up":    1,
	"north": 2,
	"south": 3,
	"west":  4,
	"east":  5,
}
