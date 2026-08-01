// Command registrygen turns authoritative Java block-state IDs and the
// CloudburstMC Bedrock palette into a small, deterministic Go lookup table.
// The input payloads stay outside the repository; only the generated mapping
// and its provenance are committed.
package main

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

type javaState struct {
	id  int
	key string
}

type namedID struct {
	id   int
	name string
}

type blockMappingTarget struct {
	name   string
	states map[string]any
}

type itemMappingResult struct {
	mapping     []int32
	exact       int
	airFallback int
	javaHash    string
	bedrockHash string
	geyserHash  string
}

func main() {
	javaPath := flag.String("java-tsv", "", "TSV containing Java state ID and canonical state key")
	palettePath := flag.String("bedrock-palette", "", "gzip-compressed Bedrock block palette NBT")
	geyserPath := flag.String("geyser-mappings", "", "Geyser mappings blocks.nbt for the same Java protocol")
	javaItemsPath := flag.String("java-items", "", "minecraft-data items.json for the same Java protocol")
	bedrockItemsPath := flag.String("bedrock-items", "", "Cloudburst runtime_item_states.json")
	geyserItemsPath := flag.String("geyser-items", "", "Geyser mappings items.json for the same Java protocol")
	javaEntitiesPath := flag.String("java-entities", "", "minecraft-data entities.json for the same Java protocol")
	outPath := flag.String("out", "", "generated Go file")
	flag.Parse()
	if *javaPath == "" || *palettePath == "" || *outPath == "" {
		panic("usage: registrygen -java-tsv path -bedrock-palette path -out path")
	}

	java, err := readJavaStates(*javaPath)
	if err != nil {
		panic(err)
	}
	paletteBytes, err := readGzip(*palettePath)
	if err != nil {
		panic(err)
	}
	var root map[string]any
	if err := nbt.UnmarshalEncoding(paletteBytes, &root, nbt.BigEndian); err != nil {
		panic(fmt.Errorf("decode Bedrock palette: %w", err))
	}
	rawBlocks, ok := root["blocks"].([]any)
	if !ok || len(rawBlocks) == 0 {
		panic("Bedrock palette has no blocks list")
	}
	desiredKeys := make([]string, len(java))
	for i, state := range java {
		desiredKeys[i] = state.key
	}
	mappedTargets := make([]blockMappingTarget, len(java))
	hasGeyserMappings := false
	var geyserHash string
	if *geyserPath != "" {
		geyserBytes, err := readMaybeGzip(*geyserPath)
		if err != nil {
			panic(err)
		}
		var geyserRoot map[string]any
		if err := nbt.UnmarshalEncoding(geyserBytes, &geyserRoot, nbt.BigEndian); err != nil {
			panic(fmt.Errorf("decode Geyser block mappings: %w", err))
		}
		rawMappings, ok := geyserRoot["bedrock_mappings"].([]any)
		if !ok || len(rawMappings) < len(java) {
			panic(fmt.Sprintf("Geyser block mappings have %d entries, need %d", len(rawMappings), len(java)))
		}
		hasGeyserMappings = true
		for i, state := range java {
			entry, ok := rawMappings[i].(map[string]any)
			if !ok {
				panic(fmt.Sprintf("Geyser block mapping %d has type %T", i, rawMappings[i]))
			}
			name := state.key
			if bracket := strings.IndexByte(name, '['); bracket >= 0 {
				name = name[:bracket]
			}
			if override, _ := entry["bedrock_identifier"].(string); override != "" && override != "unknown" {
				name = override
			}
			states, _ := entry["state"].(map[string]any)
			mappedTargets[i] = blockMappingTarget{name: name, states: states}
		}
		geyserSource, _ := os.ReadFile(*geyserPath)
		geyserHash = sha256Hex(geyserSource)
	}

	exact := make(map[string]uint32, len(rawBlocks))
	byName := make(map[string][]uint32)
	for runtimeID, raw := range rawBlocks {
		block, ok := raw.(map[string]any)
		if !ok {
			panic(fmt.Sprintf("Bedrock palette block %d has type %T", runtimeID, raw))
		}
		name, _ := block["name"].(string)
		if name == "" {
			panic(fmt.Sprintf("Bedrock palette block %d has no name", runtimeID))
		}
		states, _ := block["states"].(map[string]any)
		key := stateKey(name, states)
		exact[key] = uint32(runtimeID)
		byName[name] = append(byName[name], uint32(runtimeID))
	}

	normalized := 0
	if hasGeyserMappings {
		for _, state := range java {
			target := mappedTargets[state.id]
			if normalizeBlockMapping(state.key, &target, byName) {
				normalized++
			}
			desiredKeys[state.id] = stateKey(target.name, target.states)
		}
	}

	air, ok := exact["minecraft:air"]
	if !ok {
		panic("Bedrock palette has no minecraft:air state")
	}
	mapping := make([]uint32, len(java))
	var exactCount, nameFallback, airFallback int
	for _, state := range java {
		if id, ok := exact[desiredKeys[state.id]]; ok {
			mapping[state.id] = id
			exactCount++
			continue
		}
		name := desiredKeys[state.id]
		if bracket := strings.IndexByte(name, '['); bracket >= 0 {
			name = name[:bracket]
		}
		if ids := byName[name]; len(ids) > 0 {
			mapping[state.id] = ids[0]
			nameFallback++
			continue
		}
		mapping[state.id] = air
		airFallback++
	}

	if mapping[0] != air {
		panic(fmt.Sprintf("Java air mapped to Bedrock runtime %d, expected %d", mapping[0], air))
	}

	var itemResult itemMappingResult
	if *javaItemsPath != "" || *bedrockItemsPath != "" || *geyserItemsPath != "" {
		if *javaItemsPath == "" || *bedrockItemsPath == "" {
			panic("item mapping requires -java-items and -bedrock-items together")
		}
		itemResult, err = buildItemMapping(*javaItemsPath, *bedrockItemsPath, *geyserItemsPath)
		if err != nil {
			panic(err)
		}
	}

	var entityNames []string
	var entityHash string
	if *javaEntitiesPath != "" {
		entityNames, err = readEntityNames(*javaEntitiesPath)
		if err != nil {
			panic(err)
		}
		entitySource, readErr := os.ReadFile(*javaEntitiesPath)
		if readErr != nil {
			panic(fmt.Errorf("read Java entities source: %w", readErr))
		}
		entityHash = sha256Hex(entitySource)
	}

	sourceJava, _ := os.ReadFile(*javaPath)
	sourcePalette, _ := os.ReadFile(*palettePath)
	output := render(mapping, len(rawBlocks), exactCount, normalized, nameFallback, airFallback,
		sha256Hex(sourceJava), sha256Hex(sourcePalette), geyserHash, itemResult, entityNames, entityHash)
	formatted, err := format.Source([]byte(output))
	if err != nil {
		panic(fmt.Errorf("format generated Go: %w", err))
	}
	if err := os.WriteFile(*outPath, formatted, 0o644); err != nil {
		panic(err)
	}
	fmt.Printf("java states=%d bedrock states=%d exact=%d normalized=%d name-fallback=%d air-fallback=%d", len(java), len(rawBlocks), exactCount, normalized, nameFallback, airFallback)
	if len(itemResult.mapping) > 0 {
		fmt.Printf(" java items=%d item-exact=%d item-air-fallback=%d", len(itemResult.mapping), itemResult.exact, itemResult.airFallback)
	}
	if len(entityNames) > 0 {
		fmt.Printf(" java entities=%d", len(entityNames))
	}
	fmt.Println()
}

