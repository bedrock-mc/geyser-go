// Package data contains the versioned registry model used by translators.
// Runtime behavior must not infer vanilla support from Dragonfly's behavioral
// implementations: the catalog is generated from complete data sources.
package data

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type Catalog struct {
	Version string
	Source  SourceManifest
	Items   []ItemDefinition
	Blocks  []BlockDefinition
}

type SourceManifest struct {
	Name  string
	Files []SourceFile
}

type SourceFile struct {
	Name   string
	SHA256 string
}

type ItemDefinition struct {
	Name           string
	RuntimeID      int32
	Version        int32
	ComponentBased bool
	// SourceJSON retains fields introduced by newer Bedrock releases until a
	// typed consumer is added. This prevents the generator from silently
	// dropping authoritative data.
	SourceJSON string
}

type BlockDefinition struct {
	Name                        string
	BlockStateHash              uint64
	TranslationKey              string
	IsSolid                     bool
	Hardness                    float64
	ExplosionResistance         float64
	Friction                    float64
	RequiresCorrectToolForDrops bool
	LightEmission               int32
	SourceJSON                  string
}

func (c Catalog) Validate() error {
	if c.Version == "" {
		return fmt.Errorf("data: catalog version is empty")
	}
	if err := validateUnique("item", c.Items, func(item ItemDefinition) string { return item.Name }); err != nil {
		return err
	}
	seenBlocks := make(map[string]struct{}, len(c.Blocks))
	for _, block := range c.Blocks {
		if block.Name == "" {
			return fmt.Errorf("data: block has empty name")
		}
		key := fmt.Sprintf("%s#%d", block.Name, block.BlockStateHash)
		if _, ok := seenBlocks[key]; ok {
			return fmt.Errorf("data: duplicate block state %q", key)
		}
		seenBlocks[key] = struct{}{}
	}
	return nil
}

func validateUnique[T any](kind string, values []T, name func(T) string) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		key := name(value)
		if key == "" {
			return fmt.Errorf("data: %s has empty name", kind)
		}
		if _, ok := seen[key]; ok {
			return fmt.Errorf("data: duplicate %s %q", kind, key)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func (c Catalog) Item(name string) (ItemDefinition, bool) {
	for _, item := range c.Items {
		if item.Name == name {
			return item, true
		}
	}
	return ItemDefinition{}, false
}

func (c Catalog) Block(name string) (BlockDefinition, bool) {
	for _, block := range c.Blocks {
		if block.Name == name {
			return block, true
		}
	}
	return BlockDefinition{}, false
}

// BlockStates returns every authoritative state for a block name. Many
// Bedrock blocks have dozens or hundreds of state hashes, so a name-only
// registry would be incomplete.
func (c Catalog) BlockStates(name string) []BlockDefinition {
	states := make([]BlockDefinition, 0)
	for _, block := range c.Blocks {
		if block.Name == name {
			states = append(states, block)
		}
	}
	return states
}

// LoadCloudburst reads the JSON portions of a CloudburstMC/Data checkout.
// Binary palettes and entity data are intentionally separate inputs because
// they require their own versioned decoders; the manifest records only files
// consumed by this generator.
func LoadCloudburst(directory, version string) (Catalog, error) {
	if directory == "" {
		return Catalog{}, fmt.Errorf("data: cloudburst directory is empty")
	}
	itemsPath := filepath.Join(directory, "runtime_item_states.json")
	blocksPath := filepath.Join(directory, "block_properties.json")
	itemData, err := os.ReadFile(itemsPath)
	if err != nil {
		return Catalog{}, fmt.Errorf("data: read %s: %w", itemsPath, err)
	}
	blockData, err := os.ReadFile(blocksPath)
	if err != nil {
		return Catalog{}, fmt.Errorf("data: read %s: %w", blocksPath, err)
	}

	var rawItems []json.RawMessage
	if err := json.Unmarshal(itemData, &rawItems); err != nil {
		return Catalog{}, fmt.Errorf("data: decode %s: %w", itemsPath, err)
	}
	items := make([]ItemDefinition, 0, len(rawItems))
	for index, raw := range rawItems {
		var item struct {
			Name           string `json:"name"`
			ID             int32  `json:"id"`
			Version        int32  `json:"version"`
			ComponentBased bool   `json:"componentBased"`
		}
		if err := json.Unmarshal(raw, &item); err != nil {
			return Catalog{}, fmt.Errorf("data: decode item %d: %w", index, err)
		}
		items = append(items, ItemDefinition{
			Name: item.Name, RuntimeID: item.ID, Version: item.Version,
			ComponentBased: item.ComponentBased, SourceJSON: string(raw),
		})
	}

	var rawBlocks []json.RawMessage
	if err := json.Unmarshal(blockData, &rawBlocks); err != nil {
		return Catalog{}, fmt.Errorf("data: decode %s: %w", blocksPath, err)
	}
	blocks := make([]BlockDefinition, 0, len(rawBlocks))
	for index, raw := range rawBlocks {
		var block struct {
			Name                        string  `json:"name"`
			BlockStateHash              uint64  `json:"blockStateHash"`
			TranslationKey              string  `json:"translationKey"`
			IsSolid                     bool    `json:"isSolid"`
			Hardness                    float64 `json:"hardness"`
			ExplosionResistance         float64 `json:"explosionResistance"`
			Friction                    float64 `json:"friction"`
			RequiresCorrectToolForDrops bool    `json:"requiresCorrectToolForDrops"`
			LightEmission               int32   `json:"lightEmission"`
		}
		if err := json.Unmarshal(raw, &block); err != nil {
			return Catalog{}, fmt.Errorf("data: decode block %d: %w", index, err)
		}
		blocks = append(blocks, BlockDefinition{
			Name: block.Name, BlockStateHash: block.BlockStateHash,
			TranslationKey: block.TranslationKey, IsSolid: block.IsSolid,
			Hardness: block.Hardness, ExplosionResistance: block.ExplosionResistance,
			Friction: block.Friction, RequiresCorrectToolForDrops: block.RequiresCorrectToolForDrops,
			LightEmission: block.LightEmission, SourceJSON: string(raw),
		})
	}

	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	sort.Slice(blocks, func(i, j int) bool {
		if blocks[i].Name == blocks[j].Name {
			return blocks[i].BlockStateHash < blocks[j].BlockStateHash
		}
		return blocks[i].Name < blocks[j].Name
	})
	catalog := Catalog{
		Version: version,
		Source: SourceManifest{
			Name: "CloudburstMC/Data",
			Files: []SourceFile{
				{Name: "runtime_item_states.json", SHA256: sha256Hex(itemData)},
				{Name: "block_properties.json", SHA256: sha256Hex(blockData)},
			},
		},
		Items:  items,
		Blocks: blocks,
	}
	if err := catalog.Validate(); err != nil {
		return Catalog{}, err
	}
	return catalog, nil
}

func sha256Hex(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
