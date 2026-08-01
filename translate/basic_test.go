package translate

import (
	"testing"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
)

func TestFloorBlockPositionUsesJavaBlockCoordinates(t *testing.T) {
	position := floorBlockPosition(javaPosition{x: 9.5, y: 76, z: -263.5})
	if position[0] != 9 || position[1] != 76 || position[2] != -264 {
		t.Fatalf("publisher position=%v, want [9 76 -264]", position)
	}
}

func TestDecodePlayerRotation(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.Float32(90)
	_ = w.Float32(-25)
	rotation, err := DecodePlayerRotation(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if rotation.Yaw != 90 || rotation.Pitch != -25 {
		t.Fatalf("rotation = %+v", rotation)
	}
}