func readJavaStates(path string) ([]javaState, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read Java states: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) < 2 || !strings.HasPrefix(lines[0], "size=") {
		return nil, fmt.Errorf("Java TSV has no size header")
	}
	size, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(lines[0], "size=")))
	if err != nil || size <= 0 {
		return nil, fmt.Errorf("invalid Java state size %q", lines[0])
	}
	states := make([]javaState, size)
	seen := make([]bool, size)
	for _, line := range lines[1:] {
		fields := strings.SplitN(line, "\t", 2)
		if len(fields) != 2 {
			continue
		}
		id, err := strconv.Atoi(strings.TrimSpace(fields[0]))
		if err != nil || id < 0 || id >= size {
			return nil, fmt.Errorf("invalid Java state line %q", line)
		}
		states[id] = javaState{id: id, key: strings.TrimSpace(fields[1])}
		seen[id] = true
	}
	for id := range states {
		if !seen[id] || states[id].key == "" {
			return nil, fmt.Errorf("Java state %d is missing", id)
		}
	}
	return states, nil
}

func readNamedIDs(path, kind string) ([]namedID, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read Java %s: %w", kind, err)
	}
	var raw []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("decode Java %s: %w", kind, err)
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("Java %s is empty", kind)
	}
	maxID := -1
	for _, value := range raw {
		if value.ID < 0 || value.Name == "" {
			return nil, fmt.Errorf("invalid Java %s entry id=%d name=%q", kind, value.ID, value.Name)
		}
		if value.ID > maxID {
			maxID = value.ID
		}
	}
	values := make([]namedID, maxID+1)
	seen := make([]bool, maxID+1)
	for _, value := range raw {
		if seen[value.ID] {
			return nil, fmt.Errorf("duplicate Java %s id %d", kind, value.ID)
		}
		values[value.ID] = namedID{id: value.ID, name: value.Name}
		seen[value.ID] = true
	}
	for id, value := range values {
		if !seen[id] {
			return nil, fmt.Errorf("Java %s id %d is missing", kind, id)
		}
		if !strings.Contains(value.name, ":") {
			values[id].name = "minecraft:" + value.name
		}
	}
	return values, nil
}

