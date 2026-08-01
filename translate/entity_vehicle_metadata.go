package translate

import gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"

const javaBoatBuoyancyData = "{\"apply_gravity\":true,\"base_buoyancy\":1.0,\"big_wave_probability\":0.02999999932944775,\"big_wave_speed\":10.0,\"drag_down_on_buoyancy_removed\":0.0,\"liquid_blocks\":[\"minecraft:water\",\"minecraft:flowing_water\"],\"simulate_waves\":false}"

func javaBoatEntity(entityType string) bool {
	switch entityType {
	case "minecraft:acacia_boat", "minecraft:bamboo_raft", "minecraft:birch_boat",
		"minecraft:cherry_boat", "minecraft:dark_oak_boat", "minecraft:jungle_boat",
		"minecraft:mangrove_boat", "minecraft:oak_boat", "minecraft:pale_oak_boat",
		"minecraft:spruce_boat", "minecraft:acacia_chest_boat", "minecraft:bamboo_chest_raft",
		"minecraft:birch_chest_boat", "minecraft:cherry_chest_boat", "minecraft:dark_oak_chest_boat",
		"minecraft:jungle_chest_boat", "minecraft:mangrove_chest_boat", "minecraft:oak_chest_boat",
		"minecraft:pale_oak_chest_boat", "minecraft:spruce_chest_boat":
		return true
	default:
		return false
	}
}

func javaBoatVariant(entityType string) (int32, bool) {
	switch entityType {
	case "minecraft:oak_boat", "minecraft:oak_chest_boat":
		return 0, true
	case "minecraft:spruce_boat", "minecraft:spruce_chest_boat":
		return 1, true
	case "minecraft:birch_boat", "minecraft:birch_chest_boat":
		return 2, true
	case "minecraft:jungle_boat", "minecraft:jungle_chest_boat":
		return 3, true
	case "minecraft:acacia_boat", "minecraft:acacia_chest_boat":
		return 4, true
	case "minecraft:dark_oak_boat", "minecraft:dark_oak_chest_boat":
		return 5, true
	case "minecraft:mangrove_boat", "minecraft:mangrove_chest_boat":
		return 6, true
	case "minecraft:bamboo_raft", "minecraft:bamboo_chest_raft":
		return 7, true
	case "minecraft:cherry_boat", "minecraft:cherry_chest_boat":
		return 8, true
	case "minecraft:pale_oak_boat", "minecraft:pale_oak_chest_boat":
		return 9, true
	default:
		return 0, false
	}
}

func javaMinecartEntity(entityType string) bool {
	switch entityType {
	case "minecraft:minecart", "minecraft:chest_minecart", "minecraft:command_block_minecart",
		"minecraft:furnace_minecart", "minecraft:hopper_minecart", "minecraft:spawner_minecart",
		"minecraft:tnt_minecart":
		return true
	default:
		return false
	}
}

func translateJavaBoatMetadata(entries []JavaEntityMetadataEntry, metadata gtprotocol.EntityMetadata) {
	for _, entry := range entries {
		switch entry.Index {
		case 8:
			if value, ok := entry.Value.(int32); ok {
				metadata[gtprotocol.EntityDataKeyHurt] = value
			}
		case 9:
			if value, ok := entry.Value.(int32); ok {
				metadata[gtprotocol.EntityDataKeyHurtDirection] = value
			}
		case 10:
			if value, ok := entry.Value.(float32); ok && finiteFloat32(value) {
				metadata[gtprotocol.EntityDataKeyStructuralIntegrity] = int32(clampFloat32(40-value, 0, 40))
			}
		case 11:
			if value, ok := entry.Value.(bool); ok {
				metadata[gtprotocol.EntityDataKeyRowTimeLeft] = boolFloat(value)
			}
		case 12:
			if value, ok := entry.Value.(bool); ok {
				metadata[gtprotocol.EntityDataKeyRowTimeRight] = boolFloat(value)
			}
		case 13:
			if value, ok := entry.Value.(int32); ok {
				metadata[gtprotocol.EntityDataKeyBubbleTime] = value
			}
		}
	}
}

func translateJavaMinecartMetadata(entityType string, entries []JavaEntityMetadataEntry, metadata gtprotocol.EntityMetadata) {
	for _, entry := range entries {
		switch entry.Index {
		case 8:
			if value, ok := entry.Value.(int32); ok {
				metadata[gtprotocol.EntityDataKeyStructuralIntegrity] = value
			}
		case 9:
			if value, ok := entry.Value.(int32); ok {
				metadata[gtprotocol.EntityDataKeyHurtDirection] = value
			}
		case 10:
			if value, ok := entry.Value.(float32); ok && finiteFloat32(value) {
				metadata[gtprotocol.EntityDataKeyHurt] = int32(clampFloat32(value, 0, 15))
			}
		case 11:
			if value, ok := entry.Value.(int32); ok {
				runtimeID := uint32(0)
				if value != 0 {
					var known bool
					runtimeID, known = JavaBlockRuntimeID(value)
					if !known {
						runtimeID = 0
					}
				}
				metadata[gtprotocol.EntityDataKeyDisplayTileRuntimeID] = int32(runtimeID)
				metadata[gtprotocol.EntityDataKeyCustomDisplay] = byte(boolByte(value != 0))
			}
		case 12:
			if value, ok := entry.Value.(int32); ok {
				metadata[gtprotocol.EntityDataKeyDisplayOffset] = value
			}
		case 13:
			if value, ok := entry.Value.(bool); ok {
				metadata[gtprotocol.EntityDataKeyCustomDisplay] = boolByte(value)
			}
		case 14:
			if entityType == "minecraft:command_block_minecart" {
				if value, ok := entry.Value.(string); ok {
					metadata[gtprotocol.EntityDataKeyCommandName] = value
				}
			}
		case 15:
			if entityType == "minecraft:command_block_minecart" {
				metadata[gtprotocol.EntityDataKeyLastCommandOutput] = JavaTextComponentText(entry.Value)
			}
		}
	}
}

func boolFloat(value bool) float32 {
	if value {
		return 0.04
	}
	return 0
}
