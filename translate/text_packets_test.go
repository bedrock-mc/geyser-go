package translate

import (
	"testing"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
)

func TestDecodeSystemChat(t *testing.T) {
	w := javaprotocol.NewWriter()
	writeNetworkTextComponent(t, w, "hello")
	_ = w.Bool(true)
	chat, err := DecodeSystemChat(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if !chat.ActionBar || JavaTextComponentText(chat.Content) != "hello" {
		t.Fatalf("decoded system chat = %+v, text=%q", chat, JavaTextComponentText(chat.Content))
	}
}

func TestDecodePlayerChat(t *testing.T) {
	w := javaprotocol.NewWriter()
	for i := 0; i < 16; i++ {
		_ = w.Byte(byte(i))
	}
	_ = w.VarInt(0)
	_ = w.Bool(false)
	_ = w.String("hello")
	_ = w.Int64(123)
	_ = w.Int64(456)
	_ = w.VarInt(0)
	_ = w.Bool(false)
	_ = w.VarInt(0)
	_ = w.VarInt(1) // chat type registry reference
	writeNetworkStringComponent(t, w, "Steve")
	_ = w.Bool(false)
	chat, err := DecodePlayerChat(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if chat.Message != "hello" || JavaTextComponentText(chat.NetworkName) != "Steve" || chat.SenderUUID[15] != 15 {
		t.Fatalf("decoded player chat = %+v", chat)
	}
}

func TestJavaTextComponentText(t *testing.T) {
	component := map[string]any{
		"text": "hello ",
		"extra": []any{
			map[string]any{"translate": "world"},
			map[string]any{"text": "!"},
		},
	}
	if got := JavaTextComponentText(component); got != "hello world!" {
		t.Fatalf("text component = %q", got)
	}
}

func writeNetworkTextComponent(t *testing.T, w *javaprotocol.Writer, text string) {
	t.Helper()
	_ = w.Byte(10) // TAG_Compound
	_ = w.Byte(8)  // TAG_String
	_ = w.Int16(4)
	_ = w.BytesValue([]byte("text"))
	_ = w.Int16(int16(len(text)))
	_ = w.BytesValue([]byte(text))
	_ = w.Byte(0) // TAG_End
}

func writeNetworkStringComponent(t *testing.T, w *javaprotocol.Writer, text string) {
	t.Helper()
	_ = w.Byte(8) // TAG_String, root name omitted on the network
	_ = w.Int16(int16(len(text)))
	_ = w.BytesValue([]byte(text))
}
