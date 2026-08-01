package translate

import (
	"math"

	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
)

// translateEntityMetadataMatrix contains the entity-specific fields that do
// not fit Java's shared entity metadata layout. The indexes are for the
// pinned Java 1.21.4 inheritance tree: entity (0-7), living (8-14), mob (15),
// followed by the concrete entity's fields. In particular, 1.21.4's ageable
// base owns only index 16; newer protocol profiles add fields that shift some
// of these concrete indexes.
func translateEntityMetadataMatrix(entityType string, entries []JavaEntityMetadataEntry) gtprotocol.EntityMetadata {
	metadata := make(gtprotocol.EntityMetadata, 8)
	for _, entry := range entries {
		switch {
		case entry.Index == 6:
			translateJavaPoseMetadata(entityType, entry.Value, metadata)
		case entityType == "minecraft:allay" && entry.Index == 16:
			if dancing, ok := entry.Value.(bool); ok {
				setProjectedFlag(metadata, gtprotocol.EntityDataFlagDancing, dancing)
			}
		case entityType == "minecraft:armadillo" && entry.Index == 17:
			if state, ok := entry.Value.(int32); ok {
				setProjectedFlag(metadata, gtprotocol.EntityDataFlagRolling, state == 1)
				setProjectedFlag(metadata, gtprotocol.EntityDataFlagScared, state == 2)
			}
		case entityType == "minecraft:axolotl":
			switch entry.Index {
			case 17:
				if variant, ok := entry.Value.(int32); ok && variant >= 0 {
					switch variant {
					case 1: // Java wild (brown) -> Bedrock wild.
						variant = 3
					case 3: // Java cyan -> Bedrock cyan.
						variant = 1
					}
					metadata[gtprotocol.EntityDataKeyVariant] = variant
				}
			case 18:
				if playingDead, ok := entry.Value.(bool); ok {
					setProjectedFlag(metadata, gtprotocol.EntityDataFlagPlayingDead, playingDead)
				}
			}
		case entityType == "minecraft:bat" && entry.Index == 16:
			if flags, ok := entry.Value.(int8); ok {
				setProjectedFlag(metadata, gtprotocol.EntityDataFlagResting, byte(flags)&0x01 != 0)
			}
		case entityType == "minecraft:blaze" && entry.Index == 16:
			if flags, ok := entry.Value.(int8); ok {
				setProjectedFlag(metadata, gtprotocol.EntityDataFlagOnFire, byte(flags)&0x01 != 0)
			}
		case entityType == "minecraft:camel":
			switch entry.Index {
			case 17:
				translateCamelHorseFlags(entry.Value, metadata)
			case 18:
				if dashing, ok := entry.Value.(bool); ok {
					setProjectedFlag(metadata, gtprotocol.EntityDataFlagHasDashTimeout, dashing)
				}
			}
		case entityType == "minecraft:enderman":
			switch entry.Index {
			case 16:
				if blockState, ok := entry.Value.(int32); ok {
					if runtimeID, known := JavaBlockRuntimeID(blockState); known {
						metadata[gtprotocol.EntityDataKeyCarryBlockRuntimeID] = int32(runtimeID)
					}
				}
			case 18:
				if angry, ok := entry.Value.(bool); ok {
					setProjectedFlag(metadata, gtprotocol.EntityDataFlagAngry, angry)
				}
			}
		case entityType == "minecraft:frog":
			switch entry.Index {
			case 17:
				if variant, ok := entry.Value.(int32); ok && variant >= 0 {
					switch variant {
					case 1: // Java white -> Bedrock white.
						variant = 2
					case 2: // Java green -> Bedrock green.
						variant = 1
					}
					metadata[gtprotocol.EntityDataKeyVariant] = variant
				}
			}
		case entityType == "minecraft:ghast" && entry.Index == 16:
			if attacking, ok := entry.Value.(bool); ok {
				if attacking {
					metadata[gtprotocol.EntityDataKeyChargeAmount] = byte(1)
				} else {
					metadata[gtprotocol.EntityDataKeyChargeAmount] = byte(0)
				}
			}
		case entityType == "minecraft:horse":
			switch entry.Index {
			case 17:
				translateHorseFlags(entry.Value, metadata)
			case 18:
				if variant, ok := entry.Value.(int32); ok && variant >= 0 {
					metadata[gtprotocol.EntityDataKeyVariant] = variant & 0xff
					metadata[gtprotocol.EntityDataKeyMarkVariant] = (variant >> 8) % 5
				}
			}
		case entityType == "minecraft:donkey" || entityType == "minecraft:mule":
			switch entry.Index {
			case 17:
				translateHorseFlags(entry.Value, metadata)
			case 18:
				if chested, ok := entry.Value.(bool); ok {
					setProjectedFlag(metadata, gtprotocol.EntityDataFlagChested, chested)
				}
			}
		case entityType == "minecraft:llama" || entityType == "minecraft:trader_llama":
			switch entry.Index {
			case 17:
				translateHorseFlags(entry.Value, metadata)
			case 18:
				if chested, ok := entry.Value.(bool); ok {
					setProjectedFlag(metadata, gtprotocol.EntityDataFlagChested, chested)
				}
			case 19:
				if strength, ok := entry.Value.(int32); ok {
					metadata[gtprotocol.EntityDataKeyStrength] = strength
				}
			case 20:
				if variant, ok := entry.Value.(int32); ok && variant >= 0 {
					metadata[gtprotocol.EntityDataKeyVariant] = variant
				}
			}
		case entityType == "minecraft:mooshroom" && entry.Index == 17:
			if variant, ok := entry.Value.(string); ok {
				if variant == "brown" {
					metadata[gtprotocol.EntityDataKeyVariant] = int32(1)
				} else {
					metadata[gtprotocol.EntityDataKeyVariant] = int32(0)
				}
			}
		case entityType == "minecraft:ocelot" && entry.Index == 17:
			if trusting, ok := entry.Value.(bool); ok {
				setProjectedFlag(metadata, gtprotocol.EntityDataFlagTrusting, trusting)
			}
		case entityType == "minecraft:phantom" && entry.Index == 16:
			if size, ok := entry.Value.(int32); ok {
				if size < 0 {
					size = 0
				}
				metadata[gtprotocol.EntityDataKeyScale] = 1 + 0.15*float32(size)
			}
		case entityType == "minecraft:polar_bear" && entry.Index == 17:
			if standing, ok := entry.Value.(bool); ok {
				setProjectedFlag(metadata, gtprotocol.EntityDataFlagStanding, standing)
			}
		case entityType == "minecraft:pufferfish" && entry.Index == 17:
			if puffed, ok := entry.Value.(int32); ok {
				if puffed < 0 {
					puffed = 0
				}
				if puffed > math.MaxUint8 {
					puffed = math.MaxUint8
				}
				metadata[gtprotocol.EntityDataKeyPuffedState] = byte(puffed)
				metadata[gtprotocol.EntityDataKeyVariant] = puffed
			}
		case entityType == "minecraft:shulker":
			switch entry.Index {
			case 16:
				if direction, ok := entry.Value.(int32); ok && direction >= 0 && direction <= 5 {
					metadata[gtprotocol.EntityDataKeyAttachFace] = direction
				}
			case 17:
				if peek, ok := entry.Value.(int8); ok {
					metadata[gtprotocol.EntityDataKeyPeekID] = int32(peek)
				}
			case 18:
				if color, ok := entry.Value.(int8); ok {
					if color == 16 {
						metadata[gtprotocol.EntityDataKeyVariant] = int32(16)
					} else if color >= 0 {
						metadata[gtprotocol.EntityDataKeyVariant] = int32(math.Abs(float64(color) - 15))
					}
				}
			}
		case entityType == "minecraft:sniffer" && entry.Index == 17:
			if state, ok := entry.Value.(int32); ok {
				translateSnifferState(state, metadata)
			}
		case entityType == "minecraft:spider" || entityType == "minecraft:cave_spider":
			if entry.Index == 16 {
				if flags, ok := entry.Value.(int8); ok {
					setProjectedFlag(metadata, gtprotocol.EntityDataFlagWallClimbing, byte(flags)&0x01 != 0)
				}
			}
		case entityType == "minecraft:strider":
			switch entry.Index {
			case 18:
				if cold, ok := entry.Value.(bool); ok {
					setProjectedFlag(metadata, gtprotocol.EntityDataFlagBreathing, !cold)
					setProjectedFlag(metadata, gtprotocol.EntityDataFlagShaking, cold)
				}
			case 19:
				if saddled, ok := entry.Value.(bool); ok {
					setProjectedFlag(metadata, gtprotocol.EntityDataFlagSaddled, saddled)
				}
			}
		case entityType == "minecraft:turtle":
			switch entry.Index {
			case 18:
				if pregnant, ok := entry.Value.(bool); ok {
					setProjectedFlag(metadata, gtprotocol.EntityDataFlagPregnant, pregnant)
				}
			case 19:
				if layingEgg, ok := entry.Value.(bool); ok {
					setProjectedFlag(metadata, gtprotocol.EntityDataFlagLayingEgg, layingEgg)
				}
			}
		case entityType == "minecraft:warden" && entry.Index == 16:
			if anger, ok := entry.Value.(int32); ok {
				if anger < 0 {
					anger = 0
				}
				if anger > 80 {
					anger = 80
				}
				metadata[gtprotocol.EntityDataKeyHeartbeatIntervalTicks] = int32(40 - math.Floor(float64(anger)*30/80))
			}
		case entityType == "minecraft:wither" && entry.Index == 19:
			if ticks, ok := entry.Value.(int32); ok && ticks >= 0 {
				metadata[gtprotocol.EntityDataKeyInvulnerableTicks] = ticks
				if ticks >= 165 {
					metadata[gtprotocol.EntityDataKeyAerialAttack] = int16(0)
				} else {
					metadata[gtprotocol.EntityDataKeyAerialAttack] = int16(1)
				}
			}
		}
	}
	return metadata
}

