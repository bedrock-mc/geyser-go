package translate

import (
	"testing"

	javaprotocol "github.com/bedrock-mc/geyser-go/java/protocol"
)

func TestDecodeJavaResourcePackPush(t *testing.T) {
	w := javaprotocol.NewWriter()
	for i := 0; i < 16; i++ {
		_ = w.Byte(byte(i))
	}
	_ = w.String("https://example.invalid/pack.zip")
	_ = w.String("0123456789abcdef")
	_ = w.Bool(true)
	_ = w.Bool(false)
	pack, err := DecodeJavaResourcePackPush(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if pack.UUID[0] != 0 || pack.UUID[15] != 15 || pack.URL == "" || !pack.Required || pack.PromptMessage != nil {
		t.Fatalf("decoded resource pack = %+v", pack)
	}
}

func TestDecodeJavaResourcePackPop(t *testing.T) {
	w := javaprotocol.NewWriter()
	_ = w.Bool(true)
	for i := 0; i < 16; i++ {
		_ = w.Byte(byte(15 - i))
	}
	uuid, err := DecodeJavaResourcePackPop(w.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if uuid == nil || uuid[0] != 15 || uuid[15] != 0 {
		t.Fatalf("decoded resource pack UUID = %v", uuid)
	}
}

func TestEncodeJavaResourcePackStatus(t *testing.T) {
	var uuid [16]byte
	for i := range uuid {
		uuid[i] = byte(i)
	}
	payload := encodeJavaResourcePackStatus(uuid, 4)
	r := javaprotocol.NewReader(payload)
	got, err := r.Bytes(16)
	if err != nil {
		t.Fatal(err)
	}
	status, err := r.VarInt()
	if err != nil {
		t.Fatal(err)
	}
	if status != 4 || got[0] != 0 || got[15] != 15 || r.Remaining() != 0 {
		t.Fatalf("status payload uuid=%x status=%d remaining=%d", got, status, r.Remaining())
	}
}
