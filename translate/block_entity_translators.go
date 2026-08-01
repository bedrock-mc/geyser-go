package translate

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/bedrock-mc/geyser-go/data"
)

// projectJavaBlockEntityPayload applies the type-specific NBT projections
// needed by Bedrock's tile-entity readers. The generic identity, coordinates,
// and unknown fields are assembled by BedrockBlockEntityTag first so this
// helper can be shared by chunk payloads and standalone updates.
func projectJavaBlockEntityPayload(javaName string, tag map[string]any, stateName string) {
	switch javaName {
	case "sign", "hanging_sign":
		projectJavaSign(tag)
	case "banner":
		projectJavaBanner(tag, stateName)
	case "skull":
		projectJavaSkull(tag, stateName)
	case "jigsaw":
		projectJavaJigsaw(tag, stateName)
	case "command_block":
		projectJavaCommandBlock(tag, stateName)
	case "structure_block":
		projectJavaStructureBlock(tag)
	case "brushable_block":
		projectJavaBrushableBlock(tag, stateName)
	case "campfire":
		projectJavaCampfire(tag)
	case "beacon":
		projectJavaBeacon(tag)
	case "end_gateway":
		projectJavaEndGateway(tag)
	case "decorated_pot":
		projectJavaDecoratedPot(tag)
	case "mob_spawner":
		projectJavaMobSpawner(tag)
	case "trial_spawner":
		projectJavaTrialSpawner(tag)
	}
}

func projectJavaBanner(tag map[string]any, stateName string) {
	if color, ok := javaBannerColor(javaBlockStateBaseName(stateName)); ok {
		// Java's DyeColor enum and Bedrock's banner color IDs are reversed.
		tag["Base"] = int32(15 - color)
	}

	patterns, ok := javaNBTCompoundList(tag["patterns"])
	if !ok {
		return
	}
	converted := make([]map[string]any, 0, len(patterns))
	ominous := javaBannerIsOminous(patterns)
	for _, pattern := range patterns {
		name, found := javaNBTStringValue(pattern["pattern"])
		if !found {
			name, found = javaNBTStringValue(pattern["Pattern"])
		}
		if !found {
			continue
		}
		name = strings.TrimPrefix(name, "minecraft:")
		bedrockName, found := javaBannerPatternNames[name]
		if !found {
			continue
		}
		color, found := javaBannerDyeColor(pattern["color"])
		if !found {
			color, found = javaBannerDyeColor(pattern["Color"])
		}
		if !found || color < 0 || color > 15 {
			continue
		}
		converted = append(converted, map[string]any{
			"Pattern": bedrockName,
			"Color":   int32(15 - color),
		})
	}
	if ominous {
		// Bedrock has a dedicated ominous-banner type and does not render the
		// raw Java pattern stack correctly.
		tag["Type"] = int32(1)
	} else if len(converted) != 0 {
		tag["Patterns"] = converted
	}
	delete(tag, "patterns")
}

func javaBannerDyeColor(value any) (int64, bool) {
	if name, ok := javaNBTStringValue(value); ok {
		name = strings.TrimPrefix(name, "minecraft:")
		color, found := javaBannerDyeColors[name]
		return color, found
	}
	color, ok := javaNBTInt64Value(value)
	return color, ok && color >= 0 && color <= 15
}

func javaBannerIsOminous(patterns []map[string]any) bool {
	want := []struct {
		pattern string
		color   string
	}{
		{"mr", "cyan"}, {"bs", "light_gray"}, {"cs", "gray"}, {"bo", "light_gray"},
		{"ms", "black"}, {"hh", "light_gray"}, {"mc", "light_gray"}, {"bo", "black"},
	}
	if len(patterns) != len(want) {
		return false
	}
	for index, pattern := range patterns {
		name, ok := javaNBTStringValue(pattern["pattern"])
		if !ok {
			name, ok = javaNBTStringValue(pattern["Pattern"])
		}
		if !ok {
			return false
		}
		name = strings.TrimPrefix(name, "minecraft:")
		if javaBannerPatternNames[name] != want[index].pattern {
			return false
		}
		color, ok := javaBannerDyeColor(pattern["color"])
		if !ok {
			color, ok = javaBannerDyeColor(pattern["Color"])
		}
		wantColor, wantOK := javaBannerDyeColors[want[index].color]
		if !ok || !wantOK || color != wantColor {
			return false
		}
	}
	return true
}

