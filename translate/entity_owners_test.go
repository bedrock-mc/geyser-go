package translate

import (
	"testing"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/go-gl/mathgl/mgl32"
)

func TestJavaTameableOwnerMetadata(t *testing.T) {
	var owner [16]byte
	owner[0] = 7
	got, hasEntry, hasUUID := javaTameableOwnerMetadata("minecraft:cat", []JavaEntityMetadataEntry{{Index: 18, Type: 13, Value: owner}})
	if !hasEntry || !hasUUID || got != owner {
		t.Fatalf("owner metadata = %x, entry=%v, uuid=%v", got, hasEntry, hasUUID)
	}
	_, hasEntry, hasUUID = javaTameableOwnerMetadata("minecraft:wolf", []JavaEntityMetadataEntry{{Index: 18, Type: 13, Value: nil}})
	if !hasEntry || hasUUID {
		t.Fatalf("empty owner metadata entry=%v uuid=%v", hasEntry, hasUUID)
	}
	if _, hasEntry, _ = javaTameableOwnerMetadata("minecraft:fox", []JavaEntityMetadataEntry{{Index: 18, Type: 13, Value: owner}}); hasEntry {
		t.Fatal("non-tameable fox owner was projected")
	}
}

func TestJavaOwnerEntityIDLocked(t *testing.T) {
	b := NewBasic(javaprotocol.Java1214, nil)
	var owner [16]byte
	owner[0] = 3
	b.playerUUID = owner
	b.gameData.EntityRuntimeID = 42
	if got := b.javaOwnerEntityIDLocked(owner); got != 42 {
		t.Fatalf("local owner ID = %d", got)
	}
	var other [16]byte
	other[0] = 4
	b.entities[9] = &javaEntityState{runtimeID: 99, entityUUID: other, position: mgl32.Vec3{}}
	if got := b.javaOwnerEntityIDLocked(other); got != 99 {
		t.Fatalf("entity owner ID = %d", got)
	}
	var missing [16]byte
	missing[0] = 5
	if got := b.javaOwnerEntityIDLocked(missing); got != javaUnknownOwnerEntityID {
		t.Fatalf("unknown owner ID = %d", got)
	}
}
