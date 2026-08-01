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
	case "minecraft:acacia_boat", "minecraft:bamboo_raft", "minecraft:birch_boat",
		"minecraft:cherry_boat", "minecraft:dark_oak_boat", "minecraft:jungle_boat",
		"minecraft:mangrove_boat", "minecraft:oak_boat", "minecraft:pale_oak_boat",
		"minecraft:spruce_boat":
		return "minecraft:boat"
	case "minecraft:acacia_chest_boat", "minecraft:bamboo_chest_raft", "minecraft:birch_chest_boat",
		"minecraft:cherry_chest_boat", "minecraft:dark_oak_chest_boat", "minecraft:jungle_chest_boat",
		"minecraft:mangrove_chest_boat", "minecraft:oak_chest_boat", "minecraft:pale_oak_chest_boat",
		"minecraft:spruce_chest_boat":
		return "minecraft:chest_boat"
	case "minecraft:chest_minecart", "minecraft:command_block_minecart", "minecraft:furnace_minecart",
		"minecraft:hopper_minecart", "minecraft:minecart", "minecraft:spawner_minecart",
		"minecraft:tnt_minecart":
		return "minecraft:minecart"
	case "minecraft:end_crystal", "minecraft:ender_crystal":
		return "minecraft:ender_crystal"
	case "minecraft:evoker_fangs":
		return "minecraft:evocation_fang"
	case "minecraft:experience_bottle":
		return "minecraft:xp_bottle"
	case "minecraft:eye_of_ender":
		return "minecraft:eye_of_ender_signal"
	case "minecraft:potion", "minecraft:lingering_potion", "minecraft:splash_potion":
		// Java's registry uses potion for the thrown potion actor; Bedrock
		// exposes the splash-potion actor definition for both variants.
		return "minecraft:splash_potion"
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
	if javaThrowableItemEntity(entityType) {
		// ThrowableItemEntity applies the half-size scale and starts hidden so
		// a just-spawned projectile cannot obstruct the Bedrock camera before
		// its first draw tick.
		metadata[gtprotocol.EntityDataKeyScale] = float32(0.5)
		setProjectedFlag(metadata, gtprotocol.EntityDataFlagInvisible, true)
	} else if entityType == "minecraft:eye_of_ender" || entityType == "minecraft:eye_of_ender_signal" {
		// The Java eye is already the right actor, but its Bedrock definition
		// is twice the size without this entity-specific scale.
		metadata[gtprotocol.EntityDataKeyScale] = float32(0.5)
	}
	if javaBoatEntity(entityType) {
		if variant, ok := javaBoatVariant(entityType); ok {
			metadata[gtprotocol.EntityDataKeyVariant] = variant
		}
		metadata[gtprotocol.EntityDataKeyIsBuoyant] = byte(1)
		metadata[gtprotocol.EntityDataKeyBuoyancyData] = javaBoatBuoyancyData
		setProjectedFlag(metadata, gtprotocol.EntityDataFlagCollidable, true)
		return metadata, true
	}
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
	case "minecraft:fishing_bobber", "minecraft:fishing_hook":
		if objectData < 0 {
			return metadata, false
		}
		// The Basic session replaces this Java owner ID with the mapped
		// Bedrock runtime ID before it writes AddActor. Keep the raw payload
		// here so this pure projection remains useful to codec tests.
		metadata[gtprotocol.EntityDataKeyOwner] = int64(uint32(objectData))
	}
	return metadata, true
}

func javaThrowableItemEntity(entityType string) bool {
	switch entityType {
	case "minecraft:egg", "minecraft:ender_pearl", "minecraft:experience_bottle",
		"minecraft:lingering_potion", "minecraft:snowball", "minecraft:splash_potion",
		"minecraft:xp_bottle", "minecraft:potion":
		return true
	default:
		return false
	}
}

func javaFishingHookEntity(entityType string) bool {
	return entityType == "minecraft:fishing_bobber" || entityType == "minecraft:fishing_hook"
}

func setProjectedFlag(metadata gtprotocol.EntityMetadata, flag uint8, enabled bool) {
	key := uint32(gtprotocol.EntityDataKeyFlags)
	bit := flag
	if flag >= 64 {
		key = gtprotocol.EntityDataKeyFlagsTwo
		bit -= 64
	}
	if _, ok := metadata[key].(int64); !ok {
		metadata[key] = int64(0)
	}
	if enabled {
		metadata.SetFlag(key, bit)
	} else {
		metadata.UnsetFlag(key, bit)
	}
}