func projectJavaSkull(tag map[string]any, stateName string) {
	if rotation, ok := javaBlockStatePropertyInt(stateName, "rotation"); ok && rotation >= 0 && rotation < 16 {
		tag["Rotation"] = float32(rotation) * 22.5
	}
	if powered, ok := javaBlockStatePropertyBool(stateName, "powered"); ok && powered {
		tag["MouthMoving"] = true
	}
}

func projectJavaJigsaw(tag map[string]any, stateName string) {
	if joint, ok := javaNBTStringValue(tag["joint"]); ok {
		tag["joint"] = joint
	} else if orientation, ok := javaBlockStateProperty(stateName, "orientation"); ok {
		if javaJigsawOrientationAligned(orientation) {
			tag["joint"] = "aligned"
		} else {
			tag["joint"] = "rollable"
		}
	}
	for _, key := range []string{"name", "target_pool", "final_state", "target"} {
		if value, ok := javaNBTStringValue(tag[key]); ok {
			tag[key] = value
		}
	}
}

func projectJavaCommandBlock(tag map[string]any, stateName string) {
	if conditional, ok := javaBlockStatePropertyBool(stateName, "conditional"); ok {
		tag["conditionalMode"] = conditional
	}
}

func projectJavaStructureBlock(tag map[string]any) {
	if name, ok := javaNBTStringValue(tag["name"]); ok {
		tag["structureName"] = name
	}
	mode := int32(1)
	if value, ok := javaNBTStringValue(tag["mode"]); ok {
		switch value {
		case "LOAD":
			mode = 2
		case "CORNER":
			mode = 3
		case "DATA":
			mode = 4
		}
	}
	tag["data"] = mode
	tag["dataField"] = ""

	if mirror, ok := javaNBTStringValue(tag["mirror"]); ok {
		var value int8
		switch mirror {
		case "FRONT_BACK":
			value = 1
		case "LEFT_RIGHT":
			value = 2
		}
		tag["mirror"] = value
	}
	if rotation, ok := javaNBTStringValue(tag["rotation"]); ok {
		var value int8
		switch rotation {
		case "CLOCKWISE_90":
			value = 1
		case "CLOCKWISE_180":
			value = 2
		case "COUNTERCLOCKWISE_90":
			value = 3
		}
		tag["rotation"] = value
	}
	if value, ok := javaNBTBoolValue(tag["ignoreEntities"]); ok {
		tag["ignoreEntities"] = value
	}
	if value, ok := javaNBTBoolValue(tag["powered"]); ok {
		tag["isPowered"] = value
	}
	if value, ok := javaNBTBoolValue(tag["showboundingbox"]); ok {
		tag["showBoundingBox"] = value
	}
	if seed, ok := javaNBTInt64Value(tag["seed"]); ok {
		tag["seed"] = seed
	}
	for javaKey, bedrockKey := range map[string]string{
		"posX": "xStructureOffset", "posY": "yStructureOffset", "posZ": "zStructureOffset",
		"sizeX": "xStructureSize", "sizeY": "yStructureSize", "sizeZ": "zStructureSize",
	} {
		if value, ok := javaNBTInt64Value(tag[javaKey]); ok {
			tag[bedrockKey] = clampJavaNBTInt32(value)
		}
	}
	if integrity, ok := javaNBTFloat32Value(tag["integrity"]); ok {
		if integrity < 0 {
			integrity = 0
		}
		if integrity > 1 {
			integrity = 1
		}
		tag["integrity"] = integrity
	}
}

