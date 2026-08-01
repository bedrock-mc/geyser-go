package translate

import (
	"fmt"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	"github.com/sandertv/gophertunnel/minecraft"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

const (
	javaCommandNodeRoot     = 0
	javaCommandNodeLiteral  = 1
	javaCommandNodeArgument = 2

	javaCommandFlagExecutable  = 1 << 2
	javaCommandFlagRedirect    = 1 << 3
	javaCommandFlagSuggestions = 1 << 4

	javaCommandMaxOverloads        = 256
	javaCommandMaxPathDepth        = 32
	javaCommandNoAliases    uint32 = 0xffffffff
)

// JavaCommandNode is the bounded wire representation of a Brigadier node in
// ClientboundCommandsPacket. Parser properties are consumed while decoding so
// an unknown parser can still be treated as a semantic gap without shifting
// the following nodes.
type JavaCommandNode struct {
	Flags       byte
	Children    []int32
	Redirect    int32
	Name        string
	Parser      string
	Suggestions string
}

type JavaCommands struct {
	Nodes     []JavaCommandNode
	RootIndex int32
}

var javaCommandParserNames = []string{
	"brigadier:bool",
	"brigadier:float",
	"brigadier:double",
	"brigadier:integer",
	"brigadier:long",
	"brigadier:string",
	"minecraft:entity",
	"minecraft:game_profile",
	"minecraft:block_pos",
	"minecraft:column_pos",
	"minecraft:vec3",
	"minecraft:vec2",
	"minecraft:block_state",
	"minecraft:block_predicate",
	"minecraft:item_stack",
	"minecraft:item_predicate",
	"minecraft:color",
	"minecraft:component",
	"minecraft:style",
	"minecraft:message",
	"minecraft:nbt",
	"minecraft:nbt_tag",
	"minecraft:nbt_path",
	"minecraft:objective",
	"minecraft:objective_criteria",
	"minecraft:operation",
	"minecraft:particle",
	"minecraft:angle",
	"minecraft:rotation",
	"minecraft:scoreboard_slot",
	"minecraft:score_holder",
	"minecraft:swizzle",
	"minecraft:team",
	"minecraft:item_slot",
	"minecraft:item_slots",
	"minecraft:resource_location",
	"minecraft:function",
	"minecraft:entity_anchor",
	"minecraft:int_range",
	"minecraft:float_range",
	"minecraft:dimension",
	"minecraft:gamemode",
	"minecraft:time",
	"minecraft:resource_or_tag",
	"minecraft:resource_or_tag_key",
	"minecraft:resource",
	"minecraft:resource_key",
	"minecraft:template_mirror",
	"minecraft:template_rotation",
	"minecraft:heightmap",
	"minecraft:loot_table",
	"minecraft:loot_predicate",
	"minecraft:loot_modifier",
	"minecraft:uuid",
}

func DecodeJavaCommands(data []byte) (JavaCommands, error) {
	r := javaprotocol.NewReader(data)
	count, err := boundedJavaCount(r, "command node count")
	if err != nil {
		return JavaCommands{}, err
	}
	result := JavaCommands{Nodes: make([]JavaCommandNode, count)}
	for i := range result.Nodes {
		flags, readErr := r.Byte()
		if readErr != nil {
			return JavaCommands{}, fmt.Errorf("translate: command node %d flags: %w", i, readErr)
		}
		children, readErr := boundedJavaCount(r, fmt.Sprintf("command node %d child count", i))
		if readErr != nil {
			return JavaCommands{}, readErr
		}
		node := JavaCommandNode{Flags: flags, Redirect: -1}
		node.Children = make([]int32, children)
		for child := range node.Children {
			node.Children[child], readErr = r.VarInt()
			if readErr != nil {
				return JavaCommands{}, fmt.Errorf("translate: command node %d child %d: %w", i, child, readErr)
			}
		}
		if flags&javaCommandFlagRedirect != 0 {
			node.Redirect, readErr = r.VarInt()
			if readErr != nil {
				return JavaCommands{}, fmt.Errorf("translate: command node %d redirect: %w", i, readErr)
			}
		}

		switch flags & 0x03 {
		case javaCommandNodeLiteral:
			node.Name, readErr = r.String()
		case javaCommandNodeArgument:
			node.Name, readErr = r.String()
			if readErr == nil {
				parserID, parserErr := r.VarInt()
				if parserErr != nil {
					readErr = parserErr
				} else {
					node.Parser = javaCommandParserName(parserID)
					readErr = skipJavaCommandProperties(r, node.Parser)
				}
			}
		}
		if readErr != nil {
			return JavaCommands{}, fmt.Errorf("translate: command node %d data: %w", i, readErr)
		}
		if flags&javaCommandFlagSuggestions != 0 {
			node.Suggestions, readErr = r.String()
			if readErr != nil {
				return JavaCommands{}, fmt.Errorf("translate: command node %d suggestions: %w", i, readErr)
			}
		}
		result.Nodes[i] = node
	}
	result.RootIndex, err = r.VarInt()
	if err != nil {
		return JavaCommands{}, fmt.Errorf("translate: command root index: %w", err)
	}
	if r.Remaining() != 0 {
		return JavaCommands{}, fmt.Errorf("translate: commands has %d trailing bytes", r.Remaining())
	}
	return result, nil
}

func javaCommandParserName(id int32) string {
	if id >= 0 && id < int32(len(javaCommandParserNames)) {
		return javaCommandParserNames[id]
	}
	return fmt.Sprintf("minecraft:unknown/%d", id)
}

func skipJavaCommandProperties(r *javaprotocol.Reader, parser string) error {
	switch parser {
	case "brigadier:float", "brigadier:integer":
		return skipJavaCommandNumberProperties(r, 4)
	case "brigadier:double", "brigadier:long":
		return skipJavaCommandNumberProperties(r, 8)
	case "brigadier:string":
		_, err := r.VarInt()
		return err
	case "minecraft:entity", "minecraft:score_holder":
		_, err := r.Byte()
		return err
	case "minecraft:time":
		_, err := r.Int32()
		return err
	case "minecraft:resource_or_tag", "minecraft:resource_or_tag_key", "minecraft:resource", "minecraft:resource_key":
		_, err := r.String()
		return err
	default:
		return nil
	}
}

func skipJavaCommandNumberProperties(r *javaprotocol.Reader, width int) error {
	flags, err := r.Byte()
	if err != nil {
		return err
	}
	if flags&0x01 != 0 {
		if err := skipJavaCommandNumber(r, width); err != nil {
			return err
		}
	}
	if flags&0x02 != 0 {
		if err := skipJavaCommandNumber(r, width); err != nil {
			return err
		}
	}
	return nil
}

func skipJavaCommandNumber(r *javaprotocol.Reader, width int) error {
	switch width {
	case 4:
		_, err := r.Int32()
		return err
	case 8:
		_, err := r.Int64()
		return err
	default:
		return fmt.Errorf("translate: unsupported command number width %d", width)
	}
}

func (b *Basic) translateJavaCommands(bedrock *minecraft.Conn, data []byte) error {
	commands, err := DecodeJavaCommands(data)
	if err != nil {
		return err
	}
	available, skipped := buildBedrockCommands(commands)
	if skipped != 0 {
		b.logSemanticAnomaly("skipped Java command tree nodes", "count", skipped)
	}
	return bedrock.WritePacket(available)
}

func buildBedrockCommands(java JavaCommands) (*packet.AvailableCommands, int) {
	available := &packet.AvailableCommands{}
	if java.RootIndex < 0 || java.RootIndex >= int32(len(java.Nodes)) {
		return available, 1
	}

	enumIndexes := make(map[string]uint32)
	addEnum := func(values []string) uint32 {
		key := ""
		for _, value := range values {
			key += fmt.Sprintf("%d:%s;", len(value), value)
		}
		if index, ok := enumIndexes[key]; ok {
			return index
		}
		index := uint32(len(available.Enums))
		valueIndices := make([]uint32, len(values))
		for i, value := range values {
			valueIndices[i] = uint32(len(available.EnumValues))
			available.EnumValues = append(available.EnumValues, value)
		}
		available.Enums = append(available.Enums, gtprotocol.CommandEnum{
			Type:         fmt.Sprintf("geyser-go-command-%d", index),
			ValueIndices: valueIndices,
		})
		enumIndexes[key] = index
		return index
	}

	root := java.Nodes[java.RootIndex]
	for _, childIndex := range root.Children {
		if childIndex < 0 || childIndex >= int32(len(java.Nodes)) {
			continue
		}
		child := java.Nodes[childIndex]
		if child.Flags&0x03 != javaCommandNodeLiteral || child.Name == "" {
			continue
		}
		command := gtprotocol.Command{
			Name:            child.Name,
			PermissionLevel: gtprotocol.CommandPermissionLevelAny,
			AliasesOffset:   javaCommandNoAliases,
		}
		overloads := make([]gtprotocol.CommandOverload, 0, 4)
		visited := make(map[int32]bool)
		collectJavaCommandOverloads(java, childIndex, nil, &overloads, visited, addEnum)
		if len(overloads) == 0 {
			overloads = append(overloads, gtprotocol.CommandOverload{})
		}
		command.Overloads = overloads
		available.Commands = append(available.Commands, command)
	}
	if len(available.Commands) == 0 {
		available.Commands = append(available.Commands, gtprotocol.Command{
			Name:            "help",
			PermissionLevel: gtprotocol.CommandPermissionLevelAny,
			AliasesOffset:   javaCommandNoAliases,
			Overloads:       []gtprotocol.CommandOverload{{}},
		})
	}
	return available, 0
}

func collectJavaCommandOverloads(java JavaCommands, nodeIndex int32, path []gtprotocol.CommandParameter, overloads *[]gtprotocol.CommandOverload, visited map[int32]bool, addEnum func([]string) uint32) {
	if len(*overloads) >= javaCommandMaxOverloads || len(path) > javaCommandMaxPathDepth || nodeIndex < 0 || nodeIndex >= int32(len(java.Nodes)) || visited[nodeIndex] {
		return
	}
	visited[nodeIndex] = true
	node := java.Nodes[nodeIndex]
	if node.Flags&javaCommandFlagExecutable != 0 {
		parameters := append([]gtprotocol.CommandParameter(nil), path...)
		*overloads = append(*overloads, gtprotocol.CommandOverload{Parameters: parameters})
	}

	literalValues := make([]string, 0, len(node.Children))
	for _, childIndex := range node.Children {
		if childIndex >= 0 && childIndex < int32(len(java.Nodes)) && java.Nodes[childIndex].Flags&0x03 == javaCommandNodeLiteral {
			literalValues = append(literalValues, java.Nodes[childIndex].Name)
		}
	}
	literalEnum := uint32(0)
	if len(literalValues) != 0 {
		literalEnum = addEnum(literalValues)
	}
	for _, childIndex := range node.Children {
		if childIndex < 0 || childIndex >= int32(len(java.Nodes)) {
			continue
		}
		child := java.Nodes[childIndex]
		var parameter gtprotocol.CommandParameter
		switch child.Flags & 0x03 {
		case javaCommandNodeLiteral:
			parameter = gtprotocol.CommandParameter{
				Name: child.Name,
				Type: literalEnum | gtprotocol.CommandArgEnum | gtprotocol.CommandArgValid,
			}
		case javaCommandNodeArgument:
			parameter = javaCommandParameter(child, addEnum)
		default:
			continue
		}
		collectJavaCommandOverloads(java, childIndex, append(path, parameter), overloads, visited, addEnum)
	}
	delete(visited, nodeIndex)
}

func javaCommandParameter(node JavaCommandNode, addEnum func([]string) uint32) gtprotocol.CommandParameter {
	parameter := gtprotocol.CommandParameter{Name: node.Name, Type: gtprotocol.CommandArgTypeString | gtprotocol.CommandArgValid}
	switch node.Parser {
	case "brigadier:bool":
		index := addEnum([]string{"false", "true"})
		parameter.Type = index | gtprotocol.CommandArgEnum | gtprotocol.CommandArgValid
	case "brigadier:float", "brigadier:double", "minecraft:angle", "minecraft:rotation", "minecraft:float_range":
		parameter.Type = gtprotocol.CommandArgTypeFloat | gtprotocol.CommandArgValid
	case "brigadier:integer", "brigadier:long", "minecraft:int_range", "minecraft:time":
		parameter.Type = gtprotocol.CommandArgTypeInt | gtprotocol.CommandArgValid
	case "minecraft:entity", "minecraft:game_profile":
		parameter.Type = gtprotocol.CommandArgTypeTarget | gtprotocol.CommandArgValid
	case "minecraft:block_pos":
		parameter.Type = gtprotocol.CommandArgTypeBlockPosition | gtprotocol.CommandArgValid
	case "minecraft:column_pos", "minecraft:vec2", "minecraft:vec3":
		parameter.Type = gtprotocol.CommandArgTypePosition | gtprotocol.CommandArgValid
	case "minecraft:message":
		parameter.Type = gtprotocol.CommandArgTypeMessage | gtprotocol.CommandArgValid
	case "minecraft:component", "minecraft:style", "minecraft:nbt", "minecraft:nbt_tag", "minecraft:nbt_path":
		parameter.Type = gtprotocol.CommandArgTypeJSON | gtprotocol.CommandArgValid
	case "minecraft:operation":
		parameter.Type = gtprotocol.CommandArgTypeOperator | gtprotocol.CommandArgValid
	case "minecraft:function", "minecraft:resource_location":
		parameter.Type = gtprotocol.CommandArgTypeFilepath | gtprotocol.CommandArgValid
	}
	return parameter
}
