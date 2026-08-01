package translate

import (
	"math"
	"testing"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/go-gl/mathgl/mgl32"
)

func TestDecodeJavaVehicleMove(t *testing.T) {
	w := javaprotocol.NewWriter()
	for _, value := range []float64{12.5, 64, -7.25} {
		if err := w.Float64(value); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Float32(90); err != nil {
		t.Fatal(err)
	}
	if err := w.Float32(-15); err != nil {
		t.Fatal(err)
	}

	move, err := DecodeJavaVehicleMove(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if move.Position != (mgl32.Vec3{12.5, 64, -7.25}) || move.Yaw != 90 || move.Pitch != -15 {
		t.Fatalf("vehicle move = %+v", move)
	}
	if javaprotocol.Java1214.PlayClientboundVehicleMoveID != 0x33 {
		t.Fatalf("vehicle move packet ID = %#x, want 0x33", javaprotocol.Java1214.PlayClientboundVehicleMoveID)
	}
}

func TestDecodeJavaVehicleMoveRejectsTrailingData(t *testing.T) {
	w := javaprotocol.NewWriter()
	for range 3 {
		_ = w.Float64(1)
	}
	_ = w.Float32(0)
	_ = w.Float32(0)
	_ = w.Byte(0xff)
	if _, err := DecodeJavaVehicleMove(w.Bytes()); err == nil {
		t.Fatal("vehicle move with trailing data was accepted")
	}
}

func TestJavaCurrentVehicleTracksLocalPassenger(t *testing.T) {
	b := NewBasic(javaprotocol.Java1214, nil)
	b.gameData.EntityUniqueID = 100
	b.passengers[42] = []int32{100, 101}
	want := &javaEntityState{runtimeID: 900, entityType: "minecraft:boat"}
	b.entities[42] = want

	b.mu.Lock()
	id, got, ok := b.javaCurrentVehicleLocked()
	b.mu.Unlock()
	if !ok || id != 42 || got != want {
		t.Fatalf("current vehicle = id %d, entity %p, ok %t", id, got, ok)
	}
}

func TestJavaCurrentVehicleClearsAfterPassengerRemoval(t *testing.T) {
	b := NewBasic(javaprotocol.Java1214, nil)
	b.gameData.EntityUniqueID = 100
	b.passengers[42] = []int32{100}
	b.vehicleID = 42
	b.hasVehicle = true

	b.passengers[42] = nil
	b.mu.Lock()
	_, _, ok := b.javaCurrentVehicleLocked()
	b.mu.Unlock()
	if ok || b.hasVehicle {
		t.Fatalf("vehicle state remained active after passenger removal: has=%t id=%d", b.hasVehicle, b.vehicleID)
	}
}

func TestDecodeJavaVehicleMoveLeavesNonFiniteValuesForLenientProjection(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.Float64(math.NaN())
	_ = w.Float64(64)
	_ = w.Float64(0)
	_ = w.Float32(0)
	_ = w.Float32(0)
	move, err := DecodeJavaVehicleMove(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if finiteVec3(move.Position) {
		t.Fatal("non-finite vehicle position was normalized during decode")
	}
}