// projectJavaBrushableBlock converts Java's transient archaeology payload into
// the Bedrock fields consumed by the suspicious-sand/gravel renderer. The
// block state carries both the dusting progress and the block identifier; the
// item and hit direction remain in the block-entity NBT.
func projectJavaBrushableBlock(tag map[string]any, stateName string) {
	javaItem, hasItem := javaNBTCompound(tag["item"])
	delete(tag, "item")

	if hasItem {
		if name, ok := javaNBTStringValue(javaItem["id"]); ok {
			name = strings.TrimPrefix(name, "minecraft:")
			if name != "air" {
				if item, ok := projectJavaBlockEntityItem(javaItem); ok {
					tag["item"] = item
				}
			}
		}
	}

	if direction, ok := javaNBTInt64Value(tag["hit_direction"]); ok {
		delete(tag, "hit_direction")
		// Java uses -1 while the brushable item is fully retracted. Bedrock
		// treats that sentinel as an absent update rather than a direction.
		if direction >= 0 && direction <= 127 {
			tag["brush_direction"] = int8(direction)
		}
	}

	if dusted, ok := javaBlockStatePropertyInt(stateName, "dusted"); ok {
		if dusted < 0 {
			dusted = 0
		}
		if dusted > 3 {
			dusted = 3
		}
		tag["brush_count"] = int32(dusted)
	}
	if blockType, ok := javaBrushableBlockType(stateName); ok {
		tag["type"] = blockType
	}
}

func javaBrushableBlockType(stateName string) (string, bool) {
	switch javaBlockStateBaseName(stateName) {
	case "suspicious_sand", "suspicious_gravel":
		return "minecraft:" + javaBlockStateBaseName(stateName), true
	default:
		return "", false
	}
}

func javaBlockStateBaseName(stateName string) string {
	name := strings.TrimPrefix(stateName, "minecraft:")
	if bracket := strings.IndexByte(name, '['); bracket >= 0 {
		name = name[:bracket]
	}
	return name
}

func javaBlockStateProperty(stateName, property string) (string, bool) {
	start := strings.IndexByte(stateName, '[')
	end := strings.LastIndexByte(stateName, ']')
	if start < 0 || end <= start {
		return "", false
	}
	for _, field := range strings.Split(stateName[start+1:end], ",") {
		parts := strings.SplitN(field, "=", 2)
		if len(parts) == 2 && parts[0] == property {
			return parts[1], true
		}
	}
	return "", false
}

func javaBlockStatePropertyInt(stateName, property string) (int, bool) {
	value, ok := javaBlockStateProperty(stateName, property)
	if !ok {
		return 0, false
	}
	parsed, err := strconv.Atoi(value)
	return parsed, err == nil
}

func javaBlockStatePropertyBool(stateName, property string) (bool, bool) {
	value, ok := javaBlockStateProperty(stateName, property)
	if !ok {
		return false, false
	}
	parsed, err := strconv.ParseBool(value)
	return parsed, err == nil
}

func javaJigsawOrientationAligned(orientation string) bool {
	orientation = strings.ToLower(orientation)
	front, _, _ := strings.Cut(orientation, "_")
	return front != "up" && front != "down"
}

func javaBannerColor(blockName string) (int64, bool) {
	blockName = strings.TrimSuffix(blockName, "_wall")
	if !strings.HasSuffix(blockName, "_banner") {
		return 0, false
	}
	color := strings.TrimSuffix(blockName, "_banner")
	value, ok := javaBannerDyeColors[color]
	return value, ok
}

var javaBannerDyeColors = map[string]int64{
	"white": 0, "orange": 1, "magenta": 2, "light_blue": 3,
	"yellow": 4, "lime": 5, "pink": 6, "gray": 7,
	"light_gray": 8, "cyan": 9, "purple": 10, "blue": 11,
	"brown": 12, "green": 13, "red": 14, "black": 15,
}

