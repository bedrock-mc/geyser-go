package translate

import (
	"testing"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
)

func TestDecodeJoinGame(t *testing.T) {
	w := javaprotocol.NewWriter()
	if err := w.Int32(42); err != nil {
		t.Fatal(err)
	}
	if err := w.Bool(false); err != nil {
		t.Fatal(err)
	}
	if err := w.VarInt(1); err != nil {
		t.Fatal(err)
	}
	if err := w.String("minecraft:overworld"); err != nil {
		t.Fatal(err)
	}
	for _, value := range []int32{20, 10, 10} {
		if err := w.VarInt(value); err != nil {
			t.Fatal(err)
		}
	}
	for _, value := range []bool{false, true, false} {
		if err := w.Bool(value); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.VarInt(0); err != nil {
		t.Fatal(err)
	}
	if err := w.String("minecraft:overworld"); err != nil {
		t.Fatal(err)
	}
	if err := w.Int64(1234); err != nil {
		t.Fatal(err)
	}
	if err := w.Byte(1); err != nil {
		t.Fatal(err)
	}
	if err := w.Byte(0xff); err != nil {
		t.Fatal(err)
	}
	for _, value := range []bool{false, true, false} {
		if err := w.Bool(value); err != nil {
			t.Fatal(err)
		}
	}
	for _, value := range []int32{300, 63} {
		if err := w.VarInt(value); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Bool(true); err != nil {
		t.Fatal(err)
	}

	join, err := DecodeJoinGame(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if join.EntityID != 42 || join.World.Name != "minecraft:overworld" || join.World.GameMode != 1 || join.World.PortalCooldown != 300 || join.World.SeaLevel != 63 {
		t.Fatalf("unexpected join game: %#v", join)
	}
}

func TestDecodePositionAndHealth(t *testing.T) {
	w := javaprotocol.NewWriter()
	if err := w.VarInt(7); err != nil {
		t.Fatal(err)
	}
	for _, value := range []float64{1.5, 2.5, 3.5} {
		if err := w.Float64(value); err != nil {
			t.Fatal(err)
		}
	}
	for _, value := range []float64{0.25, -0.5, 0.75} {
		if err := w.Float64(value); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Float32(90); err != nil {
		t.Fatal(err)
	}
	if err := w.Float32(-10); err != nil {
		t.Fatal(err)
	}
	if err := w.Int32(javaPositionRelativeX | javaPositionRelativeYaw); err != nil {
		t.Fatal(err)
	}
	position, err := DecodePositionUpdate(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if position.TeleportID != 7 || position.X != 1.5 || position.Flags != javaPositionRelativeX|javaPositionRelativeYaw {
		t.Fatalf("unexpected position: %#v", position)
	}

	healthWriter := javaprotocol.NewWriter()
	if err := healthWriter.Float32(18.5); err != nil {
		t.Fatal(err)
	}
	if err := healthWriter.VarInt(20); err != nil {
		t.Fatal(err)
	}
	if err := healthWriter.Float32(4.5); err != nil {
		t.Fatal(err)
	}
	health, err := DecodeHealthUpdate(healthWriter.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if health != 18.5 {
		t.Fatalf("health=%v", health)
	}
}
