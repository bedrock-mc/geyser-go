package translate

import (
	"fmt"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
)

// JavaMultiBlockChange is the Java section-block-update packet. The section
// coordinates identify one 16x16x16 section; each change contains an absolute
// block position and a Java registry state ID.
type JavaMultiBlockChange struct {
	SectionX int32
	SectionY int32
	SectionZ int32
	Changes  []JavaBlockChange
}

func DecodeMultiBlockChange(payload []byte) (JavaMultiBlockChange, error) {
	r := javaprotocol.NewReader(payload)
	sectionPosition, err := r.Int64()
	if err != nil {
		return JavaMultiBlockChange{}, fmt.Errorf("translate: multi-block section position: %w", err)
	}
	count, err := r.VarInt()
	if err != nil {
		return JavaMultiBlockChange{}, fmt.Errorf("translate: multi-block count: %w", err)
	}
	if count < 0 || count > maxJavaCollectionSize {
		return JavaMultiBlockChange{}, fmt.Errorf("translate: multi-block count %d exceeds limit %d", count, maxJavaCollectionSize)
	}
	sectionX := signExtend(uint32(uint64(sectionPosition)>>42)&0x003fffff, 22)
	sectionZ := signExtend(uint32(uint64(sectionPosition)>>20)&0x003fffff, 22)
	sectionY := signExtend(uint32(uint64(sectionPosition))&0x000fffff, 20)
	changes := make([]JavaBlockChange, 0, count)
	for i := 0; i < int(count); i++ {
		record, err := r.VarInt()
		if err != nil {
			return JavaMultiBlockChange{}, fmt.Errorf("translate: multi-block record %d: %w", i, err)
		}
		packed := uint32(record)
		stateID := int32(packed >> 12)
		localX := int32((packed >> 8) & 0x0f)
		localZ := int32((packed >> 4) & 0x0f)
		localY := int32(packed & 0x0f)
		changes = append(changes, JavaBlockChange{
			Position: gtprotocol.BlockPos{
				sectionX*16 + localX,
				sectionY*16 + localY,
				sectionZ*16 + localZ,
			},
			StateID: stateID,
		})
	}
	if r.Remaining() != 0 {
		return JavaMultiBlockChange{}, fmt.Errorf("translate: multi-block update has %d trailing bytes", r.Remaining())
	}
	return JavaMultiBlockChange{
		SectionX: sectionX,
		SectionY: sectionY,
		SectionZ: sectionZ,
		Changes:  changes,
	}, nil
}
