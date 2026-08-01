package translate

import (
	"math"
	"math/rand"
	"strings"

	"github.com/bedrock-mc/geyser-go/data"
	"github.com/go-gl/mathgl/mgl32"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// bedrockEntityType applies the identifier overrides used by Geyser's
// VanillaEntities registry. Most Java and Bedrock actors share an identifier,
// but these object/entity definitions do not.
func bedrockEntityType(javaType string) string {
	switch javaType {
	case "minecraft:end_crystal", "minecraft:ender_crystal":
		return "minecraft:ender_crystal"
	case "minecraft:evoker_fangs":
		return "minecraft:evocation_fang"
	case "minecraft:experience_bottle":
		return "minecraft:xp_bottle"
	case "minecraft:eye_of_ender":
		return "minecraft:eye_of_ender_signal"
	case "minecraft:firework_rocket":
		return "minecraft:fireworks_rocket"
	case "minecraft:fishing_bobber":
		return "minecraft:fishing_hook"
	case "minecraft:breeze_wind_charge":
		return "minecraft:breeze_wind_charge_projectile"
	case "minecraft:wind_charge":
		return "minecraft:wind_charge_projectile"
	case "minecraft:trident":
		return "minecraft:thrown_trident"
	case "minecraft:text_display", "minecraft:interaction":
		// Geyser renders both Java display/interaction entities using an
		// invisible armor-stand-backed actor. The metadata projection below
		// removes the armor-stand body while retaining the name or hitbox.
		return "minecraft:armor_stand"
	case "minecraft:zombie_villager":
		return "minecraft:zombie_villager_v2"
	case "minecraft:zombified_piglin":
		return "minecraft:zombie_pigman"
	case "minecraft:tropical_fish":
		return "minecraft:tropicalfish"
	case "minecraft:villager":
		return "minecraft:villager_v2"
	default:
		return javaType
	}
}

func javaEntitySpawnPosition(entityType string, position mgl32.Vec3) mgl32.Vec3 {
	if entityType == "minecraft:leash_knot" {
		// Java's hanging knot position is the block corner; Bedrock expects the
		// knot actor centered on the block and raised by a quarter block.
		return position.Add(mgl32.Vec3{0.5, 0.25, 0.5})
	}
	return position
}

// javaSpawnEntityProjection returns the Bedrock metadata that must accompany
// Java's special object entities. Java puts the payload in Spawn Entity's
// object-data field, while Bedrock expects most of these values in actor
// metadata.
//
// The bool reports whether the entity can be projected safely. A fishing hook
// without an owner has no lossless Bedrock representation: Geyser drops that
// spawn rather than emitting a hook with a broken fishing line.
func javaSpawnEntityProjection(entityType string, objectData int32) (gtprotocol.EntityMetadata, bool) {
	metadata := gtprotocol.NewEntityMetadataWithCapacity(8)
	switch entityType {
	case "minecraft:text_display":
		metadata[gtprotocol.EntityDataKeyHitBox] = map[string]any{}
		metadata[gtprotocol.EntityDataKeyScale] = float32(0)
		metadata[gtprotocol.EntityDataKeyAlwaysShowNameTag] = byte(1)
	case "minecraft:interaction":
		metadata.SetFlag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagInvisible)
		metadata[gtprotocol.EntityDataKeyWidth] = float32(1)
		metadata[gtprotocol.EntityDataKeyHeight] = float32(1)
	case "minecraft:area_effect_cloud":
		metadata.SetFlag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagFireImmune)
		metadata[gtprotocol.EntityDataKeyDataDuration] = int32(math.MaxInt32)
		metadata[gtprotocol.EntityDataKeyDataRadius] = float32(3)
		metadata[gtprotocol.EntityDataKeyDataChangeRate] = float32(math.SmallestNonzeroFloat32)
		metadata[gtprotocol.EntityDataKeyDataChangeOnPickup] = float32(math.SmallestNonzeroFloat32)
	case "minecraft:end_crystal", "minecraft:ender_crystal":
		metadata.SetFlag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagFireImmune)
		metadata[gtprotocol.EntityDataKeyBlockTarget] = gtprotocol.BlockPos{}
	case "minecraft:spectral_arrow":
		// Geyser uses this Bedrock flag to select the spectral-arrow texture.
		metadata.SetFlag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagBribed)
	case "minecraft:xp_orb":
		// Geyser uses this marker to select the Bedrock XP-orb texture. The
		// Java object-data value is the orb's XP amount, not this Bedrock
		// metadata field, and the vanilla projection keeps the marker at 1.
		metadata[gtprotocol.EntityDataKeyTradeExperience] = int32(1)
	case "minecraft:falling_block":
		runtimeID, known := JavaBlockRuntimeID(objectData)
		metadata[gtprotocol.EntityDataKeyDisplayTileRuntimeID] = int32(runtimeID)
		return metadata, known
	case "minecraft:fishing_hook":
		if objectData < 0 {
			return metadata, false
		}
		// This bridge deliberately uses the Java entity ID as the Bedrock
		// runtime ID, so the owner can be projected without a second lookup.
		metadata[gtprotocol.EntityDataKeyOwner] = int64(uint32(objectData))
	}
	return metadata, true
}