func readEntityNames(path string) ([]string, error) {
	values, err := readNamedIDs(path, "entities")
	if err != nil {
		return nil, err
	}
	names := make([]string, len(values))
	for _, value := range values {
		name := strings.TrimPrefix(value.name, "minecraft:")
		if alias, ok := bedrockEntityAliases[name]; ok {
			name = alias
		}
		names[value.id] = "minecraft:" + name
	}
	return names, nil
}

var bedrockEntityAliases = map[string]string{
	"end_crystal":        "ender_crystal",
	"evoker_fangs":       "evocation_fang",
	"experience_bottle":  "xp_bottle",
	"experience_orb":     "xp_orb",
	"eye_of_ender":       "eye_of_ender_signal",
	"firework_rocket":    "fireworks_rocket",
	"fishing_bobber":     "fishing_hook",
	"tropical_fish":      "tropicalfish",
	"villager":           "villager_v2",
	"wind_charge":        "wind_charge_projectile",
	"breeze_wind_charge": "breeze_wind_charge_projectile",
	"zombie_villager":    "zombie_villager_v2",
	"zombified_piglin":   "zombie_pigman",
}

func buildItemMapping(javaPath, bedrockPath, geyserPath string) (itemMappingResult, error) {
	java, err := readNamedIDs(javaPath, "items")
	if err != nil {
		return itemMappingResult{}, err
	}
	bedrockBytes, err := os.ReadFile(bedrockPath)
	if err != nil {
		return itemMappingResult{}, fmt.Errorf("read Bedrock items: %w", err)
	}
	var rawBedrock []struct {
		Name string `json:"name"`
		ID   int32  `json:"id"`
	}
	if err := json.Unmarshal(bedrockBytes, &rawBedrock); err != nil {
		return itemMappingResult{}, fmt.Errorf("decode Bedrock items: %w", err)
	}
	bedrock := make(map[string]int32, len(rawBedrock))
	for _, item := range rawBedrock {
		if item.Name == "" {
			return itemMappingResult{}, fmt.Errorf("invalid Bedrock item id=%d name=%q", item.ID, item.Name)
		}
		bedrock[item.Name] = item.ID
	}
	air, ok := bedrock["minecraft:air"]
	if !ok {
		return itemMappingResult{}, fmt.Errorf("Bedrock item table has no minecraft:air")
	}
	aliases := make(map[string]string)
	var geyserHash string
	if geyserPath != "" {
		geyserBytes, readErr := os.ReadFile(geyserPath)
		if readErr != nil {
			return itemMappingResult{}, fmt.Errorf("read Geyser items: %w", readErr)
		}
		var raw map[string]struct {
			BedrockIdentifier string `json:"bedrock_identifier"`
		}
		if err := json.Unmarshal(geyserBytes, &raw); err != nil {
			return itemMappingResult{}, fmt.Errorf("decode Geyser items: %w", err)
		}
		for name, item := range raw {
			if item.BedrockIdentifier != "" && item.BedrockIdentifier != "unknown" {
				aliases[name] = item.BedrockIdentifier
			}
		}
		geyserHash = sha256Hex(geyserBytes)
	}
	mapping := make([]int32, len(java))
	exact := 0
	fallback := 0
	for _, item := range java {
		name := item.name
		if alias := aliases[name]; alias != "" {
			name = alias
		}
		if id, found := bedrock[name]; found {
			mapping[item.id] = id
			exact++
		} else {
			mapping[item.id] = air
			fallback++
		}
	}
	javaBytes, _ := os.ReadFile(javaPath)
	return itemMappingResult{
		mapping: mapping, exact: exact, airFallback: fallback,
		javaHash: sha256Hex(javaBytes), bedrockHash: sha256Hex(bedrockBytes), geyserHash: geyserHash,
	}, nil
}