func javaAgeableEntity(entityType string) bool {
	switch entityType {
	case "minecraft:armadillo", "minecraft:axolotl", "minecraft:bee", "minecraft:camel",
		"minecraft:cat", "minecraft:chicken", "minecraft:cow", "minecraft:donkey", "minecraft:fox",
		"minecraft:frog", "minecraft:goat", "minecraft:hoglin", "minecraft:horse", "minecraft:llama",
		"minecraft:mooshroom", "minecraft:mule", "minecraft:ocelot", "minecraft:panda", "minecraft:pig",
		"minecraft:polar_bear", "minecraft:rabbit", "minecraft:sheep", "minecraft:sniffer",
		"minecraft:strider", "minecraft:turtle", "minecraft:wolf", "minecraft:parrot":
		return true
	default:
		return false
	}
}

func javaIntegerValue(value any) (int64, bool) {
	switch value := value.(type) {
	case int8:
		return int64(value), true
	case int32:
		return int64(value), true
	case int64:
		return value, true
	default:
		return 0, false
	}
}

func javaTameableOwnerMetadata(entityType string, entries []JavaEntityMetadataEntry) (ownerUUID [16]byte, hasEntry, hasUUID bool) {
	switch entityType {
	case "minecraft:cat", "minecraft:parrot", "minecraft:wolf":
	default:
		return [16]byte{}, false, false
	}
	for _, entry := range entries {
		if entry.Index != 18 {
			continue
		}
		switch value := entry.Value.(type) {
		case [16]byte:
			return value, true, true
		case nil:
			return [16]byte{}, true, false
		default:
			return [16]byte{}, true, false
		}
	}
	return [16]byte{}, false, false
}

