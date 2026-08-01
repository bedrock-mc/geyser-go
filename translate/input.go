package translate

import (
	"fmt"
	"math"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

const bedrockPlayerEyeOffset = 1.62

func finiteRotation(yaw, pitch float32) bool {
	return !math.IsNaN(float64(yaw)) && !math.IsInf(float64(yaw), 0) &&
		!math.IsNaN(float64(pitch)) && !math.IsInf(float64(pitch), 0)
}

func encodeBedrockPositionLook(position mgl32.Vec3, yaw, pitch float32, onGround, horizontalCollision bool) ([]byte, error) {
	if !finiteVec3(position) || !finiteRotation(yaw, pitch) {
		return nil, fmt.Errorf("translate: Bedrock movement contains non-finite position or rotation")
	}
	flags := byte(0)
	if onGround {
		flags |= 1 << 0
	}
	if horizontalCollision {
		flags |= 1 << 1
	}
	w := javaprotocol.NewWriter()
	if err := w.Float64(float64(position.X())); err != nil {
		return nil, err
	}
	if err := w.Float64(float64(position.Y()) - bedrockPlayerEyeOffset); err != nil {
		return nil, err
	}
	if err := w.Float64(float64(position.Z())); err != nil {
		return nil, err
	}
	if err := w.Float32(yaw); err != nil {
		return nil, err
	}
	if err := w.Float32(pitch); err != nil {
		return nil, err
	}
	if err := w.Byte(flags); err != nil {
		return nil, err
	}
	return append([]byte(nil), w.Bytes()...), nil
}

func encodePlayerAuthInput(pk *packet.PlayerAuthInput) ([]byte, error) {
	if pk == nil {
		return nil, fmt.Errorf("translate: nil PlayerAuthInput")
	}
	return encodeBedrockPositionLook(
		pk.Position,
		pk.Yaw,
		pk.Pitch,
		loadInputFlag(pk, packet.InputFlagVerticalCollision),
		loadInputFlag(pk, packet.InputFlagHorizontalCollision),
	)
}

func loadInputFlag(pk *packet.PlayerAuthInput, flag int) bool {
	return pk.InputData.Len() > flag && pk.InputData.Load(flag)
}
