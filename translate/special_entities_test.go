package translate

import (
	"testing"

	"github.com/bedrock-mc/geyser-go/data"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestJavaSpawnEntityProjection(t *testing.T) {
	xp, ok := javaSpawnEntityProjection("minecraft:xp_orb", 17)
	if !ok || xp[gtprotocol.EntityDataKeyTradeExperience] != int32(1) {
		t.Fatalf("experience-orb metadata = %#v, ok=%t", xp, ok)
	}

	falling, ok := javaSpawnEntityProjection("minecraft:falling_block", 1)
	if !ok {
		t.Fatal("known falling-block state was rejected")
	}
	want, known := JavaBlockRuntimeID(1)
	if !known || falling[gtprotocol.EntityDataKeyDisplayTileRuntimeID] != int32(want) {
		t.Fatalf("falling-block metadata = %#v, want runtime %d", falling, want)
	}

	unknown, ok := javaSpawnEntityProjection("minecraft:falling_block", data.Java1214BlockStateCount)
	if ok || unknown[gtprotocol.EntityDataKeyDisplayTileRuntimeID] != int32(data.Java1214ToBedrock[0]) {
		t.Fatalf("unknown falling-block projection = %#v, ok=%t", unknown, ok)
	}

	hook, ok := javaSpawnEntityProjection("minecraft:fishing_hook", 42)
	if !ok || hook[gtprotocol.EntityDataKeyOwner] != int64(42) {
		t.Fatalf("fishing-hook metadata = %#v, ok=%t", hook, ok)
	}
	if _, ok := javaSpawnEntityProjection("minecraft:fishing_hook", -1); ok {
		t.Fatal("ownerless fishing hook was projected")
	}
}
