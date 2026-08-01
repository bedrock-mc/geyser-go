package translate

import (
	"testing"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestDecodeJavaBlockDestruction(t *testing.T) {
	w := javaprotocol.NewWriter()
	if err := w.VarInt(17); err != nil {
		t.Fatal(err)
	}
	if err := w.Int64(encodeJavaPosition([3]int32{-4, 65, 9})); err != nil {
		t.Fatal(err)
	}
	if err := w.VarInt(6); err != nil {
		t.Fatal(err)
	}
	destruction, err := DecodeJavaBlockDestruction(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if destruction.BreakerEntityID != 17 || destruction.Position != [3]int32{-4, 65, 9} || destruction.Stage != 6 {
		t.Fatalf("destruction = %+v", destruction)
	}
}

func TestJavaWorldEventMappings(t *testing.T) {
	for _, effectID := range []int32{1000, 1505, 2000, 2001, 3003, 3019} {
		mapping, ok := javaWorldEvents[effectID]
		if !ok || mapping.kind != javaWorldEventLevel || mapping.eventType == 0 {
			t.Fatalf("effect %d has no level-event mapping: %+v", effectID, mapping)
		}
	}
	if javaWorldEvents[1032].kind != javaWorldEventSound || javaWorldEvents[1032].sound == "" {
		t.Fatalf("portal travel mapping = %+v", javaWorldEvents[1032])
	}
	if javaSmokeEventData(2) != 1 || javaSmokeEventData(5) != 5 {
		t.Fatalf("smoke direction mapping is incorrect")
	}
	if javaWorldEvents[2000].eventType != packet.LevelEventParticlesShoot {
		t.Fatalf("smoke event type = %d", javaWorldEvents[2000].eventType)
	}
}
