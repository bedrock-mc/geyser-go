package translate

import (
	"testing"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
	gtprotocol "github.com/sandertv/gophertunnel/minecraft/protocol"
)

func TestDecodeJavaCommandsAndBuildBedrockTree(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.VarInt(3)
	writeJavaCommandNode(t, w, 0, []int32{1}, "", -1)
	writeJavaCommandNode(t, w, javaCommandNodeLiteral, []int32{2}, "say", -1)
	writeJavaCommandNode(t, w, javaCommandNodeArgument|javaCommandFlagExecutable, nil, "message", 19)
	_ = w.VarInt(0)

	commands, err := DecodeJavaCommands(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(commands.Nodes) != 3 || commands.RootIndex != 0 || commands.Nodes[2].Parser != "minecraft:message" {
		t.Fatalf("decoded commands = %+v", commands)
	}
	available, skipped := buildBedrockCommands(commands)
	if skipped != 0 || len(available.Commands) != 1 {
		t.Fatalf("available commands=%+v skipped=%d", available.Commands, skipped)
	}
	command := available.Commands[0]
	if command.Name != "say" || len(command.Overloads) != 1 || len(command.Overloads[0].Parameters) != 1 {
		t.Fatalf("command = %+v", command)
	}
	parameter := command.Overloads[0].Parameters[0]
	if parameter.Name != "message" || parameter.Type != gtprotocol.CommandArgTypeMessage|gtprotocol.CommandArgValid {
		t.Fatalf("message parameter = %+v", parameter)
	}
}

func TestDecodeJavaCommandProperties(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.VarInt(2)
	writeJavaCommandNode(t, w, 0, []int32{1}, "", -1)
	writeJavaCommandNode(t, w, javaCommandNodeArgument|javaCommandFlagExecutable|javaCommandFlagSuggestions, nil, "value", 3)
	_ = w.Byte(0x03)
	_ = w.Int32(-10)
	_ = w.Int32(10)
	_ = w.String("minecraft:ask_server")
	_ = w.VarInt(0)

	commands, err := DecodeJavaCommands(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if commands.Nodes[1].Parser != "brigadier:integer" || commands.Nodes[1].Suggestions != "minecraft:ask_server" {
		t.Fatalf("decoded argument node = %+v", commands.Nodes[1])
	}
}

func TestDecodeJavaCommandsRejectsTruncation(t *testing.T) {
	if _, err := DecodeJavaCommands([]byte{1, 0}); err == nil {
		t.Fatal("expected truncated command node error")
	}
}

func TestBuildBedrockCommandsProjectsRedirectAliases(t *testing.T) {
	java := JavaCommands{
		Nodes: []JavaCommandNode{
			{Flags: 0, Children: []int32{1, 3}, Redirect: -1},
			{Flags: javaCommandNodeLiteral, Children: []int32{2}, Redirect: -1, Name: "msg"},
			{Flags: javaCommandNodeArgument | javaCommandFlagExecutable, Redirect: -1, Name: "message", Parser: "minecraft:message"},
			{Flags: javaCommandNodeLiteral | javaCommandFlagRedirect, Redirect: 1, Name: "tell"},
		},
		RootIndex: 0,
	}
	available, skipped := buildBedrockCommands(java)
	if skipped != 0 || len(available.Commands) != 1 || len(available.Enums) != 1 {
		t.Fatalf("commands=%+v enums=%+v skipped=%d", available.Commands, available.Enums, skipped)
	}
	command := available.Commands[0]
	if command.Name != "msg" || command.AliasesOffset != 0 || len(command.Overloads) != 1 || len(available.EnumValues) != 1 || available.EnumValues[0] != "tell" {
		t.Fatalf("redirect alias projection command=%+v enums=%+v values=%v", command, available.Enums, available.EnumValues)
	}
}

func TestCollectJavaCommandOverloadsFollowsNestedRedirect(t *testing.T) {
	java := JavaCommands{Nodes: []JavaCommandNode{
		{Flags: 0, Children: []int32{1}},
		{Flags: javaCommandNodeLiteral, Children: []int32{2}, Name: "root", Redirect: -1},
		{Flags: javaCommandNodeLiteral | javaCommandFlagRedirect, Redirect: 3, Name: "alias"},
		{Flags: javaCommandNodeLiteral, Children: []int32{4}, Name: "target", Redirect: -1},
		{Flags: javaCommandNodeArgument | javaCommandFlagExecutable, Name: "value", Parser: "brigadier:integer", Redirect: -1},
	}, RootIndex: 0}
	overloads := make([]gtprotocol.CommandOverload, 0, 1)
	collectJavaCommandOverloads(java, 1, nil, &overloads, make(map[int32]bool), func([]string) uint32 { return 0 })
	if len(overloads) != 1 || len(overloads[0].Parameters) != 2 || overloads[0].Parameters[0].Name != "alias" || overloads[0].Parameters[1].Name != "value" {
		t.Fatalf("redirect overloads=%+v", overloads)
	}
}

func writeJavaCommandNode(t *testing.T, w *javaprotocol.Writer, flags byte, children []int32, name string, parser int32) {
	t.Helper()
	_ = w.Byte(flags)
	_ = w.VarInt(int32(len(children)))
	for _, child := range children {
		_ = w.VarInt(child)
	}
	if flags&javaCommandFlagRedirect != 0 {
		_ = w.VarInt(0)
	}
	switch flags & 0x03 {
	case javaCommandNodeLiteral:
		_ = w.String(name)
	case javaCommandNodeArgument:
		_ = w.String(name)
		_ = w.VarInt(parser)
	}
}