func translateJavaPoseMetadata(entityType string, value any, metadata gtprotocol.EntityMetadata) {
	pose, ok := value.(int32)
	if !ok {
		return
	}
	switch entityType {
	case "minecraft:camel":
		setProjectedFlag(metadata, gtprotocol.EntityDataFlagSitting, pose == 10)
	case "minecraft:frog":
		setProjectedFlag(metadata, gtprotocol.EntityDataFlagJumpGoal, pose == 6)
		setProjectedFlag(metadata, gtprotocol.EntityDataFlagCroaking, pose == 8)
		setProjectedFlag(metadata, gtprotocol.EntityDataFlagDigestMob, pose == 9)
	case "minecraft:sniffer":
		// Sniffer state, rather than pose, owns the Bedrock animation flags.
	case "minecraft:warden":
		setProjectedFlag(metadata, gtprotocol.EntityDataFlagDigging, pose == 14)
		setProjectedFlag(metadata, gtprotocol.EntityDataFlagEmerging, pose == 13)
		setProjectedFlag(metadata, gtprotocol.EntityDataFlagRoaring, pose == 11)
		setProjectedFlag(metadata, gtprotocol.EntityDataFlagSniffing, pose == 12)
	}
}

func translateHorseFlags(value any, metadata gtprotocol.EntityMetadata) {
	flags, ok := value.(int8)
	if !ok {
		return
	}
	bits := byte(flags)
	setProjectedFlag(metadata, gtprotocol.EntityDataFlagTamed, bits&0x02 != 0)
	setProjectedFlag(metadata, gtprotocol.EntityDataFlagEating, bits&0x10 != 0)
	setProjectedFlag(metadata, gtprotocol.EntityDataFlagStanding, bits&0x20 != 0)
	if bits&0x02 != 0 {
		metadata[gtprotocol.EntityDataKeyContainerType] = byte(gtprotocol.ContainerTypeHorse)
	} else {
		metadata[gtprotocol.EntityDataKeyContainerType] = byte(0)
	}
}

