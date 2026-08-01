package translate

import "testing"

func TestFloorBlockPositionUsesJavaBlockCoordinates(t *testing.T) {
	position := floorBlockPosition(javaPosition{x: 9.5, y: 76, z: -263.5})
	if position[0] != 9 || position[1] != 76 || position[2] != -264 {
		t.Fatalf("publisher position=%v, want [9 76 -264]", position)
	}
}