func readGzip(path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read palette: %w", err)
	}
	return gunzip(b, path)
}

func readMaybeGzip(path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if len(b) >= 2 && b[0] == 0x1f && b[1] == 0x8b {
		return gunzip(b, path)
	}
	return b, nil
}

func gunzip(b []byte, path string) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("open %s gzip: %w", path, err)
	}
	decoded, readErr := io.ReadAll(r)
	closeErr := r.Close()
	if readErr != nil {
		return nil, fmt.Errorf("read %s gzip: %w", path, readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close %s gzip: %w", path, closeErr)
	}
	return decoded, nil
}

func normalizeBlockMapping(javaKey string, target *blockMappingTarget, palette map[string][]uint32) bool {
	javaName := javaKey
	if bracket := strings.IndexByte(javaName, '['); bracket >= 0 {
		javaName = javaName[:bracket]
	}
	changed := false
	setNameIfPresent := func(name string) {
		if len(palette[name]) == 0 || target.name == name {
			return
		}
		target.name = name
		changed = true
	}

	switch javaName {
	case "minecraft:chain":
		// Cloudburst's complete Bedrock palette calls the Java chain block
		// iron_chain. The item crosswalk in the same source uses this alias.
		setNameIfPresent("minecraft:iron_chain")
	case "minecraft:pale_oak_sign":
		// Java's floor sign stores rotation and waterlogged; Bedrock stores
		// the floor variant under its explicit standing-sign name and drops
		// waterlogged from the runtime state.
		if len(palette["minecraft:pale_oak_standing_sign"]) != 0 {
			if rotation, ok := javaStateProperty(javaKey, "rotation"); ok {
				target.name = "minecraft:pale_oak_standing_sign"
				target.states = map[string]any{"ground_sign_direction": int32(rotation)}
				changed = true
			}
		}
	case "minecraft:skeleton_skull", "minecraft:skeleton_wall_skull":
		// Java has separate wall block names; Bedrock uses one typed
		// skeleton_skull palette with facing_direction 1..5.
		setNameIfPresent("minecraft:skeleton_skull")
	case "minecraft:wither_skeleton_skull", "minecraft:wither_skeleton_wall_skull":
		setNameIfPresent("minecraft:wither_skeleton_skull")
	}
	return changed
}

func javaStateProperty(key, property string) (int, bool) {
	start := strings.IndexByte(key, '[')
	end := strings.LastIndexByte(key, ']')
	if start < 0 || end <= start {
		return 0, false
	}
	for _, field := range strings.Split(key[start+1:end], ",") {
		parts := strings.SplitN(field, "=", 2)
		if len(parts) != 2 || parts[0] != property {
			continue
		}
		value, err := strconv.Atoi(parts[1])
		return value, err == nil
	}
	return 0, false
}

