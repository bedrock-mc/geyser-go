package translate

import (
	"fmt"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/sandertv/gophertunnel/minecraft"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// JavaBlockEvent is the raw shape of Java's ClientboundBlockEventPacket.
// Java interprets the two unsigned bytes according to the block registry ID;
// retaining the raw values lets the translator remain lenient for newer block
// types while still rejecting malformed wire data.
type JavaBlockEvent struct {
	Position gtprotocol.BlockPos
	Type     byte
	Value    byte
	BlockID  int32
}

// DecodeJavaBlockEvent decodes one complete Java block-action packet.
func DecodeJavaBlockEvent(data []byte) (JavaBlockEvent, error) {
	r := javaprotocol.NewReader(data)
	packed, err := r.Int64()
	if err != nil {
		return JavaBlockEvent{}, fmt.Errorf("translate: block event position: %w", err)
	}
	eventType, err := r.Byte()
	if err != nil {
		return JavaBlockEvent{}, fmt.Errorf("translate: block event type: %w", err)
	}
	eventValue, err := r.Byte()
	if err != nil {
		return JavaBlockEvent{}, fmt.Errorf("translate: block event value: %w", err)
	}
	blockID, err := r.VarInt()
	if err != nil {
		return JavaBlockEvent{}, fmt.Errorf("translate: block event block ID: %w", err)
	}
	if r.Remaining() != 0 {
		return JavaBlockEvent{}, fmt.Errorf("translate: block event has %d trailing bytes", r.Remaining())
	}
	return JavaBlockEvent{
		Position: decodeJavaPosition(packed),
		Type:     eventType,
		Value:    eventValue,
		BlockID:  blockID,
	}, nil
}

const (
	javaBlockEventNoteBlock        = 109
	javaBlockEventMobSpawner       = 198
	javaBlockEventChest            = 201
	javaBlockEventEnderChest       = 400
	javaBlockEventTrappedChest     = 470
	javaBlockEventEndGateway       = 667
	javaBlockEventShulkerBoxLower  = 677
	javaBlockEventShulkerBoxUpper  = 693
	javaBlockEventBell             = 848
	javaBlockEventCopperChestLower = 1081
	javaBlockEventCopperChestUpper = 1088
	javaBlockEventDecoratedPot     = 1155
)

func javaBlockEventIsChest(blockID int32) bool {
	return blockID == javaBlockEventChest ||
		blockID == javaBlockEventEnderChest ||
		blockID == javaBlockEventTrappedChest ||
		(blockID >= javaBlockEventShulkerBoxLower && blockID <= javaBlockEventShulkerBoxUpper) ||
		(blockID >= javaBlockEventCopperChestLower && blockID <= javaBlockEventCopperChestUpper)
}

// translateJavaBlockEvent covers the direct Bedrock BlockEvent projections
// used by Geyser for chest-like blocks, end gateways, mob spawners, and note
// blocks. Piston, bell, and decorated-pot events require a world/entity cache
// that this foundation does not yet own, so they are logged and skipped as
// semantically unsupported rather than forwarded with the wrong meaning.
func (b *Basic) translateJavaBlockEvent(bedrock *minecraft.Conn, data []byte) error {
	event, err := DecodeJavaBlockEvent(data)
	if err != nil {
		return err
	}

	var translated *packet.BlockEvent
	switch {
	case javaBlockEventIsChest(event.BlockID):
		translated = &packet.BlockEvent{
			Position:  event.Position,
			EventType: packet.BlockEventChangeChestState,
			EventData: boolInt32(event.Value != 0),
		}
	case event.BlockID == javaBlockEventEndGateway || event.BlockID == javaBlockEventMobSpawner:
		translated = &packet.BlockEvent{
			Position:  event.Position,
			EventType: packet.BlockEventChangeChestState,
		}
	case event.BlockID == javaBlockEventNoteBlock:
		translated = &packet.BlockEvent{
			Position:  event.Position,
			EventData: int32(event.Value),
		}
	case event.BlockID == javaBlockEventBell || event.BlockID == javaBlockEventDecoratedPot:
		b.logSemanticAnomaly("skipping Java block event requiring a typed block-entity projection", "block_id", event.BlockID, "type", event.Type)
		return nil
	default:
		b.logSemanticAnomaly("skipping unknown Java block event block", "block_id", event.BlockID, "type", event.Type)
		return nil
	}
	return bedrock.WritePacket(translated)
}

func boolInt32(value bool) int32 {
	if value {
		return 1
	}
	return 0
}