// Java and Bedrock use the same short banner-pattern identifiers for the
// vanilla pattern set. Keeping the explicit allow-list makes malformed or
// future server-provided identifiers harmless while retaining all 1.21.4
// vanilla patterns.
var javaBannerPatternNames = map[string]string{
	"base": "b", "b": "b", "square_bottom_left": "bl", "bl": "bl",
	"square_bottom_right": "br", "br": "br", "square_top_left": "tl", "tl": "tl",
	"square_top_right": "tr", "tr": "tr", "stripe_bottom": "bs", "bs": "bs",
	"stripe_top": "ts", "ts": "ts", "stripe_left": "ls", "ls": "ls",
	"stripe_right": "rs", "rs": "rs", "stripe_center": "cs", "cs": "cs",
	"stripe_middle": "ms", "ms": "ms", "stripe_downright": "drs", "drs": "drs",
	"stripe_downleft": "dls", "dls": "dls", "small_stripes": "ss", "ss": "ss",
	"cross": "cr", "cr": "cr", "straight_cross": "sc", "sc": "sc",
	"triangle_bottom": "bt", "bt": "bt", "triangle_top": "tt", "tt": "tt",
	"triangles_bottom": "bts", "bts": "bts", "triangles_top": "tts", "tts": "tts",
	"diagonal_left": "ld", "ld": "ld", "diagonal_up_right": "rd", "rd": "rd",
	"diagonal_up_left": "lud", "lud": "lud", "diagonal_right": "rud", "rud": "rud",
	"circle": "mc", "mc": "mc", "rhombus": "mr", "mr": "mr",
	"half_vertical": "vh", "vh": "vh", "half_horizontal": "hh", "hh": "hh",
	"half_vertical_right": "vhr", "vhr": "vhr", "half_horizontal_bottom": "hhb", "hhb": "hhb",
	"border": "bo", "bo": "bo", "curly_border": "cbo", "cbo": "cbo",
	"gradient": "gra", "gra": "gra", "gradient_up": "gru", "gru": "gru",
	"bricks": "bri", "bri": "bri", "globe": "glb", "glb": "glb",
	"creeper": "cre", "cre": "cre", "skull": "sku", "sku": "sku",
	"flower": "flo", "flo": "flo", "mojang": "moj", "moj": "moj",
	"piglin": "pig", "pig": "pig", "flow": "flw", "flw": "flw",
	"guster": "gus", "gus": "gus",
}

func projectJavaMobSpawner(tag map[string]any) {
	spawnData, ok := javaNBTCompound(tag["SpawnData"])
	if !ok {
		spawnData, ok = javaNBTCompound(tag["spawn_data"])
	}
	if !ok {
		return
	}
	entity, ok := javaNBTCompound(spawnData["entity"])
	if !ok {
		return
	}
	javaIdentifier, ok := javaNBTStringValue(entity["id"])
	if !ok {
		return
	}
	bedrockIdentifier, ok := data.BedrockEntityIdentifier(javaIdentifier)
	if !ok {
		return
	}
	tag["EntityIdentifier"] = bedrockIdentifier
	delete(tag, "SpawnData")
	delete(tag, "spawn_data")
}

// projectJavaTrialSpawner keeps the trial spawner's compact Bedrock spawn
// selector. Java stores the entity identifier and weight beneath
// spawn_data.entity; Bedrock expects TypeId and Weight directly beneath its
// own spawn_data compound.
func projectJavaTrialSpawner(tag map[string]any) {
	spawnData, ok := javaNBTCompound(tag["spawn_data"])
	if !ok {
		spawnData, ok = javaNBTCompound(tag["SpawnData"])
	}
	if !ok {
		return
	}
	delete(tag, "spawn_data")
	delete(tag, "SpawnData")

	entity, _ := javaNBTCompound(spawnData["entity"])
	projected := make(map[string]any, 2)
	if javaIdentifier, ok := javaNBTStringValue(entity["id"]); ok {
		if bedrockIdentifier, ok := data.BedrockEntityIdentifier(javaIdentifier); ok {
			projected["TypeId"] = bedrockIdentifier
		}
	}
	weight := int64(1)
	if value, ok := javaNBTInt64Value(entity["Size"]); ok {
		weight = value
	}
	if weight < 0 {
		weight = 0
	}
	if weight > 1<<31-1 {
		weight = 1<<31 - 1
	}
	projected["Weight"] = int32(weight)
	tag["spawn_data"] = projected
}

