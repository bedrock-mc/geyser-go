package translate

import (
	"fmt"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/go-gl/mathgl/mgl32"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// translateBedrockInventoryTransaction covers the legacy world-interaction
// packet still emitted by Bedrock clients for block use/break and entity use.
// Inventory movement itself belongs to ItemStackRequest; copying this packet's
// bytes to Java would be invalid because the two protocols have different
// transaction layouts.
func (b *Basic) translateBedrockInventoryTransaction(java *javaprotocol.Client, transaction *packet.InventoryTransaction) error {
	if transaction == nil || transaction.TransactionData == nil {
		return nil
	}
	switch data := transaction.TransactionData.(type) {
	case *gtprotocol.UseItemTransactionData:
		return b.translateLegacyUseItem(java, data)
	case *gtprotocol.UseItemOnEntityTransactionData:
		return b.translateLegacyUseEntity(java, data)
	case *gtprotocol.ReleaseItemTransactionData:
		// Java has no release packet. The corresponding use-item packet was
		// already sent when the Bedrock client started using the item.
		if data.ActionType != gtprotocol.ReleaseItemActionRelease && data.ActionType != gtprotocol.ReleaseItemActionConsume {
			b.logSemanticAnomaly("skipping Bedrock release transaction with unknown action", "action", data.ActionType)
		}
		return nil
	case *gtprotocol.NormalTransactionData, *gtprotocol.MismatchTransactionData:
		// Normal inventory movement is represented by ItemStackRequest in the
		// supported Bedrock protocol. Legacy drop transactions are handled by
		// their explicit PlayerAction packet when available.
		return nil
	default:
		b.logSemanticAnomaly("skipping unknown Bedrock inventory transaction", "type", fmt.Sprintf("%T", data))
		return nil
	}
}

func (b *Basic) translateLegacyUseItem(java *javaprotocol.Client, transaction *gtprotocol.UseItemTransactionData) error {
	if transaction == nil {
		return nil
	}
	if transaction.HotBarSlot < 0 || transaction.HotBarSlot > 8 {
		b.logSemanticAnomaly("skipping legacy Bedrock item use with invalid hotbar slot", "slot", transaction.HotBarSlot)
		return nil
	}
	if err := b.ensureJavaHeldSlot(java, transaction.HotBarSlot); err != nil {
		return err
	}
	sequence := b.nextInteractionSequence()
	switch transaction.ActionType {
	case gtprotocol.UseItemActionClickBlock:
		if transaction.BlockFace < 0 || transaction.BlockFace > 5 {
			b.logSemanticAnomaly("skipping legacy Bedrock block use with invalid face", "face", transaction.BlockFace)
			return nil
		}
		payload, err := encodeJavaBlockPlace(*transaction, sequence)
		if err != nil {
			b.logSemanticAnomaly("skipping legacy Bedrock block use with invalid cursor", "error", err)
			return nil
		}
		return java.Conn.WritePacket(b.Profile.PlayServerboundBlockPlaceID, payload)
	case gtprotocol.UseItemActionClickAir:
		b.mu.Lock()
		yaw, pitch := b.position.yaw, b.position.pitch
		b.mu.Unlock()
		payload, err := encodeJavaUseItem(yaw, pitch, sequence)
		if err != nil {
			b.logSemanticAnomaly("skipping legacy Bedrock air use with invalid rotation", "error", err)
			return nil
		}
		return java.Conn.WritePacket(b.Profile.PlayServerboundUseItemID, payload)
	case gtprotocol.UseItemActionBreakBlock:
		if transaction.BlockFace < 0 || transaction.BlockFace > 5 {
			b.logSemanticAnomaly("skipping legacy Bedrock block break with invalid face", "face", transaction.BlockFace)
			return nil
		}
		payload, err := encodeJavaBlockDig(2, transaction.BlockPosition, transaction.BlockFace, sequence)
		if err != nil {
			return err
		}
		return java.Conn.WritePacket(b.Profile.PlayServerboundBlockDigID, payload)
	case gtprotocol.UseItemActionUseAsAttack:
		// This legacy form has no target runtime ID. A target-specific
		// UseItemOnEntityTransactionData is required to produce Java ATTACK.
		b.logSemanticAnomaly("skipping legacy Bedrock attack without target")
		return nil
	default:
		b.logSemanticAnomaly("skipping unknown legacy Bedrock item action", "action", transaction.ActionType)
		return nil
	}
}

func (b *Basic) translateLegacyUseEntity(java *javaprotocol.Client, transaction *gtprotocol.UseItemOnEntityTransactionData) error {
	if transaction == nil || transaction.TargetEntityRuntimeID == 0 {
		b.logSemanticAnomaly("skipping legacy Bedrock entity use without target")
		return nil
	}
	if transaction.HotBarSlot < 0 || transaction.HotBarSlot > 8 {
		b.logSemanticAnomaly("skipping legacy Bedrock entity use with invalid hotbar slot", "slot", transaction.HotBarSlot)
		return nil
	}
	if err := b.ensureJavaHeldSlot(java, transaction.HotBarSlot); err != nil {
		return err
	}
	b.mu.Lock()
	sneaking := b.sneaking
	b.mu.Unlock()
	var mouse int32
	var position gtprotocol.Optional[mgl32.Vec3]
	switch transaction.ActionType {
	case gtprotocol.UseItemOnEntityActionInteract:
		// Bedrock supplies a relative hit point; Java's INTERACT_AT retains
		// that point and lets armor stands, boats, and similar entities make
		// the same hitbox decision as a Java client.
		mouse = 2
		position = gtprotocol.Option(transaction.ClickedPosition)
	case gtprotocol.UseItemOnEntityActionAttack:
		mouse = 1
	default:
		b.logSemanticAnomaly("skipping unknown legacy Bedrock entity action", "action", transaction.ActionType)
		return nil
	}
	payload, err := encodeJavaUseEntityState(int32(uint32(transaction.TargetEntityRuntimeID)), mouse, position, sneaking)
	if err != nil {
		b.logSemanticAnomaly("skipping legacy Bedrock entity use with invalid hit position", "error", err)
		return nil
	}
	if err := java.Conn.WritePacket(b.Profile.PlayServerboundUseEntityID, payload); err != nil {
		return err
	}
	if transaction.ActionType == gtprotocol.UseItemOnEntityActionAttack {
		// Vanilla Java emits a swing immediately after ATTACK. Bedrock may
		// also send Animate, but the duplicate is harmless and preserves the
		// ordering expected by Java servers when only this transaction arrives.
		payload, err = encodeJavaArmAnimation(0)
		if err != nil {
			return err
		}
		return java.Conn.WritePacket(b.Profile.PlayServerboundArmAnimationID, payload)
	}
	return nil
}
