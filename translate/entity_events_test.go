package translate

import (
	"testing"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestDecodeJavaEntityEvent(t *testing.T) {
	w := javaprotocol.NewWriter()
	if err := w.Int32(42); err != nil {
		t.Fatal(err)
	}
	if err := w.Byte(javaEntityEventLivingDeath); err != nil {
		t.Fatal(err)
	}
	event, err := DecodeJavaEntityEvent(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if event.EntityID != 42 || event.EventID != javaEntityEventLivingDeath {
		t.Fatalf("event = %+v", event)
	}
	if javaEntityEventActorTypes[event.EventID] != packet.ActorEventDeath {
		t.Fatalf("death event mapping = %d", javaEntityEventActorTypes[event.EventID])
	}
}

func TestDecodeJavaTakeItem(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.VarInt(7)
	_ = w.VarInt(8)
	_ = w.VarInt(3)
	take, err := DecodeJavaTakeItem(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if take.CollectedEntityID != 7 || take.CollectorEntityID != 8 || take.ItemCount != 3 {
		t.Fatalf("take = %+v", take)
	}
}