func projectJavaBeacon(tag map[string]any) {
	for _, key := range []string{"primary", "secondary"} {
		value, ok := javaNBTInt64Value(tag[key])
		if !ok {
			continue
		}
		if value < 0 {
			value = 0
		}
		tag[key] = int32(clampJavaNBTInt32(value))
	}
	for javaKey, bedrockKey := range map[string]string{
		"primary_effect":   "primary",
		"secondary_effect": "secondary",
	} {
		value, exists := tag[javaKey]
		if !exists {
			continue
		}
		effectID, ok := javaBeaconEffectID(value)
		if !ok {
			if _, hasNumericValue := tag[bedrockKey]; !hasNumericValue {
				tag[bedrockKey] = int32(0)
			}
		} else {
			tag[bedrockKey] = effectID
		}
		delete(tag, javaKey)
	}
}

func projectJavaEndGateway(tag map[string]any) {
	if value, ok := javaNBTInt64Value(tag["Age"]); ok {
		tag["Age"] = clampJavaNBTInt32(value)
	}

	exitPortal := []int32{0, 0, 0}
	rawExitPortal, hasExitPortal := tag["ExitPortal"]
	if !hasExitPortal {
		rawExitPortal, hasExitPortal = tag["exit_portal"]
	}
	switch value := rawExitPortal.(type) {
	case [3]int32:
		exitPortal = value[:]
	case [3]int64:
		for index, coordinate := range value {
			exitPortal[index] = clampJavaNBTInt32(coordinate)
		}
	case []int32:
		copy(exitPortal, value)
	case []int64:
		for index := 0; index < len(value) && index < len(exitPortal); index++ {
			exitPortal[index] = clampJavaNBTInt32(value[index])
		}
	case map[string]any:
		projectJavaEndGatewayExitCompound(exitPortal, value)
	}
	if hasExitPortal {
		delete(tag, "exit_portal")
	}
	if !hasExitPortal {
		if raw, ok := javaNBTCompound(tag["ExitPortal"]); ok {
			projectJavaEndGatewayExitCompound(exitPortal, raw)
		}
	}
	// Bedrock expects a three-element INT list even when Java omits the
	// optional exit portal compound.
	tag["ExitPortal"] = exitPortal
}

func projectJavaEndGatewayExitCompound(exitPortal []int32, raw map[string]any) {
	for index, key := range []string{"X", "Y", "Z"} {
		if value, found := javaNBTInt64Value(raw[key]); found {
			exitPortal[index] = clampJavaNBTInt32(value)
		}
	}
}

func projectJavaDecoratedPot(tag map[string]any) {
	values, ok := javaNBTAnyList(tag["sherds"])
	if !ok {
		return
	}
	sherds := make([]string, 0, len(values))
	for _, value := range values {
		if sherd, ok := value.(string); ok {
			sherds = append(sherds, sherd)
		}
	}
	tag["sherds"] = sherds
}

func clampJavaNBTInt32(value int64) int32 {
	if value < -1<<31 {
		return -1 << 31
	}
	if value > 1<<31-1 {
		return 1<<31 - 1
	}
	return int32(value)
}

func javaBeaconEffectID(value any) (int32, bool) {
	name, ok := javaNBTStringValue(value)
	if !ok {
		return 0, false
	}
	name = strings.TrimPrefix(name, "minecraft:")
	id, ok := javaBeaconEffectIDs[name]
	return id, ok
}

