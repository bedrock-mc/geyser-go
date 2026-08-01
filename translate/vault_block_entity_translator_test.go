package translate

import (
	"bytes"
	"math"
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/nbt"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
	gtpacket "github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestBedrockVaultProjectsJavaSharedData(t *testing.T) {
	tag, ok := BedrockBlockEntityTag(44, 12, 64, -3, map[string]any{
		"Custom": int32(7),
		"shared_data": map[string]any{
			"display_item": map[string]any{
				"id":    "minecraft:chain",
				"count": int32(2),
				"components": map[string]any{
					"minecraft:custom_data": map[string]any{"geyser_test": int32(1)},
					"minecraft:custom_name": `{"text":"Vault reward"}`,
				},
			},
			"connected_players":         []any{int64(101), [4]int32{1, 2, 3, 4}},
			"connected_particles_range": float64(6.25),
			"future_field":              int32(9),
		},
		"server_data": map[string]any{"future_server_field": int32(4)},
	})
	if !ok {
		t.Fatal("vault block entity did not translate")
	}
	if tag["id"] != "Vault" || tag["x"] != int32(12) || tag["y"] != int32(64) || tag["z"] != int32(-3) {
		t.Fatalf("vault identity = %#v", tag)
	}
	if tag["Custom"] != int32(7) || tag["server_data"].(map[string]any)["future_server_field"] != int32(4) {
		t.Fatalf("vault unknown fields were not preserved: %#v", tag)
	}

	item, ok := tag["display_item"].(map[string]any)
	if !ok || item["Name"] != "minecraft:iron_chain" || item["Count"] != byte(2) || item["Damage"] != int16(0) {
		t.Fatalf("vault display item = %#v", tag["display_item"])
	}
	itemTag, ok := item["tag"].(map[string]any)
	if !ok || itemTag["geyser_test"] != int32(1) {
		t.Fatalf("vault display item custom data = %#v", item["tag"])
	}
	display, ok := itemTag["display"].(map[string]any)
	if !ok || display["Name"] != "Vault reward" {
		t.Fatalf("vault display item name = %#v", itemTag["display"])
	}

	players, ok := tag["connected_players"].([]int64)
	if !ok || len(players) != 1 || players[0] != 101 {
		t.Fatalf("vault connected players = %#v", tag["connected_players"])
	}
	if tag["connected_particle_range"] != float32(6.25) {
		t.Fatalf("vault particle range = %#v", tag["connected_particle_range"])
	}
	sharedData := tag["shared_data"].(map[string]any)
	if sharedData["future_field"] != int32(9) {
		t.Fatalf("vault shared data was not preserved: %#v", sharedData)
	}
}

func TestBedrockVaultUsesDefaultsForAbsentFields(t *testing.T) {
	for _, payload := range []map[string]any{
		nil,
		{},
		{"shared_data": map[string]any{}},
	} {
		tag, ok := BedrockBlockEntityTag(44, 0, 64, 0, payload)
		if !ok {
			t.Fatal("empty vault block entity did not translate")
		}
		item, itemOK := tag["display_item"].(map[string]any)
		if !itemOK || item["Name"] != "" || item["Count"] != byte(0) || item["Damage"] != int16(0) || item["WasPickedUp"] != byte(0) {
			t.Fatalf("empty vault display item = %#v", tag["display_item"])
		}
		players, playersOK := tag["connected_players"].([]int64)
		if !playersOK || len(players) != 0 {
			t.Fatalf("empty vault connected players = %#v", tag["connected_players"])
		}
		if tag["connected_particle_range"] != javaVaultDefaultParticleRange {
			t.Fatalf("empty vault particle range = %#v", tag["connected_particle_range"])
		}
	}
}

func TestBedrockVaultSkipsMalformedSemanticFields(t *testing.T) {
	tag, ok := BedrockBlockEntityTag(44, 0, 64, 0, map[string]any{
		"shared_data": map[string]any{
			"display_item":              "not a compound",
			"connected_players":         []any{"not an entity ID", [3]int32{1, 2, 3}, int32(202)},
			"connected_particles_range": math.NaN(),
		},
	})
	if !ok {
		t.Fatal("malformed vault block entity did not translate")
	}
	item, itemOK := tag["display_item"].(map[string]any)
	if !itemOK || item["Name"] != "" || item["Count"] != byte(0) {
		t.Fatalf("malformed vault display item = %#v", tag["display_item"])
	}
	players, playersOK := tag["connected_players"].([]int64)
	if !playersOK || len(players) != 1 || players[0] != 202 {
		t.Fatalf("malformed vault connected players = %#v", tag["connected_players"])
	}
	if tag["connected_particle_range"] != javaVaultDefaultParticleRange {
		t.Fatalf("malformed vault particle range = %#v", tag["connected_particle_range"])
	}

	negative, ok := BedrockBlockEntityTag(44, 0, 64, 0, map[string]any{
		"shared_data": map[string]any{"connected_particles_range": float64(-1)},
	})
	if !ok || negative["connected_particle_range"] != float32(0) {
		t.Fatalf("negative vault particle range = %#v", negative["connected_particle_range"])
	}
}

func TestBedrockVaultUsesSharedChunkAndActorProjection(t *testing.T) {
	payload := map[string]any{
		"shared_data": map[string]any{
			"display_item":              map[string]any{"id": "minecraft:diamond", "count": int32(1)},
			"connected_particles_range": float64(4.5),
		},
	}
	standalone, ok := BedrockBlockEntityTag(44, 37, 70, -34, payload)
	if !ok {
		t.Fatal("standalone vault block entity did not translate")
	}
	position, chunk, ok := BedrockBlockEntityForChunk(2, -3, JavaBlockEntity{
		X: 5, Y: 70, Z: 14, Type: 44, Data: payload,
	})
	if !ok || position != (gtprotocol.BlockPos{37, 70, -34}) {
		t.Fatalf("chunk vault position = %v, %v", position, ok)
	}
	if chunk["id"] != standalone["id"] || chunk["display_item"].(map[string]any)["Name"] != "minecraft:diamond" {
		t.Fatalf("chunk vault projection = %#v", chunk)
	}

	blockActor := &gtpacket.BlockActorData{Position: position, NBTData: standalone}
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
		t.Fatalf("vault actor packet ID = %d, want %d", header.PacketID, blockActor.ID())
	}
	var decodedPosition gtprotocol.BlockPos
	var decodedTag map[string]any
	reader := gtprotocol.NewReader(&wire, 0, false)
	reader.BlockPos(&decodedPosition)
	reader.NBT(&decodedTag, nbt.NetworkLittleEndian)
	if decodedPosition != position {
		t.Fatalf("vault actor position = %v, want %v", decodedPosition, position)
	}
	decodedItem, ok := decodedTag["display_item"].(map[string]any)
	if !ok || decodedItem["Count"] != byte(1) || decodedItem["Damage"] != int16(0) || decodedTag["connected_particle_range"] != float32(4.5) {
		t.Fatalf("vault actor wire fields = %#v", decodedTag)
	}
	if _, ok := decodedTag["connected_players"].([]int64); !ok {
		t.Fatalf("vault actor connected players type = %T", decodedTag["connected_players"])
	}
}
