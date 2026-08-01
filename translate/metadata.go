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

func hasEntityMetadataEntry(entries []JavaEntityMetadataEntry, index byte) bool {
	for _, entry := range entries {
		if entry.Index == index {
			return true
		}
	}
	return false
}

// JavaGlobalPos is the bounded representation of Java's optional global
// position metadata. The dimension key is retained even though the current
// Bedrock actor metadata projection does not consume it.
type JavaGlobalPos struct {
	Dimension string
	Position  gtprotocol.BlockPos
}

// JavaPaintingVariant is the holder form used by 1.21.4 painting metadata.
// Registry references carry only an ordinal; custom holders additionally carry
// the dimensions and asset key needed to project an AddPainting packet.
type JavaPaintingVariant struct {
	RegistryID int32
	Width      int32
	Height     int32
	AssetID    string
	Custom     bool
}

// JavaWolfVariant is the 1.21.4 registry-holder form used by wolf metadata.
// Vanilla sends a registry reference; custom direct values are decoded and
// retained as Custom so the enclosing metadata packet stays aligned.
type JavaWolfVariant struct {
	RegistryID int32
	Custom     bool
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
	case 15: // optional block state (the 1.21.4 wire type is a direct VarInt)
		return r.VarInt()
	case 20: // optional unsigned int
		value, err := r.VarInt()
		if err != nil {
			return nil, err
		}
		if value == 0 {
			return nil, nil
		}
		return value - 1, nil
	case 17, 18: // particle and particle list
		// These metadata values contain registry-dependent payloads. The
		// enclosing packet is skipped until the active registry translators are
		// available; the framed Java session remains usable.
		return nil, fmt.Errorf("%w: type=%d", ErrUnsupportedJavaEntityMetadata, typeID)
	case 23: // wolf variant registry holder
		selector, err := r.VarInt()
		if err != nil {
			return nil, err
		}
		if selector < 0 {
			return nil, fmt.Errorf("negative wolf variant selector %d", selector)
		}
		if selector != 0 {
			return JavaWolfVariant{RegistryID: selector - 1}, nil
		}
		for _, field := range []string{"wolf wild texture", "wolf tame texture", "wolf angry texture"} {
			if _, err := r.String(); err != nil {
				return nil, fmt.Errorf("%s: %w", field, err)
			}
		}
		if err := decodeJavaIDSet(r); err != nil {
			return nil, fmt.Errorf("wolf biome set: %w", err)
		}
		return JavaWolfVariant{RegistryID: -1, Custom: true}, nil
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
	case 25: // optional global position
		present, err := r.Bool()
		if err != nil || !present {
			return nil, err
		}
		dimension, err := r.String()
		if err != nil {
			return nil, err
		}
		packed, err := r.Int64()
		if err != nil {
			return nil, err
		}
		return JavaGlobalPos{Dimension: dimension, Position: decodeJavaPosition(packed)}, nil
	case 26: // painting variant holder
		selector, err := r.VarInt()
		if err != nil {
			return nil, err
		}
		if selector != 0 {
			return JavaPaintingVariant{RegistryID: selector - 1}, nil
		}
		width, err := r.VarInt()
		if err != nil {
			return nil, err
		}
		height, err := r.VarInt()
		if err != nil {
			return nil, err
		}
		assetID, err := r.String()
		if err != nil {
			return nil, err
		}
		for _, field := range []string{"painting title", "painting author"} {
			present, readErr := r.Bool()
			if readErr != nil {
				return nil, readErr
			}
			if present {
				if _, readErr = decodeJavaNBTValue(r); readErr != nil {
					return nil, fmt.Errorf("%s: %w", field, readErr)
				}
			}
		}
		return JavaPaintingVariant{Width: width, Height: height, AssetID: assetID, Custom: true}, nil
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
					metadata[gtprotocol.EntityDataKeyAlwaysShowNameTag] = byte(1)
				} else {
					metadata[gtprotocol.EntityDataKeyAlwaysShowNameTag] = byte(0)
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
