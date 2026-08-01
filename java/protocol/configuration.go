package protocol

import (
	"bytes"
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

const (
	maxConfigurationCount      = 4096
	maxConfigurationTagEntries = 16384
)

// RegistryData is one Java configuration registry. Values are retained as
// decoded anonymous NBT so a versioned translator can consume dimension,
// biome, chat-type, damage-type, and custom registries without re-reading the
// wire stream.
type RegistryData struct {
	ID      string
	Entries []RegistryEntry
}

type RegistryEntry struct {
	Key      string
	HasValue bool
	Value    any
}

// Tags groups Java tag IDs by registry and tag name. Unknown registries and
// tag names are intentionally retained; consumers decide which ones have
// meaning for their active Bedrock profile.
type Tags map[string]map[string][]int32

type ConfigurationData struct {
	Registries   []RegistryData
	FeatureFlags []string
	Tags         Tags
	ResetChat    bool
}

// DecodeRegistryData decodes Java's configuration registry_data packet. A
// malformed length, NBT value, or trailing byte is a wire error; an unknown
// registry identifier is valid and is preserved for the caller.
func DecodeRegistryData(data []byte) (RegistryData, error) {
	r := NewReader(data)
	registryID, err := r.String()
	if err != nil {
		return RegistryData{}, fmt.Errorf("java protocol: registry id: %w", err)
	}
	count, err := configurationCount(r, "registry entry")
	if err != nil {
		return RegistryData{}, err
	}
	result := RegistryData{ID: registryID, Entries: make([]RegistryEntry, 0, count)}
	for i := 0; i < count; i++ {
		key, err := r.String()
		if err != nil {
			return RegistryData{}, fmt.Errorf("java protocol: registry entry %d key: %w", i, err)
		}
		hasValue, err := r.Bool()
		if err != nil {
			return RegistryData{}, fmt.Errorf("java protocol: registry entry %d presence: %w", i, err)
		}
		entry := RegistryEntry{Key: key, HasValue: hasValue}
		if hasValue {
			entry.Value, err = DecodeAnonymousNBT(r)
			if err != nil {
				return RegistryData{}, fmt.Errorf("java protocol: registry entry %d value: %w", i, err)
			}
		}
		result.Entries = append(result.Entries, entry)
	}
	if r.Remaining() != 0 {
		return RegistryData{}, fmt.Errorf("java protocol: registry data has %d trailing bytes", r.Remaining())
	}
	return result, nil
}

func DecodeFeatureFlags(data []byte) ([]string, error) {
	r := NewReader(data)
	count, err := configurationCount(r, "feature flag")
	if err != nil {
		return nil, err
	}
	flags := make([]string, 0, count)
	for i := 0; i < count; i++ {
		value, err := r.String()
		if err != nil {
			return nil, fmt.Errorf("java protocol: feature flag %d: %w", i, err)
		}
		flags = append(flags, value)
	}
	if r.Remaining() != 0 {
		return nil, fmt.Errorf("java protocol: feature flags have %d trailing bytes", r.Remaining())
	}
	return flags, nil
}

func DecodeTags(data []byte) (Tags, error) {
	r := NewReader(data)
	registryCount, err := configurationCount(r, "tag registry")
	if err != nil {
		return nil, err
	}
	result := make(Tags, registryCount)
	totalEntries := 0
	for i := 0; i < registryCount; i++ {
		registryID, err := r.String()
		if err != nil {
			return nil, fmt.Errorf("java protocol: tag registry %d: %w", i, err)
		}
		tagCount, err := configurationCount(r, "tag")
		if err != nil {
			return nil, err
		}
		registryTags := make(map[string][]int32, tagCount)
		for j := 0; j < tagCount; j++ {
			name, err := r.String()
			if err != nil {
				return nil, fmt.Errorf("java protocol: tag registry %d tag %d name: %w", i, j, err)
			}
			entryCount, err := configurationCount(r, "tag entry")
			if err != nil {
				return nil, err
			}
			totalEntries += entryCount
			if totalEntries > maxConfigurationTagEntries {
				return nil, fmt.Errorf("java protocol: tag entries exceed limit %d", maxConfigurationTagEntries)
			}
			entries := make([]int32, entryCount)
			for k := range entries {
				entries[k], err = r.VarInt()
				if err != nil {
					return nil, fmt.Errorf("java protocol: tag registry %d tag %d entry %d: %w", i, j, k, err)
				}
			}
			registryTags[name] = entries
		}
		result[registryID] = registryTags
	}
	if r.Remaining() != 0 {
		return nil, fmt.Errorf("java protocol: tags have %d trailing bytes", r.Remaining())
	}
	return result, nil
}

func configurationCount(r *Reader, field string) (int, error) {
	count, err := r.VarInt()
	if err != nil {
		return 0, fmt.Errorf("java protocol: %s count: %w", field, err)
	}
	if count < 0 || count > maxConfigurationCount {
		return 0, fmt.Errorf("java protocol: %s count %d exceeds limit %d", field, count, maxConfigurationCount)
	}
	return int(count), nil
}

// DecodeAnonymousNBT decodes the root form used by Java anonymousNbt. The
// root name is omitted on the wire. TAG_End is the optional-null form and a
// string root is handled explicitly because the bundled decoder expects a
// compound root for network NBT.
func DecodeAnonymousNBT(r *Reader) (any, error) {
	tag, err := r.Byte()
	if err != nil {
		return nil, err
	}
	if tag == 0 {
		return nil, nil
	}
	if tag == 8 {
		length, err := r.Int16()
		if err != nil {
			return nil, err
		}
		if length < 0 {
			return nil, fmt.Errorf("java protocol: negative NBT string length %d", length)
		}
		value, err := r.Bytes(int(length))
		if err != nil {
			return nil, err
		}
		if !utf8.Valid(value) {
			return nil, ErrMalformedString
		}
		return string(value), nil
	}
	if tag != 10 {
		return nil, fmt.Errorf("java protocol: unsupported anonymous NBT root tag %d", tag)
	}
	counted := &configurationCountingReader{Reader: io.MultiReader(bytes.NewReader([]byte{tag}), r)}
	var value any
	decoder := nbt.NewDecoderWithEncoding(counted, nbt.NetworkBigEndian)
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	return value, nil
}

type configurationCountingReader struct {
	io.Reader
}
