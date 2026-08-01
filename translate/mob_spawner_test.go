package translate

import (
	"bytes"
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/nbt"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
	gtpacket "github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestBedrockMobSpawnerProjectsDisplayMetadataForChunkAndStandalone(t *testing.T) {
	payload := map[string]any{
		"Delay": int16(20),
		"SpawnData": map[string]any{
			"entity": map[string]any{"id": "minecraft:zombie"},
		},
	}
	standalone, ok := BedrockBlockEntityTag(9, 37, 70, -34, payload)
	if !ok {
		t.Fatal("mob spawner block entity did not translate")
	}
	if standalone["EntityIdentifier"] != "minecraft:zombie" || standalone["isMovable"] != byte(1) {
		t.Fatalf("spawner identity/state = %#v", standalone)
	}
	if standalone["DisplayEntityWidth"] != float32(0.6) || standalone["DisplayEntityHeight"] != float32(1.8) || standalone["DisplayEntityScale"] != float32(1) {
		t.Fatalf("spawner display metadata = %#v", standalone)
	}
	if standalone["Delay"] != int16(20) {
		t.Fatalf("spawner timing changed = %#v", standalone["Delay"])
	}
	if _, exists := standalone["SpawnData"]; exists {
		t.Fatalf("Java spawn data was not removed: %#v", standalone)
	}

	position, chunk, ok := BedrockBlockEntityForChunk(2, -3, JavaBlockEntity{
		X: 5, Y: 70, Z: 14, Type: 9, Data: payload,
	})
	if !ok || position != (gtprotocol.BlockPos{37, 70, -34}) {
		t.Fatalf("chunk spawner position = %v, %v", position, ok)
	}
	for _, key := range []string{"EntityIdentifier", "DisplayEntityWidth", "DisplayEntityHeight", "DisplayEntityScale", "isMovable"} {
		if chunk[key] != standalone[key] {
			t.Fatalf("chunk/standalone %s = %#v/%#v", key, chunk[key], standalone[key])
		}
	}
}

func TestBedrockMobSpawnerProjectsInheritedDisplayDimensions(t *testing.T) {
	tests := []struct {
		identifier string
		width      float32
		height     float32
	}{
		{identifier: "minecraft:allay", width: 0.35, height: 0.6},
		{identifier: "minecraft:chest_minecart", width: 0.98, height: 0.7},
		{identifier: "minecraft:zombie_villager_v2", width: 0.6, height: 1.8},
	}
	for _, test := range tests {
		t.Run(test.identifier, func(t *testing.T) {
			tag, ok := BedrockBlockEntityTag(9, 0, 64, 0, map[string]any{
				"SpawnData": map[string]any{"entity": map[string]any{"id": test.identifier}},
			})
			if !ok {
				t.Fatal("mob spawner block entity did not translate")
			}
			if tag["DisplayEntityWidth"] != test.width || tag["DisplayEntityHeight"] != test.height || tag["DisplayEntityScale"] != float32(1) {
				t.Fatalf("inherited display metadata = %#v", tag)
			}
		})
	}
}

func TestBedrockMobSpawnerEmptyPayloadNeedsReset(t *testing.T) {
	tests := []struct {
		name    string
		payload map[string]any
		reset   bool
	}{
		{name: "absent", payload: nil, reset: false},
		{name: "empty", payload: map[string]any{}, reset: false},
		{name: "empty spawn data", payload: map[string]any{"SpawnData": map[string]any{}}, reset: true},
		{name: "empty entity", payload: map[string]any{"SpawnData": map[string]any{"entity": map[string]any{}}}, reset: true},
		{name: "lowercase empty entity", payload: map[string]any{"spawn_data": map[string]any{"entity": map[string]any{}}}, reset: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := javaMobSpawnerPayloadNeedsReset(test.payload); got != test.reset {
				t.Fatalf("reset = %t, want %t", got, test.reset)
			}
			tag, ok := BedrockBlockEntityTag(9, 0, 64, 0, test.payload)
			if !ok || tag["isMovable"] != byte(1) {
				t.Fatalf("empty spawner tag = %#v, ok=%t", tag, ok)
			}
			if _, exists := tag["DisplayEntityWidth"]; exists {
				t.Fatalf("empty spawner emitted display width: %#v", tag)
			}
		})
	}
}