func stateKey(name string, states map[string]any) string {
	if !strings.Contains(name, ":") {
		name = "minecraft:" + name
	}
	if len(states) == 0 {
		return name
	}
	keys := make([]string, 0, len(states))
	for key := range states {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var builder strings.Builder
	builder.WriteString(name)
	builder.WriteByte('[')
	for i, key := range keys {
		if i != 0 {
			builder.WriteByte(',')
		}
		builder.WriteString(key)
		builder.WriteByte('=')
		builder.WriteString(nbtValue(states[key]))
	}
	builder.WriteByte(']')
	return builder.String()
}

func nbtValue(value any) string {
	switch value := value.(type) {
	case string:
		return value
	case bool:
		return strconv.FormatBool(value)
	case int8:
		return strconv.FormatInt(int64(value), 10)
	case uint8:
		return strconv.FormatUint(uint64(value), 10)
	case int16:
		return strconv.FormatInt(int64(value), 10)
	case uint16:
		return strconv.FormatUint(uint64(value), 10)
	case int32:
		return strconv.FormatInt(int64(value), 10)
	case uint32:
		return strconv.FormatUint(uint64(value), 10)
	case int64:
		return strconv.FormatInt(value, 10)
	case uint64:
		return strconv.FormatUint(value, 10)
	default:
		return fmt.Sprint(value)
	}
}

func render(mapping []uint32, paletteCount, exact, normalized, nameFallback, airFallback int, javaHash, paletteHash, geyserHash string, items itemMappingResult, entities []string, entityHash string) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "// Code generated by cmd/registrygen; DO NOT EDIT.\n")
	fmt.Fprintf(&builder, "// Java state source SHA-256: %s\n", javaHash)
	fmt.Fprintf(&builder, "// CloudburstMC/Data block palette source SHA-256: %s\n", paletteHash)
	if geyserHash != "" {
		fmt.Fprintf(&builder, "// Geyser block mapping source SHA-256: %s\n", geyserHash)
	}
	fmt.Fprintf(&builder, "// Mapping coverage: exact=%d normalized=%d name-fallback=%d air-fallback=%d; Bedrock palette states=%d.\n\n", exact, normalized, nameFallback, airFallback, paletteCount)
	if len(items.mapping) > 0 {
		fmt.Fprintf(&builder, "// Java item source SHA-256: %s\n", items.javaHash)
		fmt.Fprintf(&builder, "// CloudburstMC/Data item source SHA-256: %s\n", items.bedrockHash)
		if items.geyserHash != "" {
			fmt.Fprintf(&builder, "// Geyser item mapping source SHA-256: %s\n", items.geyserHash)
		}
		fmt.Fprintf(&builder, "// Item coverage: exact=%d air-fallback=%d.\n", items.exact, items.airFallback)
	}
	if len(entities) > 0 {
		fmt.Fprintf(&builder, "// Java entity source SHA-256: %s\n", entityHash)
		builder.WriteString("// Entity identifiers use Geyser's versioned Bedrock aliases where Java and Bedrock names differ.\n")
	}
	builder.WriteByte('\n')
	builder.WriteString("package data\n\n")
	fmt.Fprintf(&builder, "const Java1214BlockStateCount = %d\n\n", len(mapping))
	builder.WriteString("var Java1214ToBedrock = [...]uint32{\n")
	for i, id := range mapping {
		if i%12 == 0 {
			builder.WriteString("\t")
		}
		fmt.Fprintf(&builder, "%d, ", id)
		if i%12 == 11 {
			builder.WriteByte('\n')
		}
	}
	if len(mapping)%12 != 0 {
		builder.WriteByte('\n')
	}
	builder.WriteString("}\n")
	if len(items.mapping) > 0 {
		builder.WriteString("\n")
		fmt.Fprintf(&builder, "const Java1214ItemCount = %d\n\n", len(items.mapping))
		builder.WriteString("var Java1214ToBedrockItem = [...]int32{\n")
		for i, id := range items.mapping {
			if i%16 == 0 {
				builder.WriteString("\t")
			}
			fmt.Fprintf(&builder, "%d, ", id)
			if i%16 == 15 {
				builder.WriteByte('\n')
			}
		}
		if len(items.mapping)%16 != 0 {
			builder.WriteByte('\n')
		}
		builder.WriteString("}\n")
	}
	if len(entities) > 0 {
		builder.WriteString("\nvar Java1214EntityTypeNames = [...]string{\n")
		for i, name := range entities {
			if i%4 == 0 {
				builder.WriteString("\t")
			}
			fmt.Fprintf(&builder, "%q, ", name)
			if i%4 == 3 {
				builder.WriteByte('\n')
			}
		}
		if len(entities)%4 != 0 {
			builder.WriteByte('\n')
		}
		builder.WriteString("}\n")
	}
	return builder.String()
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum[:])
}