// Java's modern beacon NBT stores a namespaced MobEffect holder while
// Bedrock's tile entity expects the legacy one-based effect ID.
var javaBeaconEffectIDs = map[string]int32{
	"speed":               1,
	"slowness":            2,
	"haste":               3,
	"mining_fatigue":      4,
	"strength":            5,
	"instant_health":      6,
	"instant_damage":      7,
	"jump_boost":          8,
	"nausea":              9,
	"regeneration":        10,
	"resistance":          11,
	"fire_resistance":     12,
	"water_breathing":     13,
	"invisibility":        14,
	"blindness":           15,
	"night_vision":        16,
	"hunger":              17,
	"weakness":            18,
	"poison":              19,
	"wither":              20,
	"health_boost":        21,
	"absorption":          22,
	"saturation":          23,
	"glowing":             24,
	"levitation":          25,
	"luck":                26,
	"unluck":              27,
	"slow_falling":        28,
	"conduit_power":       29,
	"dolphins_grace":      30,
	"bad_omen":            31,
	"hero_of_the_village": 32,
	"darkness":            33,
	"trial_omen":          34,
	"raid_omen":           35,
	"wind_charged":        36,
	"weaving":             37,
	"oozing":              38,
	"infested":            39,
}

const javaCampfireSlotCount = 4

// projectJavaCampfire converts Java's sparse item list into the four named
// Bedrock campfire slots. Campfires are not inventory windows, so these item
// compounds are the only state Bedrock uses to render their contents.
func projectJavaCampfire(tag map[string]any) {
	rawItems, exists := tag["Items"]
	if !exists {
		return
	}
	items, ok := javaNBTCompoundList(rawItems)
	if !ok {
		return
	}
	for _, item := range items {
		slot, ok := javaNBTInt64Value(item["Slot"])
		if !ok || slot < 0 || slot >= javaCampfireSlotCount {
			continue
		}
		projected, ok := projectJavaBlockEntityItem(item)
		if !ok {
			continue
		}
		tag["Item"+strconv.FormatInt(slot+1, 10)] = projected
	}
	// The Java list is a different shape from Bedrock's Item1..Item4 fields;
	// retaining it would leave a second, semantically conflicting payload.
	delete(tag, "Items")
}

// projectJavaBlockEntityItem creates the Bedrock item compound used by tile
// entities such as campfires. Java 1.21.4 stores names and counts in the
// block-entity NBT rather than the negotiated item-slot wire format, so the
// generated name table is the authoritative bridge here.
func projectJavaBlockEntityItem(javaItem map[string]any) (map[string]any, bool) {
	name, ok := javaNBTStringValue(javaItem["id"])
	if !ok {
		name, ok = javaNBTStringValue(javaItem["Id"])
	}
	if !ok {
		return nil, false
	}
	if !strings.Contains(name, ":") {
		name = "minecraft:" + name
	}
	itemID, ok := data.JavaItemID(name)
	if !ok {
		return nil, false
	}
	runtimeID, ok := data.JavaItemRuntimeID(itemID)
	if !ok {
		return nil, false
	}
	bedrockName, ok := data.BedrockItemName(runtimeID)
	if !ok || bedrockName == "" {
		return nil, false
	}

	count := int64(1)
	if value, found := javaItem["count"]; found {
		if parsed, valid := javaNBTInt64Value(value); valid {
			count = parsed
		}
	} else if value, found := javaItem["Count"]; found {
		if parsed, valid := javaNBTInt64Value(value); valid {
			count = parsed
		}
	}
	if count <= 0 {
		return nil, false
	}
	// Bedrock encodes this field as a signed NBT byte. Vanilla Java stacks
	// are much smaller, but clamp malformed remote data instead of emitting a
	// negative Bedrock count.
	if count > 127 {
		count = 127
	}

	projected := map[string]any{
		"Name":   bedrockName,
		"Count":  byte(count),
		"Damage": int16(0),
	}
	if damage, ok := javaNBTInt64Value(javaItem["Damage"]); ok {
		if damage < 0 {
			damage = 0
		}
		if damage > 32767 {
			damage = 32767
		}
		projected["Damage"] = int16(damage)
	}

	itemTag := map[string]any{}
	if rawTag, ok := javaNBTCompound(javaItem["tag"]); ok {
		for key, value := range rawTag {
			itemTag[key] = value
		}
	}
	if components, ok := javaNBTCompound(javaItem["components"]); ok {
		mergeJavaItemComponents(itemTag, components)
	}
	if len(itemTag) != 0 {
		projected["tag"] = itemTag
	}
	return projected, true
}

