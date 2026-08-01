package translate

import gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"

// javaEntityRuntimeIDLocked resolves Java's entity IDs to the Bedrock runtime
// IDs used by actor metadata. The caller holds b.mu; an unknown but valid
// target is represented by Bedrock's zero target rather than disconnecting.
func (b *Basic) javaEntityRuntimeIDLocked(entityID int32) int64 {
	if entityID < 0 {
		return 0
	}
	entity := b.entities[entityID]
	if entity == nil {
		return 0
	}
	return int64(entity.runtimeID)
}

func (b *Basic) translateEntityTargetMetadataLocked(entityType string, entries []JavaEntityMetadataEntry, metadata gtprotocol.EntityMetadata) {
	for _, entry := range entries {
		switch {
		case javaFishingHookEntity(entityType) && entry.Index == 8:
			// Java stores the hooked entity as entity-id + 1. Bedrock needs
			// the actor runtime ID, and zero is the safe clear value when the
			// entity has already despawned or has not reached this session.
			if target, ok := javaIntegerValue(entry.Value); ok {
				if target > 0 {
					metadata[gtprotocol.EntityDataKeyTarget] = b.javaEntityRuntimeIDLocked(int32(target - 1))
				} else {
					metadata[gtprotocol.EntityDataKeyTarget] = int64(0)
				}
			}
		case entityType == "minecraft:guardian" && entry.Index == 17:
			if target, ok := javaIntegerValue(entry.Value); ok {
				metadata[gtprotocol.EntityDataKeyTarget] = b.javaEntityRuntimeIDLocked(int32(target))
			}
		case entityType == "minecraft:frog" && entry.Index == 18:
			if target, ok := entry.Value.(int32); ok {
				metadata[gtprotocol.EntityDataKeyTarget] = b.javaEntityRuntimeIDLocked(target)
			} else if entry.Value == nil {
				metadata[gtprotocol.EntityDataKeyTarget] = int64(0)
			}
		case entityType == "minecraft:wither":
			if target, ok := javaIntegerValue(entry.Value); ok {
				key, targetOK := map[byte]uint32{
					16: gtprotocol.EntityDataKeyTargetA,
					17: gtprotocol.EntityDataKeyTargetB,
					18: gtprotocol.EntityDataKeyTargetC,
				}[entry.Index]
				if targetOK {
					metadata[key] = b.javaEntityRuntimeIDLocked(int32(target))
				}
			}
		case entityType == "minecraft:vex" && entry.Index == 16:
			if flags, ok := entry.Value.(int8); ok {
				if byte(flags)&0x01 != 0 {
					metadata[gtprotocol.EntityDataKeyTarget] = int64(b.gameData.EntityRuntimeID)
				} else {
					metadata[gtprotocol.EntityDataKeyTarget] = int64(0)
				}
			}
		}
	}
}

func (b *Basic) translateGoatHornMetadataLocked(entityType string, entries []JavaEntityMetadataEntry, entity *javaEntityState, metadata gtprotocol.EntityMetadata) {
	if entityType != "minecraft:goat" {
		return
	}
	changed := false
	for _, entry := range entries {
		switch entry.Index {
		case 18:
			if left, ok := entry.Value.(bool); ok {
				entity.goatLeftHorn = left
				entity.goatLeftHornKnown = true
				changed = true
			}
		case 19:
			if right, ok := entry.Value.(bool); ok {
				entity.goatRightHorn = right
				entity.goatRightHornKnown = true
				changed = true
			}
		}
	}
	if !changed {
		return
	}
	count := 0
	if entity.goatLeftHornKnown && entity.goatLeftHorn {
		count++
	}
	if entity.goatRightHornKnown && entity.goatRightHorn {
		count++
	}
	metadata[gtprotocol.EntityDataKeyGoatHornCount] = int32(count)
}
