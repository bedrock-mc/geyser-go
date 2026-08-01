package translate

import (
	"github.com/bedrock-mc/geyser-go/data"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

const javaMobSpawnerUpdateFlags = packet.BlockUpdateNeighbours | packet.BlockUpdateNetwork

// javaMobSpawnerPayloadNeedsReset mirrors Geyser's empty SpawnData.entity
// workaround. Bedrock ignores an empty EntityIdentifier update, so the block
// must be replaced before its BlockActorData is sent.
func javaMobSpawnerPayloadNeedsReset(payload map[string]any) bool {
	spawnData, ok := javaNBTCompound(payload["SpawnData"])
	if !ok {
		spawnData, ok = javaNBTCompound(payload["spawn_data"])
	}
	if !ok {
		return false
	}

	entity, present := spawnData["entity"]
	if !present {
		return true
	}
	entityCompound, ok := javaNBTCompound(entity)
	return ok && len(entityCompound) == 0
}

func javaMobSpawnerResetUpdates(position gtprotocol.BlockPos) ([]*packet.UpdateBlock, bool) {
	air, airOK := data.BedrockBlockRuntimeID("minecraft:air", nil)
	spawner, spawnerOK := data.BedrockBlockRuntimeID("minecraft:mob_spawner", nil)
	if !airOK || !spawnerOK {
		return nil, false
	}
	return []*packet.UpdateBlock{
		{
			Position:          position,
			NewBlockRuntimeID: air,
			Flags:             javaMobSpawnerUpdateFlags,
			Layer:             0,
		},
		{
			Position:          position,
			NewBlockRuntimeID: spawner,
			Flags:             javaMobSpawnerUpdateFlags,
			Layer:             0,
		},
	}, true
}

// javaMobSpawnerChunkResetPositions returns the world positions whose chunk
// block-entity payload would otherwise contain an empty SpawnData entity. The
// caller sends the reset before the LevelChunk packet, matching Geyser's
// ordering and keeping the subsequent chunk NBT as the authoritative actor
// state.
func javaMobSpawnerChunkResetPositions(chunk JavaChunk) []gtprotocol.BlockPos {
	positions := make([]gtprotocol.BlockPos, 0)
	for _, entity := range chunk.BlockEntities {
		javaName, ok := JavaBlockEntityTypeName(entity.Type)
		if !ok || javaName != "mob_spawner" || !javaMobSpawnerPayloadNeedsReset(entity.Data) {
			continue
		}
		position, ok := chunkBlockEntityPosition(chunk.X, chunk.Z, entity)
		if ok {
			positions = append(positions, position)
		}
	}
	return positions
}

func (b *Basic) resetJavaMobSpawnerBlock(bedrock interface{ WritePacket(packet.Packet) error }, position gtprotocol.BlockPos) error {
	updates, ok := javaMobSpawnerResetUpdates(position)
	if !ok {
		b.logSemanticAnomaly("could not resolve Bedrock mob spawner reset blocks", "position", position)
		return nil
	}
	for _, update := range updates {
		if err := bedrock.WritePacket(update); err != nil {
			return err
		}
	}
	return nil
}