func translateCamelHorseFlags(value any, metadata gtprotocol.EntityMetadata) {
	flags, ok := value.(int8)
	if !ok {
		return
	}
	bits := byte(flags)
	// Java CamelEntity is always tame. Its horse flag byte does not carry the
	// tame or saddle bits; saddle state arrives through the entity's equipment.
	setProjectedFlag(metadata, gtprotocol.EntityDataFlagTamed, true)
	setProjectedFlag(metadata, gtprotocol.EntityDataFlagEating, bits&0x10 != 0)
	setProjectedFlag(metadata, gtprotocol.EntityDataFlagStanding, bits&0x20 != 0)
	metadata[gtprotocol.EntityDataKeyContainerType] = byte(gtprotocol.ContainerTypeHorse)
}

func translateSnifferState(state int32, metadata gtprotocol.EntityMetadata) {
	setProjectedFlag(metadata, gtprotocol.EntityDataFlagFeelingHappy, state == 1)
	setProjectedFlag(metadata, gtprotocol.EntityDataFlagScenting, state == 3)
	setProjectedFlag(metadata, gtprotocol.EntityDataFlagSearching, state == 4)
	setProjectedFlag(metadata, gtprotocol.EntityDataFlagDigging, state == 5)
	setProjectedFlag(metadata, gtprotocol.EntityDataFlagRising, state == 6)
}
