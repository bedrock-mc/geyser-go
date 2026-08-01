package translate

import (
	"math"
	"reflect"

	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// javaPotionBedrockIDs is the Java 1.21.4 Potion registry ordinal to Bedrock
// aux-value table used by Geyser's Potion enum. The Java registry ID is the
// enum ordinal; it is not the Bedrock damage value for several potions.
var javaPotionBedrockIDs = [...]int16{
	0, 1, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
	13, 14, 15, 16, 17, 18, 42, 37, 38, 39, 19, 20,
	21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32,
	33, 34, 35, 2, 40, 41, 43, 44, 45, 46,
}

type javaFireworkNamedColor struct {
	rgb int32
	h   float64
	s   float64
	v   float64
}

// These are the sixteen Bedrock firework colours used by Geyser. Java stores
// arbitrary RGB values, while Bedrock stores the ordinal of this palette in a
// byte array.
var javaFireworkColors = [...]javaFireworkNamedColor{
	{rgb: 1973019},  // black
	{rgb: 11743532}, // red
	{rgb: 3887386},  // green
	{rgb: 5320730},  // brown
	{rgb: 2437522},  // blue
	{rgb: 8073150},  // purple
	{rgb: 2651799},  // cyan
	{rgb: 11250603}, // light gray
	{rgb: 4408131},  // gray
	{rgb: 14188952}, // pink
	{rgb: 4312372},  // lime
	{rgb: 14602026}, // yellow
	{rgb: 6719955},  // light blue
	{rgb: 12801229}, // magenta
	{rgb: 15435844}, // orange
	{rgb: 15790320}, // white
}

func init() {
	for i := range javaFireworkColors {
		javaFireworkColors[i].h, javaFireworkColors[i].s, javaFireworkColors[i].v = javaRGBToHSV(javaFireworkColors[i].rgb)
	}
}

func javaRGBToHSV(rgb int32) (h, s, v float64) {
	r := float64(uint32(rgb)>>16&0xff) / 255
	g := float64(uint32(rgb)>>8&0xff) / 255
	b := float64(uint32(rgb)&0xff) / 255
	maxValue := math.Max(r, math.Max(g, b))
	minValue := math.Min(r, math.Min(g, b))
	delta := maxValue - minValue
	v = maxValue
	if maxValue == 0 {
		return 0, 0, 0
	}
	s = delta / maxValue
	if delta == 0 {
		return 0, s, v
	}
	switch maxValue {
	case r:
		h = (g - b) / delta
	case g:
		h = 2 + (b-r)/delta
	default:
		h = 4 + (r-g)/delta
	}
	h /= 6
	if h < 0 {
		h++
	}
	return h, s, v
}

func javaFireworkColorID(rgb int32) byte {
	h, s, v := javaRGBToHSV(rgb)
	best := 0
	bestDistance := math.MaxFloat64
	for i, candidate := range javaFireworkColors {
		hueDistance := 3 * math.Min(math.Abs(h-candidate.h), 1-math.Abs(h-candidate.h))
		saturationDiff := s - candidate.s
		valueDiff := v - candidate.v
		distance := hueDistance*hueDistance + saturationDiff*saturationDiff + valueDiff*valueDiff
		if distance < bestDistance {
			best = i
			bestDistance = distance
		}
		if distance == 0 {
			break
		}
	}
	return byte(best)
}

// javaFireworkByteArray returns a runtime-sized array rather than []byte.
// Gophertunnel's NBT encoder intentionally distinguishes a byte array (Go
// array) from a list of byte tags (Go slice); reflect.ArrayOf preserves the
// Bedrock TAG_ByteArray wire type for arbitrary Java colour counts.
func javaFireworkByteArray(values []byte) any {
	arrayType := reflect.ArrayOf(len(values), reflect.TypeOf(byte(0)))
	array := reflect.New(arrayType).Elem()
	for i, value := range values {
		array.Index(i).SetUint(uint64(value))
	}
	return array.Interface()
}

func javaFireworkExplosionNBT(explosion JavaFireworkExplosion) map[string]any {
	colors := make([]byte, len(explosion.Colors))
	for i, color := range explosion.Colors {
		colors[i] = javaFireworkColorID(color)
	}
	fadeColors := make([]byte, len(explosion.FadeColors))
	for i, color := range explosion.FadeColors {
		fadeColors[i] = javaFireworkColorID(color)
	}
	return map[string]any{
		"FireworkType":    byte(explosion.Shape),
		"FireworkColor":   javaFireworkByteArray(colors),
		"FireworkFade":    javaFireworkByteArray(fadeColors),
		"FireworkTrail":   explosion.Trail,
		"FireworkFlicker": explosion.Flicker,
	}
}

func javaFireworksNBT(fireworks JavaFireworksData) map[string]any {
	explosions := make([]map[string]any, len(fireworks.Explosions))
	for i, explosion := range fireworks.Explosions {
		explosions[i] = javaFireworkExplosionNBT(explosion)
	}
	return map[string]any{
		"Flight":     byte(fireworks.FlightDuration),
		"Explosions": explosions,
	}
}

func javaFireworkStarNBT(explosion JavaFireworkExplosion) map[string]any {
	tag := javaFireworkExplosionNBT(explosion)
	if len(explosion.Colors) == 0 {
		return tag
	}
	if len(explosion.Colors) == 1 {
		tag["customColor"] = explosion.Colors[0]
		return tag
	}
	var red, green, blue int32
	for _, color := range explosion.Colors {
		red += color >> 16 & 0xff
		green += color >> 8 & 0xff
		blue += color & 0xff
	}
	count := int32(len(explosion.Colors))
	tag["customColor"] = red/count<<16 | green/count<<8 | blue/count
	return tag
}

func projectJavaFireworkItemNBT(item gtprotocol.ItemInstance, fireworks *JavaFireworksData, explosion *JavaFireworkExplosion) gtprotocol.ItemInstance {
	if fireworks == nil && explosion == nil {
		return item
	}
	if item.Stack.NBTData == nil {
		item.Stack.NBTData = make(map[string]any)
	}
	if fireworks != nil {
		item.Stack.NBTData["Fireworks"] = javaFireworksNBT(*fireworks)
	}
	if explosion != nil {
		item.Stack.NBTData["FireworksItem"] = javaFireworkStarNBT(*explosion)
	}
	return item
}

func javaPotionBedrockID(javaID int32) (int16, bool) {
	if javaID < 0 || int64(javaID) >= int64(len(javaPotionBedrockIDs)) {
		return 0, false
	}
	return javaPotionBedrockIDs[javaID], true
}

func javaPotionIsEnchanted(javaID int32) bool {
	// Water, mundane, thick, and awkward are Geyser's four non-enchanted
	// potion variants. These values are Java registry ordinals.
	switch javaID {
	case 0, 1, 2, 3:
		return false
	default:
		return true
	}
}

func javaThrownPotionEntity(entityType string) bool {
	switch entityType {
	case "minecraft:potion", "minecraft:splash_potion", "minecraft:lingering_potion":
		return true
	default:
		return false
	}
}

func translateJavaPotionEntityMetadata(entityType string, item JavaEntityItemMetadata, hasItemUpdate bool, metadata gtprotocol.EntityMetadata) {
	if !javaThrownPotionEntity(entityType) || !hasItemUpdate {
		return
	}
	auxValue := int16(0)
	enchanted := false
	if item.HasPotionID {
		if bedrockID, ok := javaPotionBedrockID(item.PotionID); ok {
			auxValue = bedrockID
			enchanted = javaPotionIsEnchanted(item.PotionID)
		}
	}
	metadata[gtprotocol.EntityDataKeyAuxValueData] = auxValue
	setProjectedFlag(metadata, gtprotocol.EntityDataFlagEnchanted, enchanted)
	setProjectedFlag(metadata, gtprotocol.EntityDataFlagLingering, entityType == "minecraft:lingering_potion")
}

// updateJavaFireworkAttachmentLocked projects Java's optional gliding-owner
// metadata into Bedrock's movement prediction effect. The caller holds b.mu.
func (b *Basic) updateJavaFireworkAttachmentLocked(entityID int32, entity *javaEntityState, entries []JavaEntityMetadataEntry) (duration int32, send bool) {
	if entity == nil || entity.entityType != "minecraft:fireworks_rocket" {
		return 0, false
	}
	for _, entry := range entries {
		if entry.Index != 9 {
			continue
		}
		owner, hasOwner := javaIntegerValue(entry.Value)
		entity.fireworkAttachedToEntity = hasOwner
		attachedToPlayer := hasOwner && owner == int64(int32(uint32(b.gameData.EntityRuntimeID)))
		if attachedToPlayer == entity.fireworkAttachedToPlayer {
			continue
		}
		entity.fireworkAttachedToPlayer = attachedToPlayer
		if b.fireworkAttachments == nil {
			b.fireworkAttachments = make(map[int32]struct{})
		}
		if attachedToPlayer {
			b.fireworkAttachments[entityID] = struct{}{}
			return 1000000, true
		}
		delete(b.fireworkAttachments, entityID)
		if len(b.fireworkAttachments) == 0 {
			return 0, true
		}
	}
	return 0, false
}

func (b *Basic) writeFireworkMovementEffect(bedrock interface {
	WritePacket(packet.Packet) error
}, duration int32) error {
	b.mu.Lock()
	runtimeID := b.gameData.EntityRuntimeID
	tick := b.clientTick
	b.mu.Unlock()
	return bedrock.WritePacket(&packet.MovementEffect{
		EntityRuntimeID: runtimeID,
		Type:            packet.MovementEffectTypeGlideBoost,
		Duration:        duration,
		Tick:            tick,
	})
}
