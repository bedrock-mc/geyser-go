package translate

import (
	"fmt"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func (b *Basic) translatePlayerAuthInputActions(bedrock *minecraft.Conn, java *javaprotocol.Client, input *packet.PlayerAuthInput) error {
	if input == nil {
		return nil
	}
	if err := b.translatePlayerAuthInputState(java, input); err != nil {
		return err
	}
	for _, action := range input.BlockActions {
		status, ok := javaPlayerActionStatus(action.Action)
		if !ok {
			b.logSemanticAnomaly("skipping unsupported Bedrock block action", "action", action.Action)
			continue
		}
		if action.Face < 0 || action.Face > 5 {
			b.logSemanticAnomaly("skipping Bedrock block action with invalid face", "face", action.Face)
			continue
		}
		data, err := encodeJavaBlockDig(status, action.BlockPos, action.Face, b.nextInteractionSequence())
		if err != nil {
			return err
		}
		if err := java.Conn.WritePacket(b.Profile.PlayServerboundBlockDigID, data); err != nil {
			return err
		}
	}

	if loadInputFlag(input, packet.InputFlagPerformItemInteraction) {
		transaction := input.ItemInteractionData
		sequence := b.nextInteractionSequence()
		switch transaction.ActionType {
		case gtprotocol.UseItemActionClickBlock:
			if transaction.BlockFace < 0 || transaction.BlockFace > 5 {
				b.logSemanticAnomaly("skipping Bedrock item interaction with invalid face", "face", transaction.BlockFace)
				break
			}
			data, err := encodeJavaBlockPlace(transaction, sequence)
			if err != nil {
				b.logSemanticAnomaly("skipping Bedrock block place with invalid cursor", "error", err)
				break
			}
			if err := java.Conn.WritePacket(b.Profile.PlayServerboundBlockPlaceID, data); err != nil {
				return err
			}
		case gtprotocol.UseItemActionClickAir:
			data, err := encodeJavaUseItem(input.InteractYaw, input.InteractPitch, sequence)
			if err != nil {
				b.logSemanticAnomaly("skipping Bedrock air interaction with invalid rotation", "error", err)
				break
			}
			if err := java.Conn.WritePacket(b.Profile.PlayServerboundUseItemID, data); err != nil {
				return err
			}
		case gtprotocol.UseItemActionBreakBlock:
			// Server-authoritative breaking is carried by BlockActions above.
		case gtprotocol.UseItemActionUseAsAttack:
			b.logSemanticAnomaly("skipping Bedrock attack without a target entity")
		default:
			b.logSemanticAnomaly("skipping unknown Bedrock item interaction", "action", transaction.ActionType)
		}
	}

	if loadInputFlag(input, packet.InputFlagPerformItemStackRequest) && len(input.ItemStackRequest.Actions) != 0 {
		return b.translateItemStackRequests(bedrock, java, []gtprotocol.ItemStackRequest{input.ItemStackRequest})
	}
	return nil
}

// translatePlayerAuthInputState forwards the edge-triggered movement state
// carried by modern Bedrock clients. The client can include both edges in one
// bitset; in that case the state already held by the bridge determines whether
// a Java transition is needed, matching Geyser's last-known-state behavior.
func (b *Basic) translatePlayerAuthInputState(java *javaprotocol.Client, input *packet.PlayerAuthInput) error {
	if err := b.translateAuthBooleanState(java, input, packet.InputFlagStartSprinting, packet.InputFlagStopSprinting, &b.sprinting, 3, 4); err != nil {
		return err
	}
	if err := b.translateAuthBooleanState(java, input, packet.InputFlagStartSneaking, packet.InputFlagStopSneaking, &b.sneaking, 0, 1); err != nil {
		return err
	}
	startGliding := loadInputFlag(input, packet.InputFlagStartGliding)
	stopGliding := loadInputFlag(input, packet.InputFlagStopGliding)
	if startGliding && !stopGliding {
		if err := b.translateTrackedState(java, &b.gliding, true, 8); err != nil { // START_ELYTRA_FLYING
			return err
		}
	} else if stopGliding {
		b.mu.Lock()
		b.gliding = false
		b.mu.Unlock()
	}
	return nil
}

func (b *Basic) translateAuthBooleanState(java *javaprotocol.Client, input *packet.PlayerAuthInput, startFlag, stopFlag int, state *bool, startAction, stopAction int32) error {
	start := loadInputFlag(input, startFlag)
	stop := loadInputFlag(input, stopFlag)
	if !start && !stop {
		return nil
	}
	desired := start
	if start && stop {
		desired = false
	}
	if desired {
		return b.translateTrackedState(java, state, true, startAction)
	}
	return b.translateTrackedState(java, state, false, stopAction)
}

func (b *Basic) translateTrackedState(java *javaprotocol.Client, state *bool, desired bool, action int32) error {
	b.mu.Lock()
	if *state == desired {
		b.mu.Unlock()
		return nil
	}
	*state = desired
	entityID := int32(uint32(b.gameData.EntityRuntimeID))
	b.mu.Unlock()
	data, err := encodeJavaEntityAction(entityID, action, 0)
	if err != nil {
		return err
	}
	return java.Conn.WritePacket(b.Profile.PlayServerboundEntityActionID, data)
}

func javaPlayerActionStatus(action int32) (int32, bool) {
	switch action {
	case gtprotocol.PlayerActionStartBreak:
		return 0, true // START_DESTROY_BLOCK
	case gtprotocol.PlayerActionAbortBreak:
		return 1, true // ABORT_DESTROY_BLOCK
	case gtprotocol.PlayerActionPredictDestroyBlock:
		return 2, true // STOP_DESTROY_BLOCK
	default:
		return 0, false
	}
}

func (b *Basic) nextInteractionSequence() int32 {
	b.mu.Lock()
	defer b.mu.Unlock()
	sequence := b.nextSequence
	if sequence <= 0 {
		sequence = 1
	}
	b.nextSequence = sequence + 1
	if b.nextSequence <= 0 {
		b.nextSequence = 1
	}
	return sequence
}

func encodeJavaBlockDig(status int32, position gtprotocol.BlockPos, face, sequence int32) ([]byte, error) {
	if status < 0 || status > 6 {
		return nil, fmt.Errorf("translate: invalid Java block-dig status %d", status)
	}
	w := javaprotocol.NewWriter()
	if err := w.VarInt(status); err != nil {
		return nil, err
	}
	if err := w.Int64(encodeJavaPosition(position)); err != nil {
		return nil, err
	}
	if err := w.Byte(byte(int8(face))); err != nil {
		return nil, err
	}
	if err := w.VarInt(sequence); err != nil {
		return nil, err
	}
	return append([]byte(nil), w.Bytes()...), nil
}

func encodeJavaBlockPlace(transaction gtprotocol.UseItemTransactionData, sequence int32) ([]byte, error) {
	if !finiteVec3(transaction.ClickedPosition) {
		return nil, fmt.Errorf("translate: Bedrock block place cursor is non-finite")
	}
	w := javaprotocol.NewWriter()
	if err := w.VarInt(0); err != nil { // MAIN_HAND
		return nil, err
	}
	if err := w.Int64(encodeJavaPosition(transaction.BlockPosition)); err != nil {
		return nil, err
	}
	if err := w.VarInt(transaction.BlockFace); err != nil {
		return nil, err
	}
	if err := w.Float32(transaction.ClickedPosition.X()); err != nil {
		return nil, err
	}
	if err := w.Float32(transaction.ClickedPosition.Y()); err != nil {
		return nil, err
	}
	if err := w.Float32(transaction.ClickedPosition.Z()); err != nil {
		return nil, err
	}
	if err := w.Bool(false); err != nil {
		return nil, err
	}
	if err := w.Bool(false); err != nil {
		return nil, err
	}
	if err := w.VarInt(sequence); err != nil {
		return nil, err
	}
	return append([]byte(nil), w.Bytes()...), nil
}

func encodeJavaUseItem(yaw, pitch float32, sequence int32) ([]byte, error) {
	if !finiteRotation(yaw, pitch) {
		return nil, fmt.Errorf("translate: Bedrock use-item rotation is non-finite")
	}
	w := javaprotocol.NewWriter()
	if err := w.VarInt(0); err != nil { // MAIN_HAND
		return nil, err
	}
	if err := w.VarInt(sequence); err != nil {
		return nil, err
	}
	if err := w.Float32(yaw); err != nil {
		return nil, err
	}
	if err := w.Float32(pitch); err != nil {
		return nil, err
	}
	return append([]byte(nil), w.Bytes()...), nil
}

func (b *Basic) translateBedrockEquipment(java *javaprotocol.Client, equipment *packet.MobEquipment) error {
	if equipment.EntityRuntimeID != b.gameData.EntityRuntimeID {
		b.logSemanticAnomaly("skipping Bedrock equipment update for another entity", "runtime_id", equipment.EntityRuntimeID)
		return nil
	}
	if equipment.WindowID != 0 {
		b.logSemanticAnomaly("skipping Bedrock offhand equipment update until Java hand-state translation is implemented", "window", equipment.WindowID)
		return nil
	}
	if equipment.HotBarSlot > 8 {
		b.logSemanticAnomaly("skipping Bedrock held slot outside Java hotbar", "slot", equipment.HotBarSlot)
		return nil
	}
	data, err := encodeJavaHeldItemSlot(equipment.HotBarSlot)
	if err != nil {
		return err
	}
	return java.Conn.WritePacket(b.Profile.PlayServerboundHeldItemSlotID, data)
}

func (b *Basic) translateBedrockAnimate(java *javaprotocol.Client, animation *packet.Animate) error {
	if animation.EntityRuntimeID != b.gameData.EntityRuntimeID {
		b.logSemanticAnomaly("skipping Bedrock animation for another entity", "runtime_id", animation.EntityRuntimeID)
		return nil
	}
	if animation.ActionType != packet.AnimateActionSwingArm {
		b.logSemanticAnomaly("skipping unsupported Bedrock animation", "action", animation.ActionType)
		return nil
	}
	data, err := encodeJavaArmAnimation(0)
	if err != nil {
		return err
	}
	return java.Conn.WritePacket(b.Profile.PlayServerboundArmAnimationID, data)
}

func (b *Basic) translateBedrockInteract(java *javaprotocol.Client, interaction *packet.Interact) error {
	if interaction.TargetEntityRuntimeID == 0 {
		b.logSemanticAnomaly("skipping Bedrock entity interaction without a target")
		return nil
	}
	var mouse int32
	switch interaction.ActionType {
	case 1: // INTERACT
		mouse = 0
	case 2: // DAMAGE
		mouse = 1
	default:
		b.logSemanticAnomaly("skipping unsupported Bedrock entity interaction", "action", interaction.ActionType)
		return nil
	}
	var position gtprotocol.Optional[mgl32.Vec3]
	if value, ok := interaction.Position.Value(); ok {
		position = gtprotocol.Option(value)
	}
	data, err := encodeJavaUseEntity(int32(uint32(interaction.TargetEntityRuntimeID)), mouse, position)
	if err != nil {
		b.logSemanticAnomaly("skipping Bedrock entity interaction with invalid position", "error", err)
		return nil
	}
	return java.Conn.WritePacket(b.Profile.PlayServerboundUseEntityID, data)
}

func (b *Basic) translateBedrockPlayerAction(java *javaprotocol.Client, action *packet.PlayerAction) error {
	if action.EntityRuntimeID != b.gameData.EntityRuntimeID {
		b.logSemanticAnomaly("skipping Bedrock player action for another entity", "runtime_id", action.EntityRuntimeID)
		return nil
	}
	if status, ok := javaPlayerActionStatus(action.ActionType); ok {
		if action.BlockFace < 0 || action.BlockFace > 5 {
			b.logSemanticAnomaly("skipping Bedrock player action with invalid face", "face", action.BlockFace)
			return nil
		}
		data, err := encodeJavaBlockDig(status, action.BlockPosition, action.BlockFace, b.nextInteractionSequence())
		if err != nil {
			return err
		}
		return java.Conn.WritePacket(b.Profile.PlayServerboundBlockDigID, data)
	}
	var javaAction int32
	switch action.ActionType {
	case gtprotocol.PlayerActionStartSneak:
		javaAction = 0 // PRESS_SHIFT_KEY
	case gtprotocol.PlayerActionStopSneak:
		javaAction = 1 // RELEASE_SHIFT_KEY
	case gtprotocol.PlayerActionStartSprint:
		javaAction = 3 // START_SPRINTING
	case gtprotocol.PlayerActionStopSprint:
		javaAction = 4 // STOP_SPRINTING
	case gtprotocol.PlayerActionStartGlide:
		javaAction = 8 // START_ELYTRA_FLYING
	default:
		b.logSemanticAnomaly("skipping unsupported Bedrock player action", "action", action.ActionType)
		return nil
	}
	data, err := encodeJavaEntityAction(int32(uint32(b.gameData.EntityRuntimeID)), javaAction, 0)
	if err != nil {
		return err
	}
	return java.Conn.WritePacket(b.Profile.PlayServerboundEntityActionID, data)
}

func encodeJavaHeldItemSlot(slot byte) ([]byte, error) {
	w := javaprotocol.NewWriter()
	if err := w.Int16(int16(slot)); err != nil {
		return nil, err
	}
	return append([]byte(nil), w.Bytes()...), nil
}

func encodeJavaArmAnimation(hand int32) ([]byte, error) {
	w := javaprotocol.NewWriter()
	if err := w.VarInt(hand); err != nil {
		return nil, err
	}
	return append([]byte(nil), w.Bytes()...), nil
}

func encodeJavaUseEntity(target, mouse int32, position gtprotocol.Optional[mgl32.Vec3]) ([]byte, error) {
	if mouse < 0 || mouse > 2 {
		return nil, fmt.Errorf("translate: invalid Java entity interaction action %d", mouse)
	}
	w := javaprotocol.NewWriter()
	if err := w.VarInt(target); err != nil {
		return nil, err
	}
	if err := w.VarInt(mouse); err != nil {
		return nil, err
	}
	if mouse == 2 {
		value, ok := position.Value()
		if !ok || !finiteVec3(value) {
			return nil, fmt.Errorf("translate: Java entity interaction is missing a finite hit position")
		}
		if err := w.Float32(value.X()); err != nil {
			return nil, err
		}
		if err := w.Float32(value.Y()); err != nil {
			return nil, err
		}
		if err := w.Float32(value.Z()); err != nil {
			return nil, err
		}
	}
	if mouse == 0 || mouse == 2 {
		if err := w.VarInt(0); err != nil { // MAIN_HAND
			return nil, err
		}
	}
	if err := w.Bool(false); err != nil {
		return nil, err
	}
	return append([]byte(nil), w.Bytes()...), nil
}

func encodeJavaEntityAction(entityID, action, jumpBoost int32) ([]byte, error) {
	w := javaprotocol.NewWriter()
	if err := w.VarInt(entityID); err != nil {
		return nil, err
	}
	if err := w.VarInt(action); err != nil {
		return nil, err
	}
	if err := w.VarInt(jumpBoost); err != nil {
		return nil, err
	}
	return append([]byte(nil), w.Bytes()...), nil
}
