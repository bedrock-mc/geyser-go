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