// mergeJavaItemComponents carries the component fields whose Bedrock NBT
// representation is stable and useful inside a block-entity item. Unknown
// components remain omitted; their containing block entity still translates.
func mergeJavaItemComponents(target, components map[string]any) {
	if customData, ok := javaNBTCompound(components["minecraft:custom_data"]); ok {
		for key, value := range customData {
			target[key] = value
		}
	}

	display, _ := target["display"].(map[string]any)
	for _, key := range []string{"minecraft:custom_name", "minecraft:item_name"} {
		if value, exists := components[key]; exists {
			if text := projectJavaItemComponentText(value); text != "" {
				if display == nil {
					display = make(map[string]any)
				}
				display["Name"] = text
			}
		}
	}
	if lore, ok := javaNBTAnyList(components["minecraft:lore"]); ok {
		lines := make([]string, 0, len(lore))
		for _, value := range lore {
			lines = append(lines, projectJavaItemComponentText(value))
		}
		if len(lines) != 0 {
			if display == nil {
				display = make(map[string]any)
			}
			display["Lore"] = lines
		}
	}
	if display != nil && len(display) != 0 {
		target["display"] = display
	}

	if javaNBTBool(components["minecraft:unbreakable"]) {
		target["Unbreakable"] = byte(1)
	}
	if javaNBTBool(components["minecraft:enchantment_glint_override"]) {
		target["ench"] = []map[string]any{}
	}
}

func projectJavaItemComponentText(value any) string {
	if raw, ok := value.(string); ok {
		trimmed := strings.TrimSpace(raw)
		if trimmed != "" {
			var component any
			if json.Unmarshal([]byte(trimmed), &component) == nil {
				return JavaTextComponentText(component)
			}
		}
		return raw
	}
	return JavaTextComponentText(value)
}

func javaNBTStringValue(value any) (string, bool) {
	result, ok := value.(string)
	return result, ok && strings.TrimSpace(result) != ""
}

func javaNBTInt64Value(value any) (int64, bool) {
	switch value := value.(type) {
	case int8:
		return int64(value), true
	case uint8:
		return int64(value), true
	case int16:
		return int64(value), true
	case uint16:
		return int64(value), true
	case int32:
		return int64(value), true
	case uint32:
		return int64(value), true
	case int64:
		return value, true
	case uint64:
		if value > uint64(^uint64(0)>>1) {
			return 0, false
		}
		return int64(value), true
	case int:
		return int64(value), true
	case uint:
		if uint64(value) > uint64(^uint64(0)>>1) {
			return 0, false
		}
		return int64(value), true
	default:
		return 0, false
	}
}

func javaNBTCompound(value any) (map[string]any, bool) {
	compound, ok := value.(map[string]any)
	return compound, ok
}

func javaNBTAnyList(value any) ([]any, bool) {
	switch value := value.(type) {
	case []any:
		return value, true
	case []string:
		result := make([]any, len(value))
		for i, entry := range value {
			result[i] = entry
		}
		return result, true
	case []map[string]any:
		result := make([]any, len(value))
		for i, entry := range value {
			result[i] = entry
		}
		return result, true
	default:
		return nil, false
	}
}

func javaNBTCompoundList(value any) ([]map[string]any, bool) {
	items, ok := javaNBTAnyList(value)
	if !ok {
		return nil, false
	}
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		compound, ok := javaNBTCompound(item)
		if !ok {
			continue
		}
		result = append(result, compound)
	}
	return result, true
}

