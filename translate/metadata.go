package translate

import (
	"errors"
	"fmt"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/go-gl/mathgl/mgl32"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
)

var ErrUnsupportedJavaEntityMetadata = errors.New("translate: unsupported Java entity metadata type")

type JavaEntityMetadata struct {
	EntityID int32
	Entries  []JavaEntityMetadataEntry
}

type JavaEntityMetadataEntry struct {
	Index byte
	Type  int32
	Value any
}

func DecodeEntityMetadata(payload []byte, nextStackID func() int32) (JavaEntityMetadata, error) {
	r := javaprotocol.NewReader(payload)
	entityID, err := r.VarInt()
	if err != nil {
		return JavaEntityMetadata{}, fmt.Errorf("translate: entity metadata ID: %w", err)
	}
	entries := make([]JavaEntityMetadataEntry, 0, 8)
	for i := 0; i < maxJavaCollectionSize; i++ {
		index, err := r.Uint8()
		if err != nil {
			return JavaEntityMetadata{}, fmt.Errorf("translate: entity metadata entry %d index: %w", i, err)
		}
		if index == 0xff {
			if r.Remaining() != 0 {
				return JavaEntityMetadata{}, fmt.Errorf("translate: entity metadata has %d trailing bytes", r.Remaining())
			}
			return JavaEntityMetadata{EntityID: entityID, Entries: entries}, nil
		}
		typeID, err := r.VarInt()
		if err != nil {
			return JavaEntityMetadata{}, fmt.Errorf("translate: entity metadata entry %d type: %w", i, err)
		}
		value, err := decodeJavaEntityMetadataValue(r, typeID, nextStackID)
		if err != nil {
			return JavaEntityMetadata{}, fmt.Errorf("translate: entity metadata entry %d: %w", i, err)
		}
		entries = append(entries, JavaEntityMetadataEntry{Index: index, Type: typeID, Value: value})
	}
	return JavaEntityMetadata{}, fmt.Errorf("translate: entity metadata has more than %d entries", maxJavaCollectionSize)
}

func decodeJavaEntityMetadataValue(r *javaprotocol.Reader, typeID int32, nextStackID func() int32) (any, error) {
	switch typeID {
	case 0: // byte
		return r.Int8()
	case 1: // varint
		return r.VarInt()
	case 2: // varlong
		return r.VarLong()
	case 3: // float
		return r.Float32()
	case 4: // string
		return r.String()
	case 5, 16: // component or compound tag
		return decodeJavaNBTValue(r)
	case 6: // optional component
		present, err := r.Bool()
		if err != nil || !present {
			return nil, err
		}
		return decodeJavaNBTValue(r)
	case 7: // item stack
		item, err := decodeJavaItemSlot(r, nextStackID)
		if err != nil {
			return nil, err
		}
		return item.Item, nil
	case 8: // boolean
		return r.Bool()
	case 9: // rotations
		pitch, err := r.Float32()
		if err != nil {
			return nil, err
		}
		yaw, err := r.Float32()
		if err != nil {
			return nil, err
		}
		roll, err := r.Float32()
		if err != nil {
			return nil, err
		}
		return mgl32.Vec3{pitch, yaw, roll}, nil
	case 10: // block position
		packed, err := r.Int64()
		if err != nil {
			return nil, err
		}
		return decodeJavaPosition(packed), nil
	case 11: // optional block position
		present, err := r.Bool()
		if err != nil || !present {
			return nil, err
		}
		packed, err := r.Int64()
		if err != nil {
			return nil, err
		}
		return decodeJavaPosition(packed), nil
	case 12: // direction
		return r.VarInt()
	case 13: // optional UUID
		present, err := r.Bool()
		if err != nil || !present {
			return nil, err
		}
		value, err := r.Bytes(16)
		if err != nil {
			return nil, err
		}
		var uuid [16]byte
		copy(uuid[:], value)
		return uuid, nil
	case 14: // block state
		return r.VarInt()
	case 15, 20: // optional block state / optional unsigned int
		value, err := r.VarInt()
		if err != nil {
			return nil, err
		}
		if value == 0 {
			return nil, nil
		}
		return value - 1, nil
	case 17, 18, 23, 25, 26:
		// These values contain registry-dependent particle/variant/global-position
		// payloads. Their packet is skipped as a unit until the active registry
		// translator is available; the framed Java session remains usable.
		return nil, fmt.Errorf("%w: type=%d", ErrUnsupportedJavaEntityMetadata, typeID)
	case 19: // villager data
		values := make([]int32, 3)
		for i := range values {
			value, err := r.VarInt()
			if err != nil {
				return nil, err
			}
			values[i] = value
		}
		return values, nil
	case 21, 22, 24, 27, 28: // pose/cat/frog/sniffer/armadillo state
		return r.VarInt()
	case 29: // vector3
		x, err := r.Float32()
		if err != nil {
			return nil, err
		}
		y, err := r.Float32()
		if err != nil {
			return nil, err
		}
		z, err := r.Float32()
		if err != nil {
			return nil, err
		}
		return mgl32.Vec3{x, y, z}, nil
	case 30: // quaternion
		values := make([]float32, 4)
		for i := range values {
			value, err := r.Float32()
			if err != nil {
				return nil, err
			}
			values[i] = value
		}
		return values, nil
	default:
		return nil, fmt.Errorf("%w: type=%d", ErrUnsupportedJavaEntityMetadata, typeID)
	}
}

func translateGenericEntityMetadata(entries []JavaEntityMetadataEntry) gtprotocol.EntityMetadata {
	metadata := gtprotocol.NewEntityMetadataWithCapacity(10)
	for _, entry := range entries {
		switch entry.Index {
		case 0:
			flags, ok := entry.Value.(int8)
			if !ok {
				continue
			}
			for _, mapping := range []struct {
				javaBit byte
				bedrock uint8
			}{
				{0, gtprotocol.EntityDataFlagOnFire},
				{1, gtprotocol.EntityDataFlagSneaking},
				{3, gtprotocol.EntityDataFlagSprinting},
				{4, gtprotocol.EntityDataFlagSwimming},
				{5, gtprotocol.EntityDataFlagInvisible},
				{7, gtprotocol.EntityDataFlagGliding},
			} {
				if byte(flags)&(1<<mapping.javaBit) != 0 {
					metadata.SetFlag(gtprotocol.EntityDataKeyFlags, mapping.bedrock)
				}
			}
		case 1:
			if value, ok := entry.Value.(int32); ok {
				metadata[gtprotocol.EntityDataKeyAirSupply] = int64(value)
			}
		case 2:
			if value := JavaTextComponentText(entry.Value); value != "" {
				metadata[gtprotocol.EntityDataKeyName] = value
			}
		case 3:
			if value, ok := entry.Value.(bool); ok {
				if value {
					metadata[gtprotocol.EntityDataKeyAlwaysShowNameTag] = int64(1)
				} else {
					metadata[gtprotocol.EntityDataKeyAlwaysShowNameTag] = int64(0)
				}
			}
		case 4:
			if value, ok := entry.Value.(bool); ok && value {
				metadata.SetFlag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagSilent)
			}
		case 5:
			if value, ok := entry.Value.(bool); ok && !value {
				metadata.SetFlag(gtprotocol.EntityDataKeyFlags, gtprotocol.EntityDataFlagHasGravity)
			}
		case 6:
			if value, ok := entry.Value.(int32); ok {
				metadata[gtprotocol.EntityDataKeyPoseIndex] = int64(value)
			}
		}
	}
	return metadata
}
