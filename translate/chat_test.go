package translate

import (
	"errors"
	"strings"
	"testing"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
)

func TestNormalizeIncomingChat(t *testing.T) {
	if got := normalizeIncomingChat("\t  /say\u00a0hello   world\n"); got != "/say hello world" {
		t.Fatalf("normalized chat = %q", got)
	}
	if got := normalizeIncomingChat(" \t\n "); got != "" {
		t.Fatalf("whitespace-only chat = %q", got)
	}
}

func TestEncodeChatMessage1214(t *testing.T) {
	payload, err := encodeChatMessage("hello")
	if err != nil {
		t.Fatal(err)
	}
	r := javaprotocol.NewReader(payload)
	message, err := r.String()
	if err != nil {
		t.Fatal(err)
	}
	if message != "hello" {
		t.Fatalf("message = %q", message)
	}
	if _, err := r.Int64(); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Int64(); err != nil {
		t.Fatal(err)
	}
	signature, err := r.Bool()
	if err != nil {
		t.Fatal(err)
	}
	if signature {
		t.Fatal("chat message unexpectedly has a signature")
	}
	offset, err := r.VarInt()
	if err != nil {
		t.Fatal(err)
	}
	if offset != 0 {
		t.Fatalf("offset = %d", offset)
	}
	acknowledged, err := r.Bytes(3)
	if err != nil {
		t.Fatal(err)
	}
	if string(acknowledged) != "\x00\x00\x00" || r.Remaining() != 0 {
		t.Fatalf("acknowledged=%x remaining=%d", acknowledged, r.Remaining())
	}
}

func TestEncodeChatCommandSigned1214(t *testing.T) {
	payload, err := encodeChatCommandSigned("time set day")
	if err != nil {
		t.Fatal(err)
	}
	r := javaprotocol.NewReader(payload)
	command, err := r.String()
	if err != nil {
		t.Fatal(err)
	}
	if command != "time set day" {
		t.Fatalf("command = %q", command)
	}
	if _, err := r.Int64(); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Int64(); err != nil {
		t.Fatal(err)
	}
	argumentSignatures, err := r.VarInt()
	if err != nil {
		t.Fatal(err)
	}
	messageCount, err := r.VarInt()
	if err != nil {
		t.Fatal(err)
	}
	acknowledged, err := r.Bytes(3)
	if err != nil {
		t.Fatal(err)
	}
	if argumentSignatures != 0 || messageCount != 0 || string(acknowledged) != "\x00\x00\x00" || r.Remaining() != 0 {
		t.Fatalf("signatures=%d messages=%d acknowledged=%x remaining=%d", argumentSignatures, messageCount, acknowledged, r.Remaining())
	}
}

func TestEncodeChatCommandRejectsEmptyOrOversized(t *testing.T) {
	for _, command := range []string{"", strings.Repeat("x", javaChatMessageLimit+1)} {
		if _, err := encodeChatCommandSigned(command); !errors.Is(err, errInvalidChatCommand) {
			t.Fatalf("command %q error = %v", command, err)
		}
	}
}