func projectJavaSign(tag map[string]any) {
	front, _ := tag["front_text"].(map[string]any)
	back, _ := tag["back_text"].(map[string]any)
	tag["FrontText"] = projectJavaSignSide(front)
	tag["BackText"] = projectJavaSignSide(back)
	tag["IsWaxed"] = javaNBTBool(tag["is_waxed"])
	delete(tag, "front_text")
	delete(tag, "back_text")
	delete(tag, "is_waxed")
}

func projectJavaSignSide(value map[string]any) map[string]any {
	result := map[string]any{
		"Text":           "",
		"IgnoreLighting": false,
	}
	if value == nil {
		return result
	}

	if rawMessages, ok := value["messages"]; ok {
		result["Text"] = projectJavaSignMessages(rawMessages)
	}
	if color, ok := javaNBTString(value, "color"); ok {
		result["SignTextColor"] = javaSignColor(color)
	}
	result["IgnoreLighting"] = javaNBTBool(value["has_glowing_text"])
	return result
}

func projectJavaSignMessages(value any) string {
	var messages []any
	switch value := value.(type) {
	case []any:
		messages = value
	case []string:
		for _, message := range value {
			messages = append(messages, message)
		}
	case []map[string]any:
		for _, message := range value {
			messages = append(messages, message)
		}
	default:
		return ""
	}

	lines := make([]string, 0, len(messages))
	for _, message := range messages {
		lines = append(lines, projectJavaSignMessage(message))
	}
	return strings.Join(lines, "\n")
}

func projectJavaSignMessage(value any) string {
	if raw, ok := value.(string); ok {
		trimmed := strings.TrimSpace(raw)
		if trimmed != "" {
			var component any
			if json.Unmarshal([]byte(trimmed), &component) == nil {
				return JavaTextComponentText(component)
			}
		}
		return raw
	}
	return JavaTextComponentText(value)
}

func javaNBTBool(value any) bool {
	switch value := value.(type) {
	case bool:
		return value
	case int8:
		return value != 0
	case uint8:
		return value != 0
	case int16:
		return value != 0
	case uint16:
		return value != 0
	case int32:
		return value != 0
	case uint32:
		return value != 0
	case int64:
		return value != 0
	case uint64:
		return value != 0
	case int:
		return value != 0
	case uint:
		return value != 0
	default:
		return false
	}
}

func javaNBTBoolValue(value any) (bool, bool) {
	switch value := value.(type) {
	case bool:
		return value, true
	case int8:
		return value != 0, true
	case uint8:
		return value != 0, true
	case int16:
		return value != 0, true
	case uint16:
		return value != 0, true
	case int32:
		return value != 0, true
	case uint32:
		return value != 0, true
	case int64:
		return value != 0, true
	case uint64:
		return value != 0, true
	case int:
		return value != 0, true
	case uint:
		return value != 0, true
	default:
		return false, false
	}
}

func javaNBTFloat32Value(value any) (float32, bool) {
	switch value := value.(type) {
	case float32:
		return value, true
	case float64:
		return float32(value), true
	default:
		return 0, false
	}
}

var javaSignColors = map[string]int32{
	"white":      16383998,
	"orange":     16351261,
	"magenta":    13061821,
	"light_blue": 3847130,
	"yellow":     16701501,
	"lime":       8439583,
	"pink":       15961002,
	"gray":       4673362,
	"light_gray": 10329495,
	"cyan":       1481884,
	"purple":     8991416,
	"blue":       3949738,
	"brown":      8606770,
	"green":      6192150,
	"red":        11546150,
}

// javaSignColor returns the ARGB value used by Bedrock's SignTextColor tag.
// The values are Geyser's vanilla Java dye-color table, including the opaque
// alpha channel expected by Bedrock.
func javaSignColor(color string) int32 {
	value := javaSignColors[color]
	return int32(uint32(value) | 0xff000000)
}
