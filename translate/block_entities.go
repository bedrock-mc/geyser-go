package translate

import (
	"fmt"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
)

// java1214BlockEntityTypeNames is the Java 1.21.4 BLOCK_ENTITY_TYPE registry
// from MCProtocolLib 1.21.4-1. The numeric value in a chunk block-entity
// record is an index into this registry, not a namespaced string.
var java1214BlockEntityTypeNames = [...]string{
	"furnace",
	"chest",
	"trapped_chest",
	"ender_chest",
	"jukebox",
	"dispenser",
	"dropper",
	"sign",
	"hanging_sign",
	"mob_spawner",
	"creaking_heart",
	"piston",
	"brewing_stand",
	"enchanting_table",
	"end_portal",
	"beacon",
	"skull",
	"daylight_detector",
	"hopper",
	"comparator",
	"banner",
	"structure_block",
	"end_gateway",
	"command_block",
	"shulker_box",
	"bed",
	"conduit",
	"barrel",
	"smoker",
	"blast_furnace",
	"lectern",
	"bell",
	"jigsaw",
	"campfire",
	"beehive",
	"sculk_sensor",
	"calibrated_sculk_sensor",
	"sculk_catalyst",
	"sculk_shrieker",
	"chiseled_bookshelf",
	"brushable_block",
	"decorated_pot",
	"crafter",
	"trial_spawner",
	"vault",
}

const Java1214BlockEntityTypeCount = int32(len(java1214BlockEntityTypeNames))

// bedrockBlockEntityIDOverrides are the irregular names documented by
// Geyser's BlockEntityUtils. All other names use the Java identifier with
// underscores removed and each word title-cased.
var bedrockBlockEntityIDOverrides = map[string]string{
	"enchanting_table": "EnchantTable",
	"jigsaw":           "JigsawBlock",
	"piston":           "PistonArm",
	"trapped_chest":    "Chest",
}

// JavaBlockEntityTypeName resolves a Java 1.21.4 block-entity registry ID.
// Unknown IDs are semantically odd but well-formed and are skipped by the
// caller rather than terminating the session.
func JavaBlockEntityTypeName(typeID int32) (string, bool) {
	if typeID < 0 || int64(typeID) >= int64(len(java1214BlockEntityTypeNames)) {
		return "", false
	}
	return java1214BlockEntityTypeNames[typeID], true
}

func bedrockBlockEntityID(javaName string) string {
	if override, ok := bedrockBlockEntityIDOverrides[javaName]; ok {
		return override
	}
	result := make([]byte, 0, len(javaName))
	upper := true
	for i := 0; i < len(javaName); i++ {
		if javaName[i] == '_' {
			upper = true
			continue
		}
		value := javaName[i]
		if upper && value >= 'a' && value <= 'z' {
			value -= 'a' - 'A'
		}
		result = append(result, value)
		upper = false
	}
	return string(result)
}

// BedrockBlockEntityTag creates the generic Bedrock tile-entity compound.
// Type-specific Geyser translators will eventually enrich this conversion;
// preserving the Java fields here already gives Bedrock a valid identity and
// position for the common vanilla records.
func BedrockBlockEntityTag(typeID int32, x, y, z int32, data map[string]any) (map[string]any, bool) {
	javaName, ok := JavaBlockEntityTypeName(typeID)
	if !ok {
		return nil, false
	}
	tag := make(map[string]any, len(data)+4)
	for key, value := range data {
		tag[key] = value
	}
	tag["x"] = x
	tag["y"] = y
	tag["z"] = z
	tag["id"] = bedrockBlockEntityID(javaName)
	projectJavaBlockEntityPayload(javaName, tag)
	return tag, true
}

func chunkBlockEntityPosition(chunkX, chunkZ int32, entity JavaBlockEntity) (gtprotocol.BlockPos, bool) {
	x := int64(chunkX)*16 + int64(entity.X)
	y := int64(entity.Y)
	z := int64(chunkZ)*16 + int64(entity.Z)
	if x < -1<<31 || x > 1<<31-1 || y < -1<<31 || y > 1<<31-1 || z < -1<<31 || z > 1<<31-1 {
		return gtprotocol.BlockPos{}, false
	}
	return gtprotocol.BlockPos{int32(x), int32(y), int32(z)}, true
}

// BedrockBlockEntityForChunk translates a chunk-local Java block entity into
// the world position and NBT compound required by a Bedrock LevelChunk or
// BlockActorData packet.
func BedrockBlockEntityForChunk(chunkX, chunkZ int32, entity JavaBlockEntity) (gtprotocol.BlockPos, map[string]any, bool) {
	position, ok := chunkBlockEntityPosition(chunkX, chunkZ, entity)
	if !ok {
		return gtprotocol.BlockPos{}, nil, false
	}
	tag, ok := BedrockBlockEntityTag(entity.Type, position[0], position[1], position[2], entity.Data)
	return position, tag, ok
}

type JavaBlockEntityUpdate struct {
	Position gtprotocol.BlockPos
	Type     int32
	Data     map[string]any
}

// DecodeBlockEntityUpdate decodes Java's ClientboundBlockEntityDataPacket.
// The packet calls the registry index "action" in the protocol data, but it
// is the same BLOCK_ENTITY_TYPE value used by chunk records.
func DecodeBlockEntityUpdate(payload []byte) (JavaBlockEntityUpdate, error) {
	r := javaprotocol.NewReader(payload)
	packed, err := r.Int64()
	if err != nil {
		return JavaBlockEntityUpdate{}, fmt.Errorf("translate: block entity position: %w", err)
	}
	typeID, err := r.VarInt()
	if err != nil {
		return JavaBlockEntityUpdate{}, fmt.Errorf("translate: block entity type: %w", err)
	}
	data, err := decodeOptionalNBT(r)
	if err != nil {
		return JavaBlockEntityUpdate{}, fmt.Errorf("translate: block entity NBT: %w", err)
	}
	if r.Remaining() != 0 {
		return JavaBlockEntityUpdate{}, fmt.Errorf("translate: block entity update has %d trailing bytes", r.Remaining())
	}
	return JavaBlockEntityUpdate{Position: decodeJavaPosition(packed), Type: typeID, Data: data}, nil
}
