package main

import (
	"fmt"
	"go/format"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type entityDisplayDimension struct {
	JavaIdentifier    string
	BedrockIdentifier string
	Width             float32
	Height            float32
}

type entityDimensionOmission struct {
	JavaIdentifier string
	Reason         string
}

type javaEntityBuilder struct {
	parent     string
	width      float32
	height     float32
	widthSet   bool
	heightSet  bool
	entityType string
	register   bool
}

func generateEntityDimensionsFile(outPath, geyserPath, javaPath, geyserCommit string) (int, int, error) {
	geyserSource, err := os.ReadFile(geyserPath)
	if err != nil {
		return 0, 0, fmt.Errorf("read Geyser VanillaEntities.java: %w", err)
	}
	entities, err := readNamedIDs(javaPath, "entities")
	if err != nil {
		return 0, 0, err
	}
	entries, omissions, err := buildEntityDisplayDimensions(geyserSource, entities)
	if err != nil {
		return 0, 0, err
	}
	javaSource, err := os.ReadFile(javaPath)
	if err != nil {
		return 0, 0, fmt.Errorf("read Java entities source: %w", err)
	}
	output := renderEntityDisplayDimensions(entries, omissions, sha256Hex(javaSource), geyserCommit, sha256Hex(geyserSource))
	formatted, err := format.Source([]byte(output))
	if err != nil {
		return 0, 0, fmt.Errorf("format generated entity dimensions: %w", err)
	}
	if err := os.WriteFile(outPath, formatted, 0o644); err != nil {
		return 0, 0, err
	}
	return len(entries), len(omissions), nil
}

var (
	javaEntityTypeCall         = regexp.MustCompile(`\.type\s*\(\s*EntityType\.([A-Z0-9_]+)\s*\)`)
	javaEntityTypeHelperCall   = regexp.MustCompile(`(?:buildBoat|buildChestBoat)\s*\([^;]*?\bEntityType\.([A-Z0-9_]+)`)
	javaEntityParentCall       = regexp.MustCompile(`(?:inherited|baseInherited)\s*\([^,]+,\s*([A-Za-z_$][A-Za-z0-9_$]*)\s*\)`)
	javaEntityHelperParentCall = regexp.MustCompile(`(?:buildBoat|buildChestBoat)\s*\(\s*([A-Za-z_$][A-Za-z0-9_$]*)\s*,`)
	javaEntityHeightAndWidth   = regexp.MustCompile(`\.heightAndWidth\s*\(\s*([0-9.]+)f?\s*\)`)
	javaEntityHeight           = regexp.MustCompile(`\.height\s*\(\s*([0-9.]+)f?\s*\)`)
	javaEntityWidth            = regexp.MustCompile(`\.width\s*\(\s*([0-9.]+)f?\s*\)`)
)

// buildEntityDisplayDimensions resolves the builder inheritance used by
// Geyser's VanillaEntities.java. The input Java entity list is the versioned
// 1.21.4 registry; source definitions outside that registry are ignored.
func buildEntityDisplayDimensions(source []byte, entities []namedID) ([]entityDisplayDimension, []entityDimensionOmission, error) {
	builders := parseJavaEntityBuilders(string(source))
	if len(builders) == 0 {
		return nil, nil, fmt.Errorf("Geyser entity source contains no entity builders")
	}

	resolved := make(map[string][2]float32, len(builders))
	state := make(map[string]uint8, len(builders))
	var resolve func(string) ([2]float32, error)
	resolve = func(name string) ([2]float32, error) {
		if value, ok := resolved[name]; ok {
			return value, nil
		}
		if state[name] == 1 {
			return [2]float32{}, fmt.Errorf("cyclic entity builder inheritance at %q", name)
		}
		builder, ok := builders[name]
		if !ok {
			return [2]float32{}, fmt.Errorf("entity builder %q has no source definition", name)
		}
		state[name] = 1
		value := [2]float32{}
		if builder.parent != "" {
			parent, err := resolve(builder.parent)
			if err != nil {
				return [2]float32{}, err
			}
			value = parent
		}
		if builder.widthSet {
			value[0] = builder.width
		}
		if builder.heightSet {
			value[1] = builder.height
		}
		state[name] = 2
		resolved[name] = value
		return value, nil
	}

	entries := make([]entityDisplayDimension, 0, len(entities))
	omissions := make([]entityDimensionOmission, 0)
	for _, entity := range entities {
		javaIdentifier := entity.name
		bedrockIdentifier := normalizedBedrockEntityIdentifier(javaIdentifier)
		entityType := javaEntityTypeForIdentifier(javaIdentifier)
		builder, ok := builders[entityType]
		if !ok || builder.entityType == "" || !builder.register {
			omissions = append(omissions, entityDimensionOmission{
				JavaIdentifier: javaIdentifier,
				Reason:         "no network definition in Geyser VanillaEntities.java",
			})
			continue
		}
		value, err := resolve(entityType)
		if err != nil {
			return nil, nil, fmt.Errorf("resolve %s: %w", javaIdentifier, err)
		}
		entries = append(entries, entityDisplayDimension{
			JavaIdentifier:    javaIdentifier,
			BedrockIdentifier: bedrockIdentifier,
			Width:             value[0],
			Height:            value[1],
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].JavaIdentifier < entries[j].JavaIdentifier
	})
	sort.Slice(omissions, func(i, j int) bool {
		return omissions[i].JavaIdentifier < omissions[j].JavaIdentifier
	})
	return entries, omissions, nil
}

func parseJavaEntityBuilders(source string) map[string]javaEntityBuilder {
	builders := make(map[string]javaEntityBuilder)
	for _, statement := range javaAssignmentStatements(source) {
		if !strings.Contains(statement, "VanillaEntityType") &&
			!strings.Contains(statement, "EntityTypeBase") &&
			!strings.Contains(statement, "EntityDefinition") &&
			!strings.Contains(statement, "buildBoat") &&
			!strings.Contains(statement, "buildChestBoat") {
			continue
		}
		name := javaAssignmentName(statement)
		if name == "" {
			continue
		}
		builder := javaEntityBuilder{
			register: !strings.Contains(statement, ".build(false)"),
		}
		if match := javaEntityParentCall.FindStringSubmatch(statement); len(match) == 2 {
			builder.parent = match[1]
		} else if match := javaEntityHelperParentCall.FindStringSubmatch(statement); len(match) == 2 {
			builder.parent = match[1]
		}
		if match := javaEntityTypeCall.FindStringSubmatch(statement); len(match) == 2 {
			builder.entityType = match[1]
		} else if match := javaEntityTypeHelperCall.FindStringSubmatch(statement); len(match) == 2 {
			builder.entityType = match[1]
		}
		if match := javaEntityHeightAndWidth.FindAllStringSubmatch(statement, -1); len(match) > 0 {
			if value, err := parseJavaFloat(match[len(match)-1][1]); err == nil {
				builder.width = value
				builder.height = value
				builder.widthSet = true
				builder.heightSet = true
			}
		}
		if match := javaEntityHeight.FindAllStringSubmatch(statement, -1); len(match) > 0 {
			if value, err := parseJavaFloat(match[len(match)-1][1]); err == nil {
				builder.height = value
				builder.heightSet = true
			}
		}
		if match := javaEntityWidth.FindAllStringSubmatch(statement, -1); len(match) > 0 {
			if value, err := parseJavaFloat(match[len(match)-1][1]); err == nil {
				builder.width = value
				builder.widthSet = true
			}
		}
		builders[name] = builder
	}
	return builders
}

func javaAssignmentStatements(source string) []string {
	var statements []string
	var current strings.Builder
	inStatement := false
	for _, line := range strings.Split(source, "\n") {
		if !inStatement {
			if !javaAssignmentLine(line) {
				continue
			}
			inStatement = true
		}
		current.WriteString(line)
		current.WriteByte('\n')
		if strings.Contains(line, ";") {
			statements = append(statements, current.String())
			current.Reset()
			inStatement = false
		}
	}
	return statements
}

func javaAssignmentLine(line string) bool {
	equal := strings.IndexByte(line, '=')
	if equal < 0 {
		return false
	}
	left := strings.TrimSpace(line[:equal])
	if left == "" || strings.HasSuffix(left, ")") || strings.HasSuffix(left, "{") {
		return false
	}
	fields := strings.Fields(left)
	if len(fields) == 0 {
		return false
	}
	name := fields[len(fields)-1]
	if strings.ContainsAny(name, "(){}[]") {
		return false
	}
	return name != "if" && name != "for" && name != "while" && name != "return"
}

func javaAssignmentName(statement string) string {
	line := strings.SplitN(statement, "\n", 2)[0]
	equal := strings.IndexByte(line, '=')
	if equal < 0 {
		return ""
	}
	fields := strings.Fields(strings.TrimSpace(line[:equal]))
	if len(fields) == 0 {
		return ""
	}
	return fields[len(fields)-1]
}

func parseJavaFloat(value string) (float32, error) {
	parsed, err := strconv.ParseFloat(value, 32)
	return float32(parsed), err
}

func javaEntityTypeForIdentifier(identifier string) string {
	name := strings.TrimPrefix(identifier, "minecraft:")
	if name == "potion" {
		return "SPLASH_POTION"
	}
	return strings.ToUpper(name)
}

func normalizedBedrockEntityIdentifier(identifier string) string {
	name := strings.TrimPrefix(identifier, "minecraft:")
	if alias, ok := bedrockEntityAliases[name]; ok {
		name = alias
	}
	return "minecraft:" + name
}

func renderEntityDisplayDimensions(entries []entityDisplayDimension, omissions []entityDimensionOmission, javaHash, geyserCommit, geyserHash string) string {
	var builder strings.Builder
	builder.WriteString("// Code generated by cmd/registrygen; DO NOT EDIT.\n")
	builder.WriteString("// Java entity source: minecraft-data entities.json for Java 1.21.4 (protocol 769).\n")
	fmt.Fprintf(&builder, "// Java entity source SHA-256: %s\n", javaHash)
	builder.WriteString("// Geyser source repository: https://github.com/GeyserMC/Geyser\n")
	fmt.Fprintf(&builder, "// Geyser source commit: %s\n", geyserCommit)
	fmt.Fprintf(&builder, "// Geyser VanillaEntities.java source SHA-256: %s\n", geyserHash)
	fmt.Fprintf(&builder, "// Generated entries: %d; explicit network-definition omissions: %d.\n\n", len(entries), len(omissions))
	builder.WriteString("package data\n\nimport \"strings\"\n\n")
	builder.WriteString("// EntityDisplayDimensions contains the Java hitbox dimensions Geyser uses when\n")
	builder.WriteString("// projecting a Bedrock mob-spawner display actor.\n")
	builder.WriteString("type EntityDisplayDimensions struct {\n\tJavaIdentifier string\n\tBedrockIdentifier string\n\tWidth float32\n\tHeight float32\n}\n\n")
	builder.WriteString("var Java1214EntityDisplayDimensions = []EntityDisplayDimensions{\n")
	for _, entry := range entries {
		fmt.Fprintf(&builder, "\t{JavaIdentifier: %q, BedrockIdentifier: %q, Width: %s, Height: %s},\n",
			entry.JavaIdentifier, entry.BedrockIdentifier, formatEntityFloat(entry.Width), formatEntityFloat(entry.Height))
	}
	builder.WriteString("}\n\n")
	builder.WriteString("// Java1214EntityDisplayDimensionOmissions records Java identifiers in the\n")
	builder.WriteString("// versioned registry for which Geyser has no network entity definition.\n")
	builder.WriteString("var Java1214EntityDisplayDimensionOmissions = []string{\n")
	for _, omission := range omissions {
		fmt.Fprintf(&builder, "\t%q, // %s\n", omission.JavaIdentifier, omission.Reason)
	}
	builder.WriteString("}\n\n")
	builder.WriteString("// BedrockEntityDisplayDimensions resolves a Java identifier and its normalized\n")
	builder.WriteString("// Bedrock alias to Geyser's versioned display dimensions.\n")
	builder.WriteString("func BedrockEntityDisplayDimensions(javaIdentifier, bedrockIdentifier string) (EntityDisplayDimensions, bool) {\n")
	builder.WriteString("\tjavaIdentifier = namespacedEntityIdentifier(javaIdentifier)\n")
	builder.WriteString("\tbedrockIdentifier = namespacedEntityIdentifier(bedrockIdentifier)\n")
	builder.WriteString("\tfor _, dimensions := range Java1214EntityDisplayDimensions {\n")
	builder.WriteString("\t\tif dimensions.JavaIdentifier == javaIdentifier || dimensions.BedrockIdentifier == bedrockIdentifier {\n")
	builder.WriteString("\t\t\treturn dimensions, true\n")
	builder.WriteString("\t\t}\n")
	builder.WriteString("\t}\n")
	builder.WriteString("\treturn EntityDisplayDimensions{}, false\n")
	builder.WriteString("}\n\n")
	builder.WriteString("func namespacedEntityIdentifier(identifier string) string {\n")
	builder.WriteString("\tif identifier != \"\" && !strings.Contains(identifier, \":\") {\n")
	builder.WriteString("\t\treturn \"minecraft:\" + identifier\n")
	builder.WriteString("\t}\n")
	builder.WriteString("\treturn identifier\n")
	builder.WriteString("}\n")
	return builder.String()
}

func formatEntityFloat(value float32) string {
	return strconv.FormatFloat(float64(value), 'f', -1, 32)
}
