package translate

import (
	"bytes"
	"math"
	"testing"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
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

func TestBedrockVaultResolvesSessionActorIDsInStandaloneAndChunkNBT(t *testing.T) {
	localWords := [4]int32{0x01020304, 0x05060708, 0x090a0b0c, 0x0d0e0f10}
	visibleWords := [4]int32{0x11121314, 0x15161718, 0x191a1b1c, 0x1d1e1f20}
	unknownWords := [4]int32{0x21222324, 0x25262728, 0x292a2b2c, 0x2d2e2f30}
	localUUID, ok := javaVaultUUID(localWords)
	if !ok || localUUID != ([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}) {
		t.Fatalf("decoded local vault UUID = %x", localUUID)
	}
	visibleUUID, ok := javaVaultUUID(visibleWords)
	if !ok {
		t.Fatal("visible UUID did not decode")
	}

	b := NewBasic(javaprotocol.Java1214, nil)
	b.playerUUID = localUUID
	b.gameData.EntityRuntimeID = 501
	b.entities[17] = &javaEntityState{entityUUID: visibleUUID, runtimeID: 702}
	actorRuntimeIDs := b.snapshotJavaVaultActorRuntimeIDs()

	payload := map[string]any{
		"shared_data": map[string]any{
			"display_item":              map[string]any{"id": "minecraft:diamond", "count": int32(2)},
			"connected_players":         []any{localWords, visibleWords, unknownWords, [3]int32{1, 2, 3}, []int32{1, 2, 3, 4}, int64(999), "malformed"},
			"connected_particles_range": float64(7.25),
		},
	}
	standalone, ok := bedrockBlockEntityTagWithResolver(44, 37, 70, -34, payload, actorRuntimeIDs)
	if !ok {
		t.Fatal("resolver-aware standalone vault did not translate")
	}
	players, ok := standalone["connected_players"].([]int64)
	if !ok || len(players) != 2 || players[0] != 501 || players[1] != 702 {
		t.Fatalf("resolver-aware standalone players = %#v", standalone["connected_players"])
	}
	item, ok := standalone["display_item"].(map[string]any)
	if !ok || item["Name"] != "minecraft:diamond" || item["Count"] != byte(2) || standalone["connected_particle_range"] != float32(7.25) {
		t.Fatalf("resolver-aware standalone projection = %#v", standalone)
	}

	wirePayload := map[string]any{
		"shared_data": map[string]any{
			"display_item":              map[string]any{"id": "minecraft:diamond", "count": int32(2)},
			"connected_players":         []any{localWords, visibleWords, unknownWords},
			"connected_particles_range": float64(7.25),
		},
	}
	wireStandalone, ok := bedrockBlockEntityTagWithResolver(44, 37, 70, -34, wirePayload, actorRuntimeIDs)
	if !ok {
		t.Fatal("resolver-aware wire vault did not translate")
	}
	position := gtprotocol.BlockPos{37, 70, -34}
	blockActor := &gtpacket.BlockActorData{Position: position, NBTData: wireStandalone}
	var wire bytes.Buffer
	if err := (&gtpacket.Header{PacketID: blockActor.ID()}).Write(&wire); err != nil {
		t.Fatal(err)
	}
	blockActor.Marshal(gtprotocol.NewWriter(&wire, 0))
	var header gtpacket.Header
	if err := header.Read(&wire); err != nil {
		t.Fatal(err)
	}
	reader := gtprotocol.NewReader(&wire, 0, false)
	var decodedPosition gtprotocol.BlockPos
	var decodedTag map[string]any
	reader.BlockPos(&decodedPosition)
	reader.NBT(&decodedTag, nbt.NetworkLittleEndian)
	decodedPlayers, ok := decodedTag["connected_players"].([]int64)
	if !ok || decodedPosition != position || len(decodedPlayers) != 2 || decodedPlayers[0] != 501 || decodedPlayers[1] != 702 {
		t.Fatalf("standalone vault actor NBT position=%v players=%#v", decodedPosition, decodedTag["connected_players"])
	}

	chunk := JavaChunk{
		X: 2, Z: -3,
		BlockEntities: []JavaBlockEntity{{
			X: 5, Y: 70, Z: 14, Type: 44, Data: wirePayload,
		}},
	}
	base, _, err := EncodeBedrockChunk(JavaChunk{X: chunk.X, Z: chunk.Z}, 0)
	if err != nil {
		t.Fatal(err)
	}
	sectionCount, minSection := bedrockDimensionSections(0)
	withVault, _, err := encodeBedrockChunkWithResolver(chunk, 0, sectionCount, minSection, nil, actorRuntimeIDs)
	if err != nil {
		t.Fatal(err)
	}
	var chunkTag map[string]any
	if err := nbt.UnmarshalEncoding(withVault[len(base):], &chunkTag, nbt.NetworkLittleEndian); err != nil {
		t.Fatal(err)
	}
	chunkPlayers, ok := chunkTag["connected_players"].([]int64)
	if !ok || chunkTag["x"] != int32(37) || chunkTag["z"] != int32(-34) || len(chunkPlayers) != 2 || chunkPlayers[0] != 501 || chunkPlayers[1] != 702 {
		t.Fatalf("chunk vault actor NBT = %#v", chunkTag)
	}
}
