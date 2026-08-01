package translate

import (
	"fmt"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type JavaAnimation struct {
	EntityID  int32
	Animation byte
}

func DecodeAnimation(payload []byte) (JavaAnimation, error) {
	r := javaprotocol.NewReader(payload)
	entityID, err := r.VarInt()
	if err != nil {
		return JavaAnimation{}, fmt.Errorf("translate: animation entity ID: %w", err)
	}
	animation, err := r.Byte()
	if err != nil {
		return JavaAnimation{}, fmt.Errorf("translate: animation type: %w", err)
	}
	if r.Remaining() != 0 {
		return JavaAnimation{}, fmt.Errorf("translate: animation has %d trailing bytes", r.Remaining())
	}
	return JavaAnimation{EntityID: entityID, Animation: animation}, nil
}

func (b *Basic) translateAnimation(bedrock *minecraft.Conn, payload []byte) error {
	animation, err := DecodeAnimation(payload)
	if err != nil {
		return err
	}
	b.mu.Lock()
	entity := b.entities[animation.EntityID]
	b.mu.Unlock()
	if entity == nil {
		return nil
	}
	var action uint8
	switch animation.Animation {
	case 0, 3: // SWING_MAIN_HAND, SWING_OFF_HAND
		action = packet.AnimateActionSwingArm
	case 2: // LEAVE_BED
		action = packet.AnimateActionStopSleep
	case 4: // CRITICAL_HIT
		action = packet.AnimateActionCriticalHit
	case 5: // ENCHANTED_HIT
		action = packet.AnimateActionMagicCriticalHit
	default:
		b.logSemanticAnomaly("skipping unsupported Java animation", "animation", animation.Animation)
		return nil
	}
	return bedrock.WritePacket(&packet.Animate{
		ActionType:      action,
		EntityRuntimeID: entity.runtimeID,
		SwingSource:     packet.AnimateSwingSourceNone,
	})
}
