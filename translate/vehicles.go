package translate

import (
	"fmt"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// JavaVehicleMove is ClientboundMoveVehiclePacket from Java 1.21.4. The
// packet has no entity ID: the protocol applies it to the vehicle carrying
// the local player.
type JavaVehicleMove struct {
	Position mgl32.Vec3
	Yaw      float32
	Pitch    float32
}

func DecodeJavaVehicleMove(data []byte) (JavaVehicleMove, error) {
	r := javaprotocol.NewReader(data)
	position, err := readJavaPosition(r, "vehicle move")
	if err != nil {
		return JavaVehicleMove{}, err
	}
	yaw, err := r.Float32()
	if err != nil {
		return JavaVehicleMove{}, fmt.Errorf("translate: vehicle move yaw: %w", err)
	}
	pitch, err := r.Float32()
	if err != nil {
		return JavaVehicleMove{}, fmt.Errorf("translate: vehicle move pitch: %w", err)
	}
	if r.Remaining() != 0 {
		return JavaVehicleMove{}, fmt.Errorf("translate: vehicle move has %d trailing bytes", r.Remaining())
	}
	return JavaVehicleMove{Position: position, Yaw: yaw, Pitch: pitch}, nil
}

func (b *Basic) translateJavaVehicleMove(bedrock *minecraft.Conn, data []byte) error {
	move, err := DecodeJavaVehicleMove(data)
	if err != nil {
		return err
	}
	if !finiteVec3(move.Position) || !finiteRotation(move.Yaw, move.Pitch) {
		b.logSemanticAnomaly("skipping Java vehicle movement with non-finite position or rotation")
		return nil
	}

	b.mu.Lock()
	vehicleID, entity, riding := b.javaCurrentVehicleLocked()
	if !riding {
		b.mu.Unlock()
		b.logSemanticAnomaly("skipping Java vehicle movement without a local passenger")
		return nil
	}
	if entity == nil {
		b.mu.Unlock()
		b.logSemanticAnomaly("skipping Java vehicle movement for an unknown vehicle", "entity", vehicleID)
		return nil
	}
	entity.position = move.Position
	entity.rotation[0] = move.Pitch
	entity.rotation[1] = move.Yaw
	entity.rotation[2] = move.Yaw
	runtimeID := entity.runtimeID
	position := entity.position
	rotation := entity.rotation
	b.mu.Unlock()

	// Geyser marks this path as teleported so the Bedrock client snaps the
	// linked rider and vehicle together instead of interpolating a stale
	// client-predicted vehicle position.
	return bedrock.WritePacket(&packet.MoveActorAbsolute{
		EntityRuntimeID: runtimeID,
		Flags:           packet.MoveFlagTeleport,
		Position:        position,
		Rotation:        rotation,
	})
}