// specialEntityFlagMasks reports the Bedrock flags whose values are owned by
// the Java metadata packet. The bridge must replace these bits rather than
// OR-ing them with the previous actor state, otherwise a Java false update
// leaves a stale Bedrock flag set.
func specialEntityFlagMasks(entityType string, entries []JavaEntityMetadataEntry) (int64, int64) {
	var mask, maskTwo int64
	add := func(flag uint8) {
		if flag >= 64 {
			maskTwo |= int64(1) << (flag - 64)
		} else {
			mask |= int64(1) << flag
		}
	}
	for _, entry := range entries {
		switch {
		case entry.Index == 15:
			add(gtprotocol.EntityDataFlagNoAI)
		case entry.Index == 6:
			switch entityType {
			case "minecraft:camel":
				add(gtprotocol.EntityDataFlagSitting)
			case "minecraft:frog":
				add(gtprotocol.EntityDataFlagJumpGoal)
				add(gtprotocol.EntityDataFlagCroaking)
				add(gtprotocol.EntityDataFlagDigestMob)
			case "minecraft:warden":
				add(gtprotocol.EntityDataFlagDigging)
				add(gtprotocol.EntityDataFlagEmerging)
				add(gtprotocol.EntityDataFlagRoaring)
				add(gtprotocol.EntityDataFlagSniffing)
			}
		case entityType == "minecraft:allay" && entry.Index == 16:
			add(gtprotocol.EntityDataFlagDancing)
		case entityType == "minecraft:armadillo" && entry.Index == 17:
			add(gtprotocol.EntityDataFlagRolling)
			add(gtprotocol.EntityDataFlagScared)
		case entityType == "minecraft:axolotl" && entry.Index == 18:
			add(gtprotocol.EntityDataFlagPlayingDead)
		case entityType == "minecraft:bat" && entry.Index == 16:
			add(gtprotocol.EntityDataFlagResting)
		case entityType == "minecraft:blaze" && entry.Index == 16:
			add(gtprotocol.EntityDataFlagOnFire)
		case entityType == "minecraft:camel" && entry.Index == 17:
			add(gtprotocol.EntityDataFlagTamed)
			add(gtprotocol.EntityDataFlagEating)
			add(gtprotocol.EntityDataFlagStanding)
		case entityType == "minecraft:camel" && entry.Index == 18:
			add(gtprotocol.EntityDataFlagHasDashTimeout)
		case entityType == "minecraft:enderman" && entry.Index == 18:
			add(gtprotocol.EntityDataFlagAngry)
		case entityType == "minecraft:horse" && entry.Index == 17:
			add(gtprotocol.EntityDataFlagTamed)
			add(gtprotocol.EntityDataFlagEating)
			add(gtprotocol.EntityDataFlagStanding)
		case (entityType == "minecraft:donkey" || entityType == "minecraft:mule" || entityType == "minecraft:llama" || entityType == "minecraft:trader_llama") && entry.Index == 17:
			add(gtprotocol.EntityDataFlagTamed)
			add(gtprotocol.EntityDataFlagEating)
			add(gtprotocol.EntityDataFlagStanding)
		case (entityType == "minecraft:donkey" || entityType == "minecraft:mule" || entityType == "minecraft:llama" || entityType == "minecraft:trader_llama") && entry.Index == 18:
			add(gtprotocol.EntityDataFlagChested)
		case entityType == "minecraft:ocelot" && entry.Index == 17:
			add(gtprotocol.EntityDataFlagTrusting)
		case entityType == "minecraft:polar_bear" && entry.Index == 17:
			add(gtprotocol.EntityDataFlagStanding)
		case entityType == "minecraft:sniffer" && entry.Index == 17:
			add(gtprotocol.EntityDataFlagFeelingHappy)
			add(gtprotocol.EntityDataFlagScenting)
			add(gtprotocol.EntityDataFlagSearching)
			add(gtprotocol.EntityDataFlagDigging)
			add(gtprotocol.EntityDataFlagRising)
		case (entityType == "minecraft:spider" || entityType == "minecraft:cave_spider") && entry.Index == 16:
			add(gtprotocol.EntityDataFlagWallClimbing)
		case entityType == "minecraft:strider" && entry.Index == 18:
			add(gtprotocol.EntityDataFlagBreathing)
			add(gtprotocol.EntityDataFlagShaking)
		case entityType == "minecraft:strider" && entry.Index == 19:
			add(gtprotocol.EntityDataFlagSaddled)
		case entityType == "minecraft:turtle" && entry.Index == 18:
			add(gtprotocol.EntityDataFlagPregnant)
		case entityType == "minecraft:turtle" && entry.Index == 19:
			add(gtprotocol.EntityDataFlagLayingEgg)
		case entry.Index == 16 && javaAgeableEntity(entityType):
			add(gtprotocol.EntityDataFlagBaby)
		case (entityType == "minecraft:end_crystal" || entityType == "minecraft:ender_crystal") && entry.Index == 9:
			add(gtprotocol.EntityDataFlagShowBottom)
		case entityType == "minecraft:tnt" && entry.Index == 8:
			add(gtprotocol.EntityDataFlagIgnited)
		case (entityType == "minecraft:arrow" || entityType == "minecraft:spectral_arrow" || entityType == "minecraft:trident") && entry.Index == 8:
			add(gtprotocol.EntityDataFlagCritical)
		case javaThrownPotionEntity(entityType) && entry.Index == 8:
			add(gtprotocol.EntityDataFlagEnchanted)
			add(gtprotocol.EntityDataFlagLingering)
		case entityType == "minecraft:trident" && entry.Index == 12:
			add(gtprotocol.EntityDataFlagEnchanted)
		case entityType == "minecraft:creeper":
			switch entry.Index {
			case 16:
				add(gtprotocol.EntityDataFlagIgnited)
			case 17:
				add(gtprotocol.EntityDataFlagPowered)
			case 18:
				add(gtprotocol.EntityDataFlagIgnited)
			}
		case entityType == "minecraft:sheep" && entry.Index == 17:
			add(gtprotocol.EntityDataFlagSheared)
		case entityType == "minecraft:armor_stand" && entry.Index == 15:
			add(gtprotocol.EntityDataFlagBaby)
			add(gtprotocol.EntityDataFlagAngry)
			add(gtprotocol.EntityDataFlagAdmiring)
		case (entityType == "minecraft:cat" || entityType == "minecraft:wolf" || entityType == "minecraft:parrot") && entry.Index == 17:
			add(gtprotocol.EntityDataFlagSitting)
			add(gtprotocol.EntityDataFlagAngry)
			add(gtprotocol.EntityDataFlagTamed)
		case entityType == "minecraft:cat" && entry.Index == 20:
			add(gtprotocol.EntityDataFlagResting)
		case entityType == "minecraft:wolf" && entry.Index == 19:
			add(gtprotocol.EntityDataFlagInterested)
		case entityType == "minecraft:wolf" && entry.Index == 21:
			add(gtprotocol.EntityDataFlagAngry)
		case entityType == "minecraft:fox" && entry.Index == 18:
			add(gtprotocol.EntityDataFlagSitting)
			add(gtprotocol.EntityDataFlagSneaking)
			add(gtprotocol.EntityDataFlagInterested)
			add(gtprotocol.EntityDataFlagSleeping)
		case entityType == "minecraft:rabbit" && entry.Index == 17:
			add(gtprotocol.EntityDataFlagBribed)
		case (entityType == "minecraft:zombie" || entityType == "minecraft:zombie_villager" || entityType == "minecraft:zombified_piglin" || entityType == "minecraft:drowned" || entityType == "minecraft:husk") && entry.Index == 16:
			add(gtprotocol.EntityDataFlagBaby)
		case (entityType == "minecraft:zombie" || entityType == "minecraft:zombie_villager" || entityType == "minecraft:zombified_piglin" || entityType == "minecraft:drowned" || entityType == "minecraft:husk") && entry.Index == 18:
			add(gtprotocol.EntityDataFlagShaking)
		case entityType == "minecraft:zombie_villager" && entry.Index == 19:
			add(gtprotocol.EntityDataFlagTransforming)
			add(gtprotocol.EntityDataFlagShaking)
		case entityType == "minecraft:bee" && entry.Index == 18:
			add(gtprotocol.EntityDataFlagAngry)
		}
	}
	return mask, maskTwo
}

