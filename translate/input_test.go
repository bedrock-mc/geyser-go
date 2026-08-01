package translate

import (
	"math"
	"testing"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/go-gl/mathgl/mgl32"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestEncodePlayerAuthInputAsJavaPositionLook(t *testing.T) {
	input := gtprotocol.NewBitset(packet.PlayerAuthInputBitsetSize)
	input.Set(packet.InputFlagVerticalCollision)
	input.Set(packet.InputFlagHorizontalCollision)
	payload, err := encodePlayerAuthInput(&packet.PlayerAuthInput{
		Position:  mgl32.Vec3{3, 65, -2},
		Yaw:       90,
		Pitch:     -10,
		InputData: input,
	})
	if err != nil {
		t.Fatal(err)
	}
	r := javaprotocol.NewReader(payload)
	x, err := r.Float64()
	if err != nil {
		t.Fatal(err)
	}
	y, err := r.Float64()
	if err != nil {
		t.Fatal(err)
	}
	z, err := r.Float64()
	if err != nil {
		t.Fatal(err)
	}
	yaw, err := r.Float32()
	if err != nil {
		t.Fatal(err)
	}
	pitch, err := r.Float32()
	if err != nil {
		t.Fatal(err)
	}
	flags, err := r.Byte()
	if err != nil {
		t.Fatal(err)
	}
	if x != 3 || math.Abs(y-(65-bedrockPlayerEyeOffset)) > 0.00001 || z != -2 || yaw != 90 || pitch != -10 || flags != 3 {
		t.Fatalf("Java position look = x=%v y=%v z=%v yaw=%v pitch=%v flags=%d", x, y, z, yaw, pitch, flags)
	}
	if r.Remaining() != 0 {
		t.Fatalf("Java movement payload has %d trailing bytes", r.Remaining())
	}
}

func TestEncodePlayerAuthInputRejectsNonFiniteMovement(t *testing.T) {
	_, err := encodePlayerAuthInput(&packet.PlayerAuthInput{
		Position: mgl32.Vec3{float32(math.NaN()), 0, 0},
	})
	if err == nil {
		t.Fatal("non-finite PlayerAuthInput unexpectedly encoded")
	}
}