// translateSpecialEntityMetadata projects metadata whose Bedrock key or
// meaning differs from Java's shared entity metadata. It returns only fields
// changed by this packet; spawn defaults are supplied by the function above.
func translateSpecialEntityMetadata(entityType string, entries []JavaEntityMetadataEntry) gtprotocol.EntityMetadata {
	metadata := make(gtprotocol.EntityMetadata, 4)
	metadata[gtprotocol.EntityDataKeyFlags] = int64(0)
	flagsChanged := false
	for _, entry := range entries {
		switch entityType {
		case "minecraft:area_effect_cloud":
			if entry.Index == 8 {
				if radius, ok := entry.Value.(float32); ok && finiteFloat32(radius) {
					metadata[gtprotocol.EntityDataKeyDataRadius] = clampFloat32(radius, 0.5, 32)
				}
			}
		case "minecraft:end_crystal", "minecraft:ender_crystal":
			switch entry.Index {
			case 8:
				if position, ok := entry.Value.(gtprotocol.BlockPos); ok {
					metadata[gtprotocol.EntityDataKeyBlockTarget] = position
				} else if entry.Value == nil {
					metadata[gtprotocol.EntityDataKeyBlockTarget] = gtprotocol.BlockPos{}
				}
			case 9:
				showBottom, ok := entry.Value.(bool)
				if !ok {
					continue
				}
				if showBottom {
					metadata[gtprotocol.EntityDataKeyFlags] = int64(1) << gtprotocol.EntityDataFlagShowBottom
				} else {
					metadata[gtprotocol.EntityDataKeyFlags] = int64(0)
				}
				flagsChanged = true
			}
		case "minecraft:tnt":
			if entry.Index == 8 {
				if fuse, ok := entry.Value.(int32); ok && fuse >= 0 {
					metadata[gtprotocol.EntityDataKeyFlags] = int64(1) << gtprotocol.EntityDataFlagIgnited
					metadata[gtprotocol.EntityDataKeyFuseTime] = fuse
					flagsChanged = true
				}
			}
		case "minecraft:arrow", "minecraft:spectral_arrow", "minecraft:trident":
			switch entry.Index {
			case 8:
				if arrowFlags, ok := entry.Value.(int8); ok {
					flags := metadata[gtprotocol.EntityDataKeyFlags].(int64)
					if arrowFlags&0x01 != 0 {
						flags |= int64(1) << gtprotocol.EntityDataFlagCritical
					} else {
						flags &^= int64(1) << gtprotocol.EntityDataFlagCritical
					}
					metadata[gtprotocol.EntityDataKeyFlags] = flags
					flagsChanged = true
				}
			case 11:
				if entityType == "minecraft:arrow" {
					if color, ok := entry.Value.(int32); ok {
						metadata[gtprotocol.EntityDataKeyCustomDisplay] = tippedArrowDisplayID(color)
					}
				}
			case 12:
				if entityType == "minecraft:trident" {
					if enchanted, ok := entry.Value.(bool); ok {
						flags := metadata[gtprotocol.EntityDataKeyFlags].(int64)
						if enchanted {
							flags |= int64(1) << gtprotocol.EntityDataFlagEnchanted
						} else {
							flags &^= int64(1) << gtprotocol.EntityDataFlagEnchanted
						}
						metadata[gtprotocol.EntityDataKeyFlags] = flags
						flagsChanged = true
					}
				}
			}
		case "minecraft:text_display":
			if entry.Index == 23 {
				// Bedrock armor stands expose their nametag as a plain string.
				// Keep the empty value as an explicit update so Java can clear
				// an existing display without leaving stale text client-side.
				metadata[gtprotocol.EntityDataKeyName] = JavaTextComponentText(entry.Value)
			}
		case "minecraft:interaction":
			switch entry.Index {
			case 8:
				if width, ok := entry.Value.(float32); ok && finiteFloat32(width) && width >= 0 {
					metadata[gtprotocol.EntityDataKeyWidth] = width
				}
			case 9:
				if height, ok := entry.Value.(float32); ok && finiteFloat32(height) && height >= 0 {
					metadata[gtprotocol.EntityDataKeyHeight] = clampFloat32(height, 0, 64)
				}
			}
		}
	}
	if !flagsChanged {
		delete(metadata, gtprotocol.EntityDataKeyFlags)
	}
	return metadata
}

// javaTextDisplayLineOffset matches Geyser's armor-stand nametag adjustment
// for Java text-display entities. Empty text has no visible line to offset.
func javaTextDisplayLineOffset(text string) float32 {
	if text == "" {
		return 0
	}
	lineCount := 1 + strings.Count(text, "\n")
	return -0.6 + 0.1414*float32(lineCount)
}

func tippedArrowDisplayID(color int32) byte {
	if color < 0 {
		return 0
	}
	if id, ok := tippedArrowColors[color]; ok {
		return id
	}
	return 0
}

// tippedArrowColors is the Geyser Potion.toTippedArrowId table for the
// version-pinned Java potion particle colours.
var tippedArrowColors = map[int32]byte{
	3694022:  1,  // water
	12779366: 6,  // night vision
	16185078: 8,  // invisibility
	16646020: 10, // leaping
	16750848: 13, // fire resistance
	3402751:  15, // swiftness
	9154528:  18, // slowness
	9274086:  38, // turtle master
	9274854:  40, // strong turtle master
	10017472: 20, // water breathing
	16262179: 22, // healing
	11101546: 24, // harming
	8889187:  26, // poison
	13458603: 29, // regeneration
	16762624: 32, // strength
	4738376:  35, // weakness
	5882118:  3,  // luck
	15978425: 41, // slow falling
	12438015: 44, // wind charging
	7891290:  45, // weaving
	10092451: 46, // oozing
	9214860:  47, // infestation
}

func javaLightningSounds(position mgl32.Vec3) []packet.PlaySound {
	return []packet.PlaySound{
		{
			SoundName: data.BedrockSoundName("entity.lightning_bolt.thunder"),
			Position:  position,
			Volume:    10000,
			Pitch:     0.8 + rand.Float32()*0.2,
		},
		{
			SoundName: data.BedrockSoundName("entity.lightning_bolt.impact"),
			Position:  position,
			Volume:    2,
			Pitch:     0.5 + rand.Float32()*0.2,
		},
	}
}
