package translate

import (
	"math"
	"testing"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/go-gl/mathgl/mgl32"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestDecodeBlockChangePositionAndState(t *testing.T) {
	w := javaprotocol.NewWriter()
	position := gtprotocol.BlockPos{-12345, -17, 54321}
	if err := w.Int64(encodeJavaPosition(position)); err != nil {
		t.Fatal(err)
	}
	if err := w.VarInt(42); err != nil {
		t.Fatal(err)
	}
	change, err := DecodeBlockChange(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if change.Position != position || change.StateID != 42 {
		t.Fatalf("decoded block change = %+v, want position %v state 42", change, position)
	}
}

func TestDecodeSpawnEntity(t *testing.T) {
	w := javaprotocol.NewWriter()
	if err := w.VarInt(7); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 16; i++ {
		if err := w.Byte(byte(i)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.VarInt(47); err != nil { // experience_orb
		t.Fatal(err)
	}
	for _, value := range []float64{1.5, 64, -2.25} {
		if err := w.Float64(value); err != nil {
			t.Fatal(err)
		}
	}
	for _, value := range []int8{-64, 64, 32} {
		if err := w.Byte(byte(value)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.VarInt(3); err != nil {
		t.Fatal(err)
	}
	for _, value := range []int16{8000, -4000, 0} {
		if err := w.Int16(value); err != nil {
			t.Fatal(err)
		}
	}
	spawn, err := DecodeSpawnEntity(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if spawn.EntityID != 7 || spawn.Type != 47 || spawn.ObjectData != 3 {
		t.Fatalf("decoded spawn header = %+v", spawn)
	}
	if got := spawn.Position; got[0] != 1.5 || got[1] != 64 || got[2] != -2.25 {
		t.Fatalf("decoded position = %v", got)
	}
	if math.Abs(float64(spawn.Velocity[0]-1)) > 0.0001 || math.Abs(float64(spawn.Velocity[1]+0.5)) > 0.0001 {
		t.Fatalf("decoded velocity = %v", spawn.Velocity)
	}
	if math.Abs(float64(spawn.Pitch+90)) > 0.0001 || math.Abs(float64(spawn.Yaw-90)) > 0.0001 {
		t.Fatalf("decoded rotation = pitch %f yaw %f", spawn.Pitch, spawn.Yaw)
	}
}

func TestDecodeEntityRelativeMove(t *testing.T) {
	w := javaprotocol.NewWriter()
	for _, value := range []any{int32(11), int16(4096), int16(-2048), int16(0), int8(64), int8(-64), true} {
		switch value := value.(type) {
		case int32:
			if err := w.VarInt(value); err != nil {
				t.Fatal(err)
			}
		case int16:
			if err := w.Int16(value); err != nil {
				t.Fatal(err)
			}
		case int8:
			if err := w.Byte(byte(value)); err != nil {
				t.Fatal(err)
			}
		case bool:
			if err := w.Bool(value); err != nil {
				t.Fatal(err)
			}
		}
	}
	move, err := DecodeEntityRelativeMove(w.Bytes(), true)
	if err != nil {
		t.Fatal(err)
	}
	if move.EntityID != 11 || move.Delta != (mgl32.Vec3{1, -0.5, 0}) || !move.OnGround {
		t.Fatalf("decoded relative move = %+v", move)
	}
}

func TestEmptyBedrockChunkPayload(t *testing.T) {
	payload := EmptyBedrockChunkPayload(0)
	if len(payload) != 26 || payload[0] != 1 || payload[1] != 0 || payload[len(payload)-1] != 0 {
		t.Fatalf("unexpected overworld empty payload length/content: len=%d bytes=%v", len(payload), payload)
	}
	for _, value := range payload[2 : len(payload)-1] {
		if value != 0xff {
			t.Fatalf("unexpected biome carry marker 0x%x", value)
		}
	}
}