func TestBedrockMobSpawnerMalformedPayloadStaysLenient(t *testing.T) {
	for _, payload := range []map[string]any{
		{"SpawnData": "not a compound"},
		{"SpawnData": map[string]any{"entity": "not a compound"}},
		{"SpawnData": map[string]any{"entity": map[string]any{"id": int32(7)}}},
		{"SpawnData": map[string]any{"entity": map[string]any{"id": "minecraft:not_a_vanilla_entity"}}},
	} {
		tag, ok := BedrockBlockEntityTag(9, 0, 64, 0, payload)
		if !ok || tag["isMovable"] != byte(1) {
			t.Fatalf("malformed spawner tag = %#v, ok=%t", tag, ok)
		}
		if _, exists := tag["DisplayEntityWidth"]; exists {
			t.Fatalf("malformed spawner emitted display width: %#v", tag)
		}
	}
}

func TestBedrockMobSpawnerNetworkNBTMarshalsDisplayMetadata(t *testing.T) {
	position := gtprotocol.BlockPos{37, 70, -34}
	tag, ok := BedrockBlockEntityTag(9, position[0], position[1], position[2], map[string]any{
		"SpawnData": map[string]any{"entity": map[string]any{"id": "minecraft:zombie_villager"}},
	})
	if !ok {
		t.Fatal("mob spawner block entity did not translate")
	}

	blockActor := &gtpacket.BlockActorData{Position: position, NBTData: tag}
	var wire bytes.Buffer
	if err := (&gtpacket.Header{PacketID: blockActor.ID()}).Write(&wire); err != nil {
		t.Fatal(err)
	}
	blockActor.Marshal(gtprotocol.NewWriter(&wire, 0))
	var header gtpacket.Header
	if err := header.Read(&wire); err != nil {
		t.Fatal(err)
	}
	if header.PacketID != blockActor.ID() {
		t.Fatalf("spawner actor packet ID = %d, want %d", header.PacketID, blockActor.ID())
	}
	var decodedPosition gtprotocol.BlockPos
	var decodedTag map[string]any
	reader := gtprotocol.NewReader(&wire, 0, false)
	reader.BlockPos(&decodedPosition)
	reader.NBT(&decodedTag, nbt.NetworkLittleEndian)
	if decodedPosition != position {
		t.Fatalf("spawner actor position = %v, want %v", decodedPosition, position)
	}
	if decodedTag["isMovable"] != byte(1) || decodedTag["DisplayEntityWidth"] != float32(0.6) || decodedTag["DisplayEntityHeight"] != float32(1.8) || decodedTag["DisplayEntityScale"] != float32(1) {
		t.Fatalf("spawner actor wire fields = %#v", decodedTag)
	}
}

func TestJavaMobSpawnerResetUpdatesReplaceBlock(t *testing.T) {
	updates, ok := javaMobSpawnerResetUpdates(gtprotocol.BlockPos{1, 64, -2})
	if !ok || len(updates) != 2 {
		t.Fatalf("spawner reset updates = %#v, ok=%t", updates, ok)
	}
	if updates[0].Position != updates[1].Position || updates[0].NewBlockRuntimeID == updates[1].NewBlockRuntimeID {
		t.Fatalf("spawner reset update order = %#v", updates)
	}
	if updates[0].Flags != javaMobSpawnerUpdateFlags || updates[1].Flags != javaMobSpawnerUpdateFlags {
		t.Fatalf("spawner reset flags = %#v", updates)
	}
}

func TestJavaMobSpawnerChunkResetPositionsUseWorldCoordinates(t *testing.T) {
	chunk := JavaChunk{
		X: 2,
		Z: -3,
		BlockEntities: []JavaBlockEntity{
			{X: 5, Y: 70, Z: 14, Type: 9, Data: map[string]any{"SpawnData": map[string]any{}}},
			{X: 6, Y: 70, Z: 14, Type: 9, Data: map[string]any{"SpawnData": map[string]any{"entity": map[string]any{"id": "minecraft:zombie"}}}},
			{X: 7, Y: 70, Z: 14, Type: 1, Data: map[string]any{"SpawnData": map[string]any{}}},
		},
	}
	positions := javaMobSpawnerChunkResetPositions(chunk)
	if len(positions) != 1 || positions[0] != (gtprotocol.BlockPos{37, 70, -34}) {
		t.Fatalf("chunk reset positions = %#v", positions)
	}
}