// translateSpecialEntityMetadata projects metadata whose Bedrock key or
// meaning differs from Java's shared entity metadata. It returns only fields
// changed by this packet; spawn defaults are supplied by the function above.
func translateSpecialEntityMetadata(entityType string, entries []JavaEntityMetadataEntry) gtprotocol.EntityMetadata {
	return translateSpecialEntityMetadataWithVariants(entityType, entries, defaultJavaEntityVariantMappings())
}

func translateSpecialEntityMetadataWithVariants(entityType string, entries []JavaEntityMetadataEntry, variants javaEntityVariantMappings) gtprotocol.EntityMetadata {
	metadata := make(gtprotocol.EntityMetadata, 4)
	metadata[gtprotocol.EntityDataKeyFlags] = int64(0)
	flagsChanged := false
	for _, entry := range entries {
		if entry.Index == 15 {
			if mobFlags, ok := entry.Value.(int8); ok {
				setProjectedFlag(metadata, gtprotocol.EntityDataFlagNoAI, byte(mobFlags)&0x01 != 0)
				flagsChanged = true
			}
		}
		if entry.Index == 16 && javaAgeableEntity(entityType) {
			if baby, ok := entry.Value.(bool); ok {
				setProjectedFlag(metadata, gtprotocol.EntityDataFlagBaby, baby)
				flagsChanged = true
			}
		}
		switch entityType {
		case "minecraft:creeper":
			switch entry.Index {
			case 16:
				if swelling, ok := entry.Value.(int32); ok {
					setProjectedFlag(metadata, gtprotocol.EntityDataFlagIgnited, swelling == 1)
					flagsChanged = true
				}
			case 17:
				if powered, ok := entry.Value.(bool); ok {
					setProjectedFlag(metadata, gtprotocol.EntityDataFlagPowered, powered)
					flagsChanged = true
				}
			case 18:
				if ignited, ok := entry.Value.(bool); ok {
					setProjectedFlag(metadata, gtprotocol.EntityDataFlagIgnited, ignited)
					flagsChanged = true
				}
			}
		case "minecraft:sheep":
			if entry.Index == 17 {
				if sheepFlags, ok := entry.Value.(int8); ok {
					setProjectedFlag(metadata, gtprotocol.EntityDataFlagSheared, byte(sheepFlags)&0x10 != 0)
					metadata[gtprotocol.EntityDataKeyColorIndex] = byte(sheepFlags) & 0x0f
					flagsChanged = true
				}
			}
		case "minecraft:armor_stand":
			if entry.Index == 15 {
				if armorFlags, ok := entry.Value.(int8); ok {
					flags := byte(armorFlags)
					setProjectedFlag(metadata, gtprotocol.EntityDataFlagBaby, flags&0x01 != 0)
					setProjectedFlag(metadata, gtprotocol.EntityDataFlagAngry, flags&0x04 == 0)
					setProjectedFlag(metadata, gtprotocol.EntityDataFlagAdmiring, flags&0x08 != 0)
					flagsChanged = true
				}
			}
		case "minecraft:cat":
			switch entry.Index {
			case 17:
				if tameableFlags, ok := entry.Value.(int8); ok {
					flags := byte(tameableFlags)
					setProjectedFlag(metadata, gtprotocol.EntityDataFlagSitting, flags&0x01 != 0)
					setProjectedFlag(metadata, gtprotocol.EntityDataFlagAngry, flags&0x02 != 0)
					setProjectedFlag(metadata, gtprotocol.EntityDataFlagTamed, flags&0x04 != 0)
					flagsChanged = true
				}
			case 19:
				if variant, ok := entry.Value.(int32); ok && variant >= 0 {
					if bedrockVariant, ok := javaEntityVariantID(entityType, variant, variants); ok {
						metadata[gtprotocol.EntityDataKeyVariant] = bedrockVariant
					}
				}
			case 20:
				if resting, ok := entry.Value.(bool); ok {
					setProjectedFlag(metadata, gtprotocol.EntityDataFlagResting, resting)
					flagsChanged = true
				}
			case 22:
				if collar, ok := entry.Value.(int32); ok && collar >= 0 && collar <= 15 {
					metadata[gtprotocol.EntityDataKeyColorIndex] = byte(collar)
				}
			}
		case "minecraft:parrot":
			if entry.Index == 19 {
				if variant, ok := entry.Value.(int32); ok && variant >= 0 {
					metadata[gtprotocol.EntityDataKeyVariant] = variant
				}
			}
			if entry.Index == 17 {
				if tameableFlags, ok := entry.Value.(int8); ok {
					flags := byte(tameableFlags)
					setProjectedFlag(metadata, gtprotocol.EntityDataFlagSitting, flags&0x01 != 0)
					setProjectedFlag(metadata, gtprotocol.EntityDataFlagAngry, flags&0x02 != 0)
					setProjectedFlag(metadata, gtprotocol.EntityDataFlagTamed, flags&0x04 != 0)
					flagsChanged = true
				}
			}
		case "minecraft:wolf":
			switch entry.Index {
			case 17:
				if tameableFlags, ok := entry.Value.(int8); ok {
					flags := byte(tameableFlags)
					setProjectedFlag(metadata, gtprotocol.EntityDataFlagSitting, flags&0x01 != 0)
					setProjectedFlag(metadata, gtprotocol.EntityDataFlagAngry, flags&0x02 != 0)
					setProjectedFlag(metadata, gtprotocol.EntityDataFlagTamed, flags&0x04 != 0)
					flagsChanged = true
				}
			case 19:
				if interested, ok := entry.Value.(bool); ok {
					setProjectedFlag(metadata, gtprotocol.EntityDataFlagInterested, interested)
					flagsChanged = true
				}
			case 20:
				if collar, ok := entry.Value.(int32); ok && collar >= 0 && collar <= 15 {
					metadata[gtprotocol.EntityDataKeyColorIndex] = byte(collar)
				}
			case 21:
				if anger, ok := javaIntegerValue(entry.Value); ok {
					setProjectedFlag(metadata, gtprotocol.EntityDataFlagAngry, anger > 0)
					flagsChanged = true
				}
			case 22:
				if variant, ok := entry.Value.(JavaWolfVariant); ok {
					bedrockVariant := int32(0)
					if variant.RegistryID >= 0 {
						if mapped, mappedOK := javaEntityVariantID(entityType, variant.RegistryID, variants); mappedOK {
							bedrockVariant = mapped
						}
					}
					metadata[gtprotocol.EntityDataKeyVariant] = bedrockVariant
				}
			}
		case "minecraft:fox":
			switch entry.Index {
			case 17:
				if variant, ok := entry.Value.(int32); ok && variant >= 0 {
					metadata[gtprotocol.EntityDataKeyVariant] = variant
				}
			case 18:
				if foxFlags, ok := entry.Value.(int8); ok {
					flags := byte(foxFlags)
					setProjectedFlag(metadata, gtprotocol.EntityDataFlagSitting, flags&0x01 != 0)
					setProjectedFlag(metadata, gtprotocol.EntityDataFlagSneaking, flags&0x04 != 0)
					setProjectedFlag(metadata, gtprotocol.EntityDataFlagInterested, flags&0x08 != 0)
					setProjectedFlag(metadata, gtprotocol.EntityDataFlagSleeping, flags&0x20 != 0)
					flagsChanged = true
				}
			}
		case "minecraft:rabbit":
			if entry.Index == 17 {
				if variant, ok := entry.Value.(int32); ok {
					if variant == 99 {
						variant = 1
						setProjectedFlag(metadata, gtprotocol.EntityDataFlagBribed, true)
					} else {
						setProjectedFlag(metadata, gtprotocol.EntityDataFlagBribed, false)
					}
					metadata[gtprotocol.EntityDataKeyVariant] = variant
					flagsChanged = true
				}
			}
		case "minecraft:bee":
			switch entry.Index {
			case 17:
				if beeFlags, ok := entry.Value.(int8); ok {
					metadata[gtprotocol.EntityDataKeyMarkVariant] = int32(0)
					if byte(beeFlags)&0x04 != 0 {
						metadata[gtprotocol.EntityDataKeyMarkVariant] = int32(1)
					}
				}
			case 18:
				if anger, ok := javaIntegerValue(entry.Value); ok {
					setProjectedFlag(metadata, gtprotocol.EntityDataFlagAngry, anger > 0)
					flagsChanged = true
				}
			}
		case "minecraft:tropicalfish", "minecraft:tropical_fish":
			if entry.Index == 17 {
				if variant, ok := entry.Value.(int32); ok {
					packed := uint32(variant)
					shape := packed & 0xff
					if shape > 1 {
						shape = 1
					}
					pattern := (packed >> 8) & 0xff
					if pattern > 5 {
						pattern = 5
					}
					baseColor := byte((packed >> 16) & 0xff)
					if baseColor > 15 {
						baseColor = 0
					}
					patternColor := byte((packed >> 24) & 0xff)
					if patternColor > 15 {
						patternColor = 0
					}
					metadata[gtprotocol.EntityDataKeyVariant] = int32(shape)
					metadata[gtprotocol.EntityDataKeyMarkVariant] = int32(pattern)
					metadata[gtprotocol.EntityDataKeyColorIndex] = baseColor
					metadata[gtprotocol.EntityDataKeyColorTwoIndex] = patternColor
				}
			}
		case "minecraft:zombie", "minecraft:zombie_villager", "minecraft:zombified_piglin", "minecraft:drowned", "minecraft:husk":
			switch entry.Index {
			case 16:
				if baby, ok := entry.Value.(bool); ok {
					setProjectedFlag(metadata, gtprotocol.EntityDataFlagBaby, baby)
					flagsChanged = true
				}
			case 18:
				if converting, ok := entry.Value.(bool); ok {
					setProjectedFlag(metadata, gtprotocol.EntityDataFlagShaking, converting)
					flagsChanged = true
				}
			case 19:
				if entityType == "minecraft:zombie_villager" {
					if transforming, ok := entry.Value.(bool); ok {
						setProjectedFlag(metadata, gtprotocol.EntityDataFlagTransforming, transforming)
						setProjectedFlag(metadata, gtprotocol.EntityDataFlagShaking, transforming)
						flagsChanged = true
					}
				}
			}
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
	for key, value := range translateEntityMetadataMatrix(entityType, entries) {
		if key == gtprotocol.EntityDataKeyFlags || key == gtprotocol.EntityDataKeyFlagsTwo {
			current, _ := metadata[key].(int64)
			additional, _ := value.(int64)
			metadata[key] = current | additional
			flagsChanged = true
		} else {
			metadata[key] = value
		}
	}
	if !flagsChanged {
		delete(metadata, gtprotocol.EntityDataKeyFlags)
		delete(metadata, gtprotocol.EntityDataKeyFlagsTwo)
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
