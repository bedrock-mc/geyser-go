package translate

import (
	"testing"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestDecodeJavaSetPassengers(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.VarInt(42)
	_ = w.VarInt(2)
	_ = w.VarInt(7)
	_ = w.VarInt(8)
	update, err := DecodeJavaSetPassengers(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if update.EntityID != 42 || len(update.PassengerIDs) != 2 || update.PassengerIDs[0] != 7 || update.PassengerIDs[1] != 8 {
		t.Fatalf("passengers = %+v", update)
	}
}

func TestPassengerLinkType(t *testing.T) {
	if passengerLinkType(0) != gtprotocol.EntityLinkRider {
		t.Fatal("first passenger is not a controlling rider")
	}
	if passengerLinkType(1) != gtprotocol.EntityLinkPassenger {
		t.Fatal("additional passenger is not an ordinary rider")
	}
}
