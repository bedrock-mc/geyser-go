// Command registrygen turns authoritative Java block-state IDs and the
// CloudburstMC Bedrock palette into a small, deterministic Go lookup table.
// The input payloads stay outside the repository; only the generated mapping
// and its provenance are committed.
package main

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
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

func main() {
	javaPath := flag.String("java-tsv", "", "TSV containing Java state ID and canonical state key")
	palettePath := flag.String("bedrock-palette", "", "gzip-compressed Bedrock block palette NBT")
	geyserPath := flag.String("geyser-mappings", "", "Geyser mappings blocks.nbt for the same Java protocol")
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
			desiredKeys[i] = stateKey(name, states)
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
	sourceJava, _ := os.ReadFile(*javaPath)
	sourcePalette, _ := os.ReadFile(*palettePath)
	output := render(mapping, len(rawBlocks), exactCount, nameFallback, airFallback,
		sha256Hex(sourceJava), sha256Hex(sourcePalette), geyserHash)
	formatted, err := format.Source([]byte(output))
	if err != nil {
		panic(fmt.Errorf("format generated Go: %w", err))
	}
	if err := os.WriteFile(*outPath, formatted, 0o644); err != nil {
		panic(err)
	}
	fmt.Printf("java states=%d bedrock states=%d exact=%d name-fallback=%d air-fallback=%d\n", len(java), len(rawBlocks), exactCount, nameFallback, airFallback)
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

func render(mapping []uint32, paletteCount, exact, nameFallback, airFallback int, javaHash, paletteHash, geyserHash string) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "// Code generated by cmd/registrygen; DO NOT EDIT.\n")
	fmt.Fprintf(&builder, "// Java state source SHA-256: %s\n", javaHash)
	fmt.Fprintf(&builder, "// CloudburstMC/Data block palette source SHA-256: %s\n", paletteHash)
	if geyserHash != "" {
		fmt.Fprintf(&builder, "// Geyser block mapping source SHA-256: %s\n", geyserHash)
	}
	fmt.Fprintf(&builder, "// Mapping coverage: exact=%d name-fallback=%d air-fallback=%d; Bedrock palette states=%d.\n\n", exact, nameFallback, airFallback, paletteCount)
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
	return builder.String()
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum[:])
}
